package invoke

import "testing"

func TestBuildResume_CarriesAttributionEnv(t *testing.T) {
	av, task := "reviewer@1.0.0", "review-pr"

	got := BuildResume("/work/it's", "/s/a.jsonl", &av, &task)
	want := `cd '/work/it'\''s' && PI_AGENT_VERSION='reviewer@1.0.0' PI_TASK_ID='review-pr' pi --session '/s/a.jsonl'`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}

	if got := BuildResume("/w", "/s.jsonl", nil, nil); got != `cd '/w' && pi --session '/s.jsonl'` {
		t.Fatalf("unattributed sessions must not get empty env vars, got %s", got)
	}
}
