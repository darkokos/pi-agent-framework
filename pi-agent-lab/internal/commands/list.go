package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/store"
)

func NewListCommand() *cobra.Command {
	var taskID string

	cmd := &cobra.Command{
		Use:       "list <agents|tasks|runs>",
		Short:     "List registered agents, tasks, or recorded runs",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"agents", "tasks", "runs"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "agents":
				agents, err := store.ListAgents()
				if err != nil {
					return err
				}

				for _, agent := range agents {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d version(s)\n", agent.ID, agent.Name, len(agent.Versions))
				}

				return nil
			case "tasks":
				tasks, err := store.ListTasks()
				if err != nil {
					return err
				}

				for _, task := range tasks {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", task.ID, task.PromptFile)
				}

				return nil
			default:
				return fmt.Errorf("not yet implemented: list %s", args[0])
			}
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "filter 'list runs' to a single task id")

	return cmd
}
