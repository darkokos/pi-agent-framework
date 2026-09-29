package invoke

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func TestBuildInvocation_ExtensionsAndSkillsAreResolvedAgainstTheRegistry(t *testing.T) {
	version := &registry.AgentVersion{
		Model:        "m",
		SystemPrompt: registry.SystemPromptRef{Mode: "replace"},
		Tools:        registry.ToolsConfig{Mode: "all"},
		Extensions: registry.ExtensionsConfig{
			Isolate: true,
			Sources: []string{"npm:@foo/bar@1.2.3", "git:github.com/u/r@v1", "../local-ext"},
		},
		Skills: registry.SkillsConfig{Isolate: true, Names: []string{"find-skills", "one.md"}},
	}

	line, err := BuildInvocation("/work/my-registry", "agent", "1.0.0", "task", "p", "/task.md", version)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"--no-extensions",
		"-e 'npm:@foo/bar@1.2.3'",
		"-e 'git:github.com/u/r@v1'",
		"-e '/work/local-ext'",
		"--no-skills",
		"--skill '/work/my-registry/agents/agent/versions/1.0.0.skills/find-skills'",
		"--skill '/work/my-registry/agents/agent/versions/1.0.0.skills/one.md'",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("missing %q in:\n%s", want, line)
		}
	}
}

func TestResolveExtensionPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]string{
		"../local-ext":  filepath.Join("/reg/root", "../local-ext"),
		"sub/ext":       filepath.Join("/reg/root", "sub/ext"),
		"/abs/path/ext": "/abs/path/ext",
		"~/ext":         filepath.Join(home, "ext"),
		"~":             home,
	}

	for source, want := range cases {
		if got := resolveExtensionPath("/reg/root", source); got != want {
			t.Errorf("resolveExtensionPath(%q) = %q, want %q", source, got, want)
		}
	}
}

func TestBuildInvocation_AbsoluteAndHomeExtensionPathsAreNotMangled(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	version := &registry.AgentVersion{
		Model:        "m",
		SystemPrompt: registry.SystemPromptRef{Mode: "replace"},
		Tools:        registry.ToolsConfig{Mode: "all"},
		Extensions:   registry.ExtensionsConfig{Sources: []string{"/home/user/my-ext", "~/my-ext", "~"}},
	}

	line, err := BuildInvocation("/reg/root", "a", "1", "t", "p", "/task.md", version)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"-e '/home/user/my-ext'",
		"-e '" + filepath.Join(home, "my-ext") + "'",
		"-e '" + home + "'",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("missing %q in:\n%s", want, line)
		}
	}

	if strings.Contains(line, "/reg/root/home") || strings.Contains(line, "/reg/root~") {
		t.Errorf("an absolute or ~ extension path must not be joined onto the registry root:\n%s", line)
	}
}

func TestBuildInvocation_NoResourcesMeansNoFlags(t *testing.T) {
	version := &registry.AgentVersion{Model: "m", SystemPrompt: registry.SystemPromptRef{Mode: "replace"}, Tools: registry.ToolsConfig{Mode: "all"}}

	line, err := BuildInvocation("/r", "a", "1", "t", "p", "/task.md", version)
	if err != nil {
		t.Fatal(err)
	}

	for _, unwanted := range []string{"--no-extensions", "-e ", "--no-skills", "--skill"} {
		if strings.Contains(line, unwanted) {
			t.Errorf("unexpected %q in:\n%s", unwanted, line)
		}
	}
}
