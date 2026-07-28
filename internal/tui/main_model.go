package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lutrarutra/lazypush/internal/config"
	"github.com/lutrarutra/lazypush/internal/gh"
	"github.com/lutrarutra/lazypush/internal/git"
	"github.com/lutrarutra/lazypush/internal/llm"
)

type screen int

const (
	screenLogin screen = iota
	screenVersion
	screenLoading
	screenReview
	screenPRAsk
	screenBranchSelect
	screenPRReview
	screenNewBranch
	screenConfirm
	screenProgress
)

type Model struct {
	screen    screen
	login     loginScreenModel
	version   versionScreenModel
	loading   loadingScreenModel
	review    reviewScreenModel
	prAsk     prAskScreenModel
	branchSel branchSelectScreenModel
	prReview  prReviewScreenModel
	confirm   confirmScreenModel
	progress  progressScreenModel

	config    *config.Config
	repo      *git.Repo
	llmClient *llm.Client

	versionTag     string
	currentBranch  string
	commitMsg      string
	targetBranch   string
	prBody         string
	prURL          string
	includePR      bool
	hasDiff        bool
	forceLogin     bool
	newBranchName  string
	newBranchInput textinput.Model
	err            error
}

func NewModel(forceLogin bool) *Model {
	nb := textinput.New()
	nb.Placeholder = "feature/my-new-feature"
	nb.Prompt = "Branch name: "
	nb.Focus()
	nb.CharLimit = 100

	return &Model{
		screen:         screenLogin,
		login:          newLoginScreen(),
		newBranchName:  "",
		newBranchInput: nb,
		review:         newReviewScreen("", ""),
		forceLogin:     forceLogin,
	}
}

func (m *Model) Init() tea.Cmd {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: could not load config: %v", err)
		cfg = &config.Config{
			APIURL: "https://api.openai.com/v1",
			Model:  "gpt-4o-mini",
		}
	}

	// forceLogin was requested via 'lazypush login' subcommand
	if m.forceLogin {
		m.config = cfg
		return m.login.Init()
	}

	cwd, _ := os.Getwd()
	repo, err := git.Open(cwd)
	if err != nil {
		log.Printf("warning: could not open git repo: %v", err)
		repo = nil
	}

	m.config = cfg
	m.repo = repo

	// Check for diff early
	if repo != nil {
		diff, err := repo.Diff()
		if err == nil {
			m.hasDiff = diff != ""
		}
	}

	// If API key is present, set up the LLM client
	if cfg.APIKey != "" && cfg.APIURL != "" && cfg.Model != "" {
		m.llmClient = llm.New(cfg.APIURL, cfg.APIKey, cfg.Model)
	}

	if !m.hasDiff && m.llmClient != nil {
		// No changes with LLM — skip version screen, go straight to PR flow
		m.loading = newLoadingScreen("Checking branch...")
		m.screen = screenLoading
		return m.loadingCmd(m.afterReviewConfirmed())
	}

	// If no API key, show a brief notice then proceed to version screen
	// (user will write commit message manually, can run 'lazypush login' later)
	m.screen = screenVersion
	return m.initVersionScreen()
}

func (m *Model) initVersionScreenNoLLM() tea.Cmd {
	tag := "v0.0.0"
	if m.repo != nil {
		latest, err := m.repo.LatestTag()
		if err == nil && latest != "" {
			tag = latest
		}
	}
	m.versionTag = tag
	m.version = newVersionScreen(tag)
	m.version.useLLM = false
	m.version.done = true
	m.screen = screenReview
	diff, err := m.repo.Diff()
	if err != nil {
		diff = ""
	}
	m.review = newReviewScreenWithLLM(diff, "", true)
	return nil
}

func (m *Model) initVersionScreen() tea.Cmd {
	tag := "v0.0.0"
	if m.repo != nil {
		latest, err := m.repo.LatestTag()
		if err == nil && latest != "" {
			tag = latest
		}
	}
	m.versionTag = tag
	m.version = newVersionScreen(tag)
	m.screen = screenVersion
	return m.version.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Ctrl+C / Ctrl+D exits immediately from anywhere
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			return m, tea.Quit
		}
	}

	switch m.screen {
	case screenLogin:
		return m.updateLogin(msg)
	case screenVersion:
		return m.updateVersion(msg)
	case screenLoading:
		return m.updateLoading(msg)
	case screenReview:
		return m.updateReview(msg)
	case screenPRAsk:
		return m.updatePRAsk(msg)
	case screenBranchSelect:
		return m.updateBranchSelect(msg)
	case screenPRReview:
		return m.updatePRReview(msg)
	case screenNewBranch:
		return m.updateNewBranch(msg)
	case screenConfirm:
		return m.updateConfirm(msg)
	case screenProgress:
		return m.updateProgress(msg)
	}
	return m, nil
}

