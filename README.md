# GopherTrunk 📻🐹

A headless, low-latency digital-trunking scanner engine in Go.

GopherTrunk manages a pool of RTL-SDR dongles, runs a custom Go DSP pipeline,
and decodes the control channels of P25, DMR, and NXDN trunked radio systems
— with the engine pieces that follow voice grants, hold talkgroups by
priority, and stream metadata + audio to a frontend layered on top.

> **Status: under active development.** Phases 0 – 9 of the build plan
> have landed (foundation, SDR hardware, DSP core, P25 Phase 1 control
> channel, system-ID & CC hunter, DMR Tier III CSBK, NXDN frame
> structure, the trunking engine, the voice-recording infrastructure,
> the API — protobuf schemas, an HTTP REST surface, an SSE event
> stream, a WebSocket bridge, and a gRPC server — and persistence: a
> SQLite call log subscribing to engine events, a `/api/v1/calls/history`
> endpoint, and a retention sweeper that ages out both DB rows and the
> recorded WAV / raw files). The full phased roadmap lives in
> [`docs/phases.md`](docs/phases.md); the architectural overview is in
> [`docs/architecture.md`](docs/architecture.md); the vocoder-licensing
> situation is in [`docs/vocoders.md`](docs/vocoders.md).

## What's built so far

| Area              | Component                                                  |
| ----------------- | ---------------------------------------------------------- |
| Hardware          | CGO `librtlsdr` binding, multi-device pool, role assignment, DC blocker, IQ-imbalance correction, file-backed IQ replay (mock) |
| DSP               | Polyphase channelizer, FIR + Kaiser LPF designer + RRC, CIC, halfband, AGC, rational resampler, FM / C4FM / H-DQPSK demods, Mueller-Müller clock recovery, frame-sync correlator |
| FEC primitives    | CRC-CCITT/FALSE, Hamming(15,11,3), Hamming(13,9,3), extended Golay(24,12,8), BPTC(196,96), 4-state ½-rate Viterbi |
| P25 Phase 1       | 48-bit FSW + sync detector, NID parser (NAC + DUID), TSBK with CRC trailer, payload parsers for GroupVoiceChannelGrant / Update / NetworkStatus / RFSSStatus, control-channel state machine |
| DMR (Tier III)    | All 9 ETSI sync patterns, burst layout (132 dibits), Color Code + Data Type, CSBK with CRC, payload parsers for TalkGroup/Private Voice grants + Aloha + AdjacentSiteStatus + SystemInfoBroadcast, control-channel state machine |
| NXDN              | 192-dibit frame layout (4800 BFSK / 9600 4-FSK), LICH parse with parity + 16-bit doubled-wire decoder, FSW correlator, CAC parser with CRC, RCCH opcode enum + payload parsers, control-channel state machine |
| Orchestration     | In-process pub/sub event bus, `System` model, JSON-on-disk last-known-CC cache, control-channel `Hunter` that retunes the SDR and parks on the first responsive frequency |
| Trunking engine   | Cross-protocol `Grant` payload, Trunk-Recorder-format talkgroup DB (CSV + JSON), priority + preemption (emergency overrides, strict-higher), voice-device pool allocator, central state machine emitting `CallStart` / `CallEnd` events with a watchdog for silent calls |
| Voice infrastructure | `Vocoder` plugin interface + `NullVocoder` baseline, 16-bit PCM mono WAV writer with patched-length trailers, per-call recorder that subscribes to `CallStart` / `CallEnd` and writes `<system>/<tg>/<UTC>_src<id>.wav` plus an optional raw-frame sidecar so users can BYO decoder |
| API               | `proto/*.proto` schemas under repo root; HTTP REST (`/api/v1/{health,version,systems,talkgroups,calls/active,calls/history}`); Server-Sent Events stream (`/api/v1/events`); WebSocket bridge (`/api/v1/events/ws`); gRPC `SystemService` + `TalkgroupService` + (stub) `AudioService` over the same in-process state |
| Persistence       | Pure-Go SQLite (`modernc.org/sqlite`) call log that subscribes to `CallStart` / `CallEnd` events; newest-first history queries with system / group / time filters; retention sweeper that ages out DB rows and recorded `.wav` / `.raw` files past configurable cutoffs (config / talkgroup CSVs are preserved) |
| Daemon            | `cmd/gophertrunk` with `version`, `sdr list`, and `run` subcommands; YAML config; `log/slog`; signal-driven shutdown |

## What's intentionally deferred

The build plan calls these out by phase; the most visible items still ahead:

- Wiring the demod pipeline (channelizer → demod → protocol decoder) and
  the trunking engine into `cmd/gophertrunk run` so the daemon does
  end-to-end CC hunt → grant follow → audio.
- A composer that subscribes to `CallStart` events, spins up a per-call
  demod chain on the bound Voice device, calls `Engine.Touch` on each
  voice frame, and `Engine.EndCall` on a release announcement.
- BCH(63,16,11) for full P25 NID validation; P25-specific trellis tables and
  the TSBK block interleaver (so the P25 Phase 1 control channel can decode
  live captures end-to-end).
- Hamming(20,8) over the DMR slot-type field; SACCH FEC + sub-frame
  interleaver for NXDN.
- Voice frame _decoding_ — the pure-Go IMBE decoder for P25 Phase 1 is
  in progress (patents have expired); AMBE+2 (P25 Phase 2 / DMR / NXDN)
  stays gated behind a `mbelib` build tag for patent-licensing clarity
  ([`docs/vocoders.md`](docs/vocoders.md)). The recorder, plugin
  interface, and raw-frame sidecar that the decoders will plug into
  have already landed.
- Hardening (metrics, reconnect, Docker — Phase 10) is the next major
  area. The persistence layer (Phase 9 — SQLite call log + retention)
  and API (Phase 8) have both landed; the gating piece is the
  demod-pipeline composer that produces the live PCM the
  `AudioService` will stream and the engine events the call log
  expects from real captures.

## ✨ Goals

- **Native concurrency.** Goroutines + typed channels carry IQ from the
  RTL-SDR async-read callback all the way through the DSP and protocol
  pipelines.
- **Multi-SDR pool.** Auto-discovery and Control / Voice role assignment
  across multiple dongles, with serial-number hints honored.
- **Protocols.** P25 Phase 1 & 2 (C4FM / H-DQPSK), DMR Tier II & III, NXDN
  4800 / 9600 baud.
- **Headless architecture.** Daemon with gRPC + WebSocket APIs for any
  frontend; the engine itself stays API-agnostic via an in-process event
  bus.
- **Priority tracking.** Talkgroup preemption based on user-defined
  priority levels; multi-site neighbor following.

## 🛠 Tech stack

- **Language:** Go 1.24+
- **Hardware:** `libusb-1.0` + `librtlsdr` via CGO (custom thin binding)
- **DSP:** `gonum/dsp/fourier` for FFT, custom polyphase channelizer,
  filters, and demodulators
- **Logging:** `log/slog` (stdlib)
- **API (planned):** gRPC + Protobuf for metadata; server-streaming RPC
  for audio; WebSocket bridge for browser frontends

## Quick start

### Prerequisites

```sh
sudo apt-get install librtlsdr-dev libusb-1.0-0-dev
```

See [`docs/hardware.md`](docs/hardware.md) for `udev` rules and DVB
blacklisting on Linux.

### Build and test

```sh
make build     # produces ./bin/gophertrunk
make test      # go test -race ./...
make vet
```

### Smoke test

```sh
./bin/gophertrunk version
./bin/gophertrunk sdr list      # enumerates attached RTL-SDR dongles
./bin/gophertrunk run --config config.yaml
```

A minimal `config.yaml`:

```yaml
log:
  level: info
  format: text
sdr:
  sample_rate: 2400000
  devices:
    - serial: "00000001"
      role: control
      ppm: -2
trunking:
  systems:
    - name: ExampleP25
      protocol: p25
      control_channels: [851000000, 852000000]
```

## Repository layout

```
cmd/gophertrunk/        daemon entrypoint + sdr list CLI
internal/sdr/           Driver interface, pool, CGO librtlsdr, mock
internal/dsp/           Channelizer, filters, demods, sync, FFT
internal/radio/         framing/ + p25/phase1/ + dmr/ + nxdn/
internal/trunking/      System, last-known-CC cache, CC Hunter
internal/events/        In-process pub/sub bus
internal/config/        YAML loader
docs/                   architecture.md · phases.md · hardware.md
```

## Contributing

The project is being built phase by phase per
[`docs/phases.md`](docs/phases.md). Each phase is independently buildable
and testable. PRs that complete deferred items in earlier phases (the
"deferred" lists in `docs/phases.md`) are very welcome — the most useful
near-term targets are the BCH + P25 trellis tables (so the P25 control
channel decodes live IQ end-to-end) and the demod-pipeline composer that
glues SDR → channelizer → demod → control-channel state machine.

## License

See [`LICENSE`](LICENSE).
