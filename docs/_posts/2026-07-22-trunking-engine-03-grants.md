---
title: "Trunking Engine, Part 3: Grants — The Engine's Only Input"
description: A field-by-field tour of GopherTrunk's protocol-agnostic Grant struct, source-bearing versus source-less grants, and the duplicate-grant guard that stops a repeated control-channel TSBK from binding a second radio.
category: deep-dives
keywords: p25 grant, control channel grant, trunking grant struct, source rid grant, duplicate grant guard, p25 phase 2 compressed grant, observedkey, gophertrunk trunking, protocol agnostic grant
tags: [trunking, go, p25, dmr, grants, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 3
---

*Part 3 of **Trunking Engine**. [Part 2]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }})
took apart the event bus the engine lives on. Now we follow the one event the
engine treats as *input* — the grant — from the wire shape the decoders hand it
to the guard that stops the same grant from allocating two radios.*

> **TL;DR:** A `Grant` is the protocol-agnostic "talkgroup T is up on frequency F"
> message that P25/DMR/NXDN control-channel decoders publish as `KindGrant`. It
> carries the talkgroup (`GroupID`), the voice frequency, an optional TDMA
> `Timeslot`, the originating radio (`SourceID`), and encryption flags
> (`Encrypted`, `AlgorithmID`, `KeyID`). Two things make grant handling subtle:
> some grants arrive **source-less** (a P25 Phase 2 compressed grant has no
> `SourceID`), and the control channel **repeats** a grant while a call is live —
> so the engine keys a logical call on `(System, GroupID, Timeslot)` and folds
> repeats into the existing call instead of binding a second radio.

**Key takeaways**

- `Grant` is **protocol-neutral**: one struct fed by every control-channel
  decoder. `FrequencyHz` must be resolved by the protocol layer; a zero-frequency
  grant is logged and dropped.
- A call's identity is `(System, GroupID, Timeslot)`, **not frequency** — a
  band-plan re-map can move a live call to a new channel without it becoming a
  new call.
- The **duplicate-grant guard** treats a repeat as "this call is still going,"
  refreshes the bind's `LastHeardAt`, and returns — one radio per logical call.
- **Source-less grants** are the seam this series pulls on: P25 Phase 2
  compressed grants land with `SourceID == 0`, which is exactly the gap
  [Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
  closes.

## Cheat sheet

| Grant field | Type | Meaning |
|---|---|---|
| `System` | `string` | which configured system granted the call |
| `Protocol` | `string` | `"p25"` / `"dmr"` / `"nxdn"` |
| `GroupID` | `uint32` | talkgroup (or, if `Individual`, a subscriber address) |
| `SourceID` | `uint32` | originating radio (RID); **0 = unknown** |
| `FrequencyHz` | `uint32` | voice-channel frequency; **0 → dropped** |
| `Timeslot` | `uint8` | TDMA slot, 1-based; 0 = N/A (Phase 1, NXDN, analog) |
| `Encrypted` | `bool` | privacy bit; `AlgorithmID`/`KeyID` fill in later |
| `Emergency` | `bool` | bypasses lockout and the scan-list gate |
| `Individual` | `bool` | `GroupID` is a 24-bit unit address, not a talkgroup |
| `CallID` | `uint64` | assigned by the voice pool on `Bind`; fences audio bleed |

## In this post

- **What a grant is** and the invariants the engine relies on.
- **The struct, field by field** — identity, TDMA, encryption, and the flags.
- **Source-bearing vs source-less** grants — and why Phase 2 arrives blank.
- **The duplicate-grant guard** — `observedKey` and the repeat-TSBK fold.

## What a grant is

On a trunked system voice doesn't live on a fixed channel. The control channel
announces assignments, and the decoders in the
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }}) series
turn each announcement into a `Grant` and publish it. The engine subscribes to
`KindGrant` and does everything else — lookup, allocation, retune, watchdog. The
grant is the *only* thing that drives a call into existence; everything else the
engine does is a reaction to one.

Two invariants are stated right on the type. First, `FrequencyHz` is the
protocol layer's job: P25 derives it from `IdentifierUpdate` band-plan TSBKs,
DMR/NXDN from the configured system. If it's zero when the grant reaches the
engine, the engine logs and drops it — a grant with no channel is not
actionable. Second, the grant is *protocol-agnostic*: the same struct carries a
P25 Phase 2 TDMA grant and an analog EDACS ProVoice grant, so the engine's
dispatch logic never branches on protocol.

