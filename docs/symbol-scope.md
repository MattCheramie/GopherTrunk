---
layout: page
title: Symbol scope panel
description: Live oscilloscope of the demodulated symbol stream — the pre-slicer soft waveform and the sliced dibit decisions — GopherTrunk's take on OP25's Symbol plot
nav_group: Operate
---

# Symbol scope panel

The **Symbol scope** (web `/symbols`) renders a live oscilloscope of
the demodulated symbol stream coming off whichever SDR you pick — the
"are the symbols clean?" view, complementing the Constellation panel's
IQ scatter. It is GopherTrunk's take on OP25's *Symbol* plot.

## What you see

The X axis is symbol index (time, left→right); the Y axis is the
symbol level. Two render modes are chosen automatically:

- **P25 C4FM** streams the **pre-slicer soft waveform** — the
  FM-discriminator + matched-filter + clock-recovered samples, just
  before slicing. A healthy 4-level channel reads as ~4 noisy
  horizontal bands; faint rails mark the mean of each decided level.
- **P25 CQPSK** has no soft tap yet, so it streams the **sliced dibit
  decisions** only, drawn as discrete rows.

Brighter (additively-blended) regions are denser symbol populations.
The trace is GopherTrunk's sky-blue accent, matching the Constellation
and Spectrum panels.

## Getting a usable picture (offset / Hold / follow)

Like the Constellation panel, the scope taps the SDR's wideband IQ, so
a channel sitting on the SDR centre is buried under the DC spike. The
same controls work around it:

- **Mode** — the receiver / demod path (P25 C4FM or P25 CQPSK).
- **Offset / Freq** — tunes a channel sitting at this offset (kHz,
  relative to the SDR centre) down to baseband before the receiver
  channelizes it, lifting an off-centre control/voice channel clear of
  the DC spike. Type the **kHz** offset (1 Hz resolution, so 6.25 /
  12.5 kHz grids land exactly) or the channel's absolute **MHz**
  frequency — the two stay in sync. The tuned frequency is shown as soon
  as an SDR is selected, before any symbols decode.
- **Hold** — when off, the Offset automatically follows the newest
  active call on the selected SDR (the "last locked channel"); turn
  Hold on (or edit the Offset / press **Centre**) to pin it.

## How it works

- The panel opens a WebSocket to
  `WS /api/v1/diag/symbols?device=...&proto=...&offset=...`.
- The daemon attaches a **parallel** decode to the SDR's iqtap broker —
  the same broker the Constellation and Spectrum panels use — so it
  never disturbs the production control-channel decoder. The parallel
  decode reuses the production down-converter
  (`ccdecoder.Downconverter`, channelizing to 48 kHz for the C4FM
  family) and the production P25 Phase 1 receiver.
- The receiver's `SoftSink` (pre-slicer waveform) and `DibitSink`
  (sliced decisions) feed `internal/scanner/symbolscope`, which batches
  ~256 symbols per frame and ships them over the WS. Soft and dibit are
  aligned index-for-index on the C4FM path.

## Offline (SigLab)

The offline [SigLab](siglab) analyzer has the same view. Run a capture
with **collect IQ diag** + **capture IQ** enabled and the Result carries
an aligned symbol series (`IQTaps.symbol_dibits` / `symbol_soft`); the
SigLab Results page renders it as a **Symbol scope** card next to the eye
diagram. The live panel and the offline viz share the same props
contract, so they read identically.

## Limitations

- **P25 Phase 1 only, today.** C4FM gets the soft waveform + dibits;
  CQPSK gets dibits only. TETRA and the rest of the C4FM family
  (DMR / NXDN / YSF / D-STAR) — and a soft waveform for them — land as
  the per-receiver soft taps ship.
- **Not a demodulator UI.** This is a diagnostic view; to actually
  decode, configure the channel as a `trunking.systems` entry and let
  the production pipeline take over.
- **One engine per panel.** Each open scope runs a full
  down-converter + receiver on its own broker subscription; CPU scales
  with the number of open scopes (the broker fan-out never
  back-pressures the production decoder).

## Implementation

| Path | Role |
| --- | --- |
| `internal/scanner/symbolscope/scope.go` | `Engine` — DDC + P25 receiver + soft/dibit taps → `Frame`s |
| `internal/api/symbols.go` | `SymbolProvider` interface + `WS /api/v1/diag/symbols` handler |
| `cmd/gophertrunk/symbol_provider.go` | Daemon provider — wires the engine to the iqtap broker per subscriber |
| `web/src/api/symbols.ts` | Typed client with auto-reconnect / backoff |
| `web/src/components/SymbolScopeChart.tsx` | Canvas scope renderer (shared contract with the SigLab viz) |
| `web/src/components/TuningControls.tsx` | Shared offset / frequency / Hold / Centre controls (also used by the Constellation) |
| `web/src/panels/SymbolScope.tsx` | Panel: SDR + mode + offset/Hold/follow controls |
