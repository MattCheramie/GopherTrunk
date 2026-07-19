---
title: "Trunking Engine, Part 10: Sites, Topology & Multi-Site Roaming"
description: How GopherTrunk accumulates a P25 system's site map from control-channel status broadcasts, tracks a radio roaming between sites, and renders an SDRtrunk-style network configuration report — all derived from the event stream.
category: deep-dives
keywords: p25 site tracking, rfss status broadcast, multi-site trunking, network topology, neighbor sites, roaming, network configuration report, site tracker, gophertrunk trunking, wacn system id
tags: [trunking, go, p25, sites, topology, event-bus]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 10
---

*Part 10 of **Trunking Engine**, a 12-part deep dive into the "brain" of
GopherTrunk. Parts 8 and 9 built derived tables from the event stream — a unit
roster, a patch registry. This one builds the biggest derived table of all: a map
of the whole radio system, site by site, as the control channel reveals it.*

> **TL;DR:** A trunked system is rarely one transmitter. The `SiteTracker`
> subscribes to `KindSiteUpdate` events — published every time the decoder parses
> an RFSS Status Broadcast — and accumulates a table of every P25 site GT has
> heard, keyed by `(system, RFSS, site)`. Entries never expire: a site roster is
> small and stable, and operators want the full list even for quiet sites.
> Alongside it, a per-system `TopologySnapshot` carries the identity, neighbours,
> and band plan that *aren't* on any per-event payload, and `RenderNetworkReport`
> turns it into an SDRtrunk-style network-configuration report.

**Key takeaways**

- **A site map is derived state**, like everything else in this series — folded
  from `KindSiteUpdate` events, not queried from the radio.
- `SiteUpdate` is the only place the **control-channel frequency and the site
  identity are joined** — grants carry the voice channel, not the CC.
- `TopologySnapshot` exists because identity fields (WACN/SYSID/RFSS/Site) are
  accumulated *inside the decoder's network model* and never ride a per-event
  payload — so the snapshot is the bridge out.
- **Roaming falls out for free**: as the hunter hops control channels, each site's
  status broadcast lands a new row, so a multi-site system surfaces every site
  over time.

## Cheat sheet

| Concept | Where it lives | One-line role |
|---|---|---|
| `SiteInfo` | `site_tracker.go` | one discovered site row: RFSS/Site, CC freq, decode quality |
| `SiteTracker` | `site_tracker.go` | subscriber that folds `KindSiteUpdate` into a live site table |
| `SiteUpdate` | `grant.go` | `KindSiteUpdate` payload; from RFSS Status Broadcast (TSBK 0x3A) |
| `TopologySnapshot` | `topology.go` | protocol-neutral system map: identity, neighbours, band plan |
| `NetworkReport` / `RenderNetworkReport` | `network_report.go` | display-ready report the renderer prints |
| `SiteTracker.Report(system)` | `site_tracker.go` | renders the network-config report for one system |

## In this post

- **Why a system is a graph of sites**, not a single transmitter.
- **The `SiteTracker`** — folding `KindSiteUpdate` into a never-expiring site
  table.
- **`TopologySnapshot`** — the bridge for identity the event stream can't carry.
- **The network report** — turning topology into an SDRtrunk-style dump — and how
  roaming emerges from control-channel hopping.

## A system is a graph of sites

A P25 system the size of a state's public-safety network is dozens of **sites** —
individual transmitter locations — tied together as one logical system. Each site
has its own RFSS (RF SubSystem) and Site ID, its own control channel, and a list
of **neighbours**: adjacent sites it advertises so a roaming radio knows where to
go next. A radio moving across the coverage area re-registers as it crosses site
boundaries, and a wide-area call is repeated on every participating site.

