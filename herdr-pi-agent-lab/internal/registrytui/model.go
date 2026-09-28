package registrytui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/runflow"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

const pollInterval = 3 * time.Second

type topView int

const (
	topAgents topView = iota
	topTasks
)

type screen int

const (
	screenList screen = iota
	screenAgentDetail
	screenVersionDetail
	screenTaskDetail
	screenForm
	screenConfirm
	screenRun
)

type confirmTarget int

const (
	confirmNone confirmTarget = iota
	confirmDeleteAgent
	confirmDeleteVersion
	confirmDeleteTask
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

	top    topView
	screen screen

	agentsCursor   int
	tasksCursor    int
	versionsCursor int

	selectedAgent   *registry.Agent
	selectedVersion *registry.AgentVersion
	selectedTask    *registry.Task

	prevScreen screen
	form       *form

	confirmMsg    string
	confirmAction func() error
	confirmTarget confirmTarget

	flow   *runflow.Flow
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
	m.tasksCursor = clampIdx(m.tasksCursor, len(m.tasks))
	if m.selectedAgent != nil {
		m.versionsCursor = clampIdx(m.versionsCursor, len(m.selectedAgent.Versions))
	}
}

func findAgent(agents []*registry.Agent, id string) *registry.Agent {
	for _, a := range agents {
		if a.ID == id {
			return a
		}
	}

	return nil
}

func findTask(tasks []*registry.Task, id string) *registry.Task {
	for _, t := range tasks {
		if t.ID == id {
			return t
		}
	}

	return nil
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

	switch msg.String() {
	case "tab":
		if m.top == topAgents {
			m.top = topTasks
		} else {
			m.top = topAgents
		}
	case "up", "k":
		if m.top == topAgents && m.agentsCursor > 0 {
			m.agentsCursor--
		}
		if m.top == topTasks && m.tasksCursor > 0 {
			m.tasksCursor--
		}
	case "down", "j":
		if m.top == topAgents && m.agentsCursor < len(m.agents)-1 {
			m.agentsCursor++
		}
		if m.top == topTasks && m.tasksCursor < len(m.tasks)-1 {
			m.tasksCursor++
		}
	case "enter":
		if m.top == topAgents && len(m.agents) > 0 {
			m.selectedAgent = m.agents[m.agentsCursor]
			m.versionsCursor = 0
			m.screen = screenAgentDetail
		}
		if m.top == topTasks && len(m.tasks) > 0 {
			m.selectedTask = m.tasks[m.tasksCursor]
			m.screen = screenTaskDetail
		}
	case "n":
		m.prevScreen = screenList
		if m.top == topAgents {
			m.form = newAgentCreateForm(m.root)
		} else {
			m.form = newTaskCreateForm(m.root)
		}

		m.screen = screenForm
	}

	return m, nil
}

func (m Model) handleAgentDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

	switch msg.String() {
	case "esc", "backspace":
		m.selectedAgent = nil
		m.screen = screenList
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
			versionID := m.selectedAgent.Versions[m.versionsCursor]
			v, err := registry.GetAgentVersion(m.root, m.selectedAgent.ID, versionID)
			if err != nil {
				m.err = err

				return m, nil
			}

			m.selectedVersion = v
			m.screen = screenVersionDetail
		}
	case "n":
		m.prevScreen = screenAgentDetail
		m.form = newVersionCreateForm(m.root, m.selectedAgent.ID)
		m.screen = screenForm
	case "e":
		m.prevScreen = screenAgentDetail
		m.form = newAgentUpdateForm(m.root, m.selectedAgent)
		m.screen = screenForm
	case "d":
		m.prevScreen = screenAgentDetail
		agent := m.selectedAgent
		m.confirmMsg = fmt.Sprintf("delete agent %q and all its versions?", agent.ID)
		m.confirmAction = func() error { return registry.DeleteAgent(m.root, agent.ID) }
		m.confirmTarget = confirmDeleteAgent
		m.screen = screenConfirm
	}

	return m, nil
}

func (m Model) handleVersionDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

	switch msg.String() {
	case "esc", "backspace":
		m.selectedVersion = nil
		m.screen = screenAgentDetail
	case "r":
		return m.startRun(m.selectedAgent.ID, m.selectedVersion.VersionID, screenVersionDetail)
	case "e":
		m.prevScreen = screenVersionDetail
		m.form = newVersionUpdateForm(m.root, m.selectedAgent.ID, m.selectedVersion)
		m.screen = screenForm
	case "d":
		m.prevScreen = screenVersionDetail
		agentID, versionID := m.selectedAgent.ID, m.selectedVersion.VersionID
		m.confirmMsg = fmt.Sprintf("delete version %q of agent %q?", versionID, agentID)
		m.confirmAction = func() error { return registry.DeleteAgentVersion(m.root, agentID, versionID) }
		m.confirmTarget = confirmDeleteVersion
		m.screen = screenConfirm
	}

	return m, nil
}

