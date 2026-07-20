---
title: "Trunking Engine, Part 7: Recovering the Source Radio ID (Issue #915)"
description: A current-work deep dive into how GopherTrunk backfills the source radio ID onto calls that bound from a source-less P25 Phase 2 grant, republishing it through the source-update path so nearly every call reports who keyed up.
category: deep-dives
keywords: source radio id, p25 rid recovery, issue 915, source-less grant backfill, call.source update, p25 phase 2 compressed grant, group voice channel user, gophertrunk source rid, republish source update
tags: [trunking, go, p25, rid, source-recovery, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 7
---

*Part 7 of **Trunking Engine**, and the first **current-work** post in the series.
[Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }}) flagged
that P25 Phase 2 grants often arrive with `SourceID == 0`; this post is the fix.
It's the story of issue #915 — why only a small fraction of calls carried the
radio ID of whoever keyed up, and how the engine now backfills it onto the running
call and republishes it so downstream consumers see the source.*

> **TL;DR:** The source radio ID (RID) is the "who" of a call. On a busy P25
> Phase 2 system a call frequently **binds from a source-less grant** — a
> compressed grant or a `GRP_VCH_UPDATE` repeat with no `SOURCE_ID` — while the
> initiating `GRP_VCH_GRANT`'s RID arrives on a *later* grant. That RID reached
> the engine but was never associated back to the running call, so the
> completed-call webhook reported a source on only **~18%** of calls. The engine
> now folds a later same-call grant's RID onto the bound call
> (`BackfillSourceFromGrant`) and **republishes it through the existing
> source-update path**, so the webhook and the live SSE/TUI view report the source
> — on the first fill only, so the repeated grant stream never floods the bus.

**Key takeaways**

- Before the fix, source RID lived on the *binding* grant. When a call bound
  source-less, the RID that arrived later was dropped on the floor.
- `BackfillSourceFromGrant(serial, sourceID)` fills a call's source **only when
  it's still zero**, so an in-call `call.source` update always wins and a later
  grant never clobbers a known RID.
- The fill **republishes a `KindCallSourceUpdate`** — the same event the
  traffic-channel recovery already used — so no downstream consumer needed a new
  code path.
- It fires **once per call**, on the zero→known transition, not once per repeat
  grant — the anti-flood invariant.
- Talkgroup-keyed matching alone reached only **~12%** coverage on a
  heavily-compressed field system, because the RID-bearing grant often arrives
  under a *different* talkgroup — the physical-channel net that closes that gap is
  [Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }}).

## Cheat sheet

| Path | Trigger | Wins over |
|---|---|---|
| `Grant.SourceID` at bind | source-bearing grant starts the call | — (baseline) |
| `BackfillSourceFromGrant` | later same-call **grant** carries the RID (#915) | nothing known yet |
| `UpdateSource` (in-call) | traffic-channel `GROUP_VOICE_CHANNEL_USER` PDU | grant backfill |
| `republishCallSource` | any of the above fills a previously-zero source | — |
| Republish guard | `filled == true` only | fires once per call |

## In this post

- **The RID database** — the per-radio analogue of the talkgroup DB.
- **The problem** — where the source RID was lost, and the ~18% number.
- **The backfill** — `BackfillSourceFromGrant` and its fill-only-when-zero rule.
- **The republish** — reusing the source-update path so consumers just work.
- **Before / after** — what changed on the wire and on the webhook.

## The RID database

The source RID is a 32-bit number; the `RIDDB` is what gives it a name. It is the
per-radio analogue of the talkgroup database from
[Part 6]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }}),
sharing the same loaders and conventions:

```go
// internal/trunking/rid.go (shape)
type RID struct {
    ID          uint32 // the subscriber unit identifier on the wire
    Alias       string // "Medic 7" — operator-facing name (↔ AlphaTag)
    Owner       string // operator / badge assigned to the radio
    Tag, Group  string // role, agency
    Priority    int
    Lockout     bool   // informational: stale / decommissioned radio
    Watch       bool   // surface in a watch-list UI (defaults true)
    Icon        string
}
```

The schema intentionally mirrors `TalkGroup` where it makes sense
(`Alias`↔`AlphaTag`, `Tag`, `Group`, `Priority`, `Icon`) so `RIDDB.LoadCSV` /
`LoadJSON` share the exact conventions of the talkgroup loaders — an operator who
already maintains a talkgroup catalog drops a parallel RID file next to it.
`Lookup(id)` is the same `RWMutex`-guarded map read. But a name is only useful if
there's a source ID to name — and that is exactly what was missing.

