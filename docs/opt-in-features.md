---
layout: page
title: Opt-in features
description: Defaults vs flags for protocol decoders, daemon surfaces, and patent-encumbered backends
nav_group: Reference
---

# Opt-in / opt-out features

GopherTrunk's protocol decoders and daemon-level surfaces have a
mix of:

- features that **were** opt-in and have since been **flipped on
  by default** (operators can still opt out per-protocol);
- features that **are still opt-in** because the default is
  correct for headless / server deployments;
- features that **stay opt-in by their nature** (per-site
  signaling, patent-encumbered backends, CI-only tests).

This document is the operator's reference for that landscape: what
default applies, why, and (where relevant) how to opt out.

## Contents

1. [Protocol-layer FEC defaults](#1-protocol-layer-fec-defaults) — on by default, opt-out per protocol
2. [Receiver clock recovery](#2-receiver-clock-recovery) — on by default, opt-out per protocol
3. [Daemon-level features](#3-daemon-level-features) — mix of on / off / auto-detect
4. [Build-time options](#4-build-time-options) — patent / CI gates that stay opt-in permanently
5. [Remaining FEC follow-ups](#5-remaining-fec-follow-ups) — per-protocol inner FEC layers pending spec / capture data
6. [How to verify what's currently enabled](#6-how-to-verify-whats-currently-enabled)

---

## 1. Protocol-layer FEC defaults

**On by default for every protocol.** The ccdecoder connector at
[`internal/scanner/ccdecoder/pipelines.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/scanner/ccdecoder/pipelines.go)
always runs the per-protocol `Parse*Mode` function over the
configured YAML value — empty strings map to the spec-correct
on-default. Operators with pre-stripped capture files (DSD-FME
`-r` dumps, OP25 fixtures, MMDVMHost / DSDcc test data) opt out
per-system with `<key>: off`.

| Protocol | YAML key | Default | Opt-out string |
| --- | --- | --- | --- |
| TETRA | `tetra_channel_coding` | `ChannelCodingOn` (full §8.3.1 chain) | `off` |
| LTR FCS | `ltr_fcs_mode` | `FCSOn` | `off` |
| LTR Manchester | `ltr_manchester_mode` | `ManchesterSoft` | `off` / `nrz` |
| P25 Phase 2 (inner trellis) | `p25_phase2_trellis_mode` | `TrellisOn` | `off` |
| P25 Phase 2 (outer RS) | `p25_phase2_rs_mode` | `RSOff` (RS(24, 16, 9) coder shipped per TIA-102.BAAA-A §5.9) | `on` flips detection-only RS verification on (drop any PDU with a non-zero syndrome); `correct` (alias `fix`/`ecc`) runs the RS as a bounded-distance error *corrector* — repairs up to t=4 symbol errors before parsing so marginal-SNR MAC PDUs still yield a source RID (weak-frame recovery for issue #915). `correct` supersets `on`; a corrected PDU is gated on a recognised opcode so a wrong descramble phase can't be miscorrected into a bogus RID |
| P25 Phase 2 (PN44 scrambler) | `p25_phase2_scrambler_mode` | `ScramblerOff` (PN44 descrambler shipped per TIA-102.BBAC-1 §7.2.5 with per-burst slot-offset blind probe; seed derived from per-system WACN + SystemID + NAC) | `on` flips PN44 descrambling on at the configured offset / `probe` walks the 12 Figure 7-5 slot offsets and accepts whichever RS-verifies |
| P25 Phase 2 (block interleaver) | `p25_phase2_interleave_mode` | `off` (no deinterleave; matches synthesized-fixture expectations) | `on` flips the TIA-102.BBAC per-burst MAC block deinterleaver on, applied to the burst dibits before trellis decoding |
| P25 Phase 2 (soft decision) | `p25_phase2_soft_decision` | `off` (hard slicer; byte-for-byte the historical behaviour) | `on` feeds the demodulator's soft symbol differentials into a per-bit soft Viterbi on the traffic-channel MAC trellis, recovering the ~1.5-2 dB of coding gain the hard slicer discards (helps surface the clear-MAC source RID on weak signals; neutral on strong ones — issue #915) |
| P25 Phase 2 (equalizer) | `p25_phase2_equalizer` | `off` (no equalization; the symbol stream is untouched) | `on` (alias `cma`) runs a blind constant-modulus (CMA) adaptive equalizer on the traffic-channel symbol stream after carrier recovery, removing residual inter-symbol interference — RRC mismatch, a fractional Gardner timing error, mild multipath — ahead of the differential decode so an ISI-corrupted burst still satisfies the outer RS. CMA is used because the H-DQPSK constellation spins π/8/symbol with no fixed phase grid. Helps ISI-limited channels; neutral on a purely AWGN-limited one, and transparent (byte-identical) on a clean signal — issue #915 |
| P25 Phase 1 (soft decision) | `p25_phase1_soft_decision` | `off` (hard TSBK trellis; byte-for-byte the historical behaviour) | `on` has the C4FM receiver derive per-bit LLRs from the 4-level soft symbols (sign axis for the MSB, inner/outer magnitude threshold for the LSB) and the control channel's TSBK trellis run a true per-bit soft Viterbi (`framing.DecodeP25TrellisSoft`), recovering the coding gain the hard slicer discards on marginal signals — measured: at the AWGN level where the hard path decodes 81/300 TSBKs, soft decodes 212/300, and at the level where hard decodes zero the soft path still recovers every burst in the end-to-end CC fixture. The TSBK CRC16 corroborates every block either way, so a soft mis-decode cannot manufacture a grant. C4FM sites only (CQPSK/LSM has no real 4-level soft track and decodes hard regardless); feature-parity port of `p25_phase2_soft_decision` |
| P25 Phase 2 (DC block) | `p25_phase2_dc_block` | `off` (the IQ stream is untouched) | `on` inserts a first-order DC-removal high-pass at the top of the traffic-channel receiver — the same zero-IF LO-spur stage the P25 Phase 1 and TETRA voice receivers run. A zero-IF front end's LO self-mixing leaves a static DC spur at 0 Hz that sits directly on an on-channel voice DDC; the constant complex bias shifts every absolute H-DQPSK symbol identically, which does not cancel in the differential decode. The ~1 Hz corner leaves the 6000-baud modulation untouched. Voice-receiver only; the CC path stays DC-untouched (the TETRA/P1 precedent) |
| TETRA (traffic LMS) | `tetra_traffic_lms` | `off` (the traffic soft path runs byte-for-byte as before) | `on` enables the midamble-trained `SnapshotLMS` equalizer on the voice composer's traffic chains (solo tap and per-carrier demux): the extractor trains on each burst's known NTS1/NTS2 midamble from the raw pre-differential symbols, freezes the taps, equalizes the BKN spans, and re-derives the soft TCH/S LLRs from the equalized symbols (`TrafficExtractor.EnableLMSEqualizer` + `SymbolSink → StashSymbols` — the #1001 follow-up lever, pinned by `traffic_lms_test.go`: synthetic multipath 13% → 0% payload bit-error, no-harm on clean). Opt-in per CLAUDE.md: the operator's `GT_TETRA_LMS=1` capture A/B (`TestTETRAMultiSlotReplay`, comparing `traffic_marked_crc_soft`) is the gate for ever defaulting it on |
| NXDN | `nxdn_viterbi_mode` | `ViterbiSpec` | `off` |
| NXDN (soft decision) | `nxdn_soft_decision` | `off` (hard slicer; byte-for-byte the historical behaviour) | `on` derives per-bit LLRs from the receiver's 4-level soft symbols (sign axis for the MSB, inner/outer magnitude threshold for the LSB) and runs the spec CAC decode through a true per-bit soft Viterbi (`framing.ViterbiK5Soft`), recovering the coding gain the hard slicer discards on marginal signals — measured on the shared K=5 primitive, ~8× info-bit error reduction at the AWGN level where the hard path loses one bit in eleven. Effective with the default `nxdn_viterbi_mode` (`spec`); the legacy fixture modes always decode hard. Feature-parity port of `p25_phase2_soft_decision`. A/B on a capture: `GT_NXDN_IQ=… GT_NXDN_SOFT=1 go test ./cmd/gophertrunk -run TestReplayNXDNRealCapture -v`, comparing `cac_crc_ok` |
| NXDN (carrier AFC) | `nxdn_afc` | `off` (no carrier correction; byte-for-byte the historical behaviour) | `on` enables the post-clock coarse carrier-offset correction (the DMR issue #836 `CoarseAFC`, ported): tracks the tuner-ppm DC bias on the recovered symbol stream and subtracts it, recentring the 4-level eye on the slicer thresholds — pinned by `TestReceiverIsCarrierOffsetInvariant` (a 400 Hz offset collapses decode without it). Opt-in unlike DMR/P25 because NXDN's CAC carries no air-interface whitening: a zero-heavy L3 payload convolutionally encodes to long constant-dibit runs, and the plain coarse tracker drifts onto that data mean (the issue #402 mode), collapsing CAC CRC yield on a clean, centred signal — measured on the repo's own synthetic SITE_INFO fixture. Turn on only for a rig with a real tuner frequency error; a capture-validated pairing with a decision-directed stage is the path to any default change |
| EDACS | `edacs_bch_mode` | `BCHOn` | `off` |
| MPT 1327 | `mpt1327_bch_mode` | `BCHOn` | `off` |
| MPT 1327 CWSC tolerance | `mpt1327_cwsc_tolerance` | `2` (Hamming distance against the 16-bit Codeword Synchronisation Code, matches commercial MPT 1327 receivers) | `0` / `exact` / `off` for pre-stripped fixtures; integer in [0, 15] for custom thresholds |
| Motorola Type II | `motorola_bch_mode` | `BCHOn` | `off` |
| D-STAR | `dstar_fec_mode` | `FECOff` (info-bits passthrough) | `on` flips to the JARL DV-mode FEC chain |

The README's [FEC opt-outs section](https://github.com/MattCheramie/GopherTrunk/blob/main/README.md#fec-opt-outs)
documents the full reference table with on-default behaviour
descriptions.

### Implementation note

The in-package `ControlChannel` constructors still zero-value to
`Off` mode so direct callers (primarily unit tests) see the legacy
behaviour without explicit setup. The connector goes through
`Parse*Mode(opts.System.X)` which maps empty strings to the new
on-defaults, then calls `SetXMode(parsed)` on the ControlChannel.
This keeps the operator-facing default `On` while preserving the
in-package fixtures' expectations of an `Off` zero value.

For TETRA specifically: the `TETRAChannelCoding` field was added
to the System struct (yaml: `tetra_channel_coding`) because the
old "zero colour code = off" rule conflicted with BSCH's
spec-defined zero colour code. Non-BSCH systems with channel
coding on need a non-zero `tetra_colour_code`; the connector
warn-logs the misconfiguration instead of silently dropping
frames.

---

## 2. Receiver clock recovery

**Gardner timing recovery on by default.** Both the P25 Phase 2
and TETRA receivers route the matched-filter output through the
Gardner symbol-timing-recovery loop in
[`internal/dsp/sync/gardner.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/dsp/sync/gardner.go).
Operators with sample-aligned synthesized IQ fixtures (some tests,
some replay tools) can opt back to the naive sps-th-sample
decimator per system.

| Receiver | YAML key | Default | Opt-out string |
| --- | --- | --- | --- |
| P25 Phase 2 | `p25_phase2_clock_mode` | `ClockGardner` | `naive` / `off` |
| TETRA | `tetra_clock_mode` | `ClockGardner` | `naive` / `off` |

---

## 2a. P25 Phase 1 demodulator selection

**C4FM on by default; opt into CQPSK/LSM only when a site genuinely
transmits it.** Almost all P25 Phase 1 sites transmit C4FM on the air —
**including most simulcast systems** — and the FM-discriminator +
4-level slicer the receiver ships with handles them. Some systems
instead transmit
[Linear Simulcast Modulation](https://www.dsheirer.com/linear-simulcast-modulation/)
(LSM, TIA-102.BAAA), a CQPSK-shaped linear waveform. LSM pushed through
an FM discriminator produces near-random dibits and the FSW never
matches — the failure mode reported against
[issue #275](https://github.com/MattCheramie/GopherTrunk/issues/275).

**Simulcast does not imply CQPSK.** Simulcast is a transmitter-
coordination technique, not a modulation — most simulcast systems
(Victoria's MMR, its multi-tower Melbourne CBD included) transmit plain
C4FM, and forcing CQPSK there kills the decode. You cannot read the
modulation off the fact a site is simulcast, nor off emission-designator
/ licensing data (`10K1D7W` covers both C4FM and CQPSK). Determine it
empirically: leave it at C4FM, and switch a system — or a single site,
via the per-channel `p25_phase1_demod_mode` override
([issue #935](https://github.com/MattCheramie/GopherTrunk/issues/935)) —
to CQPSK only when a strong, clean signal will not lock in C4FM:

| Receiver | YAML key | Default | Opt-in string |
| --- | --- | --- | --- |
| P25 Phase 1 | `p25_phase1_demod_mode` | `c4fm` (FM discriminator + 4-level slicer) | `cqpsk` / `lsm` / `linear` (complex RRC + Gardner + differential QPSK) |

The CQPSK path internally pins Gardner timing recovery on (the LSM
demod operates on complex IQ at the sample rate; naive sps-th-sample
decimation produces meaningless symbols at any timing offset). The
log line `ccdecoder: p25/phase1 pipeline configured demod=…` on
startup confirms which path is active.

Other receivers (DMR, dPMR, NXDN, EDACS, LTR, MPT 1327, Motorola
Type II, YSF) either don't need timing recovery or use protocol-
specific clock-tracking primitives. The Gardner loop is wired
where it had the largest measurable effect on noisier on-air
captures.

---

## 3. Daemon-level features

Sources are [`config.example.yaml`](https://github.com/MattCheramie/GopherTrunk/blob/main/config.example.yaml) plus
the config struct in [`internal/config/config.go`](https://github.com/MattCheramie/GopherTrunk/blob/main/internal/config/config.go).

| Feature | YAML key | Default | Why |
| --- | --- | --- | --- |
| **Live audio playback to speakers** | `audio.enabled` | `false` | Headless and container deployments stay silent by design. WAV recording is unaffected — recordings land on disk whether playback is on or off. Stays opt-in: audio-on-by-default would crash or warn loudly in distroless / container deployments. |
| **API mutation endpoints** | `api.auth.mode` | `disabled` | An empty `mode` resolves to `disabled` — mutations wide open, no auth — because GopherTrunk is overwhelmingly deployed on closed LANs; the daemon logs a loud startup warning when this default takes effect on a non-loopback bind. Opt into `auto` (bearer token on public binds, loopback bypass) or `required` (token on every mutation). See [`docs/hardening.md` §"API authentication"](hardening.md). Legacy `allow_mutations: true` maps to `auth.mode: disabled` with a deprecation warning. |
| **Manual VFO tune** | `scanner.manual_tune_enabled` / `scanner.manual_tune_disabled` | auto-detect | Auto-enables when ≥ 2 Voice SDRs are present (the daemon constructs the scanner off the spare). `manual_tune_enabled: true` forces the scanner even with only one Voice SDR; `manual_tune_disabled: true` vetoes the auto-detect for operators who want every Voice SDR reserved for trunking. |
| **CMA blind equalizer** | `recordings.equalizer.enabled` | `false` | Simulcast mitigation costs CPU and may distort clean-RF capture. Benefit is site-specific — operators not on a simulcast site pay the CPU without payoff. Stays a global opt-in until a per-call auto-tune heuristic ships. |
| **Voice enhancement ("sound-good" chain)** | `recordings.enhance.enabled` | `false` | A post-vocoder chain for decoded digital voice — rumble high-pass (~250 Hz), warmth high-shelf, telephone-band low-pass (~3.4 kHz like OP25), a louder AGC target (22000 vs the faithful 18000), and an optional soft-knee compressor. Deliberately trades a little faithfulness for the cleaner/louder sound rival decoders produce (OP25 band-limits; Trunk Recorder normalizes loudness on by default; DSD/DSDPlus run an aggressive AGC). Applied at the single per-call decode point, so it shapes **both** recordings and live monitoring. Off by default (faithful output is byte-identical); surfaced near the top of the Config Builder, and `enhance.enabled` hot-reloads via `PATCH /api/v1/settings`. Supersedes `warm_dmr_audio` by extending the warmth shelf to all protocols. |
| **Loudness normalization** | `recordings.normalize.enabled` | `false` | EBU R128 / BS.1770 per-call loudness normalization (default target -16 LUFS, true peak -1.5 dBTP), pure Go — no ffmpeg. Each finished call is measured and a single linear gain (true-peak limited) is applied so calls from different talkgroups/sources play back evenly. `apply_to` selects the target: `recording` (default — rewrite the on-disk WAV; the distributed MP3 inherits it), `distributed` (leave the WAV pristine, normalize only the outbound broadcast/stream copy), or `both`. Opt-in because it changes recorded/streamed levels and not every operator wants that; live monitoring is unaffected. |
| **Tone-out paging-tone detection** | `tone_out.profiles` | empty list | Two-tone sequential (Quick Call II) profiles are per-site / per-agency. No useful zero-config default exists. [`config.example.yaml`](https://github.com/MattCheramie/GopherTrunk/blob/main/config.example.yaml) ships an example profile plus three commented-out alternatives (single-tone, system+talkgroup scoped, tightened tolerance) so operators discover the schema without grepping source. |
| **Scan mode = list** | `scanner.scan_mode` | `"all"` | "all" is the backwards-compatible behaviour. Tag-based talkgroup curation must be done by the operator before "list" is useful. Per-deployment choice. Emergency grants bypass the gate regardless. |
| **CTCSS / DCS squelch** | per-channel `tone:` block | omitted (no gating) | Sub-audible CTCSS tone / DCS code is per-channel signalling that varies by site, repeater, and agency. No useful zero-config default. Omitting the block reverts to carrier-only squelch. |
| **Raw frame sidecar** | `recordings.write_raw` | `true` in `config.example.yaml` | On in the shipped example. Operators who don't want the `.raw` sidecar set `false`. |
| **Skip encrypted calls** | `recordings.skip_encrypted` | `false` | Records everything by default — the call log still notes encryption even when audio can't be decoded. Set `true` to keep undecodable encrypted traffic off disk: a call whose grant flags encryption is never opened, and one whose encryption only surfaces mid-stream (P25 Phase 1 LDU2, or a Phase 2 compressed grant) has its in-progress files closed and deleted. Live follow / playback is unaffected. |
| **Outbound call streaming** | `broadcast.*` | empty (disabled) | Per-feed lists for Broadcastify Calls, RdioScanner, OpenMHz and Icecast/ShoutCast. Each feed needs operator-supplied credentials, so there is no zero-config default; a feed with `enabled: false` is parsed but skipped. A talkgroup opts out of all feeds with `stream: false`. |
| **Baseband IQ recording / replay** | `baseband.record` / `baseband.replay` | empty (disabled) | Wideband IQ recordings are large; recording is per-tuner opt-in. `replay` mounts recorded WAVs as virtual tuners — record at the same rate as `sdr.sample_rate` for real-time-correct playback. |
| **Decoded-message log** | `log.message_log.enabled` | `false` | A rotating human-readable text log of every trunking event. Off by default to avoid unbounded disk use on long-running headless daemons. |
| **Power log** | `log.power_log.enabled` | `false` | A rotating per-channel IQ-power log for the wideband engine, gated on decode activity (idle / off-band channels never appear). By default records only low-power (weak-signal) windows on channels that are actively decoding — the "decoding but the signal is marginal" diagnostic. Set `log.power_log.all_windows: true` for a full power time-series of every decode-active window. |
| **Per-talkgroup record / mute** | CSV `record` / `mute` columns, JSON fields, or `PATCH /api/v1/talkgroups/{id}` | `record: true`, `mute: false` | `record: false` follows and plays a talkgroup live without writing WAV/raw files; `mute: true` suppresses it from the live audio player while still recording and streaming. |
| **Prometheus metrics** | `metrics.enabled` | `true` | On by default; listed for completeness. |
| **Call-log retention sweep** | `retention.call_log_days` | `30` (`0` disables) | Sensible default already on. Set `0` to disable the sweeper. |
| **File retention sweep** | `retention.files_days` | `14` (`0` disables) | Same as call-log. |

---

## 4. Build-time options

These stay opt-in by their nature — none of the three are candidates
to flip on by default.

| Feature | Mechanism | Default | Why |
| --- | --- | --- | --- |
| **DVSI hardware vocoder backend** | `-tags dvsi` build tag ([`docs/architecture.md`](architecture.md), [`docs/vocoders.md`](vocoders.md)) | Not built | Patent-encumbered. Requires DVSI USB-3000 / AMBE-3003 hardware. Status: Vocoder + AMBE-3003 wire protocol + voice.Vocoder interface conformance shipping behind the build tag (`internal/voice/dvsi/`); USB / FTDI transport that talks to the physical chip is a stub returning `ErrNoDevice`. Recorder fallback chain activates cleanly when no chip is connected. `make test-dvsi` exercises the wire protocol + scripted-mock Transport + loopback Transport without hardware. The pure-Go AMBE+2 backend is the default and ships everywhere. Stays opt-in permanently for licensing reasons. |
| **Integration tests** | `-tags integration` build tag | Not run by `go test ./...` | Enables a wired end-to-end daemon test that doesn't need a real SDR. Long-running, intentionally outside the default unit-test wall-time budget. CI runs the tagged suite separately. |
| **Pure-Go (no CGO)** | implicit `CGO_ENABLED=0` build | On | Already the default everywhere — no `librtlsdr` / `libusb` dependency. Listed for completeness because the README emphasises it as a design property. |

---

## 5. Remaining FEC follow-ups

Per-protocol on-air FEC chains land in stages. Most ship today as
opt-out (see §1); a handful of inner FEC layers remain as
documented follow-ups, generally blocked on either spec access or a
real-air capture to validate against:

| Item | Status | Blocker |
| --- | --- | --- |
| **NXDN per-protocol interleaver + puncture inner layer** | `ViterbiSpec` mode wired through the connector; calibration against a captured MMDVMHost / DSDcc transmission lands next | Real-air capture |
| ~~**P25 Phase 2 NSB-driven runtime seed installation**~~ (now shipping) | Trellis decoder, outer RS(24, 16, 9) verifier, PN44 descrambler with per-burst slot-offset blind probe (`ScramblerProbe`), AND NSB-driven runtime seed installation all shipping per TIA-102.BAAA-A §5.9 + TIA-102.BBAC-1 §7.2.5. `ControlChannel.Ingest` parses every Network Status Broadcast - Update MAC PDU (opcode 0xFB) and auto-recomputes the scrambler seed from the `(WACN, SystemID, ColorCode)` triple in its payload via `pn44SeedFromNSB`. Per-system static seed config still provides the initial value before NSB lands. No remaining Phase 2 spec follow-ups | None — closed |
| ~~**MPT 1327 CWSC bit-error tolerance**~~ (now shipping) | BCH(64, 48, 2) per-codeword check + 16-bit Codeword Synchronisation Code (`1100010011010111`) detection both shipping per the MPT 1327 standard. The Process adapter now matches CWSC against a Hamming-distance threshold (default 2 bits out of 16, matching commercial MPT 1327 receivers) instead of the previous exact-match. Set `mpt1327_cwsc_tolerance: 0` to opt back into exact-match for pre-stripped synthesized fixtures. False-positive math: the C(16, 0..2) = 137 / 65536 ≈ 0.21% per-window rate, combined with the BCH-validated codeword that must follow (≈ 2^-15), keeps the per-bit-position false-lock rate under 1e-7 | None — closed |
| ~~**YSF FICH interleaver / puncture validation**~~ (spec-level codec now shipping) | K=5 ½-rate trellis encoder + decoder + the full on-air codec (`EncodeFICHOnAir` / `DecodeFICHOnAir` with puncture positions `{0, 1, 102, 103}` and column-major 10×10 interleave, per MMDVMHost / DSDcc / Pi-Star) all ship. Unit tests confirm every single-bit-flip is corrected and the interleave permutation is bijective. Calibration against a captured YSF transmission still pending — schedule swaps two lines per `samples/ysf/README.md` if the on-air capture disagrees with MMDVMHost's table | Real-air capture |
| **TETRA on-air recovery margins** | Full §8.3.1 chain ships and unit tests round-trip clean fixtures. The Viterbi correction-depth instrument ships too: the opt-in `gophertrunk_tetra_viterbi_corrections` histogram (`metrics.detailed_fec: true`) records the channel bits the FEC chain corrected per recovered BSCH / SCH-HD burst, decoder-independent (re-encoded-vs-received Hamming weight). A real marginal-signal capture that locks is now committed (`internal/scanner/ccdecoder/testdata/tetra_cc_sync_loss_2s_144k.cs16`), and the equalizer tests characterise its margins in CI (~12% → ~100% CRC-clean BSCH with the blind CMA equalizer). The old `samples/tetra/TETRA IQ.wav` still lacks a validating `.metadata.json` sidecar and never decodes CRC-clean, so the skip-gated `samples/` real-air harness stays dormant; the histogram's p95/p99 calibration against a full-length live capture is the remaining piece. | Full-length validating capture in `samples/tetra/` |
| ~~**DMR Tier II synthesized IQ fixture**~~ (now shipping) | Pipeline + Process adapter + unit test all ship; `TestDaemonCCDecodesDMRTier2` now passes end-to-end. The diagnostic in `cmd/gophertrunk/dmr_tier2_diagnostic_test.go` localised the divergent statistic to the BPTC(196, 96)-encoded payload's class-3 dibit overrepresentation (21.4% Tier II vs 5.1% Tier III) and 1.27 vs 0.90 mean transition magnitude; the fix lowered `newDMRTier2Pipeline`'s ClockGain from 0.025 to 0.015 to track the harder symbol distribution. Live captures benefit equally — the more conservative gain stays within the loop's noise margin | None — closed |

These don't block protocol-level operation today — every protocol's
production pipeline ships through to `events.KindCCLocked` /
`events.KindGrant` on the bus, and the engine + recorder + API
surfaces light up unchanged. The follow-ups improve on-air decode
robustness for specific noise conditions or fully-spec-correct
encoding.

Captures that close each follow-up belong in the
[`samples/`](https://github.com/MattCheramie/GopherTrunk/tree/main/samples/) directory at the repo root — one
subfolder per protocol. Each subfolder's `README.md` documents the
capture format and the metadata schema GopherTrunk needs to
validate the decode end-to-end.

---

## 6. How to verify what's currently enabled

- **FEC defaults per system.** Open the **Settings** panel in the
  TUI — the **FEC** tab lists every configured system with a
  one-line summary (`channel coding: on (colour=…, sch/f)`,
  `viterbi: spec`, `bch: on`, etc.).
- **Programmatic introspection.** Each protocol's `ControlChannel`
  exposes matching getters (`tetra.ControlChannel.ChannelCoding()` /
  `ExpectedChannel()` / `ColourCode()`, `ltr.ControlChannel.FCSMode()` /
  `ManchesterMode()`, `p25phase2.ControlChannel.TrellisMode()`,
  `nxdn.ControlChannel.ViterbiMode()`, `edacs.ControlChannel.BCHMode()`,
  `mpt1327.ControlChannel.BCHMode()`, `motorola.ControlChannel.BCHMode()`).
- **JSON over HTTP.** The `/api/v1/systems` endpoint DTO carries
  every FEC opt-out field as `omitempty` JSON — a configured-systems
  audit is one `curl` away.
- **Daemon-level state.** Inspect the running `config.yaml` and the
  daemon's startup log lines; the config loader logs the effective
  values for every section as it parses.
- **API auth capability.** `GET /api/v1/mutations` is always open
  and reports `auth_mode` + `can_mutate` for the current request, so
  scripts and TUIs can light up write-side keybindings without
  probing real endpoints.