func (m *Model) updatePRAsk(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.prAsk, cmd = m.prAsk.Update(msg)

	if m.prAsk.cancelled {
		return m, m.revertToReview()
	}
	if m.prAsk.confirmed {
		m.prAsk.confirmed = false
		switch m.prAsk.choice {
		case prChoiceCreate:
			m.includePR = true
			m.loading = newLoadingScreen("Fetching branches...")
			m.screen = screenLoading
			return m, m.loadingCmd(m.afterPRAskCreate())
		case prChoiceBranchOut:
			m.screen = screenNewBranch
			m.newBranchName = ""
			return m, nil
		case prChoiceCommitHere:
			m.includePR = false
			m.confirm = newConfirmScreen(m.versionTag, m.version.currentTag, m.review.commitMessage.Value(), false, m.needsTagging(), m.versionTag != m.version.currentTag, "", m.currentBranch, "")
			m.screen = screenConfirm
			return m, nil
		}
	}
	return m, cmd
}

func (m *Model) afterPRAskCreate() tea.Cmd {
	return func() tea.Msg {
		branches, err := m.repo.ListBranches()
		if err != nil {
			return errMsg{err: fmt.Sprintf("list branches: %v", err)}
		}
		return branchListReadyMsg{branches: branches}
	}
}

func (m *Model) revertToReview() tea.Cmd {
	if m.hasDiff {
		m.review.cancelled = false
		m.review.confirmed = false
		m.screen = screenReview
	} else {
		// No diff — review screen was never shown, go to PR flow
		m.loading = newLoadingScreen("Checking branch...")
		m.screen = screenLoading
		return m.loadingCmd(m.afterReviewConfirmed())
	}
	return nil
}

func (m *Model) updateNewBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.newBranchInput.Value())
			if name == "" {
				return m, nil
			}
			m.includePR = false
			if err := m.repo.CreateBranchAndSwitch(name); err != nil {
				m.err = fmt.Errorf("create branch: %w", err)
				return m, tea.Quit
			}
			m.currentBranch = name
			m.confirm = newConfirmScreen(m.versionTag, m.version.currentTag, m.review.commitMessage.Value(), false, m.needsTagging(), m.versionTag != m.version.currentTag,
				fmt.Sprintf("Pushing to new branch: %s", name),
				m.currentBranch, "",
			)
			m.screen = screenConfirm
			return m, nil
		case "esc":
			m.newBranchInput.SetValue("")
			m.screen = screenPRAsk
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.newBranchInput, cmd = m.newBranchInput.Update(msg)
	return m, cmd
}

func (m *Model) updateBranchSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.branchSel, cmd = m.branchSel.Update(msg)

	if m.branchSel.cancelled {
		m.prAsk = newPRAskScreen(m.currentBranch, m.hasDiff)
		m.screen = screenPRAsk
		return m, nil
	}
	if m.branchSel.chosen {
		m.targetBranch = m.branchSel.branches[m.branchSel.selected]
		m.loading = newLoadingScreen("Generating PR description...")
		m.screen = screenLoading
		return m, m.loadingCmd(m.generatePRDescription())
	}
	return m, cmd
}

func (m *Model) updatePRReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.prReview, cmd = m.prReview.Update(msg)

	if m.prReview.confirmed {
		m.prBody = m.prReview.description.Value()
		m.confirm = newConfirmScreen(m.versionTag, m.version.currentTag, m.review.commitMessage.Value(), true, m.needsTagging(), m.versionTag != m.version.currentTag, "", m.currentBranch, m.targetBranch)
		m.screen = screenConfirm
		m.screen = screenConfirm
		return m, nil
	}
	if m.prReview.cancelled {
		m.branchSel.chosen = false
		m.includePR = false
		m.screen = screenBranchSelect
		return m, nil
	}
	return m, cmd
}

