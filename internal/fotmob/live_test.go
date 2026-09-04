package fotmob

import (
	"strings"
	"testing"
	"time"

	"github.com/jaslrobinson/golazo/internal/api"
)

// edtZone returns a fixed EDT zone for deterministic local-day tests.
func edtZone() *time.Location {
	return time.FixedZone("EDT", -4*3600)
}

func TestClassifyLeagueMatches_LiveAcrossUTCBoundary(t *testing.T) {
	// Saudi Arabia vs Uruguay: kickoff 2026-06-15T22:00Z, user clock is
	// 2026-06-16 00:30 UTC (= 2026-06-15 20:30 EDT). UTC "today" has rolled
	// to June 16, but the match is still in progress on UTC June 15. The
	// status-only filter must keep it.
	matches := []fotmobMatch{
		{
			ID: "1",
			Status: status{
				UTCTime:  "2026-06-15T22:00:00.000Z",
				Started:  boolPtr(true),
				Finished: boolPtr(false),
			},
			Home: team{ID: "100", Name: "Saudi Arabia", ShortName: "KSA"},
			Away: team{ID: "200", Name: "Uruguay", ShortName: "URU"},
		},
	}
	leagueInfo := league{ID: 77, Name: "FIFA World Cup"}

	now := time.Date(2026, 6, 15, 20, 30, 0, 0, edtZone())
	live, upcoming := classifyLeagueMatches(matches, leagueInfo, now)

	if len(live) != 1 {
		t.Fatalf("live len = %d, want 1", len(live))
	}
	if len(upcoming) != 0 {
		t.Fatalf("upcoming len = %d, want 0", len(upcoming))
	}
	if live[0].HomeTeam.Name != "Saudi Arabia" {
		t.Errorf("home = %q, want Saudi Arabia", live[0].HomeTeam.Name)
	}
	if live[0].League.ID != 77 {
		t.Errorf("league ID = %d, want 77 (filled from leagueInfo)", live[0].League.ID)
	}
}

func TestClassifyLeagueMatches_UpcomingGatedByLocalToday(t *testing.T) {
	// Two not-started matches:
	//   - kickoff 2026-06-16T00:30Z (= 2026-06-15 20:30 EDT) → local-today (keep)
	//   - kickoff 2026-06-16T20:00Z (= 2026-06-16 16:00 EDT) → tomorrow local (drop)
	matches := []fotmobMatch{
		{
			ID: "1",
			Status: status{
				UTCTime:  "2026-06-16T00:30:00.000Z",
				Started:  boolPtr(false),
				Finished: boolPtr(false),
			},
		},
		{
			ID: "2",
			Status: status{
				UTCTime:  "2026-06-16T20:00:00.000Z",
				Started:  boolPtr(false),
				Finished: boolPtr(false),
			},
		},
	}

	// User clock: 2026-06-15 18:00 EDT.
	now := time.Date(2026, 6, 15, 18, 0, 0, 0, edtZone())
	live, upcoming := classifyLeagueMatches(matches, league{ID: 77}, now)

	if len(live) != 0 {
		t.Fatalf("live len = %d, want 0", len(live))
	}
	if len(upcoming) != 1 {
		t.Fatalf("upcoming len = %d, want 1", len(upcoming))
	}
	if upcoming[0].ID != 1 {
		t.Errorf("upcoming ID = %d, want 1", upcoming[0].ID)
	}
}

func TestClassifyLeagueMatches_CancelledExcluded(t *testing.T) {
	matches := []fotmobMatch{
		{
			ID: "1",
			Status: status{
				UTCTime:   "2026-06-15T22:00:00.000Z",
				Started:   boolPtr(true),
				Finished:  boolPtr(false),
				Cancelled: boolPtr(true),
			},
		},
	}
	now := time.Date(2026, 6, 15, 20, 30, 0, 0, edtZone())
	live, upcoming := classifyLeagueMatches(matches, league{ID: 77}, now)

	if len(live) != 0 {
		t.Errorf("live len = %d, want 0 (cancelled match)", len(live))
	}
	if len(upcoming) != 0 {
		t.Errorf("upcoming len = %d, want 0 (cancelled match)", len(upcoming))
	}
}

func TestClassifyLeagueMatches_FinishedExcluded(t *testing.T) {
	matches := []fotmobMatch{
		{
			ID: "1",
			Status: status{
				UTCTime:  "2026-06-15T18:00:00.000Z",
				Started:  boolPtr(true),
				Finished: boolPtr(true),
			},
		},
	}
	now := time.Date(2026, 6, 15, 20, 30, 0, 0, edtZone())
	live, upcoming := classifyLeagueMatches(matches, league{ID: 77}, now)

	if len(live) != 0 {
		t.Errorf("live len = %d, want 0 (finished match)", len(live))
	}
	if len(upcoming) != 0 {
		t.Errorf("upcoming len = %d, want 0", len(upcoming))
	}
}

func TestClassifyLeagueMatches_MissingUTCTimeSkipped(t *testing.T) {
	matches := []fotmobMatch{
		{
			ID: "1",
			Status: status{
				Started:  boolPtr(true),
				Finished: boolPtr(false),
			},
		},
	}
	now := time.Date(2026, 6, 15, 20, 30, 0, 0, edtZone())
	live, upcoming := classifyLeagueMatches(matches, league{ID: 77}, now)
	if len(live) != 0 || len(upcoming) != 0 {
		t.Errorf("live=%d upcoming=%d, want both 0 (empty utcTime should be skipped)", len(live), len(upcoming))
	}
}

// TestFormatEvent_CFBScoringPlays guards against a regression where college
// football scoring plays (espncfb.mapScoringPlays sets Type to "touchdown",
// "field_goal", "safety", or "score" — never "goal") fell through to
// formatEvent's generic default case. That case renders event.Type as a bare
// label ("field_goal") and always shows "0'" (Minute defaulted to zero before
// mapScoringPlays started computing it), which is what the Updates panel was
// showing live on 2026-09-04 for three concurrent in-progress games.
func TestFormatEvent_CFBScoringPlays(t *testing.T) {
	parser := NewLiveUpdateParser()
	homeTeam := api.Team{ID: 1, ShortName: "EMU"}
	awayTeam := api.Team{ID: 2, ShortName: "SJSU"}

	tests := []struct {
		eventType string
		wantLabel string
	}{
		{"touchdown", "[TOUCHDOWN]"},
		{"field_goal", "[FIELD GOAL]"},
		{"safety", "[SAFETY]"},
		{"score", "[SCORE]"},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			event := api.MatchEvent{
				Minute:      21,
				Type:        tt.eventType,
				Team:        homeTeam,
				Description: "N. Kim pass to H. Mack for 5 yds, for a TD",
			}
			got := parser.formatEvent(event, homeTeam, awayTeam)
			if !strings.Contains(got, tt.wantLabel) {
				t.Errorf("formatEvent(%q) = %q, want it to contain %q", tt.eventType, got, tt.wantLabel)
			}
			if !strings.Contains(got, "21'") {
				t.Errorf("formatEvent(%q) = %q, want minute 21' (not 0')", tt.eventType, got)
			}
			if !strings.Contains(got, event.Description) {
				t.Errorf("formatEvent(%q) = %q, want it to contain the play description", tt.eventType, got)
			}
		})
	}
}
