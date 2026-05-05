# Build Phases

GopherTrunk is being built incrementally. Each phase is independently
buildable and testable; the project stays useful even if work pauses partway.

| Phase | Title                                  | Status      |
| ----- | -------------------------------------- | ----------- |
| 0     | Foundation                             | done        |
| 1     | SDR hardware layer (CGO librtlsdr)     | done        |
| 2     | DSP core (channelizer, demods)         | done        |
| 3     | P25 trunking (Phase 1 then Phase 2)    | partial     |
| 3.5   | System ID & control-channel hunting    | done        |
| 4     | DMR trunking (Tier II + Tier III)      | partial     |
| 5     | NXDN trunking                          | partial     |
| 6     | Trunking engine (grant follower)       | partial     |
| 7a    | Voice passthrough (FM, raw frames)     | done        |
| 7b    | IMBE (P25 Phase 1, default)            | upcoming    |
| 7c    | AMBE+2 (mbelib build tag, DVSI)        | upcoming    |
| 8     | API (gRPC + WebSocket)                 | deferred    |
| 9     | Persistence + recording                | upcoming    |
| 10    | Hardening (metrics, reconnect, docker) | upcoming    |

## Phase 0 — Foundation

- `go.mod`, `Makefile`, `.gitignore`, `.github/workflows/ci.yml`.
- `cmd/gophertrunk` daemon scaffold with `version`, `sdr list`, `run`.
- `internal/config` YAML loader + validation.
- `internal/log` `log/slog` factory.
- `internal/events` in-process pub/sub bus.
- `internal/version` build-time stamp.

## Phase 1 — SDR Hardware Layer

- `internal/sdr` driver registry, `Device` interface, `Pool` with role
  assignment.
- `internal/sdr/rtlsdr` thin CGO binding to librtlsdr; async read bridged
  to `chan []complex64`; PPM, gain, sample-rate, center-freq controls;
  DC blocker and IQ-imbalance correction in `calibrate.go`.
- File-backed mock drivers (`u8` and `f32`) for offline replay and tests.

Verification:
- `go build ./...` (links `-lrtlsdr`).
- `go test -race -count=1 ./...` covers config, events, calibration,
  pool, mock device. CGO binding paths are exercised by `go vet ./...`.
- `gophertrunk version` prints the embedded version.
- `gophertrunk sdr list` enumerates attached dongles when present;
  prints "no SDR devices found" otherwise.

## Phase 2 — DSP Core

- `internal/dsp/window` standard window functions (Hann, Hamming, Blackman,
  Kaiser, Rect).
- `internal/dsp/fft` swappable FFT plan interface, default backend wraps
  `gonum.org/v1/gonum/dsp/fourier`.
- `internal/dsp/filter` FIR (with stateful history), Kaiser-window LPF
  designer, root-raised-cosine pulse shaping, CIC decimator, halfband LPF.
- `internal/dsp/agc` AGC feedback loop.
- `internal/dsp/resampler` polyphase rational resampler (L/M).
- `internal/dsp/channelizer` M-channel critically-sampled polyphase
  channelizer; FFT-rotated branch outputs.
- `internal/dsp/demod` quadrature FM, C4FM (RRC matched filter + 4-level
  slicer), H-DQPSK (differential QPSK with configurable rotation).
- `internal/dsp/sync` Mueller-Müller symbol-timing recovery and a
  correlator-based frame sync.
- Verification (no external golden vectors required): impulse response
  equality for FIR; passband/stopband power tests for Kaiser LPF; tone
  steering verified for the channelizer (positive +k·Fs/M tone lands in
  channel k with adjacent-channel rejection > 20 dB); AGC convergence;
  RRC unit-energy and matched-filter peak-at-center; FM demod against a
  linear chirp; D-QPSK against constant phase steps.

## Phase 3 — P25 trunking (in progress)

Landed in this phase:

- `internal/radio/framing` shared FEC primitives — bit packing,
  CRC-CCITT/FALSE, Hamming(15,11,3), extended Golay(24,12,8), and a
  generic 4-state 1/2-rate convolutional Viterbi parameterised by next-
  state and output-dibit tables (P25 trellis polynomials plug in here).
- `internal/radio/p25/phase1`:
  - `sync.go` — 48-bit P25 frame-sync word, dibit-stream sync detector,
    and TIA-102 C4FM symbol-to-dibit mapping.
  - `nid.go` — Network ID parser (NAC + DUID enum).
  - `tsbk.go` — Trunking Signaling Block parse/assemble with CRC trailer.
  - `opcodes.go` — TSBK opcode constants and payload parsers for
    GroupVoiceChannelGrant (0x00), GroupVoiceChannelUpdate (0x02),
    NetworkStatusBroadcast (0x3B), and RFSSStatusBroadcast (0x3A).
  - `control.go` — control-channel state that emits `cc.locked` and
    `cc.lost` on the events bus.

