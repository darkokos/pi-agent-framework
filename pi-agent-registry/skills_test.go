package registry

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCopySkills_ResolvesSymlinksAndSkipsGit(t *testing.T) {
	src := t.TempDir()
	skill := filepath.Join(src, "my-skill")

	write(t, filepath.Join(skill, "SKILL.md"), "skill")
	write(t, filepath.Join(skill, "scripts", "run.sh"), "run")
	write(t, filepath.Join(skill, ".git", "HEAD"), "ref")
	write(t, filepath.Join(src, "shared.txt"), "shared file")
	write(t, filepath.Join(src, "shared-dir", "inner.txt"), "inner")

	if err := os.Symlink(filepath.Join(src, "shared.txt"), filepath.Join(skill, "link-file")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(src, "shared-dir"), filepath.Join(skill, "link-dir")); err != nil {
		t.Fatal(err)
	}

	top := filepath.Join(t.TempDir(), "linked-skill")
	if err := os.Symlink(skill, top); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "skills")

	names, err := copySkills(dst, []string{top})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(names, []string{"linked-skill"}) {
		t.Fatalf("names = %v", names)
	}

	base := filepath.Join(dst, "linked-skill")

	for path, want := range map[string]string{
		"SKILL.md":           "skill",
		"scripts/run.sh":     "run",
		"link-file":          "shared file",
		"link-dir/inner.txt": "inner",
	} {
		data, err := os.ReadFile(filepath.Join(base, path))
		if err != nil || string(data) != want {
			t.Errorf("%s = %q, %v; want %q", path, data, err, want)
		}
	}

	for _, p := range []string{"link-file", "link-dir"} {
		info, err := os.Lstat(filepath.Join(base, p))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s must be real content, not a symlink (%v)", p, err)
		}
	}

	if _, err := os.Stat(filepath.Join(base, ".git")); !os.IsNotExist(err) {
		t.Error(".git must not be copied")
	}
}

func TestCopySkills_SingleFileAndErrors(t *testing.T) {
	src := t.TempDir()
	write(t, filepath.Join(src, "one.md"), "---\nname: one\ndescription: d\n---\n")

	dst := filepath.Join(t.TempDir(), "skills")

	names, err := copySkills(dst, []string{filepath.Join(src, "one.md")})
	if err != nil || !slices.Equal(names, []string{"one.md"}) {
		t.Fatalf("got %v, %v", names, err)
	}

	write(t, filepath.Join(src, "a", "x", "SKILL.md"), "a")
	write(t, filepath.Join(src, "b", "x", "SKILL.md"), "b")

	if _, err := copySkills(filepath.Join(t.TempDir(), "s"), []string{filepath.Join(src, "a", "x"), filepath.Join(src, "b", "x")}); err == nil {
		t.Error("two skills with the same name must be rejected")
	}

	broken := filepath.Join(src, "broken")
	if err := os.Symlink(filepath.Join(src, "missing"), broken); err != nil {
		t.Fatal(err)
	}
	if _, err := copySkills(filepath.Join(t.TempDir(), "s"), []string{broken}); err == nil {
		t.Error("a broken symlink must be reported")
	}

	loop := filepath.Join(src, "loop")
	if err := os.MkdirAll(loop, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(loop, filepath.Join(loop, "again")); err != nil {
		t.Fatal(err)
	}
	if _, err := copySkills(filepath.Join(t.TempDir(), "s"), []string{loop}); err == nil {
		t.Error("a symlink loop must be reported, not followed forever")
	}
}

func TestCreateAndDeleteVersion_SkillsLifecycle(t *testing.T) {
	root := t.TempDir()
	src := t.TempDir()
	write(t, filepath.Join(src, "prompt.md"), "prompt")
	write(t, filepath.Join(src, "sk", "SKILL.md"), "skill")

	if _, err := CreateAgent(root, "a", "a", ""); err != nil {
		t.Fatal(err)
	}

	in := AgentVersionInput{
		SystemPromptFile: filepath.Join(src, "prompt.md"), Model: "m", Tools: ToolsConfig{Mode: "all"},
		ExtensionSources: []string{"npm:foo@1.0.0", "../ext"}, SkillPaths: []string{filepath.Join(src, "sk")},
	}

	v, err := CreateAgentVersion(root, "a", "1.0.0", in)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(v.Skills.Names, []string{"sk"}) || !slices.Equal(v.Extensions.Sources, []string{"npm:foo@1.0.0", "../ext"}) {
		t.Fatalf("stored config = %+v / %+v", v.Skills, v.Extensions)
	}

	skillFile := filepath.Join(VersionSkillsDir(root, "a", "1.0.0"), "sk", "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Fatalf("skill was not copied into the registry: %v", err)
	}

	in.SkillPaths = []string{filepath.Join(src, "does-not-exist")}
	if _, err := CreateAgentVersion(root, "a", "2.0.0", in); err == nil {
		t.Fatal("a missing skill must fail the create")
	}
	if _, err := os.Stat(VersionSkillsDir(root, "a", "2.0.0")); !os.IsNotExist(err) {
		t.Error("a failed create must not leave a skills folder behind")
	}

	if err := DeleteAgentVersion(root, "a", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(VersionSkillsDir(root, "a", "1.0.0")); !os.IsNotExist(err) {
		t.Error("deleting a version must delete its skills")
	}
}
