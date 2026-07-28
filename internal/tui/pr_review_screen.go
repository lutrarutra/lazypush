package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type prReviewScreenModel struct {
	description textarea.Model
	confirmed   bool
	cancelled   bool
}

func newPRReviewScreen(description string) prReviewScreenModel {
	ta := textarea.New()
	ta.SetValue(description)
	ta.SetWidth(80)
	ta.SetHeight(15)
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.Focus()

	return prReviewScreenModel{
		description: ta,
	}
}

func (m prReviewScreenModel) Init() tea.Cmd {
	return nil
}

func (m prReviewScreenModel) Update(msg tea.Msg) (prReviewScreenModel, tea.Cmd) {
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
	m.description, cmd = m.description.Update(msg)
	return m, cmd
}

func (m prReviewScreenModel) View() string {
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("141")).Render("🔀  PR Description"))
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("You can edit the PR description before confirming:"))
	s.WriteString("\n\n")
	s.WriteString(m.description.View())
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("Ctrl+s"))
	s.WriteString(lipgloss.NewStyle().Faint(true).Render(" Save PR description"))
	s.WriteString("  ")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Render("Esc"))
	s.WriteString(lipgloss.NewStyle().Faint(true).Render(" Cancel"))
	return s.String()
}
