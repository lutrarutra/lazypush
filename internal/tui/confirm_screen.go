package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmScreenModel struct {
	versionTag    string
	oldTag        string
	commitMsg     string
	includePR     bool
	needsTagging  bool
	isBump        bool
	skipNote      string
	currentBranch string
	targetBranch  string
	confirmed     bool
	cancelled     bool
}

func newConfirmScreen(versionTag, oldTag, commitMsg string, includePR, needsTagging, isBump bool, skipNote, currentBranch, targetBranch string) confirmScreenModel {
	return confirmScreenModel{
		versionTag:    versionTag,
		oldTag:        oldTag,
		commitMsg:     commitMsg,
		includePR:     includePR,
		needsTagging:  needsTagging,
		isBump:        isBump,
		skipNote:      skipNote,
		currentBranch: currentBranch,
		targetBranch:  targetBranch,
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
	if m.needsTagging && m.isBump {
		// Version changed: show old → new
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("🏷️  Tag: "))
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(m.oldTag))
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(" → "))
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(m.versionTag))
	} else {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("🏷️  Tag: "))
		s.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render(m.versionTag))
	}
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
	commitLabel := fmt.Sprintf("  ● Commit → %s", m.currentBranch)
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render(commitLabel))
	s.WriteString("\n")
	if m.needsTagging {
		var tagOp string
		if m.isBump {
			tagOp = fmt.Sprintf("  ● Tag %s → %s", m.oldTag, m.versionTag)
		} else {
			tagOp = fmt.Sprintf("  ● Tag %s (re-tag)", m.versionTag)
		}
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(tagOp))
		s.WriteString("\n")
	}
	if m.currentBranch != "" {
		pushLabel := fmt.Sprintf("  ● Push %s → origin/%s", m.currentBranch, m.currentBranch)
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Render(pushLabel))
		s.WriteString("\n")
	}
	if m.includePR && m.targetBranch != "" {
		prLabel := fmt.Sprintf("  ● Create PR %s → %s", m.currentBranch, m.targetBranch)
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Render(prLabel))
		s.WriteString("\n")
	} else if m.includePR && m.targetBranch == "" {
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
