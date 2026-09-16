package commands

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
	"github.com/darkokos/pi-agent-framework/pi-agent-validate"
)

func completeThinkingLevels(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return validate.ThinkingLevels, cobra.ShellCompDirectiveNoFileComp
}

func completeToolsModes(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return validate.ToolsModes, cobra.ShellCompDirectiveNoFileComp
}

func completeToolsList(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if mode, _ := cmd.Flags().GetString("tools-mode"); mode != "allowlist" && mode != "denylist" {
		return cobra.AppendActiveHelp(nil, "requires --tools-mode allowlist or denylist"), cobra.ShellCompDirectiveNoFileComp
	}

	prefix, last := "", toComplete
	if idx := strings.LastIndex(toComplete, ","); idx >= 0 {
		prefix, last = toComplete[:idx+1], toComplete[idx+1:]
	}

	used := map[string]bool{}
	for t := range strings.SplitSeq(strings.TrimSuffix(prefix, ","), ",") {
		used[t] = true
	}

	var out []string
	for _, t := range validate.BuiltinTools {
		if used[t] {
			continue
		}

		if strings.HasPrefix(t, last) {
			out = append(out, prefix+t)
		}
	}

	return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func completeModels(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	entries, err := validate.ListModels()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Model != entries[j].Model {
			return entries[i].Model < entries[j].Model
		}

		return entries[i].Provider < entries[j].Provider
	})

	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Provider+"/"+e.Model)
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func completeExtensionPaths(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var out []string

	if dir, err := validate.GlobalExtensionsDir(); err == nil {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if validate.ExtensionPath(p) {
					out = append(out, p)
				}
			}
		}
	}

	for _, p := range validate.SettingsAbsolutePackagePaths() {
		if validate.ExtensionPath(p) {
			out = append(out, p)
		}
	}

	return out, cobra.ShellCompDirectiveDefault
}

func completeSkillPaths(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var out []string

	if dir, err := validate.GlobalSkillsDir(); err == nil {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if validate.SkillPath(p) {
					out = append(out, p)
				}
			}
		}
	}

	return out, cobra.ShellCompDirectiveDefault
}

func completeFilePaths(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func completeAgentIDs(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	root, err := registry.ResolveRoot()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agents, err := registry.ListAgents(root)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var out []string
	for _, a := range agents {
		out = append(out, a.ID)
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func completeTaskIDs(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	root, err := registry.ResolveRoot()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	tasks, err := registry.ListTasks(root)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var out []string
	for _, t := range tasks {
		out = append(out, t.ID)
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func completeAgentIDCompoundPrefix(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if strings.Contains(toComplete, "@") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	root, err := registry.ResolveRoot()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agents, err := registry.ListAgents(root)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var out []string
	for _, a := range agents {
		out = append(out, a.ID+"@")
	}

	return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func completeAgentVersionCompound(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	root, err := registry.ResolveRoot()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if before, after, ok := strings.Cut(toComplete, "@"); ok {
		agentID, partial := before, after

		agent, err := registry.GetAgent(root, agentID)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var out []string
		for _, v := range agent.Versions {
			if strings.HasPrefix(v, partial) {
				out = append(out, agentID+"@"+v)
			}
		}

		return out, cobra.ShellCompDirectiveNoFileComp
	}

	agents, err := registry.ListAgents(root)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var out []string
	for _, a := range agents {
		out = append(out, a.ID+"@")
	}

	return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}
