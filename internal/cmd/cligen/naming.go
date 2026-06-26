package main

// Copied verbatim from go-flashduty/internal/cmd/gen/naming.go so the CLI
// generator derives the SAME service type and method names the SDK generator
// did. Keep in sync when the SDK's naming changes (rare).

import (
	"strings"
)

// initialisms are tokens forced to a canonical capitalization in Go identifiers.
var initialisms = map[string]string{
	"id": "ID", "ids": "IDs", "api": "API", "url": "URL", "uri": "URI",
	"http": "HTTP", "https": "HTTPS", "json": "JSON", "html": "HTML",
	"sls": "SLS", "csv": "CSV", "sms": "SMS", "md5": "MD5", "sha": "SHA",
	"ip": "IP", "ai": "AI", "db": "DB", "os": "OS", "rum": "RUM", "mfa": "MFA",
	"sso": "SSO", "saml": "SAML", "oidc": "OIDC", "ldap": "LDAP", "ts": "TS",
	"ack": "Ack", "ok": "OK", "ttl": "TTL", "cpu": "CPU", "qps": "QPS",
	"sla": "SLA", "mttr": "MTTR", "utc": "UTC", "tz": "TZ", "ui": "UI",
}

// tokens splits an identifier (kebab, snake, or camelCase) into lowercase words.
func tokens(s string) []string {
	var out []string
	for _, part := range splitNonAlnum(s) {
		out = append(out, splitCamel(part)...)
	}
	return out
}

func splitNonAlnum(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		//nolint:staticcheck // QF1001: keep the explicit !(isAlnum) form; De Morgan rewrite is less readable (matches go-flashduty .golangci.yml).
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
}

func splitCamel(s string) []string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return nil
	}
	isUpper := func(r rune) bool { return r >= 'A' && r <= 'Z' }
	isLower := func(r rune) bool { return r >= 'a' && r <= 'z' }

	var words []string
	start := 0
	for i := 1; i < n; i++ {
		prev, cur := runes[i-1], runes[i]
		boundary := false
		switch {
		case isLower(prev) && isUpper(cur):
			boundary = true
		case isUpper(prev) && isUpper(cur) && i+1 < n && isLower(runes[i+1]):
			boundary = true
		}
		if boundary {
			words = append(words, strings.ToLower(string(runes[start:i])))
			start = i
		}
	}
	words = append(words, strings.ToLower(string(runes[start:])))
	return words
}

func pascalToken(w string) string {
	if v, ok := initialisms[strings.ToLower(w)]; ok {
		return v
	}
	return strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
}

func pascal(toks []string) string {
	var b strings.Builder
	for _, t := range toks {
		b.WriteString(pascalToken(t))
	}
	return b.String()
}

// serviceName derives a service type prefix from a two-level tag's last segment.
func serviceName(tag string) string {
	seg := tag
	if i := strings.LastIndex(tag, "/"); i >= 0 {
		seg = tag[i+1:]
	}
	return pascal(tokens(seg))
}

func commonPrefixLen(opTokens [][]string) int {
	if len(opTokens) == 0 {
		return 0
	}
	n := 0
	for {
		for _, t := range opTokens {
			if len(t) <= n+1 {
				return n
			}
		}
		first := strings.ToLower(opTokens[0][n])
		for _, t := range opTokens[1:] {
			if strings.ToLower(t[n]) != first {
				return n
			}
		}
		n++
	}
}

var methodPrefixByTag = map[string][]string{
	"On-call/Incidents":    {"incident"},
	"On-call/Integrations": {"webhook", "history"},
}

// methodNames computes a unique Go method name for each operationId within a
// service by stripping stable service-specific leading tokens, then deduping.
func methodNames(tag string, opIDs []string) map[string]string {
	opTokens := make([][]string, len(opIDs))
	for i, id := range opIDs {
		opTokens[i] = tokens(id)
	}
	n := commonPrefixLen(opTokens)

	result := make(map[string]string, len(opIDs))
	used := make(map[string]int)
	for i, id := range opIDs {
		toks := opTokens[i][n:]
		if prefix, ok := methodPrefixByTag[tag]; ok && hasTokenPrefix(opTokens[i], prefix) {
			toks = opTokens[i][len(prefix):]
		}
		if len(toks) == 0 {
			toks = opTokens[i]
		}
		name := pascal(toks)
		if name == "" {
			name = pascal(opTokens[i])
		}
		if c := used[name]; c > 0 {
			full := pascal(opTokens[i])
			if used[full] == 0 {
				name = full
			} else {
				name += itoa(c)
			}
		}
		used[name]++
		result[id] = name
	}
	return result
}

func hasTokenPrefix(toks, prefix []string) bool {
	if len(toks) <= len(prefix) {
		return false
	}
	for i, p := range prefix {
		if strings.ToLower(toks[i]) != p {
			return false
		}
	}
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// kebab converts an identifier to kebab-case (lower words joined by '-'), used
// for CLI verb/flag names derived from wire names and path segments.
func kebab(s string) string {
	return strings.Join(tokens(s), "-")
}
