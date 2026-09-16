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
	fields := []*formField{
		newTextField("ID", "unique, immutable", "e.g. code-reviewer", ""),
		newTextField("Name", "", "human-readable name", ""),
		newTextField("Description", "", "what this agent is for", ""),
	}

	return newForm("Create Agent", fields, func(f *form) error {
		id := f.fields[0].text()
		if id == "" {
			return fmt.Errorf("ID is required")
		}

		_, err := registry.CreateAgent(root, id, f.fields[1].text(), f.fields[2].text())

		return err
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
		newTextField("Prompt File", "path to the task prompt text", "/path/to/prompt.md", ""),
		newTextField("Notes", "", "freeform notes", ""),
	}

	return newForm("Create Task", fields, func(f *form) error {
		id := f.fields[0].text()
		promptFile := f.fields[1].text()
		if id == "" || promptFile == "" {
			return fmt.Errorf("ID and Prompt File are required")
		}

		_, err := registry.CreateTask(root, id, promptFile, f.fields[2].text())

		return err
	})
}

func newTaskUpdateForm(root string, task *registry.Task) *form {
	fields := []*formField{
		newTextField("Prompt File", "leave empty to keep current content, or enter a path to replace it", "/path/to/new-prompt.md", ""),
		newTextField("Notes", "", "", task.Notes),
	}

	return newForm("Update Task "+task.ID, fields, func(f *form) error {
		task.Notes = f.fields[1].text()

		_, err := registry.ReplaceTask(root, task, f.fields[0].text())

		return err
	})
}

var thinkingOptions = append([]string{""}, validate.ThinkingLevels...)

func versionFields(systemPromptHelp string, v *registry.AgentVersion) []*formField {
	model, provider, thinking, notes := "", "", "", ""
	toolsMode := validate.ToolsModes[0]
	var toolNames, extPaths, skillPaths []string
	extIsolate, skillIsolate := false, false

	if v != nil {
		model, provider, thinking, notes = v.Model, v.Provider, v.Thinking, v.Notes
		toolsMode = v.Tools.Mode
		toolNames = v.Tools.Names
		extIsolate, extPaths = v.Extensions.Isolate, v.Extensions.Paths
		skillIsolate, skillPaths = v.Skills.Isolate, v.Skills.Paths
	}

	return []*formField{
		newTextField("System Prompt File", systemPromptHelp, "/path/to/system-prompt.md", ""),
		newModelPickerField("Model", "provider is set automatically from the chosen model", discoverModelEntries(), model, provider),
		newEnumField("Thinking", "", thinkingOptions, thinking),
		newEnumField("Tools Mode", "", validate.ToolsModes, toolsMode),
		newPickerField("Tool Names", "comma-separated, used with allowlist|denylist; not limited to the suggestions", "read,grep,find,ls", validate.BuiltinTools, strings.Join(toolNames, ","), true),
		newBoolField("Extensions Isolate", "pass --no-extensions and pin only the paths below", extIsolate),
		newPickerField("Extension Paths", "comma-separated, used when isolated; not limited to the suggestions", "/path/to/extension.ts", discoverExtensionPaths(), strings.Join(extPaths, ","), true),
		newBoolField("Skills Isolate", "pass --no-skills and pin only the paths below", skillIsolate),
		newPickerField("Skill Paths", "comma-separated, used when isolated; not limited to the suggestions", "/path/to/skill", discoverSkillPaths(), strings.Join(skillPaths, ","), true),
		newTextField("Notes", "", "", notes),
	}
}

func buildVersionInput(f *form) (registry.AgentVersionInput, error) {
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
		SystemPromptFile: f.fields[0].text(),
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
		newTextField("Version ID", "unique within this agent, immutable", "e.g. v1", ""),
	}, versionFields("required", nil)...)

	return newForm("Create Version for "+agentID, fields, func(f *form) error {
		versionID := f.fields[0].text()
		if versionID == "" {
			return fmt.Errorf("Version ID is required")
		}

		sub := &form{fields: f.fields[1:]}
		in, err := buildVersionInput(sub)
		if err != nil {
			return err
		}
		if in.SystemPromptFile == "" || in.Model == "" {
			return fmt.Errorf("System Prompt File and Model are required")
		}

		_, err = registry.CreateAgentVersion(root, agentID, versionID, in)

		return err
	})
}

func newVersionUpdateForm(root, agentID string, v *registry.AgentVersion) *form {
	fields := versionFields("leave empty to keep current content, or enter a path to replace it", v)

	return newForm(fmt.Sprintf("Update Version %s@%s", agentID, v.VersionID), fields, func(f *form) error {
		in, err := buildVersionInput(f)
		if err != nil {
			return err
		}

		v.Model, v.Provider, v.Thinking, v.Notes = in.Model, in.Provider, in.Thinking, in.Notes
		v.Tools, v.Extensions, v.Skills = in.Tools, in.Extensions, in.Skills

		_, err = registry.ReplaceAgentVersion(root, agentID, v.VersionID, v, in.SystemPromptFile)

		return err
	})
}
