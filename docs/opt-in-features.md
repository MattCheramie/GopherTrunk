# Opt-in features and protocols

GopherTrunk ships many protocol-level and daemon-level capabilities as
**opt-in**: the daemon constructs the relevant component in its
legacy / safe-default mode, and only flips the new behaviour on when
the operator sets a key in `config.yaml`, passes a receiver option, or
builds with a tag. This is deliberate — every gate either preserves
backwards compatibility with existing fixtures, avoids a security
regression, or trades a resource that operators may not be willing to
spend.

This document enumerates every opt-in in the tree, the reason it
ships off, and the concrete work that would let the project either
flip the default to enabled or remove the gate entirely.

## Contents

1. [Protocol-layer FEC opt-ins](#1-protocol-layer-fec-opt-ins)
2. [Receiver clock-recovery opt-ins](#2-receiver-clock-recovery-opt-ins)
3. [Daemon-level opt-ins](#3-daemon-level-opt-ins)
4. [Build-time opt-ins](#4-build-time-opt-ins)
5. [Cross-cutting remediation strategies](#5-cross-cutting-remediation-strategies)
6. [How to verify what's currently enabled](#6-how-to-verify-whats-currently-enabled)

---

## 1. Protocol-layer FEC opt-outs (formerly opt-ins)

**Status: flipped.** Every protocol that has a public-spec
forward-error-correction chain now ships on by default. The
ccdecoder connector at
[`internal/scanner/ccdecoder/pipelines.go`](../internal/scanner/ccdecoder/pipelines.go)
always runs the per-protocol `Parse*Mode` function over the
configured YAML value — empty strings map to the new on-defaults.
Operators with pre-stripped capture files (DSD-FME `-r` dumps,
OP25 fixtures, MMDVMHost / DSDcc test data) opt out per-system
with `<key>: off`.

The current per-protocol state is documented in the README's
["FEC opt-outs" section](../README.md#fec-opt-outs); summary:

| Protocol | YAML key | New default | Opt-out string |
| --- | --- | --- | --- |
| TETRA | `tetra_channel_coding` | ChannelCodingOn (full §8.3.1 chain) | `off` |
| LTR FCS | `ltr_fcs_mode` | FCSOn | `off` |
| LTR Manchester | `ltr_manchester_mode` | ManchesterSoft | `off` / `nrz` |
| P25 Phase 2 | `p25_phase2_trellis_mode` | TrellisOn | `off` |
| NXDN | `nxdn_viterbi_mode` | ViterbiSpec | `off` |
| EDACS | `edacs_bch_mode` | BCHOn | `off` |
| MPT 1327 | `mpt1327_bch_mode` | BCHOn | `off` |
| Motorola Type II | `motorola_bch_mode` | BCHOn | `off` |

### How it works

The in-package `ControlChannel` constructors still zero-value to
Off mode (so direct callers — primarily unit tests — see the legacy
behaviour without explicit setup). The connector goes through
`Parse*Mode(opts.System.X)` which maps empty strings to the new
on-defaults, then calls `SetXMode(parsed)` on the ControlChannel.
This keeps the operator-facing default On while preserving the
in-package fixtures' expectations of an Off zero value.

For TETRA specifically: a new `TETRAChannelCoding` field was added
to the System struct (yaml: `tetra_channel_coding`) because the
old "zero colour code = off" rule conflicted with BSCH's
spec-defined zero colour code. Non-BSCH systems with channel
coding on need a non-zero `tetra_colour_code`; the connector
warn-logs the misconfiguration instead of silently dropping
frames.

### Remediation status

Done in PR 1 on branch `claude/document-opt-in-features-gP90L`
(this branch). No further remediation needed for FEC opt-ins.

---

## 2. Receiver clock-recovery opt-ins

These live at the receiver-option level and are **not yet exposed
via YAML**. Currently only reachable programmatically (tests + any
embedder constructing the receiver directly).

| Feature | Source | Default | Opt-in | Why opt-in | Remediation |
| --- | --- | --- | --- | --- | --- |
| P25 Phase 2 Gardner clock | [`internal/radio/p25/phase2/receiver/receiver.go:67-70`](../internal/radio/p25/phase2/receiver/receiver.go) | `ClockNaive` (every sps-th sample after the matched filter) | `Options.ClockMode: ClockGardner` | The Gardner timing-recovery loop in `internal/dsp/sync/gardner.go` is new; existing receiver tests use naive timing and would need regeneration. No YAML connector exists yet. | Wire `clock_mode` into [`internal/config/config.go`](../internal/config/config.go) and `internal/trunking/site.go` so the ccdecoder connector can forward it. Regenerate Gardner-friendly fixtures, then flip the receiver default to `ClockGardner`. This is the "follow-up" called out in [`README.md:125-135`](../README.md). |
| TETRA Gardner clock | [`internal/radio/tetra/receiver/receiver.go:62-68`](../internal/radio/tetra/receiver/receiver.go) | `ClockNaive` | `Options.ClockMode: ClockGardner` | Same as P25 Phase 2: Gardner loop is new, fixtures predate it, no config connector. | Same remediation: add a config field, regenerate fixtures, flip default. The two receivers share the same `ClockMode` enum and Gardner primitive so they can be converted in one PR. |

---

## 3. Daemon-level opt-ins

Sources are [`config.example.yaml`](../config.example.yaml) plus the
config struct in [`internal/config/config.go`](../internal/config/config.go).

| Feature | YAML key | Default | Why opt-in | Remediation |
| --- | --- | --- | --- | --- |
| **Live audio playback to speakers** | `audio.enabled` | `false` ([`internal/config/config.go:35-54`](../internal/config/config.go), [`config.example.yaml:108-118`](../config.example.yaml)) | Headless and container deployments stay silent by design. WAV recording is unaffected — recordings land on disk whether playback is on or off. | Keep opt-in. This is the correct posture for a server daemon: the alternative (audio on by default) would crash or warn loudly in distroless / container deployments. If a future TUI-bundled distribution wants live audio out of the box, ship the TUI with a different default config, leaving the daemon's `false` default intact. |
| **API mutation endpoints** | `api.auth.mode` | `auto` ([`internal/config/config.go`](../internal/config/config.go) APIConfig + APIAuthConfig) | **Resolved.** Bearer-token auth landed with loopback bypass under the `auto` policy. Public-bind listeners now require a token. See [`docs/hardening.md` §"API authentication"](hardening.md). Legacy `allow_mutations: true` maps to `auth.mode: disabled` with a deprecation warning. | Done. |
| **Manual VFO tune** | `scanner.manual_tune_enabled` | `false` ([`internal/config/config.go:70-77`](../internal/config/config.go), [`config.example.yaml:77-84`](../config.example.yaml)) | The conventional scanner steals a Voice SDR from the trunking pool. Auto-enabling would silently downgrade trunked grant-following for operators with only one Voice SDR. | At startup, count Voice SDRs declared in `sdr.devices` and compare against the maximum simultaneous voice-grants the configured trunking systems can drive. If a spare Voice SDR is present, auto-enable manual tune; otherwise stay opt-in. Document the dedicated-SDR requirement in either case. |
| **CMA blind equalizer** | `recordings.equalizer.enabled` | `false` ([`config.example.yaml:29-32`](../config.example.yaml)) | Simulcast mitigation costs CPU and may slightly distort clean-RF capture. Benefit is site-specific — operators not on a simulcast site pay the CPU without payoff. | Add an auto-tune heuristic that measures multipath / ISI on the first N seconds of audio per call and toggles the equalizer for that call only. Keep the global `enabled: true` knob for operators who want the equalizer always on. |
| **Tone-out paging-tone detection** | `tone_out.profiles` | empty list ([`config.example.yaml:120-135`](../config.example.yaml)) | Two-tone sequential (Quick Call II) profiles are per-site / per-agency. No useful zero-config default exists. | Cannot flip the default reasonably. Improve discoverability instead: ship a small set of well-known regional profiles in a commented-out block in `config.example.yaml`, and link this doc from the tone-out section of the README so new operators see the path. |
| **Scan mode = list** | `scanner.scan_mode` | `"all"` ([`internal/config/config.go:60-65`](../internal/config/config.go)) | `"all"` is the legacy / backwards-compatible behaviour. Tag-based talkgroup curation must be done by the operator before `list` is useful. | Not a default-flip target. Per-deployment configuration choice; the doc should call out that `list` exists and how Emergency grants bypass the gate. |
| **CTCSS / DCS squelch** | per-channel `tone:` block in `scanner.conventional[]` ([`config.example.yaml:94-102`](../config.example.yaml)) | omitted (no gating) | Sub-audible CTCSS tone or DCS code is per-channel signalling that varies by site, repeater, and agency. No useful zero-config default. | Cannot flip — site-specific. Doc-only: link the README's CTCSS / DCS section (lines 163-197) and call out that omitting the block reverts to carrier-only squelch. |
| **Raw frame sidecar** | `recordings.write_raw` | `true` in the example ([`config.example.yaml:28`](../config.example.yaml)) | Listed here for completeness — it is **not** off by default in the shipped example. Operators who don't want the `.raw` sidecar can set `false`. | None required; the disable case is documented for symmetry. |
| **Prometheus metrics** | `metrics.enabled` | `true` ([`config.example.yaml:19`](../config.example.yaml)) | Already on by default; listed for completeness. | None required. |
| **Call-log retention sweep** | `retention.call_log_days` | `30` (`0` disables, [`config.example.yaml:35`](../config.example.yaml)) | Sensible default already on. | None required; documented for the disable-via-`0` case. |
| **File retention sweep** | `retention.files_days` | `14` (`0` disables, [`config.example.yaml:36`](../config.example.yaml)) | Sensible default already on. | None required. |

---

## 4. Build-time opt-ins

| Feature | Mechanism | Default | Why opt-in | Remediation |
| --- | --- | --- | --- | --- |
| **DVSI hardware vocoder backend** | `-tags dvsi` build tag ([`docs/architecture.md:105`](architecture.md), [`docs/vocoders.md:47`](vocoders.md)) | Not built | Patent-encumbered. Requires DVSI USB-3000 / AMBE-3003 hardware. Status: **planned, not yet shipped** per [`docs/vocoders.md:170`](vocoders.md). The pure-Go AMBE+2 backend is the default and ships everywhere. | None. This is correctly gated for licensing reasons and should stay opt-in permanently. |
| **Integration tests** | `-tags integration` build tag ([`docs/architecture.md:103`](architecture.md)) | Not run by `go test ./...` | Enables a wired end-to-end daemon test that doesn't need a real SDR. Long-running, intentionally outside the default unit-test wall-time budget. | None. CI runs the tagged suite separately; default `go test` stays fast. |
| **Pure-Go (no CGO)** | implicit `CGO_ENABLED=0` build | On (default) | Already the default everywhere — no `librtlsdr` / `libusb` dependency. Listed for completeness because the README emphasises it as a design property. | None. |

---

## 5. Cross-cutting remediation strategies

The table rows above cite four recurring remediation patterns. The
short version:

1. **Re-encode fixtures + flip the default.** Applies to every FEC
   opt-in. The blocker is legacy DSD-FME / OP25 / MMDVMHost test
   inputs that predate the spec-correct path. One mechanical PR per
   protocol once the new fixtures land; the cross-protocol path is to
   add a `legacy_fixtures` test-only mode that the regression suite
   opts into so the new on-by-default behaviour can ship without
   deleting the old fixtures.
2. **Auto-detect at runtime.** Try the FEC / better path first; fall
   back to legacy on N consecutive failures. Small CPU burst on
   broken input, zero operator config. Applies to protocol FEC
   opt-ins and to the CMA equalizer.
3. **Add an auth layer.** ✅ Landed. Bearer-token middleware with
   loopback-bypass under `auto` mode replaces the binary
   `allow_mutations` gate; see [`docs/hardening.md` §"API
   authentication"](hardening.md).
4. **Site-specific config — no useful default.** `tone_out`, per-
   channel `tone:`, conventional channel lists, and talkgroup files
   stay opt-in by nature. Improve discoverability with better example
   YAML and cross-references in the README rather than trying to flip
   a default.

---

## 6. How to verify what's currently enabled

- **FEC opt-ins per system.** Open the **Settings** panel in the
  TUI — it lists every configured system with a one-line summary of
  its FEC opt-in state (`channel coding: on (colour=…, sch/f)`,
  `viterbi: off`, `bch: on`, etc.). See [`README.md:1971-1976`](../README.md).
- **Programmatic introspection.** Each protocol's `ControlChannel`
  exposes matching getters (`tetra.ControlChannel.ChannelCoding()` /
  `ExpectedChannel()` / `ColourCode()`, `ltr.ControlChannel.FCSMode()` /
  `ManchesterMode()`, `p25phase2.ControlChannel.TrellisMode()`,
  `nxdn.ControlChannel.ViterbiMode()`, `edacs.ControlChannel.BCHMode()`,
  `mpt1327.ControlChannel.BCHMode()`, `motorola.ControlChannel.BCHMode()`).
- **JSON over HTTP.** The `/api/v1/systems` endpoint DTO carries
  every FEC opt-in field as `omitempty` JSON, so a configured-systems
  audit is one `curl` away.
- **Daemon-level opt-ins.** Inspect the running `config.yaml` and
  the daemon's startup log lines; the config loader logs the
  effective values for every section as it parses.
