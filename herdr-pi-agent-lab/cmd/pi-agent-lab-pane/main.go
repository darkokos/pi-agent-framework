package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/tui"
)

func main() {
	var startupErr error
	if os.Getenv("PI_AGENT_REGISTRY_DIR") == "" {
		startupErr = fmt.Errorf("PI_AGENT_REGISTRY_DIR must be set to the pi-agent-lab registry directory")
	}

	p := tea.NewProgram(tui.NewModel(startupErr))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
