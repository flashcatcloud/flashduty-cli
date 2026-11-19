package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
)

var digestT0 = time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)

func exploreFixture(t *testing.T, result any, extra map[string]any) *flashduty.DatasourceToolResult {
	t.Helper()
	data := map[string]any{"format": "explore_result.v1", "result": result}
	for k, v := range extra {
		data[k] = v
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return &flashduty.DatasourceToolResult{DatasourceID: 7, Tool: "prometheus.query", Data: raw}
}

// seriesFrame builds a time_series frame with points every step from digestT0.
func seriesFrame(labels map[string]string, step time.Duration, values []any) map[string]any {
	times := make([]any, len(values))
	for i := range values {
		times[i] = digestT0.Add(time.Duration(i) * step).Format(time.RFC3339Nano)
	}
	return map[string]any{"kind": "time_series", "fields": []any{
		map[string]any{"name": "time", "type": "time", "values": times},
		map[string]any{"name": "value", "type": "float", "labels": labels, "values": values},
	}}
}

// rampValues returns n points rising linearly from base, with one peak.
func rampValues(n int, base, slope float64, peakAt int, peak float64) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = base + slope*float64(i)
	}
	if peakAt >= 0 {
		out[peakAt] = peak
	}
	return out
}

func logEntries(n int, msg func(i int) string) []any {
	out := make([]any, n)
	for i := range out {
		ts := digestT0.Add(time.Duration(i) * time.Second)
		status := "200"
		if i%4 == 0 {
			status = "500"
		}
		out[i] = map[string]any{
			"timestamp_ns": fmt.Sprint(ts.UnixNano()),
			"fields": map[string]any{
				"service": "checkout",
				"level":   "info",
				"status":  status,
				"msg":     msg(i),
				"_time":   ts.Format(time.RFC3339Nano),
			},
		}
	}
	return out
}

// fatLogEntries builds n entries of 18 fields; entry 3 carries a long message.
func fatLogEntries(n int) []any {
	out := make([]any, n)
	for i := range out {
		ts := digestT0.Add(time.Duration(i) * 250 * time.Millisecond)
		msg := fmt.Sprintf("GET /api/orders/%d completed", 5000+i)
		if i == 3 {
			msg = "order validation failed: " + strings.Repeat("item quantity exceeds available stock; ", 30)
		}
		out[i] = map[string]any{
			"timestamp_ns": fmt.Sprint(ts.UnixNano()),
			"fields": map[string]any{
				"_time":       ts.Format(time.RFC3339Nano),
				"message":     msg,
				"level":       []string{"info", "info", "warn", "error"}[i%4],
				"service":     "checkout",
				"namespace":   "shop",
				"host":        fmt.Sprintf("node-%d", i%2+1),
				"pod":         fmt.Sprintf("checkout-7f9c-%c", 'a'+i%3),
				"container":   "app",
				"method":      "GET",
				"path":        fmt.Sprintf("/api/orders/%d", 5000+i),
				"referer":     fmt.Sprintf("https://shop.example.com/cart?step=review&items=sku-1001,sku-1002,sku-1003,sku-1004,sku-1005,sku-1006&coupon=WINTER%02d&ref=newsletter-winter-campaign-week-%d-variant-b", i, i),
				"url":         fmt.Sprintf("https://shop.example.com/api/orders/%d?expand=items,payments,shipping&currency=EUR&locale=de-DE&session=%08x", 5000+i, 0x5eed0000+i),
				"status":      []string{"200", "200", "429", "500"}[i%4],
				"duration_ms": fmt.Sprint(12 + i*7),
				"bytes":       fmt.Sprint(512 + i*64),
				"trace_id":    fmt.Sprintf("4bf92f3577b34da6a3ce929d0e0e%04x", i),
				"span_id":     fmt.Sprintf("00f067aa0ba9%04x", i),
				"user_agent":  "synthetic-client/1.0 (load-test; +https://example.com/bot)",
			},
		}
	}
	return out
}

// levelLogEntries builds n entries whose level cycles info/info/warn/error.
func levelLogEntries(n int) []any {
	out := make([]any, n)
	for i := range out {
		ts := digestT0.Add(time.Duration(i) * 15 * time.Second)
		out[i] = map[string]any{
			"timestamp_ns": fmt.Sprint(ts.UnixNano()),
			"fields": map[string]any{
				"msg":      fmt.Sprintf("payment attempt %d", i),
				"level":    []string{"info", "info", "warn", "error"}[i%4],
				"service":  "payments",
				"order_id": fmt.Sprintf("ord-%05d", 70000+i),
			},
		}
	}
	return out
}

// wideLogEntries builds n entries with a unique id, a repeated note longer
// than the distribution value cap, the given number of 100-char constant
// fields, and the given number of two-valued fields.
func wideLogEntries(n, constants, twoValued int) []any {
	out := make([]any, n)
	for i := range out {
		fields := map[string]any{
			"id":   fmt.Sprintf("req-%04d", i),
			"note": []string{"retry scheduled after upstream timeout on inventory lookup", "request served from cache without contacting upstream"}[i%2],
		}
		for c := range constants {
			fields[fmt.Sprintf("c%02d", c)] = strings.Repeat(string(rune('a'+c%26)), 100)
		}
		for f := range twoValued {
			fields[fmt.Sprintf("f%02d", f)] = []string{"a", "b"}[i%2]
		}
		out[i] = map[string]any{"timestamp_ns": fmt.Sprint(digestT0.Add(time.Duration(i) * time.Second).UnixNano()), "fields": fields}
	}
	return out
}

