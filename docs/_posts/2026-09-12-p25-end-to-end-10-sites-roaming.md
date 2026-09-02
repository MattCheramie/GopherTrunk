---
title: "P25 End to End, Part 10: Sites, WACNs & Roaming a Multi-Site System"
description: The P25 identity ladder — a 20-bit WACN, 12-bit System ID, RFSS and Site — and the status broadcasts that carry it in TSBK and multi-block AMBT forms. How GopherTrunk votes a system's identity out of noisy frames, accumulates a neighbour map, and roams by hunting control channels.
category: deep-dives
keywords: p25 wacn system id, rfss status broadcast, p25 adjacent site status, p25 neighbor sites, multi-site p25 scanner, p25 network status broadcast, p25 control channel hunting, p25 site roaming, gophertrunk p25
tags: [p25-end-to-end, p25, trunking, multi-site, topology, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "P25 End to End"
series_part: 10
---

*Part 10 of **P25 End to End**, a 14-part deep dive that follows North
America's dominant trunking protocol through GopherTrunk — from a raw C4FM
carrier to recorded, named, multi-site voice.
[Part 9]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }})
pinned down where "encrypted" lives on the air and what policy does about
it. This part climbs above any single call, to the question a P25 network
answers continuously on its control channel: *who am I, and who are my
neighbours?* The identity ladder from WACN down to Site, the broadcasts
that carry it in two different frame formats, and how GopherTrunk turns a
stream of individually untrustworthy frames into a site map an operator can
roam by.*

