---
title: "DMR End to End, Part 6: Conventional IPSC — Two Slots as Two Calls"
description: How GopherTrunk's conventional DMR state machine turned one IPSC repeater's two timeslots into two concurrent calls — a per-destination call map, a synthetic slot token the engine keys on, two same-carrier voice taps routed by embedded LC, and the re-key rule that recovered replies lost inside hangtime.
category: deep-dives
keywords: dmr ipsc decoder, conventional dmr two timeslots, dmr tier 2 two calls, dmr interleaved voice, dmr slot router, dmr re-key hangtime, terminator with lc release, dmr same carrier voice taps, dmr conventional state machine, gophertrunk dmr ipsc
tags: [dmr-end-to-end, dmr, ipsc, tier-2, timeslots, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 6
---

*Part 6 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }})
read the Full Link Control and built late entry on it. This part is the
state machine that consumes it — `ConventionalChannel`, the per-repeater
decoder for Tier II and IPSC — and the two field reports that rebuilt it: one
carrier carrying two calls, and replies vanishing inside hangtime.*

> **TL;DR:** A conventional DMR repeater interleaves TS1 and TS2 on one
> carrier, each able to hold its own call. `tier2.ConventionalChannel` used
> to track one scalar call with `Timeslot 0`, so two talkgroups ping-ponged
> into one and both slots' AMBE sliced into one "DJ scratchy" stream. It now
> keys calls by destination (`calls map[uint32]*convCall`) and gives each a
> **synthetic** Timeslot 1/2 (`assignSlot`) — an engine-identity token only,
> because the wire never labels a burst's physical slot. Audio rides
> `dmrSameCarrierTaps = 2` taps whose `slotRouter` binds by embedded-LC
> talkgroup; `Grant.DMRInterleavedVoice` selects the 264/288 cadence decoder.
> And the 10 Sep IPSC report — replies "completely missed while the radios
> decode them" — was a header deduped against a call the voice path had
> already released: `headerRekeyDibits` (0.25 s) and
> `superframeRekeyGapDibits` (2 s) now re-grant it, counted in `rekeys`,
> A/B'd on the reporter's flac with `GT_DMR_DROP_TERMINATORS=1` (1 grant → 5).

**Key takeaways**

- **Identity is per destination, not per slot.** Two concurrent calls carry
  distinct destinations; the synthetic slot exists only so the engine's
  `(freq, timeslot)` identity keeps them apart.
- **The wire does not say which slot a burst is in.** Both slots share the BS
  sync words and the slot type carries colour + data type only — audio is
  routed by embedded-LC talkgroup, never by slot number.
- **Two decoders of the same terminator disagree in time.** The voice path
  and the control path see the same burst on different taps; a reply can
  arrive before the control path knows the call ended.
- **A header is only ever sent at keyup.** One well past a tracked call's
  last header is a new transmission — release and re-grant.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| State machine | headers → grants, terminators → releases | `internal/radio/dmr/tier2/conventional.go` |
| Concurrent calls | destination-keyed map, synthetic slot 1/2 | `calls map[uint32]*convCall`, `assignSlot` |
| Engine identity | `(System, GroupID, Timeslot)` | `internal/trunking/engine.go` (`channelKey`) |
| Two-slot voice | interleaved decoder + per-tap router | `dmrSameCarrierTaps` (`cmd/gophertrunk/daemon.go`), `slotRouter` (`composer/dmr_voice.go`) |
| Teardown | terminator releases its own destination | `terminatorDest`, `releaseCall`, `dmrvoice.TerminatorDetector` |
| Re-key inside hangtime | header 0.25 s / LC 2 s past the anchor re-grants | `headerRekeyDibits`, `superframeRekeyGapDibits`, `Counters.Rekeys` |

## In this post

- **One scalar for two slots** — the bug as the operator heard it.
- **A map keyed by destination** — and a slot that is only a token.
- **Two taps, one router** — how audio finds its call.
- **Teardown by destination** — terminators end only their own slot.
- **The re-key inside hangtime** — the 10 Sep report, root-caused.
- **What the captures say** — scrubs, counts, what is still unverified.

