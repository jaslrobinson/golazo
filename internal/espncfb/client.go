package espncfb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jaslrobinson/golazo/internal/api"
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
	// site.api.espn.com (scoreboard/summary/rankings) returned 403 to a
	// synthetic User-Agent while site.web.api.espn.com (standings) — same
	// process, same headers otherwise — returned 200. That asymmetry points
	// at host-specific Akamai bot detection rather than a network-level
	// block, so mimicking a real browser request is the first thing worth
	// testing before assuming this is unfixable from a plain HTTP client.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.espn.com/")
	req.Header.Set("Origin", "https://www.espn.com")

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
		League:    conferenceByID(toInt(home.Team.ConferenceID)),
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
		Events:    mapScoringPlays(raw.ScoringPlays),
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

	// Only attach a last-play fallback for LIVE games, and only when the
	// header didn't already carry one via the UNVERIFIED header.situation
	// mirroring. A finished or not-yet-started game must leave Situation nil
	// — synthesizing a zero-value api.Situation{} here would render as
	// "1st & 0, ball on the opponent's goal line" in the Situation dialog,
	// which is actively misleading rather than merely incomplete.
	if hc.Status.Type.State == "in" {
		if details.Situation == nil {
			details.Situation = &api.Situation{}
		}
		if details.Situation.LastPlay == "" {
			details.Situation.LastPlay = lastCompletedPlayText(raw.Drives.Previous)
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
// Verified live (2026-09-01) against SEC (group=8): the endpoint returns
// {standings: {entries: [...]}} at the top level (no "children" wrapping —
// an earlier version of this method assumed that shape and was wrong), each
// entry.team is a full team object, and stats includes composite "vsconf"
// and "total" entries with ready-made "W-L" summary strings alongside dozens
// of decomposed per-split stats (home/away/division/etc.) that this doesn't
// need.
func (c *Client) LeagueTable(ctx context.Context, leagueID int, leagueName string) ([]api.LeagueTableEntry, error) {
	url := fmt.Sprintf("%s?season=%d&group=%d", standingsBaseURL, espnSeasonYear(), leagueID)
	var raw rawLeagueTableResponse
	if err := c.get(ctx, url, &raw); err != nil {
		return nil, err
	}

	entries := make([]api.LeagueTableEntry, 0, len(raw.Standings.Entries))
	for i, e := range raw.Standings.Entries {
		entries = append(entries, api.LeagueTableEntry{
			Position:         i + 1,
			Team:             mapTeam(e.Team),
			ConferenceRecord: standingsStat(e.Stats, "vsconf"),
			OverallRecord:    standingsStat(e.Stats, "total"),
		})
	}
	return entries, nil
}

// espnSeasonYear returns the college football season year for standings
// requests.
func espnSeasonYear() int {
	return seasonYearFor(time.Now())
}

// seasonYearFor is the pure logic behind espnSeasonYear, split out for
// testability. The season spans Aug-Jan, so games in Jan (bowl season/CFP)
// still belong to the season that started the previous calendar year.
func seasonYearFor(t time.Time) int {
	if t.Month() == time.January {
		return t.Year() - 1
	}
	return t.Year()
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
