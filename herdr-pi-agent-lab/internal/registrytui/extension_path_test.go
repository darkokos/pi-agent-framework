package registrytui

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPathBrowseMatches_RelativeToBase(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "registry")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "my-ext"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "other-ext.ts"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := pathBrowseMatches(root, "npm:foo"); got != nil {
		t.Fatalf("a non-path query must return nil (not a path query), got %v", got)
	}

	if got := pathBrowseMatches(root, "."); !slices.Equal(got, []string{"./", "../"}) {
		t.Fatalf("got %v", got)
	}
	if got := pathBrowseMatches(root, ".."); !slices.Equal(got, []string{"../"}) {
		t.Fatalf("got %v", got)
	}

	got := pathBrowseMatches(root, "../")
	if !slices.Contains(got, "../my-ext/") || !slices.Contains(got, "../other-ext.ts") {
		t.Fatalf("got %v", got)
	}

	if got := pathBrowseMatches(root, "../my"); !slices.Equal(got, []string{"../my-ext/"}) {
		t.Fatalf("got %v", got)
	}

	if got := pathBrowseMatches(root, "../nope"); len(got) != 0 {
		t.Fatalf("a query with no matches must be an empty, non-nil slice, got %v", got)
	}

	if got := pathBrowseMatches("", "../my"); len(got) != 0 || got == nil {
		t.Fatalf("./.. browsing with no base must be an empty, non-nil slice, got %v", got)
	}
}

func TestPathBrowseMatches_Home(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.MkdirAll(filepath.Join(home, "my-skills"), 0o755); err != nil {
		t.Fatal(err)
	}

	if got := pathBrowseMatches("/irrelevant", "~"); !slices.Equal(got, []string{"~/"}) {
		t.Fatalf("got %v", got)
	}

	got := pathBrowseMatches("/irrelevant", "~/my")
	if !slices.Equal(got, []string{"~/my-skills/"}) {
		t.Fatalf("got %v", got)
	}
}

func TestPathBrowseMatches_Absolute(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := pathBrowseMatches("/irrelevant", dir+"/")
	if !slices.Contains(got, dir+"/sub/") {
		t.Fatalf("got %v", got)
	}

	got = pathBrowseMatches("/irrelevant", dir+"/su")
	if !slices.Equal(got, []string{dir + "/sub/"}) {
		t.Fatalf("got %v", got)
	}
}

func TestExtensionSourcesField_BrowsesAllFourForms(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "registry")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parent, "my-ext"), 0o755); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "home-ext"), 0o755); err != nil {
		t.Fatal(err)
	}

	fields := versionSettingsFields(root)
	extField := fields[6]
	if extField.label != "Extension Sources" {
		t.Fatalf("field 6 is %q, not Extension Sources", extField.label)
	}

	set := func(v string) {
		extField.input.SetValue(v)
		extField.refilterPicker()
	}

	set("npm:@foo/bar@1.2.3,../")
	if !slices.Contains(extField.pickerMatches, "../my-ext/") {
		t.Fatalf("../ must browse the registry's parent, got %v", extField.pickerMatches)
	}

	set("~/home")
	if !slices.Contains(extField.pickerMatches, "~/home-ext/") {
		t.Fatalf("~/ must browse the home directory, got %v", extField.pickerMatches)
	}

	set(parent + "/my")
	if !slices.Contains(extField.pickerMatches, parent+"/my-ext/") {
		t.Fatalf("/ must browse absolute paths, got %v", extField.pickerMatches)
	}

	set("npm:")
	if slices.Contains(extField.pickerMatches, "../my-ext/") {
		t.Fatalf("a non-path query must fall back to the installed-package suggestions, got %v", extField.pickerMatches)
	}
}

func TestSkillPathsField_BrowsesAllFourForms(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	sibling := filepath.Join(filepath.Dir(cwd), "skill-browse-sibling-"+t.Name())
	if err := os.MkdirAll(sibling, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sibling) })

	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "home-skill"), 0o755); err != nil {
		t.Fatal(err)
	}

	fields := versionSettingsFields("/some/registry")
	skillField := fields[8]
	if skillField.label != "Skill Paths" {
		t.Fatalf("field 8 is %q, not Skill Paths", skillField.label)
	}

	want := "../" + filepath.Base(sibling) + "/"

	skillField.input.SetValue("../" + filepath.Base(sibling)[:3])
	skillField.refilterPicker()
	if !slices.Contains(skillField.pickerMatches, want) {
		t.Fatalf("../ must browse relative to the current directory, got %v", skillField.pickerMatches)
	}

	skillField.input.SetValue("~/home")
	skillField.refilterPicker()
	if !slices.Contains(skillField.pickerMatches, "~/home-skill/") {
		t.Fatalf("~/ must browse the home directory, got %v", skillField.pickerMatches)
	}
}
