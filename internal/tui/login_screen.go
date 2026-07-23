package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type loginScreenModel struct {
	inputs  []textinput.Model
	focused int
	err     string
	done    bool
}

func newLoginScreen() loginScreenModel {
	inputs := make([]textinput.Model, 3)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "https://api.openai.com/v1"
	inputs[0].Prompt = "API URL: "
	inputs[0].Focus()

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "gpt-4o-mini"
	inputs[1].Prompt = "Model: "

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "sk-..."
	inputs[2].Prompt = "API Key: "
	inputs[2].EchoMode = textinput.EchoPassword
	inputs[2].EchoCharacter = '•'

	return loginScreenModel{
		inputs:  inputs,
		focused: 0,
	}
}

func (m loginScreenModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m loginScreenModel) Update(msg tea.Msg) (loginScreenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab":
			m.inputs[m.focused].Blur()
			if msg.String() == "tab" {
				m.focused = (m.focused + 1) % len(m.inputs)
			} else {
				m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
			}
			m.inputs[m.focused].Focus()
			return m, nil
		case "enter":
			if m.focused == len(m.inputs)-1 {
				m.done = true
				return m, nil
			}
			m.inputs[m.focused].Blur()
			m.focused++
			m.inputs[m.focused].Focus()
			return m, nil
		case "esc":
			m.err = "cancelled"
			m.done = true
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m loginScreenModel) View() string {
	var s string
	s += lipgloss.NewStyle().Bold(true).Render("⚙️  lazypush Settings\n\n")
	for i := range m.inputs {
		s += m.inputs[i].View()
		s += "\n"
	}
	s += "\nPress Enter to save, Esc to cancel\n"
	if m.err != "" {
		s += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.err)
	}
	return s
}

func (m loginScreenModel) Values() (apiURL, model, apiKey string) {
	return m.inputs[0].Value(), m.inputs[1].Value(), m.inputs[2].Value()
}
