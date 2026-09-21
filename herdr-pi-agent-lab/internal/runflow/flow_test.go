package runflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func seed(t *testing.T) (root string, agents []*registry.Agent) {
	t.Helper()

	root = t.TempDir()
	src := filepath.Join(t.TempDir(), "p.md")
	if err := os.WriteFile(src, []byte("prompt"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"alpha", "beta"} {
		if _, err := registry.CreateAgent(root, id, id, ""); err != nil {
			t.Fatal(err)
		}
		for _, v := range []string{"1.0.0", "1.1.0"} {
			in := registry.AgentVersionInput{SystemPromptFile: src, Model: "m", Tools: registry.ToolsConfig{Mode: "all"}}
			if _, err := registry.CreateAgentVersion(root, id, v, in); err != nil {
				t.Fatal(err)
			}
		}
	}

	if _, err := registry.CreateTask(root, "the-task", src, ""); err != nil {
		t.Fatal(err)
	}

	agents, err := registry.ListAgents(root)
	if err != nil {
		t.Fatal(err)
	}

	return root, agents
}

func press(t *testing.T, f Flow, keys ...tea.KeyMsg) Flow {
	t.Helper()

	for _, k := range keys {
		f, _, _ = f.Update(k)
	}

	return f
}

func TestNewForTask_PicksAnAgentVersionForTheFixedTask(t *testing.T) {
	root, agents := seed(t)

	f := NewForTask(root, "the-task", agents, 100, 30)
	if !f.HasChoices() || len(f.items) != 4 {
		t.Fatalf("expected one choice per agent version, got %d", len(f.items))
	}

	down := tea.KeyMsg{Type: tea.KeyDown}
	f = press(t, f, down, down, tea.KeyMsg{Type: tea.KeyEnter})

	if !f.Confirming() {
		t.Fatalf("enter must move to the confirmation, err: %v", f.err)
	}

	line := f.confirm.Line()
	for _, want := range []string{"PI_AGENT_VERSION='beta@1.0.0'", "PI_TASK_ID='the-task'"} {
		if !strings.Contains(line, want) {
			t.Errorf("command is missing %s:\n%s", want, line)
		}
	}
}

func TestNewForTask_NoAgentVersionsMeansNoChoices(t *testing.T) {
	if NewForTask(t.TempDir(), "t", nil, 100, 30).HasChoices() {
		t.Fatal("no agents must mean no choices")
	}
}

func TestNew_FixedVersionPicksATask(t *testing.T) {
	root, _ := seed(t)

	tasks, err := registry.ListTasks(root)
	if err != nil {
		t.Fatal(err)
	}

	f := press(t, New(root, "alpha", "1.1.0", tasks, 100, 30), tea.KeyMsg{Type: tea.KeyEnter})
	if !f.Confirming() || !strings.Contains(f.confirm.Line(), "PI_AGENT_VERSION='alpha@1.1.0'") {
		t.Fatalf("unexpected state, err: %v", f.err)
	}
}
