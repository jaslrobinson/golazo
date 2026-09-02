package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/constants"
)

// SituationDialog displays the current down-and-distance and field position
// for an American football match. It replaces the Formations dialog, which
// has no equivalent in a sport without formations.
type SituationDialog struct {
	homeTeam   string
	awayTeam   string
	homeScore  int
	awayScore  int
	homeTeamID int
	awayTeamID int
	situation  *api.Situation
}

// NewSituationDialog creates a new situation dialog.
func NewSituationDialog(homeTeam, awayTeam string, homeScore, awayScore, homeTeamID, awayTeamID int, situation *api.Situation) *SituationDialog {
	return &SituationDialog{
		homeTeam:   homeTeam,
		awayTeam:   awayTeam,
		homeScore:  homeScore,
		awayScore:  awayScore,
		homeTeamID: homeTeamID,
		awayTeamID: awayTeamID,
		situation:  situation,
	}
}

// ID returns the dialog identifier.
func (d *SituationDialog) ID() string {
	return SituationDialogID
}

// Update handles input for the situation dialog.
func (d *SituationDialog) Update(msg tea.Msg) (Dialog, DialogAction) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "x", "q":
			return d, DialogActionClose{}
		}
	}
	return d, nil
}

// View renders the situation dialog.
func (d *SituationDialog) View(width, height int) string {
	dialogWidth, dialogHeight := DialogSize(width, height, 84, 24)
	content := d.renderContent(dialogWidth - 6)
	return RenderDialogFrameWithHelp(constants.PanelSituation, content, constants.HelpSituationDialog, dialogWidth, dialogHeight)
}

func (d *SituationDialog) renderContent(width int) string {
	// Down==0 means no real down-and-distance was ever set (e.g. a provider
	// attached only a last-play fallback, or a zero-valued Situation slipped
	// through) — render the same empty state as a nil Situation rather than
	// a misleading "1st & 0" on the opponent's goal line.
	if d.situation == nil || d.situation.Down == 0 {
		return dialogDimStyle.Render(constants.ErrorNoSituation)
	}
	s := d.situation

	var lines []string

	header := fmt.Sprintf("%s %d - %d %s", d.homeTeam, d.homeScore, d.awayScore, d.awayTeam)
	lines = append(lines, dialogAlignCenter(width, dialogTeamStyle.Render(header)))
	lines = append(lines, "")

	possessionLine := s.DownDistanceText
	if possessionLine == "" && s.Down > 0 {
		possessionLine = ordinal(s.Down) + " & " + fmt.Sprintf("%d", s.Distance)
	}
	if s.PossessionText != "" {
		possessionLine = s.PossessionText + " · " + possessionLine
	}
	if s.IsRedZone {
		possessionLine += "  " + neonRedCardStyle.Render("● RED ZONE")
	}
	lines = append(lines, dialogAlignCenter(width, dialogValueStyle.Bold(true).Render(possessionLine)))
	lines = append(lines, "")

	lines = append(lines, renderFieldPositionBar(width, s.YardsToEndzone, s.Distance))
	lines = append(lines, "")

	timeoutsLine := fmt.Sprintf("Timeouts   %s %s    %s %s",
		truncateString(d.homeTeam, 12), timeoutDots(s.HomeTimeouts),
		truncateString(d.awayTeam, 12), timeoutDots(s.AwayTimeouts),
	)
	lines = append(lines, dialogDimStyle.Render(timeoutsLine))
	lines = append(lines, "")

	if s.LastPlay != "" {
		lines = append(lines, dialogHeaderStyle.Render("Last play"))
		lines = append(lines, dialogValueStyle.Render(wrapText(s.LastPlay, width)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderFieldPositionBar draws a 100-yard field track with a ball marker and
// first-down marker. It's built on yardsToEndzone (yards remaining for the
// possessing team to score) rather than yardLine, since ESPN's yardLine
// reference point is ambiguous without a live game to confirm direction —
// yardsToEndzone is unambiguous by name and sufficient to place both markers.
func renderFieldPositionBar(width, yardsToEndzone, distance int) string {
	trackWidth := width
	if trackWidth < 20 {
		trackWidth = 20
	}

	track := make([]rune, trackWidth)
	for i := range track {
		track[i] = '─'
	}

	ballPos := fieldPositionIndex(yardsToEndzone, trackWidth)
	firstDownYardsToEndzone := yardsToEndzone - distance
	if firstDownYardsToEndzone < 0 {
		firstDownYardsToEndzone = 0
	}
	firstDownPos := fieldPositionIndex(firstDownYardsToEndzone, trackWidth)

	if firstDownPos >= 0 && firstDownPos < trackWidth {
		track[firstDownPos] = '┆'
	}
	if ballPos >= 0 && ballPos < trackWidth {
		track[ballPos] = '●'
	}

	labels := dialogAlignLeft(trackWidth/2, dialogDimStyle.Render("OWN GOAL")) +
		dialogAlignRight(trackWidth-trackWidth/2, dialogDimStyle.Render("OPP GOAL"))

	return lipgloss.JoinVertical(lipgloss.Left,
		labels,
		dialogValueStyle.Render(string(track)),
	)
}

// fieldPositionIndex maps yardsToEndzone (100 = own goal line, 0 = opponent's
// goal line) onto a 0..width-1 track index, own goal on the left.
func fieldPositionIndex(yardsToEndzone, width int) int {
	if yardsToEndzone < 0 {
		yardsToEndzone = 0
	}
	if yardsToEndzone > 100 {
		yardsToEndzone = 100
	}
	fromOwnGoal := 100 - yardsToEndzone
	return fromOwnGoal * (width - 1) / 100
}

func timeoutDots(remaining int) string {
	if remaining < 0 {
		remaining = 0
	}
	if remaining > 3 {
		remaining = 3
	}
	return strings.Repeat("●", remaining) + strings.Repeat("○", 3-remaining)
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return fmt.Sprintf("%dth", n)
	}
}

// wrapText performs simple word-wrapping at width, returning at most two lines.
func wrapText(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	words := strings.Fields(s)
	var lines []string
	var current string
	for _, w := range words {
		if current == "" {
			current = w
			continue
		}
		if len(current)+1+len(w) > width {
			lines = append(lines, current)
			current = w
			if len(lines) == 1 {
				break
			}
			continue
		}
		current += " " + w
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n")
}
