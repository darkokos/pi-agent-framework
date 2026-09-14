package commands

import "github.com/spf13/cobra"

func NewCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent, agent-version, or task",
	}

	cmd.AddCommand(newCreateAgentCommand())
	cmd.AddCommand(newCreateAgentVersionCommand())
	cmd.AddCommand(newCreateTaskCommand())

	return cmd
}
