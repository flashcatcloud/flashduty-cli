package output

import (
	"io"
	"os"
)

// Column defines how to extract and display a field.
type Column struct {
	Header   string
	Field    func(item any) string
	MaxWidth int // 0 = no limit
}

// Printer formats and outputs data.
type Printer interface {
	Print(data any, columns []Column) error
}

// NewPrinter returns the printer for the requested output format. noTrunc only
// affects the table printer; structured formats (JSON/TOON) never truncate.
func NewPrinter(format Format, noTrunc bool, w io.Writer) Printer {
	if w == nil {
		w = os.Stdout
	}
	switch format {
	case FormatJSON:
		return &JSONPrinter{w: w}
	case FormatTOON:
		return &TOONPrinter{w: w}
	default:
		return &TablePrinter{w: w, noTrunc: noTrunc}
	}
}