Deferred to follow-up phases:
- Full BCH(63,16,11) decoder for NID validation.
- P25-specific trellis next-state / output-dibit tables (TIA-102.BAAA
  Annex A) and the TSBK block interleaver.
- LDU1/LDU2 (voice frames) — they need Reed-Solomon and the IMBE
  decoder, both Phase 7 territory.
- P25 Phase 2 (TDMA H-DQPSK superframes) — separate phase.

## Phase 3.5 — System Identification & Control Channel Hunting

Landed in this phase:

- `internal/trunking/site.go` — `System` type (Name, Protocol enum,
  candidate ControlChannels list, WACN/SYSID/RFSS/Site identifiers) +
  `HuntOrder()` which biases scanning toward the cached last-known CC.
- `internal/trunking/cache.go` — JSON-on-disk cache of last-known CC
  frequency + NAC per system, with atomic rename on write.
- `internal/trunking/cchunt.go` — `Hunter` coordinator. Subscribes to
  the events bus, walks the candidate frequency list, retunes the
  control-role SDR (via a narrow `Tuner` interface so unit tests can
  use a fake), parks on the first matching `cc.locked` event within
  the per-frequency dwell window, and persists the locked frequency.
- Tests cover validation, cache round-trip + atomic write, biased
  hunt order, lock-on-first-responsive-frequency, full-sweep with no
  response, lock priority for cached frequency, freq mismatch
  rejection, and context-cancel propagation.

Wiring into the daemon (`cmd/gophertrunk`) belongs to Phase 6 along
with the demod pipeline that ultimately publishes `cc.locked` for the
hunter to consume — in this phase the hunter is library-ready and
unit-tested but not yet reachable from the CLI.

## Phase 4 — DMR Trunking (in progress)

Landed in this phase:

- `internal/radio/framing/hamming1393.go` — Hamming(13,9,3) encoder +
  single-error-correcting decoder. Used as the BPTC column code.
- `internal/radio/framing/bptc.go` — BPTC(196,96) encoder + iterative
  Hamming row/column decoder, plus the channel interleaver
  (out[i] = in[(i*181) mod 196]) and its inverse.
- `internal/radio/dmr/sync.go` — All 9 ETSI sync patterns (BS-Voice,
  BS-Data, MS-Voice, MS-Data, MS-RC, DM-Voice T1/T2, DM-Data T1/T2)
  as 48-bit constants and 24-dibit decompositions, plus a sliding
  sync detector.
- `internal/radio/dmr/burst.go` — DMR burst layout (132 dibits = 49 +
  5 + 24 + 5 + 49) with helpers to extract each section. PayloadBits
  concatenates the two info halves into the 196-bit BPTC codeword.
- `internal/radio/dmr/slottype.go` — Color Code + Data Type enum
  (CSBK / VoiceLCHeader / Idle / etc.) with assemble/parse.
- `internal/radio/dmr/tier3/csbk.go` — CSBK assemble/parse with CRC
  trailer, opcode enum (Aloha, RAND, Ahoy, MoveTSCC, Preamble,
  TV/PV/TD/PD-Grant, AdjacentSiteStatus, SystemInfo).
- `internal/radio/dmr/tier3/payloads.go` — payload parsers for
  TalkGroup/Private Voice grants, Aloha, AdjacentSiteStatus, and
  SystemInfoBroadcast.
- `internal/radio/dmr/tier3/control.go` — control-channel state
  machine that consumes bursts whose Slot Type identifies a CSBK,
  runs BPTC + CRC, and emits `cc.locked` / `cc.lost` events with a
  DMR-specific LockState payload.

Tests cover Hamming(13,9,3) full round-trip + every single-bit error
position; BPTC encode→decode round-trip, single-bit error correction
across all 196 positions, all-zero/all-one fills, and interleave-as-
permutation; DMR sync hex distinctness, dibit decomposition, clean +
tolerant matching, and best-match selection; burst slice geometry and
PayloadBits unpacking; CSBK round-trip + CRC corruption detection;
opcode payload parsers; and the control-channel emission path.

Deferred to follow-up phases:
- Hamming(20,8) over the 20-bit slot-type field (ETSI Annex B.1.4).
- Embedded LC reassembly across superframes for voice bursts.
- Tier II conventional / repeater operation distinct from the Tier
  III scaffolding (Tier II is mostly a configuration variation).