## The struct, field by field

```go
// internal/trunking/grant.go (shape)
type Grant struct {
    System      string // matches trunking.System.Name
    Protocol    string // "p25" / "dmr" / "nxdn"
    GroupID     uint32 // talkgroup or destination subscriber address
    SourceID    uint32 // originator (subscriber unit) — 0 when unknown
    FrequencyHz uint32 // voice channel frequency (0 → engine drops)
    Timeslot    uint8  // 1-based TDMA slot; 0 = not applicable
    Encrypted   bool
    Emergency   bool
    AlgorithmID uint8  // meaningful only when Encrypted
    KeyID       uint16
    Individual  bool   // GroupID is a 24-bit unit address, not a TG
    PatchedGroups []uint32 // member TGs when GroupID is a patch super-group
    At          time.Time
    CallID      uint64 // stamped by VoicePool.Bind
    // …RFSSID/SiteID/NAC (site labelling), DMRInterleavedVoice, ProVoice,
    //   P25Phase1DemodMode, P25Phase2Decode (per-protocol decode hints)
}
```

The fields group into four jobs:

**Identity.** `System`, `GroupID`, and `Timeslot` together name a logical call.
`Timeslot` is 1-based on purpose: `0` means "not applicable" for Phase 1, NXDN,
and analog, where frequency alone identifies the call. DMR Tier III runs *two*
independent calls on one 12.5 kHz carrier (TS1 + TS2), so the engine treats
`(FrequencyHz, Timeslot)` as the physical channel and `(System, GroupID,
Timeslot)` as the logical call. `RFSSID`/`SiteID`/`NAC` label a P25 grant by site
(they stay zero until the first RFSS Status Broadcast lands — see
[Part 10]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})).

**Encryption.** `Encrypted` is the raw privacy bit. `AlgorithmID` and `KeyID` are
meaningful only when it's set and often arrive *after* the grant — a P25 Phase 1
grant carries only the bit, and the LDU2 Encryption Sync with the real ALGID/KID
lands mid-call
([Part 11]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})).
`Grant.String()` renders `E(alg=0x84,key=0x1A2B)` only once those non-zero values
have surfaced, so a log line is self-describing.

**Classification flags.** `Emergency` bumps a call to the top of the priority
order and bypasses lockout and the scan-list gate. `Individual` marks a grant
whose `GroupID` is a 24-bit subscriber address (a unit-to-unit target,
interconnect, or SNDCP data unit) rather than a 16-bit talkgroup — recording
those as "talkgroups" produces phantom >16-bit entries, so discovery skips them.
`DataCall` and `ProVoice` route the call away from the standard voice recorder.

**Handoff bookkeeping.** `CallID` is a process-monotonic id the voice pool stamps
on `Bind` ([Part 4]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }})).
A real handoff (`Retune`) preserves it, so a followed call keeps one identity and
the recorder can reject a stale frame from the call that previously held a reused
tap serial.

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="A Grant struct grouped into four field categories: identity fields, encryption fields, classification flags, and handoff bookkeeping, all feeding the trunking engine's HandleGrant">
  <rect x="10" y="14" width="150" height="60" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="85" y="32" text-anchor="middle" fill="var(--accent)" font-size="11">identity</text>
  <text x="85" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="9">System · GroupID</text>
  <text x="85" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Timeslot · Freq</text>
  <rect x="176" y="14" width="150" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="251" y="32" text-anchor="middle" fill="currentColor" font-size="11">encryption</text>
  <text x="251" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Encrypted · AlgID</text>
  <text x="251" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">KeyID</text>
  <rect x="342" y="14" width="150" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="417" y="32" text-anchor="middle" fill="currentColor" font-size="11">classification</text>
  <text x="417" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Emergency</text>
  <text x="417" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">Individual · Data</text>
  <rect x="508" y="14" width="160" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="588" y="32" text-anchor="middle" fill="currentColor" font-size="11">handoff</text>
  <text x="588" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="9">CallID</text>
  <text x="588" y="62" text-anchor="middle" fill="var(--fg-muted)" font-size="9">PatchedGroups</text>
  <line x1="85" y1="74" x2="300" y2="112" stroke="var(--fg-muted)"/>
  <line x1="251" y1="74" x2="320" y2="112" stroke="var(--fg-muted)"/>
  <line x1="417" y1="74" x2="360" y2="112" stroke="var(--fg-muted)"/>
  <line x1="588" y1="74" x2="380" y2="112" stroke="var(--fg-muted)"/>
  <rect x="250" y="114" width="180" height="38" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="340" y="132" text-anchor="middle" fill="var(--accent)" font-size="11">HandleGrant(g)</text>
  <text x="340" y="146" text-anchor="middle" fill="var(--fg-muted)" font-size="9">one struct, every protocol</text>
