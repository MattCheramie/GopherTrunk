# Digital-protocol feature parity

A review of the advanced decode features GopherTrunk has accumulated —
mostly on the TETRA path, some on P25 — and where each one now stands
across the other digital protocols. This page records three things: the
capability matrix as of the parity series, the ports that landed (with
their A/B recipes), and the ports that were **deliberately not made** and
why. The governing rule throughout is the #764/#771 discipline from
`CLAUDE.md`: a green synthetic decode is never proof of an on-air fix, so
every decode-affecting port here landed opt-in/off-by-default (or with an
explicit no-harm argument), pinned by failing-first synthetic tests, with a
capture A/B recipe an operator can run.

## Capability matrix

Receiver / pipeline capabilities per protocol after the parity series
("opt-in" = off by default behind a per-system config key):

| Capability | TETRA TMO/DMO | P25 P1 C4FM | P25 P1 CQPSK | P25 P2 | DMR T3 | DMR T1/T2 | NXDN |
|---|---|---|---|---|---|---|---|
| Complex equalizer | SnapshotCMA on (rx) + SnapshotLMS (harness) | n/a (nonlinear path) | FSE always-on | CMA opt-in (`p25_phase2_equalizer`) | n/a | n/a | n/a |
| Soft-decision FEC | full (soft Viterbi RCPC/RM/TCH/S) | **opt-in (`p25_phase1_soft_decision`)** | — | opt-in (`p25_phase2_soft_decision`) | — | — | **opt-in (`nxdn_soft_decision`)** |
| Carrier recovery | carrierAFC (alias-reject, re-prime) | CoarseAFC (+opt-in DDA) | CV-gated seed + Costas | seed **now CV-gated** + Costas | CoarseAFC (post-clock) | CoarseAFC (post-clock) | none (follow-up) |
| DC block | voice paths | voice path | voice path | **opt-in (`p25_phase2_dc_block`)** | — | — | — |
| Decode-drought resync watchdog | checkResync + payload escalation | **yes (resyncGuard)** | **yes** | **yes** | **yes** | n/a (conventional) | **yes** |
| Diagnostic taps (soft/eye) | soft (complex diffs) | full | constellation | soft diffs | full | full | **full** |
| Symbol-scope panel | yes | yes | yes | no (backlog) | yes | yes | **yes (`nxdn`)** |
| Replay harness / GT_ knobs | richest (TMO+DMO) | fixtures + metrics test | — | GT_915_* | GT_DMR_* | GT_DMR_* | **GT_NXDN_*** |

Bold entries landed in the parity series. TETRA-only engine behaviours
(radio/talkgroup reclassification, discovery corroboration, notification
hold, channel folding, per-transmission split) are intentionally not in
this table — they rest on TETRA's GSSI/ISSI disjointness and are gated to
`protocol == "tetra"` on purpose (see "Not ported").

## What landed

### Zero-risk parity

- **`tetra-dmo` `.raw` sidecar** — DMO voice calls carry the same post-FEC
  TCH/S speech frames as TMO, so they now get the same always-on `.raw`
  sidecar (`internal/voice/recorder.go`).
- **NXDN diagnostic taps + symbol scope** — the NXDN receiver gained the
  same `SoftSink`/`EyeSink` taps the DMR/P25 receivers carry, and the web
  symbol panels gained an `nxdn` receiver, so an NXDN rig's panels no
  longer open a wrong-protocol receiver (the `symbol_proto` lesson).
- **NXDN replay harness** — `TestReplayNXDNRealCapture`
  (`cmd/gophertrunk/nxdn_realcapture_test.go`): `GT_NXDN_IQ` (cs16),
  `GT_NXDN_IQ_RATE` (default 48000), `GT_NXDN_ALLOW_EMPTY=1` for
  weak-signal baselines, `GT_NXDN_SOFT=1` for the soft-decision A/B.
  Reports FSW hits, CAC channel-decode/CRC yields (hard and soft), lock and
  grant counts. NXDN real-air captures remain the blocker for the whole
  NXDN voice path — see `docs/decoder-capture-needs.md`.
