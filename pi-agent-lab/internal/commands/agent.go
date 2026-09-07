package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/store"
)

var validToolsModes = map[string]bool{
	"all":             true,
	"none":            true,
	"extensions-only": true,
	"allowlist":       true,
	"denylist":        true,
}

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
			agent, err := store.CreateAgent(args[0], name, description)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created agent %q\n", agent.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "human-readable agent name")
	cmd.Flags().StringVar(&description, "description", "", "what this agent is for")

	return cmd
}

func newAgentVersionCommand() *cobra.Command {
	var (
		systemPromptFile  string
		model             string
		provider          string
		thinking          string
		notes             string
		toolsMode         string
		toolNames         []string
		extensionsIsolate bool
		extensionPaths    []string
		skillsIsolate     bool
		skillPaths        []string
	)

	cmd := &cobra.Command{
		Use:   "version <agent-id> <version-id>",
		Short: "Add a new version to an existing agent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !validToolsModes[toolsMode] {
				return fmt.Errorf("--tools-mode must be one of all|none|extensions-only|allowlist|denylist, got %q", toolsMode)
			}

			version, err := store.CreateAgentVersion(args[0], args[1], store.AgentVersionInput{
				SystemPromptFile: systemPromptFile,
				Model:            model,
				Provider:         provider,
				Thinking:         thinking,
				Notes:            notes,
				Tools: store.ToolsConfig{
					Mode:  toolsMode,
					Names: toolNames,
				},
				Extensions: store.ResourceIsolation{
					Isolate: extensionsIsolate,
					Paths:   extensionPaths,
				},
				Skills: store.ResourceIsolation{
					Isolate: skillsIsolate,
					Paths:   skillPaths,
				},
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created version %q for agent %q\n", version.VersionID, version.AgentID)

			return nil
		},
	}

	cmd.Flags().StringVar(&systemPromptFile, "system-prompt-file", "", "path to the system prompt text (required)")
	cmd.Flags().StringVar(&model, "model", "", "model pattern, e.g. anthropic/claude-sonnet-4-5 (required)")
	cmd.Flags().StringVar(&provider, "provider", "", "provider name")
	cmd.Flags().StringVar(&thinking, "thinking", "", "thinking level: off|minimal|low|medium|high|xhigh|max")
	cmd.Flags().StringVar(&notes, "notes", "", "freeform notes about this version")
	cmd.Flags().StringVar(&toolsMode, "tools-mode", "all", "all|none|extensions-only|allowlist|denylist")
	cmd.Flags().StringSliceVar(&toolNames, "tools", nil, "comma-separated tool names, used with --tools-mode allowlist|denylist")
	cmd.Flags().BoolVar(&extensionsIsolate, "extensions-isolate", false, "pass --no-extensions and pin only --extension paths given below (pi-session-record is re-added automatically, see show)")
	cmd.Flags().StringArrayVar(&extensionPaths, "extension", nil, "extension path to load under isolation, repeatable")
	cmd.Flags().BoolVar(&skillsIsolate, "skills-isolate", false, "pass --no-skills and pin only --skill paths given below")
	cmd.Flags().StringArrayVar(&skillPaths, "skill", nil, "skill path to load under isolation, repeatable")
	_ = cmd.MarkFlagRequired("system-prompt-file")
	_ = cmd.MarkFlagRequired("model")

	return cmd
}
