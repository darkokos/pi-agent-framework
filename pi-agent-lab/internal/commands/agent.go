package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Create and version agent definitions",
	}

	cmd.AddCommand(newAgentCreateCommand())
	cmd.AddCommand(newAgentVersionCommand())

	return cmd
}

func newAgentCreateCommand() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "create <agent-id>",
		Short: "Register a new agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented: agent create %s (--name %q --description %q)", args[0], name, description)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "human-readable agent name")
	cmd.Flags().StringVar(&description, "description", "", "what this agent is for")

	return cmd
}

func newAgentVersionCommand() *cobra.Command {
	var (
		systemPromptFile string
		model            string
		provider         string
		thinking         string
		notes            string
	)

	cmd := &cobra.Command{
		Use:   "version <agent-id> <version-id>",
		Short: "Add a new version to an existing agent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf(
				"not yet implemented: agent version %s %s (--system-prompt-file %q --model %q --provider %q --thinking %q --notes %q)",
				args[0], args[1], systemPromptFile, model, provider, thinking, notes,
			)
		},
	}

	cmd.Flags().StringVar(&systemPromptFile, "system-prompt-file", "", "path to the system prompt text (required)")
	cmd.Flags().StringVar(&model, "model", "", "model pattern, e.g. anthropic/claude-sonnet-4-5 (required)")
	cmd.Flags().StringVar(&provider, "provider", "", "provider name")
	cmd.Flags().StringVar(&thinking, "thinking", "", "thinking level: off|minimal|low|medium|high|xhigh|max")
	cmd.Flags().StringVar(&notes, "notes", "", "freeform notes about this version")
	_ = cmd.MarkFlagRequired("system-prompt-file")
	_ = cmd.MarkFlagRequired("model")

	return cmd
}
