package format

import (
	"io"
	"text/tabwriter"
	"time"
)

func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
}

func UnixMilli(ms int64) string {
	return time.UnixMilli(ms).Format(time.RFC3339)
}
