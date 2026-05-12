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

## 1. Protocol-layer FEC opt-ins

Every protocol that has a public-spec forward-error-correction chain
ships the chain as opt-in. The connector constructs each
`ControlChannel` in its legacy raw-bit mode by default and only flips
on the FEC layer when the operator sets a per-system key in
`config.yaml`. Empty / absent keys preserve the legacy path so the
synthesized-fixture tests stay green and operators with pre-stripped
capture files (DSD-FME `-r` dumps, OP25 fixtures) don't see surprise
CRC failures. The shared rationale is documented in
[`README.md` §"FEC opt-ins"](../README.md) (the existing reference
table lives at lines 1960-2005); this section adds the **Remediation**
column the README does not currently cover.

All seven YAML keys are defined on the `System` struct at
[`internal/trunking/site.go:101-177`](../internal/trunking/site.go).

| Protocol | YAML key(s) | Setter | Why opt-in | Remediation |
| --- | --- | --- | --- | --- |
| **TETRA** | `tetra_colour_code` (uint32, low 30 bits), `tetra_channel` (`sch/hd` / `sch/f` / `sch/hu` / `bsch` / `aach`) | `tetra.ControlChannel.SetChannelCoding` at [`internal/radio/tetra/control.go:156`](../internal/radio/tetra/control.go) | Pre-stripped DSD-FME / OP25 fixtures and synthesized in-package fixtures would fail CRC under the full ETSI EN 300 392-2 §8.3.1 descramble + deinterleave + depuncture + Viterbi + CRC-16 chain. Zero colour code is also valid for BSCH, so the field cannot simply infer "on" from non-zero. | Re-encode or capture TETRA TMO fixtures with valid scrambler colour codes and full type-5 → type-1 framing, then flip the connector to construct each `tetra.ControlChannel` with `ChannelCodingOn` by default. Make `tetra_colour_code` a required field for non-BSCH systems and surface validation at config load. |
| **LTR** | `ltr_fcs_mode` (`off` / `on`), `ltr_manchester_mode` (`off` / `nrz` / `strict` / `soft`) | `ltr.ControlChannel.SetFCSMode` at [`internal/radio/ltr/control.go:108`](../internal/radio/ltr/control.go); `ltr.ControlChannel.SetManchesterMode` at [`internal/radio/ltr/process.go:49`](../internal/radio/ltr/process.go) | Default NRZ + no FCS matches the in-package synthesized fixtures. Live sub-audible captures need `manchester: soft` + `fcs: on`, but flipping the default would break the existing test suite. | Re-encode in-package LTR fixtures with mid-bit transitions and valid CRC-7 FCS trailers (matching sdrtrunk's `CRCLTR.java` layout), then default to `manchester: soft` + `fcs: on`. Keep `nrz` available for synthesized-fixture regression tests behind a test-only knob. |
| **P25 Phase 2** | `p25_phase2_trellis_mode` (`off` / `on`) | `p25phase2.ControlChannel.SetTrellisMode` at [`internal/radio/p25/phase2/control.go:77`](../internal/radio/p25/phase2/control.go) | Legacy 72-dibit raw-MAC-PDU fixtures predate the trellis decoder. Live captures arrive as 146 channel dibits per the TIA-102.AABF 4-state ½-rate trellis FEC. | Regenerate Phase 2 fixtures as 146-channel-dibit streams that exercise the trellis decoder. Once the suite is green, flip the connector to construct with `TrellisOn` and either drop the raw-MAC-PDU path or relocate it to a test-only fixture mode. |
| **NXDN** | `nxdn_viterbi_mode` (`off` / `on` / `spec`) | `nxdn.ControlChannel.SetViterbiMode` at [`internal/radio/nxdn/control.go:102`](../internal/radio/nxdn/control.go) | Legacy 44-dibit raw-CAC fixtures from MMDVMHost / DSDcc. `on` matches the older synthesized fixtures (K=5 ½-rate Viterbi over 92 dibits → 88 info + 4 tail). `spec` is the full NXDN-TS-1-A §4.5.1.1 chain (150 dibits → deinterleave 25×12 → depuncture → Viterbi → CRC → 155 info bits). | Re-encode fixtures to the full spec chain, default to `spec`, and remove the intermediate `on` mode. The two-step path is a transitional artefact that costs documentation surface and operator confusion. |
| **EDACS / GE-Marc** | `edacs_bch_mode` (`off` / `on`) | `edacs.ControlChannel.SetBCHMode` at [`internal/radio/edacs/control.go:96`](../internal/radio/edacs/control.go) | Legacy CCW fixtures arrive pre-stripped: the 40 bits on the wire under `off` are treated as data only, with the LCN bit 0 + Aux fields parsed in place of BCH parity. | Rebuild fixtures with on-wire 40-bit CCWs that include BCH(40, 28, 2) parity. The connector then defaults to `BCHOn`, the parsed payload narrows to the 28 spec-defined info bits, and single/double-bit error correction comes online without operator action. |
| **MPT 1327** | `mpt1327_bch_mode` (`off` / `on`) | `mpt1327.ControlChannel.SetBCHMode` at [`internal/radio/mpt1327/control.go:108`](../internal/radio/mpt1327/control.go) | Legacy 38-bit pre-stripped codeword path. `on` decodes the 64-bit on-wire BCH(63, 38) form. | Re-encode fixtures as 64-bit BCH(63, 38) codewords, flip the connector default to `BCHOn`. |
| **Motorola Type II** | `motorola_bch_mode` (`off` / `on`) | `motorola.ControlChannel.SetBCHMode` at [`internal/radio/motorola/process.go:41`](../internal/radio/motorola/process.go) | Legacy 32-bit raw-OSW fixtures (DSD-FME `-r` dumps) carry no BCH parity. Live captures should always set `on` — the README already calls this out. | Re-encode fixtures with the dual BCH(64, 16, 11) codeword layout. Flip the connector default to `BCHOn`. This is the lowest-risk flip in the table — the docs already say live captures need `on`, so the default is actively wrong for any real-RF user. |

**Shared follow-up.** A single PR can convert all seven simultaneously
by introducing a `legacy_fixtures` test-only mode that the regression
suite opts into, then flipping the connector defaults. The
alternative — runtime auto-detection that tries the FEC path first
and falls back to raw on N consecutive CRC failures — has been
discussed for P25 Phase 2 (see the "follow-up" line in
[`README.md:125-135`](../README.md)) and is applicable to all seven
protocols. Auto-detect costs a small CPU burst on broken input but
eliminates the operator config burden entirely.

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
| **API mutation endpoints** | `api.allow_mutations` | `false` ([`internal/config/config.go`](../internal/config/config.go) APIConfig, [`config.example.yaml:9-16`](../config.example.yaml)) | The HTTP API has **no authentication**. Enabling mutations exposes end-call, retention-sweep, scanner control, talkgroup PATCH, tone-out reset, and audio PATCH endpoints to anything that can reach the listener. The flag is a deliberate security gate, not a feature toggle. | Add an auth layer (bearer token / mTLS / unix-socket peer-cred check). Once auth exists, the flag can be retired or auto-enabled based on listener binding (loopback ⇒ auto-on, public ⇒ require auth headers + explicit opt-in). Until auth lands, the gate must stay. |
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
3. **Add an auth layer.** Specific to `api.allow_mutations`. Until
   the HTTP API is authenticated, the gate must stay.
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
