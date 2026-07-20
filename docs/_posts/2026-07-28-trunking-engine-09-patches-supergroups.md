---
title: "Trunking Engine, Part 9: Patches, Supergroups & Physical-Channel RID Recovery"
description: How GopherTrunk tracks P25 patches and dynamic-regroup supergroups, and how folding a source RID by physical channel recovers the transmitter and kills a phantom duplicate call from a mis-aliased compressed grant.
category: deep-dives
keywords: p25 patch, dynamic regroup, supergroup, motorola patch, harris patch, physical channel rid recovery, source rid backfill, compressed grant, duplicate call suppression, gophertrunk trunking
tags: [trunking, go, p25, patch, event-bus, architecture]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 9
---

*Part 9 of **Trunking Engine**, a 12-part deep dive into the "brain" of
GopherTrunk. Part 8 built a roster from the event stream. This one deals with two
places where the neat one-grant-one-call model bends: a **patch**, where one RF
call belongs to many talkgroups at once, and a **compressed grant**, where the
talkgroup label on the wire lies about which call the source RID belongs to.*

> **TL;DR:** A patch (P25 dynamic regroup / supergroup) merges member talkgroups
> onto one RF channel, so a call on the supergroup physically *is* the members'
> traffic — following it means attributing, not retuning. The engine keeps a live
> `PatchRegistry` from `KindPatch` events. Separately, the #915 follow-up recovers
> a call's source RID by **physical channel**: a frequency + timeslot hosts
> exactly one transmission, so a source-carrying grant landing on an active call's
> exact channel is folded onto that call *regardless of its talkgroup label* —
> which also suppresses the phantom duplicate call a mis-aliased compressed grant
> would otherwise spawn.

**Key takeaways**

- A **patch** is an attribution relationship, not a tuning one: the supergroup and
  its members share one channel, so one recording serves all of them.
- `PatchRegistry` is a thread-safe live table keyed by supergroup, maintained from
  `KindPatch` add/cancel events — derived state, exactly like Part 8's roster.
- **Physical-channel RID recovery** keys on `(frequency, timeslot)` instead of
  talkgroup, so a source RID arriving under a *different* label than the call was
  bound with still lands on the right call.
- The same rule **suppresses a phantom duplicate**: a mis-aliased grant that would
  otherwise look like a new call is recognized as the running one and folded in.

## Cheat sheet

| Concept | Where it lives | One-line role |
|---|---|---|
| `Patch` | `patch.go` | `KindPatch` payload — a patch add or cancel the system announced |
| `PatchGroup` | `patch.go` | supergroup → member talkgroups + vendor |
| `PatchRegistry` | `patch.go` | thread-safe live table of active patches, keyed by supergroup |
| `handlePatch(p)` | `engine.go` | applies an add (record) or cancel (delete) to the registry |
| `BackfillSourceForChannel(...)` | `voicepool.go` | folds a source RID onto the active call on `(freq, timeslot)` |
| `republishCallSource(...)` | `engine.go` | re-emits the recovered RID so the webhook + live view patch |

## In this post

- **What a patch is** — supergroups, dynamic regroup, and why following one means
  attributing, not tuning.
- **The `PatchRegistry`** — a derived table maintained from `KindPatch` events.
- **The problem we hit** — a mis-aliased compressed grant spawning a phantom
  duplicate call and losing the source RID.
- **Physical-channel RID recovery** — keying on `(freq, timeslot)` to fold the RID
  and kill the duplicate.

## What a patch actually is

On P25 (and its Motorola and Harris variants) a dispatcher can **patch** several
talkgroups together, or spin up a **dynamic regroup** supergroup, so that units
scattered across those talkgroups all hear one another. The system announces this
on the control channel, and the important consequence for a scanner is physical:
the patched talkgroups now **share one RF channel**. A voice call on the
supergroup address *is* the traffic for every member simultaneously.

That flips the usual engine job on its head. For an ordinary grant, "following"
means allocating a voice SDR and retuning it. For a patch there is nothing extra
to tune — the call is already on the air on one channel — so "following" a patch
means **attribution**: recording the call once and crediting it to the supergroup
*and* every member talkgroup. The `PatchGroup` doc comment states it plainly:
following a patch "means attributing the call to every member, not retuning."

