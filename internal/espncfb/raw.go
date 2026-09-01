// Package espncfb implements api.Client (and api.RankingsProvider) against
// ESPN's undocumented "site API" for college football. Field shapes below
// were captured from live responses on 2026-08-31 (scoreboard, summary,
// rankings endpoints) except where a comment says "unverified" — those were
// filled in from ESPN's well-known cross-sport conventions because no live
// game/response was available to confirm them at capture time. Confirm those
// against a real response before depending on them.
package espncfb

// rawScoreboard is the response of GET /college-football/scoreboard.
type rawScoreboard struct {
	Season struct {
		Type int `json:"type"`
		Year int `json:"year"`
	} `json:"season"`
	Week struct {
		Number int `json:"number"`
	} `json:"week"`
	Events []rawEvent `json:"events"`
}

type rawEvent struct {
	ID           string           `json:"id"`
	Date         string           `json:"date"` // RFC3339
	Name         string           `json:"name"`
	ShortName    string           `json:"shortName"`
	Competitions []rawCompetition `json:"competitions"`
}

type rawCompetition struct {
	ID          string          `json:"id"`
	Attendance  int             `json:"attendance"`
	NeutralSite bool            `json:"neutralSite"`
	Competitors []rawCompetitor `json:"competitors"`
	Status      rawStatus       `json:"status"`
	Situation   *rawSituation   `json:"situation,omitempty"`
	Venue       *rawVenue       `json:"venue,omitempty"`
}

type rawVenue struct {
	FullName string `json:"fullName"`
}

type rawCompetitor struct {
	ID          string          `json:"id"`
	HomeAway    string          `json:"homeAway"` // "home" | "away"
	Winner      bool            `json:"winner"`
	Team        rawTeam         `json:"team"`
	Score       string          `json:"score"`
	CuratedRank *rawCuratedRank `json:"curatedRank,omitempty"`
	Records     []rawRecord     `json:"records,omitempty"`
}

type rawCuratedRank struct {
	Current int `json:"current"`
}

type rawRecord struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "total" | "home" | "road" | "vsconf"
	Summary string `json:"summary"`
}

type rawTeam struct {
	ID               string `json:"id"`
	Location         string `json:"location"`
	Name             string `json:"name"`
	Abbreviation     string `json:"abbreviation"`
	DisplayName      string `json:"displayName"`
	ShortDisplayName string `json:"shortDisplayName"`
	Logo             string `json:"logo,omitempty"`
	Logos            []struct {
		Href string `json:"href"`
	} `json:"logos,omitempty"`
}

func (t rawTeam) logoURL() string {
	if t.Logo != "" {
		return t.Logo
	}
	if len(t.Logos) > 0 {
		return t.Logos[0].Href
	}
	return ""
}

type rawStatus struct {
	Clock        float64       `json:"clock"`
	DisplayClock string        `json:"displayClock"`
	Period       int           `json:"period"`
	Type         rawStatusType `json:"type"`
}

type rawStatusType struct {
	State       string `json:"state"` // "pre" | "in" | "post"
	Completed   bool   `json:"completed"`
	Description string `json:"description"`
	ShortDetail string `json:"shortDetail"`
}

// rawSituation is UNVERIFIED against a live response (no in-progress game was
// available at capture time). Field names follow ESPN's documented
// play.start/play.end shape (down/distance/yardLine/yardsToEndzone, confirmed
// live) plus the commonly-referenced situation fields from public ESPN API
// write-ups. Re-check this shape against a real in-progress game.
type rawSituation struct {
	Down                  int          `json:"down"`
	Distance              int          `json:"distance"`
	YardLine              int          `json:"yardLine"`
	YardsToEndzone        int          `json:"yardsToEndzone"`
	Possession            string       `json:"possession"` // team id
	IsRedZone             bool         `json:"isRedZone"`
	HomeTimeouts          int          `json:"homeTimeouts"`
	AwayTimeouts          int          `json:"awayTimeouts"`
	DownDistanceText      string       `json:"downDistanceText"`
	ShortDownDistanceText string       `json:"shortDownDistanceText"`
	PossessionText        string       `json:"possessionText"`
	LastPlay              *rawLastPlay `json:"lastPlay,omitempty"`
}

type rawLastPlay struct {
	Text string `json:"text"`
}

