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
and the clean-room TETRA ACELP vocoder produce intelligible
audio. The CC Hunter supervisor and the
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

RTL-SDR, HackRF (One / Jawbreaker / Rad1o / Pro), Airspy R2 / Mini,
and Airspy HF+ (Discovery / Dual Port / legacy) are all supported by
pure-Go drivers with mock-transport unit tests. HackRF (Pro board-ID
detection, `fpga_dc_block`, `dc_avoid`, `rf_amp`) and Airspy (macOS
async bulk-IN rework, native-rate behaviour) have been exercised and
fixed against attached hardware in the field, and Airspy has a
hardware-gated harness (`internal/sdr/airspy/airspy_real_test.go`,
`GOPHERTRUNK_AIRSPY_REAL`). Remaining: Airspy HF+ has had no
attached-hardware exercise, and HackRF has no hardware-gated harness.
SDRPlay / USRP / BladeRF have no local zero-CGO driver (their vendor
libraries are C), but are reachable over SoapyRemote — USRP X310 and
B210 rigs are field-tested that way.

### Digital-voice composer chains

FM (incl. analog trunking), DMR, P25 Phase 1 / 2, TETRA TMO + DMO
(clean-room ACELP), and NXDN decode to audio. TETRA voice is verified
bit-exact against the ETSI EN 300 395-2 reference codec (via the
env-gated harness in `internal/voice/acelp/etsi_reference_test.go` —
the ETSI vectors are copyrighted and not committed); NXDN is wired
end-to-end through the composer but not yet verified on air. TETRA
DMO (direct mode, `protocol: tetra-dmo`) records audio through the
same ACELP vocoder but is experimental: call source / destination
identity is not decoded (recordings file under group 0) and the
chain still awaits its on-air A/B (issue #1003). dPMR, YSF, and
D-STAR voice chains, plus EDACS ProVoice, are still bypassed — their
calls are followed and logged but not yet turned into PCM.

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
- **TETRA on-air recovery margins — now characterised (TMO).** A
  real marginal-signal control-channel capture is committed at
  `internal/scanner/ccdecoder/testdata/tetra_cc_sync_loss_2s_144k.cs16`
  and pinned in CI by
  `internal/scanner/ccdecoder/pipelines_tetra_equalizer_test.go`:
  the marginal regime (~10 dB in-channel SNR) decodes ~12% of its
  BSCH without equalization and ~100% with the blind `SnapshotCMA`
  equalizer, and soft-decision TCH/S decoding lifts CRC-valid voice
  yield ~1.9× across the reporter's captures. What remains
  TETRA-side is the **DMO** on-air A/B (issue #1003) and a
  validating `.metadata.json` sidecar for `samples/tetra/` so the
  skip-gated TMO real-air harness runs.
- **P25 Phase 2 traffic-channel MAC descramble (issues #915, #773, #451).**
  The superframe now locks on real air under any dibit rotation (the
  differential-H-DQPSK residual-carrier ambiguity fixed in #943), so the ISCH
  classifies the MAC sub-frames correctly and `mac_pdus > 0`. What still does
  **not** work is recovering a valid MAC PDU from the payload: `mac_rs_valid`
  stays 0, so the clear-MAC source RID (#915) and talker alias (#773) never
  surface. Replaying the reporter's Victorian MMR (WACN `0xBEE00`) Phase 2 voice
  `.cfile` offline ruled out — against real bytes — every tractable hypothesis:
  the payload's trellis path-metric is ~42/73 (random) undescrambled while the
  un-scrambled sync + ISCH decode clean, so the payload *is* transformed; but
  descrambling it (both **after** the trellis at per-slot offsets
  {`index·360`, `index·144`, `0`} and **before** the trellis at offsets
  {`0`, superframe `index·360+64`}) across **all 16.7 M** `(WACN, SysID, NAC)`
  identity seeds never validates the outer RS and only nudges the best trellis
  metric 41.8 → 33.8 (still garbage). A seed-independent structure test
  (XOR two same-slot payloads, which must share the sequence under any
  per-superframe-restarting PN44 model — the sequence then cancels and the linear
  trellis code makes the XOR a clean codeword) stays at ~41 error, so the
  transform is **not** a fixed positional PN44 XOR keyed by the network identity.
  What remains is a continuous / counter-seeded scrambler or a MAC FEC chain
  (trellis params / interleaver / RS-before-conv order) different from what GT
  models — pinning it needs the TIA-102.BBAC §7.x scramble/FEC definition or an
  SDRtrunk / OP25 P25P2 reference to cross-check. The site's real System ID + NAC
  would remove the seed unknown and separate a seed-formula bug from a
  scramble-structure bug. The `mac_rs_valid` census counter (#934) is the
  before/after metric once a candidate mapping is tried, and the RS-valid gate
  keeps the mis-decoded bytes from injecting a bogus source RID in the meantime.
- **DMR 2-slot interleaved voice — now the Tier II conventional &
  Tier III default (issue #644).** A DMR carrier is 2-slot TDMA, so a real
  outbound stream interleaves both timeslots' bursts. The single-slot
  `voice.NewDecoder` splices the two slots together into garbled,
  encrypted-sounding audio — the #644 report. `voice.NewInterleavedDecoder`
  decodes the carrier correctly: it locks each slot's burst A on its own
  voice sync and gathers that slot's B–F by the same-slot stride,
  emitting one superframe per slot tagged by `VoiceSuperframe.Phase`.
  It now **auto-detects the on-air same-slot cadence** per call —
  264 dibits (no inter-burst CACH) vs 288 (a 12-dibit CACH precedes
  each burst on live BS-sourced outbound air) — by slicing bursts B–E
  at each candidate and locking onto the one that reassembles a
  CRC-valid embedded Link Control (a wrong cadence cannot). The decoder
  also surfaces that LC's talkgroup + source on `VoiceSuperframe.LC`, so
  a phase binds to a concrete talkgroup — the absolute TS1/TS2 label the
  identical BS-sourced burst-A sync cannot give. The chain is unit-tested
  against synthetic interleaved + embedded-LC vectors at **both** cadences
  (`TestInterleavedDecoderAutoDetects{CACH,NoCACH}Cadence`). A cadence chosen
  only by FEC quality (no LC yet) is held **provisionally**: a later CRC-valid
  LC — or a clear FEC winner at a different stride — overrides a wrong early
  guess and re-locks the correct cadence, so one bad guess no longer garbles
  the rest of the call
  (`TestInterleavedDecoderLCOverridesWrongProvisionalCadence`).
  The interleaved + LC path is the **default for DMR Tier II conventional
  and Tier III** (#644, extended to Tier II after a field report of garbled
  "DJ-scratch" audio on a `dmr-tier2` site — the same single-slot/2-slot
  mismatch): the daemon tags those systems' voice grants so the composer
  runs `NewInterleavedDecoder` and routes each call to its timeslot by the
  embedded LC's talkgroup (a `slotRouter`). DMR Tier I is direct-mode
  simplex (genuinely single-slot) and stays on `NewDecoder`.
  `dmr_interleaved_voice` is a tri-state override (unset = protocol default;
  true/false to force).
  One piece still wants a real IQ capture to cross-check: the exact ETSI
  embedded-signalling de-interleave order, the EMB QR(16,7) FEC (read
  systematically for now), and the 5-bit CRC polynomial — currently
  internally consistent (encode↔decode round-trip) but not yet validated
  against captured traffic. The skip-gated harness
  `internal/voice/composer/dmr_2slot_realair_test.go` (run with
  `-tags integration` and `GOPHERTRUNK_DMR_2SLOT_CFILE`) is where a
  contributor drops a real capture to confirm those remaining constants.
  Because that embedded LC is capture-pending, the `slotRouter` no longer
  hard-drops a call when the LC never decodes: a matching LC still binds (and
  corrects) the slot, but after a short grace window with no LC it falls back
  to the active slot's phase so audio still records instead of producing empty
  files (#644). The decode-quality log reports `lc_superframes` and notes once
  when a call records via the phase fallback, so a capture that exercises the
  fallback is easy to spot.

- **AMBE+2 synthesis parity with IMBE (#644 follow-up).** Once the
  timeslot fix made DMR speech intelligible it sounded metallic ("tin
  can") — the buzz from fully phase-coherent voiced synthesis. The
  AMBE+2 decoder (DMR plus P25 Phase 2 / NXDN / dPMR) now runs
  the same three post-synthesis stages the IMBE decoder already had:
  §6.3 voiced-phase regeneration (`mbe.SynthVoicedDispersed`, the
  de-buzz, scaled by the unvoiced-harmonic fraction), a DC-removal
  high-pass (`mbe.DCBlock`) ahead of the AGC, and error-rate adaptive
  smoothing (`mbe.Smoother`). The DMR voice chain forwards the per-frame
  Golay corrected-bit count (`errAwareRawSink` →
  `voice.ErrorAware.SetFrameErrors`) to drive the smoother. Clean
  fully-voiced frames are bit-identical to before; only the metallic
  timbre changes.

### Digital-voice level calibration

Pure-Go IMBE / AMBE+2 emit real audio end-to-end. The comparison
harness at `internal/voice/calibrate/` (CLI: `cmd/voice-calibrate`)
is ready, and the AMBE+2 capture fixture
(`internal/voice/ambe2/testdata/dmr-voice.raw`) is committed. Still
missing are the DSD-FME / OP25 **reference WAVs** —
`internal/voice/imbe/testdata/p25-p1-voice{.raw,-dsdfme.wav}` and
`internal/voice/ambe2/testdata/dmr-voice-dsdfme.wav` — which also
gate the final quality sign-off of the default-on spec-faithful
§6.2 spectral-amplitude enhancement
(`recordings.spec_amplitude_enhance`). Knox / call-alert AMBE+2 tones (b₁ ∈ [144, 163])
are vendor-specific and stay silent until per-vendor frequency
tables land; operators with a curated table register it via
`ambe2.RegisterPreset`. See [vocoders.md](vocoders.md) for the
licensing posture and sourcing checklist.

---

Recently-shipped items live in [`CHANGELOG.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/CHANGELOG.md);
near-term plans live in [Roadmap](roadmap.md).
