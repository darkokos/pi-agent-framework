package theme

import (
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

var (
	Dim       = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	Help      = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	Result    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	Attention = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
)

func Wrap(width int, s string) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}

func RenderMarkdown(md string, wordWrap int) string {
	if wordWrap <= 0 {
		wordWrap = 80
	}

	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(wordWrap))
	if err != nil {
		return md
	}

	rendered, err := r.Render(md)
	if err != nil {
		return md
	}

	return rendered
}
