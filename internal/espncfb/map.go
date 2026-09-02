package espncfb

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/0xjuanma/golazo/internal/api"
)

func toInt(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func toIntPtr(s string) *int {
	if s == "" {
		return nil
	}
	n := toInt(s)
	return &n
}

func mapTeam(t rawTeam) api.Team {
	name := t.DisplayName
	if name == "" {
		name = strings.TrimSpace(t.Location + " " + t.Name)
	}
	short := t.ShortDisplayName
	if short == "" {
		short = t.Abbreviation
	}
	return api.Team{
		ID:        toInt(t.ID),
		Name:      name,
		ShortName: short,
		Logo:      t.logoURL(),
	}
}

func mapStatus(s rawStatusType) api.MatchStatus {
	switch s.State {
	case "in":
		return api.MatchStatusLive
	case "post":
		return api.MatchStatusFinished
	default:
		return api.MatchStatusNotStarted
	}
}

// mapMatch converts one scoreboard event into an api.Match. Returns false if
// the event has no competitions (shouldn't happen in practice).
func mapMatch(e rawEvent) (api.Match, bool) {
	if len(e.Competitions) == 0 {
		return api.Match{}, false
	}
	comp := e.Competitions[0]

	var home, away rawCompetitor
	for _, c := range comp.Competitors {
		if c.HomeAway == "home" {
			home = c
		} else {
			away = c
		}
	}

	m := api.Match{
		ID:        toInt(e.ID),
		League:    conferenceByID(toInt(home.Team.ConferenceID)),
		HomeTeam:  mapTeam(home.Team),
		AwayTeam:  mapTeam(away.Team),
		Status:    mapStatus(comp.Status.Type),
		HomeScore: toIntPtr(home.Score),
		AwayScore: toIntPtr(away.Score),
	}

	if t, err := time.Parse(time.RFC3339, e.Date); err == nil {
		m.MatchTime = &t
	}

	if comp.Status.Type.State == "in" {
		lt := comp.Status.DisplayClock
		if comp.Status.Period > 0 {
			lt = "Q" + strconv.Itoa(comp.Status.Period) + " " + lt
		}
		m.LiveTime = &lt
	} else if comp.Status.Type.ShortDetail != "" {
		m.LiveTime = &comp.Status.Type.ShortDetail
	}

	return m, true
}

func mapSituation(s *rawSituation, homeTeamID, awayTeamID int) *api.Situation {
	if s == nil {
		return nil
	}
	return &api.Situation{
		Down:             s.Down,
		Distance:         s.Distance,
		YardLine:         s.YardLine,
		YardsToEndzone:   s.YardsToEndzone,
		PossessionTeamID: toInt(s.Possession),
		IsRedZone:        s.IsRedZone,
		DownDistanceText: s.DownDistanceText,
		PossessionText:   s.PossessionText,
		HomeTimeouts:     s.HomeTimeouts,
		AwayTimeouts:     s.AwayTimeouts,
		LastPlay:         lastPlayText(s.LastPlay),
	}
}

func lastPlayText(p *rawLastPlay) string {
	if p == nil {
		return ""
	}
	return p.Text
}

// mapStatistics zips two teams' boxscore.statistics (matched by Name) into
// the shared api.MatchStatistic shape the existing Statistics dialog already
// renders unmodified.
func mapStatistics(home, away rawBoxscoreTeam) []api.MatchStatistic {
	awayByName := make(map[string]rawStat, len(away.Statistics))
	for _, s := range away.Statistics {
		awayByName[s.Name] = s
	}

	stats := make([]api.MatchStatistic, 0, len(home.Statistics))
	for _, hs := range home.Statistics {
		as := awayByName[hs.Name]
		stats = append(stats, api.MatchStatistic{
			Key:       hs.Name,
			Label:     hs.Label,
			HomeValue: hs.DisplayValue,
			AwayValue: as.DisplayValue,
		})
	}
	return stats
}

func mapLeaders(raw []rawTeamLeaders, homeAbbr, awayAbbr string) []api.LeaderCategory {
	byCategory := map[string]*api.LeaderCategory{}
	var order []string

	for _, team := range raw {
		isHome := team.Team.Abbreviation == homeAbbr
		for _, cat := range team.Leaders {
			lc, ok := byCategory[cat.Name]
			if !ok {
				lc = &api.LeaderCategory{Key: cat.Name, Label: cat.DisplayName}
				byCategory[cat.Name] = lc
				order = append(order, cat.Name)
			}
			entries := make([]api.LeaderEntry, 0, len(cat.Leaders))
			for _, l := range cat.Leaders {
				entries = append(entries, api.LeaderEntry{
					PlayerName:   l.Athlete.DisplayName,
					DisplayValue: l.DisplayValue,
					Value:        l.Value,
				})
			}
			if isHome {
				lc.HomeLeaders = entries
			} else {
				lc.AwayLeaders = entries
			}
		}
	}

	out := make([]api.LeaderCategory, 0, len(order))
	for _, name := range order {
		out = append(out, *byCategory[name])
	}
	return out
}

// mapScoringPlays converts ESPN's scoringPlays feed into api.MatchEvent.
// Football scoring plays don't decompose into a clean Player/Assist pair
// (e.g. "Jayden Maiava 1 Yd Run (Caden Chittenden Kick)"), so the full text
// goes in Description instead.
func mapScoringPlays(raw []rawScoringPlay) []api.MatchEvent {
	events := make([]api.MatchEvent, 0, len(raw))
	for _, p := range raw {
		events = append(events, api.MatchEvent{
			ID:            toInt(p.ID),
			DisplayMinute: fmt.Sprintf("Q%d %s", p.Period.Number, p.Clock.DisplayValue),
			Type:          normalizeScoringPlayType(p.Type.Abbreviation),
			Team:          mapTeam(p.Team),
			Description:   p.Text,
		})
	}
	return events
}

func normalizeScoringPlayType(abbr string) string {
	switch abbr {
	case "TD":
		return "touchdown"
	case "FG":
		return "field_goal"
	case "SF":
		return "safety"
	default:
		return "score"
	}
}

func mapMomentum(raw []rawWinProb) []api.MomentumPoint {
	out := make([]api.MomentumPoint, 0, len(raw))
	for _, p := range raw {
		out = append(out, api.MomentumPoint{
			PlayID:     p.PlayID,
			HomeWinPct: p.HomeWinPercentage,
		})
	}
	return out
}

func mapRankingPoll(p rawPoll) api.RankingPoll {
	entries := make([]api.RankingEntry, 0, len(p.Ranks))
	for _, r := range p.Ranks {
		entries = append(entries, api.RankingEntry{
			Rank:            r.Current,
			PreviousRank:    r.Previous,
			Team:            mapTeam(r.Team),
			Points:          int(math.Round(r.Points)),
			FirstPlaceVotes: int(math.Round(r.FirstPlaceVotes)),
			Trend:           r.Trend,
		})
	}
	return api.RankingPoll{Name: p.Name, Entries: entries}
}

// standingsStat finds a stat by type ("total" = overall, "vsconf" = conference).
func standingsStat(stats []rawStandingsStat, statType string) string {
	for _, s := range stats {
		if s.Type == statType {
			return s.Summary
		}
	}
	return ""
}
