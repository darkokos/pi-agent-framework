package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/execconfirm"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/scroll"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
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
	scroller  scroll.Model
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

		m.scroller.Resize(m.width, m.height)
		m.overrideInput.Width = max(m.width-10, 10)

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
		if m.view != viewResumeConfirm {
			header, body, footer := m.parts()
			if m.scroller.HandleKey(msg.String(), func() string { return header }, func() string { return body }, func() string { return footer }) {
				return m, nil
			}
		}

		prevView := m.view

		newModel, cmd := m.handleKey(msg)
		if nm, ok := newModel.(Model); ok {
			if nm.view != prevView {
				nm.scroller.GotoTop()
			}

			return nm, cmd
		}

		return newModel, cmd
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
	headerStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle       = theme.Dim
	errStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	resultStyle    = theme.Result
	attentionStyle = theme.Attention
)

func summarize(g *compare.TaskGroup) string {
	if g == nil {
		return dimStyle.Render("no runs yet")
	}

	return resultStyle.Render(fmt.Sprintf("%d rated, %d unrated", len(g.Rated), len(g.Unrated)))
}

func (m Model) viewList(b *strings.Builder) {
	if len(m.rows) == 0 {
		b.WriteString(dimStyle.Width(m.width).Render("no tasks registered yet — pi-agent-lab task create <id> ..."))
		b.WriteString("\n")

		return
	}

	rowWidth := m.width
	if rowWidth > 0 {
		rowWidth = max(rowWidth-2, 10)
	}

	for i, row := range m.rows {
		line := theme.Wrap(rowWidth, fmt.Sprintf("%s  %s", row.task.ID, summarize(row.group)))
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
	b.WriteString(theme.Wrap(m.width, headerStyle.Render("task "+row.task.ID)))
	b.WriteString("\n\n")

	if row.group == nil || (len(row.group.Rated) == 0 && len(row.group.Unrated) == 0) {
		b.WriteString(dimStyle.Width(m.width).Render("no runs recorded for this task yet"))
		b.WriteString("\n")
		return
	}

	rowWidth := m.width
	if rowWidth > 0 {
		rowWidth = max(rowWidth-2, 10)
	}

	for i, r := range row.group.Rated {
		b.WriteString(theme.Wrap(m.width, fmt.Sprintf("%d. %s  %s", i+1, r.SessionId, strOrDash(r.AgentVersion))))
		b.WriteString("\n")
		b.WriteString(resultStyle.Width(m.width).Render(fmt.Sprintf("   outcome=%s rework=%s autonomy=%s  cost=$%.4f tokens=%.0f toolErrors=%d duration=%s",
			intOrDash(r.Outcome.Outcome), intOrDash(r.Outcome.Rework), intOrDash(r.Outcome.Autonomy),
			r.Usage.CostUsd, compare.TotalTokens(r), r.Activity.ToolErrors, durationOrDash(r.DurationMs))))
		b.WriteString("\n")
	}

	if len(row.group.Unrated) > 0 {
		b.WriteString("\n")
		b.WriteString(attentionStyle.Width(m.width).Render("needs rating:"))
		b.WriteString("\n")
		for i, r := range row.group.Unrated {
			line := theme.Wrap(rowWidth, fmt.Sprintf("%s  %s", r.SessionId, strOrDash(r.AgentVersion)))
			if i == m.runCursor {
				b.WriteString(selectedStyle.Render("> " + line))
			} else {
				b.WriteString("  ")
				b.WriteString(line)
			}

			b.WriteString("\n")
		}
	}
}

func (m Model) parts() (header, body, footer string) {
	header = headerStyle.Render("Pi Agent Lab — Review Runs")
	if m.resolvedRoot != "" {
		suffix := ""
		if m.overrideRoot != nil {
			suffix = " (override)"
		}

		header += dimStyle.Render(" - " + m.resolvedRoot + suffix)
	}

	var b strings.Builder

	if m.err != nil {
		b.WriteString(errStyle.Width(m.width).Render(fmt.Sprintf("error: %v", m.err)))
		b.WriteString("\n\n")
	}

	if m.view == viewResumeConfirm {
		b.WriteString(m.resume.View())

		return header, b.String(), ""
	}

	if m.view == viewOverridePrompt {
		b.WriteString("Registry path (empty clears override):\n")
		b.WriteString(m.overrideInput.View())

		return header, b.String(), "enter confirm  esc cancel"
	}

	if m.status != "" {
		b.WriteString(okStyle.Width(m.width).Render(m.status))
		b.WriteString("\n\n")
	}

	if m.resolvedRoot == "" && m.err != nil && m.overrideRoot == nil {
		return header, b.String(), "o use a registry for this session  q quit"
	}

	if m.loading && len(m.rows) == 0 {
		b.WriteString("loading...\n")

		return header, b.String(), ""
	}

	switch m.view {
	case viewDetail:
		m.viewDetail(&b)
	default:
		m.viewList(&b)
	}

	if m.view == viewDetail {
		footer = "j/k select unrated run  r resume it to rate  esc back  q quit  ctrl+u/ctrl+d scroll"
	} else {
		footer = "j/k move  enter open  o override registry  q quit  ctrl+u/ctrl+d scroll"
	}

	return header, b.String(), footer
}

func (m Model) View() string {
	header, body, footer := m.parts()

	if m.view == viewResumeConfirm {
		return theme.Wrap(m.width, header) + "\n\n" + body
	}
	if footer == "" {
		return m.scroller.Render(header, body)
	}

	return m.scroller.RenderWithFooter(header, body, footer)
}
