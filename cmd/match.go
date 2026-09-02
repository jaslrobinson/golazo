package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/data"
	"github.com/spf13/cobra"
)

type matchDetailsFetcher func(ctx context.Context, matchID int) (*api.MatchDetails, error)

func defaultMatchDetailsFetcher(c api.Client) matchDetailsFetcher {
	return c.MatchDetails
}

var matchFlagSet cliFlags

// runMatch is the testable core of the `match` subcommand.
// args is the positional arg slice from cobra (we expect exactly one ID).
func runMatch(stdout, stderr io.Writer, flags cliFlags, args []string) int {
	applyPretty(flags)

	if len(args) != 1 {
		return WriteError(stderr, ErrCodeInvalidArgs,
			NewInvalidArg("expected exactly one match id, got %d args", len(args)))
	}
	id, err := strconv.Atoi(args[0])
	if err != nil || id <= 0 {
		return WriteError(stderr, ErrCodeInvalidArgs,
			NewInvalidArg("match id must be a positive integer, got %q", args[0]))
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
		details *api.MatchDetails
	)
	if flags.mock {
		details, err = data.MockMatchDetails(id)
	} else {
		details, err = defaultMatchDetailsFetcher(client)(ctx, id)
	}
	if err != nil {
		return WriteError(stderr, ClassifyClientError(err, isTimeout(ctx)), err)
	}
	if details == nil {
		return WriteError(stderr, ErrCodeNotFound, fmt.Errorf("no match found for id %d", id))
	}

	if err := WriteJSON(stdout, []api.MatchDetails{*details}); err != nil {
		return WriteError(stderr, ErrCodeUpstreamError, err)
	}
	return ExitOK
}

var matchCmd = &cobra.Command{
	Use:   "match <id>",
	Short: "Get match details as JSON (events, situation, leaders, momentum)",
	Long: `Fetches detailed information (scoring plays, down/distance situation, player leaders, win-probability momentum, box score) for a single NCAA college football game by ESPN event ID.

Unlike upstream's FotMob-backed version, this doesn't need a page-slug lookup — ESPN's summary endpoint takes the event ID directly (event=<id>), so cold calls with an ID from 'live' or 'finished' work in a single invocation (verified live).

Example (mock):
  golazo match 2001 --mock

Example output (truncated):
  {"status":"ok","count":1,"data":[{"id":401520281,"home_team":{"name":"Alabama Crimson Tide"},"away_team":{"name":"Georgia Bulldogs"},"status":"live","home_score":17,"away_score":14,"events":[{"description":"A.Milroe 12 Yd Run (Touchdown)","type":"touchdown","display_minute":"08:41 - 3rd"}],"statistics":[{"key":"totalYards","label":"Total Yards","home_value":"312","away_value":"287"}],"venue":"Bryant-Denny Stadium"}]}`,
	Args:          cobra.ArbitraryArgs, // validated in runMatch for precise error envelope
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		code := runMatch(os.Stdout, os.Stderr, matchFlagSet, args)
		if code != ExitOK {
			os.Exit(code)
		}
	},
}

func init() {
	addCommonCLIFlags(matchCmd, &matchFlagSet)
	rootCmd.AddCommand(matchCmd)
}
