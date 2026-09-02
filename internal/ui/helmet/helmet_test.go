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

func TestRenderNRGBA_PacksTwoSourceRowsPerTerminalRow(t *testing.T) {
	img := solidImage(4, 4, on(50, 60, 70))

	got := renderNRGBA(img)

	if lines := strings.Count(got, "\n") + 1; lines != 2 {
		t.Fatalf("got %d terminal rows for a 4-source-row image, want 2 (4 / 2)", lines)
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
