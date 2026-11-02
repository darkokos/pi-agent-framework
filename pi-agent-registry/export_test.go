package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeRecordFile(t *testing.T, root, sessionID string, overrides map[string]any) {
	t.Helper()

	rec := baseRecord(3)
	rec["sessionId"] = sessionID
	rec["finalResponse"] = nil
	for k, v := range overrides {
		rec[k] = v
	}

	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}

	dir := RecordsDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sessionID+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExportRun_BundlesRecordMetadataAndResponse(t *testing.T) {
	root := t.TempDir()
	src := t.TempDir()
	write(t, filepath.Join(src, "prompt.md"), "system prompt")
	write(t, filepath.Join(src, "sk", "SKILL.md"), "skill content")

	if _, err := CreateAgent(root, "reviewer", "Reviewer", "desc"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateAgentVersion(root, "reviewer", "1.0.0", AgentVersionInput{
		SystemPromptFile: filepath.Join(src, "prompt.md"),
		Model:            "m",
		Tools:            ToolsConfig{Mode: "all"},
		SkillPaths:       []string{filepath.Join(src, "sk")},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTask(root, "review-pr", "Review PR", filepath.Join(src, "prompt.md"), ""); err != nil {
		t.Fatal(err)
	}

	writeRecordFile(t, root, "sess-1", map[string]any{
		"agentVersion":  "reviewer@1.0.0",
		"taskId":        "review-pr",
		"finalResponse": "the final answer",
	})

	dest := filepath.Join(t.TempDir(), "out")
	if err := ExportRun(root, dest, "sess-1", "reviewer", "1.0.0", "review-pr"); err != nil {
		t.Fatal(err)
	}

	for path, want := range map[string]string{
		"response.md":              "the final answer",
		"agent/agent.json":         "",
		"agent/version.json":       "",
		"agent/version.system.md":  "system prompt",
		"agent/skills/sk/SKILL.md": "skill content",
		"task/task.json":           "",
		"task/task.prompt.md":      "system prompt",
	} {
		data, err := os.ReadFile(filepath.Join(dest, path))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		if want != "" && string(data) != want {
			t.Errorf("%s = %q, want %q", path, data, want)
		}
	}

	recData, err := os.ReadFile(filepath.Join(dest, "session-record.json"))
	if err != nil {
		t.Fatal(err)
	}
	orig, err := os.ReadFile(recordFile(root, "sess-1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(recData) != string(orig) {
		t.Error("session-record.json must be an exact copy of the registry's record")
	}
}

func TestExportRun_NoFinalResponseWritesNoFile(t *testing.T) {
	root := t.TempDir()
	src := t.TempDir()
	write(t, filepath.Join(src, "prompt.md"), "p")

	if _, err := CreateAgent(root, "a", "a", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateAgentVersion(root, "a", "1.0.0", AgentVersionInput{
		SystemPromptFile: filepath.Join(src, "prompt.md"), Model: "m", Tools: ToolsConfig{Mode: "all"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTask(root, "t", "t", filepath.Join(src, "prompt.md"), ""); err != nil {
		t.Fatal(err)
	}

	writeRecordFile(t, root, "sess-2", map[string]any{
		"agentVersion": "a@1.0.0",
		"taskId":       "t",
	})

	dest := filepath.Join(t.TempDir(), "out")
	if err := ExportRun(root, dest, "sess-2", "a", "1.0.0", "t"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dest, "response.md")); !os.IsNotExist(err) {
		t.Error("response.md must not be written when the record has no finalResponse")
	}
	if _, err := os.Stat(filepath.Join(dest, "agent", "skills")); !os.IsNotExist(err) {
		t.Error("skills dir must not be written when the version has none")
	}
}

func TestExportRun_UnknownSessionFails(t *testing.T) {
	root := t.TempDir()

	if err := ExportRun(root, filepath.Join(t.TempDir(), "out"), "missing", "a", "1.0.0", "t"); err == nil {
		t.Error("exporting an unrecorded session must fail")
	}
}