<figure class="lab-figure">
<svg viewBox="0 0 680 180" width="680" height="180" role="img" aria-label="A single RF voice call on a supergroup address maps to several member talkgroups through the patch registry, so one recording is attributed to all members without any extra tuning">
  <rect x="16" y="66" width="150" height="48" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="91" y="86" text-anchor="middle" fill="var(--accent)" font-size="12">1 RF voice call</text>
  <text x="91" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="10">supergroup 0xF001</text>
  <line x1="166" y1="90" x2="226" y2="90" stroke="currentColor"/>
  <polygon points="226,86 236,90 226,94" fill="currentColor"/>
  <rect x="236" y="60" width="150" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="311" y="84" text-anchor="middle" fill="currentColor" font-size="12">PatchRegistry</text>
  <text x="311" y="100" text-anchor="middle" fill="var(--fg-muted)" font-size="10">MembersOf(0xF001)</text>
  <g stroke="var(--fg-muted)">
    <line x1="386" y1="76" x2="470" y2="40"/><polygon points="468,36 478,39 468,44" fill="var(--fg-muted)"/>
    <line x1="386" y1="90" x2="470" y2="90"/><polygon points="470,86 480,90 470,94" fill="var(--fg-muted)"/>
    <line x1="386" y1="104" x2="470" y2="140"/><polygon points="468,136 478,141 467,143" fill="var(--fg-muted)"/>
  </g>
  <rect x="480" y="26" width="184" height="28" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="572" y="44" text-anchor="middle" fill="var(--fg-muted)" font-size="10">talkgroup 1201 (member)</text>
  <rect x="480" y="76" width="184" height="28" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="572" y="94" text-anchor="middle" fill="var(--fg-muted)" font-size="10">talkgroup 1202 (member)</text>
  <rect x="480" y="126" width="184" height="28" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="572" y="144" text-anchor="middle" fill="var(--fg-muted)" font-size="10">talkgroup 1203 (member)</text>
  <text x="330" y="172" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one recording, three attributions — no second SDR, no retune</text>
</svg>
<figcaption>A patch is an attribution fan-out: one physical call credited to the supergroup and every member, resolved through the registry rather than by tuning more radios.</figcaption>
</figure>

## The PatchRegistry

The registry is small and unglamorous on purpose — a mutex-guarded map from
supergroup address to `PatchGroup`, with the operations the engine needs to
answer "is this address a supergroup, and who are its members?":

```go
// internal/trunking/patch.go (shape)
type PatchGroup struct {
    SuperGroup uint32
    Members    []uint32
    Vendor     string // "motorola" | "harris"
    UpdatedAt  time.Time
}

type PatchRegistry struct {
    mu     sync.Mutex
    groups map[uint32]PatchGroup
}

func (r *PatchRegistry) Apply(pg PatchGroup)          { /* record or replace */ }
func (r *PatchRegistry) Delete(superGroup uint32)     { /* patch cancelled   */ }
func (r *PatchRegistry) MembersOf(group uint32) []uint32 // copy, or nil if not a supergroup
```

The engine owns one registry and drives it from the bus. A `KindPatch` event
carries an `Add` flag — `true` when a patch goes active, `false` when it is
cancelled — and `handlePatch` is a two-line dispatch:

```go
// internal/trunking/engine.go (shape)
func (e *Engine) handlePatch(p Patch) {
    if p.Add {
        e.patches.Apply(PatchGroup{
            SuperGroup: p.SuperGroup, Members: p.Members,
            Vendor: p.Vendor, UpdatedAt: e.now(),
        })
        return
    }
    e.patches.Delete(p.SuperGroup)
}
```

This is the same derived-state pattern as Part 8: the registry is a materialized
view of the patch announcements on the control channel, not something the engine
computes independently. `MembersOf` returns a *copy* of the member slice so a
caller can't mutate the live table, and `Active()` snapshots the whole set for the
API. The scan-list gate in `HandleGrant` already respects it — a grant is
followed when the supergroup *or any member* is scanned, so patching a scanned
talkgroup into a supergroup doesn't accidentally silence it.

## The problem we hit

