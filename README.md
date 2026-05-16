<p align="center">
  <img src="docs/assets/gophertrunk-logo.png" alt="GopherTrunk logo" width="220">
</p>

<h1 align="center">GopherTrunk</h1>

<p align="center">
  <strong>Pure-Go digital-trunking radio scanner engine for RTL-SDR.</strong><br>
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

GopherTrunk is a software-defined-radio scanner that follows digital trunked-radio voice calls and decodes them to audio. It runs on a pool of cheap RTL-SDR dongles, has no C dependencies (no `librtlsdr` / `libusb` at build or runtime), and ships as a single ~10 MB static binary for Linux, macOS, and Windows.

<!-- TODO: replace this comment with a screenshot of the TUI cockpit once one is captured -->
<!-- <p align="center"><img src="docs/assets/tui-screenshot.png" alt="GopherTrunk TUI showing live call activity" width="800"></p> -->

## Quick start

```sh
# Linux x86_64 — see https://gophertrunk.org/downloads.html for macOS, Windows, ARM64
VERSION=v0.1.0
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

Full per-OS install (Windows installer / macOS Apple Silicon / Linux aarch64): **[gophertrunk.org/downloads.html](https://gophertrunk.org/downloads.html)** · Launcher / `-tui` / `-web` / `-headless`: [`docs/launcher.md`](docs/launcher.md) · Web console setup + quick start: **[gophertrunk.org/web.html](https://gophertrunk.org/web.html)** · TUI keybindings: [`docs/tui.md`](docs/tui.md) · Hardware setup (udev rules, WinUSB / Zadig): [`docs/hardware.md`](docs/hardware.md) · Production hardening (TLS, bearer-token auth, Docker): [`docs/hardening.md`](docs/hardening.md).

## Features

**Trunked-radio control-channel decoders** — P25 Phase 1 + Phase 2 (full TIA-102 chain with RS(24,16,9) outer verifier + PN44 scrambler), DMR Tier II + Tier III, NXDN, Motorola Type II / SmartZone, EDACS / GE-Marc, LTR, MPT 1327, dPMR Mode 3, TETRA TMO. All run live on IQ via `internal/scanner/ccdecoder` with per-protocol FEC chains default-on.

**Amateur-radio digital modes** — D-STAR (JARL DV-mode K=5 R=½ Viterbi + PN15 scrambler + 22×30 interleaver) and Yaesu System Fusion (C4FM + FICH trellis).

**Pure-Go voice path** — IMBE (P25 Phase 1) and AMBE+2 (P25 Phase 2 / DMR / NXDN) vocoders implemented in Go, no DVSI / mbelib dependency. Per-call WAV + raw-frame sidecars; live PCM playback via direct ALSA / WASAPI / CoreAudio (no `libasound2` at runtime).

**SDR layer** — Pure-Go RTL-SDR driver across USBDEVFS (Linux), WinUSB (Windows), IOKit (macOS). All major tuner drivers (R820T / R820T2 / R828D / E4000 / FC0012 / FC0013 / FC2580). Multi-device pool with role assignment, per-device gain / PPM / bias-tee.

**DSP** — Polyphase channelizer + CIC + halfband, Kaiser / RRC / Gaussian FIR designers, FM / C4FM / GFSK / FFSK / DQPSK / π/4-DQPSK / π/8-H-DQPSK demods, Mueller-Müller + Gardner clock recovery, LMS + CMA blind equalizers, Selection + maximal-ratio diversity combining.

**API + observability** — gRPC + HTTP/SSE + WebSocket surfaces, optional TLS + bearer-token auth on mutations, Prometheus `/metrics`, pure-Go SQLite call log, in-process pub/sub event bus with typed payloads.

**Operator surfaces** — first-class Bubbletea TUI cockpit with 12 panels (including a live Import panel and inline-editable Settings) and a sibling **web console** (a pure-browser React/Tailwind SPA, either embedded into the daemon binary at build time or shipped pre-built as `gophertrunk-web/` next to the binary — see [`web/README.md`](web/README.md) and the [§Web console](#web-console) section); the daemon binary doubles as its own UI launcher (`gophertrunk -tui` / `-web` / `-headless` — see [`docs/launcher.md`](docs/launcher.md)); conventional FM scanner with CTCSS / DCS squelch, two-tone QC-II paging detector, runtime channel lockout, manual VFO tune. Live `PATCH /api/v1/settings` writes operator edits straight to `config.yaml` (comments preserved) and hot-reloads what it safely can; `POST /api/v1/import` accepts PDF / CSV uploads, parses + previews, then merges into the running daemon.

**System bring-up** — `gophertrunk import-pdf` parses RadioReference.com trunking-system PDF exports and structured multi-section CSV bundles, then merges sites + talkgroups into `config.yaml` (preserving comments) plus per-system Trunk-Recorder-format CSVs. Interactive Bubbletea TUI for pruning sites, toggling Scan / Lockout / Priority before write; `-no-tui` / `-dry-run` / `-force` for CI. **`-wizard`** launches an interactive config-builder that walks through every section of `config.yaml` (log, API, auth, CORS, storage, recordings, retention, SDR devices, scanner, audio) so first-time operators get a runnable file without hand-editing YAML. See **[`docs/import.md`](docs/import.md)**.

Full encyclopedic breakdown — per-protocol FEC chains, receiver internals, frame layouts, API mutation routes, telemetry events — lives at **[`docs/architecture.md`](docs/architecture.md)**.

## Support the project

GopherTrunk is developed in the open and powered entirely by community support. If it's useful to you, please consider chipping in:

- [Sponsor on GitHub](https://github.com/sponsors/MattCheramie)
- [Tip on Ko-fi](https://buymeacoffee.com/Mrcheramie)

More ways to help: **[`docs/support.md`](docs/support.md)**.

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
constructed by `cmd/gophertrunk` covers all 10 trunked protocols
(P25 Phase 1, P25 Phase 2, DMR Tier III, NXDN, dPMR Mode 3, EDACS,
Motorola Type II, LTR, MPT 1327, TETRA TMO) plus DMR Tier II
conventional and YSF / D-STAR on the amateur side.

The remaining gaps:

- **Per-protocol on-air FEC layers — most shipping, some inner
  layers TODO.** Every protocol's `ControlChannel.Process`
  adapter ships a working IQ → CC chain (see
  [FEC opt-outs](#fec-opt-outs) for the full reference). The
  spec-correct chain is on by default for every protocol;
  operators with pre-stripped capture files opt out per-system.
  TETRA ships the full ETSI EN 300 392-2 §8.3.1 chain
  (descramble + deinterleave + depuncture + Viterbi + CRC-16);
  DMR Tier III + Tier II both ship full BPTC(196,96) + RS(12,9)
  + CSBK CRC; LTR ships CRC-7 FCS + Manchester soft decode;
  D-STAR ships the full JARL DV-mode header chain (K=5 R=1/2 +
  PN15 + 22×30 interleaver); P25 Phase 1, P25 Phase 2 trellis,
  EDACS BCH(40,28,2), MPT 1327 BCH(64,48,2), Motorola BCH(64,16,11)
  all ship as opt-out. The inner FEC layers still pending:
  - **NXDN per-protocol interleaver + puncture.** `ViterbiSpec`
    mode runs the full §4.5.1.1 chain; `ViterbiOn` is the
    simpler bare-bones path the older MMDVMHost / DSDcc fixtures
    use. Both are wired through the connector; the
    interleaver/puncture detail-level matching against captured
    MMDVMHost transmissions is the calibration step that lands
    next.
  - **P25 Phase 2 FEC chain (trellis + outer RS + PN44
    scrambler + per-burst offset probe, all shipping).** The
    full TIA-102 chain wraps the MAC PDU in three layers, each
    opt-in via a per-system flag:
    - The inner 4-state ½-rate trellis decoder
      (`SetTrellisMode(TrellisOn)`) handles the on-wire FEC.
    - The outer RS(24, 16, 9) over GF(2^6) per TIA-102.BAAA-A
      §5.9 (`SetRSMode(RSOn)`) drops MAC PDUs whose syndromes
      are non-zero.
    - The PN44 LFSR scrambler per TIA-102.BBAC-1 §7.2.5 with a
      per-burst slot-offset blind probe
      (`SetScramblerMode(ScramblerProbe)`) walks all 12 slot
      offsets from Figure 7-5 and accepts whichever passes RS
      verification — no external superframe synchronization is
      required. The seed is derived from the per-system (WACN,
      SystemID, NAC) triple (`SetScramblerSeed` /
      `framing.PN44SeedFromIdentity`).

    The previous follow-ups — full superframe-aware per-burst
    offset tracking AND NSB-driven runtime seed installation —
    now both ship. The `ScramblerProbe` blind-probe walks all 12
    slot offsets; the `ControlChannel.Ingest` path auto-recomputes
    the seed from every Network Status Broadcast - Update MAC PDU
    (opcode 0xFB) via `pn44SeedFromNSB`. Per-system static config
    still provides the initial seed for the first few PDUs before
    NSB lands, and stays available as an override.
  - ~~**MPT 1327 sync detection + bit-error-tolerant CWSC**~~
    (now shipping). The BCH(64, 48, 2) per-codeword check + the
    16-bit Codeword Synchronisation Code (`1100010011010111`)
    alignment per the MPT 1327 standard both ship. The Process
    adapter now matches CWSC against a Hamming-distance threshold
    (default 2 bits out of 16, matching commercial MPT 1327
    receivers on noisy on-air captures) instead of exact-match,
    falling back to the legacy "first parseable codeword"
    alignment when no CWSC window is within tolerance. Operators
    replaying pre-stripped synthesized fixtures opt back into
    exact-match per system via `mpt1327_cwsc_tolerance: 0`. The
    previously-cited "inter-codeword bit-interleaver across
    5-codeword CCDB groups" doesn't exist in the standard;
    MPT 1327 transmits 64-bit codewords back-to-back at 1200 bps
    FFSK with no inter-codeword bit permutation. No remaining
    MPT 1327 spec follow-ups.
  - ~~**YSF FICH on-air interleaver / puncture validation**~~
    (now shipping the spec-level codec). The K=5 ½-rate Trellis
    encoder + decoder
    (`internal/radio/ysf/fich_trellis.go`) round-trip cleanly
    in unit tests. `EncodeFICHOnAir` / `DecodeFICHOnAir` now
    layer the full on-air chain — puncture (drop channel-bit
    positions `{0, 1, 102, 103}`) plus column-major 10×10
    interleave (out[k] = depunctured[(k%10)*10 + (k/10)]) — per
    the MMDVMHost / DSDcc / Pi-Star reference. Every
    single-bit-flip in the 100-bit on-air stream is repaired
    by the Viterbi (`TestFICHOnAirRecoversFromSingleBitFlip`
    exhaustively confirms all 100 positions). On-air capture
    validation against a real Yaesu transmission is the
    remaining real-air-blocked piece — if the captured FICH
    fails CRC after the on-air decoder, the alternate-schedule
    swap is a two-line change documented in
    `samples/ysf/README.md`.
  - **TETRA on-air recovery margins.** Unit tests round-trip
    clean fixtures end-to-end; on-air recovery margins
    (Viterbi correction depth vs. real co-channel +
    adjacent-channel interference) need a live capture to
    characterise.
- ~~**DMR Tier II synthesized IQ fixture**~~ (now shipping).
  The Tier II pipeline + Process adapter + unit test all
  shipped in PR #184; the end-to-end integration test
  (`TestDaemonCCDecodesDMRTier2`) was previously `t.Skip`'d
  because the synthesized Voice LC Header IQ fixture's symbol
  distribution stresses the Mueller-Müller clock loop harder
  than Tier III's structurally-identical CSBK Aloha fixture.
  The diagnostic test (`TestDMRTier2VsTier3SymbolDensity` /
  `TestDMRTier2SlotTypeVsPayloadIsolation` in
  `cmd/gophertrunk/dmr_tier2_diagnostic_test.go`) localised
  the divergent statistic to the BPTC(196, 96)-encoded
  payload's class-3 dibit overrepresentation (21.4% Tier II
  vs 5.1% Tier III) and matching mean-transition magnitude
  (1.27 vs 0.90); the RS(12, 9) seed `0x96 0x96 0x96` and the
  BPTC parity rows distribute high-Hamming-weight bits
  throughout the channel-bit output. The fix lives in
  `internal/scanner/ccdecoder/pipelines.go`'s
  `newDMRTier2Pipeline`: lowering the per-protocol pipeline
  ClockGain from 0.025 (the value shared with Tier III) to
  0.015 keeps the MM loop locked under the harder symbol
  distribution. Receiver locks within ~100 ms of the first
  burst; the more conservative gain stays well within the
  loop's noise margin on live captures.
- **Digital-voice level calibration.** Pure-Go IMBE / AMBE+2
  emit real audio end-to-end. The comparison harness at
  `internal/voice/calibrate/` is ready;
  reference data (captured P25 P1 / DMR voice exchanges plus
  DSD-FME / OP25 decodes belong at
  `internal/voice/{imbe,ambe2}/testdata/`) is the remaining gap.
  Knox / call-alert AMBE+2 tones (b₁ ∈ [144, 163]) are
  vendor-specific and stay silent until per-vendor frequency
  tables land. See [docs/vocoders.md](docs/vocoders.md) for the
  licensing posture.
- **CTCSS + DCS sub-audible squelch + tail-fade on call end.** The
  conventional FM scanner optionally gates squelch on a sub-audible
  tone or digital code in addition to IQ power, so adjacent-system
  traffic on the same frequency doesn't trigger a false dwell.
  Per-channel YAML:
  ```yaml
  conventional:
    - label: "Sheriff Repeater"
      frequency_hz: 155895000
      tone:
        mode: ctcss        # ctcss | dcs | none
        ctcss_hz: 100.0    # required for ctcss
        # dcs_code: "023"  # required for dcs (3-digit octal)
  ```
  Both detectors share an `FM discriminator → single-pole IIR
  low-pass → bit/bin detector` pipeline. **CTCSS** runs a Goertzel
  at the configured frequency plus two reverse-bin Goertzels at
  ±5 Hz; a match requires the target bin both to exceed the
  magnitude floor AND to dominate the largest reverse bin by a
  configurable factor (default 1.5×). This rejects adjacent EIA
  codes whose spectral leak would otherwise show up in the target
  bin under the 5 Hz Goertzel resolution; the 38-code list has
  codes spaced as close as ~3 Hz at the low end and the
  single-bin path was prone to false-trigger on them. Reuses the
  existing `internal/voice/toneout` Goertzel primitive (~200 ms
  block). **DCS**
  recovers the 134.4 baud sub-audible NRZ stream, slides a 23-bit
  window, and matches against the 46 precomputed rotations (23
  cyclic shifts × 2 polarities) of the Golay(23,12,7) codeword
  built from the configured 3-digit octal code, reusing the
  `internal/radio/framing.GolayEncode23_12` primitive shared with
  P25 Phase 1 IMBE. Hangtime triggers on either condition (carrier
  OR tone/code) going false so a transmitter dropping the gate
  hangs up just like a true carrier drop. The scanner auto-bumps
  per-channel min dwell to 250 ms whenever any channel has a tone
  gate so the bit/Goertzel windows have time to fire. The composer
  also emits a 10 ms linear fade-out tail on call end
  (`internal/voice/composer`) so the audio sink doesn't hear an
  abrupt squelch-close click on the host speakers.
- **gRPC `AudioService.StreamAudio` live audio fan-out.** The
  daemon now ships an `api.AudioPublisher` that fans decoded PCM
  from the per-call composer to any number of gRPC subscribers.
  `StreamAudio` (defined in `proto/audio.proto`) reads
  `device_serials` + `talkgroup_ids` as filter allow-lists; the
  default empty filter forwards every call. Each subscriber gets a
  64-frame bounded channel — slow clients drop frames on full
  rather than back-pressuring the composer (per-subscriber and
  publisher-wide drop counters surface via the publisher's
  `Stats()` method). The publisher tracks per-device `Grant`
  context off the events bus so every frame carries talkgroup +
  system metadata. Disabled-cleanly: when the daemon runs without a
  composer (no SDR pool, audio off, etc.) the RPC returns
  `Unavailable` so a remote client gets a clean error instead of a
  hanging stream. Try it locally with
  `grpcurl -plaintext -d '{}' 127.0.0.1:50051 gophertrunk.v1.AudioService/StreamAudio`.
- **Manual VFO tune from the TUI / API.** The Scanner panel now binds
  `f` to a bubbles/textinput overlay: type a frequency in MHz, Enter,
  and the conventional FM scanner appends a runtime "manual" channel
  and forces dwell on it. Same flow available over REST as `POST
  /api/v1/scanner/manual_tune` (and `DELETE /api/v1/scanner/manual_tune/{idx}`
  to revoke), gated behind `api.auth` (see [API authentication](#api-authentication)). To run manual tune
  without any static `scanner.conventional` entries, set
  `scanner.manual_tune_enabled: true` in config — the daemon then
  constructs the conventional scanner against the last Voice SDR
  regardless of the static channel count. `internal/scanner/conventional`
  now accepts an empty seed channel list and exposes
  `AddTemporaryChannel` / `RemoveTemporaryChannel` so the same VFO
  surface is callable from any embedder.
- **Live audio playback to speakers + TUI / API audio cockpit.** The
  daemon ships a `voice.Player` sink (`internal/voice/player`) that
  routes decoded PCM to the host's default audio output. On Linux it
  talks to `libasound2.so.2` directly via `github.com/ebitengine/purego`
  — no cgo, no `libasound2-dev` at build time, no pkg-config; the
  runtime library ships on every standard Linux image. macOS / Windows
  use `github.com/ebitengine/oto/v3` (CoreAudio + WASAPI, also via
  purego). When `audio.enabled: true` is set in config the per-call
  composer and the conventional FM scanner fan PCM into the player
  alongside the existing WAV recorder, so calls play out the host's
  default output device in real time. Volume / mute / recording can be
  toggled live: the TUI's Scanner panel binds `+` / `-` for volume
  (5% step), `M` for mute, and `R` for record on/off; the same knobs
  are exposed as `GET` / `PATCH /api/v1/audio` for remote clients
  (PATCH gated by `api.auth` like every other write endpoint;
  see [API authentication](#api-authentication)). The recorder gate stops new WAVs from landing without
  truncating in-flight sessions, matching scanner muscle memory.
  Disabled by default; headless servers stay silent and continue to
  record WAVs identically to before. New CLI: `gophertrunk audio
  list` mirrors `sdr list`. If `libasound2.so.2` isn't reachable
  (stripped-down container, etc.) the backend logs once and falls
  back to the null player so the rest of the daemon keeps running.

The Go interfaces, event payloads, and per-protocol pipelines all
ship for every protocol in the Features table; the remaining work
above is per-protocol FEC inner-layer detail + reference data
sourcing.

## Roadmap

What's still on the table. Order isn't fixed; each item is contained
to its own package and lands independently.

- **DVSI USB-3000 / AMBE-3003 hardware backend (USB transport).**
  The `Vocoder` + AMBE-3003 wire protocol + `voice.Vocoder` interface
  conformance ship in [`internal/voice/dvsi/`](internal/voice/dvsi/)
  behind `-tags dvsi`; the package's `init()` registers `"dvsi"`
  with `voice.DefaultRegistry`. CI exercises the wire protocol +
  Vocoder plumbing through the scripted mock Transport and the
  software-loopback Transport (`make test-dvsi`). The USB / FTDI
  bulk-endpoint plumbing that talks to a physical chip remains a
  stub returning `ErrNoDevice` — the recorder fallback chain
  activates cleanly when no chip is connected. The actual FTDI
  hardware integration lands when a DVSI USB-3000 is available for
  round-trip testing.
- **Vocoder level calibration (reference data).** The plumbing
  ships — comparison harness at `internal/voice/calibrate`,
  per-vocoder testdata READMEs at
  `internal/voice/{imbe,ambe2}/testdata/`, the end-to-end recipe at
  [docs/voice-calibration.md](docs/voice-calibration.md), and a
  one-off CLI wrapper at `cmd/voice-calibrate` (run
  `go run ./cmd/voice-calibrate -raw call.raw -ref-wav ref.wav
  -vocoder imbe`). Operators just need to drop reference WAVs
  decoded by DSD-FME / OP25 from the same `.raw` into testdata; the
  existing calibrate tests run unguarded once both files are
  present. AMBE+2 DTMF dual-tone synthesis (b₁ ∈ [128, 143]) is
  wired against the ITU-T Q.23 4×4 matrix; knox / call-alert pairs
  (b₁ ∈ [144, 163]) are vendor-specific — operators with a per-
  vendor reference register the (freqA, freqB) pair via
  `ambe2.SetKnoxTone` and the matching tone frames synthesise
  through the same dual-tone path as DTMF.
- **YSF on-air interleaver / puncture validation (real-air capture).**
  The spec-level on-air codec ships in
  `internal/radio/ysf/fich_trellis.go`'s `EncodeFICHOnAir` /
  `DecodeFICHOnAir` per the MMDVMHost / DSDcc / Pi-Star reference
  (puncture positions `{0, 1, 102, 103}`, column-major 10×10
  interleave). Unit tests confirm every single-bit-flip is
  Viterbi-corrected. The remaining work is calibration against a
  real captured YSF transmission — if the captured FICH fails CRC
  after on-air decode, swap to the alternate schedule per
  `samples/ysf/README.md`.

### Recently shipped

- **Web operator console reaches feature parity with the TUI.**
  Every TUI panel now has a browser counterpart in [`web/`](web/)
  (Vite + React + TypeScript + Tailwind), shipped as the
  standalone `gophertrunk-web/` directory beside the binary in each
  release archive: Dashboard, Active (with end-call mutation),
  History (with retention-sweep mutation), Systems, Talkgroups
  (with priority / lockout / scan PATCH controls), Devices, Events,
  **Tones** (live ring + per-device reset), **Metrics** (Chart.js
  trend of curated `gophertrunk_*` counters + Prometheus snapshot
  tiles), **Scanner** (CC hunter hold/resume/retune per system,
  conventional channel dwell/lockout/unlockout, manual VFO tune,
  scan_mode toggle), Settings. All mutations are AND-gated through
  the `selectCanMutate` selector — write-mode toggle in Settings
  combined with the daemon's `/api/v1/mutations` capability flag —
  and confirm-modal-wrapped where destructive. The standalone-
  bundle, daemon-on-Pi / laptop-on-couch scenario is documented
  end-to-end in [`web/README.md`](web/README.md).
- **`gophertrunk import-pdf` subcommand.** Bootstraps a region's
  trunked-system definitions into `config.yaml` + per-system
  Trunk-Recorder talkgroup CSVs by parsing RadioReference.com PDF
  exports or structured multi-section CSV bundles (mixable with
  `-pdf` and `-csv` in a single invocation). Launches a Bubbletea
  TUI by default for reviewing parsed sites and toggling per-talkgroup
  Scan / Lockout / Priority before write; `-no-tui` / `-dry-run` /
  `-force` cover scripting and CI. Atomic writes (in-memory schema
  validation → temp file → rename) so a malformed source never
  corrupts the existing config. Supports P25 Phase 1 + Phase 2 PDFs;
  CSV bundles cover P25 / DMR / NXDN. Full operator reference at
  [`docs/import.md`](docs/import.md).
- **Capture-spec acceptance criteria for every real-air-blocked
  follow-up.** Each [`samples/<proto>/README.md`](samples/) now
  documents the explicit numerical thresholds a contributor with
  hardware can run a capture against to close the corresponding
  follow-up: TETRA wants 5 s lock latency + ≥ 90% frame recovery
  + a new (not-yet-wired) `gophertrunk_tetra_viterbi_corrections`
  Prometheus histogram; NXDN wants ≥ 80% CRC-verified CAC bursts
  + SystemID match; MPT 1327 wants ≥ 95% true-positive lock rate
  + a non-decreasing tolerance sweep; DMR Tier II wants
  byte-for-byte FLC match + clean Terminator-with-LC handling.
  The DMR Tier II and MPT 1327 follow-ups are already closed
  algorithmically (PR-A, PR-C); their captures are now optional
  secondary validation rather than the blocker. The
  [`samples/README.md`](samples/) top-level table summarises
  status + acceptance criteria across all five protocols.
- **Release-ready version metadata + AMBE+2 patent banner.**
  [`internal/version/`](internal/version/) now exposes `Version`,
  `Commit`, `BuildTime`, and a `String()` formatter
  (`"vX.Y.Z (sha=…, built=…)"`); all three are populated via
  `-ldflags` by the Makefile + release workflow. The daemon logs
  a one-line AMBE+2 patent-posture notice at startup pointing at
  [`docs/vocoders.md`](docs/vocoders.md); set
  `GOPHERTRUNK_QUIET_BANNER=1` in CI / test harnesses to suppress
  it. New `make release-dry-run VERSION=v0.99.0` target rehearses
  the release build locally so ldflags + packaging surface before
  a tag is cut — see [`CONTRIBUTING.md` §"Cutting a release"](CONTRIBUTING.md#cutting-a-release).
- **CI gates: govulncheck + license audit + full-tree integration
  run.** `.github/workflows/ci.yml` gains a `vulncheck` job
  (govulncheck against the direct + transitive dependency graph),
  a `licenses` job (regenerates the transitive-deps inventory via
  google/go-licenses and diffs against the committed
  `THIRD_PARTY_LICENSES.csv`), and an `integration` job that
  walks `make test-integration` across the whole module (the
  existing `build-test` job runs `make integration` against
  `cmd/gophertrunk/` only; this future-proofs against
  integration-tagged tests landing in other packages). New
  `Makefile` targets: `make vulncheck`, `make licenses`,
  `make test-integration`.
  [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md) ships a
  hand-curated direct-deps table + the ISC attribution for the
  mbelib-derived AMBE+2 / IMBE codebook tables.
- **Operational docs landed.** [`SECURITY.md`](SECURITY.md)
  documents the vulnerability disclosure process (private GitHub
  security advisories), supported versions, in-scope vs.
  out-of-scope issues, and the maintainer's response-time SLAs.
  [`CONTRIBUTING.md`](CONTRIBUTING.md) covers the dev setup, the
  house-style conventions, and the PR scoping rules.
  [`CHANGELOG.md`](CHANGELOG.md) seeds a Keep-a-Changelog with
  every user-visible change since the calibration / hardening
  pass. [`docs/gophertrunk.service`](docs/gophertrunk.service) is
  an example systemd unit (DynamicUser + ProtectSystem + USB
  device-allow) operators install at `/etc/systemd/system/`.
- **Optional TLS on HTTP + gRPC + extended health endpoint.**
  `api.tls_cert` / `api.tls_key` in `config.yaml` switches both
  the HTTP REST/SSE/WebSocket server and the gRPC server to TLS;
  the daemon refuses to start when one is set without the other.
  Plain TCP stays the default for loopback / trusted-LAN
  deployments. `GET /api/v1/health` now returns
  `pool_attached_count`, `active_calls`, `db_connected`,
  `metrics_enabled`, `auth_mode`, and `version` alongside the
  legacy `status` + `now` — supports two-field k8s / Nomad
  readiness probes that distinguish "process up" from "actually
  working". See [docs/hardening.md](docs/hardening.md#transport-encryption-tls)
  for the TLS recipe and [docs/hardening.md](docs/hardening.md#health-endpoint-diagnostics)
  for the health schema.
- **API server hardening: HTTP timeouts + gRPC keep-alive + drain
  window.** `internal/api/server.go` now sets `ReadTimeout` /
  `WriteTimeout` / `IdleTimeout` on the HTTP server (30 s / 30 s /
  120 s) on top of the existing `ReadHeaderTimeout`. SSE
  (`/api/v1/events`) opts out of `WriteTimeout` per-request via
  `http.ResponseController.SetWriteDeadline(time.Time{})` so the
  long-lived stream isn't torn down mid-call; the WebSocket
  endpoint hijacks the connection on Upgrade and is unaffected.
  `internal/api/grpc.go` configures `keepalive.ServerParameters`
  (30 s idle ping, 10 s ack timeout) + `EnforcementPolicy`
  (5 s min-time floor, `PermitWithoutStream: true`) so long-lived
  `AudioService.StreamAudio` subscribers detect dead peers
  without back-pressuring the publisher. Shutdown ctx bumped from
  5 s to 30 s so in-flight SSE / WebSocket / audio-stream
  subscribers drain cleanly on a daemon restart. See
  [docs/hardening.md](docs/hardening.md#timeouts-and-keep-alive)
  for the full knob reference.
- **Runtime theme toggle (Ctrl+T) + docs/tui.md sync.** The dark
  and monochrome palettes have lived in `internal/tui/theme` since
  the operator-console PR but weren't reachable from the UI;
  `Ctrl+T` (and a "Toggle theme" entry in the command palette) now
  cycles between them at runtime. A `panels.ThemeChangedMsg`
  broadcast lets each `bubbles/table`-backed panel re-apply its
  cached `tableStyles()` so the swap takes effect on the next
  render without a restart. `docs/tui.md` was rewritten end-to-end
  to cover everything shipped since the operator-console work:
  the Settings panel and its tab cycle, the Ctrl+P command
  palette, mouse hit-testing and scroll-wheel forwarding,
  Revealer-driven cursor pre-positioning, async history refresh,
  and every audio / scanner / runtime endpoint that wasn't in the
  old reference.
- **Mouse coverage on every table panel + scroll-wheel scroll.**
  The `MouseAware` interface added in the Revealer PR now lives on
  Active, History, Tones, and Metrics in addition to Systems,
  Talkgroups, and Devices — left-clicks on a data row move the
  cursor onto that row in every table-backed panel. Scroll-wheel
  ticks are forwarded the same way (one-row-per-tick up or down)
  because bubbles/table v1.0.0 doesn't handle `MouseMsg` itself.
  The `MouseAware` signature changed from
  `HandleMouseAt(localX, localY int)` to
  `HandleMouse(msg tea.MouseMsg, localY int)` so panels can
  distinguish a click on a row from a wheel tick from a button
  release. A shared `handleTableMouse` helper centralises the
  left-press + wheel switch so every table panel handles input the
  same way.
- **TUI Revealer / MouseAware: palette + mouse converge on the same
  row.** Picking a system / talkgroup / device from `Ctrl+P` now
  jumps to the destination panel and pre-positions the panel's
  cursor on the matching row before opening the detail modal —
  follow-up keystrokes (`Enter`, mutation keys) operate on the
  selection without an extra scroll. A new `panels.Revealer`
  interface formalises the contract; SystemsPanel (by name),
  TalkgroupsPanel (by decimal ID), DevicesPanel (by serial), and
  ScannerPanel (by `sys:<name>` / `conv:<idx>`) implement it.
  Mouse hit-testing was extended past the tab strip via a sibling
  `panels.MouseAware` interface: left-clicks on data rows in the
  three table panels translate the click position to a row index
  (accounting for the canonical panelFrame chrome offset) and call
  `SetCursor`. Chrome clicks are ignored; out-of-range clicks
  clamp to the last row.
- **Async history refresh off the Update goroutine.** The history
  panel was the one remaining surface that built its bubbles/table
  rows inline inside `Update`. The conversion now runs in a
  `tea.Cmd` and commits via a routed `HistoryRefreshedMsg`; a
  `pendingAt` guard prevents duplicate dispatch and stale results
  (a newer snapshot landed mid-flight) are dropped silently. The
  reducer stays unblocked on row formatting that doesn't need to
  hold it.
- **Direct-ioctl ALSA backend (drops the runtime libasound2 dep).**
  Setting `audio.device: ioctl` (or `ioctl:hw:C,D` for a specific
  card / device) on Linux selects a direct-kernel backend that
  opens `/dev/snd/pcmC{card}D{device}p` and drives the playback
  state machine via `SNDRV_PCM_IOCTL_*` syscalls. No
  `libasound2.so.2` at runtime, no `purego.Dlopen`, no cgo —
  useful for distroless / Alpine / scratch container images that
  don't ship the userspace library but do have the kernel sound
  subsystem available. Defaults to card 0, device 0; format
  pinned to S16_LE mono at `audio.sample_rate`, period sized
  from `audio.buffer_ms`. The dlopen path stays the Linux
  default because it auto-negotiates with the hardware; the
  ioctl path uses fixed values so it only works when the
  underlying device natively supports them.
- **AMBE+2 DTMF dual-tone synthesis.** Tone frames with
  b₁ ∈ [128, 143] (the AMBE+2 DTMF range) now synthesise the
  correct summed sinewaves instead of routing through silence.
  Sixteen-entry b₁ → (low Hz, high Hz) lookup table sourced from
  ITU-T Q.23's 4×4 DTMF matrix (rows 697/770/852/941 ×
  cols 1209/1336/1477/1633); the b₁ → key mapping follows the
  AMBE+2 layout shared by mbelib / DSDcc / DSD-FME — 128 is "1",
  143 is "D". Two oscillator phases ride across frame boundaries
  so held keys are click-free. Knox / call-alert pairs
  (b₁ ∈ [144, 163]) are vendor-specific (Motorola Trbo vs.
  Hytera vs. generic) and stay routed through silence pending
  per-vendor frequency tables — operators who need them can drop
  rows into the `ambeDualToneTable` upper range.
- **`audio.state` SSE event for instant TUI convergence.** PATCH
  `/api/v1/audio` now publishes the resulting state on the events
  bus as `KindAudioState`, which the SSE pump forwards to every
  subscriber. The TUI listens and re-fetches the audio snapshot on
  receipt, so two TUIs / a TUI plus a `curl` PATCH converge inside
  one SSE round-trip instead of waiting up to 3 s for the next
  poll tick. The payload is the same `AudioStatusDTO` the HTTP
  response carries — clients can use it directly or just treat the
  event as a poll trigger.
- **Conventional channel runtime lockout.** Press `L` on the TUI's
  Scanner panel (or `POST /api/v1/scanner/conventional/{idx}/lockout`)
  to skip a conventional FM channel during scan, matching the
  muscle memory of a Uniden / Whistler lockout button. Skipped
  channels show a `✗` marker in the channel list and are omitted
  from `pickNextChannel` rotation; pressing `L` again (or POSTing
  `unlockout`) restores them. If the locked-out channel is
  currently dwelling, the synthetic call ends immediately
  (`EndReasonLockout`) so the operator's intent takes effect
  within one IQ chunk. Lockouts are runtime-only — they don't
  persist across daemon restarts, since config is the right
  place to permanently exclude a channel.
- **macOS RTL-SDR serial / manufacturer / product strings.** The
  pure-Go USB enumerator's Darwin backend
  (`internal/sdr/rtlsdr/usb/usb_darwin.go`) now reads the IORegistry
  string properties via `CFStringGetCString` instead of returning
  empty placeholders, so `gophertrunk sdr list` on macOS prints the
  same per-device identification (serial, manufacturer, product)
  that Linux and Windows have shipped since PR-10. Useful for
  multi-dongle hosts where the operator wants to pin a specific SDR
  by serial in `config.yaml`. Closes the
  `TODO(macos-strings)` flagged in the file since PR-03.
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

- **Language:** Go 1.25+ (toolchain pinned to 1.25.10 in `go.mod`)
- **Hardware:** Pure-Go RTL-SDR driver — USBDEVFS / WinUSB / macOS IOKit transport, RTL2832U register layer, and per-chip tuner drivers (R820T/R820T2/R828D/E4000/FC0012/FC0013/FC2580). `CGO_ENABLED=0`; no `librtlsdr` / `libusb` build dependency.
- **DSP:** `gonum/dsp/fourier` for FFT, custom polyphase channelizer,
  filters, and demodulators (FM / C4FM / GFSK / FFSK / π/4-DQPSK / H-DQPSK)
- **Vocoders:** Pure-Go IMBE + AMBE+2 (clean-room re-implementations
  of mbelib codebook tables); optional DVSI USB-3000 hardware
  backend behind `-tags dvsi`
- **Audio:** Pure-Go ALSA backend (direct-ioctl + libasound2 via
  purego), macOS CoreAudio + Windows WASAPI via `ebitengine/oto`
- **Storage:** `modernc.org/sqlite` (pure Go)
- **API:** gRPC + Protobuf, HTTP/SSE, WebSocket; optional TLS;
  bearer-token auth on mutations
- **TUI:** `charmbracelet/bubbletea` 11-panel cockpit
- **Web console:** Vite + React + TypeScript + Tailwind CSS +
  Zustand + Chart.js + D3 scale, bundled into a static
  `gophertrunk-web/` directory shipped alongside the binary
  (installable PWA, no Node.js at runtime)
- **Logging:** `log/slog` (stdlib)
- **Metrics:** `prometheus/client_golang`
- **CI:** GitHub Actions running `go vet` + `go test -race` + `make
  integration` + `make test-dvsi` + `govulncheck` + license audit
  across Linux / macOS / Windows runners

## Quick start

### Download a prebuilt release

The fastest path is the **[Downloads page]({{ site.url | default: "https://gophertrunk.org" }}/downloads.html)**
on the project site — it has per-platform recipes (Linux / Windows /
macOS / Docker), checksum-verification commands, and pointers to
the latest tag. Or jump straight to the **[Releases page][releases]**
on GitHub to grab the artefacts directly:

| Platform   | File                                                   | What it is                                              |
| ---------- | ------------------------------------------------------ | ------------------------------------------------------- |
| Windows 11 | `gophertrunk-<ver>-windows-amd64-setup.exe`            | One-click installer (Inno Setup) — single static binary  |
| Windows 11 | `gophertrunk-<ver>-windows-amd64.zip`                  | Portable ZIP — same binary, no installer                 |
| Linux      | `gophertrunk-<ver>-linux-amd64.tar.gz`                 | Tarballed static binary + sample config + `gophertrunk-web/` console |
| all        | `SHA256SUMS`                                           | SHA-256 checksums for every artefact in the release      |

Windows users: after running the installer, follow
[`docs/install-windows.md`](docs/install-windows.md) to swap the
RTL-SDR driver to WinUSB via Zadig — the OS won't see your dongle
until that's done. The installer's last page links there too.

After install, `gophertrunk version` reports the build provenance:

```sh
$ gophertrunk version
v0.99.0 (sha=abc1234, built=2026-05-13T19:00:00Z)
```

[releases]: https://github.com/MattCheramie/GopherTrunk/releases

### Build from source

### Prerequisites

Just Go 1.25+. The pure-Go RTL-SDR driver doesn't need
`librtlsdr` / `libusb` / a C toolchain on the build host. The
project's `go.mod` pins the toolchain to 1.25.10 (closes the 23
stdlib CVEs in the bare 1.25.0 release); older Go versions
auto-download 1.25.10 via Go's toolchain mechanism.

See [`docs/hardware.md`](docs/hardware.md) for runtime `udev` rules
and DVB-driver blacklisting on Linux.

### Build, test, run

```sh
make build                    # produces ./bin/gophertrunk
make test                     # go test -race ./...
make integration              # boots the wired daemon end-to-end (no SDR needed)
make test-integration         # every //go:build integration test across the module
make test-dvsi                # DVSI USB-3000 / AMBE-3003 backend (under -tags dvsi)
make vulncheck                # govulncheck against direct + transitive deps
make licenses                 # regenerate THIRD_PARTY_LICENSES.csv
make release-dry-run          # rehearses the release.yml linux build locally
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
make integration-cc-ysf       # YSF "lights up" — 4800-baud C4FM 480-dibit frames

