package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type loadingScreenModel struct {
	spinner spinner.Model
	label   string
	step    int
	maxStep int
}

func newLoadingScreen(label string) loadingScreenModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	s.Spinner = spinner.Dot

	return loadingScreenModel{
		spinner: s,
		label:   label,
	}
}

func (m *loadingScreenModel) SetProgress(step, maxStep int) {
	m.step = step
	m.maxStep = maxStep
}

func (m loadingScreenModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m loadingScreenModel) Update(msg tea.Msg) (loadingScreenModel, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m loadingScreenModel) View() string {
	progress := ""
	if m.maxStep > 0 {
		progress = fmt.Sprintf("  (step %d/%d)", m.step, m.maxStep)
	}
	return "\n" + lipgloss.NewStyle().Bold(true).Render("  "+m.spinner.View()+" "+m.label+progress) + "\n"
}
