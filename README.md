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
| Hardware          | Pure-Go RTL-SDR driver (USBDEVFS / WinUSB transport + RTL2832U register layer + R820T/R820T2/R828D/E4000/FC0012/FC0013/FC2580 tuner drivers; `CGO_ENABLED=0` everywhere, no `librtlsdr` / `libusb` build dependency), multi-device pool, role assignment, per-device gain (`auto` / tenths-of-dB) + PPM + bias-tee (5 V LNA power, e.g. NESDR Smart v5) applied at open time, DC blocker, IQ-imbalance correction, file-backed IQ replay (mock) |
| DSP               | Polyphase channelizer, FIR + Kaiser LPF designer + RRC + Gaussian-pulse premod designer, CIC, halfband, IQ + audio AGC (attack/release envelope follower for voice), L/M polyphase resampler (complex IQ + real audio), FM / C4FM / GFSK / FFSK (audio-band 1200-baud, MPT 1327) / DQPSK / π/4-DQPSK (configurable rotation; π/4 = TETRA, π/8 = P25 Phase 2 H-DQPSK) demods, single-pole IIR de-emphasis (75/50µs), Mueller-Müller clock recovery, frame-sync correlator |
| FEC primitives    | CRC-CCITT/FALSE + CRC-CCITT/XMODEM (callable init), CRC-6 (NXDN SACCH), Hamming(15,11,3), Hamming(13,9,3), Hamming(20,8) (DMR slot-type, t=3), extended Golay(24,12,8) + non-extended Golay(23,12,7) (P25 IMBE), BCH(63,16,11), BPTC(196,96), Reed-Solomon(12,9,4) over GF(2^8) with DMR Voice LC Header / Terminator / Embedded LC seeds, 4-state ½-rate Viterbi, 16-state K=5 ½-rate Viterbi (shared by NXDN SACCH and YSF FICH) with depuncture-marker support |
| P25 Phase 1       | 48-bit FSW + sync detector, NID parser (NAC + DUID) with BCH(63,16,11) error correction + even-parity check, full TSBK channel decode (TIA-102.BAAA Annex A 4-state ½-rate trellis + 98-dibit block deinterleaver) → CRC trailer validation, payload parsers for GroupVoiceChannelGrant / Update / NetworkStatus / RFSSStatus, IdentifierUpdate band-plan resolver, control-channel state machine emitting `protocol = "p25"` grants and `decode.error` events with `nid-bch` / `tsbk-trellis` / `tsbk-crc` / `no-bandplan` stages |
| P25 Phase 2       | Outbound + inbound 20-dibit sync, 360 ms / 12-subframe superframe + SlotType enum, MAC PDU parser + opcode enum, GroupVoiceChannelGrant accessor, control-channel state machine emitting `protocol = "p25-phase2"` grants |
| DMR (Tier III)    | All 9 ETSI sync patterns, burst layout (132 dibits), Color Code + Data Type via (20,8,7) shortened-Hamming slot-type FEC (corrects up to 3 bit errors per slot type), CSBK with CRC, payload parsers for TalkGroup/Private Voice grants (LCN + timeslot) + Aloha + AdjacentSiteStatus + SystemInfoBroadcast, LCN → Hz band-plan resolver (linear + table forms), IQ → C4FM dibit receiver (`internal/radio/dmr/receiver`) composing FM demod + RRC matched filter + Mueller-Müller clock recovery + 4-level slicer to fan `dmr.DibitSink` out to a future `ControlChannel.Process` adapter, control-channel state machine emitting `protocol = "dmr-tier3"` grants and `decode.error` events with `no-bandplan` stage |
| DMR (Tier II)     | Shares the burst / slot-type / BPTC(196,96) layers with Tier III; adds a 72-bit Full Link Control parser (FLCO enum: GroupVoiceChannelUser / UnitToUnitVoice / TalkerAlias / GPS / Terminator) with RS(12,9,4) parity verification (Voice LC Header seed) and a per-repeater conventional-mode state machine that decodes Voice LC Header bursts and emits `protocol = "dmr-tier2"` grants on the bus (deduped per call, cleared on Terminator-with-LC) and `decode.error` events with `voiceheader-bptc` / `voiceheader-rs` stages |
| NXDN              | 192-dibit frame layout (4800 BFSK / 9600 4-FSK), LICH parse with parity + 16-bit doubled-wire decoder, FSW correlator, full SACCH channel decode (K=5 ½-rate convolutional Viterbi + 60-position sub-frame deinterleaver + 12-bit puncture undo + CRC-6 trailer), CAC parser with CRC, RCCH opcode enum + payload parsers, IQ → C4FM dibit receiver (`internal/radio/nxdn/receiver`) for the 9600-baud 4-FSK variant composing FM demod + RRC matched filter + Mueller-Müller clock recovery + 4-level slicer to fan `nxdn.DibitSink` out to a future `ControlChannel.Process` adapter (BFSK variant — 2-level slicer — is a follow-up), control-channel state machine |
| Motorola Type II  | OSW parser, opcode constants, LCN → Hz band-plan resolver (linear + table), control-channel state machine emitting `protocol = "motorola"` grants |
| EDACS / GE-Marc   | 40-bit CCW parser, command enum (Idle / GroupVoiceGrant / ProVoiceGrant / IndividualCall / DataGrant / SystemID / AdjacentSite / Emergency / Affiliation / Encryption), per-command accessors with encrypted / emergency flags, LCN → Hz resolver, control-channel state machine emitting `protocol = "edacs"` grants |
| LTR               | 41-bit per-repeater Status word parser, Channel → Hz resolver, optional area filter, per-repeater state machine emitting `protocol = "ltr"` grants when a status indicates an active call |
| MPT 1327          | 64-bit address-codeword parser (38 info + 26 BCH parity consumed upstream), CodewordKind enum (ALH / AHY / AHYC / GTC / ACK / Disconnect / Data / Emergency), accessors for GTC voice grants + AHYC system broadcast, channel resolver, control-channel state machine emitting `protocol = "mpt1327"` grants |
| dPMR (Mode 3)     | FS1 / FS2 / FS3 24-dibit sync, 80-bit CSBK parser, MessageType enum (RegistrationRequest / Response, VoiceServiceAllocation, IndividualVoiceAllocation, DataServiceAllocation, ServiceRequest, StandingServiceStatus, Release, Idle), AsVoiceGrant + AsSiteBroadcast accessors, PMR446 default band-plan, IQ → C4FM dibit receiver (`internal/radio/dpmr/receiver`) composing FM demod + RRC matched filter + Mueller-Müller clock recovery + 4-level slicer at the 2400-sym/s rate to fan `dpmr.DibitSink` out to a future `ControlChannel.Process` adapter, control-channel state machine emitting `protocol = "dpmr"` grants |
| TETRA (TMO)       | Normal + extended training-sequence sync, generic Layer-3 PDU parser (4-bit Discriminator + type + payload), CMCE D-CONNECT / D-TX-GRANTED / D-RELEASE accessors, MLE-SYSINFO accessor (MCC / MNC / Location Area), TETRA-380 / 410 / 800 carrier resolver, control-channel state machine emitting `protocol = "tetra"` grants |
| D-STAR            | Frame Sync + Slow Data sync, 41-byte PCH header parser (FLAG1 + RPT2 / RPT1 / UR / MY1 / MY2 + CRC-CCITT), IsGroupCall / IsEmergency / IsData accessors, repeater state machine emitting `protocol = "dstar"` grants on group transmissions |
| YSF (Yaesu System Fusion) | 4800-baud C4FM, 480-dibit / 100 ms frame layout (FSW / FICH / DCH offsets), 40-bit FSW correlator with mismatch tolerance, 32-bit Frame Information Channel parser (FrameType / CallType / Frame Number / Frame Total / DataType / VoIP / Squelch fields) with CRC-16 trailer, K=5 ½-rate Viterbi Trellis encoder + decoder over the 104-bit (48 info + 4 tail) FICH channel-bit region (`internal/radio/ysf/fich_trellis.go`, shared with NXDN SACCH), IQ → C4FM dibit receiver (`internal/radio/ysf/receiver`) composing FM demod + RRC matched filter + Mueller-Müller clock recovery + 4-level slicer to feed `ysf.DibitSink` into `ControlChannel.Process`, per-frequency state machine emitting `cc.locked` on sync detect and `protocol = "ysf"` grants (with the FICH SquelchCode as DG-ID talkgroup) on Header FICH for Group calls — Terminator FICH clears the dedup so the next transmission fires a fresh CallStart |
| Orchestration     | In-process pub/sub event bus with typed payloads (Grant / CallStart / CallEnd / DecodeError / ToneAlert / etc.) and a typed `events.Stage` enum so protocol packages can't accidentally publish a stage label that drifts from the Prometheus dashboards, `System` model, JSON-on-disk last-known-CC cache, control-channel `Hunter` that retunes the SDR and parks on the first responsive frequency |
| Trunking engine   | Cross-protocol `Grant` payload, Trunk-Recorder-format talkgroup DB (CSV + JSON, including a per-TG `Scan` flag), `ScanMode` enum (`all` / `list`) that gates HandleGrant against the scan list (Emergency bypasses), priority + preemption (emergency overrides, strict-higher), voice-device pool allocator, central state machine emitting `CallStart` / `CallEnd` events with a watchdog for silent calls, plus `HandleSyntheticCall` / `EndSyntheticCall` entry points for external scanners (conventional FM) that already own their SDR |
| Scanner subsystem | Multi-system control-channel hunter (`internal/scanner/cchunt`) that round-robins trunked systems on one control SDR, publishes `cchunt.progress` / `cchunt.failed` telemetry events, persists last-good CC per system to a JSON cache, and supports operator hold / resume / force-retune; conventional FM scan list (`internal/scanner/conventional`) with IQ-power squelch (RMS-power dBFS detector, no FM-discriminator required), per-channel hangtime + priority + label, hop-on-silence state machine, synthetic-Grant handoff to the engine so the recorder + call log + API surfaces light up unchanged; operator hold / resume / dwell-on-index; all controlled from the TUI Scanner panel (key `0`) + REST cockpit at `/api/v1/scanner` |
| Demod pipeline    | `internal/voice/composer` subscribes to `CallStart` events, opens the bound Voice device's IQ stream, runs an LPF → decimate → optional CMA equalizer → FM demod → optional 75/50µs de-emphasis → optional Kaiser audio LPF → optional audio AGC → optional polyphase L/M resample (or naive decimate fallback) → int16 PCM chain into the recorder, and pings `Engine.Touch` every second so the silent-call watchdog leaves the call alone |
| Simulcast / "True I/Q" | `internal/dsp/equalizer` (LMS + CMA blind equalizers) for inter-symbol-interference / multipath mitigation, plus `internal/dsp/diversity` (Selection + maximal-ratio combiners over a shared `Combiner` interface) for multi-receiver IQ combining |
| Tone-out alerting | `internal/voice/toneout` runs Goertzel filters against each Voice device's PCM stream, matches QC-II two-tone-sequential sequences against operator-configured profiles with per-tone duration + cooldown, and publishes `tone.alert` events that fan out through SSE / WebSocket / gRPC |
| Voice recording   | `Vocoder` plugin interface + `NullVocoder` baseline, 16-bit PCM mono WAV writer with patched-length trailers, per-call recorder writing `<system>/<tg>/<UTC>_src<id>.wav` plus an optional raw-frame sidecar so users can BYO decoder; EDACS ProVoice grants always force a `.raw` sidecar (the vocoder is patent + trade-secret encumbered) so researchers can decode out-of-band |
| API               | `proto/*.proto` schemas under repo root; HTTP REST (`/api/v1/{health,version,systems,talkgroups,calls/active,calls/history,devices,scanner}`); operator mutations gated behind `api.allow_mutations` (`GET /api/v1/mutations` capability probe; `POST /api/v1/calls/{serial}/end`; `PATCH /api/v1/talkgroups/{id}` accepts priority/lockout/scan; `POST /api/v1/retention/sweep`; `POST /api/v1/devices/{serial}/tone-reset`; `PATCH /api/v1/scanner` flips scan_mode at runtime; `POST /api/v1/scanner/hunt/{system}/{hold\|resume\|retune}` and `POST /api/v1/scanner/conventional/{hold\|resume\|{index}/dwell}` drive the police-scanner cockpit); Server-Sent Events stream (`/api/v1/events`) — per-device hot-plug surfaces as `sdr.attached` / `sdr.detached`, scanner progress as `cchunt.progress` / `cchunt.failed`; WebSocket bridge (`/api/v1/events/ws`); gRPC `SystemService` + `TalkgroupService` + `AudioService` over the same in-process state |
| Persistence       | Pure-Go SQLite (`modernc.org/sqlite`) call log subscribing to `CallStart` / `CallEnd` events; newest-first history queries with system / group / time filters; retention sweeper that ages out DB rows and recorded `.wav` / `.raw` files past configurable cutoffs |
| Observability     | Prometheus collector (events / calls / CC-locked / IQ-underrun / USB-reconnect / decode-error / SDR-attached / build-info series) exposed at `/metrics`; multi-stage `Dockerfile`; `docker-compose.yml` with RTL-SDR USB pass-through, healthcheck, and Prometheus scrape labels |
| Daemon            | `cmd/gophertrunk run` composes everything above into a single supervised process with signal-driven shutdown; every component is opt-in via `config.yaml` |
| Testing           | Per-package unit tests under `make test`; `make integration` boots the wired daemon end-to-end (no SDR), publishes a synthetic call on the bus, and asserts the engine + recorder + call log + metrics + API agree — runs on every CI build |

