package scroll

import "github.com/charmbracelet/bubbles/viewport"

type Model struct {
	vp viewport.Model
}

func (m *Model) Resize(width, height int) {
	m.vp.Width, m.vp.Height = width, height
}

func (m *Model) HandleKey(key string, content func() string) bool {
	switch key {
	case "ctrl+u":
		m.vp.SetContent(content())
		m.vp.HalfPageUp()

		return true
	case "ctrl+d":
		m.vp.SetContent(content())
		m.vp.HalfPageDown()

		return true
	}

	return false
}

func (m *Model) GotoTop() {
	m.vp.GotoTop()
}

func (m *Model) Render(content string) string {
	m.vp.SetContent(content)

	return m.vp.View()
}
