---
title: "The Field Notebook, Part 3: The TETRA Decode-Status Line"
description: An operator's field-by-field reading of GopherTrunk's tetra decode status line — sb_bursts, bsch_ok and bsch_fail, sch_pdus and sch_pdus_fail, frag_abandons — the "locked but deaf" signature where bsch_ok climbs while sch_pdus stays zero, and the signal-time resync and payload-drought watchdogs that recover it.
category: tutorials
keywords: tetra decode status log, bsch_ok bsch_fail, sch_pdus zero, tetra locked but deaf, afc alias bucket, dsp resync decode drought, frag_abandons tetra, tetra sync burst, read debug log, gophertrunk field notebook
tags: [field-notebook, logs, tetra, decode-status, afc, troubleshooting, tutorial]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Field Notebook"
series_part: 3
---

*Part 3 of **The Field Notebook**, a 14-part operator's tutorial that reads
GopherTrunk's `debug.log` one line family at a time.
[Part 2]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }})
read the lock family and left one loose end: a TETRA control channel can be
locked on paper — `tetra cc locked` printed, the stale watchdog quiet — while
decoding no useful traffic at all. This part reads the one line that tells
you which of those you have: the throttled `tetra: decode status` line. It is
a debug-only census of the TETRA decode chain, and it is where "the TETRA
site won't decode" becomes a specific, localised fault.*

> **TL;DR:** With `log.level: debug`, `internal/scanner/ccdecoder/pipelines.go`
> emits `tetra: decode status` every `tetraStatusInterval` (5 s, override
> `tetra_status_interval_secs`) with `locked`, `carrier_off_hz`, `baud`,
> `baud_dev_pct`, `sb_bursts`, `bsch_ok`, `bsch_fail`, `sysinfo`, `sch_pdus`,
> `sch_pdus_fail`, `frag_abandons`, `grants` and `colour_code`. The counters
> come straight from `ControlChannel.DrainStats` and only move at debug
> level. The signature to internalise: **`bsch_ok` climbing while `sch_pdus`
> stays 0 is "locked but deaf"** — a wrong ±f_sym/4 = 4500 Hz AFC alias
> bucket, where the heavily-coded BSCH survives the residual but every SCH
> payload fails CRC. Two watchdogs recover it, both measured in *signal
> time* (immune to CPU starvation): `checkResync` (a 1.5 s decode drought)
> and `checkPayloadResync` (12 s of BSCH-alive/SCH-dead, escalating to
> `MarkLost` after 3 tries). `frag_abandons` climbing is the continuity guard
> *working*, not a parse bug.

**Key takeaways**

- **The line is a census, not a success log.** Every branch is counted,
  including the failing ones, so a ratio like `bsch_ok`/`bsch_fail` grades
  the channel — a lone success count would flatter it.
- **`bsch_ok` up, `sch_pdus` at 0 is the signature to memorise.** Sync
  survives, payload dies: a wrong AFC alias bucket. The lock looks healthy
  because BSCH gets per-burst correction and heavy FEC.
- **The resyncs are in signal time.** `tetra: dsp resync (signal-time decode
  drought)` counts processed samples, not wall clock, so a descheduled
  goroutine under call load never fires a destructive reset falsely.
- **`frag_abandons` is the guard working.** `abandoning TM-SDU fragment
  reassembly` is the continuity check refusing to splice across a lost
  block; each abandon costs one broadcast cycle, not a decode.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The status line | 5 s census of the TETRA decode chain | `internal/scanner/ccdecoder/pipelines.go` (`maybeLogStatus`) |
