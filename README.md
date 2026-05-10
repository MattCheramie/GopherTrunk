# GopherTrunk 📻🐹

A headless, low-latency digital-trunking scanner engine in Go.

GopherTrunk manages a pool of RTL-SDR dongles, runs a custom Go DSP
pipeline, decodes the signalling layers of every major trunked-radio
family (P25 Phase 1 / Phase 2, DMR Tier II / III, NXDN, Motorola Type II /
SmartZone, EDACS / GE-Marc, LTR, MPT 1327, dPMR Mode 3, TETRA TMO)
plus the D-STAR + Yaesu System Fusion amateur modes, follows voice
grants by talkgroup priority, and streams metadata + audio to any
frontend over gRPC, HTTP/SSE, or WebSocket.

## Features

| Area              | Component                                                  |
| ----------------- | ---------------------------------------------------------- |
| Hardware          | CGO `librtlsdr` binding, multi-device pool, role assignment, per-device gain (`auto` / tenths-of-dB) + PPM + bias-tee (5 V LNA power, e.g. NESDR Smart v5) applied at open time, DC blocker, IQ-imbalance correction, file-backed IQ replay (mock) |
| DSP               | Polyphase channelizer, FIR + Kaiser LPF designer + RRC, CIC, halfband, IQ + audio AGC (attack/release envelope follower for voice), L/M polyphase resampler (complex IQ + real audio), FM / C4FM / H-DQPSK demods, single-pole IIR de-emphasis (75/50µs), Mueller-Müller clock recovery, frame-sync correlator |
| FEC primitives    | CRC-CCITT/FALSE + CRC-CCITT/XMODEM (callable init), CRC-6 (NXDN SACCH), Hamming(15,11,3), Hamming(13,9,3), Hamming(20,8) (DMR slot-type, t=3), extended Golay(24,12,8) + non-extended Golay(23,12,7) (P25 IMBE), BCH(63,16,11), BPTC(196,96), Reed-Solomon(12,9,4) over GF(2^8) with DMR Voice LC Header / Terminator / Embedded LC seeds, 4-state ½-rate Viterbi, 16-state K=5 ½-rate Viterbi (shared by NXDN SACCH + planned YSF FICH) with depuncture-marker support |
| P25 Phase 1       | 48-bit FSW + sync detector, NID parser (NAC + DUID) with BCH(63,16,11) error correction + even-parity check, full TSBK channel decode (TIA-102.BAAA Annex A 4-state ½-rate trellis + 98-dibit block deinterleaver) → CRC trailer validation, payload parsers for GroupVoiceChannelGrant / Update / NetworkStatus / RFSSStatus, IdentifierUpdate band-plan resolver, control-channel state machine emitting `protocol = "p25"` grants and `decode.error` events with `nid-bch` / `tsbk-trellis` / `tsbk-crc` / `no-bandplan` stages |
| P25 Phase 2       | Outbound + inbound 20-dibit sync, 360 ms / 12-subframe superframe + SlotType enum, MAC PDU parser + opcode enum, GroupVoiceChannelGrant accessor, control-channel state machine emitting `protocol = "p25-phase2"` grants |
| DMR (Tier III)    | All 9 ETSI sync patterns, burst layout (132 dibits), Color Code + Data Type via (20,8,7) shortened-Hamming slot-type FEC (corrects up to 3 bit errors per slot type), CSBK with CRC, payload parsers for TalkGroup/Private Voice grants (LCN + timeslot) + Aloha + AdjacentSiteStatus + SystemInfoBroadcast, LCN → Hz band-plan resolver (linear + table forms), control-channel state machine emitting `protocol = "dmr-tier3"` grants and `decode.error` events with `no-bandplan` stage |
| DMR (Tier II)     | Shares the burst / slot-type / BPTC(196,96) layers with Tier III; adds a 72-bit Full Link Control parser (FLCO enum: GroupVoiceChannelUser / UnitToUnitVoice / TalkerAlias / GPS / Terminator) with RS(12,9,4) parity verification (Voice LC Header seed) and a per-repeater conventional-mode state machine that decodes Voice LC Header bursts and emits `protocol = "dmr-tier2"` grants on the bus (deduped per call, cleared on Terminator-with-LC) and `decode.error` events with `voiceheader-bptc` / `voiceheader-rs` stages |
| NXDN              | 192-dibit frame layout (4800 BFSK / 9600 4-FSK), LICH parse with parity + 16-bit doubled-wire decoder, FSW correlator, full SACCH channel decode (K=5 ½-rate convolutional Viterbi + 60-position sub-frame deinterleaver + 12-bit puncture undo + CRC-6 trailer), CAC parser with CRC, RCCH opcode enum + payload parsers, control-channel state machine |
| Motorola Type II  | OSW parser, opcode constants, LCN → Hz band-plan resolver (linear + table), control-channel state machine emitting `protocol = "motorola"` grants |
| EDACS / GE-Marc   | 40-bit CCW parser, command enum (Idle / GroupVoiceGrant / ProVoiceGrant / IndividualCall / DataGrant / SystemID / AdjacentSite / Emergency / Affiliation / Encryption), per-command accessors with encrypted / emergency flags, LCN → Hz resolver, control-channel state machine emitting `protocol = "edacs"` grants |
| LTR               | 41-bit per-repeater Status word parser, Channel → Hz resolver, optional area filter, per-repeater state machine emitting `protocol = "ltr"` grants when a status indicates an active call |
| MPT 1327          | 64-bit address-codeword parser (38 info + 26 BCH parity consumed upstream), CodewordKind enum (ALH / AHY / AHYC / GTC / ACK / Disconnect / Data / Emergency), accessors for GTC voice grants + AHYC system broadcast, channel resolver, control-channel state machine emitting `protocol = "mpt1327"` grants |
| dPMR (Mode 3)     | FS1 / FS2 / FS3 24-dibit sync, 80-bit CSBK parser, MessageType enum (RegistrationRequest / Response, VoiceServiceAllocation, IndividualVoiceAllocation, DataServiceAllocation, ServiceRequest, StandingServiceStatus, Release, Idle), AsVoiceGrant + AsSiteBroadcast accessors, PMR446 default band-plan, control-channel state machine emitting `protocol = "dpmr"` grants |
| TETRA (TMO)       | Normal + extended training-sequence sync, generic Layer-3 PDU parser (4-bit Discriminator + type + payload), CMCE D-CONNECT / D-TX-GRANTED / D-RELEASE accessors, MLE-SYSINFO accessor (MCC / MNC / Location Area), TETRA-380 / 410 / 800 carrier resolver, control-channel state machine emitting `protocol = "tetra"` grants |
| D-STAR            | Frame Sync + Slow Data sync, 41-byte PCH header parser (FLAG1 + RPT2 / RPT1 / UR / MY1 / MY2 + CRC-CCITT), IsGroupCall / IsEmergency / IsData accessors, repeater state machine emitting `protocol = "dstar"` grants on group transmissions |
| YSF (Yaesu System Fusion) | 4800-baud C4FM, 480-dibit / 100 ms frame layout (FSW / FICH / DCH offsets), 40-bit FSW correlator with mismatch tolerance, 32-bit Frame Information Channel parser (FrameType / CallType / Frame Number / Frame Total / DataType / VoIP / Squelch fields) with CRC-16 trailer, per-frequency state machine emitting `cc.locked` on sync detect (Trellis FEC + grant emission is a follow-up) |
| Orchestration     | In-process pub/sub event bus with typed payloads (Grant / CallStart / CallEnd / DecodeError / ToneAlert / etc.) and a typed `events.Stage` enum so protocol packages can't accidentally publish a stage label that drifts from the Prometheus dashboards, `System` model, JSON-on-disk last-known-CC cache, control-channel `Hunter` that retunes the SDR and parks on the first responsive frequency |
| Trunking engine   | Cross-protocol `Grant` payload, Trunk-Recorder-format talkgroup DB (CSV + JSON), priority + preemption (emergency overrides, strict-higher), voice-device pool allocator, central state machine emitting `CallStart` / `CallEnd` events with a watchdog for silent calls |
| Demod pipeline    | `internal/voice/composer` subscribes to `CallStart` events, opens the bound Voice device's IQ stream, runs an LPF → decimate → optional CMA equalizer → FM demod → optional 75/50µs de-emphasis → optional Kaiser audio LPF → optional audio AGC → optional polyphase L/M resample (or naive decimate fallback) → int16 PCM chain into the recorder, and pings `Engine.Touch` every second so the silent-call watchdog leaves the call alone |
| Simulcast / "True I/Q" | `internal/dsp/equalizer` (LMS + CMA blind equalizers) for inter-symbol-interference / multipath mitigation, plus `internal/dsp/diversity` (Selection + maximal-ratio combiners over a shared `Combiner` interface) for multi-receiver IQ combining |
| Tone-out alerting | `internal/voice/toneout` runs Goertzel filters against each Voice device's PCM stream, matches QC-II two-tone-sequential sequences against operator-configured profiles with per-tone duration + cooldown, and publishes `tone.alert` events that fan out through SSE / WebSocket / gRPC |
| Voice recording   | `Vocoder` plugin interface + `NullVocoder` baseline, 16-bit PCM mono WAV writer with patched-length trailers, per-call recorder writing `<system>/<tg>/<UTC>_src<id>.wav` plus an optional raw-frame sidecar so users can BYO decoder; EDACS ProVoice grants always force a `.raw` sidecar (the vocoder is patent + trade-secret encumbered) so researchers can decode out-of-band |
| API               | `proto/*.proto` schemas under repo root; HTTP REST (`/api/v1/{health,version,systems,talkgroups,calls/active,calls/history}`); operator mutations gated behind `api.allow_mutations` (`GET /api/v1/mutations` capability probe; `POST /api/v1/calls/{serial}/end`; `PATCH /api/v1/talkgroups/{id}`; `POST /api/v1/retention/sweep`; `POST /api/v1/devices/{serial}/tone-reset`); Server-Sent Events stream (`/api/v1/events`); WebSocket bridge (`/api/v1/events/ws`); gRPC `SystemService` + `TalkgroupService` + `AudioService` over the same in-process state |
| Persistence       | Pure-Go SQLite (`modernc.org/sqlite`) call log subscribing to `CallStart` / `CallEnd` events; newest-first history queries with system / group / time filters; retention sweeper that ages out DB rows and recorded `.wav` / `.raw` files past configurable cutoffs |
| Observability     | Prometheus collector (events / calls / CC-locked / IQ-underrun / USB-reconnect / decode-error / SDR-attached / build-info series) exposed at `/metrics`; multi-stage `Dockerfile`; `docker-compose.yml` with RTL-SDR USB pass-through, healthcheck, and Prometheus scrape labels |
| Daemon            | `cmd/gophertrunk run` composes everything above into a single supervised process with signal-driven shutdown; every component is opt-in via `config.yaml` |
| Testing           | Per-package unit tests under `make test`; `make integration` boots the wired daemon end-to-end (no SDR), publishes a synthetic call on the bus, and asserts the engine + recorder + call log + metrics + API agree — runs on every CI build |

