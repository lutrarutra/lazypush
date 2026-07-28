package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type branchSelectScreenModel struct {
	branches      []string
	currentBranch string
	selected      int
	chosen        bool
	cancelled     bool
}

func newBranchSelectScreen(branches []string, currentBranch string) branchSelectScreenModel {
	sel := 0
	for i, b := range branches {
		if b == "main" || b == "master" {
			sel = i
			break
		}
	}
	return branchSelectScreenModel{
		branches:      branches,
		currentBranch: currentBranch,
		selected:      sel,
	}
}

func (m branchSelectScreenModel) Init() tea.Cmd {
	return nil
}

func (m branchSelectScreenModel) Update(msg tea.Msg) (branchSelectScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.branches)-1 {
				m.selected++
			}
		case "enter":
			m.chosen = true
			return m, nil
		case "esc":
			m.cancelled = true
			return m, nil
		}
	}
	return m, nil
}

func (m branchSelectScreenModel) View() string {
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("🎯  Select Target Branch"))
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("Current branch: %s", m.currentBranch)))
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("Choose the branch to PR into:"))
	s.WriteString("\n\n")

	for i, branch := range m.branches {
		if i == m.selected {
			s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(fmt.Sprintf("> %s  (default)", branch)))
		} else {
			s.WriteString(fmt.Sprintf("  %s", branch))
		}
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("  ↑/↓ navigate  Enter select  Esc back"))
	return s.String()
}
