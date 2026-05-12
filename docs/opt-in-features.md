# Opt-in / opt-out features

GopherTrunk's protocol decoders and daemon-level surfaces have a
mix of:

- features that **were** opt-in and have since been **flipped on
  by default** (operators can still opt out per-protocol);
- features that **are still opt-in** because the default is
  correct for headless / server deployments;
- features that **stay opt-in by their nature** (per-site
  signaling, patent-encumbered backends, CI-only tests).

This document is the operator's reference for that landscape: what
default applies, why, and (where relevant) how to opt out.

## Contents

1. [Protocol-layer FEC defaults](#1-protocol-layer-fec-defaults) — on by default, opt-out per protocol
2. [Receiver clock recovery](#2-receiver-clock-recovery) — on by default, opt-out per protocol
3. [Daemon-level features](#3-daemon-level-features) — mix of on / off / auto-detect
4. [Build-time options](#4-build-time-options) — patent / CI gates that stay opt-in permanently
5. [Remaining FEC follow-ups](#5-remaining-fec-follow-ups) — per-protocol inner FEC layers pending spec / capture data
6. [How to verify what's currently enabled](#6-how-to-verify-whats-currently-enabled)

---

## 1. Protocol-layer FEC defaults

**On by default for every protocol.** The ccdecoder connector at
[`internal/scanner/ccdecoder/pipelines.go`](../internal/scanner/ccdecoder/pipelines.go)
always runs the per-protocol `Parse*Mode` function over the
configured YAML value — empty strings map to the spec-correct
on-default. Operators with pre-stripped capture files (DSD-FME
`-r` dumps, OP25 fixtures, MMDVMHost / DSDcc test data) opt out
per-system with `<key>: off`.

| Protocol | YAML key | Default | Opt-out string |
| --- | --- | --- | --- |
| TETRA | `tetra_channel_coding` | `ChannelCodingOn` (full §8.3.1 chain) | `off` |
| LTR FCS | `ltr_fcs_mode` | `FCSOn` | `off` |
| LTR Manchester | `ltr_manchester_mode` | `ManchesterSoft` | `off` / `nrz` |
| P25 Phase 2 | `p25_phase2_trellis_mode` | `TrellisOn` | `off` |
| NXDN | `nxdn_viterbi_mode` | `ViterbiSpec` | `off` |
| EDACS | `edacs_bch_mode` | `BCHOn` | `off` |
| MPT 1327 | `mpt1327_bch_mode` | `BCHOn` | `off` |
| Motorola Type II | `motorola_bch_mode` | `BCHOn` | `off` |
| D-STAR | `dstar_fec_mode` | `FECOff` (info-bits passthrough) | `on` flips to the JARL DV-mode FEC chain |

The README's [FEC opt-outs section](../README.md#fec-opt-outs)
documents the full reference table with on-default behaviour
descriptions.

### Implementation note

The in-package `ControlChannel` constructors still zero-value to
`Off` mode so direct callers (primarily unit tests) see the legacy
behaviour without explicit setup. The connector goes through
`Parse*Mode(opts.System.X)` which maps empty strings to the new
on-defaults, then calls `SetXMode(parsed)` on the ControlChannel.
This keeps the operator-facing default `On` while preserving the
in-package fixtures' expectations of an `Off` zero value.

For TETRA specifically: the `TETRAChannelCoding` field was added
to the System struct (yaml: `tetra_channel_coding`) because the
old "zero colour code = off" rule conflicted with BSCH's
spec-defined zero colour code. Non-BSCH systems with channel
coding on need a non-zero `tetra_colour_code`; the connector
warn-logs the misconfiguration instead of silently dropping
frames.

---

## 2. Receiver clock recovery

**Gardner timing recovery on by default.** Both the P25 Phase 2
and TETRA receivers route the matched-filter output through the
Gardner symbol-timing-recovery loop in
[`internal/dsp/sync/gardner.go`](../internal/dsp/sync/gardner.go).
Operators with sample-aligned synthesized IQ fixtures (some tests,
some replay tools) can opt back to the naive sps-th-sample
decimator per system.

| Receiver | YAML key | Default | Opt-out string |
| --- | --- | --- | --- |
| P25 Phase 2 | `p25_phase2_clock_mode` | `ClockGardner` | `naive` / `off` |
| TETRA | `tetra_clock_mode` | `ClockGardner` | `naive` / `off` |

Other receivers (DMR, dPMR, NXDN, EDACS, LTR, MPT 1327, Motorola
Type II, YSF) either don't need timing recovery or use protocol-
specific clock-tracking primitives. The Gardner loop is wired
where it had the largest measurable effect on noisier on-air
captures.

---

## 3. Daemon-level features

Sources are [`config.example.yaml`](../config.example.yaml) plus
the config struct in [`internal/config/config.go`](../internal/config/config.go).

| Feature | YAML key | Default | Why |
| --- | --- | --- | --- |
| **Live audio playback to speakers** | `audio.enabled` | `false` | Headless and container deployments stay silent by design. WAV recording is unaffected — recordings land on disk whether playback is on or off. Stays opt-in: audio-on-by-default would crash or warn loudly in distroless / container deployments. |
| **API mutation endpoints** | `api.auth.mode` | `auto` | Bearer-token auth with loopback bypass under the `auto` policy. Public-bind listeners require a token. See [`docs/hardening.md` §"API authentication"](hardening.md). Legacy `allow_mutations: true` maps to `auth.mode: disabled` with a deprecation warning. |
| **Manual VFO tune** | `scanner.manual_tune_enabled` / `scanner.manual_tune_disabled` | auto-detect | Auto-enables when ≥ 2 Voice SDRs are present (the daemon constructs the scanner off the spare). `manual_tune_enabled: true` forces the scanner even with only one Voice SDR; `manual_tune_disabled: true` vetoes the auto-detect for operators who want every Voice SDR reserved for trunking. |
| **CMA blind equalizer** | `recordings.equalizer.enabled` | `false` | Simulcast mitigation costs CPU and may distort clean-RF capture. Benefit is site-specific — operators not on a simulcast site pay the CPU without payoff. Stays a global opt-in until a per-call auto-tune heuristic ships. |
| **Tone-out paging-tone detection** | `tone_out.profiles` | empty list | Two-tone sequential (Quick Call II) profiles are per-site / per-agency. No useful zero-config default exists. [`config.example.yaml`](../config.example.yaml) ships an example profile plus three commented-out alternatives (single-tone, system+talkgroup scoped, tightened tolerance) so operators discover the schema without grepping source. |
| **Scan mode = list** | `scanner.scan_mode` | `"all"` | "all" is the backwards-compatible behaviour. Tag-based talkgroup curation must be done by the operator before "list" is useful. Per-deployment choice. Emergency grants bypass the gate regardless. |
| **CTCSS / DCS squelch** | per-channel `tone:` block | omitted (no gating) | Sub-audible CTCSS tone / DCS code is per-channel signalling that varies by site, repeater, and agency. No useful zero-config default. Omitting the block reverts to carrier-only squelch. |
| **Raw frame sidecar** | `recordings.write_raw` | `true` in `config.example.yaml` | On in the shipped example. Operators who don't want the `.raw` sidecar set `false`. |
| **Prometheus metrics** | `metrics.enabled` | `true` | On by default; listed for completeness. |
| **Call-log retention sweep** | `retention.call_log_days` | `30` (`0` disables) | Sensible default already on. Set `0` to disable the sweeper. |
| **File retention sweep** | `retention.files_days` | `14` (`0` disables) | Same as call-log. |

---

## 4. Build-time options

These stay opt-in by their nature — none of the three are candidates
to flip on by default.

| Feature | Mechanism | Default | Why |
| --- | --- | --- | --- |
| **DVSI hardware vocoder backend** | `-tags dvsi` build tag ([`docs/architecture.md`](architecture.md), [`docs/vocoders.md`](vocoders.md)) | Not built | Patent-encumbered. Requires DVSI USB-3000 / AMBE-3003 hardware. Status: planned, not yet shipped. The pure-Go AMBE+2 backend is the default and ships everywhere. Stays opt-in permanently for licensing reasons. |
| **Integration tests** | `-tags integration` build tag | Not run by `go test ./...` | Enables a wired end-to-end daemon test that doesn't need a real SDR. Long-running, intentionally outside the default unit-test wall-time budget. CI runs the tagged suite separately. |
| **Pure-Go (no CGO)** | implicit `CGO_ENABLED=0` build | On | Already the default everywhere — no `librtlsdr` / `libusb` dependency. Listed for completeness because the README emphasises it as a design property. |

---

## 5. Remaining FEC follow-ups

Per-protocol on-air FEC chains land in stages. Most ship today as
opt-out (see §1); a handful of inner FEC layers remain as
documented follow-ups, generally blocked on either spec access or a
real-air capture to validate against:

| Item | Status | Blocker |
| --- | --- | --- |
| **NXDN per-protocol interleaver + puncture inner layer** | `ViterbiSpec` mode wired through the connector; calibration against a captured MMDVMHost / DSDcc transmission lands next | Real-air capture |
| **P25 Phase 2 outer Reed-Solomon + per-burst block interleaver** | Trellis decoder shipping (`SetTrellisMode(TrellisOn)`); the spec's outer RS layer + per-burst block interleaver are pending | TIA-102.BBAB spec access + real-air capture |
| **MPT 1327 inter-codeword interleaver** | BCH(64, 48, 2) ships per-codeword; the inter-codeword bit-interleaver across 5-codeword CCDB groups still pending | MPT 1327 spec implementation work + real-air capture |
| **YSF FICH interleaver / puncture validation** | K=5 ½-rate trellis encoder + decoder round-trip in unit tests; validating against a captured YSF transmission's exact schedule remains | Real-air capture |
| **TETRA on-air recovery margins** | Full §8.3.1 chain ships and unit tests round-trip clean fixtures; on-air recovery margins (Viterbi correction depth vs. real co-channel + adjacent-channel interference) need profiling | Live capture |
| **DMR Tier II synthesized IQ fixture** | Pipeline + Process adapter + unit test all ship; `TestDaemonCCDecodesDMRTier2` is `t.Skip`'d because the synthesized Voice LC Header IQ fixture doesn't round-trip cleanly through the C4FM modulator+demod chain | Fixture calibration (or real-air capture exercises the pipeline) |

These don't block protocol-level operation today — every protocol's
production pipeline ships through to `events.KindCCLocked` /
`events.KindGrant` on the bus, and the engine + recorder + API
surfaces light up unchanged. The follow-ups improve on-air decode
robustness for specific noise conditions or fully-spec-correct
encoding.

---

## 6. How to verify what's currently enabled

- **FEC defaults per system.** Open the **Settings** panel in the
  TUI — the **FEC** tab lists every configured system with a
  one-line summary (`channel coding: on (colour=…, sch/f)`,
  `viterbi: spec`, `bch: on`, etc.).
- **Programmatic introspection.** Each protocol's `ControlChannel`
  exposes matching getters (`tetra.ControlChannel.ChannelCoding()` /
  `ExpectedChannel()` / `ColourCode()`, `ltr.ControlChannel.FCSMode()` /
  `ManchesterMode()`, `p25phase2.ControlChannel.TrellisMode()`,
  `nxdn.ControlChannel.ViterbiMode()`, `edacs.ControlChannel.BCHMode()`,
  `mpt1327.ControlChannel.BCHMode()`, `motorola.ControlChannel.BCHMode()`).
- **JSON over HTTP.** The `/api/v1/systems` endpoint DTO carries
  every FEC opt-out field as `omitempty` JSON — a configured-systems
  audit is one `curl` away.
- **Daemon-level state.** Inspect the running `config.yaml` and the
  daemon's startup log lines; the config loader logs the effective
  values for every section as it parses.
- **API auth capability.** `GET /api/v1/mutations` is always open
  and reports `auth_mode` + `can_mutate` for the current request, so
  scripts and TUIs can light up write-side keybindings without
  probing real endpoints.
