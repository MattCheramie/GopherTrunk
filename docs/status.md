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
fixed against attached hardware in the field, and all three backends
have hardware-gated harnesses
(`internal/sdr/{airspy,airspyhf,hackrf}/*_real_test.go`, gated on
`GOPHERTRUNK_{AIRSPY,AIRSPYHF,HACKRF}_REAL`; `make test-*-real`).
Remaining: Airspy HF+ has had no attached-hardware exercise — its
harness has never been run against a real unit.
SDRPlay / USRP / BladeRF have no local zero-CGO driver (their vendor
libraries are C), but are reachable over SoapyRemote — USRP X310 and
B210 rigs are field-tested that way.

### Digital-voice composer chains

FM (incl. analog trunking), DMR, P25 Phase 1 / 2, TETRA TMO + DMO
(clean-room ACELP), NXDN, dPMR, and D-STAR decode to audio. TETRA
voice is verified bit-exact against the ETSI EN 300 395-2 reference
codec (via the env-gated harness in
`internal/voice/acelp/etsi_reference_test.go` — the ETSI vectors are
copyrighted and not committed). NXDN, dPMR, and D-STAR are wired
end-to-end through the composer but not yet verified on air: each
chain's AMBE interleave table is a documented placeholder awaiting a
real voice capture (`internal/radio/{nxdn,dpmr,dstar}/voice_ambe.go`
— dPMR anchors on the FS1/FS2 voice syncs and renders through the
AMBE+2 3600x2450 "ambe2-dmr" decoder; D-STAR anchors its 96-bit DV
cadence on the Slow Data sync and renders through the base
AMBE 3600x2400 "ambe2" decoder). TETRA DMO (direct mode,
`protocol: tetra-dmo`) records audio through the same ACELP vocoder
but is experimental: call source / destination identity is not
decoded (recordings file under group 0) and the chain still awaits
its on-air A/B (issue #1003). YSF voice and EDACS ProVoice are still
bypassed — their calls are followed and logged but not yet turned
into PCM.

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
- **P25 Motorola talker alias (issue #773) — DECODED. What remains is Phase 2
  traffic-channel MAC yield on weak air (issue #915).** The proprietary
  Motorola alias cipher was recovered by clean-room reverse engineering and is
  enabled (`motorola.CipherVerified = true`, PR #1123, shipped in v1.0.8): it
  reproduces a held-out reference set byte-for-byte (1242/1242 characters) and
  decodes the real #376 capture (RID 200062) to "CRIO 0062" with a valid
  CRC-16/GSM. Reassembly, SUID framing, the self-delimiting CRC framing and the
  live wiring on BOTH paths (voice composer and `signalling_taps` follower share
  one `MACDispatcher`) are pinned end-to-end by
  `sigfollow.TestDispatcherPublishesRealMotorolaAliasAsReliable` — which also
  caught the last live-path defect: the fragment parsers sliced to the END of
  the payload, and a live 144-bit MAC PDU is one byte longer than the 15-byte
  SDRTrunk dump fixtures, so every fragment carried a trailing byte that
  shifted the cipher region (RID parsed, name dropped). Fragments are now
  fixed-length (64-bit header / 100-bit data). Operator guide:
  `docs/talker-alias.md`. The alias rides FACCH-S MAC PDUs on the Phase 2
  *traffic* channel, so it surfaces exactly as often as a MAC PDU RS-validates
  there — and on the #915 reporter's weak Victorian MMR capture that is still
  rare. The Phase 2 signalling path was rebuilt on the real TDMA air format
  (OP25 / SDRTrunk: S-/I-ISCH timeslot grid, DUID, RS(63,35) errors-and-
  erasures + CRC-12/16, PN44 per timeslot), replacing a working model that
  matched no reference — which is why `mac_rs_valid` sat at 0 on the #915
  capture. The same pass fixed the shared Gardner timing loop (it settled on
  the zero crossing, closing the eye a few seconds into every Phase 2 call)
  and the Network Status Broadcast parser (it installed a wrong descramble
  seed). Synthetic- and reference-verified only; `mac_bursts` / `mac_pdus` in
  the composer's per-call census and the `p25p2 alias ciphertext` log line are
  the instruments for the next on-air run (#915).
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