func digestFixtures(t *testing.T) map[string]*flashduty.DatasourceToolResult {
	tableRows := 500
	cols := []any{
		map[string]any{"name": "service", "type": "string", "values": make([]any, tableRows)},
		map[string]any{"name": "status", "type": "string", "values": make([]any, tableRows)},
		map[string]any{"name": "orders", "type": "float", "values": make([]any, tableRows)},
	}
	for i := range tableRows {
		cols[0].(map[string]any)["values"].([]any)[i] = fmt.Sprintf("service-%03d", i)
		cols[1].(map[string]any)["values"].([]any)[i] = "ok"
		cols[2].(map[string]any)["values"].([]any)[i] = i * 3
	}
	many := make([]any, 134)
	for i := range many {
		many[i] = seriesFrame(map[string]string{"service": fmt.Sprintf("svc-%03d", i)}, time.Minute, rampValues(30, float64(i), 0.5, -1, 0))
	}
	manySamples := make([]any, 30)
	for i := range manySamples {
		manySamples[i] = map[string]any{"labels": map[string]string{
			"__name__": "http_requests_in_flight", "job": "api", "namespace": "shop", "pod": fmt.Sprintf("orders-%02d", i),
		}, "value": float64(i) / 4}
	}
	shared := func(service string) map[string]string {
		return map[string]string{"job": "api", "namespace": "shop", "service": service}
	}
	return map[string]*flashduty.DatasourceToolResult{
		"samples": exploreFixture(t, map[string]any{"kind": "samples", "samples": []any{
			map[string]any{"labels": map[string]string{"service": "checkout", "instance": "node-1:9100"}, "value": 0.25},
			map[string]any{"labels": map[string]string{"service": "checkout", "instance": "node-2:9100"}, "value": 12.5},
			map[string]any{"labels": map[string]string{"service": "search", "instance": "node-3:9100"}, "value": "NaN"},
			map[string]any{"labels": map[string]string{"service": "search", "instance": "node-4:9100"}, "value": 1234567},
			map[string]any{"labels": map[string]string{"service": "cart", "instance": "node-5:9100"}, "value": "+Inf"},
		}}, nil),
		"samples_many": exploreFixture(t, map[string]any{"kind": "samples", "samples": manySamples}, nil),
		"time_series": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			seriesFrame(shared("checkout"), 20*time.Second, rampValues(240, 0.1, 0.01, 150, 9.75)),
			seriesFrame(shared("search"), 20*time.Second, rampValues(240, 1, 0, 3, 3.5)),
			seriesFrame(shared("cart"), 20*time.Second, append(rampValues(238, 0.5, 0.002, -1, 0), "NaN", "+Inf")),
		}}, map[string]any{"execution": map[string]any{"kind": "range", "effective_step_seconds": 20}}),
		"time_series_short": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			seriesFrame(map[string]string{"service": "checkout"}, time.Minute, []any{1, 2.5, "NaN", 4}),
		}}, nil),
		"time_series_seconds": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			seriesFrame(nil, 20*time.Second, []any{1, 2, 3}),
		}}, nil),
		"time_series_overnight": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			seriesFrame(map[string]string{"service": "checkout"}, 2*time.Hour, rampValues(13, 0, 1, -1, 0)),
		}}, nil),
		"time_series_many": exploreFixture(t, map[string]any{"kind": "frames", "frames": many}, nil),
		"time_series_unaligned": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			seriesFrame(map[string]string{"service": "checkout"}, time.Minute, rampValues(13, 0, 1, -1, 0)),
			seriesFrame(map[string]string{"service": "search"}, time.Minute, []any{20, 30, 25}),
		}}, nil),
		"table": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			map[string]any{"kind": "table", "fields": []any{
				map[string]any{"name": "service", "type": "string", "values": []any{"checkout", "search", "cart"}},
				map[string]any{"name": "orders", "type": "float", "values": []any{120, 45.5, nil}},
				map[string]any{"name": "updated", "type": "time", "values": []any{"2026-01-02T03:04:05Z", "2026-01-02T03:05:00Z", "2026-01-02T03:06:00Z"}},
			}},
		}}, nil),
		"table_long": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			map[string]any{"kind": "table", "fields": cols},
		}}, nil),
		"table_wide_cell": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			map[string]any{"kind": "table", "fields": []any{
				map[string]any{"name": "query", "type": "string", "values": []any{strings.Repeat("select * from orders where id < 10;\n", 20)}},
			}},
		}}, nil),
		"logs": exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 20, "has_more": true,
			"entries": logEntries(8, func(i int) string { return fmt.Sprintf("order %d placed", 1000+i) })}, nil),
		"logs_long": exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 500, "has_more": false,
			"entries": logEntries(500, func(i int) string { return fmt.Sprintf("%d %s", i, strings.Repeat("payload ", 60)) })}, nil),
		"logs_fat":            exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 8, "has_more": true, "entries": fatLogEntries(8)}, nil),
		"logs_level":          exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 100, "has_more": false, "entries": levelLogEntries(40)}, nil),
		"logs_lowcard_limits": exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 30, "has_more": false, "entries": wideLogEntries(30, 0, 10)}, nil),
		"logs_meta_budget":    exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 30, "has_more": false, "entries": wideLogEntries(30, 40, 0)}, nil),
		"logs_int_ts": exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 1, "has_more": false, "entries": []any{
			map[string]any{"timestamp_ns": json.Number("1767322800000000000"), "fields": map[string]any{"msg": "ready"}},
		}}, nil),
		"empty_samples": exploreFixture(t, map[string]any{"kind": "samples", "samples": []any{}}, nil),
		"empty_frames":  exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{}}, nil),
		"empty_table": exploreFixture(t, map[string]any{"kind": "frames", "frames": []any{
			map[string]any{"kind": "table", "fields": []any{map[string]any{"name": "service", "type": "string", "values": []any{}}}},
		}}, nil),
		"empty_logs": exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 20, "has_more": false, "entries": []any{}}, nil),
	}
}

