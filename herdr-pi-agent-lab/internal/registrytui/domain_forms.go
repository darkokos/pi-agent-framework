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
	}, versionSettingsFields(nil)...)

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

func newAgentUpdateForm(root string, agent *registry.Agent) *form {
	fields := []*formField{
		newTextField("Name", "", "", agent.Name),
		newTextField("Description", "", "", agent.Description),
	}

	return newForm("Update Agent "+agent.ID, fields, func(f *form) error {
		agent.Name = f.fields[0].text()
		agent.Description = f.fields[1].text()

		_, err := registry.ReplaceAgent(root, agent)

		return err
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

func newTaskUpdateForm(root string, task *registry.Task) *form {
	fields := []*formField{
		newEditorField("Prompt", "edit the stored task prompt in your editor", func() (string, error) {
			data, err := os.ReadFile(registry.TaskPromptPath(root, task.ID))

			return string(data), err
		}),
		newTextField("Notes", "", "", task.Notes),
	}

	return newForm("Update Task "+task.ID, fields, func(f *form) error {
		task.Notes = f.fields[1].text()

		_, err := registry.ReplaceTask(root, task, f.fields[0].editedSource())

		return err
	})
}

var thinkingOptions = append([]string{""}, validate.ThinkingLevels...)

func versionSettingsFields(v *registry.AgentVersion) []*formField {
	model, provider, thinking, notes := "", "", "", ""
	toolsMode := validate.ToolsModes[0]
	promptMode := validate.SystemPromptModes[0]
	var toolNames, extPaths, skillPaths []string
	extIsolate, skillIsolate := false, false

	if v != nil {
		model, provider, thinking, notes = v.Model, v.Provider, v.Thinking, v.Notes
		promptMode = v.SystemPrompt.Mode
		toolsMode = v.Tools.Mode
		toolNames = v.Tools.Names
		extIsolate, extPaths = v.Extensions.Isolate, v.Extensions.Paths
		skillIsolate, skillPaths = v.Skills.Isolate, v.Skills.Paths
	}

	toolsModeField := newEnumField("Tools Mode", "", validate.ToolsModes, toolsMode)

	toolNamesField := newPickerField("Tool Names", "comma-separated; not limited to the suggestions", "read,grep,find,ls", validate.BuiltinTools, strings.Join(toolNames, ","), true)
	toolNamesField.visible = func() bool {
		mode := toolsModeField.options[toolsModeField.enumIdx]

		return mode == "allowlist" || mode == "denylist"
	}

	return []*formField{
		newEnumField("System Prompt Mode", "replace: the prompt takes the place of Pi's default system prompt; append: it is added after Pi's default one", validate.SystemPromptModes, promptMode),
		newModelPickerField("Model", "provider is set automatically from the chosen model", discoverModelEntries(), model, provider),
		newEnumField("Thinking", "", thinkingOptions, thinking),
		toolsModeField,
		toolNamesField,
		newBoolField("Extensions Isolate", "disable automatically-discovered extensions; paths below always load either way", extIsolate),
		newPickerField("Extension Paths", "comma-separated, always loaded regardless of isolate; not limited to the suggestions", "/path/to/extension.ts", discoverExtensionPaths(), strings.Join(extPaths, ","), true),
		newBoolField("Skills Isolate", "disable automatically-discovered skills; paths below always load either way", skillIsolate),
		newPickerField("Skill Paths", "comma-separated, always loaded regardless of isolate; not limited to the suggestions", "/path/to/skill", discoverSkillPaths(), strings.Join(skillPaths, ","), true),
		newTextField("Notes", "", "", notes),
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
	}, versionSettingsFields(nil)...)

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

func newVersionUpdateForm(root, agentID string, v *registry.AgentVersion) *form {
	fields := append([]*formField{
		newEditorField("System Prompt", "edit the stored system prompt in your editor", func() (string, error) {
			return registry.ReadVersionSystemPrompt(root, agentID, v.VersionID)
		}),
	}, versionSettingsFields(v)...)

	return newForm(fmt.Sprintf("Update Version %s@%s", agentID, v.VersionID), fields, func(f *form) error {
		in, err := buildVersionInput(&form{fields: f.fields[1:]})
		if err != nil {
			return err
		}

		v.SystemPrompt.Mode = in.SystemPromptMode
		v.Model, v.Provider, v.Thinking, v.Notes = in.Model, in.Provider, in.Thinking, in.Notes
		v.Tools, v.Extensions, v.Skills = in.Tools, in.Extensions, in.Skills

		_, err = registry.ReplaceAgentVersion(root, agentID, v.VersionID, v, f.fields[0].editedSource())

		return err
	})
}
