package runtui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	invoke "github.com/darkokos/pi-agent-framework/pi-agent-invoke"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

const pollInterval = 3 * time.Second

type screen int

const (
	screenAgent screen = iota
	screenVersion
	screenTask
	screenConfirm
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	helpStyle     = dimStyle
)

type Model struct {
	root string

	agents []*registry.Agent
	tasks  []*registry.Task

	screen screen

	agentsCursor   int
	versionsCursor int
	tasksCursor    int

	selectedAgent   *registry.Agent
	selectedVersion string
	selectedTask    string

	invocationLine string
	invocationView viewport.Model

	width  int
	height int

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

type execFinishedMsg struct{ err error }

func NewModel() Model {
	return Model{loading: true, invocationView: viewport.New(80, 10)}
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
	m.tasksCursor = clampIdx(m.tasksCursor, len(m.tasks))
	if m.selectedAgent != nil {
		m.versionsCursor = clampIdx(m.versionsCursor, len(m.selectedAgent.Versions))
	}
}

func (m *Model) layoutInvocation() {
	width := m.width
	if width <= 0 {
		width = 80
	}

	height := max(3, m.height-7)

	m.invocationView.Width = width
	m.invocationView.Height = height
	m.invocationView.SetContent(lipgloss.NewStyle().Width(width).Render(m.invocationLine))
}

func (m Model) buildInvocation() (string, error) {
	version, err := registry.GetAgentVersion(m.root, m.selectedAgent.ID, m.selectedVersion)
	if err != nil {
		return "", err
	}

	systemPrompt, err := registry.ReadVersionSystemPrompt(m.root, m.selectedAgent.ID, m.selectedVersion)
	if err != nil {
		return "", err
	}

	taskPromptPath := registry.TaskPromptPath(m.root, m.selectedTask)

	return invoke.BuildInvocation(m.selectedAgent.ID, m.selectedVersion, m.selectedTask, systemPrompt, taskPromptPath, version)
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

			m.selectedVersion = m.selectedAgent.Versions[m.versionsCursor]
			m.tasksCursor = 0
			m.err = nil
			m.screen = screenTask
		}
	}

	return m, nil
}

func (m Model) handleTaskKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

	switch msg.String() {
	case "esc", "backspace":
		m.selectedVersion = ""
		m.screen = screenVersion
	case "up", "k":
		if m.tasksCursor > 0 {
			m.tasksCursor--
		}
	case "down", "j":
		if m.tasksCursor < len(m.tasks)-1 {
			m.tasksCursor++
		}
	case "enter":
		if len(m.tasks) > 0 {
			m.selectedTask = m.tasks[m.tasksCursor].ID

			line, err := m.buildInvocation()
			if err != nil {
				m.err = err
				m.selectedTask = ""

				return m, nil
			}

			m.invocationLine = line
			m.err = nil
			m.screen = screenConfirm
			m.layoutInvocation()
			m.invocationView.GotoTop()
		}
	}

	return m, nil
}

func runInvocation(line string) tea.Cmd {
	c := exec.Command("sh", "-c", line)

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execFinishedMsg{err: err}
	})
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		return m, runInvocation(m.invocationLine)
	case "n", "esc":
		m.selectedTask = ""
		m.invocationLine = ""
		m.screen = screenTask

		return m, nil
	}

	var cmd tea.Cmd
	m.invocationView, cmd = m.invocationView.Update(msg)

	return m, cmd
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
	case screenTask:
		return m.handleTaskKey(msg)
	case screenConfirm:
		return m.handleConfirmKey(msg)
	}

	return m, nil
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
	case execFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.status = ""
		} else {
			m.err = nil
			m.status = "run finished"
		}

		m.screen = screenTask
		m.invocationLine = ""
		m.selectedTask = ""

		return m, loadData()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.screen == screenConfirm {
			m.layoutInvocation()
		}

		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func writeRow(b *strings.Builder, line string, selected bool) {
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
		b.WriteString(dimStyle.Render("no agents registered yet — add one from the Registry pane"))
		b.WriteString("\n")

		return
	}

	for i, a := range m.agents {
		line := fmt.Sprintf("%s  %s  (%d version(s))", a.ID, a.Name, len(a.Versions))
		writeRow(b, line, i == m.agentsCursor)
	}
}

func (m Model) viewVersion(b *strings.Builder) {
	fmt.Fprintf(b, "Agent: %s\n\n", m.selectedAgent.ID)

	for i, v := range m.selectedAgent.Versions {
		writeRow(b, v, i == m.versionsCursor)
	}
}

func (m Model) viewTask(b *strings.Builder) {
	fmt.Fprintf(b, "Agent version: %s@%s\n\n", m.selectedAgent.ID, m.selectedVersion)

	if len(m.tasks) == 0 {
		b.WriteString(dimStyle.Render("no tasks registered yet — add one from the Registry pane"))
		b.WriteString("\n")

		return
	}

	for i, t := range m.tasks {
		line := fmt.Sprintf("%s  %s", t.ID, t.PromptFile)
		writeRow(b, line, i == m.tasksCursor)
	}
}

func (m Model) viewConfirm(b *strings.Builder) {
	b.WriteString("about to run:\n\n")
	b.WriteString(m.invocationView.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("y confirm  n/esc cancel  j/k scroll"))
}

func (m Model) helpText() string {
	switch m.screen {
	case screenVersion:
		return "j/k move  enter pick version  esc back  q quit"
	case screenTask:
		return "j/k move  enter pick task  esc back  q quit"
	default:
		return "j/k move  enter pick agent  q quit"
	}
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Pi Agent Lab — Run"))
	if m.root != "" {
		b.WriteString(dimStyle.Render(" - " + m.root))
	}

	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("error: %v", m.err)))
		b.WriteString("\n\n")
	}
	if m.status != "" {
		b.WriteString(okStyle.Render(m.status))
		b.WriteString("\n\n")
	}

	if m.screen == screenConfirm {
		m.viewConfirm(&b)

		return b.String()
	}

	if m.loading && len(m.agents) == 0 {
		b.WriteString("loading...\n")

		return b.String()
	}

	switch m.screen {
	case screenVersion:
		m.viewVersion(&b)
	case screenTask:
		m.viewTask(&b)
	default:
		m.viewAgent(&b)
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.helpText()))

	return b.String()
}