## Status & known gaps

End-to-end audio works today for **analog FM voice channels**: the
control channel locks, the engine allocates a Voice device on a
grant, the composer pulls IQ → PCM → WAV, and the call is logged to
SQLite. The honest gaps:

- **Digital voice** (P25 Phase 1 IMBE; AMBE+2 for P25 Phase 2 / DMR
  / NXDN) is gated on the vocoders. The `Vocoder` plugin interface
  + raw-frame sidecar are in place; pure-Go IMBE now produces
  intelligible audio end-to-end ([patents have expired](docs/vocoders.md)),
  with operator-tunable AGC + §6.4 overlap-add windowing + §6.2
  spectral enhancement + frame-repeat on bad-frame indicator
  shipped — absolute-level calibration against a known-good
  reference decoder is the only remaining polish item. Pure-Go
  AMBE+2 is scaffolded under `internal/voice/ambe2/` and registers
  the `ambe2` name on every default build; the skeleton emits
  silence pending follow-up PRs that plug in 49-bit parameter
  unpacking and wire the shared `internal/voice/mbe/` synthesis
  pipeline. The AMBE+2 algorithm carries active patents in some
  jurisdictions; re-implementing it in pure Go does not change
  that posture — see [docs/vocoders.md](docs/vocoders.md).
