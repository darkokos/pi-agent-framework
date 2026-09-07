package compare

import (
	"testing"

	sessionrecord "github.com/darkokos/pi-agent-framework/pi-session-schema/go"
)

func rated(taskID, sessionID string, outcome, rework, autonomy int, costUsd float64) *sessionrecord.SessionRecord {
	return &sessionrecord.SessionRecord{
		SchemaVersion: 1,
		SessionId:     sessionID,
		TaskId:        new(taskID),
		Usage:         sessionrecord.SessionRecordUsage{CostUsd: costUsd},
		Outcome: sessionrecord.SessionRecordOutcome{
			Rated:    true,
			Outcome:  new(outcome),
			Rework:   new(rework),
			Autonomy: new(autonomy),
		},
	}
}

func unrated(taskID, sessionID string) *sessionrecord.SessionRecord {
	return &sessionrecord.SessionRecord{
		SchemaVersion: 1,
		SessionId:     sessionID,
		TaskId:        new(taskID),
		Outcome:       sessionrecord.SessionRecordOutcome{Rated: false},
	}
}

func ids(records []*sessionrecord.SessionRecord) string {
	out := ""
	for i, r := range records {
		if i > 0 {
			out += ","
		}
		out += r.SessionId
	}
	return out
}

func TestGroupByTask_RanksByOutcomeFirst(t *testing.T) {
	worse := rated("t1", "a", 3, 5, 5, 0)
	better := rated("t1", "b", 5, 1, 1, 100)

	groups := GroupByTask([]*sessionrecord.SessionRecord{worse, better})

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if got := groups[0].Rated; len(got) != 2 || got[0].SessionId != "b" || got[1].SessionId != "a" {
		t.Fatalf("expected [b, a] ranked by outcome, got %v", ids(got))
	}
}

func TestGroupByTask_ReworkTiebreaksEqualOutcome(t *testing.T) {
	lessRework := rated("t1", "a", 5, 5, 1, 0)
	moreRework := rated("t1", "b", 5, 2, 1, 0)

	groups := GroupByTask([]*sessionrecord.SessionRecord{moreRework, lessRework})

	if got := groups[0].Rated; ids(got) != "a,b" {
		t.Fatalf("expected [a, b] (higher rework first), got %v", ids(got))
	}
}

func TestGroupByTask_AutonomyTiebreaksEqualOutcomeAndRework(t *testing.T) {
	lessAutonomy := rated("t1", "a", 5, 5, 2, 0)
	moreAutonomy := rated("t1", "b", 5, 5, 5, 0)

	groups := GroupByTask([]*sessionrecord.SessionRecord{lessAutonomy, moreAutonomy})

	if got := groups[0].Rated; ids(got) != "b,a" {
		t.Fatalf("expected [b, a] (higher autonomy first), got %v", ids(got))
	}
}

func TestGroupByTask_CostOnlyTiebreaksFullTies(t *testing.T) {
	expensive := rated("t1", "a", 5, 5, 5, 10)
	cheap := rated("t1", "b", 5, 5, 5, 1)

	groups := GroupByTask([]*sessionrecord.SessionRecord{expensive, cheap})

	if got := groups[0].Rated; ids(got) != "b,a" {
		t.Fatalf("expected [b, a] (cheaper first among full ties), got %v", ids(got))
	}
}

func TestGroupByTask_UnratedNeverEntersRankedList(t *testing.T) {
	good := rated("t1", "a", 5, 5, 5, 0)
	pending := unrated("t1", "b")

	groups := GroupByTask([]*sessionrecord.SessionRecord{good, pending})

	if ids(groups[0].Rated) != "a" {
		t.Fatalf("expected only [a] in Rated, got %v", ids(groups[0].Rated))
	}
	if ids(groups[0].Unrated) != "b" {
		t.Fatalf("expected [b] in Unrated, got %v", ids(groups[0].Unrated))
	}
}

func TestGroupByTask_ExcludesRecordsWithNoTaskId(t *testing.T) {
	withTask := rated("t1", "a", 5, 5, 5, 0)
	noTask := rated("t1", "b", 5, 5, 5, 0)
	noTask.TaskId = nil

	groups := GroupByTask([]*sessionrecord.SessionRecord{withTask, noTask})

	if len(groups) != 1 || len(groups[0].Rated) != 1 {
		t.Fatalf("expected exactly 1 group with 1 record, got %d groups, %v", len(groups), groups)
	}
}

func TestGroupByTask_SeparatesDifferentTasks(t *testing.T) {
	t1 := rated("t1", "a", 5, 5, 5, 0)
	t2 := rated("t2", "b", 5, 5, 5, 0)

	groups := GroupByTask([]*sessionrecord.SessionRecord{t1, t2})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestTotalTokens(t *testing.T) {
	r := &sessionrecord.SessionRecord{
		Usage: sessionrecord.SessionRecordUsage{Input: 1, Output: 2, CacheRead: 3, CacheWrite: 4},
	}
	if got := TotalTokens(r); got != 10 {
		t.Fatalf("expected 10, got %v", got)
	}
}