func (m *Model) generatePRDescription() tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return errMsg{err: "no git repository found"}
		}
		if m.llmClient == nil {
			return prDescriptionReadyMsg{description: ""}
		}

		m.repo.SetBaseBranch(m.targetBranch)

		sys, user := llm.PRDescriptionIterPrompt(m.targetBranch)
		desc, err := m.llmClient.GenerateWithTools(
			context.Background(),
			sys, user,
			m.repo,
			100,
			nil, // no progress callback to avoid data races with the TUI
		)
		if err != nil {
			return errMsg{err: fmt.Sprintf("generate PR: %v", err)}
		}
		return prDescriptionReadyMsg{description: desc}
	}
}

func (m *Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.confirm, cmd = m.confirm.Update(msg)

	if m.confirm.confirmed {
		return m, m.executeOperations()
	}
	if m.confirm.cancelled {
		return m, tea.Quit
	}

	return m, cmd
}

func (m *Model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.login, cmd = m.login.Update(msg)

	if m.login.done {
		apiURL, model, apiKey := m.login.Values()

		if m.login.err == "cancelled" {
			// If we were forced to login, just exit
			if m.forceLogin {
				return m, tea.Quit
			}
			// User skipped login — proceed without LLM
			cwd, _ := os.Getwd()
			repo, err := git.Open(cwd)
			if err == nil {
				m.repo = repo
				diff, err := repo.Diff()
				if err == nil {
					m.hasDiff = diff != ""
				}
			}
			if m.hasDiff {
				return m, m.initVersionScreenNoLLM()
			}
			// No diff and no LLM — nothing to do
			return m, tea.Quit
		}
		if m.login.err != "" {
			return m, tea.Quit
		}

		m.config.APIURL = apiURL
		m.config.Model = model
		m.config.APIKey = apiKey

		if err := config.Save(m.config); err != nil {
			m.err = err
			return m, tea.Quit
		}

		m.llmClient = llm.New(apiURL, apiKey, model)

		// Re-check diff now that we have a repo
		if m.repo != nil {
			diff, err := m.repo.Diff()
			if err == nil {
				m.hasDiff = diff != ""
			}
		}

		if !m.hasDiff {
			m.loading = newLoadingScreen("Checking branch...")
			m.screen = screenLoading
			return m, m.loadingCmd(m.afterReviewConfirmed())
		}
		return m, m.initVersionScreen()
	}

	return m, cmd
}

func (m *Model) updateVersion(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.version, cmd = m.version.Update(msg)

	if m.version.done {
		if m.version.chosenVersion == "" {
			return m, tea.Quit
		}
		m.versionTag = m.version.chosenVersion

		// If no LLM client, skip LLM generation and go straight to manual review
		if m.llmClient == nil {
			diff, err := m.repo.Diff()
			if err != nil {
				diff = ""
			}
			m.hasDiff = diff != ""
			m.review = newReviewScreenWithLLM(diff, "", true)
			m.screen = screenReview
			return m, nil
		}

		if m.version.useLLM {
			m.loading = newLoadingScreen("Generating commit message...")
			m.screen = screenLoading
			return m, m.loadingCmd(m.generateCommitMessage())
		}

		// Write manually — go to review with an empty message
		diff, err := m.repo.Diff()
		if err != nil {
			diff = ""
		}
		m.hasDiff = diff != ""
		if m.hasDiff {
			m.review = newReviewScreen(diff, "")
			m.screen = screenReview
			return m, nil
		}
		// No changes — skip review, go straight to branch check / PR flow
		m.loading = newLoadingScreen("Checking branch...")
		m.screen = screenLoading
		return m, m.loadingCmd(m.afterReviewConfirmed())
	}

	return m, cmd
}

func (m *Model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case commitMessageReadyMsg:
		m.review = newReviewScreen(msg.diff, msg.message)
		m.screen = screenReview
		return m, nil
	case noDiffReadyMsg:
		// No changes — go straight to post-review flow
		return m, m.afterReviewConfirmed()
	case prDescriptionReadyMsg:
		m.prReview = newPRReviewScreen(msg.description)
		m.screen = screenPRReview
		return m, nil
	case branchCheckedMsg:
		m.includePR = false
		m.prAsk = newPRAskScreen(msg.branch, m.hasDiff)
		m.screen = screenPRAsk
		return m, nil
	case branchListReadyMsg:
		m.branchSel = newBranchSelectScreen(msg.branches, m.currentBranch)
		m.screen = screenBranchSelect
		return m, nil
	case errMsg:
		m.err = fmt.Errorf("%s", msg.err)
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.loading, cmd = m.loading.Update(msg)
	return m, cmd
}

