---
title: "Trunking Engine, Part 5: Priority & Preemption When Calls Outnumber Radios"
description: How GopherTrunk decides which call to drop when active calls exceed voice SDRs — the priority ranking, strict-higher preemption rule, emergency override, and the thrash and starvation it is tuned to avoid.
category: deep-dives
keywords: trunking priority, call preemption, voice sdr contention, effective priority, emergency override, strict higher preemption, trunk recorder priority, gophertrunk preemption, call thrash starvation
tags: [trunking, go, priority, preemption, scheduling, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Trunking Engine"
series_part: 5
---

*Part 5 of **Trunking Engine**. [Part 4]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }})
handed out radios until the pool ran dry. This post is about the decision the
engine makes when it runs dry: with more talkgroups keyed up than SDRs to follow
them, which call keeps its radio and which one loses it?*

> **TL;DR:** Priority is a small, sharp policy in `priority.go`. Each talkgroup
> has an integer priority 1–10 (**lower = higher priority**); unset is treated as
> lowest, and `Emergency` bumps a grant above everything. When no free radio can
> serve an incoming grant, the engine finds the **lowest-priority active call on a
> capable device** and preempts it **only if the incoming grant is strictly higher
> priority**. Equal priority never preempts — that one rule is what stops two
> same-priority talkgroups from endlessly kicking each other off the same radio.

**Key takeaways**

- Priority is `1..10`, lower is better; `0`/unset maps to a sentinel "lowest," and
  `Emergency` maps to `0` — above the highest configurable.
- `EffectivePriority(grant, tg)` collapses the talkgroup priority, the unset
  case, and the emergency override into one comparable integer.
- `CanPreempt` is **strict-higher**: `incoming < active`. Equality holds the
  incumbent — the anti-thrash rule.
- The victim search is **coverage-aware**: only calls on devices that can tune the
  new frequency are eligible, so preemption never frees a radio that can't take
  the grant.

## Cheat sheet

| Concept | Value / rule | Why |
|---|---|---|
| Priority range | `1` (highest) … `10` (lowest) | Trunk-Recorder convention |
| Unset priority (`0`) | treated as `11` (lowest) | a config gap shouldn't win a radio |
| Emergency | `EffectivePriority = 0` | above every configured priority |
| Lockout | dropped before comparison | never grant, never preempt for it |
| Preempt rule | `EffectivePriority(incoming) < EffectivePriority(active)` | strict-higher only |
| Equal priority | **does not** preempt | incumbent holds → no thrash |

## In this post

- **The priority scale** and the two sentinels that make it total.
- **`EffectivePriority`** — one function, one comparable number.
- **`CanPreempt`** — the strict-higher rule and why equality matters.
- **The engine's three-step allocation** and where preemption sits.
- **Thrash and starvation** — the trade-offs the policy is tuned against.

## The priority scale

The convention is borrowed straight from Trunk-Recorder so operators' existing
talkgroup CSVs carry over: an integer `1..10` where **1 is the highest priority
and 10 the lowest**. It reads backwards until you internalize it — think "priority
1 dispatch" — but it's the community standard, so GopherTrunk keeps it.

Two sentinels turn a partial convention into a total order:

```go
// internal/trunking/priority.go
const (
    priorityEmergency = 0  // above the highest configurable (1)
    priorityUnset     = 11 // anything ≥ 10 is "lowest"
)
```

`0` isn't a configurable talkgroup priority — it's reserved for emergencies, which
must outrank even a priority-1 dispatch channel. And a talkgroup with *no*
priority set (the zero value, or a fresh discovered talkgroup) must not
accidentally beat a configured one, so it collapses to `11` — below priority 10.
Now every call has a comparable rank and there are no ties between "unset" and
"lowest configured."

## `EffectivePriority`: one comparable number

All of that folds into a single function the rest of the engine calls:

```go
// internal/trunking/priority.go
func EffectivePriority(g Grant, tg *TalkGroup) int {
    if g.Emergency {
        return priorityEmergency // 0 — wins outright
    }
    if tg == nil || tg.Priority <= 0 {
        return priorityUnset     // 11 — loses to anything configured
    }
    return tg.Priority
}
```

The order of the checks *is* the policy. Emergency is tested first, so an
emergency grant on an otherwise-unprioritized talkgroup still ranks `0`. A `nil`
talkgroup (an ID we've never catalogued) or a non-positive priority ranks `11`.
Otherwise the configured `1..10` is used verbatim. Because it returns a plain
`int` where lower wins, the voice pool can find a preemption victim with a simple
"largest `EffectivePriority`" scan — `LowestPriorityActiveForFrequency` from
[Part 4]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }}).

<figure class="lab-figure">
<svg viewBox="0 0 680 132" width="680" height="132" role="img" aria-label="A priority number line from 0 to 11: emergency at 0 is highest, priorities 1 through 10 are configurable with 1 highest, and unset collapses to 11 the lowest">
  <line x1="40" y1="66" x2="640" y2="66" stroke="currentColor"/>
  <polygon points="640,62 650,66 640,70" fill="currentColor"/>
  <line x1="70" y1="60" x2="70" y2="72" stroke="var(--accent)"/>
  <text x="70" y="50" text-anchor="middle" fill="var(--accent)" font-size="11">0</text>
  <text x="70" y="90" text-anchor="middle" fill="var(--accent)" font-size="9">emergency</text>
  <text x="70" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="8">highest</text>
  <g stroke="currentColor">
    <line x1="150" y1="60" x2="150" y2="72"/><line x1="230" y1="60" x2="230" y2="72"/>
    <line x1="310" y1="60" x2="310" y2="72"/><line x1="450" y1="60" x2="450" y2="72"/>
  </g>
  <text x="150" y="50" text-anchor="middle" fill="currentColor" font-size="11">1</text>
  <text x="230" y="50" text-anchor="middle" fill="currentColor" font-size="11">2</text>
  <text x="310" y="50" text-anchor="middle" fill="currentColor" font-size="11">…</text>
  <text x="450" y="50" text-anchor="middle" fill="currentColor" font-size="11">10</text>
  <text x="300" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">configurable 1..10 — lower wins</text>
  <line x1="580" y1="60" x2="580" y2="72" stroke="var(--fg-muted)"/>
  <text x="580" y="50" text-anchor="middle" fill="var(--fg-muted)" font-size="11">11</text>
  <text x="580" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">unset</text>
  <text x="580" y="102" text-anchor="middle" fill="var(--fg-muted)" font-size="8">lowest</text>
</svg>
<figcaption>Emergency (0) and unset (11) are sentinels bracketing the configurable 1..10 band, so every call has a total, comparable rank.</figcaption>
</figure>

## `CanPreempt`: the strict-higher rule

Preemption is a two-line predicate, and the comparison operator is the entire
design:

```go
// internal/trunking/priority.go
func CanPreempt(active Grant, activeTG *TalkGroup, incoming Grant, incomingTG *TalkGroup) bool {
    if incomingTG != nil && incomingTG.Lockout {
        return false // a locked-out grant never preempts (defensive)
    }
    return EffectivePriority(incoming, incomingTG) < EffectivePriority(active, activeTG)
}
```

`<`, not `<=`. A new grant preempts an active call only when it is **strictly
higher** priority. Equal priority does *not* preempt, and that is deliberate: a
stable call holds its radio against same-priority grants. Without the strictness,
two priority-3 talkgroups keying up in alternation would take turns evicting each
other, and neither would ever record a coherent call — pure thrash. Strict-higher
makes the incumbent win ties, so an in-progress call is stable once it has a
radio.

The lockout short-circuit is defensive belt-and-suspenders: the engine already
drops locked-out grants earlier in dispatch
([Part 6]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})),
but `CanPreempt` refuses one anyway so callers can compose it freely without
re-checking lockout.

