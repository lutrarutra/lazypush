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
	confirmed     bool
	cancelled     bool
	noLLM         bool
}

func newReviewScreen(diff, commitMessage string) reviewScreenModel {
	return newReviewScreenWithLLM(diff, commitMessage, false)
}

func newReviewScreenWithLLM(diff, commitMessage string, noLLM bool) reviewScreenModel {
	vp := viewport.New(80, 10)
	if diff == "" {
		vp.SetContent(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("(no changes detected)"))
	} else {
		vp.SetContent(diff)
	}

	ta := textarea.New()
	ta.SetValue(commitMessage)
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.Focus()

	return reviewScreenModel{
		diff:          diff,
		commitMessage: ta,
		viewport:      vp,
		noLLM:         noLLM,
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
			return m, nil
		case "esc":
			m.cancelled = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	if m.commitMessage.Focused() {
		m.commitMessage, cmd = m.commitMessage.Update(msg)
	}
	return m, cmd
}

func (m reviewScreenModel) View() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("📝 Review Changes"))
	s.WriteString("\n\n")

	if m.noLLM {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("  ⚠ No AI provider configured — write your commit message manually"))
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("240")).Render("     Run 'lazypush login' to set up an AI provider"))
		s.WriteString("\n\n")
	}

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Diff:"))
	s.WriteString("\n")
	s.WriteString(m.viewport.View())
	s.WriteString("\n\n")

	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Commit Message:"))
	s.WriteString("\n")
	s.WriteString(m.commitMessage.View())
	s.WriteString("\n")

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("Ctrl+s"))
	s.WriteString(lipgloss.NewStyle().Faint(true).Render(" Save commit message"))
	s.WriteString("  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Render("Esc"))
	s.WriteString(lipgloss.NewStyle().Faint(true).Render(" Cancel"))

	return s.String()
}
