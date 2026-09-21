package invoke

import (
	"strings"
	"testing"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func TestBuildInvocation_SystemPromptMode(t *testing.T) {
	cases := map[string]struct {
		mode string
		want string
		not  string
	}{
		"replace": {mode: "replace", want: "--system-prompt ", not: "--append-system-prompt"},
		"append":  {mode: "append", want: "--append-system-prompt ", not: "--system-prompt "},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			version := &registry.AgentVersion{
				Model:        "m",
				SystemPrompt: registry.SystemPromptRef{Mode: tc.mode},
				Tools:        registry.ToolsConfig{Mode: "all"},
			}

			line, err := BuildInvocation("a", "1", "t", "the prompt", "/task.md", version)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(line, tc.want+"'the prompt'") {
				t.Errorf("expected %q before the prompt:\n%s", tc.want, line)
			}
			if strings.Contains(line, tc.not) {
				t.Errorf("%s mode must not also pass %q:\n%s", tc.mode, tc.not, line)
			}
		})
	}
}
