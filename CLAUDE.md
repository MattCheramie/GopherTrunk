# GopherTrunk — guidance for Claude

GopherTrunk is a Go SDR trunking scanner/decoder (P25, DMR, NXDN, TETRA, …).
This file is standing guidance for AI-assisted work in this repo. Keep it short.

## Build & test

- `make vet test` — vet + unit tests; must be green before any commit.
- `make integration` — daemon/replay integration tests; run when the daemon,
  DSP, or replay path changed.
- Single package while iterating, e.g. `go test ./internal/scanner/ccdecoder/...`.

## Change scope (mirrors CONTRIBUTING.md)

- **Bug fix**: one narrow commit plus a regression test that **fails without the
  fix and passes with it**. If you can't write a test that fails first, you have
  not yet reproduced the bug — keep digging or ask for a reproduction, don't
  guess at a fix.
- **Feature / refactor**: design first; keep refactors out of behaviour-change PRs.

## Issue-closing policy

Closing an issue is a claim that the reported problem is gone. Do not make that
claim until it is verified. This policy exists because issue #764 was closed
twice on an unverified fix while the symptom was still live (see #771); a
PreToolUse hook (`.claude/hooks/guard-issue-close.py`) now asks for human
confirmation before any close-as-completed.

- **Never close an issue as completed until the fix is verified**: a failing-first
  regression test now passes **and** the reporter has confirmed it, or you have
  reproduced the original symptom and shown this change resolves it.
- **When you can't verify, leave it open.** Post a concise status comment saying
  what you found and what's blocking (e.g. needs the reporter's capture files),
  rather than closing.
- **Address the latest follow-up, not the original report.** Never re-post the
  initial fix description as a close justification — respond to the most recent
  comment specifically.
- **In PRs, prefer `Refs #N` over `Closes #N`** until the fix is verified, so a
  merge doesn't auto-close an unverified issue.
- Closing as `not_planned` or `duplicate` is fine and is not gated.

## DSP / replay notes (so the next investigation starts ahead)