Now the harder half, and it starts with a real field failure logged in #915.
GopherTrunk had just shipped a fix so the completed-call webhook carried the
source RID on nearly every call, by folding a later same-call grant's RID onto the
bound call. The matching key was `(System, talkgroup, timeslot)` — find the active
call with the same talkgroup and stamp its source. On most systems that worked.

On a heavily-compressed P25 Phase 2 system it reached only **~12% coverage**, and
the reason was nasty: on those systems the RID-bearing grant frequently arrives
under a *different talkgroup label* than the source-less compressed grant that
originally bound the call. A mis-aliased compressed grant, or a supergroup/patch
remap, means the second grant says "talkgroup 1202, source 0x4A21" while the call
is running as "talkgroup 1201." Talkgroup-keyed matching misses it entirely, so
two bad things happen at once:

1. The source RID is **never associated** back to the running call — the webhook
   ships without it.
2. With a free voice device available, the engine treats the mismatched grant as a
   brand-new call and **binds a second SDR** — a phantom duplicate of a call
   already on the air, on the same frequency.

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="Before the fix, a source-carrying grant under a different talkgroup label misses the talkgroup match and spawns a phantom second call; after the fix, matching on frequency and timeslot folds the RID onto the running call and suppresses the duplicate">
  <text x="170" y="20" text-anchor="middle" fill="var(--fg-muted)" font-size="11">before — match on talkgroup</text>
  <rect x="20" y="30" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="95" y="49" text-anchor="middle" fill="currentColor" font-size="10">call: tg 1201 @ F, src 0</text>
  <rect x="20" y="70" width="150" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="95" y="89" text-anchor="middle" fill="var(--fg-muted)" font-size="10">grant: tg 1202 @ F, src X</text>
  <line x1="170" y1="85" x2="230" y2="85" stroke="var(--fg-muted)"/>
  <polygon points="230,81 240,85 230,89" fill="var(--fg-muted)"/>
  <rect x="240" y="66" width="96" height="38" rx="5" fill="none" stroke="currentColor"/>
  <text x="288" y="82" text-anchor="middle" fill="currentColor" font-size="10">tg mismatch</text>
  <text x="288" y="96" text-anchor="middle" fill="var(--fg-muted)" font-size="9">→ new bind</text>
  <text x="288" y="126" text-anchor="middle" fill="currentColor" font-size="11">✗ phantom 2nd SDR</text>
  <text x="288" y="140" text-anchor="middle" fill="var(--fg-muted)" font-size="9">RID lost, duplicate WAV</text>
  <line x1="360" y1="20" x2="360" y2="180" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="520" y="20" text-anchor="middle" fill="var(--accent)" font-size="11">after — match on (freq, timeslot)</text>
  <rect x="376" y="30" width="150" height="30" rx="5" fill="none" stroke="currentColor"/>
  <text x="451" y="49" text-anchor="middle" fill="currentColor" font-size="10">call: tg 1201 @ F, src 0</text>
  <rect x="376" y="70" width="150" height="30" rx="5" fill="none" stroke="var(--fg-muted)"/>
  <text x="451" y="89" text-anchor="middle" fill="var(--fg-muted)" font-size="10">grant: tg 1202 @ F, src X</text>
  <line x1="526" y1="85" x2="586" y2="85" stroke="var(--accent)"/>
  <polygon points="586,81 596,85 586,89" fill="var(--accent)"/>
  <rect x="596" y="66" width="70" height="38" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="631" y="82" text-anchor="middle" fill="var(--accent)" font-size="9">same F+TS</text>
  <text x="631" y="96" text-anchor="middle" fill="var(--accent)" font-size="9">→ fold</text>
  <text x="520" y="150" text-anchor="middle" fill="var(--accent)" font-size="11">✓ src X folded onto call 1201</text>
  <text x="520" y="166" text-anchor="middle" fill="var(--fg-muted)" font-size="9">duplicate suppressed, RID republished</text>
</svg>
<figcaption>The talkgroup label lies about identity under compression; the physical channel does not. Keying on <code>(freq, timeslot)</code> both recovers the RID and kills the duplicate call.</figcaption>
</figure>

## Physical-channel RID recovery

