---
title: "The Hunt, Part 7: Locking a P25 System — Candidate to Confirmed"
description: How GopherTrunk turns a locked P25 control channel into a confirmed DiscoveredSystem — folding WACN and System ID into an identity, recovering the IDEN_UP band plan, resolving neighbour frequencies, and honestly reporting when the NSB that carries the WACN never decoded.
category: deep-dives
keywords: p25 system identity, wacn system id nac, discovered system, iden_up band plan, network status broadcast, neighbour frequency resolution, identity note, candidate to confirmed, gophertrunk the hunt
tags: [the-hunt, p25, trunking, identity, band-plan, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 7
---

*Part 7 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. [Part 6]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
got a control channel *locked* — the decoder is reading TSBKs on the frequency under
our 851 MHz carrier. A lock is not yet a system, though. This part is the promotion
from candidate to confirmed: accumulating the decoded identity, band plan, sites, and
talkgroups into one `DiscoveredSystem` — and, crucially, being honest when a piece of
that identity simply never came over the air.*

> **TL;DR:** A locked control channel is a *stream of observations*, not a finished
> map. `Accumulate` folds each observation into a single `DiscoveredSystem`:
> identity (WACN / System ID / NAC), per-site control channels, neighbours, band
> plan, and talkgroups, all de-duplicated. The **band plan** (`IDEN_UP` slots) is
> what turns a neighbour's `(channel id, number)` into a real frequency — resolved
> at finish, after the plan is fully accumulated. And when a P25 system **locks but
> never broadcasts the Network Status Broadcast** that carries its WACN, GopherTrunk
> doesn't fabricate one — it emits an `IdentityNote` explaining exactly which
> identity messages did and didn't decode.

**Key takeaways**

- **A system is accumulated, not captured.** No single dwell sees the whole system;
  `DiscoveredSystem` is built incrementally and de-duplicated as observations fold in.
- **The band plan is the Rosetta Stone.** Neighbours and grants arrive as
  `(channel id, number)` pairs; only the accumulated `IDEN_UP` band plan turns those
  into downlink frequencies — so resolution happens at finish, when the plan is complete.
- **Talkgroups have no fixed frequency.** A trunked traffic channel is assigned per
  call, so a grant's frequency is attributed to the *site*, never to the talkgroup.
- **Missing identity is reported, not invented.** The `IdentityNote` names which P25
  identity broadcasts decoded, so "WACN unknown" comes with the reason — the system
  never emitted the NSB, not a decode bug.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The system model | identity, sites, band plan, talkgroups | `internal/hunt/system.go` (`DiscoveredSystem`) |
| Frequency resolution | (channel id, number) → downlink Hz | `internal/hunt/system.go` (`bandPlanFreq`) |
| Neighbour backfill | resolve neighbour CCs at finish | `internal/hunt/system.go` (`resolveNeighborFreqs`) |
| Decode + fold | identify, decode, accumulate one capture | `internal/hunt/decode.go` (`decodeAndAccumulate`) |
| Identity honesty | explain a blank WACN/System ID | `internal/hunt/decode.go` (`identityNote`) |
| Report flatten | system → protocol-neutral report | `internal/hunt/report.go` (`NetworkReport`) |

## In this post

- **From lock to observation** — what a decode actually hands the accumulator.
- **The DiscoveredSystem** — the incremental, de-duplicated model.
- **The band plan** — why neighbour frequencies resolve only at finish.
- **The identity-note case** — being honest about a WACN that never aired.
- **Flattening to a report** — the shape the exporters consume.

## From lock to observation

The shared body that decodes one capture and folds it in is `decodeAndAccumulate` —
the same function the offline `Discover`, the live hunter, and the auto-gain sweep all
call. Once it has a decoded `siglab.Result`, it folds it into the system and fills a
report:

```go
// internal/hunt/decode.go (shape) — decodeAndAccumulate fold
before := len(sys.Talkgroups)
Accumulate(sys, Observation{
    Protocol:       proto.String(),
    Confidence:     conf,
    Result:         res,
    FallbackFreqHz: p.FrequencyHz,
    At:             time.Now(),
})
rep.Locked = res.Locked
if res.Lock != nil { rep.ControlHz = res.Lock.FrequencyHz }
if res.Signal != nil { rep.ErrorRate = res.Signal.DecodeErrorRate }
rep.IdentityNote = identityNote(res)
rep.Encrypted, rep.EncType = encryptionFromGrants(proto.String(), res.Grants)
rep.Talkgroups = len(sys.Talkgroups) - before
```

The decode deliberately runs the *deep* P25 path (`CollectIQDiag: true`), because
that's what snapshots the system topology — WACN, System ID, RFSS, Site, neighbours,
band plan — onto `Result.Topology`. Without it, P25 runs the generic pipeline and the
map would be NAC-only. And note the decode tunes to the carrier *identify locked*
(`idr.TuneHz`), not a fresh dominant-carrier estimate: under auto-tune the control
channel may be off-centre and not the band's loudest, so re-estimating would find the
wrong carrier and fail a decode that identify already proved works.

## The DiscoveredSystem

The accumulator's target is `DiscoveredSystem`: a protocol-neutral model built up
incrementally and shaped to round-trip cleanly back into GopherTrunk's import bundle.

```go
// internal/hunt/system.go (shape) — DiscoveredSystem
type DiscoveredSystem struct {
    Name     string
    Protocol string
    WACN     uint32 // P25 / generic identity — zero ⇒ unknown
    SystemID uint16
    NAC      uint16
    Identity map[string]any // per-protocol extras (DMR ColorCode, TETRA MCC/MNC, …)
    Sites      []DiscoveredSite
    Talkgroups []DiscoveredTalkgroup
    BandPlan   []BandPlanEntry
    Confidence float64
    // …State/County/Location, FirstSeen/LastSeen
}
```

Every mutation is de-duplicating by design. `addControlChannel` keeps one entry per
frequency, upgrading confidence. `addNeighbor` keeps the first identity for a
neighbour's `(RFSS, Site)` but backfills a frequency once a later observation
resolves one. `addBandPlanEntry` keeps the first observation of a channel ID. This
matters because the same site is seen across many dwells, and a naive append would
produce a map full of duplicates. The one deliberate *non*-attribution is worth
calling out:

```go
// internal/hunt/system.go (shape) — addTalkgroup
// The voice-channel frequency a grant landed on is NOT attributed to the
// talkgroup: on a trunked system the traffic channel is assigned dynamically per
// call, so a talkgroup has no fixed frequency. The distinct voice frequencies the
// site uses are recorded on the site (addVoiceChannel).
```

That's the essence of trunking captured in a data-model decision: a talkgroup is a
*logical* group, not a channel. Attributing the grant frequency to the talkgroup
would encode a lie about how the system works. Instead the grant's resolved
frequency is recorded on the *site* as a voice channel.

## The band plan is the Rosetta Stone

Here is the crux of confirming a P25 system. Neighbours and grants don't arrive as
frequencies — they arrive as `(channel id, channel number)` pairs, and turning those
into a real downlink frequency requires the system's band plan (its `IDEN_UP`
broadcasts). A single dwell may not have seen the band-plan entry for a channel ID
before it saw a neighbour using it, so resolution can't happen eagerly — it happens
at *finish*, once the plan is fully accumulated:

