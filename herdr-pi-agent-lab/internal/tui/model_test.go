package tui

import (
	"encoding/json"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"path/filepath"
	"testing"
)

func writeJSONFile(t *testing.T, path string, v any) {
	t.Helper()

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadData_AssociatesGroupsWithRegisteredTasks(t *testing.T) {
	registryDir := t.TempDir()
	recordsDir := t.TempDir()

	t.Setenv("PI_SESSION_RECORDS", recordsDir)

	tasksDir := filepath.Join(registryDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeJSONFile(t, filepath.Join(tasksDir, "refactor-auth.json"), map[string]any{
		"id": "refactor-auth", "promptFile": "refactor-auth.prompt.md",
	})
	writeJSONFile(t, filepath.Join(tasksDir, "no-runs-yet.json"), map[string]any{
		"id": "no-runs-yet", "promptFile": "no-runs-yet.prompt.md",
	})

	writeJSONFile(t, filepath.Join(recordsDir, "s1.json"), map[string]any{
		"schemaVersion": 2, "sessionId": "s1", "sessionFile": "f", "cwd": "c",
		"endedAt": 1, "endReason": "quit",
		"agentVersion": "code-reviewer@v1", "taskId": "refactor-auth",
		"models": []any{},
		"usage":  map[string]any{"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "costUsd": 0, "unmeteredMessages": 0},
		"activity": map[string]any{
			"assistantMessages": 0, "userMessages": 0, "toolCalls": 0, "toolErrors": 0, "byTool": map[string]any{}, "compactions": 0,
		},
		"outcome": map[string]any{"rated": true, "outcome": 5, "rework": 5, "autonomy": 5, "ratedAt": 1, "source": "command"},
	})

	msg := loadData(&registryDir)()
	data, ok := msg.(dataMsg)
	if !ok {
		t.Fatalf("expected dataMsg, got %#v", msg)
	}
	if len(data.rows) != 2 {
		t.Fatalf("expected 2 task rows, got %d", len(data.rows))
	}

	byID := map[string]taskRow{}
	for _, r := range data.rows {
		byID[r.task.ID] = r
	}

	withRun := byID["refactor-auth"]
	if withRun.group == nil || len(withRun.group.Rated) != 1 {
		t.Fatalf("expected refactor-auth to have 1 rated run, got %#v", withRun.group)
	}

	noRuns := byID["no-runs-yet"]
	if noRuns.group != nil {
		t.Fatalf("expected no-runs-yet to have a nil group, got %#v", noRuns.group)
	}
}

func TestResume_SelectsUnratedRunAndBuildsCommand(t *testing.T) {
	registryDir := t.TempDir()
	recordsDir := t.TempDir()

	t.Setenv("PI_SESSION_RECORDS", recordsDir)

	if err := os.MkdirAll(filepath.Join(registryDir, "tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFile(t, filepath.Join(registryDir, "tasks", "t.json"), map[string]any{"id": "t", "promptFile": "t.prompt.md"})

	for _, id := range []string{"s1", "s2"} {
		writeJSONFile(t, filepath.Join(recordsDir, id+".json"), map[string]any{
			"schemaVersion": 2, "sessionId": id, "sessionFile": "/sessions/" + id + ".jsonl", "cwd": "/work",
			"endedAt": 1, "endReason": "quit", "agentVersion": "a@1", "taskId": "t", "models": []any{},
			"usage":    map[string]any{"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "costUsd": 0, "unmeteredMessages": 0},
			"activity": map[string]any{"assistantMessages": 0, "userMessages": 0, "toolCalls": 0, "toolErrors": 0, "byTool": map[string]any{}, "compactions": 0},
			"outcome":  map[string]any{"rated": false},
		})
	}

	m := NewModel()
	data, ok := loadData(&registryDir)().(dataMsg)
	if !ok {
		t.Fatal("expected dataMsg")
	}
	m.rows, m.width, m.height = data.rows, 100, 30

	key := func(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
	step := func(msg tea.KeyMsg) { next, _ := m.handleKey(msg); m = next.(Model) }

	step(tea.KeyMsg{Type: tea.KeyEnter})
	if m.view != viewDetail {
		t.Fatal("enter must open the task detail")
	}

	step(key("j"))
	if m.runCursor != 1 {
		t.Fatalf("j must move between unrated runs, got cursor %d", m.runCursor)
	}

	step(key("r"))
	if m.view != viewResumeConfirm || m.resume == nil {
		t.Fatal("r must open the resume confirmation")
	}

	want := "cd '/work' && PI_AGENT_VERSION='a@1' PI_TASK_ID='t' pi --session '/sessions/s2.jsonl'"
	if got := m.resume.Line(); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}

	step(key("n"))
	if m.view != viewDetail || m.resume != nil {
		t.Fatal("n must cancel back to the detail view")
	}
}