For a scanner this matters because the interesting structure — who neighbours whom,
which channel is which site's control channel, what the band plan is — is only
visible over time and only on the control channel. The decoder learns it by
parsing the site's periodic **RFSS Status Broadcast** (TSBK 0x3A) and related
status messages. GopherTrunk's job is to *accumulate* those observations into a map
that outlives any single lock.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="Three sites each publish site-update events as the decoder hops their control channels; the site tracker folds them into a keyed table and a per-system topology snapshot feeds the network report">
  <circle cx="70" cy="50" r="26" fill="none" stroke="var(--accent)"/>
  <text x="70" y="47" text-anchor="middle" fill="var(--accent)" font-size="10">RFSS 1</text>
  <text x="70" y="60" text-anchor="middle" fill="var(--fg-muted)" font-size="9">site 7</text>
  <circle cx="70" cy="130" r="26" fill="none" stroke="var(--accent)"/>
  <text x="70" y="127" text-anchor="middle" fill="var(--accent)" font-size="10">RFSS 1</text>
  <text x="70" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">site 9</text>
  <circle cx="150" cy="90" r="26" fill="none" stroke="var(--accent)"/>
  <text x="150" y="87" text-anchor="middle" fill="var(--accent)" font-size="10">RFSS 2</text>
  <text x="150" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">site 3</text>
  <line x1="94" y1="62" x2="128" y2="80" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <line x1="94" y1="118" x2="128" y2="100" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <line x1="70" y1="76" x2="70" y2="104" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="150" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">neighbours advertised on CC</text>
  <line x1="182" y1="90" x2="236" y2="90" stroke="currentColor"/>
  <polygon points="236,86 246,90 236,94" fill="currentColor"/>
  <text x="210" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="9">KindSiteUpdate</text>
  <rect x="246" y="66" width="150" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="321" y="86" text-anchor="middle" fill="var(--accent)" font-size="12">SiteTracker</text>
  <text x="321" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="10">sites[(sys,rfss,site)]</text>
  <line x1="396" y1="90" x2="450" y2="90" stroke="currentColor"/>
  <polygon points="450,86 460,90 450,94" fill="currentColor"/>
  <rect x="460" y="46" width="204" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="562" y="64" text-anchor="middle" fill="currentColor" font-size="10">Snapshot() → GET /api/v1/sites</text>
  <text x="562" y="79" text-anchor="middle" fill="var(--fg-muted)" font-size="9">every site, ordered by RFSS/Site</text>
  <rect x="460" y="96" width="204" height="40" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="562" y="114" text-anchor="middle" fill="currentColor" font-size="10">Report() → network config report</text>
  <text x="562" y="129" text-anchor="middle" fill="var(--fg-muted)" font-size="9">from the topology snapshot</text>
  <text x="330" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the system map is a fold over status broadcasts — no site is ever queried directly</text>
</svg>
<figcaption>Each site's control channel advertises its identity and neighbours; the tracker folds those updates into a keyed table that outlives any single lock.</figcaption>
</figure>

## The SiteTracker

`SiteTracker` is the same subscriber shape as the affiliation tracker, minus the
TTL sweep. Its `Run` loop drains the bus and folds a single event kind:

```go
// internal/trunking/site_tracker.go (shape)
func (t *SiteTracker) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case ev, ok := <-t.sub.C:
            if !ok {
                return nil
            }
            if ev.Kind == events.KindSiteUpdate {
                if u, ok := ev.Payload.(SiteUpdate); ok {
                    t.observe(u)
                }
            }
        }
    }
}
```

