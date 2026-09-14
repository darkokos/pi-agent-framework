package commands

import "github.com/spf13/cobra"

func NewDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an agent, agent-version, or task",
	}

	cmd.AddCommand(newDeleteAgentCommand())
	cmd.AddCommand(newDeleteAgentVersionCommand())
	cmd.AddCommand(newDeleteTaskCommand())

	return cmd
}