## Status & known gaps

Once a `grant` event lands on the bus, the engine + recorder pipeline
runs end-to-end: voice device is allocated, the composer pulls IQ →
PCM, the recorder writes a WAV (digital-voice protocols decode through
the right vocoder via `voice.DefaultVocoderForProtocol`), the call is
logged to SQLite, and the API + TUI surfaces all light up. Pure-Go
IMBE / AMBE+2 produce intelligible audio. The CC Hunter supervisor and
the conventional FM scanner are constructed by `cmd/gophertrunk` and
expose their state through `/api/v1/scanner` and the TUI cockpit
panel. The honest remaining gaps:

- **IQ → control-channel decoder daemon wiring.** Every protocol
  package ships a unit-tested control-channel state machine and the
  P25 Phase 1 IQ → C4FM dibit receiver
  (`internal/radio/p25/phase1/receiver`) now exposes both an LDU
  sink (voice path) and a raw-dibit sink (control-channel path —
  feeds `phase1.ControlChannel.Process` directly), but the connector
  that takes the control SDR's live IQ stream, picks the right
  protocol's receiver, and feeds its control-channel pipeline isn't
  constructed by `cmd/gophertrunk` yet. Until that lands the CC
  Hunter supervisor exhausts every candidate frequency without
  seeing a `cc.locked` event, so each system enters `state=failed`
  and backs off — the TUI Scanner panel surfaces this honestly
  rather than faking a lock.
