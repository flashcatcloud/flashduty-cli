package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"
)

// TablePrinter prints data as aligned tables.
type TablePrinter struct {
	w       io.Writer
	noTrunc bool
}

func (p *TablePrinter) Print(data any, columns []Column) error {
	items := toSlice(data)

	tw := tabwriter.NewWriter(p.w, 0, 4, 2, ' ', 0)

	// Header
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	// Rows
	for _, item := range items {
		vals := make([]string, len(columns))
		for i, col := range columns {
			v := col.Field(item)
			if !p.noTrunc && col.MaxWidth > 0 {
				v = Truncate(v, col.MaxWidth)
			}
			vals[i] = v
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	return tw.Flush()
}

// Truncate shortens s to maxLen, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// FormatTime formats a unix timestamp as local time.
func FormatTime(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Local().Format("2006-01-02 15:04")
}

// toSlice converts data to a []any using reflection.
func toSlice(data any) []any {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice {
		result := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = v.Index(i).Interface()
		}
		return result
	}
	return []any{data}
}
