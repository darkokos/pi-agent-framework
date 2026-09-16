package menutui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	helpStyle     = dimStyle
)

type item struct {
	label      string
	entrypoint string
}

type Model struct {
	cursor int
	status string
	err    error
}

type openResultMsg struct {
	label string
	err   error
}

var items = []item{
	{"Compare", "compare"},
	{"Registry", "registry"},
	{"Config", "config"},
}

func NewModel() Model { return Model{} }

func (m Model) Init() tea.Cmd { return nil }

func openPane(it item) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("herdr", "plugin", "pane", "open", "--plugin", "darkokos.pi-agent-lab", "--entrypoint", it.entrypoint)
		if err := cmd.Run(); err != nil {
			return openResultMsg{label: it.label, err: err}
		}

		return openResultMsg{label: it.label}
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "enter":
		m.status = ""
		m.err = nil

		return m, openPane(items[m.cursor])
	}

	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case openResultMsg:
		m.err = msg.err
		if msg.err == nil {
			m.status = "opened " + msg.label
		} else {
			m.status = ""
		}

		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Pi Agent Lab"))
	b.WriteString("\n\n")

	for i, it := range items {
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + it.label))
		} else {
			b.WriteString("  ")
			b.WriteString(it.label)
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.err != nil {
		b.WriteString(errStyle.Render("error: " + m.err.Error()))
		b.WriteString("\n\n")
	}
	if m.status != "" {
		b.WriteString(okStyle.Render(m.status))
		b.WriteString("\n\n")
	}

	b.WriteString(helpStyle.Render("j/k move  enter open  q quit"))

	return b.String()
}