</svg>
<figcaption>One protocol-agnostic struct. Every control-channel decoder fills the same four field groups; the engine's dispatch never branches on protocol.</figcaption>
</figure>

## Source-bearing vs source-less grants

`SourceID` is the RID of the radio that keyed up — the "who" of a call. On a P25
Phase 1 system the group-voice-channel grant TSBK carries it, so the grant that
starts a call already knows the source. On other systems it doesn't.

The sharp case is **P25 Phase 2**. To conserve control-channel airtime, Phase 2
systems send a *compressed* grant form (an MMR — Multi-MAC Response) that omits
`SOURCE_ID` and `SVC_OPTIONS` entirely. That grant reaches the engine with
`SourceID == 0` and `Encrypted == false` — not because the call is anonymous and
clear, but because those fields simply weren't on the wire. The real source
surfaces later, either in the traffic-channel `GROUP_VOICE_CHANNEL_USER` PDU
(handled in
[Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }}))
or on a subsequent control-channel grant repeat that *does* carry it.

This is the problem the whole of Part 7 is about: on a busy Phase 2 system, the
grant that *binds* a call is frequently the source-less one, and the RID-bearing
grant arrives moments later — sometimes even under a different talkgroup label.
For now the point is just that `SourceID == 0` is a first-class, expected state,
and the engine must not treat "no source yet" as "no source ever."

## The duplicate-grant guard

The control channel doesn't announce a call once. Phase 1 *repeats* the
voice-grant TSBK the whole time a call is live — the reporter's log in issue #356
showed two grants for `tg=32181 freq=773431250` arriving 20 ms apart. Naively,
each repeat would allocate another radio: two tuners on one call, a duplicate
WAV, and an operator view that can't tell which device is serving it.

The guard is a logical-call key:

```go
// internal/trunking/engine.go
// (System, talkgroup, timeslot) — the call's identity, NOT frequency.
func observedKey(g Grant) string {
    return fmt.Sprintf("%s|%d|%d", g.System, g.GroupID, g.Timeslot)
}
```

Inside `HandleGrant`, before allocating anything, the engine scans its active
calls for one matching `(System, GroupID, Timeslot)`. If it finds one:

- **Same frequency** → the CC is just re-asserting a live call. The engine
  `Touch`es the bind's `LastHeardAt` (feeding the watchdog in
  [Part 12]({{ '/blog/deep-dives/trunking-engine-12-cc-hunting-watchdog-testing/' | relative_url }}))
  and returns. No second radio.
- **New frequency** → a real handoff or a band-plan `IdentifierUpdate` re-mapping
  the channel. The engine *retunes the same device in place* (preserving
  `StartedAt` and `CallID`) rather than starting a new call.

