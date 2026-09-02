// Command dialog_preview renders the new American-football dialogs with
// realistic mock data so they can be eyeballed before being wired into the
// live app's key bindings. Run with normal color, or NO_COLOR=1 for a
// plain-text layout check.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/ui"
)

func main() {
	const width, height = 120, 40

	situation := ui.NewSituationDialog(
		"USC", "San José State",
		24, 17,
		30, 23,
		&api.Situation{
			Down:             2,
			Distance:         6,
			YardsToEndzone:   56,
			PossessionTeamID: 30,
			DownDistanceText: "2nd & 6",
			PossessionText:   "USC 44",
			HomeTimeouts:     2,
			AwayTimeouts:     3,
			LastPlay:         "(8:42) #6 D.Redeaux run for 6 yards to the USC44 (#24 M.Hoeft)",
		},
	)

	leaders := ui.NewLeadersDialog("USC", "San José State", []api.LeaderCategory{
		{
			Key: "passingYards", Label: "Passing Yards",
			HomeLeaders: []api.LeaderEntry{{PlayerName: "J. Maiava", DisplayValue: "25/29, 286 YDS, 2 TD", Value: 286}},
			AwayLeaders: []api.LeaderEntry{{PlayerName: "M. Latu", DisplayValue: "18/31, 210 YDS", Value: 210}},
		},
		{
			Key: "rushingYards", Label: "Rushing Yards",
			HomeLeaders: []api.LeaderEntry{{PlayerName: "K. Sims", DisplayValue: "14 CAR, 92 YDS, 1 TD", Value: 92}},
			AwayLeaders: []api.LeaderEntry{{PlayerName: "J. Baker", DisplayValue: "11 CAR, 58 YDS", Value: 58}},
		},
		{
			Key: "receivingYards", Label: "Receiving Yards",
			HomeLeaders: []api.LeaderEntry{{PlayerName: "Z. Ross", DisplayValue: "7 REC, 104 YDS, 1 TD", Value: 104}},
			AwayLeaders: []api.LeaderEntry{},
		},
	})

	momentum := ui.NewMomentumDialog("USC", "San José State", syntheticMomentum())

	rankings := ui.NewRankingsDialog([]api.RankingPoll{
		{
			Name: "AP Top 25",
			Entries: []api.RankingEntry{
				{Rank: 1, PreviousRank: 0, Team: api.Team{Name: "Ohio State Buckeyes"}, Points: 1672, FirstPlaceVotes: 40, Trend: "-1"},
				{Rank: 2, PreviousRank: 3, Team: api.Team{Name: "Georgia Bulldogs"}, Points: 1611, FirstPlaceVotes: 0, Trend: "+1"},
				{Rank: 3, PreviousRank: 2, Team: api.Team{Name: "Texas Longhorns"}, Points: 1560, FirstPlaceVotes: 0, Trend: "-1"},
				{Rank: 14, PreviousRank: 14, Team: api.Team{Name: "USC Trojans"}, Points: 1042, FirstPlaceVotes: 0, Trend: "-"},
			},
		},
	})

	dialogs := []struct {
		name string
		d    interface{ View(int, int) string }
	}{
		{"SITUATION", situation},
		{"LEADERS", leaders},
		{"MOMENTUM", momentum},
		{"RANKINGS", rankings},
	}

	reader := bufio.NewReader(os.Stdin)
	for i, entry := range dialogs {
		fmt.Print("\033[H\033[2J") // clear screen so each dialog gets a fresh view
		fmt.Println(strings.Repeat("=", 20) + " " + entry.name + " " + strings.Repeat("=", 20))
		fmt.Println(entry.d.View(width, height))
		fmt.Println()
		if i < len(dialogs)-1 {
			fmt.Printf("-- press Enter for next dialog (%d/%d) --", i+2, len(dialogs))
			reader.ReadString('\n')
		}
	}
}

// syntheticMomentum builds a plausible win-probability arc: home team
// starts favored, away team scores a go-ahead TD mid-game (a real swing),
// home team retakes the lead late. Shaped by hand since the real
// winprobability array (165 points) wasn't fully captured, only 2 samples.
func syntheticMomentum() []api.MomentumPoint {
	var points []api.MomentumPoint
	vals := []float64{
		0.55, 0.58, 0.60, 0.62, 0.65, 0.63, 0.60, // early USC drive
		0.55, 0.50, 0.45, 0.40, // SJSU answers
		0.38, 0.20, 0.18, // SJSU TD - big swing
		0.22, 0.25, 0.30, 0.35, // USC responds
		0.40, 0.45, 0.55, 0.65, 0.72, 0.78, // USC pulls away late
	}
	for i, v := range vals {
		points = append(points, api.MomentumPoint{PlayID: fmt.Sprintf("%d", i), HomeWinPct: v})
	}
	return points
}
