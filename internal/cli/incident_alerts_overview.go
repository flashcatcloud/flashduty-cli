package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flashcatcloud/go-flashduty"
)

// The default `incident alerts` output fetches up to alertOverviewPages pages
// of alertOverviewPageSize alerts and prints them as an overview bounded like
// the query digest (see digestBudget).
const (
	alertOverviewPageSize = 100
	alertOverviewPages    = 5
)

// alertSeverities orders severities from most to least severe; any other
// value ranks after them.
var alertSeverities = []string{"Critical", "Warning", "Info", "Ok"}

func severityRank(s string) int {
	for i, v := range alertSeverities {
		if s == v {
			return i
		}
	}
	return len(alertSeverities)
}

// fetchAlertOverviewPages reads pages of an incident's alerts until total
// alerts are fetched, a page comes back short, or alertOverviewPages pages
// were read. It returns the alerts and the server's total.
func fetchAlertOverviewPages(ctx *RunContext, incidentID string) ([]flashduty.AlertInfo, int, error) {
	var items []flashduty.AlertInfo
	total := 0
	for page := 1; page <= alertOverviewPages; page++ {
		req := &flashduty.ListIncidentAlertsRequest{IncidentID: incidentID}
		req.Page = page
		req.Limit = alertOverviewPageSize
		result, _, err := ctx.Client.Incidents.AlertList(cmdContext(ctx.Cmd), req)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, result.Items...)
		total = int(result.Total)
		if len(result.Items) < alertOverviewPageSize || len(items) >= total {
			break
		}
	}
	return items, max(total, len(items)), nil
}

// alertScope says which slice of an incident's alerts an overview covers.
type alertScope struct {
	incidentID string
	total      int // the server's total for the incident
	first      int // 1-based position of the first fetched alert in API order
	page       int // the explicit --page, 0 when the default pages were read
	limit      int // the explicit --limit, with page
}

type alertGroup struct {
	title   string
	alerts  []flashduty.AlertInfo
	active  int
	highest string
	first   time.Time
	last    time.Time
}

// alertOverview renders alerts as a bounded plain-text overview: counts,
// start-time span, shared and low-cardinality labels, then groups by title
// while they fit the digest budget, and how to get complete rows.
func alertOverview(alerts []flashduty.AlertInfo, scope alertScope) string {
	d := &digest{}
	n := len(alerts)
	complete := scope.page == 0 && n >= scope.total
	if complete {
		d.line("alerts: %d, %s", n, alertCounts(alerts))
	} else {
		how := fmt.Sprintf("the first %s of %d", plural((n+alertOverviewPageSize-1)/alertOverviewPageSize, "page"), alertOverviewPageSize)
		if scope.page > 0 {
			how = fmt.Sprintf("--page %d --limit %d", scope.page, scope.limit)
		}
		d.line("alerts: %d total; everything below covers only alerts %d-%d in API order (%s)",
			scope.total, scope.first, scope.first+n-1, how)
		d.line("covered: %d, %s", n, alertCounts(alerts))
	}

	var first, last time.Time
	for _, a := range alerts {
		if a.StartTime.IsZero() {
			continue
		}
		t := a.StartTime.Time()
		if first.IsZero() || t.Before(first) {
			first = t
		}
		if t.After(last) {
			last = t
		}
	}
	if !first.IsZero() {
		d.line("start times: %s .. %s", formatLocal(first), formatLocal(last))
	}

	sets := make([]map[string]string, n)
	for i, a := range alerts {
		sets[i] = a.Labels
	}
	common := commonLabels(sets)
	if len(common) > 0 {
		d.line("%s", capLine("labels on every alert: "+labelString(common)))
	}
	var lowCard []string
	for _, f := range labelFields(alerts, common) {
		if lowCardinality(f, n) {
			lowCard = append(lowCard, capLine("  "+fieldName(f.name)+": "+valueCounts(f, n)))
		}
	}
	meta := digestLogMetaBudget
	d.fieldGroup("labels with few distinct values", lowCard[:min(len(lowCard), digestLowCardMaxFields)], len(lowCard), &meta)

	groups := groupAlertsByTitle(alerts)
	blocks := make([]string, len(groups))
	for i, g := range groups {
		blocks[i] = alertGroupBlock(i+1, g, common)
	}
	footer := alertOverviewFooter(scope.incidentID, scope.total)
	head := fmt.Sprintf("groups by title: %d, by alert count descending", len(groups))
	notShown := fmt.Sprintf("(%s with %s not shown)", plural(len(groups), "more group"), plural(n, "alert"))
	shown := d.fit(blocks, linesLen([]string{head, notShown})+len(footer))
	if shown < len(groups) {
		head = fmt.Sprintf("groups by title: showing %d of %d, by alert count descending", shown, len(groups))
	}
	d.line("%s", head)
	d.items(blocks[:shown])
	if rest := groups[shown:]; len(rest) > 0 {
		hidden := 0
		for _, g := range rest {
			hidden += len(g.alerts)
		}
		d.line("(%s with %s not shown)", plural(len(rest), "more group"), plural(hidden, "alert"))
	}
	d.footer(footer)
	return d.b.String()
}

