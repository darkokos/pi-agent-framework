package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewTaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Register standardized task inputs",
	}

	cmd.AddCommand(newTaskCreateCommand())

	return cmd
}

func newTaskCreateCommand() *cobra.Command {
	var promptFile, notes string

	cmd := &cobra.Command{
		Use:   "create <task-id>",
		Short: "Register a new task (standardized input)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented: task create %s (--prompt-file %q --notes %q)", args[0], promptFile, notes)
		},
	}

	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "path to the task prompt text (required)")
	cmd.Flags().StringVar(&notes, "notes", "", "freeform notes about this task")
	_ = cmd.MarkFlagRequired("prompt-file")

	return cmd
}
