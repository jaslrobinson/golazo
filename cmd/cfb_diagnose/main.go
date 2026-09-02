// Command cfb_diagnose exercises the exact ESPN calls the TUI makes (today's
// live/upcoming matches, the 8-day stats lookback, standings, rankings) and
// prints what actually came back. Run this from your own terminal — not
// through Claude's sandbox, which is Akamai-blocked for raw HTTP — to see
// whether the app's "no data" symptom is a network problem, a date-window
// bug, or something else.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/0xjuanma/golazo/internal/api"
	"github.com/0xjuanma/golazo/internal/espncfb"
)

func main() {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		fmt.Printf("WARNING: could not load America/New_York, falling back to UTC: %v\n", err)
		loc = time.UTC
	}
	now := time.Now().In(loc)
	fmt.Printf("Local time:        %s\n", time.Now().Format(time.RFC1123))
	fmt.Printf("ESPN-basis time:   %s\n\n", now.Format(time.RFC1123))

	client := espncfb.NewClient()
	ctx := context.Background()

	fmt.Println("=== Today's matches (what the Live view fetches) ===")
	matches, err := client.MatchesByDate(ctx, now)
	if err != nil {
		fmt.Printf("ERROR: %v\n\n", err)
	} else {
		fmt.Printf("%d matches returned\n", len(matches))
		printSample(matches)
		fmt.Println()
	}

	fmt.Println("=== 8-day lookback (what the Stats/Finished view fetches) ===")
	totalFinished := 0
	for dayIndex := 0; dayIndex < 8; dayIndex++ {
		date := now.AddDate(0, 0, -dayIndex)
		dayMatches, err := client.MatchesByDate(ctx, date)
		if err != nil {
			fmt.Printf("  day %d (%s): ERROR: %v\n", dayIndex, date.Format("2006-01-02 Mon"), err)
			continue
		}
		finished := 0
		for _, m := range dayMatches {
			if m.Status == api.MatchStatusFinished {
				finished++
			}
		}
		totalFinished += finished
		fmt.Printf("  day %d (%s): %d matches (%d finished)\n", dayIndex, date.Format("2006-01-02 Mon"), len(dayMatches), finished)
		for _, m := range dayMatches {
			fmt.Printf("      [home id=%d] %s  vs  [away id=%d] %s  (status=%s)\n",
				m.HomeTeam.ID, m.HomeTeam.Name, m.AwayTeam.ID, m.AwayTeam.Name, m.Status)
		}
	}
	fmt.Printf("Total finished across 8 days: %d\n\n", totalFinished)

	fmt.Println("=== Standings (SEC, group=8) ===")
	standings, err := client.LeagueTable(ctx, 8, "SEC")
	if err != nil {
		fmt.Printf("ERROR: %v\n\n", err)
	} else {
		fmt.Printf("%d entries returned\n", len(standings))
		for i, e := range standings {
			if i >= 3 {
				break
			}
			fmt.Printf("  #%d %s — conf %s, overall %s\n", e.Position, e.Team.Name, e.ConferenceRecord, e.OverallRecord)
		}
		fmt.Println()
	}

	fmt.Println("=== Match details (what pressing Enter / auto-load fetches) ===")
	// Real match IDs from Saturday 8/29's slate, taken directly from the
	// user's golazo_debug.log where MatchDetails returned nil.
	for _, matchID := range []int{401864577, 401858202} {
		details, err := client.MatchDetails(ctx, matchID)
		if err != nil {
			fmt.Printf("  match %d: ERROR: %v\n", matchID, err)
			continue
		}
		fmt.Printf("  match %d: %s vs %s, %d events, %d stats, situation=%v\n",
			matchID, details.HomeTeam.Name, details.AwayTeam.Name, len(details.Events), len(details.Statistics), details.Situation != nil)
	}
	fmt.Println()

	fmt.Println("=== Rankings (AP Top 25) ===")
	polls, err := client.Rankings(ctx)
	if err != nil {
		fmt.Printf("ERROR: %v\n\n", err)
	} else {
		fmt.Printf("%d polls returned\n", len(polls))
		if len(polls) > 0 && len(polls[0].Entries) > 0 {
			fmt.Printf("  %s #1: %s\n", polls[0].Name, polls[0].Entries[0].Team.Name)
		}
	}
}

func printSample(matches []api.Match) {
	for i, m := range matches {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(matches)-5)
			break
		}
		fmt.Printf("  %s vs %s (status=%s)\n", m.HomeTeam.Name, m.AwayTeam.Name, m.Status)
	}
}