- Voice burst payload (two AMBE+2 frames per burst) — Phase 7.
- Vendor extensions behind FID != 0 (Hytera, Motorola Connect+).

## Phase 5 — NXDN Trunking (in progress)

Landed in this phase:

- `internal/radio/nxdn/frame.go` — Frame layout constants for the
  192-dibit NXDN frame: FSW (8 dibits) + LICH (8 wire dibits) + SACCH
  (32 dibits) + Information field (144 dibits). `BaudRate` enum
  selects between 4800 (BFSK) and 9600 (4-FSK) channel rates that
  share the same logical structure.
- `internal/radio/nxdn/lich.go` — 8-bit Link Information Channel
  parser and assembler with the field decomposition (RFCh + FCT +
  Option + Direction + Parity), even-parity validation, and a soft
  decoder for the on-air 16-bit doubled wire form (each info bit
  transmitted twice).
- `internal/radio/nxdn/sync.go` — Frame Sync Word constants
  (outbound BS→MS and inbound MS→BS) as 16-bit hex + 8-dibit
  decompositions, plus a sliding sync detector with configurable
  tolerance and per-direction match reporting.
- `internal/radio/nxdn/cac.go` — CAC (Common Access Channel)
  message-level parser/assembler with CRC-CCITT trailer; RCCH opcode
  enum (VCALL, VCALL_ASSGN, DCALL, SDCALL, SITE_INFO, SRV_INFO,
  CCH_ANNOUNCE); payload parsers for VCALL and SITE_INFO.
- `internal/radio/nxdn/control.go` — Control-channel state machine
  consuming a parsed LICH + CAC, validating RFCh = RCCH and the
  LICH parity, and publishing `cc.locked` / `cc.lost` events with an
  NXDN-specific LockState payload (FrequencyHz + BaudRate + SiteID +
  SystemID).

Tests cover frame layout consistency, LICH round-trip with parity
verification + per-bit corruption detection, LICH wire-form encode/
decode + majority voting on a single corrupted bit, FSW pattern
matching with clean / tolerant / over-budget cases and inbound vs
outbound discrimination, CAC round-trip + CRC failure, payload
parsers, and the control-channel emission path.

Deferred to follow-up phases:
- Final cross-check of FSW values against the published NXDN
  technical document (the constants above match common reference
  implementations but should be verified before live captures).
- SACCH FEC (Reed-Muller / convolutional + scrambler) and the
  sub-frame interleaver across the 144-dibit information field.
- Full RCCH state machine (paging, channel-grant follow, neighbor-
  list reaction) — Phase 6 along with the engine.
- Voice frame AMBE+2 payloads — Phase 7.

## Phase 6 — Trunking Engine (in progress)

Landed in this phase:

- `internal/trunking/grant.go` — Cross-protocol `Grant` payload type
  (System, Protocol, GroupID, SourceID, FrequencyHz, ChannelID/Num,
  Encrypted, Emergency, DataCall) used by P25/DMR/NXDN decoders to
  publish to the events bus, plus `EndReason` enum and the
  `CallStart` / `CallEnd` event payloads consumed by the API and
  recorder layers.
- `internal/trunking/talkgroup.go` — `TalkGroup` model + thread-safe
  `TalkgroupDB` with Trunk-Recorder-style CSV loader (Decimal /
  Alpha Tag / Mode / Tag / Group / Priority / Lockout — including
  the "L" priority sentinel for inferred lockouts) and a JSON
  loader.
- `internal/trunking/priority.go` — `EffectivePriority(grant,
  talkgroup)` + `CanPreempt(active, incoming)` with the
  Trunk-Recorder convention: lower number = higher priority,
  emergency = priority 0 (above everything), strict-higher
  preemption (equal does NOT preempt), and lockout drops the
  incoming grant outright.
- `internal/trunking/voicepool.go` — `VoicePool` allocator over the
  Voice-role SDR list (using the existing `Tuner` interface).
  `FindFree` for first-fit allocation, `LowestPriorityActive` for
  preemption decisions, `Bind` retunes + records an `ActiveCall`,
  `Release` frees, `Touch` refreshes the watchdog timestamp.
- `internal/trunking/engine.go` — Central state machine. Subscribes
  to `events.KindGrant` at construction time so callers can publish
  before `Run` starts, dispatches grants through lookup → priority
  → pool, emits `events.KindCallStart` / `events.KindCallEnd`,
  preempts strictly-lower-priority calls when the pool is full, and
  runs a watchdog that ends silent calls after `CallTimeout`
  (default 30 s).

