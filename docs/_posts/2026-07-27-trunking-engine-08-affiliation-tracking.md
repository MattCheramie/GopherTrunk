---
title: "Trunking Engine, Part 8: Affiliation Tracking — Building the Roster From the Event Stream"
description: How GopherTrunk builds a live "which radio is on which talkgroup" roster entirely from the trunking event stream — grants, affiliation responses, and unit registrations — with no extra radio I/O.
category: deep-dives
keywords: p25 affiliation tracking, unit registration, talkgroup roster, radio id tracking, group affiliation response, derived state, materialized view, event sourcing, gophertrunk trunking, rid tracking
tags: [trunking, go, event-bus, p25, affiliation, architecture]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 8
---

*Part 8 of **Trunking Engine**, a 12-part deep dive into the "brain" of
GopherTrunk. Parts 1–7 followed a single grant from the control channel into a
recorded call. This one steps sideways: while the engine is busy tuning radios,
a second subscriber is quietly reading the same event stream and building a
roster of every unit on the air — without touching a single SDR.*

> **TL;DR:** The `AffiliationTracker` is a subscriber, not a decoder. It watches
> the bus for `KindGrant`, `KindAffiliation`, and `KindUnitRegistration` events
> and folds them into a live table of which radio ID is active on which
> talkgroup. Because grants already carry `SourceID` + `GroupID` for *every*
> protocol, the roster works uniformly across P25, DMR, NXDN and the rest with no
> per-protocol code. The table is **derived state** — a materialized view of the
> event log — so it costs no extra radio I/O and can be rebuilt from the stream
> alone.

**Key takeaways**

- The roster is **derived, not measured**: every row comes from events the engine
  and decoders already publish. Adding the tracker added zero RF work.
- Three event kinds feed it — a **grant** (the source is transmitting on a TG), an
  explicit **affiliation** response, and a **unit registration** — plus talker
  aliases and in-call source backfills as refinements.
- It is **protocol-agnostic** because it keys on the protocol-neutral `SourceID`
  and `GroupID` fields, not on any wire format.
- It is a textbook **observer**: like the recorder and the call log, it subscribes
  to the engine's output and never calls back in.

## Cheat sheet

| Concept | Where it lives | One-line role |
|---|---|---|
| `AffiliationTracker` | `affiliation_tracker.go` | subscriber that folds events into a live unit table |
| `UnitActivity` | `affiliation_tracker.go` | one row: radio ID → talkgroup, first/last seen, call count |
| `Affiliation` | `affiliation.go` | `KindAffiliation` payload; P25 opcode 0x28 Group Affiliation Response |
| `UnitRegistration` | `affiliation.go` | `KindUnitRegistration` payload; P25 opcode 0x2C Unit Registration Response |
| `observe(...)` | `affiliation_tracker.go` | the single fold function every event kind routes through |
| `Snapshot()` / `UnitsOnTalkgroup()` | `affiliation_tracker.go` | read the materialized view; power `/rids` and the RID list |

## In this post

- **What "affiliation" means** on a trunked system, and why a grant already tells
  you most of it.
- **The three (plus two) event sources** the tracker folds, and how it stays
  protocol-agnostic.
- **Derived state** — the design principle — and how a materialized view keeps the
  hot path clean.
- **Explicit vs observed** associations, TTL expiry, and reading the snapshot.

## What affiliation is

On a trunked system a radio doesn't just start transmitting. Before it can use a
talkgroup it usually **affiliates** with it — tells the control channel "unit
0x4A21 wants to work talkgroup 1234" — and when it powers on it **registers** on
the site. Those are explicit control-channel transactions, and P25 answers them
with a *Group Affiliation Response* (opcode 0x28) and a *Unit Registration
Response* (opcode 0x2C). Decode those and you know exactly who is on what.