## The problem we hit

Every downstream consumer that wants to answer "who transmitted?" reads
`Grant.SourceID`. The completed-call webhook builds its payload from the grant
that the recorder's session was opened with — the grant that *bound* the call.
That's fine when the binding grant carries the source. On P25 Phase 2 it usually
doesn't.

Recall from [Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})
that Phase 2 systems send a *compressed* grant form (an MMR) with no `SOURCE_ID`,
to save control-channel airtime. And the control channel repeats grants: a call
often binds from a compressed grant or a `GRP_VCH_UPDATE` repeat, while the
initiating `GRP_VCH_GRANT` — the one that *does* carry the RID — arrives moments
later. That RID reached the engine on the later grant, matched the active call in
the duplicate-grant guard from
[Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})... and
was then thrown away, because the guard only refreshed `LastHeardAt` and returned.
The call kept the `SourceID == 0` it bound with.

The measured result, from the issue #915 changelog entry: the completed-call
webhook carried a source RID on only **~18%** of calls — the fraction whose
voice-side `call.source` happened to decode on the traffic channel. For the other
~82%, the operator's call log said "who: unknown" even though the RID had crossed
the control channel.

<figure class="lab-figure">
<svg viewBox="0 0 680 186" width="680" height="186" role="img" aria-label="Before the fix: a compressed grant with source zero binds the call, a later grant carrying the real RID matches the active call in the dedup guard but only refreshes the timestamp and the RID is discarded, so the completed call reports source unknown">
  <text x="340" y="16" text-anchor="middle" fill="var(--fg-muted)" font-size="11">before #915 — the RID reaches the engine and is dropped</text>
  <rect x="14" y="30" width="150" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="89" y="49" text-anchor="middle" fill="currentColor" font-size="10">compressed grant</text>
  <text x="89" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="9">src = 0 → binds call</text>
  <line x1="164" y1="52" x2="214" y2="52" stroke="currentColor"/>
  <polygon points="214,48 224,52 214,56" fill="currentColor"/>
  <rect x="226" y="30" width="150" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="301" y="49" text-anchor="middle" fill="var(--accent)" font-size="10">ActiveCall</text>
  <text x="301" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SourceID = 0</text>
  <rect x="14" y="104" width="150" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="89" y="123" text-anchor="middle" fill="currentColor" font-size="10">later GRP_VCH_GRANT</text>
  <text x="89" y="137" text-anchor="middle" fill="var(--fg-muted)" font-size="9">src = 4471 (the RID!)</text>
  <line x1="164" y1="126" x2="222" y2="70" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <polygon points="222,74 231,68 221,66" fill="var(--fg-muted)"/>
  <text x="250" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="9">dedup guard: Touch + return</text>
  <text x="250" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="9">RID discarded</text>
  <line x1="376" y1="52" x2="470" y2="52" stroke="currentColor"/>
  <polygon points="470,48 480,52 470,56" fill="currentColor"/>
  <rect x="482" y="30" width="186" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="575" y="49" text-anchor="middle" fill="currentColor" font-size="10">completed-call webhook</text>
  <text x="575" y="63" text-anchor="middle" fill="var(--fg-muted)" font-size="9">source: unknown (~82% of calls)</text>
</svg>
<figcaption>The RID crossed the control channel on a later grant and matched the active call — but the dedup guard only refreshed the timestamp, so the source was lost.</figcaption>
</figure>

## The backfill

The fix threads a single new call into the duplicate-grant guard's "same channel,
still going" branch — the exact spot where the later RID-bearing grant already
matched the active call:

```go
// internal/trunking/voicepool.go
// Fills the bound call's source from a control-channel grant, but only when
// the call has no source yet. Returns filled=true only on the zero→known
// transition, so the engine republishes once per call, not per repeat grant.
func (p *VoicePool) BackfillSourceFromGrant(serial string, sourceID uint32) (Grant, bool) {
    p.mu.Lock()
    defer p.mu.Unlock()
    ac, ok := p.active[serial]
    if !ok || sourceID == 0 || ac.Grant.SourceID != 0 {
        return Grant{}, false // no call, no RID, or source already known
    }
    ac.Grant.SourceID = sourceID
    return ac.Grant, true
}
```

