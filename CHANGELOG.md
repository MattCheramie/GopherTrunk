# Changelog

All notable user-visible changes land here, newest first.
Format adapted from [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
The project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
for tagged releases.

## [Unreleased]

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
