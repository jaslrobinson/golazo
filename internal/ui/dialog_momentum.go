package ui

import (
	"fmt"
	"strings"

	"github.com/0xjuanma/golazo/internal/api"
	"github.com/0xjuanma/golazo/internal/constants"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// sparkLevels renders 0.0-1.0 values as one of 9 block-height characters.
// This is a simpler substitute for a full braille line chart: it captures
// the shape of win-probability swings without needing 2x4-dot braille math.
var sparkLevels = []rune(" ▁▂▃▄▅▆▇█")

// MomentumDialog displays a win-probability sparkline across the match.
// There is no soccer equivalent in golazo today — FotMob's momentum is a
// qualitative per-minute indicator, whereas this is a real numeric series.
type MomentumDialog struct {
	homeTeam string
	awayTeam string
	points   []api.MomentumPoint
}

// NewMomentumDialog creates a new momentum dialog.
func NewMomentumDialog(homeTeam, awayTeam string, points []api.MomentumPoint) *MomentumDialog {
	return &MomentumDialog{homeTeam: homeTeam, awayTeam: awayTeam, points: points}
}

// ID returns the dialog identifier.
func (d *MomentumDialog) ID() string {
	return MomentumDialogID
}

// Update handles input for the momentum dialog.
func (d *MomentumDialog) Update(msg tea.Msg) (Dialog, DialogAction) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "x", "q":
			return d, DialogActionClose{}
		}
	}
	return d, nil
}

// View renders the momentum dialog.
func (d *MomentumDialog) View(width, height int) string {
	dialogWidth, dialogHeight := DialogSize(width, height, 88, 20)
	content := d.renderContent(dialogWidth - 6)
	return RenderDialogFrameWithHelp(constants.PanelMomentum, content, constants.HelpMomentumDialog, dialogWidth, dialogHeight)
}

func (d *MomentumDialog) renderContent(width int) string {
	if len(d.points) == 0 {
		return dialogDimStyle.Render(constants.ErrorNoMomentum)
	}

	values := make([]float64, len(d.points))
	for i, p := range d.points {
		values[i] = p.HomeWinPct
	}

	spark := sparkline(values, width)
	current := values[len(values)-1]

	legend := fmt.Sprintf("%s %.0f%%   %s %.0f%%",
		truncateString(d.homeTeam, 16), current*100,
		truncateString(d.awayTeam, 16), (1-current)*100,
	)

	swingLabel := describeBiggestSwing(d.points, d.homeTeam, d.awayTeam)

	return lipgloss.JoinVertical(lipgloss.Left,
		dialogValueStyle.Bold(true).Render(spark),
		axisLine(width, "Game start", "Now"),
		"",
		dialogHeaderStyle.Render(legend),
		"",
		dialogDimStyle.Render(swingLabel),
	)
}

// axisLine places a left label and a right label on one line spanning
// exactly width columns, styled dim. Unlike padding each label to the full
// width independently and concatenating, this computes the gap once so the
// result is exactly `width` regardless of terminal size.
func axisLine(width int, left, right string) string {
	gap := width - len(left) - len(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return dialogDimStyle.Render(line)
}

// sparkline downsamples values (assumed 0.0-1.0) to `width` buckets and maps
// each bucket's average to one of 9 block-height characters.
func sparkline(values []float64, width int) string {
	if width <= 0 {
		width = 1
	}
	buckets := bucketAverage(values, width)
	out := make([]rune, len(buckets))
	for i, v := range buckets {
		out[i] = sparkLevels[sparkLevelIndex(v)]
	}
	return string(out)
}

func sparkLevelIndex(v float64) int {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	idx := int(v * float64(len(sparkLevels)-1))
	if idx < 0 {
		idx = 0
	}
	if idx > len(sparkLevels)-1 {
		idx = len(sparkLevels) - 1
	}
	return idx
}

// bucketAverage downsamples values into exactly n buckets by averaging.
// If values is shorter than n, each value is repeated to fill the width.
func bucketAverage(values []float64, n int) []float64 {
	if n <= 0 {
		return nil
	}
	if len(values) == 0 {
		return make([]float64, n)
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		start := i * len(values) / n
		end := (i + 1) * len(values) / n
		if end <= start {
			end = start + 1
		}
		if end > len(values) {
			end = len(values)
		}
		sum := 0.0
		count := 0
		for j := start; j < end; j++ {
			sum += values[j]
			count++
		}
		if count == 0 {
			// Degenerate bucket (more buckets than values); fall back to nearest sample.
			idx := start
			if idx >= len(values) {
				idx = len(values) - 1
			}
			out[i] = values[idx]
			continue
		}
		out[i] = sum / float64(count)
	}
	return out
}

// describeBiggestSwing finds the largest single-play home-win-probability
// swing and names which team it favored. Quarter/clock context isn't
// available here: MomentumPoint carries only a play ID and win percentage
// (ESPN's winprobability feed doesn't include a period), so this reports the
// swing by point index rather than by game time.
func describeBiggestSwing(points []api.MomentumPoint, homeTeam, awayTeam string) string {
	if len(points) < 2 {
		return ""
	}
	biggestIdx := 1
	biggestDelta := 0.0
	for i := 1; i < len(points); i++ {
		delta := points[i].HomeWinPct - points[i-1].HomeWinPct
		if abs(delta) > abs(biggestDelta) {
			biggestDelta = delta
			biggestIdx = i
		}
	}
	favored := homeTeam
	if biggestDelta < 0 {
		favored = awayTeam
	}
	before := points[biggestIdx-1].HomeWinPct
	after := points[biggestIdx].HomeWinPct
	return fmt.Sprintf("Biggest swing: play %d/%d, %s (%.0f%%→%.0f%% home)", biggestIdx, len(points), favored, before*100, after*100)
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