The fix leans on a fact the wire can't fake: **a frequency plus a timeslot hosts
exactly one in-progress transmission.** Two radios cannot key the same FDMA
channel — or the same TDMA slot on it — at the same instant. So a source-carrying
grant that lands on an *active call's exact channel* belongs to that call, whatever
talkgroup label it is wearing. In `HandleGrant`, after the talkgroup-keyed dedup
has had its chance, a second pass keys on the physical channel:

```go
// internal/trunking/engine.go (shape) — after the talkgroup dedup loop
if g.SourceID != 0 && g.FrequencyHz != 0 {
    if serial, upd, filled := e.pool.BackfillSourceForChannel(
        g.System, g.FrequencyHz, g.Timeslot, g.SourceID); filled {
        e.pool.Touch(serial, e.now())
        e.republishCallSource(serial, upd) // patch webhook + live view
        return // treat as a repeat of that call — do NOT allocate a new device
    }
}
```

Two effects fall out of that one block. `BackfillSourceForChannel` folds the RID
onto the call on `(system, freq, timeslot)` — but only when the source is still
unknown, so an in-call `call.source` update (the radio actually keyed on the
traffic channel, from [Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }}))
always wins. And the early `return` means the mismatched grant is treated as a
*repeat*, not a new call — the phantom second bind never happens. `republishCallSource`
re-emits the recovered RID through the same `KindCallSourceUpdate` path the in-call
update uses, so the recorder patches its completed-call webhook and the SSE/TUI
live view light up. The republish fires only on the first fill, so a
heavily-repeated control-channel grant stream never floods the bus.

### How that principle shaped the Go code

The through-line is **identity by physical resource, not by label**. The engine
already learned this lesson once — the duplicate-grant guard keys a logical call
on `(System, talkgroup, timeslot)` rather than frequency, because a call's
frequency can change mid-call. Physical-channel recovery is the mirror image:
when the *label* is the unreliable field, fall back to the physical channel as the
identity of last resort. Both are the same instinct — pick the key the wire can't
corrupt — and both live in `HandleGrant` as ordered fallbacks, cheapest and most
specific match first. It stays consistent with the engine's single-writer rule:
all of this runs on the one `select` goroutine, mutating pool state no one else
writes.

## Where this goes next

[Part 10]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
zooms out from one channel to the whole system: tracking every site the control
channel advertises, folding `KindSiteUpdate` events into a network map, and
following a radio as it roams between sites. If you want the protocol background
first, the [P25 Phase 2]({{ '/reference/p25-phase-2/' | relative_url }}) and
[talkgroup]({{ '/reference/talkgroup/' | relative_url }}) references cover the
compressed-grant and regroup mechanics this post relied on.

## FAQ

**What is the difference between a patch and a supergroup?**
In GopherTrunk they're handled by the same machinery. A patch merges existing
talkgroups so their members interoperate; a dynamic-regroup supergroup is a
system-created address that member talkgroups are folded into. Either way one RF
channel carries traffic for several talkgroups, and the `PatchRegistry` maps the
supergroup address to its members.

**Does following a patch use extra voice SDRs?**
No. A patched call is already a single call on one channel, so it's recorded once
and *attributed* to the supergroup and every member through the registry. There is
no second tune and no second tuner.

**Why match a grant to a call by frequency and timeslot instead of talkgroup?**
Because under heavy compression the talkgroup label on a grant can differ from the
call it belongs to (a mis-aliased compressed grant or a patch remap). A frequency
plus timeslot hosts exactly one transmission, so it's an identity the wire can't
misreport — matching on it recovers the source RID and prevents a duplicate call.

**How does this stop a phantom duplicate call?**
When a source-carrying grant matches an active call by physical channel, the
engine treats it as a repeat of that call and returns without allocating a device.
Without the check, the mismatched talkgroup would look like a new call and bind a
second SDR to a channel already being recorded.

## Series navigation

**Part 9 of 12** · ←
[Part 8: Affiliation Tracking]({{ '/blog/deep-dives/trunking-engine-08-affiliation-tracking/' | relative_url }})
· Next →
[Part 10: Sites, Topology & Roaming]({{ '/blog/deep-dives/trunking-engine-10-sites-topology-roaming/' | relative_url }})