- `gophertrunk replay -tune-hz` uses the single-channel
  `ccdecoder.Downconverter` (`internal/scanner/ccdecoder/ddc.go`), **not** the
  multi-tap wideband `DDCBank` (`internal/dsp/tuner/ddc.go`). They are separate
  paths — a fix to one does not touch the other. (This is what made the #764
  "fix" miss the #771 replay symptom.)
- Both down-converters normalise to the per-protocol channel rate (48 kHz for
  the 4800-baud C4FM family, 144 kHz for TETRA) and the receiver/AGC are sized
  from that output rate, so the decode path is **rate-invariant** to the capture
  rate. A symptom that only appears at a higher capture rate but reproduces in
  offline replay points at the *captured data* (front-end overload / intermod /
  gain staging), not the steady-state DSP — get the raw `.cfile` to reproduce.
  See `internal/scanner/ccdecoder/ddc_highrate_test.go`.
- #764 is now verified against the reporter's own captures and confirms the rule
  above. Mt Anakie (−812.5 kHz) replays at demod SNR ≈19.7 dB / EVM 7.4% from the
  2.5 MS/s capture (locks) but ≈9.5 dB / EVM 22.5% from the 10 MS/s capture (no
  lock). Decimating the 10 MS/s file 4:1 with an *independent* resampler and
  replaying it through the proven 2.5 MS/s path reproduces the SAME ≈9.5 dB — so
  the ~10 dB deficit is baked into the captured samples, not GT's DDC. Neither
  capture clips (both peak ≈−48 dBFS, so it is not overload/IMD), and the wideband
  FFT carrier SNR is actually *higher* at 10 MS/s — carrier-clean but
  modulation-degraded is the signature of front-end phase noise / reciprocal mixing
  at the Airspy's native 10 MS/s clock. `TestDownconverterSNRInvariantAcrossRate`
  in the file above pins this: a noisy channel reaches the receiver at the same
  in-channel SNR whether decoded natively at 10 MS/s or decimated to 2.5 MS/s.
- **TETRA voice** decodes end-to-end: traffic burst → TCH/S channel decode
  (`internal/radio/tetra/tch.go`) → clean-room ACELP vocoder
  (`internal/voice/acelp`) → PCM. Two independent conformance passes against the
  **ETSI EN 300 395-2 reference C codec** underpin it, and both are reproducible
  via skip-guarded harnesses: `internal/voice/acelp/etsi_reference_test.go`
  (feed both the same 137-bit bitstream → bit-identical PCM) and
  `cmd/gophertrunk/tetra_multislot_replay_test.go` (replay a real cs16 IQ capture
  → per-slot audio, correlated against the control channel's grant timeslots).
  Build the ETSI tools with `Word32` as a 32-bit `int` — on LP64 the default
  `typedef long` is 64-bit and every saturating op returns garbage. Lessons that
  cost time: the class-2 CRC is a fixed parity-check matrix (the reference's
  `TAB_CRC` tables), **not** a `G(X)` LFSR — the wrong CRC silently dropped every
  on-air burst while synthetic round-trips passed (self-consistent bug); and the
  SB anchors the slot grid one NDB-slot before its frame's TN1 traffic
  (`ndbSBSlotShift` in `traffic.go`). TCH/S now decodes **soft-decision** like the
  control SCH path (the receiver's `SoftSink` differentials → `softType5FromDiffs`
  → soft depuncture/Viterbi in `framing.DecodeRCPCTetraMotherSoft` →
  `tetra.DecodeTCHSSoft`); it was hard-decision until then, which failed ~70% of a
  marginal same-carrier call's bursts and produced short/garbled recordings. The
  traffic extractor carries the per-burst soft LLRs parallel to its dibit buffer
  (`TrafficExtractor.StashSoft`), and the composer falls back to the hard
  `TCHSpeechFrames` when no soft info is present. When "voice doesn't decode" but the vocoder
  unit tests pass, suspect the channel coding (CRC / interleave / reorder), not
  the vocoder — validate the whole chain against the reference, not just parts.
- **TETRA demod equalizer** (`internal/dsp/equalizer.SnapshotCMA`, wired via the
  receiver's `EnableEqualizer`, enabled in the voice composer). On the reporter's
  concurrent-load captures the residual garble after soft-decision was **linear
  channel / ISI** (multipath / band-edge group delay) smearing the π/4-DQPSK
  constellation — *not* the signal-limited front-end degradation of #764. A blind
  CMA equalizer between symbol-timing recovery and the differential decoder
  inverts it and roughly **doubles** CRC-valid TCH/S yield across the six captures
  (soft-decision 410→778, ~1.9×; e.g. one call 4→207, another 42→134) with no
  loss on already-clean captures. `SnapshotCMA` is the differential-decoder-safe
  variant of the package's plain `CMA` (which applies its live, continuously-
  adapting taps — fine ahead of a coherent slicer, fatal ahead of a differential
  decoder). Lessons that cost time, all pinned by `snapshot_cma_test.go` +
  `receiver_equalizer_test.go` + the skip-guarded capture sweep:
  - **CRC yield is the only trustworthy metric; EVM is a trap.** Blind CMA
    minimises *modulus*, not correctness, and has spurious constant-modulus
    minima — a numerically-unstable variant once showed differential EVM
    collapsing 34%→8% while CRC stayed **0**. Never conclude an equalizer helps
    from EVM; decode to CRC.
  - **Never feed a continuously-adapting equalizer to a differential decoder.**
    CMA's cost is rotation-invariant, so its output phase wanders as the taps
    adapt, and a *time-varying* phase does not cancel in `s·conj(last)` — every
    dibit is corrupted (streaming-adaptive → CRC 0, frozen taps → baseline). The
    design is **adapt a tracking filter continuously but apply a FROZEN snapshot**
    (refreshed every ≫-a-burst symbols) so each 255-symbol burst sees a constant
    filter; the one straddling symbol per snapshot is absorbed by the FEC.
  - **Normalise by a constant (cumulative-mean), not a local EMA.** The CMA update
    scales with |x|³; an EMA that tracks the TDMA downlink's slot-to-slot power
    swings gives a moving modulus target and CMA converges to garbage (CRC 0 even
    though a global-RMS normalise gives the full win). A divergence guard re-seeds
    the tracking filter to pass-through if a normalisation transient / deep fade
    blows the taps up, so one bad patch cannot poison later snapshots.
- **TETRA control-channel sync loss = RF degradation the CC equalizer now
  recovers, not compute starvation.** A reporter's 1-hour session showed ~210
  `control_channel_transitions` and 11 hard CC sync losses (auto-captured by
  `on_cc_sync_loss`). Each loss is preceded by `bsch_fail` climbing / `sb_bursts`
  collapsing, then repeated `tetra: dsp resync (signal-time decode drought)`, then
  the 5 s stale watchdog → `MarkLost` → re-hunt (and a post-relock #815 carrier
  warning). **Do not chase a compute fix:** all 11 losses occurred with ~0
  concurrent voice follows and the `decode_overruns` (704) were one bursty event —
  zero correlation with call/CPU load, so the signal-time resync design (immune to
  starvation) is correct and untouched. The captures are weak (peak −44 dBFS, no
  clipping, ~−3 kHz offset) and split cleanly by in-channel SNR: a healthy
  `concurrent` capture measures ~**18 dB** and replays at BSCH 147/0 (100%); a
  `cc_sync_loss` capture measures ~**10 dB**, LOCKS, but decodes only ~**22%** of
  its BSCH with a destructive-resync storm — the marginal regime that drops lock.
  The constellation is ISI-smeared, and the primary single-channel `newTETRAPipeline`
  was the one TETRA CC path **not** running `SnapshotCMA` (the voice composer and
  `widebandt2/tetra.go` already did; the latter's comment even claimed it mirrored
  the ccdecoder settings). Enabling it there lifts the marginal fixture from ~12% to
  ~100% CRC-clean BSCH — pinned by `internal/scanner/ccdecoder/pipelines_tetra_equalizer_test.go`
  against `testdata/tetra_cc_sync_loss_2s_144k.cs16`. `EnableDCBlock` stays **off**
  the CC path (voice-only per `receiver.go`). Gotcha: the equalizer's blind CMA is
  well-defined only against a noise floor, so the synthetic `TestDaemonCCDecodesTETRA`
  now adds 40 dB AWGN — a literally noise-free constant-modulus input is a degenerate
  case (same reason `receiver_equalizer_test.go`'s clean-channel test adds 30 dB).
  The residual −44 dBFS weak front end is an RF/gain/antenna condition (the #764
  lesson) the equalizer mitigates but does not replace — raise the signal level too.
- **TETRA training-sequence equalizer (`equalizer.SnapshotLMS`) is now wired into the
  per-burst traffic path** (#1001 follow-up). The blind `SnapshotCMA` lives in the
  *receiver* (continuous, framing-free stream); the trained `SnapshotLMS` lives in
  the *extractor* (`TrafficExtractor.EnableLMSEqualizer`), because only the extractor
  knows where each burst's midamble sits. The linear channel is a clean convolution
  only in the **raw symbol** domain, not the differential domain (`s·conj(prev)` is a
  nonlinear product) — so the extractor carries the raw symbols down parallel to its
  dibit/diff buffers via the receiver's new `SymbolSink → StashSymbols` (the symbol
  analog of `SoftSink → StashSoft`). Per burst, softFrame trains on the known NTS1/NTS2
  midamble from the raw symbols, freezes the taps, equalizes a BKN1..BKN2 span (with a
  taps-long FIR warm-up so the first BKN1 differential is past the transient), and
  **re-derives the soft LLRs from the equalized symbols** — hard frame untouched. It is
  soft-path only and opt-in: no `StashSymbols`/`EnableLMSEqualizer` ⇒ byte-identical to
  before. The reference is built by differentially encoding the ideal midamble from a
  unit anchor (arbitrary start phase; the constant rotation cancels in the differential
  decode, the same property that makes a frozen snapshot safe). Pinned by
  `traffic_lms_test.go` (synthetic multipath through the real extractor: raw 13% → 0%
  payload bit-error, no-harm on clean). Production composer still runs CMA only; flip
  LMS on there once the capture A/B validates it. A/B on captures with `GT_TETRA_LMS=1`
  in `TestTETRAMultiSlotReplay` (compare `traffic_marked_crc_soft`).
- **The same training-sequence LMS lever is wired into the DMO (Direct Mode) path**
  (#1003 follow-up). `ExtractDMBurstsEqualized` (`dmo.go` → `dmo_equalizer.go`) re-derives
  each **rotation-0** DNB's `SoftBKN1/SoftBKN2` from equalized raw symbols, so the
  existing `DMBurstTCHSpeechSoft` decodes it unchanged. Two DMO-specific points vs the
  TMO wiring: the DNB geometry differs (BKN1 `[L-108,L)`, BKN2 `[L+11,L+119)`,
  `dmDNB*Start`), and it **only equalizes rotation-0 bursts** — a frozen constant-tap
  filter cannot invert the per-symbol phase ramp of a non-zero residual rotation, and
  at rotation 0 the soft decode's `(4-Rotation)&3` de-rotation is a no-op so the
  rotation-0-trained differentials slot straight in. Opt-in via `ExtractDMBurstsEqualized`
  (needs the `SymbolSink`); `ExtractDMBursts`/`ExtractDMBurstsSoft` are byte-identical.
  Pinned by `dmo_equalizer_test.go` (raw 12% → 0% DNB payload bit-error, no-harm on
  clean, byte-identical without symbols). A/B on captures with `GT_TETRA_DMO_LMS=1` in
  `TestTETRADMOReplay`. Note the whole DMO path is still unverified on air (#1003), so
  this is a lever staged for that validation, not a confirmed win.
- **DMO on-air (#1003): the "encrypted" conclusion was WRONG — it was the colour-0
  descramble skip, now fixed.** The reporter confirmed their DMO radios are **TEA0 (clear,
  no encryption)**, colour code 0 (CPS codeplug: `DMO_438.900/.800`, Security Class 1,
  `NO_KG`). That is the known-clear evidence the earlier note said was required, and it
  overturns the "air-interface encrypted" reading. What still holds from the first capture
  (`tetra_dmo_test2_20sec_cs16_144k`, 438.9 MHz cs16 at 144 kHz), pinned by
  `TestTETRADMOReplay`:
  - **Sync + signalling fully decode.** With the receiver blind-CMA equalizer (harness
    default — lifts CRC-valid SCH/S from **6 → 64** by inverting ISI, like the live TMO CC
    path), `DecodeDMSCHS` gives 64/68 SCH/S and `DecodeDMSCHH` 62/68. The **DMO DM-SYNC PDU
    reuses the TMO SYNC-PDU field layout**, so `ParseSyncPDU` decodes it (colour 0, TN, FN,
    MN, MCC/MNC=0) with a monotonically advancing frame counter — a genuine lock.
  - **DNB geometry `-108/+11` (`dmDNB*Start`) is CONFIRMED correct** — the sharp TCH/S-CRC
    optimum; osmo-tetra-dmo's TMO-copied `-115/+19` is measurably worse.
  - **The real cause of "TCH/S doesn't decode": `DMBurstTCHSpeech`/`DMBurstTCHSpeechSoft`
    skipped descrambling at `colour==0`.** TETRA scrambling is non-identity at colour 0 —
    `NewScramblerTetra(0)` seeds the LFSR to `0xC0000000` (§8.2.5.2 eq. 8.42), which is
    exactly why `DecodeBSCH`/`DecodeDMSCHS` *always* descramble and the DSB signalling
    decodes. The voice path inherited TMO `traffic.go`'s `if colour != 0` shortcut (safe
    there only because a TMO extended colour code is never 0) and so left a clear colour-0
    DNB scrambled going into the Viterbi/CRC — producing the uniform ~1/256 chance-floor
    metric that was misread as encryption. The `#1003` fix removes the guard on both DMO
    TCH/S paths (`dmo_decode.go`): descramble UNCONDITIONALLY with the DM colour code.
  - **Why the earlier "with/without a colour-0 descramble" sweep didn't catch it:** the
    synthetic round-trips scrambled *and* descrambled consistently at colour 0, so they
    passed either way and hid the asymmetry (the #764/#771 self-consistent-synthetic trap).
    `dmo_decode_test.go` now scrambles the encode side UNCONDITIONALLY (real-air behaviour),
    so the colour-0 iterations are the failing-first regression — they fail against the old
    skip and pass with the fix (verified: old code → `DMBurstTCHSpeech` returns 0 frames at
    colour 0; fixed → 2 CRC-valid frames).
  - **Still to verify ON AIR (operator loop, not yet closed):** a green synthetic ≠ on-air
    correct (#764/#771). The operator must replay their clear 438.9 MHz capture through
    `TestTETRADMOReplay` (`GT_TETRA_DMO_IQ`) and confirm `tch_crc` rises off the chance
    floor. `GT_TETRA_DMO_CLEAR=1` flips the VERDICT line: on a capture asserted clear, a
    persistent chance floor is now a *decode* defect to keep chasing (next suspects: DNB
    geometry, the DM colour code via `GT_TETRA_DMO_COLOUR`), NOT encryption. Do not mark
    #1003 verified until that A/B lands.
  - **The A/B landed on a 2nd on-air capture and CLEAR VOICE DECODES — the missing piece is
    the DM colour code, which is 3 here, NOT the 0 the signalling/CPS advertise.** The
    operator's `10aug_dmo_test_bw144_cs16.raw` (438.9 MHz DMO, cs16, replay with
    `GT_TETRA_DMO_RATE=144000`) locks signalling cleanly with the receiver CMA equalizer:
    `dsb_schs_crc=44/45`, distinct FN advancing. At the default colour 0 TCH/S sits at the
    chance floor (`tch_crc=1/269`), but a colour sweep is unambiguous: **`GT_TETRA_DMO_COLOUR=3`
    → `tch_crc=35/269`, `speech_frames=70`, 2.1 s PCM, voice-active seconds [1-8]** (all other
    colours 0-15 stay at the floor; LMS `GT_TETRA_DMO_LMS=1` doesn't help, 35→32). So the
    signal is NOT weak/encrypted (signalling decodes at the same RF) — it is a real clear-voice
    decode that only needs the correct DM colour code for the TCH/S descramble
    (`DescrambleTetra(type5, colour)` in `dmo_decode.go`).
  - **Colour recovery LANDED (C1, capture-verified).** The DM colour is NOT recoverable from
    the SCH/S (always colour-0-scrambled — reads 0). It IS in the DSB SCH/H (DM-SYNC SYSINFO,
    EN 300 396-3), but on a single capture colour 3 lights only the field's two LSBs so its
    exact bit offset is ambiguous — hardcoding it is the #764/#771 self-consistent trap
    (empirically confirmed: a colour-field scan found no unique 6-bit window). So instead
    `tetra.RecoverDMColourCode(bursts)` (`dmo_decode.go`) picks the colour (0..63) that
    maximises CRC-valid TCH/S — the correct one wins by a wide margin (measured 35 vs ≤3),
    with a confidence gate rejecting a chance-floor winner. `TestTETRADMOReplay` now
    auto-recovers the colour when `GT_TETRA_DMO_COLOUR` is UNSET: on the 10aug capture it
    recovers **colour 3 → tch_crc=35, 70 speech frames, 2.1 s PCM, NO manual override**.
    Pinned synthetic: `TestRecoverDMColourCode`. Reproduce:
    `GT_TETRA_DMO_IQ=<10aug .raw> GT_TETRA_DMO_RATE=144000 GT_TETRA_DMO_CLEAR=1
    go test ./cmd/gophertrunk -run TestTETRADMOReplay -v`.
  - **Remaining: NO production DMO decode pipeline (C2, design-first, on-air A/B gates #1003).**
    `internal/scanner/ccdecoder/pipelines.go` has one TETRA factory (`newTETRAPipeline`, the TMO
    CC state machine); a DMO capture runs through it and every SCH burst fails because TMO NCDB
    geometry (`-115/+19`) ≠ DMO DNB (`-108/+11`) — while BSCH still "locks" (DMO DSB SCH/S is
    colour-0-scrambled like TMO BSCH), the operator's live symptom (`sch_pdus=0`, constant
    `sync gap` / `dsp resync`). The `ExtractDMBursts*`/`DecodeDMSCHS/H`/`DMBurstTCHSpeech*`
    decoders + `RecoverDMColourCode` now exist and are validated offline (`TestTETRADMOReplay`),
    but are not wired into the daemon. Execution-ready plan:
    1. **Register** `ProtocolTETRADMO` (`tetra-dmo`) in `internal/trunking/site.go` (enum,
       `String()`, `ParseProtocol`, error strings) + a `factories` entry in `pipelines.go`.
    2. **`newTETRADMOPipeline`** mirroring `newTETRAPipeline`'s receiver (`EnableEqualizer` +
       `SoftSink` + AFC + channel filter), but buffering dibits/diffs and, on a cadence,
       `ExtractDMBurstsSoft` (DNB `-108/+11`) → `DecodeDMSCHS` (sync → `KindCCLocked` + SYNC-PDU
       FN-advance liveness) → `RecoverDMColourCode` (once, then lock) → `DMBurstTCHSpeechSoft`.
       Needs DMO sync-gap/resync timing (TMO's `tetraResyncTimeout`/`tetraLockStaleTimeout` are
       CC-cadence; DMO is bursty PTT — retune).
    3. **Composer voice chain**: DMO voice rides the SAME carrier (no grant→tap split), so add a
       `proto == "tetra-dmo"` branch in `internal/voice/composer/composer.go` (dispatch at
       `runVoiceChain`, ~line 466) → a `runTETRADMOVoiceChain` that decodes DMO bursts → ACELP →
       PCM to the recorder. This is the architectural piece (TMO uses the traffic extractor; DMO
       is self-contained) and the main risk.
    4. **On-air A/B** on the operator's capture through the full daemon (recording lands, audio
       intelligible) before marking #1003 verified. A green offline decode ≠ on-air correct.
- **P25 Phase 1 weak-signal voice is the under-equipped decode path — diagnosed, NOT yet
  fixed (needs a capture).** An operator whose hardware Astro Spectra decodes a marginal
  P25 Phase 1 voice call cleanly gets only ~4-5 IMBE frames from GT on the same antenna.
  Root cause is structural, not a bug: the default **C4FM** voice receiver
  (`internal/radio/p25/phase1/receiver/receiver.go`, C4FM branch) is FM-discriminator →
  fixed matched filter → CoarseAFC → Mueller-Müller timing → AGC → 4-level slicer → **hard**
  Golay/Hamming IMBE FEC — with **no channel equalizer and no soft-decision FEC**. A blind
  CMA/FSE equalizer exists only on the opt-in `DemodCQPSK`/LSM path (`cqpsk.go`, the `fse`
  field), and P25 Phase 2 wires one too — P25 P1 C4FM is the odd path without either. These
  are the same two levers that ~2×'d TETRA yield on ISI/weak captures (`SnapshotCMA` +
  soft TCH/S). ("4-5 frames" also brushes `minAutotuneLDUs=5` in `composer/p25p1_voice.go`,
  which treats a <5-LDU call as too short to trust.) **The fix is unverifiable without a
  capture** (#764/#771): drop a raw C4FM Phase-1 **voice** IQ recording into `samples/p25/`
  (+ `.metadata.json`), baseline it with `TestReplayP25RealCaptureMetrics`
  (`cmd/gophertrunk/p25_realcapture_metrics_test.go`, tag `integration`; pre-FEC EVM/SNR/
  FSW-margin/LDU yield), then either port the `cqpsk.go` CMA/FSE equalizer onto the C4FM
  path or add soft-decision to the IMBE FEC, and A/B LDU/IMBE yield against the capture. No
  change lands without that capture. See `samples/p25/README.md` (weak-signal voice section).