func (m *Model) loadingCmd(workCmd tea.Cmd) tea.Cmd {
	return tea.Batch(m.loading.Init(), workCmd)
}

func (m *Model) generateCommitMessage() tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return errMsg{err: "no git repository found"}
		}

		diff, err := m.repo.Diff()
		if err != nil {
			return errMsg{err: err.Error()}
		}

		m.hasDiff = diff != ""

		if !m.hasDiff {
			return noDiffReadyMsg{}
		}

		sys, user := llm.CommitMessagePrompt(diff)
		msg, err := m.llmClient.Generate(context.Background(), sys, user)
		if err != nil {
			return errMsg{err: err.Error()}
		}

		return commitMessageReadyMsg{diff: diff, message: msg}
	}
}

func (m *Model) updateReview(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.review, cmd = m.review.Update(msg)

	if m.review.confirmed {
		m.review.confirmed = false
		m.loading = newLoadingScreen("Checking branch...")
		m.screen = screenLoading
		return m, m.loadingCmd(m.afterReviewConfirmed())
	}
	if m.review.cancelled {
		return m, tea.Quit
	}

	return m, cmd
}

func (m *Model) afterReviewConfirmed() tea.Cmd {
	return func() tea.Msg {
		if m.repo == nil {
			return errMsg{err: "no git repository found"}
		}
		branch, err := m.repo.CurrentBranch()
		if err != nil {
			return errMsg{err: fmt.Sprintf("get current branch: %v", err)}
		}
		m.currentBranch = branch

		if branch == "main" || branch == "master" {
			return branchCheckedMsg{branch: branch, onMain: true}
		}
		return branchCheckedMsg{branch: branch, onMain: false}
	}
}

func (m *Model) executeOperations() tea.Cmd {
	steps := []string{}
	if m.hasDiff {
		steps = append(steps, "Committing...")
		if m.needsTagging() {
			steps = append(steps, "Tagging...")
		}
		steps = append(steps, "Pushing...")
	}
	if m.includePR {
		steps = append(steps, "Creating PR...")
	}
	m.progress = newProgressScreen(steps)
	m.screen = screenProgress
	return m.execStep(0)
}

func (m *Model) needsTagging() bool {
	return m.version.reTag || m.versionTag != m.version.currentTag
}

func (m *Model) execStep(displayIdx int) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch displayIdx {
		case 0:
			if m.hasDiff {
				err = m.repo.Commit(m.review.commitMessage.Value())
			} else if m.includePR {
				err = m.createPR()
			}
		case 1:
			if m.hasDiff {
				if m.needsTagging() {
					if m.version.reTag {
						_ = m.repo.DeleteTag(m.versionTag)
						err = m.repo.Tag(m.versionTag)
					} else {
						err = m.repo.Tag(m.versionTag)
					}
				} else {
					err = m.repo.Push("origin")
				}
			} else if m.includePR {
				err = m.createPR()
			}
		case 2:
			if m.hasDiff && m.needsTagging() {
				err = m.repo.Push("origin")
			} else {
				err = m.createPR()
			}
		case 3:
			err = m.createPR()
		}
		return execStepResult{step: displayIdx, err: err}
	}
}

func (m *Model) createPR() error {
	if gh.CheckInstalled() {
		prURL, err := gh.CreatePR(m.review.commitMessage.Value(), m.prBody, m.targetBranch, m.currentBranch)
		if err != nil {
			return err
		}
		m.prURL = prURL
	} else {
		return fmt.Errorf("GitHub CLI (gh) not found — install it from https://cli.github.com to enable PR creation. Your PR description was:\n\n%s", m.prBody)
	}
	return nil
}

