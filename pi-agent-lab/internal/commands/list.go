package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "list <agents|tasks|runs>",
		Short:     "List registered agents, tasks, or recorded runs",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"agents", "tasks", "runs"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented: list %s", args[0])
		},
	}

	var taskID string
	cmd.Flags().StringVar(&taskID, "task", "", "filter 'list runs' to a single task id")

	return cmd
}
