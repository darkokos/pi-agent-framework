package scroll

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
)

type Model struct {
	vp         viewport.Model
	width      int
	fullHeight int
}

func (m *Model) Resize(width, height int) {
	m.width, m.fullHeight = width, height
}

func (m *Model) sync(header, body, footer string) (wrappedHeader, wrappedFooter string) {
	wrappedHeader = theme.Wrap(m.width, header)
	reserved := 2 + strings.Count(wrappedHeader, "\n")

	if footer != "" {
		wrappedFooter = theme.Help.Width(m.width).Render(footer)
		reserved += 2 + strings.Count(wrappedFooter, "\n")
	}

	avail := max(1, m.fullHeight-reserved)
	natural := strings.Count(body, "\n") + 1

	m.vp.Width = m.width
	m.vp.Height = min(natural, avail)

	return wrappedHeader, wrappedFooter
}

func (m *Model) HandleKey(key string, header, body, footer func() string) bool {
	switch key {
	case "ctrl+u", "ctrl+d":
	default:
		return false
	}

	b := body()

	m.sync(header(), b, footer())
	m.vp.SetContent(b)

	if key == "ctrl+u" {
		m.vp.HalfPageUp()
	} else {
		m.vp.HalfPageDown()
	}

	return true
}

func (m *Model) GotoTop() {
	m.vp.GotoTop()
}

func (m *Model) Render(header, body string) string {
	wrappedHeader, _ := m.sync(header, body, "")
	m.vp.SetContent(body)

	return wrappedHeader + "\n\n" + m.vp.View()
}

func (m *Model) RenderWithFooter(header, body, footer string) string {
	wrappedHeader, wrappedFooter := m.sync(header, body, footer)
	m.vp.SetContent(body)

	return wrappedHeader + "\n\n" + m.vp.View() + "\n\n" + wrappedFooter
}
