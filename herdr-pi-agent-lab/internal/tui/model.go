package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/pi-agent-compare"
	"github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

const pollInterval = 3 * time.Second

type view int

const (
	viewList view = iota
	viewDetail
)

type taskRow struct {
	task  *registry.Task
	group *compare.TaskGroup
}

type Model struct {
	rows    []taskRow
	cursor  int
	view    view
	err     error
	loading bool
}

type dataMsg struct {
	rows []taskRow
}

type errMsg struct{ err error }

type tickMsg struct{}

func NewModel() Model {
	return Model{loading: true}
}

func loadData() tea.Msg {
	tasks, err := registry.ListTasks()
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

	return dataMsg{rows: rows}
}

func tick() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadData, tick())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Batch(loadData, tick())
	case dataMsg:
		m.loading = false
		m.err = nil
		m.rows = msg.rows
		if m.cursor >= len(m.rows) {
			m.cursor = max(0, len(m.rows)-1)
		}

		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err

		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
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
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case "enter":
		if m.view == viewList && len(m.rows) > 0 {
			m.view = viewDetail
		}
	case "esc", "backspace":
		if m.view == viewDetail {
			m.view = viewList
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
		fmt.Fprintf(b, "%d. %-28s outcome=%s rework=%s autonomy=%s  cost=$%.4f tokens=%.0f toolErrors=%d duration=%s\n",
			i+1, strOrDash(r.AgentVersion),
			intOrDash(r.Outcome.Outcome), intOrDash(r.Outcome.Rework), intOrDash(r.Outcome.Autonomy),
			r.Usage.CostUsd, compare.TotalTokens(r), r.Activity.ToolErrors, durationOrDash(r.DurationMs))
	}

	if len(row.group.Unrated) > 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("needs rating:"))
		b.WriteString("\n")
		for _, r := range row.group.Unrated {
			b.WriteString(dimStyle.Render(fmt.Sprintf("-  %s\n", strOrDash(r.AgentVersion))))
		}
	}
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Pi Agent Lab"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errStyle.Render(fmt.Sprintf("error: %v", m.err)))
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
	b.WriteString(helpStyle.Render("j/k move  enter open  esc back  q quit"))

	return b.String()
}
