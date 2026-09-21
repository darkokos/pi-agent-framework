package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAgentVersion_SystemPromptMode(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(t.TempDir(), "p.md")
	if err := os.WriteFile(src, []byte("prompt"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := CreateAgent(root, "a", "a", ""); err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct{ in, want string }{
		"default": {in: "", want: "replace"},
		"replace": {in: "replace", want: "replace"},
		"append":  {in: "append", want: "append"},
	}

	for id, tc := range cases {
		in := AgentVersionInput{SystemPromptFile: src, SystemPromptMode: tc.in, Model: "m", Tools: ToolsConfig{Mode: "all"}}
		if _, err := CreateAgentVersion(root, "a", id, in); err != nil {
			t.Fatal(err)
		}

		got, err := GetAgentVersion(root, "a", id)
		if err != nil {
			t.Fatal(err)
		}
		if got.SystemPrompt.Mode != tc.want {
			t.Errorf("%s: mode = %q, want %q", id, got.SystemPrompt.Mode, tc.want)
		}
	}
}
