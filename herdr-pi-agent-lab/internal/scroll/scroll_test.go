package scroll

import (
	"fmt"
	"strings"
	"testing"

	"github.com/darkokos/pi-agent-framework/herdr-pi-agent-lab/internal/theme"
)

func longContent(lines int) string {
	rows := make([]string, lines)
	for i := range rows {
		rows[i] = fmt.Sprintf("line %d", i)
	}

	return strings.Join(rows, "\n")
}

func noHeader() string { return "" }
func noFooter() string { return "" }

func renderOnCopy(m Model, header, body string) string {
	cp := m

	return cp.Render(header, body)
}

func lastLine(out string) string {
	lines := strings.Split(out, "\n")

	return strings.TrimRight(lines[len(lines)-1], " ")
}

func firstLine(out string) string {
	lines := strings.Split(out, "\n")

	return strings.TrimRight(lines[0], " ")
}

func TestHandleKey_ScrollDownActuallyMoves(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	content := longContent(50)
	before := renderOnCopy(m, "header", content)

	if !m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return content }, noFooter) {
		t.Fatal("ctrl+d should be consumed")
	}

	after := renderOnCopy(m, "header", content)

	if before == after {
		t.Fatal("ctrl+d did not change the rendered view — scroll did not advance")
	}
}

func TestHandleKey_ScrollUpReturnsToTop(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	content := longContent(50)
	top := renderOnCopy(m, "header", content)

	m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return content }, noFooter)
	m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return content }, noFooter)

	scrolled := renderOnCopy(m, "header", content)
	if scrolled == top {
		t.Fatal("expected the view to have scrolled away from the top")
	}

	for range 10 {
		m.HandleKey("ctrl+u", func() string { return "header" }, func() string { return content }, noFooter)
	}

	back := renderOnCopy(m, "header", content)
	if back != top {
		t.Fatalf("expected scrolling all the way up to return to the original top view\ntop:  %q\nback: %q", top, back)
	}
}

func TestHandleKey_IgnoresOtherKeys(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	if m.HandleKey("j", noHeader, func() string { return "content" }, noFooter) {
		t.Fatal("HandleKey should only consume ctrl+u/ctrl+d")
	}
}

func TestGotoTop_ResetsAfterScrolling(t *testing.T) {
	var m Model
	m.Resize(20, 5)

	content := longContent(50)
	top := renderOnCopy(m, "header", content)

	m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return content }, noFooter)
	m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return content }, noFooter)

	m.GotoTop()

	if got := renderOnCopy(m, "header", content); got != top {
		t.Fatalf("GotoTop did not reset the view to the top\ntop: %q\ngot: %q", top, got)
	}
}

func TestRenderWithFooter_FooterIsAlwaysTheLastLine(t *testing.T) {
	var m Model
	m.Resize(20, 10)

	body := longContent(50)
	footer := "help line"

	out := m.RenderWithFooter("header", body, footer)
	lines := strings.Split(out, "\n")

	if got := lastLine(out); got != footer {
		t.Fatalf("expected footer as the last line, got %q", got)
	}
	if len(lines) != 10 {
		t.Fatalf("expected total output to fill the full pane height (10 lines) since body overflows it, got %d: %q", len(lines), out)
	}
}

func TestRenderWithFooter_FooterStaysPinnedAfterScrolling(t *testing.T) {
	var m Model
	m.Resize(20, 10)

	body := longContent(50)
	footer := "help line"

	before := m.RenderWithFooter("header", body, footer)

	m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return body }, func() string { return footer })
	m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return body }, func() string { return footer })

	after := m.RenderWithFooter("header", body, footer)

	if before == after {
		t.Fatal("expected the body to have scrolled, but output is identical")
	}

	lines := strings.Split(after, "\n")
	if got := lastLine(after); got != footer {
		t.Fatalf("footer should stay pinned at the bottom after scrolling, got last line %q", got)
	}
	if len(lines) != 10 {
		t.Fatalf("expected total output to still fill the full pane height after scrolling, got %d", len(lines))
	}
}

func TestRenderWithFooter_MultiLineFooterReservesItsOwnHeight(t *testing.T) {
	var m Model
	m.Resize(20, 10)

	body := longContent(50)
	footer := "line one\nline two"

	out := m.RenderWithFooter("header", body, footer)
	lines := strings.Split(out, "\n")

	if len(lines) != 10 {
		t.Fatalf("expected total output to fill the full pane height (10 lines), got %d: %q", len(lines), out)
	}
	if got := strings.TrimRight(lines[len(lines)-2], " "); got != "line one" {
		t.Fatalf("expected the footer's first line intact, got %q", got)
	}
	if got := lastLine(out); got != "line two" {
		t.Fatalf("expected the footer's second line intact at the bottom, got %q", got)
	}
}

