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
  - **Production DMO decode pipeline is now WIRED (C2 done); only on-air A/B remains to close
    #1003.** The offline decoders are now driven by a real daemon pipeline, so a DMO capture no
    longer runs the wrong TMO CC path (the operator's `sch_pdus=0` / `sync gap` / `dsp resync`
    symptom). What landed:
    1. **`ProtocolTETRADMO` (`tetra-dmo`/`dmo`)** registered in `internal/trunking/site.go`
       (enum, `String()`, `ParseProtocol`, `Validate`) + `internal/scanner/ccdecoder/ddc.go`
       (`ddcTargetForProtocol` → 144 kHz) + the `factories` map in `pipelines.go`.
    2. **`newTETRADMOPipeline`** (`internal/scanner/ccdecoder/pipelines_dmo.go`): the TMO
       receiver knobs (`EnableEqualizer` blind CMA + `SoftSink` + AFC + channel filter) feeding a
       new **`tetra.DMStreamExtractor`** (`internal/radio/tetra/dmo_stream.go`) — a bounded
       sliding-window streaming adapter over the stateless `ExtractDMBurstsSoft` (emits each DSB/
       DNB once, in lead order, deduped). Decodes `DecodeDMSCHS` → **sticky** `KindCCLocked`
       (no cc.lost on inter-PTT silence, so a camped DMO channel isn't re-hunted) + SYNC-PDU FN
       liveness, `RecoverDMColourCode` once, and an **edge-triggered** `tetra-dmo` `KindGrant`
       on a DNB traffic train (re-armed after a `dmoGrantRearm` drought).
    3. **`runTETRADMOVoiceChain`** (`internal/voice/composer/tetra_dmo_voice.go`): a
       self-contained same-carrier chain (one same-carrier tap — `sameCarrierVoiceTaps` in
       `daemon.go`) that re-runs the extractor → `DMBurstTCHSpeechSoft` (recovers the colour
       independently, since the grant fires before the pipeline's recovery, decoding its buffer
       retroactively so no leading speech is lost) → ACELP (`tetra-dmo`→`tetra-acelp` in
       `recorder.go`). Dispatched from `composer.handleStart` on `proto == "tetra-dmo"`.
    - Pinned by `pipelines_dmo_test.go` (modulated-IQ lock+colour-3+grant), `dmo_stream_test.go`
      (chunk-invariance + soft decode), `tetra_dmo_voice_test.go` (colour recovery + retroactive
      emit), and `integration_cc_tetra_dmo_test.go` (full-daemon lock, `make integration`).
    4. **FIRST on-air run found the grant path was driven by NOISE, now fixed.** The operator's
      first live DMO run granted + opened a recording ~230 ms after startup on a silent channel,
      then never granted again — their real 10 s PTT (which *did* lock, `dsb_schs_crc=46/54`)
      produced nothing. One root cause: **a raw DNB detection is not evidence of traffic.** The
      DNB correlator is an 11-dibit training match at tolerance 2 under 8 filters (2 sequences ×
      4 rotations), so `Σ_{k≤2} C(11,k)·3^k = 529` of `4^11` match by chance ⇒ ~1.0e-3 per dibit
      position ⇒ **~18 false DNBs/s** at 18 kdibit/s. The operator's own log is the proof:
      `dnb_total` climbed 1076→4541 in 185 s (**18.7/s**) while `dsb_schs_crc` sat frozen at 46.
      `dmoGrantMinDNB=4` therefore tripped in ~0.22 s, `maybeGrant` never checked `p.locked`
      despite the counter being named `dnbSinceLock`, and the 3 s re-arm drought could never
      elapse against a 55 ms mean inter-arrival — *and* was only evaluated inside the DNB branch,
      so on a truly silent channel it could not fire at all. Fix: `tetra.DMSlotGrid`
      (`internal/radio/tetra/dmo_grid.go`) votes DNB leads onto the **255-dibit timeslot grid** —
      one radio on one clock puts every burst on one residue mod 255; noise is uniform over all
      255. The residue is **learned, never hardcoded** (a wrong spec-derived DSB→DNB lead offset
      would silently stop DMO granting, which is worse than over-granting), the latch is centred
      on the agreeing leads and tracks symbol slips, and it is **dropped when the train ends**
      (`dmGridTrainGap`) — without that, the ~0.2/s of noise landing on the latched residue keeps
      a finished transmission "alive" and the grant latches anyway. The grant now also requires
      `p.locked`, and the drought is evaluated from `Process`. Pinned failing-first by
      `TestTETRADMOPipelineIgnoresIdleChannel` / `GrantsOnlyAfterLock` / `RearmsBetweenTransmissions`
      (all three fail against the old code) plus `dmo_grid_test.go`. Note `buildDMODibitStream`
      now lays bursts on a true 255-dibit slot grid — the old arbitrary-filler layout was not a
      faithful transmitter and would not have caught this.
      Related, same run: the voice chain re-ran the 64-colour `RecoverDMColourCode` over its whole
      growing buffer on **every** burst (≈450k Viterbi decodes/call, `64·Σ(20..120)`), which
      starved its own same-carrier IQ tap (`dropped_chunks=5904`); it is now capped at 6 passes
      over a bounded scoring window, while still keeping the full buffer so leading speech is
      still decoded retroactively. And `flush()` set `colourKnown = true` outside the confidence
      gate, so a call that decoded nothing logged `colour=0 colour_known=true` — read as
      "recovered colour 0" and sends you after the wrong thing; `colourRecovered` is now separate.
    5. **STILL OPEN — on-air A/B (the #1003 gate).** A green synthetic/offline decode ≠ on-air
      correct (#764/#771): the operator must replay a clear 438.9 MHz DMO capture through the
      full daemon (`protocol: tetra-dmo`) and confirm a recording lands with intelligible audio.
      Watch on air: the DMO grant carries **no talkgroup** (`GroupID 0`, EN 300 396-3 call-control
      not decoded), so recordings file under group `0`; colour recovery needs ~20 DNBs, so a
      very short first PTT may grant before the colour is known (the voice chain re-recovers it);
      and the grant now lands ~0.5 s into a transmission (grid latch + 4 qualified DNBs) rather
      than instantly. `dnb_qualified` in the decode-status line is the number that means traffic —
      `dnb_total` is a noise meter, and a large gap between them is normal, not a fault.
    6. **On-air A/B ran (15aug): the ceiling is RF/model, NOT colour recovery — and colour
      recovery is behaving CORRECTLY.** The operator's purpose-built capture
      (`dmo_test_15aug/25sec_ptt_then_off_30sec_cs16_144khz.raw`, 25 s clear colour-0 silent PTT,
      144 kHz cs16) decodes DSB SCH/S at ~90% (105/117) but TCH/S sits near the chance floor at
      **every** DM colour. `TestTETRADMOColourScan` (`GT_TETRA_DMO_SCAN=1`, the new diagnostic)
      sweeps all 64 colours: several rise modestly at once (28→140, 57→74, 30→46 of 831 DNBs)
      rather than one dominating — which one radio scrambling with one label cannot produce, and
      which is why `RecoverDMColourCode`'s dominance gate (best ≥ 3×runner-up) correctly REFUSES
      to pick one (140 < 3×74). That refusal is the operator's "colour guessing problem": not a
      broken guesser but a marginal signal with no dominant colour. The SYNC PDU is self-consistent
      with MNI=0/colour=0 (extended colour 0, matching osmo-tetra-dmo's `tetra_scramb_get_init`
      offsets — GT's `ParseSyncPDU` bit offsets are identical), so the extended-colour path is not
      the gap here. The blind CMA equalizer (already on by default) lifts DSB 77→105 and TCH 80→140;
      LMS does not move it — so the receiver is already doing the right thing. NEXT: need a cleaner,
      KNOWN-colour (from the CPS codeplug), actually-TALKING DMO capture — the 25 s silent PTT is a
      poor vector (comfort-noise/DTX frames, and the sweep winners are partial keystream artifacts
      of a marginal signal). Do NOT change the descramble to fit a 33%-yield non-dominant colour
      (the #764/#771 self-consistency trap). The colour→colour_map is preserved in the diagnostic.
- **Vocoder "sounds awful" is a MEASURED, LOCALIZED AMBE+2 3600×2450 (DMR) high-band deficit —
  NOT RF, NOT post-processing.** Using the operator's DSD-FME (mbelib) decode of the SAME `.amb`
  frames as ground truth (`err=[0]`, identical frame count), GopherTrunk's `ambe2-dmr` decode has
  4–10× less energy above 1 kHz: octave band fractions (0-300/300-1k/1-2k/2-3k/3-4k) are
  mbelib `0.200/0.513/0.181/0.073/0.033` vs GT-raw-`ambe2-dmr` `0.405/0.540/0.041/0.008/0.005`.
  Content matches (envelope corr 0.69 — same speech, crushed highs). Isolation: the base `ambe2`
  (3600×2400) decoder shares the exact same synthesis + §6.2 `EnhanceAmplitudes` and gives
  mbelib-like highs (`0.121/0.053/0.020`), so the deficit is purely in `unpackParams2450`
  (`internal/voice/ambe2/params2450.go` + `tables2450.go`) — the 3600×2450 spectral-amplitude
  quantization reconstruction produces systematically low high-`l` amplitudes. NOT fixed this
  round: this is conformance-pinned table code, so the fix needs a frame-by-frame diff against a
  wired mbelib 2450 reference (an objective band-energy regression target), not a speculative table
  change. Config note: the operator's `warm_dmr_audio: true` + `enhance` LPF only make it worse
  (they cut highs further); the synthesis is the cause. Reproduce:
  `gophertrunk decode -in <dmr>.raw -out gt.wav -vocoder ambe2-dmr` and compare octave-band
  fractions to the DSD-FME decode of the paired `.amb`. IMBE (P25) shows the same direction, milder.
- **DSD-FME `.imb`/`.amb` interop VERIFIED (frame-exact); the "slow/pitch" playback is an upstream
  DSD-FME quirk.** `recordings.mbe_files` writes DSD-FME's native cookie-headed container and
  DSD-FME decodes it cleanly (`err=[0]`). DSD-FME's `-w single.wav` writer stamps an 8 kHz header
  on 12 kHz synthesis (file plays ~1.5× slow) — envelope correlation is 0.69 after a 1.5× time
  correction, proving content parity. Not a GT bug; use DSD-FME `-P` (per-call) or `-o pulse`, or
  relabel the `-w` WAV to 12 kHz. Documented in `docs/vocoders.md`.
- **Conventional DMR (Tier II / IPSC) now decodes a repeater's TWO timeslots as two calls.**
  A base-station DMR repeater carries TS1 and TS2 interleaved on one carrier, each able to hold
  a separate simultaneous talkgroup. The conventional path (`internal/radio/dmr/tier2`) used to
  collapse both: two symptoms — "DJ scratchy" audio (both slots' AMBE frames sliced into one
  superframe) and two talkgroups ping-ponging into one call. Root cause was NOT the DSP: (1) the
  composer picks single-slot vs interleaved decode from `Grant.DMRInterleavedVoice`, which Tier III
  stamped but Tier II never did — `dmr_interleaved_voice: true` in config was computed
  (`resolveDMRInterleavedVoice`, `daemon.go`) but dropped on the floor because `tier2.Options` had
  no such field; (2) grants carried `Timeslot 0` with scalar `inCall/lastTG/lastSrc` state, so the
  engine's `(freq, timeslot)` identity folded both slots. Fix wires `InterleavedVoice` through
  `tier2.Options` + both constructors (`ccdecoder/pipelines.go`, `widebandt2/engine.go`; Tier I
  direct mode stays single-slot), and replaces the scalar state with a per-destination map that
  assigns each concurrent call a **synthetic** Timeslot (1/2) — an engine-identity token only; the
  base-station wire format does NOT label a burst's physical slot (both slots share the BS sync
  words and the slot-type field carries only colour+datatype), and the composer's `slotRouter`
  routes audio by the embedded-LC talkgroup, not by this field. The Terminator-with-LC destination
  is now decoded (RS seed `RS129SeedTerminatorLC`) so a TS1 terminator releases only its own call;
  a lone active call still tears down promptly on an undecodable terminator, but with two concurrent
  calls an undecodable terminator defers to per-chain hangtime rather than cross-tear the other slot.
  Concurrent two-slot recording rides the pre-existing `dmrSameCarrierTaps = 2` taps (distinct
  serials → no composer per-serial collision). Pinned by `conventional_twoslot_test.go` (failing-first).
  **NOT yet on-air-verified (#764/#771 discipline):** the operator's only IQ grab
  (`dmr_ipsc_60sec_bw25k_cs16.raw`, 25 kHz cs16) is a dead capture — ~−75 dBFS RMS, carrier at
  −11.35 kHz, no frame sync — so the two-slot audio A/B still needs a decodable capture (a wideband
  `.raw`, or a properly-gained single-channel one). The synthetic regressions + the operator's own
  `debug.log` (interleaved never reaching the decoder; per-superframe log spam) corroborate the bug.
  Sharp edge to watch on air: if the embedded LC never decodes (#644), both same-carrier taps'
  `slotRouter`s fall back to phase parity and could bind the same phase (one slot recorded twice).
- **DMR group calls are no longer relabeled "individual."** The engine's known-radio →
  individual reclassification (`HandleGrant`, `internal/trunking/engine.go`) and `noteRadio`'s
  talkgroup retraction rest on a TETRA-only invariant (GSSIs and ISSIs never overlap). DMR shares
  one 24-bit space for talkgroup and radio IDs, so ungated it flipped DMR group calls whose TG
  equalled some radio ID — mis-filing recordings under `individual/<TG>/` and corrupting the
  `individual` column that backs the Radio IDs roster (its `LastTalkgroup` SQL excludes
  `individual=1` rows). The whole mechanism is now scoped to `g.Protocol == "tetra"`; DMR's FLCO
  classification is authoritative. Pinned by `engine_dmr_classify_test.go`.
- **The web symbol panels pick their receiver from the SYSTEM's protocol, not the P25 demod
  mode.** An operator on a TETRA rig saw a permanent `symbol: poor` badge next to a correct
  `decode: clean`. The two chips are deliberately different axes (frame-error rate vs raw
  constellation), but this was not that: `p25ModulationFor` (`cmd/gophertrunk/spectrum_provider.go`)
  only inspects `ProtocolP25` systems, so a TETRA/DMO/DMR-only config reported `""`, the web
  `demodModeToProto("")` fell back to `p25-c4fm`, and the panels opened a **P25 C4FM receiver on a
  π/4-DQPSK carrier**. Its 4-level soft track is meaningless there but *non-empty*, so
  `computeQuality` took the MER branch, measured ~9 dB, and `qualityVerdict` bucketed `<10 dB` as
  "poor" for ever. (Tell-tale: the Histogram panel with Mode manually set to TETRA showed
  `SNR (MER) —` and `balance ±5.7%`, which grades *clean* — two panels running different
  receivers on the same carrier.) Fix: `symbolProtoFor` + `SpectrumDevice.SymbolProto`
  (`symbol_proto`) resolve the actual `/diag/symbols` selector per protocol, and the web
  `autoProtoFor` prefers it. This also removes a whole wrong receiver per open panel.
  **Watch the CPU angle:** every `/diag/symbols` subscriber builds its own `Downconverter` from
  the FULL input rate plus a receiver (`internal/scanner/symbolscope/scope.go`), nothing is
  pooled, and Dashboard + Scanner + Plots each mount `useSignalQuality` while Histogram opens a
  fourth. Several open tabs = several full DSP chains. Pooling them is still open work.
- **`soapyremote: SDR overruns … host_drops` is a DOWNSTREAM signal, not a driver bug.**
  `sendOrDrop` (`internal/sdr/soapyremote/driver.go`) only sheds when the consumer stops draining
  a ~400 ms / ~1084-chunk channel, and it drops the OLDEST queued chunk, so each event is an IQ
  discontinuity mid-queue. `internal/sdr/soapyremote/` has not been modified since the repo
  import, so when this fires, look at what got slower on the decode side (the DMO colour brute
  force above was one such cause) — and check for the companion
  `ccdecoder: decode can't keep up with real time` WARN, which confirms CPU rather than network.
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
