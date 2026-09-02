# CLI / Agent Mode

Golazo ships with a small set of subcommands that emit JSON to stdout. They are intended for agents (Claude Code, Codex, scripts, CI) that need structured NCAA college football data without driving the TUI.

The default `golazo` invocation still opens the TUI — the subcommands below are additive.

> This fork's CLI is backed by ESPN's college-football site API (`internal/espncfb`), not upstream's FotMob (soccer) API. See the [root README](../README.md) for the fork's full framing and credit to the original author, [@0xjuanma](https://github.com/0xjuanma).

## When to use Golazo

Use this tool when the user asks about NCAA college football games. Map their question to the right command:

| User asks about... | Command |
|---|---|
| Games happening right now | `golazo live` |
| Today's results (already finished) | `golazo finished --days 1` — today only, usually empty outside Thu-Sat |
| Today's full slate (finished + still-to-come) | `golazo finished --include-upcoming` |
| Results over the last N days (≤8) | `golazo finished --days N` |
| Details for a specific game (scoring plays, situation, leaders, momentum, box score) | `golazo match <id>` |
| Which FBS conferences are tracked / what conference IDs exist | `golazo leagues` |

If the user's question doesn't map to one of the above, this tool likely cannot answer it. Golazo does not expose: full-season standings history, individual player season stats, recruiting/transfer news, or fixtures beyond the current lookback window. College football games cluster on **Thursday–Saturday** — an empty result on a Monday or Tuesday is correct, not a failure.

## Quick start (worked example)

The reliable agent flow is **discover → list**. Stop at the list level — `live` and `finished` already return all the per-game metadata an agent typically needs (teams, score, status, kickoff time, conference).

```bash
# 0. (Optional but recommended) Self-discover the CLI contract
golazo capabilities | jq '.data[0].commands'

# 1. Discover which conferences are tracked
golazo leagues

# 2. Get the last 8 days' full slate — reaches back to the prior weekend regardless of today's weekday
golazo finished --include-upcoming | jq '.data[] | {
  conference: .league.name,
  status,
  home: .home_team.name,
  away: .away_team.name,
  score: (if .home_score != null then "\(.home_score)-\(.away_score)" else null end),
  kickoff_utc: .match_time
}'
```

For scoring plays / situation / leaders: use `golazo match <id> --mock` against the bundled mock IDs (2001, 2002, ...) to validate your jq pipeline, then use a real ID from `live` or `finished`.

## Subcommands

| Command | Description |
|---|---|
| `golazo live` | Today's in-progress games across all FBS conferences |
| `golazo finished [--days N] [--include-upcoming]` | Finished games over the last N days (1..8, default 8); use `--include-upcoming` to also include today's not-yet-started games |
| `golazo match <id>` | Full game details (scoring plays, situation, leaders, momentum, box score) by ESPN event ID |
| `golazo leagues` | Every hardcoded FBS conference (ESPN "group") |
| `golazo capabilities` | Machine-readable contract describing every subcommand, flag, error code and env var — call this once at session start to self-discover the CLI |

### Common flags

| Flag | Description |
|---|---|
| `--mock` | Use bundled mock data, no network. **Note:** the mock fixtures are inherited from upstream and are soccer-flavored (Chelsea/Arsenal-style teams) — they exercise the JSON shape correctly but the team/league names won't read as college football. |
| `--debug` | Emit debug logs to stderr |
| `--timeout <dur>` | Overall request timeout (default `15s`) |
| `--pretty` | Indent JSON output |

`leagues` and `capabilities` make no network calls, so they only expose `--pretty`.

## JSON contract

### Success envelope

```json
{
  "status": "ok",
  "count": 2,
  "data": [ ... ]
}
```

Single-item responses (e.g. `match <id>`, `capabilities`) still use a `data` array with `count: 1`. **This is intentional** — every subcommand returns the same envelope shape so agents can write a single parser. Access singleton results with `.data[0]`:

```bash
golazo capabilities | jq '.data[0].tool'            # → "golazo"
golazo match 2001 --mock | jq '.data[0].home_team'  # → match details
```

### Degraded envelope

`finished` over multiple days may partially fail. When at least one day succeeds, the envelope is flagged degraded with the failing dates listed:

```json
{
  "status": "ok",
  "degraded": true,
  "failed_dates": ["2026-08-29"],
  "count": 12,
  "data": [ ... ]
}
```

### Error envelope

Errors go to **stderr**, stdout stays empty:

```json
{
  "status": "error",
  "code": "not_found",
  "message": "no match found for id 99999999"
}
```

Error codes: `invalid_args`, `not_found`, `upstream_error`, `timeout`, `offline`.

CLI-level failures (typo'd subcommand, unknown flag, bad flag value) also flow through this envelope as `invalid_args` (exit code 2). Agents can always parse stderr as JSON when the exit code is non-zero.

## Data schema

Every command's `data` array contains one of three object shapes. All field names are stable across calls. Fields marked `null when ...` are present but null in those states — agents should always nil-check.

### `Match` (returned by `live`, `finished`)

```yaml
id:          int        # ESPN event ID — pass to `golazo match`
league:
  id:        int         # ESPN conference ("group") ID
  name:      string      # e.g. "SEC", "Big Ten"
  country:   string       # always "NCAA"
home_team:
  id:        int
  name:      string      # full name, e.g. "Alabama Crimson Tide"
  short_name: string     # abbreviated, e.g. "Alabama"
away_team:   { same shape as home_team }
status:      string      # one of: "live" | "finished" | "not_started" | "postponed" | "cancelled"
home_score:  int|null    # null when status == "not_started"
away_score:  int|null    # null when status == "not_started"
match_time:  string      # RFC3339 kickoff timestamp in UTC, e.g. "2026-09-05T23:30:00Z"
live_time:   string|null # null unless status == "live"; e.g. "08:41 - 3rd"
round:       string      # usually empty for regular-season CFB games
page_url:    string      # unused by this fork's ESPN provider; always empty
```

### `MatchDetails` (returned by `match`)

`MatchDetails` embeds every `Match` field above. Fields inherited from upstream's soccer shape (`home_lineup`, `home_formation`, `home_xg`, `aggregate_score`, `half_time_score`, `penalties`, etc.) are always empty/omitted for CFB — the ESPN provider never populates them. The fields that matter for football:

```yaml
events:                          # scoring plays only
  - description:    string       # full play text, e.g. "A.Milroe 12 Yd Run (Touchdown)"
    type:            string      # "touchdown" | "field_goal" | "safety"
    display_minute:  string      # e.g. "08:41 - 3rd" (game clock + quarter, not a running minute)
    team:             Team
    timestamp:        string     # RFC3339
situation:                       # current down/distance/field position; null unless status == "live"
  down:              int         # 1-4
  distance:          int         # yards needed for a first down
  down_distance_text: string     # e.g. "2nd & 6"
  possession_text:   string      # e.g. "USC 44"
  is_red_zone:       bool
  home_timeouts / away_timeouts: int
  last_play:         string
leaders:                         # player statistical leaders, grouped by category
  - key:             string      # e.g. "passingYards"
    label:           string      # e.g. "Passing Yards"
    home_leaders / away_leaders:
      - player_name:  string
        display_value: string    # e.g. "25/29, 286 YDS, 2 TD"
        value:         number
momentum:                        # win-probability series, one point per play
  - play_id:          string
    home_win_pct:      number    # 0.0-1.0
    period:            int
statistics:                      # box score
  - { key, label, home_value, away_value }   # e.g. total yards, turnovers
venue:              string
winner:             "home"|"away"|null
```

### `League` (returned by `leagues`)

```yaml
id:           int      # ESPN conference ("group") ID
name:         string   # e.g. "SEC", "ACC"
country:      string   # always "NCAA"
country_code: string   # always empty
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | Upstream / unknown error |
| `2` | Invalid arguments |
| `3` | Not found |
| `4` | Timeout |
| `5` | Offline (network disabled via env) |

## Environment variables

| Var | Effect |
|---|---|
| `GOLAZO_AGENT=1` | Forces compact JSON, enables stderr debug logging |
| `GOLAZO_OFFLINE=1` | Refuses any network call; subcommands return `offline` unless `--mock` is set |

### Recommended agent invocation

For any non-human caller, always set `GOLAZO_AGENT=1`:

```bash
GOLAZO_AGENT=1 golazo <subcommand> [flags]
```

This single env var:

- Forces compact (single-line) JSON regardless of `--pretty` — lower token cost
- Routes all debug logging to stderr so stdout stays clean for `jq` pipelines
- Acts as a future-proof "I am not a human" signal — if Golazo ever grows interactive prompts or color escapes, this flag will suppress them

A repository-root `tool.json` manifest also describes the CLI for agent indexers; see the top of [the repo](https://github.com/jaslrobinson/golazo/blob/main/tool.json).

## Determinism guarantees

Agents that diff outputs across calls (e.g. "did the score change since five minutes ago?") need to know what's stable and what's not. Within a single game:

| Stable across calls | Changes during/after a game |
|---|---|
| `id`, `league`, `home_team`, `away_team` | `status` |
| `match_time` (kickoff time), `venue` | `home_score`, `away_score` |
|  | `live_time`, `situation` |
|  | `events[]` (appends as scoring plays happen) |
|  | `statistics[]`, `momentum[]` |

`live` and `finished` sort by `match_time` then `id` for deterministic ordering — repeated invocations produce diffable output. `leagues` sorts by conference ID instead (it has no `match_time`).

## Examples

### Basic invocations

```bash
# Live games, compact JSON
golazo live

# Finished games over the last 3 days, indented
golazo finished --days 3 --pretty

# Full slate over the default 8-day lookback (reaches the prior weekend)
golazo finished --include-upcoming

# Single game details
golazo match 2001 --mock

# Discover FBS conference IDs to interpret results
golazo leagues

# Agent mode + offline safety in CI
GOLAZO_AGENT=1 GOLAZO_OFFLINE=1 golazo live --mock
```

### jq recipes

Most agent flows pipe Golazo's output through `jq`. These are the patterns worth memorizing:

```bash
# One-line live score summary
golazo live | jq -r '.data[] | "\(.home_team.name) \(.home_score)-\(.away_score) \(.away_team.name) [\(.live_time)]"'

# Filter by conference name (case-insensitive match)
golazo finished --include-upcoming \
  | jq '[.data[] | select(.league.name | test("SEC"; "i"))]'

# Filter by status (e.g. only what's still to come)
golazo finished --include-upcoming \
  | jq '[.data[] | select(.status == "not_started")]'

# Extract just IDs, ready for chaining to `match`
golazo finished --days 3 | jq -r '.data[].id'

# Check whether the result was degraded (partial-failure-aware retry decision)
golazo finished --days 8 | jq '{ok: ((.degraded // false) | not), failed: (.failed_dates // [])}'

# Scoring plays only, ordered by game clock (mock ID example)
golazo match 2001 --mock \
  | jq '.data[0].events'
```

## Notes

- Stdout receives **only** the JSON envelope. All logs go to stderr — safe to pipe through `jq`.
- List output is sorted deterministically (`match_time` then `id`) so repeated invocations diff cleanly.
- The TUI experience is unchanged — no flags here alter the interactive default.

## Failure modes & retry policy

Use this table to decide whether to retry, fix the call, or give up.

| Error code | Exit | Typical cause | Should agent retry? |
|---|---|---|---|
| `invalid_args` | `2` | Bad flag value (e.g. `--days 99`, non-numeric match ID) | **No** — fix the call. Retrying will keep failing. |
| `not_found` | `3` | Unknown match ID (mock mode), or match has no data | **No** — pick a fresh ID via a list call. |
| `timeout` | `4` | Upstream slow or network congested | **Yes**, with a larger `--timeout` (e.g. `--timeout 30s`). |
| `upstream_error` | `1` | ESPN 4xx/5xx, network failure, bot-detection block | **Once** — transient errors recover. If it persists across retries, [file an issue](https://github.com/jaslrobinson/golazo/issues) — see [Known limitations](#known-limitations). |
| `offline` | `5` | `GOLAZO_OFFLINE=1` is set | **No** — unset the env var, or pass `--mock` for synthetic data. |

The exit code is the most reliable retry signal. The `code` field in the error envelope on stderr says the same thing in machine-readable form; agents should prefer the exit code (no JSON parsing required).

## "No matches" vs "failure" semantics

Two outputs look superficially similar but mean very different things:

```json
{"status":"ok","count":0,"data":[]}            // success, just nothing today
{"status":"error","code":"upstream_error",...} // failure on stderr, exit 1
```

A `count: 0` result means **the request succeeded and there genuinely are no games** matching the criteria (off-season, no games at 4am, a weekday with no Thu-Sat slate). It is **not** a silent failure. Do not retry on `count: 0` — you will get the same answer.

Conversely, partial failures (`finished` over multiple days where some days fetched and others didn't) are surfaced as `degraded: true` with a `failed_dates` array — those are still exit code 0, but agents can choose to retry just the failed dates.

## Rate limiting

There is no explicit rate limiter in front of ESPN's site API in this fork (upstream's FotMob-specific 200ms/10-concurrent limiter only applies to the `fotmob` package, which the CLI no longer uses). If ESPN itself rate-limits or bot-blocks a client, that surfaces as `upstream_error`. There is **no** explicit `rate_limited` error code today.

## Known limitations

### `match <id>` is verified live

`match <id>` reliably returns full details from a cold call with a real ID from `live`/`finished` — ESPN's summary endpoint takes the event ID directly, no page-slug lookup required. An earlier version of this fork's `league` field on `match <id>` came back empty (ESPN's `/summary` header team object doesn't reliably carry `conferenceId`, unlike `/scoreboard`); fixed by falling back to the boxscore team object's `conferenceId`, which does carry it.

### `--mock` data is soccer-flavored

The bundled mock fixtures (`internal/data/mock_*.go`) are inherited unchanged from upstream and describe Premier League-style matches, not college football. `--mock` is still useful for exercising the JSON envelope shape and validating a `jq` pipeline — just expect Chelsea/Arsenal-style team names in the output, not CFB teams. CFB-flavored mock fixtures are a known gap, not yet done in this fork.

### Debug logging is sparse on list endpoints

`--debug` and `GOLAZO_AGENT=1` emit one debug line per outgoing ESPN request (the URL). The list endpoints (`live`, `finished`, `leagues`) still don't log per-item processing detail. This is by design — agents are expected to interpret the JSON envelope (including `degraded` / `failed_dates`), not stderr logs.