But there is a second, cruder signal that is just as much ground truth: **a
voice grant names its source.** When the control channel announces "talkgroup
1234 is up on 851.0125 MHz, source 0x4A21," that grant *proves* unit 0x4A21 is
transmitting on talkgroup 1234 right now. You don't need the affiliation message
at all — the grant carries `SourceID` and `GroupID`, and it does so on every
protocol GopherTrunk decodes. That is the observation that makes a single, tiny
tracker work across P25, DMR, NXDN, dPMR and the rest without a line of
protocol-specific code.

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="Three event kinds — grant, affiliation response, and unit registration — flow from the bus into the affiliation tracker, which folds them into a unit-to-talkgroup roster table">
  <rect x="8" y="18" width="150" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="83" y="37" text-anchor="middle" fill="currentColor" font-size="11">KindGrant (src+TG)</text>
  <rect x="8" y="58" width="150" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="83" y="77" text-anchor="middle" fill="currentColor" font-size="11">KindAffiliation</text>
  <rect x="8" y="98" width="150" height="30" rx="6" fill="none" stroke="currentColor"/>
  <text x="83" y="117" text-anchor="middle" fill="currentColor" font-size="11">KindUnitRegistration</text>
  <g stroke="var(--fg-muted)">
    <line x1="158" y1="33" x2="238" y2="66"/><polygon points="236,62 246,68 235,71" fill="var(--fg-muted)"/>
    <line x1="158" y1="73" x2="238" y2="73"/><polygon points="238,69 248,73 238,77" fill="var(--fg-muted)"/>
    <line x1="158" y1="113" x2="238" y2="80"/><polygon points="236,76 246,79 236,84" fill="var(--fg-muted)"/>
  </g>
  <rect x="248" y="50" width="150" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="323" y="70" text-anchor="middle" fill="var(--accent)" font-size="12">AffiliationTracker</text>
  <text x="323" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="10">observe() folds each event</text>
  <line x1="398" y1="73" x2="450" y2="73" stroke="currentColor"/>
  <polygon points="450,69 460,73 450,77" fill="currentColor"/>
  <rect x="460" y="30" width="212" height="88" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="566" y="47" text-anchor="middle" fill="currentColor" font-size="11">units map[RadioID]</text>
  <text x="566" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0x4A21 → tg 1234 · calls 7</text>
  <text x="566" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0x4A22 → tg 1234 · registered</text>
  <text x="566" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0x51F0 → tg 9001 · alias "M-14"</text>
  <text x="323" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="10">no SDR, no retune, no radio I/O — the tracker only ever reads the bus</text>
  <text x="323" y="168" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the roster is a fold over events the engine already publishes</text>
</svg>
<figcaption>The tracker subscribes to the same bus the engine does. Every event kind routes through one <code>observe()</code> fold that upserts a row keyed by radio ID.</figcaption>
</figure>

## The three (plus two) sources

The tracker's `Run` loop is the same shape as the engine's: a `select` over the
subscription channel and a ticker. Every event is handed to one `handle`
switch, and every branch collapses to a single `observe` call:

```go
// internal/trunking/affiliation_tracker.go (shape)
func (t *AffiliationTracker) handle(ev events.Event) {
    switch ev.Kind {
    case events.KindGrant:
        if g, ok := ev.Payload.(Grant); ok && g.SourceID != 0 {
            t.observe(g.SourceID, g.GroupID, g.System, g.Protocol, false, false, true)
        }
    case events.KindAffiliation:
        if a, ok := ev.Payload.(Affiliation); ok && a.SourceID != 0 &&
            a.Response == AffiliationAccepted {
            t.observe(a.SourceID, a.GroupID, a.System, a.Protocol, true, false, false)
        }
    case events.KindUnitRegistration:
        if r, ok := ev.Payload.(UnitRegistration); ok && r.SourceID != 0 &&
            r.Response == RegistrationAccepted {
            t.observe(r.SourceID, 0, r.System, r.Protocol, false, true, false)
        }
    // ... KindTalkerAlias and KindCallSourceUpdate refine existing rows
    }
}
```

Read the three primary branches as a hierarchy of evidence. A **grant** is
implicit but certain — a radio is keyed *right now* — so it counts a call
(`countCall=true`) but marks the association non-explicit. An **affiliation
response** is the explicit, decoded "I want this talkgroup," so it sets
`Explicit=true`. A **unit registration** proves the radio is on the site but names
no talkgroup, so it passes `talkgroup=0`, which `observe` treats as "refresh this
unit without clobbering a known talkgroup association." Only accepted responses
count — a `denied` or `refused` affiliation is not membership.

