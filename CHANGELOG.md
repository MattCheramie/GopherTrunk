# Changelog

All notable user-visible changes land here, newest first.
Format adapted from [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
for tagged releases.

## [Unreleased]

### Security
- **Bumped the Go toolchain to 1.25.13 and `golang.org/x/net` to v0.55.0** to
  clear the CVEs govulncheck reports against the pinned toolchain: seven Go
  standard-library advisories fixed in go1.25.13 (`net/url` GO-2026-6218,
  `html/template` GO-2026-6091, `crypto/tls` GO-2026-6090, `net/http`
  GO-2026-6089, `encoding/xml` GO-2026-6088, `encoding/asn1` GO-2026-5972) and
  the `x/net/idna` Punycode advisory GO-2026-5026. Toolchain/dependency bump
  only — no code changes.

### Added
- **SoapyRemote: experimental phase-coherent MRC diversity over two RX channels.**
  A new `diversity: mrc` option on a `soapy_remote` source opens RX channels 0
  and 1 and phase-coherently maximal-ratio-combines them into one maximised-SNR
  stream, for shared-LO front-ends (USRP B210 / AD9361 and clones) whose RX0↔RX1
  phase relationship is a constant per tune. It reuses the existing
  `dsp/diversity.StaticCalibrator`: a one-time phase calibration is taken on the
  first signal-bearing window after each tune (no per-sample tracking), and the
  combined stream feeds the normal decode pipeline unchanged. Default (unset) is
  the ordinary single-channel stream — byte-identical to before. **Experimental
  (issue #1062):** the 2-channel wire de-interleave and the calibration trigger
  are validated in unit tests but not yet confirmed against a live dual-RX
  server; the feature is opt-in and cannot affect single-channel users.
- **TETRA DMO (Direct Mode Operation) now decodes in the daemon.** A new
  `protocol: tetra-dmo` (aliases `dmo` / `tetra_dmo`) camps a direct-mode
  frequency, locks on the Direct Mode Synchronisation Burst (DSB), auto-recovers
  the DM colour code, and decodes the Direct Mode Normal Burst (DNB) TCH/S speech
  train to voice on the same carrier — no separate traffic channel. Previously a
  DMO capture had to run through the TMO control-channel pipeline, whose burst
  geometry does not match DMO, so it appeared to "lock" (the DSB SCH/S is
  colour-0-scrambled like a TMO BSCH) but produced no grants and no audio. The
  new pipeline reuses the offline-validated DMO decoders behind a bounded
  streaming burst extractor. Configure with `protocol: tetra-dmo` +
  `control_channels: [<freq>]` (optional `tetra_colour_code` overrides colour
  recovery). The DM call-control protocol (EN 300 396-3 source/destination SSI,
  group) is not yet decoded, so a DMO call records without a talkgroup identity
  (filed under group `0`); and this path is validated offline/synthetically but
  not yet A/B'd against a real on-air DMO capture (see docs/reference/tetra-dmo.md).
- **HackRF Pro is now identified as such, and its narrowband filter is
  configurable.** GopherTrunk reads the firmware's board ID at open, so a HackRF
  Pro (board ID 5, *Praline*) now reports `HackRF Pro` — and a HackRF One R9
  (board ID 4) `HackRF One R9` — in `gophertrunk sdr list`, the Devices panel,
  and the startup log, instead of being lumped in with the original HackRF One.
  Two new per-device options expose the Pro's RF-path features: `narrowband_filter:
  true` engages the Pro's switchable narrowband anti-alias filter (tighter
  adjacent-channel rejection for narrowband voice like P25, at the cost of usable
  bandwidth), and `fpga_dc_block: true` strips the zero-IF DC-offset spike in the
  Pro's FPGA before samples leave the device — a hardware alternative to the P25
  voice path's software DC-block that also cleans the control channel (measured
  on hardware: raw-stream DC magnitude drops to zero). Both are ignored, with a
  startup warning, on any board without the hardware. (The Pro's 16-bit
  extended-precision RX mode is not included: it's unimplemented in the released
  Pro firmware — `fpga_init` only programs the standard bitstream and the SGPIO
  capture is hardwired to the 2-byte format — so it can't be driven from the host
  yet.)
- **`dc_avoid` now also protects voice grants, not just the control channel.**
  On a zero-IF dongle (HackRF, RTL-SDR) a granted voice carrier tuned exactly
  on-channel sits directly on the front-end DC spur / LO self-mixing / I/Q
  image. That corruption leaves a healthy *average* EVM but biases the specific
  symbol decisions the frame-sync word rides on, so the sync correlator misses
  and voice grants decode zero LDUs while the short, heavily-FEC'd control
  channel still limps through — a system that locks its CC but never produces
  audio. `dc_avoid: true` on a `role: voice` device now offset-tunes each
  granted call's LO (by `dc_avoid_offset_hz`, default `sample_rate/4`) and mixes
  the carrier back to baseband before the composer sees it, the same technique
  SDRTrunk/OP25 apply by channelising every carrier off-DC — extending the
  existing control-only offset tuning (issue #402) to the per-grant voice path.
  Measured on a HackRF Pro against a marginal simulcast P25 system: an
  on-channel capture demods at ~21 % EVM with 0 frame-sync hits (0 LDUs), while
  the same signal offset-tuned lands ~10 % EVM and locks (66 NIDs); on the air,
  voice went from never decoding to clean IMBE audio. The composer is unchanged
  — the offset is fully encapsulated in a per-device tuner wrapper.
- **The HackRF front-end RF amplifier is now configurable via `rf_amp`.** The
  HackRF has no true AGC, so `gain: auto` uses a fixed LNA/VGA split with the
  front-end amp off. Setting `rf_amp: true` on a device turns the amp on for the
  auto preset, lowering the noise figure by ~14 dB to recover a weak-signal site
  (matching SDRTrunk's amp-on default) — but because it adds gain ahead of
  everything it can overload a front end near a strong transmitter, so it is
  opt-in and off by default. Manual (positive) `gain:` values are unaffected, and
  the option is ignored, with a startup warning, on a device without a switchable
  amp.
- **DMR now auto-corrects a small residual tuner carrier offset.** The
  narrowband DMR C4FM decoder tolerates only ~±75 Hz of carrier error before
  the 4-level slicer mis-decides and nothing decodes (issue #836) — at 446 MHz
  even a fraction of a ppm exceeds that. The receiver now runs the same coarse
  AFC the P25 Phase 1 decoder already used (issue #275), recentring the symbol
  eye, so a lightly-mistuned dongle (up to roughly ±2 ppm) decodes without
  hand-setting `sdr.ppm`. Grossly-mistuned dongles still want a measured
  `sdr.ppm`; automatic correction of large offsets is a follow-up.
- **Conventional DMR / IPSC repeaters now report "site alive" from their idle
  beacons.** On a `dmr-tier2` channel that parks on a fixed carrier with no
  control channel, the periodic idle beacon a repeater emits between calls (a
  CSBK Preamble / broadcast burst carrying valid sync + colour code but no
  voice) is now recognised: a CRC-valid CSBK marks the site alive, is counted
  in the wideband engine's per-channel diagnostics (`beacons`), and logs a
  rate-limited status line — instead of being ignored or, worse, surfaced as a
  decode error. Between-beacon noise on the parked channel stays silent. Groundwork
  for conventional/IPSC monitoring (issue #1036).
- **On-air call priority now reaches the webhook sinks.** The completed-call
  and grant `broadcast.webhook` payloads already carried the emergency flag but
  not the signalled priority level (the low 3 bits of the P25 / DMR Service
  Options octet, or the TETRA CMCE Call priority); both now include a `priority`
  field, matching what the call log and `/api/v1/calls` already surface, so a
  webhook consumer can rank calls the way the local UI does.
- **Unit-to-unit / private calls are now flagged in call history.** The
  `Individual` grant flag (the call's `group_id` is a target radio address,
  not a talkgroup) already reached the live-grant API but was dropped from the
  call record, so a followed private call was indistinguishable from a group
  call in history and its 24-bit target rendered as a phantom talkgroup. Each
  call record now carries an `individual` field (persisted at call start and
  latched on at call end for TETRA, which cannot flag a unit-to-unit
  destination on first sighting), returned by the `/api/v1/calls` endpoint.
- **Call history now shows who was talking, not just the RID number.** Each
  call record carries a `source_alpha` field resolving the source radio's
  alias/name — preferring the operator-curated RID catalogue (`rid_alias_file`)
  and falling back to the most-recently-decoded over-the-air talker alias. The
  alias is persisted to the call log and returned by the `/api/v1/calls`
  history endpoint alongside the existing talkgroup alias, and is re-resolved
  at call end so a compressed grant whose source RID is backfilled mid-call
  still lands with a name.
- **TETRA call priority is now surfaced.** The CMCE parser already decoded
  the mandatory 4-bit Call priority (and derived the emergency flag from it);
  the value now reaches `Grant.Priority` and the call log, extending on-air
  call-priority metadata to TETRA alongside P25 and DMR.

### Fixed
- **Conventional DMR (IPSC) and other camp-on-idle systems no longer spam
  "hunt failed" while simply idle.** A conventional DMR / IPSC repeater has no
  continuous control channel — it sits silent between transmissions — but the
  control-channel supervisor treated a dwell that ended without a lock as a
  failure: it logged `cchunt: hunt failed — no control-channel lock`, published a
  `cchunt.failed` event, and backed off exponentially (up to 60 s), so the
  channel was often retuned away right when someone keyed up. Protocols that
  legitimately camp on a fixed idle channel — conventional DMR (`dmr-tier2`),
  DMR Tier I (`dmr-tier1`), and TETRA DMO (`tetra-dmo`) — now enter a new
  **`camped`** hunt state instead: the supervisor keeps re-dwelling on the
  frequency (no backoff, no failure event) so the decoder catches the next burst
  and locks. Trunked systems (P25, DMR Tier III, TETRA TMO, …) are unchanged — an
  idle hunt is still a real failure for them. (issue #1036)
- **Conventional scanner recordings no longer run long or hang open on brief
  noise.** A `scanner.conventional` channel released a call only after `hangtime_ms`
  of continuous below-squelch silence, but the trailing-edge logic let a *single*
  above-threshold IQ chunk (~2 ms — a carrier tail, impulse noise, intermod, or a
  momentary tone-detector dropout's recovery) zero the whole countdown. On a
  channel where such blips recur faster than the hangtime, the countdown could
  never complete and the recording stayed open indefinitely, ending only on some
  unrelated event. The scanner now de-bounces renewed activity: carrier (and tone,
  when gated) must be present for a sustained window — new `activity_debounce_ms`,
  default 50 ms — before it resets the countdown, and a new `squelch_hysteresis_db`
  margin (default 3 dB) keeps a signal hovering at the threshold from chattering
  the countdown. Both are per-channel with sane defaults, so existing configs need
  no changes. (issue #1090)
- **Conventional scanner channels now record audio.** A `scanner.conventional`
  (analog FM) channel detected activity, opened a synthetic call, and named a WAV
  path, but no file was ever written. The scanner holds the voice SDR's
  single-consumer IQ stream open for the whole dwell (to watch squelch/tone), so
  when the composer's FM voice chain tried to open a *second* `StreamIQ` on the
  same physical device the driver rejected it with `rtlsdr: stream already
  active`; the chain never started and no PCM reached the recorder. The scanner
  now streams through the device's iqtap broker and the composer's FM chain
  Subscribes to that fan-out instead of opening a colliding second stream, so it
  receives copies of the exact IQ the scanner is already reading during the call.
  (issue #1075)
- **The Plots signal-quality banner no longer reads like a bogus symbol rate.**
  The banner's "N symbols" figure is the count of symbols in the rolling
  signal-quality analysis window (capped at 4000), but sitting directly above the
  constellation's "18000 sym/s" it looked like the symbol rate stuck at 4000. It
  is now labelled "N sym analysed" with a tooltip clarifying it is the analysis
  window, not the rate.
- **TETRA radio IDs still leaked into recordings as phantom talkgroups when a
  group grant was missed.** On a same-carrier site a group call's first
  control-channel message is a source-less notification (`SourceID==0`, `dst=` the
  calling radio's own SSI); the engine holds it 500 ms for the authoritative group
  grant (`src=SSI dst=GSSI`) to supersede it. Under marginal RF that grant is
  sometimes never decoded, so the hold flushed into a recording filed under the
  radio ID as if it were a talkgroup (`recordings/<sys>/<radioID>/…`). The prior
  safety only dropped the flush when the SSI was already a *known* radio — which
  missed a radio never heard transmitting, and fired too late for one learned only
  after the call opened (the retraction cleans the Talkgroups/observed lists but
  never a live recording). A source-less TETRA notification is addressed to the
  paged radio's own SSI, so a never-superseded one is now flushed as an
  **individual** call to that SSI: filed under `recordings/<sys>/individual/<SSI>/`
  with `"individual": true`, never masquerading as a talkgroup, and no longer
  dropped (the voiced audio is preserved). No group is guessed; a call whose group
  grant was entirely missed keeps the honest individual-call label rather than a
  fabricated talkgroup, so its `srcList` reflects that (reconstructing the full
  multi-talker list would require attributing the call to a group GT never decoded).
- **DMR 2-slot voice: a wrong same-slot cadence guess could garble a whole
  call and never recover (reopened #644).** When a call opens on a section
  whose embedded Link Control doesn't decode, the interleaved decoder picks
  the same-slot stride (264 vs 288 dibits) by AMBE Golay-FEC quality. That
  guess was frozen for the rest of the call — and because a wrong stride
  slices every later burst across the timeslot boundary, the CRC-valid LC
  that would correct it can never reassemble, so one confident-but-wrong
  guess produced persistent "sounds-encrypted / DJ-scratch" garble. An
  FEC-quality lock is now **provisional**: a later CRC-valid embedded LC (or
  a later clear FEC winner at a different stride) overrides it and re-locks
  the correct cadence, after which the LC lock is authoritative. Single-slot
  (Tier I) decoding is unchanged. Pinned by
  `TestInterleavedDecoderLCOverridesWrongProvisionalCadence`.
- **macOS: RTL-SDR (R820T) dongles failed to open with "no supported tuner
  detected."** On Apple silicon the tuner's first I²C-bridge burst write could
  STALL the USB control pipe (IOKit `kIOUSBPipeStalled`, `kern_return
  0xe000404f`); unlike the Linux (`EPIPE`) and Windows (`ERROR_GEN_FAILURE`)
  backends, the macOS backend didn't recognise this as the recoverable stall it
  is, so the existing R820T cold-boot burst-write retry never fired and tuner
  init aborted — leaving a plainly-present dongle reported as having no tuner.
  The macOS backend now maps the IOKit stall to the shared retry path, matching
  the other platforms. (#1038)
- **P25 Phase 2 grants could carry a garbage encryption Algorithm ID.** A
  bit-errored MAC Encryption Sync decodes to an Algorithm ID outside the
  TIA-102 registry; the control channel stored any decoded sync and attached
  it to every subsequent encrypted grant, so a single mis-decode leaked a
  plausible-but-wrong `algorithm_id` + `key_id` onto grants (and downstream
  onto the call record / webhook). It now refuses to store an out-of-set sync
  as the reference — the same validity gate the voice path already applied —
  so a protected grant with no valid sync reports encrypted with alg/key 0
  rather than smearing garbage keys. (#924)
- **WebUI live-audio playback: silence, PTT clicks, and dropped calls.** The
  cockpit's live-audio player rebuilt a fresh `AudioContext` every time it
  followed a new call, and on teardown only suspended (never closed) it — so on
  a busy system it hit Chrome's ~6-context-per-page ceiling and went silently
  dead (worst on macOS Chrome). It now reuses a single context and re-points the
  stream in place. A short fade envelope in the ring-buffer worklet removes the
  broadband "DC-spike" click at PTT-in / PTT-out and mid-stream call seams, and a
  follow-hold policy plays the current call to completion before advancing to the
  newest — instead of cutting it off the instant a newer grant appears ("eating
  conversations"). No theme or layout changes.
- **DMR call-startup "scratch" is now squelched.** The AMBE+2/DMR decoder gained
  the same call-startup acquisition squelch the P25 IMBE decoder already had:
  while the receiver is acquiring lock, the FEC resolves the marginal dibit
  stream to random-but-valid AMBE+2 parameters that synthesise as a loud burst at
  the start of a transmission. Output is now muted until a sustained run of
  stable-pitch voiced frames confirms real speech (failsafe-released after ~2 s),
  removing the per-PTT onset scratch. Opt-in and enabled per call by the
  recorder; the raw decode path stays byte-identical. (The residual synthetic
  timbre of software AMBE+2 across a whole call is intrinsic to software decode.)
- **Encrypted / emergency DMR Tier III calls were logged clear.** A DMR
  Tier III channel-grant CSBK carries no service-options octet, so a
  followed call's encrypted / emergency / priority state never reached the
  call record even though those bits are decoded on the traffic channel. The
  DMR voice chain now backfills them (plus the source RID) from the embedded
  voice LC via the in-call `CallSourceUpdate` path the P25 Phase 2 chain
  already uses — so an encrypted Tier III call is now recorded as encrypted.
- **Mid-call emergency / priority was still dropped at the call record.**
  The engine backfills a followed call's Emergency and Priority (decoded on
  the traffic channel, since a DMR Tier III / P25 Phase 2 grant carries no
  Service Options) onto the bound grant, but the call-log end update only
  persisted the backfilled encrypted flag and source RID — so an emergency
  Tier III call was still written non-emergency at priority 0. The end update
  now latches emergency on and takes a non-zero priority, the same
  never-downgrade discipline it already applied to encrypted.

### Added
- **rdio-scanner uploads forward the talkgroup tag and group.** The RdioScanner
  broadcast backend now sends `talkgroupTag` / `talkgroupGroup` when known, so the
  console shows GopherTrunk's own tag/group instead of falling back to its static
  per-system config. (The console's system-type label — e.g. "P25" — is not
  settable: the call-upload API has no protocol/type field.)
- **`gophertrunk replay -record-voice -out-dir <dir> -freq <Hz>`** decodes and
  records voice from a capture, not just control-channel locks/grants. It wires
  the production voice path (trunking engine → voice composer → recorder) onto
  the replay decode: grants bind a same-carrier voice source fed by the decode's
  own channelized IQ, and each followed call is written as `.wav`/`.raw`/`.json`.
  Best for conventional systems whose voice rides the decoded carrier (DMR
  Tier II / IPSC, TETRA) — it's the offline way to validate that a capture
  produces audio, and the reproduction vehicle for #1036. Requires `-freq` (a
  grant with frequency 0 is dropped) and does not support `-auto-tune` (use
  `-tune-hz`). Backed by a new siglab `Config.Bus` / `Config.OnChannelIQ` seam.
- **On-air call priority is decoded and surfaced.** The call's signalled
  priority level — the low 3 bits of the P25 / DMR Service Options octet —
  is now carried on the grant (`Grant.Priority`), persisted to the call log,
  and returned by the `/api/v1/calls` history endpoint, so an operator can
  see which calls the radios flagged as high-priority alongside the existing
  emergency / encrypted metadata. Decoded from P25 Phase 1, P25 Phase 2, and
  DMR (group and unit-to-unit voice LCs). This is the calling radio's
  requested priority (distinct from a talkgroup's operator-configured
  priority); engine preemption is unchanged.
- **Followed DMR calls (Tier III / Tier I) end promptly on the Terminator.**
  The voice chain now detects the Terminator-with-LC burst on the followed
  traffic channel — reusing the trusted control-path FEC chain (slot-type
  Hamming, BPTC, RS(12,9) terminator seed) so a garbled data burst can't
  forge an end-of-call — and publishes a `call.release` for the matching
  group, so the engine tears the call down at once instead of waiting out
  the hangtime. Extends the Tier II conventional prompt-release to followed
  calls; strictly additive (the hangtime path still ends a call whose
  terminator never decodes).
- **DMR Tier II conventional calls end promptly on the Terminator.** The
  decoder now publishes a `call.release` when it sees the explicit
  Terminator-with-LC burst, so the engine tears the call down at once
  instead of waiting out the composer's hangtime / no-voice timers — the
  same prompt-teardown TETRA already drives from D-RELEASE. Recorded call
  durations match the air more closely.
- **DMR private (unit-to-unit) calls route and attribute correctly on
  interleaved carriers.** The 2-slot `slotRouter` now binds a private call's
  timeslot from its unit-to-unit embedded LC (by called subscriber), instead
  of only recording it via the phase fallback, and the voice chain learns the
  call's source radio from a unit-to-unit LC too — so talker-alias / GPS
  metadata is attributed to the caller on private calls, not just group calls.
  Completes the unit-to-unit voice-follow work.
- **DMR private (unit-to-unit) calls are now followed.** A Tier II
  conventional channel dropped any non-group Voice LC Header, so DMR private
  calls were invisible — no grant, never followed, never recorded. The
  conventional decoder now publishes a grant for a unit-to-unit voice LC
  (marked `Individual` so the called subscriber isn't listed as a
  talkgroup), and the call is followed on the tuned frequency like a group
  call.

### Fixed
- **Conventional DMR (Tier II / IPSC) voice never recorded (#1036).** A
  conventional DMR repeater carries voice on the *same* carrier it emits Voice
  LC Headers on, but the daemon's same-carrier voice tap was gated to TETRA
  only — on the assumption that "for P25/DMR voice is always on a different
  carrier," which holds for trunked Tier III but not conventional Tier II/I. So
  a single-SDR conventional DMR system had no voice source: the grant fired,
  found no voice device, was dropped, and nothing recorded (the decode itself
  was fine end to end). Same-carrier voice taps are now registered for
  `dmr-tier2`/`dmr-tier1` too (two, for the 2-slot carrier). Trunked Tier III
  and P25 still register none, so their "no voice SDR" diagnostic is preserved.
- **Trailing voice frames dropped at the end of a transmission.** When a call
  ended, the recorder finalized and deleted the recording session on
  `CallEnd` while the composer's voice chain was still draining its buffered
  audio in parallel (two independent event-bus subscribers, no ordering) — so
  the last speech frames the chain wrote during teardown landed on an
  already-deleted session and were silently dropped. This was most visible on
  TETRA same-carrier calls, whose shared-demux owner worker could still be
  draining queued bursts when the chain reported done. The recorder now defers
  finalize until the composer signals the chain has fully drained (with a
  safety timeout so a dropped `CallEnd` can't hang a recording), and the
  same-carrier chain waits for its owner worker to finish draining before
  reporting done. Digital voice (TETRA/DMR/P25/ProVoice) keeps its full tail.
- **DMR Tier III private-voice grants were mislabelled as talkgroups.** The
  shared grant path never set `Individual`, so a Tier III private
  (unit-to-unit) call's destination subscriber was published as if it were a
  talkgroup — polluting talkgroup discovery. Private-voice grants (standard
  and vendor) now carry `Individual=true`.
- **DMR GPS position decode → live map.** A DMR radio's GPS Info embedded
  Link Control (FLCO 0x08) is now decoded — a 25-bit two's-complement
  longitude and 24-bit two's-complement latitude per ETSI TS 102 361-2,
  cross-checked against the ok-dmrlib reference — and published as a
  `location` event bound to the call's source radio. The existing
  `location_log` storage and web map surface it, so a DMR subscriber's
  position appears on the map alongside P25 the same way. Stationary radios
  that rebroadcast an unchanged fix produce one row, not one per superframe.
  The wire layout is a working model pending on-air capture validation.
- **DMR talker alias decode.** A DMR radio's display name — carried in the
  voice superframe's embedded Link Control as a header (FLCO 0x04) plus up to
  three continuation blocks (0x05-0x07) — is now reassembled and published as a
  `talker.alias` event, the same surface P25 already uses, so the affiliation
  tracker, decoded-message log, API, and TUI attribute the name to the calling
  radio on DMR too. All four data formats (7-bit packed, 8-bit ISO 8859-1,
  UTF-8, UTF-16BE) decode; the header text-bit boundary follows ETSI
  TS 102 361-2 §7.2.18 cross-checked against the ok-dmrlib reference and is a
  working model pending on-air capture validation.
- **Event API reference docs (`docs/api-events.md`).** A stable-contract reference
  for the real-time telemetry surface: the SSE (`/api/v1/events`) and WebSocket
  (`/api/v1/events/ws`) transports and shared event envelope, the full JSON payload
  schema for the ten curated event DTOs (grant, call.start/end, encryption,
  affiliation, registration, unit.request, patch, DMR grant/bandplan), the per-call
  and per-grant webhook payloads, config keys, and reliability caveats (no stream
  auth, no server-side filtering, slow-subscriber drop). Refs #268.
- **Baseband auto-record trigger on control-channel sync loss
  (`baseband.auto_record.on_cc_sync_loss`).** Fires an IQ capture when a locked
  control channel suddenly loses sync (`cc.lost`, which only fires after a genuine
  lock — never for a hunt that never locked), recording the re-acquisition that
  follows. That is exactly the raw IQ needed to debug sync-loss and slow
  warm-up-lock episodes, where the carrier is present but GT fails to re-lock.
  Off by default; pair with `tap: ddc` for small, directly-replayable captures.
- **TETRA in the live Signals/DSP scopes.** The Constellation / Symbol / Histogram
  / Tuning / Mixer scopes gain a **TETRA** mode, plotting the π/4-DQPSK
  constellation (the four ±45°/±135° clusters) and dibits from the TETRA receiver's
  existing soft tap. The offline Signal Lab's deep visuals (EVM/eye/rotation) stay
  P25-only for now.
- **TETRA on the wideband multi-system path.** A `role: wideband` SDR can now
  follow multiple TETRA control channels alongside DMR/P25, so several TETRA
  sites/systems share one dongle. The blocker was never the protocol allowlist —
  the DDC/channelizer banks emitted one bank-global 48 kHz per-tap rate, while
  TETRA's 18000-baud π/4-DQPSK needs 144 kHz; `tuner.DDCBank` now supports per-tap
  output rates (`AddTapAtRate`), and the wideband engine gives each tap its
  protocol's rate and forces the per-tap DDC strategy when TETRA is present (the
  polyphase channelizer can't emit 144 kHz). The wideband path multiplexes
  control channels; TETRA voice grants still follow on a `role: voice` SDR.
- **Live signal level (dBFS) in the Scanner cockpit.** The locked carrier's mean
  channel power is surfaced per system as a numeric read-out + bar, so an operator
  can aim an antenna / trim LNA gain against a live number instead of only the
  clean/marginal/poor quality pill.
- **TETRA network identity on the Systems page.** A TETRA system's decoded MCC/MNC
  (MNI), Location Area and colour code are now surfaced in place of the P25
  WACN/RFSS/Site fields, which have no TETRA analogue.
- **Push grant webhook (`broadcast.grant_webhook`).** A new outbound sink POSTs
  one JSON object per control-channel grant the moment GopherTrunk decodes it —
  the push counterpart to the pollable `GET /api/v1/grants` and the live
  `KindGrant` SSE stream, all on one schema. The payload carries the grant as
  decoded (system, protocol, talkgroup, source RID, frequency, P25 site
  identity, timeslot, encryption + algorithm/key), stamped with the decode time.
  Grants are coverage-truthful: a grant whose TSBK carried `source_id=0` is sent
  with `source_id=0`, never backfilled — so the feed reports exactly what the
  control channel saw, the way SDRtrunk's grant log does, letting a consumer
  read the source RID off the grant at call-setup time without holding an SSE
  connection or polling. Off by default; opt in per feed with a URL, optional
  `Authorization` header, and an optional system filter. Refs #915, #268.
- **TETRA DMO (Direct Mode Operation) burst-framing foundation.** First increment
  toward decoding TETRA's infrastructure-less peer-to-peer mode (ETSI EN 300 396-2,
  part 2 — radio aspects): a burst detector/slicer for the Direct Mode
  Synchronisation Burst (DSB) and Normal Burst (DNB) that recovers each burst's
  SCH/S / BKN1 / BKN2 type-5 blocks from a demodulated dibit stream. DMO shares
  TMO's π/4-DQPSK air interface, so this reuses the existing demod, training
  sequences (verified against the DMO spec equations) and scrambler; only the
  DMO-specific burst field layout is new. Not yet wired end-to-end — the DMO
  channel decode, the EN 300 396-3 call-control protocol (source/group SSI, call
  type), and a control-channel-less scanner ingestion path are the remaining
  stages, and none is validated against a real DMO capture yet.
- **TETRA recordings split per talker, so every over is attributed to its source.**
  A group call stays on one traffic channel while members key up in turn; GT
  recorded the whole call as one WAV tagged with only the first talker, so a reply
  from a second member had no correctly-attributed file. The per-transmission split
  pipeline P25 already uses (a `CallSegment` boundary rolls the recording to a fresh
  file named from the current source) is now driven for TETRA: on a talker change
  (D-TX-GRANTED → call talker) the engine emits the boundary before backfilling the
  new source, so the closing file keeps the previous talker and the next over opens
  a file named with the new one. Honors the existing `voice_call_grouping` setting
  (default splits per transmission, matching P25; `conversation` keeps one file per
  call). No configuration change.
- **TETRA demodulator equalizer recovers garbled voice recordings.** On
  concurrent-load captures a residual garble survived the soft-decision TCH/S
  decode: a linear channel / ISI defect (multipath, band-edge group delay) was
  smearing the π/4-DQPSK constellation — the demod-side gap between GopherTrunk
  and radios that equalize. A blind Constant-Modulus-Algorithm equalizer now sits
  between symbol-timing recovery and the differential decoder on the TETRA voice
  path, inverting that channel. Across the reporter's six captures it roughly
  doubles CRC-valid TCH/S burst yield (soft-decision 410→778, ~1.9×; one call
  went 4→207 bursts, another 42→134) with no loss on already-clean captures. On
  automatically (voice path); no configuration change. Refs #764, #771, #1001.
- **Cross-site duplicate-recording suppression (`recordings.dedup`).** When
  monitoring several networked / simulcast sites where the same talkgroup carries
  the same traffic, a call heard on more than one system is now saved once
  instead of once per site. Enable with `recordings.dedup.enabled: true`
  (`window_seconds` defaults to 60): a call whose `(talkgroup, source)` was
  already recorded from a *different* system within the window is skipped. Keying
  on the calling-radio RID (globally unique in a network) means two genuinely
  different calls that share a talkgroup number across systems still both record
  when their sources are known; a re-key on the same system is never suppressed;
  and live monitoring is unaffected (recording-only). Off by default.
- **`GET /api/v1/grants` — a pollable log of recent control-channel grants.**
  GopherTrunk decodes a source RID off most `GRP_VCH_GRANT` TSBKs, and already
  streams every grant live over the `grant` SSE event; this adds the pollable
  snapshot form so a telemetry consumer can read the source RID (plus talkgroup,
  frequency, channel, site and encryption) straight off the grant — the way
  SDRtrunk's control-channel grant log exposes it — without holding an SSE
  connection. The response reuses the same stable `GrantDTO` schema the SSE
  stream emits, with a per-row `at` timestamp; `?limit=` and `?system=` narrow
  the result. Backed by a bounded in-memory ring (newest-first, always on), so a
  busy system never grows the log without bound (issue #915, reporter fix #2).
- **TETRA control-channel PDUs that span multiple MAC blocks are reassembled
  (MAC-FRAG / MAC-END).** A TM-SDU too large for one MAC block arrives as a
  start-fragment MAC-RESOURCE followed by MAC-FRAG and a MAC-END (ETSI EN 300
  392-2 §21.4.3.3/.4); those continuation PDUs were previously dropped, losing
  the L3 CMCE PDU carried across them (a large D-SETUP's call-identifier→group
  mapping, calling-party SSI, and emergency flag). The channel allocation is
  complete on the start fragment, so the voice grant still publishes immediately;
  the reassembled MAC-END now recovers the full call-control PDU and enriches the
  call with its source and emergency state.
- **TETRA network-configuration report now shows the control channel's uplink.**
  The cell's uplink carrier, derived from the SYSINFO duplex spacing and the
  frequency offset (§21.4.4.1), was computed but dropped before the report; the
  primary control channel now renders its `UPLINK:` frequency alongside the
  downlink.
- **Soft-decision decode on the TETRA grant (SCH/F + SCH/HD) path.** The
  signalling-channel decode on the grant-bearing blocks now uses the per-symbol
  soft (log-likelihood) information when available, falling back to the
  hard-decision Viterbi path, recovering grants on marginal control channels that
  a hard decision drops.
- **P25 Phase 2 blind CMA equalizer on the traffic-channel receiver
  (`p25_phase2_equalizer: on`).** A new opt-in adds a blind constant-modulus
  (CMA) adaptive equalizer on the Phase 2 symbol stream, after carrier
  recovery and before the differential decode. It removes residual
  inter-symbol interference a real channel leaves on the symbols — RRC
  pulse-shape mismatch, a fractional Gardner timing error, mild multipath —
  which widens the differential-phase decision and costs the outer RS(24,16,9)
  symbol errors on an otherwise-decodable burst. CMA (not decision-directed) is
  used because the H-DQPSK absolute constellation spins π/8 per symbol and has
  no fixed phase grid, so a phase slicer has nothing to lock to; CMA is
  rotation-invariant and drives the modulus toward unity blindly, leaving the
  phase to the differential decode. Through the production receiver, a
  multipath channel that dropped a Phase 2 MAC-payload recovery to a fraction
  of bursts is restored to near-full recovery; a clean signal decodes
  byte-identically (the centre-spike init is transparent). Off by default — an
  AWGN-limited channel gains nothing from equalization, so it is opt-in; pairs
  with `p25_phase2_soft_decision` and `p25_phase2_rs_mode: correct`. (issue #915)
- **P25 Phase 2 outer RS can now error-*correct* weak MAC PDUs
  (`p25_phase2_rs_mode: correct`).** The `p25_phase2_rs_mode` knob gains a
  `correct` (alias `fix` / `ecc`) setting that runs the RS(24, 16, 9) outer code
  as a bounded-distance error corrector — repairing up to t=4 GF(2⁶) symbol
  errors (Berlekamp-Massey + Chien + Forney) before the MAC PDU is parsed —
  instead of the detection-only `on` mode that drops any PDU with a residual
  symbol error. This is the weak-frame recovery path for the completed-call
  source-RID gap: a traffic-channel `GROUP_VOICE_CHANNEL_USER` PDU that framed
  and descrambled at the right phase but carries a handful of symbol errors from
  marginal-SNR demod now still yields its source RID, where before it was
  rejected and the call landed with no source. `correct` strictly supersets
  `on` (a clean codeword decodes identically); because t=4 correction admits a
  small fraction of random windows, a PDU that actually needed correction is
  additionally gated on a recognised MAC opcode so a wrong descramble phase
  cannot be miscorrected into a bogus source RID. Off by default; pairs with
  `p25_phase2_soft_decision: on`. (issue #915)
- **`call.end` real-time event now carries `duration_ms`.** The SSE/WebSocket
  event stream's call-completion event (`event: call.end` on `/api/v1/events`
  and the WS stream) now includes the call length in milliseconds alongside
  `started_at`/`ended_at`, matching the completed-call webhook's `duration_ms`,
  so an SSE/WS-only consumer (a Prometheus exporter, a Grafana feed) reads a
  call's duration off the completion event without pairing it back to
  `call.start` and subtracting timestamps itself. Also confirmed and pinned by
  a regression test that the P25 control-channel `nac` — site identity
  alongside `rfss_id`/`site_id` — is present on every P25 grant / affiliation /
  registration event (threaded from the decoded NID; omitted only for non-P25
  protocols where NAC does not apply). (issue #268)
- **Motorola P25 talker-alias ciphertext is logged as cryptanalysis ground
  truth.** When a Motorola FACCH-S talker alias reassembles on a Phase 2
  traffic channel, GopherTrunk now emits a `p25p2 alias ciphertext` log line
  carrying the source RID, talkgroup, and the reassembled encoded-alias bytes
  (`encoded_hex`) — the `rid,talkgroup,encoded_hex,alias` record the alias
  cipher analysis consumes. The proprietary alias cipher is still gated
  (unsolved, so the decoded name stays blank), but this lets an operator
  harvest the chosen-plaintext / known-RID corpus needed to finish it
  (`research/p25-talker-alias-chosen-plaintext.md`) using GopherTrunk alone
  instead of SDRTrunk (#773).
- **P25 Phase 2 soft-decision demod (`p25_phase2_soft_decision`).** A new
  per-system opt-in knob (on/off, default off) that builds the Phase 2
  traffic-channel receiver with soft-decision decoding: the demodulator's
  soft symbol differentials feed a per-bit soft Viterbi on the MAC trellis,
  recovering the ~1.5–2 dB of coding gain the hard slicer discards. On weak
  signals this can surface the clear-MAC source RID the hard path drops; on
  strong signals it is neutral, and with the knob off the decode is
  byte-for-byte unchanged. Applies to the voice composer and signalling
  follower (including Phase 1 control channels that grant Phase 2 traffic).
  Issue #915.
- **`baseband.auto_record.tap: ddc`** — event-triggered auto-captures can now
  record the control decoder's narrowband post-DDC stream (the pipeline rate,
  144 kHz for TETRA) instead of the full-rate wideband SDR IQ. Files are orders of
  magnitude smaller (a few MB vs ~95 MB) and directly replayable; for a
  same-carrier TETRA site the DDC tap holds all four voice timeslots of the
  control carrier. `tap: wideband` (default) is unchanged. Triggered captures also
  create their target directory if missing.
- **TETRA voice now decodes to audible audio, including up to 4 concurrent
  calls on one carrier.** The TETRA voice path recovers each traffic burst,
  channel-decodes TCH/S, and renders it with the clean-room ACELP vocoder
  (`tetra-acelp`), now bit-exact to the ETSI EN 300 395-2 reference. Each burst
  is tagged with its AACH downlink usage marker (the per-slot call identifier),
  and the daemon registers up to four `cc:same-carrier:N` taps, so up to four
  simultaneous calls on one control carrier — demultiplexed by matching each
  call's grant usage marker — record into their own files instead of only one
  binding and the rest being dropped with "no voice device available for grant".
  The same-carrier voice device serial changes from `cc:same-carrier` to
  `cc:same-carrier:1..4`.
- **Event-driven raw-IQ auto-recording (`baseband.auto_record`).** The daemon
  can now capture a short slice of the control SDR's raw IQ whenever a
  classified event fires — `on_concurrent_calls: N` (N+ calls active at once),
  `on_no_voice_device` (a grant arrived but every voice tuner was busy),
  `on_encrypted`, or `on_emergency` — plus a manual
  `POST /api/v1/siglab/autorecord/trigger`. Each capture lands in the
  configured `dir` with a self-describing name
  (`<system>_<UTC>_<reason>_<freqHz>_<rateHz>hz.<ext>`) and a `.metadata.json`
  sidecar, so it drops straight into `gophertrunk replay` / siglab. A cooldown
  and an in-flight cap keep a burst of grants from spawning a capture storm.
  This is the event-based debugging hook for capturing hard-to-decode moments
  (concurrent grants, unknown packets) as they happen.
- **`baseband.auto_record` can capture the narrowband DDC output (`tap: ddc`).**
  The default `tap: wideband` records the control SDR's full-rate raw IQ
  (~50 MB per 30 s at 2.5 MS/s); `tap: ddc` instead records the control
  decoder's channelised post-DDC stream at the pipeline rate (144 kHz for TETRA,
  ~48 kHz for the C4FM family) — orders of magnitude smaller and directly
  replayable with `replay -format wav` / siglab. For a same-carrier TETRA site
  the DDC tap holds all four voice timeslots of the control carrier, so a
  triggered capture of a hard-to-decode concurrent-call moment stays small
  enough to share. Triggered captures also now create their target directory if
  it does not exist.
- **siglab capture start time in the capture list.** Each capture row now shows
  its recording start time, so otherwise-identical grabs of the same carrier
  (e.g. three `capture-AIRSPY SN:…` captures) can be told apart at a glance.
- **Startup warning when paging is configured without storage.** A
  `paging.pocsag` / `paging.flex` / `paging.wideband` subsystem with no
  `storage.path` set decodes fine and consumes live IQ, but decoded pages are
  never persisted and `GET /api/v1/pager/messages` returns `503`, so the
  Pagers panel shows a bare `pager/messages 503` with no hint that storage is
  the missing piece. The daemon now surfaces this at load time as a startup
  warning (alongside the launcher / TUI dashboard warnings) instead of leaving
  the misconfiguration silent. (issue #565)
- **Per-site P25 Phase 1 demod mode on wideband multi-site taps.** A single P25
  system can now mix modulation across its sites: set the system-level
  `p25_phase1_demod_mode` to the majority modulation and add a per-channel
  `p25_phase1_demod_mode:` override on the wideband dongle's `channels:` entries
  for the exceptions (blank inherits the system default). Recognised values
  match the system-level key (`c4fm`/`fm` vs `cqpsk`/`lsm`/`linear`). The
  override is keyed per control-channel frequency — the one place a site's
  identity is known before its CC locks — and drives both the control-channel
  decode and the voice grants that tap issues, so a granted call on an LSM
  simulcast site is decoded on the LSM path instead of timing out with no LDU
  (a C4FM demodulator can't recover a linearly-modulated simulcast signal —
  the symptom was `control_channel_decode_quality: poor` on urban simulcast
  sites regardless of hardware). This also closes a latent gap where the
  wideband path never stamped a demod mode onto its grants at all, so wideband
  P25 voice always ran the C4FM chain regardless of the system setting.
  (issue #935)
- **LoRa Low Data Rate Optimization (LDRO).** The LoRa PHY now decodes the
  reduced-rate payload mode LoRa uses at high spreading factors, where the long
  symbol lets clock drift smear the dechirp peak across the two least-significant
  chip bins. LDRO carries the payload at SF-2 meaningful bits per symbol (the
  transmitted bin is always a multiple of four), tolerating that drift. It is
  auto-detected from the (SF, BW) pair per Semtech's Ts >= 16 ms rule — on for
  SF11/SF12 at 125 kHz and SF12 at 250 kHz — on both the demodulator and the
  built-in modulator, so no configuration is needed for standard networks;
  without it those (very common, long-range) frames could never be decoded. A
  per-sub-channel `low_data_rate_optimize: auto | on | off` override handles
  networks that deviate from the recommendation. (issue #586; the reduced-rate
  SF-2 *header* region and bit-exact Semtech interop remain gated on captured
  golden vectors.)
- **P25 Phase 2 per-call MAC framing-health signal (`mac_rs_valid`).** The
  end-of-call census line now reports how many of the decoded traffic-channel
  MAC PDUs carried a valid outer RS(24,16,9) parity, and the per-opcode
  `p25p2 mac pdu` debug line carries an `rs_valid` flag. `mac_pdus>0` with
  `mac_rs_valid=0` is the built-in fingerprint of a mis-framed / mis-descrambled
  Phase 2 superframe (random bytes parsing as MAC PDUs rather than real
  signalling) — the objective before/after metric for a superframe-framing fix,
  and the reason a system's clear-MAC source RID never lands (issue #915).
- **P25 Phase 2 control-channel MAC opcode census (`--log-level debug`).** A
  new diagnostic inventories every MAC opcode seen on the control channel —
  logging a one-shot `payload_hex` sample the first time each opcode appears
  plus a periodic `opcode:count` summary. Unlike the existing per-grant DEBUG
  line, it has no survivorship bias: it also surfaces opcodes GT does not
  currently parse as a grant, which is where a RID-bearing grant would hide on
  a system whose completed-call `source_rid` under-populates (issue #915). The
  raw opcode inventory + byte sample is what pins the remaining gap on a
  missing/mis-mapped grant opcode (decode-side) vs. a call-association gap.
- **TETRA AACH downlink usage marker parsing (per-slot control/traffic
  identification).** GopherTrunk now decodes the ACCESS-ASSIGN PDU the AACH
  carries in the centre of every TETRA downlink slot into its downlink usage
  marker (`unallocated` / `assigned control` / `common control` / `traffic`,
  per ETSI EN 300 392-2 §21.4.7). This is the per-slot MCCH indicator a
  Single Carrier Base Station running dynamic MCCH sharing relies on, where
  any of the four TDMA slots can be the control channel at a given moment
  rather than a fixed slot 1 — the building block for issue #925.

### Fixed
- **Autotune measure/log spam on TETRA.** At debug level a TETRA control channel
  emitted the autotune measurement + "suggested error" lines every second, always
  with `applied_hz=0` — the correction is only consumed by the P25 Phase 1 path, so
  measuring and logging it for TETRA was dead work. The sampler now gates on a
  narrower marker (only P25 Phase 1), leaving the #815 carrier-offset WARN intact.
- **TETRA individual calls reported as "tg == src".** A point-to-point (individual)
  call was surfaced as a phantom talkgroup whose ID equalled the transmitting
  radio's — "calling yourself." TETRA's MAC can't distinguish a group SSI from an
  individual SSI, so a call addressed to a subscriber ISSI was misread as a group.
  Using the invariant that a party can never equal the call's own identity, such
  calls are now flagged Individual with no self-source, and the self-referential
  talker update is suppressed.
- **DC spur leaked into TETRA voice under heavy multislot traffic.** Same-carrier
  TETRA voice rides the control carrier at 0 Hz offset, so the zero-IF front end's
  DC spur (worsened by ADC-clipping/IMD as concurrent-call power rises) sat on the
  wanted signal and biased the π/4-DQPSK differential decode; nothing in the voice
  path removed it. A first-order complex DC-block high-pass is now applied on the
  TETRA voice receivers (safe because π/4-DQPSK has a spectral null at DC), leaving
  the control-channel path untouched.
- **TETRA recordings ended a beat early.** On the solo voice path a control-channel
  D-RELEASE cancelled the chain immediately and dropped IQ still buffered in its
  channel; that in-flight tail is now drained before teardown. (Raising
  `voice_hangtime_ms` does not help — hangtime governs SDR release, not digital
  audio length.)
- **TETRA control-channel grant spam.** TETRA re-announces an active call's grant
  every multiframe; each was published as a separate `KindGrant` event (~3.7 per
  call). Identical grants are now de-duplicated per `(group, carrier, timeslot)`
  within a short window, while an enriched re-grant (backfilled source, emergency)
  still publishes.
- **TETRA wakeup / notification grants spawned ghost recordings named after a
  radio ID.** On a group call the SwMI sends a source-less notification (an
  Energy-Economy wakeup page, `SourceID==0`, addressed to the calling party's radio
  SSI) 50–400 ms before the authoritative group grant. The engine spawned it
  immediately, creating a ghost call and a WAV directory named after the radio ID
  (`recordings/<sys>/<radioSSI>/…src0…`) that was then torn down when the group
  grant superseded it — fragmenting audio and leaking radio IDs as talkgroups. The
  engine now holds a source-less TETRA notification for a short window (500 ms) so
  the group grant arrives first and cancels it, spawning the one real call under
  the GSSI. The hold keys on `SourceID==0` (not the `Individual` flag, which the
  first notification for a radio is published without), and a notification that is
  never superseded is dropped if it targets a known radio (else recorded under its
  real talkgroup). A genuine unit-to-unit call carries a source, so it is never
  held.
- **Cross-slot audio leaked between concurrent same-carrier TETRA calls.** The
  shared per-carrier voice demux fell back to a "sole active call" CRC gate for
  bursts it could not route by AACH marker — but a single *registered* owner is not
  the same as a single call on the *air*. When another slot carried a call GT was
  not tracking as an owner (a missed grant, a call in hangtime, a wakeup-page
  ghost), that foreign slot's speech funnelled through the one owner's CRC gate and
  bled into its recording (a valid TCH/S CRC proves a burst is speech, not *whose*).
  The demux now judges on-air concurrency from the SB-anchored physical TDMA slot
  (available on ~100% of bursts, unlike the marker) and suppresses the sole-owner
  fallback whenever two or more slots are active, so foreign speech is dropped
  rather than mis-attributed. Clean single-call captures are unchanged.
- **`ccdecoder: decode can't keep up… dropping IQ` (#402) fired at idle CPU on a
  remote USRP.** The decode queue between the IQ forwarder and the decode goroutine
  was a fixed 128-*chunk* buffer. That is only meaningful in seconds when chunks are
  large; SoapyRemote delivers a remote USRP ~369-sample datagrams, so 128 chunks was
  only ~47 ms at 1 MS/s — while the driver's own channel is sized for 400 ms and
  never overflowed, producing attributable decode overruns the driver never saw. The
  queue is now bounded by a wall-clock **sample budget** (~0.5 s) derived from the
  sample rate, so its depth-in-seconds is invariant to how the driver chunks
  delivery. Airspy/B210 (large native blocks) are unaffected.
- **`ccdecoder: iq power very low` DEBUG spam while locked and decoding.** The
  low-power hint fired whenever windowed RMS sat below −55 dBFS with no gate on lock
  state, so a USRP/SDR run at conservative gain (~−60 dBFS, ~55 dB of unused ADC
  headroom) that decoded TETRA cleanly logged it every few seconds. It is now gated
  on `!locked` — a low absolute level only matters when the decoder cannot lock — and
  throttled to 30 s to match the clip / DC-dominant siblings. The Prometheus IQ-power
  gauge still records every window.
- **TETRA radio IDs still surfaced as talkgroups in the live Active Calls list.**
  The engine's RadioID→TGID retraction cleaned the Talkgroups catalogue but not
  the control-only "observed" call set, which is the other half of what the
  Active Calls UI renders: a notification / D-CONNECT bound to the calling party
  creates a provisional call keyed by the radio ID, and the notification-supersede
  only released the pool-bound call, leaving the observed entry to linger as a
  phantom "TG <radioID>". The engine now skips individual / known-radio grants when
  recording observed calls, and retracts any observed call already tracked under an
  SSI once that SSI is revealed to be a subscriber radio.
- **TETRA metrics weren't counting.** `grants_total` was never defined or
  incremented (the metrics event handler had no `KindGrant` case), so TETRA grants
  went uncounted; TETRA voice calls were miscounted under the DMR-named
  `dmr_voice_calls_total` (its per-timeslot increment was gated only on a non-zero
  timeslot, which TETRA also carries); and the curated dashboards asked for a
  `cc_locked` metric the daemon never emits (it exports `control_channel_locked`).
  Added a protocol-labelled `grants_total`, gated `dmr_voice_calls_total` to the
  DMR protocol, and pointed the TUI/web curated panels at `control_channel_locked`.
- **A single-call same-carrier TETRA burst whose AACH usage marker miscorrected
  was dropped instead of decoded.** The shared per-carrier voice demux routes by
  AACH downlink usage marker; on a marginal signal the RM(30,14) AACH occasionally
  miscorrects to a stray marker with no registered owner, so the burst was counted
  as an `ownerless_drop` and its speech lost. When exactly one call is active the
  demux now routes such a burst to that sole call through the class-2 CRC gate
  (the same single-call fallback already used for an undecoded AACH) — the CRC
  rejects a genuinely foreign burst ~255/256, and with one call there is no peer
  to cross-talk into. With ≥2 concurrent calls the burst is still dropped, so the
  cross-talk guarantees are unchanged.
- **TETRA radio IDs leaked into the Talkgroups list.** A notification / D-CONNECT
  addressed to a call's calling party arrives as a bare grant before that SSI is
  known to be a radio, so it was published `Individual=false` and the engine
  auto-catalogued the radio ID as a phantom talkgroup (e.g. `100xxxx` source IDs
  showing up alongside real `102xxxx` talkgroups). The engine now learns which
  SSIs are subscriber radios — any grant's calling/transmitting party (`SourceID`)
  and any `Individual`-addressed destination — retracts a phantom talkgroup
  already catalogued for one (`TalkgroupDB.DeleteDiscovered`, which only removes
  auto-`Discovered` entries, never an operator-catalogued talkgroup), and refuses
  to re-discover a known radio. GSSIs and ISSIs never overlap on a TETRA network,
  so this is safe.
- **TETRA voice recordings on a same-carrier site came out short and garbled
  (low `audio_pct`).** The TCH/S traffic decoder was hard-decision while the
  control channel already ran soft-decision, so on a marginal same-carrier
  signal ~70% of a call's own speech bursts failed the class-2 CRC and were
  dropped; the recorder concatenates only the surviving bursts, so the audio
  played back choppy/robotic (the `recording shorter than call span` diagnostic
  showed `audio_pct` ≈ 30%). The TCH/S channel decode now uses the receiver's
  per-symbol soft (log-likelihood) information — a soft-input rate-1/3 Viterbi
  mother decoder with soft depuncture and descramble, mirroring the existing
  soft SCH path — recovering the ~2 dB of coding gain the hard path threw away.
  Bursts with no soft info still fall back to the hard decoder, so the change is
  never worse than before.
- **TETRA group calls could land on the wrong talkgroup with starved audio when
  the first grant was a notification.** On a same-carrier site a group call's
  first control-channel grant is often a notification / a D-CONNECT addressed to
  the calling party, which arrives before that party is known as a radio ID — so
  it is labelled with the individual's SSI (a phantom talkgroup), carries no
  source, and follows the notification's own downlink usage marker rather than
  the traffic channel's. The recording bound to that provisional identity, and
  when the authoritative group grant for the same physical channel arrived, the
  physical-channel source backfill folded it on as a mere source update — so the
  recording stayed on the wrong talkgroup and kept following the wrong usage
  marker, starving the voice demux (a few percent of the call decoded). The
  engine now supersedes a still-unfinalised, notification-bound TETRA call when
  the authoritative group grant (real GSSI, calling-party source, traffic usage
  marker) lands on its channel, so the recording follows the correct talkgroup
  and marker from the outset. TETRA-gated; P25 same-channel grants are unchanged.
- **TETRA: a corrupt control-channel block whose CRC happened to pass could
  surface a ghost grant on a foreign frequency band.** A false-positive SCH CRC
  (or a slipped bit cursor) produced a bogus voice grant whose carrier resolved
  tens of MHz from the site — e.g. `carrier 6 → 400 MHz` on a 467 MHz cell —
  spawning a phantom call on a band the site does not operate. Grants whose
  resolved carrier sits more than 10 MHz from the control channel are now
  dropped as bit-alignment / CRC artefacts (a site's downlink carriers span at
  most a few MHz).
- **`auto_record` with `tap: ddc` reported `drops=0` even when the DDC grab had
  gaps.** A triggered narrowband capture that fell behind the voice fan-out
  dropped IQ chunks (time gaps that break downstream decode), but the "captured"
  log line always showed `drops=0` — the drop count was only surfaced in a
  separate fan-out warning. The capture now reports the real subscriber drop
  count, matching the wideband tap, so a gappy grab is visible in its own result.
- **TETRA subscriber/talkgroup identities (ISSI/GSSI) were corrupted on a live
  single-carrier site.** The MAC TM-SDU is an LLC PDU (§21.2); its basic-link
  header was not stripped before the L3 CMCE PDU was parsed, so the 3-bit MLE
  discriminator and every address field were misframed and the decoded radio /
  talkgroup IDs were wrong. GopherTrunk now parses the LLC sublayer (`ParseLLC`)
  before CMCE, recovering the correct ISSI/GSSI.
- **TETRA grant frequencies ignored the SYSINFO frequency offset.** The broadcast
  frequency-offset field (0 / ±6.25 / +12.5 kHz, §21.4.4.1) was dropped, so a
  cell whose carrier actually sits at e.g. 469.88125 MHz resolved to 469.875 MHz.
  The offset (with the frequency band, duplex spacing, and reverse-operation
  flags) is now parsed and applied, so grant carriers resolve to their true
  absolute downlink/uplink frequencies.
- **Ghost TETRA recordings from teardown PDUs.** A MAC-RESOURCE whose CMCE PDU is
  a call teardown (D-RELEASE, and now D-DISCONNECT, EN 300 392-2 Table 14.9)
  carries a channel-allocation element for the resource being *reclaimed*, not a
  new call; publishing a grant for it spawned a zero-byte phantom recording. Such
  grants are now suppressed, and D-DISCONNECT is modelled so it tears the call
  down instead of leaking a ghost.
- **Individual (unit-to-unit) TETRA calls were surfaced as phantom talkgroups.** A
  grant addressed to a subscriber ISSI showed the radio ID as if it were a
  talkgroup. Calls are now classified individual-vs-group (any SSI seen as a CMCE
  calling/transmitting party is an individual radio), and an individual-addressed
  grant is flagged as such (`GrantDTO.individual`) instead of being listed as a
  talkgroup.
- **TETRA active-call tracking thrashed on a busy carrier (marker collisions,
  starved audio).** Grants for the same call on one carrier could be keyed as
  separate calls, colliding on the voice tap and starving audio. Same-carrier
  TETRA grants that share a (system, frequency, timeslot) are now folded into one
  call, backfilling the source onto the existing recording.
- **TETRA colour code locked on the first sync burst, so a single mis-decoded
  BSCH could poison a whole session.** The extended colour code (the scrambler
  seed for every BNCH/SCH/TCH block) was learned from the first BSCH and never
  re-evaluated. If that burst's bit errors slipped through the BSCH FEC as a
  valid-but-wrong codeword, the wrong scrambler locked in and every subsequent
  descramble failed silently — empty or truncated recordings until a restart.
  The colour is now adopted provisionally on the first BSCH (so a cold receiver
  still starts decoding immediately) but only locked once a second BSCH
  corroborates it; a mis-decoded first burst is corrected by the true colour
  that every later BSCH carries. An operator-configured colour is still
  authoritative.
- **Concurrent same-carrier TETRA calls could still leak audio between each
  other** — a long conversation would pick up a second slot's speech mid-stream.
  The per-call demux ran one receiver per call and routed by the AACH usage
  marker, but could not evict a peer: when a call ended and the network reused its
  usage marker for a new call while the old one lingered in hangtime, both matched
  and the old recording absorbed the new call's audio. Concurrent same-carrier
  calls now share ONE per-carrier voice demux (its sync/AACH state stays warm
  across calls) with a single owner per usage marker and most-recent-grant-wins
  eviction, so a reused marker immediately displaces the lingering call. Routing by
  the AACH usage marker (not the physical TDMA slot) is what makes this reliable:
  on real captures a single call's bursts jitter across adjacent decoded slot
  numbers, so slot-keyed routing mis-delivers them — the usage marker is stable
  per call. Grants addressed without a usage marker bind to the first unclaimed
  marker instead of accept-all-mixing.
- **A locked TETRA control channel flapped "CC hunt failed · candidates
  exhausted" every ~30–60 s while it was decoding calls fine.** Two coupled gaps:
  the control-channel hunter's success test needs a fresh `cc.locked` event within
  its dwell (default 3 s), but TETRA emits the lock only once (edge-triggered), so
  when cold acquisition (BSCH sync + colour code) outlasts the dwell the first hunt
  fails and every same-frequency re-hunt then exhausts the dwell against an
  already-locked, silent-on-the-wire pipeline; and there was no TETRA lock-loss
  watchdog (`MarkLost` had no callers), so the scanner could never leave the locked
  state on a genuine outage. Now the supervisor parks a system it already knows is
  locked instead of re-hunting it, and a new control-channel watchdog publishes
  `cc.lost` when a locked carrier decodes nothing for ~5 s (≈5 missed
  multiframes), so a genuinely dead carrier still re-hunts and recovers. (The
  ±6 kHz AFC acquisition range, #940, was already merged and is unrelated — a
  ~1.5 kHz carrier offset sits well inside it.)
- **Concurrent same-carrier TETRA calls decoded as "DJ scratches" — most calls
  produced only brief garbled fragments.** The per-slot demux dropped a burst
  whose decoded TDMA timeslot did not match the call's *granted* timeslot, but on
  real air the grant timeslot does not identify the physical slot (distinct calls
  collide on one value; the synchronisation-burst slot anchor also jitters a
  call's bursts across adjacent slot numbers), so most calls had their own speech
  discarded as "off-slot" and recorded only the handful of frames decoded before
  the slot grid anchored. The voice chain now demultiplexes concurrent calls by
  the **AACH downlink usage marker** — the per-slot call identifier the AACH
  broadcasts in every downlink slot, matched against the usage marker carried in
  the call's grant — which cleanly separates simultaneous calls on one carrier.
  A grant addressed without a usage marker, or a burst whose AACH does not decode,
  falls back to CRC-gated single-call decoding so a call's speech is never dropped
  on a guess. Verified against a real 5-capture same-carrier IQ set: the two
  concurrent calls that the timeslot filter starved now recover their full audio.
- **TETRA TCH/S class-2 CRC was computed wrong, so no on-air voice ever
  decoded.** The 8-bit CRC was a `G(X)=1+X³+X⁷` LFSR; the TETRA CRC
  (EN 300 395-2 §5.5.1) is a fixed parity-check matrix, so every received TCH/S
  burst failed the check and was dropped (recordings held only spurious frames).
  Reimplemented from the ETSI reference tap tables — real voice now decodes.
- **TETRA voice calls could hang forever, monopolising the same-carrier tap.**
  Call liveness is now driven by CRC-valid decoded speech rather than every raw
  carrier burst, so a call ends on hangtime when its transmission stops and the
  tap is freed for the next grant.
- **Config guidance no longer conflates LSM/simulcast with CQPSK.** The
  `p25_phase1_demod_mode` docs, config-builder help, `config.example.yaml`, and the
  per-site override example (issue #942) all implied that a *simulcast* site needs the
  `cqpsk`/`lsm` demod path — using Victoria's MMR / Melbourne CBD as the worked
  "→ cqpsk" example. That is backwards: simulcast is a transmitter-coordination
  technique, not a modulation, and most simulcast systems (MMR included, every site)
  transmit C4FM — forcing CQPSK there kills the decode. Linear Simulcast Modulation
  (LSM/CQPSK) is a *choice* some systems make, not implied by simulcast and not
  readable from emission-designator/licensing data. All guidance now says: leave it at
  C4FM, and switch a channel to `cqpsk` only when a strong, clean signal won't lock in
  C4FM. The per-channel override feature itself is unchanged and still correct for
  genuinely-CQPSK systems. A follow-up completes the same correction on the surfaces the
  first pass missed — the opt-in-features reference, the web config-builder's per-channel
  demod-override help, and the system-identification learn article. (issue #935)
- **P25 Phase 2 superframes now lock under any dibit rotation.** Real-air Phase 2
  is differentially decoded H-DQPSK, so a residual carrier offset near an odd
  multiple of ±1500 Hz (a quarter of the 6000-baud symbol rate) rotates every
  recovered dibit by a constant 0..3. The superframe decoder only correlated the
  outbound frame sync under the single canonical rotation, so a rotated stream
  never locked — no superframe, no traffic-channel MAC PDU, and hence no source
  RID or talker alias. It now searches all four rotations (the three non-canonical
  ones at a stricter tolerance so noise cannot false-lock) and de-rotates the
  sliced superframe to canonical before the ISCH + MAC FEC. Verified against a
  real Victorian MMR (WACN 0xBEE00) Phase 2 voice capture: 0 superframes before,
  ~430 after. (issue #915; the traffic-channel MAC descramble mapping that keeps
  `mac_rs_valid=0` even on a locked superframe is a separate, still-open blocker.)
- **P25 Phase 2 MAC descramble moved to the coded-channel-bit domain.** The PN44
  scrambler (TIA-102.BBAC-1 §7.2.5) wraps the burst "between demodulation and
  FEC", i.e. over the coded channel bits — but GopherTrunk XORed the recovered
  144 *information* bits *after* the trellis decode. The trellis code does not
  commute with the XOR, so a genuinely scrambled burst could never satisfy the
  outer RS(24,16,9) check regardless of seed — the `mac_rs_valid=0` blocker that
  suppresses the Phase 2 source RID (and talker alias). The descramble now runs
  on the raw channel dibits before deinterleave/trellis, at each sub-frame's
  channel-bit offset into the continuous 4320-bit superframe sequence, matching
  the reference decoders (SDRtrunk `Timeslot.xor()`, OP25 `handle_packet()`).
  `ScramblerProbe` self-aligns the slot phase against the RS gate. Regression
  tests build a channel-scrambled burst from the reporter's confirmed MMR
  identity (WACN 0xBEE00 / System ID 0x164 / NAC 0x161) and recover the source
  RID through the full FEC chain; live-air confirmation of the per-slot offset is
  pending the reporter's `mac_rs_valid` re-pull. (issue #915, Finding B)
- **TETRA AFC now acquires carriers several kHz off-centre.** The carrier AFC
  estimated the offset from the 4×Δφ differential mean, which wraps into
  ±f_sym/8 ≈ ±2250 Hz — so a control channel more than 2250 Hz off-frequency
  aliased into that window and left a multi-kHz constellation spin that broke
  Gardner timing, and the receiver never locked even on a clean, strong signal.
  Real SDR front-ends (RTL-SDR, Airspy, HackRF, non-GPSDO USRP) routinely sit
  several kHz off, so this was a real acquisition gap. A coarse mean-frequency
  stage (the angle of the pre-matched-filter block's lag-1 autocorrelation)
  now picks the alias bucket before the existing fine estimator refines it,
  extending acquisition to roughly ±6 kHz. (issue #940)
- **TETRA control channels now lock on real air (§8.2.5 scrambler).** The
  scrambling LFSR shifted its register the wrong direction relative to the tap
  convention, so the generated sequence diverged from ETSI EN 300 392-2 §8.2.5
  at the second bit. Because scramble and descramble share the generator, every
  encode→decode round-trip still matched and the coding tests stayed green, but
  no real, externally scrambled burst could be decoded: on air the unscrambled
  sync training sequence correlated while the scrambled BSCH never reached a
  CRC-clean codeword, so the colour code was never learned and the control
  channel never locked — the "18 000-baud symbol timing recovers but no CC
  lock" symptom, including on Single Carrier Base Station / dynamic-MCCH-sharing
  sites. Verified against a real 467.913 MHz SCBS capture, which now locks in
  ~11 ms and decodes MCC 250 / MNC 013 / colour code 0x2C. (issue #925)
- **Eye diagram panel now follows the locked channel like the other Plots
  scopes.** The Eye diagram carried the offset/frequency tuning controls from
  the constellation work (issue #557) but, unlike the Constellation and Symbol
  scope panels, never refreshed the active-call snapshot — so its "follow the
  newest call" offset froze on a stale call the moment the panel was opened
  directly, and re-checking Hold left the view on that stale call frequency
  instead of parking it on the control channel. The panel now polls active
  calls while mounted and parks on the configured control channel on Hold,
  matching the sibling scopes.
- **Completed-call `source_rid` is no longer set from an unverified P25 Phase 2
  traffic-channel MAC PDU.** The in-call source RID is backfilled from the
  `GROUP_VOICE_CHANNEL_USER` MAC PDU decoded off the voice channel, but that
  MAC path runs with the outer RS(24,16,9) check off by default, so any 18
  descrambled bytes parse — and a mis-framed / mis-descrambled superframe
  decoding random bytes occasionally lands on opcode `0x01`/`0x21` and injects
  a plausible-but-wrong RID indistinguishable from a real one (the source-side
  analogue of the `algorithm_id` smear above; the field report on MMR saw a
  near-uniform 234-of-256 MAC opcode distribution, i.e. garbage passing
  ungated). GopherTrunk now trusts a source RID only when its MAC PDU carries a
  valid outer RS parity, so a wrong RID is never carried to the webhook or
  `/api/v1/calls/history` — it stays omitted instead. Recovering the RIDs GT
  never frames on such systems is a separate Phase 2 superframe-framing fix
  (issue #915). On P25 Phase 2 (and Phase 1's residual-error
  tail) a bit-error in a traffic-channel Encryption Sync smears the decoded
  Algorithm ID roughly uniformly across `0x00-0xFF`, with a near-distinct Key
  ID per call. Those values were published straight to the completed-call
  webhook and `/api/v1/calls/history`, where they were indistinguishable from
  a real key (a 7-day MMR sample showed the real AES-256 `0x84` alongside a
  flat ~4-22 calls at *every* other algid). The composer now validates the
  decoded Algorithm ID against the TIA-102 registry (`p25.AlgorithmKnown`)
  before publishing, so an out-of-set value is dropped and the fields stay
  omitted rather than carrying garbage. Clear and every registered algorithm
  pass through unchanged. Refs #813, #924.
- **The AACH is no longer mis-routed through the CMCE/MLE Layer-3 PDU
  parser.** The decoded AACH (a MAC-layer ACCESS-ASSIGN PDU) was being handed
  to `ParsePDU`, which is for CMCE/MLE Layer-3 PDUs, so it was mis-parsed
  (and could surface as a spurious grant/release). It is now parsed as the
  ACCESS-ASSIGN PDU it actually is.
- **A TETRA channel recorded as a narrowband slice below the 144 kHz channel
  rate (e.g. the natural 48/50 kHz SDR++/SDRTrunk record rate) now decodes
  instead of emitting a nonsense symbol rate.** The `siglab` replay/analyze
  engine only ever *decimated* to the per-protocol channel rate — a capture
  already at or below the target was fed to the receiver raw. For TETRA (144 kHz
  target) that left a 50 kHz recording at 50000/18000 ≈ 2.78 samples/symbol,
  which the receiver rounded to 3 and handed the Gardner loop as a 3.0 nominal:
  an ~8% clock error it can't pull in, so it produced 50000/3 ≈ 16667 sym/s
  (−7.4%) and never locked. The down-converter now *normalises* to the channel
  rate in both directions — a sub-target capture is interpolated **up** to
  144 kHz (`dsp.Resampler` already supports L>M), restoring the receiver's
  designed 8 samples/symbol. A 50 kHz capture now recovers 18004.9 sym/s (+0.0%)
  and the training sequences correlate; the committed 48 kHz `samples/tetra`
  reference likewise normalises to exactly 144000 Hz. Only sub-target replays
  change; every wideband SDR rate reduces exactly as before.
- **The narrowband down-converter no longer shifts the channel rate on SDR
  sample rates that don't divide cleanly into the channel target.** When a
  capture rate reduced to a ratio past the resampler's L/M caps, `ccdecoder`'s
  DDC fell back to a crude integer decimator that missed the target rate by up
  to a few percent — e.g. a 3.019 MS/s stream landed the 144 kHz TETRA channel
  at 143762 Hz, i.e. 17970 sym/s instead of 18000 (−0.165%), a decode-breaking
  symbol-clock error. It now falls back to the closest L/M under the caps (the
  bounded search already used in the wideband `internal/dsp/tuner` path for
  issue #550), landing the same stream at 143998 Hz (17999.8 sym/s). Standard
  SDR rates (2.4/2.5/10 MS/s) reduce cleanly and are unaffected.
- **Signal-Lab "Capture from tuner" hung on 30 s grabs and hid the recorded
  file.** A capture of N seconds spends ≥N seconds collecting IQ in real time
  before the handler writes anything, so a 30 s grab ran past the API server's
  30 s write timeout and the response was torn down mid-write — the console sat
  on "Capturing…" forever (10 s worked). The capture handler now disables the
  per-request write deadline (as the SSE and audio-stream handlers already do),
  and the duration ceiling rose from 30 s to 120 s (bounded by a staged-file-size
  budget). The captures list also gained a persistent per-row **Download** link
  (previously the only link was transient state on the capture form, so it
  appeared to show up only after a second capture), and a long capture name no
  longer overflows the card and pushes the compare checkbox out of reach.
- **Completed-call webhook now carries the source RID on nearly every call,
  not just the ~18% whose voice-side `call.source` decoded.** The per-call
  `broadcast.webhook` sink read the source RID off the grant that *bound* the
  call, but a call frequently binds from a source-less grant (a P25 Phase 2
  compressed grant, or a `GRP_VCH_UPDATE` repeat) while the initiating
  `GRP_VCH_GRANT`'s RID arrives on a later repeat. That RID reached the engine
  but was never associated back to the running call. The engine now folds a
  later same-call grant's RID onto the bound call and republishes it through the
  existing source-update path, so the completed-call webhook (and the live
  SSE/TUI view) report the source. An in-call `call.source` update — which
  reflects the radio actually keyed on the traffic channel — still takes
  precedence, and the republish fires only on the first fill so the repeated
  control-channel grant stream never floods the bus (issue #915).
  Follow-up: on a heavily-compressed Phase 2 system a field test found that
  talkgroup-keyed matching alone reached only ~12% coverage, because the
  RID-bearing grant often arrives under a *different* talkgroup label than the
  source-less grant that bound the call (a mis-aliased compressed grant, or a
  super-group/patch remap). The engine now also recovers the source by physical
  channel: a frequency + timeslot hosts exactly one in-progress transmission, so
  a source-carrying grant landing on an active call's exact channel is folded
  onto that call regardless of its talkgroup label — which additionally
  suppresses the phantom duplicate call the mismatched talkgroup would otherwise
  spawn (issue #915).

### Changed
- **Short digital recordings now report their frame yield.** The recorder's
  `recording shorter than call span` diagnostic gained `frames`, `audio_pct`
  (the fraction of the call span that decoded to audio), and `vocoder`, so a
  sparse-decode recording (few TCH/S bursts passing CRC — the TETRA
  short-recording symptom) is diagnosable from one log line instead of by
  diffing the WAV against the call record.
- **Same-carrier voice-tap IQ drops are no longer silent.** When the control
  decoder's voice tap drops IQ to a lagging voice consumer (still dropping to
  protect the decode hot path), the dropped chunks are now counted and surfaced
  as a single warning at call end with the remedy — so starved voice decode
  (the short/gappy-recording symptom) is visible instead of silent (issue #402).
- **Live captures stream straight to disk instead of buffering the whole grab in
  RAM.** The Signal-Lab capture path (and the `gophertrunk capture` / daemon
  `--iq-capture` subcommands) previously held the entire capture in memory as
  `complex64` and then allocated a second encoded copy, so peak memory scaled
  with `seconds × sample-rate` (~4.8 GB for 30 s at 10 MS/s). They now encode and
  write one chunk at a time through a shared streaming writer — narrowband slices
  too, via a stateful down-converter — so peak memory is a single chunk
  regardless of capture length, and maximum duration is bounded by disk, not RAM.

## [v0.7.1] — 2026-07-16

### Fixed
- **P25 Phase 1 recordings opened with a full-scale "startup scratch".** While
  the receiver is still acquiring symbol lock at the start of a transmission, the
  FEC layer resolves the marginal dibit stream to random-but-valid IMBE frames
  that the vocoder synthesised as a loud burst — where a reference decoder
  (TrunkRecorder) is silent. Measured across field captures, every call blasted
  the soft-limiter rail (~24000–26000) for the first ~0.15–0.55 s. The IMBE
  decoder now applies a **call-startup acquisition squelch**: it mutes output
  until a sustained run of stable-pitch voiced frames confirms real speech
  (acquisition garbage is idle / unvoiced / pitch-jumping), then releases for the
  rest of the segment. Muted frames are zeroed, not dropped, so the recording
  keeps its length. Enabled on the recorder/live path only (the raw decoder is
  unchanged). This is a heuristic — the true acquisition state lives in the
  receiver, not the vocoder frames — so a call that never presents a stable
  voiced run (e.g. unvoiced-only) opens muted up to a failsafe window.

### Security
- **Go toolchain bumped to 1.25.12** to close `GO-2026-5856` — an Encrypted
  Client Hello privacy leak in the standard library's `crypto/tls`, which
  govulncheck flagged as call-reachable through the API TLS/gRPC servers and
  the rtl_tcp / import-client TLS paths. No source changes; the pin moves in
  `go.mod` and every CI/build workflow.

### Added
- **Per-call demod quality (EVM / SNR) on P25 Phase 1 calls.** Each decoded P25
  Phase 1 voice call now carries a measured RMS error-vector magnitude (`evm_pct`)
  and estimated symbol SNR (`snr_db`), surfaced in the call-history API and
  `call_log` alongside `signal_dbfs` — the demod-quality numbers to compare
  against another decoder (SDRTrunk). Measured over the settled decode from the
  receiver's soft/symbol taps and stamped onto `CallEnd`; nil for calls with no
  measurement (short blips, non-composer ends) and for the Phase 2 / DMR chains,
  which don't yet expose the taps (#878 follow-up). The gRPC `RIDCallRow` surface
  is unchanged (a separate proto change, as `signal_dbfs` also awaits).

### Fixed
- **Extended the wideband voice-tap channel-select fix to P25 Phase 2 and DMR.**
  The same defect fixed for P25 Phase 1 — a wideband DDC voice tap feeding the
  receiver a ±24 kHz-wide stream with no channel filter (the front end was a
  pass-through no-op at the tap rate) — was present in the Phase 2 and DMR voice
  chains too. On DMR (4FSK/FM discriminator) a stronger adjacent channel could
  be captured and, because DMR talkgroup gating is disabled, recorded *as* the
  call. On P25 Phase 2 (linear H-DQPSK) adjacent energy instead pumped the AGC
  and degraded carrier recovery, raising EVM. Both chains now channel-select each
  wideband tap to ±6.25 kHz (half the 12.5 kHz channel spacing) before the
  receiver, matching Phase 1; the dedicated-tuner path is unchanged.
- **Wideband P25 Phase 1 voice recordings came out short and garbled next to
  SDRTrunk** — on a busy multi-channel site a wideband DDC voice tap decoded
  clean *foreign* talkgroups mid-call (e.g. a 16 s over recorded as ~9 s of the
  wanted call spliced with dropped windows). The tap delivered a stream
  band-limited only to its ±24 kHz output Nyquist, and the voice chain's
  front end was a pass-through no-op at that rate — so the whole span, including
  the ±12.5 kHz adjacent channels, reached the receiver's FM discriminator. The
  capture effect then locked onto a stronger neighbour during the wanted talker's
  syllable gaps: those LDUs decoded as another talkgroup and were gated out
  (short recording), while the neighbour's energy raised in-band interference on
  the frames that *were* kept (garbled audio). The chain now channel-selects each
  wideband tap to ±6.25 kHz — half the P25 channel spacing — before the
  discriminator, dropping an adjacent channel ~80 dB while passing the wanted
  C4FM flat. The dedicated-tuner path (which already band-limits as it decimates)
  is unchanged. For diagnosing any residual truncation, the `composer: p25p1
  decode quality` log gains a `gated_ldus` count (delivered LDUs the talkgroup
  gate dropped from the WAV), and the recorder now logs when a decoded call's
  audio runs materially shorter than its wall-clock span.
- **Webhook payloads reported every call as `encrypted: false` and gave unit
  calls no `call_type`.** On systems whose encryption only resolves on the
  traffic channel (P25 Phase 2 compressed grants, Phase 1 LDU2 Encryption Sync),
  the recorder built its `CallComplete` from the grant-time session snapshot,
  which never saw the mid-call update — so the webhook said `false` (and never
  emitted `algorithm_id`/`key_id`) even though `/api/v1/calls/history`, fed from
  the engine-backfilled `CallEnd` grant, correctly showed the call encrypted. The
  recorder now mirrors the mid-call encryption / source facts onto its session
  grant, so every broadcast backend agrees with the call log. A new `call_type`
  field (`group` / `unit` / `data`, always emitted) lets a consumer tell a unit
  call — whose `talkgroup` carries a destination RID, not a talkgroup — apart from
  a group call, and unit-call `source` now populates when the RID resolves in-call
  (issue #897).
- **SoapyRemote stream arguments smuggled through `sdr.soapy_remote[].args`
  were silently ignored.** Everything in `args` is passed to the remote
  SoapySDR `make()` call, but GopherTrunk builds the `SETUP_STREAM` frame
  itself, so `remote:mtu` / `remote:window` / `remote:prot` placed in `args`
  never reached the stream setup — a user who set `remote:mtu=8000` in `args`
  stayed on the 1500-byte default with no warning (issue #876). Config load now
  rejects those reserved stream keys in `args` and points at the dedicated
  `stream_mtu` / `stream_window` / `stream_protocol` keys that actually take
  effect. `docs/hardware.md` gains a field-tested networked-USRP-X310 example,
  a B210 `args` snippet, the `SOAPY_SDR_LOG_LEVEL=DEBUG SoapySDRServer` debug
  invocation, and guidance to prefer a manually chosen gain over AGC for survey
  work.
- **Airspy on macOS aborted mid-stream with `usb: ReadPipe: 0xe00002eb`.** After
  a second or two of healthy decoding the bulk-IN pipe was aborted by macOS
  (`kIOReturnAborted`), the daemon retried, hit the same wall, and escalated to a
  fatal. It was **not** bandwidth — it reproduced at 2.5 MS/s (~10 MB/s) exactly
  as at 10 MS/s. The cause was the transport model: the darwin backend issued 32
  *concurrent synchronous* `ReadPipe` calls on one pipe, which is not a supported
  IOUSBLib usage and macOS aborts under sustained streaming. The macOS bulk-IN
  path now defaults to **asynchronous** transfers (`ReadPipeAsync` serviced by one
  CFRunLoop thread, re-arming each transfer on completion) — the model libusb (and
  SDR#/SDRtrunk) use to stream the same hardware cleanly. `GT_USB_SYNC_BULK=1`
  restores the legacy synchronous reapers. Additionally, the Airspy driver now
  runs its real→IQ conversion on a dedicated goroutine instead of inline on the
  USB reaper, keeping the transfer servicing responsive. The daemon's
  retries-exhausted fatal also no longer double-prefixes the subsystem name
  (`widebandt2: widebandt2: …`).
- **A flapping Airspy no longer kills the whole daemon.** After the silent-freeze
  fix, a macOS Airspy that keeps *recovering* — the IQ stream dies, the daemon
  reacquires and streams for a few seconds, then it dies again — still took the
  process down: the self-heal retry counter only reset after a single 60 s
  uninterrupted run, so those repeated ~seconds-long recoveries accumulated and
  the daemon escalated to a fatal shutdown after a handful of deaths (~30 s in).
  The wideband and control-channel retry loops now **decay** the counter after a
  run that streamed real data, so a recovering dongle self-heals indefinitely;
  only a device that re-dies immediately on every reopen (truly gone) escalates.
- **The exact USB stream-death cause is now surfaced.** The error that killed
  the bulk-IN reaper (the stall watchdog's `usb: bulk-IN stream stalled`, a
  `usb: device disconnected`, or a wrapped per-URB error) was discarded, so the
  daemon only ever logged a generic “IQ stream closed unexpectedly.” The airspy
  and RTL-SDR drivers now record it and the `IQ stream died; retrying` log line
  names the concrete cause — the diagnostic needed to tell a stall from a
  disconnect from an overrun without guessing.
- **Airspy on macOS no longer freezes silently.** After locking a control
  channel and decoding for a few seconds, an Airspy on macOS could go
  completely silent — the process stayed alive (heartbeats kept logging) but no
  IQ arrived and nothing decoded, with no error, EOF, or drop warning. The
  cause is a device-side USB endpoint halt that the darwin backend's blocking,
  no-timeout bulk read never noticed, so none of the existing safety nets (the
  overrun drop, the reaper-death `onStreamDead`, the enumerate-based pool
  watchdog) fired. A new **bulk-IN stall watchdog** now aborts a stream that
  delivers no data for `GT_USB_BULK_STALL_MS` (default 2 s) while the device is
  still present, converting the silent hang into a real end-of-stream; the
  wideband decoder is now supervised by a retry loop that reacquires the dongle
  and restarts (mirroring the control-channel decoder), so the daemon
  self-heals instead of quietly stopping.

### Added
- **`dc_avoid` self-suggesting diagnostic on a poorly-decoding zero-IF control
  channel.** When a P25 control channel is locked but running at zero-IF (no
  DC-spike-avoidance LO offset) and its TSBK blocks are failing Viterbi/CRC at a
  high rate, the daemon now emits one advisory WARN pointing the operator at
  `dc_avoid: true` — the exact remedy for the on-DC front-end I/Q image / 1-f
  degradation. Advisory only (it never changes tuning), silent once `dc_avoid` is
  in effect, and fires at most once per lock session (issue #402).
- **GopherTrunk Bundle (`.gtb.tar.gz`) — the capture-to-analysis case format.**
  A single portable archive that packages one SDR case — the raw IQ capture, an
  auto-carved narrowband DDC slice, logs, the SigLab signal analysis, the
  CryptoLab crypto analysis, and the hunt site/network mapping — under one
  human-and-machine-readable `MANIFEST.yaml` with per-file SHA-256, so a whole
  investigation is shared as one file and flows capture → SigLab → CryptoLab →
  back into `config.yaml`. New `gophertrunk bundle` subcommand
  (`pack`/`info`/`verify`/`extract`/`add`/`commit`), and `-bundle` on `capture`,
  `hunt` (mapping and `-survey-capture`), `analyze` (reads the capture and writes
  the result back), and `cryptolab assess` (tagged build). `commit` marks
  talkgroups encrypted from the bundle's crypto frames (algorithm named via P25
  ALGID) before merging into `config.yaml`. An opt-in `-sigmf` sidecar makes the
  capture interoperable with SigMF tooling. The daemon serves
  `/api/v1/bundle/{info,verify,download}` (confined to `recordings.dir`), and the
  SigLab, CryptoLab, and main web consoles gain a **Bundles** view to inspect,
  verify, and download a case. See [`docs/bundle.md`](docs/bundle.md).
- **Per-call received signal level in the call log.** Each recorded voice call
  now carries a `signal_dbfs` figure — the mean received channel power in dBFS,
  measured by the voice composer over the call's baseband IQ. It surfaces in the
  persisted `call_log` table and in `GET /api/v1/calls/history`. It is a
  channel-power / RSSI-style reading (0 = digital full scale; real signals
  negative), **not** calibrated absolute RSSI and **not** SNR/EVM. Calls ended
  without a measurement (watchdog timeout, preemption, shutdown) read `null`.
  Existing databases gain the column automatically on next open.
- **USRP B210 (UHD) local-setup recipe.** `docs/hardware.md` now documents
  running `SoapySDRServer` on the same host and pointing the `soapyremote`
  driver at `127.0.0.1`, with the B210 master-clock / exact-rate,
  gain-in-tenths-of-dB, and antenna-via-`args` specifics.
- **More USB debugging for the Airspy freeze.** With `RTLSDR_DEBUG_USB=1` the
  macOS backend now emits periodic bulk-stream telemetry (URBs, bytes,
  throughput, per-slot completion spread, idle gap) and a one-shot `bulk-IN
  stalled` line when the watchdog trips — enough to pin when and how a freeze
  happens. An opt-in `GT_USB_READPIPE_TIMEOUT_MS` switches the reaper to IOKit
  `ReadPipeTO` with a per-read no-data timeout as a candidate root fix. See
  `docs/reference/airspy.md` → Troubleshooting.
- **Airspy vendor control transfers are now traced under `RTLSDR_DEBUG_USB=1`.**
  The Airspy shares the RTL-SDR USB transport but never wrapped it for debug, so
  its control setup (`SET_SAMPLERATE` / `SET_FREQ` / `RECEIVER_MODE` / gain) was
  invisible even with USB debugging on; it is now wrapped like the RTL-SDR path.

## [v0.6.4] — 2026-07-08

### Fixed
- **DMR (and single-dongle P25) voice now decodes with no extra config.** A
  `role: wideband` dongle produced talkgroups/RIDs but no audio and no
  recordings, because voice grants were dropped for lack of a voice device: the
  voice pool is fed only by physical `role: voice` SDRs or virtual `voice_taps`,
  and `voice_taps` defaulted to 0. When trunking is configured and no physical
  `role: voice` SDR is present, the daemon now auto-enables a couple of virtual
  voice taps on each wideband dongle so voice decodes out of the box; set
  `voice_taps` explicitly to control concurrency. The shipped DMR samples now set
  `voice_taps`, the "no voice source" startup warning names the fix, and the
  config-builder's stale "voice taps (0–8) / capped at 8" hint is corrected.

## [v0.6.3] — 2026-07-07

### Added
- **Neighbour sites on `GET /api/v1/sites`.** Each `SiteDTO` now carries a
  `neighbors` array — the adjacent sites the site advertised over the air (P25
  Adjacent Site Status Broadcast, opcode `0x3C`), each with the neighbour's
  RFSS/Site and its band-plan-resolved control-channel `downlink_hz` (and
  `uplink_hz`). This is the same decoded data already surfaced on
  `GET /api/v1/systems`, now attached to the camped site of each system on the
  per-site endpoint, so a consumer can backfill the wider network's
  control-channel map from a single camped site without directly decoding every
  site. Empty on sites that did not broadcast a neighbour list (issue #864).

### Fixed
- **Live browser audio no longer requires a recordings directory.** The voice
  composer and its decoded-PCM tap were gated on the file recorder, which needed
  `recordings.dir`; a listen-only setup got silence. The recorder now supports a
  decode-only mode (decodes and streams live audio without writing files).
- **Live audio was silent in the browser on Windows.** The Web Audio player
  pinned its `AudioContext` to 8 kHz, which Windows/WASAPI often accepts yet
  renders as silence in every browser. It now runs at the device-native rate and
  upconverts the 8 kHz stream with the existing band-limited resampler.
- **RadioReference *sites* CSVs now fail with an actionable error, and SOAP
  faults surface their fault string.** The importer only recognises RadioReference's
  native *talkgroups* CSV and the multi-section bundle, so a native *sites* table
  export fell through to the bundle parser and failed with the opaque "data at
  line 1 before any # Section marker". A sites-shaped CSV (no `# Section` markers,
  a header with a frequency column plus a site/RFSS/NAC column) is now detected and
  rejected with a message pointing at the formats that work today (system PDF,
  Talkgroups CSV, or a bundle). Separately, when RadioReference answers a browse
  call with an HTTP 500 SOAP fault, the client now surfaces the concise
  `<faultstring>` instead of dumping the raw XML envelope (issue #849).

## [v0.6.2] — 2026-07-04

### Fixed
- **P25 Phase 2 superframes now lock on real off-air traffic.** The outbound sync
  constant was a garbled 48-bit value (`0x575F7DFF77FF`) used where a 40-bit
  (20-dibit) sync word is expected, so the top byte was silently dropped and the
  detector matched a pattern that never appears on air — `superframes=0` on every
  call, so `algorithm_id`/`key_id` never populated. Differentially decoding the
  spec-conformant outbound sync (`0x575D57F7FF`, TIA-102.BBAC) yields the 20-dibit
  stream the detector must match; setting it (plus a width guard) lets superframes
  lock. Verified against a spec-conformant synthetic; issue #813 stays open pending
  the reporter's real capture (issue #813).
- **`control_channel_carrier_offset_hz` now reports the *total* control-channel
  offset, matching the WARN log.** The field on `GET /api/v1/sites` was wired to
  the receiver's residual AFC only, so with `sdr.autotune` on — once a correction
  folds into the DDC and the receiver re-centres — it under-reported the offset the
  "carrier offset" WARN was still flagging: the log and the API disagreed. The
  published value now adds the applied DDC correction to the residual, so the two
  agree by construction. With autotune off (the default) the applied correction is
  0 and the value is unchanged (issue #815).

## [v0.6.1] — 2026-07-03

### Docs
- **Lab Bench tutorial blog series.** Three concurrent hands-on series — **Signal
  Lab**, **RF Scope**, and **Crypto Lab** — launch on the docs blog, built on a
  shared no-CDN, theme-aware, high-DPI canvas charting module (constellation, eye,
  heatmap, line, bars, timeline) that renders inline figures from post JSON, with
  per-series hub pages, trilogy cross-links, and Blog-group navigation.

## [v0.6.0] — 2026-07-02

### Added
- **Every browser console is reachable from the daemon, and they share one design
  system.** The daemon previously mounted only the operator console (`/`), Config
  Builder (`/config/`), and Crypto Lab (`/cryptolab/`); Signal Lab and RF Scope
  were reachable only via their standalone `serve` commands, absent from the nav,
  and Crypto Lab's header linked to a `/siglab/` path the daemon 404'd. Signal Lab
  now mounts at `/siglab/` and RF Scope at `/rfscope/`, both advertised via
  `RuntimeDTO` so the nav only links to what is actually reachable, and every
  sub-app gains a shared nav back to the operator console and its siblings. RF
  Scope also joins the shared `--gt-*` design tokens (dark/mono/light themes +
  reduced-motion), so all consoles look and behave alike.
- **Per-site P25 decode quality on `GET /api/v1/sites`.** Each `SiteDTO` now
  carries `control_channel_tsbk_error_rate` (cumulative % of TSBK blocks failing
  Viterbi/CRC on that site's control channel), `control_channel_tsbk_count` (the
  attempts behind it, a confidence weight), and `control_channel_decode_quality`
  (`clean` / `marginal` / `poor`). This exposes decode health independently of
  carrier lock — a well-locked carrier at range can still error heavily — so
  consumers can categorise sites as clean vs marginal rather than approximating
  from carrier offset alone. Fields are omitted until a TSBK is decoded and on
  non-TSBK Phase 2 paths (issue #858).

### Fixed
- **Airspy overrun no longer silently freezes the whole decode.** Under high load
  the wideband DSP consumer could stall, the driver's IQ channel filled, and the
  blocking send stopped the USB reaper from posting new reads — the endpoint FIFO
  overflowed and the stream halted with no error or log, which also froze the
  control-channel decode (field-reported on an Airspy R2 at 2.5 and 10 MS/s when a
  voice grant spun up a second IQ consumer). The Airspy deliver path is now
  non-blocking and drops on overrun — matching the RTL-SDR and SoapyRemote paths —
  counting the drop via `iq_underruns_total` and emitting a rate-limited "host
  can't keep up — lower `sdr.sample_rate`" WARN, so the reaper always cycles back
  and the device never underflows.
- **Strong in-channel signal with no sync now warns and points at `sdr.ppm`.** A
  narrowband C4FM carrier that is present but off-frequency (an uncorrected RTL-SDR
  tuner offset beyond the demod's ~±75 Hz tolerance) decoded nothing with no
  indication why. The wideband engine now emits a per-channel "strong signal but no
  sync" WARN when a channel holds strong power with zero sync/FEC across several
  diagnostics windows, naming `sdr.ppm` as the likely fix; the dmr-simplex sample
  config's ppm note is strengthened accordingly (issue #836).
- **RadioReference CSVs exported through Excel on Windows now import.** Such files
  carry a UTF-8 BOM (`EF BB BF`); the raw bytes reached the format sniffer, mangling
  the first header cell so the native-CSV detector failed and import errored with
  "native RR CSV header missing decimal/dec column" or "data at line 1 before any
  # Section marker". A leading BOM is now stripped at the file-read site (covering
  both the CLI import and the web-UI upload) and, defensively, in the reader-based
  entry points (issue #849).

## [v0.5.9] — 2026-07-01

### Added
- **Full-spectrum survey — scan the whole device, name everything, capture any
  signal.** `gophertrunk hunt -survey` gains an end-to-end "what's on the air"
  workflow for undocumented, protocol-dense areas:
  - **`-whole-device`** sweeps the SDR's *entire* tuning range (derived from the
    device itself — 24 MHz–1.766 GHz for an R820T, etc.) instead of a hand-typed
    `-band`, printing the resolved span, step count, and ETA. A new optional
    `sdr.FreqRanger` reports each backend's range; every tuner single-sources its
    PLL bounds so the advertised and enforced ranges can't drift.
  - **Wideband detect + name + stitch.** `-detect-wideband` (auto-on with
    `-whole-device`) finds signals far wider than a channel — cellular LTE/5G,
    WiFi, other OFDM — by detecting occupancy plateaus and *stitching them across
    tunes* to recover the true occupied bandwidth of a signal wider than the
    SDR's instantaneous bandwidth. These are surfaced as **named-but-not-decoded**
    rows: GopherTrunk identifies and can capture them, it does not demodulate them.
  - **Unified inventory row.** Every detected signal now carries a **name**,
    **frequency**, **service/purpose** (from a curated frequency-allocation table
    — FRS/GMRS, amateur, public-safety, ADS-B, ISM, the cellular bands, …), signal
    **quality**, **encrypted?** and **encryption type**, and a **wideband** flag.
    Encryption surfaces from the decode (ALGID → AES-256, ADP/RC4, …); a new
    `-detect-encryption` entropy triage flags *unknown* digital bursts that look
    encrypted. The survey CSV gains the matching columns.
  - **List-driven capture → SigLab / CryptoLab.** From a survey's JSON,
    `gophertrunk hunt -survey-capture <freq|#index> -from survey.json -to
    siglab|cryptolab` records raw IQ of the chosen signal (frequency + occupied
    bandwidth drive the capture) and hands it on — SigLab identify, or an rfscope
    pass that emits a CryptoLab `ks` frames file. The same action is a per-row
    button in the web **Hunt** tab, an `↑/↓` + `c`/`C` keybinding in the TUI Hunt
    panel, and `POST /api/v1/hunt/capture` on the daemon.

### Fixed
- **Airspy R2/Mini reported half its true IQ rate, so wideband decoded
  nothing.** The driver treated the firmware's `GET_SAMPLERATES` table as 2×
  "device" rates and reported half the rate the hardware actually delivers, so
  `ActualSampleRate()` returned e.g. 5 MS/s for an R2 streaming 10 MS/s. The
  daemon then built its wideband down-converter at half rate (logging the
  spurious "SDR streams a different sample rate than requested" warning, issue
  #402), the symbol clock ran 2× off, and no packets decoded. The table is in
  IQ output rates (R2 `{10, 2.5}` MSPS, Mini `{6, 3}` MSPS); the driver now
  matches the requested IQ rate against it directly. The IQ converter is
  unchanged — the firmware still streams the real ADC at 2× and the host
  decimates back down. The same bug also broke the **control channel on an R2
  at 2.5 MS/s** (`sample_rate: 2500000` → `actual_hz=1250000`, continuous
  `no FSW hits`); the daemon now builds the control down-converter at the true
  2.5 MS/s so the symbol clock stays aligned (#851, same root cause as #764).

## [v0.5.8] — 2026-06-30

### Added
- **RF Scope — protocol-agnostic RF network analysis ("Wireshark for the RF
  physical layer").** A new `gophertrunk rfscope` surface analyzes any band — a
  recorded IQ capture or a live SDR — with no prior knowledge of the technology,
  modulation, framing, or encryption, and produces a structured *scene*: an RF
  protocol hierarchy, per-channel I/O-graph activity, burst timing/periodicity,
  an emitter/conversation graph (frequency hoppers collapse into one emitter; it
  defers to `hunt`'s authoritative map when a control channel is present), an
  entropy/encryption triage built on the cryptolab randomness battery, and an
  expert-info anomaly list. It ships as a CLI (`rfscope analyze`/`live`/`list`
  with summary/JSON/JSONL/YAML/CSV export), a Bubbletea cockpit (`rfscope
  cockpit`, `rfscope live -tui`), and a standalone web console (`rfscope serve`,
  built with `make rfscope-web-build`). Unidentified digital payloads can be
  emitted as a cryptolab `ks` frames file (`-frames-out`) that feeds `cryptolab
  classify auto` / `ks reuse` directly.
- **cryptolab external known-key decrypt (no brute required).** When an operator
  has already recovered an unbundled cipher's key (e.g. ran their own TEA1
  brute), `assess crypto` now verifies it through the external cipher and
  decrypts + grades the capture by passing the key via `-keys` with `-extern-cmd`
  / `-extern-algid` and a known-plaintext oracle — the weak-key method reports
  the verified break, so `-brute-bits` is only needed when the key is still
  unknown.
- **cryptolab recipe split-band & rolling descramblers.** The `recipe` pipeline
  gains `descramble-splitband` (`split` fraction of Nyquist) and
  `descramble-rolling` (`frame` samples, `schedule` of split fractions or
  `auto`) ops, bringing the inline analog-voice descrambler steps to parity with
  the standalone `descramble` tool's three modes. The web Recipe Builder renders
  the new float `split` parameter.
- **Crypto Lab is discoverable from the other web consoles.** In a
  `-tags cryptolab` daemon build the main GopherTrunk console shows a *Crypto
  Lab* nav entry and the Signal Lab header shows a 🔐 Crypto Lab link; the
  Crypto Lab console links back to both. All gated on a new
  `runtime.cryptolab_console` flag (`GET /api/v1/runtime`), so the links appear
  only when the console is actually mounted — default builds and standalone
  siglab never show a dead link.
- **cryptolab external-cipher bridge (TEA1 et al.).** `assess crypto` and the
  `recipe` pipeline can now drive an operator-supplied cipher program as a
  subprocess (`engine/extcipher`), so unbundled ciphers — most importantly the
  TETRA TEA1 32-bit backdoor brute — are fully runnable by pointing the harness
  at a vetted tool, without GopherTrunk shipping a cipher it can't verify or
  whose only reference is licence-incompatible. `assess` gains `-extern-cmd` /
  `-extern-algid` (brute the backdoor natively in the plugin, then verify +
  decrypt + grade); the recipe pipeline gains an `extern-decrypt` op. Both are
  CLI-only — the web recipe endpoint refuses host-execution ops (HTTP 403).
- **cryptolab web Recipe Builder.** The cryptolab console gains a *Recipe
  Builder* tab that drives the `recipe` pipeline interactively (config-builder
  style): pick an input, assemble an ordered operation list from a palette with
  per-step parameters and move/duplicate/remove, run it server-side, and read
  the per-step report plus the final bytes (with download). Backed by new
  `GET /api/v1/cryptolab/recipe/ops` and `POST /api/v1/cryptolab/recipe`
  endpoints.
- **cryptolab DMR decryption + generic RC4.** `assess crypto -protocol dmr`
  now actively attempts DMR privacy with the real cipher cores (RC4 "Enhanced
  Privacy", DES/3DES/AES-OFB) keyed with operator-supplied material, and the
  recipe pipeline gains a generic `rc4-decrypt` op. DMR's vendor key/IV
  derivation is the analyst's to supply; the cipher cores are bundled.
- **cryptolab security-test suite** (`-tags cryptolab`). New analysis and
  active-attack tools, ported from the industry crypto-tool landscape and
  optimized for RF:
  - `assess crypto` — a security-test harness that attempts to decrypt captured
    encrypted frames by every applicable method (cipher-strength, a
    cross-protocol P25/TETRA/DMR known-weakness advisory, IV reuse,
    known-plaintext, default/weak keys against the real ADP/DES/3DES/AES
    ciphers, reduced-keyspace brute force, keystream-LFSR prediction) and
    grades each, with an overall `RESISTANT` / `PARTIAL` / `BROKEN` verdict. A
    verified complete decryption means the deployment failed the test. The
    advisory surfaces published breaks even for unbundled ciphers (e.g. the
    TETRA TEA1 32-bit backdoor, CVE-2022-24402).
  - `randomness battery`/`quick` — NIST SP 800-22 statistical randomness tests
    on a keystream/payload bitstream (strong vs. structured).
  - `classify auto` — triage an unknown payload's obfuscation class and
    recommend the next tool.
  - `ks reuse`/`mtp`/`extract` — keystream-reuse / many-time-pad recovery
    (P25 OFB/ADP keystream reuse via a repeated Message Indicator).
  - `stats period` — autocorrelation period detection and repeated-n-gram
    histogram.
  - `recipe run` — a CyberChef-style operation pipeline: chain byte transforms
    (XOR, the real ADP/DES/3DES/AES cipher decrypts, bit reversal, spectral
    inversion, hex/base64) and analyses (entropy, randomness) from a JSON/YAML
    recipe, piping the bytes between steps.
- **Live decoder → cryptolab crypto-frame bridge.** Setting
  `recordings.crypto_capture_path` makes the P25 Phase 1 voice composer append
  each encrypted superframe's Message Indicator + encrypted voice frames as
  JSONL for `cryptolab assess`/`ks`. Off by default (no effect on the voice
  path when unset).

## [v0.5.7] — 2026-06-29

Headlined by a **P25 Phase 2 voice-decode fix**: the H-DQPSK receiver now
recovers the carrier frequency, so the Phase 2 voice follower locks
superframes on ordinary non-TCXO tuners (e.g. RTL-SDR) instead of only on a
TCXO-clocked Airspy. Rounded out by live-audio quality fixes — decoded voice
fades cleanly at call end, the WebUI plays one call at a time instead of
mixing overlapping talkgroups, and recording filenames render in the
configured display timezone — plus a spectrum-panel follow-state fix.

### Added
- **P25 Phase 2 per-call MAC census** (#813). An end-of-call census logs once
  per voice chain — superframes locked, a per-`SlotType` subframe histogram,
  MAC subframes seen, and MAC PDUs decoded — so a silent decode pins exactly
  which pipeline stage (superframe sync / ISCH classification / MAC FEC)
  produced nothing on real air. Telemetry only; no protocol/FEC/geometry
  logic changes.
- **"Solution Postmortem" blog category** on the docs site (#808).

### Changed
- **Recording filenames render in the display timezone**, not always UTC. The
  per-call WAV basename now honours `display.timezone` — a configured non-UTC
  zone gets a filename-safe numeric offset like `+1000`, while UTC still
  formats as a literal `Z`, so existing filenames are unchanged.

### Fixed
- **P25 Phase 2 H-DQPSK carrier-frequency recovery** (#813). The Phase 2
  voice follower never locked a superframe on real traffic (a reporter saw
  `superframes=0` on 67/67 calls) even though the same RTL-SDR decodes Phase 1
  fine: the H-DQPSK receiver had no carrier-frequency recovery, so a real
  tuner's residual offset rotated every symbol out of its quadrant and the
  outbound sync never correlated. The receiver now does a one-shot coarse
  carrier seed → NCO de-rotation → AGC → Gardner → Costas fine loop (on the
  `ClockGardner` path only; clean zero-offset streams stay a no-op), and the
  Costas lock phase is now rotation-aware for the π/8 H-DQPSK constellation.
  Fixes both the voice follower and the control-channel path.
- **Decoded digital voice fades to zero at call end** (P25/DMR). Short
  transmissions ended on a hard waveform discontinuity — an audible
  scratch/click — because digital voice, unlike the analog FM chain, had no
  end-of-call ramp. A call now appends a short (~10 ms) linear fade to both
  the recorded WAV and the live decoded fan-out; silent/dead-key calls are a
  no-op.
- **WebUI live audio plays one call at a time.** The browser player opened the
  audio stream unfiltered, so every concurrent voice call's PCM arrived
  interleaved into one body and talkgroups overlapped. The player now follows
  the primary (most recently started) active call and filters the stream to
  its device, re-pointing as the primary changes so audio switches cleanly
  between calls — standard scanner behaviour.
- **Spectrum panels keep their follow state live** (#810). The
  Mixer/Tuning/Histogram/Constellation/SymbolScope panels read active calls
  from the shared store but nothing refreshed it off the Active/Dashboard
  routes, so opening a panel directly froze it on a stale "following call".
  Each panel now polls active calls itself, and re-checking Hold parks the
  view on the control channel instead of a possibly-stale call frequency.

### Internal
- End-to-end regression test for the adjacent-carrier offset WARN through the
  real `ccdecoder.Decoder` (#815), guarding the live behaviour the reporter
  confirmed — a control lock ~12.5 kHz off-frequency tripping the WARN —
  which the existing stubbed tests and the `replay` path did not exercise.

## [v0.5.6] — 2026-06-28

Headlined by the **`cryptolab` cryptographic-research toolkit** — a new,
opt-in subcommand and browser console for byte-oriented cipher research — and
by broader P25 visibility in the live views: the decoded **IDEN_UP
frequency-band table** and **every active talkgroup** (not just the calls a
tuner is following) now surface in the Systems and active-call panels, Phase 2
encrypted calls populate **ALGID/KID** off the on-air MAC_PTT slot, and the
control decoder now **warns when it has locked onto an adjacent site
off-frequency** instead of silently decoding the wrong site.

### Added
- **`cryptolab` — optional RF cryptographic-research toolkit**. A new,
  opt-in `gophertrunk cryptolab` subcommand collects byte-oriented research
  tools — statistical triage (entropy / index-of-coincidence / chi-square /
  XOR key-length), classical-cipher brute force (single/repeating XOR, Caesar,
  and Vigenère with English and crib scoring), LFSR/keystream analysis (Berlekamp–Massey, keystream
  extraction), CRC parameter recovery, and analog voice spectral
  descrambling — plus a pluggable subject framework for studying byte-oriented
  obfuscators, with an initial length-seeded-obfuscator recovery suite (gauge
  sweep, structure/wiring enumeration, a monotone+resumable cell solver, and
  from-seed simulation). It is **excluded from the default build** and linked
  in only with `make build TAGS=cryptolab` (the same opt-in pattern as the
  DVSI vocoder). A browser console (`gophertrunk cryptolab serve`) mirrors the
  `siglab`/`configbuilder` consoles — a schema-driven form exposes every tool,
  mode, and setting, streams the live run log, and renders the structured
  result with downloadable artifacts. See `docs/cryptolab.md`.
- **`cryptolab` toolkit expansion**. The `brute` tool gains a **monoalphabetic
  substitution** solver (frequency-seeded hill-climbing with random restarts,
  scored by an embedded English trigram model). The `descramble` tool gains
  **split-band** inversion (independent low/high sub-band inversion about a
  configurable split) and **rolling-code** inversion (per-frame split schedule,
  with `auto` detection of each frame's inversion). The web console is now also
  **mounted inside the main daemon at `/cryptolab/`** when `gophertrunk` is
  built with `-tags cryptolab`, so it is reachable from the running daemon
  without a separate `cryptolab serve`; the default daemon build is unaffected.
- **P25 IDEN_UP frequency-band table in the live views.** The full P25
  IDEN_UP / identifier-update band table GopherTrunk already decoded (and
  rendered in the network report) now overlays onto `GET /api/v1/systems` and
  the web Systems detail modal as a "Frequency bands (N)" section beside the
  neighbour-site list. Operators can read the band plan SDRtrunk-style and map a
  neighbour's channel id/number back to an absolute frequency without doing it by
  hand. Values come from the same live topology snapshot as the report, so they
  match exactly (TDMA flag and transmit offset included).
- **All active talkgroups surfaced, not just tuner-bound calls.** The active
  panels previously showed only calls bound to a voice tuner, so a single-voice-
  tuner setup could only ever display one active talkgroup even when several
  calls were up. A control-channel activity tracker now records every voice grant
  the control channel announces; the active-calls API merges tuner-decoded and
  control-channel-observed calls with a `following` flag, and the TUI and web
  active panels render observed-only calls distinctly. Audio is still limited by
  tuner count — the display of who's *up* no longer is.
- **P25 Phase 2 ALGID/KID from the on-air MAC_PTT slot** (#813). Real Phase 2
  systems carry the encryption sync (ALGID/KID/MI) in the MAC_PTT message that
  begins each transmission, keyed by slot type rather than a MAC opcode, so the
  previous opcode-keyed accessor never matched on air and encrypted calls left
  `algorithm_id`/`key_id` blank. PTT-slot PDUs are now decoded and routed through
  the existing call-encryption backfill, and the per-channel MAC census logs the
  payload hex for field confirmation. Flagged a working model pending on-air
  validation; degrades gracefully if the layout is wrong.
- **Cryptography & cryptanalysis field-guide category and a dedicated SigLab
  user guide** (docs). New cryptanalysis-method reference pages plus a standalone
  Signal Lab user guide, with a broken link fixed.

### Fixed
- **Adjacent-site off-frequency control lock now warns** (#815). A strong
  adjacent-site P25 control carrier ~12.5 kHz away can bleed through the
  narrowband receiver, lock, and decode the *neighbouring* site's identity while
  still reporting the tuned frequency — with no indication anything is wrong.
  While locked, GopherTrunk now emits a throttled WARN when the total carrier
  offset from the configured frequency exceeds a threshold (default 4 kHz,
  `sdr.carrier_offset_warn_hz`), naming both plausible causes (adjacent-site lock
  or a mistuned oscillator). Advisory only — it never retunes — and the live
  offset is surfaced on `GET /api/v1/sites` so the mismatch is visible without
  dropping to capture/spectrum.

## [v0.5.5] — 2026-06-27

A reception-and-robustness release for wideband and DMR captures, plus the
talker-alias cryptanalysis writeup. **Tier II conventional DMR now decodes
2-slot interleaved voice by default** (#644), the **wideband voice-tap DDC is
span-aware** so grants out at the band edges actually tune, and **SoapyRemote
overruns are surfaced and shed** instead of silently shredding every channel.

### Added
- **2-slot interleaved voice by default for Tier II conventional DMR** (#644).
  A DMR repeater carrier is 2-slot TDMA, so the single-slot voice decoder
  sliced alternating timeslots into each superframe — the "DJ scratch", garbled-
  audio signature from the field report (every quality line showed
  `lc_superframes=0` with real, FEC-clean AMBE). `proto=dmr-tier2` conventional
  systems now default to the 2-slot interleaved cadence so the slot router binds
  and audio reconstructs cleanly.

### Changed
- **Span-aware voice-tap DDC so outer wideband grants tune.** A wide DMR plan on
  a 10 MS/s capture has voice grants out past ±1.9 MHz; the virtual tuner
  advertised the full IQ window and the engine bound those grants, but the per-
  call DDC was built without span and rejected any offset beyond ±1.1 MHz, so the
  outer carriers produced no audio. The per-call bank is now sized from the tap
  offset (the span-aware path #764 added for the control channel), so any grant
  inside the advertised band actually tunes.
- **SoapyRemote overruns are surfaced and drained without stalling the radio.**
  At high sample rates a blocked reader stopped draining the socket and stopped
  sending flow-control ACKs, so the device overflowed and dropped samples,
  shredding every channel's framing at once — with only an invisible per-datagram
  DEBUG line as a signal. Overruns are now reported at a rate-limited WARN with
  running counts and an actionable hint, and the read loop never blocks (it sheds
  the oldest queued chunk and counts it), keeping the socket drained and ACKs
  flowing. The channelizer per-tap loop is also parallelized across CPU cores, and
  an oversampled capture rate now draws a lower-the-rate advisory at startup.

### Docs
- **Talker-alias cipher cryptanalysis** (#773): a chosen-plaintext capture
  procedure and the documented cryptanalysis findings (product-form decode
  decomposition), linked to the issue thread.

## [v0.5.4] — 2026-06-26

A wideband-robustness and capture-guidance release. Multi-tap wideband front
ends get **`gain: "auto"` guidance** and can now **host dense plans (70-DMR)
instead of crashing at init**, `gophertrunk capture` warns on **over-high
sample rates**, talkgroups **auto-populate from live control-channel grants**,
and the unverified Motorola talker-alias cipher is **gated with corrected
provenance** (#773).

### Added
- **Capture sample-rate guidance** (#771). `gophertrunk capture` now prints a
  one-line hint when a high native rate (>4 MS/s) is chosen for a narrowband
  trunking capture, explaining that the down-converter normalises to 48 kHz and
  that the top native clock on wideband front-ends (e.g. Airspy R2 at 10 MS/s)
  can degrade in-channel SNR via phase noise / reciprocal mixing — the verified
  root cause of the #771 no-lock report. A new "Choosing a sample rate" section
  in `docs/hardware.md` documents the finding and recommends ~2.4–2.5 MS/s for
  control-channel roles.
- **Auto-discover talkgroups from control-channel grants.** The Talkgroups
  database previously loaded only from a hand-maintained CSV, so an operator
  starting empty saw bare numbers. Each talkgroup is now catalogued the first
  time it is granted on the control channel (skipping unit-to-unit / interconnect
  grants), tagged "Discovered" and surfaced in Database → Talkgroups. In-memory
  only and never silently widens a curated scan list — the operator opts a learned
  talkgroup in from the UI.

### Changed
- **`gain: "auto"` guidance for multi-tap wideband devices** (#749). A single
  fixed gain on a shared front end can't serve sites of differing strength —
  picking a gain so the strongest site doesn't clip leaves weaker co-tenants at
  the ADC noise floor. GopherTrunk now WARNs at startup when a `role: wideband`
  dongle with >1 channel is pinned to a manual gain (and the per-tap low-power
  WARN suggests `gain: "auto"` too), with the AGC recommendation documented in
  `config.example.yaml` and `docs/hardware.md`.
- **Dense wideband plans are hosted on the channelizer instead of crashing at
  init.** Very dense wideband plans (e.g. 70-DMR) now stay on the channelizer
  (a per-tap DDC is too heavy at that scale) so they initialise and run, and the
  web plots follow the voice channel on wideband SDRs.

### Fixed
- **Gated the unverified Motorola talker-alias cipher; corrected false
  provenance** (#773). RIDs resolve on live Phase 2 traffic but no talker-alias
  *text* ever decodes: the per-byte substitution cipher was AI-authored in #376
  yet its comments claimed it was a reverse-engineered fact "identical across
  every open-source decoder." A clean-room attempt showed the cipher is
  mathematically underdetermined from the one partial capture available, and
  SDRtrunk's GPLv3 table is off-limits to Apache-2.0 GopherTrunk. The misleading
  provenance is corrected to the truth (structure inferred from the public #773
  protocol work) and the unverified decode is gated rather than shipping fake
  confidence.

## [v0.5.3] — 2026-06-25

Headlined by **neighbor (adjacent) sites in the live views** — the P25 Adjacent
Site Status Broadcasts (opcode 0x3C) GopherTrunk already decoded now surface in
the always-on monitoring surfaces (decoded-message log, the TUI and web Systems
panels, and the `/api/v1/systems` responses), not just the drill-in report. It
also lands a **configurable display timezone** (`display.timezone`) that renders
every human- and machine-facing timestamp in the host's local time by default
(overridable to UTC or any IANA zone), and an **opt-in voice enhancement chain**
(`recordings.enhance`) that trades a little faithfulness for the cleaner, louder
"sound-good" audio the rival decoders produce — band-limiting + uniform loudness,
shaping both recordings and live audio from a single point. Rounded out by a
power-complementary unvoiced-synthesis window that removes a buzzy ~7 dB tremolo
from the IMBE/AMBE+2 vocoders (#644).

### Added
- **Neighbor (adjacent) sites in the live views.** GopherTrunk already decoded
  P25 Adjacent Site Status Broadcasts (opcode 0x3C) but only surfaced them in the
  system drill-in report and `gophertrunk replay`. They are now visible in the
  always-on monitoring surfaces: the decoded-message log prints an SDRtrunk-style
  "Neighbor Sites" block (deduplicated — re-emitted only when the adjacent-site
  set changes), and the TUI and web Systems panels gain a neighbour count column
  plus a neighbour list in the system detail. `GET /api/v1/systems` and
  `/api/v1/systems/{name}` now carry a `neighbors` array (RFSS/Site with
  band-plan-resolved downlink/uplink frequencies).
- **Configurable display timezone** (`display.timezone`). "Local" (default),
  "UTC", or any IANA name (e.g. "America/New_York") for the human-facing
  timestamps.
- **Opt-in voice enhancement chain** (`recordings.enhance`). A "sound-good"
  post-vocoder chain for decoded digital voice — rumble high-pass (~250 Hz),
  warmth high-shelf, telephone-band low-pass (~3.4 kHz, like OP25), a louder AGC
  target (22000 vs the faithful 18000), and an optional soft-knee compressor.
  It deliberately trades a little faithfulness for the cleaner/louder sound the
  rival decoders produce: OP25 band-limits its output, Trunk Recorder ships
  loudness normalization on by default, and DSD/DSDPlus run an aggressive output
  AGC — uniform loudness + band-limiting, not exotic EQ, is what makes them sound
  "better" (SDRtrunk, by contrast, plays raw vocoder output and has open
  requests for exactly this). Because the recorder decodes each call once and
  fans the PCM out to disk and live monitoring, the chain shapes **both**
  recordings and live audio from a single point. Off by default — the faithful
  path stays byte-identical — and surfaced near the top of the Config Builder.
  Toggling `recordings.enhance.enabled` also live-applies via
  `PATCH /api/v1/settings` (takes effect on the next call). Subsumes
  `warm_dmr_audio` (now superseded) by extending the warmth shelf to all
  protocols.
- **Opt-in "warm" DMR voice decoder** (`recordings.warm_dmr_audio`, #644). A
  gentle output high-shelf (≈2 dB cut above 1.5 kHz) that softens the
  bright/thin "digital" timbre of software AMBE+2 decode, selectable via the new
  `ambe2-dmr-warm` vocoder. It is a listener tone preference, not a
  codec-quality fix — the residual synthetic character of low-bitrate AMBE+2 is
  intrinsic to software decoders (mbelib, SDRtrunk's JMBE, and GopherTrunk all
  share it); only a DVSI hardware vocoder removes it. Off by default; DMR only.

### Changed
- **Timestamps now render in local time by default**, not UTC, across every
  human- and machine-facing surface. The decoded-message log, power log and TUI
  panels (events / dashboard / tone alerts) previously forced GMT+0 ("…Z"); the
  JSON/gRPC API (RID activity & history, active calls, location reports, the
  `/health` daemon time), webhook payloads and rdioscanner uploads did the same.
  All now render in the host's local timezone, overridable via `display.timezone`
  (including back to "UTC"). The API/webhook/rdioscanner timestamps stay RFC3339
  but now carry an explicit numeric offset (e.g. `+02:00`, or `Z` when UTC), so
  they remain an unambiguous, parseable instant.

### Fixed
- **Unvoiced-synthesis tremolo in the MBE vocoders** (#644). The §6.4 unvoiced
  overlap-add used a Hann window that is not power-complementary at the
  160-sample hop, modulating the synthesized noise floor ~7 dB at the 50 Hz
  frame rate (a buzzy artifact, most audible on noisy/fricative speech).
  Replaced it with a power-complementary tapered-cosine window so the noise
  floor is flat. Shared by the IMBE (P25) and AMBE+2 (DMR/NXDN/dPMR) paths.

## [v0.5.2] — 2026-06-24

A maintenance release headlined by a **generic JSON webhook call sink** —
POST one stable-schema JSON object per completed call to any HTTP endpoint, so
GopherTrunk feeds custom dashboards and automations alongside the existing
OpenMHz / Broadcastify Calls / RdioScanner / Icecast backends. The rest is
voice-path robustness: DMR 2-slot cadence is now picked by AMBE FEC instead of
the unvalidated embedded Link Control (fixing garbled "sounds-encrypted" audio
on 288-cadence carriers, #644), calls that never deliver a single voice frame
end at a bounded 7 s window instead of hanging on the 30 s watchdog (#788), and
siglab now names *why* a control channel did not lock with an opt-in soft-sync
fallback for marginal captures (#771).

### Added
- **Generic JSON webhook call sink** (`broadcast.webhook`, #404, #268). A new
  outbound call-streaming backend that POSTs one `application/json` object per
  completed call with a documented, stable schema: system, protocol, talkgroup
  (+label), source RID, frequency, P25 site identity (channel/RFSS/site/NAC),
  timeslot, encryption + emergency flags, patched groups, and RFC3339
  start/stop timestamps. An optional `Authorization` header is sent verbatim and
  `include_audio` opt-in embeds the base64 MP3 (default off keeps it a
  lightweight metadata feed). Reuses the existing Manager fan-out (system
  filter, min-duration gate, bounded exponential-backoff retry); configure under
  `broadcast.webhook[]` and in the web Config Builder. See `config.example.yaml`.

### Changed
- **siglab names the no-lock reason + opt-in soft-sync fallback** (#771). The
  scan verdict now classifies *why* a control channel did not lock —
  SNR/EVM-limited, frame-sync-not-aligned, not-the-control-channel, or
  no-signal — from the demod and frame-decode metrics already gathered, rendered
  after the "did NOT lock" line and serialized into the CSV/JSON/YAML summaries.
  A `replay -soft-sync` flag (off by default; the daemon is unchanged) widens the
  P25 FSW correlator so sync words carrying 5–8 symbol errors decode, but admits
  those looser hits only through the existing TSBK-CRC-corroborated marginal NID
  tier, so a looser sync extends reach into marginal-SNR captures without
  manufacturing a false lock.

### Fixed
- **DMR 2-slot cadence detected by AMBE FEC, not the embedded LC** (#644). The
  interleaved DMR voice decoder picked its same-slot TDMA cadence solely by which
  candidate stride reassembled a CRC-valid embedded Link Control — a decode that
  frequently never validates on a real outbound carrier, so it fell back to the
  264-dibit cadence. On a 288-cadence carrier that sliced every burst 24 dibits
  off, splicing the other timeslot's / the CACH's bits into each AMBE frame and
  rendering audio as structured noise that "sounds encrypted" (the reopened
  symptom of #644). Cadence is now scored by AMBE Golay(23,12) corrected-bit
  count when no candidate yields an authoritative LC: a correct slice decodes
  with ~0 corrected bits, a wrong slice scores far higher, and a ceiling-and-
  margin gate keeps genuinely garbled data from locking a cadence. The LC
  remains authoritative when it does decode.
- **Silent-from-start calls end at a bounded window, not the 30 s watchdog**
  (#788). A voice grant whose channel never delivered a single matching voice
  frame (carrier already dropped, a stale CC grant re-announcement, or a
  mis-tuned / under-gained tap) was held by the boundary tracker for the engine's
  full call-timeout watchdog — 30 s by default — showing as an "active" call with
  a climbing ELAPSED and no audio. A tracker that has never seen a matching voice
  frame now ends the call (reason `timeout`) once it has run for 2× hangtime
  (7 s by default); if voice resumes, the CC re-announces the grant and a fresh
  call starts.

## [v0.5.1] — 2026-06-23

A maintenance release headlined by a new **ka9q-radio network SDR source** —
mount a channel from a remote `radiod` instance over IP multicast as a virtual
tuner, so one well-sited front end can feed many GopherTrunk decoders across a
LAN — and a fix that restores **wideband decode at sample rates above
2.5 MS/s** (#764), where a `role: wideband` dongle run at 10 MS/s went deaf
across every tap. The rest is P25 control-channel work: System ID recovered
from adjacent-site broadcasts, RFSS/Site from the Location Registration
Response, hex ID rendering, a corrected Phase 2 Motorola talker-alias
reassembly, and new raw-broadcast / event-log field diagnostics.

### Added
- **ka9q-radio network SDR source** (`sdr.ka9q_radio`, #765). Consume a channel
  from a remote ka9q-radio `radiod` instance over IP multicast, in pure Go.
  radiod runs fast-convolution downconverters on a front end and multicasts each
  channel as RTP; a channel in raw "linear" IQ mode (output_channels = 2) is
  mounted as a virtual tuner, so one well-sited radiod can feed many GopherTrunk
  decoders across a LAN. The driver discovers the channel's IQ multicast group,
  sample rate and encoding by polling radiod's status group, resolves `.local`
  instance names via mDNS, retunes via RADIO_FREQUENCY commands, and decodes
  s16/f32 (either byte order) IQ payloads. Configure under `sdr.ka9q_radio`
  with the status group `addr` and channel `ssrc`; see `config.example.yaml`.
- **P25 raw status-broadcast dump + live event log** (#779). Two additive field
  diagnostics, no decode-behaviour change. A per-opcode sampled raw-payload +
  decoded-field dump for the handled adjacent (0x3C), network (0x3B) and RFSS
  (0x3A) status broadcasts captures the bytes behind a reported neighbour /
  identity mismatch when there is no IQ to replay. An optional live event
  JSONL/NDJSON sink (`log.event_log`) records every bus event as one JSON line
  in the same envelope the SSE/WS streams emit, with size-capped rotation;
  surfaced in the web Config Builder.

### Changed
- **P25 identity recovered from more control-channel opcodes** (#766, #774).
  Systems that never transmit Network/RFSS Status Broadcasts (0x3B/0x3A) used
  to leave the web "Network identity" panel stuck on "Awaiting status
  broadcasts". The System ID is now voted in from Adjacent Site Status
  Broadcasts (0x3C), and RFSS/Site from the Location Registration Response
  (LOC_REG_RSP, 0x2B) — the practical identity source on such systems. WACN
  publishes the instant the Network Status Broadcast lands, and a new
  per-opcode seen-counter lets the hunt/siglab layers distinguish a genuinely
  unavailable WACN (NSB never transmitted) from a decode gap.
- **P25 IDs render in hex** (#774). Site-update events and the systems/sites
  REST DTOs gain `*_hex` fields (decimal values unchanged); the web "Network
  identity" card and the TUI render WACN/System ID/RFSS/Site in the configured
  base (`web.id_base`, hex default).

### Fixed
- **Wideband DDC/channelizer decode at sample rates above 2.5 MS/s** (#764). A
  `role: wideband` dongle run above 2.5 MS/s went deaf across every tap. The
  DDC path now runs a single shared decimate-by-D stage over the wideband
  stream to bring it into [2.5, 5) MS/s before the per-tap mixers, pinning
  per-tap cost to the proven ~2.5 MS/s regime so the pump goroutine keeps real
  time (below 5 MS/s the 2.4 MS/s path is unchanged). The channelizer bin count
  now scales with the sample rate to hold ~150 kHz bins (64 bins at 10 MS/s)
  instead of colliding adjacent carriers into one bin.
- **P25 Phase 2 Motorola talker-alias reassembly** (#773). The Motorola FACCH-S
  alias data fragments begin on the low nibble of their sequence octet and must
  be concatenated as a nibble stream; the previous code dropped that leading
  nibble and concatenated whole bytes, shifting the cipher region by 4 bits so
  the alias decoded to garbage and failed safe to empty. Reassembly is now
  nibble-aligned (the end-to-end decoded string remains air-unverified).
- **P25 Motorola talker-alias cipher gated as unverified** (#773). A clean-room
  attempt to derive the per-byte cipher from the one partial capture available
  (RID 200062) showed it is mathematically underdetermined — dozens of distinct
  constant-sets reproduce the known bytes while disagreeing on the unknown ones,
  and one alias character is unrecoverable from that sample — so the substitution
  table and constants cannot be pinned without ground truth (and SDRTrunk's GPLv3
  implementation cannot be ported into Apache-2.0 GT). The decode now carries a
  `CipherVerified` flag (false) and `DecodeMessage` never reports an alias as
  reliable while it is false, so a possibly-wrong table can never surface a
  fabricated name as a confirmed alias. Misleading "reverse-engineered fact"
  comments on the table were corrected to state the true unverified provenance.
- **P25 autotune no longer over-corrects** (#774). Implausible single AFC
  measurements are rejected, a warm-up of several samples is required before any
  correction is applied, and only cleanly-decoded voice calls feed the estimate,
  so convergence stays in the low-Hz range instead of drifting into the kHz.
- **Web CC Activity table freezes synchronously on pause** (#772). The pause
  snapshot is now captured during render the first time pause is seen rather
  than in a post-paint effect, closing the window where an SSE batch could leak
  live rows through after the click. The web vitest suite is also wired into CI.

## [v0.5.0] — 2026-06-22

A feature release headlined by a **standalone P25 Phase 2 talker-alias
signalling follower**: on Phase 2 systems the alias rides the traffic
channel's FACCH-S MAC signalling during hangtime, so on a busy multi-site
system — where most grants never get a voice tap and encrypted calls tear
down before hangtime — it was almost never decoded. A new signalling-only
follower allocates lightweight DDC taps on a `role: wideband` dongle and
harvests the alias off the traffic channel independent of the voice pool, the
way SDRTrunk does with two SDRs. Alongside it, a new **decode-activity-gated
power log** gives the wideband engine an opt-in, per-channel IQ-power
diagnostic that only records windows where a protocol is actually decoding —
the "decoding but the signal is marginal" view — instead of spamming every
idle or off-band channel.

### Added
- **P25 Phase 2 talker-alias signalling follower** (`signalling_taps`, #376).
  On Phase 2 systems the talker alias rides the traffic channel's FACCH-S MAC
  signalling during hangtime, not the control channel — so decoding it used to
  require a voice tuner following the call. On busy multi-site systems most
  grants never get a voice tap (and encrypted calls are torn down before
  hangtime), so the alias was almost never decoded. A new signalling-only
  follower (`internal/sigfollow`) allocates lightweight DDC taps on a `role:
  wideband` dongle's IQ stream and harvests the alias off the traffic channel
  independent of the voice pool — the way SDRTrunk does with two SDRs. Enable
  it per wideband device with `signalling_taps: N` (0 disables; 2-4 suits a
  busy Phase 2 system). The FACCH-S MAC dispatch is now shared between the
  voice composer and the follower so the two paths never diverge. Decoded
  aliases bind onto the RID automatically via the affiliation tracker, with no
  extra wiring.
- **Decode-activity-gated power log** (`log.power_log`). A new opt-in,
  rotating per-channel IQ-power log for the wideband engine. Each line is
  gated on decode activity across all four wideband protocols (DMR Tier II /
  Tier III, P25 Phase 1 / Phase 2), so idle and off-band channels never
  appear. By default it records only low-power (weak-signal) windows on
  channels that are actively decoding — the "decoding but the signal is
  marginal" diagnostic — with `log.power_log.all_windows: true` opting into a
  full power time-series of every decode-active window. To support the gate,
  DMR Tier III and P25 Phase 2 control channels now expose a `DecodedFrames()`
  counter alongside the existing Tier II / Phase 1 counters.

## [v0.4.9] — 2026-06-21

A field-driven bug-fix release built around six issues reported on a live P25
Phase 1 system, plus follow-up audio plumbing and import hardening. The
headline fix closes a **cross-call audio-bleed window in the live PCM fan-out**:
the earlier on-disk CallID fence didn't cover the streaming tap, so a reused
wideband voice-tap serial could relabel a draining call's audio with the next
call's talkgroup. The web **Scanner tab no longer crashes** on a null
systems/channels payload, **trunked hunts now converge** on identity and
topology instead of timing out in a 3 s dwell, and discovery stops inventing
bogus talkgroups from unit-to-unit / telephone / SNDCP-data grants. On the side:
RID and talkgroup **alias imports are hardened against mojibake** (UTF-16/BOM
SDRTrunk exports are transcoded and non-printable runes stripped), `sdr list`
**prints full HackRF serials** with a bounded `--probe`, and the gRPC API can
finally **stream raw IMBE/AMBE+2 vocoder frames** to `include_raw` subscribers.

### Added
- **Raw vocoder frame streaming over gRPC** (#746). The
  `AudioSubFilter.IncludeRaw` flag was a near no-op — `WriteRawFrame` was never
  wired into the publisher, so raw IMBE / AMBE+2 frames stayed recorder-only. A
  new raw tap mirrors the decoded-PCM path and fans the verbatim frame (plus
  vocoder name and CallID) to the audio publisher before decode, so it fires
  even for protocols with no in-process decoder (ProVoice, encrypted) where the
  raw bytes are the only audio. Raw is purely additive — PCM-only clients
  (including the WebUI) are unaffected, and the cross-call fence applies at both
  the publisher and recorder layers.
- **Monitor-minutes field on the blind-sweep hunt form** (#746). Only the
  parse-control-channel form sent `monitor_seconds`; the blind-sweep payload
  omitted it, so a sweep could never engage the converge-and-stop monitor
  (backend default 0 = off). The sweep form now carries a "Monitor (minutes,
  0 = off)" field, so a sweep that locks a control channel can monitor it until
  identity, neighbors, and band plan settle.

### Changed
- **`sdr list` shows full HackRF serials and bounds `--probe`** (#745). The
  fixed-width SERIAL column front-truncated to 16 chars, which for a HackRF cut
  off the meaningful tail of its 32-hex part_id+serial and left only the
  all-zero prefix (reported as `0000000000000000`); TUNER and PRODUCT were
  clipped too. Columns now size to the widest value present, and each
  `--probe` open+info+close is wrapped in a bounded helper so a wedged device
  can't hang the command. Adds a Linux (udev) setup section to the HackRF doc.
- **RID / talkgroup alias imports hardened against mojibake** (#744, issue
  #711). A shared sanitization layer is wired into both the RID and talkgroup
  CSV/JSON loaders: a UTF-16 (LE/BE) or UTF-8 BOM is honoured and transcoded to
  plain UTF-8 (neutralising Windows/SDRTrunk exports), and text fields are
  reduced to printable ASCII, dropping control chars, NULs, and non-ASCII
  mojibake. BOM-less ASCII/UTF-8 passes through untouched. Promotes
  `golang.org/x/text` to a direct require.
- **Trunked hunts default to the converge-and-stop monitor** (#743). The web
  hunt previously ran a 3 s buffered dwell; P25 status broadcasts cycle too
  slowly to land in 3 s, so only NAC surfaced. A trunked hunt now defaults to
  the streaming monitor that ends early once identity + topology settle.

### Fixed
- **Cross-call audio bleed when a voice-tap serial is reused** (#743). The
  on-disk CallID fence guarded only the WAV; the live PCM fan-out (recorder
  decodedTap → AudioPublisher) was keyed purely by device serial, so a reused
  wideband voice-tap serial leaked the previous call's still-draining audio to
  subscribers filtered on the new call's talkgroup. The session's CallID is now
  threaded through to `WritePCMForCall`, which drops a frame whose CallID no
  longer matches the serial's bound call.
- **Web Scanner tab crash on a null payload** (#743, #746). A nil Go slice
  marshals to JSON `null`; the Scanner tab dereferenced `systems.length` /
  `channels.length` and crashed with "Cannot read properties of null". The
  `/api/v1/scanner` payload now emits `[]` and the frontend null-guards the
  reads, with frontend tests covering both the null-shaped and populated
  snapshots.
- **Hunt identity/neighbors stuck on "awaiting status broadcasts"** (#743).
  Beyond defaulting to the streaming monitor, topology folding no longer drops
  a legitimate `RFSS=0` / `Site=0`.
- **Bogus talkgroup from non-group grants** (#743). Unit-to-unit / telephone /
  SNDCP-data grants publish a 24-bit target unit as the grant "group", and
  discovery was recording every grant group as a talkgroup. These grants are
  now flagged Individual and skipped when building the hunt talkgroup list (the
  frequency is still recorded on the site).
- **Spurious talkgroup↔frequency association** (#743). The per-talkgroup
  frequency list is dropped — on a trunked system the traffic channel is
  assigned dynamically per call, so a talkgroup has no fixed frequency. The
  site's distinct voice-channel pool is unchanged.

## [v0.4.8] — 2026-06-20

This release is dominated by a **six-phase operator-console overhaul** that
rebuilds the web UI's navigation, feedback, primitives, tables, dense panels,
and accessibility from the ground up. The flat tab bar is replaced by a
registry-driven app shell — a collapsible desktop sidebar, a mobile bottom nav
+ drawer, and a ⌘K/Ctrl-K command palette — that reaches all 31 routes,
including six DSP panels that were previously unreachable except by URL. On top
of that: a toast notification system with a shared polling hook that kills
re-render flicker and shows reconnect state, a reusable primitive layer with a
new light theme and a comfortable/compact density axis, a DataTable with
built-in search / pagination / inline actions / column visibility, regrouped
Hunt / Scanner / Settings panels, and a pass over focus traps, a skip link,
reduced-motion, and chart labelling (mirrored into the Signal Lab and Config
Builder SPAs). On the decode side, **P25 TSBK coverage** gains named standard
OSPs and a field-decoded Secondary Control Channel Broadcast — Explicit (0x29),
empty/all-muted recordings are suppressed to stop tiny-file spam, and a
per-call ID closes a cross-call audio-bleed window. The docs grow a new
**Digital & Trunked Radio** learning path and a **Software Development** domain
in the Reference encyclopedia.

### Added
- **Web console navigation overhaul — app shell + grouped nav + command
  palette** (#736). A single nav registry feeds four surfaces — a collapsible
  desktop sidebar, a mobile bottom nav (4 primaries + More), a full mobile
  drawer, and a ⌘K/Ctrl-K command palette — organizing all 31 routes into six
  groups (Live Ops, Discovery, Signals & DSP, Decoders & Logs, Database,
  System). This rescues 21 panels that were stranded behind a hidden mobile
  overflow row and six DSP panels (constellation, symbols, eye, mixer, tuning,
  histogram) that were in no tab list at all. The daemon's `hidden_tabs` filter
  and the Config Builder external link are preserved.
- **Notification + feedback system** (#739). A toast queue with kind-based
  auto-dismiss replaces the single manually-dismissed error strip; every
  existing `setError(...)` now surfaces a visible toast. A shared `useDataPoll`
  hook centralizes the poll/cancel pattern, only re-renders when a snapshot
  actually changed (killing poll-induced flicker), and shows a
  "reconnecting · Ns ago" chip when the daemon goes quiet. Mutations gain
  visible "saving…" / success states.
- **Reusable UI primitive layer + light theme + density axis** (#739). A
  `components/ui/` set (Card, Badge, Input, Select, Checkbox, Field, PageHeader,
  EmptyState, Skeleton, Section, Button, Spinner, ToastViewport) lets panels
  stop hand-rolling controls. Design tokens gain a `[data-theme="light"]` theme
  (Settings now offers a dark/mono/light toggle) and a `[data-density]` axis
  (comfortable/compact), both applied pre-render and persisted.
- **DataTable overhaul** (#739). The shared table gains opt-in, backward-
  compatible built-in search, pagination, loading skeletons, an inline
  row-actions cell, a toolbar slot, and a persisted column-visibility menu.
  Talkgroups, Radio IDs, Systems, Active, and History adopt these; Talkgroups
  also gets inline scan/lockout quick-toggles.
- **"Digital & Trunked Radio" learning path** (#737). A new structured path
  under `/learn/` — 6 modules, 31 lessons + glossary — covering digital radio
  and trunking end to end: a brief history, the digital signal chain (vocoders,
  modulation, framing, control channels), each trunking system GopherTrunk
  decodes (P25 Phase 1/2 and DMR in depth; TETRA, NXDN, Motorola, EDACS, LTR,
  MPT-1327, dPMR, D-STAR, YSF more briefly), and hands-on decoding. Registered
  in `_data/learn.yml`, so the chooser, hub, lesson nav, and `llms.txt` pick it
  up automatically.
- **"Software Development" domain in the Reference encyclopedia** (#738). The
  encyclopedia is restructured from a flat category list into a two-level
  Domain → Category → Entry model (the 11 existing RF categories move under an
  "RF & SDR" domain unchanged), and a full Software Development domain is built
  out: 61 new entries across 6 categories — 19 programming languages plus 42
  concepts spanning language internals, concurrency, paradigms & design
  patterns, principles & quality, and testing/tooling/delivery — each
  cross-linked and wired into the learning paths.
- **P25 Secondary Control Channel Broadcast — Explicit (0x29) decode** (#740).
  The explicit SCCB is now field-decoded and its downlink channel folded into
  the network topology (dispatched regardless of MFID like the other standard
  broadcasts), and the standard OSPs that were logging as `OSP(0xNN)`/unhandled
  — including 0x16 `SNDCP_DAT_CH_ANN_EXP` and 0x29 `SCCB_EXP` — are now named.

### Changed
- **Dense panels regrouped onto the primitives** (#739). Hunt is rebuilt on the
  UI primitives (clearing long-standing styling debt where it referenced
  undefined CSS classes), split into a "Parse a control channel" quick-start, a
  collapsed "Blind sweep" advanced section, and result Cards. Scanner's three
  sub-panels move to Cards with state pills; Settings becomes a collapsible
  Section with a plain-language tooltip on each config field.
- **Accessibility & cross-SPA polish** (#739). DetailModal and ConfirmModal now
  trap focus and restore it to the trigger on close; a visually-hidden "Skip to
  content" link jumps past the nav chrome; a `prefers-reduced-motion` rule
  neutralizes transitions (mirrored into the Config Builder and Signal Lab
  SPAs); and the Metrics trend chart gains `role="img"` + an aria-label.

### Fixed
- **P25 opcode 0x39 was mislabelled** (#740). 0x39 is the non-explicit SCCB;
  the explicit variant is 0x29 (per TIA / SDRTrunk / OP25). The two are now
  named correctly.
- **Empty/all-muted recordings no longer spam tiny files** (#740). A vocoder
  call that decodes no real speech (voiced + unvoiced == 0 — all idle-muted)
  now has its WAV/raw removed and publishes no `CallComplete`, instead of
  leaving a silent file that, in per-transmission mode, was the dominant source
  of tiny files. A per-call b₀ range + escalated WARN line lets a genuine
  dead-key be told apart from empty frames reaching the vocoder.
- **Cross-call audio bleed when a voice-tap serial is reused** (#740). Each
  bound call now gets a process-unique `Grant.CallID` (preserved across Retune
  handoffs) threaded to the recorder, which drops frames whose CallID doesn't
  match the open session — closing the window where a reused tap could bleed one
  call's audio into another.

## [v0.4.7] — 2026-06-19

This release sharpens **P25 discovery accuracy** and **identity corroboration**.
Status-broadcast identity (WACN / System ID / RFSS / Site) is now resolved by
majority vote instead of last-write-wins, so a single corrupt-but-CRC-passing
TSBK can no longer poison the system map; the always-standard band-plan /
network / site opcodes are decoded even when a Motorola trunk sends them under
the vendor MFID (#728); registration and affiliation events carry `rfss_id` /
`site_id` / `nac` for a genuine RID→site fix (#698); and a new opt-in
streaming long-dwell control-channel monitor watches a site for minutes
without recording gigabytes of IQ (#722). Encrypted-call handling moves to a
per-system policy and `metadata` mode now releases the voice tuner the moment
the talker alias completes (#711). New **autotune** tracks each dongle's
carrier error and digitally corrects it (#729), the web console gains a
group-duplicate-events toggle, a freeze-while-inspecting stream, and a fixed CC
Activity pause, and a RadioReference fix recognises feed-provider / admin
accounts as premium (#723). On the docs side: a new "Intro to Software Dev"
learning path (#725).

### Added
- **Group duplicate events in the Events & CC Activity panels.** A "Group
  duplicates" toggle (on by default) collapses repeated events with identical
  content — the same grant / registration / affiliation re-sent every few
  seconds — into one row with an ×N count and the latest timestamp, so the
  panels stop spamming. Toggle off for the raw per-event stream.
- **Motorola opcodes 0x05 / 0x09 (MFID 0x90) named and captured.** Cross-checked
  against SDRtrunk / OP25: 0x05 is `MOTOROLA_OSP_TRAFFIC_CHANNEL_ID`, 0x09 is
  `MOTOROLA_OSP_SYSTEM_LOADING` — neither carries neighbour / secondary-CC data,
  and neither reference decoder field-decodes them. So they are named and their
  raw payload is logged at INFO (up to 64 samples each, grep `motorola opcode`)
  while decoding/publishing nothing into the system map. See
  [`docs/specs/p25-motorola-opcodes.md`](docs/specs/p25-motorola-opcodes.md).
- **Autotune — per-dongle carrier-error correction** (opt-in, `sdr.autotune`).
  Ported in concept from trunk-recorder's autotune. Each SDR's crystal error
  shifts the received carrier off-centre; with autotune on, the daemon watches
  the locked P25 Phase 1 receiver's residual carrier offset (control channel
  and voice calls), keeps a running average of the last 20 measurements per
  device serial, and shifts the channel's digital down-converter by that
  average so the demod's AFC starts near lock at the next acquisition. It never
  rewrites the dongle's hardware ppm — it logs a suggested value (with a
  >3.5 PPM "verify your ppm" warning) you can bake into the device block by
  hand. Off by default and zero-cost when disabled; safe to leave on for
  TCXO-equipped units (the correction stays near zero).
- **Site identity on registration & affiliation events** (#698). The
  `unit_registration` and `affiliation` events now carry `rfss_id` / `site_id`
  (alongside `grant`), plus a `nac` field on all three. Registration and
  affiliation are handled by the radio's actual serving site, so they give a
  genuine RID→site location fix that grant-site — announced on every site of a
  wide-area call — cannot. Phase 2 (TDMA) control channels now also discover
  their sites into `GET /api/v1/sites`.
- **Streaming long-dwell control-channel monitor for Hunt** (opt-in). A new
  `monitor_seconds` (web Hunt "Monitor" field / `-monitor-seconds` CLI flag)
  decodes a locked control channel from the live SDR in real time instead of
  buffering the whole dwell, so a P25 site can be watched for minutes without
  recording gigabytes of IQ. It stops early once identity, neighbors, and the
  band plan stop changing (converge-and-stop), capped at the requested time.
- **P25 Unit-to-Unit Answer Request decoding** (opcode 0x05). The private-call
  answer-request handshake is now decoded — in both the standard and Motorola
  (MFID 0x90) forms — and published as a new `unit.request` bus event carrying
  the calling/called radio IDs, surfaced on the web CC-activity panel and the
  message log instead of being logged as an unhandled vendor TSBK.
- **"Intro to Software Dev" learning path** (#725). A third structured path
  under `/learn/` — 7 modules / 40 lessons plus a glossary — covering software
  development from a brief history through languages, principles, design
  patterns, development systems, choosing a language, and building a solo
  developer stack, with examples carrying a subtle RF/SDR slant. Joins the
  existing RF & SDR and Git & GitHub paths in the shared data-driven Jekyll
  system, so the chooser, hub, lesson nav, and `llms.txt` pick it up
  automatically.

### Changed
- **P25 TSBK opcodes now log with their TIA-102.AABC-D designations**
  (`UU_ANS_REQ`, `NET_STS_BCST`, `GRP_V_CH_GRANT`, …) instead of the
  OP25-derived names. The Go identifiers stay descriptive; `Opcode.String()`
  and a new [`docs/specs/p25-tsbk-opcodes.md`](docs/specs/p25-tsbk-opcodes.md)
  carry the canonical mnemonics (mirrors the NXDN package convention).
- **Encrypted-call handling is now per-system** (#711). The `encrypted_calls`
  policy (`mode` + `metadata_follow_ms`) moved from the global
  `trunking.encrypted_calls` key to per-system `trunking.systems[].encrypted_calls`,
  so an operator can run `metadata` on one system and `follow`/`ignore` on
  another. **Breaking:** a global `trunking.encrypted_calls` block from v0.4.6
  no longer takes effect — move it under the relevant `trunking.systems[]`
  entry. Because it is now per-system config, the setting is applied at startup
  (restart-required); the runtime API / TUI / settings-PATCH fields for it have
  been removed.
- **`metadata` mode releases the voice tuner as soon as the talker alias
  completes** (#711, building on #376), instead of always holding for the full
  `metadata_follow_ms`. On P25 Phase 2 the alias arrives during call hangtime as
  a FACCH-S block sequence; once it fully reassembles the metadata goal is met,
  so the tuner is freed immediately — `metadata_follow_ms` becomes an upper
  bound rather than a fixed wall-clock hold.

### Fixed
- **UU_ANS_REQ (opcode 0x05) is no longer mis-decoded under a vendor MFID.**
  A Motorola (MFID 0x90) 0x05 decoded with the standard Target/Source layout
  produced garbage radio IDs (`src=0`, `target=0x3C0000`); it is now left
  unhandled (vendor namespace), while the standard-MFID UU_ANS_REQ still
  publishes a `unit.request` event.
- **CC Activity "pause" now actually freezes the table.** Previously it
  re-sliced the live rows, so the list kept churning and an event was
  impossible to read or click; it now snapshots the rows on pause.
- **Events stream freezes while inspecting an event.** Opening an event for
  inspection no longer lets the live stream churn underneath the selection, so
  the row you clicked stays put until you dismiss it.
- **Systems detail no longer mislabels missing network identity as "Scanner
  offline".** WACN/System ID/RFSS/Site are decoded live from status broadcasts
  whenever the control channel is received, so an empty cell now reads
  "Awaiting status broadcasts" (or "Hunting control channel" during a hunt).
- **Systems detail modal no longer crashes on a null systems list** (#721).
  Clicking a system crashed with "Cannot read properties of null (reading
  'find')" whenever the scanner reported no active control-channel hunt
  statuses (the daemon marshals an empty hunt list as JSON `null`); the optional
  chain now guards the systems list too (`scanner?.systems?.find`).
- **Hunt progress frequency precision.** The identifying-phase progress line
  rounded the tuned frequency to 3 decimals (showing a 6.25 kHz-raster channel
  like 160.5875 MHz as "160.588"); it now prints 4 decimals, matching the rest
  of the UI. Display-only — the SDR was always tuned to the exact frequency.
- **P25 identity accuracy under noise.** Status-broadcast accumulation was
  last-write-wins, so a single corrupt-but-CRC-passing TSBK could set a wrong
  WACN / System ID. Identity (WACN/SysID/RFSS/Site) is now resolved by majority
  vote across observations, so a repeated correct value out-votes a one-shot
  wrong one. Neighbors and band-plan slots surface on the first sighting
  (de-duplicated) to match OP25's latency.
- **P25 Motorola identity/neighbors not decoding.** On some Motorola trunks the
  band-plan and network/site/secondary broadcast opcodes (IDEN, RFSS, Network
  Status, Adjacent Site) arrive under the vendor MFID (0x90); they were routed
  to vendor dispatch and dropped, so WACN / System ID / neighbors stayed empty
  even though the control channel locked. These always-standard opcodes are now
  decoded regardless of MFID (an INFO line flags when one is seen under a vendor
  MFID), matching OP25/boatbod.
- **DMR Tier III Announce Channel-Frequency** (C_BCAST anncd_type 5) is now
  recognised and its raw payload surfaced once at INFO, so its (site-specific)
  layout can be validated off-air; neighbor/voice frequencies continue to
  resolve through the configured band-plan resolver / LCN learner.
- **RadioReference feed-provider / admin accounts now recognised as premium**
  (#723). `getUserData` returns the sentinel strings `"Never - Feed Provider"` /
  `"Never - Admin"` as the subscription expiry for accounts that hold premium
  for as long as their feed is active; the parser only handled dates / UNIX
  timestamps, so these fell through to "no active premium subscription". A
  `"Never…"` expiry is now treated as active.

## [v0.4.6] — 2026-06-18

This release focuses on **P25 site and identity decoding**. Encrypted calls
are now configurable — a new `trunking.encrypted_calls.mode` stops a few long
encrypted calls from exhausting the voice-tuner pool and starving clear
traffic (#711). Every `grant` event carries the decoded RFSS/site, exposed
alongside a new `GET /api/v1/sites` endpoint (#698), the web Systems panel's
WACN / System ID / RFSS / Site fields finally populate from the live site
tracker instead of sitting on "Awaiting status broadcasts" (#673), and a
Phase 1 status-broadcast parsing bug that had every WACN / System ID / RFSS /
Site / neighbor reading one byte early is fixed (#716). Motorola **talker
aliases** now decode off the Phase 1 TDULC terminator and Phase 2 FACCH-S, not
just LDU1 (#376), and the Hunt discovery view surfaces the voice/traffic
frequencies behind each grant. On the docs side: two new deep-dive blog series
("Build in the Open" and "RF Front End") and a full Git & GitHub learning path.

### Added
- **Configurable handling of encrypted calls** (#711). A new
  `trunking.encrypted_calls.mode` selects how the engine spends scarce
  voice SDRs on encrypted calls: `follow` (default, unchanged — hold the
  tuner for the whole call), `metadata` (follow briefly to capture
  traffic-channel metadata — talker alias, source RID, encryption sync —
  then release the tuner after `metadata_follow_ms`, default 1500 ms), or
  `ignore` (never tie up a voice SDR on an encrypted call). Encryption is
  acted on both at grant time and when discovered mid-call. A call whose
  KeyID matches a configured `trunking.systems[].encryption_keys` entry is
  exempt and always followed. The mode is runtime-mutable via
  `PATCH /api/v1/settings` (`trunking_encrypted_mode` /
  `trunking_encrypted_metadata_follow_ms`), the TUI Settings panel, and
  SIGHUP reload — no daemon restart required. This stops a few long
  encrypted calls from exhausting the tuner pool and dropping clear
  traffic with `no voice device available for grant`.
- **Voice-channel frequencies in the Hunt discovery view.** The discovery
  model now records the band-plan-resolved voice/traffic frequency of every
  grant, surfaced both per-talkgroup (a Frequencies column) and per-site (a
  Voice channels column) in the Hunt panel, and in the `system` JSON as
  `talkgroups[].frequencies` and `sites[].voice_channels`.
- **P25 site identity in grant events and a `/api/v1/sites` endpoint**
  (#698). Every `grant` event now carries `rfss_id` / `site_id`,
  decoded from the camped site's RFSS Status Broadcast, so downstream
  tooling (Prometheus exporters, dashboards) can label calls by site
  instead of an opaque `channel_id`. A new `GET /api/v1/sites`
  endpoint lists the sites GT has discovered from the control channel
  — each with the control-channel frequency it was heard on — merged
  with optional human-readable names configured per system under
  `trunking.systems[].sites` (`rfss` / `site` / `name`).
- **P25 Motorola talker aliases on TDULC and FACCH-S** (#376). Real
  Motorola talker aliases ride two carriers the decoder previously skipped
  past — the Phase 1 TDULC terminator and the Phase 2 FACCH-S — not just
  LDU1 link control, so aliases never surfaced on many systems. All three
  carriers now share one decode primitive (a new `internal/radio/p25/motorola`
  package handling the cipher + WACN|System|RID|alias|CRC-16 framing), and
  completed aliases publish the existing `KindTalkerAlias` event, bound onto
  the RID and surfaced at `/api/v1/rids`.
- **P25 System Information panel populates from the live site tracker**
  (#673). The web Systems detail panel showed WACN / System ID / RFSS / Site
  stuck on "Awaiting status broadcasts" because `GET /api/v1/systems` built
  its DTOs solely from static config. The endpoints now overlay the always-on
  site tracker's decoded over-the-air identity (WACN/System ID system-wide,
  RFSS/Site from the most recently heard site), falling back to config values
  when nothing has been decoded yet.
- **"Build in the Open" blog series.** A 14-part tutorial series taking a
  project from idea to public release with GitHub and Claude Code, using
  GopherTrunk as the worked example, drip-released one post per weekday.
- **"RF Front End" blog series.** A 14-part deep dive into GopherTrunk's
  pure-Go RF source layer (RTL-SDR, Airspy, HackRF): the Device contract, the
  driver registry, the no-libusb USB transport across Linux/macOS/Windows,
  per-radio register bring-up, and IQ conversion.
- **Git & GitHub learning path.** The single "Learn RF & SDR" curriculum
  becomes a multi-path "Learn" section, adding a complete 29-lesson Git &
  GitHub path alongside the existing RF & SDR one. Old RF lesson URLs keep
  working via redirects.

### Fixed
- **P25 discovery now decodes WACN / System ID / RFSS / Site / neighbors
  correctly** (#716). The Phase 1 Network-, RFSS-, and Adjacent-Site Status
  Broadcast TSBK parsers were dropping the leading LRA byte, so every
  field read one byte early — WACN and System ID came out wrong, RFSS/Site
  read as 0, and advertised neighbors showed garbled IDs with no resolvable
  frequency. The parsers now follow the TIA-102.AABF layout (matching the
  repo's independent Phase 2 decoders), which also corrects the `rfss_id` /
  `site_id` labels on `grant` events and `KindSiteUpdate`.

## [v0.4.5] — 2026-06-17

This release pairs two operator-facing fixes with a large docs-site
expansion. The Hunt panel's control-channel parser, previously locked to
P25, now covers **every trunked protocol with a standalone control
channel** — P25, DMR Tier III, NXDN, dPMR, EDACS, Motorola, MPT 1327,
TETRA, and YSF — via a protocol selector, and its listen window becomes a
free-form seconds field instead of a fixed dropdown (#707). On the decode
side, a fast scanner retune could race the USB stream teardown and fail the
next capture with `stream already active` / `conv: StreamIQ failed`; the
re-open now waits on the in-flight teardown across all drivers (#686). And
call history finally keeps the **mid-call backfilled radio ID and
encryption** on P25 Phase 2 grants, which start with a placeholder RID and
only surface identity on the traffic channel (#696). Alongside the code,
the docs site gains the **GopherTrunk Reference** encyclopedia (≈180
cross-linked entries with hand-authored SVG diagrams) and a 14-part **SDR
Internals** deep-dive blog series.

### Added

- **GopherTrunk Reference encyclopedia (#700, #701).** A new data-driven
  `/reference/` encyclopedia (≈180 entries) spanning RF fundamentals,
  modulation, voice coding, SDR & DSP, algorithms, antennas & propagation,
  hardware, trunked radio, protocols, people, and organizations. Entries
  follow a uniform profile, most carry a hand-authored theme-aware inline
  SVG structure/concept diagram, and the set is wired into the glossary and
  `llms.txt` so it is discoverable to both readers and AI search.
- **"SDR Internals" blog series (#705).** A 14-part deep-dive series — from
  "what is software-defined radio" through the driver registry, the
  streaming pool & concurrency, DSP foundations, tuning/channelization,
  demodulation, symbol timing, equalization/FFT, framing/FEC, and the
  protocol-decoder state machines — with a series landing page and nav
  entry.
- **Download / support / community CTAs across the docs site (#697,
  #702).** Non-home pages and the reference layouts now carry consistent
  call-to-action links, with tightened per-page meta descriptions.

### Changed

- **Hunt control-channel parser now covers all trunked protocols (#707).**
  The "Parse a control channel" Hunt panel was hardcoded to `p25` over an
  already protocol-agnostic API; it now exposes a protocol selector for
  every trunked protocol with a dedicated standalone control channel (P25,
  DMR Tier III, NXDN, dPMR, EDACS, Motorola, MPT 1327, TETRA, YSF) and
  forwards the choice to the hunt start. The fixed 5/10/15/20 s "Listen
  for" dropdown becomes a free-form numeric seconds input (min 1, default
  15) with a hint about the ~19 MB/s IQ cost. No backend change.

### Fixed

- **Scanner retune no longer races the IQ stream teardown (#686).** In
  scanner mode a fast retune cancels the IQ stream's context and
  immediately re-opens it, but the USB drivers tear a stream down in a
  detached goroutine that clears `streaming` only after `StopBulkIn` and
  the receiver-off control transfer complete — so the next `StreamIQ` often
  saw `streaming == true` and failed with "stream already active" (logged
  repeatedly as "conv: StreamIQ failed"). The fail-fast guard is replaced
  with a context-bounded wait on a per-stream `streamDone` channel, applied
  identically to rtlsdr, airspy, hackrf, and airspyhf, with a per-driver
  regression test that cancels and re-opens without draining the first
  stream.
- **Call history keeps the mid-call RID + encryption on call end (#696).**
  `recordEnd` previously wrote only `ended_at` / `duration` / `reason`,
  discarding the call's final identity. On P25 Phase 2 compressed grants the
  source RID starts at 0 and is backfilled mid-call on the traffic channel,
  and `ALGID`/`KID` likewise surface mid-call (Phase 1 LDU2, Phase 2
  EncryptionSync) — so `call_log` rows kept the grant-time placeholders
  (`source_id=0`, `algorithm_id=0`) and RIDs never appeared in history on
  exactly those Phase 2 systems. The end UPDATE now also persists
  `source_id` / `encrypted` / `algorithm_id` / `key_id` from the end grant
  with never-downgrade semantics (`COALESCE(NULLIF(?, 0), col)`, and an
  encryption flag that only upgrades). (`recordings.skip_encrypted` is
  unrelated — it only gates WAV/raw file writing.)

## [v0.4.4] — 2026-06-16

This release is the **RF & SDR learning hub** — a structured, vendor-neutral
learning path at [`/learn/`](https://gophertrunk.org/learn/) that walks a reader
from complete SDR newbie to confident GopherTrunk operator across six modules
(30 lessons plus a glossary). Every lesson carries a TL;DR, answer-first
question-style sections, reference tables, a hand-authored theme-aware inline SVG
where a diagram helps, a self-check quiz, an FAQ, prerequisite chips, and "use
this in GopherTrunk" links — paired with a dependency-free **site-wide search**
over every page, lesson, and post (#693). On the operator side, the Hunt web
panel gains a **"Parse a P25 control channel"** section that decodes a single CC
frequency and renders the discovered system in radio-programming format — WACN /
System ID / NAC, the voice-channel band plan, and talkgroups (dec/hex, encrypted
flag, activity count) (#694). P25 Phase 1 now **decodes Motorola patch-group
(super-group) voice channel grant / grant-update** TSBKs that previously logged
as unhandled, so those super-group calls surface their radio ID and encryption
and route through the normal band-plan / Phase 2 TDMA path (#376). And a live
TETRA-zone survey that **tore down the WebSocket stream on a non-finite float**
(a `+Inf` SNR / confidence reaching the JSON encoder, which cannot represent it)
is fixed end-to-end: clamped at the hunt classify chokepoint, scrubbed at the
JSON wire boundary for every event kind, and bounded to a finite SNR ceiling at
the P25 estimator source (#648).

### Added

- **RF & SDR learning hub at [`/learn/`](https://gophertrunk.org/learn/)
  (#693).** A data-driven curriculum (`_data/learn.yml`) powers a hub landing
  page, a glossary, and 30 lessons across six modules — RF fundamentals (bands,
  antennas, propagation), signal anatomy & modulation, the SDR receiver chain
  (IQ data, sample rate & Nyquist, gain & AGC, hardware), DSP (FFT/waterfall,
  filtering & decimation, demodulation, clock recovery), digital voice (analog
  vs digital, vocoders, the protocol landscape, encryption, other signals), and
  operating (finding systems, tuning with scopes, calibration & troubleshooting,
  legal & ethical). Each lesson has a TL;DR key-takeaways block, answer-first
  sections, reference tables, a hand-authored theme-aware inline SVG diagram, a
  knowledge-check quiz, an FAQ (mirrored as `FAQPage` schema), prerequisite
  chips, and "use this in GopherTrunk" links; progressive-enhancement JS adds a
  `localStorage` progress tracker, frequency↔wavelength and dB/dBm calculators,
  and the quizzes (pages stay fully usable with JS off). Per-lesson JSON-LD
  (`TechArticle` / `BreadcrumbList` / `FAQPage`) and a `Course` / `ItemList` on
  the hub make it SEO- and AI-search-friendly.
- **Site-wide client-side search (#693).** A dependency-free `/search/` over the
  whole docs site (top-level pages, the 30 learn lessons, and blog posts),
  suitable for static GitHub Pages with no external service or API keys: a
  Jekyll-built `search.json` index, a compact nav search box (full-width in the
  mobile overlay), live as-you-type ranked results with match highlighting, and
  `?q=` deep-links. Progressive — the form submits `?q=` to `/search/` with JS
  off.
- **P25 control-channel parser in the Hunt web panel (#694).** A "Parse a P25
  control channel" section in the Hunt tab: enter a single CC frequency (MHz),
  pick a bounded listen duration (5–20 s), and the existing hunt decode path
  decodes it. The discovered system — already served on `GET /api/v1/hunt` — now
  renders in a human-readable radio-programming format: system identity
  (protocol, WACN, System ID, NAC in uppercase hex), the per-channel band plan
  (base / spacing / bandwidth / TX offset) needed to program Motorola P25
  radios, and talkgroups (dec/hex, encrypted flag, activity count). No backend
  changes.
- **P25 Phase 1 decodes Motorola patch-group channel grant / update (#376).**
  Field captures from a Motorola system (MMR, sites CBD/Mt Anakie) emit
  manufacturer-specific (MFID `0x90`) TSBKs at opcodes `0x02` and `0x03` that
  GopherTrunk logged as unhandled — the raw-payload diagnostic confirmed they
  are patch-group (super-group) **voice channel grant** and **grant-update**
  messages, not talker aliases. They now decode through the same band-plan /
  Phase 2 TDMA routing as the standard grant/update opcodes. The `0x02` form
  carries the source RID and the service-options encryption bit, so the
  patch-group calls that previously arrived `src=0` / `enc=false` now surface
  their radio ID and encryption, and GopherTrunk follows the super-group voice
  calls it was dropping. (The unidentified `0x90` opcode `0x16` is left in the
  unhandled-TSBK census pending ground truth.)

### Changed

- **P25 Phase 1 unhandled-TSBK diagnostic now dumps the raw payload (#376).**
  The first round of CBD/Mt Anakie field diagnostics confirmed the alias is
  *not* arriving as our control-channel working-model fragment (vendor opcode
  `0x15`, plain ASCII): zero `cc talker alias`/`fragment` lines fired on either
  system, even the cleanly-decoding one. The census instead named candidate
  opcodes (Motorola `0x90` opcode `0x16`; standard `0x15`/`0x16`/`0x30`) but we
  couldn't tell which carries the alias from the opcode number alone. The
  `p25: unhandled tsbk` census line now also prints the **numeric** opcode
  (`Opcode.String()` mislabels vendor opcodes with standard names), and a new
  `p25: unhandled tsbk payload` line logs the raw 8-byte payload hex, capped at
  8 samples per distinct `(MFID, opcode)` so a multi-block alias sequence can be
  captured and reversed against known RIDs / alias strings without flooding a
  busy control channel. The prior diagnostic round shipped only on an unmerged
  branch, so a field replay built against `main` saw no payload lines; this
  lands it on `main`.

### Fixed

- **Live survey / WebSocket stream survives non-finite floats (#648).** A live
  TETRA-zone survey could not finish: a marginal carrier produced a non-finite
  float (a `+Inf` SNR / siglab confidence, or `-Inf` analog power from a
  divide-by-zero / log-of-zero), `json.Marshal` rejected it
  (`unsupported value: +Inf`), and the WebSocket handler tore the connection
  down on the write error so the scan never completed. Fixed at three layers:
  `internal/hunt` clamps every non-finite float on `DetectedSignal` (top-level
  plus nested Features / Trunking / Analog measurements) to `0` at the
  `classifyAndRoute` chokepoint every real signal flows through (reaching the
  stored survey, the event bus, and the NDJSON sink); a centralized,
  reflection-based `scrubNonFinite` returns a JSON-safe copy at the two
  serialization chokepoints (`eventToDTO`, covering both WS and SSE for every
  event kind, and the siglab identify + wideband REST responses), copying as it
  descends so the canonical store and NDJSON sink keep their original values;
  and the WS handler now marshals-first and skips-on-error (mirroring SSE)
  instead of killing the client's stream on one bad frame. Backstopping the
  source, the four P25 SNR estimators now return a finite `99.0 dB` ceiling
  instead of `+Inf` for a noise-free input, with a digital-vs-analog SNR
  ground-truth harness added as the regression guard.

## [v0.4.3] — 2026-06-15

This release is **Signal Lab wide-band tooling**, paired with a TETRA
real-air fix and two decode-validation milestones. Offline auto-tune used to
chase only the single loudest carrier, so a control channel that was both
**off-centre and not dominant** in a wide capture was missed; siglab and Hunt
now rank every carrier in the band and prefer the one that actually *locks*,
fixing the live P25 captures that previously mis-latched (#678). Building on
that, a new **wide-band multi-carrier survey** takes one IQ grab, finds every
carrier, decodes each, and recognises a DMR Tier III trunked system in a
single shot — exposed as a siglab library call, an offline `hunt -wideband`
path, a `POST /api/v1/siglab/wideband` endpoint, and a matplotlib-style
*Wideband* panel in the web console (#677, #679). A new **`gophertrunk
spectrum`** subcommand prints the band RMS power, Welch-averaged spectrum, and
detected carriers as text / JSON / CSV — the Python-free analog of
`numpy.fft` + `matplotlib` (#680). Signal Lab's Results-view PSD and
spectrogram now compute **server-side in Go**, letting the heavy
**TensorFlow.js** browser dependency be dropped entirely (#680). On the decode
side, TETRA control channels now **recover the BSCH colour code under any
π/4-DQPSK rotation** so real-air downlinks (which sit at a non-zero rotation)
actually learn their colour code (#681, #648), with a new differential-phase
*rotation tracker* viz to make the rotation visible (#682). Finally, live
captures **confirm DMR BPTC/RS Full-LC and P25 Phase 1 C4FM control-channel
decode on air**, closing two long-standing real-air validation gaps (#675,
#676, #527).

### Added

- **`gophertrunk spectrum` subcommand.** A Python-free, no-server analog of
  `numpy.fft` + `matplotlib`: given a recorded capture it prints the band RMS
  power, the Welch-averaged spectrum, and the detected carriers as text, JSON,
  or CSV — absolute frequencies with `-freq`, otherwise offsets from DC. On the
  delivered P25 captures it lists the 449.875 MHz control channel at SNR
  ≈47 dB. (#680)
- **Signal Lab wide-band multi-carrier survey.** `siglab.SurveyWideband` /
  `SurveyWidebandIQ` take a wide-band IQ buffer, find every carrier (Welch
  spectrum → peak detection → power-weighted centring of wide C4FM humps →
  grid-snap), down-convert and identify each, then cluster the results into
  control-vs-voice system rollups — recognising a DMR Tier III trunked system
  in one shot rather than auto-tuning to a single dominant carrier. Reused by a
  new offline `hunt -wideband -in <capture>` path that folds every DMR carrier
  into one discovered system. (#677)
- **Wide-band survey over REST + web.** `POST /api/v1/siglab/wideband` surveys
  a staged capture and returns the averaged band spectrum, per-carrier results,
  and system rollups (spectrum collection gated so the offline path stays
  lean). A new *Wideband* panel in `web/siglab` renders the averaged spectrum
  with each detected carrier marked by control/voice role alongside the
  trunked-system rollup table. (#679)
- **Hunt / siglab auto-identify of off-centre, non-dominant control channels.**
  Offline auto-tune now detects and ranks multiple carrier candidates
  (`dsp.EstimateCarrierCandidatesHz`) and, under `-auto-tune`, tries each —
  preferring a carrier whose decoder actually *locks* over a louder one that
  merely looks plausible — so a control channel under a louder neighbour is
  found automatically (13/13 live P25 segments now lock, was 11/13). Hunt gains
  a `-detect-carriers` flag that FFT-averages a recorded buffer and sweeps every
  detected carrier through the classify/decode body, inventorying a whole band
  from one capture with no SDR. (#678)
- **π/4-DQPSK rotation tracker viz.** Signal Lab's Results view gains a
  `RotationTracker` plot of per-symbol differential phase against the ideal
  ±π/4 / ±3π/4 rails (populated on the CQPSK path), making constellation
  rotation directly visible. Backed by new native-Go DSP helper packages
  (`internal/dsp/stats`, `internal/dsp/phase`) consolidated from previously
  inlined numpy/scipy-style operations. (#682)

### Changed

- **Signal Lab computes the Results-view PSD and spectrogram server-side; drops
  TensorFlow.js.** New `GET /api/v1/siglab/jobs/{id}/psd` and `/spectrogram`
  endpoints compute the spectra in Go from the job's captured IQ
  (`spectrum.AverageDB` / `Spectrogram`); the PSD, Spectrogram, and Compare
  overlay views now fetch the server-computed spectrum by job id instead of
  computing it in-browser. The `@tensorflow/tfjs` dependency — used only for
  that in-browser compute — is removed from `web/siglab`, shrinking the bundle.
  Plotly, Chart.js, and D3 are unaffected. (#680)
- **Wide-band survey DSP primitives promoted into shared, tested functions.**
  The Welch-averaged spectrum (`spectrum.AverageDB`), boxcar smoothing, robust
  percentile noise floor, power-weighted carrier centroid, RMS / dB helpers,
  and an STFT spectrogram now live in `internal/dsp/spectrum` and
  `internal/carriers` instead of being inlined and duplicated across siglab and
  hunt. Behaviour-preserving refactor guarded by the existing survey tests.
  (#679, #680, #682)

### Fixed

- **TETRA BSCH colour-code recovery was rotation-blind** (#648). π/4-DQPSK
  carries data in the differential phase, so a residual carrier offset
  cyclically rotates the whole demodulated stream by an unknown 0–3; the
  synchronisation-training-sequence correlator that gates BSCH colour-code
  recovery matched only the rotation-0 orientation, so real-air downlinks
  (which sat at rotation 1) never learned a colour code and the reference
  carrier never locked — while the synthesised fixtures, all at rotation 0,
  hid the bug. `RecoverColourCode` and the live STS detector now correlate the
  sync training sequence under all four rotations. (#681)

### Validated

- **DMR BPTC/RS Full-LC decode confirmed on real air** (#527 follow-up). Live
  441 MHz / 2 MS/s DMR Tier III captures replayed through the production
  receiver decode the `BPTC(196,96) → RS(12,9) → Full LC` chain cleanly
  (Terminator-with-LC recovers a stable RS-validated FLC; control-channel CSBKs
  pass BPTC + CRC and the Tier III control channel locks), refuting the
  "real-air BPTC/RS is broken" hypothesis. Captured as no-build-tag regression
  tests with committed channelised fixtures. (#675)
- **P25 Phase 1 C4FM control-channel demod validated on a live UHF capture**
  (#676). Thirteen live 450.500 MHz / 2 MSPS captures of a real UHF P25 system
  all decode to NAC `0x2C1`, with C4FM consistently out-decoding CQPSK; the
  cleanest segment reads EVM 12.7 %, SNR ≈14.5 dB, NID trusted 31 / failed 0,
  TSBK 36 with zero CRC/trellis failures. Recorded as a demod-quality gate
  (`samples/p25/p25-450875-cc.metadata.json`) with floors below the measured
  values. (#676)

## [v0.4.2] — 2026-06-14

This release is mostly **hunt / survey maturation**, with TETRA blind identify
and MPT 1327 lock hardening alongside. The live discovery flow grows up: a real
**RadioReference cross-reference** now pairs discovered control / voice
frequencies against an existing RR system to flag PPM / tuning offsets and
not-in-RR frequencies, and lists talkgroups heard on air but absent from RR —
available both from the CLI and the web *Hunt* panel (#660, #661). Surveys gain
**crash-safe NDJSON persistence and resume** in the daemon (mirroring the CLI
flags), an optional **post-run auto-gain BER sweep** that recommends the
front-end gain minimising decode errors on each locked control channel, and
**neighbour-frequency resolution for DMR Tier III / EDACS / Motorola** rather
than P25 only (#660, #661). The sweep now **snaps candidates to the channel
raster** (6.25 / 12.5 / 25 kHz) and normalises the reported occupied bandwidth,
so a real carrier a few hundred Hz off a bin centre is tuned dead-on and reads
its true channel width instead of a skewed "11.6 kHz" (#669, #648). TETRA
control channels now clear the blind-identify gate by **recovering the cell
colour code from the BSCH** so their SCH/HD FEC actually scores (#662, #648),
with an opt-in **Viterbi correction-depth histogram** behind
`metrics.detailed_fec` (#667). MPT 1327 stops false-locking on stray
cross-protocol parses — a lock now needs a minimum number of confirmation
codewords sharing a **consistent system prefix** (#669, #670, #648). On the
voice side, digital chains are now clocked from the **DDC's actual resampled
rate** instead of the nominal 48 kHz, killing the periodic symbol-clock slips
and audible clicks on wideband P25 / DMR voice (#668, #550). SoapyRemote
endpoints gain per-endpoint **`stream_mtu` and `stream_window`** knobs (#664,
#665), and P25 Phase 1 control-channel talker-alias decode is now observable in
the daemon log (#376).

### Added

- **Hunt: RadioReference cross-reference (offsets + new talkgroups).** The RR
  step is upgraded from duplicate-detection to a real cross-reference: when a
  discovered system matches an existing RR system with high confidence,
  GopherTrunk fetches its full detail and diffs it — control / secondary
  frequencies are paired greedily by nearest RR frequency (within 5 kHz is
  flagged as a tuning / PPM offset, farther or unmatched is "not in RR"), and
  talkgroups observed on air but absent from RR are listed. The diff is rendered
  in the submission package and, via a new
  `GET /api/v1/hunt[/{id}]/radioreference` endpoint, in a *RadioReference*
  section of the web *Hunt* panel (county / SID inputs + a cross-reference
  button). (#660, #661)
- **Hunt: optional post-run auto-gain BER sweep.** A new `-auto-gain` flag (and
  an *Auto-gain* toggle + recommendations table in the *Hunt* panel) sweeps a
  set of front-end gains on each locked control channel after a run, decodes a
  short dwell at each, and recommends the gain that minimises the decode error
  rate (preferring one that locks, then lower gain to reduce front-end strain).
  Recommendations are reported and written to `gain-recommendation.json`;
  nothing is applied automatically. The web path is guarded by SDR exclusivity —
  gain control is wired only when the hunt holds a dedicated SDR, not the shared
  control SDR. (#660, #661)
- **Hunt: daemon survey persistence (NDJSON) + resume.** A survey run streams its
  classified carriers to a crash-safe NDJSON file when `persist_survey` is set,
  and on `resume` preloads the already-recorded frequencies so an interrupted
  web survey continues where it left off — matching toggles in the *Hunt* panel.
  Mirrors the CLI's `-survey-ndjson` / `-resume`, which remain for standalone
  runs. (#661)
- **Hunt: neighbour-frequency resolution for DMR Tier III / EDACS / Motorola.**
  Neighbour resolution was P25-only; DMR Tier III, EDACS and Motorola advertise
  adjacent sites by LCN and each already has a band-plan resolver, but their
  topology snapshots never applied it. They now resolve a neighbour's frequency
  via the configured band plan (an unresolved neighbour stays informational, no
  decode error), and a missing frequency is backfilled once a later observation
  resolves it. (#660)
- **SoapyRemote `stream_mtu` and `stream_window` config.** Both are SoapyRemote
  stream arguments (`remote:mtu` / `remote:window`) read client-side and
  forwarded to the server's `setupStream`, so neither could be set via the
  existing `soapy_remote.args` device kwargs. New per-endpoint knobs send them in
  `SETUP_STREAM` and size the client's in-flight flow-control credit from the
  same value so both ends agree — wired through the config (with range and
  `stream_window >= stream_mtu` validation), the daemon, the config builder, the
  web form, and the example config. (#664, #665)
- **TETRA Viterbi correction-depth histogram** (opt-in via
  `metrics.detailed_fec`). The control channel scores each recovered BSCH /
  SCH-HD burst's FEC correction depth decoder-independently (Hamming weight
  between received and re-encoded bits), surfaced through an optional observer
  hook the daemon wires to metrics only when the gate is set — so the default
  deployment does no per-burst work and carries no extra metric family. (#667,
  #648)

### Changed

- **P25 Phase 1 unhandled-TSBK diagnostic now dumps the raw payload (#376).**
  The first round of CBD/Mt Anakie field diagnostics confirmed the alias is
  *not* arriving as our control-channel working-model fragment (vendor opcode
  `0x15`, plain ASCII): zero `cc talker alias`/`fragment` lines fired on either
  system, even the cleanly-decoding one. The census instead named candidate
  opcodes (Motorola `0x90` opcode `0x16`; standard `0x15`/`0x16`/`0x30`) but we
  couldn't tell which carries the alias from the opcode number alone. The
  `p25: unhandled tsbk` census line now also prints the **numeric** opcode
  (`Opcode.String()` mislabels vendor opcodes with standard names), and a new
  `p25: unhandled tsbk payload` line logs the raw 8-byte payload hex, capped at
  8 samples per distinct `(MFID, opcode)` so a multi-block alias sequence can be
  captured and reversed against known RIDs / alias strings without flooding a
  busy control channel.
- **P25 Phase 1 control-channel talker-alias decode is now observable in the
  daemon log (#376).** A new field-tested system (CBD) locks and populates RIDs
  but shows blank talker aliases, with heavy TSBK CRC loss on the control
  channel. The CC alias path was fully wired but logged only at `Debug`, so a
  field tester running at the normal level saw nothing from it and couldn't tell
  whether aliases were arriving, completing, or riding an opcode we don't decode.
  Three diagnostics now surface at `Info`, mirroring the per-`(opcode,MFID)`
  census the Phase 2 voice composer already emits:
  - `p25: cc talker alias` once a radio's display name fully reassembles
    (promoted from `Debug`).
  - `p25: cc talker alias fragment` on the first vendor-TSBK alias fragment seen
    per source — so "fragments arrive but never complete" (CRC loss truncating
    the set before the 10 s reassembly window) is distinguishable from "no
    fragments at all".
  - `p25: unhandled tsbk` once per distinct `(MFID,opcode)` we don't dispatch
    (both vendor and standard namespaces) — a census that names any
    alias-bearing transport a given site uses that we don't yet decode. The bulk
    per-frame detail stays at `Debug`.
- **P25 Phase 1 voice-channel talker-alias events now carry the system name
  (#376).** The Phase 1 voice composer published `KindTalkerAlias` without the
  `System` field (unlike the Phase 2 composer and both CC paths), leaving the
  `/rids` System column and the `TALKER-ALIAS` message-log line systemless. The
  alias itself was unaffected.
- **Hunt: snap swept candidates to the channel grid and normalise occupied
  bandwidth** (#648). The sweep reports carriers at FFT-bin centres (~586 Hz
  bins at 2.4 MS/s), so a real carrier such as a P25 trunk at 440.125 MHz was
  reported a few hundred Hz off and 4800-baud decoders failed to lock off-centre.
  The candidate list now auto-detects the channel step (6.25 / 12.5 / 25 kHz)
  from the discovered frequencies and snaps each candidate to the raster within
  one FFT bin (without moving genuinely off-grid carriers; jittered detections of
  one channel merge, strongest SNR winning), and the reported occupied bandwidth
  is normalised to the nearest standard channel width so a 12.5 kHz channel reads
  as 12.5 kHz rather than a skewed "11.6 kHz". (#669)

### Fixed

- **P25 / DMR wideband voice glitches from an off-nominal DDC rate** (#550). Voice
  grants tapped off a wideband SDR run through a per-call DDC that rationally
  resamples toward 48 kHz, but at certain input rates the reduced L/M ratio trips
  the resampler's caps and lands a fraction of a percent off (e.g. ~48828 Hz from
  a 6.25 MS/s-derived bin). The control-channel path already clocked its symbol
  loops from `DDCBank.OutputRateHz`, but the voice path hardcoded the nominal
  48 kHz, so the symbol clock drifted off the true phase, periodically slipped,
  and the vocoder concealed each slip with an audible click. The virtual tuner now
  surfaces the DDC's actual fractional rate through the composer to every digital
  voice chain (P25 Phase 1 / Phase 2 / DMR), as a `float64` so the fractional part
  reaches the receiver's symbol clock intact. Physical SDRs (exact integer rate)
  are unchanged. (#668)
- **MPT 1327 false locks** (#648). The lock gate counted recognised Address
  codewords but ignored their identity, so a burst of inconsistent cross-protocol
  false parses — a different system prefix each time — could still reach the
  confirmation threshold and declare a false control-channel lock (and, being
  permissive lock-on-first 1200-baud FFSK, win the identify over P25 / DMR that
  failed off-centre). A lock now requires a minimum number of confirmation
  codewords (the production decoder sets two) that share a **consistent 7-bit
  prefix**: a matching prefix corroborates the run in progress, a new or changed
  prefix restarts the count. A genuine control channel repeats its prefix on every
  Aloha / AhoyChan and still confirms within a couple of codewords, while random
  parses never accumulate. (#669, #670)
- **TETRA control channels skipped during blind identify** (#648). A recognised
  TETRA control channel never cleared the 0.40 identify confidence gate ("best:
  tetra @ 0.16") because its 30 % FEC score depends on descrambling SCH/HD with
  the cell colour code, which is 0 under blind identify — wrong for any real cell,
  so every SCH/HD CRC failed and TETRA forfeited the whole FEC weight. The BSCH is
  always scrambled with colour code 0 and carries the cell's colour code, so blind
  identify now recovers it from the BSCH and descrambles SCH/HD with the recovered
  code, letting a real TETRA control channel earn its FEC score and clear the
  gate. The stale "descrambler will not lock" colour-code warning (the decoder
  auto-acquires it from the BSCH) is demoted to `Debug`. (#662)

## [v0.4.1] — 2026-06-13

This release is mostly **DMR voice quality, wideband / USRP capture, and the
live signal survey**, with the latest Tier III work finally surfaced in the web
console. On the voice side, the AMBE+2 decoder (DMR, P25 Phase 2, NXDN, dPMR,
TETRA) adopts the same post-synthesis voiced-phase dispersion / DC-block /
adaptive-smoothing chain IMBE already had, killing the metallic "tin can" buzz
(#644), and the Tier III 2-slot router degrades gracefully so a real call no
longer records an empty file when its embedded Link Control never decodes
(#644). USRP / SoapySDR sources now track the rate the device *actually*
delivers and expose a `master_clock_rate` knob (#550), and the polyphase
channelizer decodes cleanly at USRP-friendly rates like 6.25 / 5.0 MS/s where a
crude resampler fallback used to leave the control channel deaf (#550). The live
survey stops mislabeling tightly-packed DMR / TETRA carriers as analog AM /
wide-FM by isolating each candidate before classification and adding an opt-in
`-survey-deep` arbiter (#648). Live streaming closes the remaining quality gap
to recorded audio — band-limited resampling, deeper buffering, and an optional
real-time loudness AGC (#653) — the Plots panels rest on the control channel
instead of the DC-spike-buried SDR centre (#557), an out-of-the-box RTL-SDR now
defaults to tuner AGC so no-gain configs aren't ~17 dB deaf (#264), and the
config builder accepts decimal frequency / tone entry. Finally, the DMR Tier III
autoconfig learner and the live-hunt details are now visible in the web console
(#638).

### Added

- **Hunt features surfaced in the web console.** Recent hunt-family work that
  previously only reached the logs / raw event stream is now visible in the web
  UI. The *CC Activity* tab renders the DMR Tier III autoconfig learner's live
  `dmr.grant.observed` (LCN, timeslot, talkgroup, source) and
  `dmr.bandplan.learned` (base frequency, channel spacing, confidence) events —
  given clean typed DTOs instead of raw passthrough — plus the control-channel
  hunt lifecycle (`cchunt.progress` / `cchunt.failed`). The *Systems* tab shows
  the active DMR LCN→frequency band plan per system (`GET /api/v1/systems` →
  `dmr_band_plan`), whether operator-configured or learned over the air, with a
  "learned live" indicator. The *Hunt* tab now lists each candidate's capture
  report (control frequency, protocol, lock / skip reason) and the discovered
  sites with their control channels. (#638)
- **USRP/SoapySDR actual sample-rate tracking.** The `soapy_remote` driver now
  implements the `ActualSampleRate()` extension (via the `getSampleRate` RPC), so
  when a USRP coerces a requested rate to the nearest integer decimation of its
  master clock, GopherTrunk builds every per-channel down-converter and symbol
  clock from the *delivered* rate instead of the requested one — the same #402
  correction RTL-SDR already had. A new `sdr.soapy_remote[].master_clock_rate`
  option sets the USRP master clock in Hz so a target `sample_rate` lands on an
  exact divisor (e.g. `61_440_000` for a B210 to stream `6_144_000` cleanly).
  (#550)
- **Live-stream loudness AGC** (#653). An optional real-time envelope-follower AGC
  (`audio.live_loudness`, default off) on the digital decoded-PCM live tap, so the
  live stream tracks the loudness-normalized recordings instead of playing raw PCM.
  It wraps only the digital stream path — the on-disk WAV keeps its own EBU R128
  normalization and is never double-processed, and analog FM live audio (already
  shaped by the composer's AGC) is untouched.

### Changed

- **Plots view defaults to the control channel, not the SDR centre.** The
  Constellation, Symbol scope, Eye diagram, Mixer, Tuning and Histogram panels
  now rest their view on the system's control-channel frequency (resolved from
  config and reported per-SDR as `control_channel_hz` on
  `GET /api/v1/spectrum/devices`) whenever Hold is off and no call is active —
  so a freshly opened panel lands on a decodable channel clear of the centre DC
  spike instead of the useless SDR centre. An active call still takes
  precedence, and the view glides back to the control channel when the call
  ends; the Hold label shows *following call* or *on control channel*
  accordingly. SDRs with no control channel in their passband keep the previous
  centre default. (#557)
- **RTL-SDR defaults to tuner AGC when no `gain:` is configured** (#264). An
  out-of-the-box pure-Go RTL-SDR front end was left in an undefined gain state:
  bring-up ran the tuner init array (which leaves the R82xx VGA at +0 dB) but
  never called `SetGainMode`/`SetGain`, and the daemon only applied a gain when
  `gain:` was non-empty — so the VGA pin that AGC mode writes (+16.3 dB) never
  landed and the dongle ran ~17 dB low, failing to decode on every protocol while
  the same device worked in SDR# / PDW / TTT. Bring-up now establishes AGC (and
  the VGA pin) at open for every device, the daemon resolves an empty/omitted
  `gain:` to auto (-1) for both local and `rtl_tcp` devices, and the no-gain pool
  log is now an informational note rather than a deafness warning. A configured
  gain still overrides AGC.
- **Live survey: `-survey-deep` arbiter and a corrected AM / FM split** (#648).
  In a DMR / TETRA-dense band the blind classifier labeled almost every carrier
  `am` (including a 61 dB-SNR one) because the AM gate fired on envelope variance
  alone and never consulted the discriminator's `IFStd`. The gate now also
  requires low `IFStd` (little angle modulation), so a constant-envelope DMR
  carrier or an FM repeater with CTCSS falls to the FM split (`nbfm`) instead of a
  contradictory `am`; the cyclostationary baud search band was widened to ~20 kHz
  so TETRA's 18 kbaud line is searchable. A new opt-in `-survey-deep` flag also
  hands narrowband (≤25 kHz) analog-classed carriers to the authoritative siglab
  identify (lock-only), so a trunked control channel the blind classifier missed
  can still be found; the default survey stays fast.
- **Config builder accepts decimal frequency / tone entry.** A controlled
  `<input type="number">` reported an intermediate value like `131.` as empty,
  which reset the field to `0` and wiped the keystroke, so fractional frequencies
  and tones could not be typed. Fields with a fractional step now route to a
  buffered text input (modeled on `HzField`) that preserves mid-edit text and
  emits the parsed number; integer fields keep the native spinner. The float-backed
  audio tone Frequency, Tolerance, Magnitude threshold, and Squelch (dBFS) fields
  opt into the decimal path.

### Fixed

- **Wideband polyphase decode at 6.25 / 5.0 MS/s (USRP-friendly rates).** With
  `tuner_strategy: polyphase`, the channelizer's per-tap fine-tune resampler could
  silently emit the wrong sample rate at input rates whose bin rate is short on
  factors of two (e.g. 6.25 MS/s ÷16 = 390625 Hz): the exact 48 kHz ratio needed
  L=384, tripped `rationalRatio`'s L≤64 cap, and fell back to a crude integer
  decimator that produced 48828 Hz — a 1.7 % symbol-clock error that left the
  control channel deaf (P25 NID BCH uncorrectable). The fallback now picks the
  closest ratio under the caps (≤0.06 % error), and the wideband engine builds
  each receiver from the rate the bank *actually* emits (`Bank.OutputRateHz`)
  rather than a hardcoded 48 kHz. The DDC path was already exact and is unchanged.
  (#550)
- **Metallic "tin can" DMR voice: AMBE+2 synthesis parity with IMBE** (#644).
  Once the timeslot fix made Tier III speech intelligible it sounded
  metallic/buzzy — the AMBE+2 decoder (DMR, and also P25 Phase 2 / NXDN / dPMR /
  TETRA) synthesized voiced harmonics fully phase-coherently, radiating the
  classic buzzy MBE impulse train that IMBE had already fixed but AMBE+2 never
  adopted. AMBE+2 now runs the same three post-synthesis stages as IMBE:
  TIA-102.BABA §6.3 voiced-phase regeneration (per-harmonic dispersion scaled by
  the unvoiced fraction, so fully-voiced frames stay bit-identical), DC-block
  high-pass ahead of the AGC, and error-rate adaptive smoothing. The DMR voice
  chain now forwards its per-frame Golay corrected-bit count through the
  error-aware sink so smoothing engages on real bursts.
- **Empty DMR Tier III recordings: graceful slotRouter phase fallback** (#644).
  A real Tier III call could record a 0-byte `.raw` and a header-only 44-byte WAV:
  the 2-slot interleaved decoder's slotRouter dropped every voice superframe until
  a CRC-valid embedded Link Control named the call's talkgroup and bound its
  phase, but real outbound embedded signalling is still capture-pending, so on a
  carrier whose embedded LC never decodes the router bound nothing and the whole
  call was dropped. The router now degrades gracefully — a matching LC still binds
  (and can re-bind/correct) the slot, a foreign-talkgroup LC positively marks the
  other phase, and otherwise, after a short grace window with no LC, it falls back
  to the active slot's phase so audio records. Adds `lc_superframes` telemetry and
  a one-shot log when a call records via the fallback.
- **Live-vs-recorded audio quality gap closed** (#653). Live and recorded audio
  start from identical decoded 8 kHz / 16-bit / mono PCM, so the gap was entirely
  in the live transport + browser playback chain. Three fixes: a Blackman-windowed
  polyphase `SincResampler` (with continuous cross-chunk state) replaces linear
  interpolation in the WorkletSink, so a browser that rejects the 8 kHz
  AudioContext and falls back to 44.1/48 kHz no longer leaks aliasing images above
  the 4 kHz input Nyquist (first alias image now >40 dB down); the ring-buffer
  prime cushion grows 0.15 s → 0.25 s and the per-subscriber publisher channel
  64 → 256 frames so transient HTTP-write stalls no longer drop audio; and the
  optional live loudness AGC (below) tracks the normalized recordings.
- **Live survey: isolate dense carriers so DMR isn't mislabeled AM / wide-FM**
  (#648). On a tight carrier grid (DMR at 12.5 kHz) the survey mis-read nearly
  every signal as `am`/`wfm` with absurd occupied bandwidths (1000–2200 kHz) and
  then ran the analog-FM analyser on them, emitting bogus "active / CTCSS"
  results. Two coupled causes, both from never isolating a candidate from its
  neighbours: the occupied-bandwidth walk on the full-rate capture bridged
  adjacent carriers into one giant span, and modulation features extracted on the
  ±24 kHz channel let neighbours leak into the FM discriminator and destroy the
  baud line. The occupied-bandwidth estimate now takes a per-candidate neighbour
  spacing cap, and when a neighbour sits inside the wide channel the classifier
  isolates the carrier by decimating to ±12 kHz with the gentle resampler — so a
  digital carrier is correctly handed to the siglab identify → DMR.

## [v0.4.0] — 2026-06-12

This release is mostly about **DMR**. The receiver now decodes real
over-the-air control channels end-to-end (a symbol-AGC normalises the
matched-filter level so live BPTC/RS FEC actually passes), and Tier III
gained the pieces needed to follow a trunked system you didn't hand-configure:
**LCN band-plan autoconfiguration** learns the channel layout from the
wideband stream and hot-swaps it into the running control channel (#638), the
**CSBK opcode table** was rebuilt to the ETSI values so the control log stops
mislabelling broadcasts and grants (#640), and **data grants, MBC assembly,
inbound/ack CSBKs, and the embedded Reverse Channel** are now parsed (#643,
#646). On the voice side, Tier III recordings default to the 2-slot
interleaved decoder so two timeslots no longer splice into "encrypted"-sounding
noise (#644), and the voice-grant LCN is read from the correct bits so the
band-plan learner converges (#639). Audio playback gets **per-call loudness
normalization** (pure-Go EBU R128 / BS.1770, no ffmpeg) so calls from
different talkgroups play back at a consistent level (#627), plus an onset-clip
fix that caps per-frame AGC gain at the limiter knee (#635). The Plots hub adds
the **Mixer** plot — completing parity with OP25's six Plots tabs — and the
Spectrum panel gains a live **spectrum-analyzer trace** above the waterfall
(#634). Finally, a wideband **symbol-clock drift** fix lands the #402 effective
sample-rate correction on the Tier-II/III and virtual-voice DDCs so marginal
wideband captures decode (#633), and a live **Hunt** no longer dead-ends when
no control SDR is configured (#641).

### Added

- **Mixer plot — the last of OP25's Plots tabs.** A new **Mixer** plot
  (web `/plots/mixer`) renders the channel-baseband power spectrum two
  ways, completing parity with OP25's six Plots tabs (Spectrum,
  Constellation, Symbol, Datascope/Eye, Raw Mixer, Tuned Mixer).
  **Raw mixer** shows the channelized baseband as the receiver first sees
  it, with an amber marker on the carrier-offset estimate; **Tuned mixer**
  re-mixes that same window by the estimate so a locked loop pulls the
  carrier onto the centre line. Comparing the two shows the
  carrier-recovery correction at a glance. It runs the same parallel P25
  receiver the other scopes use (`WS /api/v1/diag/mixer`), reuses the
  wideband Spectrum FFT helper, and needs no new receiver tap — the tuned
  view is reconstructed from the loop's own `carrier_offset_hz`, so it
  works for both C4FM and CQPSK. See [docs/mixer.md](docs/mixer.md).
- **Spectrum-analyzer trace above the waterfall** (#634). The Spectrum panel
  now draws a live power-vs-frequency curve (the "spectrum analyzer" line, like
  SDR#'s top panel) above the existing waterfall. It reuses the same
  `SpectrumFrame` WebSocket stream, the same `[-100, 0]` dBFS range, and the
  same FFT-shifted bin→x mapping, so a peak in the analyzer lines up vertically
  with its streak in the waterfall below. It has its own 20 dB grid and a fill
  under the curve; hover and click-to-tune now work on either view.
- **Per-call loudness normalization (EBU R128 / BS.1770), pure Go** (#627). An
  optional integrated-loudness stage normalises finished recordings to a target
  LUFS — the same thing rdio-scanner gets from ffmpeg's `loudnorm`, but with no
  CGO and no ffmpeg, so the single-static-binary guarantee holds. A new
  `internal/dsp/loudness` package implements K-weighting biquads (designed by
  bilinear transform for any rate, so it works at 8 kHz voice), gated
  integrated-LUFS measurement, and 4× oversampled true-peak. It mirrors
  ffmpeg's two-pass linear `loudnorm`: measure the whole call, apply a single
  gain toward the target (clamped to ±`max_boost_db`), then back off so the true
  peak stays under the dBTP ceiling — pure gain, no compression, so within-call
  dynamics are preserved. Opt-in via `recordings.normalize.{enabled,target_lufs,
  true_peak_dbtp,max_boost_db}`, with an `apply_to` switch
  (`recording` / `distributed` / `both`) choosing whether the on-disk WAV, the
  outbound broadcast/stream MP3, or both get normalised. `gophertrunk decode
  -normalize` runs the same pass on a captured `.raw` for offline A/B. Off by
  default; see [docs/opt-in-features.md](docs/opt-in-features.md).
- **DMR Tier III trunk autoconfiguration (LCN band-plan learning)** (#638). A
  Tier III control channel transmits only a Logical Channel Number in voice
  grants, never an absolute frequency, so following voice previously required a
  hand-configured `dmr_band_plan` — without it every grant was dropped with
  `decode.error stage=no-bandplan`. GopherTrunk can now learn that band plan
  from its wideband acquisition: it observes the granted LCNs, detects which RF
  carriers key up in response across the dongle's IQ band, decode-confirms each
  is a live DMR burst, and fits the base frequency + channel spacing (snapped to
  the 6.25/12.5/25 kHz grids) for the whole LCN enumeration. The learned plan is
  hot-swapped into the running control channel — previously-dropped grants start
  following voice immediately — and written back to the config file
  (comment-preserving) so it survives a restart. Auto-enables for any planless
  wideband DMR Tier III system; a configured plan stays authoritative and
  disables learning.

### Changed

- **DMR Tier III CSBK opcode table rebuilt to the ETSI values** (#640). The
  Tier III CSBKO constants were wrong in several places, so the control-channel
  log mislabelled traffic and looked sparse next to dsd-neo: `0x28` was shown as
  "Preamble" but is really `C_BCAST` (broadcast announcements — every
  "opcode=Preamble" line was actually a broadcast), the voice-grant opcodes were
  swapped, and `SystemInfo`/`AdjStatus` were invented standalone opcodes that
  never matched real traffic. The table is rebuilt to ETSI TS 102 361-4 (names
  mirror dsd-neo: `C_ALOHA`, `C_AHOY`, `C_ACKD`, `C_BCAST`, the PV/TV/BTV/PD/TD
  grants, `C_MOVE`, real `Preamble` at `0x3D`); `C_BCAST` announcements
  (`Gen_Site_Params`, `Adjacent_Site`) now fold into the topology model, and
  every standard CSBK is logged at debug so the dominant Aloha beacon is visible
  instead of being silently consumed after lock. Pinned with real off-air
  vectors.
- **DMR Tier III data grants, MBC assembly, and inbound CSBKs** (#643, issue
  #626). Several CSBK opcodes and the multi-block-control data types were
  recognised but never acted on. Data-channel grants are now handled:
  `BTV_GRANT` (a broadcast *voice* call) is followed like a TV grant, while
  `PD_GRANT`/`TD_GRANT` (*data* grants) feed the LCN learner but are
  deliberately not published as voice grants, so the engine never retunes a
  voice device onto a packet channel. MBC header + continuation bursts are now
  buffered by a per-colour-code assembler that closes on the `LB=1` terminator
  and dispatches channel-grant opcodes (per-block BPTC FEC gates each block).
  Inbound/ack CSBKs — `C_RAND`, `C_AHOY`, and the `C_ACKVIT`/`C_ACKD`/`C_ACKU`/
  `NACK` family — are now parsed and surfaced at debug, with address offsets
  pinned to real off-air vectors. (USB packet-data payloads remain out of scope
  — this is a trunking scanner, not a data decoder.)
- **DMR embedded Reverse Channel decode** (#646). The earlier
  `C_RAND`/`C_AHOY`/`ACK` additions are uplink/ack CSBKs and were over-broadly
  labelled "reverse-channel"; that terminology is corrected to "inbound
  (uplink)". The actual Reverse Channel — which is not a CSBK but rides a burst's
  embedded-signalling field — is now decoded from the single-fragment
  (`LCSS=Single`) embedded field a downlink receiver actually observes, surfaced
  on the voice superframe and debug-logged. (The (32,11) FEC parity is preserved
  but not yet applied — capture-pending, like the EMB QR(16,7) and MBC CRC.)

### Fixed

- **Empty DMR Tier III recordings when the embedded LC doesn't decode
  (#644 follow-up).** The 2-slot interleaved decoder's slot router dropped
  *every* voice superframe until a CRC-valid embedded Link Control named the
  call's talkgroup. Real outbound embedded signalling is still capture-pending
  (the EMB FEC / 5-bit CRC constants are unvalidated against live air), so on a
  carrier whose LC never decodes the router bound nothing and the call recorded
  empty files — a 0-byte `.raw` sidecar and a header-only WAV. The router now
  degrades gracefully: a matching embedded LC still binds (and corrects) the
  slot, but after a short grace window with no LC it falls back to the active
  slot's phase so audio records. Foreign-talkgroup LCs still positively exclude
  their slot, so a call genuinely absent from the carrier still records nothing.
  The DMR decode-quality log now reports `lc_superframes` and logs once when a
  call records via the phase fallback, so LC-vs-fallback routing is visible.
- **Metallic "tin can" DMR voice — AMBE+2 brought to IMBE synthesis parity
  (#644 follow-up).** With the timeslot fix above the speech became
  intelligible but sounded metallic/buzzy ("like someone speaking into a tin
  can"). The AMBE+2 decoder (DMR, and also P25 Phase 2 / NXDN / dPMR / TETRA)
  was synthesizing voiced harmonics fully phase-coherently, which radiates a
  buzzy impulse train — the classic from-scratch-MBE artifact. It now adopts
  the three post-synthesis stages the IMBE decoder already had: §6.3
  voiced-phase regeneration (per-harmonic phase dispersion scaled by the
  unvoiced fraction — the de-buzz), a DC-removal high-pass ahead of the AGC,
  and error-rate adaptive smoothing. The DMR voice chain now also forwards the
  per-frame Golay corrected-bit count to drive that smoothing. Clean,
  fully-voiced audio is bit-identical to before (dispersion and smoothing are
  inert on a clean, voiced frame), so only the metallic timbre changes.
- **Garbled audio on recorded DMR Tier III voice (#644).** A DMR carrier is
  2-slot TDMA, so the demodulated dibit stream interleaves both timeslots'
  bursts. The per-call voice chain decoded it with the single-slot decoder,
  which locks burst A correctly but then grabs the *other* timeslot's bursts
  for B–F — splicing two unrelated calls' AMBE frames into each superframe, so
  the vocoder rendered structured noise that "sounds encrypted" on a clear
  channel. DMR Tier III now defaults to the 2-slot interleaved decoder, which
  pulls each call's own timeslot out and routes it to its talkgroup by the
  embedded Link Control. The decoder auto-detects the on-air same-slot cadence
  — 264 dibits (no inter-burst CACH) vs 288 (a CACH precedes each burst on
  outbound air) — by locking onto the cadence whose bursts B–E reassemble a
  CRC-valid embedded LC. `dmr_interleaved_voice` is now a tri-state override
  (unset = protocol default — on for Tier III; set true/false to force).
- **DMR Tier III voice-grant LCN read from the wrong bits (#639).** The
  TalkGroup/Private voice-grant CSBK parser used the wrong field layout: it
  read the "LCN" from the last payload octet — which is actually the low byte
  of the 24-bit source (subscriber) address — so the decoded Logical Channel
  Number changed with every transmitting radio, even on a system with only a
  couple of fixed channels. The grant content actually leads with a 12-bit
  Logical Physical Channel Number (LPCN), followed by the target then source
  addresses (per ETSI TS 102 361-4, cross-checked against the dsd-neo decoder).
  The parser now extracts the LPCN from the leading payload bits and the
  addresses from their correct octets, and the LCN is widened from 7 to 12 bits
  end-to-end (parser, band-plan resolver, autoconfig learner, config, events).
  This unbreaks both operator-supplied `dmr_band_plan` tables and the LCN
  autoconfiguration learner, which previously never converged because it saw a
  fresh LCN on every call. (Encrypted/Emergency are no longer reported from a
  DMR grant — those are signaled in the voice Link Control, not the channel
  grant CSBK.)

- **DMR now decodes real over-the-air control channels.** The DMR receiver
  previously decoded only the zero-offset, level-matched synthesized fixtures;
  on a real SDR capture every BPTC(196,96) payload came back uncorrectable, so
  Tier III never locked ("dmr/tier3: BPTC uncorrectable") and Tier II reported a
  bogus instant lock then nothing. Root cause: the unit-energy RRC matched
  filter has a DC gain of ~3.1, so the 4-level symbol centres landed far above
  the slicer's fixed thresholds and the inner symbols collapsed onto the outer
  rails. A symbol-AGC now normalises the matched-filter level to the slicer's
  scale (the same calibration the P25 Phase 1 receiver runs, #275), so a real
  Tier III TSCC decodes cleanly and locks end-to-end through the wideband
  channelizer.
- **DMR CSBK CRC was rejecting every real burst.** The CSBK CRC-16 used the
  wrong convention (init 0xFFFF, stored as the bitwise complement), which only
  validated the synthesized round-trip fixtures. Real ETSI Tier III CSBKs use
  CRC-CCITT (init 0x0000) XORed with the mask `0x5A5A` (TS 102 361-1 §B.3.11);
  cleanly-decoded off-air Aloha and Preamble bursts now pass and are pinned by a
  regression fixture.
- **DMR Tier III C_ALOHA opcode corrected** from `0x04` to `0x19` (the value
  real TSCC beacons carry), so the control channel locks on the Aloha CSBK.
- **DMR Tier II "instalock".** Tier II declared `cc.locked` on any slot-type
  Hamming decode, so a false sync match on noise forged an instant lock (often
  `cc=15`) that then produced nothing. The lock is now gated on a FEC-validated
  Voice LC Header (BPTC + RS both pass), mirroring Tier III's lock-after-CRC
  discipline.
- **Symbol-clock drift on wideband DMR/voice down-converters (#402)** (#633).
  The control-channel decoder already built its DDC from the SDR's *actual*
  delivered sample rate (the RTL2832U quantizes a requested rate to its
  resampler divisor, so a non-exact-divisor rate like 2.048 MS/s streams
  slightly off), but the wideband Tier-II/III engine and the wideband virtual
  voice taps still built their per-channel DDCs from the raw configured
  `sample_rate`. On a non-exact rate the tap output wasn't truly 48 kHz, so the
  DMR receiver's Mueller-Müller clock carried a standing error and the payload
  failed BPTC/RS FEC even though sync still correlated — the field symptom of
  "signals look weak / won't decode" on a wideband RTL-SDR that decodes fine in
  SDR#/SDRTrunk. The effective-rate correction now applies to both wideband
  paths; exact-divisor configs (2.4 / 0.96 MS/s) are unchanged.
- **Onset clipping from runaway AGC gain on quiet call starts** (#635). The
  IMBE/AMBE AGC computed `gain = TargetPeak/envelope`, bounded only by a large
  `MaxGain`. When a call opened with a near-silent frame the envelope seeded low
  and the next louder frame — or a frozen bad-frame replay reusing the held gain
  — was multiplied by a huge gain, slamming every sample into the soft limiter
  (field captures showed ~65 % of samples clipped on short onsets). `AGC.Apply`
  now adds a one-sided, content-aware ceiling that never amplifies a frame's own
  peak past the limiter knee, so ordinary frames and the attack/release dynamics
  are unchanged — it can only reduce clipping, and it applies to frozen replays
  too.
- **Live Hunt failed when no control SDR was assigned** (#641). A live hunt
  always bailed with "no SDR with an IQ broker available for a live hunt"
  whenever no trunked system was configured — the auto-select path only
  considered a spare voice SDR or the control SDR, but the control serial is
  only set when a trunked system exists. The natural discovery workflow (point
  one SDR at the air to find an unknown system, nothing configured) left that
  serial empty and dead-ended even with a usable SDR sitting idle, as did
  wideband-only rigs and dongles with a blank USB serial. A last-resort fallback
  now picks any pooled SDR that has a broker.
- **`iq-capture` now warns on dropped chunks.** A non-zero `drops` count means
  the capture writer fell behind the IQ stream (subscriber buffer full) — not an
  SDR overflow — leaving time gaps that corrupt downstream decode. The finish
  log now emits a `WARN` explaining the cause and the remedy (faster storage,
  lower sample rate, fewer concurrent sinks) instead of only printing the count.

## [v0.3.9] — 2026-06-11

This release is about **live audio you can actually hear** and a friendlier
config path. Digital voice (P25 Phase 1/2, DMR, NXDN) now reaches the web
console's live stream — previously only analog FM and disk recordings carried
sound, because digital frames are decoded to PCM only inside the recorder and
were never fanned to the live consumers (#598). The browser player is rebuilt
around an **AudioWorklet ring buffer** with a single continuous resampler, so
chunk boundaries no longer click and network jitter degrades to brief silence
instead of audible re-syncs (#629); the live `AudioContext` is pinned to 8 kHz
to match the recorded WAV quality (#598), and a host-speaker echo from
double-playing digital audio is gone (#598). The **Config Builder and web UI**
get a shared RadioReference login (the SOAP app key is built into the binary),
an in-place config-file browser and hot-swap/reload, and a smoother connect
screen (#621). On the DSP side: the P25 IMBE vocoder is aligned with the
OP25/JMBE references (pitch-dependent prediction, DC block, adaptive smoothing),
over-driven P25/DMR voice clipping is fixed with a soft limiter and per-call
voice telemetry, `voice_taps` is uncapped, Airspy real-ADC IQ conversion and
rate/probe-gain bugs are fixed (#454), and a DC-spike-avoidance LO offset lands
on the live control path (#402). Three new **Basics / Intermediate / Advanced**
guides extend Getting Started (#620).

### Added

- **Three-level learning path: Basics / Intermediate / Advanced guides** (#620).
  The Getting Started funnel now continues past your first recorded call with
  three pages that build level by level and cross-link out to the existing
  detailed docs rather than duplicating them: `guide-basics` (everyday web-
  console operation, playback, talkgroup tidy-up, bookmarks/radio IDs, scanning,
  feed sharing), `guide-intermediate` (`config.yaml`, multi-dongle pools, alias
  files, paging/tone alerts, Hunt, network access/hardening), and
  `guide-advanced` (TUI cockpit, signal scopes, SigLab, the other receivers,
  remote SDRs, and the REST/gRPC/Prometheus/rigctld APIs). Wired into the
  Getting Started nav group.
- **Config Builder & web UI: RadioReference login, config browsing/hot-swap,
  connect UX** (#621). The Config Builder gains a shared RadioReference login
  block (username + password only — the SOAP app key is built into the binary),
  surfaced inline in the "Add from RadioReference" modal so browsing prompts for
  credentials in place, plus a prominent "Browse…" picker that lists available
  and previously-created configs with folder, modified time, size, and validity.
  The web console pre-fills the server URL with the device's own origin (with a
  "Use this device's address" button), offers to open the Config Builder when
  the daemon has no config file, and adds a Settings config-file picker that
  hot-swaps the active config via live Reload or full Restart. Backed by a new
  gated `POST /api/v1/config/activate`.
- **DC-spike-avoidance LO offset on the live control channel** (#402). A new
  opt-in `dc_avoid` flag (with optional `dc_avoid_offset_hz`) on a control
  SDR's `sdr.devices` entry tunes the hardware LO a fixed offset *below* the
  control-channel frequency and mixes the channel back to baseband in the
  down-converter — so the live decode runs off the front-end DC spur, 1/f
  noise and its own I/Q-imbalance image, the same offset tuning SDRTrunk/OP25
  use. This closes the live-vs-replay gap on marginal urban sites where the
  channel decodes cleanly off-channel (in replay) but accumulates TSBK-CRC /
  NID-BCH failures live with the channel sitting at zero-IF. Off by default
  (`dc_avoid: false`); `dc_avoid_offset_hz: 0` auto-selects `sample_rate/4`.
- **Per-call voice audio-quality telemetry** for diagnosing decode/synthesis
  problems. The IMBE decoder accumulates a `VoiceStats` summary (pitch/f0,
  harmonic count, voiced fraction, AGC gain, output peak/RMS/crest, and
  limited-sample percentage); the recorder logs it per call at `DEBUG` as
  `recorder: voice audio quality`, escalating to `WARN` when the output is
  clipping. `gophertrunk decode` prints the same summary for a captured `.raw`
  sidecar (`-stats`, default on), so audio quality can be triaged offline.

### Changed

- **`voice_taps` is no longer capped at 8.** A wideband dongle can now host any
  number of concurrent virtual voice DDC taps. Each tap runs its own DDC so CPU
  scales roughly linearly per tap; the daemon logs a warning above 16 so an
  accidental large value doesn't quietly peg a core.

- **P25 IMBE voice quality: align the vocoder with the OP25 / JMBE reference
  decoders.** Three changes ported after comparing GopherTrunk's mbelib-lineage
  decoder against OP25's `imbe_vocoder` and DSheirer's JMBE:
  - **Pitch-dependent spectral-amplitude prediction.** The cross-frame
    log-amplitude prediction coefficient is now a function of the harmonic
    count L (`ρ = 0.4` for L≤15, `0.03·L−0.05` for 16–24, `0.7` for L≥25),
    matching both reference decoders, instead of a fixed 0.65. The old constant
    over-predicted high-pitch / low-L frames and smeared the previous frame's
    spectral envelope onto them — the main remaining cause of poor female-voice
    intelligibility. Measured on the field capture, the female call's crest
    factor rose from ~7.8 to ~9.8 (closer to the raw synthesizer's ~11).
  - **DC-removal high-pass** on the synthesized PCM (first-order, pole 0.99),
    matching OP25's `dc_rmv.cc`, so a DC offset no longer wastes AGC headroom.
  - **Adaptive smoothing** driven by the channel FEC error rate (ported in
    spirit from JMBE `IMBEModelParameters`): a running ε_R estimate caps
    error-induced amplitude spikes, reclaims obviously-voiced harmonics, and
    mutes hopeless frames (ε_R > 0.0875). Thresholds are expressed relative to a
    running local-energy estimate (vs the reference's fixed-point absolute
    constants) and are inert on a clean channel, so well-decoded audio is
    byte-identical. The P25 Phase 1 chain now threads the per-frame corrected-bit
    count to the decoder (`voice.ErrorAware`); other protocols are unaffected.

### Fixed

- **Airspy R2 / Mini received nothing — web spectrum showed only the DC spike.**
  The Airspy is a real-sampling receiver whose host-side converter decimates by
  two, so the delivered IQ rate is half the programmed device rate. The driver
  was sending the requested rate straight to firmware, so `sample_rate:
  3_000_000` ran the device at 3 MSPS (1.5 MHz of IQ) while the rest of the
  pipeline assumed 3 MHz — a 2× mismatch that mis-tuned every decoder. The
  driver now programs the device at twice the requested IQ rate, so a 3 MHz IQ
  rate correctly selects the Airspy's 6 MSPS mode.
- **`gophertrunk sdr list --probe` reported empty gains `[]` for Airspy and
  HackRF.** The per-device open path rebuilt `sdr.Info` without copying the gain
  ladder that `Enumerate` populates, so the probe (which reads the opened
  device) saw nothing. Both drivers now carry the ladder onto the opened device,
  matching RTL-SDR. Gain values are in tenths of dB (Airspy 0–500, HackRF
  0–560); see `config.example.yaml`.
- **Live audio failed silently.** When `audio.enabled: true` but the sink
  couldn't open, the player logged a single WARN and ran headless, leaving no
  signal as to why nothing played. Backend init failure now logs at ERROR with
  the reason, and the cause is surfaced via `GET /api/v1/audio`
  (`backend_error`) and `Stats`. The direct-ioctl path (`device:
  "ioctl:hw:C,D"`) now explains that it pins S16_LE mono at the configured rate
  — which onboard codecs typically reject — and points at the default libasound
  device, which resamples.
- **P25/DMR voice was over-driven into clipping (robotic, "female voices
  especially awful")**. The MBE-family AGC carried a `MinGain` floor of 10,
  forcing at least a 10× gain onto a synthesizer that already emits near-int16
  scale (raw crest factor ≈11). Every loud frame was amplified ~10× straight
  into the hard clip, flattening speech to a crest factor of ~3 and distorting
  it — independent of the demod, which decoded cleanly. The floor is now 0.05
  (the AGC may attenuate), the hard clip at ±32767 is replaced with a tanh soft
  limiter, and the target/attack were retuned against the field C4FM + CQPSK
  captures. Decoded female speech now lands at crest ≈7.8 with ~0.02% limited
  samples — in line with a reference decoder (Trunk Recorder) on the same
  speaker — instead of crest ≈3.3 railed at full scale.
- **Daemon panic on shutdown** finalizing a dormant post-segment recording
  session: `WavWriter.Close` dereferenced a nil writer. The session close now
  guards a nil WAV (and `WavWriter.Close` is nil-safe), so Ctrl-C no longer
  panics and the final recording is finalized cleanly.
- Airspy R2 / Mini now decode correctly (#454). The driver treated the
  receiver's real ADC stream as interleaved I/Q, producing a huge quadrature
  imbalance (~78°) and no image rejection (~3 dB) so nothing locked. The Airspy
  is a real-sampling front end: its samples are now converted to complex
  baseband on the host (Fs/4 translation + half-band Hilbert, decimate-by-two),
  matching libairspy's IQ modes and restoring image rejection (~70 dB).
- **Live digital voice was silent in the web console** (#598). P25 Phase 1/2,
  DMR, and NXDN reach the composer as raw vocoder frames (`WriteRawFrame`),
  which only the recorder consumes and decodes to PCM; the tone-out detector,
  live player, and audio publisher implement `WritePCM` only, so they never saw
  digital audio and the live stream stayed silent even while disk recordings
  were audible (analog FM worked because its chain calls `WritePCM` directly).
  The recorder now exposes a decoded-PCM tap that the daemon fans to the live
  consumers, so every decoded sample is streamed exactly once.
- **Live stream dropped PCM when the publisher had no cached grant** (#598).
  `AudioPublisher.WritePCM` discarded all PCM unless it held a cached grant for
  the device serial, but that cache is fed by an events-bus subscription that
  drops events into a momentarily-full channel — so on a busy control channel
  (especially with multiple voice taps) a missed `CallStart` silenced the live
  stream for the call. Neither stream consumer reads the frame's grant, so the
  requirement was pure fragility; PCM now fans to unfiltered and serial-only
  subscribers regardless of grant state (talkgroup-filtered subscribers still
  need a grant to evaluate their predicate).
- **Host-speaker echo from double-playing digital audio** (#598). Digital voice
  was being played on the host speaker twice, producing an echo; the duplicate
  playback path is removed.
- **Web audio player rebuilt on an AudioWorklet ring buffer** (#629). The
  per-chunk `AudioBufferSource` scheduler is replaced with an AudioWorklet that
  reads from a ring buffer through a single `LinearResampler` spanning the whole
  stream — so non-default analog rates (and browsers that reject the 8 kHz
  context) no longer reintroduce per-chunk boundary artifacts, and underruns
  emit brief silence instead of an audible cursor re-align under network jitter.
  Falls back to the previous scheduler where AudioWorklet is unavailable.
- **Live `AudioContext` pinned to 8 kHz** (#598) to match the recorded-WAV
  quality, so the live stream and recordings sound the same.

## [v0.3.8] — 2026-06-10

This release adds a pure-Go **LoRa / LoRaWAN receiver** (#586) and hardens the
daemon and several hardware paths. One SDR is channelized into parallel LoRa
sub-channels and decoded through the full PHY (dechirp/FFT, Gray/de-interleave/
Hamming FEC, de-whitening, CRC) with SF7–SF12 auto-detection; LoRaWAN 1.0.x
frames are MAC-decoded and, with operator session keys, MIC-verified and
decrypted, persisted to `lora_log`, served at `GET /api/v1/lora/frames`, and
rendered on a new `/lora` panel. Recording gains an opt-in
`recordings.skip_encrypted` flag (#607). The daemon **never stops silently**
anymore — component panics are recovered and logged, and a soft memory limit
plus a runtime heartbeat bound and surface the process footprint (#606, #492).
On the fix side: P25 IMBE female-voice intelligibility and a high-pitched
recording onset (#605), Airspy USB initialisation (#454), HackRF
interface-claim on macOS (#511), live-audio playback in Chrome via Web Audio
(#598), and the symbol-domain scopes now default to the control SDR (#402).

### Added

- **Skip recording encrypted calls** (#607). A new opt-in
  `recordings.skip_encrypted` flag suppresses WAV/raw files for calls the
  operator can't decode (default `false` keeps recording everything; the call
  log still notes encryption). The recorder gates at call start when a
  control-channel grant already flags encryption (P25 Phase 1, DMR, NXDN,
  EDACS, TETRA), and mid-call when encryption only surfaces on the traffic
  channel (P25 Phase 1 LDU2 Encryption Sync, or a P25 Phase 2 compressed
  grant) — the in-progress files are closed and deleted and no `CallComplete`
  is published, so the partial never reaches the upload feeds. Wired through
  the YAML config, the settings PATCH API + YAML writer, the TUI settings
  panel, and the web Config Builder.
- **LoRa decoding** (#586). A new pure-Go, zero-CGO LoRa receiver decodes the
  LoRa physical layer (chirp dechirp/FFT demodulation, preamble/sync/SFD
  acquisition with carrier-offset and timing recovery, Gray/de-interleave/
  Hamming FEC, de-whitening and CRC) with spreading-factor auto-detection
  across SF7–SF12 and bandwidths 125/250/500 kHz. One SDR is split into
  several parallel LoRa sub-channels via the tuner channelizer/DDC bank.
  LoRaWAN 1.0.x frames are MAC-decoded and, when operator session keys are
  supplied, the MIC is verified and the payload decrypted (no key recovery).
  Configure under `lora.channels`; decoded frames persist to the `lora_log`
  table, are served at `GET /api/v1/lora/frames`, and render live on the new
  `/lora` web panel.

### Changed

- **Daemon never stops silently + bounded memory footprint** (#606, #492). A
  live run could halt mid-decode with no shutdown/fatal/panic line — the
  hallmark of an external SIGKILL (OS memory-pressure killer) or an unrecovered
  goroutine panic. The daemon now installs a deferred `log.Recover()` panic
  guard on the component spawn path, the daemon-run and IQ-capture goroutines,
  the rtltcp reader, the iqtap fanout, and all four composer voice chains, so a
  panic becomes a logged ERROR + clean shutdown instead of a process kill. A
  soft memory limit is set at startup (`GOMEMLIMIT` → `diagnostics.memory_limit_mb`
  → ~70 % of physical RAM), a periodic runtime heartbeat
  (`diagnostics.heartbeat_seconds`, default 60 s) logs uptime/goroutines/heap so
  a leak or pre-kill footprint is visible in the timeline, and `net/http/pprof`
  is available behind `GOPHERTRUNK_PPROF`.

### Fixed

- **P25 IMBE female-voice intelligibility + high-pitched recording onset.**
  Follow-up to the §6.3 voiced-phase regeneration (#600). Two corrections to
  match the reference imbe_vocoder/mbelib so female (high-pitch, mostly-voiced)
  speech is no longer rendered as noise:
  - The dispersion is now a **bounded per-frame offset on a coherent phase
    memory** (`PHIl = PSIl + offset`) instead of a full `[−π,π)` step
    *accumulated* into the phase memory every frame. The old random walk
    decorrelated the upper harmonics into noise within a few frames; a
    confound-free A/B on a sustained 220 Hz vowel shows the reference model is
    ~12 % more periodic/harmonic.
  - The offset magnitude is **scaled by the unvoiced-harmonic fraction**
    (`numUv/L`), so a mostly-voiced frame gets near-zero dispersion (stays
    intelligible) while noise-dominated frames still de-buzz. Fully-voiced
    frames are now synthesized coherently, as in the reference.
  - The **idle-carrier mute now engages on the first frame at a transmission
    onset** instead of leaking one ~352 Hz buzz frame. ~60 % of field calls
    opened with that leaked frame — the "highly pitched beginning" a user
    reported (worse on CQPSK, whose warm-up is longer). The run-threshold guard
    that protects a lone idle frame *inside* speech is unchanged.
- **Airspy device initialisation** (#454). The pure-Go Airspy driver's USB
  vendor request opcodes were systematically wrong; they now match libairspy's
  `airspy_commands` enum (`SET_SAMPLERATE`=12, `GET_SAMPLERATES`=25, gain/freq
  opcodes, …). `SetSampleRate` is now a vendor-IN transfer with the rate carried
  in `wIndex` (the firmware NAK'd the previous vendor-OUT, surfacing as "set
  sample rate failed: protocol error"), the bogus host-side `SET_SAMPLE_TYPE`
  command is gone, and the bias-tee uses a GPIO write. `Open` resets the
  receiver and retries on transient device-gone errors, and the device pool now
  normalises `AIRSPY SN:` / `airspy_sn:` serial aliases when matching config
  hints. Includes opt-in real-hardware tests (`make test-airspy-real`) and a
  Windows WinUSB interface-recipient/associated-interface control-transfer
  fallback.
- **HackRF now claims its USB interface on macOS** (#511). The pure-Go USB
  backend enumerated a HackRF but failed to claim interface 0, returning
  `kIOReturnUnsupported` — `ClaimInterface` passed the device user-client type
  ID, but an interface service requires `kIOUSBInterfaceUserClientTypeID`. The
  interface path now uses the interface UUID (the device-open path keeps the
  device UUID).
- **Live audio plays in Chrome via Web Audio** (#598). The "Tap to enable
  audio" button did nothing in Chrome on macOS: the hidden `<audio>` element
  couldn't reliably play the daemon's open-ended chunked "infinite WAV", and
  the failure was swallowed. The web player now reads the stream with `fetch()`
  and a Web Audio pipeline (a `PcmFramer` reassembles the WAV header and int16
  samples across chunk boundaries, scheduling gapless buffers through a jitter
  buffer with underrun resync), surfaces failures as a visible "Audio failed —
  tap to retry" chip, reads the sample rate from the WAV header, and sends the
  bearer token so auth-gated daemons work. No backend change.
- **Symbol-domain scopes default to the control SDR** (#402). The Eye Diagram,
  Symbol Scope, Tuning, and Histogram panels each defaulted to the first
  enumerated device, which on a multi-SDR rig is often an idle voice/aux
  dongle, so a panel opened during active control-channel decode showed
  nothing. A new `defaultSymbolDevice()` prefers the control-role device
  (falling back to the first entry) for the initial selection in all four
  panels. Also adds an MMR City clean-decode regression fixture
  (`TestReplayMMRCityDecodesCleanP25`) that guards the C4FM path against
  future regressions.

## [v0.3.7] — 2026-06-09

This release sharpens **P25 Phase 1 voice** and consolidates the **install
layout**. The decoder now error-corrects the outer Reed-Solomon layer on the
LDU1 Link Control and LDU2 Encryption Sync (#589), so a real-air capture's
talkgroup gating stops fragmenting calls into ~1 s files; on top of that the
IMBE vocoder gets TIA-102.BABA §6.3 voiced-phase regeneration to kill the
"robotic" buzz (#600), idle-carrier dead keys are muted (#599), and the LDU1
Link Control octet layout is corrected (#596). The Windows installer now
prompts for **one data folder** that holds config, recordings, IQ, exports,
the database, logs, and all three browser consoles, with config path fields
resolved relative to the config directory so a single portable config works on
any OS (#602). The **Plots** scopes gain a selectable C4FM constellation (IQ
ring vs. soft levels), an auto-detected demod mode, and a channel-step nudge
(#557, #583); the **signal survey** becomes a saved, offline-decodable artifact
(#590, #592); RadioReference picks up a built-in app key and a "verify
subscription" check (#603); browser audio now seeks on Safari (#598); and the
same talkgroup is no longer shown as two duplicate "Active calls" (#593).

### Changed

- **Single data root for installed builds** (#602). The Windows installer now
  asks for one data folder (default `Documents\GopherTrunk`) instead of two,
  and lays out `config/ recordings/ iq/ exports/ data/ logs/ web/` beneath it;
  the executable still installs to Program Files. `config.example.yaml` ships
  config-relative paths (`../recordings`, `../data`, …) and `config.Load` now
  anchors every relative path field to the directory holding `config.yaml`
  (absolute and empty paths are unchanged), so one portable config lands under
  the operator's chosen root on any OS. `gophertrunk run -web` resolves the
  bundled consoles under `<DataRoot>/web` via `GOPHERTRUNK_HOME` /
  `GOPHERTRUNK_CONFIG`.

### Added

- **RadioReference built-in app key + subscription verify** (#603). A developer
  app key can be injected at build time (`-ldflags`, kept out of source) and is
  resolved explicit > env > built-in, so browse/import works without each user
  supplying a key; the subscriber's username/password (which gate premium) are
  sent per request from the edited config in both the web and TUI Config
  Builders. A new **Verify subscription** action (web button / TUI `[V]`,
  `POST /api/v1/config/rr/verify`) reports premium status and expiry inline.
- **Constellation: selectable C4FM display (IQ ring vs. soft levels)** (#557).
  C4FM is constant-envelope FM with no complex symbol constellation, so the
  Symbols view previously plotted its soft decisions as a thin horizontal line
  on the real axis. A new **Display** control (shown for C4FM) chooses between
  the **IQ ring** — the raw constant-envelope circle most operators expect,
  now the default — and the legacy **Soft levels** line. CQPSK is unchanged.

### Fixed

- **P25 Phase 1 calls no longer fragment from un-error-corrected control
  words** (#589). LDU1 Link Control (talkgroup/source) and LDU2 Encryption Sync
  (ALGID/KID) were decoded with only the inner Hamming(10,6,3) layer; the outer
  Reed-Solomon codes were never corrected, so residual bit errors corrupted the
  talkgroup the recorder's gating relies on — dropping ~71% of voice frames,
  splitting calls into ~1 s files, and producing garbage ALGIDs. The framing
  layer now does bounded-distance RS decoding over GF(2⁶) (Berlekamp-Massey +
  Chien + Forney) for RS(24,12,13), RS(24,16,9) and RS(36,20,17), run as the
  outer layer in `ParseLinkControl` / `ParseEncryptionSync`; when a word is
  RS-uncorrectable the composer leaves `tg=0` so the boundary tracker inherits
  the last match instead of ending the call on a mis-decode.
- **P25 voice no longer sounds robotic / "wrongly pitched."** The pure-Go IMBE
  synthesizer generated every voiced harmonic with a fully phase-coherent
  model (each harmonic locked to an exact multiple of the fundamental, frame
  after frame). Perfectly coherent harmonics re-align once per pitch period and
  radiate a buzzy impulse train — the classic "robotic" vocoder artifact that
  made decoded voice sound markedly worse than the reference imbe_vocoder (e.g.
  OP25), affecting both the C4FM and CQPSK demod paths since they share the
  vocoder. The decoder now applies TIA-102.BABA §6.3 voiced-phase regeneration:
  the voiced upper harmonics (l > L/4) accumulate a per-frame random phase step
  (drawn from a separate seeded source so the unvoiced-noise stream — and any
  output that depends on it — is byte-identical, and the decode stays
  deterministic), matching the reference's
  `if (i > num_harms_max/4) ph_mem[i] += rand()`. Low harmonics stay coherent
  so pitch and formant structure are preserved.
  Measured on the reported real capture, the mean voiced-frame crest factor
  dropped from ~3.2-3.4 to ~2.4 — the impulse-train peakiness behind the buzz.
  AMBE+2 (Phase 2) is unchanged.

- **P25 Phase 1 voice no longer plays a buzzy tone at the start/end of
  recordings (and on dead keys).** An unmodulated/idle voice-channel carrier —
  the brief moment before a talker actually speaks, the tail after they release,
  and whole carrier-only "kerchunk" grants — produces a near-constant C4FM dibit
  stream that the IMBE FEC resolves to a degenerate low-`b_0` frame (fundamental
  ~350 Hz, the highest-pitch / fewest-harmonic corner of the codebook). The
  vocoder was synthesizing that as an audible ~350 Hz buzz, so recordings opened
  with a tone "before the voice started" and dead-key grants were pure buzz.
  Field captures confirmed real speech never sustains that `b_0` corner across
  frames, so the IMBE decoder now mutes a *run* of these idle-tone frames to
  silence (reusing the existing silence-frame fade), while leaving an isolated
  low-`b_0` voiced frame untouched. The fix is in the decoder, so both recorded
  WAVs and live audio benefit. Regression tests decode real captured `.raw`
  sidecars (an all-tone dead key, and a call whose voice is bracketed by tone
  runs) to pin the behavior.
- **P25 Phase 1 voice recordings no longer fragment into tiny per-LDU files.**
  A single continuous transmission was being chopped into many ~1-second
  recordings (each `.raw` an exact multiple of one LDU), because the embedded
  LDU1 Link Control was reading the talkgroup from the wrong content octets.
  For the Group Voice Channel User LCO (0x00) the talkgroup lives at octets 4-5
  and the source at 6-8 (TIA-102.AABF); the decoder was reading the talkgroup
  from octets 2-3, so it always came back as the constant service-options byte
  (0x0400 = 1024) while the real talkgroup landed inside the misread source
  field. With the in-band talkgroup never matching the granted talkgroup, the
  voice composer's foreign-talkgroup gate ended every call after ~2 LDU1s and
  the control channel immediately re-granted, spawning a fresh file each time.
  The Link Control octet layout is corrected (the FEC was always fine) and a
  regression test now pins the absolute octet positions. As defense-in-depth,
  the foreign-talkgroup gate now requires the *same* foreign talkgroup across
  its debounce window so a lone RS-aliased mis-decode can't end a call.

### Added

- **Signal survey — save it, decode it, run it offline.** Follow-up to the live
  signal survey: the classified inventory is now a real artifact, written to
  `survey.json`/`survey.csv` by the CLI, served by `GET /api/v1/hunt/survey`
  (`?format=json|csv`, `+ /{id}/survey`), and downloadable from the web Hunt
  panel. Pages a survey decodes are published to the events bus and the pager
  log like a live receiver's, and each classified carrier emits a
  `hunt.candidate` event. New depth: an **offline survey** (`hunt -survey -in
  <capture>`) classifies recorded IQ with no SDR; **`-survey-audio <dir>`**
  writes a WAV clip per active analog-FM carrier; **`-classify-only`** skips
  decoding for a fast inventory; **`-max-dwell-seconds`** listens until carrier
  activity for bursty paging. The classifier's thresholds are now configurable
  (CLI `-class-*` flags / REST fields), occupied bandwidth is measured on the
  full-rate capture so wideband FM isn't mis-sized, and the digital-vs-AM order
  was fixed so pulse-shaped PSK isn't mislabeled AM. The web panel gains a
  classify-only toggle and a sortable signals table.

- **Live signal survey — `gophertrunk hunt -survey`.** The hunt sweep now does
  more than chase trunking control channels: in survey mode it classifies
  *every* detected carrier by modulation family (analog NBFM/WFM, AM, digital
  FSK/C4FM/PSK, paging, trunking) plus an occupied-bandwidth estimate, then
  decodes the conventional ones — POCSAG/FLEX paging and analog-FM activity
  (carrier + CTCSS/DCS) — while still folding any trunking control channel into
  the discovered-system map. The classifier is blind and cheap (FFT
  occupied-bandwidth, envelope coefficient-of-variation, FM-discriminator
  features, and a cyclostationary baud-line detector), reusing the existing dsp
  primitives and the POCSAG/FLEX/conventional decoders rather than duplicating
  them. The result is a `SignalSurvey` inventory surfaced across the CLI
  (printed table), the daemon REST API (`hunt.survey` request flag, `mode` +
  `signals` in `GET /api/v1/hunt`), the web Hunt panel (a Survey-mode checkbox
  and a signals table), and the TUI Hunt panel (a `v` survey-start key and a
  signal list).
- **Constellation / Symbol scope auto-detect the demod mode** (#557). The
  panels' **Mode** selector gains an **Auto** option (now the default) that
  follows the modulation the selected SDR's system is configured to decode —
  C4FM or CQPSK/LSM — instead of asking the operator to pick it. The daemon
  reports this per device on `GET /api/v1/spectrum/devices` as `p25_modulation`,
  resolved by matching the device's tuning against the configured P25 Phase 1
  systems (with a single-system fallback). An explicit C4FM/CQPSK choice still
  overrides Auto and persists.
- **Channel-step nudge in the shared tuning controls** (#557). The
  Constellation and Symbol scope offset field gains a **Step** selector
  (6.25 / 12.5 / 25 kHz) with −/+ buttons and ArrowUp/ArrowDown stepping that
  snap to the channel grid, so walking between adjacent channels no longer
  needs manual kHz entry. The chosen step is shared across panels.

### Fixed

- **Constellation / signal scopes stuck on "waiting for symbols"** (#557,
  #583). The `WS /api/v1/diag/symbols` frame encoded its `dibits` field as a
  Go `[]uint8`, which `encoding/json` serialises as a base64 string rather than
  a JSON number array. The web console drops any frame whose `dibits` isn't an
  array, so every frame was silently discarded and the Constellation, Symbol
  scope, Eye, Tuning, and Histogram panels never rendered. `dibits` now goes
  out as a number array, with a regression test asserting the wire shape.
- **Same talkgroup no longer shows as two duplicate "Active calls"** (#593).
  The duplicate-grant guard keyed an in-progress call on frequency, but a call's
  frequency can change mid-call (a P25 band-plan IdentifierUpdate re-maps the
  channel, or the system hands the call to a new channel), so the guard missed
  and a second `ActiveCall` was bound for the same talkgroup. A logical call is
  now identified by (System, GroupID, Timeslot); on a same-call grant with a
  changed frequency the engine retunes the bound device in place (preserving
  `StartedAt`, no spurious CallStart), or releases it and binds a capable one —
  still exactly one call.
- **Browser audio now plays/seeks on Safari (macOS/iOS)** (#598). Safari's media
  element refuses to play unless the server honors Range requests, but
  `/api/v1/audio/stream` only ever returned a plain open-ended 200 WAV body, so
  "Tap to enable audio" silently failed on macOS while Chrome/Firefox tolerated
  it. The endpoint now answers Safari's bounded probe and open-ended
  `bytes=N-` request with `206` + `Accept-Ranges` + `Content-Range`; requests
  with no Range header keep the existing 200 path. The web player also logs
  `play()` failures instead of swallowing them.
- **Config Builder no longer opens a blank tab** (#595). Two independent defects
  blanked `/config/`: release/installer CI only built the main console before
  `go build`, so the binary embedded an empty `web/configbuilder/dist` and the
  route was never mounted; and the main console's PWA service worker intercepted
  `/config/` navigations via `navigateFallback`. CI now builds the Config
  Builder (and siglab) in every release/installer job, and `/config/` is added
  to the service worker's `navigateFallbackDenylist`.

## [v0.3.6] — 2026-06-08

This release is about **seeing the signal**. A new **Plots hub** (`/plots`)
gathers the per-channel scopes — Constellation, Symbol scope, Eye diagram,
Tuning, Histogram — into one tabbed home that mirrors OP25's Plots tabs (#557,
#583), now with a true symbol constellation, an open four-level eye, live
receiver-state meters, and a symbol-distribution histogram. Underneath, **P25
Phase 1 voice finally decodes** after the IMBE channel-convention and LDU
voice-frame-offset fixes (#574, #578); **TETRA** gains real ETSI training
sequences, a corrected control-channel sync layer with auto-learned colour
code, and soft-decision SB-burst FEC (#569, #571, #573); and a shared
**voice-recording boundary** controller tightly bounds every call by hangtime
and talkgroup (#579). On the operator side, the web **Config Builder** reaches
dual-editor parity with the TUI (#570–#582), the **spectrum** panel gains a
hover readout and dual-pager DDC (#577), and a two-page **Getting Started**
guide lands for non-technical users (#581).

### Added

- **Universal voice recording boundaries — hangtime + per-transmission
  splitting + talkgroup gating** (applies to every voice protocol: FM, DMR,
  P25 Phase 1/2). A new shared boundary controller in the composer ends a call
  promptly once voice stops (configurable `trunking.voice_hangtime_ms`, default
  3.5 s) instead of waiting out the 30 s engine watchdog, so recordings are
  tightly bounded to the actual transmission. `trunking.voice_call_grouping`
  selects `"transmission"` (default — one WAV per over, rolled at each
  end-of-transmission boundary) or `"conversation"` (consecutive same-talkgroup
  overs in one file). On shared voice frequencies, audio from a *different*
  talkgroup is no longer appended to the wrong recording: the P25 Phase 1 chain
  gates each LDU on its decoded Link Control talkgroup and ends the call when
  another talkgroup takes the channel. Recording filenames now carry the RF
  voice-channel frequency (`<stamp>_freq<Hz>_src<src>…`).
- **Plots hub** (`/plots`) — one tabbed home for the per-channel signal
  scopes (Constellation, Symbol scope, Eye diagram, Tuning, Histogram),
  mirroring OP25's Plots tabs (#557 follow-up). The chosen sub-tab is
  reflected in the URL (`/plots/<tab>`); the individual routes still work
  for deep links, and the wideband Spectrum waterfall stays its own tab.
  This replaces the five separate scope entries in the nav with one.
- **Symbol histogram panel** (`/histogram`) — the recovered-symbol
  distribution plus a derived signal-quality readout (#557 follow-up). A
  scrambled P25 channel spreads evenly, so each of the four bins should
  sit near 25%; a **Balance** meter flags a skewed (collapsed-eye)
  distribution, and for C4FM an **SNR (MER)** estimate is derived from the
  soft-level separation vs within-level spread. Computed client-side off
  the existing symbol stream.
- **Tuning panel** (`/tuning`) — live receiver-state meters, GopherTrunk's
  take on OP25's Mixer / Tuner (FLL) tabs (#557 follow-up). Trends the
  demod's residual carrier-frequency-offset estimate (should converge to
  0 Hz on lock) and surfaces AGC level/target, symbol-clock μ/sps and (on
  CQPSK) the equalizer's CMA-error convergence proxy — all read live from
  the production receiver and carried on the existing symbol stream.
- **Eye diagram panel** (`/eye`) — GopherTrunk's take on OP25's datascope
  (#557 follow-up). The daemon's C4FM receiver gains an oversampled,
  AGC-scaled eye tap; the panel folds it over the symbol period and
  overlays the windows so the four-level eye is visible. A healthy channel
  shows four open bands with clear gaps at the decision instant; a closed
  eye flags symbol-timing or SNR trouble. C4FM only (CQPSK's quality view
  is the constellation).
- **True symbol constellation** on the Constellation panel (#557 follow-up).
  The panel gains a **View** toggle: **Symbols** (new default) plots the
  receiver's actual symbol-decision points — for **P25 CQPSK/LSM** a real
  complex constellation that forms four tight clusters on the ±45°/±135°
  diagonals on a clean signal and smears to an X as the eye closes; for
  **P25 C4FM** the four recovered soft levels on the real axis (its open
  4-level eye remains the Symbol scope's job). Amber rings mark the ideal
  cluster centres. The previous wideband-IQ scatter is still available as
  **Vector scope (raw IQ)** for identifying unknown signals. The symbols
  stream reuses the live receiver (`WS /api/v1/diag/symbols`), so it shows
  exactly what the production demod sees.
- **Web Config Builder — dual-editor parity with the TUI** (#570, #572, #576,
  #580, #582). The browser-based Config Builder gains the editor primitives it
  was missing (ListEditor, AdvancedJSON, Fieldset, HzField), a shared
  HTTP-free config core with whole-file marshal/write and per-section
  validation, and backend gap-fill (multi-error reporting, comment-preserving
  merge, file management, RadioReference name lookup). A dual-editor
  schema-drift test now fails CI if any config field is editable in one editor
  but not the other, so the web and TUI builders stay in lockstep.
- **Two-page Getting Started guide** (#581) — a non-technical walkthrough
  (`/getting-started-setup.html`) that takes a new user from download to a
  running scan, featuring the Config Builder, plus refreshed interfaces and
  source-section help sourced from the shared field registry.
- **Spectrum hover readout + dual-pager DDC** (#577). The wideband Spectrum
  waterfall now shows a live frequency/power readout under the cursor, the
  paging DDC can run two channels at once, and decoded pages carry a
  human-readable pager-type label.

### Fixed

- **TETRA control channel would not lock on real signals** (#569, #571, #573).
  The SB-burst lock chain used placeholder sync constants instead of the real
  ETSI training sequences, the control-channel sync layer mis-framed bursts,
  and the FEC was hard-decision only. The decoder now uses the ETSI normal/
  synchronisation training sequences, a corrected sync layer that auto-learns
  the colour code, and soft-decision FEC for the SB-burst, so a production
  144 kHz / 8 sps TETRA control channel locks.

- **P25 Phase 1 voice still garbled after the IMBE channel-decode fix —
  wrong LDU voice-frame positions** (#489 follow-up). With the channel
  decoder corrected, real-air voice was still noise: the LDU1/LDU2 field
  layout in `ldu.go` placed a Link Control block between voice frames u_0 and
  u_1 (`u0, LC1, u1, LC2, …`), but real P25 (per szechyjs/dsd `p25p1_ldu1.c`,
  which reads IMBE frames 1 and 2 back-to-back) is `u0, u1, LC1, u2, LC2, …,
  u7, LSD, u8`. This shifted voice subframes u_1..u_7 by one 40-bit block, so
  only u_0 and u_8 landed on the right bits and the other seven decoded to
  random pitch. `lduVoiceOffsets`, `lduLCESBlockOffsets`, and
  `lduLSDBlockOffsets` are corrected to the real layout (also repairing
  voice-channel Link Control / Encryption Sync / talker-alias metadata, which
  read the same tables). The pre-existing layout test had the DSD order
  inverted and is fixed; a new independent fixture
  (`ldu_realair_test.go`), built from the mbelib/DSD reference with voice
  frames at hard-coded canonical positions, now guards the layout end-to-end.
- **P25 Phase 1 voice decoded to garbled noise** (#489). The IMBE 4400
  channel decoder was self-consistent (its own encode/decode round-tripped)
  but did not match the on-air convention real P25 transmitters use, so every
  recovered voice frame was effectively random — audible as warbling noise.
  Three coupled faults, all invisible to the synthetic round-trip tests and
  surfacing only on real signals: (1) each Golay/Hamming vector's channel bits
  were read in reversed column order; (2) the §7.4 PRBS descrambler took its
  seed from the wrong end of u_0 and applied the keystream in reversed order;
  and (3) the per-vector FEC used `internal/radio/framing`'s Golay(24,12),
  which is a *different* code from the P25 IMBE Golay(23,12,7) and corrupted
  clean codewords. The IMBE path now uses a P25-faithful Golay(23,12,7) +
  Hamming(15,11,3) (transcribed from the mbelib/DSD reference) with the
  correct column order, descrambler seed (taken from the Golay-corrected u_0,
  matching mbelib's `eccC0`-before-`demodulate` order), and keystream
  direction. A real-air-faithful reference-vector test
  (`internal/voice/imbe/p25fec_refvec_test.go`) now pins the decode against
  mbelib/DSD-derived on-air frames, closing the long-standing "no real P25
  voice fixture" gap.

### Changed

- **Constellation & Symbol scope tuning refinements** (#557 follow-up). The
  Symbol scope now shows the tuned frequency as soon as an SDR is selected,
  instead of staying blank until symbols decode. Both panels gain precise
  channel entry: the **kHz** offset field takes 1 Hz resolution (so 6.25 /
  12.5 kHz channel grids land exactly) plus an absolute **MHz** frequency
  field that stays in sync. The Constellation plot is now a responsive square
  that fills the panel column (up to 880 px, drawn at device-pixel ratio for
  crispness) instead of a fixed thumbnail, so it renders as large as OP25's,
  and gains an adjustable **Zoom** control (up to 8×; dots scale with both
  zoom and plot size); its auto-scale now targets the ~95th-percentile radius
  so a stray outlier no longer shrinks the cloud.
- **Warn when message decoders are configured without storage** (#568). A
  decoder that produces messages (paging, MDC, DSC, …) but has no storage
  backend configured silently dropped everything; the daemon now logs a
  startup warning so the misconfiguration is visible.

## [v0.3.5] — 2026-06-07

Site/system **hunting** grows up — `gophertrunk hunt` turns from a one-shot
capture mapper into a live, daemon-integrated discovery engine driven from the
CLI, the TUI, and a web panel with a REST cockpit (#549–#558) — alongside a
live **Symbol scope** oscilloscope (#563) and a much-improved
**Constellation** panel with a server-side frequency-offset view (#559). On
the SDR side, `soapyremote` finally streams reliably (flow-control ACKs,
#545), wideband sources can run up to 20 MHz (#560), and a per-device
`iq_invert` lets spectrum-inverted front-ends lock TETRA (#562).

### Added

- **Site/system hunting — live, daemon-integrated discovery of undocumented
  trunked systems** (#549–#558). `gophertrunk hunt` now does far more than map
  a pre-recorded capture: a live spectrum-sweep discovery engine scans for
  control channels off a live SDR, with a CLI live mode driving it (#552); a
  daemon-integrated hunt manager acquires a spare SDR — else borrows one from
  the pool — to run the sweep inside the running daemon (#554); and the run is
  surfaced through TUI + web-console panels (#556) backed by a REST cockpit
  (#555). Each run honours a requested SDR serial (#558), exports by run id
  with a bounded run history (#558), and can be started straight from the TUI
  panel (#558). Discovery auto-identifies the protocol, accumulates a
  `DiscoveredSystem` map, and resolves per-protocol **site topology** —
  system id + adjacent sites — for P25 (#551), DMR Tier III, EDACS, Motorola
  Type II, NXDN, and TETRA single-site identity (#558), exporting standardized
  files plus a ready-to-paste RadioReference submission. See
  [`docs/hunt.md`](docs/hunt.md).
- **Symbol scope — live demodulated-symbol oscilloscope (OP25-style "Symbol"
  plot)** (#563). A new web panel (`/symbols`) renders the demodulated symbol
  stream off a live SDR: for **P25 C4FM** it shows the pre-slicer soft
  waveform (~4 noisy bands for a healthy channel, with rails at each decided
  level), and for **P25 CQPSK** the sliced dibit decisions. It reuses the
  **production** DSP — the same down-converter and P25 Phase 1 receiver the
  live decoder uses, run as a *parallel* decode on the iqtap broker so
  production control-channel decode is never touched — exposed through the
  receiver's existing soft/dibit taps. The panel shares the Constellation
  panel's offset / Hold / follow-active-call controls, so you can dial the
  scope onto a locked control/voice channel and lift it clear of the SDR
  centre DC spike. Backed by a new
  `WS /api/v1/diag/symbols?device=&proto=&offset=` endpoint and the
  `internal/scanner/symbolscope` engine. The offline **SigLab** analyzer gains
  the matching view: a capture run with `collect IQ diag` + `capture IQ` now
  carries an aligned symbol series on its `IQTaps`, rendered by a new SigLab
  Symbol-scope viz alongside the eye diagram. TETRA and the rest of the C4FM
  family (DMR/NXDN/YSF/D-STAR) — and a soft waveform for them — follow as
  per-receiver soft taps ship. See [`docs/symbol-scope.md`](docs/symbol-scope.md).
- **Constellation panel — frequency-offset view + cleaner render (issue
  #557)** (#559). A centre-tuned constellation is dominated by the SDR's DC
  spike (the DDC's residual carrier leakage at 0 Hz), which sits on top of any
  signal in the middle of the band and reduces the plot to one fat blob. The
  panel now offers an **Offset** control that mixes an off-centre control or
  voice channel down to baseband *server-side, before decimation* (a new
  `offset` parameter on `WS /api/v1/diag/iq`), pulling its symbols out from
  under the spike — the same approach OP25 takes. With **Hold** off the
  offset automatically follows the newest active call on the selected SDR
  (the "last locked channel"); Hold pins it. Decimation now box-averages
  each stride window as a crude anti-alias low-pass, and the render gains an
  additive scatter in GopherTrunk's sky-blue accent (distinct from OP25's
  phosphor green) with labelled ±1 axes, a **DC-block**
  (subtract the rolling mean), and an **Auto-scale** that fills the unit
  circle.
- **`soapyremote`: free-form device-args config block (issue #542)** (#546). A
  `sdr.soapy_remote.device_args` map passes arbitrary key/value pairs straight
  to SoapyRemote's device factory, so a remote front-end that needs
  driver-specific arguments (antenna path, reference clock, channel) can be
  configured without a code change.
- **`ccdecoder`: per-device spectrum-inversion (`iq_invert`) option** (#562).
  A new per-device `iq_invert` flips I/Q at the source so a spectrum-inverted
  front-end (R828D / RTL-SDR Blog V4) locks TETRA and the other control
  channels; shipped with a production-rate (144 kHz / 8 sps) TETRA
  control-channel lock test (#561, #553).

### Changed

- **`sdr.sample_rate` config ceiling raised to 20 MHz** (#560) for wideband
  sources (HackRF, Airspy, or a SoapyRemote-fronted USRP / LimeSDR) that can
  feed a wider span than the previous cap allowed.

### Fixed

- **`soapyremote`: send stream flow-control ACKs so RX actually streams (issue
  #542)** (#545). SoapyRemote's data stream is flow-controlled; without the
  periodic ACKs the server throttled itself to a stop after the initial burst,
  so the tuner appeared to connect but delivered no samples.
- **Drive the IQ pump for single-channel decoders on dedicated dongles (issue
  #547)** (#548). A single-channel decoder bound to its own dongle was not
  pumping IQ through the channelizer, so a dedicated-dongle conventional /
  single-system setup never produced samples; the pump now runs on that path.

## [v0.3.4] — 2026-06-06

High-bit-depth **SoapyRemote** network SDRs and a first-class raw-IQ
**capture** toolchain land (#540, #541), plus a fast algebraic BCH(63,16) NID
decoder that clears the P25 decode-lag (#492) and a batch of RTL-SDR R82xx /
R828D gain and PLL fixes.

### Added

- **SoapySDRServer remote SDRs — high-bit-depth network streaming + control
  from professional hardware (issue #536)** (#541). A new pure-Go (zero-CGO)
  `soapyremote` SDR backend connects to a remote `SoapySDRServer` (from
  pothosware/SoapyRemote) and mounts it as a virtual tuner alongside local
  USB dongles and `rtl_tcp` endpoints. Unlike `rtl_tcp`'s hardcoded 8-bit
  stream, it carries the full dynamic range of high-end radios — USRP,
  LimeSDR, bladeRF, HackRF, Airspy, RTL-SDR, SDRplay — as 16-bit (`CS16`) or
  32-bit float (`CF32`) IQ, with native frequency / sample-rate / gain
  control over SoapyRemote's RPC protocol. Configure under `sdr.soapy_remote`
  (addr/driver/serial/role/format/gain/…); the IQ stream uses the in-order
  TCP transport. Chosen over the originally-proposed VITA 49.2 (VRT) because
  SoapyRemote reaches the same professional hardware with a real,
  interoperable control plane and a single maintained server binary.
- **`gophertrunk capture` — record raw IQ off a live SDR to a `.cfile`**
  (#540). A first-class subcommand that opens a dongle directly (no daemon),
  records the requested number of seconds of raw IQ to a GNU Radio cfile
  (interleaved little-endian float32) or rtl_sdr-native `u8`, and writes a
  siglab `.metadata.json` sidecar so the capture is a drop-in fixture for
  `replay` / `analyze` / `test` and the `samples/` acceptance harness:
  `gophertrunk capture -freq 460000000 -sample-rate 2400000 -seconds 30
  -protocol p25 -out cc.cfile` (`gophertrunk capture -list` enumerates
  SDRs). Complements the daemon's existing `--iq-capture` diagnostic,
  which taps a control SDR already in the running pool.
- **Capture-and-export from the SigLab web console** (#540). A new "Capture
  from tuner" control on the Captures panel records a fixed-length raw-IQ
  capture off a live tuner through the daemon, stages it for immediate
  analysis, and offers the raw `.cfile` as a browser download. Backed by
  new HTTP routes `GET /api/v1/siglab/capture/devices`,
  `POST /api/v1/siglab/capture`, and
  `GET /api/v1/siglab/captures/{id}/download`. The routes return 503 when
  the console is offline (`siglab serve`) or the daemon has no SDR, so a
  build without a tuner doesn't pretend it can record.
- **DMR Tier II Voice LC Header FEC verified against MMDVM + off-air
  diagnostics** (#539). The Tier II Voice LC Header decode path is now
  cross-checked against MMDVM's reference FEC and gains off-air diagnostics so
  a failing real-capture header reports where in the BPTC / RS chain it broke.

### Fixed

- **framing: fast algebraic BCH(63,16) NID decoder clears the P25 decode lag
  (issue #492)** (#534, #537). The NID decode is replaced with an algebraic
  Berlekamp–Massey / Chien BCH(63,16) decoder, removing the per-frame latency
  that was starving the P25 control-channel decoder.
- **rtlsdr: fix inverted mixer-AGC bit and missing VGA in R82xx
  `SetGainMode`** (#535). The R82xx gain-mode path inverted the mixer-AGC
  control bit and never set the VGA, leaving manual-gain dongles deaf; both
  are corrected.
- **rtlsdr: use a VCO power reference of 1 for R828D (Blog V4) PLL fine-tune**
  (#538). The R828D / RTL-SDR Blog V4 PLL fine-tune used the wrong VCO power
  reference, hurting fine-tune accuracy on that tuner.
- **`soapyremote`: stream setup now follows SoapyRemote's real TCP handshake,
  fixing a crash against live `SoapySDRServer` hardware (issue #542)** (#543).
  The TCP stream setup was a single-reply, single-socket guess; real
  SoapyRemote is a two-phase, two-socket exchange (the server replies with the
  data port, accepts both a stream **and** a status socket, then replies with
  the integer stream id). The old code misread the first reply (`setup stream
  port: short rpc response`), which kicked the daemon into a reconnect storm
  that could segfault the remote UHD/USRP server. Setup now opens both sockets,
  reads the stream id as an int, and allows a longer deadline for cold high-end
  devices that spend seconds compiling their RFNoC graph. Verified against the
  upstream source; smoke-test against live hardware before relying on it.
- **`soapyremote`: a manual `gain` now applies on front-ends without AGC
  (issue #542)** (#543). Setting a numeric gain first disabled automatic gain
  control; on radios with no AGC at all (e.g. a USRP TwinRX) that call fails
  with `set_rx_agc() is not supported on this radio` and used to abort the
  whole gain set, leaving the device at its default. Disabling AGC is now
  best-effort, so the manual gain value is still applied.

## [v0.3.3] — 2026-06-05

The P25 CQPSK **linear path** now decodes C4FM — a T/2 fractionally-spaced
equalizer (#532, #492) plus a multipath-gated carrier seed (#529) — and
**SigLab** grows a standalone web SPA over an offline HTTP API (#530). Plus
RTL-SDR Blog V4 detection diagnostics (#528) and a DMR Tier II BPTC/RS
bit-layout fix (#527).

### Added

- **SigLab: standalone web SPA + offline HTTP API** (#530). `siglab serve`
  exposes the offline signal-analysis engine over HTTP and ships a standalone
  Signal Lab single-page app with multi-capture visualization, backed by a new
  in-memory decode path and decimated-IQ taps so a capture can be analysed
  without writing intermediate files.
- **RTL-SDR Blog V4 detection diagnostics + manual override (issue #264)**
  (#528). The tuner-detection path now reports why it did (or didn't) classify
  a dongle as a Blog V4, with a manual override for the ambiguous R828D case.
- **Docs: decoder live-capture requirements summary** (#526). A new summary of
  what each decoder needs from a live capture (sample rate, span, SNR) to lock.
  See [`docs/decoder-capture-needs.md`](docs/decoder-capture-needs.md).

### Fixed

- **DMR: BPTC/RS bit layout corrected so real Tier II Voice LC Headers decode**
  (#527). The BPTC(196,96) + RS(12,9,4) bit ordering didn't match on-air Tier
  II Voice LC Headers, so real captures failed FEC; the layout now matches
  MMDVM and decodes live headers.
- **p25/cqpsk: T/2 fractionally-spaced equalizer so the linear path decodes
  C4FM (issue #492)** (#532). A symbol-spaced equalizer can't correct the
  timing error a C4FM signal carries on the linear (CQPSK) path; the new T/2
  fractionally-spaced equalizer does, so the linear demodulator recovers C4FM.
- **p25/cqpsk: gate the carrier seed on multipath; un-skip the #492 repro**
  (#529). The coarse carrier seed only helps under multipath, so it is now
  gated on a multipath estimate (it was biasing clean-signal locks), and the
  #492 reproduction test is un-skipped.

## [v0.3.2] — 2026-06-04

DMR grows up — multi-slot, Tier III band-plan voice, and license-free
direct mode — and a new offline signal toolkit (`siglab`) lands. The DMR
Tier III control channel now resolves voice grants through a configurable
LCN→frequency band plan (#510) and follows both TDMA timeslots of a
carrier as concurrent, separately-recorded calls (#512, #513), backed by
a stride-aware 2-slot voice decoder (#514), embedded Link Control
timeslot→talkgroup labelling (#515), opt-in composer wiring (#516), and
per-slot metrics / active-call views (#517). DMR Tier I (PMR446 simplex
direct mode) decodes too (#523), and `replay` now runs DMR Tier III / II
captures offline with a `-conjugate` flag for spectrum-inverted
front-ends (#518). The headline addition is **siglab** (#519–#523): a
protocol-agnostic offline replay / test / analysis toolkit that drives
all 14 protocols through the production decode pipelines —
`gen` / `test` / `analyze` / `replay` / `identify` subcommands, a
standalone TUI, structured exporters, synthesis fixtures for every
protocol, per-protocol FEC-outcome tallies, and an auto-detecting signal
identifier. On the P25 side, #524 pins the CQPSK equaliser's centre-tap
phase so the constant-modulus taps stop random-walking into a false
carrier offset.

### Added

- **Offline DMR decode in `gophertrunk replay`.** The `replay` subcommand
  now decodes DMR Tier III / Tier II captures, not just P25 Phase 1: pass
  `-protocol dmr-tier3` (or `dmr-tier2`) to run a raw IQ file through the
  same production `dmr/receiver` + `tier3`/`tier2` control-channel chain
  the daemon uses, printing the locked color code / system ID. A new
  `-conjugate` flag negates Q **before** channelization to decode a
  spectrum-inverted / I-Q-swapped front-end (the RTL-SDR Blog V4 / R828D
  "are I and Q reversed?" case, issue #264) — applied at the source so an
  off-DC channel is no longer pulled from the mirror offset, which the
  post-channelization dual-polarity burst decode cannot recover on its
  own. Combined with `-tune-hz` / `-auto-tune` this makes a captured
  `.cfile` a reproducible DMR test fixture and the primary tool for
  confirming whether a dongle is actually receiving the intended signal.
- **Per-timeslot observability for DMR calls.** A DMR carrier's two
  concurrent calls are now distinguishable in the live views and
  metrics: the TUI active-call Flags column shows `TS1` / `TS2`
  (alongside `E` / `!`), the web active-call detail surfaces a
  Timeslot field, and a new
  `gophertrunk_dmr_voice_calls_total{system,timeslot}` Prometheus
  counter splits DMR voice starts by slot so an operator can spot a
  slot that never carries traffic (a routing/decode gap). Non-slotted
  protocols are unaffected (no slot shown, counter not touched).
- **DMR 2-slot interleaved voice wired into the composer (opt-in).** The
  interleaved decoder + embedded-LC labelling from the previous changes
  are now reachable end-to-end on the production voice path behind a new
  per-system `dmr_interleaved_voice: true`. When set, the DMR Tier III
  control channel tags its voice grants (`Grant.DMRInterleavedVoice`),
  and the composer runs `voice.NewInterleavedDecoder` and routes each
  call to its timeslot with a `slotRouter` — it keeps only the
  superframes whose embedded Link Control names the grant's talkgroup,
  binding that slot's phase so subsequent LC-less superframes still
  route correctly. Defaults off (untouched configs keep the single-slot
  decoder). Verified end-to-end against synthetic modulated 2-slot IQ
  (one talkgroup per slot → only the granted talkgroup's audio reaches
  the recorder). A skip-gated `-tags integration` harness
  (`GOPHERTRUNK_DMR_2SLOT_CFILE`) is the place to validate the on-air
  constants against a real capture before promoting it to the default —
  see [docs/status.md](docs/status.md) and `config.example.yaml`.
- **DMR embedded Link Control decode → per-timeslot talkgroup labelling.**
  On a BS-sourced carrier both timeslots use the identical burst-A voice
  sync, so the sync alone cannot say which slot (and which talkgroup) a
  superframe belongs to. The voice decoder now reassembles the embedded
  Link Control carried by the sync field of bursts B–E — EMB split →
  the new variable `framing` BPTC(128,72) (Hamming(16,11,4) rows + a
  5-bit CRC) → the existing `dmr.FLC` parser — and, on a clean CRC,
  surfaces the call's talkgroup + source on `VoiceSuperframe.LC`.
  Combined with the interleaved decoder's `Phase`, that lets a consumer
  bind each timeslot to a concrete talkgroup. New FEC primitives
  (`framing.HammingEncode/Decode16_11`, `framing.Encode/DecodeEmbeddedLC`,
  `dmr.SplitEmbeddedField` / `dmr.ReassembleEmbeddedLC`) are round-trip
  + single-error-correction tested. The exact ETSI embedded-signalling
  de-interleave order, EMB QR(16,7) FEC, and 5-bit CRC polynomial are
  internally consistent but still pending a real-capture cross-check, so
  the path stays opt-in at the library level — see
  [docs/status.md](docs/status.md).
- **DMR 2-slot interleaved voice decoder.** The DMR voice superframe
  decoder previously assumed a single-slot stream — bursts A–F at a
  contiguous 132-dibit cadence — which only holds for synthetic
  single-slot vectors. A real DMR carrier is 2-slot TDMA: the two
  timeslots' bursts interleave, so a call's own bursts are 264 dibits
  apart. New `voice.NewInterleavedDecoder` (stride 2) handles that — it
  locks each slot's burst A on its own voice sync, gathers that slot's
  B–F by striding over the interleaved other-slot burst, and emits one
  superframe per slot, told apart by the new `VoiceSuperframe.Phase`
  field. `NewDecoder` (stride 1) is unchanged for single-slot streams.
  The exact same-slot cadence on live BS-sourced air (CACH/guard
  handling) still needs a real IQ capture before the interleaved path
  replaces the single-slot decoder on the production composer, so it
  stays opt-in at the library level for now — see
  [docs/status.md](docs/status.md).
- **DMR timeslot is now a first-class call attribute (TS1/TS2).** A DMR
  Tier III carrier interleaves two independent calls — one per TDMA
  timeslot — but the slot was parsed from the grant CSBK and then
  thrown away, so the two calls could not be told apart downstream. The
  grant now carries a 1-based `Timeslot` (0 = not applicable, 1 = TS1,
  2 = TS2), mapped from the CSBK's slot bit on both the standard and
  vendor (Capacity Plus / Connect Plus) grant paths, and surfaced
  through the JSON/SSE API, the gRPC `Grant` message, and the web DTO.
  This is the foundation for separating concurrent same-carrier calls;
  engine/recorder routing and per-slot voice decode land in follow-ups.
- **DMR timeslot routing: TS1 + TS2 are now followed as concurrent
  calls.** Building on the grant attribute above, the trunking engine
  treats `(frequency, timeslot)` as the call identity: a TS2 grant on a
  carrier already running a TS1 call is no longer folded into it by the
  duplicate-grant guard (which previously matched on talkgroup +
  frequency only), so both slots bind their own voice tap / `role: voice`
  SDR and run simultaneously. Each slot is recorded as a distinct WAV
  (`…_ts1.wav` / `…_ts2.wav`, so same-talkgroup slots no longer collide
  on disk), persisted to the call log's new `timeslot` column (added by
  an idempotent migration on existing databases), and surfaced through
  the REST/SSE/gRPC call-history APIs and the web DTO. Following both
  slots of one carrier at once requires at least two voice taps/devices
  that cover the frequency — see
  [docs/hardware.md](docs/hardware.md).

- **DMR Tier III band plan → T3 voice on the wideband dongle.** A
  Tier III voice-grant CSBK references its traffic channel by a 7-bit
  Logical Channel Number (LCN), not an absolute frequency, so the
  decoder needs an LCN→frequency map to follow a call. That resolver
  was never wired from config — both the wideband (`widebandt2`) and
  dedicated-dongle (`ccdecoder`) decode paths built the Tier III
  `ControlChannel` with a nil resolver, so every T3 voice grant was
  dropped with `decode.error stage=no-bandplan` before it reached the
  voice pool. New per-system `dmr_band_plan` config (`linear`
  base/spacing/offset grid **or** an explicit `table` of `{lcn,
  freq_hz}`) is converted to a `tier3.Resolver` and threaded into both
  paths via `tier3.ResolverFromPlan`. Resolved grants are served by the
  existing virtual voice pool (`voice_taps` DDC taps on the wideband
  dongle) or a physical `role: voice` SDR. A `protocol: dmr` system with
  no band plan warns at start-up and keeps decoding the control channel.
  See [`docs/hardware.md`](docs/hardware.md) and `config.example.yaml`.

- **`siglab` — an offline signal replay / test / analysis toolkit**
  (#519–#523). A new protocol-agnostic engine (`internal/siglab`) drives
  any of the 14 protocols GopherTrunk decodes through the same production
  `ccdecoder` pipelines the daemon uses, collecting a structured `Result`
  with exporters and a metadata-driven acceptance harness. It is surfaced
  through five `gophertrunk` subcommands and a standalone (daemon-free)
  Bubbletea TUI:
  - `replay` now routes every protocol — not just the three native
    deep-diagnostic paths (`p25p1`, `dmr-tier3`, `dmr-tier2`) — through
    the shared engine, so `replay -protocol <any>` covers all protocols
    while preserving the P25/DMR receiver-state + soft-eye
    instrumentation.
  - `analyze` decodes a capture and exports a structured signal-quality
    report (`text` / `json` / `jsonl` / `yaml` / `csv` / `csv-events`).
  - `gen` synthesises a test capture + metadata sidecar for a protocol
    with impairment knobs (SNR, carrier offset, DC, I/Q imbalance);
    `test` decodes a capture and grades it against the sidecar's
    acceptance criteria, exiting 0/1 for CI gating. Synthesis fixtures
    now cover every protocol (P25 Phase 1/2, DMR Tier I/II/III, NXDN,
    dPMR, YSF, TETRA, EDACS, Motorola Type II, LTR, MPT 1327, D-STAR).
  - `identify` auto-detects the protocol in a capture — it scans a
    bounded prefix of each registered protocol and scores lock + frame
    sync-cadence + FEC evidence, then runs and renders the full analysis
    of the winner (low-confidence results are flagged inconclusive rather
    than asserted).
  - Per-protocol **deep analysis**: a symbol histogram, a sync-correlation
    landscape against each protocol's own sync word(s), and FEC-outcome
    tallies (clean / corrected / uncorrectable, or CRC pass/fail) — DMR
    slot-type Hamming, EDACS BCH(40,28,2), Motorola BCH(64,16,11),
    D-STAR header CRC-16, NXDN LICH + CAC Viterbi, P25 Phase 2 ISCH
    Golay + MAC trellis, and TETRA SCH/HD RCPC Viterbi.

  The hard-won P25/DMR replay diagnostics that previously lived as
  text-only code in `cmd/` are now consolidated in the engine, so they
  are structured and exportable (`analyze -out-format json|yaml|csv`),
  not just stderr text. See [`samples/README.md`](samples/README.md) for
  the toolkit walkthrough and the unified metadata schema.
- **DMR Tier I (license-free direct mode).** GopherTrunk now decodes DMR
  Tier I — the PMR446 / simplex direct-mode tier. Tier I is wire-identical
  to conventional Tier II (132-dibit burst, BPTC(196,96) + RS(12,9,4)
  Voice LC Header, slot-type Hamming); only the direct-mode sync words and
  the protocol tag differ, so the Tier II conventional channel is
  parameterised by sync word + protocol tag rather than duplicated. The
  new `dmr-tier1` protocol restricts to the four ETSI direct-mode syncs
  (DM-Voice/Data TS1/TS2) so it won't false-lock on base-station traffic,
  and is wired through trunking config, the `ccdecoder` factory, wideband
  validation, and the voice recorder/composer (#523).

### Fixed

- **P25 CQPSK equaliser centre-tap phase pinned** (#524, #492) — the
  constant-modulus equaliser's cost is invariant to a global rotation of
  its tap vector, so the taps random-walked in phase along that null. The
  drift looked like a frequency offset to the downstream Costas loop,
  which integrated it. The centre tap is now anchored to the positive real
  axis after each update, removing the ambiguity without changing `|y|` and
  stabilising the equaliser output phase. A new skip-gated
  `TestCQPSKDemodRecoversFSWWithMultipathAndOffset` reproduces the
  near-spectral-null simulcast case that biases the raw-IQ lag-1 coarse
  seed into a spurious offset; it becomes the regression guard once the
  robust seed fix is validated against a real capture.

## [v0.3.1] — 2026-06-03

RTL-SDR Blog V4 reception finally works and the issue #402 live-decode
push lands its structural fix. #506 cures V4 deafness (the V4 runs a
28.8 MHz crystal and a switched HF/VHF/UHF input bank the stock driver
never handled), #501 opens the WinUSB child interface of composite
(usbccgp) dongles on Windows, and #499 decodes spectrum-inverted DMR
bursts on the R828D. On the #402 front, #507 decouples live IQ ingest
from decode (a forwarder goroutine + a deeper bounded decode queue) and
#508 pools the queued buffers to fix the aliasing that introduced, while
#496 surfaces ADC clipping and #505 stops the driver shedding live IQ.
#497/#503 add CQPSK carrier recovery so a real tuner offset no longer
kills control-channel lock, #498 corrects the P25 Phase 1 LDU
voice-frame interleaving, and #502 adds a diagnostic banner plus verbose
error reporting across every surface. #504 bumps the Go toolchain to
1.25.11 to clear two stdlib advisories.

### Added

- **Diagnostic banner + verbose error reporting across all surfaces**
  (#502) — a new `internal/diag` package prepends a banner (build
  version, OS / kernel, host specs, detected dongles) to every error
  surface and offers a full verbose trace (unwrapped `%w` chain + a
  goroutine stack dump). CLI / launcher error exits route through a
  shared reporter (banner + concise error, then the trace on a verbose
  build or on demand on a TTY); the daemon emits a one-time banner to the
  log at start-up. New top-level `diagnostics.verbose_errors`
  (overridable by `-verbose-errors` / `GOPHERTRUNK_VERBOSE_ERRORS`); the
  HTTP API attaches the banner to the JSON error envelope when enabled
  and exposes `GET /api/v1/diag/banner`; gRPC interceptors decorate
  failing RPCs (config flag or `gophertrunk-verbose` metadata); the web
  `ErrorBoundary` surfaces the diag block in a collapsible panel.
- **ADC-clipping detection** (#496, #402) — a hot, strong-signal site can
  pin the 8-bit RTL ADC rail and shred TSBK CRC while the RMS
  `iq_power_dbfs` gauge averages the peak clipping away. The `ccdecoder`
  now counts rail-pinned IQ samples in the existing power window (no
  extra pass) and exposes an `iq_clip_ratio` gauge plus a throttled WARN
  advising to *reduce* gain / add attenuation; the startup low-gain hint
  is caveated so it no longer points operators the wrong way on an
  overloaded front end.
- **`cchunt.failed` now explains *why*** (#500) — the control-channel
  hunter only ever reported the symptom (retuned everywhere, no lock).
  It now carries the control SDR's live IQ health (dBFS power, DC-bin
  ratio, clip ratio — the #402 signals) with a one-line diagnosis on the
  `cchunt.failed` event payload and a new WARN line; when the decoder saw
  no IQ at all, that absence becomes the diagnosis (check `sdr list
  --probe` / `sdr doctor` / antenna).

### Changed

- **Go toolchain bumped 1.25.10 → 1.25.11** (#504) — clears two stdlib
  advisories `govulncheck` flags (`GO-2026-5037` crypto/x509,
  `GO-2026-5039` net/textproto); both are toolchain-version issues fixed
  only by building against the patched standard library. `go.mod` and the
  `setup-go` version across CI / release / installer workflows updated.
- **Live IQ ingest is decoupled from decode** (#507, #402) — the
  control-channel decoder previously decoded inline on the same goroutine
  that drained the SDR's delivery channel, so any stall (pipeline
  rebuild, GC pause, host contention) made the driver silently drop
  real-time IQ and splice the C4FM stream — the live-fails / replay-green
  signature. A lightweight forwarder now drains the SDR channel into a
  larger bounded decode queue, so a transient stall backs up instead of
  dropping RF. New `ccdecoder_decode_overruns_total` (distinct from
  `sdr_iq_underruns_total`) makes a CPU/host overload provable.
- **Queued IQ buffers are pooled; power/clip/DC observed on the
  forwarder** (#508, #402) — the deep decode queue from #507 could hold
  more driver buffers than the #489 reuse ring allows in flight, so a
  recycled ring slot could corrupt IQ already queued for decode. The
  forwarder now copies each chunk into a pooled, decoder-owned buffer
  before queueing and releases the driver slot immediately, restoring the
  ring invariant; IQ power / clip / DC observation moves onto the
  forwarder so the gauges reflect every chunk the SDR delivered,
  including those dropped at the queue under overload.

### Fixed

- **RTL-SDR Blog V4 deafness** (#506, #264) — the V4 received only noise
  (a raw capture was pure complex white noise across the band), so the
  earlier "color code changes constantly" was the decoder false-locking
  on noise. Two V4-specific gaps versus the rtlsdr-blog librtlsdr fork:
  the V4 runs a **28.8 MHz** reference crystal (PR #266 had keyed every
  R828D to 16 MHz by chip type, mis-tuning every V4 LO by ~1.8×), and the
  V4's switched HF/VHF/UHF input bank was never routed (stock R828D init
  leaves both Cable-1 and Air-In off, so no RF reaches the tuner). The
  fix detects the V4 from its USB strings and, gated entirely on that,
  restores the crystal and ports the fork's per-band input switching,
  notch windows, GPIO5 upconverter relay, and HF tracking-filter bypass —
  R820T2 / non-V4 R828D paths are byte-for-byte unchanged.
- **WinUSB composite (usbccgp) dongles on Windows** (#501) — a composite
  RTL-SDR (e.g. the V4) presents its parent bound to `usbccgp` and the
  real SDR driver on the Interface 0 (`&MI_00`) child node that Zadig
  binds to WinUSB. GopherTrunk only walked the parent-registered device
  interface, so `Open` initialised the wrong node and `sdr doctor` read
  the parent's `usbccgp` service and reported a false BAD. New
  Windows-only discovery walks the USB device-node tree, matches VID/PID +
  `&MI_00`, and opens / inspects the WinUSB child; the parsing logic is
  factored into platform-independent helpers with table tests.
- **DMR spectrum-inverted (I/Q-reversed) bursts on R828D / V4** (#264) —
  a conjugated IQ stream negates the FM discriminator, flipping the
  slicer by `(dibit + 2) mod 4`; P25 Phase 1 already tolerated this but
  DMR did not, and DMR's sync words are closed under the flip so sync
  alone can't resolve polarity. The Tier II / III adapters now decode
  each matched burst at both polarities and let the slot-type Hamming +
  BPTC + CSBK CRC drop the wrong one — identity is tried first, so clean
  R820T2 streams take exactly the same path as before.
- **CQPSK carrier recovery** (#497, #492) — the CQPSK / LSM path had no
  carrier-frequency recovery, so a residual tuner offset spun the whole
  differential constellation and the Frame Sync Word never correlated
  (the synthetic fixtures injected zero offset, hiding it). A two-stage
  recovery now runs: a one-shot lag-1 (Kay) coarse estimate on the raw IQ
  feeding an NCO, then a decision-free second-order `QPSKCostas` loop that
  tracks slow drift. Replay's `carrier_hz_est` diag now shows the loop
  converging to the tuner offset.
- **CQPSK carrier seed under streaming chunk sizes** (#503, #492) — the
  #497 coarse seed only fired when a single `process()` call carried
  ≥ 2048 samples, but production hands the decoder only ~160–200 complex
  samples per call, so the seed never tripped and the full offset reached
  Gardner. The lag-1 autocorrelation is now accumulated across calls
  until the threshold is met, then seeded once (resetting Costas + CMA,
  which had wound up against the uncorrected signal).
- **P25 Phase 1 LDU voice-frame interleaving offsets** (#498, #489) —
  even after the §7.5 IMBE deinterleaver landed, voice decode stayed
  ~100% uncorrectable because the LDU voice-frame slice offsets were
  wrong: the on-air LDU interleaves an LC/ES block between every voice
  subframe with both LSD blocks between u_6 and u_7, so only u_0 sliced
  correctly. The offset tables are corrected to the real interleaving
  (also fixing LC/ES and LSD extraction), pinned by a new field-sequence
  test.
- **Control-channel SDR shedding live IQ** (#505, #489) — a control SDR
  was dropping 25–48% of live IQ chunks/sec (`consumer can't keep up`),
  corrupting the dibit stream into uncorrectable LDUs / TSBK CRC
  failures: the pure-Go deliver path allocated a fresh ~64 KiB buffer per
  chunk and the consumer channel was only 8 deep. A per-stream reuse ring
  (allocation-free hot path), a `u8→complex64` lookup table (bit-identical
  output), and a deeper (8 → 32) stream channel give the resample loop
  jitter headroom; drop-on-overrun stays real.

## [v0.3.0] — 2026-06-02

The issue #402 live-decode investigation drives this release. #486 fixes
a broker close-race panic and surfaces previously-silent live IQ drops
(the live-fails / replay-green tell), #491 hardens the live
control-channel acquisition path and pins the reverted AFC /
adaptive-slicer experiments so they can't silently return, and #493
fixes live CQPSK control-channel lock (an over-gained Gardner timing loop
that only locked on sample-aligned fixtures). #490 corrects P25 Phase 1
voice decode with the IMBE §7.5 deinterleave, #487 applies PPM correction
to the tuner LO rather than only the resampler (#264), #480 extends log
retention to every decoder table and adds a currently-visible aircraft
endpoint, and #488 surfaces silent recorder / composer misconfigs.

### Added

- **Retention sweep across all decoder log tables** (#480) — the sweeper
  only ever deleted `call_log` rows (+ recording files), so `pager_log`,
  `aprs_log`, `vessel_log`, `dsc_log`, `aircraft_log`, `mdc1200_log`,
  `m17_log`, and `location_log` grew unbounded. A new `LogRowMaxAge` knob
  (driven by `retention.log_days`; zero = disabled) deletes rows older
  than the cutoff from each table via a fixed allow-list of table names
  (no user input in the SQL). `config.example.yaml` + `docs/hardening.md`
  updated.
- **Currently-visible aircraft endpoint** (#480) — `aircraft_log` stores
  one Mode-S message type per row, so the raw log can't answer "what's
  flying right now". `GET /api/v1/adsb/aircraft/current` (`?max_age_s=`,
  default 300, max 3600) coalesces the latest non-empty value of each
  field group (callsign / position / altitude / velocity) per ICAO over a
  horizon, newest-last-seen first.
- **Live IQ-drop telemetry** (#486, #402) — IQ chunks dropped on overrun
  by an SDR backend (the consumer falling behind) were silent, making
  live IQ loss indistinguishable from RF problems. Drops now bump the
  existing `iq_underruns_total` Prometheus counter (labelled by driver +
  serial) and emit a warning throttled to one line per second per device,
  via a process-wide `sdr.SetIQDropObserver` hook the daemon installs at
  start-up. A rising counter during decode confirms a live-path overrun
  (offline replay never drops) and explains downstream TSBK CRC failures.

### Changed

- **iqtap broker primary handoff is now lightly buffered** (#486, #402) —
  the broker's primary IQ channel gained a small (2-chunk) buffer so the
  fan-out goroutine isn't stalled by a momentarily-busy primary consumer
  (the per-chunk copy plus a brief decode hiccup), which previously could
  back up the SDR reaper and force whole-chunk drops. The inner driver's
  buffer still bounds latency, so sustained back-pressure still drops as
  before.
- **RTL-SDR PPM correction now re-tunes the tuner LO** (#487, #264) —
  `Device.SetPPM` only wrote the RTL2832U resampler-ratio registers, so a
  configured `ppm` corrected the sample clock but left the tuner carrier
  offset in the signal (a V4's `ppm: -4` had no visible effect and broke
  digital decode). The R82xx tuner now biases its reference crystal by
  `xtal·(1 + ppm·1e-6)` (librtlsdr's `APPLY_PPM_CORR`) and re-tunes;
  `ppm == 0` reproduces the existing register math byte-for-byte, and only
  R82xx-family tuners participate.
- **Live control-channel acquisition path hardened** (#491, #402) — the
  remaining #402 failure was live-only (replay decoded the reporter's
  captures cleanly), isolating it to the acquisition chain replay never
  exercises. A same-`(system, frequency)` `HuntProgress` retune is now
  idempotent (a single-candidate system re-hunting every dwell never
  converged before); the down-converter is built from the SDR's
  *actual* delivered sample rate so a non-exact-divisor rate doesn't
  drift the symbol clock; a too-low-gain warning covers the 51–149 tenths
  band the dB-mistake check missed; and the reverted DDA / adaptive-C4FM
  experiments are pinned off so they can't return.
- **Surface silent recorder / composer misconfigs** (#488) — three
  defensive diagnostics for issues that previously produced only
  INFO-level output: a WARN at P25 Phase 2 chain start when trellis
  decoding is off (live MAC PDUs are trellis-encoded), a Windows WARN when
  `recordings.dir` / `storage.path` / `storage.cc_cache_file` are rooted
  but carry no drive letter (the Unix-style defaults normalise to a
  surprising drive root), and collapse of an exact trailing duplicate word
  in imported system names.

### Fixed

- **iqtap broker `send on closed channel` panic** (#486, #402) — closing
  an IQ subscriber (live spectrum, `--iq-capture`, diagnostics)
  concurrently with an in-flight fan-out could crash the daemon: `fanout`
  checked the closed flag and then sent after dropping `subsMu`, while
  `Subscriber.Close` closed the channel under the lock, leaving a window
  where the send raced the close. Per-subscriber send and close now share
  a `sendMu` so a fan-out send can never land on a closed channel.
  Covered by a new `-race` regression test in `internal/sdr/iqtap`.
- **P25 Phase 1 voice: apply IMBE §7.5 deinterleave** (#490, #489) — voice
  decode reported ~100% uncorrectable LDUs on real signals because
  `DecodeChannelToFrame` ran descramble + per-vector Golay/Hamming FEC on
  the raw on-air bits without first undoing the TIA-102.BABA §7.5 144-bit
  interleaver, so every codeword exceeded its correction radius. The
  symmetric non-interleaved encode/fixture path kept round-trip tests
  green while live air failed. The deinterleave now runs before
  descramble + FEC, with a bijection guard and on-air fixture tests.
- **Live CQPSK control-channel lock: over-gained Gardner loop** (#493,
  #492) — live CQPSK control-channel decode was ~0% while the same
  capture decoded when replayed un-decimated. The Gardner loop's
  effective per-symbol gain is `gain/sps`, and the CQPSK path inherited
  the generic 0.03 default — ~5× too hot at the 48 kHz channel rate — so
  it overshot the timing null and only locked when the input was already
  symbol-aligned (the one phase every synthetic fixture starts on). The
  default drops to 0.005, matching the sibling π/4-DQPSK Phase 2 / TETRA
  pipelines; a starting-phase sweep guards it.
- **replay: decimate CQPSK like production** (#493, #492) — `replay` gated
  its production-matching decimation on `demod == c4fm`, so a wideband
  capture replayed with `-demod cqpsk` ran the whole receiver at the raw
  SDR rate (~417 samples/symbol instead of ~10), invalidating the
  replay-vs-live comparison. The DDC target is now chosen by sample rate
  alone, so both demod modes decimate when the input exceeds the
  production target.

## [v0.2.9] — 2026-06-01

Phase 3 paging completes and M17 joins the digital lineup, while the
Windows RTL-SDR control path finally works on real hardware. #478 lands a
FLEX paging decoder (1600 bps / 2-level) that decodes off the air
alongside POCSAG, both sharing the `pager_log` table and `/pager` panel;
#479 decodes M17 link-setup metadata (who's calling whom, in what mode)
off the LICH without touching audio. #476 flips the P25 Phase 2 MAC-PDU
scrambler default to on so live systems actually decode (issue #451), and
#458 documents that RTL2838U dongles are already supported. The headline
is a four-PR chain (#481–#484) that makes RTL-SDR control transfers work
under WinUSB: #483 is the root cause — the `WINUSB_SETUP_PACKET` was
passed by pointer instead of by value, so every vendor control transfer
sent garbage — backed by #481 (warmup write non-fatal), #482 (clear-halt
+ retry + a USB diagnostics dump), and #484 (NESDR v5 R82xx burst
recovery now fires on Windows pipe stalls too).

### Added

- **FLEX paging decoder (1600/2 mode)** (#478) — completes Phase 3: FLEX
  now decodes off the air alongside POCSAG, both sharing the `pager_log`
  table and `/pager` web panel, tagged by protocol. New
  `internal/radio/pager/flex` carries the logical layer (sync marker +
  mode code → frame-info word → block de-interleave → BCH(31,21) → BIW /
  address / vector / message-word walk) and a streaming decoder for the
  1600 bps / 2-level mode (alphanumeric / numeric / tone vectors). The
  FLEX BCH(31,21)+parity primitive (`internal/radio/framing/bch_flex.go`)
  reuses the tested POCSAG codeword via bit-reversal (info-low layout),
  with round-trip + 2-bit-correction coverage. The receiver mirrors the
  POCSAG DSP frontend (FM demod → resample → slicer → decoder) and
  publishes `KindPagerMessage` with `Protocol="flex"`; `pager_log` gains
  a `protocol` column (default `pocsag`).
- **M17 link-layer metadata decoder** (#479) — Milestone 4 of the
  roadmap: recover M17 link-setup metadata (caller, callee, mode) without
  decoding audio (Codec2 voice is a later milestone). New
  `internal/radio/m17` parses the LSF (base-40 callsigns, TYPE mode/CAN,
  CRC-16 poly 0x5935), reassembles the LICH (Golay(24,12), six chunks →
  240-bit LSF), and runs a streaming decoder that hunts the 0xFF5D stream
  sync → LICH → LSF, so an in-progress transmission is picked up within
  ~240 ms with no convolutional machinery. The receiver adds a C4FM DSP
  frontend (FM demod → resample → matched filter → Mueller-Müller timing
  → 4FSK slice → dibit) and publishes `events.KindM17LinkSetup`; new
  `m17_log` table, `GET /api/v1/m17/linksetups`, and an `m17.channels`
  config block. Spec constants are validated against a synthetic encoder;
  real-capture calibration and the Codec2 payload are documented
  follow-ups. See [docs/m17.md](docs/m17.md).

### Changed

- **RTL2838U dongles documented as supported** (#458) — the RTL2838U is
  the Realtek demodulator / USB-bridge chip (a variant of the RTL2832U),
  not a tuner; dongles labelled "RTL2838U" enumerate as `0x0bda:0x2838`
  and are already fully supported (the real R820T2 / R828D tuner inside
  is handled by the tuners package). The device-whitelist friendly name
  and `docs/hardware.md` now say so, so users searching for "RTL2838U"
  find confirmation their hardware works out of the box.

### Fixed

- **P25 Phase 2: default the MAC-PDU scrambler to on** (#476, issue
  #451). A live Phase 2 system logged `composer: p25p2 macCfg suggests
  live MAC PDU decode will fail` with a valid identity-derived seed but
  `scrambler=0`. Every on-air P25 Phase 2 MAC PDU is PN44-scrambled per
  TIA-102.BBAC-1 §7.2.5, so with descrambling off, MAC decode (source ID,
  talker alias, encryption sync) can never succeed. `ParseScramblerMode("")`
  now defaults to `ScramblerOn` (was `ScramblerOff`, which only suited the
  synthesized unscrambled test fixtures), mirroring `ParseTrellisMode`.
  `ScramblerOn` (not `Probe`) is correct because both production MAC paths
  already feed the spec per-slot PN44 offset from superframe sync, so no RS
  verification is needed to pick the offset.
- **Windows RTL-SDR: pass `WINUSB_SETUP_PACKET` by value** (#483) — the
  actual reason RTL-SDR control transfers never worked on real Windows
  hardware. `WinUsb_ControlTransfer` takes the setup packet *by value* and
  the x64/arm64 calling convention passes the 8-byte struct in a single
  integer register; GopherTrunk passed a *pointer*, so WinUSB read the
  pointer's low bytes as `bmRequestType/bRequest/wValue/wIndex/wLength` — a
  garbage vendor request the device timed out on (`ERROR_SEM_TIMEOUT`) or
  rejected (`ERROR_GEN_FAILURE`). Descriptor reads went through a different
  prototype and succeeded, which is why the dongle reported
  `winusb-bound=true` while every vendor transfer failed. The setup packet
  is now folded into the `uintptr` argument (little-endian, matching its
  in-memory image) at all three call sites, with a golden test pinning the
  packing.
- **Windows RTL-SDR: clear-halt + retry stalled control writes, append USB
  diagnostics** (#482). `winTransport.ControlOut` now clears the
  control-pipe halt (`WinUsb_ResetPipe` pipe 0) and retries the write once
  when it stalls with `ERROR_GEN_FAILURE`, since some clone RTL2832U
  firmwares need the explicit `CLEAR_FEATURE` the USB spec says a SETUP
  should auto-clear. When bring-up still fails, `openDevice` now appends a
  full USB diagnostics dump (bound driver — WinUSB / libusbK / DVB / none —
  device + config descriptors, and a control-IN read probe), so a single
  `gophertrunk sdr list --probe` captures everything needed to triage a
  dongle that rejects control transfers.
- **Windows RTL-SDR: make the USB warmup write non-fatal** (#481). The
  warmup write is librtlsdr's sacrificial "dummy write" that absorbs the
  first control-transfer NAK some clone dongles emit right after the
  interface is claimed; librtlsdr never checks its result. GopherTrunk had
  treated it as a must-succeed gate, and each retry re-opened the device
  and re-armed the same NAK, so the dongle never reached `InitBaseband` and
  `Open` failed with `ERROR_GEN_FAILURE`. `runBringup` now swallows any
  warmup error (logging it under `RTLSDR_DEBUG_USB`) and proceeds to
  `InitBaseband` step 0, whose byte-identical transfer is the one that
  actually needs to land; genuine stalls are still caught by the outer
  reset+retry envelope. Stale troubleshooting URLs in the bring-up hints
  now point at `gophertrunk.org` / `install-windows.html`.
- **Windows RTL-SDR: fire NESDR v5 R82xx burst recovery on Windows pipe
  stalls** (#484, issue #248). The R82xx tuner-init burst-write recovery
  (per-chunk retry + 16→8→4 chunk-size halving — the librtlsdr-parity fix
  for the NESDR v5 cold-boot I²C stall) keyed its retry guards solely on
  `syscall.EPIPE`. On Windows the identical I²C-bridge stall surfaces as
  `usb.ErrPipeStalled` (`ERROR_GEN_FAILURE`), so every layer of recovery
  was skipped and the first chunk failure propagated straight out. The
  guards are now a shared `isI2CBurstStall` predicate matching both
  classes, so per-chunk retry and the halving fallback fire on Windows
  exactly as on Linux.

## [v0.2.8] — 2026-05-31

The issue #402 control-channel decode-quality push lands its first real
win: #470 makes the P25 decoder read every TSBK in a data unit (not just
the first) and adds `replay` channel tuning for off-centre captures,
roughly tripling the TSBKs recovered on the MMR Site 9 capture. #455 lets
operators declutter the UI by switching off navigation tabs they don't
use, and #459 corrects a complex-LMS equalizer weight update while
evaluating an IQ-domain equalizer for the #402 multipath.

### Added

- **`replay` channel tuning for off-centre captures** (#402) — the
  `gophertrunk replay` subcommand can now frequency-shift a recorded
  wideband IQ file so an off-centre control channel lands at 0 Hz before
  the demodulator, the way the SDR tuner does on a live device. `-tune-hz`
  applies a fixed offset; `-auto-tune` estimates the dominant carrier from
  the start of the file. This lets a captured file whose channel was not at
  the recording centre (e.g. MMR Site 9, ~+37 kHz off) be replayed the same
  way it decodes live. Backed by a reusable `dsp.NCO` frequency shifter, a
  `dsp.EstimateCarrierOffsetHz` carrier estimator, and a tuning-offset mode
  on the `ccdecoder` down-converter. A channelised slice of the real Site 9
  control channel ships as a decode regression fixture.
- **UI navigation tabs are now configurable** (#455) — operators running
  GopherTrunk for a single task can declutter the nav by switching off tabs
  they don't use. Every tab shows by default; setting a key to `false` under
  `web.tabs` hides it from the nav strip in both the web SPA and the
  terminal TUI (routes stay mounted — nav-only hiding). New `WebConfig.Tabs`
  map with a `KnownUITabs` canonical set (`Validate()` rejects unknown
  keys); the read-only `/api/v1/runtime` snapshot carries the hidden list so
  both clients filter from one source of truth.

### Fixed

- **P25 control channel: decode every TSBK in a data unit, not just the
  first** (#402). A P25 trunking data unit packs up to three 98-dibit TSBK
  blocks after one FSW + NID, the last flagged LB=1; the control-channel
  decoder only ever decoded the first, silently dropping the ~2/3 of a busy
  site's signalling (grants, affiliations, status broadcasts) carried in the
  second and third blocks. It now decodes every block in the unit, stopping
  at the last-block flag, and resumes blocks that span receive batches — so
  the yield is the same whether the dibit stream arrives a frame at a time
  or in tiny USB transfers. On the MMR Site 9 capture this roughly triples
  the TSBKs recovered (14 → 41 in ~1 s, all CRC-clean). A non-contiguous
  dibit stream (a resync or capture gap) now also flushes the partial-frame
  buffer instead of trying to stitch a frame across the break.
- **Equalizer: correct complex-LMS weight-update conjugation** (#402). The
  complex LMS update computed `w_k += μ·x·conj(e)` instead of the correct
  `w_k += μ·e·conj(x)`; for the non-Hermitian FIR the two differ only in the
  sign of the imaginary cross-term (identical on a real channel, which is
  why the existing real-coefficient test missed it). A genie-trained
  equalizer using the corrected update fully recovers a two-ray echo (dibit
  SER 0.086 → 0.000 through the real receiver) and is a no-op on clean
  signal. No production code calls LMS yet, so no behaviour change ships
  beyond the equalizer package; a new complex-channel regression guards it.

## [v0.2.7] — 2026-05-30

Phase 5 finishes its DSP frontends and the analog side fills in. ADS-B
reaches end-to-end both ways — #440 consumes BEAST output from an existing
dump1090 / readsb with a per-ICAO CPR pair-tracker, and #449 adds a native
1090 MHz PPM Mode-S receiver so aircraft decode straight off the air; #448
gives DSC its FFSK frontend (the last "no DSP" hole in Phase 5); and #441
lands MDC1200 Motorola signaling. #445 adds a gain-units guardrail for the
common tenths-vs-dB mistake, #444 forces decoded calls to the
vocoder-native 8 kHz WAV rate (fixing garbled playback), and the #402
slicer work settles on the fixed C4FM slicer as the default (#450) with the
adaptive slicer behind a flag and its outer-rail tracking corrected (#447).

### Added

- **ADS-B end-to-end via BEAST upstreams + per-ICAO CPR pair-tracker.**
  Most 1090 MHz receive chains already run dump1090 / readsb / BeastSplitter
  against a dedicated RTL-SDR; GopherTrunk now consumes their BEAST binary
  output over TCP and feeds the frames into the same
  `events.KindAircraftReport` bus / `aircraft_log` SQLite /
  `/api/v1/adsb/aircraft` REST / `/adsb` web panel stack that shipped in
  #434. Operators add an `adsb.beast_upstreams` entry (typically
  `127.0.0.1:30005` — the standard dump1090 / readsb BEAST port) and
  aircraft start landing on the live map immediately. Reconnect-with-backoff
  on upstream drops; the embedded CPR tracker resets between reconnects so
  stale even/odd halves don't pair across the gap. New
  `internal/radio/adsb.Tracker` is the per-ICAO state machine that buffers
  the most-recent CPR half and calls `CPRDecodeGlobal` when both halves
  arrive within the spec's 10 s window (DO-260B §2.2.3.2.3.7); `Prune(now)`
  evicts ICAOs idle > 10 s. New `internal/radio/adsb/beast` package — frame
  parser (`ReadFrame` handles the 0x1A byte-stuffing, hunts for sync after a
  torn TCP segment) + reconnecting TCP client (`Client.Run`) that pipes each
  Mode-S frame through `adsb.Decode` → `Tracker.Update` → `bus.Publish`.
- **ADS-B native 1090 MHz PPM Mode-S receiver** (#449) — ADS-B now decodes
  straight off the air as an alternative to running a separate dump1090 /
  readsb. New `internal/radio/adsb/ppm` takes IQ → resample to 2 Msps →
  magnitude envelope → dump1090-style 8 µs preamble correlation → PPM bit
  slice → DF frame-length (56/112) → frame bytes, with a magnitude carry
  buffer so a preamble split across two IQ chunks still decodes. The decode
  → CRC gate → CPR track → `AircraftReport` mapping is factored into a
  shared `adsb.ProcessFrame` so the PPM and BEAST paths produce identical
  reports. `ADSBConfig` gains a `channels` list (default 1090 MHz) and the
  daemon pins the SDR off its iqtap broker, mirroring the AIS receivers.
- **DSC FFSK DSP frontend + bit-stream receiver** (#448) — closes the last
  "no DSP" hole in Phase 5: DSC had a parser, BCH(10,7), storage, REST, and
  panel scaffolding but no way to turn IQ into sequences. New
  `internal/radio/dsc/ffsk` takes IQ → FM demod → resample to 9600 sps →
  FFSK discriminator (1300/2100 Hz) → Mueller-Müller timing → direct-FSK
  slicer; the receiver slides a 10-bit window, BCH-syncs on the repeating
  phasing DX character (dual-polarity), samples the DX grid to recover 7-bit
  symbols, detects EOS, and publishes `KindDSCMessage`. New `DSCConfig` /
  channel config and daemon spawn loops, mirroring the AIS receivers.
- **MDC1200 Motorola signaling decode** (#438) — end-to-end pipeline for the
  analog FFSK data burst Motorola radios key at the head / tail of a
  transmission on conventional VHF / UHF voice channels. 1200-baud CCIR FFSK
  DSP frontend (FM demod → FFSK discriminator at 1200 / 1800 Hz →
  Mueller-Müller timing → NRZ slicer, reusing the existing `demod.FFSK`), a
  40-bit sync framer with inverted-polarity tolerance, 16×7 de-interleave,
  op / arg / unit-ID decode with a CRC-16-CCITT check, and an op/arg label
  table (PTT ANI, emergency, status, radio check, call alert, selective
  call, radio inhibit / enable, remote monitor). Plus
  `events.KindMDC1200Message`, SQLite `mdc1200_log`, `GET
  /api/v1/mdc1200/messages`, the `/mdc1200` web panel, and an
  `mdc1200.channels` config block. Clean-room implementation under
  Apache-2.0. See [docs/mdc1200.md](docs/mdc1200.md).

### Changed

- **Gain-units guardrail.** `sdr.devices[].gain` (and the rtl_tcp
  equivalent) is in *tenths* of a dB — `"320"` = 32 dB — but operators
  coming from SDRTrunk / OP25 / gqrx routinely paste a whole-dB value like
  `"32"`, which parses to 3.2 dB and snaps to the bottom of the tuner
  ladder, leaving the radio effectively deaf (no control-channel lock, no
  decodes) with no feedback. The daemon now WARNs at startup when a
  bare-integer gain parses to ≤ 5.0 dB (`gain looks like dB, not
  tenths-of-dB …`, suggesting the ×10 value), and the SDR pool now logs the
  applied gain in dB on every device (`sdr: gain set … gain_db=…`) so a
  units mistake is visible without enabling debug. No behaviour change for
  valid configs; decimal forms like `"32.0"` are still taken as whole dB.
  Docs (`config.example.yaml`, `docs/hardware.md`) updated.
- **P25 Phase 1: fixed C4FM slicer is the default; adaptive slicer behind a
  flag** (#402). On the MMR Site 9 capture the fixed-threshold slicer is the
  best performer; every adaptive variant that moved the +1/+3 threshold
  above the fixed nominal decoded worse, because the +3 eye is spread low by
  an RF-domain asymmetry the slicer can't fix. Mirroring the #430 DDA
  precedent, the adaptive C4FM slicer is now opt-in
  (`Options.EnableAdaptiveC4FMSlicer`, default off; `replay
  -adaptive-slicer` for A/B); production pipelines (`ccdecoder`,
  `widebandt2`) revert to the fixed slicer. The adaptive slicer's threshold
  model was also improved (inward-only cap + variance-aware boundaries) so
  it is no worse than fixed on a stretched eye.
- **Voice: force vocoder-native WAV rate + decode-quality telemetry**
  (#356). The IMBE/AMBE vocoders always emit 8 kHz PCM and the recorder
  appended those samples without resampling, but the WAV header used the
  configured `recordings.sample_rate` — so a non-default rate played decoded
  P25/DMR calls back at the wrong speed (garbled). `handleStart` now
  instantiates the vocoder before opening the WAV and forces the header to
  8 kHz for decoded calls (analog/NBFM fed via `WritePCM` still honour the
  configured rate), and `CallComplete` publishes the session's actual rate,
  matching the offline decoder.

### Fixed

- **Adaptive C4FM slicer outer-rail under-tracking** (#402). The
  soft-responsibility level update scaled the data-directed pull by the
  per-symbol responsibility but leaked toward nominal at full weight every
  sample, halving the intended 0.8 mix toward the observed centroid — so a
  stretched +3 rail under-tracked and held the +1/+3 threshold below
  optimal. Scaling the leak by responsibility too restores a true
  responsibility-weighted EMA, landing the threshold at the ~0.22 optimal
  midpoint. (Behind the now-opt-in adaptive slicer flag.)

## [v0.2.6] — 2026-05-29

Phase 5 expands across marine + aviation and the panels gain a shared
map. AIS reaches end-to-end live: #427 lands the protocol layer + bus /
storage / REST / `/ais` panel scaffolding, #428 wires the 9600 Bd GMSK
DSP frontend + receiver glue (FM demod → 76,800 sps resample → GFSK
matched filter at BT 0.4 → Mueller-Müller timing → NRZI → HDLC → CRC →
`ais.Decode`), so pinning one SDR to 161.975 / 162.025 MHz lights up
vessel positions. #433 adds the DSC marine scaffolding (ITU-R M.493-15
distress / urgency / safety / routine call decode, BCH(10,7) syndrome
check, MMSI + position codecs) and #434 the ADS-B aviation scaffolding
(ICAO Annex 10 Mode-S CRC-24, DF 17 / 18 extended-squitter
identification / position / velocity decode, globally-unambiguous CPR).
#435 ties them together with a shared Leaflet `PositionMap` across the
APRS / AIS / DSC / ADS-B panels — per-protocol marker colours, XSS-safe
tooltips, camera auto-fit. Plus #419 ports the full APRS Mic-E decoder.
Trunking robustness: #426 distinguishes a carrier-drop natural call end
from a silent-timeout reap and #431 fans raw IMBE / AMBE frames out to
`rawFrameSinks` (both issue #356); #417 makes `sdr.devices` a strict-mode
allowlist (issue #264); #418 settles the warmup→step-0 race on Windows
clone dongles (issue #395); #423 builds the wideband voice taps before
the voice pool (fix #422) and #424 makes voice-grant preemption
frequency-aware; #425 corrects the Motorola alias cipher stop
recurrence. Issue #402 (RTL-SDR DC-spike on P25 control) continues:
#429 fixes the DDA-AFC handoff regression that froze a wrong carrier
offset, #430 defaults to CoarseAFC-alone and fixes the 10x AFC
diagnostic, and #432 swaps in an adaptive 4-level C4FM slicer that
tracks an asymmetric eye.

### Added

- **Live map across APRS / AIS / DSC / ADS-B.** Position-bearing
  decoded rows now plot on a shared Leaflet map at the top of
  each protocol panel. APRS station fixes (Mic-E + uncompressed
  positions) render as blue markers; AIS Class A/B vessel
  positions as cyan; DSC distress alerts that included a
  position as red (oversized for high visibility); ADS-B
  aircraft (once per-ICAO CPR pairing lands) as purple. Marker
  tooltips render the per-protocol short label (callsign /
  MMSI / ICAO+altitude / nature-of-distress); the camera
  auto-fits to the active point set on every poll-refresh.
  New `web/src/components/PositionMap.tsx` is a single
  `<PositionMap points={...}>` component the four panels share —
  one Leaflet `L.map` per panel, points → `L.circleMarker`
  diff-update keyed by stable row IDs so a row-set update
  patches markers in place instead of tearing them down. Adds
  `leaflet@^1.9.4` + `@types/leaflet` as web deps; tiles served
  from the standard OSM tile servers (compliant with the OSM
  Tile Usage Policy for the single-user self-hosted operator
  console; larger fleets configuring their own tile cache is
  the obvious follow-up). XSS-safe tooltip rendering (HTML
  escapes on all user-derived label / detail fields).
  Tests: 5 new (`PositionMap.test.tsx` — container renders,
  per-kind marker colours, distress radius / colour, HTML
  escape on tooltips, camera auto-fit). All 115 web tests
  passing (20 test files).


- **ADS-B aviation — protocol layer + bus / storage / REST / panel
  scaffolding.** First slice of Phase 5 ADS-B: every commercial
  passenger flight, most general-aviation, and all military
  aircraft over US / EU airspace continuously broadcasts on
  1090 MHz — the same data that powers FlightRadar24 /
  FlightAware / adsb.lol / OpenSky. GopherTrunk now has the
  protocol layer to decode it on the operator's own SDR.
  New `internal/radio/adsb` package decodes ICAO Annex 10 Vol IV
  Mode-S frames: CRC-24 verification with polynomial 0xFFF409
  (verified directly on DF 11 / 17 / 18; the ICAO-overlay scheme
  for DF 0 / 4 / 5 / 20 / 21 recovers the address by XORing the
  computed CRC). Extended-squitter (DF 17 / 18) type-code
  dispatch for the operator-visible majority: identification
  (TC 1-4 with the 6-bit ICAO alphabet decoding 8-char
  callsigns), airborne position (TC 9-18 / 20-22 with CPR-encoded
  lat/lon and 12-bit Q-bit altitude at 25-ft resolution), surface
  position (TC 5-8), airborne velocity (TC 19 with ground speed
  + track for subtypes 1/2, air speed + heading for 3/4, common
  vertical-rate field). Globally-unambiguous CPR position
  decoder (DO-260B §2.2.3.2.3.7) from an even+odd pair, with NL
  table matching the dump1090 reference. Validated against the
  canonical mode-s.org reference samples (identification
  "KLM1023" / ICAO 4840D6; CPR pair decodes to lat 52.2572 N /
  lon 3.91937 E / alt 38000 ft; velocity GS 159 kn / track ≈ 183°
  / VR -832 fpm).
  New `events.KindAircraftReport` event + `storage.AircraftReport`
  payload + `aircraft_log` SQLite table (one row per decoded
  frame, indexes on `(received_at)` and `(icao, received_at)`).
  `storage.AircraftLog` subscriber drains the bus and writes one
  row per message; the daemon spawns it alongside `dscLog` /
  `vesselLog` / `aprsLog` / `pagerLog`. New REST endpoint
  `GET /api/v1/adsb/aircraft?limit=N` (default 200, max 5000)
  and web panel `/adsb` with columns Received / ICAO / Kind /
  Callsign / Lat-Lon / Alt / GS-Track / VR. CRC-failed frames
  highlight yellow.
  Tests: 13 protocol-layer (identification decode, CPR pair
  global decode against the dump1090 reference vectors, velocity
  decode, all-call DF 11, short-frame safety, CRC self-
  consistency + corruption detection, NL table boundary values,
  altitude Q=1 round-trip), 4 storage (insert position / ident /
  filter / order), 3 REST (503 / list / limit), 5 web (empty /
  position / ident / velocity / error). All passing.
- DSP frontend (1 Msps PPM + Mode-S preamble correlation +
  frame extraction) follows as the next slice. See
  [docs/adsb.md](docs/adsb.md).

- **DSC marine — protocol layer + bus / storage / REST / panel
  scaffolding.** First slice of Phase 5 DSC: GMDSS Digital
  Selective Calling messages — distress alerts, urgency / safety
  broadcasts, individual / group / all-ships routine calls — are
  the SOLAS-mandated digital signalling on marine VHF channel 70
  (156.525 MHz) and the HF DSC channels. A coast-guard MMSI
  lighting up the channel-70 stream is near-instant visibility
  into SAR activity.
  New `internal/radio/dsc` package decodes ITU-R M.493-15
  formats: Distress (self-MMSI + nature + position + UTC time),
  All-Ships safety / urgency / routine, Individual call
  (target + source MMSI), Group, Geographic-area, and
  Auto-Individual. BCH(10,7) syndrome check (CRC-3 with
  `g(x) = x³+x+1`) — the spec calls it "BCH" but min Hamming
  distance is 2, so single-bit errors are reliably **detected**
  but not corrected at this layer; DSC achieves the actual
  correction via DX / RX redundancy at the bit-stream layer
  above (each character is sent twice and the receiver compares
  the two streams).
  MMSI codec unpacks 5 symbols × 2 digits → 9-digit MMSI.
  Position codec decodes the 10-digit `Q.DD.MM.DDD.MM` format
  with quadrant-bit hemisphere flip (0 = NE, 1 = NW, 2 = SE,
  3 = SW). The all-9s "position unknown" sentinel collapses
  `HasPosition` to false.
  New `events.KindDSCMessage` event + `storage.DSCMessage`
  payload + `dsc_log` SQLite table (one row per decoded
  sequence, indexes on `(received_at)` and
  `(self_mmsi, received_at)`). `storage.DSCLog` subscriber
  drains the bus and writes one row per message; the daemon
  spawns it alongside `vesselLog` / `aprsLog` / `pagerLog`.
  New REST endpoint `GET /api/v1/dsc/messages?limit=N` (default
  200, max 5000) and web panel `/dsc` with columns Received /
  Format / Category / Self MMSI / Target-or-Nature / Body /
  Lat-Lon. Rows tint by category — distress = red, urgency =
  orange, safety = blue, routine = default.
  Tests: 15 protocol-layer (BCH round-trip + syndrome check +
  single-bit error detection, MMSI codec, position quadrant
  signs, position unknown-sentinel, end-to-end distress decode
  with position + nature + UTC time, individual-call decode,
  all-ships safety decode, short-payload safety), 4 storage
  (insert distress / individual / filter / order), 3 REST
  (503 / list / limit), 4 web (empty / distress / individual /
  error). All passing.
- DSP frontend (1200 Bd FSK at 1300 / 2100 Hz tones + 10-bit
  symbol assembly + DX/RX redundancy merge) follows as the
  next slice. See [docs/dsc.md](docs/dsc.md).
- **AIS DSP frontend + receiver glue — pipeline is now end-to-end.**
  Second slice of Phase 5 AIS: `internal/radio/ais/receiver` is the
  bit-stream orchestrator (HDLC framer → CRC-CCITT validation →
  MSB-first bit unpack → `ais.Decode` → bus event); on top of it
  `internal/radio/ais/gmsk` is the IQ-to-bits frontend (FM demod
  → real resampler to 76,800 sps → GFSK matched filter at
  BT = 0.4, span 4 symbols → Mueller-Müller symbol-timing
  recovery → zero-threshold slicer → NRZI decode → `receiver.Push`).
  New top-level `ais.channels` config schema mirroring
  `aprs.channels` (serial, frequency_hz, drop_bad_fcs,
  drop_non_position). The daemon constructs one `gmsk.Receiver`
  per entry, subscribes each to its SDR's iqtap broker via the
  standard spawn closure, and the AIS pipeline goes live the
  moment an operator pins one SDR to 161.975 (channel 87B) or
  162.025 (88B). Same `Inner()` accessor for frame-counter
  metrics that `aprs/afsk` exposes. The bit-stream layer
  validates the same HDLC FCS algorithm AX.25 uses (reflected
  polynomial 0x8408, init 0xFFFF, final XOR 0xFFFF) — AIS
  inherits the link-layer conventions verbatim per
  ITU-R M.1371-5 §4.2. End-to-end synthetic test drives a real
  AIVDM type-1 payload (gpsd canonical sample, lat 37.802 N,
  lon -122.342 W, MMSI 366053209) through `buildAISFrame` →
  `wrapHDLC` → `Receiver.Push` and asserts the bus event
  carries the correct MMSI + decoded position. 9 new bit-stream
  tests + 8 new DSP tests, all passing.

- **AIS marine — protocol layer + bus / storage / REST / panel
  scaffolding.** First slice of Phase 5 AIS: every SOLAS-covered
  vessel (passenger ships, tankers, cargo > 300 GT) broadcasts an
  AIS position every 2-10 s on marine VHF channels 87B / 88B
  (161.975 / 162.025 MHz) — free wide-area positional data
  GopherTrunk now has the protocol layer to decode. New
  `internal/radio/ais` package decodes the operator-visible
  majority of ITU-R M.1371-5 message types: Class A position
  reports (types 1/2/3, layout in §3.3.1), Class B position
  reports (type 18), Class B extended (type 19), base-station
  reports (type 4), static + voyage data (type 5: vessel name,
  IMO, call-sign, destination, ETA, ship type, dimensions), and
  Class B static data (type 24 Parts A + B). MSB-first
  bit-field readers (`readBitsUint`, `readBitsInt` with proper
  two's-complement sign-extension) decode the spec's signed
  lat/lon (28-bit longitude, 27-bit latitude, 1/600000 minute
  resolution). The 6-bit ASCII text table (M.1371-5 Table 47)
  unpacks vessel-name / call-sign / destination fields with
  trailing-padding stripped. Spec "not available" sentinels
  (lat 91°, lon 181°) collapse the `HasPosition` flag.
  New `events.KindAISMessage` event + `storage.AISMessage`
  payload + `vessel_log` SQLite table (one row per decoded
  message, indexed on `(received_at)` and
  `(mmsi, received_at)`). `storage.VesselLog` subscriber drains
  the bus and writes one row per message; the daemon spawns it
  alongside `aprsLog` / `pagerLog`. New REST endpoint
  `GET /api/v1/ais/vessels?limit=N` (default 200, max 5000) and
  web panel `/ais` with columns Received / MMSI / Type / Body /
  Lat-Lon / SOG-COG. Static-data rows show vessel name + call-
  sign + destination; position-data rows show lat/lon + SOG /
  COG. CRC-failed messages highlight yellow. Decoder validated
  against the gpsd AIVDM canonical samples (Class A position
  matches lat 37.802118 N, lon -122.341618 W; static-voyage
  decodes a non-empty vessel name + call-sign).
  Tests: 14 protocol-layer (bit-readers, sign-extension, 6-bit
  ASCII table, type dispatch, AIVDM round-trip for types 1, 18,
  5, "not available" sentinel handling, hex round-trip), 4
  storage (insert / static / filter / order), 3 REST (503 / list
  / limit), 4 web (empty / position / static / error). All
  passing.
- DSP frontend (9600 Bd GMSK + HDLC framer) follows as the next
  slice. See [docs/ais.md](docs/ais.md).

## [v0.2.5] — 2026-05-28

Issue #376 follow-up (Motorola MMR P25 talker alias) closes end-to-end +
Phase-5 (APRS) goes live + issue #402 (RTL-SDR DC-spike on P25 control)
three-phase investigation. The Motorola MMR talker-alias path now lands:
#397 ports Motorola's vendor LCO 0x15 / 0x17 form for Phase 1 voice
channels (the standard TIA-102.AABF form #389 implemented doesn't match
what real MMR systems emit), #403 dispatches MAC PDUs on the Phase 2
voice chain so MMR Phase 2 talker-alias decodes too, and #409 backfills
source RID + ALGID / KID encryption from the voice channel by parsing
`GROUP_VOICE_CHANNEL_USER_ABBREVIATED` (opcode 0x01, previously
mis-named `OpMACPTT` and silently discarded). APRS reaches end-to-end
live: #401 adds the HDLC framer + receiver glue, #411 wires the
Bell-202 AFSK DSP frontend (IQ → FM → real resample → tone
discriminator → Mueller-Müller timing → NRZI → HDLC → AX.25 + APRS
info-field → events bus), so configuring `aprs.channels` with a serial
+ frequency lights up the bus, SQLite log, REST endpoint, and `/aprs`
web panel from #384 / #390. Issue #402 (RTL-SDR DC-spike pulls the
P25 control-channel offset estimator into the spike) lands in three
slices: #406 adds CCStats + per-sample recording-power diagnostics,
#408 mirrors the replay path through the production DDC and adds
state-evolution diagnostics, and #412 swaps in a decision-directed AFC
that defeats data-DC integration. Plus: #399 makes the P25 Phase 1
voice composer honour `trunking.systems[].p25_phase1_demod_mode` so
simulcast / LSM grants don't silently fail on FM-discriminator
hardcode; #398 widens the Windows RTL-SDR cold-boot recovery envelope
to 5 attempts with 200 / 400 / 800 / 1200 ms backoff and 150 ms
WinUSB settle (issue #395); #400 surfaces two silent-degradation
paths at startup (no `gain:` configured per SDR, conventional tone
gating with zero `sdr.sample_rate`); #413 routes Phase 1 TDMA-channel
grants to the Phase 2 voice chain; #407 promotes Motorola patch
member talkgroups over the super-group in CC Activity (issue #405);
and #396 adds a Markdown blog with per-category archives, RSS, and
SEO meta to the Pages site.

### Added

- **APRS Mic-E decoder.** Mobile-tracker packets (Kenwood TH-D74,
  Yaesu FT-3D, vehicle trackers) compressed-encode position +
  speed + course + altitude + a 3-bit message code across the
  7-byte AX.25 destination address and a 9-byte info field — a
  third the size of an uncompressed beacon, which is why every
  mobile tracker emits it. `aprs.DecodeWithDst(info, dst)` walks
  the Table 10.5 destination-char encoding (six latitude digits +
  message bits + N/S + lon-offset + W/E), then the §10.4
  speed/course interleaved encoding with the standard 800/400
  wrap corrections, then the optional base-91 `XXX}` altitude
  marker. Resulting `MicE` carries Latitude / Longitude / Speed
  (knots) / Course (deg) / SymbolTable / SymbolCode / MessageCode
  (`"M3 Returning"`, `"Emergency"`, custom-code variants) /
  Standard (std vs custom range) / Altitude (m) / HasAltitude /
  Comment. Latitude + Longitude also surface through the standard
  `Position` field so the storage row, the `/api/v1/aprs/packets`
  payload, and the `/aprs` panel pick the coordinates up without
  special-casing Mic-E. The bit-stream orchestrator
  (`aprs/receiver`) calls `DecodeWithDst` with the AX.25
  destination call so the path is wired end-to-end. Spec: APRS
  Protocol Reference 1.0.1 §10. Refreshes the `/aprs` panel
  empty-state copy now that the DSP frontend has shipped.
- **APRS DSP frontend — pipeline is now end-to-end.** Fifth and
  load-bearing slice of Phase 5 (#365 plan): the
  `internal/radio/aprs/afsk` package wires an `afsk.Receiver`
  per configured APRS channel between the iqtap broker and the
  bit-stream orchestrator that shipped in #401. Pipeline: IQ →
  `demod.FM` → real resampler down to 9600 sps → `demod.FFSK`
  tone discriminator (mark 1200 Hz, space 2200 Hz) → Mueller-
  Müller symbol-timing recovery → DC-tracking slicer → NRZI
  decode → HDLC framer → AX.25 + APRS info-field parse →
  `events.KindAPRSPacket`. New top-level `aprs.channels` config
  schema (`internal/config.APRSChannelConfig`, mirroring
  `paging.pocsag`); daemon constructs one receiver per entry,
  subscribes each to its SDR's iqtap broker via the standard
  spawn closure. `Stats()` surfaces IQ-samples-seen + bits-
  emitted; the bit-stream layer's frame counters remain reachable
  via `Inner().Stats()`. Operators add an entry like
  `serial: antenna-pi, frequency_hz: 144_390_000` and packets
  start landing on the bus, the `aprs_log` SQLite table,
  `/api/v1/aprs/packets`, and the `/aprs` web panel.
  Tests cover NRZI round-trip (transition / no-transition
  polarity, clamping, reset), receiver option validation,
  Process ctx-cancel + nil-input + clean-close, and stats
  counter accumulation. The synthetic IQ end-to-end test is
  currently `t.Skip`-ped pending a captured `samples/aprs/`
  fixture (same posture as POCSAG #378 — the receiver code is
  exercised by the unit-level coverage above and the orchestrator
  tests from #401).
- **P25 Phase 2 traffic-channel metadata backfill (issue #376
  follow-up).** Resolves the symptoms surfaced by @er-imagery's
  2026-05-28 MMR field test: Phase 2 grants on encrypted
  talkgroups arrived with `src=0` + `enc=false`, ALGID/KID never
  populated, and `composer: p25p2 talker alias` log lines never
  fired — even after #403 wired alias dispatch into the voice
  chain. Root cause: the MAC opcode constant `OpMACPTT = 0x01`
  was a fictional name; the real TIA-102 / SDRTrunk opcode at
  0x01 is `GROUP_VOICE_CHANNEL_USER_ABBREVIATED`, the in-call
  broadcast that carries SOURCE_ID + SVC_OPTIONS on the traffic
  channel during an active call. Real MMR PDUs at 0x01 were
  being parsed as "MAC PTT" and discarded.
  - `phase2.OpMACPTT` is removed and replaced by
    `phase2.OpGroupVoiceChannelUserAbbreviated = 0x01`. New
    `OpGroupVoiceChannelUserExtended = 0x21` covers the SUID-
    extended variant.
  - New `phase2.GroupVoiceChannelUser` struct +
    `MACPDU.AsGroupVoiceChannelUser()` accessor parses the
    SDRTrunk-confirmed layout: SVC_OPTIONS at payload[0],
    GROUP_ADDRESS at payload[1..2], SOURCE_ADDRESS at
    payload[3..5].
  - New `events.KindCallSourceUpdate` event +
    `trunking.CallSourceUpdate` payload + `VoicePool.UpdateSource`
    method + `Engine.handleCallSourceUpdate` handler form the
    backfill path: composer publishes, engine patches
    `ActiveCall.Grant.SourceID/.Encrypted`, republishes with the
    call's identity. `AffiliationTracker` subscribes so RID
    chips populate from the backfilled source.
  - The voice composer's Phase 2 chain now also dispatches
    in-call `OpEncryptionSync` (existing parser, just hooked up)
    via the existing `KindCallEncryption` event, mirroring the
    Phase 1 LDU2 path. ALGID/KID flow onto the active call as
    the EncryptionSync PDU arrives.
  - Diagnostic safety net: one Info log line per (opcode, MFID)
    per call —
    `composer: p25p2 mac pdu system=… serial=… opcode=… mfid=…
    payload_len=…` — so if MMR emits a vendor opcode we still
    don't dispatch (e.g. a different talker-alias opcode), the
    next field test pinpoints exactly what we saw.
  - Pre-existing `phase2.OpGroupVoiceChannelUserExt = 0x46` is
    renamed to `OpUnitToUnitGrantUpdateAbbreviated` to match
    its actual TIA-102 / SDRTrunk identity. No parser was
    wired to it; the rename is name-only.
- **P25 Phase 2 voice-channel talker-alias decode.** Resolves the
  follow-up half of #376: on Motorola MMR (and any Phase 2 system
  whose CC never emits talker-alias PDUs), display names ride MAC
  sub-frames that interleave with voice sub-frames on the traffic
  channel. The voice composer's Phase 2 chain now runs the same
  MAC-PDU dispatch the CC does — refactored into the new exported
  `phase2.DecodeSuperframeMACPDUs` — and publishes
  `events.KindTalkerAlias` when a fragment sequence completes. The
  CC's per-channel FEC config (trellis / RS / interleave /
  scrambler mode + 44-bit PN44 seed) rides on the published Grant
  via a new `trunking.P25Phase2Decode` field so the composer can
  decode MAC PDUs without owning a CC reference. Field-reporter
  re-test on MMR is the real verifier; #397's Phase 1
  Motorola-form path is unchanged.
- **APRS HDLC framer + receiver.** Fourth slice of Phase 5 (#365).
  `internal/radio/aprs/hdlc` is the bit-stream → frame-bytes
  layer: sliding-flag detector with bit-stuffing reversal,
  shared-flag packing tolerance, and 7+-ones abort sequence
  handling. `internal/radio/aprs/receiver` is the orchestrator
  that threads bits through the framer, parses each emitted
  frame with `ax25.Parse`, decodes the info field with
  `aprs.Decode`, and publishes one `events.KindAPRSPacket` per
  successfully-decoded UI frame. The bus payload is a
  `storage.APRSPacket` carrying the AX.25 envelope + APRS
  sub-type label + summary + (for position-bearing types)
  lat/lon, so the SQLite log + REST endpoint + `/aprs` web
  panel from #384 light up the moment a DSP layer pushes wire
  bits at `receiver.Push`. `DropBadFCS` / `DropNonUI` opt-ins;
  in/parsed/CRC-failed/emitted counters for future `/metrics`.
  See [docs/aprs.md](docs/aprs.md).

### Fixed

- **P25 Phase 1 voice chain now honours `p25_phase1_demod_mode`
  (issue #356 follow-up, reporter @v2maldo).** The per-call P25
  Phase 1 voice receiver was hardcoded to the C4FM
  FM-discriminator path regardless of the system-level
  `trunking.systems[].p25_phase1_demod_mode` setting. On a
  simulcast / LSM site the control channel decoded fine (the
  ccdecoder connector already honoured the setting) but every
  voice grant landed in an FM-discriminator that couldn't sync on
  LSM-modulated dibits — the LDU sink never fired, the
  frame-activity counter from #356's earlier fix never advanced,
  and the watchdog reaped the call at `call_timeout_ms` with an
  empty WAV. The mode string is now plumbed through
  `trunking.Grant` and the voice composer passes it into
  `p25p1rx.Options.DemodMode`. Empty / unrecognised values warn-log
  and fall back to C4FM so a typo doesn't silently kill a
  previously-working system.
- **RTL-SDR cold-boot stall on Windows: wider recovery envelope for the
  most stubborn clone dongles (issue #395).** A Windows 10 reporter on
  v0.2.4 still hit `rtlsdr: init baseband: init baseband step 0 ...
  ERROR_GEN_FAILURE` after the prior #382 + #393 fixes — warmup succeeded
  but the byte-identical step 0 of `InitBaseband` failed, and all three
  attempts of the previous 3-attempt / 100 ms+200 ms backoff envelope
  also failed. The open-time bring-up envelope now runs 5 attempts (4
  resets) with exponential backoff (200 / 400 / 800 / 1200 ms), and the
  WinUSB `Reset()` settle grows from 50 ms to 150 ms — both targeted at
  Windows USB-stack timing for the wedged-firmware recovery path.
  Healthy dongles still open on attempt 0 with zero delay; only dongles
  that actually need recovery pay the new costs. The surfaced hint for
  `ErrPipeStalled` now also recommends unplugging the dongle for 10 s
  before re-plugging (which physically clears the firmware state) and
  references the issue for users hitting this after a Windows
  sleep/resume.

### Changed

- **Operator-visible warnings for two silent-degradation paths
  surfaced by issue #356 triage.** Both fix observability gaps
  rather than behaviour, so a working config keeps working but a
  misconfigured one now logs a single line at startup pointing
  the operator at the fix.
  - `sdr: no gain configured for device ... use \`gain: auto\` for
    AGC or a specific tenth-dB value` — fires once per device that
    has a `sdr.devices[]` entry but no `gain:` key. The librtlsdr
    default isn't safe across every tuner / antenna / LNA chain;
    on some clones it leaves the SDR deaf and the symptom looks
    like a broken voice chain. See [docs/hardware.md](docs/hardware.md).
  - `conv: tone gating configured but scanner sample rate is zero;
    tone gate disabled` — fires when a conventional-scanner channel
    has `tone.mode: ctcss` or `dcs` but `sdr.sample_rate` is
    unset. The channel previously appeared in scan rotation with
    the gate silently bypassed (every signal passing), with no log
    explaining why CTCSS / DCS wasn't engaging.
- **Motorola voice-channel talker-alias decoder (issue #376
  follow-up).** Field-testing on a real MMR system surfaced that
  the standard TIA-102.AABF HEADER + BLOCK1 + BLOCK2 form #389
  implemented does NOT match what Motorola actually emits — real
  Motorola P25 systems use a vendor-specific variant: LCO 0x15
  header (talkgroup + variable block_count + sequence number) +
  N × LCO 0x17 data blocks (44-bit fragment each), with the
  reassembled message running the encoded alias through a
  proprietary lookup-table + accumulator cipher to recover the
  UTF-16 character stream. Replaced `StandardTalkerAliasBuf`
  with a clean-room Go port of the Motorola form
  (`phase1.MotorolaTalkerAliasBuf` +
  `phase1.decodeAliasBytes`). The voice composer dispatch on
  `IsTalkerAliasLCO` is unchanged at the call site; the Info
  log line now reads "composer: p25p1 motorola talker alias
  src=... alias=..." so operators can see decode events in the
  daemon log. The cipher LUT and arithmetic are treated as
  facts about Motorola's wire protocol (the algorithm is
  reverse-engineered prior art across multiple open-source
  decoders).

## [v0.2.4] — 2026-05-27

Phase-5 (APRS) + Phase-3 (POCSAG) + Phase-1 (Radio IDs) feature-density
follow-up to v0.2.3. The APRS scaffold landed (events bus / SQLite log /
REST / web panel — #384) and immediately got its protocol layer
(pure-Go AX.25 frame parser + APRS info-field decoder — #390), with
the Bell-202 AFSK DSP receiver as the remaining follow-up. POCSAG
closed end-to-end with the DSP receiver + daemon wiring (#378), so a
tuned SDR's IQ now flows demod → bit-slicer → syncer → page event →
SQLite log / REST / web panel without further plumbing. Radio IDs
landed in three slices: the `RIDDB` alias catalogue + REST + gRPC +
`/rids` web panel mirroring `TalkgroupDB` (#387), the standard
TIA-102.AABF P25 voice-channel talker-alias LC decoder (LDU1 LCOs
0x15 / 0x16 / 0x17 — #389) closing the second half of issue #376, and
a docs pass under [docs/radio-ids.md](docs/radio-ids.md). One-dongle
deployments got more powerful: the `role: wideband` channelizer now
hosts P25 Phase 1 and Phase 2 control channels alongside DMR T2/T3
(#385), and a new "virtual voice pool" (#386) follows trunked voice
grants whose frequency lands inside the wideband IQ window — so a
single SDR can cover P25 CC + voice end-to-end. The wideband engine
also routes through the iqtap broker so the spectrum view works on
wideband-only deployments (#377). Two more Windows RTL-SDR cold-boot
stall paths now self-recover: #382 classifies the
`ERROR_GEN_FAILURE` NAK as `ErrPipeStalled` and clears the control
halt, and #393 makes WinUSB `Reset` re-open the device handle
(matching `libusb_reset_device`) and allows up to two settles during
open. Plus polish: r82xx PLL nint encoding limit widened to 268 so
V4-class dongles tune above ~140 MHz on the 16 MHz xtal (#391,
closes #264), CC Activity super-group patches finally render member
counts (#392, closes #374), and the misleading "voice pool full"
message is replaced with an actionable startup WARN pointing at
`docs/hardware.md` when no `role: voice` SDR is attached (#383,
closes #379).

### Added

- **AX.25 frame parser + APRS info-field decoder.** Third slice
  of Phase 5 (#365), the protocol layer that plugs into the
  bus/log/REST/UI scaffolding from #384. Pure-Go AX.25 frame
  parser (`internal/radio/aprs/ax25`): 7-byte address packing,
  up to 8 digipeater path entries, HDLC CRC-16-CCITT validation,
  conventional `W1AW-9` / `WIDE2-1*` display helpers. Plus an
  APRS info-field decoder (`internal/radio/aprs`) for positions
  (`!`, `=`, `/`, `@`), messages (`:`) with ack/rej + bulletins,
  status (`>`); Mic-E / weather / telemetry / object types are
  type-tagged with payloads stashed for follow-up decoders. The
  DSP receiver (Bell-202 AFSK demod → HDLC de-stuff → frame
  delivery → bus event) is the next focused PR. See
  [docs/aprs.md](docs/aprs.md).
- **Radio IDs as first-class entities (#387, #376).** New
  `trunking.RIDDB` operator-configured alias catalogue mirroring
  `TalkgroupDB`: per-system `rid_alias_file` (CSV or JSON, dispatched
  by extension) carrying `Decimal/DEC/ID` plus optional `Alias`,
  `Description`, `Tag`, `Group`, `Owner`, `Priority`, `Lockout`,
  `Watch`, `Icon` columns. `AffiliationTracker` gained `TalkerAlias`,
  `TalkerAliasAt`, `CallCount`, `FirstSeen` on `UnitActivity` and
  now subscribes to `KindTalkerAlias`. New HTTP routes `GET
  /api/v1/rids`, `GET /api/v1/rids/{id}`, `GET
  /api/v1/rids/{id}/history` (backed by `HistoryFilter.SourceID`),
  and `PATCH /api/v1/rids/{id}`. New gRPC `RIDService`
  (`ListRIDs` / `GetRID` / `ListRIDHistory`). New `/rids` web panel
  with the configured ∪ live merge, last-50-calls detail modal, and
  write-mode mutation controls. CC Activity RID chips are now
  clickable links into the detail view. See [docs/radio-ids.md](docs/radio-ids.md).
- **Standard P25 talker-alias voice-channel decoder.** Follow-up to
  #387 closing the second half of issue #376. Phase 1 LDU1 Link
  Control opcodes 0x15 (HEADER) / 0x16 (BLOCK1) / 0x17 (BLOCK2) are
  now reassembled by `phase1.StandardTalkerAliasBuf` (one buffer
  per active voice chain) and published as `KindTalkerAlias` events
  with the call's SourceID; the affiliation tracker stamps the
  decoded alias onto the RID row so it surfaces in
  `/api/v1/rids` and the Radio IDs panel. The existing Motorola
  vendor TSBK form (control channel) is unchanged. Phase 2 voice-MAC
  alias dispatch remains a follow-up.
- **APRS bus event + SQLite log + REST + web panel.** Second
  slice of Phase 5 (#365), building on the protocol layer from
  #381. New `events.KindAPRSPacket` bus event, `aprs_log`
  SQLite table, `storage.APRSLog` bus subscriber (mirrors
  `PagerLog`), `GET /api/v1/aprs/packets?limit=N` REST endpoint,
  and `/aprs` web panel rendering the live packet list (received
  time, src → dst + path, type, body, lat/lon, CRC-OK flag with
  yellow highlight on CRC failure). DSP wiring (Bell-202 AFSK
  demod → HDLC de-stuff → AX.25 framer → packet decoder → bus)
  is the remaining piece and lands in a focused follow-up PR.
- **POCSAG DSP receiver + daemon wiring.** Third slice of Phase 3
  (#365). New `internal/radio/pager/pocsag/receiver` package wires
  the FM demod → rational resampler → integrator-and-slicer → bit
  syncer pipeline together so a tuned SDR's IQ stream now flows
  end-to-end into the pager bus event. New `paging.pocsag` YAML
  section pins SDRs to paging frequencies (`serial` +
  `frequency_hz` + optional `baud_hz`). The daemon retunes the
  SDR on startup, subscribes to the iqtap broker, and runs one
  receiver per configured entry as a non-essential spawn (so a
  misconfigured paging frequency doesn't bring down the trunking
  pipeline). Synthetic-IQ end-to-end test is skipped pending
  real captured fixtures; receiver API surface (Options
  validation, ctx cancel, nil input) is unit-tested. See
  [docs/pocsag.md](docs/pocsag.md) for the configuration knob and
  what's pending (timing-recovery tuning against real fixtures,
  multi-channel-from-one-SDR DDC, FLEX).
- **Wideband channelizer hosts P25 Phase 1 + Phase 2 control
  channels (#385).** A single SDR pinned to a centre frequency can
  now host a P25 trunked control channel inside the wideband
  channelizer, alongside the existing DMR Tier II and Tier III state
  machines. The per-channel wiring uses a small `narrowbandReceiver`
  interface (`Process([]complex64)`) so the engine itself stays
  protocol-agnostic; P25 Phase 1 honours the system's
  `p25_phase1_demod_mode` (C4FM vs CQPSK / LSM) and any
  operator-supplied `P25BandPlan` entries, and P25 Phase 2 reuses the
  existing trellis / RS / interleave / scrambler / clock-mode knobs
  and the PN44 seed derivation so a wideband CC tap decodes
  identically to a dedicated CC dongle. Config validator accepts
  protocol `p25` / `p25-phase2` for wideband channels with the same
  control-channel-membership rule that already applies to DMR Tier
  III. Docs and `config.example.yaml` updated with worked P25
  examples. Voice grants on these protocols still route to the
  daemon's existing physical voice pool — the virtual voice pool
  (next bullet) covers in-window grants.
- **Virtual voice pool on the wideband dongle (#386).** A wideband
  dongle can now also follow trunked voice grants whose frequency
  lands inside its IQ window — DMR Tier III, P25 Phase 1, P25
  Phase 2 — without a separate `role: voice` SDR. New
  `internal/sdr/wbvoice` package: `VirtualTuner` implements both
  `trunking.Tuner` (`SetCenterFreq`, `CanTune`) and
  `composer.IQSource` (`StreamIQ`, `SampleRateHz`). Each tap
  subscribes to the wideband dongle's iqtap broker on demand, runs a
  single-tap DDC at the (target − wideband) offset, and emits 48
  kHz IQ to the composer's existing P25 / DMR voice chains — no
  changes to the receivers themselves. `voicepool.FindFreeForFrequency`
  consults an optional `FrequencyChecker.CanTune` on each free
  device, so a voice grant outside the wideband window passes over
  a virtual tuner and lands on the physical `role: voice` SDR when
  one is configured. One SDR end-to-end for any system whose
  carriers fit in a single 2.4 MHz band.
- **Wideband engine routes IQ + tuning through the iqtap broker
  (#377).** Wideband-only DMR Tier 2 deployments (single SDR,
  `role: wideband`, multiple T2 systems) couldn't render the
  spectrum waterfall because the engine consumed `StreamIQ` from
  the raw device and never fed the broker's fan-out. The wideband
  engine now takes the broker (mirroring the CC decoder wiring) so
  the spectrum panel works on wideband-only deployments. Also seeds
  each broker's sample-rate cache in `wrapIQBrokers` from
  `cfg.SDR.SampleRate` — the pool programs the rate on the raw
  device before the broker wraps it, so `Broker.SetSampleRate`'s
  cache path never ran and frames stamped `sample_rate_hz=0` for
  every device.

### Fixed

- **RTL-SDR cold-boot stall on Windows: deeper recovery for wedged
  clone dongles (issue #333).** The previous fix (#382) mapped
  `ERROR_GEN_FAILURE (0x1F)` to `ErrPipeStalled` and ran one
  clear-halt + re-claim retry, which recovers a stale endpoint halt
  but not a wedged firmware state from a prior crashed process.
  WinUSB `Transport.Reset()` now matches what `libusb_reset_device`
  does on Windows: clear-halt the control endpoint, drop the WinUSB
  handles, then re-open the device via `CreateFile` +
  `WinUsb_Initialize` (a true device-object re-bind, not just a pipe
  reset). The open-time bring-up envelope now allows up to two such
  resets per `Open` with 100 ms / 200 ms backoff, giving clones that
  need two settles to come back a chance to recover before surfacing
  the Zadig / port-choice / `gophertrunk sdr doctor` hint. Healthy
  dongles still open with zero resets and zero delay.
- **RTL-SDR cold-boot stall on Windows now self-recovers (#382).**
  Clone dongles (and some power-marginal hubs) latch the first
  USB_SYSCTL=0x09 vendor-OUT write, then NAK the byte-identical
  second write in `init baseband` step 0 with `ERROR_GEN_FAILURE
  (0x1F)`. The Linux equivalent (`EPIPE`) was already covered by the
  bring-up reset+retry envelope; the Windows path wasn't because (a)
  `ERROR_GEN_FAILURE` wasn't classified as resetable, and (b) the
  WinUSB `Transport.Reset()` was a no-op. WinUSB now clears the
  control-pipe halt via `WinUsb_ResetPipe(0)` (USB
  `CLEAR_FEATURE(ENDPOINT_HALT)`), the new `usb.ErrPipeStalled`
  sentinel keys the existing retry envelope, and a clone-dongle hint
  pointing at Zadig / port choice / `gophertrunk sdr doctor` is
  appended when the second attempt still fails.
- **r82xx setPLL nint encoding limit widened to 268 (closes #264).**
  The overflow guard used `0x3F + 13 = 76`, which only accounts for
  ni's 6-bit width and ignores that si's 2 extra bits also encode
  part of nint (register 0x14 = `ni | si<<6`; nint = `13 + 4*ni + si`).
  The real encoding cap is `13 + 4*0x3F + 0x3 = 268`. With R820T /
  R820T2's 28.8 MHz xtal the VCO range capped nint near 67 so the
  bug was latent; PR #266's correct R828D xtal (16 MHz) halves
  `pllRef` and pushes nint up to ~121 — the guard then rejected
  tunes above ~140 MHz on the V4 dongle, e.g. 153.5875 MHz →
  nint=78 overflows. Regression test pins the nint=78 math for the
  reporter's frequency.
- **CC Activity panel renders super-group patches with member counts
  (closes #374).** `eventToDTO` had no case for `trunking.Patch`,
  so the payload fell through to default and was JSON-marshalled
  with Go's PascalCase names (`SuperGroup`, `Members`, `Add`). The
  CC Activity panel reads snake_case fields (`super_group`,
  `members`, `add`) and was getting `undefined` for all of them —
  hence "super-group 0 · add" on every patch. New `PatchDTO`
  mirrors the established DTO pattern (snake_case JSON tags),
  `eventToDTO` dispatches to it, and the frontend cancel-detect
  honours the wire field (`add: false`) alongside the existing
  legacy fallbacks. SSE wire shape pinned by test using the values
  from the issue report.
- **Actionable "voice pool empty" diagnostic when no `role: voice`
  SDR is attached (closes #379).** When an operator booted with a
  trunked system but no voice SDR, every grant logged "voice pool
  full but no actives" — which read as "pool full" while the pool
  was in fact empty, and gave no clue that a second SDR or a
  wideband channelizer is required. `HandleGrant` now distinguishes
  the two cases: empty pool logs a one-shot actionable WARN
  pointing at [docs/hardware.md](docs/hardware.md) and drops
  subsequent grants at DEBUG; the genuine impossible state
  (devices > 0 but no actives) becomes Error so the bug stays
  visible. A new one-shot startup WARN from `Daemon.Run` surfaces
  the problem before the first grant arrives. Non-trunked
  deployments (POCSAG, conventional FM scanner, wideband T2
  capture-only, baseband recording) still run cleanly because the
  warning is gated on `len(systems) > 0`.

## [v0.2.3] — 2026-05-26

The "multi-consumer SDR + new operator panels" release. The new
iqtap broker (#365) made multi-consumer SDR fan-out possible without
forking IQ streams in each subscriber, which immediately unlocked a
batch of new operator-console capabilities: a Constellation viewer
that renders live IQ scatter alongside decode (#370), a CC Activity
panel that filters the events stream down to control-channel chatter
(#369), a UI-managed Bookmarks frequency manager backed by a new
SQLite table (#368), spectrum-panel click-to-tune + bookmark markers
(#371), a Hamlib `rigctld` TCP server for external amateur tooling
(Cloudlog, GridTracker, PSTRotator, `rigctl(1)` — #367), and a
remote `rtl_tcp` driver mounting any number of remote SDR servers as
virtual tuners alongside locally-attached USB dongles (#366). POCSAG
paging landed as the first two slices of Phase 3 of the
trunking-adjacent feature plan (#365): the BCH(31,21) FEC + codeword
wrapper + numeric / alphanumeric message decoders shipped as a
pure-protocol slice (#372), and the syncer + page assembler + bus /
log / REST / web panel scaffold plugged it into the operator surface
(#373); the DSP receiver wiring landed the following day in v0.2.4.
The wideband channelizer gained DMR Tier III control-channel support
(#363) and per-channel `ClockGain` matching the dedicated-dongle
path (#364) so wideband-hosted DMR repeaters lock as cleanly.
Windows 11 RTL-SDR driver-binding woes got a diagnostic answer
(`gophertrunk sdr doctor` — #359) since Windows has no equivalent
of `USBDEVFS_DISCONNECT`. Airspy R2 open ordering on Windows fixed
(#358) so it stops failing with `device disconnected` when
`sdr list` did detect the dongle. And the stuck voice-chain footgun
(#356) closed: the four voice composers now gate `Engine.Touch` on
actual decoder progress so the 30 s inactivity watchdog can fire
and release the bound voice SDR when transmission stops.

### Added

- **POCSAG syncer + page assembler + bus event + SQLite log +
  web panel.** Second slice of Phase 3 (#365), building on the
  protocol layer landed in #372. The new `pocsag.Syncer`
  consumes a packed bit stream, locks on the POCSAG sync
  codeword (with polarity-inverse fallback so a flipped FM
  demod still works), carves batches, decodes through
  BCH(31,21), and reassembles pages by correlating address +
  message codewords. Pages publish on a new
  `events.KindPagerMessage` bus event; a new SQLite `pager_log`
  table persists them; `GET /api/v1/pager/messages?limit=N`
  returns the most recent rows; `/pagers` web panel renders the
  live list (received time, RIC, function code, encoding, body,
  bit-error count). DSP wiring (FM demod → bit slicer →
  `Syncer.Push`) is the remaining piece and lands in a focused
  follow-up PR. See [docs/pocsag.md](docs/pocsag.md).
- **POCSAG paging protocol layer.** First slice of Phase 3 of the
  trunking-adjacent feature plan (#365). Adds BCH(31,21)
  encode/decode (corrects up to 2 bit errors per codeword) plus
  the POCSAG-specific codeword wrapper (sync `0x7CD215D8` + idle
  `0x7A89C197` recognition, trailing overall-parity check,
  address/message/function decoding), batch carve-up (sync + 16
  codewords × 8 frame slots, full-RIC reconstruction from the
  18-bit address-codeword field + slot index), and the
  numeric (CCIR 584 extended BCD: 0-9, *, U, space, -, ), ( ) +
  alphanumeric (7-bit LSB-first ASCII) message decoders. Pure
  protocol — the DSP wiring (FM demod → bit slicer → sync
  detector → batch decoder → bus event → SQLite log → web/TUI
  panel) lands in a focused follow-up PR. See
  [docs/pocsag.md](docs/pocsag.md).
- **Spectrum panel: click-to-tune + bookmark markers.** Closes the
  click-to-tune TODO from the bookmarks PR (#368). Clicking
  anywhere on the waterfall canvas now posts the bin's centre
  frequency to a new `POST
  /api/v1/spectrum/devices/{serial}/tune` endpoint and the SDR
  retunes immediately. The bookmarks list is polled every 30 s
  and rendered as small cyan ticks across the top of the
  waterfall wherever a bookmark frequency falls inside the visible
  band. Tune goes through the iqtap broker so the frequency stays
  coherent across the spectrum, constellation, rigctld, and CC
  decoder views, and survives `pool.Reacquire`.
- **Constellation viewer.** New web panel at `/constellation` that
  renders a live 2D scatter of decimated IQ samples (2 ksps
  default). Brighter dots = newer samples; reference rings at
  |z|=0.5 and |z|=1.0; per-frame dBFS energy banner. Identifies
  signal shape visually — PSK clusters, FSK arcs, AM rotation,
  noise circles, DC bias, frequency-offset spirals — without
  launching a separate SDR receiver alongside GopherTrunk. Builds
  on the iqtap broker (PR #365) so multiple subscribers share the
  same SDR's IQ stream without disturbing decode.
  `internal/dsp/diag` adds a pure-Go stride decimator + per-frame
  energy estimator; `WS /api/v1/diag/iq?device=...&rate=2000`
  exposes it. See [docs/constellation.md](docs/constellation.md).
- **CC Activity panel.** New web panel at `/cc` that filters the
  events stream down to control-channel chatter: voice grants,
  affiliations, registrations, patches / dynamic regroups, talker
  aliases, CC lock / loss, and call start/end. Per-row rendering
  pulls the right detail out of each payload (talkgroup + source
  + frequency + tags for grants, member count for patches,
  response codes for affiliations, the alias string for talker
  aliases). Kind + system substring filters narrow the view; a
  pause button freezes the display without disconnecting the
  bus. Pure filter view over events already on the bus — no new
  bus kinds or storage.
- **Bookmarks / frequency manager.** UI-managed conventional
  channel list (marine VHF, NOAA weather, FRS/GMRS, repeater
  outputs, public-safety conventional fall-backs) backed by a new
  `bookmarks` table in the daemon's SQLite database. Each row
  carries name, frequency, mode, optional CTCSS / DCS, freeform
  notes, and an operator-defined group tag. REST endpoints under
  `/api/v1/bookmarks` (read open; create / update / delete gated
  the same as every other write route); web panel at
  `/bookmarks`. Mutations publish `bookmark.{created,updated,
  deleted}` events on the bus so SSE / WS subscribers refresh
  without polling.
- **Hamlib `rigctld` TCP server.** Opt-in (`api.rigctld:
  "127.0.0.1:4532"`) endpoint speaking the standard rigctld wire
  protocol so external amateur-radio tooling (Cloudlog,
  GridTracker, PSTRotator, satellite trackers, `rigctl(1)`) can
  read and set the control SDR's frequency without learning the
  GopherTrunk REST API. Implements the ~10 commands real clients
  send (`F` / `f`, `M` / `m`, `V` / `v`, `T` / `t`, `chk_vfo`,
  `dump_state`, `q`); unknown commands return `RPRT -1` per
  Hamlib's "unsupported" convention. RX-only backend — `set_ptt 1`
  is rejected. Tuning routes through the iqtap broker so external
  retunes stay coherent with the spectrum panel's frequency axis
  and survive USB-disconnect cycles. See
  [docs/rigctld.md](docs/rigctld.md).
- **Remote `rtl_tcp` SDRs.** A new `rtltcp` driver mounts any number
  of remote `rtl_tcp` servers as virtual tuners alongside locally-
  attached USB dongles. The driver speaks the well-known librtlsdr
  wire protocol (12-byte `RTL0` header, u8 IQ stream, 5-byte command
  packets) used by SDR++, Gqrx, and OpenWebRX, so any host running
  `rtl_tcp` can publish its dongle to the daemon. Configure under
  `sdr.rtl_tcp` in `config.yaml`; each entry carries `addr`,
  optional `serial`, `role`, `ppm`, `gain`, `bias_tee`, and
  `connect_timeout_ms`. Pool roles, broker fan-out, baseband
  recording, and the live spectrum panel all work against remote
  sources just like local ones. Plaintext on the wire — restrict
  to trusted networks or wrap with SSH/WireGuard/Tailscale. See
  [docs/hardware.md](docs/hardware.md).
- **`role: wideband` SDR devices — one dongle, many DMR Tier II
  repeaters and DMR Tier III control channels.** A single SDR pinned
  to a centre frequency now decodes every conventional DMR repeater
  AND a DMR Tier III control channel inside its IQ bandwidth (e.g.
  several 12.5 kHz carriers within a 2.4 MHz IQ window around
  453 MHz), no extra hardware needed. Add a `role: wideband` entry to
  `sdr.devices` with a `center_freq_hz` and a `channels: [...]` list
  binding each frequency to a `trunking.systems` entry; per channel,
  systems with `protocol: dmr-tier2` get a Tier II `ConventionalChannel`
  state machine, systems with `protocol: dmr` get a Tier III
  `ControlChannel` (channel frequency must match one of the system's
  `control_channels`). T2 and T3 can mix on the same dongle. The
  daemon's `internal/scanner/widebandt2` engine fans the dongle's IQ
  out via the `internal/dsp/tuner` package (DDC-per-channel or shared
  polyphase channelizer, picked by channel count). See
  [`docs/hardware.md` § Sharing one dongle across multiple repeaters](docs/hardware.md)
  and `samples/dmr-tier2-multichannel/`. Tier III voice grants still
  route through the existing physical voice pool (a `role: voice`
  SDR follows the call); decoding T3 voice directly on the wideband
  dongle via a virtual voice pool is the next planned step (landed
  in v0.2.4 as #386).
- **`gophertrunk sdr doctor` — per-dongle driver-binding report.**
  Many Windows 11 users reported their RTL-SDR dongles weren't being
  recognized despite appearing in Device Manager, mirroring the
  Linux kernel-driver collision fixed in v0.2.2. Windows has no
  equivalent of `USBDEVFS_DISCONNECT` (you can't programmatically
  rebind a USB function driver), so the fix is diagnostic rather
  than mechanical: a new `sdr doctor` subcommand walks the OS USB
  tree, reads the bound function driver via SetupAPI
  (`SPDRP_SERVICE` / `SPDRP_DEVICEDESC`) on Windows or the
  interface-0 sysfs symlink on Linux, and prints a row per dongle
  with an actionable next step (run Zadig; pick Interface 0 not
  the composite parent; re-target WinUSB instead of libusbK;
  blacklist `dvb_usb_rtl28xxu`; etc.). Read-only — safe to run as
  a regular user alongside a live daemon.
- **Smarter `WinUsb_Initialize` error on Windows.** The error now
  embeds the currently-bound driver name and points the operator at
  `sdr doctor`, replacing the generic "driver not bound? run Zadig"
  message that gave the user no insight into what to actually fix.
- **Windows 11 driver-binding troubleshooting section** in
  `docs/user-guide-windows.md` § 4.2, covering Core Isolation /
  Memory Integrity, Smart App Control, Driver Signature Enforcement,
  Windows Update DVB-driver re-binding, multi-dongle gotchas,
  composite-device interface selection, libusbK / libusb-win32
  mistakes, USB Selective Suspend, xHCI controller quirks,
  antivirus blocking, Windows S mode, and Group Policy device-install
  restrictions.

### Fixed

- **Wideband DMR receiver loop-gain now matches the single-channel
  ccdecoder path.** The Stage 2 / Stage 3 wideband engine was
  instantiating `dmr/receiver.Receiver` with the default
  `ClockGain: 0.05`, which the existing ccdecoder pipelines
  explicitly lowered (0.015 for Tier II, 0.025 for Tier III) because
  the default doesn't reliably lock the Mueller-Müller clock loop on
  T2/T3 symbol distributions. The wideband engine now picks the
  right value per channel based on the system's tier, so wideband-
  hosted DMR repeaters lock as cleanly as the dedicated-dongle path.
  Verified by a new in-package end-to-end test in
  `internal/scanner/widebandt2/engine_e2e_test.go` that feeds
  synthesized Voice LC Header IQ through the engine and asserts a
  grant event lands on the bus.
- **trunking/composer**: Voice chains no longer keep a call alive
  forever via an unconditional 1 s heartbeat. The four chains
  (P25 Phase 1, P25 Phase 2, DMR, NBFM) now gate `Engine.Touch` on
  actual decoder progress — an LDU / superframe / voice subframe /
  PCM batch — so the 30 s inactivity watchdog can fire and release
  the bound voice SDR when transmission stops. Before this fix a
  stalled decoder (simulcast garbage, vocoder hang) refreshed
  `LastHeardAt` every tick regardless of whether any voice frames
  were decoded, leaving the active call permanently locked on a
  single talkgroup and every subsequent grant logging "no voice
  device available for grant" (issue #356, reporter @KN4MSH).
- **config**: New `trunking.call_timeout_ms` knob lets operators
  tune the watchdog timeout (still 30 s by default). Useful on
  systems with consistently clean signaling (lower for snappier
  teardown) or chatty channels with long transmission pauses
  (higher). Issue #356.
- **airspy**: Defer `SET_SAMPLE_TYPE` from `Open()` to `StreamIQ()`,
  matching libairspy's open ordering (`GET_SAMPLERATES` IN first,
  no vendor OUT during open). Fixes Airspy R2 failing to open on
  Windows with `winusb: WinUsb_ControlTransfer OUT: usb: device
  disconnected` even though `sdr list` detected the device
  (issue #270, reporter @VA7DBI).
- **windows usb backend**: Stop folding `ERROR_GEN_FAILURE` into
  `ErrDeviceGone`. That conflation printed "usb: device
  disconnected" for what is actually a firmware NAK / stalled
  pipe / wrong-driver-bound condition, and actively misled the
  issue #270 reporter. The error now names the Win32 code and
  suggests re-binding via Zadig.

## [v0.2.2] — 2026-05-25

Operational-recovery + Mt Anakie follow-up release. The reporter in
issue #345 — a NESDR SMArt v5 dropping off the USB bus multiple
times per day — was the proving ground for a full USB-disconnect
recovery suite: the bulk-IN reaper-death channel now surfaces silent
stalls through the ccdecoder retry loop, control SDRs reacquire by
serial without a daemon restart, voice SDRs reacquire on grant-time
tune failure, and a new SDR-pool watchdog re-enumerates registered
drivers periodically so a missing serial is re-bound the moment it
reappears. The same Mt Anakie site exposed two more P25 control-
channel gaps that v0.2.1's BCH + TSBK fixes uncovered: the site
broadcasts the TDMA `IdentifierUpdate` opcode (0x33 — v0.2.1 only
wired the VUHF variant 0x34), and grants arrive on channel IDs
before the matching IDEN_UP TSBK lands, so a pending-grant ring
(plus a config-driven band-plan seed for sites that never broadcast
some IDs at all) now drains every grant against the freshly-applied
slot. P25 calls also surface ALGID / KID end-to-end — log lines,
TUI, and both web panels render the algorithm name (`0x84
(AES-256)` / `0x81 (DES-OFB)` / `0xAA (ADP/RC4)`) the instant the
LDU2 Encryption Sync lands rather than just an opaque `enc=true`
flag. Web operator-console polish: empty WACN / SystemID / RFSS /
Site fields in the system detail modal now explain *why* they're
empty (control-channel hunt state). Repo polish: README trimmed
from 2,826 → ~210 lines with the long-form Status and Roadmap
chapters extracted into their own pages, the docs nav surfaces
previously-orphan pages (launcher, live-edits, DMR encryption,
release process), and the Dockerfile bumps to `golang:1.25` so
builds stop silently downloading the newer toolchain at every run.

### Added

- **TDMA `IdentifierUpdate` (TSBK opcode 0x33) wired through the
  Phase 1 dispatcher (issue #345).** v0.2.1 added the FDMA-
  flavoured VUHF variant (0x34, channel IDs 2 / 3 / 4 / 6 / 7 /
  8 / 14 / 15); the Mt Anakie site survey confirmed it broadcasts
  IDEN_UP for id=10 only as the TDMA variant (0x33, covering ids
  0 / 1 / 5 / 9 / 11 / 12 / 13), which the dispatcher silently
  ignored. Every Phase 2 grant on a TDMA id was black-holing with
  `decode.error stage=no-bandplan`. `ParseIdentifierUpdateTDMA`
  mirrors the VUHF bit packing (the on-air frequency-field layout
  per TIA-102.AABF Table 14 is identical; only byte 0's lower
  nibble differs — channel-type code vs bandwidth code), and
  channel-type → bandwidth mapping covers the documented Phase 2
  codes (0x1 → 6.25 kHz, 0x2 → 12.5 kHz, 0x3 → 6.25 kHz). Mt
  Anakie id=10 + num=176 now resolves to 468.6125 MHz.

- **Per-channel-ID deferred grant queue (issue #345).** Grants
  that reference a `BandPlan` channel ID before the matching
  `IdentifierUpdate` TSBK lands are now held in a bounded ring
  (cap 4 per ID, 5 s TTL) instead of dropping with
  `decode.error stage=no-bandplan`. When the IDEN_UP arrives the
  ring drains and re-publishes every queued grant through
  `publishVoiceGrant` against the freshly-applied slot. Covers
  the race where IDEN_UP cadence is slower than the first grant
  after CC lock.

- **Config-driven P25 band-plan seed.** New `p25_band_plan` list
  on `SystemConfig` with `channel_id` / `base_hz` / `spacing_hz`
  / `tx_offset_hz` / `bandwidth_hz` fields, validated for range
  and duplicates. The Phase 1 pipeline factory calls
  `BandPlan.Apply` for each entry at startup so sites that never
  broadcast IDEN_UP for a given channel ID can still resolve
  grants. Over-the-air IDEN_UPs override seeded entries through
  the same `Apply` path — entries are a floor, not a ceiling.

- **P25 ALGID / KID encryption metadata surfaced end-to-end
  (closes #353).** Phase 2 was already populating `Grant.ALGID`
  / `KID` but nothing downstream consumed them; Phase 1 carried
  them as zero until the LDU2 Encryption Sync arrived after
  voice acquisition. A new `KindCallEncryption` event lets the
  voice composer publish ALGID/KID the instant the LDU2 lands;
  the engine updates the bound `ActiveCall.Grant` via a new
  `VoicePool.UpdateEncryption` helper and republishes through
  the events bus. Wire-format additions cover REST/SSE
  (`GrantDTO`, `CallEncryptionDTO`), gRPC (pb `Grant` message),
  the TUI client mirror, and the web SPA (`GrantDTO`,
  `CallRow`, new `CallEncryptionEvent`). A new P25 algorithm-
  name registry renders `0x84 (AES-256)` / `0x81 (DES-OFB)` /
  `0xAA (ADP/RC4)` uniformly across the log line, the TUI
  active-call flag column, and both web panels' pills + detail
  views. Storage schema already had the columns.

- **SDR-pool periodic watchdog + voice-pool reacquire hook
  (issue #345).** Following the control-SDR re-acquire path
  shipped in PR #349, the same recovery now extends to voice
  dongles and to idle devices. When `VoicePool.Bind`'s
  `SetCenterFreq` fails — typically because a voice dongle
  disconnected between calls — the pool's new reacquire hook
  (wired by the daemon to `sdr.Pool.Reacquire`) re-opens the
  device by serial, swaps the fresh `Tuner` into the
  `VoiceDevice`, and retries the tune once before the call
  drops. Independently, the SDR pool runs a periodic watchdog
  (`sdr.watchdog_interval_ms`, default 30 s, opt-out via `-1`)
  that re-enumerates registered drivers, surfaces missing
  serials via `KindSDRDetached`, and calls `Pool.Reacquire` the
  moment a previously-missing serial reappears — so the next
  consumer touches a live handle instead of paying the
  reacquire round-trip mid-use. The watchdog only acts on the
  missing → reappeared transition: continuously-present devices
  are never touched.

- **Empty WACN / SystemID / RFSS / Site fields on the web
  systems detail modal now explain *why* they're empty (#342).**
  Those four identity fields populate from decoded P25 status
  broadcasts (TSBK 0x3A / 0x3B), not config, so they're empty
  until the control channel is locked and the broadcasts
  arrive. The detail modal used to show a bare em-dash, leaving
  operators unable to tell config mistakes from "not yet
  decoded". The scanner snapshot (`hunting` / `locked` / other)
  now drives per-field hint copy through a new `DetailField`
  `emptyHint` prop, pulled from the Systems-panel poll so the
  hint stays correct without visiting the Scanner page first.

### Fixed

- **Control SDR USB disconnect / re-enumerate now recovers
  in-process without a daemon restart (issue #345).** PR #348
  surfaced the silent-stall failure through the ccdecoder retry
  loop and escalated to a fatal exit so systemd / docker could
  restart the process; on a dongle that disconnects repeatedly
  (the reporter in issue #345 saw multiple drops per day on a
  NESDR SMArt v5) that meant the daemon kept exiting. The retry
  loop now first asks the `sdr.Pool` to re-acquire the control
  device by serial: best-effort close of the dead handle,
  driver re-enumerate, fresh `Open()` by the new USB index,
  sample rate + per-device Hint (PPM, gain, bias-tee) re-
  applied to the new handle, `Device` swapped in place in the
  `PoolEntry`, and `KindSDRDetached` + `KindSDRAttached` events
  republished so the API / TUI / web snapshot reflect the
  swap. `cchunt.Supervisor.SwapTuner` feeds the fresh handle to
  in-flight hunters by closing any armed retune channels so the
  next hunt round picks up the new tuner. The existing
  1 s / 2 s / 5 s / 10 s retry budget still applies — if the
  device stays gone after re-enumerate or `Open` fails, retries
  exhaust and the daemon still escalates to a clean fatal for
  the supervisor restart path.

- **`ccdecoder.StreamIQ` open-time errors now classify as
  `ErrIQStreamClosed` so the retry loop recovers (issue #345).**
  After the v0.2.1 retry path shipped, the reporter still saw
  the daemon's ccdecoder silently exit on a real RTL-SDR USB
  disconnect: the reaper would die mid-stream returning
  `ErrIQStreamClosed`, the retry loop would rebuild the decoder
  against the same dead `Tuner`, the rebuilt `StreamIQ` would
  fail with `usb: device disconnected` at the control-transfer
  `ResetBuffer` step, and the retry loop's `errors.Is` against
  `ErrIQStreamClosed` would miss. Non-context `StreamIQ` open
  errors are now wrapped as `%w: %w` against
  `ErrIQStreamClosed` so both shapes (mid-stream EOF and
  open-time `device disconnected`) classify the same way; the
  underlying error stays inspectable via `errors.Is` for the
  root cause.

- **USB bulk-IN reaper death now surfaces to the decoder
  instead of stalling silently (issue #345).** The shared
  bulk-IN reaper goroutine on every platform (linux / windows
  / darwin) used to exit silently when every URB became
  unrecoverable, leaving the driver's IQ consumer channel
  neither sending nor closed. ccdecoder's `select` blocked on
  the dead stream forever, `decoder.pump` stopped running, and
  every downstream `events.Publish` froze — the daemon went
  idle at 0% CPU with `gophertrunk_events_total` counters
  stuck, alive but inert. A new
  `usb.Transport.StartBulkIn.onStreamDead` callback fires
  exactly once when the reaper exits without `StopBulkIn`;
  each hardware driver (purego / airspy / airspyhf / hackrf)
  wires it into its existing cleanup goroutine via a
  `streamDead` channel + `sync.Once` so the consumer channel
  always closes — exactly once — on either ctx-cancel or
  reaper death. `ccdecoder.Run` then returns
  `ErrIQStreamClosed` on unexpected EOF, hitting the backoff-
  driven restart loop above (1 s / 2 s / 5 s / 10 s, with the
  attempt counter reset after a 60 s healthy run).

### Changed

- **README trimmed from 2,826 → ~210 lines.** The long-form
  "Status & known gaps" extracted into a new `docs/status.md`,
  the "Roadmap" into a new `docs/roadmap.md`, and the inline
  "Recently shipped" log removed because it duplicated
  `CHANGELOG.md`. Chapters that already live under `docs/` (TUI,
  Web console, API auth, FEC opt-outs, Repository layout,
  encyclopedic Quick Start) are now linked rather than
  duplicated. Nav (`docs/_data/nav.yml`) surfaces previously-
  orphan pages: launcher, live-edits, DMR encryption, release,
  and the new status / roadmap pages. Added Jekyll front matter
  to `launcher.md` and `dmr-encryption.md` so they render under
  the right group.

- **Dockerfile bumped `golang:1.24` → `golang:1.25`** to match
  `go.mod`'s Go 1.25.0 / toolchain 1.25.10. Builds were
  silently downloading the newer toolchain at every run.
  `CONTRIBUTING.md` bumps "Go 1.24+" → "Go 1.25+" to match.
  `.gitignore` now excludes `.env` / `.env.*` since
  contributors occasionally drop streaming credentials there
  while iterating. A new minimal
  `.github/pull_request_template.md` covers scope, test plan,
  breaking changes, and the docs/CHANGELOG checklist.

## [v0.2.1] — 2026-05-24

P25-on-live-air follow-up release, fixing every NID/TSBK-decode
bug that surfaced once real captures from the Mt Anakie site
went through the pipeline that landed in v0.2.0. The BCH(63,16,11)
generator polynomial is now spec-correct (was wrong by 10 exponents
against TIA-102.BAAA Annex A — synthetic round-trip tests had passed
because encoder + decoder shared the same wrong polynomial), the
TSBK CRC verifier switches to the augmented variant per TIA-102.AABF
(the previous CRC-CCITT/FALSE rejected clean Viterbi output), and
the VHF / UHF `IdentifierUpdateVUHF` band-plan opcode (0x34) is
wired into the dispatcher so UHF P25 sites resolve grants without
stalling on `no-bandplan`. A new C4FM symbol-AGC keeps the matched-
filter outer-symbol centres scaled correctly on real RTL-SDR
captures, and the offline `gophertrunk replay` / `iq-diag` tool
grows a TSBK dump + per-instance NID-search span so stubborn
captures are debuggable without a radio on the bench. Operator-
visible polish: the daemon's blank 404 at `/` (when a binary was
built without first running `make web-build`) now serves an HTML
page explaining the fix; `make dist` is the one-shot build target
that always embeds the SPA; duplicate SDR serials in `sdr.devices`
are caught at config-validation time with both indices named;
WinUSB `ERROR_ACCESS_DENIED` on Windows gets a remediation hint
pointing at other SDR apps; `internal/version` now auto-stamps
from Go's VCS info on a bare `go build`, so the version string is
no longer a useless `dev` when an operator skipped `make build`.

### Added

- **P25 Phase 1 `IdentifierUpdateVUHF` (TSBK opcode 0x34) wired
  through the dispatcher — UHF P25 sites resolve voice grants
  without stalling on `no-bandplan`.** The 0x34 opcode constant
  was already defined in `internal/radio/p25/phase1/opcodes.go`,
  but it had no parser and no `switch` case, so `IDEN_UP_VUHF`
  TSBKs arriving from a VHF / UHF site were silently dropped —
  the `BandPlan` stayed empty and every subsequent
  `GroupVoiceChannelGrant` emitted `decode.error
  stage=no-bandplan`. The CC lock itself worked fine; the failure
  was downstream of the lock and invisible without inspecting the
  events bus. `ParseIdentifierUpdateVUHF` /
  `AssembleIdentifierUpdateVUHF` decode the VHF/UHF bit packing
  per TIA-102.AABF Table 14a (4-bit `BW` lookup → 6.25 / 12.5 kHz
  per Table 16, 1-bit sign + 13-bit magnitude `TxOffset` whose
  unit is the channel step rather than a fixed 250 kHz, plus the
  same 10-bit `STEP × 125 Hz` and 32-bit `FREQ × 5 Hz` as the
  0x3D variant) and populate the existing `IdentifierUpdate`
  struct, so `BandPlan.Apply` / `BandPlan.Frequency` need no
  change. Cross-checked bit-by-bit against OP25
  (`op25/gr-op25_repeater/apps/trunking.py` `iden_up vhf uhf`)
  and SDRTrunk (`FrequencyBandUpdateVUHF.java`). Round-trip tests
  cover both negative offset (the typical UHF -5 MHz case) and
  positive offset (sign-bit coverage); a new end-to-end test
  feeds a VUHF `IdentifierUpdate` plus a subsequent grant through
  the real `ControlChannel.Process` chain and asserts the grant
  resolves to the expected frequency rather than falling to
  `decode.error`.

- **C4FM symbol-AGC on the P25 Phase 1 receive path (issue
  #275).** The P25 receive filter (`P25C4FMRxTaps`) is normalised
  to a DC gain of `sps`, so on real RTL-SDR captures the matched-
  filter outer-symbol centres land at `sps × 2π·deviation /
  sampleRate` radians — orders of magnitude larger than the
  ±3/±1 dibit decision boundaries the slicer expects. A per-
  symbol AGC now scales the matched-filter output back into the
  slicer's expected range, which is what made the BCH-decode
  fixes below visible on live air rather than just on synthetic
  modulator round-trip tests.

- **Offline `gophertrunk replay` / `iq-diag` tool grows TSBK dump
  + per-instance NID-search span (issue #275).** `replay -in
  capture.iq -diag` now appends the first 24 TSBK dibits at each
  perfect-distance FSW, which distinguishes a periodic fixed
  beacon (identical NID + identical TSBK) from a real CC
  (identical NID + varying TSBK) without running the trellis
  decoder. A new `-nid-search-span N` flag widens the
  NID-alignment search beyond the production default (±6 dibits)
  as a bisect knob for stubborn captures; the production
  `ccdecoder` is unchanged (zero in `Options` falls back to the
  default span). The tool is now documented in the README and
  `docs/hardware.md` so operators can use it without re-reading
  source.

- **`make dist` one-shot release-build target.** `make dist`
  runs `web-build` then `build` so the daemon binary always
  embeds the SPA; `make cross-build`, `make release-dry-run`,
  and `make run` now depend on `web-build` for the same reason.
  Closes the v0.1.x footgun where `go build ./cmd/gophertrunk`
  without first running `make web-build` produced a binary that
  silently 404'd at `/` (see Fixed below).

### Fixed

- **P25 Phase 1 BCH(63,16,11) generator polynomial was wrong by
  10 exponents against TIA-102.BAAA Annex A (issue #275).**
  `bch6316Generator` was `0xF391E2F34B99`; the spec polynomial —
  the product of the minimal polynomials of α, α³, α⁵, …, α²¹
  over GF(2⁶) with primitive `p(x) = x⁶ + x + 1` — is
  `0xCD930BDD3B2B`. Synthetic-modulator round-trip tests passed
  because the encoder and decoder both used the wrong polynomial,
  so the bug was invisible until the Mt Anakie capture went
  through the live pipeline (197/197 NID failures with the wrong
  polynomial, 195/197 clean decodes with the spec one). Per-DUID
  parity tables are now derived from the spec polynomial as well.
  A test shim with the old wrong polynomial hardcoded inline has
  been removed from `motorola/process_test.go` so the test
  exercises the same code path the daemon does.

- **P25 Phase 1 TSBK CRC verifier now uses the spec-correct
  augmented variant per TIA-102.AABF (issue #275).** The original
  trailer code used the "CRC-CCITT/FALSE" variant (init=0xFFFF,
  no final XOR, trailer stored inverted). The P25 spec — cross-
  checked against OP25 (`crc16_ccitt_xor`) and SDRTrunk
  (`CRCP25.checkCRCCCITT`) — uses the **augmented** variant
  (init=0xFFFF, the trailer participates in the LFSR shift, no
  final XOR or inversion). With PR #337's BCH polynomial fix
  alone, the Mt Anakie capture's TSBKs all came out of the
  trellis decoder with metric=0 (clean Viterbi path) but still
  failed CRC; with this fix the CRC verifier agrees with the
  trellis decoder and the TSBKs actually decode.

- **Motorola Type II patch members no longer emitted as
  triplicated talkgroup IDs (`[32501 32501 32501]`) — issue
  #275.** Audit ruled out a parser bug: `AsMotorolaPatchGroup`
  correctly reads three independent 16-bit fields, and the
  on-air payload bytes really are `0x7EF5` triplicated (Motorola
  pads short patch lists with the first member). The parser now
  deduplicates members on parse so a one-member patch is reported
  as one member instead of three.

- **Daemon now serves a helpful HTML page (not a blank stdlib
  404) at `/` when the SPA isn't embedded (issue #290).**
  `//go:embed all:dist` snapshots `web/dist/` at Go compile time,
  so a binary built without first running `make web-build`
  embeds only the `.gitkeep` sentinel and silently 404s at `/`.
  The 404 body now explains the cause and points at `make dist`;
  status code stays 404 so proxies/healthchecks are unaffected.
  Combined with the new `make dist` target above, the case
  shouldn't arise for release binaries.

- **`sdr.devices` config now rejects duplicate device serials at
  validation time (issue #333).** A Windows user listed the same
  RTL-SDR serial twice (control + voice) and the pool silently
  collapsed the hint, leaving WinUSB to fail the second
  `CreateFile` with `ERROR_ACCESS_DENIED` ("Toegang
  geweigerd") — a cryptic OS-level error that obscured a config
  mistake. `config.Validate()` now rejects duplicate serials in
  `sdr.devices` with a message naming both offending indices and
  explaining the one-SDR-per-role rule, and the RTL-SDR USB open
  path emits a remediation hint on Windows
  `ERROR_ACCESS_DENIED` pointing at other SDR apps that might
  be holding the dongle.

- **`internal/version` auto-stamps from Go's VCS info on a bare
  `go build` (issue #275).** Without the Makefile's `-ldflags`
  injection the version package stayed at its zero defaults
  (`Version="dev"`, `Commit=""`, `BuildTime=""`) and the
  `ccdecoder: p25/phase1 pipeline configured` log line printed
  `build=dev` even when source HEAD was a real commit. The
  package now falls back to `debug.ReadBuildInfo()` for both
  commit and build time when ldflags were not set, so issue-#275
  retest cycles where operators paste log excerpts always carry
  identifying build provenance. The Makefile-injected values
  still take precedence in production / release builds.

- **`TestDaemonCCDecodesDPMR` integration deadline is no longer
  flaky under `-race`.** dPMR runs at half the symbol rate the
  sibling P25 / DMR / NXDN tests use (2400 vs 4800 sym/s), so
  the same mock-SDR IQ chunk carries half the dibits per second
  and the cold-start path occasionally exceeded the 5 s lock
  deadline on slower hardware (~3% under `-race`). The deadline
  is now 30 s; steady-state lock time is still ~0.4 s, so the
  bump only affects worst-case slow paths.

## [v0.2.0] — 2026-05-23

SDR-fleet + DMR-voice + P25-lock release. The pure-Go SDR
backend grows from RTL-SDR-only into a full fleet — HackRF One
/ Jawbreaker / Rad1o, Airspy R2 / Mini, and the entire Airspy
HF+ family all gain native drivers with no `libhackrf` /
`libairspy` / `libairspyhf` at build or runtime, so the
single-static-binary guarantee holds across every supported
front-end. DMR gains its missing voice path: an AMBE+2 3600 ×
2450 vocoder decodes Tier II / Tier III voice superframes to
WAV. P25 Phase 1 control-channel lock on live air gets the
final attention pass it needed — NID-alignment search after
FSW, TSBK-CRC corroboration for marginal NIDs, restricted C4FM
rotation set, and a per-dibit error-pattern diagnostic that
makes lock failures debuggable from the log. A new
`gophertrunk replay` subcommand decodes captured wideband IQ
offline so issue triage doesn't need a radio on the bench.
RTL-SDR's classic "device busy" failure mode is gone — the USB
layer now auto-detaches the bound `dvb_usb_rtl28xxu` kernel
driver the way `libusb` does for `librtlsdr`, so the daemon
opens dongles out of the box without first blacklisting the
DVB module.

### Added

- **Airspy HF+ Discovery / Dual Port / legacy HF+ pure-Go driver.**
  New `internal/sdr/airspyhf` package implements the `sdr.Driver` /
  `sdr.Device` interfaces on top of the same pure-Go USB transport
  (USBDEVFS / WinUSB / IOKit) the RTL-SDR / HackRF / Airspy drivers
  use — no `libairspyhf` at build or runtime, the zero-CGO
  single-binary guarantee still holds. The driver speaks the
  documented libairspyhf USB vendor protocol (RECEIVER_MODE,
  SET_FREQ, GET_SAMPLERATES, SET_HF_AGC, SET_HF_ATT, SET_HF_LNA,
  SET_BIAS_TEE, GET_VERSION_STRING) and decodes the HF+'s
  interleaved int16 IQ payload into complex64. All three known
  variants (Discovery, Dual Port, legacy) enumerate on VID:PID
  `0x03eb:0x800c`; the USB descriptor's Product string drives the
  `TunerName` distinction. Coverage: 9 kHz – 31 MHz HF + 60 –
  260 MHz VHF; HF AGC plus a 0–48 dB attenuator (6 dB steps) and
  +6 dB LNA preamp. Registered on init so a blank import from
  `cmd/gophertrunk` is the only wiring needed. The wire protocol
  is unit-tested against `usb.MockTransport`; on-air validation
  against attached HF+ hardware is the documented follow-up.

- **HackRF firmware-aware identification.** The HackRF driver now
  reads `BOARD_ID_READ` and `VERSION_STRING_READ` at Open time and
  uses the firmware's self-reported identity (rather than the USB
  descriptor's Product string) to populate `sdr.Info.Product` as
  `HackRF One` / `HackRF Jawbreaker` / `Rad1o`. The running
  firmware version is appended to `TunerName` (`MAX2839+MAX5864
  (fw git-2024.02.1)`), and PortaPack / Mayhem builds are
  auto-detected and tagged with `+ PortaPack` so the operator can
  see at a glance which board is on which USB port. `Enumerate`
  also normalises Product based on the PID, so listings are
  consistent even before Open.

- **Airspy R2 vs Mini distinction in `TunerName`.** The Airspy
  driver now detects the `MINI` substring in the USB Product
  string at enumeration time and emits `R820T (Airspy R2)` or
  `R820T (Airspy Mini)` accordingly. Both variants share the same
  VID:PID, same R820T tuner, and same wire protocol — the split
  surfaces purely through the operator-visible label so multi-
  dongle pools can pick the right unit by name.

- **HackRF One and Airspy R2 / Mini pure-Go drivers.** New
  `internal/sdr/hackrf` and `internal/sdr/airspy` packages implement
  the `sdr.Driver` / `sdr.Device` interfaces on top of the same
  pure-Go USB transport (USBDEVFS / WinUSB / IOKit) the RTL-SDR
  driver uses — no `libhackrf` or `libairspy` at build or runtime,
  so the zero-CGO single-binary guarantee holds. The drivers speak
  the documented libhackrf and libairspy USB vendor protocols
  (transceiver / receiver mode, frequency, sample rate, LNA / VGA /
  mixer / amp / bias-tee gains, bulk-IN sample reaper with real-time
  decode of HackRF int8 IQ and Airspy INT16_IQ into complex64). Both
  register themselves with the SDR driver registry on init, so a
  blank import from `cmd/gophertrunk` is the only wiring needed. The
  wire protocols are unit-tested against `usb.MockTransport`; on-air
  validation against attached HackRF / Airspy hardware is the
  documented follow-up.

- **DMR voice decodes to playable WAV (issue #276).** The DMR
  voice path is now end-to-end: a Tier II / Tier III voice
  superframe decoder slices the AMBE+2 burst layout into the
  three 49-bit voice frames per burst, and a clean-room pure-Go
  AMBE+2 3600 × 2450 vocoder takes the on-air FEC-protected
  frames through soft-decision deinterleave → Golay(23,12,7) +
  Hamming(15,11,3) FEC → b₀…b₈ parameter extraction → MBE
  synthesis → 8 kHz PCM. The composer wires the chain into the
  recorder so a DMR voice grant now produces a WAV instead of an
  empty `.raw` sidecar. Encrypted DMR voice calls are detected
  (PI header keyword + signalling-flag check), tagged on the
  call record, and logged so an operator can tell at a glance
  why a recording is silent.

- **`gophertrunk replay` subcommand for offline IQ decoding.**
  A new top-level subcommand mounts a wideband IQ recording (the
  two-channel 16-bit WAV layout the daemon writes, or
  SDRtrunk's) into the SDR pool as a virtual tuner and runs the
  full decode pipeline against it with no radio attached. Issue
  triage (especially for #275) can now reproduce a control-
  channel-lock failure off a customer-supplied capture instead
  of needing the original site on the bench.

- **P25 Phase 1 control-channel lock on live air (issue #275).**
  Four targeted fixes to the NID acquisition path: (1) the
  alignment search now sweeps across symbols after the FSW
  rather than assuming bit-exact synchrony, fixing a class of
  marginal sites that previously never locked; (2) NID
  candidates with one or two residual bit errors are
  corroborated against the next TSBK's CRC before being accepted
  or rejected, so a single noisy NID dibit no longer drops the
  whole superframe; (3) the C4FM rotation set is restricted to
  the four physically realisable dibit phases, eliminating false
  locks on rotated noise; (4) on NID failure the decoder logs
  the per-dibit error pattern so a capture-driven debugger can
  see which specific symbols disagreed with the expected NID.
  At startup the `ccdecoder` now logs its NID-search parameters
  so the parameters used on a given run are visible in the log
  without having to read source.

- **DMR encryption guide.** A new
  [`docs/dmr-encryption.md`](docs/dmr-encryption.md) page
  documents the DMR encryption landscape (basic + enhanced
  privacy, ARC4 vs AES, key-management), what GopherTrunk does
  detect (the PI header, the signalling-flag bit, vendor key
  IDs) and what it deliberately does not do (decrypt without an
  operator-supplied key), with worked log examples.

### Fixed

- **RTL-SDR dongles now open even with the DVB kernel driver still
  bound.** On Linux the kernel binds `dvb_usb_rtl28xxu` (the DVB-T
  TV-tuner driver) to RTL-SDR dongles at plug time. An operator who
  hadn't blacklisted that module saw the daemon fail every device
  with `open device failed … claim interface 0: device or resource
  busy` followed by `SDR pool open failed … no SDR devices opened` —
  even though `sdr list` (which only reads USB descriptors) happily
  showed the dongles. The USB layer now detaches the bound kernel
  driver and retries the claim — the same auto-detach-kernel-driver
  behaviour `librtlsdr` gets from libusb — so GopherTrunk opens the
  dongle out of the box. Blacklisting the module is still recommended
  (it stops the kernel grabbing the device first) but no longer
  required. A claim error that survives the auto-detach now carries a
  hint that another user-space process is holding the dongle.
- **Empty talkgroup CSV no longer reported as a load failure.** A
  talkgroup CSV that existed but was empty (a freshly-touched
  placeholder, or a system whose talkgroups aren't catalogued yet)
  made the daemon log a scary `WARN talkgroup load failed … err="read
  csv header: EOF"`. An empty file is a legitimate "no talkgroups"
  state: `LoadCSV` now loads it cleanly as zero records, and preflight
  surfaces an actionable `talkgroup_file … is empty` warning instead.

## [v0.1.8] — 2026-05-21

P25 reception + voice-path release. The bulk of the work makes
trunked control-channel decode actually lock on live RTL-SDR
hardware (issue #275): IQ-stream channelization, cross-chunk
frame assembly, symbol-clock chunk-boundary fixes, a CQPSK / LSM
demodulator path with a blind equalizer and AGC for simulcast
sites, and coarse AFC for tuner carrier offset. On top of that,
P25 Phase 1 and Phase 2 are built out to functional SDRtrunk
parity with working voice decoding, and DMR gains a voice
decoding path (issue #276) where it previously decoded control
channels only. The web console's connect-time render loop and
WebSocket reconnect storm (issue #290) are both fixed.

### Added

- **Protocol-agnostic affiliation tracker.** A new
  `trunking.AffiliationTracker` maintains a live "which radio unit
  is on which talkgroup" table, fed by `KindGrant` (the grant's
  source/group is ground truth), explicit `KindAffiliation` events,
  and `KindUnitRegistration`. Because every protocol's grant carries
  a source and group, the table works uniformly across P25, DMR
  (all tiers and vendors) and NXDN with no per-protocol decoding.
  Idle units expire after a TTL. Served at `GET /api/v1/affiliations`.
- **Per-talkgroup mute and icon assignment.** A talkgroup can carry
  a `mute` flag (suppresses its calls from the live audio player
  while still following, recording and streaming them) and an
  `icon` name (the data model behind SDRtrunk's Icon Manager) — set
  via CSV column, JSON field, or `PATCH /api/v1/talkgroups/{id}`,
  and surfaced in the talkgroup API DTO.
- **Analog-trunking voice decoding.** Motorola Type II / SmartZone,
  EDACS, LTR and MPT 1327 calls now decode to audio through the
  composer's FM voice chain — they carry plain narrowband FM, so the
  existing FM chain is the correct decoder. EDACS ProVoice (digital,
  patent-encumbered) stays on the `.raw` sidecar path.
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
- **Per-talkgroup recording assignment.** A talkgroup can now be
  flagged `record: false` (CSV column, JSON field, or
  `PATCH /api/v1/talkgroups/{id}`) to follow and play its calls live
  while writing no WAV/raw files for it — the recording analogue of
  the `stream` opt-out. Both `stream` and `record` are now surfaced
  in the talkgroup API DTO and accepted by the PATCH endpoint.
- **Decoded-message log.** A new optional `MessageLog`
  (`internal/log`) writes a human-readable, timestamped text log of
  every trunking event the bus carries — grants, control-channel
  lock/loss, affiliations, registrations, patches, talker aliases,
  locations, tone alerts, decode errors — the GopherTrunk analogue
  of SDRtrunk's per-channel decoded message log. The file rotates to
  `<path>.1` past a configurable size cap. Enabled via a new
  `log.message_log` config block.
- **GPS / location subsystem.** Geographic fixes a subscriber unit
  reports over the air now flow through a new `KindLocation` event
  (`trunking.Location` payload) to a `location_log` SQLite table and
  out via `GET /api/v1/locations` for map display. A new
  `internal/radio/location` package implements a strict NMEA-0183
  GGA/RMC parser — the format Tait CCDI and many MOTOTRBO GPS
  profiles transport verbatim — with checksum verification. The
  per-protocol binary GPS PDU extractors (P25 Motorola Unit GPS,
  L3Harris Talker GPS, DMR LRRP) and the web map page build on this
  backbone; their bit-exact wiring is pending capture validation.
- **DMR vendor-trunking recognition (FID-aware CSBK dispatch).**
  The Tier III control-channel decoder now dispatches each CSBK on
  its feature-set ID (FID) before opcode, so a Motorola or Hytera
  vendor CSBK is no longer misdecoded against the standard ETSI
  opcode table — previously a vendor CSBK whose 6-bit opcode
  collided with `0x30` would emit a bogus voice grant. Motorola
  Capacity Plus / Capacity Max voice grants (FID 0x10), which carry
  the ETSI-shaped 8-octet payload, now decode to real grants, and
  the Capacity Plus rest channel is tracked from its system-info
  CSBK. Connect Plus and Hytera XPT CSBKs are recognised and routed
  to a vendor handler; bit-exact decoding of those proprietary
  payloads is pending on-air capture validation.
- **Wideband baseband (IQ) recording and offline replay.** A new
  `internal/sdr/baseband` package adds two capabilities SDRtrunk
  has and GopherTrunk lacked. A `RecordingDevice` decorator tees a
  live tuner's IQ stream to a two-channel 16-bit WAV (in-phase in
  channel 1, quadrature in channel 2 — the same layout as
  SDRtrunk's baseband recordings). A `FileDriver` mounts those
  recordings (and SDRtrunk's) back into the SDR pool as virtual
  tuners, so a capture can be decoded offline with no radio
  attached; replay loops on EOF to behave like a continuous
  source. Both are configured under a new `baseband:` config
  section (`record:` and `replay:` lists).
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
- **DMR voice decoding path and Enhanced Privacy key
  configuration** (issue #276, PR #298, #301, #304, #305). DMR
  previously decoded control channels only. The voice path now
  ships: a DMR voice superframe decoder plus AMBE+2
  forward-error-correction (`internal/radio/dmr/voice/` — 72-bit
  on-air frame → C0/C1 Golay(23,12) + C1 descramble → 49-bit
  vocoder payload, ported from mbelib / DSD), and a composer DMR
  voice chain that runs IQ → DMR receiver → superframe decoder →
  AMBE FEC and writes the FEC-decoded frames to the call's
  `.raw` sidecar. A dependency-free RC4 keystream generator
  (`internal/crypto/rc4/`) and per-system `encryption_keys`
  config (`key_id` + `algorithm: rc4` + hex `key`, validated at
  load) lay the foundation for known-key Enhanced Privacy voice
  decryption.
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
- **P25 Phase 1 CQPSK blind equalizer for simulcast multipath**
  (issue #275, PR #306). A P25 simulcast site sums several
  synchronised transmitters into a multipath channel that closes
  the CQPSK constellation, so the Frame Sync Word never
  correlates and the control channel never locks. Because LSM is
  a linear modulation the distortion is linear in the complex
  symbols: the `equalizer.CMA` blind (Constant Modulus
  Algorithm) equalizer is now wired onto the CQPSK symbol stream
  between Gardner timing recovery and the differential decode.
  It needs no training sequence and is a near-noop on a clean
  constant-modulus signal. The #275 IQ-impairment harness gains
  a multipath channel model.
- **Coarse AFC on the P25 Phase 1 C4FM control channel** (issue
  #275, PR #303). A residual RTL-SDR carrier offset leaves the
  FM discriminator with a constant DC bias that shifts the C4FM
  4-level slicer's eye off its decision regions; at ≥500 Hz the
  Frame Sync Word stops correlating entirely. A new coarse-AFC
  stage (`demod.CoarseAFC`) between the matched filter and the
  symbol clock tracks the bias with a slow single-pole average
  and subtracts it, recentring the eye. On a clean signal the
  estimate converges to ~0 and the stage is a near-noop.
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
- **Web console SPA render loop blanked the UI on connect**
  (issue #290, PR #295). `selectClientConfig` returned a fresh
  object on every call, so the WebSocket effect — which listed
  the derived config in its deps and synchronously wrote
  connection status to the store — re-fired without bound (React
  error #185), blanking the UI and churning the socket open /
  close. The selector is now memoised to a stable reference
  until the server URL / token actually change; the event
  WebSocket URL is rebuilt with the URL API (handles uppercase
  schemes, never emits a host-less URL); and a top-level
  `ErrorBoundary` shows a fallback instead of a blank page on a
  render crash.
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
- **P25 Phase 1 symbol-clock loops miscounted symbols across
  IQ-chunk boundaries** (issue #275, PR #300, #311). Both
  symbol-timing-recovery loops rebuild their working buffer each
  call but mishandled the chunk seam, so the recovered dibit
  count depended on IQ chunk size — a live RTL-SDR delivers
  ~19-symbol USB transfers, and the drift scattered dibit errors
  so the Frame Sync Word never aligned and the control channel
  never locked. The Gardner loop (CQPSK / LSM path) re-emitted
  ~1 surplus symbol per call; the Mueller-Müller loop (C4FM
  path) dropped `src[0]` of every continuation chunk. Both now
  treat the carried-over samples as pure look-back context, so
  the recovered dibit stream is byte-identical regardless of
  chunk size.
- **P25 Phase 1 dibit-rotation inversion broke simulcast
  control-channel lock** (PR #296). The FSW sync detector
  reports rotation `k` such that `(received + k) mod 4` is
  canonical, so dibits are recovered by adding `k` — but
  `rotateDibits` added `(4-k) & 3`, correct only for even
  rotations. The odd quadrant slips (1, 3) that the CQPSK / LSM
  demod leaves on simulcast P25 recovered every dibit off by
  two, so the NID BCH decode failed and the control channel
  never locked.
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
- **macOS device enumeration panicked before listing any
  RTL-SDR** (issue #257, PR #293). The macOS USB enumerator
  registered CoreFoundation function pointers whose signatures
  named a `[16]byte` array type; purego's `RegisterLibFunc`
  panics with "unsupported kind array" on any array in a
  registered signature, so IOKit failed to load for every macOS
  user before a single call ran and `sdr list` found no devices.
  The 16-byte `CFUUIDBytes` is now passed as two `uint64`
  register halves. Per-driver enumerate errors also surface from
  `EnumerateAll`, so `sdr list` prints the failure instead of a
  silent empty list.
- **Config rejected valid trunking protocols** (issue #291, PR
  #294). Config validation hardcoded a `p25|dmr|nxdn` whitelist
  that was never updated as the other protocols landed, so a
  valid `protocol: tetra` (or edacs / ltr / mpt1327 / …) system
  failed at load despite being fully implemented. Validation now
  routes through `trunking.ParseProtocol` — the same parser the
  daemon uses — so the canonical protocol list is the single
  source of truth.

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
