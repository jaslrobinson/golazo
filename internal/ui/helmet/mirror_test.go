package helmet

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestMirrorHorizontal_FlipsPixelColumns(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255}) // red
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255}) // green
	img.SetNRGBA(2, 0, color.NRGBA{B: 255, A: 255}) // blue

	got := mirrorHorizontal(img)

	if c := got.NRGBAAt(0, 0); c.B != 255 {
		t.Errorf("col 0 = %+v, want blue (mirrored from col 2)", c)
	}
	if c := got.NRGBAAt(1, 0); c.G != 255 {
		t.Errorf("col 1 = %+v, want green (unchanged, center column)", c)
	}
	if c := got.NRGBAAt(2, 0); c.R != 255 {
		t.Errorf("col 2 = %+v, want red (mirrored from col 0)", c)
	}
}

func TestMirrorHorizontal_PreservesRowsIndependently(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255}) // top-left red
	img.SetNRGBA(1, 0, color.NRGBA{A: 0})           // top-right transparent
	img.SetNRGBA(0, 1, color.NRGBA{A: 0})           // bottom-left transparent
	img.SetNRGBA(1, 1, color.NRGBA{B: 255, A: 255}) // bottom-right blue

	got := mirrorHorizontal(img)

	if c := got.NRGBAAt(1, 0); c.R != 255 || c.A != 255 {
		t.Errorf("top-right = %+v, want red (row 0's red moved from col 0 to col 1)", c)
	}
	if c := got.NRGBAAt(0, 1); c.B != 255 || c.A != 255 {
		t.Errorf("bottom-left = %+v, want blue (row 1's blue moved from col 1 to col 0)", c)
	}
}

func TestRenderMirroredNRGBA_MirrorsSextantGlyphs(t *testing.T) {
	// A 4x6 image (one sextant cell) with only the top-left sub-pixel
	// block opaque renders unmirrored as SEXTANT-1 (mask 0b000001);
	// mirrored, those pixels move to the top-right position, so it must
	// render as SEXTANT-2 (mask 0b000010) instead.
	img := image.NewNRGBA(image.Rect(0, 0, 4, 6))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}

	got := renderMirroredNRGBA(img)

	wantGlyph := string(rune(0x1FB01))     // SEXTANT-2 (top-right)
	unwantedGlyph := string(rune(0x1FB00)) // SEXTANT-1 (top-left)
	if !strings.Contains(got, wantGlyph) {
		t.Errorf("got %q, want it to contain %U (mirrored top-left sub-pixel)", got, rune(0x1FB01))
	}
	if strings.Contains(got, unwantedGlyph) {
		t.Errorf("got %q, should not contain the unmirrored %U glyph", got, rune(0x1FB00))
	}
}

func TestRenderMirrored_UnknownTeamIDReturnsEmptyString(t *testing.T) {
	if got := RenderMirrored(999999999); got != "" {
		t.Fatalf("RenderMirrored(unknown) = %q, want empty string", got)
	}
}

func TestRenderMirrored_SeedTeamReturnsSameShapeAsRender(t *testing.T) {
	const pennState = 213
	plain := Render(pennState)
	mirrored := RenderMirrored(pennState)

	if mirrored == "" {
		t.Fatalf("RenderMirrored(%d) = empty, want curated art", pennState)
	}
	if strings.Count(plain, "\n") != strings.Count(mirrored, "\n") {
		t.Fatalf("Render and RenderMirrored produced different row counts")
	}
	if len([]rune(plain)) == 0 || plain == mirrored {
		t.Fatalf("expected mirrored art to differ from the unmirrored render for a non-symmetric helmet")
	}
}
