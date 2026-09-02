# Colored ASCII Helmet Art Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render each team's real helmet as colored half-block terminal art in the match details view, sourced from curated greengridiron.com product photos, background-removed offline.

**Architecture:** A one-time, unshipped Python script downloads a hand-picked side-profile photo per team, removes the background with `rembg`, crops, and downsamples it to a 30x40 RGBA PNG. Those PNGs are committed to the repo and embedded into the Go binary via `//go:embed`. A new `internal/ui/helmet` package decodes them and renders colored half-block (`▀`/`▄`) art with a 4-state transparency rule (both/top/bottom/neither pixel opaque), keyed by ESPN's numeric team ID. `internal/ui/match_details.go` renders both teams' helmets side by side in its header.

**Tech Stack:** Go 1.x (existing module `github.com/0xjuanma/golazo`), `charmbracelet/lipgloss` (already a dependency), Python 3 + `rembg` + `Pillow` + `numpy` for the offline generator only (not a build/runtime dependency of the Go app).

**Spec:** `docs/superpowers/specs/2026-09-01-helmet-ascii-art-design.md`

## Global Constraints

- The Python generator (`scripts/helmets/`) is a developer-run tool, never invoked by `go build`, `go test`, or CI — its only output artifact is the committed PNGs.
- Generated art is stored as embedded PNG files (`internal/assets/helmets/<espn_team_id>.png`), never as Go source literals — avoids compile-time/source-size bloat.
- `internal/ui/helmet` must not import `internal/ui` (or any package that imports it) — `internal/ui/match_details.go` imports `internal/ui/helmet`, and an import cycle would break the build. It may import `internal/assets` and third-party/stdlib packages only.
- A team with no curated art renders as an empty string from `helmet.Render`, and call sites must treat that as "render nothing" (the existing convention from `worldcup.RenderPixelFlag`), never an error.
- Target grid size is fixed at 30 columns x 40 source rows (20 terminal rows) — confirmed by the user against a real rendered sample; do not change without re-confirming.

---

### Task 1: Offline generator + first seed asset (Penn State)

**Files:**
- Create: `scripts/helmets/requirements.txt`
- Create: `scripts/helmets/generate.py`
- Create: `scripts/helmets/manifest.json`
- Create (generated, not hand-written): `internal/assets/helmets/213.png`

**Interfaces:**
- Consumes: nothing from other tasks (first task).
- Produces: `internal/assets/helmets/213.png`, a 30x40 RGBA PNG, which Task 2's `//go:embed` pattern (`helmets/*.png`) will pick up, and which Task 3's tests will load by ID `213`.

- [ ] **Step 1: Create the requirements file**

`scripts/helmets/requirements.txt`:
```
rembg
pillow
numpy
```

- [ ] **Step 2: Create the seed manifest**

`scripts/helmets/manifest.json`:
```json
[
  {
    "espn_team_id": 213,
    "name": "Penn State Nittany Lions",
    "image_url": "https://cdn.shopify.com/s/files/1/2458/4861/files/PennStateNittanyLionsRiddellSpeedReplica01_7e371291-6085-4dc8-8716-a905f9c77236.jpg?v=1748088837"
  }
]
```

Note: `213` is ESPN's well-documented numeric team ID for Penn State
(stable across ESPN's public site/API URLs, e.g.
`espn.com/college-football/team/_/id/213/`). Before moving past this
task, spot-check it against the running app if possible (any live
Penn State match should show the helmet once Task 4 is done) — a
wrong ID just means no art renders for that team, not a crash, but
it's worth confirming.

- [ ] **Step 3: Write the generator script**

