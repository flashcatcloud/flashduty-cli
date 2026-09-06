package cli

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/flashcatcloud/go-flashduty"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// TOON and the table renderer treat RawMessage as []byte. Expand only the new
// tool envelope at this presentation boundary; JSON output stays byte-exact.
func datasourceToolOutput(value any, format output.Format) (any, error) {
	if _, ok := value.(*flashduty.DatasourceToolResult); !ok || format == output.FormatJSON {
		return value, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	return toolDisplayNumbers(decoded), nil
}

// toon-go rounds json.Number through float64. Preserve integer width and use
// decimal strings for values its numeric representation cannot retain.
func toolDisplayNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		if n, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			return n
		}
		if n, err := strconv.ParseUint(v.String(), 10, 64); err == nil {
			return n
		}
		if n, err := strconv.ParseFloat(v.String(), 64); err == nil && strconv.FormatFloat(n, 'g', -1, 64) == v.String() {
			return n
		}
		return v.String()
	case map[string]any:
		for key, item := range v {
			v[key] = toolDisplayNumbers(item)
		}
	case []any:
		for i, item := range v {
			v[i] = toolDisplayNumbers(item)
		}
	}
	return value
}
