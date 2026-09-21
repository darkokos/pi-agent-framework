package registry

import (
	"encoding/json"
	"testing"
)

func baseRecord(version int) map[string]any {
	return map[string]any{
		"schemaVersion": version, "sessionId": "s", "sessionFile": "f", "cwd": "c",
		"endedAt": 1, "endReason": "quit", "agentVersion": nil, "taskId": nil,
		"models": []any{},
		"usage":  map[string]any{"input": 0, "output": 0, "cacheRead": 0, "cacheWrite": 0, "costUsd": 0, "unmeteredMessages": 0},
		"activity": map[string]any{
			"assistantMessages": 0, "userMessages": 0, "toolCalls": 0, "toolErrors": 0, "byTool": map[string]any{}, "compactions": 0,
		},
		"outcome": map[string]any{"rated": false},
	}
}

func TestRecordSchema_RunIdIsVersionDependent(t *testing.T) {
	schema, err := recordSchema()
	if err != nil {
		t.Fatal(err)
	}

	valid := func(rec map[string]any) bool {
		data, _ := json.Marshal(rec)

		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			t.Fatal(err)
		}

		return schema.Validate(v) == nil
	}

	v1 := baseRecord(1)
	if valid(v1) {
		t.Error("a version 1 record without runId must be invalid")
	}
	v1["runId"] = "r"
	if !valid(v1) {
		t.Error("a version 1 record with runId must stay valid")
	}

	v2 := baseRecord(2)
	if !valid(v2) {
		t.Error("a version 2 record without runId must be valid")
	}
	v2["runId"] = "r"
	if valid(v2) {
		t.Error("a version 2 record must not carry runId")
	}

	if valid(baseRecord(3)) {
		t.Error("an unknown schema version must be rejected")
	}
}
