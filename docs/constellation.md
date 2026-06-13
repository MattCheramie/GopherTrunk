---
layout: page
title: Constellation panel
description: Live symbol constellation and raw-IQ vector scope for judging modulation quality and identifying signal shape
nav_group: Operate
---

# Constellation panel

The **Constellation panel** (web `/constellation`) renders a live
2D scatter of the signal in the complex plane. It answers two
different questions depending on the **View** you pick:

- **Symbols** (the default) — *how clean is the modulation?* This is
  the true symbol constellation: the receiver's symbol-decision points,
  sampled once per symbol after matched filtering, timing recovery and
  carrier recovery. It is the same modulation-quality picture OP25 shows
  on its Constellation tab.
- **Vector scope (raw IQ)** — *what kind of signal is this?* The
  wideband decimated IQ trajectory, every sample including the
  transitions between symbols. Useful as a general "what does this look
  like" view before committing to a decode pipeline.

## Two views

Switch between them with the **View** control. The difference matters:
a raw-IQ vector scope draws *every* sample (symbol centres **and** the
curved transitions between them), so even a perfect signal looks like a
web of arcs — it is not a constellation. A constellation samples only at
the symbol-decision instants, so a clean signal collapses to a handful
of tight dots. Use Symbols to judge demod/equalizer health; use the
vector scope to identify an unknown signal.

### Symbols view

Pick the demodulator with the **Mode** selector:

- **P25 CQPSK** (LSM / simulcast) — a true complex constellation. A
  clean signal forms **four tight clusters** on the ±45°/±135°
  diagonals; as the eye closes (noise, multipath, an unconverged
  equalizer) the clusters smear toward a central **X**. Amber reference
  rings mark the ideal cluster centres.
- **P25 C4FM** — C4FM is a frequency modulation with no complex
  constellation, so its **four soft levels** (±1, ±3) are plotted on the
  real axis, with amber rings at the ideal positions. C4FM's natural
  quality view is the open 4-level eye on the
  [Symbol scope](symbol-scope.html); the constellation here is the
  level-separation summary.

The symbols stream comes from a parallel receiver the daemon runs on the
selected channel — the same one that feeds the Symbol scope — so it
reflects exactly what the production demod sees.

### Vector scope (raw IQ) view

The X axis is the in-phase (I) component, the Y axis the quadrature (Q),
both normalized to ±1 with ticks at ±0.5 and ±1. Points are drawn
additively in GopherTrunk's sky-blue accent so a dense cluster blooms
toward cyan-white while the noise floor stays dim; the newest samples
are brightest. Reference rings show |z| = 0.5 and |z| = 1.0.

## Getting a usable picture (the DC-spike problem)

