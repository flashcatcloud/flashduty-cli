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

// NewPrinter returns a table or JSON printer based on mode flags.
func NewPrinter(jsonMode bool, noTrunc bool, w io.Writer) Printer {
	if w == nil {
		w = os.Stdout
	}
	if jsonMode {
		return &JSONPrinter{w: w}
	}
	return &TablePrinter{w: w, noTrunc: noTrunc}
}
