package runtui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/runflow"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/scroll"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

const pollInterval = 3 * time.Second

type screen int

const (
	screenAgent screen = iota
	screenVersion
	screenRun
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle      = theme.Dim
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	helpStyle     = theme.Help
)

type Model struct {
	root string

	agents []*registry.Agent
	tasks  []*registry.Task

	screen screen

	agentsCursor   int
	versionsCursor int

	selectedAgent *registry.Agent
	flow          *runflow.Flow

	width    int
	height   int
	scroller scroll.Model

	err     error
	status  string
	loading bool
}

type tickMsg struct{}

type dataMsg struct {
	root   string
	agents []*registry.Agent
	tasks  []*registry.Task
}

type errMsg struct{ err error }

func NewModel() Model {
	return Model{loading: true}
}

func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func loadData() tea.Cmd {
	return func() tea.Msg {
		root, err := registry.ResolveRoot()
		if err != nil {
			return errMsg{err}
		}

		agents, err := registry.ListAgents(root)
		if err != nil {
			return errMsg{err}
		}

		tasks, err := registry.ListTasks(root)
		if err != nil {
			return errMsg{err}
		}

		return dataMsg{root: root, agents: agents, tasks: tasks}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadData(), tick())
}

func clampIdx(i, n int) int {
	if n == 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	if i < 0 {
		return 0
	}

	return i
}

func (m *Model) clamp() {
	m.agentsCursor = clampIdx(m.agentsCursor, len(m.agents))
	if m.selectedAgent != nil {
		m.versionsCursor = clampIdx(m.versionsCursor, len(m.selectedAgent.Versions))
	}
}

func (m Model) handleAgentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

	switch msg.String() {
	case "up", "k":
		if m.agentsCursor > 0 {
			m.agentsCursor--
		}
	case "down", "j":
		if m.agentsCursor < len(m.agents)-1 {
			m.agentsCursor++
		}
	case "enter":
		if len(m.agents) > 0 {
			agent := m.agents[m.agentsCursor]
			if len(agent.Versions) == 0 {
				m.err = fmt.Errorf("agent %q has no versions", agent.ID)

				return m, nil
			}

			m.selectedAgent = agent
			m.versionsCursor = 0
			m.err = nil
			m.screen = screenVersion
		}
	}

	return m, nil
}

func (m Model) handleVersionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

	switch msg.String() {
	case "esc", "backspace":
		m.selectedAgent = nil
		m.screen = screenAgent
	case "up", "k":
		if m.versionsCursor > 0 {
			m.versionsCursor--
		}
	case "down", "j":
		if m.versionsCursor < len(m.selectedAgent.Versions)-1 {
			m.versionsCursor++
		}
	case "enter":
		if len(m.selectedAgent.Versions) > 0 {
			if len(m.tasks) == 0 {
				m.err = fmt.Errorf("no tasks registered yet")

				return m, nil
			}

			f := runflow.New(m.root, m.selectedAgent.ID, m.selectedAgent.Versions[m.versionsCursor], m.tasks, m.width, m.height)
			m.flow = &f
			m.err = nil
			m.screen = screenRun
		}
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	switch m.screen {
	case screenAgent:
		return m.handleAgentKey(msg)
	case screenVersion:
		return m.handleVersionKey(msg)
	}

	return m, nil
}

func (m Model) updateFlow(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, isKey := msg.(tea.KeyMsg); isKey {
		m.err, m.status = nil, ""
	}

	f, cmd, outcome := m.flow.Update(msg)
	m.flow = &f

	switch outcome {
	case runflow.Cancelled:
		m.flow = nil
		m.screen = screenVersion
	case runflow.Finished:
		if f.RunErr != nil {
			m.err, m.status = f.RunErr, ""
		} else {
			m.err, m.status = nil, "run finished"
		}

		return m, tea.Batch(cmd, loadData())
	}

	return m, cmd
}