- **Higher-fidelity audio**: the FM chain now has opt-in 75/50µs
  de-emphasis, a Kaiser-windowed audio LPF, audio AGC, and a
  polyphase L/M audio resampler — the full polish stack ships.
  Defaults stay analog-FM-friendly so digital protocols and
  passthrough callers see no behavior change.

The Go interfaces and event payloads carry digital protocols already,
so the remaining paths light up once IMBE drops in.

## Roadmap

What's still on the table. Order isn't fixed; each item is contained
to its own package and lands independently.

- **Pure-Go IMBE vocoder.** A native-Go IMBE 4400 bps decoder for
  default builds without a CGO dependency. Core US patents are
  expired; the algorithm is implementable from TIA-102.BABA. The
  `mbelib` build-tagged path already covers IMBE for operators with
  libmbe installed. The synthesis half of the pipeline (cross-frame
  log-amplitude prediction, voiced harmonic generator, unvoiced
  FFT excitation + overlap-add window, §6.2 spectral-amplitude
  enhancement, per-frame AGC) lives in `internal/voice/mbe/` so
  the in-progress AMBE+2 decoder can consume the same primitives
  via `mbe.SynthState` + `mbe.Params`. Status: skeleton + Vocoder
  interface registered as `imbe` (the canonical name; the pure-Go
  decoder is the sole IMBE backend in default builds); per-vector
  channel-coding FEC inverse (Golay(23,12) for u_0..u_3 +
  Hamming(15,11) for u_4..u_6 + no-FEC u_7 passthrough) is in
  (`internal/voice/imbe/channel.go`); the TIA-102.BABA §7.4
  u_0-keyed LCG pseudo-random scrambler is in
  (`internal/voice/imbe/scrambler.go`); full §5.3 / §5.4 / Annex E
  parameter unpack (b_0 → ω₀ + L + K + Vl voicing + Gm PRBA
  gains + Cik spectral coefficients + Tl log-amplitude residuals
  via two inverse DCTs) is in
  (`internal/voice/imbe/params.go` / `tables.go`); §6.1 cross-frame
  log2(Ml) prediction (eqs. 75-77 — γ = 0.65 interpolation of
  prev-frame harmonics at l · ω₀_curr/ω₀_prev positions, DC-bias
  removal, Tl residual addition) is in on a `SynthState` that the
  excitation step extends (`internal/voice/mbe/synth.go`); §6.2
  amplitude prep (log2(Ml) → linear Ml = 2^log2(Ml), the
  R_M0 = Σ Ml² and R_M1 = Σ Ml² · cos(ω₀·l) spectral moments, and
  a voicing-fraction summary that the synthesis combiner consumes)
  is in (`internal/voice/mbe/amps.go`); §6.3 voiced harmonic
  generator (per-harmonic sinusoid at l · ω₀ with linear amp tilt
  M_prev → M_curr + quadratic phase integration of the ω₀ drift,
  dual-frame iteration so voiced↔unvoiced transitions fade in /
  out cleanly) is in (`internal/voice/mbe/synth_voiced.go`,
  `SynthState` extended with `PrevPhase` + `PrevMl`); §6.4 unvoiced
  excitation (256-point FFT spectrum shaping — bins under voiced
  harmonics zeroed, bins under unvoiced harmonics scaled by Ml[l],
  bins outside [1..L] zeroed, conjugate-mirror invariant preserved
  so the IFFT output stays real-valued) is in
  (`internal/voice/mbe/synth_unvoiced.go`); caller supplies the
  noise buffer so unit tests stay deterministic; the §6.4
  overlap-add synthesis window (256-sample periodic Hann × IFFT,
  96-sample tail threaded through `SynthState.PrevUnvoicedTail` so
  frame boundaries are click-free) is in via
  `SynthUnvoicedOverlapAdd`; the §6.2 spectral-amplitude
  enhancement (per-harmonic W_l = (0.96 · num/den)^0.25 clamped to
  [0.5, 1.2] for mid/high-band harmonics + low-band W = 1, followed
  by an energy-preserving rescale that holds R_M0 stable) is in
  (`internal/voice/mbe/enhance.go`); the output gain calibration is
  a per-frame fast-attack / slow-release peak-envelope tracker
  shared with AMBE+2 (target peak 24000, attack 0.4, release 0.02,
  gain clamped to [10, 1e5]) with first-frame seeding,
  freeze-on-silence, and Reset clearing — replaces the prior
  `pcmGain = 4096` magic constant with consistent loudness across
  speech-pause-speech transitions
  (`internal/voice/mbe/agc.go`).
  **`Decode()` emits real audio**: 88 info bits → params →
  §6.1 prediction → linear Ml → §6.2 enhancement → §6.3 voiced
  harmonic sum + §6.4 unvoiced excitation with overlap-add
  additive into one buffer → state roll-forward → per-frame AGC
  → int16 PCM at 8 kHz. Silence-window frames (b_0 ∈ [216, 219])
  still fade the prev unvoiced tail through the overlap region
  before resetting SynthState — no click on the silence
  boundary, and the AGC envelope is preserved across the
  silence so the next non-silent frame applies the same gain.
  Three decoder constructors are exposed: `New()` seeds the
  unvoiced noise source from a fixed default for reproducibility;
  `NewWithSeed(seed)` lets parallel calls + production callers
  spread noise across decoders; `NewWithConfig(seed, mbe.AGCConfig{...})`
  takes the shared `mbe.AGCConfig` (TargetPeak / Attack / Release /
  MinGain / MaxGain / NoiseFloor) so operators can dial level +
  responsiveness for their downstream chain — zero-value fields
  backfill from `mbe.DefaultAGCConfig()` so partial overrides like
  `mbe.AGCConfig{TargetPeak: 16000}` (drop level by ~3 dB) keep the
  rest of the defaults. **Frame-repeat on bad-frame indicator**:
  a bad frame (UnpackParams error from upstream FEC slip)
  following a good frame replays the cached params with M scaled
  by 0.7 per consecutive bad frame; up to 6 consecutive bad
  frames bridge ~120 ms of weak signal before Decode emits silence
  + clears the cache. The repeat path freezes the AGC envelope
  so the attenuation is audible (signals signal degradation).
  **Remaining audio polish**: absolute-level calibration against a
  known-good reference decoder (DSD-FME or OP25 — capture a P25
  Phase 1 voice exchange, decode through both, compare RMS +
  cross-correlation against the reference WAV under
  `internal/voice/imbe/testdata/`); enhancement filter tuning if
  real-world frames show mid-band envelope drift.
