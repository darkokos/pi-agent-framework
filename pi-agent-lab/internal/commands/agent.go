package commands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/format"
	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/validate"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func requireValidThinkingLevel(thinking string) error {
	if thinking == "" || validate.ThinkingLevel(thinking) {
		return nil
	}

	return fmt.Errorf("--thinking must be one of %s, got %q", strings.Join(validate.ThinkingLevels, "|"), thinking)
}

func requireValidPaths(flag string, paths []string, valid func(string) bool) error {
	for _, p := range paths {
		if !valid(p) {
			return fmt.Errorf("--%s %q is not a valid path", flag, p)
		}
	}

	return nil
}

func parseAgentVersionRef(s string) (agentID, versionID string, err error) {
	agentID, versionID, ok := strings.Cut(s, "@")
	if !ok {
		return "", "", fmt.Errorf("expected <agent-id>@<version-id>, got %q", s)
	}

	return agentID, versionID, nil
}

func newCreateAgentCommand() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "agent <agent-id>",
		Short: "Register a new agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			agent, err := registry.CreateAgent(root, args[0], name, description)
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

func newCreateAgentVersionCommand() *cobra.Command {
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
		Use:               "agent-version <agent-id>@<version-id>",
		Short:             "Add a new version to an existing agent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeAgentIDCompoundPrefix,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID, versionID, err := parseAgentVersionRef(args[0])
			if err != nil {
				return err
			}
			if !slices.Contains(validate.ToolsModes, toolsMode) {
				return fmt.Errorf("--tools-mode must be one of %s, got %q", strings.Join(validate.ToolsModes, "|"), toolsMode)
			}
			if err := requireValidThinkingLevel(thinking); err != nil {
				return err
			}
			if err := validate.ModelProvider(model, provider); err != nil {
				return err
			}
			if err := requireValidPaths("extension", extensionPaths, validate.ExtensionPath); err != nil {
				return err
			}
			if err := requireValidPaths("skill", skillPaths, validate.SkillPath); err != nil {
				return err
			}

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			version, err := registry.CreateAgentVersion(root, agentID, versionID, registry.AgentVersionInput{
				SystemPromptFile: systemPromptFile,
				Model:            model,
				Provider:         provider,
				Thinking:         thinking,
				Notes:            notes,
				Tools: registry.ToolsConfig{
					Mode:  toolsMode,
					Names: toolNames,
				},
				Extensions: registry.ResourceIsolation{
					Isolate: extensionsIsolate,
					Paths:   extensionPaths,
				},
				Skills: registry.ResourceIsolation{
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
	cmd.Flags().BoolVar(&extensionsIsolate, "extensions-isolate", false, "pass --no-extensions and pin only --extension paths given below (pi-session-record is re-added automatically, see invocation)")
	cmd.Flags().StringArrayVar(&extensionPaths, "extension", nil, "extension path to load under isolation, repeatable")
	cmd.Flags().BoolVar(&skillsIsolate, "skills-isolate", false, "pass --no-skills and pin only --skill paths given below")
	cmd.Flags().StringArrayVar(&skillPaths, "skill", nil, "skill path to load under isolation, repeatable")
	_ = cmd.MarkFlagRequired("system-prompt-file")
	_ = cmd.MarkFlagRequired("model")

	_ = cmd.RegisterFlagCompletionFunc("system-prompt-file", completeFilePaths)
	_ = cmd.RegisterFlagCompletionFunc("model", completeModels)
	_ = cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	_ = cmd.RegisterFlagCompletionFunc("thinking", completeThinkingLevels)
	_ = cmd.RegisterFlagCompletionFunc("tools-mode", completeToolsModes)
	_ = cmd.RegisterFlagCompletionFunc("tools", completeToolsList)
	_ = cmd.RegisterFlagCompletionFunc("extension", completeExtensionPaths)
	_ = cmd.RegisterFlagCompletionFunc("skill", completeSkillPaths)

	return cmd
}

func newListAgentsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List registered agents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			agents, err := registry.ListAgents(root)
			if err != nil {
				return err
			}

			tw := format.NewTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "ID\tNAME\tDESCRIPTION\tVERSIONS")
			for _, agent := range agents {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", agent.ID, agent.Name, agent.Description, strings.Join(agent.Versions, ", "))
			}

			return tw.Flush()
		},
	}
}

func newListAgentVersionsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-versions <agent-id>",
		Short: "List the versions of an agent",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeAgentIDs(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			agent, err := registry.GetAgent(root, args[0])
			if err != nil {
				return err
			}

			tw := format.NewTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "VERSION ID\tMODEL\tPROVIDER\tTHINKING\tTOOLS MODE\tNOTES\tCREATED AT")
			for _, versionID := range agent.Versions {
				version, err := registry.GetAgentVersion(root, agent.ID, versionID)
				if err != nil {
					return err
				}

				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					version.VersionID, version.Model, version.Provider, version.Thinking,
					version.Tools.Mode, version.Notes, format.UnixMilli(version.CreatedAt),
				)
			}

			return tw.Flush()
		},
	}
}

func newShowAgentCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "agent <agent-id>",
		Short: "Show full details of a single agent",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return completeAgentIDs(cmd, args, toComplete)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			agent, err := registry.GetAgent(root, args[0])
			if err != nil {
				return err
			}

			tw := format.NewTabWriter(cmd.OutOrStdout())
			fmt.Fprintf(tw, "ID:\t%s\n", agent.ID)
			fmt.Fprintf(tw, "Name:\t%s\n", agent.Name)
			fmt.Fprintf(tw, "Description:\t%s\n", agent.Description)
			fmt.Fprintf(tw, "Created At:\t%s\n", format.UnixMilli(agent.CreatedAt))
			fmt.Fprintf(tw, "Versions:\t%s\n", strings.Join(agent.Versions, ", "))

			return tw.Flush()
		},
	}
}

func newShowAgentVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "agent-version <agent-id>@<version-id>",
		Short:             "Show full details of a single agent version",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeAgentVersionCompound,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID, versionID, err := parseAgentVersionRef(args[0])
			if err != nil {
				return err
			}

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			version, err := registry.GetAgentVersion(root, agentID, versionID)
			if err != nil {
				return err
			}

			tw := format.NewTabWriter(cmd.OutOrStdout())
			fmt.Fprintf(tw, "Agent ID:\t%s\n", version.AgentID)
			fmt.Fprintf(tw, "Version ID:\t%s\n", version.VersionID)
			fmt.Fprintf(tw, "Created At:\t%s\n", format.UnixMilli(version.CreatedAt))
			fmt.Fprintf(tw, "Model:\t%s\n", version.Model)
			fmt.Fprintf(tw, "Provider:\t%s\n", version.Provider)
			fmt.Fprintf(tw, "Thinking:\t%s\n", version.Thinking)
			fmt.Fprintf(tw, "System Prompt:\t%s (%s)\n", version.SystemPrompt.File, version.SystemPrompt.Mode)
			fmt.Fprintf(tw, "Tools:\t%s %s\n", version.Tools.Mode, strings.Join(version.Tools.Names, ", "))
			fmt.Fprintf(tw, "Extensions:\t isolate=%t %s\n", version.Extensions.Isolate, strings.Join(version.Extensions.Paths, ", "))
			fmt.Fprintf(tw, "Skills:\t isolate=%t %s\n", version.Skills.Isolate, strings.Join(version.Skills.Paths, ", "))
			fmt.Fprintf(tw, "Notes:\t%s\n", version.Notes)

			return tw.Flush()
		},
	}
}

func newUpdateAgentCommand() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:               "agent <agent-id>",
		Short:             "Update fields of an existing agent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeAgentIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			agent, err := registry.GetAgent(root, args[0])
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("name") {
				agent.Name = name
			}
			if cmd.Flags().Changed("description") {
				agent.Description = description
			}

			updated, err := registry.ReplaceAgent(root, agent)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "updated agent %q\n", updated.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "human-readable agent name")
	cmd.Flags().StringVar(&description, "description", "", "what this agent is for")

	return cmd
}

