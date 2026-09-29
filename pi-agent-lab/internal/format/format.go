package format

import (
	"io"
	"text/tabwriter"

	"github.com/charmbracelet/glamour"
)

func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
}

func RenderMarkdown(md string) string {
	rendered, err := glamour.Render(md, "auto")
	if err != nil {
		return md
	}

	return rendered
}