```go
// internal/hunt/system.go (shape) — bandPlanFreq + finish-time resolution
func (s *DiscoveredSystem) bandPlanFreq(channelID uint8, channelNumber uint16) (uint32, bool) {
    for _, e := range s.BandPlan {
        if e.ChannelID != channelID { continue }
        hz := e.BaseHz + uint64(channelNumber)*uint64(e.SpacingHz) // P25 IDEN_UP math
        if hz == 0 || hz > 0xFFFFFFFF { return 0, false }
        return uint32(hz), true
    }
    return 0, false
}

// resolveNeighborFreqs runs from sortAll() on every finish/export — idempotent, so a
// neighbour already carrying a frequency is left untouched.
```

`sortAll` puts everything in deterministic order (stable golden output) *and* calls
`resolveNeighborFreqs`, so a neighbour advertised early by a system whose band plan
arrived late still gets its frequency once both are in hand. It deliberately applies
`BaseHz + number*spacing` and *not* the transmit offset — that offset is the uplink;
a control-channel reference is the downlink. Getting that direction wrong would put
every neighbour's frequency on the wrong side of the duplex split.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="A control channel decode produces three streams of observations: identity broadcasts giving WACN and System ID, IDEN_UP band-plan entries, and neighbour and grant references carrying channel id and number pairs; the band plan resolves those pairs into downlink frequencies at finish, and everything folds into one discovered system">
  <rect x="20" y="80" width="120" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="80" y="100" text-anchor="middle" fill="var(--accent)" font-size="10">locked CC</text>
  <text x="80" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TSBK stream</text>
  <line x1="140" y1="86" x2="188" y2="52" stroke="currentColor"/><polygon points="184,49 194,48 188,57" fill="currentColor"/>
  <line x1="140" y1="101" x2="188" y2="101" stroke="currentColor"/><polygon points="188,97 198,101 188,105" fill="currentColor"/>
  <line x1="140" y1="116" x2="188" y2="150" stroke="currentColor"/><polygon points="184,145 194,153 183,154" fill="currentColor"/>
  <rect x="198" y="34" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="273" y="50" text-anchor="middle" fill="currentColor" font-size="10">identity (NSB/RFSS)</text>
  <text x="273" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="9">WACN · System ID</text>
  <rect x="198" y="83" width="150" height="36" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="273" y="99" text-anchor="middle" fill="var(--accent)" font-size="10">band plan (IDEN_UP)</text>
  <text x="273" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="9">base · spacing per ch id</text>
  <rect x="198" y="132" width="150" height="36" rx="6" fill="none" stroke="currentColor"/>
  <text x="273" y="148" text-anchor="middle" fill="currentColor" font-size="10">neighbours / grants</text>
  <text x="273" y="161" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(ch id, number)</text>
  <line x1="348" y1="101" x2="348" y2="150" stroke="var(--accent)" stroke-dasharray="3 3"/><polygon points="344,146 348,156 352,146" fill="var(--accent)"/>
  <text x="360" y="140" fill="var(--accent)" font-size="9">resolves</text>
  <line x1="348" y1="101" x2="430" y2="101" stroke="currentColor"/><polygon points="430,97 440,101 430,105" fill="currentColor"/>
  <rect x="440" y="80" width="150" height="42" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="515" y="100" text-anchor="middle" fill="var(--accent)" font-size="10">DiscoveredSystem</text>
  <text x="515" y="114" text-anchor="middle" fill="var(--fg-muted)" font-size="9">confirmed + mapped</text>
  <text x="340" y="190" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the band plan is the Rosetta Stone: (channel id, number) → downlink Hz, resolved at finish once the plan is complete</text>
