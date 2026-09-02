package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/data"
)

// TestWriteFinishedResult_SuccessAfterDeadlineExpiryIsNotReportedAsTimeout
// reproduces a race that's real against ESPN even though every individual
// fetch succeeds: collectFinished can still be mid-flight (or just finish)
// after ctx's deadline has technically elapsed, because it makes `days`
// sequential HTTP calls against one shared budget. Unlike FotMob's
// goroutine aggregator (which could swallow cancellation and return
// empty-with-nil-error, the scenario the old post-hoc isTimeout(ctx) guard
// was written to catch), collectFinished only returns a nil error when
// every day genuinely completed — so a ctx that happens to be expired by
// the time the caller checks it afterward must not override that success.
func TestWriteFinishedResult_SuccessAfterDeadlineExpiryIsNotReportedAsTimeout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	matches := []api.Match{{ID: 1, Status: api.MatchStatusFinished}}
	code := writeFinishedResult(&stdout, &stderr, matches, nil, nil, true /* timedOut, but collectErr is nil */)

	if code != ExitOK {
		t.Fatalf("code = %d, want %d (stderr=%s)", code, ExitOK, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr not empty on success: %s", stderr.String())
	}
	var env struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal stdout: %v\nraw: %s", err, stdout.String())
	}
	if env.Count != 1 {
		t.Errorf("count = %d, want 1", env.Count)
	}
}

func TestWriteFinishedResult_CollectErrIsClassifiedAndTimedOutIsHonored(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := writeFinishedResult(&stdout, &stderr, nil, []string{"2026-09-01"}, errors.New("all days failed"), true)

	if code != ExitTimeout {
		t.Fatalf("code = %d, want %d", code, ExitTimeout)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty on error, got: %s", stdout.String())
	}
}

func TestWriteFinishedResult_DegradedWhenFailedDatesPresent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	matches := []api.Match{{ID: 1, Status: api.MatchStatusFinished}}
	code := writeFinishedResult(&stdout, &stderr, matches, []string{"2026-09-01"}, nil, false)

	if code != ExitOK {
		t.Fatalf("code = %d, want %d (stderr=%s)", code, ExitOK, stderr.String())
	}
	var env struct {
		Degraded bool `json:"degraded"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal stdout: %v\nraw: %s", err, stdout.String())
	}
	if !env.Degraded {
		t.Errorf("expected degraded:true in output: %s", stdout.String())
	}
}

func TestRunFinished_MockReturnsExpectedCount(t *testing.T) {
	t.Setenv(EnvOffline, "")
	t.Setenv(EnvAgent, "")

	var stdout, stderr bytes.Buffer
	code := runFinished(&stdout, &stderr, finishedFlags{cliFlags: cliFlags{mock: true, timeout: time.Second}, days: 1})

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d. stderr=%s", code, ExitOK, stderr.String())
	}
	var env struct {
		Status string      `json:"status"`
		Count  int         `json:"count"`
		Data   []api.Match `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, stdout.String())
	}
	if env.Count != len(data.MockFinishedMatches()) {
		t.Errorf("count = %d, want %d", env.Count, len(data.MockFinishedMatches()))
	}
}

func TestRunFinished_InvalidDays(t *testing.T) {
	cases := []int{0, -1, 9, 100}
	for _, days := range cases {
		t.Run("", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := runFinished(&stdout, &stderr, finishedFlags{cliFlags: cliFlags{mock: true, timeout: time.Second}, days: days})
			if code != ExitInvalidArgs {
				t.Errorf("days=%d: exit = %d, want %d", days, code, ExitInvalidArgs)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout should be empty on invalid args, got: %s", stdout.String())
			}
			var env errEnvelope
			if err := json.Unmarshal(stderr.Bytes(), &env); err != nil {
				t.Fatalf("unmarshal stderr: %v", err)
			}
			if env.Code != ErrCodeInvalidArgs {
				t.Errorf("code = %q, want %q", env.Code, ErrCodeInvalidArgs)
			}
		})
	}
}

