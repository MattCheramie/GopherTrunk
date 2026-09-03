# GopherTrunk — guidance for Claude

GopherTrunk is a Go SDR trunking scanner/decoder (P25, DMR, NXDN, TETRA, …).
This file is standing guidance for AI-assisted work in this repo. Keep it short.

## Daily issue review

Open issues are reviewed **daily**. Each pass surfaces new issues and issues
with new responses since the previous day, and flags what needs work. Scope the
review to issues **updated since the prior day** unless asked for a full sweep,
and distinguish maintainer replies from genuine new reporter responses.

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

- **FLAC is now a first-class container on every recorder that had a container at all**,
  with ONE shared stereo encode core (`baseband.FLACIQEncoder` — `siglab.IQContainer`
  delegates to it) and a mono voice twin (`voice.FlacWriter`): `capture -format wav|flac`
  and the API staged capture route through `siglab.IQContainer` (both used to silently
  write a mislabeled/garbage body — the streaming `EncodeCapture` has NO container case
  and falls through to u8, so gen/synth now reject wav/flac at the boundary);
  `baseband.record[].format: flac` covers both taps (wideband + ddc) and the replay
  `FileDriver` mounts .flac back as a virtual tuner (content-sniffed via the fLaC
  marker, never the extension); `recordings.format: flac` switches per-call voice
  recordings, and the WHOLE downstream chain reads either container via content
  sniffing (`voice.ReadAudioSamples`): loudness normalize rewrites flac-in/flac-out,
  broadcast MP3 transcode, the `/calls/{id}/audio` handler (audio/flac + .flac in its
  extension allowlist), the retention sweeper, and the web RecordingPlayer's download
  name. `FlacWriter.DataBytes()` reports UNCOMPRESSED PCM bytes so the recorder's
  duration/dead-key math is container-independent. `diversity_capture` now takes
  `diversity_capture_format: flac` too (operator space-saving request): the branch
  FLAC is a bit-exact twin of the cs16 (same clamp/scale, pinned by
  `TestBranchRecorderFLACIsBitExactTwin`) so the alignment invariant and every
  downstream conclusion are container-independent; the harness content-sniffs the
  branches, and rates > 1 MS/s fall back to cs16 (STREAMINFO ceiling + the encode
  runs on the stream goroutine). Not converted (deliberately):
  `hunt -survey-capture` stays f32.
- **A P25 "MBT data CRC failed" line whose identity fields match a decoded broadcast
  is TWO different PDU frames, not a contradiction**: every field the failure line
  prints (opcode/blocks/nac) comes from the HEADER block, which carries its own
  CCITT-16 and decoded clean — only a data block failed its CRC-32 (RF residual
  errors; the broadcast repeats, so a clean copy usually logs nearby). The CRC span
  was re-verified byte-for-byte against OP25 `process_PDU` (single-block AMBT
  included), so do NOT chase a parser bug from that log pair alone; the line now
  carries the worst data-block Viterbi metric + an explanatory cause. Related naming
  rule now enforced in code: never render a vendor TSBK or an undecoded AMBT opcode
  through the standard `Opcode.String()` OSP map (it mislabels — MFID 0x90 opcode
  0x00 reads GRP_V_CH_GRANT); `ambtOpcodeLabel` names only the decoded AMBT forms.

- **P25 discovery: a PDU (DUID 0xC) on the control channel is Multi-Block
  Trunking, not noise — GT now decodes AMBT (`mbt.go`).** The operator's "only 1
  neighbor site, no WACN" report was this: their system broadcasts Network
  Status / Adjacent Status (with explicit downlink+uplink channels) only in AMBT
  form, and GT logged every one as `non-control DUID duid=PDU` and dropped it.
  MBT blocks reuse the TSBK 98-dibit trellis coding; the header ends in the same
  augmented CRC-CCITT16 as a TSBK trailer, the data blocks in a CRC-32 (OP25
  `process_PDU` is the validation reference; SDRTrunk AMBTC* classes the field
  layouts). An explicit uplink channel resolves as plain base+spacing —
  NO tx offset (uplink channel numbers already encode the uplink frequency).
  Also fixed in the same pass: TSBK SCCB (0x39) read channel B one byte early
  (`p[4:6]` vs the correct `p[5:7]`) and its round-trip test passed because the
  assembler encoded the same wrong layout — pin parsers with LITERAL byte
  vectors cross-checked against an independent decoder, not just round-trips.
