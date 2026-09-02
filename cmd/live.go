package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/data"
	"github.com/spf13/cobra"
)

// Common per-subcommand flag values. Each subcommand declares its own copies
// so the TUI's rootCmd flag set stays untouched.
type cliFlags struct {
	mock    bool
	debug   bool
	timeout time.Duration
	pretty  bool
}

// addCommonCLIFlags registers --mock, --debug, --timeout, --pretty on a subcmd.
func addCommonCLIFlags(cmd *cobra.Command, f *cliFlags) {
	cmd.Flags().BoolVar(&f.mock, "mock", false, "Use mock data instead of real API")
	cmd.Flags().BoolVar(&f.debug, "debug", false, "Emit debug logs to stderr")
	cmd.Flags().DurationVar(&f.timeout, "timeout", 15*time.Second, "Overall request timeout")
	cmd.Flags().BoolVar(&f.pretty, "pretty", false, "Indent JSON output")
}

// addPrettyOnlyFlag registers only --pretty. Used by subcommands that perform
// no network I/O (leagues, capabilities), so --mock/--debug/--timeout would be
// inert and confuse agents reading the capabilities contract.
func addPrettyOnlyFlag(cmd *cobra.Command, f *cliFlags) {
	cmd.Flags().BoolVar(&f.pretty, "pretty", false, "Indent JSON output")
}

// applyPretty syncs the package-level Pretty toggle. Subcommands call this
// from RunE before emitting output. GOLAZO_AGENT forces compact regardless.
func applyPretty(f cliFlags) {
	if agentMode() {
		Pretty = false
		return
	}
	Pretty = f.pretty
}

// liveFetcher abstracts the live-matches data source so runLive can be tested
// without spinning up an HTTP client. The default implementation calls into
// api.Client; mock-mode callers bypass it entirely.
type liveFetcher func(ctx context.Context) ([]api.Match, error)

// defaultLiveFetcher fetches today's full slate (ESPN's scoreboard has no
// separate "live" endpoint, unlike FotMob) and filters to matches currently
// in progress.
func defaultLiveFetcher(c api.Client) liveFetcher {
	return func(ctx context.Context) ([]api.Match, error) {
		matches, err := c.MatchesByDate(ctx, espnNow())
		if err != nil {
			return nil, err
		}
		return filterByStatus(matches, api.MatchStatusLive), nil
	}
}

// filterByStatus returns the subset of matches with the given status.
func filterByStatus(matches []api.Match, status api.MatchStatus) []api.Match {
	out := make([]api.Match, 0, len(matches))
	for _, m := range matches {
		if m.Status == status {
			out = append(out, m)
		}
	}
	return out
}

// runLive is the testable core of the `live` subcommand. It writes the JSON
// envelope to stdout and any error envelope to stderr; returns the exit code.
func runLive(stdout, stderr io.Writer, flags cliFlags) int {
	applyPretty(flags)

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

	var matches []api.Match
	if flags.mock {
		matches = data.MockLiveMatches()
	} else {
		matches, err = defaultLiveFetcher(client)(ctx)
		if err != nil {
			return WriteError(stderr, ClassifyClientError(err, isTimeout(ctx)), err)
		}
		// Guard against silent timeouts: the underlying client aggregates
		// per-league fetches and returns nil/empty when all goroutines fail
		// due to context cancellation. Agents must be able to distinguish
		// "no live matches" from "timed out".
		if isTimeout(ctx) {
			return WriteError(stderr, ErrCodeTimeout,
				fmt.Errorf("live matches fetch timed out after %s", flags.timeout))
		}
	}

	SortMatches(matches)
	if err := WriteJSON(stdout, matches); err != nil {
		return WriteError(stderr, ErrCodeUpstreamError, err)
	}
	return ExitOK
}

var liveFlags cliFlags

var liveCmd = &cobra.Command{
	Use:   "live",
	Short: "List live matches as JSON",
	Long: `Fetches today's in-progress NCAA college football games (ESPN, all FBS conferences) and prints a JSON envelope to stdout. College football games cluster on Thursday-Saturday, so this is often empty outside that window — that's correct, not a failure.

Example output:
  {"status":"ok","count":1,"data":[{"id":401520281,"league":{"id":8,"name":"SEC"},"home_team":{"id":333,"name":"Alabama Crimson Tide","short_name":"Alabama"},"away_team":{"id":61,"name":"Georgia Bulldogs","short_name":"Georgia"},"status":"live","home_score":17,"away_score":14,"match_time":"2026-09-05T23:30:00Z","live_time":"08:41 - 3rd"}]}`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		code := runLive(os.Stdout, os.Stderr, liveFlags)
		if code != ExitOK {
			os.Exit(code)
		}
	},
}

func init() {
	addCommonCLIFlags(liveCmd, &liveFlags)
	rootCmd.AddCommand(liveCmd)
}
