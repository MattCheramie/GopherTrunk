# Build Phases

GopherTrunk is being built incrementally. Each phase is independently
buildable and testable; the project stays useful even if work pauses partway.

| Phase | Title                                  | Status      |
| ----- | -------------------------------------- | ----------- |
| 0     | Foundation                             | done        |
| 1     | SDR hardware layer (CGO librtlsdr)     | done        |
| 2     | DSP core (channelizer, demods)         | upcoming    |
| 3     | P25 trunking (Phase 1 then Phase 2)    | upcoming    |
| 3.5   | System ID & control-channel hunting    | upcoming    |
| 4     | DMR trunking (Tier II + Tier III)      | upcoming    |
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

- `internal/dsp/filter` (FIR, CIC, halfband).
- `internal/dsp/channelizer` polyphase channelizer over an FFT abstraction
  (`internal/dsp/fft` wrapping `gonum.org/v1/gonum/dsp/fourier`).
- `internal/dsp/demod` (FM, C4FM, H-DQPSK).
- `internal/dsp/sync` (Mueller-Müller, Gardner, frame correlator).
- Verification: golden vectors generated against GNU Radio reference.

…subsequent phases follow the plan in
`/root/.claude/plans/using-the-readme-md-as-sleepy-fairy.md`.
