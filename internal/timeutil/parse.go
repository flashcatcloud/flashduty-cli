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
//   - Future duration with "+" prefix: "+24h", "+7d"
//     Interpreted as "now plus duration"
//   - Day shorthand: "7d", "30d" — converted to hours automatically
//   - Date: "2026-04-01" (parsed as local midnight)
//   - Datetime: "2026-04-01 10:00:00" or "2026-04-01T10:00:00" (parsed as local time)
//   - RFC3339 with timezone: "2026-04-01T10:00:00+08:00" / "...Z" (the format the SDK emits)
//   - Unix timestamp in seconds: "1712000000" (passed through)
//   - Unix timestamp in milliseconds: "1712000000000" (divided down to seconds;
//     distinguished from seconds by magnitude — see msThreshold below)
func Parse(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "now" {
		return time.Now().Unix(), nil
	}

	// Check for future duration prefix "+"
	future := false
	raw := s
	if strings.HasPrefix(s, "+") {
		future = true
		raw = s[1:]
	}

	// Try Go duration (relative), expanding "d" shorthand first
	durStr := expandDays(raw)
	if d, err := time.ParseDuration(durStr); err == nil {
		if d < 0 {
			return 0, fmt.Errorf("negative duration %q is not supported", s)
		}
		if future {
			return time.Now().Add(d).Unix(), nil
		}
		return time.Now().Add(-d).Unix(), nil
	}

	// Try RFC3339 / ISO8601 with an explicit timezone first. This is the
	// format the flashduty SDK renders timestamps in (e.g.
	// "2026-05-29T00:00:00+08:00"), so the agent naturally round-trips those
	// values straight back as --since/--until. time.Parse honors the embedded
	// offset ("Z" or "+08:00").
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Unix(), nil
		}
	}

	// Try date: 2006-01-02
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t.Unix(), nil
	}

	// Try datetime without timezone, space- or "T"-separated → local time.
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t.Unix(), nil
		}
	}

	// Try unix timestamp, seconds or milliseconds. A value at or above
	// msThreshold (100 billion) would be seconds only for a date past the
	// year 5138, so any realistic timestamp that large must be milliseconds.
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		switch {
		case ts >= msThreshold:
			return ts / 1000, nil
		case ts > 1000000000:
			return ts, nil
		}
	}

	return 0, fmt.Errorf("unable to parse time %q: expected duration (24h), RFC3339 (2006-01-02T15:04:05Z07:00), date (2006-01-02), datetime (2006-01-02 15:04:05), or unix timestamp in seconds or milliseconds", s)
}

// msThreshold is the magnitude cutoff separating a unix timestamp in seconds
// from one in milliseconds: 100,000,000,000 as seconds is the year 5138, so
// any input at or above it is treated as milliseconds instead.
const msThreshold = 100_000_000_000

// expandDays converts day shorthand (e.g. "7d", "30d") to hours for time.ParseDuration.
// If the string does not end with "d" or is not purely numeric before it, returns as-is.
func expandDays(s string) string {
	if !strings.HasSuffix(s, "d") {
		return s
	}
	numPart := s[:len(s)-1]
	if days, err := strconv.Atoi(numPart); err == nil {
		return fmt.Sprintf("%dh", days*24)
	}
	return s
}
