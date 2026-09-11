package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

type modelEntry struct {
	Provider string
	Model    string
}

var thinkingLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}

var toolsModes = []string{"all", "none", "extensions-only", "allowlist", "denylist"}

var builtinTools = []string{"read", "bash", "powershell", "edit", "write", "grep", "find", "ls"}

func isValidThinkingLevel(level string) bool {
	return slices.Contains(thinkingLevels, level)
}

func listModels() ([]modelEntry, error) {
	out, err := exec.Command("pi", "--list-models").Output()
	if err != nil {
		return nil, fmt.Errorf("running pi --list-models: %w", err)
	}

	var entries []modelEntry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	headerSeen := false
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if !headerSeen {
			if len(fields) >= 2 && fields[0] == "provider" && fields[1] == "model" {
				headerSeen = true
			}

			continue
		}

		if len(fields) < 2 {
			continue
		}

		entries = append(entries, modelEntry{Provider: fields[0], Model: fields[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !headerSeen {
		return nil, fmt.Errorf("unexpected output from pi --list-models: %s", strings.TrimSpace(string(out)))
	}

	return entries, nil
}

func validateModelProvider(model, provider string) error {
	if model == "" && provider == "" {
		return nil
	}

	entries, err := listModels()
	if err != nil {
		return err
	}

	for _, e := range entries {
		if (model == "" || e.Model == model) && (provider == "" || e.Provider == provider) {
			return nil
		}
	}

	if provider != "" {
		return fmt.Errorf("no model %q for provider %q found in pi --list-models output", model, provider)
	}

	return fmt.Errorf("model %q not found in pi --list-models output", model)
}

func completeThinkingLevels(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return thinkingLevels, cobra.ShellCompDirectiveNoFileComp
}

func completeToolsModes(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return toolsModes, cobra.ShellCompDirectiveNoFileComp
}

func completeToolsList(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	prefix, last := "", toComplete
	if idx := strings.LastIndex(toComplete, ","); idx >= 0 {
		prefix, last = toComplete[:idx+1], toComplete[idx+1:]
	}

	var out []string
	for _, t := range builtinTools {
		if strings.HasPrefix(t, last) {
			out = append(out, prefix+t)
		}
	}

	return out, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func completeModels(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	entries, err := listModels()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	seen := map[string]bool{}
	var names []string
	for _, e := range entries {
		if !seen[e.Model] {
			seen[e.Model] = true
			names = append(names, e.Model)
		}
	}
	sort.Strings(names)

	return names, cobra.ShellCompDirectiveNoFileComp
}

func completeProviders(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	entries, err := listModels()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	seen := map[string]bool{}
	var names []string
	for _, e := range entries {
		if !seen[e.Provider] {
			seen[e.Provider] = true
			names = append(names, e.Provider)
		}
	}
	sort.Strings(names)

	return names, cobra.ShellCompDirectiveNoFileComp
}

func isValidExtensionPath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if !info.IsDir() {
		return strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".js")
	}

	for _, name := range []string{"index.ts", "index.js"} {
		if _, err := os.Stat(filepath.Join(path, name)); err == nil {
			return true
		}
	}

	return packageDeclaresExtensions(path)
}

func packageDeclaresExtensions(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}

	var pkg struct {
		Pi struct {
			Extensions json.RawMessage `json:"extensions"`
		} `json:"pi"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false
	}

	return len(pkg.Pi.Extensions) > 0
}

func isValidSkillPath(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if info.IsDir() {
		_, err := os.Stat(filepath.Join(path, "SKILL.md"))

		return err == nil
	}

	return strings.HasSuffix(path, ".md") && hasSkillFrontmatter(path)
}

func hasSkillFrontmatter(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return false
	}

	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return false
	}

	frontmatter := content[4 : 4+end]

	return strings.Contains(frontmatter, "name:") && strings.Contains(frontmatter, "description:")
}

func globalExtensionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".pi", "agent", "extensions"), nil
}

func globalSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".pi", "agent", "skills"), nil
}

func settingsAbsolutePackagePaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "settings.json"))
	if err != nil {
		return nil
	}

	var settings struct {
		Packages []string `json:"packages"`
	}
	if json.Unmarshal(data, &settings) != nil {
		return nil
	}

	var out []string
	for _, pkg := range settings.Packages {
		if filepath.IsAbs(pkg) {
			out = append(out, pkg)
		}
	}

	return out
}

func completeExtensionPaths(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var out []string

	if dir, err := globalExtensionsDir(); err == nil {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if isValidExtensionPath(p) {
					out = append(out, p)
				}
			}
		}
	}

	for _, p := range settingsAbsolutePackagePaths() {
		if isValidExtensionPath(p) {
			out = append(out, p)
		}
	}

	return out, cobra.ShellCompDirectiveDefault
}

func completeSkillPaths(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var out []string

	if dir, err := globalSkillsDir(); err == nil {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if isValidSkillPath(p) {
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

func completeAgentVersions(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
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
		for _, v := range a.Versions {
			out = append(out, a.ID+"@"+v)
		}
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}