## Where preemption sits in dispatch

Priority only comes into play after the pool has failed to find a free radio. The
tail of `HandleGrant` is a three-step ladder:

```go
// internal/trunking/engine.go (shape) — after dedup/backfill
// 1) free capable device? allocate.
if free := e.pool.FindFreeForFrequency(g.FrequencyHz); free != nil {
    e.startCall(free, g, tg)
    return
}
// 2) no free device can serve this frequency — find a preemptable victim
//    *on a device that can tune the grant*.
victim := e.pool.LowestPriorityActiveForFrequency(g.FrequencyHz)
if victim == nil { /* coverage gap / empty pool — Part 4 diagnostics */ return }
// 3) preempt only if strictly higher priority.
if !CanPreempt(victim.Grant, victim.Talkgroup, g, tg) {
    e.log.Info("no voice device available for grant", "grant", g.String())
    return
}
e.endCall(victim, EndReasonPreempted)
e.startCall(victim.Device, g, tg)
```

Step 2 is coverage-aware for the reason from Part 4: the victim must be on a
device that can actually tune the incoming frequency, or preempting it frees a
radio that then can't bind the grant — a call ended for nothing. When preemption
does fire, the victim ends with `EndReasonPreempted`, a distinct end reason so the
operator's call log shows *why* a call was cut short — a higher-priority grant
took the radio, not a decode failure or a timeout. And if the incoming grant
*isn't* higher priority, nothing happens: the grant is simply not followed this
time, and the engine logs it at INFO. A repeat of that same grant a moment later
gets another shot once a radio frees up.

<figure class="lab-figure">
<svg viewBox="0 0 680 176" width="680" height="176" role="img" aria-label="Preemption decision: an incoming priority-2 grant finds all radios busy, the lowest-priority active call is priority-8, strict-higher holds so the priority-8 call is preempted and its radio reassigned to the priority-2 grant">
  <rect x="14" y="20" width="150" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="89" y="38" text-anchor="middle" fill="var(--accent)" font-size="11">incoming grant</text>
  <text x="89" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">priority 2</text>
  <line x1="164" y1="40" x2="214" y2="40" stroke="currentColor"/>
  <polygon points="214,36 224,40 214,44" fill="currentColor"/>
  <rect x="226" y="16" width="180" height="48" rx="6" fill="none" stroke="currentColor"/>
  <text x="316" y="36" text-anchor="middle" fill="currentColor" font-size="11">all capable radios busy</text>
  <text x="316" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">lowest active = priority 8</text>
  <line x1="316" y1="64" x2="316" y2="92" stroke="currentColor"/>
  <polygon points="312,92 316,102 320,92" fill="currentColor"/>
  <rect x="196" y="104" width="240" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="316" y="121" text-anchor="middle" fill="var(--accent)" font-size="10">CanPreempt? 2 &lt; 8 → yes</text>
  <text x="316" y="133" text-anchor="middle" fill="var(--fg-muted)" font-size="9">strict-higher</text>
  <line x1="436" y1="121" x2="486" y2="121" stroke="var(--accent)"/>
  <polygon points="486,117 496,121 486,125" fill="var(--accent)"/>
  <rect x="498" y="98" width="170" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="583" y="116" text-anchor="middle" fill="var(--accent)" font-size="10">end priority-8 call</text>
  <text x="583" y="130" text-anchor="middle" fill="var(--fg-muted)" font-size="9">EndReasonPreempted → rebind</text>
  <text x="316" y="164" text-anchor="middle" fill="var(--fg-muted)" font-size="10">if incoming were priority 8 (equal) or lower → grant not followed, incumbent holds</text>
</svg>
<figcaption>Preemption fires only on a strictly-higher grant. An equal-or-lower grant leaves the incumbent on its radio and is simply skipped this cycle.</figcaption>
</figure>

## Thrash and starvation: the trade-offs

### How that principle shaped the Go code