The three guard conditions encode the whole precedence policy:

- **`sourceID == 0`** — nothing to fill; a source-less repeat is a no-op.
- **`ac.Grant.SourceID != 0`** — the source is already known, so *leave it*. A
  later grant never clobbers a known RID, and — crucially — the in-call
  `UpdateSource` path (the traffic-channel `GROUP_VOICE_CHANNEL_USER` PDU, which
  reflects the radio *actually keyed* on the traffic channel) always wins over
  this control-channel fallback, because if it ran first, `SourceID` is already
  non-zero and the backfill bows out.
- **`!ok`** — the call already ended; drop it.

The engine calls it from inside the dedup guard, right after `Touch`:

```go
// internal/trunking/engine.go (shape) — same-channel repeat branch
e.pool.Touch(ac.Device.Serial, e.now())
if g.SourceID != 0 {
    if upd, filled := e.pool.BackfillSourceFromGrant(ac.Device.Serial, g.SourceID); filled {
        e.republishCallSource(ac.Device.Serial, upd) // fires once, on first fill
    }
}
return
```

`filled` is the anti-flood latch. The control channel repeats the RID-bearing
grant many times over a call; without the fill-only-when-zero rule the engine
would republish on every repeat and drown the bus. Because the first fill flips
`SourceID` non-zero, every subsequent repeat returns `filled == false` and is
silent.

## The republish: reuse the source-update path

The clever part is what the backfill *doesn't* build. There was already a
mechanism for "a source surfaced mid-call" — the `KindCallSourceUpdate` event the
voice composer emits when it recovers the RID from the traffic channel, handled
by `handleCallSourceUpdate`. The #915 fix routes the control-channel-recovered
RID through that *same* event:

```go
// internal/trunking/engine.go
func (e *Engine) republishCallSource(serial string, g Grant) {
    e.bus.Publish(events.Event{
        Kind: events.KindCallSourceUpdate,
        Payload: CallSourceUpdate{
            DeviceSerial: serial,
            System:       g.System,   // System set → engine's own subscription
            Protocol:     g.Protocol, //   short-circuits, no re-entrant loop
            GroupID:      g.GroupID,
            SourceID:     g.SourceID,
            Encrypted:    g.Encrypted,
            At:           e.now(),
        },
    })
}
```

Every consumer that already handled a mid-call source update — the recorder
patching its completed-call webhook grant, the SSE feed, the TUI's live row —
needed *zero* new code. They were already subscribed to `KindCallSourceUpdate`
from the traffic-channel path
([Part 2]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }}) is
why that's possible — publish, don't call). One detail closes the loop safely:
`republishCallSource` sets `System`, and `handleCallSourceUpdate` short-circuits
any event whose `System` is already populated — so when the engine's own
subscription reads its own republished event back off the bus, it drops it, and
there's no re-entrant loop.

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="After the fix: the later grant's RID is backfilled onto the active call and republished as a call.source event, which the recorder webhook, SSE feed, and TUI all already subscribe to, so nearly every completed call reports the source">
  <text x="340" y="16" text-anchor="middle" fill="var(--accent)" font-size="11">after #915 — the RID is folded on and republished</text>
  <rect x="14" y="66" width="150" height="46" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="89" y="86" text-anchor="middle" fill="currentColor" font-size="10">later grant, src≠0</text>
  <text x="89" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="9">matches active call</text>
  <line x1="164" y1="89" x2="210" y2="89" stroke="var(--accent)"/>
  <polygon points="210,85 220,89 210,93" fill="var(--accent)"/>
  <rect x="222" y="66" width="164" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="304" y="85" text-anchor="middle" fill="var(--accent)" font-size="10">BackfillSourceFromGrant</text>
  <text x="304" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">fill iff SourceID == 0</text>
  <line x1="386" y1="89" x2="432" y2="89" stroke="var(--accent)"/>
  <polygon points="432,85 442,89 432,93" fill="var(--accent)"/>
  <rect x="444" y="66" width="164" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="526" y="85" text-anchor="middle" fill="var(--accent)" font-size="10">republishCallSource</text>
  <text x="526" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">KindCallSourceUpdate</text>
  <line x1="526" y1="112" x2="526" y2="132" stroke="var(--fg-muted)"/>
  <polygon points="522,132 526,142 530,132" fill="var(--fg-muted)"/>
  <g stroke="var(--fg-muted)">
    <line x1="470" y1="146" x2="360" y2="160"/><polygon points="360,156 351,161 362,164" fill="var(--fg-muted)"/>
    <line x1="526" y1="146" x2="526" y2="160"/>
    <line x1="582" y1="146" x2="660" y2="160"/><polygon points="658,156 668,160 658,164" fill="var(--fg-muted)"/>
  </g>
  <text x="345" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="9">recorder webhook</text>
  <text x="526" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SSE feed</text>
  <text x="645" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="9">TUI row</text>
