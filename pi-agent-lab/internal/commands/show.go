package commands

import "github.com/spf13/cobra"

func NewShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show full details of a single agent, agent-version, or task",
	}

	cmd.AddCommand(newShowAgentCommand())
	cmd.AddCommand(newShowAgentVersionCommand())
	cmd.AddCommand(newShowTaskCommand())

	return cmd
}
