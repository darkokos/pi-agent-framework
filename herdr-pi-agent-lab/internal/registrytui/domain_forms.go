package registrytui

import (
	"fmt"
	"os"
	"path/filepath"
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

func discoverExtensionPaths() []string {
	var out []string

	if dir, err := validate.GlobalExtensionsDir(); err == nil {
		if entries, err := os.ReadDir(dir); err == nil {
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

	return out
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

func validatePaths(paths []string, valid func(string) bool, flag string) error {
	for _, p := range paths {
		if !valid(p) {
			return fmt.Errorf("%s %q is not a valid path", flag, p)
		}
	}

	return nil
}

func newAgentCreateForm(root string) *form {
	fields := append([]*formField{
		newTextField("ID", "unique, immutable", "e.g. code-reviewer", ""),
		newTextField("Name", "", "human-readable name", ""),
		newTextField("Description", "", "what this agent is for", ""),
		newTextField("Initial Version ID", "unique within this agent, immutable", "e.g. 1.0.0", ""),
		newPathField("System Prompt File", "required; paths start from your home directory", "~/system-prompt.md", "~/"),
	}, versionSettingsFields()...)

	return newForm("Create Agent", fields, func(f *form) error {
		id := f.fields[0].text()
		if id == "" {
			return fmt.Errorf("ID is required")
		}

		versionID := f.fields[3].text()
		if versionID == "" {
			return fmt.Errorf("Initial Version ID is required")
		}

		in, err := buildVersionInput(&form{fields: f.fields[5:]})
		if err != nil {
			return err
		}

		if in.SystemPromptFile, err = requireFile(f.fields[4]); err != nil {
			return err
		}

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

func versionSettingsFields() []*formField {
	toolsModeField := newEnumField("Tools Mode", "", validate.ToolsModes, validate.ToolsModes[0])

	toolNamesField := newPickerField("Tool Names", "comma-separated; not limited to the suggestions", "read,grep,find,ls", validate.BuiltinTools, "", true)
	toolNamesField.visible = func() bool {
		mode := toolsModeField.options[toolsModeField.enumIdx]

		return mode == "allowlist" || mode == "denylist"
	}

	return []*formField{
		newEnumField("System Prompt Mode", "replace: the prompt takes the place of Pi's default system prompt; append: it is added after Pi's default one", validate.SystemPromptModes, validate.SystemPromptModes[0]),
		newModelPickerField("Model", "provider is set automatically from the chosen model", discoverModelEntries(), "", ""),
		newEnumField("Thinking", "", thinkingOptions, ""),
		toolsModeField,
		toolNamesField,
		newBoolField("Extensions Isolate", "disable automatically-discovered extensions; paths below always load either way", false),
		newPickerField("Extension Paths", "comma-separated, always loaded regardless of isolate; not limited to the suggestions", "/path/to/extension.ts", discoverExtensionPaths(), "", true),
		newBoolField("Skills Isolate", "disable automatically-discovered skills; paths below always load either way", false),
		newPickerField("Skill Paths", "comma-separated, always loaded regardless of isolate; not limited to the suggestions", "/path/to/skill", discoverSkillPaths(), "", true),
		newTextField("Notes", "", "", ""),
	}
}

func buildVersionInput(f *form) (registry.AgentVersionInput, error) {
	promptMode := f.fields[0].options[f.fields[0].enumIdx]

	modelField := f.fields[1]
	if modelField.resolvedModel == "" {
		return registry.AgentVersionInput{}, fmt.Errorf("Model: select an entry from the list (type to search, enter to confirm)")
	}

	thinking := f.fields[2].options[f.fields[2].enumIdx]
	if thinking != "" && !validate.ThinkingLevel(thinking) {
		return registry.AgentVersionInput{}, fmt.Errorf("invalid thinking level %q", thinking)
	}

	extPaths := splitCommaList(f.fields[6].text())
	if err := validatePaths(extPaths, validate.ExtensionPath, "extension path"); err != nil {
		return registry.AgentVersionInput{}, err
	}

	skillPaths := splitCommaList(f.fields[8].text())
	if err := validatePaths(skillPaths, validate.SkillPath, "skill path"); err != nil {
		return registry.AgentVersionInput{}, err
	}

	return registry.AgentVersionInput{
		SystemPromptMode: promptMode,
		Model:            modelField.resolvedModel,
		Provider:         modelField.resolvedProvider,
		Thinking:         thinking,
		Notes:            f.fields[9].text(),
		Tools: registry.ToolsConfig{
			Mode:  f.fields[3].options[f.fields[3].enumIdx],
			Names: splitCommaList(f.fields[4].text()),
		},
		Extensions: registry.ResourceIsolation{
			Isolate: f.fields[5].boolVal,
			Paths:   extPaths,
		},
		Skills: registry.ResourceIsolation{
			Isolate: f.fields[7].boolVal,
			Paths:   skillPaths,
		},
	}, nil
}

func newVersionCreateForm(root, agentID string) *form {
	fields := append([]*formField{
		newTextField("Version ID", "unique within this agent, immutable", "e.g. 1.0.1", ""),
		newPathField("System Prompt File", "required; paths start from your home directory", "~/system-prompt.md", "~/"),
	}, versionSettingsFields()...)

	return newForm("Create Version for "+agentID, fields, func(f *form) error {
		versionID := f.fields[0].text()
		if versionID == "" {
			return fmt.Errorf("Version ID is required")
		}

		in, err := buildVersionInput(&form{fields: f.fields[2:]})
		if err != nil {
			return err
		}

		if in.SystemPromptFile, err = requireFile(f.fields[1]); err != nil {
			return err
		}

		_, err = registry.CreateAgentVersion(root, agentID, versionID, in)

		return err
	})
}
