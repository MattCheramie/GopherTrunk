---
layout: page
title: Bookmarks & frequency manager
description: UI-managed conventional channel list — marine VHF, NOAA weather, FRS/GMRS, repeater outputs, public-safety fall-backs
nav_group: Operate
---

# Bookmarks & frequency manager

GopherTrunk's **Bookmarks** panel is the operator's UI-managed
shortlist of conventional channels — the frequencies you want one
click away to park the SDR on. Marine VHF Ch 16, NOAA weather
station 1-7, the local 2 m repeater output, a public-safety
conventional fall-back channel, the FRS/GMRS family channels: all
the places that aren't trunked but you still want quick access to.

Bookmarks live in the daemon's SQLite database alongside the call
log and location log. They survive restarts and ride along on the
normal backup workflow (just back up `storage.path`).

## What you get

- A compact table of bookmarks grouped by operator-defined "group"
  tag (`marine`, `weather`, `ham-2m`, `utility`, `public-safety`,
  whatever you want).
- Each bookmark carries name, frequency (Hz), mode (FM, NFM, AM,
  USB, LSB, CW, DMR, P25), optional CTCSS / DCS tone, freeform
  notes, and the group tag.
- Inline create / edit / delete from the web panel.
- REST API at `/api/v1/bookmarks` for scripting bulk imports.
- Live SSE updates — peer edits show up in your browser within a
  few seconds without refreshing.

## REST surface

| Method | Path | Auth | What it does |
| --- | --- | --- | --- |
| `GET` | `/api/v1/bookmarks` | open | Returns all bookmarks. |
| `POST` | `/api/v1/bookmarks` | mutation-gated | Creates one. Body matches the shape returned by GET. `name` and `freq_hz` are required; `mode` defaults to `FM`. |
| `PATCH` | `/api/v1/bookmarks/{id}` | mutation-gated | Updates the row in place. Same body shape as POST. |
| `DELETE` | `/api/v1/bookmarks/{id}` | mutation-gated | Removes the row. Idempotent — second delete returns 404 but doesn't error. |

The mutation gate is the same one every other write endpoint uses;
see [hardening.md](hardening.md#api-authentication) for the
options.

## JSON shape

```json
{
  "id": 1,
  "name": "Marine Ch 16",
  "freq_hz": 156800000,
  "mode": "FM",
  "ctcss_hz": 0,
  "dcs_code": 0,
  "notes": "International distress / calling",
  "group": "marine",
  "created_at": "2026-05-26T12:00:00Z",
  "updated_at": "2026-05-26T12:00:00Z"
}
```

## Bulk import (script)

For a one-shot CSV → bookmarks import (e.g. seeding the daemon
with a 200-row marine + weather + ham-band list you maintain in a
spreadsheet), the REST POST is the easy path:

```sh
TOKEN="$(cat ~/.config/gophertrunk/token)"
awk -F, 'NR>1 {
  printf "{\"name\":\"%s\",\"freq_hz\":%d,\"mode\":\"%s\",\"group\":\"%s\"}\n", $1, $2, $3, $4
}' channels.csv | while read body; do
  curl -fsS -X POST \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$body" \
    http://localhost:8080/api/v1/bookmarks
done
```

## Click-to-tune from the spectrum panel

Shipped. The Spectrum panel polls `/api/v1/bookmarks` every 30 s
and renders the bookmarks list as cyan tick markers across the
top of the waterfall canvas wherever a bookmark's frequency
falls inside the visible band. Out-of-band bookmarks are
silently omitted from the overlay (they're still on the
`/bookmarks` panel).

Clicking anywhere on the waterfall posts to `POST
/api/v1/spectrum/devices/{serial}/tune` with a body of
`{"center_hz": N}` derived from the canvas X position; the
daemon routes the call through the iqtap broker so the SDR
retunes and the new centre frequency is reflected on every
downstream panel (spectrum, constellation, CC Activity) plus
any external rigctld client.

The tune endpoint is gated like every other mutation — daemons
started with `auth.mode: required` reject it without a bearer
token. CORS / auth setup is documented in
[hardening.md](hardening.md).

## What bookmarks are *not*

- **Not the trunked-system scanner.** Trunked systems and their
  control channels live in `trunking.systems` in `config.yaml`;
  bookmarks are conventional-only.
- **Not the conventional FM scanner channel list either** — yet.
  The conventional scanner currently reads from
  `scanner.conventional` in YAML; a follow-up will let it read
  bookmarks too so a single source of truth covers both UI and
  scanner needs.
- **Not encrypted.** Bookmarks are operator notes about
  frequencies, not the kind of thing that needs auth-at-rest.
  Database file ACLs (700 on `storage.path`) are sufficient.
