---
layout: page
title: Decoders that need live RF captures
description: Which GopherTrunk decoders are blocked on a real over-the-air capture before they can be called fully validated — and exactly what to record for each (format, sample rate, modulation, duration, and pass bar).
nav_group: Technical docs
---

# GopherTrunk — Decoders that still need live RF captures

A walk through every decoder in the tree, flagging which ones are
**blocked on a real over-the-air capture** before they can be called
fully validated, and exactly what to record. Everything below decodes
*synthetic* IQ today — the gap is confirming the on-air constants
(bit order, deviation, interleave schedule, FEC margins) against a real
signal.

## ⚠️ Read this first: IQ vs. audio

GopherTrunk's pipelines start at **complex IQ baseband**. For anything
that carries information in **phase or 4-level amplitude**, an MP3/WAV of
the *demodulated audio* is useless — the constellation is already gone.

- **Must be IQ** (`.cfile` / `.bin` / `.iq`, or a *stereo* IQ `.wav` with
  I=left / Q=right): TETRA, NXDN, YSF, DMR, M17, P25.
- **Audio is OK** only where the data rides audio-band FFSK tones:
  MPT 1327 (and as a fallback for POCSAG/FLEX/DSC, though IQ is preferred).
- `.cfile` = GNU Radio interleaved **float32** IQ (8 bytes/sample).
  `.bin` = complex **int16** (4 bytes/sample). Neither embeds the sample
  rate or center frequency — **always drop a `metadata.json` sidecar**
  stating `sample_rate_hz` + `center_freq_hz`.

Drop captures in `samples/<protocol>/` with a `metadata.json`. Skip-gated
integration tests pick them up automatically.

---

## 🔴 Tier 1 — Blocking: decoder unproven on air, harness ready

These have a finished decode chain and a waiting test harness, but **zero
real-air confirmation**. A single labeled capture closes each one.

### NXDN (NXDN-TS-1-A control channel)
- **What to capture:** Outbound RCCH control-channel carrier.
- **Format:** complex float32 `.cfile` (or int16 `.bin`).
- **Sample rate:** ≥ 48 kHz (48 kHz nominal).
- **Modulation:** 4-level FSK @ 4800 sym/s, 6.25/12.5 kHz channel.
- **Duration / size:** ≥ 5 s (~2 MB at 48 kHz f32); 10 s is better.
- **Specifics:** spec peak deviation is 1800 Hz — if the dibit
  distribution comes out lopsided, sweep `nxdn_deviation_hz`
  (non-spec rigs land ~2200–2700 Hz). Note: an *NXDN-BFSK* (2-level,
  4800 bps) capture will **not** decode — the receiver is 4-FSK only.
- **Cross-check:** MMDVMHost log + DSDcc 1.9.5.
- **Pass bar:** ≥ 80% CAC bursts CRC-OK, SystemID/SiteID/RAN byte-match,
  CC lock < 3 s.

### TETRA (TMO downlink control)
- **What to capture:** BS downlink carrier (SCH/HD, SCH/F, BSCH frames).
- **Format:** complex float32 `.cfile` (or int16 `.bin`).
- **Sample rate:** ≥ 72 kHz strongly recommended (≥ 4 samples/symbol at
  18 ksym/s for reliable Gardner lock — a 48 kHz capture is too low and
  will not lock).
- **Modulation:** π/4-DQPSK @ 18 ksym/s, 25 kHz channel.
- **Duration / size:** ≥ 30 s (~17 MB at 72 kHz f32) — include an idle
  window for noise-floor profiling.
- **Specifics:** **cleartext frames only** — TEA1–4 encrypted traffic
  can't be used (key recovery is out of scope). Co-channel / adjacent-
  channel interference is *welcome*; that's exactly the Viterbi margin
  the capture is meant to profile.
- **Cross-check:** telive 1.5 / osmo-tetra.
- **Pass bar:** CC lock < 5 s, ≥ 90% frame recovery, Viterbi correction
  depth p95 ≤ 8 / p99 ≤ 12 bit-errors per 116-bit block.
- **Verification progress (live 468.5 MHz / 1 Msps captures):** two
  blockers found and one fixed. (1) The captured front-end was
  **spectrum-inverted** — set `iq_invert: true` (or replay with
  `-conjugate`); with that the normal-burst demod is clean (NTS1 at
  Hamming distance 0). (2) A real-air decoder bug: the BSCH colour-code
  recovery only correlated the rotation-0 orientation of the
  synchronisation training sequence, but π/4-DQPSK's residual-CFO term
  rotates the whole dibit stream — now fixed to try all four rotations
  (`internal/radio/tetra/process.go`, guarded by
  `TestRecoverColourCodeUnderRotation` +
  `TestProcessLearnsColourCodeFromSBBurstUnderRotation`). Still open:
  the supplied clips are only ≈3.3 s each (too few frame-18 SB slots)
  and their SB region does not demodulate cleanly, so no `cc.locked`
  fires yet. Closing this needs a **≥30 s** cleartext capture spanning
  several clean synchronisation bursts. See `samples/tetra/README.md`.

