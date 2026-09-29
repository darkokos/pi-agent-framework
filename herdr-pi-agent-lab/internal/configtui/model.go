package configtui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

var (
	headerStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle       = theme.Dim
	errStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	okStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	helpStyle      = theme.Help
	attentionStyle = theme.Attention
)

type screen int

const (
	screenView screen = iota
	screenEdit
)

type Model struct {
	screen    screen
	root      string
	suggested string
	input     textinput.Model
	width     int
	err       error
	status    string
	loading   bool
}

type dataMsg struct{ root string }

type errMsg struct{ err error }

func NewModel() Model {
	ti := textinput.New()
	ti.Prompt = ""

	return Model{loading: true, input: ti}
}

func loadRoot() tea.Cmd {
	return func() tea.Msg {
		root, err := registry.ResolveRoot()
		if err != nil {
			return errMsg{err}
		}

		return dataMsg{root: root}
	}
}

func (m Model) Init() tea.Cmd {
	return loadRoot()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.screen == screenEdit {
		return m.handleEditKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "e":
		start := m.root
		if start == "" {
			start = m.suggested
		}

		m.input.SetValue(start)
		m.input.CursorEnd()
		m.status = ""
		m.err = nil
		m.screen = screenEdit

		return m, m.input.Focus()
	case "d":
		dir, err := registry.SuggestedRegistryDir()
		if err == nil {
			err = registry.SetRegistryDir(dir)
		}
		if err != nil {
			m.err = err

			return m, nil
		}

		m.root = dir
		m.status = "registry set to the default location"
		m.err = nil
	}

	return m, nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dataMsg:
		m.loading = false
		m.root = msg.root

		return m, nil
	case errMsg:
		m.loading = false

		if unset, ok := errors.AsType[*registry.NoRegistryError](msg.err); ok {
			m.suggested = unset.Suggested

			return m, nil
		}

		m.err = msg.err

		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.input.Width = max(m.width-10, 10)

		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenView
		m.err = nil

		return m, nil
	case "ctrl+w":
		path := strings.TrimSpace(m.input.Value())
		if path == "" {
			m.err = fmt.Errorf("registry directory is required")

			return m, nil
		}

		abs, err := filepath.Abs(path)
		if err != nil {
			m.err = err

			return m, nil
		}

		if err := registry.SetRegistryDir(abs); err != nil {
			m.err = err

			return m, nil
		}

		m.root = abs
		m.status = "saved"
		m.err = nil
		m.screen = screenView

		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(theme.Wrap(m.width, headerStyle.Render("Pi Agent Lab — Config")))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString("loading...\n")

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

	if m.screen == screenEdit {
		b.WriteString("Registry Directory:\n")
		b.WriteString(m.input.View())
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Width(m.width).Render("persisted to ~/.config/pi-agent-lab/config.json"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Width(m.width).Render("ctrl+w save  esc cancel"))

		return b.String()
	}

	if m.root == "" {
		b.WriteString(attentionStyle.Width(m.width).Render("No registry directory is set."))
		b.WriteString(theme.Wrap(m.width, "\n\nChoose where your agents and tasks are stored. Nothing works until you do.\n"))
		if m.suggested != "" {
			b.WriteString("\nSuggested location:\n")
			b.WriteString(theme.Wrap(m.width, m.suggested))
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Width(m.width).Render("e choose a directory  d use the suggested location  q quit"))

		return b.String()
	}

	b.WriteString("Registry Directory:\n")
	b.WriteString(theme.Wrap(m.width, m.root))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Width(m.width).Render("e edit  d use the default location  q quit"))

	return b.String()
}