</svg>
<figcaption>Confirming a P25 system is a join. Identity, band plan, and channel references arrive as separate streams; the band plan resolves the references into frequencies as everything folds into one system.</figcaption>
</figure>

### How that principle shaped the Go code

- **Resolution is deferred and idempotent.** `resolveNeighborFreqs` runs from
  `sortAll` on every finish/export and skips neighbours that already have a
  frequency, so it's safe to call repeatedly and never depends on observation order.
- **Sites are keyed, not appended.** `site(rfss, siteID)` returns a pointer to an
  existing site or creates one, so many dwells of the same RF site converge on one
  `DiscoveredSite` rather than duplicating it.
- **The model is export-shaped.** `DiscoveredSystem`'s fields convert cleanly onto
  the importer's parsed system, so a discovery exported as a bundle re-imports
  without loss — the accumulator and the exporter agree on the shape.

## The identity-note case

Now the honest part, and the reason this series has an issue-closing policy about
not claiming things you can't verify. A P25 system can lock cleanly, decode
thousands of TSBKs, and *still* have a blank WACN — because on P25 Phase 1 the
**Network Status Broadcast** (NSB, opcode `0x3B`) is the *only* message that carries
the WACN, and some systems rarely or never transmit it. GopherTrunk refuses to
either fabricate a WACN or silently show it blank; instead `identityNote` explains
precisely what happened:

```go
// internal/hunt/decode.go (shape) — identityNote
func identityNote(res *siglab.Result) string {
    if res == nil || !res.Locked || res.Topology == nil { return "" }
    if res.Topology.WACN != 0 && res.Topology.SystemID != 0 { return "" } // resolved
    d, ok := res.Detail.(*siglab.P25P1Detail)
    if !ok || d.CCStats == nil { return "" } // not the P25 deep path
    cc := d.CCStats
    if res.Topology.SystemID != 0 && res.Topology.WACN == 0 {
        // System ID voted from adjacent-site broadcasts, but the NSB that carries
        // WACN never decoded — widening the dwell won't help if it never airs.
        return fmt.Sprintf("System ID resolved from adjacent-site broadcasts, but WACN is "+
            "unavailable: the Network Status Broadcast (NSB, 0x3B) that carries it was never "+
            "decoded (saw NSB×%d, RFSS×%d, adjacent×%d, …)", cc.NetStatusSeen, cc.RFSSStatusSeen, cc.AdjacentSeen)
    }
    // …NSB decoded but parsed zero ⇒ a parse problem, not a capture gap
    // …no NSB at all ⇒ widen --dwell-seconds or --monitor-seconds to catch the periodic NSB
}
```

