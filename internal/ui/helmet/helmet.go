// Package helmet renders curated team helmet artwork as colored terminal
// ASCII art, using the half-block (▀) technique from internal/ui/worldcup's
// pixel flags, extended to support a transparent background so only the
// helmet silhouette renders — everything outside the cutout is left blank
// so the terminal's own background shows through.
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

// alphaThreshold is the minimum alpha (0-255) for a source pixel to be
// treated as part of the helmet rather than removed background. Matches
// the threshold used by scripts/helmets/generate.py.
const alphaThreshold = 40

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
// half-block terminal art. Each terminal row packs two source pixel rows:
// both opaque renders a full block with the top pixel as foreground and the
// bottom as background; only one opaque renders a half-block in that
// pixel's color; neither renders a blank space so the terminal's own
// background shows through.
func renderNRGBA(img *image.NRGBA) string {
	bounds := img.Bounds()
	var b strings.Builder
	for y := bounds.Min.Y; y+1 < bounds.Max.Y; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			topOn, tc := pixelAt(img, x, y)
			botOn, bc := pixelAt(img, x, y+1)

			switch {
			case topOn && botOn:
				b.WriteString(lipgloss.NewStyle().SetString("▀").Foreground(tc).Background(bc).String())
			case topOn:
				b.WriteString(lipgloss.NewStyle().SetString("▀").Foreground(tc).String())
			case botOn:
				b.WriteString(lipgloss.NewStyle().SetString("▄").Foreground(bc).String())
			default:
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func pixelAt(img *image.NRGBA, x, y int) (bool, lipgloss.Color) {
	c := img.NRGBAAt(x, y)
	if c.A < alphaThreshold {
		return false, ""
	}
	return true, lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B))
}
