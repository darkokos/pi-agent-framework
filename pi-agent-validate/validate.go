package validate

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type ModelEntry struct {
	Provider string
	Model    string
}

var SystemPromptModes = []string{"replace", "append"}

var ThinkingLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}

var ToolsModes = []string{"all", "none", "extensions-only", "allowlist", "denylist"}

var BuiltinTools = []string{"read", "bash", "powershell", "edit", "write", "grep", "find", "ls"}

func ThinkingLevel(level string) bool {
	return slices.Contains(ThinkingLevels, level)
}

func ListModels() ([]ModelEntry, error) {
	out, err := exec.Command("pi", "--list-models").Output()
	if err != nil {
		return nil, fmt.Errorf("running pi --list-models: %w", err)
	}

	var entries []ModelEntry
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

		entries = append(entries, ModelEntry{Provider: fields[0], Model: fields[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !headerSeen {
		return nil, fmt.Errorf("unexpected output from pi --list-models: %s", strings.TrimSpace(string(out)))
	}

	return entries, nil
}

func ModelProvider(model, provider string) error {
	if model == "" && provider == "" {
		return nil
	}

	entries, err := ListModels()
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

func ParseModel(pattern string) (model, provider string, err error) {
	entries, err := ListModels()
	if err != nil {
		return "", "", err
	}

	if p, m, ok := strings.Cut(pattern, "/"); ok {
		for _, e := range entries {
			if e.Model == m && e.Provider == p {
				return m, p, nil
			}
		}

		return "", "", fmt.Errorf("no model %q for provider %q found in pi --list-models output", m, p)
	}

	matches := ModelsForID(entries, pattern)
	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("model %q not found in pi --list-models output", pattern)
	case 1:
		return pattern, matches[0].Provider, nil
	default:
		var providers []string
		for _, e := range matches {
			providers = append(providers, e.Provider)
		}

		return "", "", fmt.Errorf(
			"model %q is offered by multiple providers (%s); disambiguate with provider/model, e.g. %s/%s",
			pattern, strings.Join(providers, ", "), providers[0], pattern,
		)
	}
}

func ModelsForID(entries []ModelEntry, id string) []ModelEntry {
	var out []ModelEntry
	for _, e := range entries {
		if e.Model == id {
			out = append(out, e)
		}
	}

	return out
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

func packageHasExtensionsDir(dir string) bool {
	entries, err := os.ReadDir(filepath.Join(dir, "extensions"))
	if err != nil {
		return false
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			for _, index := range []string{"index.ts", "index.js"} {
				if _, err := os.Stat(filepath.Join(dir, "extensions", name, index)); err == nil {
					return true
				}
			}

			continue
		}

		if strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".js") {
			return true
		}
	}

	return false
}

func ExtensionPath(path string) bool {
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

	return packageDeclaresExtensions(path) || packageHasExtensionsDir(path)
}

func SkillPath(path string) bool {
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

func GlobalExtensionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".pi", "agent", "extensions"), nil
}

func GlobalSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".pi", "agent", "skills"), nil
}

func SettingsAbsolutePackagePaths() []string {
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
