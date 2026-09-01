package api

import (
	"context"
	"time"
)

// Client defines the interface for a football API client.
// This abstraction allows us to swap implementations (FotMob, other APIs, mock, etc.)
type Client interface {
	// MatchesByDate retrieves all matches for a specific date.
	MatchesByDate(ctx context.Context, date time.Time) ([]Match, error)

	// MatchDetails retrieves detailed information about a specific match.
	MatchDetails(ctx context.Context, matchID int) (*MatchDetails, error)

	// Leagues retrieves available leagues.
	Leagues(ctx context.Context) ([]League, error)

	// LeagueMatches retrieves matches for a specific league.
	LeagueMatches(ctx context.Context, leagueID int) ([]Match, error)

	// LeagueTable retrieves the league table/standings for a specific league.
	// leagueName is used to detect parent leagues for knockout competitions.
	LeagueTable(ctx context.Context, leagueID int, leagueName string) ([]LeagueTableEntry, error)
}

// RankingsProvider is implemented by clients whose sport has ranking polls with
// no equivalent in the base Client interface (e.g. AP Top 25 / Coaches Poll for
// American football). Callers should type-assert an api.Client against this
// interface before offering a rankings view.
type RankingsProvider interface {
	// Rankings retrieves all available polls (e.g. "AP Top 25", "AFCA Coaches Poll").
	Rankings(ctx context.Context) ([]RankingPoll, error)
}
