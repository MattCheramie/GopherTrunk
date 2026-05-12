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
| P25 Phase 2       | Outbound + inbound 20-dibit sync, 360 ms / 12-subframe superframe + SlotType enum, MAC PDU parser + opcode enum, GroupVoiceChannelGrant accessor, IQ → H-DQPSK dibit receiver (`internal/radio/p25/phase2/receiver`) composing the `demod.PiOver4DQPSK` helper with π/8 rotation + naive symbol-time decimation at 6000 sym/s to fan `phase2.DibitSink` out to a future `ControlChannel.Process` adapter (full symbol-time clock recovery on complex IQ is a follow-up), control-channel state machine emitting `protocol = "p25-phase2"` grants |
| DMR (Tier III)    | All 9 ETSI sync patterns, burst layout (132 dibits), Color Code + Data Type via (20,8,7) shortened-Hamming slot-type FEC (corrects up to 3 bit errors per slot type), CSBK with CRC, payload parsers for TalkGroup/Private Voice grants (LCN + timeslot) + Aloha + AdjacentSiteStatus + SystemInfoBroadcast, LCN → Hz band-plan resolver (linear + table forms), IQ → C4FM dibit receiver (`internal/radio/dmr/receiver`) composing FM demod + RRC matched filter + Mueller-Müller clock recovery + 4-level slicer to fan `dmr.DibitSink` out to a future `ControlChannel.Process` adapter, control-channel state machine emitting `protocol = "dmr-tier3"` grants and `decode.error` events with `no-bandplan` stage |
| DMR (Tier II)     | Shares the burst / slot-type / BPTC(196,96) layers with Tier III; adds a 72-bit Full Link Control parser (FLCO enum: GroupVoiceChannelUser / UnitToUnitVoice / TalkerAlias / GPS / Terminator) with RS(12,9,4) parity verification (Voice LC Header seed) and a per-repeater conventional-mode state machine that decodes Voice LC Header bursts and emits `protocol = "dmr-tier2"` grants on the bus (deduped per call, cleared on Terminator-with-LC) and `decode.error` events with `voiceheader-bptc` / `voiceheader-rs` stages |
| NXDN              | 192-dibit frame layout (4800 BFSK / 9600 4-FSK), LICH parse with parity + 16-bit doubled-wire decoder, FSW correlator, full SACCH channel decode (K=5 ½-rate convolutional Viterbi + 60-position sub-frame deinterleaver + 12-bit puncture undo + CRC-6 trailer), CAC parser with CRC, RCCH opcode enum + payload parsers, IQ → C4FM dibit receiver (`internal/radio/nxdn/receiver`) for the 9600-baud 4-FSK variant composing FM demod + RRC matched filter + Mueller-Müller clock recovery + 4-level slicer to fan `nxdn.DibitSink` out to a future `ControlChannel.Process` adapter (BFSK variant — 2-level slicer — is a follow-up), control-channel state machine |
| Motorola Type II  | OSW parser, opcode constants, LCN → Hz band-plan resolver (linear + table), IQ → MSK bit receiver (`internal/radio/motorola/receiver`) composing FM demod + Gaussian matched filter (BT = 0.5 approximation of MSK matched filter) + Mueller-Müller clock recovery at 3600 baud + 2-level slicer to fan `motorola.BitSink` out to a future `ControlChannel.Process` adapter, control-channel state machine emitting `protocol = "motorola"` grants |
| EDACS / GE-Marc   | 40-bit CCW parser, command enum (Idle / GroupVoiceGrant / ProVoiceGrant / IndividualCall / DataGrant / SystemID / AdjacentSite / Emergency / Affiliation / Encryption), per-command accessors with encrypted / emergency flags, LCN → Hz resolver, IQ → GFSK bit receiver (`internal/radio/edacs/receiver`) composing FM demod + Gaussian matched filter (BT = 0.3) + Mueller-Müller clock recovery + 2-level slicer at 9600 baud to fan `edacs.BitSink` out to a future `ControlChannel.Process` adapter, control-channel state machine emitting `protocol = "edacs"` grants |
| LTR               | 41-bit per-repeater Status word parser, Channel → Hz resolver, optional area filter, IQ → sub-audible bit receiver (`internal/radio/ltr/receiver`) composing FM demod + narrow sub-audible LPF (~300 Hz Kaiser-windowed FIR) + Mueller-Müller clock recovery at 300 baud + 2-level slicer to fan `ltr.BitSink` out to a future `ControlChannel.Process` adapter (Manchester decode + 41-bit framing live there), per-repeater state machine emitting `protocol = "ltr"` grants when a status indicates an active call |
| MPT 1327          | 64-bit address-codeword parser (38 info + 26 BCH parity consumed upstream), CodewordKind enum (ALH / AHY / AHYC / GTC / ACK / Disconnect / Data / Emergency), accessors for GTC voice grants + AHYC system broadcast, channel resolver, IQ → FFSK bit receiver (`internal/radio/mpt1327/receiver`) composing FM demod + FFSK tone discriminator (mark = 1200 Hz / space = 1800 Hz CCIR FFSK) + Mueller-Müller clock recovery at 1200 baud to fan `mpt1327.BitSink` out to a future `ControlChannel.Process` adapter, control-channel state machine emitting `protocol = "mpt1327"` grants |
| dPMR (Mode 3)     | FS1 / FS2 / FS3 24-dibit sync, 80-bit CSBK parser, MessageType enum (RegistrationRequest / Response, VoiceServiceAllocation, IndividualVoiceAllocation, DataServiceAllocation, ServiceRequest, StandingServiceStatus, Release, Idle), AsVoiceGrant + AsSiteBroadcast accessors, PMR446 default band-plan, IQ → C4FM dibit receiver (`internal/radio/dpmr/receiver`) composing FM demod + RRC matched filter + Mueller-Müller clock recovery + 4-level slicer at the 2400-sym/s rate to fan `dpmr.DibitSink` out to a future `ControlChannel.Process` adapter, control-channel state machine emitting `protocol = "dpmr"` grants |
| TETRA (TMO)       | Normal + extended training-sequence sync, generic Layer-3 PDU parser (4-bit Discriminator + type + payload), CMCE D-CONNECT / D-TX-GRANTED / D-RELEASE accessors, MLE-SYSINFO accessor (MCC / MNC / Location Area), TETRA-380 / 410 / 800 carrier resolver, IQ → π/4-DQPSK dibit receiver (`internal/radio/tetra/receiver`) composing the `demod.PiOver4DQPSK` helper with π/4 rotation + α = 0.35 RRC + naive symbol-time decimation at 18000 sym/s to fan `tetra.DibitSink` out to a future `ControlChannel.Process` adapter (full symbol-time clock recovery is a follow-up), control-channel state machine emitting `protocol = "tetra"` grants |
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
panel. **Every trunked control modulation in the Features table now
has an end-to-end IQ → CC chain shipping** — the `ccdecoder` connector
constructed by `cmd/gophertrunk` covers all 10 protocols (P25 Phase 1,
P25 Phase 2, DMR Tier III, NXDN, dPMR Mode 3, EDACS, Motorola Type II,
LTR, MPT 1327, TETRA TMO) plus YSF on the amateur side.

The remaining gaps:

