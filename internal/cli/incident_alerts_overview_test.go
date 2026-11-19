package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
)

const overviewIncidentID = "64f0c0ffee0123456789abcd"

// overviewFixture is a small incident: five alerts, three titles.
func overviewFixture() []map[string]any {
	at := func(hhmm string) int64 {
		t, _ := time.Parse(time.RFC3339, "2026-01-02T"+hhmm+":00Z")
		return t.Unix()
	}
	alert := func(id, title, severity, status, start string, labels map[string]string) map[string]any {
		return map[string]any{
			"alert_id": id, "title": title, "alert_severity": severity, "alert_status": status,
			"start_time": at(start), "labels": labels,
		}
	}
	return []map[string]any{
		alert("a00000000000000000000001", "High error rate", "Critical", "Critical", "06:00", map[string]string{"cluster": "prod-a", "env": "prod", "host": "web-1"}),
		alert("a00000000000000000000002", "High error rate", "Warning", "Ok", "06:05", map[string]string{"cluster": "prod-a", "env": "prod", "host": "web-2"}),
		alert("a00000000000000000000003", "High error rate", "Critical", "Critical", "06:10", map[string]string{"cluster": "prod-a", "env": "prod", "host": "web-1"}),
		alert("a00000000000000000000004", "Disk almost full", "Warning", "Warning", "06:02", map[string]string{"cluster": "prod-a", "env": "prod", "host": "db-1", "mount": "/data"}),
		alert("a00000000000000000000005", "Queue lag", "Info", "Ok", "06:20", map[string]string{"cluster": "prod-a", "env": "prod", "queue": "orders"}),
	}
}

// serveAlertPages answers /incident/alert/list from all by the request's
// p/limit and returns the recorded request bodies.
func serveAlertPages(stub *gfStub, all []map[string]any, total int) *[]map[string]any {
	var mu sync.Mutex
	var bodies []map[string]any
	stub.dataForPath = func(path string, body map[string]any) any {
		if path != "/incident/alert/list" {
			return nil
		}
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		page, limit := 1, 20
		if v, ok := body["p"].(float64); ok {
			page = int(v)
		}
		if v, ok := body["limit"].(float64); ok {
			limit = int(v)
		}
		lo := min((page-1)*limit, len(all))
		hi := min(lo+limit, len(all))
		return map[string]any{"items": all[lo:hi], "total": total}
	}
	return &bodies
}

// structuredAlertsOutput runs `incident alerts` against overviewFixture in
// json and toon, with default and explicit paging, and concatenates each
// run's arguments, request bodies, stdout and stderr.
func structuredAlertsOutput(t *testing.T) string {
	t.Helper()
	pinClock(t, time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC))
	var b strings.Builder
	for _, args := range [][]string{
		{"--output-format", "json"},
		{"--output-format", "toon"},
		{"--output-format", "json", "--limit", "2", "--page", "2"},
		{"--output-format", "toon", "--limit", "2", "--page", "2"},
	} {
		saveAndResetGlobals(t)
		stub := newGFStub(t)
		bodies := serveAlertPages(stub, overviewFixture(), 5)
		stdout, stderr, err := execCommandSplit(append([]string{"incident", "alerts", overviewIncidentID}, args...)...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		b.WriteString("== " + strings.Join(args, " ") + "\n")
		for _, body := range *bodies {
			b.WriteString("-- request " + fmt.Sprint(body) + "\n")
		}
		b.WriteString("-- stdout\n" + stdout + "-- stderr\n" + stderr)
	}
	return b.String()
}

// TestIncidentAlertsStructuredUnchanged: json and toon print the same bytes,
// send the same requests and write the same stderr as before the default
// output became an overview. The golden was produced by the previous code.
func TestIncidentAlertsStructuredUnchanged(t *testing.T) {
	want, err := os.ReadFile("testdata/incident_alerts_structured.golden")
	if err != nil {
		t.Fatal(err)
	}
	if got := structuredAlertsOutput(t); got != string(want) {
		t.Errorf("structured output changed:\n%s", got)
	}
}

