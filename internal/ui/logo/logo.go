// Package logo renders an NCAA-TUI wordmark in a stylized way.
package logo

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jaslrobinson/golazo/internal/ui/design"
	"github.com/lucasb-eyer/go-colorful"
)

const diag = `╱`

// wordmarkArt is the main-menu title, generated with `figlet -f standard
// "NCAA-TUI"`. Two earlier attempts at this word didn't work out: hand-built
// 3-row Unicode-block glyphs (matching the original GOLAZO wordmark's style)
// read as an illegible blob once the per-character gradient was applied at
// that resolution, and plain letter-spaced text lacked any presence. A
// proven figlet font gives a taller, unambiguous letterform per character
// without hand-tuning glyphs — baked in as a static string since embedding
// figlet itself (or a font file) isn't worth the dependency for one banner.
const wordmarkArt = ` _   _  ____    _        _       _____ _   _ ___
| \ | |/ ___|  / \      / \     |_   _| | | |_ _|
|  \| | |     / _ \    / _ \ _____| | | | | || |
| |\  | |___ / ___ \  / ___ \_____| | | |_| || |
|_| \_|\____/_/   \_\/_/   \_\    |_|  \___/|___|`

// Opts are the options for rendering the NCAA-TUI title art.
type Opts struct {
	FieldColorHex    string // diagonal lines color
	GradientStartHex string // left gradient ramp point
	GradientEndHex   string // right gradient ramp point
	Width            int    // width of the rendered logo
}

// DefaultOpts returns default options using the theme colors.
func DefaultOpts() Opts {
	startHex, endHex := design.AdaptiveGradientColors()
	return Opts{
		FieldColorHex:    startHex,
		GradientStartHex: startHex,
		GradientEndHex:   endHex,
		Width:            80,
	}
}

// Render renders the NCAA-TUI logo.
// The compact argument determines whether it renders compact (for sidebar)
// or wider (for main pane).
func Render(version string, compact bool, o Opts) string {
	fg := func(hexColor string, s string) string {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(hexColor)).Render(s)
	}

	// Apply the gradient per line so it reads consistently top-to-bottom
	// rather than restarting per row.
	b := new(strings.Builder)
	for line := range strings.SplitSeq(wordmarkArt, "\n") {
		if line != "" {
			b.WriteString(applyLineGradient(line, o.GradientStartHex, o.GradientEndHex))
		}
		b.WriteString("\n")
	}
	wordmark := strings.TrimSuffix(b.String(), "\n")
	wordmarkWidth := lipgloss.Width(wordmark)

	// Version row
	versionStyled := fg(o.GradientEndHex, version)
	gap := max(0, wordmarkWidth-lipgloss.Width(version))
	metaRow := strings.Repeat(" ", gap) + versionStyled

	// Join the meta row and the title
	wordmark = strings.TrimSpace(wordmark + "\n" + metaRow)

	// Narrow/compact version
	if compact {
		field := fg(o.FieldColorHex, strings.Repeat(diag, wordmarkWidth))
		return strings.Join([]string{field, wordmark, field}, "\n")
	}

	fieldHeight := lipgloss.Height(wordmark)

	// Left field
	const leftWidth = 4
	leftFieldRow := fg(o.FieldColorHex, strings.Repeat(diag, leftWidth))
	leftField := new(strings.Builder)
	for range fieldHeight {
		fmt.Fprintln(leftField, leftFieldRow)
	}

	// Right field with step-down effect
	rightWidth := max(10, o.Width-wordmarkWidth-leftWidth-2)
	rightField := new(strings.Builder)
	for i := range fieldHeight {
		width := rightWidth - i
		if width < 0 {
			width = 0
		}
		fmt.Fprint(rightField, fg(o.FieldColorHex, strings.Repeat(diag, width)), "\n")
	}

	// Join horizontally
	const hGap = " "
	logo := lipgloss.JoinHorizontal(lipgloss.Top, leftField.String(), hGap, wordmark, hGap, rightField.String())

	// Truncate to width if needed
	if o.Width > 0 {
		lines := strings.Split(logo, "\n")
		for i, line := range lines {
			if lipgloss.Width(line) > o.Width {
				lines[i] = truncateAnsi(line, o.Width)
			}
		}
		logo = strings.Join(lines, "\n")
	}

	return logo
}

// RenderCompact renders a smaller inline version suitable for headers.
func RenderCompact(width int) string {
	return design.RenderHeader("NCAA-TUI", width)
}

// applyLineGradient applies a gradient to a single line of text.
func applyLineGradient(text string, startHex, endHex string) string {
	startColor, err1 := colorful.Hex(startHex)
	endColor, err2 := colorful.Hex(endHex)
	if err1 != nil || err2 != nil {
		return text
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}

	var result strings.Builder
	for i, char := range runes {
		if char == ' ' {
			result.WriteRune(' ')
			continue
		}
		ratio := float64(i) / float64(max(len(runes)-1, 1))
		color := startColor.BlendLab(endColor, ratio)
		hexColor := color.Hex()
		charStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(hexColor)).Bold(true)
		result.WriteString(charStyle.Render(string(char)))
	}

	return result.String()
}

// truncateAnsi truncates a string with ANSI codes to a given width.
func truncateAnsi(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	// Simple truncation - not perfect but works for most cases
	runes := []rune(s)
	if len(runes) > width {
		return string(runes[:width])
	}
	return s
}