func TestRenderWithFooter_ShortBodyDoesNotStretchToFillThePane(t *testing.T) {
	var m Model
	m.Resize(20, 10)

	body := "line 1\nline 2\nline 3"
	footer := "help"

	out := m.RenderWithFooter("header", body, footer)
	lines := strings.Split(out, "\n")

	if len(lines) != 7 {
		t.Fatalf("expected footer to sit right after the short body instead of stretching to the full pane height, got %d lines: %q", len(lines), out)
	}
	if got := lastLine(out); got != footer {
		t.Fatalf("expected footer as the last line, got %q", got)
	}
}

func TestRenderWithFooter_LongFooterWrapsInsteadOfBeingCropped(t *testing.T) {
	var m Model
	m.Resize(10, 20)

	footer := "this help text is definitely longer than ten columns wide"

	out := m.RenderWithFooter("h", "x", footer)
	lines := strings.Split(out, "\n")

	if len(lines) <= 5 {
		t.Fatalf("expected the footer to wrap onto multiple lines at width 10, got %d total line(s): %q", len(lines), out)
	}

	for i, line := range lines {
		if got := len([]rune(line)); got > m.width {
			t.Fatalf("line %d is %d columns wide, wider than the pane (%d) — footer ran off the edge instead of wrapping: %q", i, got, m.width, line)
		}
	}

	for word := range strings.FieldsSeq(footer) {
		if !strings.Contains(out, word) {
			t.Fatalf("expected footer word %q to survive wrapping, got %q", word, out)
		}
	}
}

func TestHandleKey_ScrollingWithAFooterActuallyAdvances(t *testing.T) {
	var m Model
	m.Resize(20, 10)

	body := longContent(50)
	footer := "help line"

	renderCopy := func() string {
		cp := m

		return cp.RenderWithFooter("header", body, footer)
	}

	before := renderCopy()

	if !m.HandleKey("ctrl+d", func() string { return "header" }, func() string { return body }, func() string { return footer }) {
		t.Fatal("ctrl+d should be consumed")
	}

	after := renderCopy()

	if before == after {
		t.Fatal("ctrl+d did not change the rendered view — scroll did not advance with a footer present")
	}
}

func TestRender_HeaderIsAlwaysTheFirstLine(t *testing.T) {
	var m Model
	m.Resize(20, 10)

	out := m.Render("my header", longContent(50))

	if got := firstLine(out); got != "my header" {
		t.Fatalf("expected header as the first line, got %q", got)
	}
}

func TestRender_HeaderStaysPinnedAfterScrolling(t *testing.T) {
	var m Model
	m.Resize(20, 10)

	body := longContent(50)
	header := "my header"

	m.HandleKey("ctrl+d", func() string { return header }, func() string { return body }, noFooter)
	m.HandleKey("ctrl+d", func() string { return header }, func() string { return body }, noFooter)

	out := m.Render(header, body)

	if got := firstLine(out); got != header {
		t.Fatalf("header should stay pinned at the top after scrolling, got first line %q", got)
	}
}

func TestRender_MultiLineHeaderReservesItsOwnHeight(t *testing.T) {
	var m Model
	m.Resize(10, 10)

	header := "a rather long header that will wrap"
	body := longContent(50)

	wantHeaderLines := strings.Count(theme.Wrap(10, header), "\n") + 1
	if wantHeaderLines < 2 {
		t.Fatalf("test setup: expected the header to actually wrap onto multiple lines, got %d", wantHeaderLines)
	}

	out := m.Render(header, body)
	lines := strings.Split(out, "\n")

	if len(lines) != m.fullHeight {
		t.Fatalf("expected total output to fill exactly the pane height (%d) since body overflows it, got %d: %q", m.fullHeight, len(lines), out)
	}
	if got := strings.Join(lines[:wantHeaderLines], "\n"); got != theme.Wrap(10, header) {
		t.Fatalf("expected the header's %d wrapped lines intact at the top, got %q", wantHeaderLines, got)
	}
}

func TestHandleKey_ScrollingWithAWrappingHeaderActuallyAdvances(t *testing.T) {
	var m Model
	m.Resize(10, 10)

	header := "a rather long header that will wrap"
	body := longContent(50)

	renderCopy := func() string {
		cp := m

		return cp.Render(header, body)
	}

	before := renderCopy()

	if !m.HandleKey("ctrl+d", func() string { return header }, func() string { return body }, noFooter) {
		t.Fatal("ctrl+d should be consumed")
	}

	after := renderCopy()

	if before == after {
		t.Fatal("ctrl+d did not change the rendered view — scroll did not advance with a wrapping header present")
	}
}
