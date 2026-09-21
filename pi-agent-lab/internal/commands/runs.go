package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-compare"
	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/format"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func strOrDash(p *string) string {
	if p == nil {
		return "-"
	}

	return *p
}

func intOrDash(p *int) string {
	if p == nil {
		return "-"
	}

	return fmt.Sprintf("%d", *p)
}

func durationOrDash(p *float64) string {
	if p == nil {
		return "-"
	}

	return fmt.Sprintf("%.0fms", *p)
}

func printTaskGroup(w io.Writer, group *compare.TaskGroup) {
	fmt.Fprintf(w, "TASK %s\n", group.TaskID)

	if len(group.Rated) > 0 {
		tw := format.NewTabWriter(w)
		fmt.Fprintln(tw, "  RANK\tSESSION\tAGENT@VERSION\tOUTCOME\tREWORK\tAUTONOMY\tCOST\tTOKENS\tTOOL ERRORS\tDURATION")
		for i, r := range group.Rated {
			fmt.Fprintf(tw, "  %d\t%s\t%s\t%s\t%s\t%s\t$%.4f\t%.0f\t%d\t%s\n",
				i+1, r.SessionId, strOrDash(r.AgentVersion),
				intOrDash(r.Outcome.Outcome), intOrDash(r.Outcome.Rework), intOrDash(r.Outcome.Autonomy),
				r.Usage.CostUsd, compare.TotalTokens(r), r.Activity.ToolErrors, durationOrDash(r.DurationMs),
			)
		}
		tw.Flush()
	}

	if len(group.Unrated) > 0 {
		fmt.Fprintln(w, "  UNRATED (needs /rate)")
		tw := format.NewTabWriter(w)
		fmt.Fprintln(tw, "    SESSION\tAGENT@VERSION")
		for _, r := range group.Unrated {
			fmt.Fprintf(tw, "    %s\t%s\n", r.SessionId, strOrDash(r.AgentVersion))
		}
		tw.Flush()
	}
}

func newListRunsCommand() *cobra.Command {
	var taskID string

	cmd := &cobra.Command{
		Use:   "runs",
		Short: "List recorded runs, ranked by rating",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if taskID != "" {
				root, err := registry.ResolveRoot()
				if err != nil {
					return err
				}

				if _, err := registry.GetTask(root, taskID); err != nil {
					return err
				}
			}

			records, err := registry.ListRecords()
			if err != nil {
				return err
			}

			groups := compare.GroupByTask(records)
			found := false
			for _, group := range groups {
				if taskID != "" && group.TaskID != taskID {
					continue
				}

				found = true
				printTaskGroup(cmd.OutOrStdout(), group)
			}

			if taskID != "" && !found {
				fmt.Fprintf(cmd.OutOrStdout(), "no runs recorded for task %q\n", taskID)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "filter to a single task id")

	_ = cmd.RegisterFlagCompletionFunc("task", completeTaskIDs)

	return cmd
}