- **Pure-Go AMBE+2 vocoder.** A native-Go AMBE+2 2400 bps decoder
  for P25 Phase 2, DMR (Tier II / III), and NXDN voice frames.
  AMBE+2 is the same MBE-family algorithm as IMBE — same 8 kHz /
  20 ms / 160 PCM cadence, same harmonic + unvoiced FFT synthesis
  shape — so the synthesis half reuses `internal/voice/mbe/`
  directly. Only the front half (bit-level unpack from 49
  information bits into `mbe.Params`) is AMBE+2-specific. The
  AMBE+2 algorithm carries active patents in some jurisdictions;
  re-implementing it in Go does not change that posture
  ([docs/vocoders.md](docs/vocoders.md)). Status: skeleton +
  Vocoder interface registered as `ambe2` on the default build
  (`internal/voice/ambe2/decoder.go`), `Decode` emits silence
  pending parameter unpack + synthesis wiring. **Roadmap**:
  parameter unpack (b₀ → ω₀ + L + voicing-pattern table → Vl,
  gain VQ → gain block, two-stage spectral VQ → DCT residuals
  → Tl) — codebook tables transcribed from szechyjs/mbelib's
  `ambe3600x2400.c` / `ambe3600x2450_const.h` under ISC; synthesis
  wire-up via the shared `mbe.PredictLog2Ml` → `mbe.SynthVoiced`
  → `mbe.SynthUnvoicedOverlapAdd` → `mbe.AGC.Apply` pipeline +
  the shared `mbe.MaxBadFrames` / `mbe.BadFrameAttenuation`
  frame-repeat path; AGC calibration against a DSD-FME-decoded
  DMR reference recording.
