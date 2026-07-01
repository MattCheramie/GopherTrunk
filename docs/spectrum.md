---
layout: page
title: Spectrum waterfall
description: The wideband Spectrum panel — a live FFT analyzer and scrolling waterfall over a whole SDR's tuned span, with click-to-tune and bookmark markers
nav_group: Operate
---

# Spectrum waterfall

The **Spectrum panel** (web `/spectrum`) is GopherTrunk's **wideband
FFT view**: a live spectrum analyzer above a scrolling waterfall, drawn
over a whole SDR's tuned span rather than a single channel baseband. It
answers "what's on the air right now, and where?" — the wide-angle
counterpart to the per-channel [Plots](plots.html) scopes.

Unlike the [Mixer](mixer.html) and other plots (which tap a single
channelized receiver), the Spectrum panel streams the SDR's full FFT, so
it stays its own top-level tab instead of a `/plots` sub-tab.

## What you see

Two stacked, width-aligned views sharing one FFT-shifted bin → x mapping,
so a peak in the analyzer lines up vertically with its streak in the
waterfall below:

- **Spectrum analyzer** — the latest frame as a **Power (dBFS) vs
  Frequency** line plot across the tuned span.
- **Waterfall** — the last few hundred frames stacked newest-on-top, each
  frame one row. dBFS is colour-mapped on a fixed **[−100, 0] dB** range
  with a **blue → cyan → yellow → red** palette, so a hot carrier reads
  red against a blue noise floor.

Hovering reads out the **frequency and dBFS** under the cursor, and any
[bookmarks](bookmarks.html) whose frequency falls in view are drawn as
markers along the top of the waterfall.

| What you see | Reading |
| --- | --- |
| A red vertical streak | A steady carrier — click it to tune |
| Bursty streaks that come and go | Intermittent transmissions (voice/data) |
| A raised, textured noise floor | Broadband noise, gain too high, or a nearby overload |
| A flat blue field | Quiet span — nothing active, or the SDR isn't streaming |

## Click-to-tune

Clicking anywhere on the waterfall or analyzer retunes the selected SDR to
that frequency: the panel POSTs to
`/api/v1/spectrum/devices/{serial}/tune`, and downstream panels
(constellation, CC Activity, the per-channel scopes) follow the new centre.
Bookmark markers make known control/voice channels a single click away —
see [Bookmarks](bookmarks.html) for how the overlay is populated.

## Controls

Pick the SDR from the daemon's broker pool; the panel opens a stream to
that device and starts filling the waterfall. Because the view is the
device's full span, there is no per-channel **Offset/Mode** here — those
belong to the single-channel [Plots](plots.html). Use click-to-tune (or a
bookmark) to move the centre frequency.

## How it works

- The panel discovers SDRs with `GET /api/v1/spectrum/devices`, then opens
  `WS /api/v1/spectrum/stream?device=...&fps=...&bins=...`. Frames arrive
  as JSON text messages at the negotiated rate (default **10 fps**, a
  **4096-bin** FFT; both are query-tunable, and the server pings every
  30 s to keep idle proxies from reaping the socket).
- Each frame is an FFT-shifted dBFS array computed by the shared
  `internal/dsp/spectrum` helpers — the same `PowerDB` normalization the
  [Mixer](mixer.html) plot uses, so absolute dBFS levels are comparable
  across the two views.
- Retunes go through `POST /api/v1/spectrum/devices/{serial}/tune`, gated
  by the API's write guard.

## Implementation

| Path | Role |
| --- | --- |
| `internal/dsp/spectrum/spectrum.go` | `PowerDB` — shared windowed-FFT dBFS helper |
| `internal/api/spectrum.go` | `devices` / `stream` / `tune` handlers |
| `internal/api/server.go` | Route registration for `/api/v1/spectrum/*` |
| `cmd/gophertrunk/spectrum_provider.go` | Daemon provider over the SDR broker pool |
| `web/src/api/spectrum.ts` | Device list, stream, and tune API client |
| `web/src/panels/Spectrum.tsx` | Spectrum-analyzer line plot + scrolling waterfall canvas |