- **Digital-voice level calibration.** Pure-Go IMBE / AMBE+2 emit
  real audio end-to-end with shared AGC, frame-repeat on bad-frame
  indicator, phase-aware fade-in, and §6.2 spectral enhancement
  shipping. The comparison harness lives at
  `internal/voice/calibrate/` — `calibrate.Compare(raw, refWav,
  vocoderName)` returns `RMSRatioDb` + normalised cross-correlation
  + lag against a reference WAV from DSD-FME or OP25, and the
  package's `TestCompare{IMBE,AMBE2}SkipsWithoutFixtures` tests
  enforce `|RMS offset| < 3 dB` and peak xcorr > 0.85 once
  fixtures are in place. The remaining gap is sourcing the
  reference data: captured P25 P1 / DMR voice exchanges plus
  DSD-FME / OP25 decodes belong at
  `internal/voice/{imbe,ambe2}/testdata/`. AMBE+2 single-tone
  synthesis works; dual-tone (b₁ ∈ [128, 163]) routes through
  silence pending a frequency-pair lookup that the public spec
  doesn't document. See [docs/vocoders.md](docs/vocoders.md) for
  the licensing posture.
- **YSF FICH on-air interleaver / puncture validation.** The K=5
  ½-rate Trellis encoder + decoder are in
  (`internal/radio/ysf/fich_trellis.go`) and round-trip cleanly in
  unit tests; calibration against a captured YSF transmission's
  exact interleaver / puncture schedule lands once a real-air capture
  is available.

