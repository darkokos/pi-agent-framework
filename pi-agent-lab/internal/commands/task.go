package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/store"
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
			task, err := store.CreateTask(args[0], promptFile, notes)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created task %q\n", task.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "path to the task prompt text (required)")
	cmd.Flags().StringVar(&notes, "notes", "", "freeform notes about this task")
	_ = cmd.MarkFlagRequired("prompt-file")

	return cmd
}
