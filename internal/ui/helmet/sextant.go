// Sextant-block rendering: an alternative to the quadrant technique in
// helmet.go using Unicode "BLOCK SEXTANT-N" characters (Symbols for Legacy
// Computing, U+1FB00 range). Each terminal cell packs a 2-column x 3-row
// arrangement of sub-pixels (6 vs quadrant's 4), trading a slightly coarser
// horizontal split for a finer vertical one — see
// docs/superpowers/specs for the size/detail comparison this prototypes.
package helmet

import (
	"fmt"
	"image"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// sextantGlyph returns the Unicode glyph for a 6-bit sub-pixel mask
// (bit0=TL, bit1=TR, bit2=ML, bit3=MR, bit4=BL, bit5=BR). Three masks alias
// pre-existing Block Elements characters instead of the dedicated
// SEXTANT-N codepoints: 0 (space), the left column 0b010101 (LEFT HALF
// BLOCK), the right column 0b101010 (RIGHT HALF BLOCK), and all six
// (FULL BLOCK). The remaining 60 masks map onto U+1FB00-U+1FB3B in
// ascending mask order with those three values skipped - verified against
// unicodedata's "BLOCK SEXTANT-N" names rather than hand-transcribed (see
// sextantWantTable in sextant_test.go).
func sextantGlyph(mask uint8) rune {
	const leftCol, rightCol, all = 0b010101, 0b101010, 0b111111
	switch mask {
	case 0:
		return ' '
	case all:
		return '█'
	case leftCol:
		return '▌'
	case rightCol:
		return '▐'
	}
	index := int(mask) - 1
	if mask > leftCol {
		index--
	}
	if mask > rightCol {
		index--
	}
	return rune(0x1FB00 + index)
}

// sextantMask returns the 6-bit "on" mask for the six sub-pixels in
// TL,TR,ML,MR,BL,BR order.
func sextantMask(tl, tr, ml, mr, bl, br quadrant) uint8 {
	var m uint8
	for i, q := range [6]quadrant{tl, tr, ml, mr, bl, br} {
		if q.on {
			m |= 1 << uint(i)
		}
	}
	return m
}

// sextantSplit is one candidate way to divide six sub-pixels into a
// foreground shape (groupAMask) and everything else, for approximating a
// fully-opaque cell with two colors.
type sextantSplit struct {
	groupAMask uint8
}

var sextantSplitCandidates = []sextantSplit{
	{groupAMask: 0b000011}, // top row vs middle+bottom rows
	{groupAMask: 0b001111}, // top+middle rows vs bottom row
	{groupAMask: 0b010101}, // left column vs right column
}

// renderSextantCell picks the sextant glyph matching which of the six
// sub-pixels are opaque, and the foreground/background colors to render it
// in:
//   - 0 opaque: a blank space (terminal background shows through).
//   - 1-5 opaque: the glyph for exactly that mask, colored by the average
//     of the opaque sub-pixels - the transparent ones show through, so
//     only one color is needed.
//   - 6 opaque: no sub-pixel can show through, so this needs a real
//     two-color approximation. Each candidate split in
//     sextantSplitCandidates is scored by total within-group color
//     variance; the lowest-error split is rendered as that group's glyph
//     in its average color, foreground, against the other group's average
//     color as background.
func renderSextantCell(tl, tr, ml, mr, bl, br quadrant) string {
	cells := [6]quadrant{tl, tr, ml, mr, bl, br}
	mask := sextantMask(tl, tr, ml, mr, bl, br)

	switch mask {
	case 0:
		return " "
	case 0b111111:
		return renderSplitCell(cells)
	default:
		var on []quadrant
		for _, c := range cells {
			if c.on {
				on = append(on, c)
			}
		}
		style := lipgloss.NewStyle().SetString(string(sextantGlyph(mask)))
		return style.Foreground(avgColorN(on...)).String()
	}
}

func renderSplitCell(cells [6]quadrant) string {
	bestErr := -1
	var bestA, bestB []quadrant
	var bestMaskA uint8

	for _, split := range sextantSplitCandidates {
		var groupA, groupB []quadrant
		for i, c := range cells {
			if split.groupAMask&(1<<uint(i)) != 0 {
				groupA = append(groupA, c)
			} else {
				groupB = append(groupB, c)
			}
		}
		err := groupVariance(groupA) + groupVariance(groupB)
		if bestErr == -1 || err < bestErr {
			bestErr = err
			bestA, bestB = groupA, groupB
			bestMaskA = split.groupAMask
		}
	}

	style := lipgloss.NewStyle().SetString(string(sextantGlyph(bestMaskA)))
	return style.Foreground(avgColorN(bestA...)).Background(avgColorN(bestB...)).String()
}

func avgColorN(cells ...quadrant) lipgloss.Color {
	var rSum, gSum, bSum uint32
	for _, c := range cells {
		rSum += uint32(c.r)
		gSum += uint32(c.g)
		bSum += uint32(c.b)
	}
	n := uint32(len(cells))
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", rSum/n, gSum/n, bSum/n))
}

func groupVariance(cells []quadrant) int {
	var rSum, gSum, bSum int
	for _, c := range cells {
		rSum += int(c.r)
		gSum += int(c.g)
		bSum += int(c.b)
	}
	n := len(cells)
	mr, mg, mb := rSum/n, gSum/n, bSum/n

	var total int
	for _, c := range cells {
		dr, dg, db := int(c.r)-mr, int(c.g)-mg, int(c.b)-mb
		total += dr*dr + dg*dg + db*db
	}
	return total
}

// renderSextantNRGBA converts a background-removed helmet image into
// colored sextant-block terminal art. Each terminal cell packs a 2x3
// arrangement of sub-pixels, each itself an average of a subsample x
// subsample block of source pixels (see quadrantAt in helmet.go).
func renderSextantNRGBA(img *image.NRGBA) string {
	bounds := img.Bounds()
	stepX := subsample * 2
	stepY := subsample * 3
	var b strings.Builder
	for y := bounds.Min.Y; y+stepY-1 < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x+stepX-1 < bounds.Max.X; x += stepX {
			tl := quadrantAt(img, x, y)
			tr := quadrantAt(img, x+subsample, y)
			ml := quadrantAt(img, x, y+subsample)
			mr := quadrantAt(img, x+subsample, y+subsample)
			bl := quadrantAt(img, x, y+2*subsample)
			br := quadrantAt(img, x+subsample, y+2*subsample)
			b.WriteString(renderSextantCell(tl, tr, ml, mr, bl, br))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// RenderSextantNRGBA renders a decoded, background-removed helmet image
// using the sextant-block technique. Exported for prototype comparison
// tooling only; RenderSextant(espnTeamID int) will wrap this against
// embedded assets once a rollout is confirmed.
func RenderSextantNRGBA(img *image.NRGBA) string {
	return renderSextantNRGBA(img)
}
