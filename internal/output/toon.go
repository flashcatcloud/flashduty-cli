package output

import (
	"fmt"
	"io"

	sdk "github.com/flashcatcloud/flashduty-sdk"
)

// TOONPrinter prints data as TOON (Token-Oriented Object Notation). It routes
// through sdk.Marshal so the encoding stays identical to the Flashduty MCP
// server's `--output-format toon` path — one source of truth for how Flashduty
// serializes TOON.
type TOONPrinter struct {
	w io.Writer
}

func (p *TOONPrinter) Print(data any, _ []Column) error {
	out, err := sdk.Marshal(HumanizeTimestamps(data), sdk.OutputFormatTOON)
	if err != nil {
		return fmt.Errorf("failed to marshal TOON: %w", err)
	}
	_, err = fmt.Fprintln(p.w, string(out))
	return err
}
