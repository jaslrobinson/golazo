package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xjuanma/golazo/internal/api"
)

// fakeClient is a minimal api.Client for testing commands.go's fetch
// functions without hitting a real provider.
type fakeClient struct {
	matches       []api.Match
	matchDetails  *api.MatchDetails
	standings     []api.LeagueTableEntry
	err           error
	matchesByDate func(date time.Time) ([]api.Match, error) // optional override
}

func (f *fakeClient) MatchesByDate(ctx context.Context, date time.Time) ([]api.Match, error) {
	if f.matchesByDate != nil {
		return f.matchesByDate(date)
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.matches, nil
}

func (f *fakeClient) MatchDetails(ctx context.Context, matchID int) (*api.MatchDetails, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.matchDetails, nil
}

func (f *fakeClient) Leagues(ctx context.Context) ([]api.League, error) { return nil, nil }

func (f *fakeClient) LeagueMatches(ctx context.Context, leagueID int) ([]api.Match, error) {
	return f.matches, f.err
}

func (f *fakeClient) LeagueTable(ctx context.Context, leagueID int, leagueName string) ([]api.LeagueTableEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.standings, nil
}

func TestSplitByStatus(t *testing.T) {
	matches := []api.Match{
		{ID: 1, Status: api.MatchStatusLive},
		{ID: 2, Status: api.MatchStatusNotStarted},
		{ID: 3, Status: api.MatchStatusFinished},
		{ID: 4, Status: api.MatchStatusLive},
	}

	live, upcoming := splitByStatus(matches)
	if len(live) != 2 || live[0].ID != 1 || live[1].ID != 4 {
		t.Errorf("live matches wrong: %+v", live)
	}
	if len(upcoming) != 1 || upcoming[0].ID != 2 {
		t.Errorf("upcoming matches wrong: %+v", upcoming)
	}
}

func TestFetchLiveBatchData_SingleCallAlwaysLast(t *testing.T) {
	client := &fakeClient{
		matches: []api.Match{
			{ID: 1, Status: api.MatchStatusLive},
			{ID: 2, Status: api.MatchStatusNotStarted},
		},
	}

	cmd := fetchLiveBatchData(context.Background(), client, false, 0)
	msg, ok := cmd().(liveBatchDataMsg)
	if !ok {
		t.Fatalf("expected liveBatchDataMsg, got %T", cmd())
	}
	if !msg.isLast {
		t.Error("expected isLast=true on the very first call — ESPN has nothing to batch")
	}
	if len(msg.matches) != 1 || msg.matches[0].ID != 1 {
		t.Errorf("live matches wrong: %+v", msg.matches)
	}
	if len(msg.upcoming) != 1 || msg.upcoming[0].ID != 2 {
		t.Errorf("upcoming matches wrong: %+v", msg.upcoming)
	}
}

func TestFetchLiveBatchData_ClientError(t *testing.T) {
	client := &fakeClient{err: errors.New("boom")}
	cmd := fetchLiveBatchData(context.Background(), client, false, 0)
	msg, ok := cmd().(liveBatchDataMsg)
	if !ok {
		t.Fatalf("expected liveBatchDataMsg, got %T", cmd())
	}
	if !msg.isLast {
		t.Error("expected isLast=true even on error, so the loader doesn't hang")
	}
	if msg.err == nil {
		t.Error("expected err to be propagated")
	}
}

func TestFetchLiveBatchData_NilClient(t *testing.T) {
	cmd := fetchLiveBatchData(context.Background(), nil, false, 0)
	msg, ok := cmd().(liveBatchDataMsg)
	if !ok {
		t.Fatalf("expected liveBatchDataMsg, got %T", cmd())
	}
	if !msg.isLast {
		t.Error("expected isLast=true for nil client")
	}
}

func TestFetchStatsDayData_UsesPlainMatchesByDate(t *testing.T) {
	var gotDate time.Time
	client := &fakeClient{
		matchesByDate: func(date time.Time) ([]api.Match, error) {
			gotDate = date
			return []api.Match{
				{ID: 1, Status: api.MatchStatusFinished},
				{ID: 2, Status: api.MatchStatusNotStarted},
			}, nil
		},
	}

	cmd := fetchStatsDayData(context.Background(), client, false, 0, 5)
	msg, ok := cmd().(statsDayDataMsg)
	if !ok {
		t.Fatalf("expected statsDayDataMsg, got %T", cmd())
	}
	if !msg.isToday {
		t.Error("dayIndex=0 should be isToday=true")
	}
	if msg.isLast {
		t.Error("dayIndex=0 of 5 total days should not be isLast")
	}
	if len(msg.finished) != 1 || msg.finished[0].ID != 1 {
		t.Errorf("finished matches wrong: %+v", msg.finished)
	}
	if len(msg.upcoming) != 1 || msg.upcoming[0].ID != 2 {
		t.Errorf("upcoming matches wrong: %+v", msg.upcoming)
	}
	if gotDate.IsZero() {
		t.Error("expected a real date to be passed to MatchesByDate")
	}
}

func TestFetchStatsDayData_PastDayHasNoUpcoming(t *testing.T) {
	client := &fakeClient{
		matches: []api.Match{
			{ID: 1, Status: api.MatchStatusFinished},
			{ID: 2, Status: api.MatchStatusNotStarted}, // should be dropped: not today
		},
	}

	cmd := fetchStatsDayData(context.Background(), client, false, 4, 5)
	msg, ok := cmd().(statsDayDataMsg)
	if !ok {
		t.Fatalf("expected statsDayDataMsg, got %T", cmd())
	}
	if msg.isToday {
		t.Error("dayIndex=4 should not be isToday")
	}
	if !msg.isLast {
		t.Error("dayIndex=4 of 5 total days (0-4) should be isLast")
	}
	if len(msg.upcoming) != 0 {
		t.Errorf("past days should never report upcoming matches, got %+v", msg.upcoming)
	}
}

func TestFetchStandings_MapsToStandingsMsg(t *testing.T) {
	client := &fakeClient{
		standings: []api.LeagueTableEntry{
			{Position: 1, Team: api.Team{Name: "Georgia"}, ConferenceRecord: "1-0"},
		},
	}

	cmd := fetchStandings(client, 8, "SEC", 100, 200)
	msg, ok := cmd().(standingsMsg)
	if !ok {
		t.Fatalf("expected standingsMsg, got %T", cmd())
	}
	if msg.leagueID != 8 || msg.leagueName != "SEC" {
		t.Errorf("league fields wrong: %+v", msg)
	}
	if msg.homeTeamID != 100 || msg.awayTeamID != 200 {
		t.Errorf("team ID passthrough wrong: %+v", msg)
	}
	if len(msg.standings) != 1 || msg.standings[0].Team.Name != "Georgia" {
		t.Errorf("standings wrong: %+v", msg.standings)
	}
}

func TestFetchMatchDetails_NilClientDoesNotPanic(t *testing.T) {
	// fetchMatchDetails has no nil-client guard (unlike the others) since
	// model.go always constructs a real client — this documents that
	// assumption rather than silently relying on it.
	details := &api.MatchDetails{Match: api.Match{ID: 42}}
	client := &fakeClient{matchDetails: details}

	cmd := fetchMatchDetails(client, 42, false)
	msg, ok := cmd().(matchDetailsMsg)
	if !ok {
		t.Fatalf("expected matchDetailsMsg, got %T", cmd())
	}
	if msg.details == nil || msg.details.ID != 42 {
		t.Errorf("details wrong: %+v", msg.details)
	}
}