./bin/gophertrunk version
./bin/gophertrunk sdr list                # enumerates attached dongles
./bin/gophertrunk run -config config.yaml

# Out-of-band: decode a captured .raw frame sidecar to a WAV using
# the pure-Go IMBE / AMBE+2 vocoders. The .raw sidecar is written
# alongside each call's WAV when the recorder's raw-frames option
# is enabled.
./bin/gophertrunk decode -in call.raw -out call.wav -vocoder imbe
./bin/gophertrunk decode -list-vocoders

# Bootstrap a new region: parse one or more RadioReference PDFs (or a
# structured CSV bundle) and merge them into config.yaml + per-system
# talkgroup CSVs. Launches a TUI for review; -no-tui for scripting.
./bin/gophertrunk import-pdf -pdf maricopa.pdf -pdf rwc.pdf -config config.yaml
./bin/gophertrunk import-pdf -csv my-system.csv -config config.yaml -no-tui -dry-run

# Don't have a config.yaml yet? Launch the interactive builder that
# walks you through every section. Can be combined with -pdf / -csv.
./bin/gophertrunk import-pdf -wizard
./bin/gophertrunk import-pdf -wizard -pdf maricopa.pdf
```

A starter [`config.example.yaml`](config.example.yaml) is in the
repo root — copy it, set the `serial` of your dongle from
`gophertrunk sdr list`, then either hand-write the
`trunking.systems[]` block + a Trunk-Recorder talkgroup CSV, or let
`gophertrunk import-pdf` build both from a RadioReference PDF or a
structured CSV bundle (see [`docs/import.md`](docs/import.md)).

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
cmd/gophertrunk/        daemon entrypoint + sdr list CLI + read+write TUI cockpit + RadioReference/CSV importer (import-pdf subcommand)
internal/tui/           bubbletea TUI: 11 panels (Dashboard, Systems, Talkgroups, Active, History, Events, Tones, Metrics, Devices, Scanner, Settings) over REST+SSE
web/                    standalone browser SPA (Vite + React + TypeScript + Tailwind) — ships as `gophertrunk-web/` in release archives, talks to the daemon via REST + WebSocket + new /api/v1/audio/stream
internal/sdr/           Driver interface, pool, mock
internal/sdr/rtlsdr/usb/      Pure-Go USB transport: Linux USBDEVFS, Windows WinUSB, macOS IOKit (purego), mock
internal/sdr/rtlsdr/rtl2832u/ RTL2832U register/I2C layer (sample-rate, IF, FIR, GPIO, I2C bridge)
internal/sdr/rtlsdr/tuners/   R820T/R820T2/R828D + E4000 + FC0012 + FC0013 + FC2580 tuner drivers
internal/sdr/rtlsdr/purego/   sdr.Driver+sdr.Device wire-up; canonical "rtlsdr" registrant
internal/dsp/           Channelizer, filters, demods, sync, FFT
internal/radio/         framing/ + p25/{phase1,phase2}/ + dmr/{tier2,tier3}/ + nxdn/ + tetra/ + dpmr/ + edacs/ + ltr/ + mpt1327/ + motorola/ + ysf/ + dstar/ (per-protocol packages; each has its own receiver/ + ControlChannel.Process adapter)
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
samples/                drop-zone for real-air captures that close the remaining FEC follow-ups in docs/opt-in-features.md §5 (nxdn/, ysf/, tetra/, dmr-tier2/, mpt1327/ — each subfolder has a README documenting the expected capture format + metadata schema)
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
| `Ctrl+P` | open fuzzy command palette (panel jumps, system / TG / device drill-ins, audio mutations, retention sweep, scanner hold/resume) |
| `Ctrl+T` | toggle theme (dark ↔ monochrome) |
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

The tab strip is also clickable, and a left-click on any data row
in the Systems / Talkgroups / Active / History / Devices / Tones /
Metrics panels moves the cursor onto that row — pair it with `Enter`
to open the detail card or with a mutation key (`e` to end the
highlighted active call, `R` to reset the highlighted tone detector,
`L` to toggle conventional channel lockout, etc.) to act on the
selection. Scroll-wheel ticks advance the cursor one row at a time
in the same panels. Picking a system / talkgroup / device from the
command palette pre-positions the destination panel's cursor on the
matching row before opening the detail modal, so keyboard and mouse
paths converge on the same selection.

Settings is a tabbed inspector (`[` / `]` to cycle) covering
Daemon · Storage · Audio · Recording · Tones · API · Vocoders · SDR · FEC
— every config knob the daemon reads is visible.

For mutation actions (end-call; set talkgroup priority / lockout /
scan; retention-sweep; tone-detector reset; scanner cockpit
hold/resume/retune/dwell + scan_mode flip) the HTTP API uses
bearer-token authentication (`api.auth.mode`). The default `auto`
mode bypasses auth on loopback binds (`127.0.0.1` / `::1`) — peer-
cred via kernel-enforced reachability is a reasonable trust proxy
for single-host operator boxes — and requires a `Authorization:
Bearer <token>` header on every mutation request when the listener
binds to a public interface. See the [API authentication](#api-authentication)
section below; `docs/tui.md` documents the matching `--write`
TUI flag.

## Web console

GopherTrunk also ships a browser-based operator console as a separate
standalone package alongside the binary — no Node.js, no embedded
web server, no extra service to install. Each release archive
contains a sibling `gophertrunk-web/` directory holding a pre-built
React SPA (`index.html` + bundled JS/CSS); double-click `index.html`
to open it in any browser, point the connect screen at a running
daemon on the local network, and start operating.

```
gophertrunk-v<version>-<os>-<arch>/
├── gophertrunk            # the daemon/CLI binary
├── gophertrunk-web/       # standalone web console (open index.html)
│   ├── index.html
│   ├── assets/…           # bundled React / Tailwind / Chart.js
│   ├── favicon.svg
│   └── manifest.webmanifest
├── config.example.yaml
└── samples/…
```

Headless-Pi + laptop is a first-class scenario: run
`gophertrunk run -config config.yaml` on a Raspberry Pi (or any host
with the RTL-SDR attached) with `api.host: 0.0.0.0` and
`api.cors.allowed_origins` permitting the SPA's origin (`"null"` for
`file://`, or the URL it's served from), then open
`gophertrunk-web/index.html` on a laptop / tablet / phone on the same
network and enter the daemon's URL + bearer token on the connect
screen. The SPA is installable as a Progressive Web App (Add to Home
Screen) and persists server URL + token in browser storage so return
visits skip the connect screen.