`observe` upserts a `SiteInfo` keyed by `siteKey{system, rfss, site}`. The row
carries more than identity — it records the control-channel frequency, the demod's
measured carrier offset (a large value flags an adjacent site bleeding through at
12.5 kHz spacing, #815), and a cumulative TSBK error rate that tracks decode
quality independently of carrier lock (#858). The upsert is careful about
zero-valued fields: a fresh lock or a non-TSBK Phase 2 update carries no stats, so
`observe` only refreshes the error rate when the update actually has a TSBK count,
never clobbering a real reading with a zero.

The deliberate difference from Part 8 is that **entries never expire**. A unit
roster tracks a live, churning population, so it ages out. A site roster is small
and stable — you want the full list of a system's sites even for one that has gone
quiet at 3 a.m. — so the tracker accumulates forever. This is why it can't just
mirror the decoder's camped-site model, which only ever knows the *current* site:
as the hunter hops control channels the tracker keeps every site it has ever heard.

```go
// internal/trunking/site_tracker.go (shape)
type SiteInfo struct {
    System                        string
    RFSSID, SiteID                uint8
    ControlChannelHz              uint32
    ControlChannelCarrierOffsetHz int32   // off-frequency lock flag (#815)
    ControlChannelTSBKErrorRate   float64 // decode quality (#858)
    ControlChannelTSBKCount       int64
    WACN                          uint32
    SystemID                      uint16
    FirstSeen, LastSeen           time.Time
}
```

## Why TopologySnapshot exists

Here is a subtlety that is easy to miss and that the code calls out explicitly.
The identifying fields of a P25 system — WACN, System ID, RFSS, Site, and the band
plan — are **not carried on any per-event payload**. They are accumulated *inside*
the decoder's network model from a run of periodic status broadcasts. A consumer
reading the event stream alone cannot reconstruct them. So `TopologySnapshot` is
the deliberate bridge: a protocol-neutral struct the decoder fills in and attaches,
carrying identity, secondary control channels, neighbours, and the band plan out
to anyone who needs the *shape* of the system rather than its moment-to-moment
activity.

It lives in package `trunking` (not in the signal-lab engine) specifically so the
per-protocol control-channel decoders can implement `TopologyProvider` without an
import cycle. Capability varies by protocol — P25 surfaces full identity plus
neighbours plus band plan; DMR Tier III, EDACS and Motorola add identity and
neighbours; NXDN and TETRA give single-site identity — and every field is optional,
so a decoder fills in only what it can observe. `SiteUpdate` carries the latest
snapshot when it has one, and `observe` stores it per system name so the report can
be rendered later without reaching back into the decoder.

<figure class="lab-figure">
<svg viewBox="0 0 680 170" width="680" height="170" role="img" aria-label="Per-event payloads carry only per-call activity, while system identity accumulates inside the decoder network model and is exported through the topology snapshot, which is rendered into the network report">
  <rect x="14" y="30" width="150" height="52" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="89" y="50" text-anchor="middle" fill="currentColor" font-size="11">decoder network model</text>
  <text x="89" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="9">WACN/SYSID/RFSS/site,</text>
  <text x="89" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">neighbours, band plan</text>
  <line x1="164" y1="56" x2="222" y2="56" stroke="var(--accent)"/>
  <polygon points="222,52 232,56 222,60" fill="var(--accent)"/>
  <text x="196" y="46" text-anchor="middle" fill="var(--accent)" font-size="9">TopologyProvider</text>
  <rect x="232" y="30" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="307" y="54" text-anchor="middle" fill="var(--accent)" font-size="12">TopologySnapshot</text>
  <text x="307" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="9">the bridge out</text>
  <line x1="382" y1="56" x2="440" y2="56" stroke="currentColor"/>
  <polygon points="440,52 450,56 440,60" fill="currentColor"/>
  <rect x="450" y="30" width="130" height="52" rx="6" fill="none" stroke="currentColor"/>
  <text x="515" y="50" text-anchor="middle" fill="currentColor" font-size="11">ReportFrom-</text>
  <text x="515" y="64" text-anchor="middle" fill="currentColor" font-size="11">Topology()</text>
  <line x1="515" y1="82" x2="515" y2="112" stroke="currentColor"/>
  <polygon points="511,112 515,122 519,112" fill="currentColor"/>
  <rect x="410" y="122" width="210" height="34" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="515" y="143" text-anchor="middle" fill="var(--fg-muted)" font-size="10">RenderNetworkReport → text dump</text>
  <rect x="30" y="112" width="300" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="180" y="130" text-anchor="middle" fill="currentColor" font-size="10">per-event payloads (Grant, CallEnd, …)</text>
  <text x="180" y="145" text-anchor="middle" fill="var(--fg-muted)" font-size="9">carry per-call activity only — no system identity</text>
</svg>
<figcaption>Activity rides the event stream; system identity does not. The topology snapshot is the deliberate side channel that carries the map out to the report renderer.</figcaption>
</figure>

## The network report

The last mile turns a topology snapshot into GopherTrunk's answer to SDRtrunk's
P25 network-configuration dump. `NetworkReport` is a display-ready, protocol-neutral
struct deliberately kept separate from `TopologySnapshot` — the adapter
(`ReportFromTopology`) does all the band-plan math, resolving each channel to
absolute downlink and uplink frequencies, so the renderer stays pure and
golden-test friendly:

```text
P25 Network Configuration — Metro Regional
Network
  WACN:BEE00[781824] SYSTEM:2C2[706] NAC:293[659] LRA:5[5]
Current Site
  RFSS:1[1] SITE:7[7] LRA:5[5]
  PRI CONTROL CHANNEL:1-1754 DOWNLINK:851.012500 MHz UPLINK:806.012500 MHz
  NEIGHBOR RFSS:1[1] SITE:9[9] CHANNEL:1-1782 DOWNLINK:851.362500 MHz ...
Frequency Bands
  BAND:1 TDMA BASE:851.000000 MHz BANDWIDTH:12.5 kHz SPACING:12.5 kHz OFFSET:-45.000000 MHz
```

`RenderNetworkReport` sorts sites, neighbours, and bands into a stable order and
suppresses empty sections. When more than one site is present the header switches
from `Current Site` to `Sites`, which is exactly the **roaming** case: as the
hunter hops between a system's control channels over a session, each site's status
broadcast lands a fresh row, and the same renderer that prints one camped site
prints the whole discovered network. `SiteTracker.Report(system)` wires this up for
`GET /api/v1/systems/{name}/report`, and `RenderNeighborLines` reuses the same
band-plan resolution to annotate the decoded-message log the way SDRtrunk's
"Neighbor Sites" block does.

### How that principle shaped the Go code

Two seams keep this clean. First, **separation of accumulation from rendering**:
the tracker accumulates raw observations, the snapshot carries the map, the report
type resolves frequencies, and the renderer only formats. Each stage is testable in
isolation, and the renderer never does band-plan arithmetic. Second, **derived
state again**: the tracker adds no RF work — it reads the same bus the engine does.
Multi-site roaming isn't a feature that was *built*; it's what the fold produces
once the hunter is hopping control channels, because absence of a fresh update for
a site simply leaves its last-known row in place.

## Where this goes next

[Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})
turns to policy: what the engine does when a call it has tuned turns out to be
encrypted — follow it, grab its metadata and release the tuner, or ignore it
outright — and why tuner starvation makes that choice matter. For the multi-site
background here, the [roaming]({{ '/reference/roaming/' | relative_url }}) and
[control-channel]({{ '/reference/control-channel/' | relative_url }}) references
fill in the P25 mechanics.

## FAQ

**How does GopherTrunk discover a system's other sites?**
It folds every `KindSiteUpdate` — published when the decoder parses an RFSS Status
Broadcast — into a table keyed by `(system, RFSS, site)`. As the control-channel
hunter hops frequencies over a session, each site it locks contributes its
identity and its advertised neighbours, so the roster fills in over time.

**Why don't site entries expire like affiliation entries do?**
A radio population churns constantly, so the unit roster ages out. A system's site
list is small and stable, and operators want the complete list even for a site
that has gone quiet — so the site tracker accumulates entries permanently rather
than sweeping them.

**Why is there a separate TopologySnapshot instead of just using events?**
Because a P25 system's identity (WACN, System ID, RFSS/Site) and band plan aren't
on any per-event payload — they accumulate inside the decoder's network model from
periodic status broadcasts. The snapshot is the bridge that carries that
un-event-able state out to the API and the report.

**What is the network report and how does it show roaming?**
It's GopherTrunk's SDRtrunk-style network-configuration dump — network identity,
each site's control channels and neighbours, and the band plan, with frequencies
fully resolved. When more than one site has been heard the renderer switches its
header from "Current Site" to "Sites", so a multi-site session prints the whole
discovered network.

## Series navigation

**Part 10 of 12** · ←
[Part 9: Patches & Supergroups]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }})
· Next →
[Part 11: Encrypted-Mode Handling]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})