> **TL;DR:** A P25 site names itself with a ladder of identifiers — a
> 20-bit **WACN**, 12-bit **System ID**, 8-bit **RFSS** and **Site** — and
> repeats them in four status broadcasts: Network Status (0x3B), RFSS
> Status (0x3A), Secondary Control Channel (0x39), Adjacent Site Status
> (0x3C). Each also exists in a multi-block **AMBT** form
> ([Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }})),
> and some systems broadcast their richest data — the WACN, most of the
> neighbour list — *only* that way. `phase1.NetworkModel`
> (`internal/radio/p25/phase1/network.go`) accumulates them with a
> **majority vote** on identity scalars, because a corrupt-but-CRC-passing
> TSBK must not poison the reported identity. Above it,
> `trunking.SiteTracker` folds every `KindSiteUpdate` into a per-system
> site roster (`GET /api/v1/sites`, issue #698) — and roaming falls out of
> control-channel hunting (`System.HuntOrder`), not a feature anyone
> built.

**Key takeaways**

- **Identity arrives in fragments, on the system's schedule.** No single
  frame carries the whole ladder: 0x3B has the WACN but no RFSS/Site, 0x3A
  the reverse. A decoder waiting for one authoritative message reports
  blanks forever.
- **The same broadcast wears two frames, and one may be missing.** Every
  status message has a TSBK and an AMBT form with complementary fields.
  GopherTrunk once decoded only the TSBK form — and showed one neighbour
  where SDRtrunk listed twelve.
- **CRC-clean is not true.** Identity scalars are resolved by majority
  vote, zeros never get a vote, and a neighbour merge keeps whichever half
  a new observation lacks — corroboration, not last-write-wins.
- **Roaming is a fold, not a feature.** The site tracker accumulates every
  site the hunter ever locks; hopping the configured control channels *is*
  the roam, biased toward the last-known-good frequency.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Identity + neighbour accumulator | votes WACN/SysID/RFSS/Site, merges neighbours | `internal/radio/p25/phase1/network.go` (`NetworkModel`) |
| Status broadcast parsers | TSBK 0x3B/0x3A/0x39/0x3C + AMBT twins | `phase1/tsbk.go`, `phase1/mbt.go` (`ApplyMBT*` folds) |
| Protocol-neutral snapshot | identity + neighbours + band plan, per system | `internal/trunking/topology.go` (`TopologySnapshot`) |
| Live site roster | every site seen, keyed (system, RFSS, site) | `internal/trunking/site_tracker.go` (`SiteTracker`, `GET /api/v1/sites`) |
| Network report | SDRtrunk-style rendered topology | `internal/trunking/network_report.go` (`RenderNeighborLines`) |
| CC hunting order | last-known-good CC first, then configured order | `internal/trunking/site.go` (`System.HuntOrder`), `trunking.Hunter` |
| Wrong-site instrument | carrier offset of the locked CC vs tuned centre | `SiteInfo.ControlChannelCarrierOffsetHz` (issue #815) |

## In this post

- **The identity ladder** — WACN, System, RFSS, Site, and what each scopes.
- **Four broadcasts, two frame formats** — and the systems that only send one.
- **Votes, not last-write-wins** — surviving corrupt-but-CRC-passing frames.
- **The neighbour map** — merging complementary halves of the same site.
- **Roaming is a fold** — the site tracker, the hunter, and the #815 instrument.

## The identity ladder

Four numbers, nested like postal geography:

| Field | Width | Scopes | Carried by |
|---|---|---|---|
| WACN | 20 bits | a wide-area network (one licensee's whole P25 world) | Network Status (0x3B) |
| System ID | 12 bits | one trunked system inside the WACN | 0x3B, 0x3A, and every 0x3C neighbour |
| RFSS | 8 bits | an RF sub-system — a cluster of sites | RFSS Status (0x3A) |
| Site | 8 bits | one physical site inside the RFSS | 0x3A, plus 0x3C for neighbours |

A radio's fully-qualified home is WACN + System ID; RFSS + Site say *where
in the network* it currently is. None of this is the NAC from
[Part 2]({{ '/blog/deep-dives/p25-end-to-end-02-sync-nid-lock/' | relative_url }})
— the NAC is a 12-bit per-frame access colour code that gates which frames
a receiver accepts, and while operators often align it with the site, it
identifies nothing beyond "this transmission belongs here." The
[wacn]({{ '/reference/wacn/' | relative_url }}) and
[rfss]({{ '/reference/rfss/' | relative_url }}) reference pages carry the
field-guide versions;
[Trunking Engine Part 10]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
covers the engine-side machinery this part grounds in the P25 bits.

The point of the ladder is roaming: a radio camped on a fading site needs
to know which *adjacent* sites carry the same system, on which control
channels — exactly the information a scanner wants for the same reason.

## Four broadcasts, two frame formats

A P25 control channel repeats a rotation of status broadcasts between
grants
([Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})):

| Broadcast | TSBK opcode | What it contributes |
|---|---|---|
| Network Status | 0x3B | WACN, System ID, primary CC channel |
| RFSS Status | 0x3A | System ID, RFSS, Site, primary CC |
| Secondary CC | 0x39 (+ 0x29 explicit) | additional control channels of this site |
| Adjacent Site Status | 0x3C | one neighbour: RFSS, Site, its CC, CFVA flags |

Every one also exists in the **AMBT** form — the same semantic message
carried in a multi-block PDU
([Part 4]({{ '/blog/deep-dives/p25-end-to-end-04-mbt-ambt/' | relative_url }}))
— and the two forms are not redundant copies. The AMBT Adjacent Site
Status alone names the neighbour's **explicit uplink channel**; the TSBK
form alone carries the CFVA flags and service class. And many systems,
notably Motorola, broadcast most of their neighbour list — sometimes the
WACN itself — **only** in AMBT form.

That asymmetry is the war story this series keeps returning to. GopherTrunk
originally decoded only the TSBK forms and logged every control-channel PDU
as `non-control DUID duid=PDU` — so an operator watched SDRtrunk list
**twelve neighbours with uplinks in fifteen seconds** while GT showed one
neighbour and "No Network Status Broadcast yet" on the same carrier. The
frames carrying everything they were missing were being counted as spam.
The fix is `mbt.go`'s AMBT decoder, validated against OP25's `process_PDU`
and SDRtrunk's AMBTC message classes, feeding the same accumulator through
the `ApplyMBT*` folds. One detail worth repeating from Part 4 because it
decides real frequencies: an explicit uplink channel resolves as plain
base + spacing with **no transmit offset** — the uplink channel number
already encodes the uplink frequency
([Part 5]({{ '/blog/deep-dives/p25-end-to-end-05-channels-band-plans/' | relative_url }})
owns that arithmetic).

## Votes, not last-write-wins

Every frame reaching the accumulator has already survived trellis decode
and CRC
([Part 3]({{ '/blog/deep-dives/p25-end-to-end-03-tsbk-workhorse/' | relative_url }})).
That is not enough. A CRC-CCITT16 occasionally passes on corrupted payload,
and the control-channel state machine evaluates a grid of NID/TSBK
alignment hypotheses per frame, multiplying the exposure. A single
corrupt-but-CRC-passing frame must not rewrite the reported WACN or inject
a phantom neighbour. So identity scalars are **voted**:

```go
// internal/radio/p25/phase1/network.go (shape)
func (m *NetworkModel) ApplyNetworkStatus(n NetworkStatusBroadcast) {
    /* … lock, ensure maps … */
    // WACN/SysID are meaningless at zero (absent), so a zero never gets
    // a vote — it cannot out-rank a real value seen the same number of times.
    if n.WACN != 0 {
        m.wacnVotes[n.WACN]++
    }
    if n.SystemID != 0 {
        m.sysidVotes[n.SystemID]++
    }
    m.votePrimary(n.ChannelID, n.ChannelNumber)
}

// Snapshot reports the most-observed value per field.
func majority[K ~uint8 | ~uint16 | ~uint32](votes map[K]int) K { /* … */ }
```

The asymmetries are deliberate. A lone observation is still surfaced —
latency matters on a fresh lock — but a repeated correct value out-votes a
one-shot wrong one. Zero never votes for WACN/SysID/LRA (zero means
"absent" there) while **RFSS 0 and Site 0 always count**, because RFSS 0 is
a real, common value. Neighbours and band-plan slots surface on first
sighting, de-duplicated, matching OP25's latency — the identity vote
already absorbs the one-shot corruption that actually hurts.

Two quieter tricks fill gaps real systems leave. An Adjacent Site broadcast
votes the *neighbour's* System ID into our own tally — every site of a P25
system shares one System ID, so a neighbour names us, and on systems that
emit 0x3C but never 0x3B/0x3A this is the **only** System ID source (OP25
and SDRtrunk corroborate the same way). And an *accepted* Location
Registration Response (0x2B) votes RFSS/Site, since it names the camped
site — but only response value 0: a denied registration may name a
different site.

## The neighbour map

Neighbours are keyed by (RFSS, Site) and **merged**, not overwritten,
because the two frame forms carry complementary halves:

```go
// internal/radio/p25/phase1/network.go (shape) — upsertNeighbor
if old, ok := m.neighborData[key]; ok {
    if n.UplinkID == 0 && n.UplinkNumber == 0 { // AMBT-only field
        n.UplinkID, n.UplinkNumber = old.UplinkID, old.UplinkNumber
    }
    if !n.CFVAKnown { // TSBK-only fields
        n.CFVA, n.CFVAKnown, n.ServiceClass = old.CFVA, old.CFVAKnown, old.ServiceClass
    }
    /* … keep old LRA / SystemID when the new observation lacks them … */
}
m.neighborData[key] = n
```

`CFVAKnown` exists because "flags all zero" and "never seen" are different
facts — the same discipline as the voted zeros. The snapshot of all this
(`NetworkConfig`) crosses into the protocol-neutral
`trunking.TopologySnapshot`, which the
[network configuration report]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
renders and the hunt accumulator folds into discovered systems.

<figure class="lab-figure">
<svg viewBox="0 0 680 236" width="680" height="236" role="img" aria-label="The P25 identity tree: a 20-bit WACN contains a 12-bit System ID, which contains RF sub-systems, which contain sites. Three sites are drawn under RFSS 1; the camped site is highlighted and double-headed adjacency arrows labelled with the 0x3C broadcast connect it to its two neighbours, one annotated as AMBT-only with an explicit uplink.">
  <rect x="240" y="10" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="29" text-anchor="middle" fill="currentColor" font-size="10">WACN (20 bits) — e.g. 0xBEE00</text>
  <line x1="340" y1="40" x2="340" y2="56" stroke="var(--fg-muted)"/>
  <rect x="240" y="56" width="200" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="75" text-anchor="middle" fill="currentColor" font-size="10">System ID (12 bits) — shared by all sites</text>
  <line x1="340" y1="86" x2="340" y2="102" stroke="var(--fg-muted)"/>
  <rect x="255" y="102" width="170" height="28" rx="6" fill="none" stroke="currentColor"/>
  <text x="340" y="120" text-anchor="middle" fill="currentColor" font-size="10">RFSS 1 (8 bits)</text>
  <line x1="300" y1="130" x2="150" y2="160" stroke="var(--fg-muted)"/>
  <line x1="340" y1="130" x2="340" y2="160" stroke="var(--fg-muted)"/>
  <line x1="380" y1="130" x2="530" y2="160" stroke="var(--fg-muted)"/>
  <rect x="80" y="160" width="140" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="150" y="176" text-anchor="middle" fill="currentColor" font-size="10">Site 1</text>
  <text x="150" y="191" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CC 851.0125 MHz</text>
  <rect x="270" y="160" width="140" height="40" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="340" y="176" text-anchor="middle" fill="var(--accent)" font-size="10">Site 2 — camped</text>
  <text x="340" y="191" text-anchor="middle" fill="var(--fg-muted)" font-size="9">0x3A votes RFSS/Site</text>
  <rect x="460" y="160" width="140" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="530" y="176" text-anchor="middle" fill="currentColor" font-size="10">Site 3</text>
  <text x="530" y="191" text-anchor="middle" fill="var(--fg-muted)" font-size="9">AMBT-only: explicit uplink</text>
  <line x1="222" y1="180" x2="268" y2="180" stroke="var(--accent)"/>
  <polygon points="226,176 218,180 226,184" fill="var(--accent)"/><polygon points="264,176 272,180 264,184" fill="var(--accent)"/>
  <line x1="412" y1="180" x2="458" y2="180" stroke="var(--accent)"/>
  <polygon points="416,176 408,180 416,184" fill="var(--accent)"/><polygon points="454,176 462,180 454,184" fill="var(--accent)"/>
  <text x="340" y="222" text-anchor="middle" fill="var(--fg-muted)" font-size="9">0x3C Adjacent Site Status, TSBK + AMBT forms merged per (RFSS, Site) — each neighbour also names our System ID</text>
</svg>
<figcaption>The identity tree GopherTrunk assembles from broadcasts: identity scalars voted, neighbours merged from two frame forms that each carry half the picture.</figcaption>
</figure>

## Roaming is a fold, not a feature

Above the decoder, `trunking.SiteTracker` subscribes to the bus and folds
every `KindSiteUpdate` — published when the control channel decodes an RFSS
Status Broadcast — into a live table keyed (system, RFSS, site). Where the
`NetworkModel` only knows the *currently camped* site, the tracker
remembers **every site the decoder has ever locked**, and entries never
expire: a site roster is small and stable, and operators want the full
list even for sites gone quiet. That table backs `GET /api/v1/sites`
(issue #698), with operator names merged on from the system config.

Roaming, then, is not a subsystem — it is what falls out when the
[cchunt supervisor]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
hops control channels over a session. The candidate list is the system's
configured `control_channels`; `System.HuntOrder` moves the last-locked
frequency (persisted in the hunt cache) to the front, so a daemon restart
or a brief fade goes straight back to the site that was working. When a
locked CC goes silent, the decoder publishes `cc.lost` and the supervisor
re-hunts —
[Trunking Engine Part 12]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }})
covers the watchdog and backoff, and
[The Hunt Part 7]({{ '/blog/deep-dives/the-hunt-07-locking-a-p25-system/' | relative_url }})
what "locked" demands on P25. Each site the hunter lands on contributes its
broadcasts and the roster grows — the fold *is* the roam. (Watching several
sites *simultaneously* instead of serially is Part 11's subject.)

One instrument earns special mention because it catches the multi-site
failure mode nothing else sees: each `SiteInfo` row carries
`ControlChannelCarrierOffsetHz`, the demod's measured carrier offset of the
locked CC against the tuned centre. A lock sitting well off the configured
frequency is the signature of an **adjacent site bleeding through** — you
are decoding a different tower than you tuned (issue #815, and the
[carrier-offset-adjacent-lock]({{ '/reference/carrier-offset-adjacent-lock/' | relative_url }})
reference). The companion WARN requires the offset to *persist* before
firing, because a per-chunk estimator blip is not a site change.

### How the fragment problem shaped the Go code

- **Accumulate in the decoder, snapshot at the boundary.** Identity is not
  on any per-event payload — `TopologySnapshot` exists precisely because a
  consumer cannot reconstruct WACN/SysID/RFSS/Site from the event stream.
- **Distinguish absent from zero, everywhere.** Zero-vote guards, `CFVAKnown`,
  and the 0x2B accepted-only rule all encode the same idea: a field you
  never saw is not a field that equals zero.
- **Two frame forms, one fold.** The AMBT `Apply*` methods normalise into
  the TSBK structures and share the merge — so a fix to the accumulator
  lands on both forms, the standing
  [twin-path rule]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
  applied at the message layer.

## Where this goes next

Hunting visits sites one at a time.
[Part 11]({{ '/blog/deep-dives/p25-end-to-end-11-wideband/' | relative_url }})
removes the "one at a time": pin one SDR across a band and channelize every
control channel out of one capture — the `DDCBank` tuner bank, voice taps
that follow grants without retuning, and the twin down-converter paths
whose drift taught this project its most expensive lesson.

## FAQ

**What is a WACN and do I need to configure it?**
The Wide Area Communication Network ID — a 20-bit number identifying the
licensee's whole P25 network, above the System ID. You don't configure it;
GopherTrunk decodes it from Network Status broadcasts (TSBK or AMBT form)
and reports it in the site table and network report — how you confirm two
configured systems are actually the same network.

**Why did GopherTrunk show one neighbour site when SDRtrunk showed twelve?**
Because those neighbours were broadcast only in the multi-block AMBT form,
which early GopherTrunk logged as `non-control DUID duid=PDU` and dropped.
Since the `mbt.go` decoder landed, both forms feed one accumulator — and
the AMBT form contributes fields the TSBK form doesn't carry, like the
neighbour's explicit uplink channel.

**Does GopherTrunk automatically roam between P25 sites?**
Effectively, yes — one site at a time. The hunter locks the best candidate
from your `control_channels` list, biased toward the last-known-good
frequency, re-hunts on `cc.lost`, and the tracker accumulates every site it
visits. It does not (yet) auto-add decoded neighbour frequencies to the
candidate list; the network report tells you which are worth adding.

**Is the NAC the same thing as the System ID?**
No. The NAC is a 12-bit access code in every frame's NID — a colour code
that gates frame acceptance. The System ID is a 12-bit network identity
carried in status broadcasts. Many systems set memorable NACs per site,
but nothing requires the two to relate.

**My site shows RFSS 0 — is that a decode failure?**
No. RFSS 0 is a real, common value, which is why the accumulator counts
RFSS/Site votes even at zero, unlike WACN/System ID where zero means
"absent." A genuinely undecoded identity shows as no row at all, or blank
WACN/SysID awaiting status broadcasts.

## Series navigation

**Part 10 of 14** · ←
[Part 9: Encryption Signalling — Flags, Metadata & Policy]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }})
· Next →
[Part 11: Wideband P25 — Watching the Whole System at Once]({{ '/blog/deep-dives/p25-end-to-end-11-wideband/' | relative_url }})
