package commands

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-compare"
	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/store"
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
	fmt.Fprintf(w, "task %s\n", group.TaskID)

	for i, r := range group.Rated {
		fmt.Fprintf(w, "  %d. %s\toutcome=%s rework=%s autonomy=%s\tcost=$%.4f tokens=%.0f toolErrors=%d duration=%s\n",
			i+1, strOrDash(r.AgentVersion),
			intOrDash(r.Outcome.Outcome), intOrDash(r.Outcome.Rework), intOrDash(r.Outcome.Autonomy),
			r.Usage.CostUsd, compare.TotalTokens(r), r.Activity.ToolErrors, durationOrDash(r.DurationMs),
		)
	}

	for _, r := range group.Unrated {
		fmt.Fprintf(w, "  -  %s\t(needs /rate)\n", strOrDash(r.AgentVersion))
	}
}

func NewListCommand() *cobra.Command {
	var taskID string

	cmd := &cobra.Command{
		Use:       "list <agents|tasks|runs>",
		Short:     "List registered agents, tasks, or recorded runs",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"agents", "tasks", "runs"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "agents":
				agents, err := store.ListAgents()
				if err != nil {
					return err
				}

				for _, agent := range agents {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d version(s)\n", agent.ID, agent.Name, len(agent.Versions))
				}

				return nil
			case "tasks":
				tasks, err := store.ListTasks()
				if err != nil {
					return err
				}

				for _, task := range tasks {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", task.ID, task.PromptFile)
				}

				return nil
			case "runs":
				records, err := store.ListRecords()
				if err != nil {
					return err
				}

				groups := compare.GroupByTask(records)
				for _, group := range groups {
					if taskID != "" && group.TaskID != taskID {
						continue
					}

					printTaskGroup(cmd.OutOrStdout(), group)
				}

				return nil
			default:
				return fmt.Errorf("not yet implemented: list %s", args[0])
			}
		},
	}

	cmd.Flags().StringVar(&taskID, "task", "", "filter 'list runs' to a single task id")

	return cmd
}
