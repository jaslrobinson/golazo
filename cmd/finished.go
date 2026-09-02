package cmd

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/data"
	"github.com/spf13/cobra"
)

// MaxFinishedDays is the maximum supported lookback window for `golazo finished`.
// College football games cluster Thu-Sat rather than every day, so this must
// reach back far enough to always find the prior weekend's games regardless
// of what day "today" is — 8 mirrors internal/app/commands.go's
// StatsLookbackDays (same reasoning: a Friday "today" needs 6 days back to
// reach the prior Saturday; 8 leaves a day of slack).
const MaxFinishedDays = 8

// DefaultFinishedDays is the --days default. Set to MaxFinishedDays (rather
// than "just today") so `golazo finished` with no flags reliably returns
// recent games instead of an empty array on the many days CFB doesn't play.
const DefaultFinishedDays = MaxFinishedDays

// finishedDayFetcher abstracts a single day's match fetch for testing.
type finishedDayFetcher func(ctx context.Context, date time.Time) ([]api.Match, error)

// collectFinished iterates `days` calendar days ending today, calling the
// per-day fetcher for each. It deduplicates by Match.ID and returns:
//   - the union of finished matches
//   - the list of date strings (YYYY-MM-DD) whose fetch failed
//   - an error iff ALL days failed (callers may then return upstream_error)
//
// This is the layer where the multi-day, dedup, and degraded-surface behavior
// lives, so tests target it directly.
//
// When includeUpcoming is true, today's not-yet-started matches are also
// included in the result. Past days never contain not_started matches.
func collectFinished(ctx context.Context, fetch finishedDayFetcher, now time.Time, days int, includeUpcoming bool) ([]api.Match, []string, error) {
	dedup := make(map[int]api.Match, days*10)
	var failedDates []string
	successCount := 0
	var lastErr error

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		isToday := i == 0

		matches, err := fetch(ctx, date)
		if err != nil {
			failedDates = append(failedDates, dateStr)
			lastErr = err
			continue
		}
		successCount++
		for _, m := range matches {
			switch m.Status {
			case api.MatchStatusFinished:
				dedup[m.ID] = m
			case api.MatchStatusNotStarted:
				if includeUpcoming && isToday {
					dedup[m.ID] = m
				}
			}
		}
	}

	if successCount == 0 {
		return nil, failedDates, lastErr
	}

	out := make([]api.Match, 0, len(dedup))
	for _, m := range dedup {
		out = append(out, m)
	}
	return out, failedDates, nil
}

func defaultFinishedFetcher(c api.Client) finishedDayFetcher {
	return c.MatchesByDate
}

// finishedFlags extends the common flag set with --days and --include-upcoming.
type finishedFlags struct {
	cliFlags
	days            int
	includeUpcoming bool
}

var finishedFlagSet finishedFlags

// runFinished is the testable core of the `finished` subcommand.
func runFinished(stdout, stderr io.Writer, flags finishedFlags) int {
	applyPretty(flags.cliFlags)

	if flags.days < 1 || flags.days > MaxFinishedDays {
		return WriteError(stderr, ErrCodeInvalidArgs,
			NewInvalidArg("--days must be between 1 and %d, got %d", MaxFinishedDays, flags.days))
	}

	client, ctx, cancel, err := newHeadlessClient(runtimeOpts{
		mock:    flags.mock,
		debug:   flags.debug,
		timeout: flags.timeout,
	})
	defer cancel()
	if err == ErrOffline {
		return WriteError(stderr, ErrCodeOffline, err)
	}
	if err != nil {
		return WriteError(stderr, ErrCodeUpstreamError, err)
	}

	var (
		matches     []api.Match
		failedDates []string
	)

	if flags.mock {
		// Mock data is single-day; serve it regardless of --days.
		matches = data.MockFinishedMatches()
	} else {
		matches, failedDates, err = collectFinished(ctx, defaultFinishedFetcher(client), espnNow(), flags.days, flags.includeUpcoming)
	}

	return writeFinishedResult(stdout, stderr, matches, failedDates, err, isTimeout(ctx))
}

// writeFinishedResult reports a collectFinished outcome as an exit code and
// (on success) writes the JSON envelope to stdout.
//
// collectFinished only returns a non-nil err when every requested day
// failed — each day's fetch goes through espncfb.Client.MatchesByDate,
// which returns a real per-day error on deadline exceeded rather than
// silently swallowing cancellation the way FotMob's goroutine aggregator
// could. So err == nil here means every day genuinely completed, even a day
// with zero games; a ctx that happens to be expired by the time this is
// called (collectFinished makes `days` sequential HTTP calls against one
// shared timeout budget, so the deadline can elapse just after the last
// call returns) must not override that legitimate result.
func writeFinishedResult(stdout, stderr io.Writer, matches []api.Match, failedDates []string, collectErr error, timedOut bool) int {
	if collectErr != nil {
		return WriteError(stderr, ClassifyClientError(collectErr, timedOut), collectErr)
	}

	SortMatches(matches)

	var writeErr error
	if len(failedDates) > 0 {
		writeErr = WriteDegraded(stdout, matches, failedDates)
	} else {
		writeErr = WriteJSON(stdout, matches)
	}
	if writeErr != nil {
		return WriteError(stderr, ErrCodeUpstreamError, writeErr)
	}
	return ExitOK
}

var finishedCmd = &cobra.Command{
	Use:   "finished",
	Short: "List finished matches over a day window as JSON",
	Long: `Fetches finished NCAA college football games (ESPN, all FBS conferences) for the last --days days (default 8, since games cluster Thursday-Saturday). Use --include-upcoming to also include today's not-yet-started matches. Partial failures surface as degraded:true with failed_dates listed.

Example output:
  {"status":"ok","count":2,"data":[{"id":401520281,"league":{"id":8,"name":"SEC"},"home_team":{"name":"Alabama Crimson Tide","short_name":"Alabama"},"away_team":{"name":"Georgia Bulldogs","short_name":"Georgia"},"status":"finished","home_score":27,"away_score":24,"match_time":"2026-09-05T23:30:00Z"}]}

Degraded example (one date failed but others succeeded):
  {"status":"ok","degraded":true,"failed_dates":["2026-09-03"],"count":12,"data":[...]}`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		code := runFinished(os.Stdout, os.Stderr, finishedFlagSet)
		if code != ExitOK {
			os.Exit(code)
		}
	},
}

func init() {
	addCommonCLIFlags(finishedCmd, &finishedFlagSet.cliFlags)
	finishedCmd.Flags().IntVar(&finishedFlagSet.days, "days", DefaultFinishedDays, "Number of days to look back (1..8)")
	finishedCmd.Flags().BoolVar(&finishedFlagSet.includeUpcoming, "include-upcoming", false, "Also include today's not-yet-started matches in the result")
	rootCmd.AddCommand(finishedCmd)
}
