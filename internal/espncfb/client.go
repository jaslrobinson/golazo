package espncfb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/0xjuanma/golazo/internal/api"
)

const (
	siteBaseURL = "https://site.api.espn.com/apis/site/v2/sports/football/college-football"

	// UNVERIFIED: different host than the rest of this client. Captured from
	// ESPN's well-known cross-sport standings convention, not a live
	// response for college football specifically. Confirm the host, path,
	// and `group` query param before relying on LeagueTable.
	standingsBaseURL = "https://site.web.api.espn.com/apis/v2/sports/football/college-football/standings"
)

// Client implements api.Client and api.RankingsProvider against ESPN's
// undocumented site API for college football.
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 10 * time.Second}}
}

var (
	_ api.Client           = (*Client)(nil)
	_ api.RankingsProvider = (*Client)(nil)
)

func (c *Client) get(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; golazo-cfb/0.1)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("espncfb: GET %s: status %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// MatchesByDate retrieves all FBS games for a specific date.
func (c *Client) MatchesByDate(ctx context.Context, date time.Time) ([]api.Match, error) {
	url := fmt.Sprintf("%s/scoreboard?dates=%s&limit=200", siteBaseURL, date.Format("20060102"))
	var raw rawScoreboard
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	return mapEvents(raw.Events), nil
}

// LeagueMatches retrieves matches for one conference (ESPN "group").
//
// UNVERIFIED: the `groups` scoreboard filter was not exercised against a
// live response during capture (only the unfiltered scoreboard was). It is
// ESPN's standard convention across other sports' site APIs, but confirm it
// returns the expected subset before relying on it.
func (c *Client) LeagueMatches(ctx context.Context, leagueID int) ([]api.Match, error) {
	url := fmt.Sprintf("%s/scoreboard?groups=%d&limit=200", siteBaseURL, leagueID)
	var raw rawScoreboard
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	return mapEvents(raw.Events), nil
}

func mapEvents(events []rawEvent) []api.Match {
	matches := make([]api.Match, 0, len(events))
	for _, e := range events {
		if m, ok := mapMatch(e); ok {
			matches = append(matches, m)
		}
	}
	return matches
}

// Leagues returns the hardcoded FBS conference table (see conferences.go).
func (c *Client) Leagues(ctx context.Context) ([]api.League, error) {
	return fbsConferences, nil
}

// MatchDetails retrieves the full box score, situation, leaders, and
// win-probability series for one game.
func (c *Client) MatchDetails(ctx context.Context, matchID int) (*api.MatchDetails, error) {
	url := fmt.Sprintf("%s/summary?event=%d", siteBaseURL, matchID)
	var raw rawSummary
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	if len(raw.Header.Competitions) == 0 {
		return nil, fmt.Errorf("espncfb: summary for event %d has no competitions", matchID)
	}
	hc := raw.Header.Competitions[0]

	var home, away rawCompetitor
	for _, comp := range hc.Competitors {
		if comp.HomeAway == "home" {
			home = comp
		} else {
			away = comp
		}
	}

	base := api.Match{
		ID:        matchID,
		HomeTeam:  mapTeam(home.Team),
		AwayTeam:  mapTeam(away.Team),
		Status:    mapStatus(hc.Status.Type),
		HomeScore: toIntPtr(home.Score),
		AwayScore: toIntPtr(away.Score),
	}
	if t, err := time.Parse(time.RFC3339, hc.Date); err == nil {
		base.MatchTime = &t
	}

	details := &api.MatchDetails{
		Match:     base,
		Situation: mapSituation(hc.Situation, base.HomeTeam.ID, base.AwayTeam.ID),
		Leaders:   mapLeaders(raw.Leaders, home.Team.Abbreviation, away.Team.Abbreviation),
		Momentum:  mapMomentum(raw.WinProbability),
	}

	if len(raw.Boxscore.Teams) == 2 {
		var homeBox, awayBox rawBoxscoreTeam
		for _, t := range raw.Boxscore.Teams {
			if t.HomeAway == "home" {
				homeBox = t
			} else {
				awayBox = t
			}
		}
		details.Statistics = mapStatistics(homeBox, awayBox)
	}

	// Fall back to the last completed play's text when the header didn't
	// carry a live situation (e.g. finished games, or if the UNVERIFIED
	// header.situation mirroring doesn't hold).
	if details.Situation == nil || details.Situation.LastPlay == "" {
		if lp := lastCompletedPlayText(raw.Drives.Previous); lp != "" {
			if details.Situation == nil {
				details.Situation = &api.Situation{}
			}
			details.Situation.LastPlay = lp
		}
	}

	return details, nil
}

func lastCompletedPlayText(drives []rawDrive) string {
	if len(drives) == 0 {
		return ""
	}
	last := drives[len(drives)-1]
	if len(last.Plays) == 0 {
		return ""
	}
	return last.Plays[len(last.Plays)-1].Text
}

// LeagueTable retrieves conference standings.
//
// UNVERIFIED end-to-end: hits a different host (site.web.api.espn.com) whose
// response shape was not captured live for college football. Confirm before
// shipping — this is a best-effort implementation based on ESPN's
// documented cross-sport standings convention.
func (c *Client) LeagueTable(ctx context.Context, leagueID int, leagueName string) ([]api.LeagueTableEntry, error) {
	url := fmt.Sprintf("%s?season=%d&group=%d", standingsBaseURL, time.Now().Year(), leagueID)
	var raw struct {
		Children []struct {
			Standings struct {
				Entries []rawStandingsEntry `json:"entries"`
			} `json:"standings"`
		} `json:"children"`
	}
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}

	var entries []api.LeagueTableEntry
	pos := 1
	for _, child := range raw.Children {
		for _, e := range child.Standings.Entries {
			entries = append(entries, api.LeagueTableEntry{
				Position:         pos,
				Team:             api.Team{ID: toInt(e.ID), Name: e.Team, ShortName: e.Team},
				ConferenceRecord: standingsStat(e.Stats, "vsconf"),
				OverallRecord:    standingsStat(e.Stats, "total"),
			})
			pos++
		}
	}
	return entries, nil
}

// Rankings retrieves the current AP Top 25, Coaches Poll, and other
// available polls.
func (c *Client) Rankings(ctx context.Context) ([]api.RankingPoll, error) {
	url := siteBaseURL + "/rankings"
	var raw rawRankingsResponse
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}
	polls := make([]api.RankingPoll, 0, len(raw.Rankings))
	for _, p := range raw.Rankings {
		polls = append(polls, mapRankingPoll(p))
	}
	return polls, nil
}
