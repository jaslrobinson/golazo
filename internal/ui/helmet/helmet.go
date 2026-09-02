// Package helmet renders curated team helmet artwork as colored terminal
// ASCII art. Each terminal cell packs a full 2x2 block of source pixels
// using Unicode quadrant-block characters (▘▝▖▗▀▄▌▐▚▞▛▜▙▟) rather than the
// simpler top/bottom half-block technique in internal/ui/worldcup — this
// roughly doubles effective resolution per terminal cell, letting the art
// occupy a smaller on-screen footprint at the same level of detail.
// Transparent quadrants render as blank space so the terminal's own
// background shows through.
package helmet

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"strings"
	"sync"

	"github.com/0xjuanma/golazo/internal/assets"
	"github.com/charmbracelet/lipgloss"
)

// alphaThreshold is the minimum alpha (0-255), averaged over a quadrant's
// sampled block (see subsample below), for that quadrant to be treated as
// part of the helmet rather than removed background.
const alphaThreshold = 40

// subsample is the size (in source pixels) of the block averaged into each
// quadrant. Each terminal cell therefore consumes a (2*subsample) x
// (2*subsample) block of the embedded PNG - subsample=2 needs a 4x4 block
// per cell. Averaging multiple source pixels per quadrant (rather than
// sampling one directly) smooths edges without changing the terminal
// footprint; scripts/helmets/generate.py's GRID_COLS/GRID_ROWS must be a
// multiple of 2*subsample to match.
const subsample = 2

var (
	cacheMu sync.Mutex
	cache   = map[int]*image.NRGBA{}
)

// Render returns the given ESPN team ID's helmet as colored half-block
// terminal art, or "" if no curated artwork is embedded for that team.
func Render(espnTeamID int) string {
	img, ok := loadImage(espnTeamID)
	if !ok {
		return ""
	}
	return renderNRGBA(img)
}

func loadImage(espnTeamID int) (*image.NRGBA, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	if img, ok := cache[espnTeamID]; ok {
		return img, img != nil
	}

	data, err := assets.HelmetsFS.ReadFile(fmt.Sprintf("helmets/%d.png", espnTeamID))
	if err != nil {
		cache[espnTeamID] = nil
		return nil, false
	}

	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		cache[espnTeamID] = nil
		return nil, false
	}

	img := toNRGBA(decoded)
	cache[espnTeamID] = img
	return img, true
}

func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, img, b.Min, draw.Src)
	return dst
}