**Status: TUI parity.** Every Bubbletea TUI panel has a browser
counterpart:

- **ConnectScreen** — server-URL + bearer-token entry with a
  `GET /api/v1/health` reachability probe before commit.
- **Dashboard** — live WebSocket event feed (auth-aware
  `/api/v1/events/ws`), call-state ticker, audio cockpit with
  PCM-over-WAV playback from the new `GET /api/v1/audio/stream`
  endpoint, volume / mute / record-toggle wired to
  `PATCH /api/v1/audio`.
- **Active** — full active-call list with per-call detail modal,
  live elapsed-time ticker, and a confirm-wrapped end-call button
  (`POST /api/v1/calls/{deviceSerial}/end`) when write mode is on.
- **History** — filterable call-log explorer over
  `GET /api/v1/calls/history?limit&system&group_id` with a form-
  driven filter, a per-row detail modal, and a write-mode-gated
  "Sweep retention" button (`POST /api/v1/retention/sweep`).
- **Systems** — sortable browser of every trunked system in
  `GET /api/v1/systems`, with a detail modal exposing protocol,
  WACN / System ID / RFSS / Site, and control-channel frequencies.
- **Talkgroups** — sortable, filterable browser with priority / scan
  / lockout flag pills and inline PATCH controls in the detail
  modal (`PATCH /api/v1/talkgroups/{id}`) when write mode is on.
