---
layout: page
title: "Design: Neighbor-site following (multi-site)"
description: Acting on decoded P25 adjacent-site broadcasts — roaming follow and stronger-site selection
nav_group: Reference
---

# Design: Neighbor-site following (multi-site)

**Status:** proposal (design-first; no behaviour change in this document).
**Scope:** P25 Phase 1/2 first; the data model is protocol-neutral so DMR T3 /
NXDN can follow.

## Problem

GopherTrunk already *decodes and displays* the P25 broadcasts that describe a
system's topology, but it does not *act* on them. The package doc in
`internal/trunking/site.go:1-4` lists "(later) multi-site neighbor tracking" as
an explicit gap. SDRTrunk uses neighbor broadcasts two ways operators ask for:

1. **Roaming follow** — when a radio moves to an adjacent site, follow its calls
   there instead of losing the conversation at the site boundary.
2. **Stronger-site selection** — when the camped control channel degrades, hop
   to a neighbor site's CC that decodes more cleanly.

This proposal designs how to add those two behaviours on top of the topology data
GopherTrunk already has, **without** widening the behaviour-change surface into a
refactor (per CONTRIBUTING.md / CLAUDE.md "design first").

## What already exists (reuse, do not rebuild)

The decode + display path is complete and is the foundation here:

- **Decode.** `internal/radio/p25/phase1/opcodes.go` parses the three topology
  broadcasts: Network Status (`0x3B`), RFSS Status (`0x3A`), and **Adjacent Site
  Status** (`0x3C`, `ParseAdjacentSiteStatusBroadcast`, ~`opcodes.go:648`).
- **Accumulate.** `internal/radio/p25/phase1/network.go` folds them into a
  `NetworkModel`: `ApplyAdjacentSite` (~`network.go:200`) de-duplicates neighbors
  by `(RFSS, Site)` and votes System ID; `Snapshot()` returns the topology with
  neighbors sorted by `(RFSS, Site)`.
- **Resolve + expose.** `control.go:TopologySnapshot()` (~`control.go:225`)
  resolves each neighbor's channel ID/number to Hz via the IDEN_UP band plan and
  emits `TopoNeighborRef{RFSS, Site, ChannelID, ChannelNumber, FrequencyHz}`.
  `internal/trunking/grant.go` carries it on `SiteUpdate.Topology`;
  `internal/trunking/site_tracker.go:Topology()` keeps the latest snapshot per
  system; `internal/api/{types,handlers}.go` serves it as `SystemDTO.Neighbors`;
  `internal/trunking/network_report.go` renders it (`ReportNeighbor`,
  `RenderNeighborLines`).

**Key point:** the neighbor control-channel *frequencies* are already resolved
and live in the per-system topology snapshot. Following is a control-flow /
device-allocation problem, not a decode problem.

## Non-goals

- No automatic *wideband* multi-site coverage (one front end watching N sites at
  once) — that is a separate, hardware-bound feature. This design covers a single
  control receiver that *re-tunes* between sites, plus the existing voice-follow
  device.
- No change to how neighbors are decoded, voted, or displayed.
- No new config DSL beyond a couple of opt-in toggles (below).

## Design

### Data already in hand

`SiteTracker.Topology(system)` → `TopologySnapshot` with:
`PrimaryCC`, `Secondary[]`, and `Neighbors []TopoNeighborRef` (each with a
resolved `FrequencyHz`). That is the candidate set for both behaviours.

### Behaviour 1 — stronger-site selection (control-channel roam)

Today the CC hunter (`System.HuntOrder`, `site.go:416`) ranks only the
*operator-configured* `ControlChannels`. Extension:

1. Feed decoded neighbor CC frequencies into the hunt candidate set as a
   **secondary tier**, below the configured list (configured CCs remain the
   floor/seed; neighbors are discovered extras). Keep them separate so a bad
   decode can't permanently pollute the configured order.
2. Add a **lock-quality signal** to the decision. The decoder already surfaces
   per-lock SNR/EVM-style metrics (see CLAUDE.md DSP notes and the
   `gophertrunk_sdr_*` metrics); gate a roam on a *sustained* deficit on the
   camped CC (hysteresis + dwell, not a single bad frame) so we don't flap.
3. On roam, re-tune the control receiver to the chosen neighbor CC and let the
   existing hunter re-lock. The neighbor's own broadcasts then refine topology
   from the new vantage point.

Selection policy (start simple, document it): prefer the configured CC; consider
a neighbor only when the camped CC is below a quality threshold for N seconds;
among neighbors, pick highest recent lock quality, breaking ties by `(RFSS,
Site)` for determinism.

### Behaviour 2 — roaming voice follow

A grant on the camped site references a channel resolvable from that site's band
plan, so same-site follow is unchanged. Cross-site follow is the harder case and
is **explicitly staged last** because it needs a second receiver or a re-tune
window the control path can tolerate. Initial version: surface a *roam event*
(the call's talkgroup last seen here, now active on neighbor `(RFSS, Site)`) on
the bus and in the API, so an operator/automation can act, **without** yet
stealing the single control receiver mid-call. Full automatic cross-site audio
follow is a follow-on once Behaviour 1's re-tune machinery and quality gating are
proven.

### Config surface (opt-in, minimal)

- `system.neighbor_follow: off|select|roam` (default `off`) — `select` enables
  Behaviour 1; `roam` additionally emits roam events (Behaviour 2 stage 1).
- Reuse existing quality thresholds where possible; expose at most one
  `neighbor_roam_dwell` knob rather than a new tuning DSL.

### Interaction with existing machinery

- **CC hunter** (`site.go` / the hunt supervisor): neighbor CCs extend the
  candidate set; `HuntOrder` stays the configured-floor source of truth.
- **Grant follow** (`internal/trunking` grant path): unchanged for same-site;
  gains a roam-event emission for cross-site.
- **SiteTracker**: already the per-system topology owner — it becomes the source
  the selection policy reads from. No new global state.

## Staged implementation plan

1. **Plumb neighbor CCs into the hunt candidate set** (no auto-roam yet): expose
   them as discovered candidates, log/metric them, keep configured order
   authoritative. Verifiable with a topology fixture → expected candidate list.
2. **Quality-gated control-channel roam** (Behaviour 1): add hysteresis/dwell
   selection over camped-vs-neighbor lock quality; re-tune on sustained deficit.
   Verifiable with a synthetic two-CC replay where one CC degrades.
3. **Roam-event emission** (Behaviour 2 stage 1): bus + API event when a tracked
   talkgroup appears on a neighbor site. No receiver stealing.
4. **Automatic cross-site voice follow** (Behaviour 2 stage 2): only after 1-3
   are proven and the device/re-tune budget is understood.

Each stage is a separate PR with its own failing-first test; stages 1-3 are
testable offline against topology/replay fixtures (no second radio required).

## Open questions for the maintainer

- Single re-tuning control receiver vs. requiring a dedicated dongle per site for
  true simultaneous multi-site (the `gophertrunk_sdr_iq_power_dbfs`/per-tap story
  in `daemon.go` already warns weak co-tenant sites need their own front end).
- Whether `select` should ever override a *configured* CC, or only ever add
  neighbors as extras.
- Quality metric to gate on: reuse the demod SNR/EVM the replay path reports, or
  a cheaper proxy (TSBK CRC pass rate) for the live decision.
