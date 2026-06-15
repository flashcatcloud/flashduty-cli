package cli

import (
	"fmt"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

// colSpec is a display-only column for the generic table renderer: which row
// field to show, its header, and an optional width cap. It NEVER affects flags or
// json/toon output — a wrong entry degrades a single table column at worst, it
// can't cause a functional error. Field is the Go struct field name on the row
// type; timestamp fields are detected and formatted automatically. Format, when
// set, renders the raw field value with a semantic formatter (percent, duration)
// the default scalar rendering doesn't apply — display-only, like the rest.
type colSpec struct {
	Header   string
	Field    string
	MaxWidth int
	Format   func(any) string
}

// fmtPercent renders a 0..1 ratio as a whole-number percentage ("85%"), matching
// the curated insight tables. A non-float value yields "".
func fmtPercent(v any) string {
	if f, ok := v.(float64); ok {
		return fmt.Sprintf("%.0f%%", f*100)
	}
	return ""
}

// fmtSecondsDuration renders a seconds count (int or float) as a human duration
// ("2m 30s"), matching the curated insight MTTA/MTTR/engaged columns.
func fmtSecondsDuration(v any) string {
	switch n := v.(type) {
	case float64:
		return output.FormatDurationFloat(n)
	case int64:
		return output.FormatDuration(int(n))
	case int:
		return output.FormatDuration(n)
	default:
		return ""
	}
}

// displayColumns maps a go-flashduty response row type (by Go type name) to its
// human table columns, seeded from the hand-written column sets the curated
// commands used before the CLI converged on generated commands. Row types with
// no entry fall back to the reflective heuristic in generic_table.go.
//
// Names are intentionally not resolved here (e.g. ChannelItem shows TEAM_ID /
// CREATOR_ID, not team/creator names): id→name enrichment belongs in the API
// response, not the client. Until the API carries those names, the table shows
// the ids; json/toon is unaffected either way.
var displayColumns = map[string][]colSpec{
	"IncidentInfo": {
		{Header: "ID", Field: "IncidentID"},
		{Header: "TITLE", Field: "Title", MaxWidth: 50},
		{Header: "SEVERITY", Field: "IncidentSeverity"},
		{Header: "PROGRESS", Field: "Progress"},
		{Header: "CHANNEL", Field: "ChannelName"},
		{Header: "CREATED", Field: "StartTime"},
	},
	"PastIncidentItem": {
		{Header: "ID", Field: "IncidentID"},
		{Header: "TITLE", Field: "Title", MaxWidth: 50},
		{Header: "SEVERITY", Field: "IncidentSeverity"},
		{Header: "PROGRESS", Field: "Progress"},
		{Header: "CHANNEL", Field: "ChannelName"},
		{Header: "CREATED", Field: "StartTime"},
	},
	"AlertInfo": {
		{Header: "ALERT_ID", Field: "AlertID"},
		{Header: "TITLE", Field: "Title", MaxWidth: 50},
		{Header: "SEVERITY", Field: "AlertSeverity"},
		{Header: "STATUS", Field: "AlertStatus"},
		{Header: "STARTED", Field: "StartTime"},
	},
	"AlertItem": {
		{Header: "ID", Field: "AlertID"},
		{Header: "TITLE", Field: "Title", MaxWidth: 50},
		{Header: "SEVERITY", Field: "AlertSeverity"},
		{Header: "STATUS", Field: "AlertStatus"},
		{Header: "EVENTS", Field: "EventCnt"},
		{Header: "CHANNEL", Field: "ChannelName"},
		{Header: "STARTED", Field: "StartTime"},
	},
	"AlertEventItem": {
		{Header: "EVENT_ID", Field: "EventID"},
		{Header: "ALERT_ID", Field: "AlertID"},
		{Header: "SEVERITY", Field: "EventSeverity"},
		{Header: "STATUS", Field: "EventStatus"},
		{Header: "TIME", Field: "EventTime"},
		{Header: "TITLE", Field: "Title", MaxWidth: 50},
	},
	"ChangeItem": {
		{Header: "ID", Field: "ChangeID"},
		{Header: "TITLE", Field: "Title", MaxWidth: 50},
		{Header: "STATUS", Field: "ChangeStatus"},
		{Header: "CHANNEL", Field: "ChannelName"},
		{Header: "TIME", Field: "StartTime"},
	},
	"ChannelItem": {
		{Header: "ID", Field: "ChannelID"},
		{Header: "NAME", Field: "ChannelName", MaxWidth: 40},
		{Header: "TEAM_ID", Field: "TeamID"},
		{Header: "CREATOR_ID", Field: "CreatorID"},
		{Header: "STATUS", Field: "Status"},
	},
	"TeamItem": {
		{Header: "ID", Field: "TeamID"},
		{Header: "NAME", Field: "TeamName", MaxWidth: 40},
	},
	"MemberItem": {
		{Header: "ID", Field: "MemberID"},
		{Header: "NAME", Field: "MemberName", MaxWidth: 30},
		{Header: "EMAIL", Field: "Email"},
		{Header: "STATUS", Field: "Status"},
		{Header: "TIMEZONE", Field: "TimeZone"},
	},
	"FieldItem": {
		{Header: "ID", Field: "FieldID"},
		{Header: "NAME", Field: "FieldName"},
		{Header: "DISPLAY_NAME", Field: "DisplayName"},
		{Header: "TYPE", Field: "FieldType"},
	},
	"WarRoomItem": {
		{Header: "INTEGRATION", Field: "IntegrationID"},
		{Header: "CHAT_ID", Field: "ChatID"},
		{Header: "INCIDENT_ID", Field: "IncidentID"},
		{Header: "STATUS", Field: "Status"},
		{Header: "PLUGIN", Field: "PluginType"},
		{Header: "CREATED", Field: "CreatedAt"},
	},
	"WarRoomPersonItem": {
		{Header: "PERSON_ID", Field: "PersonID"},
		{Header: "NAME", Field: "PersonName"},
		{Header: "EMAIL", Field: "Email"},
		{Header: "STATUS", Field: "Status"},
	},
	// DimensionInsightItem backs both `insight team` and `insight channel` (same
	// Go type, different populated name field) — show both name columns, the
	// irrelevant one renders empty. Columns mirror the curated insight tables.
	"DimensionInsightItem": {
		{Header: "TEAM", Field: "TeamName", MaxWidth: 30},
		{Header: "CHANNEL", Field: "ChannelName", MaxWidth: 30},
		{Header: "INCIDENTS", Field: "TotalIncidentCnt"},
		{Header: "ACK%", Field: "AcknowledgementPct", Format: fmtPercent},
		{Header: "MTTA", Field: "MeanSecondsToAck", Format: fmtSecondsDuration},
		{Header: "MTTR", Field: "MeanSecondsToClose", Format: fmtSecondsDuration},
		{Header: "NOISE_REDUCTION", Field: "NoiseReductionPct", Format: fmtPercent},
		{Header: "ALERTS", Field: "TotalAlertCnt"},
		{Header: "EVENTS", Field: "TotalAlertEventCnt"},
	},
	"ResponderInsightItem": {
		{Header: "RESPONDER", Field: "ResponderName", MaxWidth: 30},
		{Header: "INCIDENTS", Field: "TotalIncidentCnt"},
		{Header: "ACK%", Field: "AcknowledgementPct", Format: fmtPercent},
		{Header: "MTTA", Field: "MeanSecondsToAck", Format: fmtSecondsDuration},
		{Header: "INTERRUPTIONS", Field: "TotalInterruptions"},
		{Header: "ENGAGED", Field: "TotalEngagedSeconds", Format: fmtSecondsDuration},
	},
}
