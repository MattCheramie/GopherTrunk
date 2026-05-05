# GopherTrunk 📻🐹

A headless, low-latency digital-trunking scanner engine in Go.

GopherTrunk manages a pool of RTL-SDR dongles, runs a custom Go DSP
pipeline, decodes the control channels of P25, DMR, and NXDN trunked
radio systems, follows voice grants by talkgroup priority, and streams
metadata + audio to any frontend over gRPC, HTTP/SSE, or WebSocket.

## Features

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
| Demod pipeline    | `internal/voice/composer` subscribes to `CallStart` events, opens the bound Voice device's IQ stream, runs an LPF → decimate → FM demod → decimate → int16 PCM chain into the recorder, and pings `Engine.Touch` every second so the silent-call watchdog leaves the call alone |
| Voice recording   | `Vocoder` plugin interface + `NullVocoder` baseline, 16-bit PCM mono WAV writer with patched-length trailers, per-call recorder writing `<system>/<tg>/<UTC>_src<id>.wav` plus an optional raw-frame sidecar so users can BYO decoder |
| API               | `proto/*.proto` schemas under repo root; HTTP REST (`/api/v1/{health,version,systems,talkgroups,calls/active,calls/history}`); Server-Sent Events stream (`/api/v1/events`); WebSocket bridge (`/api/v1/events/ws`); gRPC `SystemService` + `TalkgroupService` + `AudioService` over the same in-process state |
| Persistence       | Pure-Go SQLite (`modernc.org/sqlite`) call log subscribing to `CallStart` / `CallEnd` events; newest-first history queries with system / group / time filters; retention sweeper that ages out DB rows and recorded `.wav` / `.raw` files past configurable cutoffs |
| Observability     | Prometheus collector (events / calls / CC-locked / IQ-underrun / USB-reconnect / decode-error / SDR-attached / build-info series) exposed at `/metrics`; multi-stage `Dockerfile`; `docker-compose.yml` with RTL-SDR USB pass-through, healthcheck, and Prometheus scrape labels |
| Daemon            | `cmd/gophertrunk run` composes everything above into a single supervised process with signal-driven shutdown; every component is opt-in via `config.yaml` |
| Testing           | Per-package unit tests under `make test`; `make integration` boots the wired daemon end-to-end (no SDR), publishes a synthetic call on the bus, and asserts the engine + recorder + call log + metrics + API agree — runs on every CI build |

## Status & known gaps

End-to-end audio works today for **analog FM voice channels**: the
control channel locks, the engine allocates a Voice device on a
grant, the composer pulls IQ → PCM → WAV, and the call is logged to
SQLite. The honest gaps:

- **Live P25 control-channel decoding** still needs the
  TIA-102.BAAA-A trellis tables and the TSBK block interleaver
  before the existing TSBK parser receives real data. BCH(63,16,11)
  for the NID is also stubbed.
- **DMR Tier II** is mostly a configuration variation on the Tier
  III scaffolding that's already in place; both share the burst,
  slot-type, and BPTC pieces.
- **DMR slot-type** still wants the Hamming(20,8) over the 20-bit
  field; **NXDN** wants the SACCH FEC + sub-frame interleaver.
- **Digital voice** (P25 Phase 1 IMBE; AMBE+2 for P25 Phase 2 / DMR
  / NXDN) is gated on the vocoders. The `Vocoder` plugin interface
  + raw-frame sidecar are in place; pure-Go IMBE is in progress
  ([patents have expired](docs/vocoders.md)) and AMBE+2 stays
  behind the `mbelib` build tag.
- **Higher-fidelity audio**: the FM chain currently does naive
  decimation rather than proper polyphase resampling, and skips
  de-emphasis + post-demod LPF + AGC. Quality is good enough to
  verify wiring; real DSP polish is follow-up work.

The Go interfaces and event payloads carry digital protocols already,
so dropping in a band-plan resolver and IMBE will light up the
remaining paths without further changes elsewhere.

## Tech stack

- **Language:** Go 1.24+
- **Hardware:** `libusb-1.0` + `librtlsdr` via CGO (custom thin binding)
- **DSP:** `gonum/dsp/fourier` for FFT, custom polyphase channelizer,
  filters, and demodulators
- **Storage:** `modernc.org/sqlite` (pure Go)
- **API:** gRPC + Protobuf, HTTP/SSE, WebSocket
- **Logging:** `log/slog` (stdlib)
- **Metrics:** `prometheus/client_golang`

## Quick start

### Prerequisites

```sh
sudo apt-get install librtlsdr-dev libusb-1.0-0-dev
```

See [`docs/hardware.md`](docs/hardware.md) for `udev` rules and DVB
blacklisting on Linux.

### Build, test, run

```sh
make build         # produces ./bin/gophertrunk
make test          # go test -race ./...
make integration   # boots the wired daemon end-to-end (no SDR needed)

./bin/gophertrunk version
./bin/gophertrunk sdr list                # enumerates attached dongles
./bin/gophertrunk run -config config.yaml
```

A starter [`config.example.yaml`](config.example.yaml) is in the
repo root — copy it, set the `serial` of your dongle from
`gophertrunk sdr list`, point `talkgroup_file` at a
Trunk-Recorder-format CSV, and you're going.

### Docker

```sh
docker compose up -d
curl -s http://localhost:8080/api/v1/health
curl -s http://localhost:8080/metrics | grep gophertrunk_build_info
```

[`docs/hardening.md`](docs/hardening.md) has the full operator
playbook — Prometheus catalogue, USB pass-through recipe, smoke
tests.

## Repository layout

```
cmd/gophertrunk/        daemon entrypoint + sdr list CLI
internal/sdr/           Driver interface, pool, CGO librtlsdr, mock
internal/dsp/           Channelizer, filters, demods, sync, FFT
internal/radio/         framing/ + p25/phase1/ + dmr/ + nxdn/
internal/trunking/      System, talkgroup DB, priority, engine, CC hunter
internal/voice/         Recorder, vocoder plugin, demod composer
internal/storage/       SQLite call log + retention sweeper
internal/api/           HTTP REST + SSE + WebSocket + gRPC
internal/metrics/       Prometheus collector
internal/events/        In-process pub/sub bus
internal/config/        YAML loader
proto/                  *.proto schemas (events, system, talkgroup, audio)
docs/                   architecture · hardware · vocoders · hardening
```

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — layered overview,
  concurrency model, driver registry, build tags
- [`docs/hardware.md`](docs/hardware.md) — udev rules, DVB blacklist,
  IQ capture for replay
- [`docs/vocoders.md`](docs/vocoders.md) — IMBE / AMBE+2 licensing
  realities and the plugin model
- [`docs/hardening.md`](docs/hardening.md) — Prometheus catalogue,
  Docker / compose USB pass-through, smoke-test checklist

## License

See [`LICENSE`](LICENSE).
