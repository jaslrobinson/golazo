package espncfb

import (
	"encoding/json"
	"testing"
	"time"
)

// These fixtures are trimmed excerpts of real responses captured from
// site.api.espn.com on 2026-08-31 (San José State at USC, event 401864494,
// and the concurrent Week 1 scoreboard/rankings). They exist to catch a
// mismatched json tag that a compile-clean-but-untested mapping wouldn't.

// TestMatchDetailsLeague_FallsBackToBoxscoreConferenceID reproduces a real
// gap found live (2026-09-02, event 401856766, TCU vs UNC): MatchDetails'
// League came back {"id":0,"name":""} even though the same team's
// conferenceId is populated correctly on /scoreboard responses. ESPN's
// /summary header competitor team object is a trimmed blob that (per that
// live response) doesn't carry conferenceId — but the boxscore team object
// does — so the header value alone isn't reliable and must fall back to the
// boxscore's.
func TestMatchDetailsLeague_FallsBackToBoxscoreConferenceID(t *testing.T) {
	got := matchDetailsLeague("", "8")
	want := conferenceByID(8)
	if got != want {
		t.Errorf("matchDetailsLeague(\"\", \"8\") = %+v, want %+v", got, want)
	}
}

func TestMatchDetailsLeague_PrefersHeaderConferenceIDWhenPresent(t *testing.T) {
	got := matchDetailsLeague("5", "8")
	want := conferenceByID(5)
	if got != want {
		t.Errorf("matchDetailsLeague(\"5\", \"8\") = %+v, want %+v (header should win when non-empty)", got, want)
	}
}

func TestMatchDetailsLeague_EmptyWhenNeitherSourceHasIt(t *testing.T) {
	got := matchDetailsLeague("", "")
	if got.ID != 0 || got.Name != "" {
		t.Errorf("matchDetailsLeague(\"\", \"\") = %+v, want zero-value League", got)
	}
}

func TestMapStatistics_RealShape(t *testing.T) {
	homeJSON := `{
		"team": {"abbreviation": "USC"},
		"homeAway": "home",
		"statistics": [
			{"name": "firstDowns", "displayValue": "18", "value": 18, "label": "1st Downs"},
			{"name": "totalYards", "displayValue": "336", "value": "-", "label": "Total Yards"},
			{"name": "netPassingYards", "displayValue": "234", "value": 234, "label": "Passing"}
		]
	}`
	awayJSON := `{
		"team": {"abbreviation": "SJSU"},
		"homeAway": "away",
		"statistics": [
			{"name": "firstDowns", "displayValue": "12", "value": 12, "label": "1st Downs"},
			{"name": "totalYards", "displayValue": "290", "value": "-", "label": "Total Yards"},
			{"name": "netPassingYards", "displayValue": "210", "value": 210, "label": "Passing"}
		]
	}`

	var home, away rawBoxscoreTeam
	if err := json.Unmarshal([]byte(homeJSON), &home); err != nil {
		t.Fatalf("unmarshal home: %v", err)
	}
	if err := json.Unmarshal([]byte(awayJSON), &away); err != nil {
		t.Fatalf("unmarshal away: %v", err)
	}

	stats := mapStatistics(home, away)
	if len(stats) != 3 {
		t.Fatalf("expected 3 stats, got %d", len(stats))
	}
	if stats[0].Key != "firstDowns" || stats[0].Label != "1st Downs" || stats[0].HomeValue != "18" || stats[0].AwayValue != "12" {
		t.Errorf("firstDowns row wrong: %+v", stats[0])
	}
	if stats[2].HomeValue != "234" || stats[2].AwayValue != "210" {
		t.Errorf("netPassingYards row wrong: %+v", stats[2])
	}
}