`scripts/helmets/generate.py`:
```python
#!/usr/bin/env python3
"""Offline generator: turns curated greengridiron helmet photos into the
small transparent PNGs internal/ui/helmet embeds and renders.

Not run by the app, `go build`, or CI. Run manually after editing
manifest.json:

    python3 -m venv .venv
    .venv/bin/pip install -r requirements.txt
    .venv/bin/python generate.py

First run downloads the rembg background-removal model automatically
(~1GB, one-time, cached under ~/.rembg after that).
"""
import io
import json
import urllib.request
from pathlib import Path

import numpy as np
from PIL import Image
from rembg import remove

MANIFEST = Path(__file__).parent / "manifest.json"
OUT_DIR = Path(__file__).parent.parent.parent / "internal" / "assets" / "helmets"
GRID_COLS = 30
GRID_ROWS = 40  # 2 source rows per terminal row (half-block technique)
ALPHA_THRESHOLD = 40
CROP_PADDING = 8


def fetch(url: str) -> Image.Image:
    req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
    with urllib.request.urlopen(req) as resp:
        data = resp.read()
    return Image.open(io.BytesIO(data)).convert("RGB")


def process(src: Image.Image) -> Image.Image:
    removed = remove(src)
    arr = np.array(removed)
    mask = arr[:, :, 3] > ALPHA_THRESHOLD
    ys, xs = np.where(mask)
    if len(ys) == 0:
        raise ValueError("background removal produced an empty mask - check the source photo")
    y0 = max(ys.min() - CROP_PADDING, 0)
    y1 = min(ys.max() + CROP_PADDING, arr.shape[0])
    x0 = max(xs.min() - CROP_PADDING, 0)
    x1 = min(xs.max() + CROP_PADDING, arr.shape[1])
    cropped = removed.crop((x0, y0, x1, y1))
    return cropped.resize((GRID_COLS, GRID_ROWS), Image.LANCZOS)


def main():
    manifest = json.loads(MANIFEST.read_text())
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    for entry in manifest:
        team_id = entry["espn_team_id"]
        name = entry["name"]
        url = entry["image_url"]
        print(f"[{team_id}] {name} <- {url}")
        grid = process(fetch(url))
        out_path = OUT_DIR / f"{team_id}.png"
        grid.save(out_path)
        print(f"  wrote {out_path} ({grid.size[0]}x{grid.size[1]})")


if __name__ == "__main__":
    main()
```

- [ ] **Step 4: Run the generator**

```bash
cd scripts/helmets
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/python generate.py
```

Expected output ends with:
```
[213] Penn State Nittany Lions <- https://cdn.shopify.com/...
  wrote .../internal/assets/helmets/213.png (30x40)
```

- [ ] **Step 5: Verify the generated asset**

```bash
python3 -c "
from PIL import Image
im = Image.open('internal/assets/helmets/213.png')
assert im.size == (30, 40), im.size
assert im.mode == 'RGBA', im.mode
print('ok', im.size, im.mode)
"
```
Expected: `ok (30, 40) RGBA`

- [ ] **Step 6: Commit**

```bash
git add scripts/helmets/requirements.txt scripts/helmets/generate.py scripts/helmets/manifest.json internal/assets/helmets/213.png
git commit -m "feat: add offline helmet-art generator and seed Penn State asset"
```

---

### Task 2: Embed the helmets directory in Go

**Files:**
- Modify: `internal/assets/embed.go`
- Create: `internal/assets/embed_test.go`

**Interfaces:**
- Consumes: `internal/assets/helmets/213.png` (from Task 1) — required for the `//go:embed helmets/*.png` pattern to match at least one file, or the build fails.
- Produces: `assets.HelmetsFS embed.FS`, read by Task 3's `internal/ui/helmet` package as `assets.HelmetsFS.ReadFile(fmt.Sprintf("helmets/%d.png", id))`.

- [ ] **Step 1: Write the failing test**

`internal/assets/embed_test.go`:
```go
package assets

import "testing"

func TestHelmetsFS_ContainsSeededPennStateAsset(t *testing.T) {
	data, err := HelmetsFS.ReadFile("helmets/213.png")
	if err != nil {
		t.Fatalf("HelmetsFS.ReadFile(helmets/213.png) failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("HelmetsFS.ReadFile(helmets/213.png) returned empty data")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/assets/... -run TestHelmetsFS -v`
Expected: FAIL — `undefined: HelmetsFS` (compile error)

- [ ] **Step 3: Add the embed**

Modify `internal/assets/embed.go` — replace the whole file:
```go
// Package assets provides embedded static assets for the golazo application.
package assets

import (
	"embed"
)

// Logo is the golazo logo PNG image, embedded at compile time.
// Used for desktop notifications on Linux and Windows.
//
//go:embed golazo-logo.png
var Logo []byte

// HelmetsFS contains curated, background-removed team helmet artwork,
// generated offline by scripts/helmets/generate.py and rendered by
// internal/ui/helmet. Filenames are ESPN's numeric team ID (e.g. "213.png").
//
//go:embed helmets/*.png
var HelmetsFS embed.FS
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/assets/... -run TestHelmetsFS -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/assets/embed.go internal/assets/embed_test.go
git commit -m "feat: embed curated helmet PNGs via internal/assets.HelmetsFS"
```

---

### Task 3: `internal/ui/helmet` renderer package

**Files:**
- Create: `internal/ui/helmet/helmet.go`
- Test: `internal/ui/helmet/helmet_test.go`

**Interfaces:**
- Consumes: `assets.HelmetsFS` (from Task 2), `image`/`image/png`/`image/draw` stdlib, `github.com/charmbracelet/lipgloss`.
- Produces: `func Render(espnTeamID int) string` — the public entry point Task 4's `match_details.go` calls for both home and away teams. Returns `""` when no curated art exists for that ID.

