package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

const columnGap = 2

// TablePrinter prints data as aligned tables.
type TablePrinter struct {
	w       io.Writer
	noTrunc bool
}

func (p *TablePrinter) Print(data any, columns []Column) error {
	items := toSlice(data)

	// Build all cell values and compute column widths using display width.
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}

	rows := make([][]string, len(items))
	for r, item := range items {
		vals := make([]string, len(columns))
		for i, col := range columns {
			v := col.Field(item)
			if !p.noTrunc && col.MaxWidth > 0 {
				v = Truncate(v, col.MaxWidth)
			}
			vals[i] = v
		}
		rows[r] = vals
	}

	// Compute max display width per column.
	colWidths := make([]int, len(columns))
	for i, h := range headers {
		colWidths[i] = runewidth.StringWidth(h)
	}
	for _, row := range rows {
		for i, v := range row {
			if w := runewidth.StringWidth(v); w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	// Print header.
	if err := p.printRow(headers, colWidths); err != nil {
		return err
	}
	// Print data rows.
	for _, row := range rows {
		if err := p.printRow(row, colWidths); err != nil {
			return err
		}
	}
	return nil
}

func (p *TablePrinter) printRow(cells []string, colWidths []int) error {
	var sb strings.Builder
	for i, cell := range cells {
		sb.WriteString(cell)
		if i < len(cells)-1 {
			pad := colWidths[i] - runewidth.StringWidth(cell) + columnGap
			sb.WriteString(strings.Repeat(" ", pad))
		}
	}
	_, err := fmt.Fprintln(p.w, sb.String())
	return err
}

// Truncate shortens s to maxLen display columns, appending "..." if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 || runewidth.StringWidth(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return runewidth.Truncate(s, maxLen, "")
	}
	return runewidth.Truncate(s, maxLen, "...")
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