An SDR's digital down-converter leaks a residual carrier at 0 Hz — a
**DC spike** that sits exactly at the centre of the band. Anything you
tune to the middle lands right on top of it, and the constellation
collapses into one fat blob. The same zero-IF region also folds a
direct-conversion front end's I/Q-imbalance image back onto a channel
sitting at DC, so even a clean-looking signal can decode with elevated
TSBK-CRC / NID-BCH failures (issue #402).

For the **live control channel** specifically, set `dc_avoid: true` on the
control SDR's `sdr.devices` entry: the daemon then tunes the hardware LO a
fixed offset *below* the control-channel frequency and mixes the channel
back to baseband in the down-converter — so the live decode runs off the
DC spike and its image, exactly like an off-channel capture replays
cleanly. It is off by default; `dc_avoid_offset_hz` pins the offset (0 =
auto, `sample_rate/4`). This is the same offset tuning SDRTrunk/OP25 use.

For the *panels* (Constellation / Eye / Symbol scope), three controls work
around the same problem:

- **Offset** — mixes a frequency offset (relative to the SDR centre)
  down to baseband *server-side, before decimation*, so an off-centre
  control or voice channel is pulled out from under the DC spike and
  rendered as a clean constellation. This is the same trick OP25 uses.
  Drag the slider for a coarse sweep, or type an exact value: the
  **kHz** field takes 1 Hz resolution (so channel grids like 6.25 /
  12.5 kHz land precisely), and the **MHz** field lets you type the
  channel's absolute frequency — the two stay in sync.
- **Hold** — when off, the Offset automatically follows the newest
  active call on the selected SDR (the "last locked channel"), and
  rests on the system's **control channel** when no call is up —
  resolved from config, so the panel defaults onto a decodable channel
  off the centre DC spike instead of the useless SDR centre, and glides
  back to the control channel when a call ends (#557). The label next to
  the checkbox shows *following call* or *on control channel*
  accordingly. Turn Hold on to pin the view on a specific offset and
  stop it from jumping as calls come and go. Editing the Offset or
  pressing **Centre** pins it too. (Devices with no control channel in
  their passband keep the old centre default.)
- **DC block** / **Auto scale** — DC-block subtracts the rolling mean
  to remove any residual offset; auto-scale eases a gain so the cloud
  fills the unit circle (targeting the ~95th-percentile radius, so a
  stray outlier doesn't shrink the whole cloud). Both default on.
- **Zoom** — magnifies the plotted cloud and the dot size together (up to
  8×), so the scatter reads as dots rather than pin-pricks; dial it to
  taste to punch in on the symbol structure. The setting persists across
  visits.

The plot itself is a responsive square that fills the panel column (up to
880 px) rather than a fixed thumbnail, so it renders as large as OP25's —
and it stays crisp at any size because the canvas is drawn at the display's
device-pixel ratio.

Common shapes in the **vector scope** view:

| Shape | Likely signal |
| --- | --- |
| One bright dot off-centre | DC bias / unmodulated carrier |
| Two clusters at ±0.5 + 0i | BPSK |
| Four clusters in a square | QPSK / π/4-DQPSK |
| Two horizontal arcs | 2-FSK |
| Four arcs (top + bottom of unit circle) | C4FM / 4-FSK |
| Diffuse rotating cluster, amplitude-modulated | AM voice |
| Diffuse circle around the origin | Wideband noise / nothing on this frequency |
| Spiral expanding outward | Strong frequency offset (re-tune or fix PPM) |

## How it works

In the **Symbols** view the panel opens
`WS /api/v1/diag/symbols?device=...&proto=<p25-cqpsk|p25-c4fm>&offset=<hz>`.
The daemon channelizes the selected offset and runs a parallel P25 Phase 1
receiver; the CQPSK path emits the post-carrier-recovery complex symbols
(`sym_i`/`sym_q`) and the C4FM path emits the recovered soft levels — both
aligned with the sliced dibits. See the
[Symbol scope](symbol-scope.html) for the shared receiver plumbing.

The **Vector scope** view uses the raw IQ stream:

- The panel opens a WebSocket to
  `WS /api/v1/diag/iq?device=...&rate=2000&offset=<hz>`.
- When `offset` is non-zero the daemon mixes that frequency down to
  baseband with an NCO before decimating, so the requested off-centre
  channel ends up at the origin. The magnitude is clamped to the
  device's Nyquist (`±sample_rate/2`).
- The daemon decimates the SDR's full-rate IQ stream by a stride
  (`input_rate / target_rate`), taking a **box average** over each
  stride window — a crude anti-alias low-pass that keeps wideband
  noise from folding on top of the symbol cloud. The goal is
  visualization, not faithful spectral reconstruction.
- Frames arrive every ~50 ms with ~100 points each; the panel
  keeps a rolling buffer of the last 2000 points and repaints the
  canvas each frame.
- Energy (the average power of the *pre*-decimation chunk in
  dBFS) is stamped on every frame and shown in the tuning line —
  helps tell "signal present" apart from "wideband noise that
  happens to look circular."

## Limitations

- **Visualization, not decode.** The Symbols view runs a real receiver,
  but the panel only *plots* the result — it doesn't log talkgroups or
  play audio. To actually decode, configure the channel as a
  `trunking.systems` entry or `scanner.conventional` channel and let the
  per-protocol pipeline take over.
- **Vector scope decimation is brutal.** The stride decimator throws away
  spectral content above `target_rate / 2`. Wideband signals
  appear smeared. The constellation is most useful for symbol-
  domain signals that have already been narrow-channelized — e.g.
  pointed at a single FM repeater or P25 voice channel — rather
  than a 2.4 MHz wideband capture.
- **Single-SDR-at-a-time.** Each WS subscriber runs its own
  decimator; CPU scales linearly with the number of open panels.
  Fine for the single-operator deployments GopherTrunk targets.

## Implementation

| Path | Role |
| --- | --- |
| `internal/radio/p25/phase1/receiver/receiver.go` | `SymbolSink` — emits the per-symbol complex constellation points (CQPSK path) |
| `internal/scanner/symbolscope/scope.go` | Channelizes + runs the receiver; carries `SymI`/`SymQ` alongside soft/dibits |
| `internal/api/symbols.go` | `SymbolProvider` interface + `WS /api/v1/diag/symbols` handler |
| `internal/dsp/diag/iqstream.go` | `Decimator` — vector-scope path: decimates IQ chunks by stride, computes per-frame energy |
| `internal/api/diag.go` | `DiagProvider` interface + `WS /api/v1/diag/iq` handler |
| `cmd/gophertrunk/diag_provider.go` / `symbol_provider.go` | Daemon-side providers wiring each stream to the iqtap broker |
| `web/src/api/diag.ts` / `web/src/api/symbols.ts` | Typed WS clients with auto-reconnect / backoff |
| `web/src/panels/Constellation.tsx` | Canvas scatter renderer, View/Mode toggles, ideal-cluster markers, zoom |
| `web/src/components/TuningControls.tsx` | Shared offset / frequency / Hold / Centre controls (also used by the Symbol scope) |

The decimator runs on top of the iqtap broker (PR #365), so it
fans out from the same IQ source the trunking decoder is reading
without disturbing decode. Multiple panels open against the same
SDR don't double the SDR's CPU cost — they share the broker
subscription up to the broker's drop-on-full bound.
