package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lutrarutra/lazypush/internal/version"
)

var (
	green = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	red   = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	faint = lipgloss.NewStyle().Faint(true)
	bold  = lipgloss.NewStyle().Bold(true)
)

type versionScreenModel struct {
	currentTag    string
	choices       []string
	labels        []string
	bodies        []string
	selected      int
	customInput   textinput.Model
	showCustom    bool
	confirmLLM    bool
	confirmReTag  bool
	reTag         bool
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

	labels := []string{"Keep", "Patch", "Minor", "Major", "Custom"}
	bodies := []string{
		currentTag,
		base.Bump(version.Patch).String(),
		base.Bump(version.Minor).String(),
		base.Bump(version.Major).String(),
		"",
	}

	choices := make([]string, len(labels))
	copy(choices, labels)

	ci := textinput.New()
	ci.Placeholder = "v0.0.0"
	ci.Prompt = "Custom version: "

	return versionScreenModel{
		currentTag:  currentTag,
		choices:     choices,
		labels:      labels,
		bodies:      bodies,
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

		if m.confirmReTag {
			switch msg.String() {
			case "y", "Y":
				m.reTag = true
				m.confirmReTag = false
				m.confirmLLM = true
				return m, nil
			case "n", "N":
				m.reTag = false
				m.confirmReTag = false
				m.confirmLLM = true
				return m, nil
			case "esc":
				m.confirmReTag = false
				return m, nil
			}
			return m, nil
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
			m.reTag = false
			switch m.selected {
			case 0:
				// Keep — ask about moving the tag
				m.confirmReTag = true
				return m, nil
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
			if m.selected != 0 {
				m.confirmLLM = true
			}
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

	if m.confirmReTag {
		var s strings.Builder
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")).Render("⚠️  Tag Already Exists"))
		s.WriteString("\n\n")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("Version: "))
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(m.chosenVersion))
		s.WriteString("\n\n")
		s.WriteString("This tag already exists. ")
		s.WriteString(bold.Render("Move it to the current commit?"))
		s.WriteString("\n\n")
		s.WriteString(green.Render("  y"))
		s.WriteString("  Yes, move the tag\n")
		s.WriteString(red.Render("  n"))
		s.WriteString("  No, keep it where it is\n")
		s.WriteString("\n")
		s.WriteString(faint.Render("  Esc to go back"))
		return s.String()
	}

	if m.confirmLLM {
		var s strings.Builder
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("🤖  Generate Commit Message"))
		s.WriteString("\n\n")
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("Version: "))
		s.WriteString(bold.Render(m.chosenVersion))
		s.WriteString("\n\n")
		s.WriteString("Use an LLM to generate the commit message from the diff?")
		s.WriteString("\n\n")
		s.WriteString(green.Render("  Y"))
		s.WriteString("  Yes, generate with AI\n")
		s.WriteString(red.Render("  n"))
		s.WriteString("  No, write it manually\n")
		s.WriteString("\n")
		s.WriteString(faint.Render("  Esc to go back"))
		return s.String()
	}

	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("🏷️  Select Tag (version)"))
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("Current: %s\n\n", m.currentTag)))

	if m.showCustom {
		s.WriteString(m.customInput.View())
		s.WriteString("\n\n")
		s.WriteString(green.Render("  Enter"))
		s.WriteString(" confirm   ")
		s.WriteString(red.Render("Esc"))
		s.WriteString(" go back\n")
		return s.String()
	}

	labelColors := []lipgloss.Color{
		lipgloss.Color("15"),  // keep   — white
		lipgloss.Color("114"), // patch  — green
		lipgloss.Color("220"), // minor  — yellow
		lipgloss.Color("204"), // major  — red
		lipgloss.Color("141"), // custom — purple
	}
	descriptions := []string{
		"no version change",
		"bug fixes / small changes",
		"new features (backward-compat)",
		"breaking changes",
		"enter a custom version",
	}

	// s.WriteString(faint.Render("Select a version bump:\n\n"))

	for i := range m.labels {
		// Cursor (col 0-1) — use > which is exactly 1 cell wide
		if i == m.selected {
			s.WriteString("> ")
		} else {
			s.WriteString("  ")
		}

		// Build un-styled row text, then style the whole line
		body := m.bodies[i]
		if body != "" {
			body = "> " + body
		}
		row := fmt.Sprintf("%-7s  %-16s  . %s", m.labels[i], body, descriptions[i])

		if i == m.selected {
			s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(row))
		} else {
			s.WriteString(lipgloss.NewStyle().Foreground(labelColors[i]).Render(row))
		}
		s.WriteString("\n")
	}
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("  ↑/↓ navigate • Enter select • Esc quit"))
	return s.String()
}
