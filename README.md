![GopherTrunk](GopherTrunkLogo.png)

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

## Roadmap

These are the next four feature areas on the table, mapped to where
they plug into the existing architecture. Order isn't fixed; each is
landable independently.

### 1. Tone-out alerting ✅ landed

Mirrors the "Tone-Out" feature on hardware scanners: fire an alarm
only when a specific paging tone (or tone pair) is heard on a
configured channel. Most US fire / EMS departments use **Two-Tone
Sequential** (Motorola Quick Call II — typically a 0.5–1.5 s
"A-tone" in the 250–1100 Hz range followed by a ~3 s "B-tone" in
roughly the same range).

What's wired:

- `internal/voice/toneout` runs Goertzel filters against each
  Voice device's PCM stream (one filter per unique target
  frequency across all profiles, so a profile with N tones costs
  N Goertzel passes per block, not N × profiles).
- A per-device state machine matches sequential tones against the
  configured profile list, honouring per-tone min/max duration,
  per-profile inter-tone gap, and a refractory cooldown that
  suppresses re-fires.
- Detections publish `events.KindToneAlert` with the profile name,
  alpha tag, device serial, matched timestamp, and the actual
  matched frequencies. The bus payload flows through the SSE /
  WebSocket stream automatically.
- The composer's PCM output is fanned into both the recorder and
  the detector via a `fanoutSink` in the daemon, so existing
  recordings keep working.
- YAML config: `tone_out.profiles` with `name`, `alpha_tag`,
  `cooldown`, optional `system` / `group_id` filters, and a
  per-tone `frequency_hz` + `min_duration` / `max_duration`. See
  [`config.example.yaml`](config.example.yaml).

Tests cover Goertzel on/off-target magnitude separation,
single-tone matching, two-tone sequential matching with realistic
QC-II timing, wrong-frequency rejection, too-short-tone rejection,
cooldown suppression, and per-device state isolation.

Future work: live-frequency refinement (track the actually-matched
bin rather than the configured one), DTMF / concurrent multi-tone
profiles, and a Prometheus counter for fired alerts.

### 2. TrunkTracker-style multi-system grant following 🟡 partial

Marketed by Uniden as TrunkTracker, this is exactly what the engine
already does for the protocols we currently decode: parse a control
channel, see a Group Voice Channel Grant, retune a Voice SDR, follow
the call. The architecture covers it; what's needed is **decoders
for additional trunking systems** so the existing
`events.KindGrant` → engine pipeline carries them.

What's wired:

- **Motorola Type II / SmartZone.** `internal/radio/motorola/` —
  OSW (Outbound Status Word) parser, opcode constants
  (GroupVoiceChannelGrant, GroupVoiceChannelGrantUpdate,
  AdjacentSiteStatus, SystemIDExtended, Idle, Encryption,
  Emergency), an `LCN → Hz` band-plan resolver with linear and
  table strategies, and a control-channel state machine that
  ingests OSWs and publishes `cc.locked` (on
  SystemIDExtended) and `grant` (on the voice-grant opcodes) on
  the events bus, with the `trunking.Grant.Protocol` tag set to
  `"motorola"`. Same shape as the P25 / DMR / NXDN packages.
  Tests cover OSW round-trip, opcode + LCN extraction, the
  per-grant-type accessors, both band-plan strategies, sync
  detection with tolerance, and the control-channel emission path.
- The Motorola **MSK demodulator** + **BCH(64,16,11)** error
  corrector that turn IQ into the bits the OSW parser consumes
  are honest deferrals — the package's doc comment lists them
  explicitly so a contributor can pick them up. Same pattern as
  the P25 trellis-tables / TSBK-interleaver gap.
