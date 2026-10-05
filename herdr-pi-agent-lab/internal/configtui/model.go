package configtui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/pathfield"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/scroll"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/tuistate"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

var (
	headerStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle       = theme.Dim
	errStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	attentionStyle = theme.Attention
)

type mode int

const (
	modeList mode = iota
	modeInit
	modeAdd
)

type Model struct {
	mode     mode
	paths    []string
	active   string
	cursor   int
	input    textinput.Model
	matches  []string
	width    int
	height   int
	scroller scroll.Model
	status   string
	err      error
	loading  bool
}

type listMsg struct {
	paths  []string
	active string
	err    error
}

type doneMsg struct {
	status string
	err    error
}

func NewModel() Model {
	ti := textinput.New()
	ti.Prompt = ""

	return Model{loading: true, input: ti}
}

func loadCmd() tea.Cmd {
	return func() tea.Msg {
		paths, err := registry.ListRegistered()
		if err != nil {
			return listMsg{err: err}
		}

		active, err := tuistate.Load()

		return listMsg{paths: paths, active: active, err: err}
	}
}

func qualify(typed string) string {
	if pathfield.IsExplicit(typed) {
		return typed
	}

	return "~/" + typed
}

func expandPath(typed string) (string, error) {
	return filepath.Abs(pathfield.Resolve(qualify(strings.TrimSpace(typed))))
}

func typedMatches(typed string) []string {
	q := qualify(typed)
	matches := pathfield.Matches(q, true)

	if q == typed {
		return matches
	}

	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, strings.TrimPrefix(m, "~/"))
	}

	return out
}

func (m *Model) refreshMatches() {
	m.matches = typedMatches(m.input.Value())
}

func initCmd(p string) tea.Cmd {
	return func() tea.Msg {
		abs, err := expandPath(p)
		if err != nil {
			return doneMsg{err: err}
		}

		m, err := registry.InitAndRegister(abs, "", time.Now())
		if err != nil {
			return doneMsg{err: err}
		}

		return doneMsg{status: fmt.Sprintf("initialised %q at %s", m.Name, abs)}
	}
}

func addCmd(p string) tea.Cmd {
	return func() tea.Msg {
		abs, err := expandPath(p)
		if err != nil {
			return doneMsg{err: err}
		}

		_, err = registry.Register(abs)
		if errors.Is(err, registry.ErrAlreadyRegistered) {
			return doneMsg{status: "already registered: " + abs}
		}
		if err != nil {
			return doneMsg{err: err}
		}

		return doneMsg{status: "registered " + abs}
	}
}

func (m Model) Init() tea.Cmd {
	return loadCmd()
}

func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.err = nil

		return m, nil
	case "enter":
		value := strings.TrimSpace(m.input.Value())
		if value == "" {
			m.err = errors.New("a path is required")

			return m, nil
		}

		if m.mode == modeInit {
			return m, initCmd(value)
		}

		return m, addCmd(value)
	case "tab":
		if len(m.matches) == 0 {
			return m, nil
		}

		lcp := pathfield.LongestCommonPrefix(m.matches)
		if len(lcp) > len(m.input.Value()) {
			m.input.SetValue(lcp)
			m.input.CursorEnd()
			m.refreshMatches()
		}

		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refreshMatches()

	return m, cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeList {
		return m.handleInputKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.paths)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.paths) == 0 {
			return m, nil
		}

		chosen := m.paths[m.cursor]
		if err := tuistate.Save(chosen); err != nil {
			m.err = err

			return m, nil
		}

		m.active = chosen
		m.err = nil
		m.status = "active registry: " + chosen
	case "i", "a":
		m.mode = modeInit
		if msg.String() == "a" {
			m.mode = modeAdd
		}

		m.input.SetValue("")
		m.input.CursorEnd()
		m.refreshMatches()
		m.status = ""
		m.err = nil

		return m, m.input.Focus()
	}

	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case listMsg:
		m.loading = false
		m.err = msg.err
		m.paths = msg.paths
		m.active = msg.active
		m.cursor = min(m.cursor, max(0, len(m.paths)-1))

		return m, nil
	case doneMsg:
		m.mode = modeList
		m.err = msg.err
		m.status = msg.status

		return m, loadCmd()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.scroller.Resize(m.width, m.height)
		m.input.Width = max(m.width-10, 10)

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

func (m Model) writeInputAndMatches(b *strings.Builder) {
	b.WriteString(m.input.View())
	b.WriteString("\n")

	const shown = 8
	for i, match := range m.matches {
		if i == shown {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  … %d more", len(m.matches)-shown)))
			b.WriteString("\n")

			break
		}

		b.WriteString(dimStyle.Render("  " + match))
		b.WriteString("\n")
	}
}

func (m Model) parts() (header, body, footer string) {
	header = headerStyle.Render("Pi Agent Lab — Registries")

	var b strings.Builder

	if m.err != nil {
		b.WriteString(errStyle.Width(m.width).Render(fmt.Sprintf("error: %v", m.err)))
		b.WriteString("\n\n")
	}
	if m.status != "" {
		b.WriteString(okStyle.Width(m.width).Render(m.status))
		b.WriteString("\n\n")
	}

	switch m.mode {
	case modeInit:
		b.WriteString("Create a registry at this path (an existing directory is adopted).\n")
		b.WriteString(dimStyle.Width(m.width).Render("Relative paths start from your home directory."))
		b.WriteString("\n\n")
		m.writeInputAndMatches(&b)

		return header, b.String(), "enter init  tab complete  esc cancel"
	case modeAdd:
		b.WriteString("Register the existing registry at this path.\n")
		b.WriteString(dimStyle.Width(m.width).Render("Relative paths start from your home directory."))
		b.WriteString("\n\n")
		m.writeInputAndMatches(&b)

		return header, b.String(), "enter add  tab complete  esc cancel"
	}

	if m.loading {
		b.WriteString("loading...\n")

		return header, b.String(), ""
	}

	if len(m.paths) == 0 {
		b.WriteString(attentionStyle.Width(m.width).Render("No registries registered yet."))
		b.WriteString("\n")

		return header, b.String(), "i init a new registry  a add an existing one  q quit  ctrl+u/ctrl+d scroll"
	}

	for i, p := range m.paths {
		marker := " "
		if p == m.active {
			marker = "*"
		}

		name, status := "-", "ok"
		if mf, err := registry.ReadManifest(p); err != nil {
			status = "missing or invalid manifest"
		} else {
			name = mf.Name
		}

		line := fmt.Sprintf("%s %s  %s  (%s)", marker, name, p, status)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + line))
		} else {
			b.WriteString("  ")
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	return header, b.String(), "j/k move  enter set active (*)  i init  a add  q quit  ctrl+u/ctrl+d scroll"
}

func (m Model) View() string {
	header, body, footer := m.parts()

	return m.scroller.RenderWithFooter(header, body, footer)
}