func TestMapLeaders_RealShape(t *testing.T) {
	raw := []rawTeamLeaders{
		{
			Team: rawDriveTeam{Abbreviation: "USC"},
			Leaders: []rawLeaderCategory{
				{
					Name:        "passingYards",
					DisplayName: "Passing Yards",
					Leaders: []rawLeaderEntry{
						{DisplayValue: "25/29, 286 YDS, 2 TD", Value: 286, Athlete: rawAthlete{DisplayName: "Jayden Maiava"}},
					},
				},
			},
		},
		{
			Team: rawDriveTeam{Abbreviation: "SJSU"},
			Leaders: []rawLeaderCategory{
				{
					Name:        "passingYards",
					DisplayName: "Passing Yards",
					Leaders: []rawLeaderEntry{
						{DisplayValue: "18/31, 210 YDS", Value: 210, Athlete: rawAthlete{DisplayName: "M. Latu"}},
					},
				},
			},
		},
	}

	cats := mapLeaders(raw, "USC", "SJSU")
	if len(cats) != 1 {
		t.Fatalf("expected 1 category, got %d", len(cats))
	}
	if cats[0].Key != "passingYards" || cats[0].Label != "Passing Yards" {
		t.Errorf("category header wrong: %+v", cats[0])
	}
	if len(cats[0].HomeLeaders) != 1 || cats[0].HomeLeaders[0].PlayerName != "Jayden Maiava" {
		t.Errorf("home leaders wrong: %+v", cats[0].HomeLeaders)
	}
	if len(cats[0].AwayLeaders) != 1 || cats[0].AwayLeaders[0].PlayerName != "M. Latu" {
		t.Errorf("away leaders wrong: %+v", cats[0].AwayLeaders)
	}
}

func TestMapMomentum_RealShape(t *testing.T) {
	rawJSON := `[
		{"homeWinPercentage": 0.59, "tiePercentage": 0, "playId": "4018644942"},
		{"homeWinPercentage": 0.6033, "tiePercentage": 0, "playId": "4018644943"}
	]`
	var raw []rawWinProb
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	points := mapMomentum(raw)
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].PlayID != "4018644942" || points[0].HomeWinPct != 0.59 {
		t.Errorf("point 0 wrong: %+v", points[0])
	}
}

func TestMapRankingPoll_RealShape(t *testing.T) {
	rawJSON := `{
		"name": "AP Top 25",
		"ranks": [
			{
				"current": 1, "previous": 0, "points": 1672, "firstPlaceVotes": 40, "trend": "-1",
				"team": {"id": "194", "location": "Ohio State", "name": "Buckeyes", "abbreviation": "OSU"}
			}
		]
	}`
	var poll rawPoll
	if err := json.Unmarshal([]byte(rawJSON), &poll); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	mapped := mapRankingPoll(poll)
	if mapped.Name != "AP Top 25" {
		t.Errorf("poll name wrong: %q", mapped.Name)
	}
	if len(mapped.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(mapped.Entries))
	}
	e := mapped.Entries[0]
	if e.Rank != 1 || e.Points != 1672 || e.FirstPlaceVotes != 40 || e.Trend != "-1" {
		t.Errorf("entry fields wrong: %+v", e)
	}
	// Fixture omits "displayName" (unconfirmed whether the real rankings
	// response includes it), so mapTeam falls back to "location name" —
	// same full-name convention Team.Name uses elsewhere.
	if e.Team.ID != 194 || e.Team.Name != "Ohio State Buckeyes" {
		t.Errorf("entry team wrong: %+v", e.Team)
	}
}

func TestMapRankingPoll_FloatFormattedPoints(t *testing.T) {
	// Regression test: a live rankings call returned "points": 1672.0 for a
	// non-AP poll (Coaches or FCS — the initial capture only sampled AP Top
	// 25's whole-number format), which crashed json.Unmarshal when Points
	// was typed int. Points/FirstPlaceVotes must accept float-formatted
	// whole numbers too.
	rawJSON := `{
		"name": "AFCA Coaches Poll",
		"ranks": [
			{
				"current": 1, "previous": 1, "points": 1672.0, "firstPlaceVotes": 40.0, "trend": "-",
				"team": {"id": "194", "displayName": "Ohio State Buckeyes"}
			}
		]
	}`
	var poll rawPoll
	if err := json.Unmarshal([]byte(rawJSON), &poll); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	mapped := mapRankingPoll(poll)
	if len(mapped.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(mapped.Entries))
	}
	e := mapped.Entries[0]
	if e.Points != 1672 {
		t.Errorf("Points wrong: got %d, want 1672", e.Points)
	}
	if e.FirstPlaceVotes != 40 {
		t.Errorf("FirstPlaceVotes wrong: got %d, want 40", e.FirstPlaceVotes)
	}
}

func TestRawPlay_PeriodIsAnObjectNotAnInt(t *testing.T) {
	// Regression test: rawPlay.Period was originally typed as a plain int,
	// which crashed json.Unmarshal on every real drives.previous[].plays[]
	// entry — MatchDetails failed for every match, live and finished. The
	// field is unused by any mapping logic today (hence uncaught by the
	// initial live capture, which never printed a play's "period"), but it
	// must still decode without error since it's embedded in every play.
	rawJSON := `{
		"text": "(15:00) #99 M.Brown kickoff 64 yards to the USC01",
		"period": {"number": 1},
		"clock": {"displayValue": "15:00"},
		"start": {"down": 0, "distance": 0, "yardLine": 35, "yardsToEndzone": 35},
		"end": {"down": 0, "distance": 0, "yardLine": 1, "yardsToEndzone": 1}
	}`
	var play rawPlay
	if err := json.Unmarshal([]byte(rawJSON), &play); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if play.Period.Number != 1 {
		t.Errorf("period wrong: got %+v", play.Period)
	}
}

