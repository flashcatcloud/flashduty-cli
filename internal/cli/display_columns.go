package cli

// colSpec is a display-only column for the generic table renderer: which row
// field to show, its header, and an optional width cap. It NEVER affects flags or
// json/toon output — a wrong entry degrades a single table column at worst, it
// can't cause a functional error. Field is the Go struct field name on the row
// type; timestamp fields are detected and formatted automatically.
type colSpec struct {
	Header   string
	Field    string
	MaxWidth int
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
}
