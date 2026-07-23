package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lutrarutra/lazypush/internal/version"
)

type versionScreenModel struct {
	currentTag    string
	choices       []string
	selected      int
	customInput   textinput.Model
	showCustom    bool
	confirmLLM    bool
	done          bool
	chosenVersion string
	useLLM        bool
}

func newVersionScreen(currentTag string) versionScreenModel {
	v, err := version.Parse(currentTag)
	var base version.Version
	if err == nil {
		base = v
	}

	choices := []string{
		fmt.Sprintf("Keep (%s)", currentTag),
		fmt.Sprintf("Bump Patch (%s)", base.Bump(version.Patch).String()),
		fmt.Sprintf("Bump Minor (%s)", base.Bump(version.Minor).String()),
		fmt.Sprintf("Bump Major (%s)", base.Bump(version.Major).String()),
		"Custom",
	}

	ci := textinput.New()
	ci.Placeholder = "v0.0.0"
	ci.Prompt = "Custom version: "

	return versionScreenModel{
		currentTag:  currentTag,
		choices:     choices,
		selected:    1,
		customInput: ci,
		showCustom:  false,
		confirmLLM:  false,
		useLLM:      true,
	}
}

func (m versionScreenModel) Init() tea.Cmd {
	return nil
}

func (m versionScreenModel) Update(msg tea.Msg) (versionScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showCustom {
			switch msg.String() {
			case "enter":
				m.chosenVersion = m.customInput.Value()
				m.useLLM = true
				m.showCustom = false
				m.confirmLLM = true
				return m, nil
			case "esc":
				m.showCustom = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.customInput, cmd = m.customInput.Update(msg)
				return m, cmd
			}
		}

		if m.confirmLLM {
			switch msg.String() {
			case "y", "Y":
				m.useLLM = true
				m.done = true
				return m, nil
			case "n", "N":
				m.useLLM = false
				m.done = true
				return m, nil
			case "esc":
				m.confirmLLM = false
				return m, nil
			}
			return m, nil
		}

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
			if m.selected == len(m.choices)-1 {
				m.showCustom = true
				m.customInput.Focus()
				return m, nil
			}
			m.chosenVersion = m.currentTag
			switch m.selected {
			case 1:
				v, _ := version.Parse(m.currentTag)
				m.chosenVersion = v.Bump(version.Patch).String()
			case 2:
				v, _ := version.Parse(m.currentTag)
				m.chosenVersion = v.Bump(version.Minor).String()
			case 3:
				v, _ := version.Parse(m.currentTag)
				m.chosenVersion = v.Bump(version.Major).String()
			}
			m.confirmLLM = true
			return m, nil
		case "esc":
			m.chosenVersion = ""
			m.done = true
			return m, nil
		}
	}

	return m, nil
}

func (m versionScreenModel) View() string {
	if m.currentTag == "" {
		m.currentTag = "v0.0.0"
	}

	if m.confirmLLM {
		var s string
		s += lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("🏷️  Version: %s\n\n", m.chosenVersion))
		s += "Generate commit message with LLM? (Y/n)  \n"
		s += "\nEsc to go back\n"
		return s
	}

	var s string
	s += lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("🏷️  Current tag: %s\n\n", m.currentTag))

	if m.showCustom {
		s += m.customInput.View()
		s += "\n\nEnter to confirm, Esc to go back\n"
		return s
	}

	for i, choice := range m.choices {
		cursor := " "
		if i == m.selected {
			cursor = "▸"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}

	s += "\n↑/↓ to navigate, Enter to select, Esc to cancel\n"
	return s
}
