package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func NewExportCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "export <session-id> <dest>",
		Short: "Export a finished run's record, agent/task metadata, and final response to a directory",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeSessionIDs(cmd, args, toComplete)
			}

			return completeFilePaths(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID, dest := args[0], args[1]

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			record, err := registry.GetRecord(root, sessionID)
			if err != nil {
				return err
			}
			if record.AgentVersion == nil || record.TaskId == nil {
				return fmt.Errorf("session %q has no agent/task metadata recorded (PI_AGENT_VERSION/PI_TASK_ID were unset), nothing to export", sessionID)
			}

			agentID, versionID, err := parseAgentVersionRef(*record.AgentVersion)
			if err != nil {
				return fmt.Errorf("record's agentVersion %q: %w", *record.AgentVersion, err)
			}

			taskID := *record.TaskId
			if err := registry.ExportRun(root, dest, sessionID, agentID, versionID, taskID); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "exported session %s to %s\n", sessionID, dest)

			return nil
		},
	}
}