</svg>
<figcaption>The recovered RID rides the existing source-update event, so the recorder webhook, SSE feed, and TUI patch their view with no new code — the payoff of publish-don't-call.</figcaption>
</figure>

## The residual gap (and where it's closed)

Talkgroup-keyed matching gets the easy case: the later grant matches the active
call because it carries the *same* talkgroup. But a field test on a
heavily-compressed Phase 2 system measured that alone at only **~12%** coverage.
The reason is subtle: on such systems the RID-bearing `GRP_VCH_GRANT` frequently
arrives under a *different* talkgroup label than the source-less grant that bound
the call — a mis-aliased compressed grant, or a super-group/patch remap — so it
never matches the `(System, GroupID, Timeslot)` dedup key and this backfill misses
it. Worse, with a free radio, that mismatched grant would spawn a *phantom
second call*.

The remedy is a wider net: a frequency + timeslot hosts exactly one in-progress
transmission, so a source-carrying grant landing on an active call's *exact
channel* belongs to that call regardless of its talkgroup label. That
physical-channel recovery — `BackfillSourceForChannel` — folds the RID on by
channel and suppresses the phantom duplicate at the same time. It's the same
fill-only-when-zero, republish-once discipline, cast wider, and it's the subject
of [Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }}),
where the patch/super-group remap that causes the label mismatch is the through
line.

## Where this goes next

With a source RID on nearly every call, the affiliation tracker in
[Part 8]({{ '/blog/deep-dives/trunking-engine-08-affiliation-tracking/' | relative_url }})
has real data to work with — "who is on which talkgroup" is only meaningful once
"who" is known. Then
[Part 9]({{ '/blog/deep-dives/trunking-engine-09-patches-supergroups/' | relative_url }})
closes the residual coverage gap with physical-channel recovery and unpacks the
patch/super-group remaps that made talkgroup-keyed matching miss.

## FAQ

**What is a source RID?**
The source radio ID — the subscriber-unit identifier of the radio that keyed up
to start a transmission. It's the "who" of a call; `SourceID` on the grant. The
RID database gives that number an operator-facing alias, owner, and grouping.

**Why did only ~18% of calls have a source before the fix?**
Because the completed-call webhook read the source off the grant that *bound* the
call, and on P25 Phase 2 a call frequently binds from a source-less compressed
grant while the RID-bearing grant arrives later. That later RID matched the active
call but was discarded by the dedup guard. Only the ~18% whose voice-side
`call.source` decoded on the traffic channel carried a source.

**Does the grant backfill override an in-call source update?**
No — the reverse. `BackfillSourceFromGrant` only fills when `SourceID` is still
zero, so the in-call `UpdateSource` (from the traffic-channel
`GROUP_VOICE_CHANNEL_USER` PDU), which reflects the radio actually keyed on the
channel, always takes precedence. A control-channel grant is a fallback.

**Why republish an event instead of just updating the call in place?**
Because downstream consumers — the recorder building the webhook payload, the SSE
feed, the TUI — already learned the source from the traffic-channel
`KindCallSourceUpdate` event. Republishing the recovered RID through that same
event means none of them needed new code. It's the observer decoupling from Part 2
paying off.

**Why does the republish fire only once per call?**
The control channel repeats the RID-bearing grant many times. The fill-only-when-
zero rule flips `SourceID` non-zero on the first fill, so every subsequent repeat
returns `filled == false` and republishes nothing — keeping the heavily-repeated
grant stream from flooding the event bus.

## Series navigation

**Part 7 of 12** · ←
[Part 6: Talkgroups, Aliases & Scan Modes]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})
· Next →
[Part 8: Affiliation Tracking]({{ '/blog/deep-dives/trunking-engine-08-affiliation-tracking/' | relative_url }})
