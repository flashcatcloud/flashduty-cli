package timeutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parse converts a time string to a unix timestamp (seconds).
// Supported formats:
//   - Go duration (relative to now): "5m", "1h", "24h", "168h"
//     Interpreted as "now minus duration"
//   - Date: "2026-04-01" (parsed as local midnight)
//   - Datetime: "2026-04-01 10:00:00" (parsed as local time)
//   - Unix timestamp: "1712000000" (passed through)
func Parse(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "now" {
		return time.Now().Unix(), nil
	}

	// Try Go duration (relative)
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d).Unix(), nil
	}

	// Try date: 2006-01-02
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t.Unix(), nil
	}

	// Try datetime: 2006-01-02 15:04:05
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t.Unix(), nil
	}

	// Try unix timestamp
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil && ts > 1000000000 {
		return ts, nil
	}

	return 0, fmt.Errorf("unable to parse time %q: expected duration (24h), date (2006-01-02), datetime (2006-01-02 15:04:05), or unix timestamp", s)
}