- **EDACS / GE-Marc.** `internal/radio/edacs/` — 40-bit Control
  Channel Word (CCW) parser, command enum
  (Idle, GroupVoiceGrant, ProVoiceGrant, IndividualCall,
  DataGrant, SystemID, AdjacentSite, Emergency, Affiliation,
  Encryption), per-command payload accessors that surface
  encrypted / emergency status flags, an `LCN → Hz` band-plan
  resolver mirroring the Motorola one, and a control-channel
  state machine that publishes `cc.locked` on SystemID and
  `grant` (with `trunking.Grant.Protocol = "edacs"`) on voice
  grants. Tests cover CCW round-trip in bytes + bits, command
  + LCN + status extraction, the per-command accessors
  (including the ProVoice flag), both band-plan strategies,
  sync detection with tolerance, and the full control-channel
  emission path. The 9600-baud GMSK demodulator + the
  interleaved Reed-Solomon-derived FEC are honest deferrals,
  spelled out in the package's doc comment.
- **LTR (Logic Trunked Radio).** `internal/radio/ltr/` —
  41-bit per-repeater Status word parser
  (`Sync + Area + Group flag + Channel + Home + GroupID + Free
  + FCS`), `Channel → Hz` band-plan resolver (linear and
  table), and a per-repeater state machine that publishes
  `cc.locked` on the first valid status and `grant` (with
  `trunking.Grant.Protocol = "ltr"`) when a status indicates
  an active call. Architecturally different from the
  central-CC trunked systems above: each repeater is its own
  conventional channel and carries its own status word at
  300 bps under the in-band voice. Optional area filter so
  one physical channel can host multiple LTR systems without
  cross-talk. Tests cover status round-trip, area filtering,
  active-call detection, no-republish-on-same-group, no-resolver
  fallback, and the cc.locked / cc.lost emissions. The 300-baud
  sub-audible status-word demodulator is the honest deferral.
- **MPT 1327.** `internal/radio/mpt1327/` — 64-bit address-codeword
  parser (38 information bits + 26 BCH parity, with BCH consumed
  by the upstream FEC), CodewordKind enum
  (Aloha / Ahoy / AhoyChan / GoToChannel / Ack / Disconnect /
  Data / Emergency), per-kind payload accessors that surface
  the GTC voice-grant channel + the AHYC system-broadcast
  identifier, channel-number → Hz band-plan resolver
  (linear and table), and a control-channel state machine that
  publishes `cc.locked` on the first ALH or AHYC frame and
  `grant` (with `trunking.Grant.Protocol = "mpt1327"`) on each
  GoToChannel codeword. The grant's `GroupID` packs
  `(prefix << 16) | ident` so prefixes that re-use idents stay
  disambiguated. Tests cover codeword round-trip in bytes + bits,
  Kind + payload extraction, both lock paths (ALH and AHYC),
  the GTC grant publication, no-resolver fallback, data-codeword
  filtering, and `MarkLost`. The 1200-baud FFSK demodulator and
  BCH(63,38) decoder are honest deferrals, listed in the package's
  doc comment.

Still ahead:

- **P25 Phase 2 (TDMA H-DQPSK superframes).** The H-DQPSK demod and
  most framing primitives are already in place; what's missing is
  the TDMA superframe sync and the superframe / voice slot
  state machine. Same `Grant` once the channel-number resolves.

Each addition is a contained `internal/radio/<system>/` package that
plugs into the engine via the existing event bus — no changes to
the engine, recorder, composer, or API surfaces needed.

### 3. Simulcast mitigation (the SDR-side equivalent of "True I/Q") 🟡 partial

