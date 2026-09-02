package api

import "time"

// League represents a football league
type League struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Country        string `json:"country"`
	CountryCode    string `json:"country_code"`
	Logo           string `json:"logo,omitempty"`
	ParentLeagueID int    `json:"parent_league_id,omitempty"` // Parent league ID for sub-season leagues (e.g., Liga MX Clausura -> Liga MX)
}

// Team represents a football team
type Team struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Logo      string `json:"logo,omitempty"`
}

// MatchStatus represents the status of a match
type MatchStatus string

const (
	MatchStatusNotStarted MatchStatus = "not_started"
	MatchStatusLive       MatchStatus = "live"
	MatchStatusFinished   MatchStatus = "finished"
	MatchStatusPostponed  MatchStatus = "postponed"
	MatchStatusCancelled  MatchStatus = "cancelled"
)

// Match represents a football match
type Match struct {
	ID        int         `json:"id"`
	League    League      `json:"league"`
	HomeTeam  Team        `json:"home_team"`
	AwayTeam  Team        `json:"away_team"`
	Status    MatchStatus `json:"status"`
	HomeScore *int        `json:"home_score,omitempty"`
	AwayScore *int        `json:"away_score,omitempty"`
	MatchTime *time.Time  `json:"match_time,omitempty"`
	LiveTime  *string     `json:"live_time,omitempty"` // e.g., "45+2", "HT", "FT"
	Round     string      `json:"round,omitempty"`
	PageURL   string      `json:"page_url,omitempty"` // FotMob match page slug (e.g., "/matches/team-vs-team/abc123")
}

// MatchEvent represents an event in a match (goal, card, substitution, a
// football scoring play, etc.)
type MatchEvent struct {
	ID            int       `json:"id"`
	Minute        int       `json:"minute"`                   // Base minute (e.g., 45)
	DisplayMinute string    `json:"display_minute,omitempty"` // Formatted minute with stoppage time (e.g., "45+2'"); football providers use this for "Q1 8:55" instead
	Type          string    `json:"type"`                     // "goal", "card", "substitution", "touchdown", "field_goal", "safety", etc.
	Team          Team      `json:"team"`
	Player        *string   `json:"player,omitempty"`
	Assist        *string   `json:"assist,omitempty"`
	EventType     *string   `json:"event_type,omitempty"` // "yellow", "red", "in", "out", etc.
	OwnGoal       *bool     `json:"own_goal,omitempty"`   // Indicates if this is an own goal
	Timestamp     time.Time `json:"timestamp"`

	// Description is a full human-readable play description (e.g. "Jayden
	// Maiava 1 Yd Run (Caden Chittenden Kick)"). Football scoring plays don't
	// decompose cleanly into Player/Assist, so providers without a clean
	// scorer field populate this instead of Player.
	Description string `json:"description,omitempty"`
}

// MatchStatistic represents a single match statistic (possession, shots, etc.)
type MatchStatistic struct {
	Key       string `json:"key"`        // e.g., "possession", "shots_total"
	Label     string `json:"label"`      // e.g., "Possession", "Total Shots"
	HomeValue string `json:"home_value"` // Value for home team
	AwayValue string `json:"away_value"` // Value for away team
}

// PlayerInfo represents basic player information for lineups
type PlayerInfo struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Number   int    `json:"number,omitempty"`
	Position string `json:"position,omitempty"`
	Rating   string `json:"rating,omitempty"` // Player rating (e.g., "7.2")
}

// MatchDetails contains detailed information about a match
type MatchDetails struct {
	Match
	Events     []MatchEvent `json:"events"`
	HomeLineup []string     `json:"home_lineup,omitempty"`
	AwayLineup []string     `json:"away_lineup,omitempty"`

	// Additional match information
	HalfTimeScore *struct {
		Home *int `json:"home,omitempty"`
		Away *int `json:"away,omitempty"`
	} `json:"half_time_score,omitempty"`
	Venue         string  `json:"venue,omitempty"`          // Stadium name
	Winner        *string `json:"winner,omitempty"`         // "home" or "away"
	MatchDuration int     `json:"match_duration,omitempty"` // 90, 120, etc.
	ExtraTime     bool    `json:"extra_time,omitempty"`     // If match went to extra time
	Penalties     *struct {
		Home *int `json:"home,omitempty"`
		Away *int `json:"away,omitempty"`
	} `json:"penalties,omitempty"`

	// Extended statistics
	Statistics []MatchStatistic `json:"statistics,omitempty"` // Match statistics (possession, shots, etc.)

	// Match context
	Referee    string `json:"referee,omitempty"`    // Referee name
	Attendance int    `json:"attendance,omitempty"` // Stadium attendance

	// Team formations
	HomeFormation string `json:"home_formation,omitempty"` // e.g., "4-3-3"
	AwayFormation string `json:"away_formation,omitempty"` // e.g., "4-4-2"

	// Starting lineups with full details
	HomeStarting []PlayerInfo `json:"home_starting,omitempty"`
	AwayStarting []PlayerInfo `json:"away_starting,omitempty"`

	// Substitutes
	HomeSubstitutes []PlayerInfo `json:"home_substitutes,omitempty"`
	AwaySubstitutes []PlayerInfo `json:"away_substitutes,omitempty"`

	// Momentum/xG data (if available)
	HomeXG *float64 `json:"home_xg,omitempty"` // Expected goals for home team
	AwayXG *float64 `json:"away_xg,omitempty"` // Expected goals for away team

	// Highlight video (if available)
	Highlight *MatchHighlight `json:"highlight,omitempty"` // Official highlight video link

	// Aggregate score (two-legged knockout ties only)
	AggregateScore     string `json:"aggregate_score,omitempty"`       // e.g. "5 - 7"
	WhoLostOnAggregate string `json:"who_lost_on_aggregate,omitempty"` // team name eliminated on aggregate

	// Current down-and-distance / field position (American football; nil for sports without the concept)
	Situation *Situation `json:"situation,omitempty"`

	// Player statistical leaders, grouped by category (e.g. passing/rushing/receiving)
	Leaders []LeaderCategory `json:"leaders,omitempty"`

	// Win-probability series across the match, one point per play (American football momentum)
	Momentum []MomentumPoint `json:"momentum,omitempty"`
}

