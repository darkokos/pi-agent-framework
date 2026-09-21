package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/execconfirm"
	"github.com/darkokos/pi-agent-framework/pi-agent-compare"
	invoke "github.com/darkokos/pi-agent-framework/pi-agent-invoke"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

const pollInterval = 3 * time.Second

type view int

const (
	viewList view = iota
	viewDetail
	viewOverridePrompt
	viewResumeConfirm
)

type taskRow struct {
	task  *registry.Task
	group *compare.TaskGroup
}

type Model struct {
	rows          []taskRow
	cursor        int
	view          view
	err           error
	loading       bool
	resolvedRoot  string
	overrideRoot  *string
	overrideInput textinput.Model

	runCursor int
	resume    *execconfirm.Model
	status    string
	width     int
	height    int
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "path to registry (empty clears override)"

	return Model{loading: true, overrideInput: ti}
}

type dataMsg struct {
	rows []taskRow
	root string
}

type errMsg struct{ err error }

type tickMsg struct{}

type resumeFinishedMsg struct{ err error }

func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func loadData(overrideRoot *string) tea.Cmd {
	return func() tea.Msg {
		root := ""
		if overrideRoot != nil {
			root = *overrideRoot
		} else {
			resolved, err := registry.ResolveRoot()
			if err != nil {
				return errMsg{err}
			}

			root = resolved
		}

		tasks, err := registry.ListTasks(root)
		if err != nil {
			return errMsg{err}
		}

		records, err := registry.ListRecords()
		if err != nil {
			return errMsg{err}
		}

		groupsByTask := map[string]*compare.TaskGroup{}
		for _, g := range compare.GroupByTask(records) {
			groupsByTask[g.TaskID] = g
		}

		rows := make([]taskRow, 0, len(tasks))
		for _, t := range tasks {
			rows = append(rows, taskRow{task: t, group: groupsByTask[t.ID]})
		}

		return dataMsg{rows: rows, root: root}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadData(m.overrideRoot), tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(loadData(m.overrideRoot), tick())
	case dataMsg:
		m.loading = false
		m.err = nil
		m.rows = msg.rows
		m.resolvedRoot = msg.root
		if m.cursor >= len(m.rows) {
			m.cursor = max(0, len(m.rows)-1)
		}

		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err

		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.resume != nil {
			c := *m.resume
			c.Resize(m.width, m.height)
			m.resume = &c
		}

		return m, nil
	case resumeFinishedMsg:
		m.resume = nil
		m.view = viewDetail

		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.status = "session closed"
		}

		return m, loadData(m.overrideRoot)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) unratedCount() int {
	if m.cursor >= len(m.rows) || m.rows[m.cursor].group == nil {
		return 0
	}

	return len(m.rows[m.cursor].group.Unrated)
}

func (m Model) handleResumeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	}

	c, cmd, action := m.resume.Update(msg)
	m.resume = &c

	switch action {
	case execconfirm.Run:
		return m, execconfirm.Exec(c.Line(), func(err error) tea.Msg { return resumeFinishedMsg{err: err} })
	case execconfirm.Cancel:
		m.resume = nil
		m.view = viewDetail
	}

	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.view == viewResumeConfirm {
		return m.handleResumeKey(msg)
	}

	if m.view == viewOverridePrompt {
		switch msg.String() {
		case "esc":
			m.view = viewList

			return m, nil
		case "enter":
			value := strings.TrimSpace(m.overrideInput.Value())
			if value == "" {
				m.overrideRoot = nil
			} else {
				m.overrideRoot = &value
			}

			m.view = viewList
			m.loading = true

			return m, loadData(m.overrideRoot)
		}

		var cmd tea.Cmd
		m.overrideInput, cmd = m.overrideInput.Update(msg)

		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.view == viewDetail {
			if m.runCursor > 0 {
				m.runCursor--
			}
		} else if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.view == viewDetail {
			if m.runCursor < m.unratedCount()-1 {
				m.runCursor++
			}
		} else if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "enter", "r":
		if m.view == viewList && msg.String() == "enter" && len(m.rows) > 0 {
			m.view = viewDetail
			m.runCursor = 0
			m.status = ""
		} else if m.view == viewDetail && m.unratedCount() > 0 {
			run := m.rows[m.cursor].group.Unrated[m.runCursor]
			c := execconfirm.New(invoke.BuildResume(run.Cwd, run.SessionFile, run.AgentVersion, run.TaskId), m.width, m.height)
			m.resume = &c
			m.view = viewResumeConfirm
			m.status = ""
		}
	case "esc", "backspace":
		if m.view == viewDetail {
			m.view = viewList
		}
	case "o":
		if m.view == viewList {
			if m.overrideRoot != nil {
				m.overrideInput.SetValue(*m.overrideRoot)
			} else {
				m.overrideInput.SetValue("")
			}

			m.overrideInput.CursorEnd()
			m.view = viewOverridePrompt

			return m, m.overrideInput.Focus()
		}
	}

	return m, nil
}

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle     = dimStyle
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