- **DVSI USB-3000 / AMBE-3003 hardware backend.** A `Vocoder`
  factory that opens a connected DVSI USB chip. Same plug-in shape
  as `internal/voice/ambe2`; the daemon picks the factory by name
  from `voice.DefaultRegistry`.
- **YSF Trellis decode + grant emission.** Sync, frame layout, and
  the post-FEC FICH bit-level parser are in; what's left is the
  K=5 ½-rate Viterbi Trellis decoder over the on-air 100-bit FICH
  region and the control-channel wiring that publishes
  `protocol = "ysf"` grants on the bus when a Header FICH lands.
- **Higher-fidelity FM voice chain.** ✅ Shipped: opt-in 75/50µs
  de-emphasis (`composer.DeEmphasisConfig`), Kaiser-windowed audio
  LPF (`composer.AudioLPFConfig`), audio AGC
  (`composer.AudioAGCConfig`), and polyphase L/M audio resampler
  (`composer.AudioResamplerConfig`).

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

### Download a prebuilt release

Each tagged release publishes installers / archives on the
[**Releases page**][releases]:

| Platform   | File                                                   | What it is                                              |
| ---------- | ------------------------------------------------------ | ------------------------------------------------------- |
| Windows 11 | `gophertrunk-<ver>-windows-amd64-setup.exe`            | One-click installer (Inno Setup, bundles librtlsdr DLLs) |
| Windows 11 | `gophertrunk-<ver>-windows-amd64.zip`                  | Portable ZIP — same files, no installer                  |
| Linux      | `gophertrunk-<ver>-linux-amd64.tar.gz`                 | Tarballed binary + sample config                         |