func TestIncidentAlertsOverviewText(t *testing.T) {
	saveAndResetGlobals(t)
	pinClock(t, time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC))
	stub := newGFStub(t)
	bodies := serveAlertPages(stub, overviewFixture(), 5)
	out, err := execCommand("incident", "alerts", overviewIncidentID)
	if err != nil {
		t.Fatal(err)
	}
	want := `alerts: 5, 3 active, 2 recovered; severity Critical 2, Warning 2, Info 1
start times: 2026-01-02T14:00:00+08:00 .. 2026-01-02T14:20:00+08:00
labels on every alert: {cluster="prod-a", env="prod"}
groups by title: 3, by alert count descending
[1] High error rate
    3 alerts, 2 active, highest Critical, started 2026-01-02T14:00:00+08:00 .. 2026-01-02T14:10:00+08:00
    labels: host: web-1 (2), web-2 (1)
[2] Disk almost full
    1 alert, 1 active, highest Warning, started 2026-01-02T14:02:00+08:00
    labels: host="db-1"; mount="/data"
[3] Queue lag
    1 alert, 0 active, highest Info, started 2026-01-02T14:20:00+08:00
    labels: queue="orders"
complete rows: flashduty incident alerts 64f0c0ffee0123456789abcd --output-format json --limit 100 --page 1 prints each page as a JSON array; active or recovered only: flashduty incident alert-list 64f0c0ffee0123456789abcd --is-active (or --is-active=false) --output-format json
`
	if out != want {
		t.Errorf("overview:\n%s\nwant:\n%s", out, want)
	}
	if len(*bodies) != 1 || (*bodies)[0]["limit"] != float64(100) || (*bodies)[0]["p"] != float64(1) {
		t.Errorf("requests = %v, want one page of 100", *bodies)
	}
}

// syntheticAlerts builds n alerts over a few titles and hosts.
func syntheticAlerts(n int) []map[string]any {
	titles := []string{"High error rate", "Disk almost full", "Queue lag", "Pod restarting"}
	out := make([]map[string]any, n)
	base := time.Date(2026, 1, 2, 6, 0, 0, 0, time.UTC).Unix()
	for i := range out {
		status := "Warning"
		if i%3 == 0 {
			status = "Ok"
		}
		out[i] = map[string]any{
			"alert_id":       fmt.Sprintf("b%023d", i),
			"title":          titles[i%len(titles)],
			"alert_severity": "Warning",
			"alert_status":   status,
			"start_time":     base + int64(i),
			"labels":         map[string]string{"cluster": "prod-a", "host": fmt.Sprintf("node-%d", i%3)},
		}
	}
	return out
}

// TestIncidentAlertsOverviewPaging: an incident with more alerts than the
// overview reads is fetched in 5 pages of 100, and the overview says which
// alerts it covers.
func TestIncidentAlertsOverviewPaging(t *testing.T) {
	saveAndResetGlobals(t)
	pinClock(t, time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC))
	stub := newGFStub(t)
	bodies := serveAlertPages(stub, syntheticAlerts(1234), 1234)
	out, err := execCommand("incident", "alerts", overviewIncidentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(*bodies) != 5 {
		t.Fatalf("requests = %d, want 5", len(*bodies))
	}
	for i, b := range *bodies {
		if b["p"] != float64(i+1) || b["limit"] != float64(100) {
			t.Errorf("request %d = %v, want p=%d limit=100", i, b, i+1)
		}
	}
	for _, want := range []string{
		"alerts: 1234 total; everything below covers only alerts 1-500 in API order (the first 5 pages of 100)\n",
		"covered: 500, 333 active, 167 recovered; severity Warning 500\n",
		"labels with few distinct values:\n  host: node-0 (167), node-1 (167), node-2 (166)\n",
		"--page N (N = 1..13)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("overview lacks %q:\n%s", want, out)
		}
	}
}

// TestIncidentAlertsOverviewExplicitPage: --page/--limit in the default
// format make the overview cover exactly that page.
func TestIncidentAlertsOverviewExplicitPage(t *testing.T) {
	saveAndResetGlobals(t)
	pinClock(t, time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC))
	stub := newGFStub(t)
	bodies := serveAlertPages(stub, overviewFixture(), 5)
	out, err := execCommand("incident", "alerts", overviewIncidentID, "--page", "2", "--limit", "2")
	if err != nil {
		t.Fatal(err)
	}
	if len(*bodies) != 1 || (*bodies)[0]["p"] != float64(2) || (*bodies)[0]["limit"] != float64(2) {
		t.Fatalf("requests = %v, want one request p=2 limit=2", *bodies)
	}
	if want := "alerts: 5 total; everything below covers only alerts 3-4 in API order (--page 2 --limit 2)\ncovered: 2, 2 active, 0 recovered; severity Critical 1, Warning 1\n"; !strings.HasPrefix(out, want) {
		t.Errorf("overview:\n%s\nwant prefix:\n%s", out, want)
	}
}

