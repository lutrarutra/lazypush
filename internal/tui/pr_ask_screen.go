package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type prChoice int

const (
	prChoiceCreate prChoice = iota
	prChoiceBranchOut
	prChoiceCommitHere
)

type prAskScreenModel struct {
	choices       []string
	currentBranch string
	selected      int
	choice        prChoice
	confirmed     bool
	cancelled     bool
}

func newPRAskScreen(currentBranch string, hasDiff bool) prAskScreenModel {
	choices := []string{
		"Create PR to another branch",
	}
	if hasDiff {
		choices = append(choices, "Push to new branch", fmt.Sprintf("Commit to %s", currentBranch))
	}
	if currentBranch == "main" || currentBranch == "master" {
		if !hasDiff {
			return prAskScreenModel{
				choices:       choices,
				selected:      0,
				currentBranch: currentBranch,
			}
		}
		return prAskScreenModel{
			choices:       choices[1:],
			selected:      1,
			currentBranch: currentBranch,
		}
	}
	return prAskScreenModel{
		choices:       choices,
		selected:      len(choices) - 1,
		currentBranch: currentBranch,
	}
}

func (m prAskScreenModel) Init() tea.Cmd {
	return nil
}

func (m prAskScreenModel) Update(msg tea.Msg) (prAskScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.choices)-1 {
				m.selected++
			}
		case "enter":
			if len(m.choices) == 1 {
				m.choice = prChoiceCreate
				m.confirmed = true
				return m, nil
			}
			// Map selection back to actual choice
			// We use the original order: Create(0), BranchOut(1), CommitHere(2)
			// If on main and choices is [BranchOut, CommitHere]:
			//   selected 0 → BranchOut(1), selected 1 → CommitHere(2)
			if len(m.choices) == 2 {
				// on main — choices are [BranchOut, CommitHere]
				switch m.selected {
				case 0:
					m.choice = prChoiceBranchOut
				case 1:
					m.choice = prChoiceCommitHere
				}
			} else {
				switch m.selected {
				case 0:
					m.choice = prChoiceCreate
				case 1:
					m.choice = prChoiceBranchOut
				case 2:
					m.choice = prChoiceCommitHere
				}
			}
			m.confirmed = true
			return m, nil
		case "esc":
			m.cancelled = true
			return m, nil
		}
	}
	return m, nil
}

func (m prAskScreenModel) View() string {
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("🔀  Next Step"))
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("What would you like to do?"))
	s.WriteString("\n\n")

	for i, choice := range m.choices {
		if i == m.selected {
			s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(fmt.Sprintf("> %s", choice)))
		} else {
			s.WriteString(fmt.Sprintf("  %s", choice))
		}
		s.WriteString("\n")
	}
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("  ↑/↓ navigate  Enter select  Esc back"))
	return s.String()
}
