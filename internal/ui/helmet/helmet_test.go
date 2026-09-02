package helmet

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func nrgba(r, g, b, a uint8) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: a}
}

func TestPixelAt_OpaquePixelReturnsHexColor(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, nrgba(0x12, 0x34, 0x56, 255))

	on, c := pixelAt(img, 0, 0)

	if !on {
		t.Fatalf("expected opaque pixel to be on")
	}
	if c != "#123456" {
		t.Fatalf("got color %q, want #123456", c)
	}
}

func TestPixelAt_BelowThresholdAlphaIsOff(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	img.SetNRGBA(0, 0, nrgba(0xff, 0x00, 0x00, alphaThreshold-1))

	on, c := pixelAt(img, 0, 0)

	if on {
		t.Fatalf("expected below-threshold pixel to be off")
	}
	if c != "" {
		t.Fatalf("got color %q, want empty", c)
	}
}

func TestRenderNRGBA_GlyphSelectionPerAlphaCombo(t *testing.T) {
	tests := []struct {
		name                     string
		topOn, botOn             bool
		wantGlyph, unwantedGlyph string
	}{
		{"both opaque", true, true, "▀", "▄"},
		{"top only", true, false, "▀", "▄"},
		{"bottom only", false, true, "▄", "▀"},
		{"neither", false, false, " ", "▀"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img := image.NewNRGBA(image.Rect(0, 0, 1, 2))
			if tt.topOn {
				img.SetNRGBA(0, 0, nrgba(255, 0, 0, 255))
			} else {
				img.SetNRGBA(0, 0, nrgba(255, 0, 0, 0))
			}
			if tt.botOn {
				img.SetNRGBA(0, 1, nrgba(0, 255, 0, 255))
			} else {
				img.SetNRGBA(0, 1, nrgba(0, 255, 0, 0))
			}

			got := renderNRGBA(img)

			if !strings.Contains(got, tt.wantGlyph) {
				t.Errorf("renderNRGBA() = %q, want it to contain %q", got, tt.wantGlyph)
			}
			if strings.Contains(got, tt.unwantedGlyph) {
				t.Errorf("renderNRGBA() = %q, should not contain %q", got, tt.unwantedGlyph)
			}
		})
	}
}

func TestRenderNRGBA_TwoTerminalRowsProduceOneNewline(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 2; x++ {
			img.SetNRGBA(x, y, nrgba(10, 20, 30, 255))
		}
	}

	got := renderNRGBA(img)

	if lines := strings.Count(got, "\n"); lines != 1 {
		t.Fatalf("got %d newlines for a 4-source-row (2 terminal-row) image, want 1", lines)
	}
}

func TestRender_UnknownTeamIDReturnsEmptyString(t *testing.T) {
	if got := Render(999999999); got != "" {
		t.Fatalf("Render(unknown) = %q, want empty string", got)
	}
}

func TestRender_SeedTeamReturnsNonEmptyArt(t *testing.T) {
	const pennState = 213
	got := Render(pennState)
	if got == "" {
		t.Fatalf("Render(%d) = empty, want curated art for the seeded Penn State asset", pennState)
	}
	if lines := strings.Count(got, "\n") + 1; lines != 20 {
		t.Fatalf("Render(%d) produced %d lines, want 20 (40 source rows / 2)", pennState, lines)
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
	}
	for _, tc := range seeded {
		if got := Render(tc.id); got == "" {
			t.Errorf("Render(%d) (%s) = empty, want curated art", tc.id, tc.name)
		}
	}
}
