#!/usr/bin/env python3
"""Offline generator: turns curated greengridiron helmet photos into the
small transparent PNGs internal/ui/helmet embeds and renders.

Not run by the app, `go build`, or CI. Run manually after editing
manifest.json:

    python3 -m venv .venv
    .venv/bin/pip install -r requirements.txt
    .venv/bin/python generate.py [espn_team_id ...]

With no arguments, regenerates every team in manifest.json. Pass one or
more ESPN team IDs to regenerate only those (e.g. after swapping one
team's image_url for a better source photo).

First run downloads the rembg background-removal model automatically
(~1GB, one-time, cached under ~/.rembg after that).
"""
import io
import json
import sys
import urllib.request
from pathlib import Path

import numpy as np
from PIL import Image
from rembg import remove

MANIFEST = Path(__file__).parent / "manifest.json"
OUT_DIR = Path(__file__).parent.parent.parent / "internal" / "assets" / "helmets"
GRID_COLS = 36
GRID_ROWS = 30  # 4 source cols and 6 source rows per terminal cell:
                # internal/ui/helmet's sextant renderer averages a 2x2
                # subsample block into each of a cell's 6 sub-pixels
                # (2 cols x 3 rows), for a 9x5 terminal footprint - 25%
                # fewer cells than the earlier 10x6 quadrant technique at
                # the same or better perceived detail.
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
    if len(sys.argv) > 1:
        wanted = {int(a) for a in sys.argv[1:]}
        manifest = [e for e in manifest if e["espn_team_id"] in wanted]
        missing = wanted - {e["espn_team_id"] for e in manifest}
        if missing:
            raise SystemExit(f"no manifest entry for team id(s): {sorted(missing)}")
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
