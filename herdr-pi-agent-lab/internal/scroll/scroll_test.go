package scroll

import (
	"fmt"
	"strings"
	"testing"
)

func longContent(lines int) string {
	rows := make([]string, lines)
	for i := range rows {
		rows[i] = fmt.Sprintf("line %d", i)
	}

	return strings.Join(rows, "\n")
}

func renderOnCopy(m Model, content string) string {
	cp := m

	return cp.Render(content)
}

func TestHandleKey_ScrollDownActuallyMoves(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	content := longContent(50)
	before := renderOnCopy(m, content)

	if !m.HandleKey("ctrl+d", func() string { return content }) {
		t.Fatal("ctrl+d should be consumed")
	}

	after := renderOnCopy(m, content)

	if before == after {
		t.Fatal("ctrl+d did not change the rendered view — scroll did not advance")
	}
}

func TestHandleKey_ScrollUpReturnsToTop(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	content := longContent(50)
	top := renderOnCopy(m, content)

	m.HandleKey("ctrl+d", func() string { return content })
	m.HandleKey("ctrl+d", func() string { return content })

	scrolled := renderOnCopy(m, content)
	if scrolled == top {
		t.Fatal("expected the view to have scrolled away from the top")
	}

	for range 10 {
		m.HandleKey("ctrl+u", func() string { return content })
	}

	back := renderOnCopy(m, content)
	if back != top {
		t.Fatalf("expected scrolling all the way up to return to the original top view\ntop:  %q\nback: %q", top, back)
	}
}

func TestHandleKey_IgnoresOtherKeys(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	if m.HandleKey("j", func() string { return "content" }) {
		t.Fatal("HandleKey should only consume ctrl+u/ctrl+d")
	}
}

func TestGotoTop_ResetsAfterScrolling(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	content := longContent(50)
	top := renderOnCopy(m, content)

	m.HandleKey("ctrl+d", func() string { return content })
	m.HandleKey("ctrl+d", func() string { return content })

	m.GotoTop()

	if got := renderOnCopy(m, content); got != top {
		t.Fatalf("GotoTop did not reset the view to the top\ntop: %q\ngot: %q", top, got)
	}
}
