package helmet

import (
	"image"
	"strings"
	"testing"
)

// sextantWantTable is the full mask -> codepoint table, independently
// derived from Unicode's "BLOCK SEXTANT-N" character names (Symbols for
// Legacy Computing, U+1FB00 range) via unicodedata.name(), not
// hand-transcribed. Bit layout: bit0=TL, bit1=TR, bit2=ML, bit3=MR,
// bit4=BL, bit5=BR.
var sextantWantTable = map[uint8]rune{
	0x00: 0x0020, 0x01: 0x1FB00, 0x02: 0x1FB01, 0x03: 0x1FB02,
	0x04: 0x1FB03, 0x05: 0x1FB04, 0x06: 0x1FB05, 0x07: 0x1FB06,
	0x08: 0x1FB07, 0x09: 0x1FB08, 0x0a: 0x1FB09, 0x0b: 0x1FB0A,
	0x0c: 0x1FB0B, 0x0d: 0x1FB0C, 0x0e: 0x1FB0D, 0x0f: 0x1FB0E,
	0x10: 0x1FB0F, 0x11: 0x1FB10, 0x12: 0x1FB11, 0x13: 0x1FB12,
	0x14: 0x1FB13, 0x15: 0x258C, 0x16: 0x1FB14, 0x17: 0x1FB15,
	0x18: 0x1FB16, 0x19: 0x1FB17, 0x1a: 0x1FB18, 0x1b: 0x1FB19,
	0x1c: 0x1FB1A, 0x1d: 0x1FB1B, 0x1e: 0x1FB1C, 0x1f: 0x1FB1D,
	0x20: 0x1FB1E, 0x21: 0x1FB1F, 0x22: 0x1FB20, 0x23: 0x1FB21,
	0x24: 0x1FB22, 0x25: 0x1FB23, 0x26: 0x1FB24, 0x27: 0x1FB25,
	0x28: 0x1FB26, 0x29: 0x1FB27, 0x2a: 0x2590, 0x2b: 0x1FB28,
	0x2c: 0x1FB29, 0x2d: 0x1FB2A, 0x2e: 0x1FB2B, 0x2f: 0x1FB2C,
	0x30: 0x1FB2D, 0x31: 0x1FB2E, 0x32: 0x1FB2F, 0x33: 0x1FB30,
	0x34: 0x1FB31, 0x35: 0x1FB32, 0x36: 0x1FB33, 0x37: 0x1FB34,
	0x38: 0x1FB35, 0x39: 0x1FB36, 0x3a: 0x1FB37, 0x3b: 0x1FB38,
	0x3c: 0x1FB39, 0x3d: 0x1FB3A, 0x3e: 0x1FB3B, 0x3f: 0x2588,
}

func TestSextantGlyph_MatchesUnicodeBlockSextantTable(t *testing.T) {
	for mask := 0; mask <= 0x3f; mask++ {
		m := uint8(mask)
		want := sextantWantTable[m]
		got := sextantGlyph(m)
		if got != want {
			t.Errorf("sextantGlyph(0x%02x) = %U, want %U", m, got, want)
		}
	}
}

func TestRenderSextantCell_NoneOnIsBlank(t *testing.T) {
	got := renderSextantCell(off(), off(), off(), off(), off(), off())
	if got != " " {
		t.Fatalf("got %q, want a single space", got)
	}
}

func TestRenderSextantCell_PartialOnUsesGlyphWithoutMixingInOffColor(t *testing.T) {
	// top row on (mask 0x03 -> SEXTANT-12), rest off.
	got := renderSextantCell(on(10, 20, 30), on(10, 20, 30), off(), off(), off(), off())
	want := string(rune(0x1FB02))
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want it to contain %U", got, rune(0x1FB02))
	}
}

func TestRenderSextantCell_AllOnPicksTopRowVsRestSplit(t *testing.T) {
	red := on(200, 0, 0)
	blue := on(0, 0, 200)
	// top row uniform red, remaining four cells uniform blue: zero
	// within-group variance for the top-row/rest split, so it must win
	// over the top-two-rows/bottom-row and column splits.
	got := renderSextantCell(red, red, blue, blue, blue, blue)
	want := string(rune(0x1FB02)) // SEXTANT-12 (top row)
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want it to contain top-row glyph %U", got, rune(0x1FB02))
	}
}

func TestRenderSextantCell_AllOnPicksTopTwoRowsVsBottomRowSplit(t *testing.T) {
	red := on(200, 0, 0)
	blue := on(0, 0, 200)
	got := renderSextantCell(red, red, red, red, blue, blue)
	want := string(rune(0x1FB0E)) // SEXTANT-1234 (top two rows)
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want it to contain top-two-rows glyph %U", got, rune(0x1FB0E))
	}
}

func TestRenderSextantCell_AllOnPicksColumnSplit(t *testing.T) {
	red := on(200, 0, 0)
	blue := on(0, 0, 200)
	got := renderSextantCell(red, blue, red, blue, red, blue)
	if !strings.Contains(got, "▌") {
		t.Errorf("got %q, want it to contain the left-half-block glyph ▌", got)
	}
}

func TestRenderSextantNRGBA_PacksSixSourceRowsPerTerminalRow(t *testing.T) {
	// step_y = subsample*3 = 6: a 12-source-row image is exactly 2
	// terminal rows (each cell consumes a 4x6 block of source pixels).
	img := solidImage(4, 12, on(50, 60, 70))

	got := renderSextantNRGBA(img)

	if lines := strings.Count(got, "\n") + 1; lines != 2 {
		t.Fatalf("got %d terminal rows for a 12-source-row image, want 2 (12 / 6)", lines)
	}
}

func TestRenderSextantNRGBA_TransparentImageRendersAllBlank(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 6))

	got := renderSextantNRGBA(img)

	if strings.TrimSpace(got) != "" {
		t.Fatalf("got %q, want an all-blank render for a fully transparent image", got)
	}
}