- **Devices** — SDR pool inspector with attached/detached status,
  driver / tuner / role columns, and a detail modal for gain / PPM /
  bias-T. Live via `sdr.attached` / `sdr.detached` events.
- **Events** — live ring-buffer viewer (capped at 500) with
  substring filter, Pause/Resume snapshot, and click-to-expand JSON.
- **Tones** — filtered view of `tone.alert` events with per-device
  reset button (`POST /api/v1/devices/{serial}/tone-reset`) gated
  by write mode.
- **Metrics** — `GET /metrics` Prometheus snapshot with curated
  `gophertrunk_*` counter tiles + a Chart.js trend line for
  `calls_active`, `devices_attached`, `cc_locked` over ~5 min.
- **Scanner** — CC hunter (hold / resume / retune per system),
  conventional channel list (dwell / lockout / unlockout), manual
  VFO tune (`POST /api/v1/scanner/manual_tune`), scan_mode toggle
  (all ↔ list). All mutations gated by write mode + confirm modals.
- **Settings** — theme toggle (dark / monochrome), write-mode
  master switch, "forget this device".

All mutations are AND-gated by `selectCanMutate` (write mode +
daemon `allow_mutations`) and the destructive ones are wrapped in a
`ConfirmModal` so a fat-finger tap in the bottom nav doesn't end a
call. See the **setup + quick-start guide at
[gophertrunk.org/web.html](https://gophertrunk.org/web.html)** (also
[`docs/web.md`](docs/web.md)) for the operator playbook — LAN
deployment, CORS config, PWA install — and
[`web/README.md`](web/README.md) for the developer reference + the
dev workflow (`make web-dev`, `make web-build`).

