package cli

import (
	"testing"

	gflashduty "github.com/flashcatcloud/go-flashduty"
)

func TestScheduleLayerCount(t *testing.T) {
	tests := []struct {
		name  string
		input gflashduty.ScheduleItem
		want  string
	}{
		{
			name:  "raw layers",
			input: gflashduty.ScheduleItem{Layers: []gflashduty.ScheduleLayer{{}, {}}},
			want:  "2",
		},
		{
			name:  "schedule layers fallback",
			input: gflashduty.ScheduleItem{ScheduleLayers: []gflashduty.ScheduleCalculatedLayer{{}, {}, {}}},
			want:  "3",
		},
		{
			name:  "layer schedules fallback",
			input: gflashduty.ScheduleItem{LayerSchedules: []gflashduty.ScheduleCalculatedLayer{{}, {}}},
			want:  "2",
		},
		{
			name:  "unknown when no layer arrays are present",
			input: gflashduty.ScheduleItem{},
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