Tests cover Grant string formatting, talkgroup CSV + JSON round-
trip including the Trunk-Recorder priority "L" sentinel, priority +
preemption logic across emergency / lockout / equal / nil-talkgroup
cases, voice-pool allocation + retuning + release + lowest-priority
selection + watchdog touch, the engine's grant dispatch path
(start-on-grant, drop-locked-out, drop-zero-frequency,
preempt-lower-priority, hold-on-equal, emergency-preempts-anything,
explicit EndCall, watchdog timeout via injected clock), and the Run
loop dispatching events published on the bus.

Bug caught and fixed during testing: the engine originally
subscribed to the bus inside `Run`, which lost any grant published
between `NewEngine` and the first `Run` iteration; moved the
subscription into `NewEngine` with a paired `Close()` so the engine
is reachable as soon as it's constructed.

Deferred to follow-up phases:
- Wiring protocol decoders (P25 phase1 / DMR tier3 / NXDN) to
  publish `events.KindGrant` when they decode an actual grant
  opcode — this is gated on completing the Phase 3 / 4 / 5
  channel-coding deferrals (P25 trellis tables + interleaver, DMR
  slot-type Hamming(20,8), NXDN SACCH FEC) and on a band-plan
  resolver that converts (channel ID, channel number) → Hz.
- Demod-pipeline composer that subscribes to `CallStart` events,
  tunes a per-call demod chain on the bound Voice device, calls
  `Engine.Touch` on every voice frame, and `Engine.EndCall` on
  voice-channel release announcements or PTT-off.
- Multi-site neighbor list + affiliation-based site selection
  (extension to `internal/trunking/site.go`).
- Daemon-side wiring of the Engine into `cmd/gophertrunk run`.

## Phase 7a — Voice infrastructure (recorder + plugin model)

Landed in this phase:

- `internal/voice/vocoder.go` — `Vocoder` interface (Name, FrameSize,
  Decode, Reset, Close) + thread-safe `Registry` keyed by name plus
  a `VocoderFactory` so each call gets a fresh decoder instance and
  internal state can't bleed between calls. `NullVocoder` is
  registered by default and emits 20 ms of silence per frame, giving
  the daemon a safe always-available decoder.
- `internal/voice/wav.go` — 16-bit PCM mono WAV writer over an
  `io.WriteSeeker` with a `NewWavFile` convenience for on-disk
  recording. RIFF / data length fields are patched on `Close` so a
  daemon crash leaves a still-readable file.
- `internal/voice/recorder.go` — Per-call recorder that subscribes
  to `events.KindCallStart` / `KindCallEnd` from the trunking
  engine, opens a WAV (and optional raw-frame sidecar) under
  `<OutDir>/<system>/<talkgroup>/<UTC>_src<src>.{wav,raw}`, exposes
  `WritePCM(serial, samples)` and `WriteRawFrame(serial, frame)`
  for the future demod pipeline to push into, and closes everything
  cleanly on `Close()`.
- `docs/vocoders.md` — Vocoder licensing realities (IMBE patents
  expired, AMBE+2 still encumbered), the plugin-model rationale, and
  build-tag-gated paths for `mbelib` and DVSI hardware backends.

Tests cover NullVocoder silence output + frame size, registry
listing + factory lookup, default registry contents, WAV header
shape + little-endian sample serialisation + length patching +
double-Close idempotency + zero-rate rejection, recorder per-call
WAV emission with the documented filename layout, raw-frame sidecar
appending, dropped writes when no session exists, clean Close
draining open sessions, and filename sanitisation.

Bugs caught and fixed during testing:
- Recorder originally nil-ed `r.sub` after closing it, racing with
  the still-running Run goroutine's read of `r.sub.C`. Wrapped the
  Close path in `sync.Once` and rely on `Subscription.Close`'s own
  idempotency, removing the field mutation entirely.
- Test inspected `r.sessions` without holding the lock; added
  `SessionCount()` and `HasSession()` accessors that take the lock
  internally, race-free for tests.

Deferred:
- Pure-Go IMBE decoder (`internal/voice/imbe`) for P25 Phase 1
  voice frames. Core patents have expired; this lands alongside
  actual P25 LDU1 / LDU2 decoding (Phase 7b in the plan).
- `mbelib` CGO wrapper behind `-tags mbelib` for AMBE+2 (Phase 7c).
- DVSI USB-3000 / AMBE-3003 hardware backend.
- Demod pipeline that produces the PCM samples / raw frames the
  recorder consumes (gated on the Phase 6 wiring deferrals).

…subsequent phases follow the plan in
`/root/.claude/plans/using-the-readme-md-as-sleepy-fairy.md`.