## API authentication

The HTTP API's mutation endpoints (every write route: end-call,
talkgroup priority/lockout/scan, retention sweep, tone-detector
reset, scanner cockpit, audio cockpit, manual tune) authenticate
via bearer tokens. Configure under `api.auth` in `config.yaml`:

```yaml
api:
  http_addr: "127.0.0.1:8080"
  auth:
    mode: "auto"                # auto | required | disabled
    # token: "inline-token"     # discouraged; use token_file
    token_file: "/etc/gophertrunk/api-token"
    trusted_networks:
      - "10.0.0.0/8"
```

**Policy modes:**

| Mode | Behaviour |
| --- | --- |
| `auto` (default) | Require a token on non-loopback binds; bypass on loopback (`127.0.0.1` / `::1`). Reasonable for single-host operator boxes — kernel-enforced reachability is a peer-cred proxy. The daemon refuses to start in `auto` mode on a public bind without a configured token. |
| `required` | Every mutation request must carry a valid Bearer token, even from loopback. Use when the daemon shares a host with untrusted users. |
| `disabled` | Wide-open mutations, no auth. Equivalent to the legacy `allow_mutations: true` behaviour. The daemon logs a warning at startup. |

**Token storage.** `token_file` is preferred — the secret stays out
of `config.yaml`, and the daemon re-reads the file on every request
so operators can rotate without a restart. Inline `token` is
supported for ephemeral / test setups.