Matching on identity rather than frequency is the subtle part, and it's a fix in
its own right: an earlier version keyed on frequency, so a mid-call band-plan
re-map missed the match and bound a *second* tap to the same talkgroup — two
"Active calls" rows for one call. `System` keeps two systems' identical
talkgroup numbers apart; `Timeslot` keeps a DMR Tier III carrier's two per-slot
calls apart. That same repeat-grant path is also where a later grant's `SourceID`
gets folded onto a call that bound source-less (`BackfillSourceFromGrant`) — the
hook Part 7 builds on.

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="A repeat grant matching an active call by system, group, and timeslot is folded into the existing call: same frequency refreshes LastHeardAt, a new frequency retunes the device in place, and only an unmatched grant allocates a new radio">
  <rect x="14" y="70" width="120" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="74" y="88" text-anchor="middle" fill="var(--accent)" font-size="12">grant arrives</text>
  <text x="74" y="103" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(sys, tg, ts)</text>
  <line x1="134" y1="90" x2="188" y2="90" stroke="currentColor"/>
  <polygon points="188,86 198,90 188,94" fill="currentColor"/>
  <rect x="200" y="66" width="150" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="275" y="86" text-anchor="middle" fill="currentColor" font-size="11">match active call?</text>
  <text x="275" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">scan pool by observedKey</text>
  <line x1="350" y1="80" x2="440" y2="34" stroke="var(--fg-muted)"/>
  <polygon points="440,38 449,32 439,30" fill="var(--fg-muted)"/>
  <line x1="350" y1="90" x2="440" y2="90" stroke="var(--fg-muted)"/>
  <polygon points="440,86 450,90 440,94" fill="var(--fg-muted)"/>
  <line x1="350" y1="100" x2="440" y2="146" stroke="var(--accent)"/>
  <polygon points="440,142 450,146 440,150" fill="var(--accent)"/>
  <rect x="452" y="16" width="216" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="560" y="32" text-anchor="middle" fill="currentColor" font-size="10">same freq → Touch LastHeardAt</text>
  <text x="560" y="45" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(refresh; no new radio)</text>
  <rect x="452" y="73" width="216" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="560" y="89" text-anchor="middle" fill="currentColor" font-size="10">new freq → Retune in place</text>
  <text x="560" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(handoff; keep CallID)</text>
  <rect x="452" y="130" width="216" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="560" y="146" text-anchor="middle" fill="var(--accent)" font-size="10">no match → allocate radio</text>
  <text x="560" y="159" text-anchor="middle" fill="var(--fg-muted)" font-size="9">(new call — Part 4)</text>
</svg>
<figcaption>Only an unmatched grant becomes a new call. Repeats refresh; handoffs retune. One logical call, one radio.</figcaption>
</figure>

### The design principle

The grant is a *value*, not a command with side effects. `HandleGrant` takes a
copy, and every mutation — the discovered talkgroup, the patch members, the
backfilled `CallID` — happens on that copy or on the pool's stored `ActiveCall`,
never on caller state. That keeps the hot path free of aliasing surprises: a
grant can be logged, matched, folded, or dropped without any of those paths
racing on shared memory, because the only long-lived copy is the one the voice
pool owns once a call is bound.

## Where this goes next

An unmatched grant means the engine needs a radio.
[Part 4]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }})
opens the voice pool — how a scarce set of Voice-role SDRs is allocated, retuned,
and bound to an `ActiveCall`, and what happens when a wideband rig's tuning
window doesn't cover the granted frequency. Then
[Part 5]({{ '/blog/deep-dives/trunking-engine-05-priority-preemption/' | relative_url }})
handles the case where every radio is already busy, and
[Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
returns to the source-less-grant problem introduced here.

## FAQ

**What is the difference between `GroupID` and `SourceID`?**
`GroupID` is the *destination* — the talkgroup the call is on (or, when
`Individual` is set, a target radio's 24-bit address). `SourceID` is the
*originator* — the RID of the radio that keyed up. `SourceID == 0` means the
source is unknown, which is a normal, expected state on compressed P25 Phase 2
grants.

**Why does the engine key a call on `(System, GroupID, Timeslot)` and not
frequency?**
Because a call's frequency can change mid-call — a band-plan `IdentifierUpdate`
re-maps the channel, or the system hands the call to a new one. Keying on
frequency made a re-grant miss the active call and bind a second radio. Timeslot
is in the key so a DMR Tier III carrier's two independent per-slot calls stay
distinct.

**What happens when the same grant arrives twice?**
The duplicate-grant guard matches it to the active call and, if the frequency is
unchanged, just refreshes that call's `LastHeardAt` and returns. The control
channel repeats grant TSBKs continuously while a call runs; the guard treats each
repeat as "still going," not as a new call.

**Why do P25 Phase 2 grants arrive without a source RID?**
To save control-channel airtime, Phase 2 uses a compressed grant form that omits
`SOURCE_ID` and `SVC_OPTIONS`. The source surfaces later — in the traffic-channel
`GROUP_VOICE_CHANNEL_USER` PDU or a subsequent grant repeat. Recovering it is the
subject of Part 7.

**What does a zero `FrequencyHz` mean?**
That the protocol layer couldn't resolve the channel to a frequency — usually a
missing or not-yet-seen band plan. The engine logs the grant and drops it; there
is nothing to tune.

## Series navigation

**Part 3 of 12** · ←
[Part 2: The Event Bus]({{ '/blog/deep-dives/trunking-engine-02-event-bus/' | relative_url }})
· Next →
[Part 4: The Voice Pool]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }})