- [ ] **Step 1: Write the failing tests**

`internal/ui/helmet/helmet_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/helmet/... -v`
Expected: FAIL — package `helmet` doesn't exist yet (no non-test `.go` file)

- [ ] **Step 3: Write the implementation**

`internal/ui/helmet/helmet.go`:
```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/ui/helmet/... -v`
Expected: PASS (all cases)

- [ ] **Step 5: Commit**

```bash
git add internal/ui/helmet/helmet.go internal/ui/helmet/helmet_test.go
git commit -m "feat: add internal/ui/helmet half-block renderer with transparency"
```

---

### Task 4: Wire helmets into the match details header

**Files:**
- Modify: `internal/ui/match_details.go:1-12` (imports), `:61-64` (insertion point)
- Modify: `internal/ui/match_details_test.go`

**Interfaces:**
- Consumes: `helmet.Render(espnTeamID int) string` (from Task 3).
- Produces: nothing new consumed by later tasks — this is the UI integration point.

- [ ] **Step 1: Write the failing tests**

Add to `internal/ui/match_details_test.go`:
```go
func TestRenderMatchDetails_HelmetsRow_RendersForSeededTeam(t *testing.T) {
	details := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusNotStarted,
			HomeTeam: api.Team{ID: 213, Name: "Penn State Nittany Lions", ShortName: "Penn St"},
			AwayTeam: api.Team{Name: "Some Other Team", ShortName: "SOT"},
		},
	}

	header, _ := RenderMatchDetails(MatchDetailsConfig{
		Width:   80,
		Height:  40,
		Details: details,
	})

	if !strings.Contains(header, "▀") && !strings.Contains(header, "▄") {
		t.Errorf("RenderMatchDetails header should contain helmet art for a team with seeded artwork")
	}
}

func TestRenderMatchDetails_HelmetsRow_OmittedWhenNeitherTeamSeeded(t *testing.T) {
	details := &api.MatchDetails{
		Match: api.Match{
			Status:   api.MatchStatusNotStarted,
			HomeTeam: api.Team{Name: "Arsenal", ShortName: "ARS"},
			AwayTeam: api.Team{Name: "Chelsea", ShortName: "CHE"},
		},
	}

	header, _ := RenderMatchDetails(MatchDetailsConfig{
		Width:   80,
		Height:  40,
		Details: details,
	})

	if strings.Contains(header, "▀") || strings.Contains(header, "▄") {
		t.Errorf("RenderMatchDetails header should not contain helmet glyphs when no team has seeded art")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ui/... -run TestRenderMatchDetails_HelmetsRow -v`
Expected: FAIL — `TestRenderMatchDetails_HelmetsRow_RendersForSeededTeam` fails because no helmet glyphs are rendered yet (the second test passes trivially since nothing renders helmets yet — that's expected at this step, it'll stay meaningful once Step 3 lands).

- [ ] **Step 3: Add the import**

In `internal/ui/match_details.go`, change:
```go
import (
	"fmt"
	"strings"

	"github.com/0xjuanma/golazo/internal/api"
	"github.com/0xjuanma/golazo/internal/constants"
	"github.com/0xjuanma/golazo/internal/ui/design"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)
```
to:
```go
import (
	"fmt"
	"strings"

	"github.com/0xjuanma/golazo/internal/api"
	"github.com/0xjuanma/golazo/internal/constants"
	"github.com/0xjuanma/golazo/internal/ui/design"
	"github.com/0xjuanma/golazo/internal/ui/helmet"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)
```

- [ ] **Step 4: Insert the helmets row and add the helper function**

In `internal/ui/match_details.go`, change:
```go
	// Status and league info
	headerLines = append(headerLines, renderStatusLine(details, contentWidth))
	headerLines = append(headerLines, "")

	// Teams display
	teamsDisplay := fmt.Sprintf("%s  vs  %s",
```
to:
```go
	// Status and league info
	headerLines = append(headerLines, renderStatusLine(details, contentWidth))
	headerLines = append(headerLines, "")

	// Helmet artwork row (renders nothing when neither team has curated art)
	if helmetsRow := renderHelmetsRow(details.HomeTeam.ID, details.AwayTeam.ID, contentWidth); helmetsRow != "" {
		headerLines = append(headerLines, helmetsRow)
		headerLines = append(headerLines, "")
	}

	// Teams display
	teamsDisplay := fmt.Sprintf("%s  vs  %s",
```

