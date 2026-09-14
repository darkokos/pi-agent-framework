package commands

import "github.com/spf13/cobra"

func NewUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update fields of an existing agent, agent-version, or task",
	}

	cmd.AddCommand(newUpdateAgentCommand())
	cmd.AddCommand(newUpdateAgentVersionCommand())
	cmd.AddCommand(newUpdateTaskCommand())

	return cmd
}
