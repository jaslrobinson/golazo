package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/constants"
)

// RankingsDialog displays a ranking poll (AP Top 25, Coaches Poll, etc).
// There is no soccer equivalent in golazo today — league position there
// comes purely from the standings table, not a subjective/voted poll.
type RankingsDialog struct {
	polls       []api.RankingPoll
	pollIndex   int
	scrollIndex int
	maxVisible  int
}

// NewRankingsDialog creates a new rankings dialog.
func NewRankingsDialog(polls []api.RankingPoll) *RankingsDialog {
	return &RankingsDialog{polls: polls, maxVisible: 25}
}

// ID returns the dialog identifier.
func (d *RankingsDialog) ID() string {
	return RankingsDialogID
}

// Update handles input for the rankings dialog.
func (d *RankingsDialog) Update(msg tea.Msg) (Dialog, DialogAction) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "x", "q":
			return d, DialogActionClose{}
		case "tab":
			if len(d.polls) > 0 {
				d.pollIndex = (d.pollIndex + 1) % len(d.polls)
				d.scrollIndex = 0
			}
		case "j", "down":
			if p := d.currentPoll(); p != nil {
				d.scrollIndex = scrollDown(d.scrollIndex, max(len(p.Entries)-d.maxVisible, 0))
			}
		case "k", "up":
			d.scrollIndex = scrollUp(d.scrollIndex)
		}
	}
	return d, nil
}

func (d *RankingsDialog) currentPoll() *api.RankingPoll {
	if d.pollIndex < 0 || d.pollIndex >= len(d.polls) {
		return nil
	}
	return &d.polls[d.pollIndex]
}

// View renders the rankings dialog.
func (d *RankingsDialog) View(width, height int) string {
	dialogWidth, dialogHeight := DialogSize(width, height, 84, 32)
	poll := d.currentPoll()
	title := "Rankings"
	if poll != nil {
		title = poll.Name
	}
	content := d.renderContent(dialogWidth - 6)
	return RenderDialogFrameWithHelp(title, content, constants.HelpRankingsDialog, dialogWidth, dialogHeight)
}

func (d *RankingsDialog) renderContent(width int) string {
	poll := d.currentPoll()
	if poll == nil || len(poll.Entries) == 0 {
		return dialogDimStyle.Render(constants.ErrorNoRankings)
	}

	var lines []string
	lines = append(lines, d.renderHeaderRow(width))
	lines = append(lines, dialogSeparatorStyle.Render(repeatRune('─', width)))

	endIdx := min(d.scrollIndex+d.maxVisible, len(poll.Entries))
	for _, e := range poll.Entries[d.scrollIndex:endIdx] {
		lines = append(lines, d.renderEntryRow(e, width))
	}

	if len(poll.Entries) > d.maxVisible {
		lines = append(lines, "", dialogDimStyle.Render(fmt.Sprintf("(%d-%d of %d)", d.scrollIndex+1, endIdx, len(poll.Entries))))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

const (
	rankingsColRank  = 4
	rankingsColTrend = 6
	rankingsColPts   = 6
	rankingsColVotes = 5
)

func (d *RankingsDialog) renderHeaderRow(width int) string {
	teamWidth := width - rankingsColRank - rankingsColTrend - rankingsColPts - rankingsColVotes - 3
	return lipgloss.JoinHorizontal(lipgloss.Top,
		dialogHeaderStyle.Width(rankingsColRank).Align(lipgloss.Right).Render("#"),
		" ",
		dialogHeaderStyle.Width(teamWidth).Align(lipgloss.Left).Render("Team"),
		dialogHeaderStyle.Width(rankingsColTrend).Align(lipgloss.Right).Render("Trend"),
		dialogHeaderStyle.Width(rankingsColPts).Align(lipgloss.Right).Render("Pts"),
		dialogHeaderStyle.Width(rankingsColVotes).Align(lipgloss.Right).Render("1st"),
	)
}

func (d *RankingsDialog) renderEntryRow(e api.RankingEntry, width int) string {
	teamWidth := width - rankingsColRank - rankingsColTrend - rankingsColPts - rankingsColVotes - 3
	teamName := truncateString(e.Team.Name, teamWidth-1)

	votes := ""
	if e.FirstPlaceVotes > 0 {
		votes = fmt.Sprintf("%d", e.FirstPlaceVotes)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		dialogAlignRight(rankingsColRank, fmt.Sprintf("%d", e.Rank)),
		" ",
		dialogAlignLeft(teamWidth, teamName),
		dialogAlignRight(rankingsColTrend, renderTrend(e.Trend)),
		dialogAlignRight(rankingsColPts, fmt.Sprintf("%d", e.Points)),
		dialogAlignRight(rankingsColVotes, votes),
	)
}

func renderTrend(trend string) string {
	switch {
	case trend == "" || trend == "-":
		return dialogDimStyle.Render("—")
	case trend[0] == '-':
		return neonRedCardStyle.Render("▼" + trend[1:])
	case trend[0] == '+':
		return dialogTeamStyle.Render("▲" + trend[1:])
	default:
		return dialogTeamStyle.Render("▲" + trend)
	}
}