// rawSummary is the response of GET /college-football/summary?event={id}.
type rawSummary struct {
	Header         rawHeader        `json:"header"`
	Boxscore       rawBoxscore      `json:"boxscore"`
	Drives         rawDrives        `json:"drives"`
	Leaders        []rawTeamLeaders `json:"leaders"`
	WinProbability []rawWinProb     `json:"winprobability"`
	Standings      *rawStandings    `json:"standings,omitempty"`
}

// rawHeader carries the base game state (teams, score, status). Its
// Competitions[0].Situation field is UNVERIFIED — confirmed present on
// rawCompetition (scoreboard, live games only) but no live game was
// available to confirm ESPN also mirrors it onto the summary header.
type rawHeader struct {
	Competitions []rawHeaderCompetition `json:"competitions"`
}

type rawHeaderCompetition struct {
	Date        string          `json:"date"`
	Competitors []rawCompetitor `json:"competitors"`
	Status      rawStatus       `json:"status"`
	Situation   *rawSituation   `json:"situation,omitempty"`
}

type rawBoxscore struct {
	Teams []rawBoxscoreTeam `json:"teams"`
}

type rawBoxscoreTeam struct {
	Team       rawTeam   `json:"team"`
	HomeAway   string    `json:"homeAway"`
	Statistics []rawStat `json:"statistics"`
}

type rawStat struct {
	Name         string `json:"name"`
	DisplayValue string `json:"displayValue"`
	Label        string `json:"label"`
}

type rawDrives struct {
	Previous []rawDrive `json:"previous"`
}

type rawDrive struct {
	Description   string       `json:"description"`
	Result        string       `json:"result"`
	DisplayResult string       `json:"displayResult"`
	Yards         int          `json:"yards"`
	Team          rawDriveTeam `json:"team"`
	Plays         []rawPlay    `json:"plays"`
}

type rawDriveTeam struct {
	Abbreviation string `json:"abbreviation"`
}

type rawPlay struct {
	Text   string          `json:"text"`
	Period int             `json:"period"`
	Clock  rawClock        `json:"clock"`
	Start  rawPlayPosition `json:"start"`
	End    rawPlayPosition `json:"end"`
}

type rawClock struct {
	DisplayValue string `json:"displayValue"`
}

type rawPlayPosition struct {
	Down           int `json:"down"`
	Distance       int `json:"distance"`
	YardLine       int `json:"yardLine"`
	YardsToEndzone int `json:"yardsToEndzone"`
}

type rawTeamLeaders struct {
	Team    rawDriveTeam        `json:"team"`
	Leaders []rawLeaderCategory `json:"leaders"`
}

type rawLeaderCategory struct {
	Name        string           `json:"name"`
	DisplayName string           `json:"displayName"`
	Leaders     []rawLeaderEntry `json:"leaders"`
}

type rawLeaderEntry struct {
	DisplayValue string     `json:"displayValue"`
	Value        float64    `json:"value"`
	Athlete      rawAthlete `json:"athlete"`
}

type rawAthlete struct {
	DisplayName string `json:"displayName"`
}

type rawWinProb struct {
	PlayID            string  `json:"playId"`
	HomeWinPercentage float64 `json:"homeWinPercentage"`
}

// rawStandings mirrors summary.standings, which only covers the two
// conferences relevant to one match. LeagueTable uses the separate
// /apis/v2/.../standings?group={id} endpoint instead — see UNVERIFIED note
// on rawConferenceStandings in client.go.
type rawStandings struct {
	Groups []rawStandingsGroup `json:"groups"`
}

type rawStandingsGroup struct {
	Standings struct {
		Entries []rawStandingsEntry `json:"entries"`
	} `json:"standings"`
}

type rawStandingsEntry struct {
	Team  string             `json:"team"`
	ID    string             `json:"id"`
	Stats []rawStandingsStat `json:"stats"`
}

type rawStandingsStat struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // "total" (overall) | "vsconf" (conference)
	Summary      string `json:"summary"`
	DisplayValue string `json:"displayValue"`
}

// rawRankingsResponse is the response of GET /college-football/rankings.
type rawRankingsResponse struct {
	Rankings []rawPoll `json:"rankings"`
}

type rawPoll struct {
	Name  string    `json:"name"`
	Ranks []rawRank `json:"ranks"`
}

type rawRank struct {
	Current         int     `json:"current"`
	Previous        int     `json:"previous"`
	Points          int     `json:"points"`
	FirstPlaceVotes int     `json:"firstPlaceVotes"`
	Trend           string  `json:"trend"`
	Team            rawTeam `json:"team"`
}
