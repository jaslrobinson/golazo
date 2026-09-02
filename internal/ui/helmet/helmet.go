// Package helmet renders curated team helmet artwork as colored terminal
// ASCII art. Each terminal cell packs a 2x3 block of source pixels using
// Unicode sextant-block characters (see sextant.go) — 6 sub-pixels per
// cell rather than the simpler top/bottom half-block technique in
// internal/ui/worldcup, letting the art occupy a smaller on-screen
// footprint at the same level of detail. Transparent sub-pixels render as
// blank space so the terminal's own background shows through.
package helmet

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"sync"

	"github.com/0xjuanma/golazo/internal/assets"
)

// alphaThreshold is the minimum alpha (0-255), averaged over a quadrant's
// sampled block (see subsample below), for that quadrant to be treated as
// part of the helmet rather than removed background.
const alphaThreshold = 40

// subsample is the size (in source pixels) of the block averaged into each
// sub-pixel sample. A sextant cell therefore consumes a (2*subsample) x
// (3*subsample) block of the embedded PNG - subsample=2 needs a 4x6 block
// per cell. Averaging multiple source pixels per sample (rather than
// sampling one directly) smooths edges without changing the terminal
// footprint; scripts/helmets/generate.py's GRID_COLS/GRID_ROWS must be a
// multiple of 2*subsample and 3*subsample respectively to match.
const subsample = 2

var (
	cacheMu sync.Mutex
	cache   = map[int]*image.NRGBA{}
)

// Render returns the given ESPN team ID's helmet as colored sextant-block
// terminal art, or "" if no curated artwork is embedded for that team.
func Render(espnTeamID int) string {
	img, ok := loadImage(espnTeamID)
	if !ok {
		return ""
	}
	return renderSextantNRGBA(img)
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

// quadrant is one sub-pixel sample packed into a terminal cell (used by
// both the quadrant-shaped 2x2 sampling grid and the sextant-shaped 2x3
// one in sextant.go - the name predates the sextant renderer but the
// struct itself is layout-agnostic).
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
