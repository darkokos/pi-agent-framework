package invoke

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isPackageSource(s string) bool {
	for _, prefix := range []string{"npm:", "git:", "https://", "http://", "ssh://", "git@"} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}

	return false
}

func resolveExtensionPath(root, source string) string {
	switch {
	case source == "~":
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}

		return source
	case strings.HasPrefix(source, "~/"):
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, source[2:])
		}

		return source
	case filepath.IsAbs(source):
		return source
	default:
		return filepath.Join(root, source)
	}
}

func BuildInvocation(root, agentID, versionID, taskID, systemPrompt, taskPromptPath string, version *registry.AgentVersion) (string, error) {
	parts := []string{
		`id=$(uuidgen) &&`,
		"PI_AGENT_VERSION=" + shellQuote(agentID+"@"+versionID),
		"PI_TASK_ID=" + shellQuote(taskID),
		"pi",
		"--session-id", `"$id"`,
		"--model", shellQuote(version.Model),
	}

	if version.Provider != "" {
		parts = append(parts, "--provider", shellQuote(version.Provider))
	}
	if version.Thinking != "" {
		parts = append(parts, "--thinking", shellQuote(version.Thinking))
	}

	switch version.Tools.Mode {
	case "", "all":
	case "none":
		parts = append(parts, "--no-tools")
	case "extensions-only":
		parts = append(parts, "--no-builtin-tools")
	case "allowlist":
		parts = append(parts, "--tools", shellQuote(strings.Join(version.Tools.Names, ",")))
	case "denylist":
		parts = append(parts, "--exclude-tools", shellQuote(strings.Join(version.Tools.Names, ",")))
	default:
		return "", fmt.Errorf("unknown tools mode %q", version.Tools.Mode)
	}

	if version.Extensions.Isolate {
		parts = append(parts, "--no-extensions")
	}
	for _, source := range version.Extensions.Sources {
		if !isPackageSource(source) {
			source = resolveExtensionPath(root, source)
		}

		parts = append(parts, "-e", shellQuote(source))
	}

	if version.Skills.Isolate {
		parts = append(parts, "--no-skills")
	}
	for _, name := range version.Skills.Names {
		parts = append(parts, "--skill", shellQuote(filepath.Join(registry.VersionSkillsDir(root, agentID, versionID), name)))
	}

	systemPromptFlag := "--system-prompt"
	if version.SystemPrompt.Mode == "append" {
		systemPromptFlag = "--append-system-prompt"
	}

	parts = append(parts, systemPromptFlag, shellQuote(systemPrompt))

	parts = append(parts, shellQuote("@"+taskPromptPath))

	return strings.Join(parts, " "), nil
}

func BuildResume(cwd, sessionFile string, agentVersion, taskID *string) string {
	parts := []string{"cd", shellQuote(cwd), "&&"}

	if agentVersion != nil {
		parts = append(parts, "PI_AGENT_VERSION="+shellQuote(*agentVersion))
	}
	if taskID != nil {
		parts = append(parts, "PI_TASK_ID="+shellQuote(*taskID))
	}

	parts = append(parts, "pi", "--session", shellQuote(sessionFile))

	return strings.Join(parts, " ")
}
