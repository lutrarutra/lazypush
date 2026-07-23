package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type reviewScreenModel struct {
	diff          string
	commitMessage textarea.Model
	viewport      viewport.Model
	showPR        bool
	prDescription textarea.Model
	confirmed     bool
	cancelled     bool
	includePR     bool
}

func newReviewScreen(diff, commitMessage string) reviewScreenModel {
	vp := viewport.New(80, 20)
	vp.SetContent(diff)

	ta := textarea.New()
	ta.SetValue(commitMessage)
	ta.SetWidth(80)
	ta.SetHeight(5)
	ta.Prompt = "Commit message: "
	ta.Focus()

	pr := textarea.New()
	pr.SetValue(commitMessage)
	pr.SetWidth(80)
	pr.SetHeight(8)
	pr.Prompt = "PR description: "

	return reviewScreenModel{
		diff:          diff,
		commitMessage: ta,
		viewport:      vp,
		prDescription: pr,
	}
}

func (m reviewScreenModel) Init() tea.Cmd {
	return nil
}

func (m reviewScreenModel) Update(msg tea.Msg) (reviewScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			m.confirmed = true
			m.includePR = false
			return m, nil
		case "ctrl+p":
			m.confirmed = true
			m.includePR = true
			return m, nil
		case "ctrl+c":
			m.cancelled = true
			return m, nil
		case "ctrl+e":
			m.showPR = !m.showPR
			return m, nil
		case "tab":
			if m.showPR {
				if m.commitMessage.Focused() {
					m.commitMessage.Blur()
					m.prDescription.Focus()
				} else {
					m.prDescription.Blur()
					m.commitMessage.Focus()
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.commitMessage.Focused() {
		m.commitMessage, cmd = m.commitMessage.Update(msg)
	} else if m.showPR && m.prDescription.Focused() {
		m.prDescription, cmd = m.prDescription.Update(msg)
	}
	return m, cmd
}

func (m reviewScreenModel) View() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📝 Review Changes\n\n"))

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Diff:\n"))
	s.WriteString(m.viewport.View())
	s.WriteString("\n\n")

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Commit Message:\n"))
	s.WriteString(m.commitMessage.View())
	s.WriteString("\n")

	if m.showPR {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render("PR Description:\n"))
		s.WriteString(m.prDescription.View())
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("Ctrl+s: Commit  Ctrl+p: Commit + PR  Ctrl+e: Toggle PR  Ctrl+c: Cancel  Tab: switch fields\n"))

	return s.String()
}