Premium hardware scanners advertise a "True I/Q" front-end that
fights **simulcast distortion** — the audio garble you hear when
multiple cell towers transmit the identical signal on the same
frequency and the differing arrival delays at the receiver smear
the symbols (it's effectively self-multipath). GopherTrunk runs on
SDR hardware so it always has full I/Q access; the actual win
versus a hardware scanner is what you do with it once you have it.

What's wired:

- **`internal/dsp/equalizer`** ships two adaptive equalizers for
  the demod chain:
  - **LMS** (`lms.go`) — complex tapped-delay-line FIR with the
    standard Least-Mean-Squares update
    `w[n+1] = w[n] + μ · x[n] · conj(e[n])`. Trained with a
    reference symbol (training preamble) or in decision-directed
    mode with the slicer's hard decision. Centre-spike init makes
    a clean channel benign; on a 2-tap multipath QPSK channel the
    test sweep drives MSE down by > 4× over 4 000 symbols.
  - **CMA** (`cma.go`) — Constant Modulus Algorithm blind
    equalizer (Godard/CMA-2 cost) for PSK-family signals where no
    training preamble is available. Drives `|y|^2` toward a
    configurable `R^2`; documented phase-blindness caveat.
  - Both expose `Reset()`, `Taps()`, and a single-sample
    `Process` so a chain stage can drop them in between the
    channelizer and the symbol-time-recovery / demodulator stages.
- Tests cover LMS convergence on a 2-tap multipath QPSK channel
  (early vs late MSE), centre-spike reset, bad-config panics,
  CMA constellation opening (RMS deviation of `|y|^2` from `R^2`
  after settle), CMA reset, and delay-aligned passthrough on a
  clean channel.

Still ahead:

- **Wiring into the composer chain** so a per-call equaliser slot
  lights up automatically for protocols that benefit. This is the
  natural follow-up once a digital protocol with pilot symbols
  (P25 Phase 1 once the trellis tables land, Motorola Type II
  once the MSK demod ships) is end-to-end.
- **Multi-receiver IQ combining.** When two RTL-SDR dongles are
  connected to antennas at different positions (or the same
  antenna via a splitter), per-tower IQ streams can be coherently
  combined via maximal-ratio or selection combining to recover
  the cleaner stream. `sdr.Pool` already supports multiple
  devices; the composer's chain abstraction lets a diversity
  combiner stage slot in ahead of the demodulator.

Both items live under `internal/dsp/` and wire into the existing
demod chains without touching the trunking engine or higher
layers.

### 4. Expanding digital-mode coverage 🟡 partial

The hardware-scanner equivalent is firmware "digital upgrades" that
unlock DMR, NXDN, and ProVoice. GopherTrunk's `Vocoder` plugin
interface and raw-frame sidecar are already in place; the work
splits into vocoders (decoding voice frames) and protocols
(decoding signalling).

Vocoders:

- ✅ **`mbelib` CGO wrapper** — `internal/voice/mbelib` ships
  IMBE 4400 bps and AMBE+2 2400 bps decoders backed by the
  szechyjs [`mbelib`](https://github.com/szechyjs/mbelib) library,
  gated behind the `mbelib` build tag. With libmbe + headers
  installed, `make build TAGS=mbelib` registers `imbe` and
  `ambe2` factories on `voice.DefaultRegistry`. Default builds
  use a no-op stub and link nothing extra. CI exercises the
  stub path only — the wrapper is verified at build time when
  an operator opts in. Full operator instructions in
  [`docs/vocoders.md`](docs/vocoders.md).
- **IMBE pure-Go** — for default builds without any C
  dependency. Core US patents are expired; the algorithm is
  implementable. Bigger DSP undertaking; lands in a follow-up.
- **DVSI USB-3000 / AMBE-3003 hardware backend** — for operators
  with the chip. Same `Vocoder` plug-in shape as the mbelib
  wrapper; the daemon picks the factory by name from config.
- **ProVoice** (GE/Harris, EDACS family) is patent- and
  trade-secret-encumbered with limited public documentation.
  Realistic path: raw-frame export only, with decoding deferred
  to hardware vocoders or operator-supplied plugins via the
  same `Vocoder` registry.

Protocols:

- **dPMR (digital PMR446)** — public ETSI TS 102 658 spec, similar
  framing to NXDN. A natural addition once NXDN's SACCH FEC lands.
- **TETRA** — full public spec (ETSI TS 100 392); large undertaking
  but well-defined.
- **D-STAR / Yaesu System Fusion** — amateur-radio digital modes
  with public specs. Lower priority.

The vocoder pieces are pure plug-ins. New protocols follow the
same `internal/radio/<protocol>/` shape used for P25 / DMR / NXDN
and publish events via the existing bus.

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