## One scalar for two slots

[Cookbook Part 3]({{ '/blog/tutorials/operator-cookbook-03-conventional-dmr-two-slots/' | relative_url }})
gave the operator's view: a [Tier II]({{ '/reference/dmr-tier-2/' | relative_url }})
repeater is two-slot [TDMA]({{ '/reference/tdma/' | relative_url }}), two
independent conversations on one carrier. The old `ConventionalChannel` had one
scalar call (`inCall`, `lastTG`, `lastSrc`) and every grant carried
`Timeslot 0`. The two Voice LC Headers alternated and **ping-ponged the
state**, and the engine, which keys a call on `(System, GroupID, Timeslot)`
([Trunking Engine Part 3]({{ '/blog/deep-dives/trunking-engine-03-grants/' | relative_url }})),
folded both onto one `(freq, 0)` channel.

The audio symptom had a second cause. The composer picks the AMBE decoder
from `Grant.DMRInterleavedVoice`. Tier III stamped it; Tier II never did —
`dmr_interleaved_voice` was resolved in `daemon.go`
(`resolveDMRInterleavedVoice`) and **dropped on the floor**, because
`tier2.Options` had no such field. So the single-slot decoder spliced two
calls' AMBE frames into one superframe: the "DJ scratchy" audio of #644.

## A map keyed by destination, and a slot that is only a token

The fix replaces the scalar with a map and hands each concurrent call an
identity token:

```go
// internal/radio/dmr/tier2/conventional.go (shape)
calls map[uint32]*convCall // keyed by decoded destination (TG or called RID)

func (c *ConventionalChannel) assignSlot(now time.Time) uint8 {
    /* scan c.calls for used1 / used2 and the stalest call */
    if !used1 { return 1 }
    if !used2 { return 2 }
    slot := c.calls[stalestDest].slot   // both taken ⇒ a terminator was missed: evict
    delete(c.calls, stalestDest)
    return slot
}
```

A new destination claims the lowest free synthetic slot; a repeated header for
the same `(dest, src)` is a dedupe; a new source on a tracked destination is a
talker change on the same slot. `publishGrant` stamps the slot into
`Grant.Timeslot` and `c.interleavedVoice` into `Grant.DMRInterleavedVoice`,
wired through `tier2.Options.InterleavedVoice` from both constructors; Tier I
stays single-slot.

Why *synthetic*? Because the wire does not say. A base-station burst carries
the BS sync word on **both** slots and the slot type holds only colour code
and data type. Tier III's grant CSBK carries a real timeslot bit, so
`tier3.publishGrant` maps `slot + 1` into the same field
([Part 8]({{ '/blog/deep-dives/dmr-end-to-end-08-tier3-trunking/' | relative_url }}));
Tier II has nothing to read. The token only makes the engine's `channelKey`
differ between two concurrent calls — it never routes audio.

## Two taps, one router

Conventional voice rides the **same carrier** the state machine decodes, so
the daemon registers same-carrier taps instead of a voice SDR:

```go
// cmd/gophertrunk/daemon.go (shape)
const dmrSameCarrierTaps = 2 // one per TDMA slot
// sameCarrierVoiceTaps: DMR Tier II / Tier I → 2, TETRA → 4, DMO → 1,
// trunked protocols → 0 (their voice hops to a separate traffic channel)
```

Each tap runs the composer's DMR chain
([Voice Coding Part 9]({{ '/blog/deep-dives/voice-coding-09-the-composer/' | relative_url }})).
With the interleaved flag it builds `dmrvoice.NewInterleavedDecoder`, which
knows same-slot bursts sit 264 dibits apart (no CACH) or 288 (a 12-dibit
[CACH]({{ '/reference/dmr-cach/' | relative_url }}) before each burst),
auto-detects which cadence reassembles a CRC-valid embedded LC, and tags each
superframe with `VoiceSuperframe.Phase` — a *relative* discriminator, not a
TS1/TS2 label.

