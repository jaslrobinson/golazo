package cmd

import (
	"context"
	"io"
	"os"
	"sort"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/espncfb"
	"github.com/spf13/cobra"
)

var leaguesFlagSet cliFlags

// resolveLeagues returns every FBS conference, sorted by ID for determinism.
// Pure in-memory read via espncfb.Client.Leagues; no network call.
func resolveLeagues() []api.League {
	out, _ := espncfb.NewClient().Leagues(context.Background())
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// runLeagues is the testable core of the `leagues` subcommand.
func runLeagues(stdout, stderr io.Writer, flags cliFlags) int {
	applyPretty(flags)
	leagues := resolveLeagues()
	if err := WriteJSON(stdout, leagues); err != nil {
		return WriteError(stderr, ErrCodeUpstreamError, err)
	}
	return ExitOK
}

var leaguesCmd = &cobra.Command{
	Use:   "leagues",
	Short: "List FBS conferences as JSON",
	Long: `Prints a JSON envelope listing every hardcoded FBS conference (ESPN "group"). No network calls. Useful for discovering conference IDs to interpret live/finished results.

Example output:
  {"status":"ok","count":3,"data":[{"id":1,"name":"ACC","country":"NCAA"},{"id":4,"name":"Big 12","country":"NCAA"},{"id":5,"name":"Big Ten","country":"NCAA"}]}`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		code := runLeagues(os.Stdout, os.Stderr, leaguesFlagSet)
		if code != ExitOK {
			os.Exit(code)
		}
	},
}

func init() {
	addPrettyOnlyFlag(leaguesCmd, &leaguesFlagSet)
	rootCmd.AddCommand(leaguesCmd)
}
