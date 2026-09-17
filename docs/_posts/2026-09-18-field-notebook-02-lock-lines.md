---
title: "The Field Notebook, Part 2: Lock Lines — Locked, Lost & Transitions"
description: An operator's reading of GopherTrunk's control-channel lock family — cc-hunt trying and locked, the per-protocol cc locked lines, the cc.lost events behind a re-hunt, cchunt hunt failed with its IQ diagnosis, the camped idle state, and the transition counter that turns "it drops sometimes" into a number.
category: tutorials
keywords: control channel lock log, cc locked cc lost, cchunt hunt failed, control channel transitions, sdr scanner re-hunt loop, camped conventional idle, tetra cc locked mcc mnc, park until unlocked, read debug log, gophertrunk field notebook
tags: [field-notebook, logs, control-channel, hunt, tetra, troubleshooting, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 2
---

*Part 2 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time.
[Part 1]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }})
read the startup block up to `ccdecoder: digital down-converter configured`,
where the hunt begins. This part reads the lock family — the lines the
`cchunt.Supervisor` and the per-protocol decoders print as a system moves
between hunting, locked, lost and camped. It is the heartbeat of a trunked
rig, and the difference between "the CC is fine" and "the CC is churning"
is one counter you can read at a glance.*

> **TL;DR:** The hunter prints `cc-hunt: trying system= freq_hz=` per
> candidate and `cc-hunt: locked system= freq_hz= nac=` on success
> (`internal/trunking/cchunt.go`), but it never *decides* the lock — the
> IQ-domain decoder does, publishing `cc.locked` which surfaces as
> `control channel locked nac= freq=` (P25), `tetra cc locked freq= mcc=
> mnc= la=`, `dmr cc locked`, and so on. A `cc.locked` **parks** the system
> (`parkUntilUnlocked`); a `cc.lost` un-parks it and re-hunting resumes. A
> failed round logs `cchunt: hunt failed — no control-channel lock` with an
> IQ-health `diagnosis`, then backs off. A conventional/DMO system that
> finds nothing is `camped on conventional channel — idle`, not failed. The
> number that turns churn into a measurement is
> `control_channel_transitions_total{system,event}`; identity changes on a
> locked TETRA CC need `lockIdentityConfirmThreshold` = 2 agreeing decodes,
> so one mis-corrected BSCH can't flip your MCC/MNC.

**Key takeaways**

- **Two layers, two lines.** `cc-hunt: locked` is the supervisor tuning a
  candidate; the per-protocol `cc locked` line is the decoder confirming it.
  The supervisor never fabricates a lock — it only orchestrates.
- **Lock means park, not re-hunt.** After `cc.locked` the system is parked
  until `cc.lost`; re-hunting a locked edge-triggered decoder would exhaust
  the dwell and falsely report failure while calls decode.
- **A failed hunt explains itself.** `cchunt: hunt failed` carries
  `iq_power_dbfs`, `iq_clip_ratio`, `pipeline_active` and a one-line
  `diagnosis` — "no IQ", "front-end overload", "signal present but no lock".
- **`control_channel_transitions_total` is the churn meter.** A system
  that is *technically* locked but flipping through dozens of transitions an
  hour is a weak install, not a healthy one — read the counter, not the vibe.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Candidate + lock | `cc-hunt: trying` / `cc-hunt: locked` per hunt | `internal/trunking/cchunt.go` (`Hunter.Hunt`) |
| The decoder's lock | per-protocol `cc locked`, publishes `cc.locked` | `internal/radio/*/control.go` (`maybeLock`) |
| Park on lock | stop re-hunting a working system | `internal/scanner/cchunt/supervisor.go` (`parkUntilUnlocked`) |
| Lock loss | `cc.lost` → leave `StateLocked`, re-hunt | `control.go` (`MarkLost`), `supervisor.go` (`recordLost`) |
| Failure + diagnosis | `hunt failed` + IQ health | `supervisor.go` (`markFailed`, `noIQDiagnosis`) |
| Camped idle | conventional/DMO idle is not a failure | `supervisor.go` (`markCamped`), `Protocol.CampsWhenIdle` |
| Identity gate | 2 agreeing BSCH before MCC/MNC changes | `internal/radio/tetra/control.go` (`lockIdentityConfirmThreshold`) |
| Churn meter | lock/lost transitions per system | `internal/metrics/prom.go` (`control_channel_transitions_total`) |

## In this post

- **What this line family is telling you** — the two-layer lock handshake.
- **What healthy looks like** — trying, locked, and a stable park.
- **Lost, and the re-hunt** — `cc.lost`, `MarkLost`, and who drives it.
- **When the hunt fails** — the diagnosis field and the camped exception.
- **When it doesn't look like that** — symptom → cause → where to read.

## What this line family is telling you