Two more event kinds *refine* rows rather than create the "who's on what"
association. `KindTalkerAlias` stamps a unit's over-the-air display name
(reassembled from P25 Phase 2 vendor MAC PDUs), and `KindCallSourceUpdate` — the
in-call source-RID backfill from [Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
— records the real source when the original grant fired with `src=0`. Both keep
the unit alive in the table and fill in identity without inventing a new call.

## Explicit vs observed, and why both are truth

The `UnitActivity` row carries an `Explicit` flag, and it is worth dwelling on
because it is a subtle distinction that a lot of scanners get wrong. Both an
affiliation message and a grant are *ground truth* about membership — a grant
naming a source as the transmitter is arguably stronger evidence than a stale
affiliation from twenty minutes ago. The flag doesn't rank confidence; it records
*provenance*, so a UI can show "affiliated" versus "seen transmitting" without
pretending one is a guess:

```go
// internal/trunking/affiliation_tracker.go (shape)
type UnitActivity struct {
    RadioID    uint32
    Talkgroup  uint32
    Explicit   bool // from a decoded affiliation, not just a grant
    Registered bool // seen in a unit-registration message
    FirstSeen  time.Time
    LastSeen   time.Time
    CallCount  uint64 // grants observed since the unit entered the table
    TalkerAlias           string // over-the-air display name
    TalkerAliasUnreliable bool   // CRC-passed but held non-printable bytes (#711)
}
```

`CallCount` is the "recurring radio" signal from issue #376 — a simple count of
grants seen for this unit since it entered the table, so the RID list can sort by
who is actually busy. `TalkerAliasUnreliable` is the corruption guard from #711:
an alias that passed CRC but decoded to non-ASCII-printable bytes is surfaced as
suspect rather than as a confirmed name.

### How that principle shaped the Go code — derived state

The design idea here is **derived (materialized) state**: the roster is not a
thing the system *has*, it is a thing the system can *recompute* from its event
log. Nothing in the decode path knows the tracker exists. The engine tunes
radios; the decoders emit grants and affiliation responses; the tracker is a pure
downstream fold. That buys three concrete properties:

- **Zero extra RF cost.** The roster adds no tune, no dwell, no second receiver.
  Every input already existed on the bus for the recorder and the call log. The
  tracker is, quite literally, free intelligence.
- **Reconstructable.** Because state is a fold over events, the same stream
  replays to the same table. A test publishes a handful of synthetic grants and
  asserts `Snapshot()` — no radio, exactly the harness [Part 12]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }})
  builds out.
- **Bounded without coordination.** A materialized view can be *lossy on
  purpose*. Entries expire after an idle TTL (30 minutes by default) via a
  once-a-minute `sweep`, so the table tracks the live population instead of
  growing forever. Nothing upstream has to tell it a radio left — absence of
  events *is* the signal.

The concurrency story is the same one-writer discipline the engine uses. The
tracker's `Run` goroutine is the only mutator; `Snapshot`, `UnitsOnTalkgroup`,
and `Len` take the mutex only to copy out a consistent view for read-only API
callers.