// Situation represents the current down-and-distance and field position of a match
// (American football and similar sports; not applicable to sports without downs).
type Situation struct {
	Down             int    `json:"down"`             // 1-4
	Distance         int    `json:"distance"`         // yards needed for a first down
	YardLine         int    `json:"yard_line"`        // 0-100, distance from the possessing team's own goal line
	YardsToEndzone   int    `json:"yards_to_endzone"` // yards from the current spot to the opponent's end zone
	PossessionTeamID int    `json:"possession_team_id"`
	IsRedZone        bool   `json:"is_red_zone,omitempty"`
	DownDistanceText string `json:"down_distance_text,omitempty"` // e.g. "2nd & 6"
	PossessionText   string `json:"possession_text,omitempty"`    // e.g. "USC 44"
	HomeTimeouts     int    `json:"home_timeouts,omitempty"`
	AwayTimeouts     int    `json:"away_timeouts,omitempty"`
	LastPlay         string `json:"last_play,omitempty"`
}

// LeaderEntry is a single player's line in a statistical leaders category.
type LeaderEntry struct {
	PlayerName   string  `json:"player_name"`
	DisplayValue string  `json:"display_value"` // e.g. "25/29, 286 YDS, 2 TD"
	Value        float64 `json:"value"`         // sortable underlying value
}

// LeaderCategory groups home/away statistical leaders for one stat category
// (e.g. passing yards, rushing yards, receiving yards).
type LeaderCategory struct {
	Key         string        `json:"key"`   // e.g. "passingYards"
	Label       string        `json:"label"` // e.g. "Passing Yards"
	HomeLeaders []LeaderEntry `json:"home_leaders,omitempty"`
	AwayLeaders []LeaderEntry `json:"away_leaders,omitempty"`
}

// MomentumPoint is one sample of a win-probability series (one per play).
type MomentumPoint struct {
	PlayID     string  `json:"play_id"`
	HomeWinPct float64 `json:"home_win_pct"` // 0.0-1.0
	Period     int     `json:"period,omitempty"`
}

// RankingEntry represents one team's position in a poll (e.g. AP Top 25).
type RankingEntry struct {
	Rank            int    `json:"rank"`
	PreviousRank    int    `json:"previous_rank,omitempty"`
	Team            Team   `json:"team"`
	Record          string `json:"record,omitempty"` // e.g. "1-0"
	Points          int    `json:"points,omitempty"`
	FirstPlaceVotes int    `json:"first_place_votes,omitempty"`
	Trend           string `json:"trend,omitempty"` // e.g. "+1", "-1", "-" (no change)
}

// RankingPoll is a named poll (e.g. "AP Top 25", "AFCA Coaches Poll") and its ranked entries.
type RankingPoll struct {
	Name    string         `json:"name"`
	Entries []RankingEntry `json:"entries"`
}

// MatchHighlight represents an official highlight video for a match
type MatchHighlight struct {
	URL    string `json:"url"`              // Direct link to highlight video
	Image  string `json:"image,omitempty"`  // Thumbnail image URL
	Source string `json:"source,omitempty"` // Video source (e.g., "www.youtube.com")
	Title  string `json:"title,omitempty"`  // Video title (optional)
}

// LeagueTableEntry represents a team's position in the league table.
//
// Soccer providers populate the W/D/L/goals/points fields. American-football
// providers (which have no draws or points system) instead populate
// ConferenceRecord/OverallRecord as "W-L" strings and leave the rest zero;
// the standings dialog switches rendering based on which set is present.
type LeagueTableEntry struct {
	Position         int    `json:"position"`
	Team             Team   `json:"team"`
	Played           int    `json:"played"`
	Won              int    `json:"won"`
	Drawn            int    `json:"drawn"`
	Lost             int    `json:"lost"`
	GoalsFor         int    `json:"goals_for"`
	GoalsAgainst     int    `json:"goals_against"`
	GoalDifference   int    `json:"goal_difference"`
	Points           int    `json:"points"`
	ConferenceRecord string `json:"conference_record,omitempty"` // e.g. "5-2" (American football)
	OverallRecord    string `json:"overall_record,omitempty"`    // e.g. "8-3" (American football)
}
