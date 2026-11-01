package cli

import (
	"encoding/json"
	"testing"

	"github.com/flashcatcloud/go-flashduty"

	"github.com/flashcatcloud/flashduty-cli/internal/output"
)

func TestScheduleLayerCount(t *testing.T) {
	tests := []struct {
		name  string
		input flashduty.ScheduleItem
		want  string
	}{
		{
			name:  "raw layers",
			input: flashduty.ScheduleItem{Layers: []flashduty.ScheduleLayer{{}, {}}},
			want:  "2",
		},
		{
			name:  "schedule layers fallback",
			input: flashduty.ScheduleItem{ScheduleLayers: []flashduty.ScheduleCalculatedLayer{{}, {}, {}}},
			want:  "3",
		},
		{
			name:  "layer schedules fallback",
			input: flashduty.ScheduleItem{LayerSchedules: []flashduty.ScheduleCalculatedLayer{{}, {}}},
			want:  "2",
		},
		{
			name:  "unknown when no layer arrays are present",
			input: flashduty.ScheduleItem{},
			want:  "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scheduleLayerCount(tt.input)
			if got != tt.want {
				t.Fatalf("scheduleLayerCount() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNobodyOnCallRendersPlaceholders(t *testing.T) {
	var s flashduty.ScheduleItem
	if err := json.Unmarshal([]byte(`{"cur_oncall":null,"next_oncall":null}`), &s); err != nil {
		t.Fatal(err)
	}
	if got := formatOncallMembers(s.CurOncall, nil); got != "-" {
		t.Fatalf("formatOncallMembers = %q", got)
	}
	start, end := oncallWindow(s.NextOncall)
	if got := output.FormatTime(start); got != "-" {
		t.Fatalf("start = %q", got)
	}
	if got := output.FormatTime(end); got != "-" {
		t.Fatalf("end = %q", got)
	}
	if got := scheduleGroupName(nil); got != "-" {
		t.Fatalf("scheduleGroupName = %q", got)
	}
}
