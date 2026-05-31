<p align="center">
  <img src="docs/assets/gophertrunk-logo.png" alt="GopherTrunk logo" width="220">
</p>

<h1 align="center">GopherTrunk</h1>

<p align="center">
  <strong>Pure-Go digital-trunking radio scanner engine for RTL-SDR · HackRF · Airspy · Airspy HF+.</strong><br>
  P25 · DMR · TETRA · NXDN · Motorola Type II · EDACS · LTR · MPT 1327 · dPMR · D-STAR · YSF.<br>
  Zero CGO, single static binary, headless daemon + Bubbletea TUI cockpit + browser web console.
</p>

<p align="center">
  <a href="https://github.com/MattCheramie/GopherTrunk/actions/workflows/ci.yml"><img src="https://github.com/MattCheramie/GopherTrunk/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/MattCheramie/GopherTrunk/releases"><img src="https://img.shields.io/github/v/release/MattCheramie/GopherTrunk?include_prereleases&sort=semver&color=155799" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/MattCheramie/GopherTrunk?color=155799" alt="License"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/MattCheramie/GopherTrunk?color=155799" alt="Go version"></a>
  <a href="https://goreportcard.com/report/github.com/MattCheramie/GopherTrunk"><img src="https://goreportcard.com/badge/github.com/MattCheramie/GopherTrunk" alt="Go Report Card"></a>
  <a href="https://gophertrunk.org"><img src="https://img.shields.io/badge/docs-gophertrunk.org-155799" alt="Docs"></a>
</p>

---

## What is this?

GopherTrunk is a software-defined-radio scanner that follows digital
trunked-radio voice calls and decodes them to audio. It runs on a
pool of RTL-SDR (every osmocom tuner), HackRF (One / Jawbreaker /
Rad1o), Airspy R2 / Mini, and Airspy HF+ dongles, has no C
dependencies at build or runtime (no `librtlsdr` / `libhackrf` /
`libairspy` / `libairspyhf` / `libusb` / `libasound2` /
`libmp3lame`), and ships as a single ~10 MB static binary for Linux,
macOS, and Windows.