- **P25 Phase 2 carrier-seed multipath gate** — the Phase 1 CQPSK
  `seedModulusCV` gate (issue #492) is now applied to the Phase 2 coarse
  carrier seed. A simulcast-like two-ray channel biases the seed by ~1 kHz
  on a 0 Hz carrier and mis-tunes the NCO past the Costas pull-in for
  good; the gate detects the ISI a wrong de-rotation leaves and lets the
  Costas loop acquire instead (measured on the synthetic fixture: tail SER
  0.56 → 0.01 with the #915 CMA equalizer). Default-on: rejection falls
  back to the pre-existing weak-coherence behaviour, and a genuine
  1500 Hz seed still fires (pinned no-harm).

### Pipeline hygiene

- **Signal-time decode-drought watchdog** (`resyncGuard`,
  `internal/scanner/ccdecoder/resyncguard.go`) — the TETRA `checkResync`
  design generalised to the P25 Phase 1/2, DMR Tier III and NXDN CC
  pipelines and their widebandt2 taps: a full window of PROCESSED signal
  (not wall clock — starvation-immune) with no CRC-clean decode forces a
  fast reacquire from centre. Each control channel gained a
  `LastActivityNano` heartbeat stamped on its own decode unit (TSBK /
  MAC PDU / CSBK+MBC / CAC). DMR T3 and NXDN also gained `ResyncReset`:
  their Process adapters key on absolute dibit indices, and a receiver
  reset without dropping that state would silently discard every
  post-reset sync match (T3's `bufStart` wedge). Not wired on
  conventional/camped channels (DMR Tier I/II, TETRA DMO) where
  inter-transmission silence is normal; TETRA's `checkPayloadResync`
  escalation stays TETRA-only.
- **`p25_phase2_dc_block`** (opt-in) — the P1/TETRA voice-receiver
  DC-removal high-pass, ported to the Phase 2 traffic-channel receiver
  and threaded through the grant (`P25Phase2Decode.DCBlock`) to the
  composer, both pipeline factories, and every config surface. CC path
  stays DC-untouched, per the TETRA/P1 rule.

### Opt-in soft-decision FEC (the TETRA yield lever, ported)

- **`nxdn_soft_decision`** — the NXDN receiver derives two per-bit LLRs
  per dibit from the 4-level soft symbols (sign axis / magnitude
  threshold) and the spec CAC decode runs a true per-bit soft Viterbi
  (`framing.ViterbiK5Soft`, `nxdn.DecodeCACChannelSoft`,
  `ControlChannel.ProcessSoft`). Measured on the seeded synthetic channel:
  CAC CRC yield 26/200 hard → 151/200 soft at σ=0.7; at the level where
  hard decodes nothing the soft CC still locks. A/B a capture with
  `GT_NXDN_IQ=… GT_NXDN_SOFT=1 go test ./cmd/gophertrunk -run
  TestReplayNXDNRealCapture -v` and compare `cac_crc_ok` vs
  `cac_crc_ok_soft`.
- **`p25_phase1_soft_decision`** — the C4FM receiver emits the same
  per-bit LLRs (`BitLLRSink`), the control channel keeps them in lockstep
  with its dibit buffer (`StashSoft`, the TETRA contract — dropped to a
  hard fallback on any misalignment), and the TSBK trellis runs
  `framing.DecodeP25TrellisSoft` through the same framing, status-symbol
  stripping, rotation handling and CRC gate. Measured: TSBK yield 81/300
  hard → 212/300 soft at σ=0.75; 0 → 12/12 on the end-to-end CC fixture at
  σ=0.85. C4FM only (a CQPSK site decodes hard regardless). A/B a capture
  offline via siglab metadata (`p25_phase1_soft_decision: on`) against the
  `TestReplayP25RealCaptureMetrics` fixture flow, comparing TSBK CRC-ok
  counts.

Both stay **off by default** until an operator capture A/B confirms the
gain on air — the same posture the TETRA `SnapshotLMS` and P25 Phase 2
`#915` knobs shipped under.

## Deliberately NOT ported

1. **Complex equalizers (CMA/LMS/FSE/Snapshot\*) onto the C4FM family**
   (P25 P1 C4FM, DMR, NXDN). The FM discriminator is nonlinear: after it,
   a multipath channel is no longer a linear complex convolution, so the
   complex linear equalizers that ~2×'d TETRA yield do not model the
   channel. A real-valued post-discriminator DFE is new algorithm
   research, not a port. The portable lever for this family is
   soft-decision FEC — which is what landed.
2. **P25 P1 soft IMBE (voice) FEC.** `CLAUDE.md`'s weak-signal P25 note is
   explicit: no voice-quality change lands without the `samples/p25/`
   weak-signal voice capture. Soft Golay(23,12)/Hamming decoding is also
   new algorithm work (Chase-style), not a port. Deferred until that
   capture lands; the TSBK soft path above is the control-channel half of
   the same lever.
3. **TETRA engine niceties onto DMR** (radio/TG reclassification,
   discovery corroboration, channel fold, notification hold). DMR shares
   one 24-bit ID space between talkgroups and radios, so the TETRA
   invariant these rest on does not hold — the gating is deliberate and
   pinned by `engine_dmr_classify_test.go`.
4. **`checkPayloadResync`/MarkLost escalation to other protocols.** It was
   tuned against a specific observed TETRA field failure (the 19 Aug AFC
   alias latch). Porting the destructive escalation without an equivalent
   observed symptom is speculation; the first-tier drought reacquire is
   what generalises.
5. **DMR EMB QR(16,7) and RC(32,11) FEC.** The in-code posture notes
   (`dmr/emb.go`, `dmr/rc.go`) are explicit: no off-air capture and no
   open reference decoder implements these codes, so implementing
   spec-derived tables would be unvalidatable — the self-consistent
   synthetic trap (#764/#771; the TETRA class-2 CRC lesson). The parity
   is preserved raw; the embedded-LC BPTC+CRC downstream remains the
   integrity gate. (The "unapplied AMBE FEC" reading was stale: the
   composer's DMR voice chain FEC-decodes every AMBE frame via
   `dmrvoice.DecodeAMBEFrame` before the vocoder.)
6. **NXDN voice deinterleave / scramble / CAC structural changes.** All
   flagged "UNVERIFIED ON AIR" placeholders; capture-gated. The new
   `GT_NXDN_*` harness is how a contributed capture gets baselined.
7. **Backlog (new features, not ports):** a P25 Phase 2 symbol-scope
   panel; NXDN / DMR Tier I / TETRA DMO widebandt2 channels; the NXDN
   4800-baud BFSK receiver; NXDN AFC (port the DMR post-clock CoarseAFC
   pattern once an NXDN capture exists to validate against); a DMR
   pre-clock/freeze AFC stage (the acknowledged follow-up in
   `dmr/receiver/receiver.go`); production wiring for the TETRA traffic
   `SnapshotLMS` (on-air-gated per `CLAUDE.md`).

## Verifying

- `make vet test` — everything above is pinned by unit tests, including
  the failing-first ones named in each section.
- `make integration` — daemon-level CC decode for the pipelines the
  watchdog touched.
- Capture A/Bs (operator-run, the gates for turning any opt-in on by
  default): the `GT_NXDN_*` recipe above; the siglab
  `p25_phase1_soft_decision` A/B; `GT_915_*` for the Phase 2 seed gate on
  a real simulcast capture.