`slotRouter` decides which superframes belong to this tap's call:

```go
// internal/voice/composer/dmr_voice.go (shape)
func (r *slotRouter) accept(sf dmrvoice.VoiceSuperframe) bool {
    if sf.HasLC {
        if dest, ok := lcCallDestination(sf.LC); ok {
            if dest == r.groupID { r.bound = int(sf.Phase); return true }
            r.foreignPhaseMask |= 1 << (sf.Phase & 1) // positively the other slot
            return false
        }
    }
    if r.bound >= 0 { return int(sf.Phase) == r.bound }
    /* unbound: after unboundPhaseFallbackGrace (2) LC-less superframes, bind the active phase */
}
```

An LC naming this call's destination binds the phase; one naming another
destination marks that phase foreign for ever. The fallback keeps a carrier
whose LC never decodes recording — and is the one **sharp edge** the code
names: with no LC at all (#644), both routers fall back to phase parity and
could bind the same phase, recording one slot twice.

## Teardown by destination

A Terminator with LC carries the header's FLC under the terminator RS seed,
so `terminatorDest` recovers *which* call is ending and `releaseCall(dest)`
publishes one `call.release` keyed by `(System, GroupID)` — a TS1 terminator
cannot tear down the TS2 call. When its LC does not decode,
[Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }})'s
rule applies with a two-call twist:

```
DBG dmr/tier2: terminator slot type without a BPTC-valid payload ignored cc=12
DBG dmr/tier2: ambiguous terminator (LC undecodable, two calls active); leaving to hangtime cc=12
DBG dmr/tier2: terminator dst=11 slot=1
```

A lone call still ends promptly on a BPTC-valid but RS-failing terminator;
with two calls active a guess could cross-tear the wrong slot, so the channel
releases nothing and each voice chain's hangtime (`voice_hangtime_ms`,
default 3500) closes its call. The composer runs its own
`dmrvoice.TerminatorDetector`, ending its call only when the FLC destination
equals `groupID` (`composer: dmr terminator — releasing call`). Two
independent detectors of one terminator, on two taps. Keep that in mind.

## The re-key inside hangtime

The 10 Sep IPSC report: "some calls are completely missed by GT while the
radios decode them." The operator's `debug.log` told the story. The composer's
detector released each call at PTT release. The tier2 slicer's copy of the
**same** terminator — on its weaker channelizer bin-edge tap — decoded
0.8–6.5 s *later*, or never (`terminator slot type without a BPTC-valid
payload ignored`). The control path kept the call tracked, so the reply's
header for the same `(tg, radio)` hit the dedupe branch as a repeated header
and was never granted. The fix is Part 5's anchor arithmetic, read the other way:

```go
// internal/radio/dmr/tier2/conventional.go (shape)
const headerRekeyDibits       = 1200 // 0.25 s past the last header copy
const superframeRekeyGapDibits = 9600 // 2 s from the last LC's superframe start

if existing.src == src {
    if c.ingestDibit-existing.anchorDibit <= headerRekeyDibits { /* dedupe */ }
    c.cnt.rekeys.Add(1)
    c.log.Debug("dmr/tier2: re-key — Voice LC Header for a still-tracked call; previous terminator undecoded here",
        "dst", dest, "src", src, "gap_dibits", c.ingestDibit-existing.anchorDibit)
    c.releaseCall(dest)                 // idempotent for an engine that already ended it
    c.publishGrant(dest, src, individual, enc, emer, prio, ts, slot.ColorCode, false)
}
```

**A Voice LC Header is only ever sent at keyup**, so one more than 0.25 s
past the tracked call's last header is a new transmission. Copies of one
header span ~120–180 ms; a re-key's header is at least one superframe
(360 ms) past the previous over's last superframe start — and the capture has
one reply keyed so fast that a 0.5 s rule missed it. When the header is *also* lost, a CRC-valid
embedded LC naming a tracked call more than `superframeRekeyGapDibits` after
its last LC re-grants by late entry (`dmr/tier2: re-key by late entry`): 2 s
from the last superframe *start* is ~1.6 s of silence, so a four-superframe
fade is tolerated and a reply after a ≥3 s hang time is caught.

