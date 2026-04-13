package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONPrinter prints data as pretty-printed JSON.
type JSONPrinter struct {
	w io.Writer
}

func (p *JSONPrinter) Print(data any, columns []Column) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	_, err = fmt.Fprintln(p.w, string(out))
	return err
}
