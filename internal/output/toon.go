package output

import (
	"fmt"
	"io"

	toon "github.com/toon-format/toon-go"
)

// TOONPrinter prints data as TOON (Token-Oriented Object Notation). It routes
// through toon.Marshal directly — the same encoder the Flashduty SDKs and MCP
// server use, so the on-the-wire encoding stays identical across tools.
type TOONPrinter struct {
	w io.Writer
}

func (p *TOONPrinter) Print(data any, _ []Column) error {
	out, err := toon.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal TOON: %w", err)
	}
	_, err = fmt.Fprintln(p.w, string(out))
	return err
}
