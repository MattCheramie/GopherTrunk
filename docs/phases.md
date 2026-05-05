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
| 5     | NXDN trunking                          | upcoming    |
| 6     | Trunking engine (grant follower)       | upcoming    |
| 7a    | Voice passthrough (FM, raw frames)     | upcoming    |
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

…subsequent phases follow the plan in
`/root/.claude/plans/using-the-readme-md-as-sleepy-fairy.md`.
