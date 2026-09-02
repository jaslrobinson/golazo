package notify

import (
	"strings"
	"testing"

	"github.com/jaslrobinson/golazo/internal/api"
)

func TestFormatGoalMessage_FootballUsesDescriptionAndDisplayMinute(t *testing.T) {
	// Football providers (ESPN) don't decompose scoring plays into
	// Player/Assist - they populate Description with the full play text
	// and DisplayMinute with a clock like "Q1 8:55" instead of a running
	// minute count. Both must be preferred over the soccer-only fields.
	event := api.MatchEvent{
		Type:          "touchdown",
		DisplayMinute: "Q1 8:55",
		Description:   "Jayden Maiava 1 Yd Run (Caden Chittenden Kick)",
	}
	home := api.Team{ShortName: "USC"}
	away := api.Team{ShortName: "LSU"}

	got := formatGoalMessage(event, home, away, 7, 0)

	if !strings.Contains(got, "Jayden Maiava 1 Yd Run (Caden Chittenden Kick)") {
		t.Errorf("got %q, want it to contain the play description", got)
	}
	if !strings.Contains(got, "Q1 8:55") {
		t.Errorf("got %q, want it to contain DisplayMinute %q", got, "Q1 8:55")
	}
	if !strings.Contains(got, "USC 7 - 0 LSU") {
		t.Errorf("got %q, want it to contain the score line", got)
	}
}

func TestFormatGoalMessage_SoccerUsesPlayerAssistAndMinute(t *testing.T) {
	// Unchanged soccer behavior: no Description, so fall back to
	// Player/Assist/Minute.
	assist := "Kevin De Bruyne"
	event := api.MatchEvent{
		Type:   "goal",
		Minute: 34,
		Player: strPtr("Erling Haaland"),
		Assist: &assist,
		Team:   api.Team{ShortName: "MCI"},
	}
	home := api.Team{ShortName: "MCI"}
	away := api.Team{ShortName: "ARS"}

	got := formatGoalMessage(event, home, away, 2, 1)

	if !strings.Contains(got, "Erling Haaland") {
		t.Errorf("got %q, want it to contain the scorer", got)
	}
	if !strings.Contains(got, "34'") {
		t.Errorf("got %q, want it to contain the minute marker", got)
	}
	if !strings.Contains(got, "Kevin De Bruyne") {
		t.Errorf("got %q, want it to contain the assist", got)
	}
}

func strPtr(s string) *string { return &s }

func TestNotificationTitle_FootballScoringTypesGetOwnTitles(t *testing.T) {
	tests := []struct {
		eventType string
		want      string
	}{
		{"touchdown", "🏈 TOUCHDOWN!"},
		{"field_goal", "🏈 FIELD GOAL!"},
		{"safety", "🏈 SAFETY!"},
		{"goal", "⚽ GOLAZO!"},
		{"", "⚽ GOLAZO!"},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if got := notificationTitle(tt.eventType); got != tt.want {
				t.Errorf("notificationTitle(%q) = %q, want %q", tt.eventType, got, tt.want)
			}
		})
	}
}