func (m *Model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case execStepResult:
		ok := msg.err == nil
		msgStr := ""
		if msg.err != nil {
			msgStr = msg.err.Error()
		}

		m.progress, _ = m.progress.Update(progressStepDone{
			index:   msg.step,
			ok:      ok,
			message: msgStr,
		})

		if !ok {
			return m, nil
		}
		nextIdx := msg.step + 1
		if nextIdx < m.progress.StepCount() {
			return m, m.execStep(nextIdx)
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.progress, cmd = m.progress.Update(msg)
	return m, cmd
}

func (m *Model) loadingLabelMatch(stepLabel string) bool {
	// When on the loading screen, match based on the label text
	label := m.loading.label
	switch {
	case stepLabel == "💬  Commit" && label == "Generating commit message...":
		return true
	case stepLabel == "📝  PR" && label == "Generating PR description...":
		return true
	case stepLabel == "🎯  Branch" && label == "Fetching branches...":
		return true
	case stepLabel == "🔀  Next" && (label == "Checking branch..." || label == "Fetching branches..."):
		return true
	}
	return false
}

func (m *Model) workflowLine() string {
	steps := []struct {
		screen screen
		label  string
	}{
		{screenVersion, "🏷️  Tag"},
		{screenReview, "💬  Commit"},
		{screenPRAsk, "🔀  Next"},
		{screenBranchSelect, "🎯  Branch"},
		{screenNewBranch, "🌿  New"},
		{screenPRReview, "📝  PR"},
		{screenConfirm, "✅  Confirm"},
	}

	// Build dots
	var s strings.Builder
	for i, step := range steps {
		if i > 0 {
			s.WriteString("  ")
		}
		if step.screen == m.screen || (m.screen == screenLoading && m.loadingLabelMatch(step.label)) {
			s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("236")).Background(lipgloss.Color("39")).Render(" " + step.label + " "))
		} else if m.isWorkflowStepCompleted(i) {
			s.WriteString(lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("114")).Render(step.label))
		} else {
			s.WriteString(lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("240")).Render(step.label))
		}
	}
	return s.String()
}

func (m *Model) isWorkflowStepCompleted(index int) bool {
	switch index {
	case 0: // Tag — past if we're past version screen
		return m.screen > screenVersion
	case 1: // Commit — past review
		return m.screen > screenReview
	case 2: // Next — past PR ask
		return m.screen > screenPRAsk
	case 3: // Branch select — past it
		return m.screen > screenBranchSelect
	case 4: // New branch — past it
		return m.screen > screenNewBranch
	case 5: // PR review — past it
		return m.screen > screenPRReview
	case 6: // Confirm — at or past it
		return m.screen >= screenConfirm
	}
	return false
}

func (m *Model) View() string {
	switch m.screen {
	case screenLogin:
		return m.workflowLine() + "\n\n" + m.login.View()
	case screenVersion:
		return m.workflowLine() + "\n\n" + m.version.View()
	case screenLoading:
		return m.workflowLine() + "\n\n" + m.loading.View()
	case screenReview:
		return m.workflowLine() + "\n\n" + m.review.View()
	case screenPRAsk:
		return m.workflowLine() + "\n\n" + m.prAsk.View()
	case screenBranchSelect:
		return m.workflowLine() + "\n\n" + m.branchSel.View()
	case screenPRReview:
		return m.workflowLine() + "\n\n" + m.prReview.View()
	case screenNewBranch:
		var s strings.Builder
		s.WriteString(m.workflowLine())
		s.WriteString("\n\n")
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("🌿  New Branch"))
		s.WriteString("\n\n")
		s.WriteString(lipgloss.NewStyle().Faint(true).Render("Create a new branch to push these changes to:"))
		s.WriteString("\n\n")
		s.WriteString(m.newBranchInput.View())
		s.WriteString("\n\n")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("  Enter"))
		s.WriteString(lipgloss.NewStyle().Faint(true).Render(" Create branch and commit"))
		s.WriteString("  ")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Render("Esc"))
		s.WriteString(lipgloss.NewStyle().Faint(true).Render(" Back"))
		return s.String()
	case screenConfirm:
		return m.workflowLine() + "\n\n" + m.confirm.View()
	case screenProgress:
		return m.workflowLine() + "\n\n" + m.progress.View()
	}
	return ""
}

type commitMessageReadyMsg struct {
	diff    string
	message string
}

type noDiffReadyMsg struct{}

type branchCheckedMsg struct {
	branch string
	onMain bool
}

type branchListReadyMsg struct {
	branches []string
}

type prDescriptionReadyMsg struct {
	description string
}

type errMsg struct {
	err string
}

type execStepResult struct {
	step int
	err  error
}
