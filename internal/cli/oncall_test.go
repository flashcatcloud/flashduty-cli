package cli

import (
	"testing"

	"github.com/flashcatcloud/go-flashduty"
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