func findAgent(agents []*registry.Agent, id string) *registry.Agent {
	for _, a := range agents {
		if a.ID == id {
			return a
		}
	}

	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.screen == screenRun {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if k := msg.String(); k == "ctrl+c" || k == "q" {
				return m, tea.Quit
			}

			return m.updateFlow(msg)
		case tea.WindowSizeMsg:
			m.width, m.height = msg.Width, msg.Height

			return m.updateFlow(msg)
		case tickMsg, dataMsg, errMsg:
		default:
			return m.updateFlow(msg)
		}
	}

	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(loadData(), tick())
	case dataMsg:
		m.loading = false
		m.err = nil
		m.root = msg.root
		m.agents = msg.agents
		m.tasks = msg.tasks

		if m.selectedAgent != nil {
			m.selectedAgent = findAgent(m.agents, m.selectedAgent.ID)
			if m.selectedAgent == nil {
				m.screen = screenAgent
			}
		}

		m.clamp()

		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err

		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scroller.Resize(m.width, m.height)

		return m, nil
	case tea.KeyMsg:
		if m.scroller.HandleKey(msg.String(), m.content) {
			return m, nil
		}

		prevScreen := m.screen

		newModel, cmd := m.handleKey(msg)
		if nm, ok := newModel.(Model); ok {
			if nm.screen != prevScreen {
				nm.scroller.GotoTop()
			}

			return nm, cmd
		}

		return newModel, cmd
	}

	return m, nil
}

func rowWidth(width int) int {
	if width <= 0 {
		return width
	}

	return max(width-2, 10)
}

func writeRow(b *strings.Builder, line string, selected bool, width int) {
	line = theme.Wrap(rowWidth(width), line)

	if selected {
		b.WriteString(selectedStyle.Render("> " + line))
	} else {
		b.WriteString("  ")
		b.WriteString(line)
	}

	b.WriteString("\n")
}

func (m Model) viewAgent(b *strings.Builder) {
	if len(m.agents) == 0 {
		b.WriteString(dimStyle.Width(m.width).Render("no agents registered yet — add one from Agents & Tasks"))
		b.WriteString("\n")

		return
	}

	for i, a := range m.agents {
		line := fmt.Sprintf("%s  %s  (%d version(s))", a.ID, a.Name, len(a.Versions))
		writeRow(b, line, i == m.agentsCursor, m.width)
	}
}

func (m Model) viewVersion(b *strings.Builder) {
	b.WriteString(theme.Wrap(m.width, fmt.Sprintf("Agent: %s", m.selectedAgent.ID)))
	b.WriteString("\n\n")

	for i, v := range m.selectedAgent.Versions {
		writeRow(b, v, i == m.versionsCursor, m.width)
	}
}

func (m Model) helpText() string {
	switch m.screen {
	case screenVersion:
		return "j/k move  enter pick version  esc back  q quit"
	default:
		return "j/k move  enter pick agent  q quit"
	}
}

func (m Model) content() string {
	var b strings.Builder

	header := headerStyle.Render("Pi Agent Lab — Start a run")
	if m.root != "" {
		header += dimStyle.Render(" - " + m.root)
	}

	b.WriteString(theme.Wrap(m.width, header))
	b.WriteString("\n\n")

	if m.screen == screenRun && m.flow.Confirming() {
		b.WriteString(m.flow.View())

		return b.String()
	}

	if m.err != nil {
		b.WriteString(errStyle.Width(m.width).Render(fmt.Sprintf("error: %v", m.err)))
		b.WriteString("\n\n")
	}
	if m.status != "" {
		b.WriteString(okStyle.Width(m.width).Render(m.status))
		b.WriteString("\n\n")
	}

	if m.root == "" && m.err != nil {
		b.WriteString(helpStyle.Width(m.width).Render("q quit"))

		return b.String()
	}

	if m.loading && len(m.agents) == 0 {
		b.WriteString("loading...\n")

		return b.String()
	}

	if m.screen == screenRun {
		b.WriteString(m.flow.View())

		return b.String()
	}

	switch m.screen {
	case screenVersion:
		m.viewVersion(&b)
	default:
		m.viewAgent(&b)
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Width(m.width).Render(m.helpText() + "  ctrl+u/ctrl+d scroll"))

	return b.String()
}

func (m Model) View() string {
	if m.screen == screenRun {
		return m.content()
	}

	return m.scroller.Render(m.content())
}
