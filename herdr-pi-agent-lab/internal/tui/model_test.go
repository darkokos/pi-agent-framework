package tui

import (
	"encoding/json"
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

	t.Setenv("PI_AGENT_REGISTRY_DIR", registryDir)
	t.Setenv("PI_SESSION_RECORDS", recordsDir)

	tasksDir := filepath.Join(registryDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeJSONFile(t, filepath.Join(tasksDir, "refactor-auth.json"), map[string]any{
		"id": "refactor-auth", "promptFile": "refactor-auth.prompt.md", "createdAt": 1,
	})
	writeJSONFile(t, filepath.Join(tasksDir, "no-runs-yet.json"), map[string]any{
		"id": "no-runs-yet", "promptFile": "no-runs-yet.prompt.md", "createdAt": 1,
	})

	writeJSONFile(t, filepath.Join(recordsDir, "s1.json"), map[string]any{
		"schemaVersion": 1, "sessionId": "s1", "sessionFile": "f", "cwd": "c",
		"endedAt": 1, "endReason": "quit",
		"agentVersion": "code-reviewer@v1", "taskId": "refactor-auth", "runId": "s1",
		"models": []any{},
		"usage":  map[string]any{"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "costUsd": 0, "unmeteredMessages": 0},
		"activity": map[string]any{
			"assistantMessages": 0, "userMessages": 0, "toolCalls": 0, "toolErrors": 0, "byTool": map[string]any{}, "compactions": 0,
		},
		"outcome": map[string]any{"rated": true, "outcome": 5, "rework": 5, "autonomy": 5, "ratedAt": 1, "source": "command"},
	})

	msg := loadData()

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