The lock family has a strict division of labour, and reading it correctly
means knowing which line is a *claim* and which is a *confirmation*. The
`cchunt.Supervisor` owns the control tuner and walks each system's candidate
frequencies; the `Hunter` logs `cc-hunt: trying system= freq_hz=` before each
`SetCenterFreq`, publishes a `HuntProgress` so the cockpit can render "trying
(2/3)" without scraping logs, then waits its dwell (`dwell_ms`, default 3000)
for a `cc.locked` event *on that frequency*. It does not decode. The
[supervisor deep dive]({{ '/blog/deep-dives/the-hunt-06-control-channel-hunting/' | relative_url }})
puts it plainly: without the IQ-domain decoder feeding `cc.locked`, the
hunter always exhausts its candidates and reports failed — that's by design.

The confirmation comes from the decoder. Each protocol's `control.go` has a
`maybeLock` that publishes `events.KindCCLocked` and logs its own line:
`control channel locked nac= freq= rot= delta=` (P25 Phase 1),
`tetra cc locked freq= mcc= mnc= la=`, `dmr cc locked freq= cc= sysid=`,
`nxdn cc locked`, and so on. That `cc.locked` is what the hunter's
`waitForLock` was waiting for; the supervisor's `listen` goroutine turns it
into `StateLocked` and calls `parkUntilUnlocked` — a 250 ms ticker that
returns the moment the state leaves `StateLocked`. So a locked system stops
being hunted entirely, which matters because TETRA's `maybeLock` is
edge-triggered and never re-affirms a steady lock; re-hunting it would burn
the dwell and report a false failure while it decodes calls happily.

<figure class="lab-figure">
<svg viewBox="0 0 680 250" width="680" height="250" role="img" aria-label="The two-layer lock handshake. On the left the cchunt supervisor loop tunes a candidate and logs cc-hunt trying, then waits. In the middle the IQ-domain decoder finds frame sync and publishes cc.locked, logging its own per-protocol cc locked line. On the right the supervisor observes cc.locked, transitions to StateLocked and parks. Below, a cc.lost from the decoder's watchdog un-parks the system and the loop resumes hunting; each locked and lost edge increments the control_channel_transitions counter.">
  <rect x="14" y="40" width="150" height="70" rx="6" fill="none" stroke="currentColor"/>
  <text x="89" y="60" text-anchor="middle" fill="currentColor" font-size="10">cchunt supervisor</text>
  <text x="89" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cc-hunt: trying</text>
  <text x="89" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">freq_hz= (2/3)</text>
  <text x="89" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">…waits its dwell</text>
  <rect x="264" y="40" width="150" height="70" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="339" y="60" text-anchor="middle" fill="var(--accent)" font-size="10">IQ decoder</text>
  <text x="339" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="9">frame sync found</text>
  <text x="339" y="90" text-anchor="middle" fill="var(--accent)" font-size="9">cc locked mcc= mnc=</text>
  <text x="339" y="104" text-anchor="middle" fill="var(--fg-muted)" font-size="9">publishes cc.locked</text>
  <rect x="514" y="40" width="152" height="70" rx="6" fill="none" stroke="currentColor"/>
  <text x="590" y="60" text-anchor="middle" fill="currentColor" font-size="10">supervisor observes</text>
  <text x="590" y="76" text-anchor="middle" fill="var(--fg-muted)" font-size="9">StateLocked</text>
  <text x="590" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">parkUntilUnlocked</text>
  <line x1="164" y1="75" x2="264" y2="75" stroke="var(--fg-muted)"/>
  <text x="214" y="70" text-anchor="middle" fill="var(--fg-muted)" font-size="8">tune</text>
  <line x1="414" y1="75" x2="514" y2="75" stroke="var(--accent)"/>
  <text x="464" y="70" text-anchor="middle" fill="var(--accent)" font-size="8">cc.locked (bus)</text>
  <line x1="590" y1="110" x2="590" y2="160" stroke="var(--fg-muted)"/>
  <text x="590" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">carrier goes silent</text>
  <line x1="339" y1="110" x2="339" y2="160" stroke="var(--fg-muted)"/>
  <text x="339" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">watchdog: MarkLost</text>
  <rect x="264" y="164" width="326" height="34" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="427" y="185" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cc.lost → recordLost → StateHunting → the loop re-hunts</text>
  <line x1="264" y1="181" x2="164" y2="90" stroke="var(--fg-muted)" stroke-dasharray="3 3"/>
  <text x="340" y="222" text-anchor="middle" fill="currentColor" font-size="10">every locked / lost edge → control_channel_transitions_total{system,event}</text>
</svg>
<figcaption>The supervisor tunes and parks; the decoder decides. A cc.lost from the decoder's own watchdog un-parks the system, and each edge in either direction increments the transition counter that measures churn.</figcaption>
</figure>

