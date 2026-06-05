package cli

import (
	"fmt"
	"strings"
)

// parseKVSlice converts a slice of "KEY=VALUE" entries into a map.
// Returns nil (not an error) for an empty input so callers can pass nil
// maps through to the SDK without triggering omitempty issues.
func parseKVSlice(entries []string) (map[string]string, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		i := strings.IndexByte(e, '=')
		if i < 0 {
			return nil, fmt.Errorf("missing '=': %q", e)
		}
		out[e[:i]] = e[i+1:]
	}
	return out, nil
}