- **Per-protocol on-air FEC layers.** The connector adapters that
  shipped between PRs #113–#121 reach the CC state machines via
  each protocol's `ControlChannel.Process(stream, baseIdx)` method.
  Most adapters skip the on-air FEC and read information bits
  straight from the wire — this works on test fixtures + clean
  signals but typically fails on captured on-air traffic.
  **DMR Tier III** ships full FEC end-to-end (BPTC(196,96) +
  CSBK CRC). **NXDN** now has `SetViterbiMode(ViterbiOn)` to
  run K=5 ½-rate Viterbi over the 92 encoded CAC dibits in the
  Info field (88 CAC info bits + 4 tail bits → 184 wire bits =
  92 dibits — the bare-bones K=5 chain MMDVMHost / DSDcc / op25
  share); the per-protocol interleaver + puncture inner layer
  is the next follow-up. The CAC CRC-CCITT-16 strict-mode
  remains enforced. **LTR** has a `SetManchesterMode` config
  for deployments that bi-phase-encode the sub-audible status
  word, and `SetFCSMode(FCSOn)` to verify the 7-bit CRC trailer
  per DSheirer/sdrtrunk's CRCLTR.java on the Ingest path. Under
  FCSOn the CRC covers a 24-bit message vector (gophertrunk's
  Group F-bit as sdrtrunk's "Area", plus Channel/Home/GroupID/
  Free); the gophertrunk 5-bit `Status.Area` stays as opaque
  metadata for the multi-system filter and only the low 7
  bits of the 12-bit `Status.FCS` field are CRC-protected.
  **Motorola Type II** has `SetBCHMode(BCHOn)` to run
  BCH(64,16,11) over each codeword pair. **P25 Phase 2** now
  has `SetTrellisMode(TrellisOn)` to run the TIA-102 Annex A
  4-state ½-rate trellis Viterbi decoder over the 146 channel
  dibits of each MAC PDU; the spec's Reed-Solomon outer layer
  + per-burst block interleaver are documented follow-ups.
  **MPT 1327** now has `SetBCHMode(BCHOn)` to run BCH(64,48,2)
  (polynomial 0x6815, init 0x0001, matching DSheirer/sdrtrunk's
  CRCFleetsync) over each 64-bit on-wire codeword with
  single-bit correction; the 10-bit Op field that the spec
  carries between Ident and Function isn't modelled by the
  Codeword struct yet (the wiring extracts the 38 info bits
  the existing struct cares about and drops Op). **EDACS** now
  has `SetBCHMode(BCHOn)` to run BCH(40, 28, 2) (generator
  0x1539, parameters from lwvmobile/edacs-fm's bch3.h) over
  each 40-bit on-wire CCW with single + double-bit error
  correction; under BCHOn the effective CCW carries 28 info
  bits (Command + Status + Address + 4 high bits of LCN), the
  legacy struct's LCN bit 0 and Aux become BCH parity rather
  than data. **TETRA** ships the full §8.3.1 signaling-channel
  chain — K=5 R=1/3 speech-traffic-channel code
  (`framing/rcpc_tetra.go`, EN 300 395-2 §5.4.3), K=5 R=1/4
  signaling-channel code (`framing/rcpc_tetra_sig.go`,
  EN 300 392-2 §8.2.3.1) with rates 2/3, 1/3, 292/432 and
  148/432, shortened (30,14) Reed-Muller code
  (`framing/rm_30_14_tetra.go`, §8.2.3.2) for AACH, 32-tap
  scrambler + (K,a) block interleaver from PR #138, and the
  per-channel encode/decode helpers from PR #139 — composed
  on the `tetra.ControlChannel.Process` adapter via the
  `SetChannelCoding(ChannelCodingOn)` opt-in and wired into
  the live decoder by the `ccdecoder` connector reading
  `tetra_colour_code` + `tetra_channel` off each
  `trunking.System` per PR #141. The remaining TETRA work
  on the "lights up live trunked reception" path is
  validation against a captured TETRA TMO IQ exchange —
  unit tests round-trip clean fixtures end-to-end, but
  on-air recovery margins (Viterbi correction depth vs.
  real co-channel + adjacent-channel interference) need a
  live capture to characterise.
- **Symbol-time clock recovery on complex IQ.** The Gardner
  timing-recovery primitive in `internal/dsp/sync/gardner.go`
  is now threaded into both the **P25 Phase 2** and **TETRA**
  receivers via a per-receiver `ClockMode` opt-in
  (`ClockNaive` default preserves the pre-Gardner behaviour for
  existing tests; `ClockGardner` routes the matched-filter
  output through the Gardner loop). Noisier on-air captures
  whose symbol clock isn't aligned with the SDR sample clock
  should lock cleanly under `ClockGardner`. The connector
  configuration for picking the mode at runtime is the
  follow-up.
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
- **Manual VFO tune from the TUI / API.** The Scanner panel now binds
  `f` to a bubbles/textinput overlay: type a frequency in MHz, Enter,
  and the conventional FM scanner appends a runtime "manual" channel
  and forces dwell on it. Same flow available over REST as `POST
  /api/v1/scanner/manual_tune` (and `DELETE /api/v1/scanner/manual_tune/{idx}`
  to revoke), gated behind `api.allow_mutations`. To run manual tune
  without any static `scanner.conventional` entries, set
  `scanner.manual_tune_enabled: true` in config — the daemon then
  constructs the conventional scanner against the last Voice SDR
  regardless of the static channel count. `internal/scanner/conventional`
  now accepts an empty seed channel list and exposes
  `AddTemporaryChannel` / `RemoveTemporaryChannel` so the same VFO
  surface is callable from any embedder.
- **Live audio playback to speakers + TUI / API audio cockpit.** The
  daemon ships a `voice.Player` sink (`internal/voice/player`) wrapping
  github.com/ebitengine/oto/v3 (ALSA on Linux, CoreAudio on macOS,
  WASAPI on Windows; libasound2-dev required at build time on Linux).
  When `audio.enabled: true` is set in config the per-call composer
  and the conventional FM scanner fan PCM into the player alongside
  the existing WAV recorder, so calls play out the host's default
  output device in real time. Volume / mute / recording can be
  toggled live: the TUI's Scanner panel binds `+` / `-` for volume
  (5% step), `M` for mute, and `R` for record on/off; the same knobs
  are exposed as `GET` / `PATCH /api/v1/audio` for remote clients
  (PATCH gated by `api.allow_mutations` like every other write
  endpoint). The recorder gate stops new WAVs from landing without
  truncating in-flight sessions, matching scanner muscle memory.
  Disabled by default; headless servers stay silent and continue to
  record WAVs identically to before. New CLI: `gophertrunk audio
  list` mirrors `sdr list`. Drop-the-libasound2-dev-build-dep
  follow-up tracked under Workstream F of the active plan.

The Go interfaces and event payloads carry every protocol already;
the remaining decoder wiring is the load-bearing follow-up.

## Roadmap

What's still on the table. Order isn't fixed; each item is contained
to its own package and lands independently.

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

- **YSF integration-cc + P25 P1 grant-chain extension.**
  **Closes the original planning roadmap** — every trunked
  protocol gophertrunk decodes now has an end-to-end "lights
  up live trunked reception" integration test, *and* the P25
  Phase 1 path now asserts the full status → IdentifierUpdate
  → GroupVoiceChannelGrant TSBK chain through the production
  daemon + bus + supervisor + API + metrics chain (the
  previous version stopped at cc.locked).
  - `cmd/gophertrunk/integration_cc_ysf_test.go` boots the
    daemon with synthesized 4800-baud C4FM IQ carrying
    back-to-back YSF FSW-bearing frames (480-dibit frame
    layout, FSWPattern at offset 0, zero-filled FICH +
    payload regions) and asserts the production
    `newYSFPipeline` + supervisor + API + metrics chain
    recovers the lock. Same C4FM modulator + RRC pulse
    shaping as P25 P1 / NXDN / DMR / dPMR; the receiver's
    `Options.DeviationHz` slicer calibration knob now ships
    on `internal/radio/ysf/receiver` (1800 Hz peak spec
    deviation per Yaesu's C4FM TX chain). `make
    integration-cc-ysf` runs it standalone.
  - `cmd/gophertrunk/integration_cc_test.go` grows a second
    test, `TestDaemonCCDecodesP25Phase1GrantChain`, that
    uses `ccdecoder.SetTestFactory` to install a stub
    pipeline pumping the synthesized FSW + NID + TSBK dibit
    stream straight into a real `phase1.ControlChannel` on
    the first IQ chunk. Exercises everything *above* IQ →
    dibit — the factory dispatch, the band plan, the bus
    publication, the engine, the supervisor, the API, the
    metrics handler — through the production code paths,
    without depending on the receiver's Mueller-Müller
    clock loop landing every subsequent FSW + NID + 98-dibit
    TSBK trellis window in one streaming pass (which it
    reliably does for the *first* lock but not for the
    multi-frame status → identifier → grant sequence the
    grant chain needs). `make integration-cc-grant` runs it
    standalone.
  - `trunking.ProtocolYSF` lands in the protocol enum
    (string form `"ysf"`); `ParseProtocol` + `Validate`
    accept it. The ccdecoder factory map registers
    `newYSFPipeline` for `ProtocolYSF` so live config
    `protocol: ysf` slots into the production hunt chain.
  - 30-run flakiness sweep on all three integration-cc P25
    P1 / YSF / grant tests clean.

  Punch-list status: **all 9 protocols + 4 modulator
  primitives + YSF + grant chain shipped.** The original
  roadmap that opened with "every protocol package ships a
  CC state machine, every trunking surface lights up the
  moment a grant lands, *and yet the daemon never publishes
  its first live cc.locked event*" is now fully closed —
  the daemon publishes cc.locked + grant on every supported
  protocol when synthesized IQ matching the protocol arrives
  on the control SDR.
- **Sub-audible NRZ modulator + `make integration-cc-ltr`.**
  **Closes the per-protocol "lights up live trunked
  reception" punch list — every trunked protocol gophertrunk
  decodes now has an end-to-end integration test running
  synthesized IQ through the production daemon + receiver
  chain.**
  - `internal/dsp/demod/subaudible_nrz_modulator.go` ships the
    TX counterpart to the LTR receiver's FM-demod →
    narrow-LPF → MM clock recovery → zero-threshold slicer
    chain: bit → bipolar symbol (±audioAmp) → FM modulator
    (phase advances by `audioAmp` per sample) → IQ. The audio
    amplitude is tuned so the FM demod output sits comfortably
    inside the receiver's LPF passband (below 300 Hz) at the
    9600× lower symbol rate vs the other protocols.
  - `ltr.LockState` now implements `trunking.LockedPayload`.
    **Eighth and final protocol with the same latent-bug class
    fixed** (NXDN / dPMR / EDACS / Motorola / TETRA / P25 Phase
    2 / MPT 1327 / LTR). LTR doesn't have a P25-style NAC; the
    `(Area, Repeater)` pair gets packed into the NAC slot as
    `(Area << 8) | Repeater`.
  - The integration test synthesizes 80 back-to-back idle
    Status words (no gap — LTR's 41-bit Status word stream is
    continuous) at 300 baud, modulates via the new sub-audible
    primitive, and asserts the daemon recovers the lock with
    the expected Area + Repeater. Warmup is all-zero (the
    parser's sliding 41-bit window would otherwise commit to
    a spurious Sync=1 alignment from alternating-pattern
    warmup).
  - Round-trip modulator tests cover the chain end-to-end
    against the FM discriminator + Kaiser LPF (100 random
    bits, every bit recovered exactly past the LPF group-
    delay warmup), phase continuity across chunked Modulate
    calls, constant envelope, and Reset semantics. 30-run
    flakiness check clean.

  Punch-list status: **all 9 protocols + 4 modulator
  primitives shipped** — P25 P1 / NXDN / DMR Tier III / dPMR
  Mode 3 / EDACS / Motorola Type II / TETRA / P25 Phase 2 /
  MPT 1327 / LTR. The C4FM modulator (PR #148) drove the
  4-FSK family; GFSK (PR #152) drove EDACS + Motorola;
  π/4-DQPSK (PR #154) drove TETRA + P25 P2; FFSK (PR #156)
  drove MPT 1327; sub-audible NRZ (this PR) drove LTR.
- **FFSK modulator + `make integration-cc-mpt1327`.**
  First integration test to exercise audio-band FSK
  modulation. Lights up MPT 1327 end-to-end through the
  daemon's mock-SDR + production-receiver chain.
  - `internal/dsp/demod/ffsk_modulator.go` ships the TX
    counterpart to the existing FFSK tone discriminator:
    bit → tone select (mark / space) → continuous-phase
    audio sinusoid at the tone frequency → FM modulator
    (phase accumulator integrates audio) → IQ.
    `FFSKModulator` carries both the audio-phase and the
    RF-phase accumulators across `Modulate` calls so long
    streams stay phase-continuous; `ModulateFFSK` is the
    single-shot convenience.
  - `mpt1327.LockState` now implements
    `trunking.LockedPayload` (`LockedFrequencyHz` +
    `LockedNAC`). **Seventh protocol with the same latent
    bug fixed** (NXDN / dPMR / EDACS / Motorola / TETRA /
    P25 Phase 2 / MPT 1327). MPT 1327 doesn't have a
    P25-style NAC; the AHYC SystemID is the closest
    per-cell identifier and gets plumbed into the NAC slot.
  - The integration test synthesizes 100 back-to-back
    BCH(63, 38)-encoded ALH (Aloha) codewords (the
    canonical "lock me" address codeword), modulates via
    the new FFSK primitive at the standard CCIR FFSK
    tone pair (1200 Hz mark / 1800 Hz space) at 1200
    baud, and asserts the daemon recovers the lock via
    `mpt1327_bch_mode: on`. 30-run flakiness check clean.
  - Round-trip modulator tests cover the FFSK chain
    against the existing FM discriminator + FFSK tone
    discriminator (200 random bits, every bit recovered
    exactly past the LPF group-delay warmup), phase
    continuity across chunked Modulate calls,
    constant-envelope (|IQ| = 1 ± 1e-6), and Reset
    semantics.
- **`make integration-cc-p25p2` — P25 Phase 2 end-to-end
  lights-up check.** Second protocol to use the
  π/4-DQPSK modulator shipped in PR #154; reuses the
  primitive with `rotation = π/8` to synthesize H-DQPSK
  (the π/8-shifted variant P25 Phase 2 specifies).
  - 6000 sym/s, α = 0.20 RRC, sps = 8 at the test's 48 kHz
    sample rate — different from TETRA's 18000 sym/s /
    α = 0.35 / sps = 4 path, but the modulator's rotation
    + sps + α parameters cover both cleanly.
  - `p25phase2.LockState` now implements
    `trunking.LockedPayload`. Sixth protocol with the same
    latent-bug class fixed (NXDN / dPMR / EDACS / Motorola
    / TETRA / P25 Phase 2). P25 Phase 2's MAC PDU header
    doesn't carry a NAC equivalent — the NAC lives one
    layer up in the Phase 2 superframe — so `LockedNAC`
    returns 0; the supervisor uses it only as a cache key
    on retune, so 0 is harmless.
  - The P25 Phase 2 pipeline factory tunes its Gardner
    `ClockGain` to 0.005 (same value as the TETRA factory
    in PR #154; the 0.03 default over-corrects on clean
    H-DQPSK signals and slips).
  - Integration test synthesizes 80 back-to-back
    `OpMACPTT` MAC PDUs (the canonical "lock me" non-idle
    PDU) through the production trellis encoder
    (`framing.EncodeP25Trellis`), wraps each in a 20-dibit
    outbound sync, and asserts the daemon recovers the
    lock via `p25_phase2_trellis_mode: on`. 30-run
    flakiness check clean on first try.
- **π/4-DQPSK modulator + `make integration-cc-tetra`.**
  First integration test to exercise a non-FSK modulation
  family, lighting up the full TETRA TMO control-channel
  decode end-to-end against synthesized IQ.
  - `internal/dsp/demod/piover4_dqpsk_modulator.go` ships
    the TX counterpart to the existing `PiOver4DQPSK`
    demodulator: dibit → raw phase delta ∈ {0, π/2, π,
    -π/2} → +rotation per symbol → cumulative phase →
    complex symbol → impulse train × sps → unit-energy RRC
    pulse shape → IQ. The rotation argument selects between
    true π/4-DQPSK (TETRA TMO, rotation = π/4) and
    π/8-shifted H-DQPSK (P25 Phase 2, rotation = π/8).
    `PiOver4DQPSKModulator` carries phase + FIR history
    across `Modulate` calls so long streams can be chunked.
  - `tetra.LockState` now implements
    `trunking.LockedPayload` (`LockedFrequencyHz` +
    `LockedNAC`). Fifth protocol with the same
    latent-bug class fixed on NXDN / dPMR / EDACS / Motorola
    in PRs #149 / #151 / #152 / #153. TETRA doesn't have a
    P25-style NAC; the LocationArea is the closest per-cell
    identifier and gets plumbed into the NAC slot.
  - The TETRA pipeline factory tunes the Gardner clock loop
    down from the 0.03 default to 0.005. At 18000 sym/s
    the standard gain over-corrects on clean signals and
    slips with > 50% dibit errors; 0.005 tracks both clean
    synthesized IQ and noisier on-air captures within the
    loop's lock-acquisition margin. Same pattern as the
    DMR Tier III ClockGain tweak in PR #150.
  - The integration test synthesizes a full §8.3.1 SCH/HD
    burst (38-dibit normal training-sequence sync +
    108-dibit channel-coded SCH/HD carrying an MLE SYSINFO
    PDU with a known LocationArea), modulates via the new
    π/4-DQPSK primitive, and asserts the daemon recovers
    the lock through the production `newTETRAPipeline` with
    `tetra_channel_coding: on` + `tetra_colour_code`
    config.
  - Round-trip modulator tests cover dibit recovery
    through the existing RRC matched filter + DQPSK
    quadrant decoder (200 random dibits, every one
    recovered exactly), phase continuity across chunked
    Modulate calls, and Reset semantics.
  - 30-run integration flakiness check clean.
- **`make integration-cc-motorola` — Motorola Type II
  end-to-end lights-up check.** Second non-C4FM protocol
  to light up through the daemon; reuses the GFSK
  modulator shipped in PR #152 with different per-protocol
  framing (Motorola Type II OSW vs EDACS CCW) and a
  different FEC chain (per-codeword BCH(64, 16, 11)
  wrapping each 16-bit OSW half vs EDACS' single
  BCH(40, 28, 2) over the whole CCW).
  - 3600-baud 2-FSK with BT = 0.5 — the SmartZone
    standard's tighter-bandwidth profile vs EDACS' 0.3.
    Sample rate picked at 97.2 kHz so an integer sps = 27
    matches the receiver's float computation with no
    rounding drift.
  - `motorola.LockState` now implements
    `trunking.LockedPayload` (`LockedFrequencyHz` +
    `LockedNAC`). Same latent-bug class fixed on NXDN /
    dPMR / EDACS in PRs #149 / #151 / #152 — fourth
    protocol with the same shape.
  - The test synthesizes an `OpSystemIDExtended` OSW
    (carrying a SystemID announcement) through
    `framing.BCHEncode64_16` × 2 for the two halves,
    sandwiches it between the 24-bit outbound sync and
    idle padding, and asserts the daemon recovers the
    lock via the `motorola_bch_mode: on` opt-in.
  - 30-run flakiness check clean.
- **GFSK modulator + `make integration-cc-edacs`.** First
  non-C4FM protocol to light up end-to-end through the
  daemon's mock-SDR + production-receiver chain.
  - `internal/dsp/demod/gfsk_modulator.go` ships the
    Gaussian-FSK TX counterpart to the existing GFSK
    demodulator: bit → bipolar symbol → impulse train × sps
    → unit-sum-normalised Gaussian premod filter → FM
    modulator (phase accumulator) → IQ. `GFSKModulator` is
    stateful across `Modulate` calls so long streams can be
    chunked; `ModulateGFSK` is the single-shot convenience.
  - The receiver-side slicer at zero threshold needs no
    `DeviationHz` calibration knob — GFSK is symmetric
    around DC, and the receiver's existing zero-threshold
    slicer Just Works once the modulator produces a real
    Gaussian-shaped FSK signal.
  - `edacs.LockState` now implements `trunking.LockedPayload`
    (`LockedFrequencyHz` + `LockedNAC`). Same latent-bug
    class as the NXDN / dPMR fixes in PRs #149 / #151 —
    without these methods, the supervisor's type-assertion
    on cc.locked silently drops the event and
    `/api/v1/scanner` never surfaces `state=locked` for
    EDACS systems.
  - `make integration-cc-edacs` boots the daemon with
    synthesized 9600-baud GFSK IQ (BT = 0.3, ±2.4 kHz peak
    deviation) carrying a 24-bit outbound sync + 40-bit
    BCH(40, 28, 2)-encoded CmdSystemID CCW. The test
    enables `edacs_bch_mode: on` so the FEC layer is
    exercised end-to-end on the recovered bits.
  - Round-trip tests cover the modulator against the
    existing GFSK demodulator (200 random bits, every
    bit recovered exactly), phase continuity across chunked
    calls, constant-envelope (|IQ| = 1 ± 1e-6), and Reset
    semantics. 30-run integration flakiness check clean.
- **`make integration-cc-dpmr` — dPMR Mode 3 end-to-end
  lights-up check.** Fourth per-protocol sibling of
  `integration-cc`. Boots the daemon with a mock SDR
  replaying synthesized dPMR Mode 3 IQ (24-dibit FS3 sync
  + 40-dibit / 80-bit `StandingServiceStatus` CSBK), and
  asserts the production `newDPMRPipeline` + supervisor +
  API + metrics chain recovers the lock.
  - `internal/radio/dpmr/receiver` picks up the same
    `Options.DeviationHz` slicer-calibration knob as the
    P25 P1 / NXDN / DMR receivers (PRs #148 / #149 / #150).
    The ccdecoder's `newDPMRPipeline` passes **900 Hz** —
    half the P25 / DMR / YSF deviation, matching the
    6.25 kHz channel spacing dPMR targets.
  - `dpmr.LockState` now implements
    `trunking.LockedPayload` (`LockedFrequencyHz` +
    `LockedNAC`). dPMR doesn't have a P25-style NAC; the
    low 16 bits of SystemID are the closest per-cell
    identifier and get plumbed into the NAC slot. Same
    latent-bug class as the NXDN fix in PR #149 — without
    these methods, the supervisor's type-assertion on
    cc.locked silently drops the event and
    `/api/v1/scanner` never surfaces `state=locked`.
  - The C4FM modulator from PR #148 handles dPMR's
    half-rate 2400 sym/s modulation directly via the `sps`
    parameter (20 instead of P25/DMR/NXDN's 10 at the same
    sample rate). No DSP changes needed.
  - 30-run flakiness check clean on first try — the lower
    symbol rate + lower deviation gives the MM clock loop
    a comfortable margin without needing the ClockGain
    tweak DMR needed in PR #150.
- **`make integration-cc-dmr` — DMR Tier III end-to-end
  lights-up check.** Third per-protocol sibling of
  `integration-cc`. Boots the daemon with a mock SDR
  replaying a fully-synthesized 132-dibit DMR Tier III
  burst (49-dibit first-half payload + 5-dibit slot-type +
  24-dibit BS-Data sync + 5-dibit slot-type + 49-dibit
  second-half payload, with the payload carrying an Aloha
  CSBK through BPTC(196, 96)), and asserts the production
  `newDMRTier3Pipeline` + supervisor + API + metrics chain
  recovers the lock.
  - `internal/radio/dmr/receiver` picks up the same
    `Options.DeviationHz` slicer-calibration knob shipped
    on the P25 P1 + NXDN receivers (PRs #148 + #149). The
    ccdecoder's `newDMRTier3Pipeline` passes 1944 Hz —
    the ETSI TS 102 361-1 §6.3 spec deviation.
  - The same factory also bumps `ClockGain` down to 0.025
    (from the 0.05 default). DMR's 1944 Hz deviation is
    ~8% larger per-sample phase excursion than P25 P1's
    1800 Hz; the standard MM gain slips on the harder
    symbol transitions inside random BPTC payloads. The
    lower gain tracks cleanly on synthesized IQ and stays
    well within the loop's noise margin for live captures.
  - The DMR Tier III `LockState` already implemented
    `trunking.LockedPayload`, so no per-protocol wiring
    bug surfaced here (unlike NXDN in PR #149).
  - The C4FM modulator from PR #148 handles DMR's 4800-baud
    4-FSK / α = 0.20 modulation identically to P25 P1 + NXDN;
    the only per-protocol differences are the deviation
    (1944 Hz), the burst framing (132-dibit TDMA bursts vs
    P25's continuous stream), and the channel coding
    (BPTC(196, 96) + slot-type Hamming(20, 8) vs P25's
    trellis-encoded TSBK).
  - 30-run flakiness check clean. The flakiness fix was a
    longer (800-dibit) warmup prefix so the lower-gain MM
    loop has time to fully converge before the first burst's
    random payload tests it.
- **`make integration-cc-nxdn` — NXDN end-to-end lights-up
  check.** First sibling target of `integration-cc` covering
  a second protocol end-to-end. Boots the daemon with a mock
  SDR replaying a fully-synthesized NXDN-TS-1-A §4.6 RCCH
  outbound frame (FSW + LICH + 150-dibit CAC carrying a
  SITE_INFO message through the §4.5.1.1 spec FEC chain
  shipped in PR #144), and asserts the production
  `newNXDNPipeline` + `nxdn_viterbi_mode: spec` recover the
  lock and surface it through the bus + supervisor + API +
  metrics.
  - `internal/radio/nxdn/receiver` gains the same
    `Options.DeviationHz` calibration knob the P25 Phase 1
    receiver picked up in PR #148. The ccdecoder's
    `newNXDNPipeline` factory passes the spec 1800 Hz value
    so live captures slice correctly out of the box.
  - `nxdn.LockState` now implements `trunking.LockedPayload`
    (`LockedFrequencyHz` + `LockedNAC` methods). NXDN
    doesn't have a P25-style NAC; the SiteID is the closest
    per-cell identifier and is plumbed into the NAC slot.
    Without this, cc.locked events fired correctly but the
    cchunt supervisor's state machine silently dropped them
    on the type-assertion check and `/api/v1/scanner` never
    surfaced `state=locked` for NXDN systems.
  - The C4FM modulator from PR #148 carries straight over —
    NXDN's 9600-baud 4-FSK / α = 0.20 / 1800 Hz deviation
    matches P25 Phase 1's modulation params exactly. The
    only differences are framing (192-dibit / 80 ms frames,
    8-dibit FSW vs P25's 24-dibit FSW, LICH + CAC vs NID +
    TSBK) and the channel-coding chain above the demod —
    both of which were already wired up by earlier PRs.
  20-run flakiness check clean.
- **C4FM modulator + RRC pulse shaping + receiver-side
  slicer calibration.** Closes the last stub in the
  `make integration-cc` chain. The IQ → dibit demodulation
  step is now exercised end-to-end against real synthesized
  IQ (no factory stub, no dibit injection).
  - `internal/dsp/demod/c4fm_modulator.go` implements the
    full TX chain: dibit → ±1/±3 symbol → impulse train ×
    sps → RRC pulse-shape filter (unit-energy, matches the
    receiver's RRC matched filter) → FM modulator (phase
    accumulator) → IQ. `C4FMModulator` is stateful across
    Modulate calls so long streams can be chunked; the
    `ModulateC4FM` convenience wraps a single-shot call.
  - `internal/radio/p25/phase1/receiver` gains
    `Options.DeviationHz` — when set the slicer thresholds
    are calibrated against the FM-discriminator output
    level (`2π · DeviationHz / SampleRateHz` at symbol ±3)
    instead of the legacy hardcoded `slicerScale = 1.0`.
    The default (no DeviationHz) preserves the existing
    fixture behaviour for back-compat. The ccdecoder's
    `newP25Phase1Pipeline` factory hardcodes 1800 Hz per
    TIA-102.BAAA-A so live captures slice correctly out of
    the box; a future revision can plumb this through
    per-system YAML if non-standard deviation comes up.
  - `cmd/gophertrunk/integration_cc_test.go` is rewritten
    to feed real C4FM-modulated IQ through the production
    `newP25Phase1Pipeline` instead of stubbing the factory.
    The dibit stream is unchanged (FSW + NID + trellis-
    encoded TSBK), but it's now passed through
    `demod.ModulateC4FM` → u8-IQ file → mock SDR →
    `phase1/receiver` → `phase1.ControlChannel.Process` →
    `cc.locked`. 20-run flakiness check clean.
  Tests cover the modulator round-trip against the
  receiver chain (200 random dibits with every symbol
  level represented, all recover correctly), phase
  continuity across chunked Modulate calls, constant-
  envelope (|IQ| = 1 ± 1e-6) sanity, and the
  dibit→symbol mapping pinned as the inverse of
  `phase1.SymbolToDibit`. The earlier
  `ccdecoder.SetTestFactory` test hook stays exported for
  any future protocol-pipeline integration tests that need
  to inject behaviour above the demod.
- **`make integration-cc` — the "lights up live trunked
  reception" milestone.** Closes Workstream A of the
  original plan. The new target boots the wired daemon
  (mock SDR + cchunt supervisor + ccdecoder + API +
  metrics) and asserts the full chain above the IQ → dibit
  demod recovers a P25 Phase 1 lock end-to-end:
    - daemon construction
    - cchunt supervisor publishing `KindHuntProgress`
    - ccdecoder factory dispatch + pipeline construction
    - `pipeline.Process` invoked on every IQ chunk
    - `phase1.ControlChannel.Process` driving the state
      machine from FSW + NID + TSBK dibit fixtures
    - state machine emitting `cc.locked` on the bus
    - supervisor consuming `cc.locked` → `state=locked`
    - `/api/v1/scanner` reflecting the lock
    - `gophertrunk_control_channel_locked{system=…}` = 1
    - `gophertrunk_events_total{kind="cc.locked"}` = 1
  The one chain step the test stubs is C4FM IQ→dibit
  demodulation (RRC pulse shaping + continuous-phase
  integration are a non-trivial DSP layer in their own
  right). The receiver layer is covered by
  `internal/radio/p25/phase1/receiver`'s unit tests; this
  PR validates everything *above* it.

  Plumbing changes:
    - `ccdecoder.SetTestFactory` is a new exported
      tests-only hook that replaces the registered pipeline
      factory for a single protocol and returns a restore
      function. Production code must not call it.
    - `ccdecoder.Decoder` now subscribes to the events bus
      at `New` time rather than inside `Run`. That removes
      a race where the cchunt supervisor could publish
      `KindHuntProgress` before the decoder's subscription
      landed, causing the first lock attempt to silently
      miss the connector and the test to fail intermittently.
      The change also makes the production daemon's startup
      deterministic — no more "first hunt round drops on
      the floor" timing dependency.

  A future PR can land a proper C4FM modulator + RRC
  shaping primitive in `internal/dsp/`, swap the factory
  stub for real synthesized IQ, and exercise the demod
  layer in the same integration test.
- **Motorola Type II BCH(64, 16, 11) wired through the
  connector.** Closes the last unfinished FEC opt-in in the
  TETRA / LTR / P25 P2 / NXDN / EDACS / MPT 1327 / Motorola
  family. The BCH layer existed on `motorola.ControlChannel`
  for a while (`SetBCHMode(BCHOn)` reads two 64-bit codewords
  after sync, decodes each via `framing.BCHDecode64_16`, and
  reassembles the 32-bit OSW from the two recovered 16-bit
  halves with single- through 11-bit-error correction per
  codeword); this PR threads it through the same per-system
  YAML pipeline every other protocol uses:
    - `trunking.System` gains `MotorolaBCHMode string`;
      `config.SystemConfig` exposes it as `motorola_bch_mode`
      (`""` / `"off"` / `"on"`).
    - `motorola.ParseBCHMode` + `motorola.ControlChannel.BCHMode()`
      mirror the accessors on every other FEC-opt-in protocol.
    - `newMotorolaPipeline` calls `SetBCHMode` before any
      sample flows; empty string preserves the legacy 32-bit
      raw-OSW path for synthesized-fixture tests.
    - `api.SystemDTO` + `client.SystemDTO` carry the field as
      `omitempty` JSON; the TUI **Settings** panel renders a
      `bch: on` / `bch: off` row for Motorola systems.
    - README's FEC opt-ins table gains a Motorola row.
  With this PR, every protocol whose ControlChannel exposes a
  tunable on-air FEC layer is now connector-configurable from
  per-system YAML.
- **Reference spec PDFs consolidated under `docs/specs/`.** The
  NXDN-TS-1-A and ETSI EN 300 392-2 PDFs that drive the
  on-air FEC implementations were previously sitting at the
  repo root with vendor-supplied filenames; the M/A-COM
  LBI-38463C "EDACS System Manager Supervisor's Guide"
  uploaded as a candidate EDACS air-interface reference was
  only in the chat. All three now live under
  [`docs/specs/`](docs/specs/) with normalised filenames
  (`nxdn-ts-1-a-v1.3.pdf`, `etsi-en-300-392-2-v3.8.1.pdf`,
  `lbi-38463c-edacs-system-manager.pdf`) and a
  [`docs/specs/README.md`](docs/specs/README.md) that maps
  each PDF to the code paths it backs (NXDN → §4.5 channel
  coding; TETRA → §8.2/§8.3.1 chain) and explains why the
  LBI is a **negative reference** — it documents the
  system-admin workstation UI, not the air interface, so
  future readers looking for an EDACS spec know to skip it
  and pursue LBI-39031 / LBI-39154 / LBI-38894 instead.
  `git mv` preserves history for the two previously-tracked
  PDFs.
- **EDACS FEC documentation correction.** Earlier package
  docstrings + README bullets called out an "interleaved
  Reed-Solomon-derived FEC layer above the BCH" on the
  EDACS CCW as missing / a future PR. Per the canonical
  open reference (`lwvmobile/edacs-fm`) and a careful read
  of the existing `internal/radio/framing/bch_edacs.go`
  implementation, **no such outer layer exists in Standard
  EDACS** — BCH(40, 28, 2) per CCW is the only on-wire FEC,
  and it's already shipping behind `edacs_bch_mode: on`.
  Each affected docstring is updated to say so explicitly,
  and the historical "Recently shipped" entries that named
  the imaginary RS layer as a follow-up gain a corrective
  footnote. No code logic changes — only documentation.
- **NXDN CAC spec-correct interleave + puncture per
  NXDN-TS-1-A rev 1.3 §4.5.1.1.** Closes the "blocked on
  spec data" gap on the previous round — the user uploaded
  the NXDN-TS-1-A spec (now in
  [`docs/specs/nxdn-ts-1-a-v1.3.pdf`](docs/specs/nxdn-ts-1-a-v1.3.pdf))
  and the full outbound CAC channel coding chain landed
  end-to-end.
  - `internal/radio/nxdn/cac_channel.go` adds `EncodeCACChannel`
    + `DecodeCACChannel` implementing the spec's six-stage
    chain: 155 info bits (8 SR + 144 L3 Data + 3 Null) ‖
    16-bit CRC-CCITT (poly `0x1021`, init `0xFFFF`, no XOR,
    evaluated bit-level since 155 isn't byte-aligned) ‖
    4 zero tail bits → K=5 R=½ convolutional encode (350
    bits) → puncture matrix `1111111 / 1011101` drops 50
    pre-puncture positions (300 bits) → 25×12 block
    interleaver (write rows, read columns) → 300 channel
    bits = 150 dibits on air.
  - The puncture positions are derived from the spec's
    matrix at package-init time so a future spec revision
    can patch the matrix in one place. An `init()` invariant
    panics if the matrix, encoder length, or channel-bit
    arithmetic ever drift apart.
  - `ViterbiMode` gains a new `ViterbiSpec` value; the
    Process adapter under `ViterbiSpec` slices 158 post-sync
    dibits (8 LICH + 150 CAC) per the §4.6 RCCH outbound
    layout (FSW + LICH + CAC + E + Post = 384 bits / 192
    dibits), runs the full decode chain, and forwards the
    recovered L3 prefix into the existing `ParseCAC`. The
    spec's outer CRC has already validated the 155-bit info
    block, so the inner-CRC sentinel `ParseCAC` enforces is
    re-synthesized locally over the recovered L3 prefix.
  - `ParseViterbiMode` recognises `"spec"` (case-insensitive,
    whitespace tolerated) so the existing `nxdn_viterbi_mode`
    YAML key + `ccdecoder` connector + TUI Settings panel
    all light up without further plumbing.
  - The legacy `ViterbiOn` path (8 LICH + 32 SACCH + 92
    encoded CAC dibits) is preserved for back-compat with
    the older MMDVMHost / DSDcc fixtures; existing tests
    keep passing.
  Tests cover the framing primitives (round-trip across four
  seeds, single-bit error correction, heavy-corruption CRC
  catch, wrong-size rejection, puncture-matrix algebra,
  interleaver bijection, byte-aligned CRC sanity against the
  existing `framing.CRCCCITT`) plus the Process integration
  (spec-encoded SITE_INFO recovers `KindCCLocked` with the
  expected SiteID / SystemID; heavily-corrupted spec frames
  drop silently).
- **TUI Settings panel + README FEC opt-ins reference.** The
  11th TUI panel (`Tab` past Scanner) renders each configured
  system with a one-line summary of its FEC opt-in state across
  every protocol that has a public-spec FEC chain — TETRA channel
  coding, LTR FCS + Manchester, P25 Phase 2 trellis, NXDN
  Viterbi, EDACS + MPT 1327 BCH. The panel reads the new opt-in
  fields off `/api/v1/systems`' per-system DTO; the API
  `SystemDTO` was extended to expose every opt-in flag as an
  `omitempty` JSON value, and the client mirror picks them up
  without further plumbing.

  The panel is read-only; the bottom-line hint says "Edit
  config.yaml + restart daemon to change", which matches the
  existing wiring (opt-ins flow `SystemConfig` →
  `trunking.System` → `ccdecoder.PipelineFactory` at construction
  / on each `HuntProgress` retune). Runtime mutation is a future
  follow-up that requires a PATCH endpoint + daemon-side
  reconfig of active pipelines.

  README gained an "FEC opt-ins" section with a table covering
  every YAML key, its default behaviour, and what the on-path
  unlocks. Each protocol's `ControlChannel` also picked up
  matching getters (`tetra.ChannelCoding()` / `ExpectedChannel()`
  / `ColourCode()`, `ltr.FCSMode()` / `ManchesterMode()`,
  `p25phase2.TrellisMode()`, `nxdn.ViterbiMode()`,
  `edacs.BCHMode()`, `mpt1327.BCHMode()`) — the TUI uses them
  indirectly through the DTO; tests + observability code use
  them directly.
- **`ccdecoder` connector threads the remaining per-protocol
  FEC opt-ins from per-system config.** Closes out the
  connector-side FEC wiring for every protocol whose
  ControlChannel exposes a tunable on-air FEC layer. Same
  pattern as PRs #141 (TETRA channel coding) and #142 (LTR
  FCS + Manchester) — operators set one YAML key per
  protocol and the matching pipeline factory turns the FEC
  layer on automatically:
    - `p25_phase2_trellis_mode: on` → `SetTrellisMode(TrellisOn)`
      on the 4-state ½-rate trellis decoder over P25 Phase 2
      MAC PDUs (146 channel dibits → 72 info dibits per
      TIA-102.AABF).
    - `nxdn_viterbi_mode: on` → `SetViterbiMode(ViterbiOn)`
      on the K=5 ½-rate Viterbi decoder over the NXDN CAC
      region (92 dibits → 88 info bits + 4 tail zeros per
      MMDVMHost's NXDNConvolution).
    - `edacs_bch_mode: on` → `SetBCHMode(BCHOn)` on the
      BCH(40, 28, 2) decoder over the EDACS CCW (generator
      0x1539, single/double-bit correction).
    - `mpt1327_bch_mode: on` → `SetBCHMode(BCHOn)` on the
      BCH(63, 38) decoder over the MPT 1327 codeword
      (64-bit on-wire → 38 info bits + 26 parity).
  Each protocol also gains a `ParseXxxMode` helper +
  `XxxMode()` accessor mirroring the TETRA / LTR pattern
  shipped in PR #141 / #142 so tests + observability code
  can introspect configured state. Empty strings preserve
  the legacy raw-bit path across all four protocols so
  existing synthesized-fixture tests stay green. Unknown
  values warn-log and fall back to the off default rather
  than failing the retune.

  With this PR, the only connector wiring that remains is
  protocol-by-protocol on-air interleaver / puncture layers
  for protocols whose public specs don't fully document
  them. NXDN CAC interleave / puncture landed as a separate
  follow-up (see the NXDN CAC entry above). EDACS Standard
  has no outer FEC layer above the BCH — earlier README
  claims of an "interleaved Reed-Solomon-derived FEC layer"
  on the CCW were a documentation error; per the canonical
  open reference (`lwvmobile/edacs-fm`) the BCH(40, 28, 2)
  is the only on-wire FEC, and that path already ships.
- **`ccdecoder` connector threads LTR FCS + Manchester modes
  from per-system config.** Same pattern as the TETRA wiring in
  PR #141 — operators set `ltr_fcs_mode` + `ltr_manchester_mode`
  once in `config.yaml` and the `newLTRPipeline` factory calls
  `ltr.ControlChannel.SetFCSMode` / `SetManchesterMode` before
  any sample flows. Both `ltr.ControlChannel` primitives have
  shipped for a while (CRC-7 FCS check against sdrtrunk's
  CRCLTR.java layout, Manchester decode with strict / soft
  variants); this PR flips them on under config control.
  - `trunking.System` gains `LTRFCSMode string` + `LTRManchesterMode
    string`; `config.SystemConfig` exposes them as
    `ltr_fcs_mode` (recognises `"off"` / `"on"`) +
    `ltr_manchester_mode` (recognises `"off"` / `"nrz"` /
    `"strict"` / `"soft"`, all case-insensitive with whitespace
    tolerated).
  - `ltr.ParseFCSMode` + `ltr.ParseManchesterMode` map the
    YAML string into the typed mode; unknown values warn-log
    and fall back to the legacy off / NRZ default rather than
    failing the retune.
  - `ltr.ControlChannel.FCSMode` + `ManchesterMode` accessors
    mirror the TETRA pattern so tests + observability code can
    introspect configured state without poking at unexported
    fields.
  - Empty strings preserve the legacy `FCSOff` + `ManchesterOff`
    raw-NRZ path so existing synthesized-fixture tests stay
    green. Live captures of sub-audible LTR signaling typically
    need `ltr_manchester_mode: soft` + `ltr_fcs_mode: on` to
    pass the CRC.
  Tests cover the config-string parsers across every recognised
  value (plus a misconfigured-input case), the factory applying
  both modes when the System carries non-empty strings, and the
  factory preserving the legacy modes when both strings are
  empty. The connector now configures every protocol whose
  control-channel state machine has a tunable on-air FEC layer
  (TETRA channel coding, LTR FCS + Manchester); per-protocol
  FEC wiring for NXDN CAC / EDACS CCW / P25 Phase 2 trellis
  remains the next code work, gated on the public-references
  question (NXDN CAC interleave / puncture isn't documented
  in the public spec).
- **`ccdecoder` connector threads TETRA channel coding from
  per-system config.** Closes the last gap between the
  daemon's YAML and the §8.3.1 type-5 → type-1 decoder
  shipped in PR #140 — operators set the cell's extended
  colour code + signaling channel once in `config.yaml` and
  the `newTETRAPipeline` factory flips
  `tetra.ControlChannel.SetChannelCoding(ChannelCodingOn)`
  automatically on every retune.
  - `trunking.System` gains `TETRAColourCode uint32` (low
    30 bits of the §8.2.5 extended colour code, bits 30..31
    silently ignored) and `TETRAChannel string` (the
    config-side name for the logical channel that lives in
    each burst window).
  - `config.SystemConfig` exposes those as `tetra_colour_code`
    + `tetra_channel` YAML keys; `cmd/gophertrunk/daemon.go`
    forwards them into `trunking.System` on construction.
  - `ccdecoder.PipelineOptions` carries the full
    `trunking.System` so per-protocol factories can read
    protocol-specific config without a new field per protocol.
  - `tetra.ParseChannelType` maps the YAML string
    (`"sch/hd" | "sch/f" | "sch/hu" | "bsch" | "aach"`,
    case-insensitive, `"/"` and `"_"` both accepted, empty
    defaults to `sch/hd`) into a `tetra.ChannelType`;
    unknown values fall back to SCH/HD with a warn-level
    log entry.
  - `tetra.ControlChannel.ChannelCoding` / `ExpectedChannel`
    / `ColourCode` accessors let tests + observability code
    introspect the configured state without poking at
    unexported fields.
  - Zero `TETRAColourCode` preserves the legacy
    `ChannelCodingOff` raw-dibit path so existing
    synthesized-fixture tests stay green.
  Tests cover the config-string → ChannelType parser across
  every recognised value (plus a misconfigured-input
  warning case), the factory turning channel coding on
  with the right colour code + channel under a populated
  System, and the factory leaving channel coding off when
  the colour code is left at the zero default. The remaining
  work toward "lights up live trunked reception" is now
  protocol-by-protocol FEC wiring across the other 9
  protocols, not connector plumbing.
- **TETRA `SetChannelCoding(ChannelCodingOn)` opt-in wires
  per-channel FEC decode into `Process`.** Lights up the
  full ETSI EN 300 392-2 §8.3.1 type-5 → type-1 chain
  (descramble + deinterleave + depuncture + Viterbi +
  CRC-16 verify + tail strip) on the `tetra.ControlChannel`
  Process adapter so live IQ captures — not just synthesized
  type-1 fixtures — can drive `cc.locked` / Grant events.
  New API mirrors the BCH wirings on MPT 1327 / EDACS /
  Motorola:
    - `SetChannelCoding(ChannelCodingOff | ChannelCodingOn)` —
      default off (legacy 48-dibit raw path); on enables the
      full FEC chain.
    - `SetExpectedChannel(ChannelSCHHD | ChannelSCHF |
      ChannelSCHHU | ChannelBSCH | ChannelAACH)` — picks
      which logical channel lives in each burst window
      under the on path. Default `ChannelSCHHD`.
    - `SetColourCode(uint32)` — 30-bit extended colour code
      seeding the scrambler (low 30 bits; masked to
      `0x3FFFFFFF`). Ignored by BSCH per §8.2.5.2.
  Under `ChannelCodingOn` the adapter slices the
  channel-appropriate dibit window (108 for SCH/HD, 216
  for SCH/F, 84 for SCH/HU, 60 for BSCH, 15 for AACH),
  routes through the matching `DecodeSCHHD` / `DecodeSCHF`
  / `DecodeSCHHU` / `DecodeBSCH` / `DecodeAACH` helper
  shipped in PR #139, and silently drops frames whose
  CRC fails. Tests round-trip a real `MLE SYSINFO` PDU
  through SCH/HD → `KindCCLocked`, a `CMCE D-CONNECT` PDU
  through SCH/F → `KindGrant`, plus heavy-corruption
  rejection (30 adjacent bit flips) and wrong-colour-code
  rejection. Wiring this into the `ccdecoder` connector
  so per-system config (colour code, expected channel)
  flows from `trunking.System` into the live decoder is
  the next PR.
- **TETRA per-channel encode/decode helpers in `tetra/`.**
  Composes the framing primitives shipped in PRs #137 and
  #138 (RCPC + (30,14) RM + (K,a) block interleaver +
  scrambler + the existing CRC-16 CCITT) into the full
  type-1 → type-5 encode chain and its inverse per ETSI
  EN 300 392-2 §8.3.1 for every standard π/4-DQPSK signaling
  channel:
    - `EncodeSCHHD` / `DecodeSCHHD` — 124 ↔ 216 bits
      (§8.3.1.4.1, also covers BNCH + STCH)
    - `EncodeSCHF` / `DecodeSCHF` — 268 ↔ 432 bits
      (§8.3.1.4.5)
    - `EncodeSCHHU` / `DecodeSCHHU` — 92 ↔ 168 bits
      (§8.3.1.4.3)
    - `EncodeBSCH` / `DecodeBSCH` — 60 ↔ 120 bits, colour
      code fixed at 0 per §8.2.5.2 (§8.3.1.2)
    - `EncodeAACH` / `DecodeAACH` — 14 ↔ 30 bits, simpler
      chain (RM + scramble only, no RCPC or interleave per
      §8.3.1.1)
  Tests round-trip every channel cleanly across multiple
  colour codes, confirm CRC-fail detection on heavily-
  corrupted streams, single-bit-error correction by the
  Viterbi inner decoder under R=2/3 puncturing, wrong-
  colour-code failure, and wrong-input-size rejection. The
  CRC-16 used in §8.2.3.3 is the spec's `(K1+16, K1)` block
  code — equivalent to CRC-CCITT with `init = 0xFFFF`,
  `final XOR = 0xFFFF`, processed bit-level for the
  non-byte-aligned K1 values TETRA uses. Wiring these
  helpers into `tetra.ControlChannel.Process` (with the
  burst-position discrimination from EN 300 392-2 §9 to
  pick which channel decode runs per slot) is the next
  PR.
- **TETRA scrambler + (K, a) block-interleaver primitives in
  framing/.** Closes the remaining framing-layer gap before
  the full TETRA channel-decode chain can be wired together.
  - `framing/scramble_tetra.go` — 32-tap LFSR scrambler per
    ETSI EN 300 392-2 §8.2.5 with connection polynomial
    `c(x) = 1 + X + X² + X⁴ + X⁵ + X⁷ + X⁸ + X¹⁰ + X¹¹ +
    X¹² + X¹⁶ + X²² + X²³ + X²⁶ + X³²` (tap mask 0x82608EDB).
    Seeded by the 30-bit extended colour code (set 0 for
    BSCH / BSCH-Q per §8.2.5.2). Single XOR is symmetric so
    `ScrambleTetra` and `DescrambleTetra` are the same
    operation aliased for call-site readability.
  - `framing/interleave_tetra.go` — `(K, a)` block
    interleaver per §8.2.4.1 with the formula
    `b₄(k) = b₃(i)` where `k = 1 + ((a × i) mod K)`.
    Per-channel constants (`InterleaveK*`, `InterleaveA*`)
    cover BSCH (120, 11), SCH/HD/BNCH/STCH (216, 101),
    SCH/HU (168, 13), SCH/F (432, 103) per §8.3.1.
  Together with the K=5 R=1/4 RCPC mother code + four
  puncturing schemes (PR #137) and the existing CRC-16
  CCITT helper, this completes the framing primitives
  needed for end-to-end π/4-DQPSK signaling-channel
  decode. Tests cover symmetric XOR for the scrambler
  across 4 colour-code values, BSCH initial-state output
  prediction, sequence-balance entropy sanity, 64 random
  round-trips, and per-channel interleaver round-trip /
  permutation / spec-formula checks for all four
  (K, a) constants. Wiring all the primitives together
  into `tetra.ControlChannel.Process` is the next PR.
- **TETRA signaling-channel RCPC + (30,14) RM primitives in
  framing/.** Adds the K=5 R=1/4 16-state convolutional mother
  code and the four puncturing schemes TETRA uses on every
  π/4-DQPSK signaling channel (BSCH, SCH/HD, BNCH, STCH,
  SCH/HU, SCH/F), plus the shortened (30,14) Reed-Muller block
  code used by AACH. Per ETSI EN 300 392-2 §8.2.3.1 / .2 —
  distinct from the K=5 R=1/3 speech-traffic-channel code in
  PR #135 (EN 300 395-2 §5.4.3): same 16-state structure but
  four generator polynomials and a different puncturing
  table family. Generator polynomials: `G₁(D) = 1+D+D⁴`,
  `G₂(D) = 1+D²+D³+D⁴`, `G₃(D) = 1+D+D²+D⁴`, `G₄(D) =
  1+D+D³+D⁴`. Puncturing schemes shipped: rate-2/3 (P=(1,2,5),
  used by all standard signaling channels), rate-1/3
  (stronger protection, P=(1,2,3,5,6,7)), plus rate-292/432
  and rate-148/432 (special long-block patterns with index-
  shift helpers). The (30,14) RM code uses the spec's
  14×16 parity matrix from §8.2.3.2 and is systematic in
  the first 14 bits. Tests cover round-trip on clean
  channels for both rates 2/3 and 1/3, single-bit error
  correction at the mother-code and punctured layers,
  encoder impulse-response sanity against the four
  generator polynomials, all 30 single-bit error positions
  on the RM code, parity-matrix-row consistency, and
  index-shift monotonicity for the special rates. The
  TETRA `ControlChannel` adapter wiring (depuncture +
  Viterbi + CRC-16 strip → ParsePDU per channel type) is
  the follow-up PR.
- **MPT 1327 Op field extension.** Adds the spec's 10-bit Op
  field (between Ident and Function) to `mpt1327.Codeword`,
  closing the documented follow-up from PR #129. New 48-bit
  helpers — `AssembleCodeword48` / `ParseCodeword48` /
  `CodewordFromBits48` / `CodewordBits48` — operate on the
  full information set (Type + Prefix + Ident + Op +
  Function = 48 bits, MSB-first per field). The legacy
  38-bit `AssembleCodeword` / `ParseCodeword` /
  `CodewordFromBits` / `CodewordBits` stay back-compat: they
  silently drop Op on encode and leave it at zero on
  decode, so existing fixtures + tests that pre-date the Op
  field keep working byte-identically. The BCH wiring in
  `process.go` now routes through `CodewordFromBits48` so
  under `SetBCHMode(BCHOn)` the recovered codeword carries
  all 48 information bits, surfacing the full spec layout
  to downstream `Ingest`. Tests cover 48-bit round-trip
  preserving Op, legacy 38-bit round-trip dropping Op,
  reject-wrong-length error paths, the 10-bit Op mask
  preventing overflow into Ident, and a BCHOn end-to-end
  round-trip that verifies a non-zero Op survives encode →
  BCH-protect → decode → CCW recovery.
- **ClockGardner wired into the ccdecoder connector for the
  π/4-DQPSK pipelines.** The `newP25Phase2Pipeline` and
  `newTETRAPipeline` factories in
  `internal/scanner/ccdecoder/pipelines.go` now pass
  `ClockMode: ClockGardner` into the receiver constructor, so
  every live SDR retune through the connector runs symbol
  recovery via the Gardner timing-recovery loop landed in
  PR #128 + threaded into the receivers in PR #130. The
  `ClockNaive` path stays available for in-package
  receiver-level tests that synthesize sample-aligned IQ
  fixtures. Other pipelines (P25 Phase 1, DMR, NXDN, EDACS,
  etc.) are unaffected — they use 4FSK / GFSK / FFSK demods
  where the existing Mueller-Müller path already handles
  symbol-time recovery. Existing factory tests continue to
  pass; the change is purely additive at the connector
  layer.
- **TETRA RCPC primitive in framing/.** New shared
  `framing/rcpc_tetra.go` adds the K=5 ½-rate→1/3-rate
  16-state convolutional mother code plus puncturing /
  depuncturing helpers per ETSI EN 300 395-2 §5.4.3. Generator
  polynomials `G₁(D) = 1 + D + D² + D³ + D⁴` (= 0x1F),
  `G₂(D) = 1 + D + D³ + D⁴` (= 0x1B), `G₃(D) = 1 + D² + D⁴`
  (= 0x15) — distinct from the K=5 R=½ code in
  `viterbi_k5.go` (NXDN / YSF), so this is a separate
  primitive with the same 16-state structure but three
  outputs per input. Includes spec-verbatim puncturing tables
  for the three rates TETRA's normal + stealing-mode speech
  traffic channels use: rate-8/12 (= 2/3) for class-1 bits
  (P = (1, 2, 4), Period = 6, §5.5.2.1), rate-8/18 for
  class-2 bits in normal traffic (P = (1..5, 7, 8, 10, 11),
  Period = 12, §5.5.2.2), and rate-8/17 for class-2 bits
  under frame-stealing (17-element P, Period = 24,
  §5.6.2.1). The mother-code `DecodeRCPCTetraMother` is a
  16-state hard-decision Viterbi; depunctured positions use
  the same `DepunctureMark` sentinel as the K=5 R=½ code so
  callers can mix the two via a single decoder pattern.
  Tests cover mother-code round-trip + single-bit
  correction, encoder impulse-response sanity against the
  three generator polynomials, round-trips for all three
  puncturing schemes, single-bit-error correction over a
  punctured rate-2/3 channel, and a schedule-sanity check
  asserting the puncturing tables are strictly increasing
  and bounded by their Period. Wiring this primitive into
  the TETRA `ControlChannel.Process` adapter (sliced 432-bit
  type-3 → type-2 stream per §5.5 / §5.6) is the
  documented follow-up.
- **LTR `SetFCSMode(FCSOn)` opt-in.** Wires the
  `framing.CRC7LTR` primitive from PR #131 into the LTR
  `ControlChannel.Ingest` path. Under FCSOn, Ingest computes
  the CRC-7 over a 24-bit message vector derived from Status
  fields (per DSheirer/sdrtrunk's CRCLTR.java layout: 1-bit
  Group / F-bit as sdrtrunk's "Area", then Channel/Home/
  GroupID/Free), compares it to the low 7 bits of
  `Status.FCS`, and drops the frame on mismatch. `ComputeStatusFCS`
  is exported so test fixtures + future encoders can populate
  the trailer correctly. The 5-bit gophertrunk `Status.Area`
  field stays as opaque metadata for the multi-system filter
  (a different layer than the CRC-protected message); under
  this wiring the gophertrunk Group F-bit is the canonical
  sdrtrunk "Area" bit. Tests cover valid CRCs accepted,
  corrupted CRCs dropped, corrupted message fields dropped,
  FCSOff bypass preserved, default mode, and CRC-changes-with-
  the-Group-bit sanity. Doesn't yet resolve the broader
  layout disagreement between sdrtrunk's 7-bit CRC reading and
  gophertrunk's 12-bit `Status.FCS` field (only the low 7 bits
  are CRC-protected in this wiring) — that's a documented
  follow-up.
- **EDACS `SetBCHMode(BCHOn)` opt-in.** Wires the
  `BCHEncodeEDACS` / `BCHDecodeEDACS` framing primitive from
  PR #132 into the EDACS `ControlChannel.Process` adapter via
  `SetBCHMode(BCHOff | BCHOn)`. Same opt-in shape as the
  MPT 1327 wiring (PR #129) and Motorola's pre-existing
  `SetBCHMode`. Under BCHOn the adapter slices 40-bit on-wire
  codewords, runs the BCH(40, 28, 2) validation +
  single/double-bit correction over each slice, then
  re-encodes the corrected 28-bit info into a 40-bit wire
  word that the existing `CCWFromBits` parser interprets.
  Uncorrectable codewords (≥ 3 bit errors in unfavourable
  positions) drop the frame. Under BCHOn the effective CCW
  model carries Command (4) + Status (4) + Address (16) +
  LCN (4 high bits, position 12..15) = 28 info bits; the
  legacy struct's LCN bit 0 and Aux (11 bits) become BCH
  parity, not data. Tests cover BCHOn round-trip (an
  encoded GroupVoiceGrant publishes a Grant with the right
  Address + LCN), single-bit error correction, double-bit
  error correction (BCH(40, 28, 2)'s full t=2 capability),
  triple-bit error rejection, and default-mode regression.
- **EDACS BCH(40, 28, 2) primitive in framing/.** New shared
  `framing/bch_edacs.go` adds `BCHEncodeEDACS` /
  `BCHDecodeEDACS` for the EDACS Standard control-channel word
  check. Parameters confirmed from lwvmobile/edacs-fm's
  `bch3.h` (the most-cited public reference for EDACS channel
  coding): shortened BCH(40, 28, 2) derived from BCH(63, 51, 2)
  over GF(2^6) with primitive polynomial x^6 + x + 1.
  Generator polynomial `g(x) = m₁(x) · m₃(x) = x^12 + x^10 +
  x^8 + x^5 + x^4 + x^3 + 1 = 0x1539`, designed minimum
  distance d = 5, corrects up to t = 2 bit errors per
  codeword. The decoder precomputes a 40-entry single-bit-
  error syndrome table at package init, then handles
  single-bit corrections via direct lookup and double-bit
  corrections by iterating the 780 ordered pairs. Tests cover
  round-trip cleanly across constants + 1024 random info
  values, single-bit correction across all 40 positions,
  double-bit correction across all (40 choose 2) = 780 pairs,
  triple-bit error rejection (> 95% detected / mis-corrected),
  syndrome-table uniqueness + bit-width sanity, and
  encoded-codeword self-syndrome zero check. Not yet wired
  into the EDACS `ControlChannel` adapter; the existing 40-bit
  CCW struct needs cross-checking against this layout —
  documented follow-up.
- **LTR Standard CRC-7 primitive in framing/.** New shared
  `framing/crc_ltr.go` adds `CRC7LTR` / `VerifyCRC7LTR` for
  the LTR Standard message check (polynomial 0xFD, initial
  fill 0x00) per DSheirer/sdrtrunk's `edac/CRCLTR.java`. The
  24-entry syndrome lookup table covers the four fields LTR
  protects (Area, Channel, Home, Group, Free — 24 bits total),
  with direction-aware verification: OSW (outbound) frames
  must match the calculated checksum as-is; ISW (inbound)
  frames carry the bit-inverted checksum. The primitive isn't
  yet wired into the LTR `ControlChannel` adapter because the
  bit layout sdrtrunk documents (1-bit Area, 5-bit Channel)
  disagrees with the GopherTrunk `Status` struct (5-bit Area,
  4-bit Channel) — reconciling the two LTR Standard
  interpretations is the documented follow-up. Tests cover
  zero-message zero-checksum, single-bit syndrome matches,
  256 random-message round-trips, single-bit-error detection
  across all 24 positions, ISW checksum inversion, and
  table-uniqueness / 7-bit-bound sanity.
- **Gardner clock recovery threaded into the P25 Phase 2 +
  TETRA receivers.** Each π/4-DQPSK receiver gains an
  `Options.ClockMode` (`ClockNaive` default, `ClockGardner`
  opt-in) that swaps the naive every-sps-th-sample decimation
  for the `sync.Gardner` timing-recovery loop landed in PR
  #128. The Gardner loop manages its own cross-call tail
  state, so chunked streams converge once rather than per
  chunk; `Reset()` clears the loop state alongside the rest
  of the receiver. The existing test fixtures (which assume
  fixed sample alignment) keep passing under the default
  `ClockNaive`. New tests confirm the Gardner path produces
  valid in-range dibits, the loop is constructed only when
  requested, and `Reset()` restarts the dibit-base counter.
  The ccdecoder connector now wires both pipelines with
  `ClockMode: ClockGardner` so every live SDR retune
  through `newP25Phase2Pipeline` / `newTETRAPipeline` runs
  Gardner symbol recovery automatically.
- **MPT 1327 `SetBCHMode(BCHOn)` opt-in.** Wires the
  `BCHEncodeMPT1327` / `BCHDecodeMPT1327` framing primitive
  into the MPT 1327 `ControlChannel.Process` adapter. When on,
  the adapter slices 64-bit on-wire codewords (instead of the
  default 38-bit pre-stripped info windows), runs BCH(64,48,2)
  decode + single-bit error correction, then extracts the 38
  info bits the existing `Codeword` struct models (Type +
  Prefix + Ident + Function, with the spec's 10-bit Op field
  between Ident and Function dropped — the struct doesn't yet
  model it). The alignment search picks the first 64-bit
  window that BCH-passes, which is much more selective than
  the 38-bit "recognised opcode" search BCHOff uses, so
  live-air captures whose first few codewords carry single-bit
  errors still synchronise. Tests cover BCHOn round-trip (an
  Aloha → GoToChannel stream produces cc.locked + Grant),
  single-bit error correction (one flipped bit per codeword
  still locks), uncorrectable-codeword rejection (two-bit
  flips drop the frame), and default-mode preservation.
- **MPT 1327 BCH(64,48) primitive in framing/.** New shared
  `framing/bch_mpt1327.go` adds `BCHEncodeMPT1327` /
  `BCHDecodeMPT1327` for the 64-bit codeword layout MPT 1327
  uses (48 info bits + 15 BCH check + 1 overall parity bit).
  Polynomial `g(x) = x^15 + x^14 + x^13 + x^11 + x^4 + x^2 + 1`
  (= 0x6815 without the implicit leading x^15) and 0x0001
  initial fill — the parameters DSheirer/sdrtrunk uses in
  `edac/CRCFleetsync.java` for Fleetsync and MPT 1327
  (which share the codeword format). 48-entry syndrome table
  generated at package init from `x^i mod g(x)` for the info
  bits. Single-bit error correction is best-effort: info-bit
  errors (positions 0..47) and parity-bit errors (position 63)
  recover the info field exactly; CRC-bit errors (positions
  48..62) have known syndrome collisions with info bits 0..14
  and are resolved by preferring info-bit correction (garbage
  at the info layer gets rejected by the protocol parser
  anyway). Tests cover round-trip, single-bit detection across
  all 64 positions, exact info-bit recovery for the
  unambiguous half of the position space, random round-trips,
  and a double-bit-error detection sanity check. Wiring this
  primitive into the MPT 1327 adapter via a `SetBCHMode` opt-in
  is the follow-up.
- **Gardner symbol-time recovery for complex IQ.**
  `internal/dsp/sync/gardner.go` adds a non-data-aided
  feedback timing-recovery loop sibling to the existing
  real-valued `MuellerMuller`. Uses the standard Gardner 1986
  detector — `e[n] = Re{(s[n] − s[n−1])* · m[n]}` over the
  symbol-time samples and the midpoint sample between them —
  which converges before the demod has acquired symbol
  polarity, so it works for π/4-DQPSK / QPSK / QAM IQ streams
  where Mueller-Muller would need an upstream rotation pass.
  Cross-call state preserves the timing estimate so chunked
  streams converge once rather than per-chunk. Tests cover
  aligned QPSK recovery, fractional-sample phase-offset
  pull-in, chunked-vs-contiguous symbol agreement, and reset
  semantics. Closes the README's "Symbol-time clock recovery
  on complex IQ" primitive gap; threading it into the
  π/4-DQPSK receivers (P25 Phase 2, TETRA) is the follow-up.
- **P25 Phase 2 4-state ½-rate trellis FEC opt-in over the MAC
  PDU.** Second heavy-FEC PR. `phase2.SetTrellisMode(TrellisOn)`
  switches the `ControlChannel.Process` adapter from "read 72
  raw MAC PDU dibits off the wire" to "collect 146 channel
  dibits + run them through the TIA-102 Annex A 4-state ½-rate
  trellis Viterbi decoder". The trellis tables (16-entry
  constellation table from Annex A Table A.1) are extracted
  into a new shared primitive
  `internal/radio/framing/p25_trellis.go` (`EncodeP25Trellis` /
  `DecodeP25Trellis`) so both Phase 1 (TSBKs, 48 → 98 dibits)
  and Phase 2 (MAC PDUs, 72 → 146 dibits) can drive the same
  code; Phase 1's existing local copy stays in place for
  backward compatibility. Tests cover the framing primitive
  (round-trip + single-dibit-error correction + length-check)
  plus end-to-end Phase 2 paths (KindCCLocked from a
  trellis-encoded `OpMACPTT` PDU; `KindGrant` from a
  trellis-encoded `OpGroupVoiceChannelGrant` PDU). The spec's
  Reed-Solomon outer layer + per-burst block interleaver
  (which wrap around the trellis-coded MAC bits) are
  documented follow-ups; on-air decode of full P25 P2 traffic
  needs both layers and accurate symbol-time recovery
  (Gardner) to land.
- **NXDN K=5 ½-rate Viterbi FEC opt-in for the CAC region of the
  Info field.** First heavy-FEC PR. `nxdn.SetViterbiMode(
  ViterbiOn)` switches the `ControlChannel.Process` adapter
  from "read 44 raw CAC dibits off the wire" to "collect 92
  encoded CAC dibits + run them through the K=5 ½-rate Viterbi
  primitive in `internal/radio/framing/viterbi_k5.go` to
  recover 88 CAC info bits + 4 tail bits". The convolutional
  primitive (constraint length 5, generator pair g1 = 0x19 /
  g2 = 0x17 octal 31/27) is the same one MMDVMHost / DSDcc /
  op25 use across NXDN SACCH and other K=5 open-spec systems;
  this PR wires it into the CAC slot. Tests round-trip
  CAC bytes → `EncodeK5` → 184 channel bits → 92 dibits →
  Process → ParseCAC → cc.locked. The per-protocol interleave
  + puncture inner layer NXDN applies inside the Info field
  isn't reversed yet (the public references don't fully
  document it); ViterbiOn is the bare-bones convolutional
  decode, ViterbiOff (default) preserves the legacy raw-wire
  behaviour for test fixtures and clean synthesized streams.
- **Cross-protocol strict-validation FEC bundle: LTR + dPMR +
  TETRA + P25 Phase 2 `SetStrictValidation(bool)`.** Extends
  the soft-FEC noise filter from the previous PR across the
  remaining four protocols, completing the family of seven.
  Same pattern as the EDACS / Motorola / MPT 1327 bundle: each
  Ingest path now drops frames whose opcode / type / range
  fields fall outside the documented set before the state
  machine acts on them. dPMR rejects CSBKs whose 5-bit
  MessageType is unallocated (per ETSI TS 102 658 §6.5.2);
  TETRA rejects PDUs whose (Discriminator, Type) pair isn't in
  the documented CMCE / MLE set (also drops MM and SDS sub-
  protocols, which the state machine doesn't surface for
  trunking); P25 Phase 2 rejects MAC PDUs whose 8-bit Opcode
  is outside the TIA-102.AABF / BBAB table; LTR rejects Status
  words whose Channel or Home field falls outside the
  documented 1..20 range. Each protocol also gains an
  `IsKnown()` / `IsWellFormed()` method on its enum / status
  type for callers that want to apply the same allow-list
  themselves. Strict validation is now available on every
  protocol with an enumerable opcode space.
- **Cross-protocol strict-validation FEC bundle: EDACS +
  Motorola + MPT 1327 `SetStrictValidation(bool)`.** Each
  adapter gains a soft-FEC noise-reduction mode that rejects
  parsed control-channel frames whose opcode / kind / command
  field falls outside the documented set. Same pattern across
  all three protocols: when on, the Ingest path drops frames
  with unrecognised `Command` / `Opcode` / `Codeword.Kind`
  before the state machine acts on them. Each protocol also
  gains an `IsKnown()` method on its enum type for callers that
  want to apply the same allow-list themselves. Doesn't correct
  bit errors — that's what BCH / RS / Viterbi do per protocol —
  but it cheaply eliminates the largest source of false-positive
  `KindCCLocked` / `KindGrant` events from misaligned codewords
  in environments without per-protocol FEC.
- **Motorola BCH(64,16,11) FEC.** `framing/bch.go` gains
  `BCHEncode64_16` / `BCHDecode64_16` — the existing BCH(63,16,11)
  primitive used by P25 Phase 1 NID, extended with an overall-
  even-parity bit. The Motorola adapter gains `SetBCHMode(BCHOff
  | BCHOn)`; when on, the adapter reads two 64-bit codewords
  (128 channel bits) after each sync, decodes each via the
  framing primitive, and concatenates the recovered 16-bit
  halves into the 32-bit OSW. Uncorrectable codewords (> 11
  errors) drop the frame silently. Tests cover the framing
  primitive (round-trip, single-bit corrections, parity-flip
  detection, > 11-bit rejection) plus an end-to-end Motorola
  Process call decoding a BCH-encoded `OpGroupVoiceChannelGrant`
  OSW.
- **FEC bundle: framing `ManchesterEncode` / `ManchesterDecode` /
  `ManchesterDecodeMajority` helpers + LTR Manchester opt-in +
  NXDN CAC CRC strict-mode.** First FEC implementations PR.
  `framing/manchester.go` adds a generic bi-phase encoder /
  strict decoder / soft (majority-decode) decoder usable by any
  protocol that ships Manchester-encoded bits on the wire. LTR
  gains a `SetManchesterMode(ManchesterStrict | ManchesterSoft |
  ManchesterOff)` config so deployments that use bi-phase
  encoding decode correctly; the default stays NRZ. NXDN's CAC
  CRC-CCITT-16 (already verified inside `ParseCAC`) is now
  enforced by the Process adapter — frames whose CRC fails get
  dropped silently instead of dragging the state machine
  through an Ingest call. Future EDACS / Motorola adapters can
  adopt the Manchester helpers in the same opt-in shape.
- **TETRA TMO `ControlChannel.Process(stream, baseIdx)` adapter +
  ccdecoder factory.** Closes the IQ → CC sync layer for TETRA —
  the last per-protocol adapter from the connector roadmap. The
  receiver's `DibitSink` forwards π/4-DQPSK dibits into
  `tetra.ControlChannel.Process`, which buffers across calls +
  detects the 38-dibit normal training-sequence sync + slices a
  48-dibit PDU (1 header byte + 11 payload bytes = 96 bits) +
  parses it via `ParsePDU` + dispatches through the existing
  `Ingest`. `trunking.Protocol` gains `ProtocolTETRA` (config
  string `"tetra"`). RCPC / RM FEC + interleaving across the
  full TDMA slot are documented follow-ups; until they land the
  adapter works on test fixtures but typically fails to lock on
  captured TETRA traffic. **With this PR every trunked control
  modulation listed in the Features table has an end-to-end
  IQ → CC chain shipping.**
- **P25 Phase 2 `ControlChannel.Process(stream, baseIdx)` adapter +
  ccdecoder factory.** Closes the IQ → CC sync layer for P25
  Phase 2: the receiver's `DibitSink` forwards H-DQPSK dibits
  into `phase2.ControlChannel.Process`, which buffers across
  calls + detects the 20-dibit outbound sync + slices a 72-dibit
  MAC PDU (1 opcode + 17 payload bytes = 144 bits) + parses it
  via `ParseMACPDU` + dispatches through the existing `Ingest`.
  `trunking.Protocol` gains `ProtocolP25Phase2` (config string
  `"p25-phase2"`). Trellis FEC + slot-type extraction across the
  full 180-dibit subframe are documented follow-ups; until they
  land the adapter works on test fixtures but typically fails to
  lock on captured Phase 2 traffic.
- **DMR Tier III `ControlChannel.Process(stream, baseIdx)` adapter +
  ccdecoder factory.** Closes the IQ → CC chain for DMR — the
  most layered protocol in the family. The receiver's `DibitSink`
  forwards C4FM dibits into the adapter, which buffers across
  calls + runs `dmr.SyncDetector` against all 9 ETSI sync words
  in parallel + slices the 132-dibit burst around each match
  (49-dibit first half + 5-dibit slot type before + 24-dibit
  sync + 5-dibit slot type after + 49-dibit second half) +
  parses the slot-type Hamming(20,8) codeword + hands the
  `(Burst, SlotType)` pair to the existing `IngestBurst`. From
  there the dmr/tier3 package's BPTC(196,96) + CSBK CRC chain
  runs end-to-end — no FEC is bypassed for DMR. The adapter
  retains a 163-dibit cross-call buffer so bursts that straddle
  chunk boundaries decode correctly.
- **MPT 1327 `ControlChannel.Process(stream, baseIdx)` adapter +
  ccdecoder factory.** Closes the IQ → CC alignment layer for
  MPT 1327: the receiver's `BitSink` forwards FFSK bits into
  `mpt1327.ControlChannel.Process`, which slides a 38-bit window
  over the stream, commits to the first window that parses as a
  recognised Address codeword (Aloha / AhoyChan / GoToChan /
  Ack / Disconnect / Data / Emergency), follows the alignment
  forward, and auto-unlocks + re-searches after 8 consecutive
  frames whose codeword fails the recognised-codeword check.
  `trunking.Protocol` gains `ProtocolMPT1327` (config string
  `"mpt1327"`). The 64-bit on-air BCH(63,38) FEC + de-
  interleaving are documented follow-ups; until they land the
  adapter works on noise-free test fixtures but typically fails
  to lock on captured MPT 1327 traffic.
- **LTR `ControlChannel.Process(stream, baseIdx)` adapter +
  ccdecoder factory.** Closes the IQ → CC alignment layer for
  LTR: the receiver's `BitSink` forwards sub-audible bits into
  `ltr.ControlChannel.Process`, which buffers across calls,
  slides a 41-bit window over the stream, commits to the first
  position whose Sync bit is set, and follows the alignment
  forward — unlocking + re-searching if a subsequent frame's
  Sync bit drops to 0. Each successfully-aligned Status word is
  dispatched into the existing `Ingest` path. `trunking.Protocol`
  gains `ProtocolLTR` (config string `"ltr"`). FCS verification
  over the 12-bit trailer + Manchester decoding of the on-air
  bit stream are documented follow-ups; until they land the
  adapter is honest about its noise floor (spurious alignments
  drop through the state machine's Area / activeGroup dedup,
  correctly-aligned frames drive cc.locked + grants).
- **Motorola Type II `ControlChannel.Process(stream, baseIdx)`
  adapter + ccdecoder factory.** Closes the IQ → CC sync layer
  for Motorola: the receiver's `BitSink` forwards bits into
  `motorola.ControlChannel.Process`, which buffers across calls
  + detects the 24-bit outbound sync + slices a 32-bit OSW out
  of the wire + parses it via `OSWFromBits` + dispatches via the
  existing `Ingest`. `trunking.Protocol` gains `ProtocolMotorola`
  (config string `"motorola"`). The BCH(64,16,11) FEC + de-
  interleaving over the OSW are follow-ups; until they ship the
  adapter sync-locks but typically fails OSW parsing on noisy
  on-air signals.
- **EDACS / GE-Marc `ControlChannel.Process(stream, baseIdx)`
  adapter + ccdecoder factory.** Closes the IQ → CC loop for
  EDACS: the receiver's `BitSink` forwards bits into
  `edacs.ControlChannel.Process`, which buffers across calls +
  detects the 24-bit outbound sync + slices the 40-bit CCW +
  parses it via `CCWFromBits` + dispatches via the existing
  `Ingest`. `trunking.Protocol` gains `ProtocolEDACS` (config
  string `"edacs"`). On-air recovery margins improve once the
  per-CCW BCH(40, 28, 2) FEC layer (later wired as
  `edacs_bch_mode: on`) is enabled — Standard EDACS uses BCH
  as its only CCW-level FEC per the `lwvmobile/edacs-fm`
  reference. Earlier README revisions called out an outer
  "interleaved Reed-Solomon-derived FEC" as a follow-up; that
  was a documentation error, no such layer exists in Standard
  EDACS.
- **NXDN `ControlChannel.Process(stream, baseIdx)` adapter +
  ccdecoder factory.** Closes the IQ → CC sync layer for NXDN:
  the receiver's `DibitSink` forwards into
  `nxdn.ControlChannel.Process`, which buffers across calls +
  detects the 8-dibit outbound FSW + parses the LICH from the
  next 16 wire bits (doubled-bit majority decode via
  `DecodeLICHWire` → `ParseLICH`) + pulls the first 44 dibits of
  the 144-dibit Info field as raw CAC bits → `ParseCAC` →
  `IngestFrame`. The CAC FEC layer (K=5 ½-rate Viterbi +
  interleaver + puncture across the full 288-wire-bit Info field)
  is the next NXDN follow-up; until it ships the adapter
  sync-locks but typically fails the CAC CRC on real on-air
  signals. Inbound (MS → BS) FSW matches are silently ignored
  since they don't carry the CC announcement payloads the state
  machine locks on.
- **dPMR Mode 3 `ControlChannel.Process(stream, baseIdx)` adapter +
  ccdecoder factory.** Closes the IQ → CC loop for dPMR: the
  receiver's `DibitSink` forwards into `dpmr.ControlChannel.Process`,
  which buffers across calls + detects the 24-dibit FS3 sync +
  slices the 40-dibit / 80-bit CSBK + parses it via `CSBKFromBits`
  + dispatches via the existing `Ingest`. `trunking.Protocol` gains
  `ProtocolDPMR` (config string `"dpmr"`) so the ccdecoder factory
  map can resolve it. First of the per-protocol adapter follow-ups
  from the connector PR.
- **Daemon wiring for the IQ → CC decoder connector**
  (`cmd/gophertrunk/daemon.go`). When the daemon's pool has a
  control-role SDR + at least one trunked system configured, it
  constructs a `ccdecoder.Decoder` next to the existing
  `cchunt.Supervisor` and spawns it as a daemon goroutine. The
  connector owns the control SDR's `StreamIQ` loop, swaps the
  active per-protocol pipeline on every `KindHuntProgress` retune,
  and pumps IQ chunks through the active pipeline whose CC state
  machine publishes `cc.locked` / `grant` events back on the bus
  — the trigger that lights up every downstream surface (engine,
  recorder, call log, API, TUI). `make integration` now boots the
  full chain with a mock SDR and asserts the connector is
  constructed + runs without crashing.
- **IQ → control-channel decoder connector** (`internal/scanner/ccdecoder`)
  — subscribes to `events.KindHuntProgress`, owns one
  `StreamIQ(ctx)` loop on the control SDR, swaps the active
  per-protocol pipeline (IQ → symbol-domain decoder → CC state
  machine) on every supervisor retune, and pumps IQ chunks through
  the active pipeline whose CC state machine publishes
  `cc.locked` / `grant` events back on the bus. Closes the
  load-bearing gap from "Status & known gaps". P25 Phase 1 and YSF
  pipelines wire end-to-end today; other protocols register their
  factories once the per-protocol `ControlChannel.Process(stream,
  baseIdx)` adapters ship.
- **TETRA TMO IQ → π/4-DQPSK dibit receiver** (`internal/radio/tetra/receiver`)
  composing the `demod.PiOver4DQPSK` helper (RRC matched filter at
  α = 0.35, π/4-rotated differential decode) with naive symbol-
  time decimation at 18000 sym/s into one entry point that fans
  dibits out via the new `tetra.DibitSink` callback. **Last
  per-protocol receiver** in the family — every trunked control
  modulation listed in the Features table now has an IQ → symbol
  / bit chain shipping in tree. Full symbol-time clock recovery
  (Gardner on complex IQ or eye-tracking on |y|²) is a follow-up;
  the connector that lands next wraps a timing-recovery loop
  around the π/4-DQPSK family when a real-air capture is
  available. The `ControlChannel.Process(dibits, baseIdx)`
  adapter that does 38-dibit training-sequence sync detect +
  burst slice + L3 PDU dispatch is the next layer up.
- **P25 Phase 2 IQ → H-DQPSK dibit receiver** (`internal/radio/p25/phase2/receiver`)
  composing the `demod.PiOver4DQPSK` helper (RRC matched filter +
  π/8-rotated differential decode) with naive symbol-time
  decimation at 6000 sym/s into one entry point that fans dibits
  out via the new `phase2.DibitSink` callback. Ninth per-protocol
  receiver — the first π/4-DQPSK-family one, leaning on the
  helper shipped earlier in the roadmap. Full symbol-time clock
  recovery (Gardner on complex IQ or eye-tracking on |y|²) is a
  follow-up; the connector will wrap a timing-recovery loop
  around this when a real-air capture is available. The
  `ControlChannel.Process(dibits, baseIdx)` adapter that does
  20-dibit sync detect + MAC PDU slice + opcode dispatch is the
  next layer up.
- **Motorola Type II IQ → MSK bit receiver** (`internal/radio/motorola/receiver`)
  composing FM demod + Gaussian matched filter (BT = 0.5, the
  closest fit for an MSK matched filter) + Mueller-Müller clock
  recovery at 3600 baud + 2-level slicer into one entry point that
  fans bits out via the new `motorola.BitSink` callback. Eighth
  per-protocol receiver in the family — reuses the `demod.GFSK`
  helper since MSK (mod-index 0.5 CPFSK) decodes cleanly through
  the same FM-discriminator + matched-filter chain. The
  `ControlChannel.Process(bits, baseIdx)` adapter that does 24-bit
  sync detect + 84-bit OSW slice + BCH(64,16) decode + `ParseOSW`
  + `Ingest` is the next layer up.
- **LTR IQ → sub-audible bit receiver** (`internal/radio/ltr/receiver`)
  composing FM demod + a narrow sub-audible LPF (Kaiser-windowed
  FIR, ~300 Hz cutoff) + Mueller-Müller clock recovery at 300 baud
  + 2-level slicer into one entry point that fans bits out via the
  new `ltr.BitSink` callback. Seventh per-protocol receiver in the
  family. Manchester decoding + 41-bit Status framing live in the
  follow-up `ControlChannel.Process(bits, baseIdx)` adapter.
- **MPT 1327 IQ → FFSK bit receiver** (`internal/radio/mpt1327/receiver`)
  composing FM demod + FFSK tone discriminator (CCIR FFSK:
  mark = 1200 Hz / space = 1800 Hz) + Mueller-Müller clock
  recovery at 1200 baud into one entry point that fans bits out
  via the new `mpt1327.BitSink` callback. Sixth per-protocol
  receiver in the family and the first audio-band-FSK one — leans
  on the `demod.FFSK` helper shipped earlier in the roadmap. The
  `ControlChannel.Process(bits, baseIdx)` adapter that does
  cross-call bit buffering + 64-bit codeword slice + BCH(63,38)
  parity verification + `ParseCodeword` + `Ingest` is the next
  layer up.
- **EDACS / GE-Marc IQ → GFSK bit receiver** (`internal/radio/edacs/receiver`)
  composing FM demod + Gaussian matched filter (BT = 0.3) +
  Mueller-Müller clock recovery + 2-level slicer at 9600 baud
  into one entry point that fans bits out via the new
  `edacs.BitSink` callback. First non-C4FM per-protocol receiver
  in the family — leans on the `demod.GFSK` helper shipped earlier
  in the roadmap. The `ControlChannel.Process(bits, baseIdx)`
  adapter that does 24-bit sync detect + 40-bit CCW slice +
  `CCWFromBits` + `Ingest` is the next layer up.
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
make build                    # produces ./bin/gophertrunk
make test                     # go test -race ./...
make integration              # boots the wired daemon end-to-end (no SDR needed)
make integration-cc           # P25 Phase 1 "lights up live trunked reception"
make integration-cc-nxdn      # NXDN "lights up" — synthesizes spec FEC chain
make integration-cc-dmr       # DMR Tier III "lights up" — Aloha CSBK via BPTC
make integration-cc-dpmr      # dPMR Mode 3 "lights up" — FS3 sync + 80-bit CSBK
make integration-cc-edacs     # EDACS "lights up" — GFSK + BCH(40, 28, 2) CCW
make integration-cc-motorola  # Motorola Type II "lights up" — GFSK + BCH(64, 16, 11) OSW
make integration-cc-tetra     # TETRA TMO "lights up" — π/4-DQPSK + full §8.3.1 chain
make integration-cc-p25p2     # P25 Phase 2 "lights up" — H-DQPSK + trellis MAC PDU
make integration-cc-mpt1327   # MPT 1327 "lights up" — audio-band FFSK + BCH(63, 38)
make integration-cc-ltr       # LTR "lights up" — sub-audible NRZ at 300 baud

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
internal/scanner/       cchunt/ (multi-system CC supervisor) + conventional/ (analog FM scan list) + ccdecoder/ (IQ→CC connector)
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

Eleven panels covering every read surface plus the operator scanner
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

## FEC opt-ins

Every protocol that has a public-spec FEC chain ships the chain as
an **opt-in**: the connector constructs each `ControlChannel` in
its legacy raw-bit mode by default and only flips on the FEC layer
when the operator sets a per-system key in `config.yaml`. Empty /
absent keys preserve the legacy path so the synthesized-fixture
tests stay green and operators with pre-stripped capture files
(DSD-FME `-r` dumps, OP25 fixtures) don't see surprise CRC
failures.

Verify which opt-ins are active by opening the **Settings** panel
in the TUI — it lists every configured system with a one-line
summary of its FEC opt-in state (`channel coding: on (colour=…,
sch/f)`, `viterbi: off`, `bch: on`, etc.). The panel is read-only;
runtime mutation is a future PR. To change a mode, edit
`config.yaml` and restart the daemon.

| Protocol | YAML key(s) | Off (default) | On |
| --- | --- | --- | --- |
| TETRA | `tetra_colour_code` (uint32, low 30 bits), `tetra_channel` (`"sch/hd"` / `"sch/f"` / `"sch/hu"` / `"bsch"` / `"aach"`, default `sch/hd`) | Legacy 48-dibit raw-PDU path. CRC fails on live captures. | Full ETSI EN 300 392-2 §8.3.1 type-5 → type-1 chain (descramble + deinterleave + depuncture + Viterbi + CRC-16 verify + tail strip) per burst. Non-zero `tetra_colour_code` flips it on. |
| LTR | `ltr_fcs_mode` (`""` / `"off"` / `"on"`), `ltr_manchester_mode` (`""` / `"off"` / `"nrz"` / `"strict"` / `"soft"`) | NRZ Status bits, no FCS verification. Matches synthesized-fixture path. | CRC-7 FCS check against sdrtrunk's CRCLTR.java layout (`on`) and/or Manchester decode of sub-audible signaling (`soft` = majority decode, `strict` = require mid-bit transition). Live sub-audible captures typically need `manchester: soft` + `fcs: on`. |
| P25 Phase 2 | `p25_phase2_trellis_mode` (`""` / `"off"` / `"on"`) | Legacy 72-dibit raw-MAC-PDU path. | 4-state ½-rate trellis FEC over the MAC PDU window (146 channel dibits → 72 info dibits per TIA-102.AABF). |
| NXDN | `nxdn_viterbi_mode` (`""` / `"off"` / `"on"` / `"spec"`) | Legacy 44-dibit raw-CAC path. | `on`: simplified K=5 ½-rate Viterbi over the CAC region (92 dibits → 88 info bits + 4 tail zeros — matches the older MMDVMHost / DSDcc fixtures). `spec`: full NXDN-TS-1-A rev 1.3 §4.5.1.1 outbound chain (150 dibits = 300 channel bits → deinterleave 25×12 → depuncture 50/350 → K=5 Viterbi → 16-bit CRC verify → 155 info bits = 8 SR + 144 L3 + 3 Null). Use `spec` for live captures; `on` for back-compat with the older synthesized fixtures. |
| EDACS | `edacs_bch_mode` (`""` / `"off"` / `"on"`) | Legacy pre-stripped 40-bit CCW; payload struct's LCN bit 0 + Aux fields are data. | BCH(40, 28, 2) with single/double-bit correction over the 40-bit on-wire CCW; under `on` the effective CCW carries 28 info bits (Command + Status + Address + high LCN bits), the remaining bits become BCH parity. |
| MPT 1327 | `mpt1327_bch_mode` (`""` / `"off"` / `"on"`) | Legacy 38-bit pre-stripped codeword. | BCH(63, 38) decode over the 64-bit on-wire codeword. |
| Motorola Type II | `motorola_bch_mode` (`""` / `"off"` / `"on"`) | Legacy 32-bit raw-OSW path. | Two 64-bit BCH(64, 16, 11) codewords reassembled into the 32-bit OSW with single- through 11-bit-error correction per codeword. Live captures should set `on`. |

All string values are case-insensitive with whitespace tolerated;
recognised on-values include `"on"` / `"true"` / `"1"`, off-values
`""` / `"off"` / `"false"` / `"0"`. Unrecognised values fall back
to off with a `warn`-level log line ("ccdecoder: unrecognised
`<key>`; falling back to off") so a typo doesn't silently break
the decoder.

Each protocol's `ControlChannel` exposes matching getters
(`tetra.ControlChannel.ChannelCoding()` / `ExpectedChannel()` /
`ColourCode()`, `ltr.ControlChannel.FCSMode()` / `ManchesterMode()`,
`p25phase2.ControlChannel.TrellisMode()`,
`nxdn.ControlChannel.ViterbiMode()`,
`edacs.ControlChannel.BCHMode()`,
`mpt1327.ControlChannel.BCHMode()`) so tests + observability code
can introspect the configured state without poking at unexported
fields. The TUI Settings panel reads these via the
`/api/v1/systems` endpoint's per-system DTO, which carries every
opt-in field as a `omitempty` JSON value.

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
- [`docs/specs/`](docs/specs/) — reference air-interface PDFs the
  on-air FEC implementations derive from (NXDN-TS-1-A,
  ETSI EN 300 392-2 TETRA, plus a negative-reference M/A-COM LBI
  for EDACS that documents *not* what to look for)

## License

See [`LICENSE`](LICENSE).