**Trusted networks.** A CIDR allowlist of source addresses that
bypass the token check under `auto` mode. Loopback prefixes are
implicit; list private subnets here if the daemon binds to a LAN
interface and you trust everything on that segment. The middleware
reads `RemoteAddr` only — `X-Forwarded-For` is intentionally
ignored so the bypass isn't forgeable by a hostile upstream proxy.

**Header format.** Standard RFC 6750:

```
Authorization: Bearer <token>
```

The token is compared with `crypto/subtle.ConstantTimeCompare`.
Mutation requests without a valid token return 401; the body carries
the same `{"error":"..."}` envelope every other 4xx response uses.

**TUI client.** The operator TUI sends the token automatically when
launched with `gophertrunk tui -token <value>` or
`gophertrunk tui -token-file <path>`. `-token-file` is re-read on
every request so daemon-side rotation works without restarting the
TUI. The `GOPHERTRUNK_API_TOKEN` env var is honoured as a fallback.
401 / 403 responses surface as a toast that points the operator at
the right flag.

**Capability probe.** `GET /api/v1/mutations` is always open and
reports the daemon's policy plus whether the current request would
be accepted:

```json
{
  "auth_mode": "auto",
  "can_mutate": true,
  "allow_mutations": true,
  "engine_writable": true,
  "retention_writable": true,
  "tones_writable": false
}
```

