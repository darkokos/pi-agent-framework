package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/commands"
)

func main() {
	root := &cobra.Command{
		Use:   "pi-agent-lab",
		Short: "Create, version, and compare Pi agents",
	}

	root.AddCommand(commands.NewAgentCommand())
	root.AddCommand(commands.NewTaskCommand())
	root.AddCommand(commands.NewListCommand())
	root.AddCommand(commands.NewShowCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
