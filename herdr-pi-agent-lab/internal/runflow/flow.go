package runflow

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/execconfirm"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
	invoke "github.com/darkokos/pi-agent-framework/pi-agent-invoke"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

type Outcome int

const (
	Active Outcome = iota
	Cancelled
	Finished
)

var (
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle      = theme.Dim
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type finishedMsg struct{ err error }

type item struct {
	label     string
	detail    string
	agentID   string
	versionID string
	taskID    string
}

type Flow struct {
	root  string
	title string
	empty string
	items []item

	cursor  int
	confirm *execconfirm.Model

	width  int
	height int

	err    error
	RunErr error
}

func New(root, agentID, versionID string, tasks []*registry.Task, width, height int) Flow {
	items := make([]item, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, item{label: t.ID, detail: t.PromptFile, agentID: agentID, versionID: versionID, taskID: t.ID})
	}

	return Flow{
		root: root, items: items, width: width, height: height,
		title: fmt.Sprintf("Run %s@%s against a task:", agentID, versionID),
		empty: "no tasks registered yet",
	}
}

func NewForTask(root, taskID string, agents []*registry.Agent, width, height int) Flow {
	var items []item

	for _, a := range agents {
		for _, v := range a.Versions {
			items = append(items, item{label: a.ID + "@" + v, detail: a.Name, agentID: a.ID, versionID: v, taskID: taskID})
		}
	}

	return Flow{
		root: root, items: items, width: width, height: height,
		title: fmt.Sprintf("Run task %s with an agent version:", taskID),
		empty: "no agent versions registered yet",
	}
}

func (f Flow) invocation(it item) (string, error) {
	version, err := registry.GetAgentVersion(f.root, it.agentID, it.versionID)
	if err != nil {
		return "", err
	}

	systemPrompt, err := registry.ReadVersionSystemPrompt(f.root, it.agentID, it.versionID)
	if err != nil {
		return "", err
	}

	return invoke.BuildInvocation(f.root, it.agentID, it.versionID, it.taskID, systemPrompt, registry.TaskPromptPath(f.root, it.taskID), version)
}

func (f Flow) Update(msg tea.Msg) (Flow, tea.Cmd, Outcome) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width, f.height = msg.Width, msg.Height
		if f.confirm != nil {
			c := *f.confirm
			c.Resize(f.width, f.height)
			f.confirm = &c
		}

		return f, nil, Active
	case finishedMsg:
		f.confirm = nil
		f.RunErr = msg.err

		return f, nil, Finished
	case tea.KeyMsg:
		if f.confirm != nil {
			return f.updateConfirm(msg)
		}

		return f.updateChoices(msg)
	}

	return f, nil, Active
}

func (f Flow) updateConfirm(msg tea.KeyMsg) (Flow, tea.Cmd, Outcome) {
	c, cmd, action := f.confirm.Update(msg)
	f.confirm = &c

	switch action {
	case execconfirm.Run:
		return f, execconfirm.Exec(c.Line(), func(err error) tea.Msg { return finishedMsg{err: err} }), Active
	case execconfirm.Cancel:
		f.confirm = nil
	}

	return f, cmd, Active
}

func (f Flow) updateChoices(msg tea.KeyMsg) (Flow, tea.Cmd, Outcome) {
	switch msg.String() {
	case "esc", "backspace":
		return f, nil, Cancelled
	case "up", "k":
		if f.cursor > 0 {
			f.cursor--
		}
	case "down", "j":
		if f.cursor < len(f.items)-1 {
			f.cursor++
		}
	case "enter":
		if len(f.items) == 0 {
			return f, nil, Active
		}

		line, err := f.invocation(f.items[f.cursor])
		if err != nil {
			f.err = err

			return f, nil, Active
		}

		f.err = nil
		c := execconfirm.New(line, f.width, f.height)
		f.confirm = &c
	}

	return f, nil, Active
}

func (f Flow) HasChoices() bool { return len(f.items) > 0 }

func (f Flow) Confirming() bool { return f.confirm != nil }

func (f Flow) View() string {
	if f.confirm != nil {
		return f.confirm.View()
	}

	var b strings.Builder

	if f.err != nil {
		b.WriteString(errStyle.Width(f.width).Render(fmt.Sprintf("error: %v", f.err)))
		b.WriteString("\n\n")
	}

	b.WriteString(theme.Wrap(f.width, f.title))
	b.WriteString("\n\n")

	if len(f.items) == 0 {
		b.WriteString(dimStyle.Width(f.width).Render(f.empty))
		b.WriteString("\n")

		return b.String()
	}

	rowWidth := f.width
	if rowWidth > 0 {
		rowWidth = max(rowWidth-2, 10)
	}

	for i, it := range f.items {
		line := theme.Wrap(rowWidth, fmt.Sprintf("%s  %s", it.label, it.detail))
		if i == f.cursor {
			b.WriteString(selectedStyle.Render("> " + line))
		} else {
			b.WriteString("  ")
			b.WriteString(line)
		}

		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Width(f.width).Render("j/k move  enter pick  esc back  q quit"))

	return b.String()
}
