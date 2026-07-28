package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmScreenModel struct {
	versionTag   string
	commitMsg    string
	includePR    bool
	needsTagging bool
	skipNote     string
	confirmed    bool
	cancelled    bool
}

func newConfirmScreen(versionTag, commitMsg string, includePR, needsTagging bool, skipNote string) confirmScreenModel {
	return confirmScreenModel{
		versionTag:   versionTag,
		commitMsg:    commitMsg,
		includePR:    includePR,
		needsTagging: needsTagging,
		skipNote:     skipNote,
	}
}

func (m confirmScreenModel) Init() tea.Cmd {
	return nil
}

func (m confirmScreenModel) Update(msg tea.Msg) (confirmScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.confirmed = true
			return m, nil
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, nil
		}
	}
	return m, nil
}

func (m confirmScreenModel) View() string {
	var s strings.Builder

	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("✅  Confirm Operations"))
	s.WriteString("\n\n")

	// Summary card
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Background(lipgloss.Color("255")).Padding(0, 2).Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render("Review your changes before proceeding"),
	))
	s.WriteString("\n\n")

	// Version tag
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("🏷️  Tag: "))
	s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(m.versionTag))
	s.WriteString("\n")

	// Commit message (preview — first 3 lines)
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("💬  Message:"))
	s.WriteString("\n")
	lines := strings.Split(m.commitMsg, "\n")
	previewLines := lines
	if len(previewLines) > 3 {
		previewLines = previewLines[:3]
	}
	for _, line := range previewLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("255")).PaddingLeft(4).Render(line))
		s.WriteString("\n")
	}
	if len(lines) > 3 {
		s.WriteString(lipgloss.NewStyle().Italic(true).Faint(true).PaddingLeft(4).Render(fmt.Sprintf("... and %d more lines", len(lines)-3)))
		s.WriteString("\n")
	}
	s.WriteString("\n")

	// Operations
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("⚡  Operations:"))
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("  ● Commit"))
	s.WriteString("\n")
	if m.needsTagging {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render("  ● Tag "))
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(m.versionTag))
		s.WriteString("\n")
	}
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("  ● Push"))
	s.WriteString("\n")
	if m.includePR {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Render("  ● Create Pull Request"))
		s.WriteString("\n")
	}
	s.WriteString("\n")

	// Optional skip note
	if m.skipNote != "" {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Italic(true).Render("  ⓘ " + m.skipNote))
		s.WriteString("\n\n")
	}

	// Confirm prompt
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render("  Enter"))
	s.WriteString("  Confirm and execute\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("204")).Render("  Esc"))
	s.WriteString("   Cancel\n")

	return s.String()
}