The Go interfaces and event payloads carry every protocol already;
the remaining decoder wiring is the load-bearing follow-up.

## Roadmap

What's still on the table. Order isn't fixed; each item is contained
to its own package and lands independently.

- **IQ-domain control-channel decoder daemon wiring.** The protocol
  packages all ship unit-tested control-channel state machines; the
  P25 Phase 1 IQ → C4FM dibit receiver is in
  (`internal/radio/p25/phase1/receiver`); the CC Hunter supervisor
  (`internal/scanner/cchunt`) round-robins systems and retunes the
  control SDR. What's missing is the connector that takes the live IQ
  stream from the control device, runs it through the right protocol's
  receiver + control-channel pipeline, and publishes the resulting
  `cc.locked` / `grant` events on the bus. Without it the supervisor
  will always report `state=failed` per system (which the TUI Scanner
  panel shows truthfully). This is the load-bearing follow-up that
  lights up live trunked reception.
- **DVSI USB-3000 / AMBE-3003 hardware backend.** A `Vocoder`
  factory that opens a connected DVSI USB chip. Same plug-in shape
  as `internal/voice/ambe2`; the daemon picks the factory by name
  from `voice.DefaultRegistry`. Useful for operators who want
  vendor-blessed AMBE+2 decode on jurisdictions where the patent
  posture matters.