### YSF (Yaesu System Fusion, DN mode)
- **What to capture:** A YSF carrier (Pi-Star/reflector or a keyed HT).
- **Format:** complex float32 `.cfile` / int16 `.bin` (or stereo IQ WAV).
- **Sample rate:** ≥ 48 kHz nominal.
- **Modulation:** C4FSK @ 4800 sym/s, ±2700 Hz peak deviation, 12.5 kHz.
- **Duration / size:** ≥ 10 s (~4 MB at 48 kHz f32) — enough FICH cycles.
- **Specifics:** confirms the FICH interleave/puncture schedule
  (`{0,1,102,103}` + column-major 10×10, per MMDVMHost). If CRC fails
  there's a documented two-line swap to the DSDcc alternate schedule —
  if so, publish the K=5 generator pair you landed on.
- **Cross-check:** DSDcc 1.9.5 in YSF mode / Pi-Star FICH log.
- **Pass bar:** 100% FICH CRC at clean SNR, ≤ 4 trellis bit-errors per
  100-bit block at ≥ 12 dB.

### DMR 2-slot interleaved voice (opt-in `dmr_interleaved_voice`)
- **What to capture:** A live DMR carrier with **both timeslots active**
  (Tier III trunked voice, or a busy Tier II repeater).
- **Format:** GNU Radio interleaved float32 `.cfile`.
- **Sample rate:** state it in the env/sidecar (48 kHz works with the
  harness).
- **Modulation:** C4FM @ 4800 sym/s.
- **Duration / size:** long enough for a full call on the followed TG
  (≥ 10–15 s, ~4–6 MB at 48 kHz).
- **Specifics:** this is the gate before the interleaved decoder can
  replace the single-slot decoder as the default. It confirms two
  unknowns: (1) the same-slot dibit cadence on live BS air — whether a
  CACH makes the stride 288 vs. 264 dibits; (2) the ETSI embedded-LC
  de-interleave order, EMB QR(16,7) FEC, and 5-bit CRC polynomial.
  Run via `GOPHERTRUNK_DMR_2SLOT_CFILE` / `_RATE_HZ` / `_TG`.
- **Pass bar:** the followed talkgroup's embedded LC is recovered and its
  (and only its) audio is routed to the sidecar.

### DMR Full-LC FEC on air (issue #527 follow-up) — ✅ CONFIRMED
- **Status:** **Closed by live capture.** A set of 441 MHz / 2 MS/s GNU
  Radio f32 captures of a DMR Tier III trunked system was channelised and
  replayed through the production receiver. The receiver's real-air dibit
  recovery and end-to-end bit ordering through BPTC(196,96) → RS(12,9) →
  Full LC parse — the exact stack the Voice LC Header uses and the only
  thing the field report left unproven — decode **cleanly on air**.
- **What settled it:** the Voice LC Header and the **Terminator-with-LC**
  carry the *same* Full LC PDU through the *same* BPTC + RS(12,9) FEC (only
  the RS seed differs, 0x96 vs 0x99, both pinned by
  `TestRS129MatchesIndependentReferenceEncoder`). On-air Terminator-with-LC
  bursts recover a stable RS-validated FLC (TG 24 / source 4209000), and
  the control channel's CSBKs pass BPTC + CRC ~97–98%, with the Tier III
  ControlChannel locking on the Aloha beacon.
- **Regression guards:** `TestReplayDMRVoiceLCFECDecodesRealAir` +
  `TestReplayDMRTier3ControlDecodesRealAir` in
  `cmd/gophertrunk/dmr_realcapture_test.go` (fixtures
  `cmd/gophertrunk/testdata/dmr-{voice-term,t3-cc}.cfile`).
