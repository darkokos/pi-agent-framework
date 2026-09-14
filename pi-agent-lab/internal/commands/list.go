package commands

import "github.com/spf13/cobra"

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agents, agent-versions, tasks, or recorded runs",
	}

	cmd.AddCommand(newListAgentsCommand())
	cmd.AddCommand(newListAgentVersionsCommand())
	cmd.AddCommand(newListTasksCommand())
	cmd.AddCommand(newListRunsCommand())

	return cmd
}
