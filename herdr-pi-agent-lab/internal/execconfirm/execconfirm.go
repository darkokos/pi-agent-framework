package execconfirm

import (
	"os/exec"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Action int

const (
	None Action = iota
	Run
	Cancel
)

const reservedLines = 7

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

type Model struct {
	line string
	view viewport.Model
}

func New(line string, width, height int) Model {
	m := Model{line: line, view: viewport.New(80, 10)}
	m.Resize(width, height)

	return m
}

func (m *Model) Resize(width, height int) {
	if width <= 0 {
		width = 80
	}

	m.view.Width = width
	m.view.Height = max(3, height-reservedLines)
	m.view.SetContent(lipgloss.NewStyle().Width(width).Render(m.line))
}

func (m Model) Line() string { return m.line }

func (m Model) Update(msg tea.KeyMsg) (Model, tea.Cmd, Action) {
	switch msg.String() {
	case "y", "enter":
		return m, nil, Run
	case "n", "esc":
		return m, nil, Cancel
	}

	var cmd tea.Cmd
	m.view, cmd = m.view.Update(msg)

	return m, cmd, None
}

func (m Model) View() string {
	return "about to run:\n\n" + m.view.View() + "\n\n" + helpStyle.Render("y confirm  n/esc cancel  j/k scroll")
}

func Exec(line string, done func(error) tea.Msg) tea.Cmd {
	return tea.ExecProcess(exec.Command("sh", "-c", line), done)
}