Completed calls stream to Broadcastify Calls, RdioScanner, OpenMHz,
and live Icecast / ShoutCast mountpoints out of the box. Why does
this exist? Read **[The Story of GopherTrunk](https://gophertrunk.org/story.html)**.

## Quick start

```sh
# Linux x86_64 — see https://gophertrunk.org/downloads.html for macOS, Windows, ARM64.
VERSION=v0.2.6
curl -L -o gophertrunk.tar.gz \
  https://github.com/MattCheramie/GopherTrunk/releases/download/${VERSION}/gophertrunk-${VERSION}-linux-amd64.tar.gz
tar xzf gophertrunk.tar.gz && cd gophertrunk-${VERSION}-linux-amd64
cp config.example.yaml config.yaml
./gophertrunk version
# Plain `./gophertrunk` (no subcommand, on a TTY) drops into the
# interactive launcher: pick [1] TUI, [2] Web, or [3] Headless.
# Skip the prompt with -tui / -web / -headless.
./gophertrunk -config config.yaml
```

Windows users get a one-click installer that bundles Zadig for
WinUSB driver setup; macOS users get notarised tarballs for Apple
Silicon and Intel. Full per-OS recipes at
**[gophertrunk.org/downloads.html](https://gophertrunk.org/downloads.html)**.

## Features

- **Trunked control-channel decoders** — P25 Phase 1 + Phase 2 (full
  TIA-102 chain), DMR Tier II + Tier III (vendor-aware: Capacity
  Plus / Capacity Max grants and rest-channel tracking), NXDN,
  Motorola Type II / SmartZone, EDACS / GE-Marc, LTR, MPT 1327,
  dPMR Mode 3, TETRA TMO. Amateur-radio: D-STAR and Yaesu System
  Fusion.
- **POCSAG paging** — protocol layer for the dominant wireline
  FSK pager protocol (CCIR 584): BCH(31,21) FEC, batch carve-up,
  numeric (5 BCD/codeword) + alphanumeric (7-bit packed ASCII)
  decoders. Foundation for fire / EMS dispatch text alongside
  the trunked-voice pipeline; DSP wiring follows in the next PR.
  See [docs/pocsag.md](docs/pocsag.md).
- **APRS / AX.25 packet** — end-to-end pipeline for the
  amateur-radio APRS metadata bus (position beacons, messages,
  bulletins, status, Mic-E mobile-tracker compressed format).
  Bell-202 AFSK DSP frontend (FM demod → FFSK tone discriminator
  → symbol-time recovery → NRZI → HDLC framer), AX.25 frame
  parser with CRC-16-CCITT, APRS info-field decoders including
  full Mic-E (lat/lon, speed, course, altitude, message code),
  plus `events.KindAPRSPacket` bus event, SQLite `aprs_log`,
  `GET /api/v1/aprs/packets`, and `/aprs` web panel.
  See [docs/aprs.md](docs/aprs.md).
- **AIS marine** — end-to-end pipeline for the Automatic
  Identification System every commercial vessel broadcasts on
  marine VHF 87B / 88B (161.975 / 162.025 MHz). 9600 Bd GMSK DSP
  frontend (FM demod → GFSK matched filter at BT = 0.4 →
  symbol-time recovery → NRZI → HDLC framer → CRC-CCITT
  validation), ITU-R M.1371-5 message-type dispatch (Class A
  position reports 1/2/3, Class B 18 + 19 extended, base-station
  4, static + voyage 5, Class B static 24 A + B), signed-integer
  lat/lon decoder (1/600000 minute resolution), 6-bit ASCII text
  fields (vessel name, call-sign, destination), spec
  "not-available" sentinels. Plus `events.KindAISMessage` bus
  event, SQLite `vessel_log`, `GET /api/v1/ais/vessels`, and
  `/ais` web panel. See [docs/ais.md](docs/ais.md).
- **MDC1200 signaling** — end-to-end pipeline for Motorola's
  analog FFSK data burst keyed at the head / tail of a
  transmission on conventional VHF / UHF voice channels. 1200-baud
  CCIR FFSK DSP frontend (FM demod → FFSK discriminator at
  1200 / 1800 Hz → Mueller-Müller timing → NRZ slicer), 40-bit
  sync framer with polarity tolerance, 16×7 de-interleave, op /
  arg / unit-ID decode with CRC-16-CCITT check, and an op/arg
  table (PTT ANI, emergency, status, radio check, call alert,
  selective call). Plus `events.KindMDC1200Message` bus event,
  SQLite `mdc1200_log`, `GET /api/v1/mdc1200/messages`, and
  `/mdc1200` web panel. See [docs/mdc1200.md](docs/mdc1200.md).
- **DSC marine distress** — protocol layer for the GMDSS Digital
  Selective Calling system that fires every marine VHF channel-
  70 distress alert. ITU-R M.493-15 format dispatch (Distress,
  All-Ships, Individual, Group, Geographic, Auto-Individual),
  BCH(10,7) syndrome check, position decoder with quadrant
  hemisphere flip, nature-of-distress table. Plus
  `events.KindDSCMessage` bus event, SQLite `dsc_log`,
  `GET /api/v1/dsc/messages`, and `/dsc` web panel (rows tint
  by category — distress=red, urgency=orange, safety=blue).
  **DSP frontend** decodes straight off the air: FM demod →
  FFSK discriminator (1200 Bd, 1300/2100 Hz tones) →
  symbol-timing recovery → direct-FSK slicer → BCH(10,7)
  character sync → ITU-R M.493 parser. Pin an SDR via
  `dsc.channels`. See [docs/dsc.md](docs/dsc.md).
- **ADS-B aviation** — end-to-end pipeline for the 1090 MHz
  Mode-S transponder broadcasts every commercial flight emits.
  ICAO Annex 10 Vol IV / DO-260B parser (CRC-24 + DF dispatch +
  type-code dispatch for identification, airborne / surface
  position with 12-bit Q-bit altitude, airborne velocity);
  globally-unambiguous CPR position decoder + per-ICAO
  even+odd pair tracker. **BEAST upstream** consumes Mode-S
  frames from any dump1090 / readsb / BeastSplitter via TCP —
  most 1090 MHz receive chains already run one, GopherTrunk
  decodes its output. **Native PPM DSP frontend** is the
  alternative: pin an SDR (>= 2 Msps) to 1090 MHz via
  `adsb.channels` and GopherTrunk demodulates Mode-S itself —
  magnitude envelope → 8 µs preamble correlation → PPM bit
  slice → CRC/DF dispatch — feeding the same decode/track path
  the BEAST upstream uses. Plus `events.KindAircraftReport` bus
  event, SQLite `aircraft_log`, `GET /api/v1/adsb/aircraft`,
  and `/adsb` web panel. See [docs/adsb.md](docs/adsb.md).
- **Live map** — Shared Leaflet map at the top of each
  position-bearing panel (APRS / AIS / DSC / ADS-B) renders
  decoded positions over OpenStreetMap tiles, colour-coded per
  protocol (blue / cyan / red-distress / purple), with marker
  tooltips and camera auto-fit. Same `<PositionMap>` component
  drives all four panels.
- **Pure-Go voice path** — IMBE (P25 Phase 1) and AMBE+2 (P25
  Phase 2 / DMR) vocoders in Go, no DVSI / mbelib dependency.
  Per-call WAV + raw-frame sidecars; live PCM playback via direct
  ALSA / WASAPI / CoreAudio.
- **Pure-Go SDR drivers** — RTL-SDR, HackRF, Airspy R2 / Mini,
  Airspy HF+ family. USB transport on Linux (USBDEVFS), Windows
  (WinUSB), macOS (IOKit). USB-disconnect self-healing recovers
  dongles that drop off the bus and re-enumerate without
  restarting the daemon. See
  [docs/hardware.md](docs/hardware.md).
- **Remote rtl_tcp SDRs** — Mount any number of `rtl_tcp`
  endpoints as virtual tuners alongside local USB dongles. The
  SDR can live on a Raspberry Pi at the antenna while the daemon
  runs on a beefier host; one entry per remote in `sdr.rtl_tcp`.
- **Live spectrum / waterfall** — In-browser FFT waterfall served
  off the same IQ stream the trunking decoder consumes. New
  `internal/sdr/iqtap` multi-consumer fan-out lets future
  trunking-adjacent decoders (paging, AIS, ADS-B, ...) tap the
  same source without disturbing CC decode. `GET /api/v1/spectrum/devices`
  + `WS /api/v1/spectrum/stream`; web panel under `/spectrum`.
- **CC Activity panel** — focused web view of the trunked control-
  channel chatter (grants, affiliations, registrations, patches,
  talker aliases, CC lock / loss). Pure filter over the events
  stream with per-row payload rendering; web panel at `/cc`. RIDs
  in the feed are clickable chips that pivot into the per-radio
  detail view.
- **Radio IDs panel** — per-radio (subscriber-unit) entity browser
  with the same shape as Talkgroups. Merges the operator-configured
  alias catalogue (per-system `rid_alias_file` CSV or JSON: alias,
  owner, tag, group, priority, lockout, watch) with the live
  affiliation tracker (last talkgroup, last seen, call count,
  decoded talker alias). Detail modal pulls the last 50 calls for
  the RID from the persisted call log. Web panel at `/rids`; REST
  at `/api/v1/rids`; gRPC `RIDService`. Talker-alias decoders
  cover the Motorola vendor TSBK form (control channel) and the
  Motorola voice-channel LCs (P25 Phase 1 LDU1 LCO 0x15 header
  + N × LCO 0x17 data blocks, run through Motorola's
  reverse-engineered alias cipher).
- **Constellation viewer** — live IQ scatter visualization that
  taps the same broker the trunking decoder reads. Useful for
  identifying signal shape (PSK / QPSK / FSK / C4FM / AM /
  noise), spotting frequency offset, and checking demod /
  equalizer health. Decimated to 2 ksps for the wire; canvas
  scatter with energy banner. Web panel at `/constellation`.
- **Bookmarks / frequency manager** — UI-managed conventional
  channel list (marine VHF, NOAA weather, FRS/GMRS, repeater
  outputs, public-safety fall-back channels) stored in the
  daemon's SQLite database. Edit / create / delete from the web
  panel under `/bookmarks`; REST at `/api/v1/bookmarks`.
- **One dongle, many carriers** — `role: wideband` pins a single
  SDR to a centre frequency and runs an internal channelizer so
  one dongle decodes every DMR Tier II conventional repeater, DMR
  Tier III control channel, P25 Phase 1 control channel, AND P25
  Phase 2 control channel that fit inside its IQ bandwidth (e.g.
  several 12.5 kHz carriers inside a 2.4 MHz IQ window). Mix
  protocols on the same dongle.
- **One dongle, control + voice** — with `voice_taps: N` on a
  wideband entry, the daemon allocates per-grant DDC tuners from
  the dongle's IQ stream so trunked voice grants (DMR T3, P25
  Phase 1, P25 Phase 2) decode inline on the same SDR that's
  already hosting the control channel — no separate `role: voice`
  dongle needed for grants inside the wideband window. Out-of-
  window grants spill over to a physical voice SDR when present.
  See [docs/hardware.md](docs/hardware.md) and
  [samples/dmr-tier2-multichannel/](samples/dmr-tier2-multichannel/).
- **DSP** — Polyphase channelizer, Kaiser / RRC / Gaussian FIRs,
  FM / C4FM / GFSK / FFSK / DQPSK / π/4-DQPSK / π/8-H-DQPSK
  demods, Mueller-Müller + Gardner clock recovery, LMS + CMA
  equalizers, diversity combining.
- **APIs** — gRPC + HTTP/SSE + WebSocket; optional TLS +
  bearer-token auth on mutations; Prometheus `/metrics`; pure-Go
  SQLite call log; in-process pub/sub event bus.
- **Hamlib `rigctld` integration** — optional TCP server speaking
  the standard Hamlib wire protocol so loggers, satellite
  trackers, and amateur-radio tooling (Cloudlog, GridTracker,
  PSTRotator, `rigctl(1)`) can read and set the control SDR's
  frequency. RX-only backend; see
  [docs/rigctld.md](docs/rigctld.md).
- **Outbound call streaming** — Broadcastify Calls, RdioScanner,
  OpenMHz, live Icecast / ShoutCast with pre-encoded silence keep-alive.
  Pure-Go MP3 encoder. See `internal/broadcast`.
- **Baseband recording + offline replay** — Two-channel 16-bit WAV
  capture and a replay driver that mounts captures back into the
  SDR pool as virtual tuners. Looping replay simulates a
  continuous source.
- **Operator surfaces** — Bubbletea TUI cockpit with 12 panels,
  pure-browser React SPA web console, runtime config editing via
  `PATCH /api/v1/settings`, RadioReference PDF / CSV importer with
  a config-builder wizard.
- **Location + affiliation** — NMEA-0183 GGA / RMC over the air
  decoded into a SQLite `location_log`; protocol-agnostic
  affiliation tracker fed from grants / registrations / affiliation
  events.

For the full per-protocol FEC chain reference, receiver internals,
frame layouts, and API routes, see
[docs/architecture.md](docs/architecture.md) and
[docs/opt-in-features.md](docs/opt-in-features.md).

## Status snapshot

Once a `grant` event lands on the bus, the engine + recorder pipeline
runs end-to-end: voice device is allocated, the composer pulls IQ →
PCM, the recorder writes a WAV, the call is logged to SQLite, and
the API + TUI surfaces all light up. Every trunked control
modulation in the Features list has an end-to-end IQ → CC chain
shipping. SDRtrunk-parity subsystems (outbound streaming, baseband
recording, GPS / location, affiliation tracking, decoded-message
log, per-talkgroup policy) all ship.

**Remaining gaps:**

- **Digital-voice composer chains.** FM, DMR, P25 Phase 1 / 2 decode
  to audio. NXDN, dPMR, TETRA, YSF, D-STAR voice (plus EDACS
  ProVoice) are followed and logged but not yet turned into PCM.
- **Additional SDR validation.** HackRF / Airspy / HF+ drivers
  exercise the documented USB vendor protocols under unit tests
  against a mock transport; on-air validation against attached
  hardware is the documented follow-up.
- **FEC inner-layer real-air validation.** NXDN per-protocol
  interleaver and TETRA on-air recovery margins need live captures
  to characterise.
- **Vocoder level calibration** awaits reference WAVs in
  `internal/voice/{imbe,ambe2}/testdata/`.

The long-form status, per-protocol detail, and shipping-vs-pending
checklist live in **[docs/status.md](docs/status.md)**. Near-term
plans live in **[docs/roadmap.md](docs/roadmap.md)**. Released work
lives in **[CHANGELOG.md](CHANGELOG.md)**.

## Build from source

```sh
make dist     # SPA + daemon — single binary that serves the web console at /
make build    # Go-only — fast iteration; daemon shows a helpful 404 at / until bundled
make test     # go test -race ./...
make vet      # go vet ./...
make integration  # daemon end-to-end test (no SDR required)
```

The per-protocol "lights up" integration tests
(`make integration-cc-<proto>`) and the DVSI hardware-backend tests
(`make test-dvsi`) are documented in
[`CONTRIBUTING.md`](CONTRIBUTING.md).

A bare `go build ./cmd/gophertrunk` works too — the binary
auto-stamps its version line from Go's built-in VCS info. Useful
when attaching a log to an issue and you want the commit hash in
the build-info line.

### Docker

```sh
docker compose up -d
curl -s http://localhost:8080/api/v1/health
curl -s http://localhost:8080/metrics | grep gophertrunk_build_info
```

USB pass-through recipe and the operator hardening playbook (TLS,
bearer-token auth, Prometheus catalogue, smoke tests) live in
[docs/hardening.md](docs/hardening.md).

## Documentation

Operator-facing docs live at **[gophertrunk.org](https://gophertrunk.org)**
(rendered from this `docs/` tree):

- **Install** — [Downloads](https://gophertrunk.org/downloads.html) ·
  [Hardware](docs/hardware.md) ·
  [Linux](docs/install-linux.md) ·
  [macOS](docs/install-macos.md) ·
  [Windows](docs/install-windows.md)
- **Operate** — [Launcher](docs/launcher.md) ·
  [Windows user guide](docs/user-guide-windows.md) ·
  [TUI](docs/tui.md) ·
  [Web console](docs/web.md) ·
  [Live config editing](docs/live-edits.md) ·
  [Import (PDF / CSV)](docs/import.md) ·
  [Hardening](docs/hardening.md)
- **Reference** — [Architecture](docs/architecture.md) ·
  [Vocoders](docs/vocoders.md) ·
  [Voice calibration](docs/voice-calibration.md) ·
  [DMR encryption](docs/dmr-encryption.md) ·
  [Opt-in features](docs/opt-in-features.md) ·
  [Status](docs/status.md) ·
  [Roadmap](docs/roadmap.md) ·
  [Cutting a release](docs/release.md)
- **Project metadata** — [CHANGELOG](CHANGELOG.md) ·
  [CONTRIBUTING](CONTRIBUTING.md) ·
  [SECURITY](SECURITY.md) ·
  [THIRD_PARTY_LICENSES](THIRD_PARTY_LICENSES.md)

## Support the project

GopherTrunk is developed in the open and powered entirely by
community support. If it's useful to you, please consider chipping
in:

- [Sponsor on GitHub](https://github.com/sponsors/MattCheramie)
- [Tip on Ko-fi](https://buymeacoffee.com/Mrcheramie)

More ways to help: [docs/support.md](docs/support.md).

## License

See [LICENSE](LICENSE).