func TestExploreDigest(t *testing.T) {
	pinClock(t, digestT0)
	fixtures := digestFixtures(t)
	for _, tc := range []struct {
		name    string
		want    []string
		notWant []string
	}{
		{"samples", []string{
			"samples: 5, sorted by value descending\n",
			"non-finite or non-numeric values: 2\n",
			"  +Inf     {instance=\"node-5:9100\", service=\"cart\"}\n  1234567  {instance=\"node-4:9100\", service=\"search\"}\n  12.5     ",
			"  NaN      {instance=\"node-3:9100\", service=\"search\"}\n" + samplesFooter,
		}, []string{"median"}},
		{"samples_many", []string{
			"samples: 30, sorted by value descending\n" +
				"common labels: {__name__=\"http_requests_in_flight\", job=\"api\", namespace=\"shop\"}\n" +
				"  7.25  {pod=\"orders-29\"}\n",
			"  0     {pod=\"orders-00\"}\n" + samplesFooter,
		}, []string{"showing", "median"}},
		{"time_series", []string{
			"time_series: 3 series (3 frames), sorted by max descending; step 20s\n" +
				"common labels: {job=\"api\", namespace=\"shop\"}\n" +
				"all series: 240 points, 2026-01-02T11:00:00+08:00 .. 2026-01-02T12:19:40+08:00\n" +
				"[1] {service=\"checkout\"}\n" +
				"    max 9.75 at 2026-01-02T11:50:00+08:00, min 0.1 at 2026-01-02T11:00:00+08:00\n" +
				"    avg 1.329, first 0.1, last 2.49\n" +
				"    shape (bucket start → max, 6m40s buckets): 11:00 0.29 | 11:06 0.49 | 11:13 0.69 | 11:20 0.89 | 11:26 1.09 | 11:33 1.29 | 11:40 1.49 | 11:46 9.75 | 11:53 1.89 | 12:00 2.09 | 12:06 2.29 | 12:13 2.49\n" +
				"[2] {service=\"search\"}\n",
			"[3] {service=\"cart\"}  (2 non-finite excluded from stats)\n",
			"avg 0.737, first 0.5, last non-finite\n",
		}, nil},
		{"time_series_short", []string{
			"common labels: {service=\"checkout\"}\n",
			"[1] {}  (1 non-finite excluded from stats)\n",
			"    points: 11:00 1 | 11:01 2.5 | 11:02 non-finite | 11:03 4\n",
			"max 4 at 2026-01-02T11:03:00+08:00, min 1 at 2026-01-02T11:00:00+08:00",
			"avg 2.5, first 1, last 4",
		}, []string{"shape"}},
		{"time_series_seconds", []string{"    points: 11:00:00 1 | 11:00:20 2 | 11:00:40 3\n"}, nil},
		{"time_series_overnight", []string{"(bucket start → max, 2h10m buckets): 01-02T11:00 1 | 01-02T13:10 2 | ", " | 01-03T00:00 7 | ", " | 01-03T10:50 12\n"}, nil},
		{"time_series_many", []string{"time_series: showing 18 of 134 series (134 frames), sorted by max descending\n", "[1] {service=\"svc-133\"}\n", "[18] {service=\"svc-116\"}\n"}, []string{"svc-115", "common labels"}},
		{"time_series_unaligned", []string{
			"time_series: 2 series (2 frames), sorted by max descending\n[1] {service=\"search\"}  3 points, 2026-01-02T11:00:00+08:00 .. 2026-01-02T11:02:00+08:00\n",
			"[2] {service=\"checkout\"}  13 points, 2026-01-02T11:00:00+08:00 .. 2026-01-02T11:12:00+08:00\n",
			"    shape (bucket start → max, 1m5s buckets): 11:00 1 | 11:01 2 | 11:02 3 | ",
		}, []string{"all series", "common labels"}},
		{"table", []string{
			"table: 3 rows × 3 columns\n  service | orders | updated\n  checkout | 120 | 2026-01-02T11:04:05+08:00\n  search | 45.5 | 2026-01-02T11:05:00+08:00\n  cart | null | 2026-01-02T11:06:00+08:00\n",
		}, nil},
		{"table_long", []string{"table: 500 rows × 3 columns, showing ", " of 500 rows\n", "  service-000 | ok | 0\n"}, []string{"service-499"}},
		{"table_wide_cell", []string{`select * from orders where id < 10;\nselect`, "…(+520 chars)\""}, nil},
		{"logs", []string{
			"logs: 8 entries (applied_limit 20, has_more true), first 2026-01-02T11:00:00+08:00, last 2026-01-02T11:00:07+08:00\n",
			"constant fields (same in every entry):\n  level=info\n  service=checkout\n",
			"entries (varying fields):\n  2026-01-02T11:00:00+08:00 msg=\"order 1000 placed\" status=500\n",
			"  2026-01-02T11:00:07+08:00 msg=\"order 1007 placed\" status=200\n" + logsFooter,
		}, []string{"_time", "few distinct values"}},
		{"logs_fat", []string{
			"constant fields (same in every entry):\n  container=app\n  method=GET\n  namespace=shop\n  service=checkout\n",
			"entries (varying fields):\n  2026-01-02T11:00:00+08:00 message=\"GET /api/orders/5000 completed\" bytes=512 ",
			"item quantity…(+572 chars)\" …(+11 more fields)\n",
			"; projecting fewer fields in the query narrows each entry\n",
		}, []string{"few distinct values", "_time="}},
		{"logs_level", []string{
			"constant fields (same in every entry):\n  service=payments\nfields with few distinct values:\n  level: info (20), error (10), warn (10)\nentries (varying fields):\n",
			"  2026-01-02T11:09:45+08:00 msg=\"payment attempt 39\" level=error order_id=ord-70039\n" + logsFooter,
		}, []string{"msg: ", "order_id: ", "projecting"}},
		{"logs_lowcard_limits", []string{
			"fields with few distinct values, showing 8 of 10:\n  f00: a (15), b (15)\n",
		}, []string{"f08: ", "note: ", "id: "}},
		{"logs_meta_budget", []string{"constant fields (same in every entry), showing ", " of 40:\n", "entries (varying fields):\n"}, nil},
		{"logs_long", []string{"logs: 500 entries", "fields with few distinct values:\n  status: 200 (375), 500 (125)\n", "entries (varying fields), showing ", " of 500:", "; projecting fewer fields"}, nil},
		{"logs_int_ts", []string{"logs: 1 entry (applied_limit 1, has_more false), first 2026-01-02T11:00:00+08:00", "constant fields (same in every entry):\n  msg=ready\n"}, nil},
		{"empty_samples", []string{"query succeeded: no samples matched in the window\n"}, []string{"complete result"}},
		{"empty_frames", []string{"query succeeded: no series or rows matched in the window\n"}, []string{"complete result"}},
		{"empty_table", []string{"query succeeded: no series or rows matched in the window\n"}, nil},
		{"empty_logs", []string{"query succeeded: no log entries matched in the window\n"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := exploreDigest(fixtures[tc.name], "")
			if !ok {
				t.Fatal("digest declined an explore_result.v1 result")
			}
			if strings.HasPrefix(tc.name, "empty") && strings.Count(got, "\n") != 1 {
				t.Errorf("empty result should print one line, got %q", got)
			}
			if len(got) > 8192 {
				t.Errorf("digest is %d bytes, want ≤ 8192", len(got))
			}
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("missing %q in:\n%s", w, got)
				}
			}
			for _, w := range tc.notWant {
				if strings.Contains(got, w) {
					t.Errorf("unexpected %q in:\n%s", w, got)
				}
			}
			again, _ := exploreDigest(fixtures[tc.name], "")
			if again != got {
				t.Error("digest is not deterministic")
			}
		})
	}
}