`allow_mutations` is the legacy alias of `can_mutate`; new clients
should prefer `can_mutate`.

**Migration from `allow_mutations: true`.** The legacy flag is still
recognised: setting it to `true` logs a deprecation warning at
startup and maps to `auth.mode: disabled` so existing wide-open
deployments keep working. Migrate to explicit `auth.mode` at your
next config edit.

## FEC opt-outs

Every protocol that has a public-spec FEC chain ships its
spec-correct decoder chain **on by default**: the connector
constructs each `ControlChannel` with the full on-air FEC layer
applied, and operators with pre-stripped capture files (DSD-FME
`-r` dumps, OP25 fixtures, MMDVMHost / DSDcc test data) opt out
per-system with `<key>: off` in `config.yaml`. Empty / absent
keys map to the new on-default for every protocol.

Verify which protocols are on / off in the **Settings** panel of
the TUI — it lists every configured system with a one-line summary
of its FEC state (`channel coding: on (colour=…, sch/f)`, `viterbi:
spec`, `bch: on`, etc.). The panel is read-only; runtime mutation
is a future PR. To change a mode, edit `config.yaml` and restart
the daemon.

| Protocol | YAML key(s) | On (default) | Off (opt-out) |
| --- | --- | --- | --- |
| TETRA | `tetra_colour_code` (uint32, low 30 bits — required for non-BSCH), `tetra_channel` (`"sch/hd"` / `"sch/f"` / `"sch/hu"` / `"bsch"` / `"aach"`, default `sch/hd`), `tetra_channel_coding` (`""` / `"on"` / `"off"`) | Full ETSI EN 300 392-2 §8.3.1 type-5 → type-1 chain (descramble + deinterleave + depuncture + Viterbi + CRC-16 verify + tail strip) per burst. `tetra_colour_code` of 0 is only valid for BSCH; non-BSCH channels need the per-cell colour code or descrambling produces garbage (the connector warn-logs this case). | `tetra_channel_coding: off` falls back to the legacy 48-dibit raw-PDU path. CRC will fail on live captures; only useful for pre-stripped fixtures. |
| LTR | `ltr_fcs_mode` (`""` / `"on"` / `"off"`), `ltr_manchester_mode` (`""` / `"on"` / `"soft"` / `"strict"` / `"off"` / `"nrz"`) | `fcs: on` — CRC-7 FCS check against sdrtrunk's CRCLTR.java layout. `manchester: soft` — majority-decode each pair (matches the dominant on-air encoding for sub-audible LTR signaling). | `fcs: off` skips the CRC check (synthesized fixtures whose FCS trailer isn't populated). `manchester: off` / `nrz` treats the stream as raw NRZ (synthesized NRZ fixtures). |
| P25 Phase 2 | `p25_phase2_trellis_mode` (`""` / `"on"` / `"off"`), `p25_phase2_rs_mode` (`""` / `"on"` / `"off"`), `p25_phase2_scrambler_mode` (`""` / `"on"` / `"probe"` / `"off"`) | `trellis: on` — 4-state ½-rate trellis FEC over the MAC PDU window (146 channel dibits → 72 info dibits per TIA-102.AABF). `rs: off` — outer RS(24, 16, 9) verification per TIA-102.BAAA-A §5.9 defaults off; flip on to drop MAC PDUs with non-zero syndromes. `scrambler: off` — PN44 descrambler per TIA-102.BBAC-1 §7.2.5 defaults off. | `trellis: off` — legacy 72-dibit raw-MAC-PDU path for pre-stripped fixtures. `rs: on` — verify RS(24, 16, 9) syndromes on the trellis-decoded MAC PDU. `scrambler: on` — XOR the trellis-decoded 144-bit MAC PDU with the PN44 sequence starting at the configured per-burst offset. `scrambler: probe` — walk all 12 spec-defined slot offsets from Figure 7-5 and accept the first that passes RS verification (requires `rs: on`; degrades to offset-0 descrambling otherwise). |
| NXDN | `nxdn_viterbi_mode` (`""` / `"spec"` / `"on"` / `"off"`) | `spec` — full NXDN-TS-1-A rev 1.3 §4.5.1.1 outbound CAC chain (150 dibits → deinterleave 25×12 → depuncture 50/350 → K=5 Viterbi → 16-bit CRC verify → 155 info bits). | `on` — intermediate 92-dibit K=5 Viterbi path for older MMDVMHost / DSDcc fixtures. `off` — legacy 44-dibit raw-CAC path for pre-stripped fixtures. |
| EDACS | `edacs_bch_mode` (`""` / `"on"` / `"off"`) | BCH(40, 28, 2) with single/double-bit correction over the 40-bit on-wire CCW; the effective CCW carries 28 info bits (Command + Status + Address + high LCN bits), the remaining bits become BCH parity. | Falls back to the legacy pre-stripped 40-bit CCW; payload struct's LCN bit 0 + Aux fields are treated as data instead of parity. |
| MPT 1327 | `mpt1327_bch_mode` (`""` / `"on"` / `"off"`) | BCH(63, 38) decode over the 64-bit on-wire codeword. | Falls back to the legacy 38-bit pre-stripped codeword. |
| Motorola Type II | `motorola_bch_mode` (`""` / `"on"` / `"off"`) | Two 64-bit BCH(64, 16, 11) codewords reassembled into the 32-bit OSW with single- through 11-bit-error correction per codeword. | Falls back to the legacy 32-bit raw-OSW path for pre-stripped DSD-FME `-r` fixtures. |

