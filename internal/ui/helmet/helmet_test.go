package helmet

import (
	"image"
	"image/color"
	"strings"
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

func TestRenderQuadrantCell_NoneOnIsBlank(t *testing.T) {
	got := renderQuadrantCell(off(), off(), off(), off())
	if got != " " {
		t.Fatalf("got %q, want a single space", got)
	}
}

func TestRenderQuadrantCell_OneOnUsesSingleQuadrantGlyph(t *testing.T) {
	tests := []struct {
		name                     string
		tl, tr, bl, br           quadrant
		wantGlyph, unwantedGlyph string
	}{
		{"top-left", on(10, 20, 30), off(), off(), off(), "▘", "▝"},
		{"top-right", off(), on(10, 20, 30), off(), off(), "▝", "▖"},
		{"bottom-left", off(), off(), on(10, 20, 30), off(), "▖", "▗"},
		{"bottom-right", off(), off(), off(), on(10, 20, 30), "▗", "▘"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderQuadrantCell(tt.tl, tt.tr, tt.bl, tt.br)
			if !strings.Contains(got, tt.wantGlyph) {
				t.Errorf("got %q, want it to contain %q", got, tt.wantGlyph)
			}
			if strings.Contains(got, tt.unwantedGlyph) {
				t.Errorf("got %q, should not contain %q", got, tt.unwantedGlyph)
			}
		})
	}
}

func TestRenderQuadrantCell_TwoOnUsesMatchingHalfGlyph(t *testing.T) {
	c := func() quadrant { return on(200, 0, 0) }
	tests := []struct {
		name           string
		tl, tr, bl, br quadrant
		wantGlyph      string
	}{
		{"top pair", c(), c(), off(), off(), "▀"},
		{"bottom pair", off(), off(), c(), c(), "▄"},
		{"left pair", c(), off(), c(), off(), "▌"},
		{"right pair", off(), c(), off(), c(), "▐"},
		{"diagonal tl-br", c(), off(), off(), c(), "▚"},
		{"diagonal tr-bl", off(), c(), c(), off(), "▞"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderQuadrantCell(tt.tl, tt.tr, tt.bl, tt.br)
			if !strings.Contains(got, tt.wantGlyph) {
				t.Errorf("got %q, want it to contain %q", got, tt.wantGlyph)
			}
		})
	}
}

func TestRenderQuadrantCell_ThreeOnUsesMatchingThreeQuarterGlyph(t *testing.T) {
	c := func() quadrant { return on(0, 200, 0) }
	tests := []struct {
		name           string
		tl, tr, bl, br quadrant
		wantGlyph      string
	}{
		{"missing bottom-right", c(), c(), c(), off(), "▛"},
		{"missing bottom-left", c(), c(), off(), c(), "▜"},
		{"missing top-right", c(), off(), c(), c(), "▙"},
		{"missing top-left", off(), c(), c(), c(), "▟"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderQuadrantCell(tt.tl, tt.tr, tt.bl, tt.br)
			if !strings.Contains(got, tt.wantGlyph) {
				t.Errorf("got %q, want it to contain %q", got, tt.wantGlyph)
			}
		})
	}
}

func TestRenderQuadrantCell_FourOnPicksLowestErrorSplit(t *testing.T) {
	// tl/tr are near-identical red, bl/br are near-identical blue: the
	// vertical (top/bottom) split has far less color error than the
	// horizontal or diagonal splits, so it must be chosen.
	tl := on(200, 0, 0)
	tr := on(202, 2, 2)
	bl := on(0, 0, 200)
	br := on(2, 2, 202)

	got := renderQuadrantCell(tl, tr, bl, br)

	if !strings.Contains(got, "▀") {
		t.Fatalf("got %q, want the vertical split glyph ▀", got)
	}
}

func TestRenderNRGBA_PacksFourSourceRowsPerTerminalRow(t *testing.T) {
	// step = subsample*2 = 4: an 8-source-row image is exactly 2 terminal
	// rows (each cell consumes a 4x4 block of source pixels).
	img := solidImage(8, 8, on(50, 60, 70))

	got := renderNRGBA(img)

	if lines := strings.Count(got, "\n") + 1; lines != 2 {
		t.Fatalf("got %d terminal rows for an 8-source-row image, want 2 (8 / 4)", lines)
	}
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
