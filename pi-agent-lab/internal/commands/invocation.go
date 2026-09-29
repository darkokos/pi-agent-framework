package commands

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	invoke "github.com/darkokos/pi-agent-framework/pi-agent-invoke"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func confirm(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)

	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		return false, err
	}

	switch strings.TrimSpace(line) {
	case "y", "Y", "yes", "YES":
		return true, nil
	default:
		return false, nil
	}
}

func NewInvocationCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "invocation <agent-id>@<version-id> <task-id>",
		Short: "Run an agent version against a task, after confirming the command",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeAgentVersionCompound(cmd, args, toComplete)
			}

			return completeTaskIDs(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID, versionID, err := parseAgentVersionRef(args[0])
			if err != nil {
				return err
			}

			taskID := args[1]

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			version, err := registry.GetAgentVersion(root, agentID, versionID)
			if err != nil {
				return err
			}

			if _, err := registry.GetTask(root, taskID); err != nil {
				return err
			}

			systemPrompt, err := registry.ReadVersionSystemPrompt(root, agentID, versionID)
			if err != nil {
				return err
			}

			taskPromptPath := registry.TaskPromptPath(root, taskID)
			line, err := invoke.BuildInvocation(root, agentID, versionID, taskID, systemPrompt, taskPromptPath, version)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), line)

			ok, err := confirm(cmd, "run this command? [y/N] ")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(cmd.OutOrStdout(), "aborted")

				return nil
			}

			run := exec.Command("sh", "-c", line)
			run.Stdin = cmd.InOrStdin()
			run.Stdout = cmd.OutOrStdout()
			run.Stderr = cmd.ErrOrStderr()

			return run.Run()
		},
	}
}