| The counters | `sb_bursts`, `bsch_ok/fail`, `sch_pdus/fail`, `frag_abandons` | `internal/radio/tetra/control.go` (`Stats`, `DrainStats`) |
| Cadence override | change the 5 s window | `tetra_status_interval_secs` (`TETRAStatusIntervalSecs`) |
| Decode-drought resync | reset timing/AFC on 1.5 s of no decode | `pipelines.go` (`checkResync`, `tetraResyncTimeout`) |
| Payload-drought resync | 12 s BSCH-alive/SCH-dead → reset → MarkLost | `pipelines.go` (`checkPayloadResync`, `tetraPayloadResyncTimeout`) |
| The AFC alias trap | ±f_sym/4 = 4500 Hz wrong bucket | `internal/radio/tetra/receiver/afc.go`, [AFC alias traps]({{ '/reference/afc-alias-traps/' | relative_url }}) |
| Fragment continuity | drop a chain across a lost block | `control.go` (`abandonFragment`, `FragAbandons`) |

## In this post

- **What this line is telling you** — the census and its ratios.
- **What healthy looks like** — a decoding TETRA CC, field by field.
- **The "locked but deaf" signature** — `bsch_ok` up, `sch_pdus` at 0.
- **The two resync watchdogs** — signal-time, and what they print.
- **When it doesn't look like that** — symptom → cause → where to read.

## What this line is telling you

