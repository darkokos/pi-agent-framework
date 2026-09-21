package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/runtui"
)

func main() {
	p := tea.NewProgram(runtui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
