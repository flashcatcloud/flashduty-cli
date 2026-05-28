package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
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
// MonitAgentInvokeTool entries. The `name` key is required; `params` is
// optional and defaults to `{}` so the server-side decoder accepts it. Splits
// each spec on ',' first then on the first '=', mirroring parseKVSlice — that
// means params JSON containing commas isn't supported; specs with complex
// params must keep their objects single-keyed.
func parseToolSpecs(specs []string) ([]flashduty.MonitAgentInvokeTool, error) {
	out := make([]flashduty.MonitAgentInvokeTool, 0, len(specs))
	for _, s := range specs {
		var name string
		params := json.RawMessage("{}")
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
				params = json.RawMessage(v)
			default:
				return nil, fmt.Errorf("unknown key %q in tool-spec", k)
			}
		}
		if name == "" {
			return nil, fmt.Errorf("missing name= in spec %q", s)
		}
		out = append(out, flashduty.MonitAgentInvokeTool{Tool: name, Params: params})
	}
	return out, nil
}
