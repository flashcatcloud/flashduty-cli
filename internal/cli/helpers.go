package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	gflashduty "github.com/flashcatcloud/go-flashduty"
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

// parseToolSpecs converts a slice of "name=<tool>[,params=<json>]" specs into
// go-flashduty ToolInvokeRequestToolsItem entries. The `name` key is required;
// `params` is optional and defaults to an empty object. Splits each spec on ','
// first then on the first '=', mirroring parseKVSlice — that means params JSON
// containing commas isn't supported; specs with complex params must keep their
// objects single-keyed.
func parseToolSpecs(specs []string) ([]gflashduty.ToolInvokeRequestToolsItem, error) {
	out := make([]gflashduty.ToolInvokeRequestToolsItem, 0, len(specs))
	for _, s := range specs {
		var name string
		var rawParams string
		for _, kv := range strings.Split(s, ",") {
			i := strings.IndexByte(kv, '=')
			if i < 0 {
				return nil, fmt.Errorf("missing '=' in %q", kv)
			}
			k, v := kv[:i], kv[i+1:]
			switch k {
			case "name":
				name = v
			case "params":
				rawParams = v
			default:
				return nil, fmt.Errorf("unknown key %q in tool-spec", k)
			}
		}
		if name == "" {
			return nil, fmt.Errorf("missing name= in spec %q", s)
		}
		// go-flashduty models params as a decoded object. Default to an empty
		// map so no-arg tools serialize as `{}`.
		params := map[string]any{}
		if rawParams != "" {
			if err := json.Unmarshal([]byte(rawParams), &params); err != nil {
				return nil, fmt.Errorf("invalid params JSON in spec %q: %w", s, err)
			}
		}
		out = append(out, gflashduty.ToolInvokeRequestToolsItem{Tool: name, Params: params})
	}
	return out, nil
}
