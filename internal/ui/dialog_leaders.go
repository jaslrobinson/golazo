package ui

import (
	"fmt"

	"github.com/0xjuanma/golazo/internal/api"
	"github.com/0xjuanma/golazo/internal/constants"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LeadersDialog displays player statistical leaders grouped by category
// (passing/rushing/receiving/etc). It replaces the Top Scorers dialog, which
// assumes a single soccer-style leaderboard rather than football's
// role-based categories.
type LeadersDialog struct {
	homeTeam    string
	awayTeam    string
	categories  []api.LeaderCategory
	scrollIndex int
	maxVisible  int
}

// NewLeadersDialog creates a new player leaders dialog.
func NewLeadersDialog(homeTeam, awayTeam string, categories []api.LeaderCategory) *LeadersDialog {
	return &LeadersDialog{
		homeTeam:   homeTeam,
		awayTeam:   awayTeam,
		categories: categories,
		maxVisible: 18,
	}
}

// ID returns the dialog identifier.
func (d *LeadersDialog) ID() string {
	return LeadersDialogID
}

// Update handles input for the leaders dialog.
func (d *LeadersDialog) Update(msg tea.Msg) (Dialog, DialogAction) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "x", "q":
			return d, DialogActionClose{}
		case "j", "down":
			maxScroll := max(len(d.renderLines(200))-d.maxVisible, 0)
			d.scrollIndex = scrollDown(d.scrollIndex, maxScroll)
		case "k", "up":
			d.scrollIndex = scrollUp(d.scrollIndex)
		}
	}
	return d, nil
}

// View renders the leaders dialog.
func (d *LeadersDialog) View(width, height int) string {
	dialogWidth, dialogHeight := DialogSize(width, height, 90, 32)
	content := d.renderContent(dialogWidth - 6)
	return RenderDialogFrameWithHelp(constants.PanelPlayerLeaders, content, constants.HelpLeadersDialog, dialogWidth, dialogHeight)
}

func (d *LeadersDialog) renderContent(width int) string {
	if len(d.categories) == 0 {
		return dialogDimStyle.Render(constants.ErrorNoLeaders)
	}

	lines := d.renderLines(width)
	endIdx := min(d.scrollIndex+d.maxVisible, len(lines))
	visible := lines[d.scrollIndex:endIdx]

	if len(lines) > d.maxVisible {
		visible = append(visible, "", dialogDimStyle.Render(fmt.Sprintf("(%d-%d of %d)", d.scrollIndex+1, endIdx, len(lines))))
	}

	return lipgloss.JoinVertical(lipgloss.Left, visible...)
}

// renderLines flattens all categories into individually-scrollable lines.
func (d *LeadersDialog) renderLines(width int) []string {
	var lines []string
	for i, cat := range d.categories {
		if i > 0 {
			lines = append(lines, "")
		}
		label := cat.Label
		if label == "" {
			label = cat.Key
		}
		lines = append(lines, dialogHeaderStyle.Render(label))
		lines = append(lines, dialogSeparatorStyle.Render(repeatRune('─', width)))
		lines = append(lines, leaderLine(d.homeTeam, cat.HomeLeaders, width))
		lines = append(lines, leaderLine(d.awayTeam, cat.AwayLeaders, width))
	}
	return lines
}

func leaderLine(team string, entries []api.LeaderEntry, width int) string {
	teamCol := dialogTeamStyle.Width(14).Render(truncateString(team, 13))
	if len(entries) == 0 {
		return teamCol + dialogDimStyle.Render("—")
	}
	entry := entries[0]
	statCol := dialogValueStyle.Render(fmt.Sprintf("%s  %s", entry.PlayerName, entry.DisplayValue))
	return lipgloss.JoinHorizontal(lipgloss.Top, teamCol, statCol)
}

func repeatRune(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]rune, n)
	for i := range out {
		out[i] = r
	}
	return string(out)
}