func summarize(g *compare.TaskGroup) string {
	if g == nil {
		return dimStyle.Render("no runs yet")
	}

	return dimStyle.Render(fmt.Sprintf("%d rated, %d unrated", len(g.Rated), len(g.Unrated)))
}

func (m Model) viewList(b *strings.Builder) {
	if len(m.rows) == 0 {
		b.WriteString(dimStyle.Render("no tasks registered yet — pi-agent-lab task create <id> ..."))
		b.WriteString("\n")

		return
	}

	for i, row := range m.rows {
		line := fmt.Sprintf("%s  %s", row.task.ID, summarize(row.group))
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + line))
		} else {
			b.WriteString("  ")
			b.WriteString(line)
		}

		b.WriteString("\n")
	}
}

func strOrDash(p *string) string {
	if p == nil {
		return "-"
	}

	return *p
}

func intOrDash(p *int) string {
	if p == nil {
		return "-"
	}

	return fmt.Sprintf("%d", *p)
}

func durationOrDash(p *float64) string {
	if p == nil {
		return "-"
	}

	return fmt.Sprintf("%.0fms", *p)
}

func (m Model) viewDetail(b *strings.Builder) {
	row := m.rows[m.cursor]
	b.WriteString(headerStyle.Render("task " + row.task.ID))
	b.WriteString("\n\n")

	if row.group == nil || (len(row.group.Rated) == 0 && len(row.group.Unrated) == 0) {
		b.WriteString(dimStyle.Render("no runs recorded for this task yet"))
		b.WriteString("\n")
		return
	}

	for i, r := range row.group.Rated {
		fmt.Fprintf(b, "%d. %s  %s\n", i+1, r.SessionId, strOrDash(r.AgentVersion))
		b.WriteString(dimStyle.Render(fmt.Sprintf("   outcome=%s rework=%s autonomy=%s  cost=$%.4f tokens=%.0f toolErrors=%d duration=%s",
			intOrDash(r.Outcome.Outcome), intOrDash(r.Outcome.Rework), intOrDash(r.Outcome.Autonomy),
			r.Usage.CostUsd, compare.TotalTokens(r), r.Activity.ToolErrors, durationOrDash(r.DurationMs))))
		b.WriteString("\n")
	}

	if len(row.group.Unrated) > 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("needs rating:"))
		b.WriteString("\n")
		for i, r := range row.group.Unrated {
			line := fmt.Sprintf("%s  %s", r.SessionId, strOrDash(r.AgentVersion))
			if i == m.runCursor {
				b.WriteString(selectedStyle.Render("> " + line))
			} else {
				b.WriteString(dimStyle.Render("  " + line))
			}

			b.WriteString("\n")
		}
	}
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Pi Agent Lab"))
	if m.resolvedRoot != "" {
		suffix := ""
		if m.overrideRoot != nil {
			suffix = " (override)"
		}

		b.WriteString(dimStyle.Render(" - " + m.resolvedRoot + suffix))
	}

	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.view == viewResumeConfirm {
		b.WriteString(m.resume.View())

		return b.String()
	}

	if m.view == viewOverridePrompt {
		b.WriteString("Registry path (empty clears override):\n")
		b.WriteString(m.overrideInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("enter confirm  esc cancel"))

		return b.String()
	}

	if m.status != "" {
		b.WriteString(okStyle.Render(m.status))
		b.WriteString("\n\n")
	}

	if m.loading && len(m.rows) == 0 {
		b.WriteString("loading...\n")

		return b.String()
	}

	switch m.view {
	case viewDetail:
		m.viewDetail(&b)
	default:
		m.viewList(&b)
	}

	b.WriteString("\n")
	if m.view == viewDetail {
		b.WriteString(helpStyle.Render("j/k select unrated run  r resume it to rate  esc back  q quit"))
	} else {
		b.WriteString(helpStyle.Render("j/k move  enter open  o override registry  q quit"))
	}

	return b.String()
}