This policy is small on purpose, and every choice trades one failure mode for
another:

- **Anti-thrash (strict `<`).** The whole reason for strict-higher is to make the
  incumbent stable. The cost is that two equal-priority calls are served
  first-come — if radio contention is high and everything is the same priority,
  later grants of equal priority just wait. That's acceptable: a coherent
  recording of one call beats two shredded half-recordings of both.
- **Starvation is bounded, not eliminated.** A permanently-busy priority-1
  talkgroup can keep a lower-priority call off the air indefinitely — that's the
  *point* of priority. Because there's no aging, a priority-10 call never
  "earns" its way past a priority-1 hog; the operator's lever is the priority
  numbers themselves and the number of radios. The engine doesn't try to be fair
  across priorities; it tries to honor the priorities it's given.
- **Emergency always wins.** By mapping `Emergency` to `0`, an emergency grant
  preempts even a priority-1 call. This is the one case where a *lower*-priority
  talkgroup's grant can evict a higher-priority one — the emergency flag on the
  grant, not the talkgroup's configured rank, decides.
- **No partial state.** Preemption is `endCall` then `startCall` — the victim is
  fully torn down (its `EndReasonPreempted` published, its radio released) before
  the new call binds. There's no half-migrated device, which keeps the
  single-writer invariant from
  [Part 1]({{ '/blog/deep-dives/trunking-engine-01-grant-to-call/' | relative_url }})
  intact: the loop is still the only mutator, doing two ordered operations.

The result is a policy an operator can reason about from the CSV alone: lower
number wins, ties hold, emergencies jump the queue, and the only way to follow
more simultaneous calls is more radios.

## Where this goes next

Priority reads a talkgroup's `Priority` and `Lockout` fields — which means it
depends entirely on the talkgroup database being right.
[Part 6]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})
opens that database: alias lookup, hold and lockout, and the scan modes that
decide which grants are even eligible for a radio before priority ever runs —
including flipping scan mode live from the cockpit. After that,
[Part 7]({{ '/blog/deep-dives/trunking-engine-07-source-rid-recovery/' | relative_url }})
returns to the source-less-grant problem from Part 3.

## FAQ

**Why is priority 1 higher than priority 10?**
It's the Trunk-Recorder / RadioReference convention — lower number, higher
priority, like a priority-1 dispatch channel. GopherTrunk follows it so operators'
existing talkgroup CSVs work unchanged.

**Why doesn't an equal-priority grant preempt an active call?**
To prevent thrash. `CanPreempt` uses strict `<`, so a stable in-progress call
holds its radio against same-priority grants. If equality preempted, two
same-priority talkgroups could take turns evicting each other and neither would
record a coherent call.

**What happens to a call that gets preempted?**
It ends with `EndReasonPreempted` — a distinct reason so the call log shows it was
cut short by a higher-priority grant, not by a decode error or timeout. Its radio
is released and immediately rebound to the incoming grant.

**How does an emergency call get a radio when everything is busy?**
`Emergency` maps to `EffectivePriority` 0, above every configured priority, so an
emergency grant is strictly higher than any normal active call and preempts it —
even a priority-1 call. It's the one case where the grant's flag, not the
talkgroup's configured rank, decides.

**Can a low-priority call be starved forever?**
Yes, by design — there's no aging. A continuously-busy high-priority talkgroup
can hold a radio indefinitely and keep lower-priority calls off the air. The
operator's controls are the priority numbers and the number of voice SDRs; the
engine honors the priorities it's configured with rather than enforcing
cross-priority fairness.

## Series navigation

**Part 5 of 12** · ←
[Part 4: The Voice Pool]({{ '/blog/deep-dives/trunking-engine-04-voice-pool/' | relative_url }})
· Next →
[Part 6: Talkgroups, Aliases & Scan Modes]({{ '/blog/deep-dives/trunking-engine-06-talkgroups-scan-modes/' | relative_url }})
