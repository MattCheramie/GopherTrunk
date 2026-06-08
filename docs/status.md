---
layout: page
title: Status & known gaps
description: What ships today and what's still pending in the GopherTrunk pipeline
nav_group: Reference
---

# Status & known gaps

This page tracks what's shipping end-to-end versus where the engine
follows a call but doesn't yet turn it into audio (or doesn't yet
correct every on-air FEC layer). The high-level summary lives in
the [README](https://github.com/MattCheramie/GopherTrunk#status-snapshot);
this page is the long-form reference.

## What ships today

Once a `grant` event lands on the bus, the engine + recorder
pipeline runs end-to-end: voice device is allocated, the composer
pulls IQ → PCM, the recorder writes a WAV (digital-voice protocols
decode through the right vocoder via
`voice.DefaultVocoderForProtocol`), the call is logged to SQLite,
and the API + TUI surfaces all light up. Pure-Go IMBE / AMBE+2
produce intelligible audio. The CC Hunter supervisor and the
conventional FM scanner are constructed by `cmd/gophertrunk` and
expose their state through `/api/v1/scanner` and the TUI cockpit
panel.

**Every trunked control modulation in the Features table has an
end-to-end IQ → CC chain shipping.** The `ccdecoder` connector
covers all 10 trunked protocols (P25 Phase 1, P25 Phase 2, DMR
Tier III, NXDN, dPMR Mode 3, EDACS, Motorola Type II, LTR,
MPT 1327, TETRA TMO) plus DMR Tier II conventional and YSF /
D-STAR on the amateur side.

**SDRtrunk-parity subsystems.** Outbound call streaming
(Broadcastify Calls / RdioScanner / OpenMHz / Icecast), wideband
baseband recording + offline replay, the GPS / location and
affiliation subsystems, the decoded-message log, and
per-talkgroup stream / record / mute / icon policy all ship and
are covered by the test suite. Analog FM trunking (Motorola
Type II, EDACS, LTR, MPT 1327) decodes voice through the
composer's FM chain.

## Remaining gaps

### Additional SDR hardware

RTL-SDR, HackRF (One / Jawbreaker / Rad1o), Airspy R2 / Mini, and
Airspy HF+ (Discovery / Dual Port / legacy) are all supported by
pure-Go drivers; on-air validation of the HackRF / Airspy / HF+
backends against attached hardware is the documented follow-up.
SDRPlay / USRP / BladeRF need vendor C libraries and are out of
scope for the zero-CGO build.

### Digital-voice composer chains

FM (incl. analog trunking), DMR, and P25 Phase 1 / 2 decode to
audio. NXDN, dPMR, TETRA, YSF, and D-STAR voice chains, plus
EDACS ProVoice, are still bypassed — their calls are followed
and logged but not yet turned into PCM.

### Per-protocol on-air FEC inner layers

Every protocol's `ControlChannel.Process` adapter ships a working
IQ → CC chain. The spec-correct chain is on by default for every
protocol; operators with pre-stripped capture files opt out
per-system. See [opt-in-features.md](opt-in-features.md) for the
full table.

The inner FEC layers still pending real-air validation:

- **NXDN per-protocol interleaver + puncture.** `ViterbiSpec`
  mode runs the full §4.5.1.1 chain; `ViterbiOn` is the simpler
  bare-bones path the older MMDVMHost / DSDcc fixtures use. Both
  are wired through the connector. Calibration against captured
  MMDVMHost transmissions is the step that lands next. The
  4-FSK slicer's peak-deviation reference surfaces as a
  per-system `nxdn_deviation_hz` knob (default 1800 Hz per the
  Common Air Interface). The skip-gated real-air harness at
  `cmd/gophertrunk/integration_cc_nxdn_realair_test.go` runs
  acceptance criteria automatically once a contributor drops a
  `.cfile` + `.metadata.json` pair into `samples/nxdn/`.
- **TETRA on-air recovery margins.** Unit tests round-trip clean
  fixtures end-to-end; on-air recovery margins (Viterbi
  correction depth vs. real co-channel + adjacent-channel
  interference) need a live capture to characterise.
- **DMR 2-slot interleaved voice cadence + embedded-LC labelling.**
  A DMR carrier is 2-slot TDMA, so a real outbound stream
  interleaves both timeslots' bursts. `voice.NewInterleavedDecoder`
  (stride 2) decodes that: it locks each slot's burst A on its own
  voice sync and gathers that slot's B–F 264 dibits apart, emitting
  one superframe per slot tagged by `VoiceSuperframe.Phase`. The
  decoder also reassembles the embedded Link Control carried by
  bursts B–E (EMB split → variable BPTC(128,72) + Hamming(16,11,4)
  rows + 5-bit CRC → `dmr.FLC`) and, on a clean CRC, surfaces the
  call's talkgroup + source on `VoiceSuperframe.LC`, so a phase can
  be bound to a concrete talkgroup — the absolute TS1/TS2 label the
  identical BS-sourced burst-A sync cannot give. The whole chain is
  unit-tested against synthetic interleaved + embedded-LC vectors.
  Two pieces still need a real IQ capture before this replaces the
  single-slot `NewDecoder` on the production voice path: (1) the
  exact same-slot dibit cadence on live BS-sourced air — where a
  CACH may precede each burst, making the stride 288 rather than 264
  dibits; and (2) the exact ETSI embedded-signalling de-interleave
  order, the EMB QR(16,7) FEC (read systematically for now), and the
  5-bit CRC polynomial — all currently internally consistent
  (encode↔decode round-trip) but not yet cross-checked against
  captured traffic. The interleaved + LC path is now wired through the
  production composer behind a per-system opt-in,
  `dmr_interleaved_voice: true`: when set, each DMR voice grant is
  tagged so the composer runs `NewInterleavedDecoder` and routes each
  call to its timeslot by matching the embedded LC's talkgroup (a
  `slotRouter`). It defaults off, so untouched configs keep the
  single-slot decoder. The skip-gated harness
  `internal/voice/composer/dmr_2slot_realair_test.go` (run with
  `-tags integration` and `GOPHERTRUNK_DMR_2SLOT_CFILE`) is where a
  contributor drops a real capture to confirm the on-air constants
  and, once green, promote the interleaved decoder to the default.

### Digital-voice level calibration

Pure-Go IMBE / AMBE+2 emit real audio end-to-end. The comparison
harness at `internal/voice/calibrate/` is ready; reference data
(captured P25 P1 / DMR voice exchanges plus DSD-FME / OP25
decodes at `internal/voice/{imbe,ambe2}/testdata/`) is the
remaining gap. Knox / call-alert AMBE+2 tones (b₁ ∈ [144, 163])
are vendor-specific and stay silent until per-vendor frequency
tables land; operators with a curated table register it via
`ambe2.RegisterPreset`. See [vocoders.md](vocoders.md) for the
licensing posture and sourcing checklist.

---

Recently-shipped items live in [`CHANGELOG.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CHANGELOG.md);
near-term plans live in [Roadmap](roadmap.md).
