package format

import (
	"io"
	"text/tabwriter"
)

func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
}
