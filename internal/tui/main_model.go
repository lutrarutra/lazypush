package tui

import (
	"context"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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
	screenProgress
)

type Model struct {
	screen   screen
	login    loginScreenModel
	version  versionScreenModel
	loading  loadingScreenModel
	review   reviewScreenModel
	progress progressScreenModel

	config    *config.Config
	repo      *git.Repo
	llmClient *llm.Client

	versionTag string
	commitMsg  string
	prBody     string
	prURL      string
	err        error
}

func NewModel() *Model {
	return &Model{
		screen: screenLogin,
		login:  newLoginScreen(),
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

	cwd, _ := os.Getwd()
	repo, err := git.Open(cwd)
	if err != nil {
		log.Printf("warning: could not open git repo: %v", err)
		repo = nil
	}

	m.config = cfg
	m.repo = repo

	if cfg.APIKey != "" && cfg.APIURL != "" && cfg.Model != "" {
		m.llmClient = llm.New(cfg.APIURL, cfg.APIKey, cfg.Model)
		m.screen = screenVersion
		return m.initVersionScreen()
	}

	return m.login.Init()
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
	switch m.screen {
	case screenLogin:
		return m.updateLogin(msg)
	case screenVersion:
		return m.updateVersion(msg)
	case screenLoading:
		return m.updateLoading(msg)
	case screenReview:
		return m.updateReview(msg)
	case screenProgress:
		return m.updateProgress(msg)
	}
	return m, nil
}

func (m *Model) updateLogin(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.login, cmd = m.login.Update(msg)

	if m.login.done {
		if m.login.err != "" {
			return m, tea.Quit
		}

		apiURL, model, apiKey := m.login.Values()
		m.config.APIURL = apiURL
		m.config.Model = model
		m.config.APIKey = apiKey

		if err := config.Save(m.config); err != nil {
			m.err = err
			return m, tea.Quit
		}

		m.llmClient = llm.New(apiURL, apiKey, model)
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

		if m.version.useLLM {
			m.loading = newLoadingScreen("Generating commit message...")
			m.screen = screenLoading
			return m, m.generateCommitMessage()
		}

		// Write manually — go to review with an empty message
		diff, err := m.repo.Diff()
		if err != nil {
			diff = ""
		}
		m.review = newReviewScreen(diff, "")
		m.screen = screenReview
		return m, nil
	}

	return m, cmd
}

func (m *Model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case commitMessageReadyMsg:
		m.review = newReviewScreen(msg.diff, msg.message)
		m.screen = screenReview
		return m, nil
	case errMsg:
		m.err = fmt.Errorf("%s", msg.err)
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.loading, cmd = m.loading.Update(msg)
	return m, cmd
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

		if diff == "" {
			// No changes — show empty review screen
			return commitMessageReadyMsg{diff: "", message: ""}
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
		return m, m.executeOperations()
	}
	if m.review.cancelled {
		return m, tea.Quit
	}

	return m, cmd
}

func (m *Model) executeOperations() tea.Cmd {
	m.progress = newProgressScreen()
	m.screen = screenProgress
	return m.execStep0()
}

func (m *Model) execStep0() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Commit(m.review.commitMessage.Value())
		return execStepResult{step: 0, err: err}
	}
}

func (m *Model) execStep1() tea.Cmd {
	return func() tea.Msg {
		// Determine if tagging should happen
		if m.version.reTag {
			// Keep + re-tag: delete old, create new
			_ = m.repo.DeleteTag(m.versionTag)
			err := m.repo.Tag(m.versionTag)
			return execStepResult{step: 1, err: err}
		}
		if m.versionTag != m.version.currentTag {
			// New version: create tag
			err := m.repo.Tag(m.versionTag)
			return execStepResult{step: 1, err: err}
		}
		// Keep + no re-tag: skip tagging
		return execStepResult{step: 1, err: nil}
	}
}

func (m *Model) execStep2() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.Push("origin")
		return execStepResult{step: 2, err: err}
	}
}

func (m *Model) execStep3() tea.Cmd {
	return func() tea.Msg {
		if m.review.includePR && gh.CheckInstalled() {
			prURL, err := gh.CreatePR(m.review.commitMessage.Value(), m.review.prDescription.Value())
			if err != nil {
				return execStepResult{step: 3, err: err}
			}
			m.prURL = prURL
		}
		return execStepResult{step: 3, err: nil}
	}
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
			return m, nil // keep showing final failed state
		}
		if m.progress.done {
			return m, nil // keep showing final success state
		}

		// Chain to next step
		switch msg.step + 1 {
		case 1:
			return m, m.execStep1()
		case 2:
			return m, m.execStep2()
		case 3:
			return m, m.execStep3()
		case 4:
			// All done — keep showing final state
			return m, nil
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.progress, cmd = m.progress.Update(msg)
	return m, cmd
}

func (m *Model) View() string {
	switch m.screen {
	case screenLogin:
		return m.login.View()
	case screenVersion:
		return m.version.View()
	case screenLoading:
		return m.loading.View()
	case screenReview:
		return m.review.View()
	case screenProgress:
		return m.progress.View()
	}
	return ""
}

type commitMessageReadyMsg struct {
	diff    string
	message string
}

type errMsg struct {
	err string
}

type execStepResult struct {
	step int
	err  error
}