<figure class="lab-figure">
<svg viewBox="0 0 680 170" width="680" height="170" role="img" aria-label="An append-only event log is folded left into a materialized unit table, and an idle sweep drops rows older than the TTL, keeping the derived view bounded">
  <text x="90" y="24" text-anchor="middle" fill="var(--fg-muted)" font-size="11">event log (append-only)</text>
  <g stroke="currentColor" fill="none">
    <rect x="20" y="34" width="60" height="26" rx="4"/><rect x="88" y="34" width="60" height="26" rx="4"/>
    <rect x="156" y="34" width="60" height="26" rx="4"/><rect x="224" y="34" width="60" height="26" rx="4"/>
  </g>
  <text x="50" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grant</text>
  <text x="118" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="9">affil</text>
  <text x="186" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="9">reg</text>
  <text x="254" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grant</text>
  <line x1="292" y1="47" x2="340" y2="47" stroke="var(--accent)"/>
  <polygon points="340,43 350,47 340,51" fill="var(--accent)"/>
  <text x="322" y="34" text-anchor="middle" fill="var(--accent)" font-size="10">observe()</text>
  <rect x="352" y="24" width="150" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="427" y="44" text-anchor="middle" fill="var(--accent)" font-size="12">materialized</text>
  <text x="427" y="60" text-anchor="middle" fill="var(--accent)" font-size="12">unit table</text>
  <line x1="502" y1="50" x2="550" y2="50" stroke="currentColor"/>
  <polygon points="550,46 560,50 550,54" fill="currentColor"/>
  <rect x="560" y="26" width="110" height="48" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="615" y="46" text-anchor="middle" fill="currentColor" font-size="11">Snapshot()</text>
  <text x="615" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">/rids · RID list</text>
  <line x1="427" y1="76" x2="427" y2="110" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <polygon points="423,110 427,120 431,110" fill="var(--fg-muted)"/>
  <rect x="330" y="120" width="195" height="26" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="427" y="137" text-anchor="middle" fill="var(--fg-muted)" font-size="10">sweep() drops rows idle &gt; TTL</text>
  <text x="340" y="163" text-anchor="middle" fill="var(--fg-muted)" font-size="10">absence of events is the "unit left" signal — no coordination needed</text>
</svg>
<figcaption>The roster is a left-fold of the event log; a periodic idle sweep keeps the derived view bounded to the live radio population.</figcaption>
</figure>

## Reading the roster

Two query methods cover the surfaces that use the tracker. `Snapshot()` returns
every unit most-recently-seen first — the data behind the `/rids` RID list.
`UnitsOnTalkgroup(tg)` inverts the index to answer "who is on this talkgroup right
now," which is what lets a UI expand a live call into its participants. Both
return copies under the mutex, so a slow API consumer never blocks the fold. The
P25-specific serving-site fields on `Affiliation` and `UnitRegistration`
(`RFSSID`, `SiteID`, `NAC`) go further — because an affiliation is handled by the
radio's *actual* serving site rather than every site in a wide-area call, they're
a genuine RID→site location fix, which is where multi-site roaming in
[Part 10]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
picks up. For the protocol background on what an affiliation transaction is, see
the [affiliation reference]({{ '/reference/affiliation/' | relative_url }}).

## Where this goes next

[Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }})
stays in "derived from the stream" territory but tackles a harder case: patches
and supergroups, where one physical call belongs to *several* talkgroups at once,
and the physical-channel RID recovery that suppresses a phantom duplicate call
when a compressed grant arrives under a mis-aliased talkgroup. After that,
[Part 10]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
uses the serving-site fields introduced here to build a full multi-site network
map.

## FAQ

**Does affiliation tracking need its own radio or a second tuner?**
No. It is a pure subscriber to the event bus. Every input — grants, affiliation
responses, unit registrations — is already published by the engine and the
decoders for other reasons, so the roster costs zero additional RF work.

**How does one tracker work for P25, DMR, and NXDN at once?**
It keys on the protocol-neutral `SourceID` and `GroupID` fields that every
grant carries, not on any wire format. Explicit affiliation and registration
messages are folded when available, but the grant path alone gives a working
roster on every protocol.

**What's the difference between an "explicit" and an "observed" association?**
An explicit association came from a decoded Group Affiliation Response; an
observed one was inferred from a voice grant that named the unit as the source.
Both are ground truth — the flag records provenance, not confidence — and a grant
is often the stronger evidence.

**Why do units disappear from the roster?**
Entries expire after a configurable idle TTL (30 minutes by default), swept once
a minute. The table is a materialized view of *recent* activity, not a permanent
log, so a radio that goes quiet ages out — absence of events is the signal that
it left.

## Series navigation

**Part 8 of 12** · ←
[Part 7: Source RID Recovery]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
· Next →
[Part 9: Patches & Supergroups]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }})
