package ui

import (
	"strings"
	"testing"

	"github.com/0xjuanma/golazo/internal/api"
)

func TestRenderAggregateSection_WithData(t *testing.T) {
	details := &api.MatchDetails{
		AggregateScore:     "5 - 7",
		WhoLostOnAggregate: "Juventus",
	}

	lines := renderAggregateSection(details, 60)
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "AGG.") {
		t.Errorf("expected output to contain %q, got:\n%s", "AGG.", joined)
	}
	if !strings.Contains(joined, "5 - 7") {
		t.Errorf("expected output to contain %q, got:\n%s", "5 - 7", joined)
	}
	if !strings.Contains(joined, "Juventus eliminated") {
		t.Errorf("expected output to contain %q, got:\n%s", "Juventus eliminated", joined)
	}
}

func TestRenderAggregateSection_NoEliminated(t *testing.T) {
	details := &api.MatchDetails{
		AggregateScore:     "3 - 1",
		WhoLostOnAggregate: "",
	}

	lines := renderAggregateSection(details, 60)
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, "AGG.") {
		t.Errorf("expected output to contain %q", "AGG.")
	}
	if !strings.Contains(joined, "3 - 1") {
		t.Errorf("expected output to contain %q", "3 - 1")
	}
	if strings.Contains(joined, "eliminated") {
		t.Errorf("expected no %q line when WhoLostOnAggregate is empty", "eliminated")
	}
}

func TestRenderMatchDetails_AggregateSection_Rendered(t *testing.T) {
	home, away := 3, 2
	details := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusFinished,
			HomeTeam: api.Team{Name: "Galatasaray", ShortName: "Gala"},
			AwayTeam: api.Team{Name: "Juventus", ShortName: "Juve"},
		},
		AggregateScore:     "5 - 7",
		WhoLostOnAggregate: "Juventus",
	}
	details.HomeScore = &home
	details.AwayScore = &away

	header, _ := RenderMatchDetails(MatchDetailsConfig{
		Width:   80,
		Height:  40,
		Details: details,
	})

	if !strings.Contains(header, "AGG.") {
		t.Errorf("RenderMatchDetails header should contain %q when AggregateScore is set", "AGG.")
	}
}

func TestRenderMatchDetails_NoAggregateSection_WhenEmpty(t *testing.T) {
	home, away := 2, 1
	details := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusFinished,
			HomeTeam: api.Team{Name: "Arsenal", ShortName: "ARS"},
			AwayTeam: api.Team{Name: "Chelsea", ShortName: "CHE"},
		},
	}
	details.HomeScore = &home
	details.AwayScore = &away

	header, _ := RenderMatchDetails(MatchDetailsConfig{
		Width:   80,
		Height:  40,
		Details: details,
	})

	if strings.Contains(header, "AGG.") {
		t.Errorf("RenderMatchDetails header should NOT contain %q for non-knockout match", "AGG.")
	}
}

func TestRenderMatchDetails_HelmetsRow_RendersForSeededTeam(t *testing.T) {
	// Technique-agnostic (doesn't assume which glyph vocabulary the
	// current internal/ui/helmet renderer uses): a header with a seeded
	// team's helmet art must have more lines than an otherwise-identical
	// header with no seeded team.
	seeded := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusNotStarted,
			HomeTeam: api.Team{ID: 213, Name: "Penn State Nittany Lions", ShortName: "Penn St"},
			AwayTeam: api.Team{Name: "Some Other Team", ShortName: "SOT"},
		},
	}
	unseeded := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusNotStarted,
			HomeTeam: api.Team{Name: "Some Team", ShortName: "ST"},
			AwayTeam: api.Team{Name: "Some Other Team", ShortName: "SOT"},
		},
	}

	withArt, _ := RenderMatchDetails(MatchDetailsConfig{Width: 80, Height: 40, Details: seeded})
	withoutArt, _ := RenderMatchDetails(MatchDetailsConfig{Width: 80, Height: 40, Details: unseeded})

	if strings.Count(withArt, "\n") <= strings.Count(withoutArt, "\n") {
		t.Errorf("RenderMatchDetails header with a seeded team should have more lines than one without")
	}
}

func TestRenderHelmetsRow_NonEmptyWhenOneTeamSeeded(t *testing.T) {
	if got := renderHelmetsRow(213, 999999999, 80); got == "" {
		t.Fatalf("renderHelmetsRow(213, unseeded, 80) = empty, want helmet art")
	}
}

func TestRenderHelmetsRow_EmptyWhenNeitherTeamSeeded(t *testing.T) {
	if got := renderHelmetsRow(999999999, 999999999, 80); got != "" {
		t.Fatalf("renderHelmetsRow(unseeded, unseeded, 80) = %q, want empty string", got)
	}
}

func TestRenderHelmetsRow_OmittedWhenContentWidthTooNarrow(t *testing.T) {
	// Two 9-column helmets plus a 6-column gap need exactly 24 columns.
	if got := renderHelmetsRow(213, 333, 23); got != "" {
		t.Fatalf("renderHelmetsRow(213, 333, 23) = %q, want empty string (1 column too narrow)", got)
	}
	if got := renderHelmetsRow(213, 333, 24); got == "" {
		t.Fatalf("renderHelmetsRow(213, 333, 24) = empty, want helmet art (exact fit)")
	}
}

func TestRenderStatusLine_FinishedPrefersProviderLiveTime(t *testing.T) {
	// ESPN's CFB mapper populates LiveTime with its own native finished
	// text ("Final", "Final/OT", ...) even for finished matches - the
	// generic "FT" (soccer's Full Time) fallback must not override it.
	liveTime := "Final/OT"
	details := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusFinished,
			LiveTime: &liveTime,
			League:   api.League{Name: "NCAAF"},
		},
	}

	got := renderStatusLine(details, 60)

	if !strings.Contains(got, "Final/OT") {
		t.Errorf("renderStatusLine() = %q, want it to contain the provider's own %q", got, "Final/OT")
	}
	if strings.Contains(got, "FT") {
		t.Errorf("renderStatusLine() = %q, should not contain the generic FT fallback when LiveTime is set", got)
	}
}

func TestRenderStatusLine_FinishedFallsBackToFTWhenNoLiveTime(t *testing.T) {
	// FotMob (soccer) leaves LiveTime nil once a match finishes - the
	// generic "FT" fallback must still apply in that case.
	details := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusFinished,
			LiveTime: nil,
			League:   api.League{Name: "Premier League"},
		},
	}

	got := renderStatusLine(details, 60)

	if !strings.Contains(got, "FT") {
		t.Errorf("renderStatusLine() = %q, want it to contain the fallback %q when LiveTime is nil", got, "FT")
	}
}