- **Never gate a per-channel health WARN on absolute dBFS when decode evidence
  exists.** The widebandt2 "channel iq power very low" WARN fired every 5 s on a
  Tier III CC decoding every C_ALOHA at −56 dBFS. Any channel whose decode
  counter advanced within `lowPowerDecodeGrace` is healthy whatever its power
  gauge reads. Same family as the MRC dBFS-gate lesson below.

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
  7. **The 15aug "no dominant colour" was the MNI-0 BLIND SPOT, root-caused: the reporter's radios
      run a NON-ZERO MNI (Motorola MTP8500Ex, MCC 250 / MNC 1), and GT's colour recovery only ever
      searched the 6-bit colour with MNI 0.** TETRA seeds the scrambler from the FULL 30-bit extended
      colour code `get_init(mcc,mnc,colour)` (EN 300 392-2 §8.2.5.2 / osmo-tetra-dmo `tetra_scramb_get_init`;
      cross-checked against ctn008/TetraDMO-Receiver + lkurkela/osmo-tetra-dmo — GT's `ParseSyncPDU`
      offsets and `ExtendedColourCode` seed match osmo byte-for-byte). So on an MNI≠0 network the TCH/S
      traffic seed is `ExtendedColourCode(250,1,colour)`, and `RecoverDMColourCode`'s `uint32(c)` sweep over
      0..63 (= MNI 0) can NEVER reach it — every candidate sits at the chance floor and several rise
      modestly with none dominant, which is EXACTLY the 15aug signature (the earlier note read it as
      "marginal signal / partial keystream", but the real cause was the wrong MNI in the seed). The DSB
      SCH/S is always colour-0 scrambled and carries MNI 0 on air (both refs agree; it is NOT a parse
      bug — reading MCC/MNC=0 from the SCH/S is correct), so the network MNI must come from config.
      Fix (`RecoverDMColourCode(bursts, baseMNI)` + `tetra_mcc`/`tetra_mnc` system config, threaded
      through the DMO pipeline → grant `TETRADMOBaseMNI` → voice chain): fold the configured MNI into
      every colour candidate (`baseMNI | c`) and the clear-fallback seed. Pinned failing-first by
      `TestRecoverDMColourCodeNonZeroMNI` (scramble with `ExtendedColourCode(250,1,7)` from the
      independent osmo formula → the MNI-0 search finds nothing, the MNI-folded search recovers the exact
      seed). **STILL ON-AIR-GATED (#764/#771):** synthetic ≠ on-air. The operator must A/B their 438.9 MHz
      250/1 capture: `GT_TETRA_DMO_IQ=<cap> GT_TETRA_DMO_RATE=144000 GT_TETRA_DMO_MCC=250 GT_TETRA_DMO_MNC=1
      go test ./cmd/gophertrunk -run TestTETRADMOReplay -v` (and `-run TestTETRADMOColourScan` with
      `GT_TETRA_DMO_SCAN=1` + the same MCC/MNC to see whether a colour now dominates). If a colour
      dominates with MNI 250/1 where none did at MNI 0, that is the confirmation; if it still doesn't,
      the next suspect is the MNI value itself or DNB geometry — NOT encryption (the radios are TEA0).
  8. **20aug on-air run — "bogus DMO recordings that end at a timeout, not PTT release" is DIAGNOSED to
      the composer voice chain, NOT the control pipeline.** The operator's `dmo_test_20aug` run (X310,
      MRC on / single antenna / no cavity — their own caveat: a poor vector) shows the two DMO decode
      paths disagree sharply, and the recording symptom is a second-order effect of the voice chain's
      colour handling — pinned by the run's `debug.log`:
      - The control **pipeline** (`newTETRADMOPipeline`) decodes fine: locks, `RecoverDMColourCode`
        → **colour 39**, and `tch_crc` climbs to ~200 CRC-valid TCH/S across one PTT window.
      - The separate composer **voice chain** (`runTETRADMOVoiceChain`) produced a **silent** recording
        for that same PTT: grant fired at `colour_hint=0` (before the pipeline finished recovery), the
        chain buffered 242 DNBs, its buffer hit `dmoVoiceColourMax=120` and the give-up path fell back
        to **colour 0** (`d.colour = d.baseMNI`) BEFORE it ever adopted the pipeline's 39 — so all 242
        DNBs decoded as **BFI**, `speech_frames=0 colour_recovered=false`. `tryAdoptLiveColour` never
        cleared afterward either: its local re-verification (≥2 CRC-valid at the hinted colour on the
        chain's OWN bursts) kept failing even though the pipeline's bursts decode at 39 — i.e. the voice
        chain's receiver is producing bursts the pipeline's is not. Prime suspect for the divergence:
        the voice chain runs `EnableDCBlock: true` while the CC pipeline runs it **off** (`receiver.go`
        / the CC-path note above) — try `EnableDCBlock:false` in `tetra_dmo_voice.go` and A/B.
      - **Why it "ends at a timeout":** zero real speech frames ⇒ `boundaryTracker.onVoice` is never
        called ⇒ the call is torn down by hangtime as `reason=timeout` (~7 s) instead of at PTT release.
        DMO decodes no explicit release PDU, so even a GOOD DMO call ends on hangtime — but a decoding
        call keeps refreshing liveness and runs to the real tail; a BFI-only call dies at the first
        hangtime. So the fix for BOTH the bogus audio and the early cut is the same: make the voice
        chain actually decode (adopt the pipeline's colour instead of falling back to 0 / re-verifying
        with a divergent receiver).
      - **Not fixed this round (on-air-gated, #764/#771):** the calls that started AFTER the pipeline
        knew the colour (`colour_hint=39`) yielded only 2–4 speech frames, but the pipeline decoded ~0
        new `tch_crc` in those same windows too — weak for BOTH paths on this contaminated capture, so
        they don't isolate a voice-chain bug. Needs a clean, known-colour, actually-talking DMO capture
        (single antenna, MRC OFF, cavity filter) to A/B a voice-chain fix. The web-side symptom (a
        silent ~0.1 s WAV still published to History) is inherent: the recorder only drops `dataBytes==0`
        or `StatProvider` zero-voice calls, and ACELP is neither, so a 2-frame call becomes a tiny
        bogus row. Staged, not shipped.
- **TETRA MAC fragment reassembly had off-by-bits at every seam — found via D-NWRK-BROADCAST,
  and the neighbour-cell decode is now live + capture-verified.** `macFragmentPayload` skipped
  only type+subtype on MAC-FRAG (the fill-bit indication leaked into the payload, +1 bit) and
  fill+length on MAC-END (the slot-granting + channel-allocation flags leaked, +2 or more) —
  the round-trip test was green because its encoder shared the wrong layout (the #764/#771
  self-consistent trap, again), single-block PDUs were unaffected, and the corruption surfaced
  only in fragmented L3 PDUs from the seam onward. Diagnosed on the operator's 120 s 467.9125 MHz
  MRC capture: a D-NWRK-BROADCAST rotating neighbour list decoded ~2 cells then garbage; deleting
  exactly ONE bit at the seam made all seven advertised cells decode — the raw MAC-FRAG blocks
  confirm the payload starts one bit later (layouts pinned against osmo-tetra rx_macfrag/rx_macend).
  `ParseDNwrkBroadcast` (`mle_parse.go`, layout cross-checked osmo-tetra + tetra-kit) now feeds
  `TopologySnapshot.Neighbors` (Site=5-bit cell id, StatusFlags carries sync+mcc/mnc/la) → the
  systems report's "Neighbor sites"; cells surface only after the SAME content decodes twice —
  the 6-bit PD+type gate lets a rare corrupted-but-CRC-passing TL-SDU parse plausibly (one-shot
  surfaced 18 "neighbours"; the repeating 8 were real: carriers 467.99-470.00 MHz, LAs 1021-1089,
  `TestTETRANeighbourReportReplay` is the skip-guarded harness). **Follow-up (Sep 3): the operator's
  "totally bogus entries in the neighbours list, like a 1.5 GHz one" — confirmed-twice garbage — was
  TWO MORE MAC boundary holes, both deterministic (so identical corruption repeats and the
  confirm-twice gate cannot help): (1) `tmSDU`/`macFragmentPayload` handed everything to the BLOCK
  end — the MAC length indication (total PDU octets; osmo rx_resrc `macpdu_length*8`, tetra-kit
  `decodeLength(li)*8 - pos`) was parsed but never applied and the fill-bit indication never
  stripped fill ('1' then '0's, §23.4.3.2) — so fill/multiplexed tail bits leaked into the TL-SDU
  and D-NWRK-BROADCAST's trailing P-bit reads turned them into phantom optionals (band bits 1111 ⇒
  a 1.5 GHz "neighbour"); prefix-reading parsers never noticed, which is why everything else looked
  fine. (2) fragment reassembly had NO continuity check — a MAC-END spliced onto however-old a start
  fragment across lost blocks; now pieces must be stream-adjacent (`fragMaxGapDibits`, the NCDB
  detector's dibit position plumbed through `decodeDownlinkSlot`) and an AACH-confirmed control slot
  that decodes nothing abandons the chain (osmo-sq5bpf ages fragslots the same way). Test builders
  used to write a PLACEHOLDER length indication the decoder ignored — honouring the field made them
  encode real lengths (`stampMACResourceLength`), the encoder/decoder-drift trap in yet another
  dress. `plausibleNeighbourCell` (carrier≠0, band 1..9) is defence-in-depth behind those fixes, and
  the neighbour log/StatusFlags now print MCC/MNC/LA (and the newly surfaced §18.5.17 status
  optionals — raw values; their spec bit maps stay un-named until capture-confirmed, per the
  CommsType rule) only when present, so "mcc=0" can no longer mean "absent".** Related dedup in the same pass:
  retransmitted D-RELEASE/D-DISCONNECT now publish ONE `call.release` per teardown (`releaseSeen`,
  re-armed by a fresh grant) — the "release spam" report, same family as `grantSeen`/`lastTalker`.
- **The "sausages" in the operator's TETRA waterfall are the BS toggling discontinuous-downlink
  (timeshare/MCCH-sharing) mode per multiframe — not RF trouble and not a GT defect.** On both
  1-2 Sep 120 s MRC captures the CC's occupied bandwidth alternates ~±10 kHz ↔ ~±12.3 kHz in
  sharp ~1.02 s (= one multiframe) segments while total in-channel power stays constant (±0.3 dB)
  and branch fades are uncorrelated (r≈-0.06); slot-boundary ramp dips (14.2 ms) are always
  present, and the +2.4 kHz "tone" is the SB frequency-correction sequence, both normal. GT
  decodes straight through the segmented windows (BSCH ~99.9%, all harness arms at ceiling) —
  same family as the reporter's #925 SCBS/dynamic-MCCH-sharing observations. Don't chase a
  receiver bug from a "sausage" waterfall alone; check decode yield first.
- **TETRA individual/private-call SRC is restored across mid-call PDUs via a callID→source
  binding; cold-start group-vs-individual classification stays capture-gated.** ETSI compresses
  the SSIs out of mid-call and traffic-channel signalling (§14): once a call is set up, a
  D-CONNECT or a follow-on `MAC-RESOURCE` addressed only by call identifier carries NO party
  element, so the source went blank after the setup burst (the reporter's "missing SRC on private
  calls"). Fix mirrors the existing `callGroups` (callID→GSSI) map with `callSrc` (callID→calling/
  transmitting party ISSI), learned from D-SETUP's calling party and D-TX-GRANTED's transmitting
  party, dropped on release; `classifyParties` backfills a blank grant source from it (guarded
  against a `dest==src` self-source), and it flows to `trunking.Grant.SourceID` (the grant-dedup
  snapshot already keys on `src`, so a grant that GAINS a source re-publishes). Pinned by
  `TestCallSourceRestoredMidCall`. **What is NOT done, and why:** correctly flagging the FIRST PDU
  of an unseen private call as individual (vs surfacing the called ISSI as a phantom talkgroup)
  needs a single-PDU group/individual discriminator. Three independent decoders (sq5bpf, tetra-kit,
  Wireshark) confirm the D-SETUP layout and that the discriminator is the 2-bit **communication
  type** sub-field of Basic Service Information (bits 29..30 — GT now decodes it via
  `readBasicService` into `CMCEMessage.CommsType`, alongside circuit mode + the service encryption
  flag). BUT no decoder NAMES the enum values, and none of them classify from it — they defer to
  the downstream ecosystem (telive), primarily on temporary-address presence. So the mapping
  (0=p2p, 1=p2mp, 2=p2mp-ack, 3=broadcast) is spec-derived and capture-UNCONFIRMED. Per #764/#771
  it is NOT wired into `classifyParties`; instead the value is logged on D-SETUP (`comms_type_raw`
  in `tetra: d-setup basic service`, debug) so an operator's KNOWN individual-vs-group capture can
  confirm the split empirically. Only then does `CommsType` earn a place in classification — the
  named `Comms*` constants are staged for that one-line wiring. Downlink D-SETUP carries NO
  called-party element (only U-SETUP uplink does), confirming the dest must come from the MAC
  address, as GT does.
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
- **`sdr.input_sample_rate` is a systemwide pre-decimation stage at the Device boundary.** When
  set (and an exact integer multiple of `sdr.sample_rate`), the hardware runs at the higher NATIVE
  rate and `internal/sdr/decimate.Device` integer-decimates (polyphase anti-alias FIR,
  `dsp.NewResampler(1, M, …)`) down to `sample_rate` BEFORE anything downstream — DDC bank, demods,
  recording taps, `baseband.auto_record`/iqtap, spectrum. It is wired via `Pool.WrapDevice`
  (`sdrInputDecimator` in `daemon.go`) so it applies in `OpenWith` AND survives `Reacquire`; the
  wrapper programs the inner device at `sample_rate*M` and streams the M:1 result, so `sample_rate`
  stays the DECODE rate every existing `cfg.SDR.SampleRate` reader sees (zero downstream audit — the
  whole reason it wraps at the Device boundary rather than threading a native/effective split
  through the daemon). Two things it deliberately does NOT touch: the pre-combine `diversity_capture`
  tap lives INSIDE the soapyremote driver, below the wrapper, so it still records native branches
  (correct — a diversity A/B needs native); and it does NOT fix front-end degradation baked into a
  high native-rate capture (#764 — decimating 10→2.5 MS/s does not recover the Airspy's native-clock
  phase noise). It is a load/recording-size lever, not an RF fix. Pinned by
  `internal/sdr/decimate/decimate_test.go` (rate math, ActualSampleRate÷M, anti-alias rejection).
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
- **The Motorola Type II / SmartNet framing was FABRICATED and never matched the air interface —
  rebuilt from OP25/trunk-recorder (#1143), on-air verification still pending.** The original
  package (24-bit sync `0xA4D7AA`, 32-bit OSW, BCH(64,16,11)) matched no real reference; every
  synthetic test was green because encoder and decoder shared the invented format (the #764/#771
  self-consistent trap — same as the SoapyRemote opcode bug below), while no real capture could
  ever lock (`cchunt: hunt failed`, the reporter's symptom). The real format, ported from OP25
  `rx_smartnet.cc/h` + trunk-recorder `SmartnetParser` (both proven on air): 8-bit sync `0xAC`,
  84-bit frames back-to-back (a frame is only trusted when the NEXT sync arrives 76 bits later),
  76-bit payload → stride-19 deinterleave → (info,parity) pairs with `parity[i]=info[i]^info[i-1]`
  → 27 data + 10 CRC bits, data INVERTED on the wire (address `^0xCC38`, command `^0x0D5`, CRC
  complemented); OSW = 16-bit address + group bit + 10-bit command, where a command ≤ the band
  plan's range IS the voice channel number and grants/sysID span 1-3 consecutive OSWs (no
  opcodes — `motorola_bch_mode` is now accepted-but-ignored, `motorola_band_plan` selects
  800_standard/800_rebanded/800_splinter/900). Physical layer: 3600-baud 2-FSK at ±1.2 kHz
  deviation (NOT MSK/±900 Hz), channelized to an 18 kHz DDC target (5 sps, mirrors
  trunk-recorder; the old 48 kHz target's ±24 kHz passband also admitted 25 kHz-spaced
  neighbours into the discriminator), with a slow post-discriminator DC tracker — at ±1.2 kHz
  deviation a few-hundred-Hz carrier offset is a large slicer bias
  (`TestReceiverToleratesCarrierOffset`). Pinned by reference-literal tests (sync bits,
  interleave permutation, XOR masks — the only tests that catch constant drift) +
  failing-first `TestProcessDecodesRealAirFormat` (real-air stream → old decoder = zero
  decodes). Per #764/#771: synthetic-green ≠ on-air-verified — the #1143 reporter's 854.5625 MHz
  capture (Airspy R2, 3 MS/s cfile on Google Drive; unreachable from the dev environment's
  network policy) is the outstanding verification gate. Their "≈550.3 ppm" capture warning was
  almost certainly the probe estimator latching a different momentarily-strong carrier in the
  3 MHz span (the probe is the FIRST 32768 samples ≈ 11 ms, and `capture` never checks
  `ActualSampleRate` — though sample-count math shows their file really is 3 MS/s).
- **SoapyRemote RPC opcodes are an upstream enum, and a fake server cannot check them.**
  `callSetAntenna` was **600**, which is `HAS_DC_OFFSET_MODE`; `SET_ANTENNA` is **501**
  (`pothosware/SoapyRemote common/SoapyRemoteDefs.hpp`). The wire carries no schema, so the
  wrong id silently invoked a different handler whose first two arguments happened to match:
  `antennas: [RX1, RX2]` did nothing, every radio kept its driver default (which is why an
  X310 and a B210 behaved differently on one config), the leftover string produced the
  operator-visible `~SoapyRPCUnpacker: Unconsumed payload bytes 9` once per channel, and the
  `bool` reply carried no exception so `rpcVoid` logged "rx antenna set" for a call that never
  happened. **The unit tests could not catch this**: `driver_test.go`'s fake server switched on
  the same constant, so both sides moved together — the #764/#771 self-consistent-synthetic
  trap in a new dress. Two nets now, and they catch different things: the fake asserts every
  request body is FULLY CONSUMED (mirroring the real `~SoapyRPCUnpacker`, wired into
  `newFakeSoapyServer` so every test gets it) which catches ARGUMENT-SHAPE drift and found two
  more calls the fake had not been parsing; and `TestOpenSetAntennaUsesUpstreamOpcode` pins the
  numeric opcodes against upstream LITERALS, which is the only thing that can catch OPCODE
  drift. `applyAntennas` now validates against `LIST_ANTENNAS` (500) and reads back with
  `GET_ANTENNA` (502) — port names are device-specific (B210: `TX/RX`, `RX2`; TwinRX: `RX1`,
  `RX2`) so a config moved between rigs must fail loudly.
- **Never gate a DSP decision on absolute dBFS — gate on coherence.** MRC calibration was
  gated on the reference branch clearing −40 dBFS, so an operator raised front-end gain from
  65.0 → 82.0 dB purely to push a number past a software constant (they landed at −39.8 dBFS,
  clearing it by 0.2 dB). Any absolute-power gate is a gain-staging trap that re-fires on the
  next front end. The scale-invariant question is the normalised cross-correlation
  `|rho| = |Σ x1·conj(x0)| / sqrt(Σ|x0|²·Σ|x1|²)` (`diversity.CrossStats`), which is also what
  actually decides whether a gain estimate is trustworthy. `|rho| = γ/(1+γ)` for equal
  per-branch SNR γ makes thresholds interpretable (0.50 ≈ 0 dB, 0.35 ≈ −2.7 dB) against a
  noise-only floor near `sqrt(π/4N)`. Absolute power survives only as a digitally-dead-branch
  reject at −100 dBFS. **DC removal in the correlator is load-bearing, not hygiene**: both
  receivers of a front end share LO leakage, and an UNCENTRED correlator on two branches of
  independent noise plus a common DC reports `|rho| → 1` and freezes `h = dc1/dc0` — confidently
  calibrating on nothing, in exactly the weak-signal regime the gate protects. Pinned by
  `TestCrossStatsRejectsCommonDCOffset`.
- **Tracking MRC is safe ahead of a differential decoder; CMA was not — and the difference is
  structural, not a tuning choice.** `diversity.TrackingCalibrator` re-estimates the branch
  gain per 2 ms window and smooths one-pole (τ ≈ 200 ms), because a frozen constant is only
  right for a shared-LO single-chip front end. The reporter's `rx_subdev_spec=B:0 A:0` puts the
  branches on separate TwinRX daughterboards with independent PLLs — frequency-locked to a
  common reference, but random relative phase per lock and walking after it — so a constant
  decays. The SnapshotCMA lesson does not transfer: CMA's cost is rotation-invariant so nothing
  constrains its absolute output phase, whereas here `h_0` is pinned to exactly `1+0j`, so the
  output phase is ANCHORED to the reference branch's own and only the estimate ERROR can move
  it (`arg(y/x0) ≈ −ε·|h|²/(1+|h|²)`, zero at convergence). That claim is measured, not
  asserted: `TestTrackingCalibratorIsDifferentialSafe` bounds the per-window output phase step
  against the 45° π/4-DQPSK decision spacing. A rejected window **holds** the previous gains and
  never drops back to passthrough — falling back mid-stream is itself a large phase step, the
  exact failure class being avoided. `mrc-static` (α = 0) is the one-shot escape hatch; the
  first accepted window is snapped, not smoothed, so it is bit-identical to a one-shot LS
  calibration and both modes start the same way. Also fixed: the reference branch was
  re-chosen by `argmax` on EVERY datagram while uncalibrated, so an ordinary ~1 dB crossover
  between two healthy receivers swapped the phase anchor mid-stream.
- **MRC combines the WIDEBAND stream, so one complex scalar optimises whole-band SNR, not the
  target channel's.** That is exact only if the branches differ by a frequency-flat constant.
  Two antennas metres apart give each carrier its own phase difference set by geometry and
  direction of arrival; a scalar aligns whichever is loudest and partially cancels the rest.
  The signature is a wideband `coherence` stuck around 0.3–0.5 that no amount of tracking
  improves, and the fix for that regime is combining AFTER the per-channel DDC (one gain per
  narrowband channel) — a much larger change, not built. The coherence figure in the health
  line is what makes this visible instead of silent, which is arguably a bigger contribution
  than the gain-independence.
- **Diversity captures must be PRE-combine, or they answer nothing.** The combiner lives in the
  driver, so `baseband.auto_record`, the `iqtap` brokers and the scope taps are all downstream
  of it — a capture from any of them has one combiner already baked in and cannot be replayed
  through another. `sdr.soapy_remote[].diversity_capture` taps the branches straight after
  de-interleave, writing one headerless cs16 file per branch (each independently playable via
  `replay -format cs16`) plus a sidecar. **Alignment is the invariant**: a datagram that did not
  carry every branch is dropped from BOTH files and counted, never written short, because one
  short write silently desynchronises them and every later conclusion is wrong with nothing to
  show for it. A/B with `GT_DIVERSITY_CAPTURE=<prefix>.diversity.json go test ./cmd/gophertrunk
  -run TestDiversityCombinerReplay -v`: it prints a windowed coherence/gain/**phase** trace
  (flat phase ⇒ the frozen constant was fine on that hardware; walking phase ⇒ tracking is doing
  real work, in °/s) and decodes four arms — each branch alone, static, tracking — scored by
  CRC-clean BSCH. **Yield is the verdict, never EVM.** Tracking-as-default is NOT yet verified
  on air; that A/B on the operator's own capture is the gate.
- **The A/B ran on the operator's first pre-combine capture (17 Aug, X310 `rx_subdev_spec=B:0
  A:0`, internal clock, TETRA CC 467.9125 MHz, 250 kS/s, 30 s, 0 dropped datagrams) — the X310
  dual-daughterboard stream over SoapyRemote IS coherent, no external oscillator sync needed.**
  Wideband coherence median 0.658, narrowband (post-DDC 144 k) 0.713; branch phase ~145° walking
  only −0.22°/s (frozen constant decays over minutes, so tracking is right, but static is fine
  over any one calibration). Decode arms: branch0 885 / branch1 896 / wb-static 898 /
  wb-tracking 898 / nb arms 896 CRC-clean BSCH — MRC ≥ best branch (no harm) but this capture
  sits at its ~96% BSCH ceiling, so it CANNOT demonstrate a real gain; a weak-signal capture
  (BSCH well below ceiling on each branch alone) is what closes the tracking-as-default gate.
  Two calibrated readings of the same capture disagree by design, don't chase them: the live
  health line logged coherence 0.28–0.55 / branch_gain −11..−17 dB where the offline harness
  measures 0.66 / −6.4 dB power ratio — the driver's per-window LS estimate is biased low by
  noise (the logged gain IS the MRC weight, ∝ ρ·√(P₀/P₁), not a power ratio). Related bug found
  and fixed in the same pass: `mrcTrackAlpha` derived the loop coefficient from the NOMINAL 2 ms
  window while `mrcCalWindowSamples` clamps to ≥4096 samples, so at 250 kHz (16.4 ms windows)
  the tracking 1/e time was silently ~1.6 s instead of the documented 200 ms — alpha now follows
  the actual window duration (`TestMRCTrackAlphaHonorsClampedWindow`). Operator-visible symptom
  ruled OUT as a regression: the "bad" web constellation at 250 kHz is the honest π/4-DQPSK
  symbol plot (the `symbol_proto` fix stopped TETRA rigs drawing the flattering raw-IQ ring of a
  wrong P25 receiver) of a marginal signal plus ~400 Hz carrier offset (≈8° differential
  rotation); the same run decoded ~96% BSCH and recorded voice.
- **MMSE-IRC cannot work blind, and that is a property of the estimator, not of one
  formula.** The reporter's #1062 RFC proposes IRC to null a directional interferer that MRC
  amplifies. Two separate problems, and only the first is fixable inside the algorithm. (1) The
  RFC's residual `e_k = x_k − h_k·x_0` is degenerate: `h` is estimated by least squares against
  `x_0`, so `h_0 = 1` by construction and `e_0 ≡ 0`, `R_nn`'s reference row and column are zero,
  the diagonal-loading constant alone sets the reference weight and no null can form. Taking the
  residual against the MRC output instead fixes that. (2) The channel estimate itself is
  contaminated: with a co-channel interferer `cov(x_k,x_0) = h_wanted·P_signal + h_interf·P_interf`,
  so LS returns a POWER-WEIGHTED BLEND — measured, a true 0.95∠40° reads back as 0.32∠1°, and the
  null is steered at a mixture. Blind IRC measures **0.0 dB** over MRC on the synthetic
  co-channel scene; the same code given a training sequence measures **+23.6 dB**
  (`internal/dsp/diversity/irc_test.go`). A training sequence exists only AFTER the per-channel
  DDC — which is also where the per-channel combining the wideband limitation calls for has to
  go, so both roads lead past the DDC. `IRCCalibrator` is therefore an offline harness arm, not
  a driver mode.
- **The replay harness now measures narrowband coherence too, and that is the number that
  decides the architecture.** `TestDiversityCombinerReplay` traces coherence on the wideband
  stream AND after each branch's own DDC, then scores eight arms (add `wb-irc-blind`, `nb-static`,
  `nb-tracking`, `nb-branch0-only`). High narrowband against low wideband is direct proof that the
  single wideband gain is the limitation on that hardware. A calibration that locks once and then
  holds every window afterwards now WARNs after three health intervals — the operator's 17 Aug
  X310 log had `updates` frozen at 6682 for eleven minutes while `holds` climbed ~6000 per
  interval, and the line said INFO the whole time because `calibrated` was still true.
- **`branch_phase_deg` in the MRC log line is the no-capture field instrument.** Constant ⇒
  shared-LO front end; walking ⇒ independent PLLs. It tells an operator which hardware class
  they have, and therefore whether `mrc` or `mrc-static` is right, before anyone records
  anything.
- **A fixed coherence threshold was a BANDWIDTH-staging trap — the MRC gates now bound the
  ESTIMATE'S phase error, not |rho|.** Wideband |rho| is diluted by every hertz of noise-only
  bandwidth around the coherent carrier (`rho_wb ≈ rho_ch·sqrt(f0·f1)` for in-channel power
  fractions f_k), so the fixed 0.50 lock constant made calibration depend on the configured
  capture bandwidth and each branch's own noise floor. Proven by the operator's 18 Aug X310
  A/B (three 30 s pre-combine captures): at 200/250 kS/s + 70 dB gain the CC decoded 1425
  CRC-clean BSCH off branch 1 while wideband |rho| sat at ~0.16 (phase measurable to ~4° per
  4096-sample window, walking only −7°/s) and MRC NEVER calibrated — `updates=0`, WARN "check
  antennas/clock, raising RF gain will NOT help" — until +5 dB of gain pushed the number past
  the constant (their branch 0 is ~9 dB down with a gain-independent floor, so +5 dB lifted
  its in-channel fraction 0.14→0.37 and the diluted rho with it: the WARN's claim was
  exactly wrong in that regime, and the text now points at per-branch gain staging instead).
  The gates (`diversity.TrackingOptions.LockPhaseSigmaRad/TrackPhaseSigmaRad`, defaults
  0.10/0.16 rad) accept a window when the LS estimate's projected phase error
  `sqrt((1−ρ²)/(2Nρ²))` is bounded, i.e. `ρ ≥ 1/sqrt(1+2Nσ²)` — falls as 1/√N yet stays
  ~8×/~5× above the noise-only floor `sqrt(π/4N)` (false accept ~e^−50 / 3e−9 per window; noise
  still can NEVER lock, pinned by `TestTrackingCalibratorNeverLocksOnIndependentNoise`). The
  health/WARN line logs the effective `lock_gate` for the stream's window. Pinned failing-first
  by `TestTrackingCalibratorLocksOnBandwidthDilutedCoherence` +
  `TestMRCCombinerCalibratesOnBandwidthDilutedCoherence` (fixture ρ≈0.2 locks, recovers the
  true phase to <5°; under a 0.5-equivalent bound it never locks). Post-fix replay of all
  three captures: every arm calibrates on every run and wb-tracking matches the best branch
  within 1 BSCH (run1 1424 vs 1425, run2 401 vs 402, run3 1591 vs 1591 — no harm, and no gain
  is expected while branch 0 is floor-limited), and the harness arms now anchor on the LOUDER
  branch like the driver —
  they used to anchor on file-order branch 0, so on these captures (weak br0) every combined
  arm degenerated to the weak branch's passthrough and measured nothing. Two things the gate
  change does NOT alter: a genuinely dominant-carrier-misaligned rig (antennas metres apart)
  is still the wideband-scalar KNOWN LIMITATION — per-bin cross-phase on these captures shows
  distinct per-carrier phases (−157°/−87°/+20°) but the camped CC carries 71–94% of the
  cross-power so the scalar aligns it — and MRC on a 9 dB-down floor-limited branch is ~no-gain
  either way (the honest fix for run1/run2 is raising branch 0's gain, which run3 proved).
- **19 Aug X310 field log (10.5 h) — three verified failure modes, all fixed; and the real MRC
  bottleneck on this rig turned out to be an INTER-BRANCH TIMING SKEW, now aligned in the
  driver.** The operator swapped the antennas between RX1/RX2 mid-run: the ~5–10 dB branch
  deficit FOLLOWED the swap ⇒ antenna/feedline, not the TwinRX. Findings, each with a
  failing-first regression:
  - **TETRA CC "locked but deaf" ~37 min total (12 min in one stretch, log ends in it): the
    AFC's EMA can latch the wrong ±f_sym/4 alias bucket.** A decode-drought resync re-primes
    `omegaEMA` from ONE weak block; `omegaAliasReject` (π/4 = 2250 Hz) then discards every
    CORRECT estimate forever. BSCH survives the constant ~17°/sym residual (4-rotation search +
    heavy FEC) while all SCH fails ⇒ `bsch_ok` ~100%, `sch_pdus=0` — and every BSCH stamps the
    ONE activity heartbeat both the 1.5 s resync and 5 s CheckStale feed on, so no recovery
    could fire (the #815 WARN is advisory). Fixes: (1) `carrierAFC.track` re-primes after 3
    consecutive rejects that agree with each other (`afc_reprime_test.go`); (2) a second
    payload heartbeat (`LastPayloadNano`, stamped on CRC-clean SCH/F, SCH/HD, or the SB's
    SCH/HD-coded BNCH) drives `checkPayloadResync`: 12 s of signal with a live lock and no
    payload forces a resync, 3 fruitless resyncs escalate to `MarkLost` → re-hunt (mirrored in
    `widebandt2/tetra.go`). Also: a single mis-corrected BSCH could rewrite the locked MCC/MNC
    (4 bogus "tetra cc locked" flaps, mcc=996 etc.) — identity CHANGES now need 2 consecutive
    agreeing decodes, mirroring `colourConfirmThreshold`.
  - **"AFC spikes to −5 kHz" on a locked, centred TETRA CC = the 19 Aug re-prime escape
  hatch FIRING FALSELY on an alias-class transient — fixed with a confidence gate, and
  the #815 wrong-site WARN now requires persistence.** A reporter filmed the mixer panel's
  carrier readout jittering ±400 Hz then spiking to ~−5 kHz; the log's #815 WARN said
  `offset_hz=5004` on a carrier really ~500 Hz off. 5004 = 504 + 4500 and f_sym/4 = 4500 Hz:
  the spike is EXACTLY one AFC alias bucket, not a carrier move. Mechanism: the per-block raw
  estimate is coarse (spectral centroid) + fine (4×Δφ, unambiguous only within ±f_sym/8 =
  ±2250 Hz), so ANY transient coarse bias > 2250 Hz (a neighbour's skirts through the
  channel-filter edge while the wanted carrier fades) lands the raw estimate exactly ±4500 Hz
  off — and three such consecutive blocks (~170 ms) tripped the 19 Aug clustered-reject
  re-prime, slamming a long-corroborated EMA into the wrong bucket (SCH fails ~100% there;
  BSCH survives via per-burst SB correction, so the lock looks healthy). The two failure
  modes are the SAME signature at different ages: a wrong one-block seed (19 Aug) needs a
  FAST escape, a transient on an established track needs a HOLD. Fix (`afc.go`): `track`
  counts accepted blocks since the last prime (`accSincePrime`); an alias-class reject
  cluster (cluster mean ≡ EMA mod π/2, `aliasClassOffset`) on an established track
  (≥16 accepted blocks) needs `omegaReprimeStreakEstablished`=18 blocks (~1 s) to re-prime,
  while fresh tracks and non-alias clusters keep the fast 3-block escape; a wrong slow
  re-prime self-heals fast (it leaves a fresh track) and `checkPayloadResync` bounds the
  worst case. Pinned failing-first by `afc_alias_transient_test.go` (old code jumps to
  4950 Hz after exactly 3 alias blocks). Companion fix (`ccdecoder/decoder.go`): the #815
  WARN diagnosed a persistent condition from ONE per-chunk instantaneous sample, so every
  sub-second estimator blip produced a scary wrong-site WARN — the "fake triggering of freq
  offset" report; it now requires the total offset to sit over threshold continuously for
  `carrierOffsetWarnPersist` (10 s, test-overridable field), with the excursion clock reset
  on dips and on pipeline teardown. Pinned by `TestDecoderCarrierOffsetWarnRequiresPersistence`.
- **The MRC anchor flipped mid-stream on the 08:58 cc-hunt retune** (rearm set `refIdx=-1` →
    bare argmax re-pick), and the applied gain random-walked to −34 dB with `calibrated=true`
    (divergence bounds gated only the proposal). Fixes: rearm/`setSampleRate`/short-payloads
    KEEP the anchor (dead-incumbent escape still works), every anchor change is logged,
    `SetCenterFreq` short-circuits same-frequency retunes, and `clampMagnitude` floors the
    APPLIED gain (smoothed branch only — the snap/mrc-static equivalence pin is untouched).
    Differential safety is now also pinned at the driver's real α=0.1024 (200 kHz clamped
    window; the τ comment claimed windows ≪ a burst — false at low rates, corrected).
  - **The replay harness's "tracking" arms were silently STATIC** (`TrackingOptions{Alpha: 0}`
    = one-shot), so the 17/18 Aug "tracking matches best branch" verdicts compared static to
    static. Arms now derive α like `mrcTrackAlpha` and fatal on α=0.
  - **The big one: branch 0 lagged branch 1 by a CONSTANT 2.60 samples (13 µs at 200 kS/s).**
    Per-frequency coherence 0.99 with broadband ρ diluted to 0.78 is the signature of a pure
    delay — it is why this rig's wideband coherence never exceeded ~0.8 in ANY session. A
    scalar gain cannot represent a delay, so combining skewed branches decoded 22% FEWER BSCH
    than the best branch alone (886 vs 1142 on the 19 Aug capture) — MRC was HURTING; the
    identical combine after alignment scored exactly 1142/0. `soapyremote.branchAligner` now
    measures the skew per stream/retune (0.65 s buffer, ±16-lag scan + parabolic fraction,
    DC-removed, ρ≥0.1 latch gate) and delays the EARLY branch through an interpolating delay
    line, resetting the calibrator on latch so it locks on the aligned stream. The harness
    measures/reports the skew and carries a `wb-aligned-static` arm. Diagnose with
    `TestDiversityCombinerReplay`: "inter-branch delay: other lags ref by N±f samples".
    Open question for the next capture: whether the skew is per-stream (start skew) or fixed
    (DDC group delay) — the aligner re-measures per stream either way. A post-fix long run
    from the operator is still the on-air gate (#764/#771 discipline). The
  symbol/spectrum/diag/mixer handlers resolved the SDR serial AFTER `upgrader.Upgrade`, so an
  unknown device produced a successful 101 followed by a 1011 close. Browser clients reset
  their reconnect backoff in `onopen`, so a handshake that always succeeds makes `MAX_BACKOFF`
  dead code: one stale tab across a daemon restart with different hardware reconnected 2–4×/s
  forever, across six mount sites. Fixes, all three needed: resolve the device BEFORE the
  upgrade and return **404** (`api.ErrUnknownDevice`, `rejectUnknownDevice`); reset the client
  backoff only once a frame actually ARRIVES (or the socket held open past a grace period), not
  on `onopen`; and reconcile the selected serial against a refreshed device list instead of
  setting it once and never revisiting. `web/src/api/reconnectingSocket.ts` now owns the logic
  the four clients had each copied — including the `onerror`+`onclose` double-bind that
  scheduled two timers per failure while remembering only one handle.
- **29 Aug X310 field material (18.5 min debug.log + 60 s pre-combine capture at 200 kS/s) —
  the 19 Aug fixes HOLD on air, and the skew question is answered: PER-STREAM.** After the
  operator fixed the weak antenna/feedline (the 19 Aug deficit followed the swap), the branches
  sit balanced (~−51 dBFS each, branch_gain within ±1.4 dB) and the whole session is clean:
  wideband coherence 0.95–0.96 every health interval (vs ≤0.8 in every pre-aligner session),
  `updates` climbing continuously with `holds=0`, no anchor flips, ONE benign WARN in the whole
  log, CC locked in 0.3 s and never lost (`bsch_fail=0` in every 5 s status line), 0 overruns,
  3 drought resyncs in 18.5 min, voice flowing (demux total tch_frames=7052, vocoder_drops=0
  per call). The inter-branch skew measured **0.41 samples** on this stream where 19 Aug
  measured 2.60 on the same rig — so the skew is a per-stream START skew, not a fixed DDC
  group delay, and the aligner's re-measure-per-stream/retune design is the right one (latched
  at peak |rho|=0.94). Offline A/B on the capture (`TestDiversityCombinerReplay`): branch0
  2589 / branch1 2626 / wb-static 2625 / wb-tracking 2626 / wb-irc-blind 2626 /
  wb-aligned-static 2625 / nb-static 2625 / nb-tracking 2626 — every combined arm matches the
  best branch (NO harm, the pre-aligner 22%-loss regime is gone), narrowband coherence (0.958)
  ≈ wideband (0.945) so the wideband scalar is NOT the bottleneck on this rig, and phase walks
  −0.11°/s (a frozen constant decays over minutes ⇒ tracking stays the right TwinRX default).
  What this capture CANNOT close: it decodes at its ~100% BSCH ceiling, so a real MRC gain
  over the best branch is still undemonstrated — that gate needs a WEAK-signal capture
  (per-branch BSCH well below ceiling), not a longer one. `diversity_capture_seconds` cap
  raised 60→120 on the operator's request (at 200 kS/s two CS16 branches are ~1.6 MB/s total;
  the 1 GiB/branch recorder cap still bounds high rates). The demux teardown counters that look
  alarming in this log (`undecoded_drops=24784`, `concurrency_suppressed=33125`) are by-design
  cross-slot-leak protection on a busy multi-slot carrier, not defects.
- **A green `ci.yml` does not mean the web console builds.** `npm test` (vitest) transpiles
  with esbuild and never typechecks, so the SPA jobs in `ci.yml` pass on code that
  `npm run build` (`tsc --noEmit && vite build`, i.e. `make web-build`/`make dist`) rejects.
  Three `TS6133` unused-parameter errors in `web/src/api/reconnectingSocket.test.ts` landed on
  main that way and broke the build for an operator. The only job that caught them was
  **"Windows installer (PR)"** — which is not a required check, so the PR merged red and it
  read as "CI passed". Two things follow: **when a web change is in the diff, run
  `cd web && npm run typecheck` (or `make web-build`) before pushing** — `npm test` alone is
  not sufficient evidence; and check a PR's non-required jobs, not just the required ones.
  `ci.yml`'s `web-typecheck` job now runs `tsc --noEmit` across all five SPAs (`web`,
  `configbuilder`, `siglab`, `rfscope`, `cryptolab` — the last two are built by no other
  workflow). Note `tsconfig.json` sets `noUnusedLocals`/`noUnusedParameters`, so an unused
  parameter needs a leading underscore (`_fn`), and test files are inside `include: ["src"]`
  and are typechecked like production code.
- **A graceful-shutdown window is a lie unless something tells the handlers to leave.**
  Ctrl-C took a fixed 30 s. `Daemon.Close` stops the HTTP server FIRST, and
  `http.Server.Shutdown` waits for active non-hijacked requests without cancelling their
  `r.Context()` — while the `bus.Close()` that would end `handleSSE` runs at the very END of
  Close, ~90 lines later. So any attached SSE / live-audio / siglab subscriber pinned the whole
  window, then `Shutdown` returned `context.DeadlineExceeded`, which `daemon.spawn` counts as a
  clean exit. Completely silent. The window had already been widened 5 s → 30 s "to cut
  user-visible connection drops", which made the pause 6× longer without making anything drain —
  the reason this kept coming back. `api.Server` now closes a `stopStreams` channel at the top of
  `shutdown()` and the long-lived handlers select on it; the 30 s is a CAP for a handler that
  misses the signal, and reaching it WARNs. Two more teardown stalls hid behind it:
  `GRPCServer.Stop` was an unbounded `GracefulStop` (an open `StreamAudio` waits forever), and
  the soapyremote/rtl_tcp read loops checked `ctx.Done()` only BETWEEN reads and then parked in
  `io.ReadFull` behind a 30 s deadline — a quiet server is the normal state at shutdown, so both
  added their own 30 s tail. Watch for this shape anywhere: a bounded wait whose exit condition
  is satisfied later in the same teardown.
- **`cmd/gophertrunk/integration_test.go` used to give `Run` 3 s and then move on silently**,
  which is why a 30 s shutdown stall never failed CI. A cleanup that swallows a timeout is not a
  test. It asserts now.
- **Every SDR source list has to be named in the pool-construction gate at `daemon.go`.** The
  gate read `SDR.Devices || Baseband.Replay || SDR.RTLTCP` and the network drivers are registered
  INSIDE that block — so a config with only `sdr.soapy_remote` (or only `ka9q_radio`) registered
  no driver at all and the daemon started quietly with no radio. Adding a new source type means
  adding it to that condition; `TestRemoteOnlySDRConfigsRegisterTheirDriver` pins it, asserting on
  REGISTRATION rather than on `d.pool` because the pool is set to nil whenever it fails to open.
- **The SPA client's response-envelope keys are unchecked and fail silently.** The call-history
  client read `r.rows` where the daemon sends `{"calls":[…]}`, with a `?? []` fallback — so the
  History panel rendered "No calls in the daemon's call log for this filter" for every query
  since the file was written, and it read as "search by talkgroup is broken" while the handler,
  the filter and the SQL were all correct. `?? []` on a mistyped key is indistinguishable from an
  empty result. Panels without a test are where this hides; History had none.
- **Operator-applied names live in a `labels` table, layered over the alias files at startup.**
  `PATCH /api/v1/rids/{id}` used to 404 for a radio absent from `rid_alias_file` — i.e. exactly
  the radios worth naming, since one showing up live is by definition not in a file yet — and
  `PATCH /api/v1/talkgroups/{id}` had no name field at all. Both now create the catalogue entry,
  using the CSV loaders' own defaults so a synthesised row behaves like a loaded one. A
  synthesised talkgroup must NOT be tagged `Discovered`: `DeleteDiscovered` would sweep away the
  operator's name behind their back. The label wins over the file (it is the operator's most
  recent explicit act) and logs a WARN when they disagree. Keyed `(kind, system, target_id)`
  because `talkgroup_file` / `rid_alias_file` are per-system config keys and the CSV export has
  to be able to emit one file per system. Only the naming fields persist; priority / lockout /
  watch / scan stay in memory as before.