Windows users: after running the installer, follow
[`docs/install-windows.md`](docs/install-windows.md) to swap the
RTL-SDR driver to WinUSB via Zadig — the OS won't see your dongle
until that's done. The installer's last page links there too.

[releases]: https://github.com/MattCheramie/GopherTrunk/releases

### Build from source

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
cmd/gophertrunk/        daemon entrypoint + sdr list CLI + read-only TUI
internal/tui/           bubbletea TUI: 8 read-only panels over REST+SSE
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

## TUI

GopherTrunk ships a read-only operator TUI that points at a running
daemon. From a second terminal:

```bash
gophertrunk tui                    # default: http://127.0.0.1:8080
gophertrunk tui -server http://10.0.0.5:8080
gophertrunk tui -no-color          # disable ANSI colour
gophertrunk tui -insecure          # skip TLS verification
```

Eight panels covering every read surface, vim-style navigation, live
SSE event stream, periodic REST refresh, automatic reconnect on
disconnect:

| Key | Action |
| --- | --- |
| `Tab` / `Shift+Tab` | next / previous panel |
| `1`–`8` | jump to Dashboard / Systems / Talkgroups / Active / History / Events / Tones / Metrics |
| `j` / `k` | move row up / down inside a table |
| `/` | filter (Talkgroups, Events) |
| `s` | cycle sort (Talkgroups) |
| `p` | pause auto-scroll (Events) |
| `r` | reload (History) |
| `?` | toggle help |
| `q` / `Ctrl+C` | quit |

For mutation actions (end-call, set-priority, lockout,
retention-sweep, tone-detector reset) start the daemon with
`api.allow_mutations: true` and the TUI with `--write`. Both ends
gate independently because the HTTP API has no authentication.
See [`docs/tui.md`](docs/tui.md) for the full reference.

## Documentation

- [`docs/architecture.md`](docs/architecture.md) — layered overview,
  concurrency model, driver registry, build tags
- [`docs/tui.md`](docs/tui.md) — TUI keybindings, panel reference,
  troubleshooting
- [`docs/hardware.md`](docs/hardware.md) — udev rules, DVB blacklist,
  IQ capture for replay
- [`docs/vocoders.md`](docs/vocoders.md) — IMBE / AMBE+2 licensing
  realities and the plugin model
- [`docs/hardening.md`](docs/hardening.md) — Prometheus catalogue,
  Docker / compose USB pass-through, smoke-test checklist

## License

See [`LICENSE`](LICENSE).
