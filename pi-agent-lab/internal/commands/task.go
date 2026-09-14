package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/format"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func newCreateTaskCommand() *cobra.Command {
	var promptFile, notes string

	cmd := &cobra.Command{
		Use:   "task <task-id>",
		Short: "Register a new task (standardized input)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			task, err := registry.CreateTask(root, args[0], promptFile, notes)
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

	_ = cmd.RegisterFlagCompletionFunc("prompt-file", completeFilePaths)

	return cmd
}

func newListTasksCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tasks",
		Short: "List registered tasks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			tasks, err := registry.ListTasks(root)
			if err != nil {
				return err
			}

			tw := format.NewTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "ID\tPROMPT FILE\tNOTES")
			for _, task := range tasks {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", task.ID, task.PromptFile, task.Notes)
			}

			return tw.Flush()
		},
	}
}

func newShowTaskCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "task <task-id>",
		Short: "Show full details of a single task",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeTaskIDs(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			task, err := registry.GetTask(root, args[0])
			if err != nil {
				return err
			}

			tw := format.NewTabWriter(cmd.OutOrStdout())
			fmt.Fprintf(tw, "ID:\t%s\n", task.ID)
			fmt.Fprintf(tw, "Prompt File:\t%s\n", task.PromptFile)
			fmt.Fprintf(tw, "Notes:\t%s\n", task.Notes)
			fmt.Fprintf(tw, "Created At:\t%s\n", format.UnixMilli(task.CreatedAt))

			return tw.Flush()
		},
	}
}