## What healthy looks like

A P25 system locking, then a TETRA system locking on a different rig, at
`log.level: info`:

```
INF cc-hunt: trying system=Metro-P25 freq_hz=857262500
INF control channel locked nac=659 freq=857262500 rot=0 delta=0.02
INF cc-hunt: locked system=Metro-P25 freq_hz=857262500 nac=659
```

Read the order. `cc-hunt: trying` is the supervisor pointing the tuner;
`control channel locked … nac=659` is the P25 decoder finding frame sync and
reading the Network Access Code; `cc-hunt: locked` is the supervisor
observing the matching `cc.locked` and parking. After that, silence on the
lock family is *healthy* — a parked system logs nothing until it loses lock.
TETRA's line carries more identity:

```
INF tetra cc locked freq=467912500 mcc=250 mnc=13 la=1 system=250_013
```

`mcc`/`mnc` are the Mobile Network Identity; `la` is the location area. Once
this prints, a steady lock is silent — but the identity fields are protected.
An identity *change* on an already-locked channel needs
`lockIdentityConfirmThreshold` (2) consecutive agreeing BSCH decodes before
it is adopted, logging `tetra: cc identity change pending confirmation` in
the meantime. This exists because a single CRC-passing but FEC-mis-corrected
BSCH carries a valid-but-wrong MCC/MNC; a 19 Aug field log had four bogus
"tetra cc locked mcc=996" flaps from exactly that before the gate landed.

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `cc-hunt: trying freq_hz=` | the candidate the supervisor is tuning | cycles then stops on a lock | cycles forever — no candidate locks |
| `control channel locked nac=` / `cc locked` | the decoder's confirmed lock | fires once, then silence | never fires — see `hunt failed` below |
| `tetra cc locked mcc= mnc= la=` | the decoded cell identity | matches the site you expect | `identity change pending` repeating, or a wrong MCC |
| `nac=` / `cc=` / `sysid=` | the protocol's network id | matches RadioReference | wrong id — you locked another site |
| `cc-hunt: locked` (absence after lock) | parked, not re-hunting | no further lock lines while decoding | a re-`trying` line means it lost lock |

## Lost, and the re-hunt

A lock ends when the decoder publishes `cc.lost` (`events.KindCCLost`). Every
protocol's `control.go` has a `MarkLost` that resets the locked flag and
publishes it; the supervisor's `recordLost` flips the system back to
`StateHunting`, `parkUntilUnlocked` returns, and the next loop iteration
hunts it again. Nothing in the supervisor decides loss — the decoder owns its
own watchdog, and the mechanism differs by protocol. TETRA runs
`ControlChannel.CheckStale`, which calls `MarkLost` when a locked channel has
decoded nothing for `tetraLockStaleTimeout` (5 s, about five missed
multiframes at the ~1 s BSCH cadence); a healthy carrier never trips it.
[TETRA End to End Part 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})
walks a real one-hour session with eleven of these, and its escape from the
"locked but deaf" state — the payload-drought resync that ends in
`tetra: payload drought persists across resyncs … declaring lock lost` — is
the other path to a re-hunt.

The re-hunt is the *point* of `cc.lost`, but a rig that re-hunts often is a
weak rig. That is what `control_channel_transitions_total{system,event}`
measures: it increments on every `cc.locked` (`event="locked"`) and every
`cc.lost` (`event="lost"`), so a system flipping through dozens of
transitions an hour is churning under poor SNR even if it reads "locked" right
now.
[Running It For Real Part 4]({{ '/blog/deep-dives/running-it-for-real-04-metrics-that-matter/' | relative_url }})
built the counter for exactly this — a system that is technically locked but
churning is the failure a lock/unlock gauge alone would hide. Correlate a
transition burst against the [carrier-offset WARN]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }})
and the [TETRA decode-status line]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
before blaming the software.

## When the hunt fails

When a trunked system's hunt round exhausts its candidates without a lock,
`markFailed` doubles the backoff (from `backoff_ms` 5000, capped at
`max_backoff_ms` 60000) and logs the line operators actually paste into bug
reports — a self-triaging WARN with the control SDR's IQ health attached:

```
WRN cchunt: hunt failed — no control-channel lock system=Metro-P25 backoff_ms=5000 iq_observed=true iq_samples=4915200 iq_power_dbfs=-52.3 iq_clip_ratio=0 iq_dc_ratio_db=-28.1 pipeline_active=true diagnosis="signal present but no control-channel lock — verify the control-channel frequency is correct and current, set the right ppm, and confirm the system is active"
```

