package invoke

import (
	"fmt"
	"strings"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func BuildInvocation(agentID, versionID, taskID, systemPrompt, taskPromptPath string, version *registry.AgentVersion) (string, error) {
	parts := []string{
		"PI_AGENT_VERSION=" + shellQuote(agentID+"@"+versionID),
		"PI_TASK_ID=" + shellQuote(taskID),
		"PI_RUN_ID=$(uuidgen)",
		"pi",
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
	for _, p := range version.Extensions.Paths {
		parts = append(parts, "-e", shellQuote(p))
	}

	if version.Skills.Isolate {
		parts = append(parts, "--no-skills")
	}
	for _, p := range version.Skills.Paths {
		parts = append(parts, "--skill", shellQuote(p))
	}

	systemPromptFlag := "--system-prompt"
	if version.SystemPrompt.Mode == "append" {
		systemPromptFlag = "--append-system-prompt"
	}

	parts = append(parts, systemPromptFlag, shellQuote(systemPrompt))

	parts = append(parts, shellQuote("@"+taskPromptPath))

	return strings.Join(parts, " "), nil
}