The gotcha that cost a round: `Process` assembles a chunk's superframes
**before** slicing its data bursts, so a new over's first superframe refreshes
the call ahead of its own header — against the last LC, that header looked
stale. Hence two clocks per call, `anchorDibit` (header rule) and `lastDibit`
(silence rule), and `convCall.touch` only moves forward.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="Two timelines of one IPSC carrier: the composer voice path releases transmission one at PTT release, while on the control path's weaker tap the same terminator never decodes and the call stays tracked, so the reply's header used to be deduped; the re-key rule now releases the stale call and grants the reply.">
  <text x="20" y="22" fill="currentColor" font-size="11" font-weight="bold">two decoders of one terminator</text>
  <text x="20" y="74" fill="var(--fg-muted)" font-size="9">voice path</text>
  <line x1="150" y1="70" x2="660" y2="70" stroke="var(--fg-muted)"/>
  <rect x="160" y="58" width="150" height="24" fill="none" stroke="currentColor"/>
  <text x="235" y="74" text-anchor="middle" fill="currentColor" font-size="9">over 1 · tg 11 src A</text>
  <rect x="312" y="58" width="22" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="323" y="52" text-anchor="middle" fill="var(--accent)" font-size="8">TERM</text>
  <text x="400" y="74" fill="var(--accent)" font-size="9">call.release at PTT release</text>
  <text x="20" y="144" fill="var(--fg-muted)" font-size="9">control path (weaker tap)</text>
  <line x1="150" y1="140" x2="660" y2="140" stroke="var(--fg-muted)"/>
  <rect x="160" y="128" width="150" height="24" fill="none" stroke="currentColor"/>
  <text x="235" y="144" text-anchor="middle" fill="currentColor" font-size="9">over 1 tracked · anchor</text>
  <rect x="312" y="128" width="22" height="24" fill="none" stroke="var(--fg-muted)" stroke-dasharray="3 2"/>
  <text x="323" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="8">TERM?</text>
  <rect x="420" y="128" width="26" height="24" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="433" y="122" text-anchor="middle" fill="var(--accent)" font-size="8">HDR</text>
  <rect x="450" y="128" width="150" height="24" fill="none" stroke="currentColor"/>
  <text x="525" y="144" text-anchor="middle" fill="currentColor" font-size="9">over 2 · tg 11 src A</text>
  <line x1="310" y1="182" x2="420" y2="182" stroke="var(--accent)"/>
  <polygon points="414,178 424,182 414,186" fill="var(--accent)"/>
  <text x="365" y="196" text-anchor="middle" fill="var(--accent)" font-size="9">gap &gt; headerRekeyDibits (1200)</text>
  <text x="120" y="222" fill="var(--fg-muted)" font-size="9">old: deduped as a repeated header — reply never granted</text>
  <text x="120" y="238" fill="var(--accent)" font-size="9">new: releaseCall + re-grant · rekeys++</text>
</svg>
<figcaption>The 10 Sep miss: the voice path released the call, the control path's copy of the terminator never decoded, and the reply's header was deduped against a call already ended — until the anchor distance made it a re-key.</figcaption>
</figure>

## What the captures say

`TestDMRIPSCReplay` (`cmd/gophertrunk/dmr_ipsc_replay_test.go`) reads
wav/flac directly and prints a grant / release / terminator timeline in stream
time (4800 dibits/s). Two scrubs model the field conditions on the reporter's
`dmr_ipsc_beacon_voice.flac`:

