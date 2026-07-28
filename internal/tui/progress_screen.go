package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type progressStep struct {
	label string
	done  bool
	ok    bool
	msg   string
}

type progressScreenModel struct {
	spinner spinner.Model
	steps   []progressStep
	current int
	done    bool
}

func (m progressScreenModel) StepCount() int {
	return len(m.steps)
}

func newProgressScreen(labels []string) progressScreenModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	s.Spinner = spinner.Dot

	steps := make([]progressStep, len(labels))
	for i, l := range labels {
		steps[i] = progressStep{label: l}
	}

	return progressScreenModel{
		spinner: s,
		steps:   steps,
		current: 0,
	}
}

func (m progressScreenModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m progressScreenModel) Update(msg tea.Msg) (progressScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.done {
			// Any key quits when finished
			return m, tea.Quit
		}
	case progressStepDone:
		if msg.index < len(m.steps) {
			m.steps[msg.index].done = true
			m.steps[msg.index].ok = msg.ok
			m.steps[msg.index].msg = msg.message
			if msg.ok {
				m.current = msg.index + 1
			}
		}
		if m.current >= len(m.steps) || !msg.ok {
			m.done = true
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m progressScreenModel) View() string {
	var s strings.Builder
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("🚀 Progress"))
	s.WriteString("\n\n")

	for i, step := range m.steps {
		if step.done {
			if step.ok {
				s.WriteString("✅ ")
			} else {
				s.WriteString("❌ ")
			}
			s.WriteString(step.label)
			if step.msg != "" {
				s.WriteString(" ")
				s.WriteString(step.msg)
			}
			s.WriteString("\n")
		} else if i == m.current {
			s.WriteString(m.spinner.View())
			s.WriteString(" ")
			s.WriteString(step.label)
			s.WriteString("\n")
		} else {
			s.WriteString("  ")
			s.WriteString(step.label)
			s.WriteString("\n")
		}
	}

	if m.done {
		s.WriteString("\n")
		s.WriteString(lipgloss.NewStyle().Faint(true).Render("Press any key to close"))
	}

	return s.String()
}

type progressStepDone struct {
	index   int
	ok      bool
	message string
}

func ProgressDone(index int, ok bool, message string) tea.Cmd {
	return func() tea.Msg {
		return progressStepDone{index: index, ok: ok, message: message}
	}
}