Then add this new function near `renderStatusLine` (after it is fine):
```go
// renderHelmetsRow renders both teams' curated helmet artwork side by side,
// centered within contentWidth. Returns "" if neither team has curated art.
// Note: if only one team has art, the other side renders as blank space
// rather than re-centering around the single helmet — acceptable for v1.
func renderHelmetsRow(homeTeamID, awayTeamID, contentWidth int) string {
	homeArt := helmet.Render(homeTeamID)
	awayArt := helmet.Render(awayTeamID)
	if homeArt == "" && awayArt == "" {
		return ""
	}

	gap := lipgloss.NewStyle().Width(6).Render("")
	row := lipgloss.JoinHorizontal(lipgloss.Center, homeArt, gap, awayArt)
	return lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center).Render(row)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ui/... -v`
Expected: PASS, including all pre-existing `match_details_test.go` tests (no regressions)

- [ ] **Step 6: Commit**

```bash
git add internal/ui/match_details.go internal/ui/match_details_test.go
git commit -m "feat: render curated helmet art in match details header"
```

---

### Task 5: Expand curated batch to Alabama, Oregon, Ohio State

**Files:**
- Modify: `scripts/helmets/manifest.json`
- Create (generated): `internal/assets/helmets/333.png`, `internal/assets/helmets/2483.png`, `internal/assets/helmets/194.png`
- Modify: `internal/ui/helmet/helmet_test.go`

**Interfaces:**
- Consumes: `generate.py` (Task 1), `helmet.Render` (Task 3) — no interface changes, just more data.
- Produces: nothing new consumed by later tasks (this plan's last task).

- [ ] **Step 1: Expand the manifest**

Replace `scripts/helmets/manifest.json` with:
```json
[
  {
    "espn_team_id": 213,
    "name": "Penn State Nittany Lions",
    "image_url": "https://cdn.shopify.com/s/files/1/2458/4861/files/PennStateNittanyLionsRiddellSpeedReplica01_7e371291-6085-4dc8-8716-a905f9c77236.jpg?v=1748088837"
  },
  {
    "espn_team_id": 333,
    "name": "Alabama Crimson Tide",
    "image_url": "https://cdn.shopify.com/s/files/1/2458/4861/files/AlabamaCrimsonTideRiddellSpeedReplica1801.webp?v=1748033073"
  },
  {
    "espn_team_id": 2483,
    "name": "Oregon Ducks",
    "image_url": "https://cdn.shopify.com/s/files/1/2458/4861/files/OregonDucksSpeedAuthenticAppleGreen01.webp?v=1748087987"
  },
  {
    "espn_team_id": 194,
    "name": "Ohio State Buckeyes",
    "image_url": "https://cdn.shopify.com/s/files/1/2458/4861/files/OhioStateBuckeyesRiddellSpeedReplicaHelmet01.jpg?v=1747939571"
  }
]
```

Note: `333` (Alabama), `2483` (Oregon), and `194` (Ohio State) are
ESPN's well-documented numeric team IDs. As with Penn State in Task 1,
spot-check these against the live app (or `espn.com/college-football/team/_/id/<id>/`)
before relying on them — a mismatch only means that team's helmet
won't render, not a crash.

- [ ] **Step 2: Re-run the generator**

```bash
cd scripts/helmets
.venv/bin/python generate.py
```
Expected: 4 lines of `[id] name <- url` / `wrote ...` output, one per manifest entry (Penn State regenerates identically; the other three are new).

- [ ] **Step 3: Write the failing coverage test**

Add to `internal/ui/helmet/helmet_test.go`:
```go
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
```

- [ ] **Step 4: Run test to verify it fails before the assets exist**

(Only meaningful if run before Step 2's regeneration — if Step 2 already ran, skip straight to Step 5's pass check and note this in your task summary.)

Run: `go test ./internal/ui/helmet/... -run TestRender_AllSeededTeamsProduceNonEmptyArt -v`
Expected: FAIL for the three new IDs if the PNGs aren't present yet

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/ui/helmet/... -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add scripts/helmets/manifest.json internal/assets/helmets/333.png internal/assets/helmets/2483.png internal/assets/helmets/194.png internal/ui/helmet/helmet_test.go
git commit -m "feat: add Alabama, Oregon, and Ohio State helmet art"
```

---

## Explicitly out of scope for this plan

- The remaining ~64 Power-conference teams — same pipeline, add entries to `manifest.json` and re-run `generate.py` incrementally.
- A secondary team/conference detail panel integration point (spec section 4) — revisit once this plan's match-details integration is verified live.
- Any rendering in list rows or the standings dialog (spec explicitly excludes this).
- Responsive layout for terminals narrower than ~66 columns of `contentWidth` — two 30-column helmets plus a 6-column gap will overflow on very narrow terminals; not addressed here.