The note distinguishes three genuinely different situations, and the distinction is
actionable. If the System ID came from adjacent-site broadcasts but the NSB never
aired, widening the dwell is pointless — the WACN simply isn't recoverable from this
system. If the NSB decoded but parsed to zero, that's a parse bug, not a capture gap.
If no NSB decoded at all, a longer `--monitor-seconds` might catch the periodic one.
The `identity_note_test.go` table pins every branch: "locked but NSB never decoded"
expects the "Network Status Broadcast" text; "System ID resolved from adjacent, WACN
still missing" expects "WACN is unavailable"; "NSB decoded but no WACN parsed"
expects "parsed as zero". This is the same discipline as the issue-closing policy —
say what you *know*, name what's blocking, don't claim a resolution you can't back.

## Flattening to a report

When it's time to render or export, `NetworkReport` flattens the multi-site
`DiscoveredSystem` into the protocol-neutral `trunking.NetworkReport` the shared
renderer consumes — one `ReportSite` per site under a shared identity header and band
plan. Site control channels are stored as *resolved* downlink frequencies (the band
plan is already applied), while neighbours keep their `(id, number)` and get an
uplink derived from the matching band's transmit offset — the one place the offset
*is* applied, because a neighbour reference legitimately carries both directions. The
report is the boundary between "what the hunt found" and "what gets written to a
RadioReference submission or a scanner import."

## Where this goes next

We've confirmed a single P25 system from one control channel. But real systems have
neighbours, and a wide receiver can watch several of their channels at once.
[Part 8]({{ '/blog/deep-dives/the-hunt-08-dmr-lcn-correlation/' | relative_url }})
turns to DMR Tier III, where the band plan isn't broadcast at all — it has to be
*learned* by correlating logical channel numbers to the physical carriers that key up
in response. Then Part 9 returns to P25 to watch many channels across a wideband tune.

## FAQ

**What's the difference between a locked control channel and a confirmed system?**
A lock means the decoder is reading the control channel. A confirmed system is the
accumulated result of that decoding: identity (WACN/System ID/NAC), the band plan,
the sites and their control channels, the neighbours (with resolved frequencies), and
the talkgroups — folded and de-duplicated into one `DiscoveredSystem`.

**Why can a P25 system lock but have no WACN?**
Because on P25 Phase 1 only the Network Status Broadcast (NSB, opcode `0x3B`) carries
the WACN, and some systems transmit it rarely or never. A lock decodes TSBKs and
grants fine without ever seeing an NSB, so the WACN stays blank — and GopherTrunk
emits an `IdentityNote` saying so rather than inventing a value.

**Why resolve neighbour frequencies at finish instead of when they arrive?**
Because a neighbour can be advertised before the band-plan entry that resolves its
`(channel id, number)` has decoded. Deferring resolution to `sortAll` — after the
whole band plan is accumulated — means every resolvable neighbour gets a frequency
regardless of the order observations arrived. The resolution is idempotent, so it's
safe to run on every export.

**Why isn't a grant's frequency attributed to its talkgroup?**
Because trunking assigns traffic channels dynamically per call — a talkgroup has no
fixed frequency. Recording the grant frequency on the talkgroup would misrepresent
how the system works. The frequency is attributed to the *site* as a voice channel,
where it correctly describes which channels the site uses.

**How is this the same engine offline and live?**
`decodeAndAccumulate` is shared: offline `Discover` feeds it captures, the live
hunter feeds it captured buffers, and the streaming monitor feeds it a live stream —
all folding into the same `DiscoveredSystem` via the same `Accumulate`. What you
confirm from a recording is what you'd confirm on the air.

## Series navigation

**Part 7 of 14** · ←
[Part 6: Control-Channel Hunting — The Supervisor]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
· Next →
[Part 8: DMR LCN Correlation — Rebuilding a Channel Map]({{ '/blog/deep-dives/the-hunt-08-dmr-lcn-correlation/' | relative_url }})