- **Vocoder level calibration.** Pure-Go IMBE / AMBE+2 produce real
  audio end-to-end today. Absolute-level calibration against a
  DSD-FME or OP25 reference recording (capture a P25 P1 / DMR voice
  exchange, decode through both, compare RMS + cross-correlation
  against a reference WAV under `internal/voice/{imbe,ambe2}/testdata/`)
  is a polish item — useful when downstream pipelines need consistent
  loudness across decoders. AMBE+2 dual-tone synthesis
  (b₁ ∈ [128, 163]) routes through silence pending a frequency-pair
  lookup that the public spec doesn't document.
- **YSF on-air interleaver / puncture validation.** The K=5 ½-rate
  Trellis encoder + decoder round-trip cleanly; matching the exact
  on-air interleaver / puncture schedule to a captured YSF
  transmission lands once a real-air capture is available.

### Recently shipped

- **dPMR Mode 3 IQ → C4FM dibit receiver** (`internal/radio/dpmr/receiver`)
  composing FM demod + RRC matched filter + Mueller-Müller clock
  recovery + 4-level slicer at the 2400-sym/s rate (half the P25 P1
  / DMR / NXDN / YSF rate, matching dPMR's 6.25 kHz channel
  spacing) into one entry point that fans dibits out via the new
  `dpmr.DibitSink` callback. The `ControlChannel.Process(dibits,
  baseIdx)` adapter that does FS3 sync detect + 80-bit CSBK slice +
  `CSBKFromBits` + `Ingest` is the next layer up.
- **NXDN IQ → C4FM dibit receiver** (`internal/radio/nxdn/receiver`)
  composing FM demod + RRC matched filter + Mueller-Müller clock
  recovery + 4-level slicer into one entry point that fans dibits
  out via the new `nxdn.DibitSink` callback. Targets the 9600-baud
  4-FSK variant (the same C4FM modulation P25 P1 / DMR / YSF use);
  the 4800-baud BFSK variant — 2-level slicer rather than 4-level —
  is a follow-up. The `ControlChannel.Process(dibits, baseIdx)`
  adapter that does 8-dibit FSW detect + 192-dibit frame slice +
  LICH / SACCH decode + `IngestFrame` is the next layer up.
- **DMR IQ → C4FM dibit receiver** (`internal/radio/dmr/receiver`)
  composing FM demod + RRC matched filter + Mueller-Müller clock
  recovery + 4-level slicer into one entry point that fans dibits
  out via the new `dmr.DibitSink` callback. Same shape as the
  YSF / P25 P1 receivers (4800-baud C4FM is shared across the
  4FSK family); the cross-call buffering + sync-detect + 132-dibit
  burst assembly + `Process(dibits, baseIdx)` adapter on
  `tier3.ControlChannel` is the next layer up.
