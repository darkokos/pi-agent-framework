package menutui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/scroll"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

type item struct {
	label       string
	description string
	entrypoint  string
}

type Model struct {
	cursor   int
	width    int
	height   int
	scroller scroll.Model
	status   string
	err      error
}

type openResultMsg struct {
	label string
	err   error
}

var items = []item{
	{"Start a run", "Pick an agent version and a task, then run it", "run"},
	{"Agents & Tasks", "Create and delete your agents, their versions, and tasks", "registry"},
	{"Review Runs", "See how each agent version did on a task, and rate unrated runs", "compare"},
	{"Registries", "Pick the registry these panes work on, or init/add one", "config"},
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
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scroller.Resize(m.width, m.height)

		return m, nil
	case openResultMsg:
		m.err = msg.err
		if msg.err == nil {
			m.status = "opened " + msg.label
		} else {
			m.status = ""
		}

		return m, nil
	case tea.KeyMsg:
		header, body, footer := m.parts()
		if m.scroller.HandleKey(msg.String(), func() string { return header }, func() string { return body }, func() string { return footer }) {
			return m, nil
		}

		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) parts() (header, body, footer string) {
	header = headerStyle.Render("Pi Agent Lab")

	var b strings.Builder

	for i, it := range items {
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + it.label))
		} else {
			b.WriteString("  ")
			b.WriteString(it.label)
		}

		b.WriteString("\n    ")
		descWidth := m.width
		if descWidth > 0 {
			descWidth = max(descWidth-4, 10)
		}
		b.WriteString(theme.Dim.Width(descWidth).Render(it.description))
		b.WriteString("\n\n")
	}

	if m.err != nil {
		b.WriteString(errStyle.Width(m.width).Render("error: " + m.err.Error()))
		b.WriteString("\n\n")
	}
	if m.status != "" {
		b.WriteString(okStyle.Width(m.width).Render(m.status))
		b.WriteString("\n\n")
	}

	return header, b.String(), "j/k move  enter open  q quit  ctrl+u/ctrl+d scroll"
}

func (m Model) View() string {
	header, body, footer := m.parts()

	return m.scroller.RenderWithFooter(header, body, footer)
}
