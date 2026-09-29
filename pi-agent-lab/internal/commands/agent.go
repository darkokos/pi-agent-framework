package commands

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/format"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
	"github.com/darkokos/pi-agent-framework/pi-agent-validate"
)

func writeItemList(w io.Writer, label, header string, items []string) {
	fmt.Fprintf(w, "%s\t%s\n", label, header)
	for _, item := range items {
		fmt.Fprintf(w, "\t  - %s\n", item)
	}
}

const systemPromptModeHelp = "replace|append: whether the prompt replaces Pi's default system prompt or is added to it"

func requireValidThinkingLevel(thinking string) error {
	if thinking == "" || validate.ThinkingLevel(thinking) {
		return nil
	}

	return fmt.Errorf("--thinking must be one of %s, got %q", strings.Join(validate.ThinkingLevels, "|"), thinking)
}

func requireValidSystemPromptMode(mode string) error {
	if slices.Contains(validate.SystemPromptModes, mode) {
		return nil
	}

	return fmt.Errorf("--system-prompt-mode must be one of %s, got %q", strings.Join(validate.SystemPromptModes, "|"), mode)
}

func prepareResources(cmd *cobra.Command, toolsMode string, toolNames, extensionSources, skillPaths []string) ([]string, error) {
	warn := func(format string, args ...any) {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: "+format+"\n", args...)
	}
	note := func(format string, args ...any) {
		fmt.Fprintf(cmd.ErrOrStderr(), "note: "+format+"\n", args...)
	}

	for _, source := range extensionSources {
		parsed, err := validate.ParseExtensionSource(source)
		if err != nil {
			return nil, fmt.Errorf("--extension: %w", err)
		}

		if !parsed.Pinned {
			warn("extension %q is not pinned, so runs may change when the package is updated; pin it with an exact version (npm:name@1.2.3) or a git tag or commit (git:host/repo@ref)", source)
		}
		if !parsed.Portable {
			warn("extension %q is a path tied to this machine; it will not resolve if the registry is moved or shared. Consider a path relative to the registry directory (e.g. ../my-extension) instead, or publish it as an npm:/git: package", source)
		}
	}

	if toolsMode == "allowlist" || toolsMode == "denylist" {
		for _, tool := range validate.NonBuiltinTools(toolNames) {
			note("tool %q is not one of Pi's built-in tools; this is expected if an extension you're loading provides it, but Pi will silently ignore it otherwise", tool)
		}
	}

	if len(skillPaths) > 0 && !canReadSkillFiles(toolsMode, toolNames) {
		warn("skills are attached, but the tools mode leaves out both read and bash; Pi only lists skills to the model when one of them is available, so it will not see these skills")
	}

	skills := make([]string, 0, len(skillPaths))
	for _, p := range skillPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}

		if !validate.SkillPath(abs) {
			return nil, fmt.Errorf("--skill %q is not a skill: expected a folder containing SKILL.md, or a .md file with name and description frontmatter", p)
		}

		skills = append(skills, abs)
	}

	return skills, nil
}

func canReadSkillFiles(toolsMode string, toolNames []string) bool {
	switch toolsMode {
	case "none", "extensions-only":
		return false
	case "allowlist":
		return slices.Contains(toolNames, "read") || slices.Contains(toolNames, "bash")
	case "denylist":
		return !slices.Contains(toolNames, "read") || !slices.Contains(toolNames, "bash")
	default:
		return true
	}
}

func parseAgentVersionRef(s string) (agentID, versionID string, err error) {
	agentID, versionID, ok := strings.Cut(s, "@")
	if !ok {
		return "", "", fmt.Errorf("expected <agent-id>@<version-id>, got %q", s)
	}

	return agentID, versionID, nil
}

