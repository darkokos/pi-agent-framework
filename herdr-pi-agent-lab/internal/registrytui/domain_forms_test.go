package registrytui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestVersionFields(t *testing.T, root string) []*formField {
	t.Helper()

	fields := versionSettingsFields(root)
	fields[1].resolvedModel, fields[1].resolvedProvider = "some-model", "some-provider"

	return fields
}

func writeSkill(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "sk")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("---\nname: sk\ndescription: d\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestBuildVersionInput_AbsoluteExtensionPathIsAcceptedWithWarning(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[6].input.SetValue("/abs/path")

	in, _, warnings, err := buildVersionInput(&form{fields: fields})
	if err != nil {
		t.Fatalf("an absolute extension path must be accepted, got %v", err)
	}
	if in.ExtensionSources[0] != "/abs/path" {
		t.Fatalf("sources = %v", in.ExtensionSources)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "tied to this machine") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestBuildVersionInput_HomeExtensionPathIsAcceptedWithWarning(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[6].input.SetValue("~/ext")

	in, _, warnings, err := buildVersionInput(&form{fields: fields})
	if err != nil {
		t.Fatalf("a ~ extension path must be accepted, got %v", err)
	}
	if in.ExtensionSources[0] != "~/ext" {
		t.Fatalf("sources = %v", in.ExtensionSources)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "tied to this machine") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestBuildVersionInput_RegistryRelativeExtensionPathWarnsNothing(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[6].input.SetValue("../my-extension")

	_, _, warnings, err := buildVersionInput(&form{fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("a registry-relative path must not warn, got %v", warnings)
	}
}

func TestBuildVersionInput_UnpinnedExtensionWarns(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[6].input.SetValue("git:github.com/user/repo")

	in, notices, warnings, err := buildVersionInput(&form{fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if len(notices) != 0 {
		t.Fatalf("notices = %v", notices)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "not pinned") {
		t.Fatalf("warnings = %v", warnings)
	}
	if in.ExtensionSources[0] != "git:github.com/user/repo" {
		t.Fatalf("sources = %v", in.ExtensionSources)
	}
}

func TestBuildVersionInput_NonBuiltinToolIsANoticeNotAWarning(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[3].enumIdx = indexOf(fields[3].options, "allowlist")
	fields[4].input.SetValue("read,web_search")

	_, notices, warnings, err := buildVersionInput(&form{fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("a non-built-in tool name must not be a warning, got %v", warnings)
	}
	if len(notices) != 1 || !strings.Contains(notices[0], `"web_search"`) {
		t.Fatalf("notices = %v", notices)
	}
}

func TestBuildVersionInput_SkillMustBeAValidSkill(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[8].input.SetValue(t.TempDir())

	_, _, _, err := buildVersionInput(&form{fields: fields})
	if err == nil || !strings.Contains(err.Error(), "is not a skill") {
		t.Fatalf("expected a not-a-skill error, got %v", err)
	}
}

func TestBuildVersionInput_SkillsInvisibleWithoutReadOrBashWarns(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[3].enumIdx = indexOf(fields[3].options, "none")
	fields[8].input.SetValue(writeSkill(t, t.TempDir()))

	in, _, warnings, err := buildVersionInput(&form{fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.SkillPaths) != 1 || !filepath.IsAbs(in.SkillPaths[0]) {
		t.Fatalf("skill paths = %v, want one absolute path", in.SkillPaths)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "will not see these skills") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestBuildVersionInput_SkillsVisibleWithReadWarnsNothing(t *testing.T) {
	fields := newTestVersionFields(t, "")
	fields[3].enumIdx = indexOf(fields[3].options, "allowlist")
	fields[4].input.SetValue("read")
	fields[8].input.SetValue(writeSkill(t, t.TempDir()))

	_, _, warnings, err := buildVersionInput(&form{fields: fields})
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestBuildVersionInput_ModelMustBeResolved(t *testing.T) {
	fields := versionSettingsFields("")

	_, _, _, err := buildVersionInput(&form{fields: fields})
	if err == nil || !strings.Contains(err.Error(), "select an entry") {
		t.Fatalf("expected a model-selection error, got %v", err)
	}
}

func indexOf(options []string, want string) int {
	for i, o := range options {
		if o == want {
			return i
		}
	}

	return -1
}
