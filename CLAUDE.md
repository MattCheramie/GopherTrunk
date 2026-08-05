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
