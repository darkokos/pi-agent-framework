package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <agent-id>@<version-id> <task-id>",
		Short: "Print (never run) the pi invocation for an agent version + task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented: show %s %s", args[0], args[1])
		},
	}
}
