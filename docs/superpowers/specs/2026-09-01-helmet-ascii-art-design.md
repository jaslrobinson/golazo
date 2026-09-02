# Colored ASCII helmet art (design)

## Goal

Show each team's helmet as colored terminal ASCII art (half-block
technique, same family as the existing World Cup flag renderer) in
detail views of the CFB TUI, sourced from real helmet product
photography on greengridiron.com.

## Spike findings (validated before this doc was written)

- Background removal via `rembg` (local ONNX segmentation model, one-time
  ~1GB model download, fully offline afterward) produces a clean
  helmet-only cutout from a product photo.
- **Photo angle is the deciding factor for legibility, not resolution.**
  greengridiron's dramatic 3/4 "hero shot" angle (used as the default
  photo on some products, e.g. the plain `-helmet` handle) downsamples
  into an unrecognizable blob at every size tested (25x14 up to 50x35).
  A clean **side-profile** shot of the same team reads clearly as a
  football helmet even at 25x14, and reads very well at 30x20.
- Side-profile shots exist somewhere in most teams' product catalog
  (mini helmet, "Speed Display/Replica", or the "authentic" full-size
  listing all sometimes have one) but which product has it is
  inconsistent — there's no reliable naming/handle heuristic. Each
  team's source image must be picked by a human glance, not scraped
  blind.
- A real half-block renderer (30x20 cells, JetBrains Mono, background
  cells fully transparent) was built and run against Penn State,
  Alabama, Oregon, and Ohio State. All four are clearly recognizable
  and distinctly colored (Alabama's "18" and crimson shell, Oregon's
  yellow "O" on green, Ohio State's gray/scarlet, Penn State's
  white/navy). User confirmed 30x20 is the target size.

## Scope (v1)

- **Placement: detail views only.** Match details header
  (`internal/ui/match_details.go`) is the confirmed integration point.
  A team/conference detail panel is a secondary candidate, to be
  confirmed during implementation once the current conference-browser
  flow (`internal/ui/conferences_view.go`) is reread — list rows
  (`list_items.go`, `dialog_standings.go`) are explicitly out of scope;
  they're 1-3 terminal lines tall and can't hold a 20-row render.
- **Team coverage: Power conference teams first** (SEC, Big Ten, Big 12,
  ACC, ~68 teams), each hand-verified for a usable side-profile source
  photo. Not all FBS teams up front — curation is a manual,
  per-team step and doesn't scale to a single pass. Teams without
  curated art render nothing (see Fallback).
- Four teams (Penn State, Alabama, Oregon, Ohio State) are already
  spiked and can seed the initial curated batch.

## Architecture

### 1. Offline generator (not shipped, not run by the app)

A Python script (`scripts/helmets/generate.py`) run manually/occasionally
by the maintainer, not part of `go build` or CI:

- Input: a checked-in curation manifest
  (`scripts/helmets/manifest.yaml`), entries of
  `{espn_team_id, name (comment only), image_url}` — the human picks
  `image_url` by eyeballing the team's greengridiron product photos for
  a usable side-profile shot.
- Per entry: download image → `rembg` background removal → crop to
  helmet bounding box → resize to 30 cols x 40 source rows (2 source
  rows per terminal cell, half-block technique) via Lanczos → save as
  a small RGBA PNG.
- Output: `internal/assets/helmets/<espn_team_id>.png` (checked into
  the repo as generated data, same spirit as the flag data files —
  the pipeline that made it is not run by end users).

### 2. Embedding & storage

- PNG-per-team (not Go source literals). At ~68 teams x 30x40px, a
  `[40][30]string` hex-literal encoding (the existing flag approach)
  would produce a multi-hundred-KB to low-MB generated `.go` file;
  small embedded PNGs decoded once at first use are simpler to
  generate (rembg/PIL output PNG naturally), avoid Go source bloat,
  and decode cost is trivial (tiny images, decoded lazily and cached).
- `internal/assets/helmets/*.png` embedded via `//go:embed` next to
  the existing embed setup in `internal/assets/embed.go`.
- Filename = ESPN numeric team ID (`<id>.png`) — this is the same ID
  already flowing through `api.Team.ID` (via `internal/espncfb/map.go`
  `mapTeam`), so no separate name-matching table is needed in the
  shipped Go code. All the fuzzy matching (Miami FL vs OH, service
  academies, Ole Miss, etc.) happens once, by hand, when building the
  curation manifest — not at runtime.

### 3. Renderer

New package `internal/ui/helmet` (sibling to `internal/ui/worldcup`):

```go
func RenderHelmet(espnTeamID int) string
```

- Looks up `<id>.png` in the embedded FS; returns `""` if not present
  (graceful no-op, same contract as `worldcup.RenderPixelFlag` — call
  sites already need to handle "no art for this one").
- Decodes the PNG once per ID and caches the decoded RGBA grid
  (map + mutex, or `sync.Once` per entry) — match details view re-renders
  on every Bubble Tea update, so decoding must not happen per-frame.
- Rendering: for each terminal row, look at the two source pixel rows'
  alpha per column:
  - both opaque → `▀`, `Foreground(top)`, `Background(bottom)`
  - top only → `▀`, `Foreground(top)`, no `.Background()` call
    (inherits terminal background)
  - bottom only → `▄`, `Foreground(bottom)`, no `.Background()` call
  - neither → a plain space, unstyled
- This is a new renderer, not a copy of `RenderPixelFlag` — that
  function unconditionally sets both foreground and background and
  has no transparency case.

### 4. Integration

- `internal/ui/match_details.go`: render home/away helmets in the
  match details header next to team name/score.
- Secondary panel (conference/team detail) confirmed during
  implementation, not blocking v1.

## Fallback & testing

- No curated art for a team ID → `RenderHelmet` returns `""` → call
  sites render their existing text-only layout unchanged (same
  pattern as `RenderPixelFlag`'s `ok` check).
- Unit tests: small fixture PNGs (e.g. 2x2, one of each alpha state)
  to deterministically test the 4-case rendering logic, plus a
  "missing ID returns empty string" test. A coverage-style test
  (mirroring `flag_coverage_test.go`) can assert every ID in the
  curation manifest has a corresponding embedded PNG, if useful.

## Source note

This pipeline scrapes product photography from a commerce site and
commits a small (30x40px), heavily abstracted derived rendering of it
into this repo. This is a personal, non-commercial hobby project with
a single maintainer/user — noted here rather than decided silently.

## Explicitly out of scope for v1

- All FBS teams (Group of 5 + independents) — future pass.
- Any rendering in list rows or the standings dialog.
- Automatic/heuristic photo-angle selection — curation stays manual.
