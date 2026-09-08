package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/tui"
)

func main() {
	if os.Getenv("PI_AGENT_REGISTRY_DIR") == "" {
		fmt.Fprintln(os.Stderr, "PI_AGENT_REGISTRY_DIR must be set to the pi-agent-lab registry directory")

		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