The `diagnosis` field is the whole value. It reuses the same thresholds the
live decoder logs fire on, so the two never disagree, and it distinguishes
the failure classes: `no IQ observed from the control SDR` (a dead handle or
unbound driver — the USB fault from Part 1), `front-end overload (ADC
clipping) — reduce gain` (`iq_clip_ratio` over threshold), `an on-channel DC
spike dominates` (issue #402), and the "signal present but no lock" above.
`iq_observed=false` is the loudest of them: it means the tuner is not
delivering samples at all, and no frequency or ppm change will help until
`sdr doctor` says the driver is bound.

The exception is a conventional or direct-mode system. `Protocol.CampsWhenIdle`
is true for conventional DMR, DMR Tier I and TETRA DMO — channels that
legitimately sit silent between transmissions — so a hunt round that ends
without a lock is not a failure. `markCamped` keeps the backoff at its floor,
publishes no `KindHuntFailed`, and logs once on the transition:

```
INF cchunt: camped on conventional channel — idle, waiting for traffic system=Site-DMO
```

That line fires *once* per transition into camped (issue #1036 made it
transition-only, so a quiet night doesn't spam ~25 lines a second). It is the
healthy idle state, not an error — [Cookbook Part 5]({{ '/blog/tutorials/operator-cookbook-05-tetra-dmo/' | relative_url }})'s
DMO recipe camps on it deliberately, and the DMO status line in
[Part 6]({{ '/blog/tutorials/field-notebook-06-dmo-status-line/' | relative_url }})
reads what happens once someone keys up.

### How this line shapes operator practice

- **Match the identity, not just the lock.** A `cc locked` line with the
  wrong `nac`/`mcc`/`sysid` means you locked a neighbour. Check it against
  RadioReference before trusting anything downstream.
- **Read the churn counter before believing "it's locked".**
  `control_channel_transitions_total` climbing is a weak install; a stable
  count is a stable lock.
- **`iq_observed=false` is a hardware verdict.** No frequency, ppm or
  antenna change fixes a tuner that is not streaming — go to `sdr doctor`.
- **`camped … idle` is not `hunt failed`.** On a conventional or DMO
  system, camping is the working state; the fault, if any, is upstream RF.

## Where this goes next

A TETRA lock that reads `bsch_ok` climbing but `sch_pdus=0` is "locked but
deaf" — locked on the lock family, dead on the decode family.
[Part 3]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
reads the `tetra: decode status` line field by field — `sb_bursts`,
`bsch_ok`/`bsch_fail`, `sch_pdus`/`sch_pdus_fail`, `frag_abandons` — and shows
why an AFC alias bucket can keep the lock alive while every payload block
fails, and how the signal-time resync and payload-drought watchdogs recover.

## FAQ

**Why are there two "locked" log lines for one control channel?**
Two layers announce it. `cc-hunt: locked` is the `cchunt.Supervisor` noting
that a candidate it tuned produced a lock; the per-protocol `control channel
locked` / `tetra cc locked` / `dmr cc locked` line is the IQ decoder that
actually found frame sync and read the network id. The decoder's line is the
authoritative one; the supervisor only orchestrates.

**What does "cchunt: hunt failed" with a diagnosis field mean?**
A trunked system's hunt round found no lock. The `diagnosis` reuses the live
decoder's IQ thresholds to name the cause: no IQ observed (dead tuner),
front-end overload, an on-channel DC spike, or "signal present but no lock"
(wrong or stale frequency). Read the field — it distinguishes a dead radio
from a wrong frequency.

**Why does my conventional system log "camped … idle" instead of locking?**
Conventional and direct-mode channels sit silent between transmissions, so a
hunt round with no lock is not a failure. `Protocol.CampsWhenIdle` systems
camp on the frequency and re-dwell promptly, logging the camped line once per
transition. The channel decodes the next transmission that arrives; nothing is
wrong.

**Why did my TETRA site briefly show a wrong MCC/MNC?**
On older builds, one CRC-passing but FEC-mis-corrected BSCH could rewrite the
locked identity. GopherTrunk now requires `lockIdentityConfirmThreshold` (2)
consecutive agreeing BSCH decodes before an identity change is adopted,
logging `tetra: cc identity change pending confirmation` meanwhile — so a
single bad burst can no longer flip your site.

**How do I tell a healthy lock from a churning one?**
Watch `control_channel_transitions_total{system,event}`. A healthy lock
increments it once (`locked`) and holds; a system flipping between locked and
lost dozens of times an hour is churning under poor SNR, which reads as
"locked" on a plain gauge but is really a weak-signal problem to fix at the
antenna.

## Series navigation

**Part 2 of 14** · ←
[Part 1: Startup — The Lines Before the First Lock]({{ '/blog/tutorials/field-notebook-01-startup-lines/' | relative_url }})
· Next →
[Part 3: The TETRA Decode-Status Line]({{ '/blog/tutorials/field-notebook-03-tetra-decode-status/' | relative_url }})