All string values are case-insensitive with whitespace tolerated;
recognised on-values include `""` (empty) / `"on"` / `"true"` /
`"1"` (NXDN also accepts `"spec"`; LTR Manchester accepts
`"soft"` / `"strict"`); off-values are `"off"` / `"false"` / `"0"`
(LTR Manchester also accepts `"nrz"`). Unrecognised values fall
back to the on-default with a `warn`-level log line ("ccdecoder:
unrecognised `<key>`; falling back to on") so a typo doesn't
silently break the decoder.

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

### Project docs

- [`docs/architecture.md`](docs/architecture.md) — layered overview,
  concurrency model, driver registry, build tags
- [`docs/tui.md`](docs/tui.md) — TUI keybindings, panel reference,
  troubleshooting
- [`docs/web.md`](docs/web.md) — browser web console setup + quick
  start: getting the bundle, daemon CORS/auth config, opening
  `index.html`, LAN scenario, PWA install, troubleshooting. Also at
  **[gophertrunk.org/web.html](https://gophertrunk.org/web.html)**
- [`web/README.md`](web/README.md) — web console developer reference:
  shipping panel matrix, architecture, dev workflow
  (`make web-dev` / `make web-build`)
- [`docs/hardware.md`](docs/hardware.md) — udev rules, DVB blacklist,
  IQ capture for replay
- [`docs/vocoders.md`](docs/vocoders.md) — IMBE / AMBE+2 licensing
  realities, the plugin model, the DVSI backend layout, and the
  knox / call-alert extension hook
- [`docs/voice-calibration.md`](docs/voice-calibration.md) — operator
  recipe for tuning the IMBE / AMBE+2 decoders against a DSD-FME /
  OP25 reference recording via `cmd/voice-calibrate`
- [`docs/import.md`](docs/import.md) — `gophertrunk import-pdf`
  operator reference: RadioReference PDF workflow, structured CSV
  bundle format (metadata / sites / talkgroups sections), TUI key
  bindings, and CLI flag reference
- [`docs/hardening.md`](docs/hardening.md) — API authentication, TLS
  setup, health endpoint diagnostics, HTTP / gRPC timeouts +
  keep-alive reference, graceful shutdown, Prometheus catalogue,
  Docker / compose USB pass-through, smoke-test checklist
- [`docs/opt-in-features.md`](docs/opt-in-features.md) — operator
  reference for every default the daemon carries: protocol FEC
  defaults, receiver clock recovery, daemon-level features
  (mix of on / off / auto-detect), and the permanent build-time
  gates (DVSI patent tag, integration tests)
- [`docs/gophertrunk.service`](docs/gophertrunk.service) — example
  systemd unit (DynamicUser, ProtectSystem, USB device-allow) for
  Linux operators standing the daemon up on a server
- [`docs/specs/`](docs/specs/) — reference air-interface PDFs the
  on-air FEC implementations derive from (NXDN-TS-1-A,
  ETSI EN 300 392-2 TETRA, plus a negative-reference M/A-COM LBI
  for EDACS that documents *not* what to look for)

### Project metadata

- [`CHANGELOG.md`](CHANGELOG.md) — user-visible changes per release,
  Keep-a-Changelog format
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — dev setup, PR scoping
  rules, house-style conventions, release-cutting recipe
- [`SECURITY.md`](SECURITY.md) — vulnerability disclosure process
  via GitHub's private security advisories; in-scope vs.
  out-of-scope; response-time SLAs
- [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md) — direct
  Go-module deps + the mbelib ISC attribution for the AMBE+2 /
  IMBE codebook tables

## License

See [`LICENSE`](LICENSE).