func TestExploreDigestDeclines(t *testing.T) {
	for name, data := range map[string]string{
		"diagnostic tool": `{"version":"7.2","connected_clients":12}`,
		"other format":    `{"format":"explore_result.v2","result":{"kind":"samples","samples":[]}}`,
		"unknown kind":    `{"format":"explore_result.v1","result":{"kind":"heatmap"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := exploreDigest(&flashduty.DatasourceToolResult{Data: json.RawMessage(data)}, ""); ok {
				t.Fatal("digest rendered a result it should leave to the generic renderer")
			}
		})
	}
}

func TestFmtNum(t *testing.T) {
	for in, want := range map[float64]string{
		0: "0", 3: "3", -42: "-42", 1234567: "1234567", 12.3456: "12.346", 0.000123456: "0.00012346",
		1234.56: "1234.6", 1000.2: "1000.2", 1000.4: "1000.4", 123456.7: "123460", 1e20: "1e+20", 2.5e-7: "2.5e-07",
	} {
		if got := fmtNum(in); got != want {
			t.Errorf("fmtNum(%v) = %q, want %q", in, got, want)
		}
	}
}

// TestMonitQueryDigestDefault: the default table format prints the digest;
// json keeps the complete result; diagnostic tools keep the generic view.
func TestMonitQueryDigestDefault(t *testing.T) {
	pinClock(t, digestT0)
	for _, tc := range []struct {
		name, tool string
		data       string
		args       []string
		want       string
		notWant    string
	}{
		{"table digest", "prometheus.query",
			`{"format":"explore_result.v1","result":{"kind":"samples","samples":[{"labels":{"job":"api"},"value":1.25}]}}`,
			nil, "samples: 1, sorted by value descending", "datasource_id"},
		{"json complete", "prometheus.query",
			`{"format":"explore_result.v1","result":{"kind":"samples","samples":[{"labels":{"job":"api"},"value":1.25}]}}`,
			[]string{"--output-format", "json"}, `"explore_result.v1"`, "sorted by value"},
		{"legacy json flag", "prometheus.query",
			`{"format":"explore_result.v1","result":{"kind":"samples","samples":[]}}`,
			[]string{"--json"}, `"explore_result.v1"`, "query succeeded"},
		{"diagnostic generic", "redis_node.overview",
			`{"connected_clients":12}`, nil, "connected_clients", "complete result"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			stub.data = json.RawMessage(`{"datasource_id":42,"tool":"` + tc.tool + `","data":` + tc.data + `}`)
			stdout, _, err := execCommandSplit(append([]string{"monit-query", "42", "--tool", tc.tool}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout, tc.want) || strings.Contains(stdout, tc.notWant) {
				t.Fatalf("stdout want %q, not %q:\n%s", tc.want, tc.notWant, stdout)
			}
		})
	}
}

// timedFrame builds a time_series frame from explicit point times.
func timedFrame(labels map[string]string, times []time.Time, values []any) map[string]any {
	stamps := make([]any, len(times))
	for i, t := range times {
		stamps[i] = t.Format(time.RFC3339Nano)
	}
	return map[string]any{"kind": "time_series", "fields": []any{
		map[string]any{"name": "time", "type": "time", "values": stamps},
		map[string]any{"name": "value", "type": "float", "labels": labels, "values": values},
	}}
}

func evenTimes(n int, step time.Duration) []time.Time {
	out := make([]time.Time, n)
	for i := range out {
		out[i] = digestT0.Add(time.Duration(i) * step)
	}
	return out
}

func mustDigest(t *testing.T, out *flashduty.DatasourceToolResult) string {
	t.Helper()
	got, ok := exploreDigest(out, "")
	if !ok {
		t.Fatal("digest declined an explore_result.v1 result")
	}
	if len(got) > 8192 {
		t.Errorf("digest is %d bytes, want ≤ 8192", len(got))
	}
	return got
}

func framesFixture(t *testing.T, frames ...any) *flashduty.DatasourceToolResult {
	return exploreFixture(t, map[string]any{"kind": "frames", "frames": frames}, nil)
}

// A series whose points all share one timestamp has a zero span: it gets the
// stats lines but no bucketed shape.
func TestExploreDigestZeroSpanSeries(t *testing.T) {
	pinClock(t, digestT0)
	times := make([]time.Time, 13)
	for i := range times {
		times[i] = digestT0
	}
	got := mustDigest(t, framesFixture(t, timedFrame(map[string]string{"service": "checkout"}, times, rampValues(13, 1, 1, -1, 0))))
	want := "time_series: 1 series (1 frame), sorted by max descending\n" +
		"common labels: {service=\"checkout\"}\n" +
		"all series: 13 points, 2026-01-02T11:00:00+08:00 .. 2026-01-02T11:00:00+08:00\n" +
		"[1] {}\n" +
		"    max 13 at 2026-01-02T11:00:00+08:00, min 1 at 2026-01-02T11:00:00+08:00\n" +
		"    avg 7, first 1, last 13\n" +
		framesFooter
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// Long axes with many points must not overflow the bucket width: every bucket
// index stays within range and the width is the span split twelve ways.
func TestExploreDigestLongAxisNoOverflow(t *testing.T) {
	pinClock(t, digestT0)
	for _, tc := range []struct {
		name  string
		n     int
		step  time.Duration
		width string
	}{
		{"2000 daily points", 2000, 24 * time.Hour, "166d16h"},
		{"30 days at 11000 points", 11000, 30 * 24 * time.Hour / 10999, "60h0m20s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := mustDigest(t, framesFixture(t, timedFrame(nil, evenTimes(tc.n, tc.step), rampValues(tc.n, 0, 1, -1, 0))))
			shape := strings.Split(got, "\n")[5]
			if !strings.HasPrefix(shape, "    shape (bucket start → max, "+tc.width+" buckets): ") || strings.Count(shape, " | ") != 11 {
				t.Fatalf("unexpected shape line:\n%s", got)
			}
			if !strings.HasSuffix(shape, fmt.Sprintf(" %d", tc.n-1)) {
				t.Fatalf("last bucket lost the final point:\n%s", got)
			}
		})
	}
}

// Control characters and invalid UTF-8 from any part of a result are escaped
// before they reach the terminal.
func TestExploreDigestEscapesControlCharacters(t *testing.T) {
	pinClock(t, digestT0)
	const esc = "\u001b[2Jx"
	results := map[string]*flashduty.DatasourceToolResult{
		"logs": exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 2, "has_more": false, "entries": []any{
			map[string]any{"timestamp_ns": fmt.Sprint(digestT0.UnixNano()), "fields": map[string]any{"msg": esc, "k" + esc: "a"}},
			map[string]any{"timestamp_ns": fmt.Sprint(digestT0.Add(time.Second).UnixNano()), "fields": map[string]any{"msg": "ok", "k" + esc: "b"}},
		}}, nil),
		"table": framesFixture(t, map[string]any{"kind": "table", "fields": []any{
			map[string]any{"name": "col" + esc, "type": "string", "values": []any{esc}},
		}}),
		"samples": exploreFixture(t, map[string]any{"kind": "samples", "samples": []any{
			map[string]any{"labels": map[string]string{"key" + esc: "v" + esc}, "value": 1},
			map[string]any{"labels": map[string]string{"key" + esc: "w"}, "value": 2},
		}}, nil),
		"series": framesFixture(t, timedFrame(map[string]string{"pod" + esc: "a" + esc}, evenTimes(2, time.Minute), []any{1, 2})),
	}
	// JSON decoding already replaces invalid UTF-8; escapeText still quotes it
	// for any caller that hands it raw bytes.
	if got := escapeText("a\xffb"); got != `"a\xffb"` {
		t.Errorf("escapeText(invalid UTF-8) = %s", got)
	}
	truncated := exploreFixture(t, map[string]any{"kind": "samples", "samples": []any{}}, nil)
	truncated.Truncated = &flashduty.DatasourceToolTruncation{Reason: "limit" + esc}
	results["truncated reason"] = truncated
	for name, in := range results {
		t.Run(name, func(t *testing.T) {
			got := mustDigest(t, in)
			if strings.ContainsRune(got, 0x1b) || !utf8.ValidString(got) {
				t.Fatalf("raw control or invalid bytes reached the output: %q", got)
			}
			if !strings.Contains(got, `\x1b[2Jx`) {
				t.Fatalf("escaped text missing: %q", got)
			}
		})
	}
}

// A newest-to-oldest axis is read in time order: the range runs forward, the
// buckets start at the oldest point, and first/last follow time.
func TestExploreDigestDescendingAxis(t *testing.T) {
	pinClock(t, digestT0)
	asc := evenTimes(24, 5*time.Minute)
	desc := make([]time.Time, len(asc))
	values := make([]any, len(asc))
	for i := range asc {
		desc[i] = asc[len(asc)-1-i]
		values[i] = len(asc) - 1 - i // value = index in time order
	}
	got := mustDigest(t, framesFixture(t, timedFrame(map[string]string{"service": "checkout"}, desc, values)))
	want := "time_series: 1 series (1 frame), sorted by max descending\n" +
		"common labels: {service=\"checkout\"}\n" +
		"all series: 24 points, 2026-01-02T11:00:00+08:00 .. 2026-01-02T12:55:00+08:00\n" +
		"[1] {}\n" +
		"    max 23 at 2026-01-02T12:55:00+08:00, min 0 at 2026-01-02T11:00:00+08:00\n" +
		"    avg 11.5, first 0, last 23\n" +
		"    shape (bucket start → max, 10m buckets): 11:00 1 | 11:10 3 | 11:20 5 | 11:30 7 | 11:40 9 | 11:50 11 | 12:00 13 | 12:10 15 | 12:20 17 | 12:30 19 | 12:40 21 | 12:50 23\n" +
		framesFooter
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// The byte budget is shared across sections: a table that no longer fits
// after the time series is reduced to its counts.
func TestExploreDigestSharedBudget(t *testing.T) {
	pinClock(t, digestT0)
	frames := make([]any, 0, 22)
	for i := range 20 {
		frames = append(frames, timedFrame(map[string]string{"service": fmt.Sprintf("svc-%02d-%s", i, strings.Repeat("x", 60))},
			evenTimes(40, time.Minute), rampValues(40, float64(i), 1.37, -1, 0)))
	}
	cols := make([]any, 6)
	for c := range cols {
		values := make([]any, 120)
		for r := range values {
			values[r] = fmt.Sprintf("cell-%d-%d", r, c)
		}
		cols[c] = map[string]any{"name": fmt.Sprintf("c%d", c), "type": "string", "values": values}
	}
	table := map[string]any{"kind": "table", "fields": cols}
	frames = append(frames, table, table, table)
	got := mustDigest(t, framesFixture(t, frames...))
	for _, w := range []string{
		"time_series: showing 15 of 20 series",
		"table 1/3: 120 rows × 6 columns not shown\n(2 more table frames, 240 rows, not shown)\n" + framesFooter,
	} {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in:\n%s", w, got)
		}
	}
}

func TestExploreDigestNumericStringSample(t *testing.T) {
	got := mustDigest(t, exploreFixture(t, map[string]any{"kind": "samples", "samples": []any{
		map[string]any{"labels": map[string]string{"pod": "a"}, "value": "9"},
		map[string]any{"labels": map[string]string{"pod": "b"}, "value": 12.5},
	}}, nil))
	if !strings.Contains(got, "  12.5  {pod=\"b\"}\n  9     {pod=\"a\"}\n") || strings.Contains(got, "non-numeric") {
		t.Fatalf("numeric string not read as a number:\n%s", got)
	}
}

func TestExploreDigestTruncatedReasonBounded(t *testing.T) {
	in := exploreFixture(t, map[string]any{"kind": "samples", "samples": []any{}}, nil)
	in.Truncated = &flashduty.DatasourceToolTruncation{Reason: strings.Repeat("too many rows\n", 100)}
	got := mustDigest(t, in)
	first := strings.SplitN(got, "\n", 2)[0]
	if !strings.HasPrefix(first, `note: result truncated by the server: "too many rows\ntoo many rows\n`) || !strings.Contains(first, "…(+1200 chars)") {
		t.Fatalf("reason not folded and capped: %q", got)
	}
}

// Series that agree on length, first and last time but differ in between do
// not share an axis.
func TestExploreDigestSharedAxisComparesEveryPoint(t *testing.T) {
	pinClock(t, digestT0)
	a := evenTimes(3, time.Minute)
	b := []time.Time{a[0], a[0].Add(30 * time.Second), a[2]}
	got := mustDigest(t, framesFixture(t,
		timedFrame(map[string]string{"service": "a"}, a, []any{1, 2, 3}),
		timedFrame(map[string]string{"service": "b"}, b, []any{4, 5, 6})))
	if strings.Contains(got, "all series") || !strings.Contains(got, "[1] {service=\"b\"}  3 points, ") {
		t.Fatalf("differing axes reported as shared:\n%s", got)
	}
}

// A hostile result cannot push the digest past its bound: an enormous field
// name and number literals, and long label pairs shared by every series.
func TestExploreDigestHostileResultStaysBounded(t *testing.T) {
	pinClock(t, digestT0)
	huge := strings.Repeat("9", 5000)
	hugeName := strings.Repeat("n", 200000)
	entries := make([]any, 3)
	for i := range entries {
		entries[i] = map[string]any{"timestamp_ns": fmt.Sprint(digestT0.Add(time.Duration(i) * time.Second).UnixNano()),
			"fields": map[string]any{hugeName: fmt.Sprint(i), "msg": strings.Repeat("m", 5000)}}
	}
	shared := map[string]string{}
	for k := range 30 {
		shared[fmt.Sprintf("label_%02d_%s", k, strings.Repeat("k", 100))] = strings.Repeat("v", 1000)
	}
	frames := make([]any, 30)
	for i := range frames {
		labels := map[string]string{"service": fmt.Sprintf("svc-%02d", i)}
		for k, v := range shared {
			labels[k] = v
		}
		frames[i] = timedFrame(labels, evenTimes(40, time.Minute), rampValues(40, float64(i), 1, -1, 0))
	}
	tinyRows := func(typ string, v any) *flashduty.DatasourceToolResult {
		values := make([]any, 5000)
		for i := range values {
			values[i] = v
		}
		return framesFixture(t, map[string]any{"kind": "table", "fields": []any{map[string]any{"name": "c", "type": typ, "values": values}}})
	}
	for name, in := range map[string]*flashduty.DatasourceToolResult{
		"tiny string rows": tinyRows("string", "1"),
		"tiny int rows":    tinyRows("number", 1),
		"logs":             exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": json.Number(huge), "has_more": true, "entries": entries}, nil),
		"series": exploreFixture(t, map[string]any{"kind": "frames", "frames": frames},
			map[string]any{"execution": map[string]any{"kind": "range", "effective_step_seconds": json.Number(huge)}}),
		"samples": exploreFixture(t, map[string]any{"kind": "samples", "samples": func() []any {
			out := make([]any, 30)
			for i := range out {
				labels := map[string]string{"pod": fmt.Sprint(i)}
				for k, v := range shared {
					labels[k] = v
				}
				out[i] = map[string]any{"labels": labels, "value": i}
			}
			return out
		}()}, nil),
	} {
		t.Run(name, func(t *testing.T) {
			got := mustDigest(t, in)
			if strings.Contains(got, strings.Repeat("9", 100)) || strings.Contains(got, strings.Repeat("n", 300)) {
				t.Fatalf("uncapped text in digest:\n%.2000s", got)
			}
		})
	}
}

// Header lines are fitted against their real length: the body never pushes
// the digest past the budget plus one forced item.
func TestExploreDigestHeaderCountsTowardBudget(t *testing.T) {
	pinClock(t, digestT0)
	shared := map[string]string{"description": strings.Repeat("d", 1400)}
	samples := make([]any, 200)
	for i := range samples {
		samples[i] = map[string]any{"labels": map[string]string{"description": shared["description"], "pod": fmt.Sprintf("orders-%03d-%s", i, strings.Repeat("p", 250))}, "value": i}
	}
	got := mustDigest(t, exploreFixture(t, map[string]any{"kind": "samples", "samples": samples}, nil))
	body := strings.TrimSuffix(got, samplesFooter)
	if len(body) > digestBudget {
		t.Fatalf("body is %d bytes, budget %d", len(body), digestBudget)
	}
}

// A time_series frame whose only fields are its time axis and non-numeric
// columns is declined, not reported as empty; string fields are not series.
func TestExploreDigestTimeSeriesWithoutValues(t *testing.T) {
	stamps := []any{"2026-01-02T03:00:00Z", "2026-01-02T03:01:00Z"}
	onlyStrings := framesFixture(t, map[string]any{"kind": "time_series", "fields": []any{
		map[string]any{"name": "time", "type": "time", "values": stamps},
		map[string]any{"name": "host", "type": "string", "values": []any{"a", "b"}},
	}})
	if got, ok := exploreDigest(onlyStrings, ""); ok {
		t.Fatalf("time axis without values rendered as a digest:\n%s", got)
	}
	mixed := mustDigest(t, framesFixture(t, map[string]any{"kind": "time_series", "fields": []any{
		map[string]any{"name": "time", "type": "time", "values": stamps},
		map[string]any{"name": "host", "type": "string", "values": []any{"a", "b"}},
		map[string]any{"name": "ok", "type": "bool", "values": []any{true, false}},
		map[string]any{"name": "value", "type": "float", "labels": map[string]string{"job": "api"}, "values": []any{1, 2}},
	}}))
	if !strings.Contains(mixed, "time_series: 1 series (1 frame)") || strings.Contains(mixed, "host") || strings.Contains(mixed, "non-finite") {
		t.Fatalf("non-numeric fields rendered as series:\n%s", mixed)
	}
}

func TestExploreDigestSampleWidthCountsRunes(t *testing.T) {
	got := mustDigest(t, exploreFixture(t, map[string]any{"kind": "samples", "samples": []any{
		map[string]any{"labels": map[string]string{"pod": "a"}, "value": 1},
		map[string]any{"labels": map[string]string{"pod": "b"}, "value": "écrit"},
	}}, nil))
	if !strings.Contains(got, "  1      {pod=\"a\"}\n  écrit  {pod=\"b\"}\n") {
		t.Fatalf("value column misaligned:\n%s", got)
	}
}

func TestEscapeTextFormatCharacters(t *testing.T) {
	if got := escapeText("abc\u202edef"); got != `"abc\u202edef"` {
		t.Fatalf("bidi override not escaped: %s", got)
	}
	if got := escapeText("plain café"); got != "plain café" {
		t.Fatalf("printable text altered: %s", got)
	}
}

func equalSamples(n int) *flashduty.DatasourceToolResult {
	samples := make([]any, n)
	for i := range samples {
		// Reverse insertion order so the output order comes from the sort.
		samples[i] = map[string]any{"labels": map[string]string{"__name__": fmt.Sprintf("metric_%04d", n-1-i)}, "value": 1}
	}
	raw, _ := json.Marshal(map[string]any{"format": "explore_result.v1", "result": map[string]any{"kind": "samples", "samples": samples}})
	return &flashduty.DatasourceToolResult{Data: raw}
}

// Samples are limited by the byte budget only; equal values order by label.
func TestExploreDigestSamplesFitBySize(t *testing.T) {
	got := mustDigest(t, equalSamples(120))
	lines := strings.Split(got, "\n")
	if lines[0] != "samples: 120, sorted by value descending" {
		t.Fatalf("header %q", lines[0])
	}
	for i := range 120 {
		if want := fmt.Sprintf(`  1  {__name__="metric_%04d"}`, i); lines[1+i] != want {
			t.Fatalf("line %d = %q, want %q", 1+i, lines[1+i], want)
		}
	}

	got = mustDigest(t, equalSamples(2000))
	if !strings.HasPrefix(got, "samples: showing ") || !strings.Contains(got, " of 2000, sorted by value descending\nall 2000 finite values: min 1, median 1, max 1\n") ||
		!strings.Contains(got, "  1  {__name__=\"metric_0000\"}\n") {
		t.Fatalf("cut not stated:\n%.600s", got)
	}
}

func messageEntries(msg string) *flashduty.DatasourceToolResult {
	entries := make([]any, 2)
	for i := range entries {
		entries[i] = map[string]any{"timestamp_ns": fmt.Sprint(digestT0.Add(time.Duration(i) * time.Second).UnixNano()),
			"fields": map[string]any{"message": fmt.Sprintf("%d %s", i, msg), "pod": fmt.Sprintf("orders-%d", i), "trace_id": fmt.Sprintf("trace-%d", i)}}
	}
	raw, _ := json.Marshal(map[string]any{"format": "explore_result.v1", "result": map[string]any{"kind": "logs", "applied_limit": 2, "has_more": false, "entries": entries}})
	return &flashduty.DatasourceToolResult{Data: raw}
}

// The message field is limited only by the entry line; other fields take the
// room that is left.
func TestExploreDigestLogMessageFirstClaim(t *testing.T) {
	pinClock(t, digestT0)
	prefix := "[ERROR][2026-01-02 03:00:00.000][order_service.go:217] "
	msg500 := prefix + strings.Repeat("x", 500-len(prefix))
	got := mustDigest(t, messageEntries(msg500))
	if !strings.Contains(got, `  2026-01-02T11:00:00+08:00 message="0 `+msg500+`" pod=orders-0 trace_id=trace-0`+"\n") {
		t.Fatalf("500-char message not printed whole:\n%s", got)
	}

	got = mustDigest(t, messageEntries(prefix+strings.Repeat("y", 5000)))
	for _, line := range strings.Split(got, "\n") {
		if !strings.HasPrefix(line, "  2026-") {
			continue
		}
		if len(line) > digestLogLineCap || !strings.Contains(line, `chars)" …(+2 more fields)`) || !strings.Contains(line, prefix) {
			t.Fatalf("long message not cut at the line cap with markers (%d bytes): %s", len(line), line)
		}
	}
	if !strings.Contains(got, "; projecting fewer fields in the query narrows each entry\n") {
		t.Fatalf("footer misses the projection hint:\n%s", got)
	}
}

// A multi-byte message fills the room of its line as an ASCII one does, and
// a long one keeps a prefix before the marker.
func TestExploreDigestLogMessageMultiByte(t *testing.T) {
	pinClock(t, digestT0)
	for _, tc := range []struct {
		runes, atLeast int
		cut            bool
	}{{300, 150, true}, {6000, 150, true}, {100, 100, false}} {
		got := mustDigest(t, messageEntries(strings.Repeat("日", tc.runes)))
		if !utf8.ValidString(got) {
			t.Fatalf("%d runes: digest is not valid UTF-8", tc.runes)
		}
		for _, line := range strings.Split(got, "\n") {
			if !strings.HasPrefix(line, "  2026-") {
				continue
			}
			if kept := strings.Count(line, "日"); kept < tc.atLeast || len(line) > digestLogLineCap {
				t.Fatalf("%d runes: kept %d characters in %d bytes: %s", tc.runes, kept, len(line), line)
			}
			if strings.Contains(line, " chars)") != tc.cut {
				t.Fatalf("%d runes: cut marker presence wrong: %s", tc.runes, line)
			}
		}
	}
}

// The message field is the conventional name present in the most entries;
// a name earlier in the convention wins only on a tie.
func TestExploreDigestMessageFieldMostPresent(t *testing.T) {
	pinClock(t, digestT0)
	entries := make([]any, 6)
	for i := range entries {
		fields := map[string]any{"message": fmt.Sprintf("request %d done", i), "pod": fmt.Sprintf("orders-%d", i)}
		if i == 0 {
			fields["_msg"] = "start"
		}
		entries[i] = map[string]any{"timestamp_ns": fmt.Sprint(digestT0.Add(time.Duration(i) * time.Second).UnixNano()), "fields": fields}
	}
	got := mustDigest(t, exploreFixture(t, map[string]any{"kind": "logs", "applied_limit": 6, "has_more": false, "entries": entries}, nil))
	for i := 1; i < 6; i++ {
		want := fmt.Sprintf(` message="request %d done" pod=orders-%d`+"\n", i, i)
		if !strings.Contains(got, want) {
			t.Fatalf("entry %d does not lead with message:\n%s", i, got)
		}
	}
	if !strings.Contains(got, `+08:00 message="request 0 done" _msg=start pod=orders-0`+"\n") {
		t.Fatalf("entry 0 does not lead with message:\n%s", got)
	}
}

// A result summary is one line after the window line, counted toward the
// bound; an empty summary prints nothing.
func TestExploreDigestSummaryLine(t *testing.T) {
	window := "window 2026-01-02T09:00:00+08:00..2026-01-02T11:00:00+08:00 (2h, ended now)"
	in := equalSamples(2000)
	summary := "2000 samples\nfrom " + strings.Repeat("s", 5000)
	in.Summary = &summary
	got, ok := exploreDigest(in, window)
	lines := strings.SplitN(got, "\n", 3)
	if !ok || len(lines) < 3 || lines[0] != window || !strings.HasPrefix(lines[1], `summary: "2000 samples\nfrom sss`) ||
		!strings.HasSuffix(lines[1], `chars)"`) || !strings.HasPrefix(lines[2], "samples: showing ") || len(got) > 8192 {
		t.Fatalf("summary line missing, unescaped or unbounded (%d bytes):\n%.600s", len(got), got)
	}
	for _, empty := range []*string{nil, new(string)} {
		in.Summary = empty
		got, _ = exploreDigest(in, window)
		if strings.Contains(got, "summary: ") {
			t.Fatalf("empty summary printed:\n%.300s", got)
		}
	}
}

// The window line leads the digest and counts toward its bound.
func TestExploreDigestWindowLineLeads(t *testing.T) {
	window := "window 2026-01-02T09:00:00+08:00..2026-01-02T11:00:00+08:00 (2h, ended now)"
	got, ok := exploreDigest(equalSamples(2000), window)
	if !ok || !strings.HasPrefix(got, window+"\nsamples: showing ") || len(got) > 8192 {
		t.Fatalf("window line missing or bound exceeded (%d bytes):\n%.300s", len(got), got)
	}
	empty, _ := exploreDigest(equalSamples(0), window)
	if empty != window+"\nquery succeeded: no samples matched in the window\n" {
		t.Fatalf("empty result: %q", empty)
	}
}

// Each footer's path resolves against the --output-format json output of a
// result of its kind, so the advertised projection matches the real shape.
func TestExploreDigestFooterPathsResolve(t *testing.T) {
	pinClock(t, digestT0)
	fixtures := digestFixtures(t)
	for _, tc := range []struct {
		fixture, footer string
	}{
		{"samples", samplesFooter},
		{"time_series", framesFooter},
		{"table", tablesFooter},
		{"logs", logsFooter},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			digest, _ := exploreDigest(fixtures[tc.fixture], "")
			if !strings.HasSuffix(digest, tc.footer) {
				t.Fatalf("digest does not end with the %s footer:\n%s", tc.fixture, digest)
			}
			saveAndResetGlobals(t)
			stub := newGFStub(t)
			result := *fixtures[tc.fixture]
			result.DatasourceID, result.Tool = 42, "prometheus.query"
			stub.data = &result
			stdout, _, err := execCommandSplit("monit-query", "42", "--tool", "prometheus.query", "--output-format", "json")
			if err != nil {
				t.Fatal(err)
			}
			var doc any
			if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
				t.Fatalf("json output: %v", err)
			}
			path := tc.footer[strings.Index(tc.footer, "(")+1 : strings.LastIndex(tc.footer, ")")]
			resolveFooterPath(t, doc, path)
		})
	}
}