// alertCounts renders active/recovered and per-severity counts; severities
// outside the documented set are counted together as "other".
func alertCounts(alerts []flashduty.AlertInfo) string {
	active := 0
	bySeverity := make([]int, len(alertSeverities)+1)
	for _, a := range alerts {
		if a.AlertStatus != "Ok" {
			active++
		}
		bySeverity[severityRank(a.AlertSeverity)]++
	}
	var parts []string
	for i, c := range bySeverity {
		if c == 0 {
			continue
		}
		name := "other"
		if i < len(alertSeverities) {
			name = alertSeverities[i]
		}
		parts = append(parts, fmt.Sprintf("%s %d", name, c))
	}
	return fmt.Sprintf("%d active, %d recovered; severity %s", active, len(alerts)-active, strings.Join(parts, ", "))
}

// labelFields tallies the values of every label not in skip, sorted by name.
func labelFields(alerts []flashduty.AlertInfo, skip map[string]string) []*logField {
	byName := map[string]*logField{}
	for _, a := range alerts {
		for k, v := range a.Labels {
			if _, ok := skip[k]; ok {
				continue
			}
			f := byName[k]
			if f == nil {
				f = &logField{name: k, counts: map[string]int{}}
				byName[k] = f
			}
			f.present++
			f.counts[v]++
		}
	}
	fields := make([]*logField, 0, len(byName))
	for _, f := range byName {
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].name < fields[j].name })
	return fields
}

// groupAlertsByTitle groups alerts by exact title, largest group first, ties
// by title.
func groupAlertsByTitle(alerts []flashduty.AlertInfo) []*alertGroup {
	byTitle := map[string]*alertGroup{}
	var groups []*alertGroup
	for _, a := range alerts {
		g := byTitle[a.Title]
		if g == nil {
			g = &alertGroup{title: a.Title, highest: a.AlertSeverity}
			byTitle[a.Title] = g
			groups = append(groups, g)
		}
		g.alerts = append(g.alerts, a)
		if a.AlertStatus != "Ok" {
			g.active++
		}
		if severityRank(a.AlertSeverity) < severityRank(g.highest) {
			g.highest = a.AlertSeverity
		}
		if !a.StartTime.IsZero() {
			t := a.StartTime.Time()
			if g.first.IsZero() || t.Before(g.first) {
				g.first = t
			}
			if t.After(g.last) {
				g.last = t
			}
		}
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if len(groups[i].alerts) != len(groups[j].alerts) {
			return len(groups[i].alerts) > len(groups[j].alerts)
		}
		return groups[i].title < groups[j].title
	})
	return groups
}

// alertGroupBlock renders one group: its title, its counts and start-time
// span, and its labels beyond those every alert carries — the ones constant
// in the group as k="v", the varying ones as value counts when few, else as
// a distinct-value count.
func alertGroupBlock(n int, g *alertGroup, common map[string]string) string {
	var b strings.Builder
	b.WriteString(capLine(fmt.Sprintf("[%d] %s", n, escapeText(capText(g.title, digestCellCap)))))
	stats := fmt.Sprintf("\n    %s, %d active, highest %s", plural(len(g.alerts), "alert"), g.active,
		escapeText(capText(g.highest, digestLowCardValueCap)))
	switch {
	case g.first.IsZero():
	case g.first.Equal(g.last):
		stats += ", started " + formatLocal(g.first)
	default:
		stats += fmt.Sprintf(", started %s .. %s", formatLocal(g.first), formatLocal(g.last))
	}
	b.WriteString(stats)

	var constant, varying []string
	for _, f := range labelFields(g.alerts, common) {
		switch {
		case f.present == len(g.alerts) && len(f.counts) == 1:
			for v := range f.counts {
				constant = append(constant, fieldName(f.name)+"="+strconv.Quote(capText(v, digestFieldCap)))
			}
		case len(f.counts) <= digestLowCard && shortValues(f):
			varying = append(varying, fieldName(f.name)+": "+valueCounts(f, len(g.alerts)))
		default:
			varying = append(varying, fieldName(f.name)+": "+plural(len(f.counts), "distinct value"))
		}
	}
	parts := make([]string, 0, len(constant)+len(varying))
	parts = append(parts, constant...)
	parts = append(parts, varying...)
	if len(parts) == 0 {
		return b.String()
	}
	// The labels line keeps to digestLogLineCap, stating how many labels it
	// left out.
	const markerRoom = 24 // "; …(+NN more labels)"
	line := "    labels:"
	omitted := 0
	for i, p := range parts {
		sep := " "
		if i > 0 {
			sep = "; "
		}
		if omitted > 0 || len(line)+len(sep)+len(p) > digestLogLineCap-markerRoom {
			omitted++
			continue
		}
		line += sep + p
	}
	if omitted > 0 {
		line += "; …(+" + plural(omitted, "more label") + ")"
	}
	b.WriteString("\n" + line)
	return b.String()
}

func shortValues(f *logField) bool {
	for v := range f.counts {
		if utf8.RuneCountInString(v) > digestLowCardValueCap {
			return false
		}
	}
	return true
}

// alertOverviewFooter names the commands that return complete alert rows.
func alertOverviewFooter(incidentID string, total int) string {
	incidentID = escapeText(capText(incidentID, digestCellCap))
	pages := max((total+alertOverviewPageSize-1)/alertOverviewPageSize, 1)
	pageArg := "1"
	if pages > 1 {
		pageArg = fmt.Sprintf("N (N = 1..%d)", pages)
	}
	return fmt.Sprintf("complete rows: flashduty incident alerts %s --output-format json --limit %d --page %s prints each page as a JSON array;"+
		" active or recovered only: flashduty incident alert-list %s --is-active (or --is-active=false) --output-format json\n",
		incidentID, alertOverviewPageSize, pageArg, incidentID)
}