- **YSF IQ → C4FM dibit receiver** (`internal/radio/ysf/receiver`)
  composing FM demod + RRC matched filter + Mueller-Müller clock
  recovery + 4-level slicer into one entry point that fans dibits
  out via the new `ysf.DibitSink` callback — wire it into
  `ysf.ControlChannel.Process` to drive the per-frequency state
  machine on real IQ. Same shape as the P25 Phase 1 receiver
  (4800-baud C4FM is the shared modulation); SymbolToDibit follows
  the P25 / DSDcc convention pending real-air FSW-pattern
  validation.
- **Vocoder calibration harness** (`internal/voice/calibrate/`)
  — `Compare(raw, refWav, vocoderName)` returns RMS-ratio (dB),
  normalised cross-correlation, and best alignment lag against an
  external decoder's reference WAV. Unit tests cover the RMS +
  cross-correlation primitives + a WAV round-trip via the shared
  `voice.WavWriter`; integration tests for IMBE / AMBE+2 skip
  cleanly until the testdata fixtures land. The harness's failure
  output names the AGC constant in
  `internal/voice/mbe/agc.go:DefaultAGCConfig` to adjust.
- **Police-scanner subsystem** (`internal/scanner/{cchunt,conventional}`)
  — multi-system CC Hunter supervisor with hold/resume/force-retune,
  conventional FM scan list with IQ-power squelch, talkgroup scan
  list with global ScanMode, 10th TUI panel and REST cockpit at
  `/api/v1/scanner`.
- **TUI Devices panel** + `GET /api/v1/devices` + `sdr.attached` /
  `sdr.detached` event publishing in the SDR pool.
- **TUI drill-in modals** on Systems and Talkgroups (Enter).
- **P25 Phase 1 IQ → C4FM dibit receiver** (`internal/radio/p25/phase1/receiver`)
  composing FM demod + RRC matched filter + Mueller-Müller clock
  recovery + 4-level slicer into one entry point that fans out to
  both the LDU assembler (voice path) and an optional raw-dibit sink
  (`phase1.DibitSink` — control-channel path).
- **YSF FICH Trellis decoder + grant emission** on Header FICH for
  Group calls (`internal/radio/ysf/fich_trellis.go` + extended
  `control.go`).
- **Pure-Go RTL-SDR driver** (`internal/sdr/rtlsdr/{usb,rtl2832u,tuners,purego}/`)
  replaced the `librtlsdr` + `libusb` C dependency. Pure-Go USB
  transports for Linux (USBDEVFS), Windows (WinUSB), and macOS (IOKit
  via `purego`); RTL2832U register/I2C layer; R820T/R820T2/R828D +
  E4000 + FC0012 + FC0013 + FC2580 tuner drivers. Default builds
  run `CGO_ENABLED=0` end-to-end.
- **Pure-Go IMBE vocoder** (`internal/voice/imbe/` + shared
  `internal/voice/mbe/`) and **pure-Go AMBE+2 vocoder**
  (`internal/voice/ambe2/`) — both produce intelligible audio
  end-to-end with shared AGC, §6.2 spectral enhancement, frame-repeat
  on bad-frame indicator, phase-aware fade-in.
- **Higher-fidelity FM voice chain**: opt-in 75/50µs de-emphasis,
  Kaiser-windowed audio LPF, audio AGC, polyphase L/M audio
  resampler (`composer.{DeEmphasis,AudioLPF,AudioAGC,AudioResampler}Config`).

## Tech stack

- **Language:** Go 1.24+
- **Hardware:** Pure-Go RTL-SDR driver — USBDEVFS / WinUSB transport, RTL2832U register layer, and per-chip tuner drivers (R820T/R820T2/R828D/E4000/FC0012/FC0013/FC2580). `CGO_ENABLED=0`; no `librtlsdr` / `libusb` build dependency.
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
| Windows 11 | `gophertrunk-<ver>-windows-amd64-setup.exe`            | One-click installer (Inno Setup) — single static binary  |
| Windows 11 | `gophertrunk-<ver>-windows-amd64.zip`                  | Portable ZIP — same binary, no installer                 |
| Linux      | `gophertrunk-<ver>-linux-amd64.tar.gz`                 | Tarballed static binary + sample config                  |