The `tetra: decode status` line is a deliberate census of a decode chain that
either survives or dies at each boundary — exactly the "count every branch"
design [From Spec to Shipping Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
argues for. It is gated on debug: the counters in `ControlChannel.Stats` only
move when the logger is at debug level (`addStat` is a no-op otherwise), and
`DrainStats` resets them each interval, so every line is a per-window
snapshot. `maybeLogStatus` throttles it to `tetraStatusInterval` (5 s), which
`tetra_status_interval_secs` overrides.

Read it as ratios, not absolutes. `sb_bursts` counts synchronisation-burst
candidates entering the SB decoder; `bsch_ok` and `bsch_fail` are how many of
those recovered CRC-clean under any rotation versus none — the four-rotation
correlator throws off false candidates on noise, so a high `bsch_fail` beside
a healthy `bsch_ok` is normal, and the pair grades the sync layer.
`sch_pdus` counts CRC-clean signalling blocks (SCH/F, SCH/HD) off a real NCDB
slot; `sch_pdus_fail` counts AACH-confirmed control slots that yielded no
clean block. `sysinfo` is decoded SYSINFO PDUs, `grants` is voice grants
published, `frag_abandons` is fragment chains dropped by the continuity
guard, and `colour_code` is the low 6 bits of the confirmed extended colour
code. The two carrier fields — `carrier_off_hz` (the AFC's residual, rounded
to 0.1 Hz) and `baud`/`baud_dev_pct` (symbols over wall time, the same figure
siglab reports) — tell you whether the DSP front of the chain is even on
frequency.

<figure class="lab-figure">
<svg viewBox="0 0 680 240" width="680" height="240" role="img" aria-label="The TETRA decode chain as four stages left to right, each with the status-line counter that measures it. Stage one, the SB decoder, counts sb_bursts entering and splits into bsch_ok and bsch_fail. Stage two, SCH payload on NCDB slots, splits into sch_pdus and sch_pdus_fail. Stage three, fragment reassembly, counts frag_abandons when the continuity guard drops a chain. Stage four publishes grants. A callout marks the locked-but-deaf signature: bsch_ok climbing while sch_pdus stays zero, caused by a wrong AFC alias bucket at the front of the chain.">
  <rect x="12" y="46" width="120" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="72" y="66" text-anchor="middle" fill="currentColor" font-size="10">SB decoder</text>
  <text x="72" y="82" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sb_bursts →</text>
  <text x="72" y="96" text-anchor="middle" fill="var(--accent)" font-size="9">bsch_ok / bsch_fail</text>
  <rect x="180" y="46" width="120" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="240" y="66" text-anchor="middle" fill="currentColor" font-size="10">SCH payload</text>
  <text x="240" y="82" text-anchor="middle" fill="var(--accent)" font-size="9">sch_pdus /</text>
  <text x="240" y="96" text-anchor="middle" fill="var(--accent)" font-size="9">sch_pdus_fail</text>
  <rect x="348" y="46" width="120" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="408" y="66" text-anchor="middle" fill="currentColor" font-size="10">reassembly</text>
  <text x="408" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="9">frag_abandons</text>
  <rect x="516" y="46" width="120" height="60" rx="6" fill="none" stroke="currentColor"/>
  <text x="576" y="66" text-anchor="middle" fill="currentColor" font-size="10">grant</text>
  <text x="576" y="86" text-anchor="middle" fill="var(--fg-muted)" font-size="9">grants</text>
  <line x1="132" y1="76" x2="180" y2="76" stroke="currentColor"/>
  <line x1="300" y1="76" x2="348" y2="76" stroke="currentColor"/>
  <line x1="468" y1="76" x2="516" y2="76" stroke="currentColor"/>
  <rect x="12" y="150" width="288" height="66" rx="6" fill="none" stroke="var(--accent)" stroke-dasharray="4 3"/>
  <text x="156" y="170" text-anchor="middle" fill="var(--accent)" font-size="10">"locked but deaf"</text>
  <text x="156" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="9">bsch_ok climbs, sch_pdus = 0</text>
  <text x="156" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="9">wrong ±f_sym/4 (4500 Hz) AFC alias bucket</text>
  <line x1="120" y1="150" x2="72" y2="106" stroke="var(--accent)" stroke-dasharray="3 3"/>
  <rect x="348" y="150" width="288" height="66" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="492" y="170" text-anchor="middle" fill="currentColor" font-size="10">recovery (signal time)</text>
  <text x="492" y="186" text-anchor="middle" fill="var(--fg-muted)" font-size="9">checkResync 1.5 s · checkPayloadResync 12 s</text>
  <text x="492" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="9">3 fruitless payload resyncs → MarkLost → re-hunt</text>
</svg>
<figcaption>Every stage of the TETRA decode chain has a counter on the status line. The locked-but-deaf fault sits at the very front — a wrong AFC alias bucket — and shows up as bsch_ok climbing while sch_pdus stays flat at zero.</figcaption>
</figure>

## What healthy looks like

A locked TETRA control channel decoding cleanly, at debug level (one 5 s
window):

```
DBG tetra: decode status system=250_013 locked=true carrier_off_hz=-412.5 baud=18000 baud_dev_pct=0 sb_bursts=6 bsch_ok=5 bsch_fail=1 sysinfo=5 sch_pdus=47 sch_pdus_fail=3 frag_abandons=0 grants=2 colour_code=1
```

Read it front to back. `carrier_off_hz=-412.5` is a small, sane residual —
the AFC is on the carrier. `baud=18000 baud_dev_pct=0` confirms the symbol
clock. `bsch_ok=5 bsch_fail=1` is a healthy sync layer (a few false
candidates are expected). The line that proves the channel is genuinely
decoding is `sch_pdus=47`: signalling blocks are landing CRC-clean, so
SYSINFO, grants and neighbour broadcasts all work. `frag_abandons=0` means
no fragmented broadcast lost a block this window. This is the ~18 dB regime
from [TETRA End to End Part 10]({{ '/blog/deep-dives/tetra-end-to-end-10-control-channel-sync-loss/' | relative_url }})
where a healthy `concurrent` capture replays BSCH at 100%.

| Field | Measures | Healthy | Worry when |
|---|---|---|---|
| `carrier_off_hz` | AFC residual carrier offset | within a few hundred Hz | near ±4500 (an alias bucket) — deaf |
| `baud` / `baud_dev_pct` | symbol rate vs nominal 18000 | 18000, ~0% | drifting — clock not locked |
| `sb_bursts` | SB candidates entering the decoder | steady per window | zero — no sync at all |
| `bsch_ok` / `bsch_fail` | CRC-clean sync bursts vs not | `bsch_ok` climbing | `bsch_ok=0` with signal — off frequency |
| `sch_pdus` / `sch_pdus_fail` | CRC-clean signalling blocks | climbing every window | **`sch_pdus=0` while `bsch_ok` climbs** — locked but deaf |
| `frag_abandons` | fragment chains dropped by the guard | 0 or a low rate | a high rate — marginal RF (not a bug) |
| `grants` | voice grants published | tracks traffic | 0 on a busy site with `sch_pdus>0` — parser issue |

## The "locked but deaf" signature

The one field pattern to memorise: **`bsch_ok` climbing every window while
`sch_pdus` stays at 0**. That is not a quiet site — a real TETRA control
channel carries SYSINFO and signalling continuously — it is a wrong AFC alias
bucket, the failure the 19 Aug field log sat in for ~37 minutes:

```
DBG tetra: decode status system=250_013 locked=true carrier_off_hz=-3630.0 baud=18000 baud_dev_pct=0 sb_bursts=6 bsch_ok=5 bsch_fail=1 sysinfo=0 sch_pdus=0 sch_pdus_fail=42 frag_abandons=0 grants=0 colour_code=1
```

The mechanism is structural. TETRA's π/4-DQPSK AFC raises the per-symbol
differential phase to the 4th power, which is unambiguous only within
±f_sym/8 (±2250 Hz) and aliases in steps of f_sym/4 = 4500 Hz. If the
EMA-tracked estimate is ever primed from one bad block — a decode-drought
resync re-seeding from a weak burst — it can latch a bucket ±4500 Hz off and
then reject every correct estimate forever. The cruel part is what *survives*:
the BSCH gets per-burst frequency correction plus heavy FEC, so it decodes
through the ~17°/symbol residual (`bsch_ok` stays healthy, the stale watchdog
stays quiet) while every SCH block fails its CRC (`sch_pdus=0`,
`sch_pdus_fail` climbing). The `carrier_off_hz=-3630` here is the tell — a
residual sitting near an alias offset, not near zero. The full mechanism and
its fixes are in [AFC alias traps]({{ '/reference/afc-alias-traps/' | relative_url }})
and Part 4's reading of the [carrier-offset WARN]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }}).

## The two resync watchdogs

GopherTrunk escapes both droughts with watchdogs measured in **signal time**
— processed IQ samples, not wall clock — so a decode goroutine descheduled
under concurrent-call CPU load never fires a destructive reset falsely (a
wall-clock window would age out purely because the goroutine was starved).
The first, `checkResync`, fires on a full `tetraResyncTimeout` (1.5 s of
samples) with no decode at all — genuine off-lock:

```
DBG tetra: dsp resync (signal-time decode drought; reacquiring symbol timing from centre) system=250_013
```

Reset-to-centre discards the converged Gardner timing and AFC, so it must
fire only on a real drought — which is exactly why the budget is a sample
count. The second, `checkPayloadResync`, is the escape from locked-but-deaf:
it watches the stricter *payload* heartbeat and fires after
`tetraPayloadResyncTimeout` (12 s, 8× the ordinary window) of BSCH-alive but
SCH-dead, re-priming the AFC each time. After `tetraPayloadResyncMaxAttempts`
(3) fruitless resets — the re-primed AFC re-latching the same wrong alias on a
weak carrier — it gives up and forces a re-hunt:

```
DBG tetra: dsp resync (payload drought; sync bursts still decoding but no SCH payload) system=250_013 attempt=1
WRN tetra: payload drought persists across resyncs (sync bursts decoding, no SCH payload) — declaring lock lost to force a re-hunt system=250_013 attempts=3
```

That WARN publishes `cc.lost`, so the [lock family]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }})
takes over and the supervisor re-hunts from cold — ~36–48 s worst case
against the field's 12-minute stucks. The `widebandt2` TETRA path mirrors the
same watchdog. Both stayed untouched by the Part 10 equalizer fix: the data
showed the losses had zero correlation with CPU load, so the design was
confirmed, not implicated.

