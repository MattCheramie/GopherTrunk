# Changelog

All notable user-visible changes land here, newest first.
Format adapted from [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
for tagged releases.

## [Unreleased]

### Added

- **Outbound call streaming to aggregators and live audio
  servers.** Completed calls are now encoded to MP3 and streamed
  to external services, closing the largest functional gap
  against SDRtrunk. A new `internal/broadcast` subsystem
  subscribes to a `KindCallComplete` event the recorder
  publishes once a call's WAV is flushed, encodes the audio via
  a pure-Go MP3 encoder (`internal/voice/mp3`, no CGO), and
  fans the call out to every configured backend with bounded
  exponential-backoff retry. Four backends ship: Broadcastify
  Calls (two-step metadata + audio upload), RdioScanner
  (native call-upload API), OpenMHz, and live Icecast/ShoutCast
  (a continuous paced source connection topped up with silence
  between calls). Feeds are configured under a new `broadcast:`
  config section; each feed takes an optional `systems:` filter
  and a talkgroup can opt out of all feeds with `stream: false`
  in its CSV/JSON. Feed counters are exposed at
  `GET /api/v1/broadcast`.
- **P25 Phase 1 voice decoding and broader control-channel
  coverage** (PR #310). A `p25` voice grant now decodes
  end-to-end — modulated C4FM IQ → Phase 1 receiver → LDU
  assembly → IMBE voice frames → WAV; the composer previously
  bypassed the P25 Phase 1 voice path and produced no audio.
  The control-channel decoder gains wider TSBK grant coverage
  (unit-to-unit voice grant, explicit/implicit group update,
  telephone-interconnect grant, SNDCP data-channel grant),
  manufacturer-specific TSBK dispatch by MFID (Motorola /
  Harris group-regroup, multi-fragment vendor talker alias),
  LDU1 Link Control and LDU2 Encryption Sync decode (algorithm
  and key ID surfaced — identify, not decrypt), a `NetworkModel`
  that accumulates system topology (WACN, RFSS / site IDs,
  secondary control channels, neighbour sites), and a
  packet-data decode layer (PDU reassembly → SNDCP → IPv4
  header). Patch / regroup and talker-alias announcements
  publish through the new `KindPatch` / `KindTalkerAlias` event
  kinds.
- **P25 Phase 2 TDMA decode path** (PR #308, #309). P25 Phase 2
  grew from a control-channel-only stub into a full TDMA
  decoder. A `SuperframeDecoder` locks the 360 ms superframe and
  slices its 12 sub-frames; SlotType decode separates voice from
  MAC sub-frames; `ExtractVoiceFrames` pulls AMBE+2 frames from
  4V / 2V voice slots; and a composer voice chain decodes a
  `p25-phase2` grant end-to-end (modulated IQ → receiver →
  superframe decode → AMBE+2 → WAV). The live control-channel
  pipeline now runs through the structured `SuperframeDecoder`.
  Parity additions: encryption identification (`Encrypted` /
  `Emergency` / `AlgorithmID` / `KeyID` on the grant), Motorola /
  Harris patch / regroup feeding an engine `PatchRegistry`,
  multi-fragment talker-alias reassembly, band-plan
  channel-to-frequency resolution, MFID-keyed vendor MAC
  dispatch, and the opt-in TIA-102.BBAC per-burst block
  deinterleaver (`p25_phase2_interleave_mode`). Phase 2 now
  emits `KindAffiliation` / `KindUnitRegistration` / `KindPatch`
  / `KindTalkerAlias` like Phase 1.
- **P25 Phase 1 CQPSK / LSM demodulator path** for simulcast P25
  sites (issue #275). New per-system YAML key
  `p25_phase1_demod_mode: cqpsk` routes the control-channel IQ
  through a complex RRC matched filter + Gardner timing recovery +
  differential QPSK quadrant decode with LSM dibit remap, replacing
  the FM-discriminator + 4-level slicer path that produces near-
  random dibits on Linear Simulcast Modulation. The C4FM path stays
  the default for conventional non-simulcast deployments. Pipeline
  construction now logs `ccdecoder: p25/phase1 pipeline configured
  demod=…` so operators can confirm which path is active.
- **Multi-rotation FSW search** on the P25 Phase 1 sync detector.
  `SyncDetector.ProcessWithRotation` tries all four cyclic shifts
  of the dibit alphabet against the canonical FrameSyncWord and
  returns the rotation that matched, absorbing residual symbol-
  polarity / I-Q-swap ambiguity. The downstream control-channel
  parser inverts the rotation before NID BCH + TSBK trellis decode.
  Rotation=0 wins on ties so existing clean-fixture tests stay
  green.

### Fixed

- **Web console WebSocket reconnect storm and intermittent
  crash** (issue #290, PR #302). The event-stream client reset
  its reconnect backoff the instant a socket opened, so a
  connection that opened then dropped immediately
  reconnect-stormed at the floor delay forever; the backoff now
  resets only after a connection holds open for a stability
  window, and reconnect delays carry equal jitter. Socket
  teardown nulls every handler and gates status writes behind a
  `closed` flag, so a late event from an in-flight socket can no
  longer write to the store after teardown and trip a React
  render crash. The health-check and event-stream effects are
  keyed on the primitive server URL / token values instead of a
  derived object so they re-run only on a real server change.
- **P25 Phase 1 CQPSK control channel locked only in a narrow
  RTL-SDR gain window** (issue #275, PR #307). The CMA blind
  equalizer added for simulcast P25 made the CQPSK path
  gain-sensitive: the Gardner timing-error detector and the CMA
  weight update both use un-normalised, amplitude-dependent
  error terms, so the chain converged only when the signal sat
  in a narrow amplitude band. An AGC on the matched-filter
  output now normalises every capture to the level the Gardner
  and CMA loops are tuned for, restoring scale invariance
  regardless of front-end gain. `dsp.AGC` was reworked from a
  per-sample feedback loop — which spiked into gain runaway on a
  near-zero symbol of a linear-modulation stream — into a robust
  power-EMA feed-forward normaliser.
- **Trunked control-channel decode on live RTL-SDR hardware**
  (issue #275). The ccdecoder fed every per-protocol receiver the
  full, un-channelized SDR IQ stream (commonly 2.048 MHz), so the
  matched filter + symbol-clock loop ran at ≈427 samples per symbol
  against a ±1 MHz swath and the Frame Sync Word never correlated —
  no protocol could lock on-air, regardless of gain, PPM, or demod
  mode. A digital down-converter now decimates each raw IQ chunk
  (rational polyphase resample) to the narrowband channel rate the
  per-protocol receivers are matched-filter-tuned for — ~48 kHz for
  the 4800-baud C4FM family, 144 kHz for TETRA — before the pipeline
  sees it. The IQ-power gauge still reports the raw SDR input level.
- **P25 Phase 1 control channel never locked on live SDR chunking**
  (issue #275). The control-channel state machine discarded every
  Frame Sync Word hit unless the whole 154-dibit frame (FSW + NID +
  TSBK) fell inside a single `Process` call. A live RTL-SDR delivers
  16 KiB USB transfers — only ~19 P25 symbols per call — so the NID
  never fit and the channel never locked, even with the IQ stream
  correctly channelized. `ControlChannel.Process` now accumulates
  dibits across calls and assembles frames that straddle IQ-chunk
  boundaries.

## [v0.1.7] — 2026-05-19

Observability + import-pipeline release. Twelve merged PRs land the
first batch of per-system Prometheus metrics (issue #269), unblock
RadioReference imports for the post-layout-change PDF format plus
non-US (Australian MMR) systems and native RR CSV downloads (issue
\#271, #278, #279), and close two RTL-SDR silent-failure modes that
prevented P25 control-channel lock on plug-in: a missing
`SetSampleRate` on pool open (issue #275, PR #281) and a Windows
cold-boot warmup timeout that wasn't on the bring-up retry envelope
(PR #274). P25 phase-1 affiliation and unit-registration events now
flow through the SSE/WS telemetry stream (slice of issue #268, PR
\#285). New `gophertrunk_sdr_iq_power_dbfs` gauge + throttled
low-power log catch the gain-at-zero / antenna-disconnected case
operators previously had to guess at (issue #275 follow-ups, PR
\#282).

### Added

- **Prometheus metrics for per-system call rate, encryption breakdown,
  control-channel health, and SDR device tuning state** (issue #269,
  PR #272). New series:
  `gophertrunk_calls_started_total{system,protocol,encrypted}`,
  `gophertrunk_control_channel_frequency_hz{system}`,
  `gophertrunk_control_channel_transitions_total{system,event}`,
  `gophertrunk_sdr_gain_db{driver,serial,role}`,
  `gophertrunk_sdr_gain_auto{driver,serial,role}`,
  `gophertrunk_sdr_ppm{driver,serial,role}`,
  `gophertrunk_sdr_bias_tee{driver,serial,role}`. SDR tuning gauges
  come from a scrape-time snapshot collector so they always reflect
  live pool state.
- **`gophertrunk_sdr_iq_power_dbfs{system}` gauge** updated roughly
  once per second from the cc decoder with mean |IQ|² converted to
  dBFS (issue #275 follow-ups, PR #282). Idle is ~-45 dBFS, healthy
  signal ~-25 dBFS, > -3 means the ADC is clipping. The series is
  dropped on decoder teardown so stale dBFS doesn't outlive the
  active system. Paired with a throttled low-power debug log on the
  same path: < -55 dBFS prints `ccdecoder: iq power very low — check
  antenna, gain, USB` at most once per 5 s — catches the
  gain-at-zero / antenna-disconnected / USB-stuck cases without
  flooding the log.
- **P25 phase-1 affiliation and unit-registration telemetry events**
  (slice of issue #268, PR #285). The cc decoder previously
  recognised TSBK opcodes 0x28 (Group Affiliation Response) and 0x2C
  (Unit Registration Response) but silently dropped them at the
  `dispatchTSBK` default branch. Both opcodes now decode through new
  parsers in `internal/trunking`, publish via two new event kinds
  (`KindAffiliation`, `KindUnitRegistration`), and reach the
  `/api/v1/events` SSE/WS stream as JSON-tagged DTOs. Byte layouts
  follow OP25's `trunk_p25.py` reference. Two regression tests pin
  the JSON shape so downstream dashboards can rely on stable field
  names.
- **Native RadioReference CSV import** for `gophertrunk import-pdf`
  (issue #271, PR #273). RadioReference's `/db/sid/<sid>/download`
  CSV is a flat talkgroup list with no metadata — the importer
  auto-detects the format and the new `-name` / `-sysid` flags
  supply the missing fields (filename stem is used when `-name` is
  omitted). Native CSV carries no sites; combine with a `-pdf` (or
  bundle CSV) when you need control-channel frequencies.
- **`-extract-only` flag for `gophertrunk import-pdf`** (PR #273).
  Paired with a single `-pdf`, dumps the positioned-text rows
  extracted from the PDF as JSON to stdout and exits, so parser bug
  reports can ship a ready-to-replay fixture without sharing the
  original PDF.
- **Per-(VID, PID) bias-tee GPIO table** for the pure-Go RTL-SDR
  driver (issue #275 follow-ups, PR #282). The hardcoded `GPIO 0`
  constant in `device.go` moved to a `knownDevice.BiasTeeGPIO`
  field. Every current entry inherits `GPIO 0` (the dominant
  RTL-SDR.com v3+ / NESDR Smart v5 pinout), but the mechanism now
  exists for boards with a different pinout to be added without
  forking the driver.
- **Throttled "no sync hits" debug log on P25 phase-1 and phase-2
  process paths** (PR #281). A 2 s-throttled line fires when the
  sync detector finds zero hits in a chunk — surfaces the
  previously-silent "IQ isn't reaching the decoder" case operators
  couldn't tell apart from a wrong-frequency cc.
- **"The Story of GopherTrunk" page** on the Pages site
  (PR #280) — project origin and design philosophy, linked from
  the README intro and support page.
- **Discord and Reddit community callouts** on the Pages site
  (PR #286).

### Changed

- `gophertrunk_calls_total` now carries `{system,protocol,encrypted,reason}`
  labels (was `{reason}`); `gophertrunk_calls_active` is now a
  GaugeVec keyed by `{system,protocol}` (was a bare gauge).
  Dashboards that previously scraped the unlabeled shape can recover
  with
  `sum without(system,protocol,encrypted) (gophertrunk_calls_total)`.
- **SDR pool now programs the IQ sample rate at device open** (issue
  \#275, PR #281). `Pool.Open` takes the rate as its first argument
  and calls `SetSampleRate` on every device immediately after the
  USB open; `SetSampleRate` failure closes that device and drops it
  from the pool rather than letting a wrong-rate radio poison the
  decoder. The pure-Go RTL-SDR driver also programs 2.048 MS/s in
  `runBringup` as a belt-and-suspenders default for any future
  consumer of the driver.
- `docs/import.md` and `docs/user-guide-windows.md`: RadioReference
  moved the PDF export from the page footer to the top **Download**
  menu (PDF / CSV / DSD options at `/db/sid/<sid>/download`).
  Instructions updated.

### Fixed

- **RTL-SDR P25 control channel never locked on a freshly opened
  device** (issue #275, PR #281). The pool opened devices and
  applied PPM / gain / bias-tee but never called `SetSampleRate`,
  so the chip's resampler stayed at whatever divisor it powered up
  with while every decoder pipeline downstream did its
  matched-filter and symbol-clock math against `cfg.SDR.SampleRate`.
  Symptom on real hardware was a silent failure: symbol timing
  wrong, FSW / 20-dibit outbound sync detector never matched, and
  the only log line that fired was the cc-hunt retune. The pool now
  programs the rate at open time (see Changed above).
- **`gophertrunk sdr list --probe` fatal-erroring on Windows cold
  boot** (PR #274). The WinUSB warmup sysctl-write returned
  `ErrTimeout` (the Windows equivalent of the Linux EPIPE stall),
  but `isBringupResetable` only matched EPIPE / `ErrDeviceGone`, so
  the existing bring-up `USBDEVFS_RESET` + re-claim retry envelope
  skipped this path. `ErrTimeout` is now treated as resetable; the
  retry stays one-shot, so worst-case cost on a genuine
  (non-cold-boot) timeout is one wasted ~200 ms reset before the
  original error resurfaces. `tunerBringupHint` also grew a
  Windows-aware remediation pointing at the Zadig step for the case
  where the retry also times out.
- `gophertrunk import-pdf` no-System-Name error now prints the
  first ~30 extracted rows inline so the failure is self-diagnosing
  (issue #271, PR #273).
- `parseMetaLine` accepts case-insensitive and whitespace-variant
  labels (`SYSTEM NAME:`, `System Name :`, double-spaces). Falls
  back to the page-title banner ("`<System> Menu`") when no
  explicit `System Name:` line is present, so a minor RadioReference
  layout tweak no longer breaks extraction (issue #271, PR #273).
- `extractPDFRows` now auto-detects RadioReference's two PDF font
  encodings (issue #271, PR #277). Older RR PDFs ship raw glyph
  bytes that need a `+27` ASCII shift; newer ones (e.g. MMR.pdf,
  sid 7197) embed a proper font CMap and arrive already-decoded.
  The extractor sniffs the first 50 rows for anchor strings
  (`System Name`, `Sites and Frequencies`, `Talkgroups`, `WACN`,
  `Last Updated`) and applies the shift only when those anchors are
  absent. `decodeShift` also leaves literal `0x20` spaces alone —
  the new library release emits the occasional in-text literal
  space alongside the encoded `0x05` separator-space, and shifting
  it was corrupting output as `;`.
- The PDF parser now handles RadioReference's non-US layout (e.g.
  Australian MMR system) (issue #278, PR #283). New `siteRowDashRE`
  pattern matches dash-joined `RFSS-Site (X-Y) Name freqs` rows;
  `System Frequencies` and `System Talkgroups` are accepted as
  section markers; `Display` is recognised as an alias for the
  `Alpha Tag` column; `a`-suffix secondary-control-channel
  frequencies are now captured; talkgroup hex columns with leading
  zeros (e.g. `065` for dec=101) are validated numerically rather
  than by string match.
- The `gophertrunk import-pdf` TUI is now usable on systems with
  dozens of sites or hundreds of talkgroups (issue #279, PR #284).
  The Sites tab previously rendered every row unconditionally and
  spilled off-screen; both tabs now paginate to fit the terminal
  height (with a 20-row fallback when `tea.WindowSizeMsg` hasn't
  arrived yet), show a `Site N of M  (showing X-Y)` position
  indicator, and accept `pgup`/`pgdn` for page jumps plus
  `home`/`end` / `g`/`G` to jump to the first/last entry. The
  footer hints are updated.

## [v0.1.6] — 2026-05-18

RTL-SDR driver stabilization release. Eleven merged PRs land
librtlsdr-parity fixes for tuner init bursts, I²C bridge timing,
crystal-frequency selection, macOS IOKit enumeration, and a new
wire-level USB debug-trace switch — layered defenses against the
long-running issue #248 burst-EPIPE reproduction (PRs #255, #256,
#258, #259, #260, #261, #262, #263, #265, #266) plus the macOS
enumeration miss (issue #257, PR #261). Issues #248 and #257
remain open pending field validation on the reporter hardware.
No daemon-level behavior changes outside the RTL-SDR driver.

### Added

- **`RTLSDR_DEBUG_USB=1` environment variable for wire-level debug
  traces.** When set, every USB control transfer the RTL-SDR driver
  issues — `ControlIn`, `ControlOut`, `Reset` — is logged to stderr
  with the bmRequestType, wValue/wIndex/wLength, the payload hex
  (capped at 64 bytes per call), and the outcome (ok / err + duration).
  Output is diffable against `LIBUSB_DEBUG=4` traces from osmocom
  librtlsdr's `rtl_test`, so users can pinpoint exactly which
  transfer stalls on hardware that still misbehaves after the
  librtlsdr-parity fixes. Also emits a per-service trace from the
  macOS IOKit enumerator (matched IOKit class, locationID, VID/PID,
  dropped-property reason) when set — intended for diagnosing
  dongles that don't appear in `sdr list` output. Off by default;
  zero allocation when unset. Documented in the install-linux and
  install-macos troubleshooting tables.

### Changed

- **RTL-SDR tuner I²C bridge now toggles per public method instead of
  per register write.** Every tuner driver (R82xx, E4000, FC0012,
  FC0013, FC2580) previously turned the RTL2832U I²C repeater on
  before each `writeReg`/`readReg` and back off after it — three USB
  control transfers per single-byte chip register access. The
  repeater is now opened once at the top of each public method
  (`Init`, `Standby`, `SetFreq`, `SetBandwidth`, `SetGain`,
  `SetGainMode`) and closed at the end, matching librtlsdr's
  `rtlsdr_set_tuner_*` wrap pattern. For an R820T2 `SetFreq` call
  (~10–15 register writes) this drops 40–60 USB control transfers per
  retune to the steady-state two — measurably faster on USB 2.0 hubs
  and meaningfully less timing-fragile on marginal cabling. Compatible
  with the issue #248 fix: `R82xx.Init`'s leading
  `SetI2CRepeater(true)` is the fresh wire write the chip needs to
  arm the bridge before its multi-byte burst, and the cache state
  ends up `false` post-Detect (off-toggle defer) so the on-toggle
  is real rather than a cache no-op.
- **RTL-SDR tuner detection now follows librtlsdr's exact rtlsdr_open
  probe order and GPIO bring-up dance.** The Go port previously
  probed R820T → R828D → E4000 → FC0013 → FC0012 → FC2580 with no
  GPIO pulses, which silently broke detection of non-R820T tuners
  (FC2580/FC0013/E4000/FC0012) on dongles whose chip-enable lines
  hold the IC in reset until pulsed. The orchestrator now mirrors
  `librtlsdr.c` exactly: R820T → R828D → GPIO5 high→low reset →
  FC2580 → GPIO4 output enable → FC0013 → E4000 → FC0012 (followed by
  a GPIO6 reset pulse if FC0012 was found). FC0012's `Init` also no
  longer emits the two spurious `0x0C` register writes ("soft-reset")
  the pre-fix code shipped — librtlsdr never wrote those; the chip
  reset is the GPIO5 pulse.

### Fixed

- **RTL-SDR R828D-family tuners (RTL-SDR Blog V4 and similar) now
  use the correct 16 MHz reference crystal.** `NewR82xx`
  previously initialized every R820T/R820T2/R828D instance with
  `r.xtalHz = 28_800_000`, the R820T value. R828D variants run
  from a 16 MHz crystal per librtlsdr's `R828D_XTAL_FREQ`. The
  divergence didn't surface during init (the burst uses fixed
  register values), but every `SetFreq` call on an R828D would
  compute PLL parameters against the wrong reference — every
  tuned frequency landed at ~28.8/16 = 1.8× the requested LO,
  rendering V4 dongles unusable for tuning once they did open.
  `NewR82xx` now picks the per-chip default; `SetXtal` keeps
  working as the explicit override for boards with non-standard
  crystals. Closes [issue #264](https://github.com/MattCheramie/GopherTrunk/issues/264)'s
  tuning-after-init half; the init-burst EPIPE half is covered
  by the existing layered defense from issues #248 / PRs
  #258 / #260 / #262 / #263 / #265, which apply to R828D writes
  identically.

- **RTL-SDR R820T burst-init now adds a chip-settle window and
  chunk-size fallback for the EPIPE-on-first-burst case.** Sixth
  iteration on issue #248 after PR #263's per-chunk EPIPE retry +
  open-path USBDEVFS_RESET envelope still failed to close it on two
  NESDR SMArt v5 units. The post-#263 trace confirms the USB reset
  doesn't change the chip's response to the 17-byte burst,
  `Demod.InitBaseband` matches librtlsdr's `rtlsdr_init_baseband`
  byte-for-byte across all 20 register writes + the 20-byte FIR
  upload, the load-bearing `SetI2CRepeater(true)` toggle from PR #262
  is on the wire immediately before each burst attempt, and EP0
  stays healthy post-EPIPE (subsequent control transfers succeed
  without `USBDEVFS_CLEAR_HALT`). Two new defenses ship in this
  round, layered before the existing inner+outer retry from PR #263:
  - `R82xx.Init` now sleeps 5 ms between opening the I²C repeater
    and emitting the burst, covering a chip-settle window librtlsdr
    gets incidentally via function-call latency that our tight
    PrepareDemod → Init back-to-back path doesn't.
  - `writeBurstRaw` now halves the chunk size on
    EPIPE-after-inner-retry-exhausted (16 → 8 → 4 floor) and re-runs
    the whole burst at the smaller size before giving up. Probes the
    chip's effective I²C-bridge FIFO depth empirically — librtlsdr's
    `NMAX_WRITES = 16` may exceed what specific firmware revisions
    accept. The final-failure error wraps as
    `tried chunk sizes 16,8,4; all EPIPE'd: ...` so reporters see
    attribution. Idempotent-write contract called out at the
    function comment — register writes through this path must stay
    safe to replay across the halving walk.
  If this still reproduces, kernel-level usbmon packet traces become
  the prerequisite — `LIBUSB_DEBUG=4` doesn't dump payloads and the
  diagnostic data inferrable from existing traces is exhausted.
  Continues [issue #248](https://github.com/MattCheramie/GopherTrunk/issues/248).

- **RTL-SDR R820T burst-init EPIPE now recovers via a single in-place
  retry + one-shot open-path reset hammer.** Two NESDR SMArt v5 units
  reproduced an EPIPE on the very first `r82xx_init_array` I²C-bridge
  OUT even after PR #262's load-bearing `SetI2CRepeater(true)` wire
  toggle was confirmed firing on the wire (per the post-#262 paired
  `RTLSDR_DEBUG_USB=1` / `LIBUSB_DEBUG=4` capture). The wire bytes
  are byte-identical to librtlsdr's `r82xx_write` first chunk, EP0 is
  not halted (subsequent control transfers succeed without
  `USBDEVFS_CLEAR_HALT`), and `rtl_test` never calls
  `libusb_reset_device` — the EPIPE is a request-specific NACK inside
  the chip, not a USB endpoint state issue.
  `R82xx.writeBurstChunk` now retries a failing chunk once after an
  8 ms settle (no extra repeater toggles — PR #262's contract intact;
  retry attribution is wrapped into the error as
  `after 1 retry on EPIPE: ...` so traces show whether it fired).
  `openDevice` now wraps the entire bring-up sequence (USB warmup →
  baseband init → tuner detect → demod prep → tuner.Init → IF freq)
  in a 1-shot reset+retry envelope on EPIPE / `ErrDeviceGone` —
  subsumes the previous warmup-only retry from PR #255 and extends
  it past the warmup phase. Non-EPIPE errors return immediately
  (reset is the wrong hammer for them). At most one USBDEVFS_RESET
  per `Open` call. `docs/install-linux.md` gains a usbmon
  packet-capture recipe for the next round of diagnostics if this
  doesn't close it — `LIBUSB_DEBUG=4` doesn't dump control-transfer
  payloads, usbmon does. Continues
  [issue #248](https://github.com/MattCheramie/GopherTrunk/issues/248).
- **RTL-SDR `tuners.Detect` again toggles the I²C repeater off on
  return.** An earlier change in this cycle had Detect leave the
  repeater ON across the tuner bring-up window under the theory
  that the wire toggle was a wasteful divergence from librtlsdr.
  Empirically on NESDR v5 silicon the toggle is load-bearing —
  even though the demod register already holds the on-value, the
  chip needs the fresh write to arm the I²C bridge for the next
  multi-byte burst. `R82xx.writeBurstRaw`'s leading
  `SetI2CRepeater(true)` is now a real wire write again (cache=false
  on entry post-Detect), matching librtlsdr's `rtlsdr_open` flow.
  The `PrepareDemod` sequence shipped earlier this cycle is
  unchanged — it remains independently correct librtlsdr-parity
  work that runs after Detect's off-toggle and before the tuner
  burst. Re-closes
  [issue #248](https://github.com/MattCheramie/GopherTrunk/issues/248)
  after the user retest showed the EPIPE persisting.

- **RTL-SDR enumeration on macOS now matches both legacy
  `IOUSBDevice` and modern `IOUSBHostDevice` IOKit classes.** The
  macOS USB enumerator previously matched only `IOUSBDevice`, which
  yields zero services on some Apple Silicon + macOS combinations
  where Apple's IOUSBFamily compatibility bridge is a no-op.
  `gophertrunk sdr list` returned an empty slice with no error and
  no diagnostic — dongles that worked fine in SDRTrunk, GQRX, and
  Homebrew `lsusb` were invisible to GopherTrunk. Both IOKit
  classes are now matched and their results unioned (deduplicated
  by IOKit `locationID`) in both `List` and `Open`. Closes
  [issue #257](https://github.com/MattCheramie/GopherTrunk/issues/257).

- **RTL-SDR open path now matches librtlsdr's R820T/R828D demod-prep
  sequence between `detect_tuner` and `tuner->init`.** The previous
  flow ran `tuners.Detect` (which toggled the I²C repeater off on
  return), then `tuner.Init`, then a generic `SetIFFreq` — skipping
  four demod-register writes librtlsdr emits before tuner init:
  disable Zero-IF mode (page 1, addr 0xB1, val 0x1A), enable
  In-phase ADC input only (page 0, addr 0x08, val 0x4D),
  `set_if_freq(3.57 MHz)`, and enable spectrum inversion (page 1,
  addr 0x15, val 0x01). Without those four writes the R820T-family
  chip is brought up against a Zero-IF / IQ datapath / inversion
  configuration that diverges from what librtlsdr ships, which has
  been the residual divergence after the chunking fix shipped in
  this cycle. New `R82xx.PrepareDemod` runs the sequence; `openDevice`
  invokes it on the R820T-family branch.
- **RTL-SDR `tuners.Detect` now leaves the I²C repeater on across the
  tuner bring-up window.** Previously Detect deferred
  `SetI2CRepeater(false)` and tuner.Init then re-enabled the repeater
  per burst, producing an off→on toggle between Detect and the very
  first I²C OUT — the wire byte right before the multi-byte burst
  that some NESDR v5 dongles stall on. Detect now leaves the
  repeater on on success (or toggles it off on the no-tuner
  error path); the new `openDevice` step list owns the post-Init
  off toggle.
- **RTL-SDR R820T/R820T2 manual gain now uses librtlsdr's balanced
  LNA+Mixer split.** `R82xx.SetGain` previously walked the LNA gain
  ladder to maximum-not-exceeding-target, then walked the mixer
  ladder — landing on the same numeric gain as librtlsdr but with all
  the gain concentrated on the LNA. The result was a worse noise
  figure and worse front-end linearity at every ladder entry. The
  walk now alternates LNA and mixer with pre-increment, matching
  `r82xx_set_gain` in osmocom librtlsdr. Affects every R820T/R820T2
  dongle (the common case) the moment the user picks a manual gain.
- **RTL-SDR E4000 (Elonics) tuner frequency setting now writes the
  correct synthesizer registers.** `E4000.SetFreq` was writing the
  fractional `X` value to `SYNTH5`/`SYNTH6` (off-by-one register) and
  never writing the band-select / R-divider byte to `SYNTH7` at all,
  so the chip would mistune at every frequency. The PLL math itself
  was correct; only the wire-level register addresses were wrong.
  Now matches librtlsdr's `e4k_tune_params` exactly. Affects E4000
  dongles (legacy hardware — NOXON DAB sticks and similar).
- **RTL-SDR R820T/R820T2 init burst now chunks at 16 bytes to match
  librtlsdr.** The 27-byte register flood at the top of `R82xx.Init`
  previously went on the wire as a single 28-byte I²C-bridge OUT
  (1 register pointer + 27 data bytes). Some NESDR v5 dongles stall
  the very first multi-byte OUT when its data payload exceeds 16
  bytes — librtlsdr's `r82xx_write` has chunked at `NMAX_WRITES = 16`
  for exactly this reason. `writeBurstRaw` now splits the data into
  ≤16-byte segments under one repeater on/off pair, advancing the
  register pointer per chunk (the chip auto-increments). The wire
  bytes are otherwise unchanged.
  Follow-up to the warmup probe shipped earlier in this cycle;
  addresses the residual reproduction in
  [issue #248](https://github.com/MattCheramie/GopherTrunk/issues/248).
- **RTL-SDR tuner init no longer fails on dongles left in a
  half-initialised USB state.** Open now performs librtlsdr's
  dummy-write probe (`USB_SYSCTL = 0x09`) immediately after claiming
  the interface and, on `EPIPE` / `ErrDeviceGone`, runs a one-shot
  `USBDEVFS_RESET` + re-claim before retrying. Dongles whose endpoint
  was left stalled by a crashed prior session or a freshly-unbound
  DVB kernel driver now open transparently instead of surfacing the
  EPIPE as "r82xx init: burst write: I2CWrite addr=0x34: broken pipe".
  When both attempts fail the existing tuner-bringup hint is still
  appended.
  Addresses [issue #248](https://github.com/MattCheramie/GopherTrunk/issues/248).

## [v0.1.5] — 2026-05-16

### Added

- **Remediation hint on tuner-init I²C failures.** The RTL-SDR
  driver now appends a one-line hint pointing at the three known
  root causes (DVB kernel driver still bound, marginal USB power,
  flaky cable / USB 3.0 hub) when the tuner doesn't ack on the I²C
  bus during bring-up — both the EPIPE-on-first-burst case and the
  mid-init `ErrDeviceGone` case. `docs/install-linux.md`'s
  troubleshooting table grows a matching row keyed on the literal
  error string so operators searching for "broken pipe" land
  somewhere actionable.
  Shipped in [PR #251](https://github.com/MattCheramie/GopherTrunk/pull/251),
  addressing [issue #248](https://github.com/MattCheramie/GopherTrunk/issues/248).
- **Bundled Zadig WinUSB driver installer in the Windows installer.**
  The Windows `setup.exe` now ships `zadig.exe` alongside
  `gophertrunk.exe`, so first-run operators no longer have to chase a
  separate download to bind the RTL-SDR's WinUSB driver. Setup adds a
  Start Menu shortcut **"Install RTL-SDR driver (Zadig)"** and offers
  an unchecked **"Run Zadig now"** option on the final wizard page;
  Zadig's own manifest handles the UAC elevation. The uninstaller
  also now strips the `{app}` entry from the system PATH (previously
  leaked across uninstalls) and asks whether to wipe the editable
  `config.yaml` + the Setup-created `gophertrunk-web` subfolder —
  default **No**, so user data is preserved unless explicitly opted
  in. Bundled binary is `zadig-2.9.exe` from libwdi `v1.5.1`
  (GPL-3.0); see [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md)
  for attribution.
  Shipped in [PR #249](https://github.com/MattCheramie/GopherTrunk/pull/249).
- **NXDN deviation surfaces on the TUI Settings → FEC tab.**
  The `nxdn_deviation_hz` knob shipped in [PR #243](https://github.com/MattCheramie/GopherTrunk/pull/243)
  but wasn't visible from the operator console. The
  per-system FEC summary now appends `deviation: 1800 Hz`
  (or whatever override is configured) alongside the existing
  `viterbi:` mode, matching the pattern P25 Phase 2 / MPT 1327
  use for their per-protocol opt-outs. The hash gate that
  controls FEC table refresh covers the new field so a
  config-reloaded override surfaces inside one SSE round-trip.
- **NXDN real-air integration harness skeleton.**
  [`cmd/gophertrunk/integration_cc_nxdn_realair_test.go`](cmd/gophertrunk/integration_cc_nxdn_realair_test.go)
  is the skip-gated companion to the existing synthesized
  `TestDaemonCCDecodesNXDN`. When a contributor drops a single
  `*.cfile` + sibling `*.metadata.json` pair into
  [`samples/nxdn/`](samples/nxdn/), the harness:
   - registers the in-tree `sdr.MockFloat32Driver` against the
     capture,
   - tunes the daemon to `metadata.center_freq_hz` at
     `metadata.sample_rate_hz` (both required at the top level
     since GNU Radio cfiles don't embed them),
   - boots the daemon with `nxdn_viterbi_mode: spec`,
   - waits up to 3 s wall time for `events.KindCCLocked`,
   - asserts `LockState.SystemID` / `SiteID` / `FrequencyHz`
     match the documented `metadata.expected` values
     byte-for-byte.
  
  CI stays green via a documented `t.Skipf` fall-through until
  a capture lands. Multiple `*.cfile` candidates surface as an
  explicit test error so the contributor knows to disambiguate.
  Metadata schema documented in
  [`samples/nxdn/README.md`](samples/nxdn/README.md).

- **Per-system NXDN deviation tunability** (`nxdn_deviation_hz`).
  The NXDN receiver's 4-FSK slicer was hardcoded to the Common Air
  Interface spec value of 1800 Hz peak deviation, which produces a
  bimodal dibit distribution on captures from transmitters that
  deviate from spec (e.g. `samples/nxdn/NXDN96 IQ.wav` reports
  3 / 50 / 3 / 44 % through the production pipeline). Operators can
  now set `nxdn_deviation_hz: 2400` (or any positive value) on a
  per-system basis to recalibrate the slicer against the captured
  signal's actual deviation. Zero / unset keeps the spec default.
  See [`samples/nxdn/README.md`](samples/nxdn/README.md#tuning-deviation-for-non-spec-captures)
  for the sweep recipe.
- **AMBE+2 knox preset bundles** (`ambe2.RegisterPreset` /
  `ambe2.ListPresets`). The existing `SetKnoxTone` hook (b₁ ∈
  [144, 163]) registers one vendor-specific dual-tone pair at a
  time; the new preset API takes a named bundle of entries and
  records the preset name for operator diagnostics. Lets per-vendor
  sub-packages ship curated tables via a single `RegisterPreset`
  call instead of repeated `SetKnoxTone`s. The in-tree code ships
  no vendor presets because the public AMBE+2 spec does not
  document the [144, 163] frequency range — see
  [`docs/vocoders.md`](docs/vocoders.md#sourcing-vendor-frequencies)
  for the sourcing checklist.

### Internal

- **Polish pass: config example completeness, YSF acceptance criteria,
  tuner math coverage.**
  - `config.example.yaml` now shows commented examples for every
    per-system FEC opt-out documented in the README's
    [§FEC opt-outs](https://github.com/MattCheramie/GopherTrunk#fec-opt-outs)
    table. NXDN (`nxdn_viterbi_mode`, `nxdn_deviation_hz`), P25
    Phase 2 (`p25_phase2_{trellis,rs,scrambler,clock}_mode`),
    TETRA (`tetra_colour_code`, `tetra_channel`,
    `tetra_channel_coding`, `tetra_clock_mode`), EDACS
    (`edacs_bch_mode`), MPT 1327 (`mpt1327_bch_mode`,
    `mpt1327_cwsc_tolerance`), and D-STAR (`dstar_fec_mode`)
    previously had docs but no example block to copy from.
  - `samples/ysf/README.md` grows the explicit
    `## Acceptance criteria` section the other four sample
    READMEs (`nxdn`, `dmr-tier2`, `mpt1327`, `tetra`) already
    have. Three numbered criteria — CRC pass-through against the
    metadata's `fich_sequence`, MMDVMHost-vs-DSDcc schedule
    locked, and trellis correction-depth bounded ≤ 4 errors per
    100-bit on-air block at SNR ≥ 12 dB.
  - `internal/sdr/rtlsdr/tuners` coverage rises from 30.3% to
    43.5% via ten new tests covering: E4000 PLL Σ-Δ synth math
    (hand-computed Z / X for 50 MHz / 100 MHz / 433 MHz / 868 MHz
    / 1.5 GHz against the band-table walk in `e4k.go:84-97`),
    `ErrUnsupportedFreq` exact-boundary inclusivity for E4000 /
    FC0012 / FC0013 / FC2580 (the production `< minHz || > maxHz`
    guard accepts the endpoints), `nearestGainIndex` rounding
    behaviour on E4000's 17-step LNA ladder + the shared helper's
    clamp / tie-break invariants, and `fc0012NearestGainIndex`
    rounding parity. No production-code changes — pure post-hoc
    coverage of math paths that don't need RTL-SDR hardware.

- **DVSI mock-transport error-path coverage.** The
  `internal/voice/dvsi` test suite previously exercised the happy
  paths (scripted exchange, loopback silence, ErrNoDevice fall-
  through) but left the error-wrapping branches uncovered.
  Fifteen new tests now lock in: `Open(DefaultOptions())` returns
  `ErrNoDevice` carrying VID/PID/serial diagnostics, zero-valued
  VID/PID falls back to the documented FT2232H defaults, explicit
  `Transport` beats `LoopbackOnly` in `Open`'s switch, `Decode`
  wraps `transport.Write` / `transport.Read` errors with their
  origin labels, the loopback `Transport` rejects `Read` before
  `Write` + `Write`/`Read` after `Close` + malformed packets,
  and `PktControl` / unknown-type packets get cleanly Ack-mirrored
  so a future fuzz target won't stall on them. Hardware
  integration unchanged — `openUSBTransport` still returns
  `ErrNoDevice` until a chip is available for round-trip
  testing.

- **Calibrate harness math is testable without external fixtures.**
  Extracted `calibrate.CompareSamples([]int16, []int16) Result` so
  the RMS-ratio + cross-correlation math can be exercised on
  synthetic streams. The two existing skip-gated tests
  (`TestCompareIMBE*`, `TestCompareAMBE2*`) keep waiting for
  captured DSD-FME / OP25 reference WAVs; the new
  `TestCompareSamplesSyntheticGainOffset` validates the math
  unconditionally (a +3 dB louder reference must produce
  `RMSRatioDb = −3.0 ± 0.5` and `PeakXcorr ≥ 0.99`). Regressions
  in the loudness / similarity math now fail CI without needing
  any external reference data to land first.

- **Cleanup & coverage round.**
  - `web/scripts/seal-node-modules.mjs` is registered as the npm
    `postinstall` hook. It drops a sentinel `web/node_modules/go.mod`
    so Go's recursive package discovery (`go list ./...`,
    `go test ./...`) skips the stray Go packages npm dependencies
    occasionally ship inside their tarballs (e.g.
    `flatted/golang/pkg/flatted`). No more spurious entries in Go
    package listings on developer machines that have run
    `npm install`.
  - `cmd/gophertrunk/launcher.go` grows three injectable seams
    (`hasWebAssetsFn`, `canOpenBrowserFn`, `openBrowserFn`) so
    `openWebUI` can be exercised end-to-end without spawning a real
    browser. New tests verify the embedded-SPA branch wins when
    `gtweb.HasAssets()` returns true, the headless-fallback prints
    instead of launching, the no-embed sibling-discovery path runs
    cleanly, and the missing-HTTP-addr error fires.
  - `watchReloadSignal` now installs `signal.Notify` synchronously
    before spawning its goroutine — fixes a latent race where
    SIGHUP delivered immediately after the call could kill the
    process (default SIGHUP action) before the goroutine got
    around to registering its handler. Visible only in tightly-
    timed tests; harmless in production where SIGHUP arrives long
    after startup.
  - New `TestSIGHUP_TriggersReload` and
    `TestSIGHUP_BadConfigDoesNotCrash` send real SIGHUP signals to
    the test process and assert the watcher's reload path runs and
    that malformed-YAML reloads leave the in-memory config intact.

- **Test infrastructure: web SPA + in-process TUI.**
  - SPA gains Vitest + React Testing Library. `Import.test.tsx`
    covers the no-config / no-mutations banners + the
    Stage→Preview→Result happy path + commit / discard / error
    flows; `Settings.test.tsx` covers the inline-edit state
    machine, client-side validation, server PATCH errors, and
    restart-required badges. Run with `npm test`.
  - The in-process TUI launcher path (`runInProcessTUI`) is split
    into a testable `prepareInProcessTUI` (URL resolve, log
    redirect, model construction) and a thin `prog.Run()` wrapper.
    New tests cover missing-HTTP-addr error, log-redirect
    correctness, cleanup restoring the original writer, the
    constructed client actually reaching the daemon, plus a
    teatest-driven smoke test of the bubbletea Update loop against
    a stub HTTP daemon.
  - `internal/api.Server` now exposes `BoundAddr()`, and
    `Daemon.HTTPListenAddr()` prefers the actually-bound address
    when the listener has resolved an ephemeral `:0` port. Fixes
    a long-standing bug in the `HTTPListenAddr` docstring claim
    "helpful for tests using an ephemeral `:0` port" — it really
    is now.

### Added

- **Interactive daemon launcher.** `gophertrunk` (no args) now prompts
  the operator on a TTY for what to drive: `[1]` in-process TUI, `[2]`
  bundled web SPA in the system browser, or `[3]` stay headless.
  Non-TTY stdin (systemd, Windows service, Docker) auto-selects
  headless so service managers see no behaviour change. New flags
  preselect: `-tui`, `-web`, `-headless`; the three are mutually
  exclusive. See [`docs/launcher.md`](docs/launcher.md).
- **Live settings editing.** New `PATCH /api/v1/settings` endpoint
  accepts a sparse patch (every field optional), writes the result to
  `config.yaml` preserving comments + formatting, and hot-reloads the
  fields the daemon knows how to change in-process (audio volume /
  mute / recording, scanner scan mode, log level). Other fields
  ("restart required") are written to disk and flagged in the
  response so the SPA / TUI can render badges. An mtime guard refuses
  to clobber a config.yaml that was edited externally while the
  daemon was running.
- **Live import.** New `POST /api/v1/import` (multipart),
  `POST /api/v1/import/{id}/commit`, `DELETE /api/v1/import/{id}`
  endpoints let operators upload RadioReference PDFs / multi-section
  CSVs to a running daemon, preview the parsed systems, and commit
  into `config.yaml` without restarting. The TUI grows an Import
  panel (Stage → Preview → Result); the web SPA grows a matching
  `/import` route with a native file picker.
- **Startup hardening.** A new pre-flight step auto-creates the
  recordings / storage / cc-cache parent dirs and verifies TLS
  cert/key parse cleanly before the daemon binds. SDR-pool open
  failures and missing talkgroup CSVs collect into `startup_warnings`
  (surfaced on the runtime DTO + the launcher menu) instead of
  vanishing into the log. HTTP and gRPC bind failures now abort the
  daemon cleanly instead of being demoted to warnings — the launcher
  never lands against a half-dead daemon.
- **Embedded web SPA.** The daemon binary now embeds the built SPA
  (when `make web-build` was run before `go build`) and serves it
  at `/` on the HTTP API. `gophertrunk -web` opens the daemon URL
  directly; client-side routes (`/scanner`, `/settings`, `/import`,
  …) fall back to `index.html` so React-Router takes over. Fresh
  checkouts without a `web/dist/` build keep the existing sibling-
  directory discovery path. See [`docs/web.md`](docs/web.md).
- **Inline-editable Settings.** Every editable runtime knob the
  daemon hot-reloads (audio volume / mute, log level, scanner scan
  mode, …) plus the restart-required ones are now editable from
  both the TUI Settings panel (cursor + Enter to edit, Enter to
  save, Esc to cancel) and the web SPA's `/settings` route. Rows
  show a `[restart]` badge when the daemon can't hot-apply.
- **SIGHUP config reload.** Sending `SIGHUP` to a running daemon
  reloads `config.yaml`, diff-applies hot-reloadable fields, and
  logs a list of restart-required changes. The signal handler is a
  no-op on Windows.
- **Single-instance lock.** The daemon now flocks
  `<configdir>/.gophertrunk.lock` at startup so two instances aimed
  at the same `config.yaml` can't both try to claim the same
  RTL-SDR devices. The contender exits with a clear "another
  gophertrunk is running (pid=…, started=…)" message instead of an
  opaque libusb error.
- **Friendlier YAML errors.** `config: <path>: parse error …` now
  carries the resolved config path and a hint to run the wizard or
  recheck indentation.
- **Patent-posture notice plumbed through `startup_warnings`.**
  The AMBE+2 advisory no longer scrolls past on the daemon log
  immediately before the launcher prompt; it lands in the warnings
  channel and surfaces on the launcher menu / TUI dashboard / runtime
  DTO. `GOPHERTRUNK_QUIET_BANNER=1` still suppresses it for CI.

### Changed

- **Security defaults flipped for closed-LAN deployments.** Empty
  `api.auth.mode` now defaults to `disabled` (was `auto`) and empty
  `api.cors.allowed_origins` now permits any origin (was strict). The
  daemon still warns loudly at startup when these defaults take
  effect on a non-loopback bind, but the common single-host setup no
  longer needs explicit auth + CORS config to talk to the web SPA
  from `file://`. Operators on hostile networks opt back in via
  explicit `api.auth.mode: required` + `api.cors.allowed_origins:
  ["http://laptop.local:5173"]`. The default `api.http_addr` is now
  `127.0.0.1:8080` (was empty) so the bundled launcher's TUI / web
  paths work out of the box.

- **Config auto-discovery.** `gophertrunk run` (no `-config` flag)
  now walks `$GOPHERTRUNK_CONFIG` → `<UserConfigDir>/GopherTrunk/config.yaml`
  → `<Home>/Documents/GopherTrunk/config.yaml` → `./config.yaml`
  and loads the first match, printing `config: loaded <path>` on
  startup. When the chosen directory holds 2+ `*.yaml`/`*.yml`
  files, an interactive numbered picker prompts the operator on
  stdin (non-TTY launches like Windows services / systemd / CI
  auto-select the first match with a stderr warning instead of
  hanging). `internal/config.Discover()` + `DiscoverWith(opts)` for
  programmatic callers.
- **Windows installer "editable-files folder" page.** The Inno
  Setup wizard now asks where the operator's `config.yaml` should
  live (default `Documents\GopherTrunk`), seeds a starter file
  there (preserved across re-install + uninstall), pins
  `HKCU\Environment\GOPHERTRUNK_CONFIG` so the daemon finds it
  without `-config`, and adds a Start Menu shortcut "Edit my
  config.yaml (Notepad)". See [`install-windows.md`](docs/install-windows.md).
- **`gophertrunk sdr list --probe`** opens each enumerated device
  long enough to run the demod + tuner bring-up, populating the
  TUNER + gains columns. Without the flag those columns stay
  blank (Enumerate only reads USB descriptors, so the command is
  fast and never collides with a running daemon).
- **Config-builder wizard quality-of-life.** `←` / `→` toggles
  boolean fields (the footer hint already promised this). The
  path field expands `%VAR%` (Windows), `$VAR` / `${VAR}` (POSIX),
  and leading `~` at write time; the review screen shows
  "resolves to: \<abs\>" when expansion changes the path. The
  default write target now consults `$GOPHERTRUNK_CONFIG` and
  falls back to `<UserConfigDir>/GopherTrunk/config.yaml` when
  the current directory isn't writable (fixes "Access is denied"
  when the binary is launched from `C:\Program Files\GopherTrunk\`).
  `MkdirAll` errors on commit are surfaced instead of swallowed.
- `gophertrunk import-pdf` subcommand parses trunking-system data
  from RadioReference.com PDF exports **and** from structured
  multi-section CSV bundles, merging both into the operator's
  `config.yaml` plus per-system Trunk-Recorder-style talkgroup CSVs.
  Launches a Bubbletea TUI by default for reviewing/pruning sites and
  toggling per-talkgroup Scan/Lockout/Priority before write;
  `-no-tui`/`-dry-run`/`-force` flags cover scripting and CI bring-up.
  PDF and CSV sources are mixable in a single invocation (`-pdf` and
  `-csv` are both repeatable). Atomic writes (in-memory schema
  validation + temp file + rename) so a malformed source never
  corrupts the existing config. Supports P25 Phase 1 + Phase 2 PDFs;
  CSV bundles cover P25/DMR/NXDN. See
  [`docs/import.md`](docs/import.md) for the full operator reference
  and CSV format spec.
- Capture-spec **acceptance criteria** for every real-air-blocked
  follow-up at [`samples/<proto>/README.md`](samples/): TETRA
  wants 5 s lock latency + ≥ 90% frame recovery + a new
  `gophertrunk_tetra_viterbi_corrections` Prometheus histogram
  (gated by `metrics.detailed_fec: true`, not yet wired); NXDN
  wants ≥ 80% CRC-verified CAC bursts + SystemID match + 3 s
  lock; DMR Tier II wants byte-for-byte FLC match + clean
  Terminator-with-LC handling; MPT 1327 wants ≥ 95% true-positive
  lock rate + monotone tolerance sweep. [`samples/README.md`](samples/README.md)'s
  top-level table now shows status (✅ closed vs ⏳ capture
  pending) plus per-protocol "what captures buy" — DMR Tier II
  and MPT 1327 captures are optional secondary validation rather
  than the blocker (closed algorithmically in PR-A / PR-C).
- `internal/version` now exposes `Version`, `Commit`, and
  `BuildTime` (all `-ldflags`-injectable) plus a `String()`
  formatter (`"vX.Y.Z (sha=…, built=…)"`). Makefile and the
  release workflow both populate all three. `gophertrunk version`
  CLI subcommand prints the formatted string; the daemon logs it
  on startup.
- AMBE+2 patent-posture banner: daemon logs a one-line notice at
  startup pointing operators at
  [`docs/vocoders.md`](docs/vocoders.md). Suppressible via
  `GOPHERTRUNK_QUIET_BANNER=1` for CI / test harnesses.
- `make release-dry-run VERSION=v0.99.0` rehearses the release
  build locally — produces a `dist/dry-run/gophertrunk` with the
  supplied version metadata injected and a `SHA256SUMS` file.
  See [`CONTRIBUTING.md` §"Cutting a release"](CONTRIBUTING.md#cutting-a-release).
- Toolchain pinned to Go 1.25.10 (closes 23 stdlib CVEs in the
  default 1.25.0 toolchain auto-downloaded by `go 1.25.0` in
  go.mod).
- CI hardening: `vulncheck` job runs `govulncheck` against the
  direct + transitive dependency graph; `licenses` job regenerates
  the transitive-deps inventory via `google/go-licenses` and
  diffs against the committed `THIRD_PARTY_LICENSES.csv`;
  `integration` job runs `make test-integration` across the whole
  module to backstop the existing `cmd/gophertrunk/`-only target.
- `Makefile` targets: `make vulncheck`, `make licenses`,
  `make test-integration`.
- [`THIRD_PARTY_LICENSES.md`](THIRD_PARTY_LICENSES.md) — hand-
  curated direct-deps license table sourced from `go.mod` plus the
  ISC attribution for the mbelib-derived AMBE+2 / IMBE codebook
  tables.
- `SECURITY.md`, `CONTRIBUTING.md`, and a systemd unit template
  ([`docs/gophertrunk.service`](docs/gophertrunk.service)) for
  operators standing the daemon up on Linux servers.
- Optional TLS on both the HTTP API and the gRPC server via
  `api.tls_cert` / `api.tls_key` in `config.yaml`. Plain TCP
  stays the default for loopback / trusted-LAN deployments. See
  [`docs/hardening.md` §"Transport encryption (TLS)"](docs/hardening.md#transport-encryption-tls).
- Extended `GET /api/v1/health` diagnostics:
  `pool_attached_count`, `active_calls`, `db_connected`,
  `metrics_enabled`, `auth_mode`, `version` alongside the legacy
  `status` + `now`. Supports k8s / Nomad readiness probes that
  distinguish "process up" from "actually working".
- HTTP server now sets `ReadTimeout` (30 s), `WriteTimeout`
  (30 s), and `IdleTimeout` (120 s) on top of the existing
  `ReadHeaderTimeout`. Streaming endpoints (SSE, audio stream)
  opt out per-request via
  `http.ResponseController.SetWriteDeadline(time.Time{})`.
- gRPC server now configures `keepalive.ServerParameters`
  (30 s idle ping, 10 s ack timeout) +
  `KeepaliveEnforcementPolicy` (5 s min-time floor,
  `PermitWithoutStream: true`) so long-lived `StreamAudio`
  subscribers detect dead peers cleanly.
- Graceful shutdown drain window for the HTTP server bumped from
  5 s to 30 s so in-flight SSE / WebSocket / audio subscribers
  drain instead of being torn down mid-frame.
- AMBE+2 knox / call-alert dual-tone vendor-override hook:
  [`ambe2.SetKnoxTone`](internal/voice/ambe2/knox.go). Operators
  with a per-vendor reference register
  `(freqA, freqB)` pairs for `b1 ∈ [144, 163]` and the matching
  tone frames synthesise through the same DTMF dual-tone path
  (phase-continuous + AGC-scaled).
- Voice calibration plumbing:
  [`cmd/voice-calibrate`](cmd/voice-calibrate/) CLI wrapping
  `calibrate.Compare`, per-vocoder testdata READMEs, and an
  end-to-end recipe at
  [`docs/voice-calibration.md`](docs/voice-calibration.md).
- DVSI USB-3000 / AMBE-3003 hardware backend scaffolding behind
  `-tags dvsi`. AMBE-3003 wire protocol + `Vocoder` + `Transport`
  interface + `voice.Vocoder` conformance + `init()`
  registration all ship; the USB / FTDI plumbing remains a stub
  returning `ErrNoDevice` (hardware integration follows when a
  chip is available for round-trip testing). Loopback `Transport`
  exercises the wire protocol + Vocoder state machine in CI.
- YSF FICH on-air codec: `EncodeFICHOnAir` / `DecodeFICHOnAir`
  in [`internal/radio/ysf/fich_trellis.go`](internal/radio/ysf/fich_trellis.go)
  per the MMDVMHost / DSDcc / Pi-Star reference (puncture
  positions `{0, 1, 102, 103}` + column-major 10×10 interleave).
  Exhaustive single-bit-flip recovery test confirms every one of
  the 100 on-air positions is Viterbi-corrected.
- DMR Tier II / Tier III symbol-density diagnostic test pair in
  [`cmd/gophertrunk/dmr_tier2_diagnostic_test.go`](cmd/gophertrunk/dmr_tier2_diagnostic_test.go)
  that localises the divergent statistic between the two
  synthesized fixtures.
- MPT 1327 CWSC Hamming-distance tolerance via the new
  `mpt1327_cwsc_tolerance` per-system config key. Default value
  is `2` (matches commercial MPT 1327 receivers on noisy on-air
  captures); operators replaying pre-stripped synthesized
  fixtures opt back into exact-match with `0`.

### Changed

- DMR Tier II pipeline `ClockGain` lowered from 0.025 to 0.015
  in [`internal/scanner/ccdecoder/pipelines.go`](internal/scanner/ccdecoder/pipelines.go)'s
  `newDMRTier2Pipeline`. The diagnostic test above surfaced that
  Tier II's BPTC(196, 96)-encoded payload's class-3 dibit
  overrepresentation (21.4% vs Tier III's 5.1%) and matching
  mean-transition magnitude (1.27 vs 0.90) slipped the
  Mueller-Müller clock loop at 0.025. The more conservative gain
  stays locked under the harder symbol distribution; live
  captures benefit equally. Lifts the
  `TestDaemonCCDecodesDMRTier2` `t.Skip` that's been in place
  since PR #184.

### Fixed

- `TestDaemonCCDecodesDMRTier2` no longer skips — see the
  Tier II ClockGain change above.

### Documentation

- New: [`SECURITY.md`](SECURITY.md), [`CONTRIBUTING.md`](CONTRIBUTING.md),
  [`docs/voice-calibration.md`](docs/voice-calibration.md),
  [`docs/gophertrunk.service`](docs/gophertrunk.service).
- Extended: [`docs/hardening.md`](docs/hardening.md) gains
  "Transport encryption (TLS)", "Health endpoint diagnostics",
  "Connection-drain window", and "Timeouts and keep-alive"
  sections.
- Extended: [`docs/vocoders.md`](docs/vocoders.md) gains
  "Voice calibration plumbing", "Knox / call-alert extension
  hook", and "DVSI backend layout" sections.
- Updated: README's `Status & known gaps` and `Roadmap`
  sections — MPT 1327 CWSC, DMR Tier II fixture, YSF on-air
  codec, and vocoder calibration plumbing all moved from
  "remaining follow-up" to "now shipping" or "real-air capture
  pending".

---

## Historical entries

The project's pre-changelog history is captured in git — every
merged PR has a descriptive title and commit body. Reconstruct a
historical changelog from a tagged release with:

```sh
git log --oneline --no-merges <prev-tag>..<this-tag>
```

The first tagged release will fold this `Unreleased` section into
a versioned heading and start a fresh `Unreleased` for ongoing
work.