func TestIncidentAlertsOverviewNoAlerts(t *testing.T) {
	saveAndResetGlobals(t)
	stub := newGFStub(t)
	serveAlertPages(stub, nil, 0)
	out, err := execCommand("incident", "alerts", overviewIncidentID)
	if err != nil || out != "No alerts.\n" {
		t.Errorf("out = %q, err = %v", out, err)
	}
}

// TestAlertOverviewHostileBound: the overview stays within 8 KB of valid
// UTF-8 however many alerts, titles, labels and bytes the input carries.
func TestAlertOverviewHostileBound(t *testing.T) {
	pinClock(t, time.Date(2026, 1, 2, 7, 0, 0, 0, time.UTC))
	long := func(unit string, n int) string { return strings.Repeat(unit, n) }
	base := flashduty.Timestamp(time.Date(2026, 1, 2, 6, 0, 0, 0, time.UTC).Unix())
	cases := map[string]func(i int) flashduty.AlertInfo{
		"distinct long multi-byte titles and long labels": func(i int) flashduty.AlertInfo {
			labels := map[string]string{}
			for k := range 30 {
				labels[fmt.Sprintf("label_%02d_%s", k, long("键", 50))] = fmt.Sprintf("%d-%s", i, long("值", 400))
			}
			return flashduty.AlertInfo{Title: fmt.Sprintf("%d %s", i, long("告警标题", 600)), AlertSeverity: "Critical", AlertStatus: "Critical", StartTime: base + flashduty.Timestamp(i), Labels: labels}
		},
		"few groups with long shared and varying labels": func(i int) flashduty.AlertInfo {
			labels := map[string]string{"shared_" + long("k", 300): long("v", 3000), "low": fmt.Sprint(i % 4)}
			for k := range 40 {
				labels[fmt.Sprintf("varying_%02d", k)] = fmt.Sprintf("%s-%d", long("é", 300), i%(k+2))
			}
			return flashduty.AlertInfo{Title: fmt.Sprintf("High error rate %d", i%3), AlertSeverity: "Warning", AlertStatus: "Ok", StartTime: base, Labels: labels}
		},
		"control characters, invalid UTF-8 and unknown severities": func(i int) flashduty.AlertInfo {
			return flashduty.AlertInfo{
				Title:         fmt.Sprintf("\x1b[2J\xff%d %s", i, long("\n\t", 500)),
				AlertSeverity: fmt.Sprintf("sev-%d-%s", i, long("\x00", 100)),
				AlertStatus:   "Critical",
				StartTime:     base - flashduty.Timestamp(i),
				Labels:        map[string]string{"\x1b]0;x\a" + fmt.Sprint(i%7): "\xfe" + long("\r", 200)},
			}
		},
	}
	for name, mk := range cases {
		alerts := make([]flashduty.AlertInfo, 5000)
		for i := range alerts {
			alerts[i] = mk(i)
		}
		out := alertOverview(alerts, alertScope{incidentID: overviewIncidentID, total: 123456, first: 1})
		t.Logf("%s: %d bytes", name, len(out))
		if len(out) > 8192 {
			t.Errorf("%s: %d bytes, want ≤ 8192", name, len(out))
		}
		if !utf8.ValidString(out) {
			t.Errorf("%s: invalid UTF-8", name)
		}
		if strings.ContainsAny(out, "\x1b\x00\r\a") {
			t.Errorf("%s: raw control character in output", name)
		}
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		if !strings.HasPrefix(lines[len(lines)-1], "complete rows: ") {
			t.Errorf("%s: last line = %q", name, lines[len(lines)-1])
		}
		if !strings.Contains(out, "everything below covers only alerts 1-5000") {
			t.Errorf("%s: coverage line missing", name)
		}
	}
}
