package espncfb

import (
	"encoding/json"
	"testing"
)

// These fixtures are trimmed excerpts of real responses captured from
// site.api.espn.com on 2026-08-31 (San José State at USC, event 401864494,
// and the concurrent Week 1 scoreboard/rankings). They exist to catch a
// mismatched json tag that a compile-clean-but-untested mapping wouldn't.

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
