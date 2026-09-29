package registrytui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
	"github.com/darkokos/pi-agent-framework/pi-agent-validate"
)

func discoverModelEntries() []validate.ModelEntry {
	entries, err := validate.ListModels()
	if err != nil {
		return nil
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Model != entries[j].Model {
			return entries[i].Model < entries[j].Model
		}

		return entries[i].Provider < entries[j].Provider
	})

	return entries
}

func discoverExtensionSources() []string {
	return validate.InstalledExtensionSources()
}

func discoverSkillPaths() []string {
	var out []string

	if dir, err := validate.GlobalSkillsDir(); err == nil {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				p := filepath.Join(dir, e.Name())
				if validate.SkillPath(p) {
					out = append(out, p)
				}
			}
		}
	}

	return out
}

func splitCommaList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}

	return out
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

func newAgentCreateForm(root string) *form {
	fields := append([]*formField{
		newTextField("ID", "unique, immutable", "e.g. code-reviewer", ""),
		newTextField("Name", "", "human-readable name", ""),
		newTextField("Description", "", "what this agent is for", ""),
		newTextField("Initial Version ID", "unique within this agent, immutable", "e.g. 1.0.0", ""),
		newPathField("System Prompt File", "required; paths start from your home directory", "~/system-prompt.md", "~/"),
	}, versionSettingsFields(root)...)

	return newForm("Create Agent", fields, func(f *form) error {
		id := f.fields[0].text()
		if id == "" {
			return fmt.Errorf("ID is required")
		}

		versionID := f.fields[3].text()
		if versionID == "" {
			return fmt.Errorf("Initial Version ID is required")
		}

		in, notices, warnings, err := buildVersionInput(&form{fields: f.fields[5:]})
		if err != nil {
			return err
		}

		if in.SystemPromptFile, err = requireFile(f.fields[4]); err != nil {
			return err
		}

		f.notices, f.warnings = notices, warnings

		agent, err := registry.CreateAgent(root, id, f.fields[1].text(), f.fields[2].text())
		if err != nil {
			return err
		}

		if _, err := registry.CreateAgentVersion(root, agent.ID, versionID, in); err != nil {
			_ = registry.DeleteAgent(root, agent.ID)

			return err
		}

		return nil
	})
}

func newTaskCreateForm(root string) *form {
	fields := []*formField{
		newTextField("ID", "unique, immutable", "e.g. refactor-auth", ""),
		newPathField("Prompt File", "required; paths start from your home directory", "~/prompt.md", "~/"),
		newTextField("Notes", "", "freeform notes", ""),
	}

	return newForm("Create Task", fields, func(f *form) error {
		id := f.fields[0].text()
		if id == "" {
			return fmt.Errorf("ID is required")
		}

		promptFile, err := requireFile(f.fields[1])
		if err != nil {
			return err
		}

		_, err = registry.CreateTask(root, id, promptFile, f.fields[2].text())

		return err
	})
}

var thinkingOptions = append([]string{""}, validate.ThinkingLevels...)

func versionSettingsFields(root string) []*formField {
	toolsModeField := newEnumField("Tools Mode", "", validate.ToolsModes, validate.ToolsModes[0])

	toolNamesField := newPickerField("Tool Names", "comma-separated; not limited to the suggestions", "read,grep,find,ls", validate.BuiltinTools, "", true)
	toolNamesField.visible = func() bool {
		mode := toolsModeField.options[toolsModeField.enumIdx]

		return mode == "allowlist" || mode == "denylist"
	}

	extensionSourcesHelp := "comma-separated, always loaded regardless of isolate: a package reference (npm:name@1.2.3 or git:host/repo@ref, pinned to a version), an absolute path (works only on this machine), or type ./ or ../ to browse a path relative to the registry directory"
	if root != "" {
		extensionSourcesHelp += " (" + root + ")"
	}

	extensionSourcesField := newPickerField("Extension Sources", extensionSourcesHelp, "npm:name@1.2.3", discoverExtensionSources(), "", true)
	extensionSourcesField.pathBase = root

	return []*formField{
		newEnumField("System Prompt Mode", "replace: the prompt takes the place of Pi's default system prompt; append: it is added after Pi's default one", validate.SystemPromptModes, validate.SystemPromptModes[0]),
		newModelPickerField("Model", "provider is set automatically from the chosen model", discoverModelEntries(), "", ""),
		newEnumField("Thinking", "", thinkingOptions, ""),
		toolsModeField,
		toolNamesField,
		newBoolField("Extensions Isolate", "don't load extensions from your Pi setup; the ones below still load", false),
		extensionSourcesField,
		newBoolField("Skills Isolate", "don't load skills from your Pi setup; the ones below still load", false),
		newPickerField("Skill Paths", "comma-separated, copied into the registry so the version stays portable; not limited to the suggestions", "/path/to/skill", discoverSkillPaths(), "", true),
		newTextField("Notes", "", "", ""),
	}
}