One last line you will see and misread: `tetra: abandoning TM-SDU fragment
reassembly — awaiting rebroadcast (continuity guard, not a parse error)`,
counted as `frag_abandons`. It is the reassembler refusing to splice a
MAC-END onto a start fragment across a lost block — a *safety* response that
protects against the phantom-neighbour corruption a naive splice produced.
Each abandon costs one broadcast cycle, and the broadcast repeats within
seconds, so a rising `frag_abandons` tracks marginal RF, not a decoder bug.

### How this line shapes operator practice

- **Read `sch_pdus`, not `bsch_ok`, to judge a TETRA lock.** Sync surviving
  is not decoding; a climbing `sch_pdus` is. `bsch_ok` up with `sch_pdus=0`
  is the deaf signature.
- **Check `carrier_off_hz` against the alias grid.** A residual near ±4500 Hz
  (or a multiple) is a wrong AFC bucket, not a mistuned dongle.
- **Treat `frag_abandons` as a health gauge, not an error.** A low rate is
  normal; a high rate is marginal RF, and the periodic abandon DEBUG line is
  the guard doing its job.
- **Don't chase a resync storm as a compute bug.** The watchdogs are
  signal-time; a storm means real decode droughts on real signal — an RF
  problem, per Part 10.

## Where this goes next