| Run | Grants | Notes |
|---|---|---|
| `GT_DMR_DROP_TERMINATORS=1`, old code | 1 of 5 | every reply deduped |
| `GT_DMR_DROP_TERMINATORS=1`, new | 5 | `rekeys=4` |
| unscrubbed | 5 | one on-air re-key had no terminator at all |
| `GT_DMR_DROP_HEADERS=1` | 4 late entries | unchanged |

Mid-hangtime the repeater also repeats the *other* slot's last terminator
(tg 25862 here) — harmless. The synthetic pins are
`conventional_twoslot_test.go` (two destinations → two grants with distinct
slots, a TS1 terminator releasing only its own) and
`conventional_rekey_test.go` (`TestConventionalRekeyWithinHangtimeRegrants`
fails against the old dedupe with 1 grant, 0 rekeys).

Two verification states, stated the way
[From Spec to Shipping Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})
insists. The **re-key rule** is real-air A/B'd offline on the operator's
capture and corroborated by their log; the live run is pending. The
**two-slot audio path** is synthetic-verified only: the one concurrent-traffic
IQ contributed (`dmr_ipsc_60sec_bw25k_cs16.raw`) is dead — ~−75 dBFS RMS, no
frame sync — so that A/B still needs a decodable capture (#764/#771).

### How the two-slot problem shaped the Go code

- **Identity tokens are typed as such.** `convCall.slot` is documented as an
  engine-identity token never consulted for audio, so nobody later "fixes"
  routing by reading it.
- **Two clocks, both stream-time.** `anchorDibit` and `lastDibit` exist
  because one clock served two rules badly.
- **Releases are idempotent by design.** `releaseCall` publishes for a call
  the voice path may already have ended, so two detectors can disagree in
  time.
- **Counters carry the field report.** `rekeys` is the number that would have
  been non-zero in the 10 Sep log — the lesson of
  [Issue Tracker Part 22]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})'s
  drifting pipelines.

## Where this goes next

Between calls a keyed IPSC repeater is not silent — it fills both slots with
Idle bursts, and sometimes with a train of CSBKs nobody can yet read.
[Part 7]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }})
pins the idle beacon against MMDVMHost's constant, turns it into a site-alive
lock, and is honest about the cc=7 CSBK train still unpinned.

## FAQ

**How does GopherTrunk decode both DMR timeslots of one repeater?**
The conventional state machine keys calls by decoded destination and gives
each a synthetic Timeslot 1 or 2 so the engine keeps them apart; two
same-carrier voice taps run the interleaved AMBE decoder and a `slotRouter`
that keeps only superframes whose embedded LC names its call.

**Why is the DMR timeslot "synthetic" in conventional mode?**
Because a base-station burst carries no reliable physical slot label — both
slots use the same BS sync words and the slot type holds only colour code and
data type. The token separates two concurrent calls in the engine; audio is
routed by embedded-LC talkgroup, not by slot number.

**Why did GopherTrunk miss DMR replies on an IPSC repeater?**
The voice path released each call at PTT release while the control path's
copy of the terminator decoded late or never on a weaker tap, so the reply's
header was deduped against a call already ended. A header more than 0.25 s
past the call's anchor now re-grants (`rekeys`).

**What does `dmr_interleaved_voice` do?**
It is a tri-state override for the cadence-detecting AMBE decoder. Unset,
every DMR protocol gets it (`trunking.DMRVoiceCadenceDetected`); the value is
stamped on each grant as `DMRInterleavedVoice` so the composer picks the right
decoder. Forcing `false` reproduces the single-slot splice.

**Is the two-slot DMR path verified on air?**
Partly. The re-key rule is A/B'd on the reporter's real capture; the two-slot
audio path is pinned by failing-first synthetic tests but still awaits a
decodable concurrent-traffic capture — the only one so far was undecodable at
~−75 dBFS.

## Series navigation

**Part 6 of 14** · ←
[Part 5: Link Control, Embedded LC & Late Entry]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }})
· Next →
[Part 7: The Idle Beacon & the CSBK Train]({{ '/blog/deep-dives/dmr-end-to-end-07-idle-beacon-csbk/' | relative_url }})
