package app

import "testing"

func TestIsScoringEvent(t *testing.T) {
	tests := []struct {
		eventType string
		want      bool
	}{
		{"goal", true},
		{"GOAL", true},
		{"touchdown", true},
		{"field_goal", true},
		{"safety", true},
		{"card", false},
		{"substitution", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if got := isScoringEvent(tt.eventType); got != tt.want {
				t.Errorf("isScoringEvent(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}
