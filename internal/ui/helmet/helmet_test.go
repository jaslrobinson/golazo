package helmet

import (
	"image"
	"image/color"
	"testing"
)

func on(r, g, b uint8) quadrant { return quadrant{on: true, r: r, g: g, b: b} }
func off() quadrant             { return quadrant{} }

func solidImage(w, h int, q quadrant) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	a := uint8(0)
	if q.on {
		a = 255
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: q.r, G: q.g, B: q.b, A: a})
		}
	}
	return img
}

func TestQuadrantAt_AveragesOpaqueBlockColor(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	img.SetNRGBA(0, 1, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	img.SetNRGBA(1, 1, color.NRGBA{R: 200, G: 200, B: 200, A: 255})

	q := quadrantAt(img, 0, 0)

	if !q.on {
		t.Fatalf("expected fully opaque block to be on")
	}
	if q.r != 150 || q.g != 150 || q.b != 150 {
		t.Fatalf("got (%d,%d,%d), want (150,150,150) average", q.r, q.g, q.b)
	}
}

func TestQuadrantAt_BelowThresholdAverageAlphaIsOff(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 0, B: 0, A: 10})
		}
	}

	q := quadrantAt(img, 0, 0)

	if q.on {
		t.Fatalf("expected below-threshold average alpha to be off")
	}
}

func TestQuadrantAt_TransparentPixelsDoNotDiluteOpaqueColor(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255}) // fully opaque red
	img.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 0, B: 0, A: 0})     // transparent; color should not matter
	img.SetNRGBA(0, 1, color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	img.SetNRGBA(1, 1, color.NRGBA{R: 0, G: 0, B: 0, A: 0})

	q := quadrantAt(img, 0, 0)

	if !q.on {
		t.Fatalf("expected 1-of-4 opaque pixels (avg alpha ~64) to clear the threshold")
	}
	if q.r != 255 || q.g != 0 || q.b != 0 {
		t.Fatalf("got (%d,%d,%d), want pure red - transparent pixels must not dilute the color", q.r, q.g, q.b)
	}
}

func TestRender_UnknownTeamIDReturnsEmptyString(t *testing.T) {
	if got := Render(999999999); got != "" {
		t.Fatalf("Render(unknown) = %q, want empty string", got)
	}
}

func TestRender_AllSeededTeamsProduceNonEmptyArt(t *testing.T) {
	seeded := []struct {
		id   int
		name string
	}{
		{213, "Penn State"},
		{333, "Alabama"},
		{2483, "Oregon"},
		{194, "Ohio State"},
		{30, "USC"},
		{23, "San Jose State"},
		{2628, "TCU"},
		{153, "North Carolina"},
		{258, "Virginia"},
		{152, "NC State"},
		{2449, "North Dakota State"},
		{55, "Jacksonville State"},
		{24, "Stanford"},
		{62, "Hawaii"},
		{52, "Florida State"},
		{166, "New Mexico State"},
		{2439, "UNLV"},
		{235, "Memphis"},
		{8, "Arkansas"},
		{2, "Auburn"},
		{57, "Florida"},
		{61, "Georgia"},
		{96, "Kentucky"},
		{99, "LSU"},
		{344, "Mississippi State"},
		{142, "Missouri"},
		{145, "Ole Miss"},
		{201, "Oklahoma"},
		{2579, "South Carolina"},
		{2633, "Tennessee"},
		{251, "Texas"},
		{245, "Texas A&M"},
		{238, "Vanderbilt"},
	}
	for _, tc := range seeded {
		if got := Render(tc.id); got == "" {
			t.Errorf("Render(%d) (%s) = empty, want curated art", tc.id, tc.name)
		}
	}
}