func TestCollectFinished_DedupsByID(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	calls := 0
	fetch := func(ctx context.Context, date time.Time) ([]api.Match, error) {
		calls++
		// Return the same finished match each day; expect dedup to keep it once.
		return []api.Match{
			{ID: 100, Status: api.MatchStatusFinished},
			{ID: 101, Status: api.MatchStatusFinished},
			{ID: 200, Status: api.MatchStatusNotStarted}, // must be filtered
		}, nil
	}

	matches, failed, err := collectFinished(context.Background(), fetch, now, 3, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if len(failed) != 0 {
		t.Errorf("failedDates = %v, want empty", failed)
	}
	if calls != 3 {
		t.Errorf("fetch called %d times, want 3", calls)
	}
	if len(matches) != 2 {
		t.Errorf("returned %d matches, want 2 (dedup keeps 100,101; drops 200)", len(matches))
	}
	for _, m := range matches {
		if m.Status != api.MatchStatusFinished {
			t.Errorf("non-finished match leaked: %+v", m)
		}
	}
}

func TestCollectFinished_PartialFailureFlagsDegraded(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	callCount := 0
	fetch := func(ctx context.Context, date time.Time) ([]api.Match, error) {
		callCount++
		if callCount == 2 {
			return nil, errors.New("upstream blew up")
		}
		return []api.Match{
			{ID: callCount, Status: api.MatchStatusFinished},
		}, nil
	}

	matches, failed, err := collectFinished(context.Background(), fetch, now, 3, false)
	if err != nil {
		t.Fatalf("partial-success should not return err, got %v", err)
	}
	if len(failed) != 1 {
		t.Errorf("failedDates = %v, want 1 entry", failed)
	}
	if len(matches) != 2 {
		t.Errorf("matches len = %d, want 2", len(matches))
	}
}

func TestCollectFinished_AllFailureReturnsError(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	fetch := func(ctx context.Context, date time.Time) ([]api.Match, error) {
		return nil, errors.New("nope")
	}

	matches, failed, err := collectFinished(context.Background(), fetch, now, 2, false)
	if err == nil {
		t.Fatalf("expected err when all days fail")
	}
	if matches != nil {
		t.Errorf("matches not nil on total failure: %v", matches)
	}
	if len(failed) != 2 {
		t.Errorf("failedDates = %v, want 2", failed)
	}
}

func TestCollectFinished_FetchesOneCallPerDay(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	var gotDates []time.Time
	fetch := func(ctx context.Context, date time.Time) ([]api.Match, error) {
		gotDates = append(gotDates, date)
		return nil, nil
	}

	_, _, _ = collectFinished(context.Background(), fetch, now, 2, false)
	if len(gotDates) != 2 {
		t.Fatalf("expected 2 fetches (one per day), got %d", len(gotDates))
	}
	if !gotDates[0].Equal(now) {
		t.Errorf("day 0 = %v, want today (%v)", gotDates[0], now)
	}
	wantYesterday := now.AddDate(0, 0, -1)
	if !gotDates[1].Equal(wantYesterday) {
		t.Errorf("day 1 = %v, want yesterday (%v)", gotDates[1], wantYesterday)
	}
}

func TestRunFinished_TimeoutNotSwallowed(t *testing.T) {
	t.Setenv(EnvOffline, "")
	t.Setenv(EnvAgent, "")

	// 1ns timeout — collectFinished may swallow per-day failures and return
	// no error. The CLI must still report timeout to the agent.
	var stdout, stderr bytes.Buffer
	code := runFinished(&stdout, &stderr, finishedFlags{cliFlags: cliFlags{mock: false, timeout: 1}, days: 1})
	if code != ExitTimeout {
		t.Errorf("exit = %d, want %d (stderr=%s)", code, ExitTimeout, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty on timeout, got: %s", stdout.String())
	}
}

func TestCollectFinished_IncludeUpcomingTodayOnly(t *testing.T) {
	now := time.Date(2026, 6, 12, 12, 0, 0, 0, time.UTC)
	fetch := func(ctx context.Context, date time.Time) ([]api.Match, error) {
		// Same payload returned for every day: one finished, one not_started.
		return []api.Match{
			{ID: int(date.Unix()) + 1, Status: api.MatchStatusFinished},
			{ID: int(date.Unix()) + 2, Status: api.MatchStatusNotStarted},
		}, nil
	}

	// includeUpcoming=false → not_started filtered everywhere
	matches, _, err := collectFinished(context.Background(), fetch, now, 2, false)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, m := range matches {
		if m.Status != api.MatchStatusFinished {
			t.Errorf("include=false leaked status=%q match=%+v", m.Status, m)
		}
	}
	if len(matches) != 2 {
		t.Errorf("include=false count = %d, want 2 (finished from each day)", len(matches))
	}

	// includeUpcoming=true → today's not_started included, past-day not_started still filtered
	matches, _, err = collectFinished(context.Background(), fetch, now, 2, true)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	statusCounts := map[api.MatchStatus]int{}
	for _, m := range matches {
		statusCounts[m.Status]++
	}
	if statusCounts[api.MatchStatusFinished] != 2 {
		t.Errorf("finished count = %d, want 2", statusCounts[api.MatchStatusFinished])
	}
	if statusCounts[api.MatchStatusNotStarted] != 1 {
		t.Errorf("not_started count = %d, want 1 (today only)", statusCounts[api.MatchStatusNotStarted])
	}
}

func TestRunFinished_IncludeUpcomingFlag(t *testing.T) {
	t.Setenv(EnvOffline, "")
	t.Setenv(EnvAgent, "")

	// Mock data is finished-only; the flag should not break mock execution.
	var stdout, stderr bytes.Buffer
	code := runFinished(&stdout, &stderr, finishedFlags{
		cliFlags:        cliFlags{mock: true, timeout: time.Second},
		days:            1,
		includeUpcoming: true,
	})
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, stderr.String())
	}
}