The deaf signature points at one field above all others: `carrier_off_hz`
sitting an alias bucket off. [Part 4]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }})
reads the WARN that surfaces it — `ccdecoder: control carrier offset far from
configured frequency` (issue #815) — why `offset_hz=5004` decomposes as
504 + 4500, why the WARN now requires 10 s of continuous excursion before it
fires, and how to tell a real wrong-site lock from a sub-second AFC spike.

## FAQ

**Why is my TETRA decode-status line all zeros?**
The counters only move at `log.level: debug` — `addStat` is a no-op at info
level so production decode isn't slowed. Set the level to debug to see
`sb_bursts`, `bsch_ok/fail`, `sch_pdus` and the rest; at info the line does
not print at all.

**What does "bsch_ok climbing but sch_pdus=0" mean on a TETRA site?**
"Locked but deaf." The AFC has latched a wrong ±f_sym/4 = 4500 Hz alias
bucket: the heavily-coded BSCH sync burst survives the residual (so the lock
looks alive) while every SCH signalling block fails CRC. Check `carrier_off_hz`
— a residual near ±4500 Hz confirms it. The payload-drought watchdog recovers
it within seconds.

**What is "tetra: dsp resync (signal-time decode drought)"?**
A recovery reset. After 1.5 s of processed signal with no CRC-clean decode,
the pipeline resets its symbol timing and AFC to centre and reacquires. It is
measured in signal time, not wall clock, so a descheduled goroutine under call
load never triggers it falsely — a storm of these means real decode droughts,
i.e. an RF problem.

**Is "abandoning TM-SDU fragment reassembly" a bug?**
No — it's the continuity guard working. It refuses to splice a fragmented L3
broadcast across a lost or unreadable block, which would produce phantom
neighbour cells. Each abandon costs one broadcast cycle; the rate
(`frag_abandons`) simply tracks marginal RF. The DEBUG line says so
explicitly.

**How often does the decode-status line print?**
Every `tetraStatusInterval`, 5 seconds by default, and the counters
accumulate over that window before resetting. Change the cadence per system
with `tetra_status_interval_secs`; the window its per-interval counters cover
changes with it. It is debug-only diagnostics with no effect at info level.

## Series navigation

**Part 3 of 14** · ←
[Part 2: Lock Lines — Locked, Lost & Transitions]({{ '/blog/tutorials/field-notebook-02-lock-lines/' | relative_url }})
· Next →
[Part 4: The Carrier-Offset WARN — Alias Buckets & Persistence]({{ '/blog/tutorials/field-notebook-04-carrier-offset-warn/' | relative_url }})
