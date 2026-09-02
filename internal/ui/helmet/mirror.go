package helmet

import "image"

// mirrorHorizontal returns a copy of img flipped left-right. Mirroring the
// source pixels before rendering (rather than post-processing the glyph
// string) means the existing sub-pixel renderer - quadrant today, sextant
// if that lands - reproduces the flipped shape correctly with no
// glyph-swapping logic of its own.
func mirrorHorizontal(img *image.NRGBA) *image.NRGBA {
	b := img.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			mirroredX := b.Min.X + b.Max.X - 1 - x
			dst.SetNRGBA(mirroredX, y, img.NRGBAAt(x, y))
		}
	}
	return dst
}

func renderMirroredNRGBA(img *image.NRGBA) string {
	return renderSextantNRGBA(mirrorHorizontal(img))
}

// RenderMirrored returns the given ESPN team ID's helmet flipped left-right,
// as colored terminal art - used to make two helmets appear to face each
// other (e.g. away team art in a match details header). Returns "" if no
// curated artwork is embedded for that team, matching Render's contract.
func RenderMirrored(espnTeamID int) string {
	img, ok := loadImage(espnTeamID)
	if !ok {
		return ""
	}
	return renderMirroredNRGBA(img)
}
