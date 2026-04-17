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
	if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
		return err
	}

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
		if _, err := fmt.Fprintln(tw, strings.Join(vals, "\t")); err != nil {
			return err
		}
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

// FormatDuration formats seconds into human-readable duration (e.g., "2m 30s", "1h 15m").
func FormatDuration(seconds int) string {
	if seconds <= 0 {
		return "-"
	}
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// FormatDurationFloat formats float64 seconds into human-readable duration.
func FormatDurationFloat(seconds float64) string {
	return FormatDuration(int(seconds))
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
