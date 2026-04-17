package cli

import (
	"testing"

	flashduty "github.com/flashcatcloud/flashduty-sdk"
)

func TestScheduleLayerCount(t *testing.T) {
	tests := []struct {
		name  string
		input flashduty.ScheduleDetail
		want  string
	}{
		{
			name:  "raw layers",
			input: flashduty.ScheduleDetail{Layers: []flashduty.ScheduleLayer{{}, {}}},
			want:  "2",
		},
		{
			name:  "schedule layers fallback",
			input: flashduty.ScheduleDetail{ScheduleLayers: []flashduty.ScheduleCalculatedLayer{{}, {}, {}}},
			want:  "3",
		},
		{
			name:  "layer schedules fallback",
			input: flashduty.ScheduleDetail{LayerSchedules: []flashduty.ScheduleCalculatedLayer{{}, {}}},
			want:  "2",
		},
		{
			name:  "unknown when only computed snapshots exist",
			input: flashduty.ScheduleDetail{FinalSchedule: flashduty.ScheduleCalculatedLayer{LayerName: "final"}},
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