func TestMapPlayPosition_RealShape(t *testing.T) {
	// From drives.previous[0].plays[0].start in the real captured response.
	rawJSON := `{"down": 1, "distance": 10, "yardLine": 65, "yardsToEndzone": 65}`
	var pos rawPlayPosition
	if err := json.Unmarshal([]byte(rawJSON), &pos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if pos.Down != 1 || pos.Distance != 10 || pos.YardLine != 65 || pos.YardsToEndzone != 65 {
		t.Errorf("play position wrong: %+v", pos)
	}
}

func TestMapScoringPlays_RealShape(t *testing.T) {
	rawJSON := `[{
		"id": "40186449471",
		"type": {"id": "68", "text": "Rushing Touchdown", "abbreviation": "TD"},
		"text": "Jayden Maiava 1 Yd Run (Caden Chittenden Kick)",
		"awayScore": 0,
		"homeScore": 7,
		"period": {"number": 1},
		"clock": {"value": 535, "displayValue": "8:55"},
		"team": {"id": "30", "displayName": "USC Trojans", "abbreviation": "USC"}
	}]`
	var raw []rawScoringPlay
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	events := mapScoringPlays(raw)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.Type != "touchdown" {
		t.Errorf("type wrong: got %q, want touchdown", e.Type)
	}
	if e.Description != "Jayden Maiava 1 Yd Run (Caden Chittenden Kick)" {
		t.Errorf("description wrong: %q", e.Description)
	}
	if e.DisplayMinute != "Q1 8:55" {
		t.Errorf("display minute wrong: %q", e.DisplayMinute)
	}
	if e.Team.Name != "USC Trojans" {
		t.Errorf("team wrong: %+v", e.Team)
	}
}

func TestConferenceByID_KnownAndUnknown(t *testing.T) {
	sec := conferenceByID(8)
	if sec.Name != "SEC" {
		t.Errorf("expected SEC for id 8, got %+v", sec)
	}
	unknown := conferenceByID(9999)
	if unknown.ID != 9999 || unknown.Name != "" {
		t.Errorf("unknown conference should keep the ID with an empty name, got %+v", unknown)
	}
}

func TestMapMatch_PopulatesLeagueFromConferenceID(t *testing.T) {
	// Regression test: mapMatch originally never set League at all, which
	// silently broke the standings keybinding (LeagueTable(ctx, 0, "") always
	// failed). conferenceId is confirmed live as a plain JSON string.
	rawJSON := `{
		"id": "401858425",
		"date": "2026-09-05T23:30:00Z",
		"competitions": [{
			"status": {"type": {"state": "pre"}},
			"competitors": [
				{"homeAway": "home", "team": {"id": "84", "location": "Indiana", "abbreviation": "IU", "conferenceId": "5"}, "score": "0"},
				{"homeAway": "away", "team": {"id": "249", "location": "North Texas", "abbreviation": "UNT", "conferenceId": "151"}, "score": "0"}
			]
		}]
	}`
	var e rawEvent
	if err := json.Unmarshal([]byte(rawJSON), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m, ok := mapMatch(e)
	if !ok {
		t.Fatal("mapMatch returned ok=false")
	}
	if m.League.ID != 5 || m.League.Name != "Big Ten" {
		t.Errorf("League should come from the home team's conferenceId, got %+v", m.League)
	}
}

func TestRawLeagueTableResponse_RealShape(t *testing.T) {
	// Trimmed excerpt of a real response from
	// site.web.api.espn.com/apis/v2/.../standings?season=2026&group=8
	// (SEC), captured 2026-09-01. Regression test: an earlier version of
	// this type assumed a "children[].standings.entries" wrapper that
	// doesn't exist — entries live directly under top-level "standings".
	rawJSON := `{
		"standings": {
			"entries": [
				{
					"team": {"id": "2", "location": "Auburn", "name": "Tigers", "abbreviation": "AUB", "displayName": "Auburn Tigers"},
					"stats": [
						{"name": "wins", "type": "vsconf_wins", "value": 0, "displayValue": "0"},
						{"name": "overall", "type": "total", "summary": "0-0", "displayValue": "0-0"},
						{"name": "vs. Conf.", "type": "vsconf", "summary": "0-0", "displayValue": "0-0"}
					]
				}
			]
		}
	}`
	var raw rawLeagueTableResponse
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(raw.Standings.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(raw.Standings.Entries))
	}
	entry := raw.Standings.Entries[0]
	if entry.Team.DisplayName != "Auburn Tigers" {
		t.Errorf("team wrong: %+v", entry.Team)
	}
	if got := standingsStat(entry.Stats, "vsconf"); got != "0-0" {
		t.Errorf("vsconf composite lookup wrong: got %q", got)
	}
	if got := standingsStat(entry.Stats, "total"); got != "0-0" {
		t.Errorf("total composite lookup wrong: got %q", got)
	}

	mapped := mapTeam(entry.Team)
	if mapped.Name != "Auburn Tigers" || mapped.ID != 2 {
		t.Errorf("mapTeam(entry.Team) wrong: %+v", mapped)
	}
}

func TestEspnSeasonYear(t *testing.T) {
	// Just documents the January-belongs-to-previous-season rule; doesn't
	// depend on the actual current date.
	jan := time.Date(2027, time.January, 5, 0, 0, 0, 0, time.UTC)
	sep := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	if got := seasonYearFor(jan); got != 2026 {
		t.Errorf("January should belong to the previous season year, got %d", got)
	}
	if got := seasonYearFor(sep); got != 2026 {
		t.Errorf("September should belong to its own year, got %d", got)
	}
}

func TestMapMatch_ScoreboardShape(t *testing.T) {
	rawJSON := `{
		"id": "401864494",
		"date": "2026-08-30T23:30:00Z",
		"competitions": [{
			"status": {"clock": 0, "displayClock": "0:00", "period": 4, "type": {"state": "post", "completed": true, "shortDetail": "Final"}},
			"competitors": [
				{"homeAway": "home", "team": {"id": "30", "location": "USC", "abbreviation": "USC"}, "score": "24"},
				{"homeAway": "away", "team": {"id": "23", "location": "San José State", "abbreviation": "SJSU"}, "score": "17"}
			]
		}]
	}`
	var e rawEvent
	if err := json.Unmarshal([]byte(rawJSON), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m, ok := mapMatch(e)
	if !ok {
		t.Fatal("mapMatch returned ok=false")
	}
	if m.HomeTeam.Name != "USC" || m.AwayTeam.Name != "San José State" {
		t.Errorf("team names wrong: home=%q away=%q", m.HomeTeam.Name, m.AwayTeam.Name)
	}
	if m.HomeScore == nil || *m.HomeScore != 24 || m.AwayScore == nil || *m.AwayScore != 17 {
		t.Errorf("scores wrong: home=%v away=%v", m.HomeScore, m.AwayScore)
	}
	if m.Status != "finished" {
		t.Errorf("status wrong: %q", m.Status)
	}
}

// TestMapMatch_DateWithoutSeconds guards against a regression where ESPN's
// scoreboard sends dates without a seconds field (e.g. "2026-09-05T00:00Z"
// for games at the top of the hour) — confirmed live 2026-09-04 against both
// /scoreboard and /summary. time.RFC3339 alone rejects that string outright,
// silently leaving MatchTime nil and the UI showing "--:--" for every
// upcoming kickoff.
func TestMapMatch_DateWithoutSeconds(t *testing.T) {
	rawJSON := `{
		"id": "401856664",
		"date": "2026-09-05T00:00Z",
		"competitions": [{
			"status": {"clock": 0, "displayClock": "0:00", "period": 0, "type": {"state": "pre", "completed": false, "shortDetail": "9/4 - 8:00 PM EDT"}},
			"competitors": [
				{"homeAway": "home", "team": {"id": "201", "location": "Oklahoma", "abbreviation": "OU"}, "score": "0"},
				{"homeAway": "away", "team": {"id": "2638", "location": "UTEP", "abbreviation": "UTEP"}, "score": "0"}
			]
		}]
	}`
	var e rawEvent
	if err := json.Unmarshal([]byte(rawJSON), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m, ok := mapMatch(e)
	if !ok {
		t.Fatal("mapMatch returned ok=false")
	}
	if m.MatchTime == nil {
		t.Fatal("MatchTime is nil for a seconds-less ESPN date")
	}
	want := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	if !m.MatchTime.Equal(want) {
		t.Errorf("MatchTime = %v, want %v", m.MatchTime, want)
	}
}

// TestParseESPNTime covers parseESPNTime directly, since both mapMatch
// (scoreboard) and MatchDetails (summary, client.go:162) share it — a direct
// test protects the summary call site too, without needing to stand up an
// httptest server just to exercise date parsing.
func TestParseESPNTime(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
		ok    bool
	}{
		{"RFC3339 with seconds", "2026-09-05T23:30:00Z", time.Date(2026, 9, 5, 23, 30, 0, 0, time.UTC), true},
		{"seconds omitted", "2026-09-05T00:00Z", time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC), true},
		{"garbage", "not-a-date", time.Time{}, false},
		{"empty", "", time.Time{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseESPNTime(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if ok && !got.Equal(tt.want) {
				t.Errorf("parseESPNTime(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestMapCurrentSituation_RealShape guards against a regression where the
// Situation dialog always showed "no data available" for live games. Root
// cause: /summary never populates header.competitions[0].situation (what the
// old mapSituation read) — confirmed live 2026-09-04 against three
// concurrent in-progress games. The real down-and-distance data lives in
// drives.current.plays[last].end instead; this fixture is a trimmed excerpt
// of that response (SJSU at E Michigan, event 401864495, 2026-09-04).
func TestMapCurrentSituation_RealShape(t *testing.T) {
	drivesJSON := `{
		"current": {
			"description": "0 plays, 0 yards, 0:00",
			"plays": [{
				"text": "(09:19) #37 N.Dibert kickoff 65 yards to the SJSU00, Touchback",
				"period": {"number": 2},
				"clock": {"displayValue": "9:19"},
				"start": {"down": 1, "distance": 10, "yardLine": 35, "yardsToEndzone": 65, "team": {"id": "2199"}},
				"end": {"down": 1, "distance": 10, "yardLine": 75, "downDistanceText": "1st & 10 at SJSU 25", "shortDownDistanceText": "1st & 10", "possessionText": "SJSU 25", "team": {"id": "23"}}
			}]
		}
	}`
	var drives rawDrives
	if err := json.Unmarshal([]byte(drivesJSON), &drives); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	home := rawCompetitor{ID: "23", TimeoutsUsed: 1}
	away := rawCompetitor{ID: "2199", TimeoutsUsed: 0}

	s := mapCurrentSituation(drives, home, away)
	if s == nil {
		t.Fatal("mapCurrentSituation returned nil, want a populated Situation")
	}
	if s.Down != 1 || s.Distance != 10 {
		t.Errorf("Down/Distance = %d/%d, want 1/10", s.Down, s.Distance)
	}
	if s.DownDistanceText != "1st & 10 at SJSU 25" {
		t.Errorf("DownDistanceText = %q", s.DownDistanceText)
	}
	if s.PossessionTeamID != 23 {
		t.Errorf("PossessionTeamID = %d, want 23", s.PossessionTeamID)
	}
	if s.YardsToEndzone != 25 {
		t.Errorf("YardsToEndzone = %d, want 25 (100 - yardLine 75)", s.YardsToEndzone)
	}
	if s.HomeTimeouts != 2 || s.AwayTimeouts != 3 {
		t.Errorf("HomeTimeouts/AwayTimeouts = %d/%d, want 2/3", s.HomeTimeouts, s.AwayTimeouts)
	}
}

func TestMapCurrentSituation_NilWhenNoCurrentDrive(t *testing.T) {
	if s := mapCurrentSituation(rawDrives{}, rawCompetitor{}, rawCompetitor{}); s != nil {
		t.Errorf("mapCurrentSituation() = %+v, want nil when there's no current drive", s)
	}
}

func TestMapCurrentSituation_NilWhenCurrentDriveHasNoPlaysYet(t *testing.T) {
	drives := rawDrives{Current: &rawDrive{Plays: nil}}
	if s := mapCurrentSituation(drives, rawCompetitor{}, rawCompetitor{}); s != nil {
		t.Errorf("mapCurrentSituation() = %+v, want nil right after a drive starts with no plays yet", s)
	}
}

func TestElapsedMinute(t *testing.T) {
	tests := []struct {
		period       int
		clockDisplay string
		want         int
	}{
		{1, "15:00", 0},  // kickoff, nothing elapsed
		{1, "0:00", 15},  // end of Q1
		{2, "9:19", 21},  // (2-1)*15 + (15-9) = 21, matches "Q2 9:19" shown live
		{4, "0:00", 60},  // end of regulation
	}
	for _, tt := range tests {
		got := elapsedMinute(tt.period, tt.clockDisplay)
		if got != tt.want {
			t.Errorf("elapsedMinute(%d, %q) = %d, want %d", tt.period, tt.clockDisplay, got, tt.want)
		}
	}
}