func newCreateAgentCommand() *cobra.Command {
	var (
		name              string
		description       string
		versionID         string
		systemPromptFile  string
		systemPromptMode  string
		model             string
		thinking          string
		notes             string
		toolsMode         string
		toolNames         []string
		extensionsIsolate bool
		extensionSources  []string
		skillsIsolate     bool
		skillPaths        []string
	)

	cmd := &cobra.Command{
		Use:   "agent <agent-id>",
		Short: "Register a new agent, along with its first version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireValidSystemPromptMode(systemPromptMode); err != nil {
				return err
			}
			if !slices.Contains(validate.ToolsModes, toolsMode) {
				return fmt.Errorf("--tools-mode must be one of %s, got %q", strings.Join(validate.ToolsModes, "|"), toolsMode)
			}
			if err := requireValidThinkingLevel(thinking); err != nil {
				return err
			}
			resolvedModel, resolvedProvider, err := validate.ParseModel(model)
			if err != nil {
				return err
			}
			skills, err := prepareResources(cmd, toolsMode, toolNames, extensionSources, skillPaths)
			if err != nil {
				return err
			}

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			agent, err := registry.CreateAgent(root, args[0], name, description)
			if err != nil {
				return err
			}

			version, err := registry.CreateAgentVersion(root, agent.ID, versionID, registry.AgentVersionInput{
				SystemPromptFile: systemPromptFile,
				SystemPromptMode: systemPromptMode,
				Model:            resolvedModel,
				Provider:         resolvedProvider,
				Thinking:         thinking,
				Notes:            notes,
				Tools: registry.ToolsConfig{
					Mode:  toolsMode,
					Names: toolNames,
				},
				ExtensionsIsolate: extensionsIsolate,
				ExtensionSources:  extensionSources,
				SkillsIsolate:     skillsIsolate,
				SkillPaths:        skills,
			})
			if err != nil {
				_ = registry.DeleteAgent(root, agent.ID)

				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created agent %q with version %q\n", agent.ID, version.VersionID)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "human-readable agent name")
	cmd.Flags().StringVar(&description, "description", "", "what this agent is for")
	cmd.Flags().StringVar(&versionID, "initial-version-id", "", "id for the agent's first version (required)")
	cmd.Flags().StringVar(&systemPromptFile, "system-prompt-file", "", "path to the system prompt text (required)")
	cmd.Flags().StringVar(&systemPromptMode, "system-prompt-mode", "replace", systemPromptModeHelp)
	cmd.Flags().StringVar(&model, "model", "", "model id, or provider/model when the model is offered by more than one provider (required)")
	cmd.Flags().StringVar(&thinking, "thinking", "", "thinking level: off|minimal|low|medium|high|xhigh|max")
	cmd.Flags().StringVar(&notes, "notes", "", "freeform notes about this version")
	cmd.Flags().StringVar(&toolsMode, "tools-mode", "all", "all|none|extensions-only|allowlist|denylist")
	cmd.Flags().StringSliceVar(&toolNames, "tools", nil, "comma-separated tool names, used with --tools-mode allowlist|denylist")
	cmd.Flags().BoolVar(&extensionsIsolate, "extensions-isolate", false, "don't load extensions from your Pi setup; the ones added with --extension still load")
	cmd.Flags().StringArrayVar(&extensionSources, "extension", nil, "extension to load, repeatable: a package reference (npm:name@1.2.3 or git:host/repo@ref, pinned to a version), a path relative to the registry directory (see `pi-agent-lab config show`), or an absolute path (works only on this machine)")
	cmd.Flags().BoolVar(&skillsIsolate, "skills-isolate", false, "don't load skills from your Pi setup; the skills added with --skill still load")
	cmd.Flags().StringArrayVar(&skillPaths, "skill", nil, "path to a skill folder or file, repeatable; it is copied into the registry so the version stays portable")
	_ = cmd.MarkFlagRequired("initial-version-id")
	_ = cmd.MarkFlagRequired("system-prompt-file")
	_ = cmd.MarkFlagRequired("model")

	_ = cmd.RegisterFlagCompletionFunc("system-prompt-file", completeFilePaths)
	_ = cmd.RegisterFlagCompletionFunc("system-prompt-mode", completeSystemPromptModes)
	_ = cmd.RegisterFlagCompletionFunc("model", completeModels)
	_ = cmd.RegisterFlagCompletionFunc("thinking", completeThinkingLevels)
	_ = cmd.RegisterFlagCompletionFunc("tools-mode", completeToolsModes)
	_ = cmd.RegisterFlagCompletionFunc("tools", completeToolsList)
	_ = cmd.RegisterFlagCompletionFunc("extension", completeExtensionSources)
	_ = cmd.RegisterFlagCompletionFunc("skill", completeSkillPaths)

	return cmd
}

func newCreateAgentVersionCommand() *cobra.Command {
	var (
		systemPromptFile  string
		systemPromptMode  string
		model             string
		thinking          string
		notes             string
		toolsMode         string
		toolNames         []string
		extensionsIsolate bool
		extensionSources  []string
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
			if err := requireValidSystemPromptMode(systemPromptMode); err != nil {
				return err
			}
			if !slices.Contains(validate.ToolsModes, toolsMode) {
				return fmt.Errorf("--tools-mode must be one of %s, got %q", strings.Join(validate.ToolsModes, "|"), toolsMode)
			}
			if err := requireValidThinkingLevel(thinking); err != nil {
				return err
			}
			resolvedModel, resolvedProvider, err := validate.ParseModel(model)
			if err != nil {
				return err
			}
			skills, err := prepareResources(cmd, toolsMode, toolNames, extensionSources, skillPaths)
			if err != nil {
				return err
			}

			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			version, err := registry.CreateAgentVersion(root, agentID, versionID, registry.AgentVersionInput{
				SystemPromptFile: systemPromptFile,
				SystemPromptMode: systemPromptMode,
				Model:            resolvedModel,
				Provider:         resolvedProvider,
				Thinking:         thinking,
				Notes:            notes,
				Tools: registry.ToolsConfig{
					Mode:  toolsMode,
					Names: toolNames,
				},
				ExtensionsIsolate: extensionsIsolate,
				ExtensionSources:  extensionSources,
				SkillsIsolate:     skillsIsolate,
				SkillPaths:        skills,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created version %q for agent %q\n", version.VersionID, version.AgentID)

			return nil
		},
	}

	cmd.Flags().StringVar(&systemPromptFile, "system-prompt-file", "", "path to the system prompt text (required)")
	cmd.Flags().StringVar(&systemPromptMode, "system-prompt-mode", "replace", systemPromptModeHelp)
	cmd.Flags().StringVar(&model, "model", "", "model id, or provider/model when the model is offered by more than one provider (required)")
	cmd.Flags().StringVar(&thinking, "thinking", "", "thinking level: off|minimal|low|medium|high|xhigh|max")
	cmd.Flags().StringVar(&notes, "notes", "", "freeform notes about this version")
	cmd.Flags().StringVar(&toolsMode, "tools-mode", "all", "all|none|extensions-only|allowlist|denylist")
	cmd.Flags().StringSliceVar(&toolNames, "tools", nil, "comma-separated tool names, used with --tools-mode allowlist|denylist")
	cmd.Flags().BoolVar(&extensionsIsolate, "extensions-isolate", false, "don't load extensions from your Pi setup; the ones added with --extension still load")
	cmd.Flags().StringArrayVar(&extensionSources, "extension", nil, "extension to load, repeatable: a package reference (npm:name@1.2.3 or git:host/repo@ref, pinned to a version), a path relative to the registry directory (see `pi-agent-lab config show`), or an absolute path (works only on this machine)")
	cmd.Flags().BoolVar(&skillsIsolate, "skills-isolate", false, "don't load skills from your Pi setup; the skills added with --skill still load")
	cmd.Flags().StringArrayVar(&skillPaths, "skill", nil, "path to a skill folder or file, repeatable; it is copied into the registry so the version stays portable")
	_ = cmd.MarkFlagRequired("system-prompt-file")
	_ = cmd.MarkFlagRequired("model")

	_ = cmd.RegisterFlagCompletionFunc("system-prompt-file", completeFilePaths)
	_ = cmd.RegisterFlagCompletionFunc("system-prompt-mode", completeSystemPromptModes)
	_ = cmd.RegisterFlagCompletionFunc("model", completeModels)
	_ = cmd.RegisterFlagCompletionFunc("thinking", completeThinkingLevels)
	_ = cmd.RegisterFlagCompletionFunc("tools-mode", completeToolsModes)
	_ = cmd.RegisterFlagCompletionFunc("tools", completeToolsList)
	_ = cmd.RegisterFlagCompletionFunc("extension", completeExtensionSources)
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
			fmt.Fprintln(tw, "VERSION ID\tMODEL\tPROVIDER\tTHINKING\tTOOLS MODE\tNOTES")
			for _, versionID := range agent.Versions {
				version, err := registry.GetAgentVersion(root, agent.ID, versionID)
				if err != nil {
					return err
				}

				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
					version.VersionID, version.Model, version.Provider, version.Thinking,
					version.Tools.Mode, version.Notes,
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
			fmt.Fprintf(tw, "Model:\t%s\n", version.Model)
			fmt.Fprintf(tw, "Provider:\t%s\n", version.Provider)
			fmt.Fprintf(tw, "Thinking:\t%s\n", version.Thinking)
			fmt.Fprintf(tw, "System Prompt:\t%s (%s)\n", version.SystemPrompt.File, version.SystemPrompt.Mode)
			writeItemList(tw, "Tools:", version.Tools.Mode, version.Tools.Names)
			writeItemList(tw, "Extensions:", fmt.Sprintf("isolate=%t", version.Extensions.Isolate), version.Extensions.Sources)
			writeItemList(tw, "Skills:", fmt.Sprintf("isolate=%t", version.Skills.Isolate), version.Skills.Names)
			fmt.Fprintf(tw, "Notes:\t%s\n", version.Notes)

			return tw.Flush()
		},
	}
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