// renderNRGBA converts a background-removed helmet image into colored
// quadrant-block terminal art. Each terminal cell packs a 2x2 arrangement
// of quadrants (top-left, top-right, bottom-left, bottom-right), each
// itself an average of a subsample x subsample block of source pixels; see
// renderQuadrantCell for how opacity and color combine into a glyph.
func renderNRGBA(img *image.NRGBA) string {
	bounds := img.Bounds()
	step := subsample * 2
	var b strings.Builder
	for y := bounds.Min.Y; y+step-1 < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x+step-1 < bounds.Max.X; x += step {
			tl := quadrantAt(img, x, y)
			tr := quadrantAt(img, x+subsample, y)
			bl := quadrantAt(img, x, y+subsample)
			br := quadrantAt(img, x+subsample, y+subsample)
			b.WriteString(renderQuadrantCell(tl, tr, bl, br))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// quadrant is one of the four sub-pixels packed into a terminal cell.
type quadrant struct {
	on      bool
	r, g, b uint8
}

// quadrantAt averages the subsample x subsample block of source pixels at
// (x0, y0) into a single quadrant value. Color is alpha-weighted so fully
// (or mostly) transparent pixels in the block don't dilute the color of
// the opaque ones with whatever arbitrary RGB they happen to hold.
func quadrantAt(img *image.NRGBA, x0, y0 int) quadrant {
	var rSum, gSum, bSum, aSum uint32
	for dy := 0; dy < subsample; dy++ {
		for dx := 0; dx < subsample; dx++ {
			c := img.NRGBAAt(x0+dx, y0+dy)
			a := uint32(c.A)
			rSum += uint32(c.R) * a
			gSum += uint32(c.G) * a
			bSum += uint32(c.B) * a
			aSum += a
		}
	}
	avgAlpha := aSum / uint32(subsample*subsample)
	if avgAlpha < alphaThreshold {
		return quadrant{}
	}
	return quadrant{on: true, r: uint8(rSum / aSum), g: uint8(gSum / aSum), b: uint8(bSum / aSum)}
}

// renderQuadrantCell picks the Unicode quadrant-block glyph matching which
// of the four sub-pixels are opaque, and the foreground/background colors
// to render them in:
//   - 0 opaque: a blank space (terminal background shows through).
//   - 1 or 3 opaque: a single-color glyph (▘▝▖▗ or ▛▜▙▟) in the average of
//     the opaque sub-pixels' colors — the transparent ones show through.
//   - 2 opaque: a half-block glyph (▀▄▌▐▚▞) matching exactly which pair is
//     on, colored by their average — the other two are fully transparent,
//     not a second color, so there's nothing to approximate.
//   - 4 opaque: no sub-pixel can show through, so this is the one case that
//     needs a real two-color approximation. Each of the three ways to split
//     the four into two pairs (top/bottom, left/right, the two diagonals)
//     is scored by how different the colors within each pair are; the
//     lowest-error split is rendered as that pair's two-color glyph.
func renderQuadrantCell(tl, tr, bl, br quadrant) string {
	on := 0
	for _, q := range [4]quadrant{tl, tr, bl, br} {
		if q.on {
			on++
		}
	}

	style := lipgloss.NewStyle()
	switch on {
	case 0:
		return " "
	case 1:
		switch {
		case tl.on:
			return style.SetString("▘").Foreground(solidColor(tl)).String()
		case tr.on:
			return style.SetString("▝").Foreground(solidColor(tr)).String()
		case bl.on:
			return style.SetString("▖").Foreground(solidColor(bl)).String()
		default:
			return style.SetString("▗").Foreground(solidColor(br)).String()
		}
	case 2:
		switch {
		case tl.on && tr.on:
			return style.SetString("▀").Foreground(avgColor(tl, tr)).String()
		case bl.on && br.on:
			return style.SetString("▄").Foreground(avgColor(bl, br)).String()
		case tl.on && bl.on:
			return style.SetString("▌").Foreground(avgColor(tl, bl)).String()
		case tr.on && br.on:
			return style.SetString("▐").Foreground(avgColor(tr, br)).String()
		case tl.on && br.on:
			return style.SetString("▚").Foreground(avgColor(tl, br)).String()
		default: // tr && bl
			return style.SetString("▞").Foreground(avgColor(tr, bl)).String()
		}
	case 3:
		switch {
		case !br.on:
			return style.SetString("▛").Foreground(avg3Color(tl, tr, bl)).String()
		case !bl.on:
			return style.SetString("▜").Foreground(avg3Color(tl, tr, br)).String()
		case !tr.on:
			return style.SetString("▙").Foreground(avg3Color(tl, bl, br)).String()
		default: // !tl.on
			return style.SetString("▟").Foreground(avg3Color(tr, bl, br)).String()
		}
	default: // 4
		vertErr := colorDistSq(tl, tr) + colorDistSq(bl, br)
		horizErr := colorDistSq(tl, bl) + colorDistSq(tr, br)
		diagErr := colorDistSq(tl, br) + colorDistSq(tr, bl)
		switch {
		case vertErr <= horizErr && vertErr <= diagErr:
			return style.SetString("▀").Foreground(avgColor(tl, tr)).Background(avgColor(bl, br)).String()
		case horizErr <= diagErr:
			return style.SetString("▌").Foreground(avgColor(tl, bl)).Background(avgColor(tr, br)).String()
		default:
			return style.SetString("▚").Foreground(avgColor(tl, br)).Background(avgColor(tr, bl)).String()
		}
	}
}

func solidColor(q quadrant) lipgloss.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", q.r, q.g, q.b))
}

func avgColor(a, b quadrant) lipgloss.Color {
	r := (uint16(a.r) + uint16(b.r) + 1) / 2
	g := (uint16(a.g) + uint16(b.g) + 1) / 2
	bl := (uint16(a.b) + uint16(b.b) + 1) / 2
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, bl))
}

func avg3Color(a, b, c quadrant) lipgloss.Color {
	r := (uint16(a.r) + uint16(b.r) + uint16(c.r)) / 3
	g := (uint16(a.g) + uint16(b.g) + uint16(c.g)) / 3
	bl := (uint16(a.b) + uint16(b.b) + uint16(c.b)) / 3
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, bl))
}

func colorDistSq(a, b quadrant) int {
	dr := int(a.r) - int(b.r)
	dg := int(a.g) - int(b.g)
	db := int(a.b) - int(b.b)
	return dr*dr + dg*dg + db*db
}