func buildVersionInput(f *form) (registry.AgentVersionInput, []string, []string, error) {
	var notices, warnings []string

	promptMode := f.fields[0].options[f.fields[0].enumIdx]

	modelField := f.fields[1]
	if modelField.resolvedModel == "" {
		return registry.AgentVersionInput{}, nil, nil, fmt.Errorf("Model: select an entry from the list (type to search, enter to confirm)")
	}

	thinking := f.fields[2].options[f.fields[2].enumIdx]
	if thinking != "" && !validate.ThinkingLevel(thinking) {
		return registry.AgentVersionInput{}, nil, nil, fmt.Errorf("invalid thinking level %q", thinking)
	}

	toolsMode := f.fields[3].options[f.fields[3].enumIdx]
	toolNames := splitCommaList(f.fields[4].text())
	if toolsMode == "allowlist" || toolsMode == "denylist" {
		for _, tool := range validate.NonBuiltinTools(toolNames) {
			notices = append(notices, fmt.Sprintf("tool %q is not one of Pi's built-in tools; this is expected if an extension you're loading provides it, but Pi will silently ignore it otherwise", tool))
		}
	}

	extSources := splitCommaList(f.fields[6].text())
	for _, source := range extSources {
		parsed, err := validate.ParseExtensionSource(source)
		if err != nil {
			return registry.AgentVersionInput{}, nil, nil, fmt.Errorf("extension: %w", err)
		}

		if !parsed.Pinned {
			warnings = append(warnings, fmt.Sprintf("extension %q is not pinned, so runs may change when the package is updated; pin it with an exact version (npm:name@1.2.3) or a git tag or commit (git:host/repo@ref)", source))
		}
		if !parsed.Portable {
			warnings = append(warnings, fmt.Sprintf("extension %q is a path tied to this machine; it will not resolve if the registry is moved or shared. Consider a path relative to the registry directory (e.g. ../my-extension) instead, or publish it as an npm:/git: package", source))
		}
	}

	skillPaths := splitCommaList(f.fields[8].text())
	for i, p := range skillPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return registry.AgentVersionInput{}, nil, nil, err
		}
		if !validate.SkillPath(abs) {
			return registry.AgentVersionInput{}, nil, nil, fmt.Errorf("skill %q is not a skill: expected a folder containing SKILL.md, or a .md file with name and description frontmatter", p)
		}

		skillPaths[i] = abs
	}

	if len(skillPaths) > 0 && !canReadSkillFiles(toolsMode, toolNames) {
		warnings = append(warnings, "skills are attached, but the tools mode leaves out both read and bash; Pi only lists skills to the model when one of them is available, so it will not see these skills")
	}

	return registry.AgentVersionInput{
		SystemPromptMode:  promptMode,
		Model:             modelField.resolvedModel,
		Provider:          modelField.resolvedProvider,
		Thinking:          thinking,
		Notes:             f.fields[9].text(),
		Tools:             registry.ToolsConfig{Mode: toolsMode, Names: toolNames},
		ExtensionsIsolate: f.fields[5].boolVal,
		ExtensionSources:  extSources,
		SkillsIsolate:     f.fields[7].boolVal,
		SkillPaths:        skillPaths,
	}, notices, warnings, nil
}

func newVersionCreateForm(root, agentID string) *form {
	fields := append([]*formField{
		newTextField("Version ID", "unique within this agent, immutable", "e.g. 1.0.1", ""),
		newPathField("System Prompt File", "required; paths start from your home directory", "~/system-prompt.md", "~/"),
	}, versionSettingsFields(root)...)

	return newForm("Create Version for "+agentID, fields, func(f *form) error {
		versionID := f.fields[0].text()
		if versionID == "" {
			return fmt.Errorf("Version ID is required")
		}

		in, notices, warnings, err := buildVersionInput(&form{fields: f.fields[2:]})
		if err != nil {
			return err
		}

		if in.SystemPromptFile, err = requireFile(f.fields[1]); err != nil {
			return err
		}

		f.notices, f.warnings = notices, warnings

		_, err = registry.CreateAgentVersion(root, agentID, versionID, in)

		return err
	})
}