func (m Model) handleTaskDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""

	switch msg.String() {
	case "esc", "backspace":
		m.selectedTask = nil
		m.screen = screenList
	case "r":
		return m.startRunForTask(m.selectedTask.ID, screenTaskDetail)
	case "e":
		m.prevScreen = screenTaskDetail
		m.form = newTaskUpdateForm(m.root, m.selectedTask)
		m.screen = screenForm
	case "d":
		m.prevScreen = screenTaskDetail
		task := m.selectedTask
		m.confirmMsg = fmt.Sprintf("delete task %q?", task.ID)
		m.confirmAction = func() error { return registry.DeleteTask(m.root, task.ID) }
		m.confirmTarget = confirmDeleteTask
		m.screen = screenConfirm
	}

	return m, nil
}

func (m Model) startRun(agentID, versionID string, back screen) (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		m.err = fmt.Errorf("no tasks registered yet - add one from the Tasks tab first")

		return m, nil
	}

	return m.openFlow(runflow.New(m.root, agentID, versionID, m.tasks, m.width, m.height), back)
}

func (m Model) startRunForTask(taskID string, back screen) (tea.Model, tea.Cmd) {
	f := runflow.NewForTask(m.root, taskID, m.agents, m.width, m.height)
	if !f.HasChoices() {
		m.err = fmt.Errorf("no agent versions registered yet - add one from the Agents tab first")

		return m, nil
	}

	return m.openFlow(f, back)
}

