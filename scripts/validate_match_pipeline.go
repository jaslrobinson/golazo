package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/fotmob"
)

// Validates the full match data pipeline: league page fetch -> match list -> match details.
// Tests both the stats view flow (finished matches) and live view flow.
// Exit code 0 = all tests pass, 1 = failure.
func main() {
	client := fotmob.NewClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	passed := 0
	failed := 0

	// Test 1: Fetch finished matches (stats view flow)
	fmt.Println("=== Test 1: Stats view - finished matches ===")
	var finishedMatches []api.Match
	var testDate time.Time
	for daysBack := 1; daysBack <= 7; daysBack++ {
		testDate = time.Now().AddDate(0, 0, -daysBack)
		matches, err := client.MatchesByDateWithTabs(ctx, testDate, []string{"results"})
		if err != nil {
			fmt.Printf("  Day -%d (%s): error: %v\n", daysBack, testDate.Format("2006-01-02"), err)
			continue
		}
		if len(matches) > 0 {
			finishedMatches = matches
			fmt.Printf("  Day -%d (%s): found %d finished matches\n", daysBack, testDate.Format("2006-01-02"), len(matches))
			break
		}
		fmt.Printf("  Day -%d (%s): no matches\n", daysBack, testDate.Format("2006-01-02"))
	}

	if len(finishedMatches) == 0 {
		fmt.Println("  FAIL: No finished matches found in last 7 days")
		failed++
	} else {
		// Verify match data quality
		withScores := 0
		withPageURL := 0
		withLeague := 0
		for _, m := range finishedMatches {
			if m.HomeScore != nil && m.AwayScore != nil {
				withScores++
			}
			if m.PageURL != "" {
				withPageURL++
			}
			if m.League.ID > 0 {
				withLeague++
			}
		}
		fmt.Printf("  Total: %d | With scores: %d | With pageURL: %d | With league: %d\n",
			len(finishedMatches), withScores, withPageURL, withLeague)

		if withScores == 0 {
			fmt.Println("  FAIL: No matches have scores")
			failed++
		} else if withPageURL == 0 {
			fmt.Println("  FAIL: No matches have pageURL")
			failed++
		} else {
			fmt.Println("  PASS")
			passed++
		}
	}

	// Test 2: Fetch match details for a finished match
	fmt.Println("\n=== Test 2: Match details (page-based fetch) ===")
	var testMatchID int
	var testMatchName string
	for _, m := range finishedMatches {
		if m.PageURL != "" && m.HomeScore != nil {
			testMatchID = m.ID
			testMatchName = fmt.Sprintf("%s %d-%d %s", m.HomeTeam.Name, *m.HomeScore, *m.AwayScore, m.AwayTeam.Name)
			break
		}
	}

	if testMatchID == 0 {
		fmt.Println("  SKIP: No suitable finished match for details test")
	} else {
		fmt.Printf("  Testing: %s (ID: %d)\n", testMatchName, testMatchID)
		details, err := client.MatchDetails(ctx, testMatchID)
		if err != nil {
			fmt.Printf("  FAIL: MatchDetails error: %v\n", err)
			failed++
		} else if details == nil {
			fmt.Println("  FAIL: MatchDetails returned nil")
			failed++
		} else {
			fmt.Printf("  Score: %d - %d\n", *details.HomeScore, *details.AwayScore)
			fmt.Printf("  Status: %s\n", details.Status)
			fmt.Printf("  Events: %d\n", len(details.Events))
			fmt.Printf("  Statistics: %d\n", len(details.Statistics))
			fmt.Printf("  Home starting XI: %d\n", len(details.HomeStarting))
			fmt.Printf("  Away starting XI: %d\n", len(details.AwayStarting))
			fmt.Printf("  Venue: %s\n", details.Venue)
			if details.HomeFormation != "" {
				fmt.Printf("  Formations: %s vs %s\n", details.HomeFormation, details.AwayFormation)
			}
			if details.Highlight != nil {
				fmt.Printf("  Highlight: %s\n", details.Highlight.URL)
			}

			// Validate key fields
			issues := 0
			if details.HomeScore == nil || details.AwayScore == nil {
				fmt.Println("  WARNING: Missing score")
				issues++
			}
			if len(details.Events) == 0 {
				fmt.Println("  WARNING: No events")
				issues++
			}
			if len(details.HomeStarting) == 0 {
				fmt.Println("  WARNING: No home lineup")
				issues++
			}
			if details.Venue == "" {
				fmt.Println("  WARNING: No venue")
				issues++
			}

			if issues > 0 {
				fmt.Printf("  PASS (with %d warnings)\n", issues)
			} else {
				fmt.Println("  PASS")
			}
			passed++
		}
	}

	// Test 3: Fetch today's fixtures (live view flow)
	fmt.Println("\n=== Test 3: Live view - today's fixtures ===")
	today := time.Now()
	todayMatches, err := client.MatchesByDateWithTabs(ctx, today, []string{"fixtures"})
	if err != nil {
		fmt.Printf("  FAIL: Error fetching today's fixtures: %v\n", err)
		failed++
	} else {
		liveCount := 0
		upcomingCount := 0
		for _, m := range todayMatches {
			if m.Status == api.MatchStatusLive {
				liveCount++
			} else if m.Status == api.MatchStatusNotStarted {
				upcomingCount++
			}
		}
		fmt.Printf("  Live: %d | Upcoming: %d | Total: %d\n", liveCount, upcomingCount, len(todayMatches))
		fmt.Println("  PASS (fixture fetch works, live/upcoming count is time-dependent)")
		passed++
	}

	// Test 4: League table (also uses page-based fetch now)
	fmt.Println("\n=== Test 4: League table ===")
	table, err := client.LeagueTable(ctx, 47, "Premier League")
	if err != nil {
		fmt.Printf("  FAIL: LeagueTable error: %v\n", err)
		failed++
	} else if len(table) == 0 {
		fmt.Println("  FAIL: Empty league table")
		failed++
	} else {
		fmt.Printf("  %d teams in table\n", len(table))
		if len(table) >= 3 {
			for _, entry := range table[:3] {
				fmt.Printf("    %d. %s - %dpts (%dW %dD %dL)\n",
					entry.Position, entry.Team.Name, entry.Points,
					entry.Won, entry.Drawn, entry.Lost)
			}
		}
		fmt.Println("  PASS")
		passed++
	}

	// Test 5: Progressive league fetch (used by live view)
	fmt.Println("\n=== Test 5: Single league fetch (progressive loading) ===")
	leagueMatches, err := client.MatchesForLeagueAndDate(ctx, 47, today, "fixtures")
	if err != nil {
		fmt.Printf("  FAIL: MatchesForLeagueAndDate error: %v\n", err)
		failed++
	} else {
		fmt.Printf("  Premier League today: %d matches\n", len(leagueMatches))
		for _, m := range leagueMatches {
			status := string(m.Status)
			score := ""
			if m.HomeScore != nil && m.AwayScore != nil {
				score = fmt.Sprintf(" (%d-%d)", *m.HomeScore, *m.AwayScore)
			}
			fmt.Printf("    %s vs %s [%s]%s\n", m.HomeTeam.Name, m.AwayTeam.Name, status, score)
		}
		fmt.Println("  PASS")
		passed++
	}

	// Summary
	fmt.Printf("\n=== Results: %d passed, %d failed ===\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