// resolveFooterPath checks a footer path "<.a.b[].c[]> | .x, .y, .f[\"<name>\"]":
// every array on the way is non-empty, and every projected key is present on
// at least one element (an object key for the bracket form).
func resolveFooterPath(t *testing.T, doc any, path string) {
	t.Helper()
	walk, projections, ok := strings.Cut(path, " | ")
	if !ok {
		t.Fatalf("path %q has no projection", path)
	}
	nodes := []any{doc}
	for _, step := range strings.Split(strings.TrimPrefix(walk, "."), ".") {
		key, each := strings.CutSuffix(step, "[]")
		var next []any
		for _, n := range nodes {
			obj, ok := n.(map[string]any)
			if !ok {
				t.Fatalf("%q: %v is not an object", step, n)
			}
			v, present := obj[key]
			if !present {
				t.Fatalf("%q: key %q missing in %v", path, key, obj)
			}
			if !each {
				next = append(next, v)
				continue
			}
			arr, ok := v.([]any)
			if !ok || len(arr) == 0 {
				t.Fatalf("%q: %q is not a non-empty array", path, key)
			}
			next = append(next, arr...)
		}
		nodes = next
	}
	for _, proj := range strings.Split(projections, ", ") {
		key, bracket, _ := strings.Cut(strings.TrimPrefix(proj, "."), "[")
		found := false
		for _, n := range nodes {
			v, present := n.(map[string]any)[key]
			if !present {
				continue
			}
			if bracket != "" {
				if _, isObj := v.(map[string]any); !isObj {
					t.Fatalf("%q: %q is not an object", path, key)
				}
			}
			found = true
		}
		if !found {
			t.Fatalf("%q: projection %q matches no element", path, proj)
		}
	}
}