func (m Model) openFlow(f runflow.Flow, back screen) (tea.Model, tea.Cmd) {
	m.flow = &f
	m.prevScreen = back
	m.err = nil
	m.screen = screenRun

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
		m.screen = m.prevScreen
	case runflow.Finished:
		m.flow = nil
		m.screen = m.prevScreen

		if f.RunErr != nil {
			m.err = f.RunErr
		} else {
			m.status = "run finished"
		}

		return m, tea.Batch(cmd, loadData())
	}

	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.screen == screenForm {
		cmd, submitted, cancelled := m.form.handleKey(msg)
		if cancelled {
			m.screen = m.prevScreen
			m.form.close()
			m.form = nil

			return m, nil
		}
		if submitted {
			m.screen = m.prevScreen
			m.form.close()
			m.form = nil
			m.status = "saved"

			return m, loadData()
		}

		return m, cmd
	}

	if m.screen == screenConfirm {
		switch msg.String() {
		case "y", "enter":
			err := m.confirmAction()
			m.confirmAction = nil
			target := m.confirmTarget
			m.confirmTarget = confirmNone

			if err != nil {
				m.err = err
				m.screen = m.prevScreen

				return m, nil
			}

			m.status = "deleted"
			switch target {
			case confirmDeleteAgent:
				m.selectedAgent = nil
				m.screen = screenList
			case confirmDeleteVersion:
				m.selectedVersion = nil
				m.screen = screenAgentDetail
			case confirmDeleteTask:
				m.selectedTask = nil
				m.screen = screenList
			default:
				m.screen = m.prevScreen
			}

			return m, loadData()
		case "n", "esc":
			m.confirmTarget = confirmNone
			m.screen = m.prevScreen
			m.confirmAction = nil

			return m, nil
		}

		return m, nil
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	switch m.screen {
	case screenList:
		return m.handleListKey(msg)
	case screenAgentDetail:
		return m.handleAgentDetailKey(msg)
	case screenVersionDetail:
		return m.handleVersionDetailKey(msg)
	case screenTaskDetail:
		return m.handleTaskDetailKey(msg)
	}

	return m, nil
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
			id := m.selectedAgent.ID
			m.selectedAgent = findAgent(m.agents, id)
		}
		if m.selectedTask != nil {
			id := m.selectedTask.ID
			m.selectedTask = findTask(m.tasks, id)
		}

		m.clamp()

		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err

		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		return m, nil
	case editorDoneMsg:
		if m.form != nil {
			m.form.handleEditorDone(msg)
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

func (m Model) viewAgentDetail(b *strings.Builder) {
	a := m.selectedAgent
	b.WriteString(headerStyle.Render("Agent " + a.ID))
	b.WriteString("\n")
	fmt.Fprintf(b, "ID:           %s\n", a.ID)
	fmt.Fprintf(b, "Name:         %s\n", a.Name)
	fmt.Fprintf(b, "Description:  %s\n\n", a.Description)
	b.WriteString("Versions:\n")

	if len(a.Versions) == 0 {
		b.WriteString(dimStyle.Render("no versions yet — press n to create one"))
		b.WriteString("\n")

		return
	}

	for i, v := range a.Versions {
		writeRow(b, v, i == m.versionsCursor)
	}
}

func (m Model) viewVersionDetail(b *strings.Builder) {
	v := m.selectedVersion
	fmt.Fprintf(b, "Agent ID:       %s\n", v.AgentID)
	fmt.Fprintf(b, "Version ID:     %s\n", v.VersionID)
	fmt.Fprintf(b, "Model:          %s\n", v.Model)
	fmt.Fprintf(b, "Provider:       %s\n", v.Provider)
	fmt.Fprintf(b, "Thinking:       %s\n", v.Thinking)
	fmt.Fprintf(b, "System Prompt:  %s (%s)\n", v.SystemPrompt.File, v.SystemPrompt.Mode)
	fmt.Fprintf(b, "Tools:          %s %s\n", v.Tools.Mode, strings.Join(v.Tools.Names, ","))
	fmt.Fprintf(b, "Extensions:     isolate=%t %s\n", v.Extensions.Isolate, strings.Join(v.Extensions.Paths, ","))
	fmt.Fprintf(b, "Skills:         isolate=%t %s\n", v.Skills.Isolate, strings.Join(v.Skills.Paths, ","))
	fmt.Fprintf(b, "Notes:          %s\n", v.Notes)
}

func (m Model) viewTaskDetail(b *strings.Builder) {
	t := m.selectedTask
	fmt.Fprintf(b, "ID:           %s\n", t.ID)
	fmt.Fprintf(b, "Prompt File:  %s\n", t.PromptFile)
	fmt.Fprintf(b, "Notes:        %s\n", t.Notes)
}

func (m Model) helpText() string {
	switch m.screen {
	case screenAgentDetail:
		return "j/k move  enter open version  n new version  e edit agent  d delete agent  esc back  q quit"
	case screenVersionDetail:
		return "r run  e edit  d delete  esc back  q quit"
	case screenTaskDetail:
		return "r run  e edit  d delete  esc back  q quit"
	default:
		return "tab switch agents/tasks  j/k move  enter open  n new  q quit"
	}
}

func (m Model) viewList(b *strings.Builder) {
	if m.top == topAgents {
		b.WriteString(selectedStyle.Render("[ Agents ]"))
		b.WriteString("  ")
		b.WriteString(dimStyle.Render("Tasks"))
	} else {
		b.WriteString(dimStyle.Render("Agents"))
		b.WriteString("  ")
		b.WriteString(selectedStyle.Render("[ Tasks ]"))
	}

	b.WriteString("\n\n")

	if m.top == topAgents {
		if len(m.agents) == 0 {
			b.WriteString(dimStyle.Render("no agents registered yet — press n to create one"))
			b.WriteString("\n")

			return
		}

		for i, a := range m.agents {
			line := fmt.Sprintf("%s  %s  (%d version(s))", a.ID, a.Name, len(a.Versions))
			writeRow(b, line, i == m.agentsCursor)
		}

		return
	}

	if len(m.tasks) == 0 {
		b.WriteString(dimStyle.Render("no tasks registered yet — press n to create one"))
		b.WriteString("\n")

		return
	}

	for i, t := range m.tasks {
		line := fmt.Sprintf("%s  %s", t.ID, t.PromptFile)
		writeRow(b, line, i == m.tasksCursor)
	}
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Pi Agent Lab — Registry"))
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

	if m.screen == screenRun {
		b.WriteString(m.flow.View())

		return b.String()
	}

	if m.screen == screenForm {
		b.WriteString(m.form.view())

		return b.String()
	}

	if m.screen == screenConfirm {
		b.WriteString(m.confirmMsg)
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("y confirm  n/esc cancel"))

		return b.String()
	}

	if m.loading && len(m.agents) == 0 && len(m.tasks) == 0 {
		b.WriteString("loading...\n")

		return b.String()
	}

	switch m.screen {
	case screenAgentDetail:
		m.viewAgentDetail(&b)
	case screenVersionDetail:
		m.viewVersionDetail(&b)
	case screenTaskDetail:
		m.viewTaskDetail(&b)
	default:
		m.viewList(&b)
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.helpText()))

	return b.String()
}