- **Root cause of the original field failure:** the unit-energy RRC matched
  filter overdrove the fixed 4-level slicer, collapsing inner symbols onto
  the outer rails so every BPTC payload came back uncorrectable while the
  more forgiving sync + slot-type FEC still passed — exactly the reported
  symptom. The symbol-AGC calibration in `internal/radio/dmr/receiver`
  (issue #275) fixed it; these captures confirm the fix on real bits.

### DMR Tier II Voice LC Header (conventional repeater) — 🟡 optional
- **What to capture:** A live *conventional* DMR repeater carrying voice —
  key-up to un-key, so the capture spans a **Voice LC Header** and the
  **Terminator-with-LC** that bracket a transmission.
- **Format:** complex int16 `.bin` or GNU Radio float32 `.cfile` under
  `samples/dmr-tier2/`. **Not** demodulated audio.
- **Sample rate:** ≥ 48 kHz. C4FM @ 4800 sym/s, ~1944 Hz deviation, 12.5 kHz.
- **Duration / size:** ≥ 5 s (~1 MB int16) — one call is enough.
- **No longer blocking:** the underlying real-air BPTC/RS Full-LC concern is
  resolved above. A conventional capture now only adds *direct* coverage of
  the Tier II call-setup path (Voice LC Header → grant → Terminator) rather
  than the trunked Terminator-with-LC that shares its FEC.
- **Cross-check:** MMDVMHost log / DSD-FME / radio display.
- **Pass bar:** the captured Voice LC Header decodes to the expected
  talkgroup + source ID + color code, with no decode.error stream on a
  clean-SNR call.

---

## 🟡 Tier 2 — Calibration: works on synthetic, needs real to lock constants

Decoders that run end-to-end against a synthetic encoder; a real capture
calibrates timing/deviation/level constants or confirms bit order.

### IMBE vocoder (P25 Phase 1 voice level calibration)
- **What to capture:** A 5 s+ P25 Phase 1 voice call, recorded with
  `recordings.write_raw: true`.
- **Deliverables (two files, same call):**
  - `p25-p1-voice.raw` — raw IMBE frames, **11 bytes/frame**, MSB-first
    packed, no header (the daemon's `.raw` sidecar).
  - `p25-p1-voice-dsdfme.wav` — the **same** `.raw` run through DSD-FME /
    OP25, 8 kHz / 16-bit / mono PCM.
- **Size:** tiny (raw frames + a short WAV).
- **Purpose:** lets `calibrate.Compare` quantify the AGC/level offset.
- **Pass bar:** |RMS ratio| < 3 dB, peak cross-correlation > 0.85.

### AMBE+2 vocoder (DMR / NXDN / dPMR / D-STAR voice level calibration)
- **What to capture:** A voice call on any AMBE+2 protocol, `write_raw: true`.
- **Deliverables (two files, same call):**
  - `dmr-voice.raw` — raw AMBE+2 frames, **7 bytes/frame**, MSB-first.
  - `dmr-voice-dsdfme.wav` — same `.raw` via DSD-FME / OP25, 8 kHz/16-bit/mono.
- **Specifics:** Knox / call-alert tones (b₁ ∈ [144,163]) are vendor-
  specific and decode as silence unless registered — fine, just note it.
- **Pass bar:** |RMS ratio| < 3 dB, peak cross-correlation > 0.85.

### POCSAG paging
- **What to capture:** A POCSAG paging channel.
- **Format:** complex float32 `.cfile` under `samples/pocsag/`.
- **Sample rate:** ≥ 48 kHz.
- **Modulation:** 2-FSK / FFSK on NBFM; 512 / 1200 / 2400 bps.
- **Duration:** ≥ 30 s to catch several batches (~12 MB at 48 kHz f32).
- **Purpose:** tune the integrator/slicer timing against real bits (the
  running-mean slicer likely becomes a proper Mueller-Müller + matched
  filter once there's signal to calibrate against).

### FLEX paging
- **What to capture:** A FLEX paging channel.
- **Format:** complex float32 `.cfile` under `samples/flex/`.
- **Sample rate:** ≥ 48 kHz.
- **Modulation:** 1600 bps / 2-level mode (sync `0xA6C6AAAA`, mode `0x870C`).
- **Duration:** ≥ 30 s.
- **Purpose:** confirm sync framing, BCH(31,21) bit order, and capcode
  mapping on air (3200/6400 bps and 4-level modes are out of scope).

### DSC (marine VHF Ch-70 distress)
- **What to capture:** A real ITU-R M.493 DSC burst on channel 70.
- **Format:** IQ `.cfile` (FFSK rides the audio band, but IQ preferred).
- **Modulation:** 1200 Bd FFSK, 1300/2100 Hz tones on NBFM.
- **Purpose:** confirm on-wire bit order, tone sense, and DX/RX time-
  diversity offset against a captured signal.

### APRS / AX.25 packet
- **What to capture:** A real APRS channel (144.390 in NA).
- **Format:** `.wav` or `.cfile` under `samples/aprs/`.
- **Modulation:** Bell-202 AFSK, 1200 bps.
- **Duration:** long enough for several beacons (~30–60 s).
- **Purpose:** replay through `internal/sdr/baseband` to assert the full
  AFSK → HDLC → AX.25 → APRS chain (only the synthetic test exists today).

### AIS (marine vessel positions, 161.975 / 162.025 MHz)
- **What to capture:** Marine VHF 87B/88B.
- **Format:** IQ recording under `samples/ais/`.
- **Modulation:** 9600 Bd GMSK (BT ≈ 0.4).
- **Purpose:** exercise the GMSK frontend on air (only the bit-stream
  synthetic is tested today). Single-slot messages only for now.

### M17 (link-setup metadata)
- **What to capture:** An M17 transmission.
- **Format:** IQ `.cfile` / stereo IQ WAV.
- **Modulation:** 4FSK / C4FM @ 4800 sym/s.
- **Purpose:** confirm the Golay matrix, C4FM slicer deviation, and
  symbol-timing constants (validated against a synthetic encoder so far).
  Codec2 voice is a separate future milestone, not a capture gap.

---

## 🟢 Tier 3 — Optional: pipeline closed, captures only add robustness

Already validated end-to-end (synthetic fixture passes through the
production pipeline). Real captures are *welcome* for burst-error /
false-positive coverage but **not blocking**.

### MPT 1327 (control channel)
- 48 kHz IQ `.cfile` **or** 8 kHz demod-audio `.wav` (FFSK is audio-band,
  so audio works here). FFSK 1200 baud, 1200/1800 Hz tones, NBFM.
  ≥ 60 s (~23 MB IQ / ~1 MB audio) to characterise the noisy-CWSC
  distribution. Closes empirical false-positive rate + per-vendor
  (Tait/Motorola/Simoco) sync bit-error patterns. *Already decodes
  real audio end-to-end today.*

---

## Not capture-blocked (listed so the question is fully answered)

- **ADS-B** — validated against the canonical dump1090 / mode-s.org
  reference vectors; no live capture outstanding.
- **MDC1200** — over-the-air FEC redundancy isn't exploited yet, but
  that's a decode-improvement follow-up, not a capture gap.
- **P25 Phase 1 / Phase 2** — full TIA-102 chains ship and decode to
  audio; Phase 2 inner FEC (trellis / RS / PN44) is closed. The Phase 1
  C4FM control-channel demod is now confirmed on real air: a live UHF
  capture (449.875 MHz, NAC 0x2C1) was channelised to 48 kHz and run
  through the `samples/p25/` demod-quality harness
  (`TestReplayP25RealCaptureMetrics`), which grades it at EVM ≈ 12.7%,
  SNR ≈ 14.5 dB, NID 31/0, TSBK 36/0. The capture binary is **not**
  committed (per `samples/.gitignore`); the committed record is the
  `samples/p25/p25-450875-cc.metadata.json` sidecar, and the capture is
  reproducible locally via `TestGenerateP25Fixture`. (The Tier 2 IMBE
  *voice*-level calibration above is a separate gap — it needs raw IMBE
  frames + a DSD-FME reference WAV, which a control-channel IQ capture
  does not provide.)
- **EDACS, LTR, Motorola Type II, dPMR control** — control chains ship;
  FEC is on by default with no outstanding capture.

### Separate from captures: digital-voice composer chains
YSF and EDACS ProVoice voice are followed and logged but **not yet
turned into PCM** — that's an *implementation* gap (vocoder/composer
wiring), not something a capture unblocks. (TETRA, NXDN, dPMR, and
D-STAR voice now decode to PCM — TETRA verified bit-exact against the
ETSI EN 300 395-2 reference; NXDN, dPMR, and D-STAR wired end-to-end
but unverified on air: each needs a real voice capture to confirm its
AMBE interleave table — a documented placeholder in
`internal/radio/{nxdn,dpmr,dstar}/voice_ambe.go` — plus the frame
geometry and codebook choice, exactly the #764/#771 gate.)

---

## TL;DR priority order for anyone with an antenna

1. **NXDN** outbound control (≥ 5 s IQ, 48 kHz) — harness fully ready.
2. **TETRA** TMO downlink (≥ 30 s IQ, **≥ 72 kHz**, cleartext).
3. **YSF** DN mode (≥ 10 s IQ, 48 kHz).
4. **DMR 2-slot** both-timeslots-active call (≥ 10 s `.cfile`).
5. **IMBE + AMBE+2** voice `.raw` + DSD-FME WAV pairs (level calibration).
6. **dPMR / D-STAR voice** (≥ 10 s IQ, 48 kHz, clear voice) — confirms
   the placeholder AMBE interleave tables + frame geometry behind the
   new experimental voice chains.
7. POCSAG / FLEX / DSC / APRS / AIS / M17 real-fixture confirmation.

Every capture needs a `metadata.json` sidecar with `sample_rate_hz`,
`center_freq_hz`, and the expected decode (SystemID / talkgroup /
callsign / message text) so the harness can grade it, not just smoke-test it.
