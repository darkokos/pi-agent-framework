package theme

import "github.com/charmbracelet/lipgloss"

var (
	Dim       = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	Help      = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	Result    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117"))
	Attention = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
)

func Wrap(width int, s string) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}