func newUpdateAgentVersionCommand() *cobra.Command {
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
		Use:               "agent-version <agent-id>@<version-id>",
		Short:             "Update fields of an existing agent version",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeAgentVersionCompound,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID, versionID, err := parseAgentVersionRef(args[0])
			if err != nil {
				return err
			}

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			version, err := registry.GetAgentVersion(root, agentID, versionID)
			if err != nil {
				return err
			}

			flags := cmd.Flags()

			newModel, newProvider := version.Model, version.Provider
			if flags.Changed("model") {
				newModel = model
			}
			if flags.Changed("provider") {
				newProvider = provider
			}
			if flags.Changed("model") || flags.Changed("provider") {
				if err := validate.ModelProvider(newModel, newProvider); err != nil {
					return err
				}
			}
			version.Model, version.Provider = newModel, newProvider

			if flags.Changed("thinking") {
				if err := requireValidThinkingLevel(thinking); err != nil {
					return err
				}

				version.Thinking = thinking
			}
			if flags.Changed("notes") {
				version.Notes = notes
			}
			if flags.Changed("tools-mode") {
				if !slices.Contains(validate.ToolsModes, toolsMode) {
					return fmt.Errorf("--tools-mode must be one of %s, got %q", strings.Join(validate.ToolsModes, "|"), toolsMode)
				}

				version.Tools.Mode = toolsMode
			}
			if flags.Changed("tools") {
				version.Tools.Names = toolNames
			}
			if flags.Changed("extensions-isolate") {
				version.Extensions.Isolate = extensionsIsolate
			}
			if flags.Changed("extension") {
				if err := requireValidPaths("extension", extensionPaths, validate.ExtensionPath); err != nil {
					return err
				}

				version.Extensions.Paths = extensionPaths
			}
			if flags.Changed("skills-isolate") {
				version.Skills.Isolate = skillsIsolate
			}
			if flags.Changed("skill") {
				if err := requireValidPaths("skill", skillPaths, validate.SkillPath); err != nil {
					return err
				}

				version.Skills.Paths = skillPaths
			}

			newSystemPromptFile := ""
			if flags.Changed("system-prompt-file") {
				newSystemPromptFile = systemPromptFile
			}

			updated, err := registry.ReplaceAgentVersion(root, agentID, versionID, version, newSystemPromptFile)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "updated version %q for agent %q\n", updated.VersionID, updated.AgentID)

			return nil
		},
	}

	cmd.Flags().StringVar(&systemPromptFile, "system-prompt-file", "", "path to the system prompt text")
	cmd.Flags().StringVar(&model, "model", "", "model pattern, e.g. anthropic/claude-sonnet-4-5")
	cmd.Flags().StringVar(&provider, "provider", "", "provider name")
	cmd.Flags().StringVar(&thinking, "thinking", "", "thinking level: off|minimal|low|medium|high|xhigh|max")
	cmd.Flags().StringVar(&notes, "notes", "", "freeform notes about this version")
	cmd.Flags().StringVar(&toolsMode, "tools-mode", "", "all|none|extensions-only|allowlist|denylist")
	cmd.Flags().StringSliceVar(&toolNames, "tools", nil, "comma-separated tool names, used with --tools-mode allowlist|denylist")
	cmd.Flags().BoolVar(&extensionsIsolate, "extensions-isolate", false, "pass --no-extensions and pin only --extension paths given below")
	cmd.Flags().StringArrayVar(&extensionPaths, "extension", nil, "extension path to load under isolation, repeatable")
	cmd.Flags().BoolVar(&skillsIsolate, "skills-isolate", false, "pass --no-skills and pin only --skill paths given below")
	cmd.Flags().StringArrayVar(&skillPaths, "skill", nil, "skill path to load under isolation, repeatable")

	_ = cmd.RegisterFlagCompletionFunc("system-prompt-file", completeFilePaths)
	_ = cmd.RegisterFlagCompletionFunc("model", completeModels)
	_ = cmd.RegisterFlagCompletionFunc("provider", completeProviders)
	_ = cmd.RegisterFlagCompletionFunc("thinking", completeThinkingLevels)
	_ = cmd.RegisterFlagCompletionFunc("tools-mode", completeToolsModes)
	_ = cmd.RegisterFlagCompletionFunc("tools", completeToolsList)
	_ = cmd.RegisterFlagCompletionFunc("extension", completeExtensionPaths)
	_ = cmd.RegisterFlagCompletionFunc("skill", completeSkillPaths)

	return cmd
}

func newDeleteAgentCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "agent <agent-id>",
		Short:             "Delete an agent and all its versions",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeAgentIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			if err := registry.DeleteAgent(root, args[0]); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "deleted agent %q\n", args[0])

			return nil
		},
	}
}

func newDeleteAgentVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "agent-version <agent-id>@<version-id>",
		Short:             "Delete a single agent version",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeAgentVersionCompound,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID, versionID, err := parseAgentVersionRef(args[0])
			if err != nil {
				return err
			}

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			if err := registry.DeleteAgentVersion(root, agentID, versionID); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "deleted version %q for agent %q\n", versionID, agentID)

			return nil
		},
	}
}
