package compare

import (
	"sort"

	sessionrecord "github.com/darkokos/pi-agent-framework/pi-session-schema/go"
)

type TaskGroup struct {
	TaskID  string
	Rated   []*sessionrecord.SessionRecord
	Unrated []*sessionrecord.SessionRecord
}

func intOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func rankLess(a, b *sessionrecord.SessionRecord) bool {
	if ao, bo := intOrZero(a.Outcome.Outcome), intOrZero(b.Outcome.Outcome); ao != bo {
		return ao > bo
	}
	if ar, br := intOrZero(a.Outcome.Rework), intOrZero(b.Outcome.Rework); ar != br {
		return ar > br
	}
	if aa, ba := intOrZero(a.Outcome.Autonomy), intOrZero(b.Outcome.Autonomy); aa != ba {
		return aa > ba
	}
	return a.Usage.CostUsd < b.Usage.CostUsd
}

func GroupByTask(records []*sessionrecord.SessionRecord) []*TaskGroup {
	groups := map[string]*TaskGroup{}
	var order []string

	for _, r := range records {
		if r.TaskId == nil {
			continue
		}

		taskID := *r.TaskId
		g, ok := groups[taskID]
		if !ok {
			g = &TaskGroup{TaskID: taskID}
			groups[taskID] = g
			order = append(order, taskID)
		}

		if r.Outcome.Rated {
			g.Rated = append(g.Rated, r)
		} else {
			g.Unrated = append(g.Unrated, r)
		}
	}

	result := make([]*TaskGroup, 0, len(order))
	for _, taskID := range order {
		g := groups[taskID]
		sort.SliceStable(g.Rated, func(i, j int) bool {
			return rankLess(g.Rated[i], g.Rated[j])
		})
		result = append(result, g)
	}

	return result
}

func TotalTokens(r *sessionrecord.SessionRecord) float64 {
	return r.Usage.Input + r.Usage.Output + r.Usage.CacheRead + r.Usage.CacheWrite
}