Windows users: after running the installer, follow
[`docs/install-windows.md`](docs/install-windows.md) to swap the
RTL-SDR driver to WinUSB via Zadig — the OS won't see your dongle
until that's done. The installer's last page links there too.

[releases]: https://github.com/MattCheramie/GopherTrunk/releases

### Build from source

### Prerequisites

Just Go 1.24+. The pure-Go RTL-SDR driver doesn't need
`librtlsdr` / `libusb` / a C toolchain on the build host.

See [`docs/hardware.md`](docs/hardware.md) for runtime `udev` rules
and DVB-driver blacklisting on Linux.

### Build, test, run

```sh
make build         # produces ./bin/gophertrunk
make test          # go test -race ./...
make integration   # boots the wired daemon end-to-end (no SDR needed)

./bin/gophertrunk version
./bin/gophertrunk sdr list                # enumerates attached dongles
./bin/gophertrunk run -config config.yaml

# Out-of-band: decode a captured .raw frame sidecar to a WAV using
# the pure-Go IMBE / AMBE+2 vocoders. The .raw sidecar is written
# alongside each call's WAV when the recorder's raw-frames option
# is enabled.
./bin/gophertrunk decode -in call.raw -out call.wav -vocoder imbe
./bin/gophertrunk decode -list-vocoders
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
cmd/gophertrunk/        daemon entrypoint + sdr list CLI + read+write TUI cockpit
internal/tui/           bubbletea TUI: 10 panels (Scanner cockpit is panel #10) over REST+SSE
internal/sdr/           Driver interface, pool, mock
internal/sdr/rtlsdr/usb/      Pure-Go USB transport: Linux USBDEVFS, Windows WinUSB, macOS IOKit (purego), mock
internal/sdr/rtlsdr/rtl2832u/ RTL2832U register/I2C layer (sample-rate, IF, FIR, GPIO, I2C bridge)
internal/sdr/rtlsdr/tuners/   R820T/R820T2/R828D + E4000 + FC0012 + FC0013 + FC2580 tuner drivers
internal/sdr/rtlsdr/purego/   sdr.Driver+sdr.Device wire-up; canonical "rtlsdr" registrant
internal/dsp/           Channelizer, filters, demods, sync, FFT
internal/radio/         framing/ + p25/phase1/ (+ phase1/receiver IQ→dibit) + dmr/ + nxdn/ + ysf/
internal/trunking/      System, talkgroup DB (Scan flag), engine (ScanMode, HandleSyntheticCall), priority, CC hunter primitive, cc cache
internal/scanner/       cchunt/ (multi-system CC supervisor) + conventional/ (analog FM scan list)
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

GopherTrunk ships an operator TUI that points at a running
daemon. From a second terminal:

```bash
gophertrunk tui                    # default: http://127.0.0.1:8080
gophertrunk tui -server http://10.0.0.5:8080
gophertrunk tui -no-color          # disable ANSI colour
gophertrunk tui -insecure          # skip TLS verification
```

Ten panels covering every read surface plus the operator scanner
cockpit, vim-style navigation, live SSE event stream, periodic REST
refresh, automatic reconnect on disconnect:

| Key | Action |
| --- | --- |
| `Tab` / `Shift+Tab` | next / previous panel |
| `1`–`9`, `0` | jump to Dashboard / Systems / Talkgroups / Active / History / Events / Tones / Metrics / Devices / Scanner |
| `j` / `k` | move row up / down inside a table |
| `/` | filter (Talkgroups, Events) |
| `s` | cycle sort (Talkgroups) |
| `S` | toggle scan flag (Talkgroups; mutates) |
| `Enter` | open detail card (Systems, Talkgroups) or dwell (Scanner conv row) |
| `h` | hold/resume highlighted system or conv channel (Scanner; mutates) |
| `r` | force re-hunt highlighted system (Scanner; mutates) |
| `m` | cycle scan_mode list↔all (Scanner; mutates) |
| `p` | pause auto-scroll (Events) |
| `r` | reload (History) |
| `?` | toggle help |
| `q` / `Ctrl+C` | quit |

For mutation actions (end-call; set talkgroup priority / lockout /
scan; retention-sweep; tone-detector reset; scanner cockpit
hold/resume/retune/dwell + scan_mode flip) start the daemon with
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
