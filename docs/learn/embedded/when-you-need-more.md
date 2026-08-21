---
slug: when-you-need-more
title: When you need more than a Pi
description: The signs a project has outgrown a single-board computer — sustained CPU saturation, I/O bottlenecks, RAM pressure — and the step up to mini PCs and small servers when it has.
keywords: Raspberry Pi alternatives, mini PC vs Raspberry Pi, outgrowing an SBC, x86 mini PC, small home server, when to upgrade, N100 mini PC, SBC limits
level: beginner
status: full
prereq:
  - picking-a-board
---

# When you need more than a Pi

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Upgrade on **evidence, not vibes**: the real signs are **sustained CPU saturation**
(load average pinned above core count), **RAM pressure** (heavy swapping), and
**I/O bottlenecks** (USB or storage can't keep up) — measured, not guessed. The next
rung is a **mini PC**: 3–10× the compute, real SSD storage, ample USB, still small
and quiet, at 2–4× the price and power draw. Everything this module teaches —
headless Linux, systemd, monitoring — **transfers unchanged**, because a mini PC is
just a bigger appliance. Wideband multi-system decoding is the classic GopherTrunk
reason to climb.
</div>

Unit 2 closes with the honest question every SBC fan eventually faces: is this board
still the right tool? The answer should come from measurements you already know how
to take — and the comforting news is that "moving up" abandons none of your skills.

## What are the real signs you've outgrown the board?

Three measurable symptoms, in the order they usually appear:

- **Sustained CPU saturation.** The **load average** (from `uptime` or `htop`) sits
  at or above the number of CPU cores for long stretches — the machine has more
  runnable work than cores. For a real-time decoder this shows up as the daemon
  logging that decoding can't keep up, dropped samples, or broken audio during busy
  periods. Occasional peaks are fine; *pinned* is the signal.
- **RAM pressure.** `free -h` shows almost no available memory and swap in constant
  use; the system feels sticky and the storage (busy swapping) wears faster.
- **I/O bottlenecks.** The workload needs more USB streaming bandwidth than the board
  has (several wideband SDRs at once), or storage throughput the SD interface can't
  deliver.

```bash
$ uptime
 21:14:03 up 12 days,  4:11,  1 user,  load average: 4.12, 4.05, 3.98
# on a 4-core board: saturated, no headroom
```

Before shopping, spend an hour on the cheaper fixes: Unit 6's
[Tuning for small CPUs](/learn/embedded/tuning-for-small-cpus/) — sensible sample
rates, fewer simultaneous channels, no wasted services — often buys back the margin.
Upgrade when *tuned* load still saturates.

## What does the next rung actually look like?

The step up from an SBC is usually a **mini PC** — a small, quiet x86 box (the
NUC-style form factor, and the many cheap efficient-CPU models around it):

| | Full-size SBC | Mini PC | Small server / desktop |
|---|---|---|---|
| **CPU** | 4 efficient ARM cores | 4–8 faster x86 cores | Many fast cores |
| **RAM** | Soldered, 1–8 GB | Socketed, 8–64 GB | 32 GB+ |
| **Storage** | SD card / one SSD | Real NVMe SSD(s) | Many drives, RAID |
| **Power draw** | 3–15 W | 10–40 W | 40 W+ |
| **Noise** | Silent (passive) | Near-silent | Audible |
| **Price** | $35–100 | $150–400 | $400+ |
| **GPIO header** | Yes | No | No |

Note what you give up: the GPIO header (Unit 4's direct electronics — a mini PC
talks to hardware only over USB) and the last word in power draw. Note also what you
*don't* give up: every skill. A mini PC runs the same Debian-family Linux, the same
[systemd services](/learn/embedded/services-with-systemd/), the same SSH workflow,
the same monitoring. It is a bigger appliance, administered identically — and on
x86 you'd simply grab GopherTrunk's x86-64 build instead of the ARM one.

## What's the classic GopherTrunk reason to climb?

One trunked system at modest sample rates is comfortable SBC territory. The climb
usually starts when ambition compounds: **multiple systems at once**, a **wideband
capture** that lets one SDR cover many channels simultaneously (more samples per
second in means proportionally more DSP), always-on **recording of everything**, plus
a growing database and busier web console. Each is CPU and I/O; together they cross
the line. The [deployment module](/learn/deployment/what-is-deployment/) picks up the
story of running GopherTrunk on bigger hosts.

## Could you use several small boards instead?

Sometimes the right answer is sideways, not up: two SBCs, each dedicated to one radio
system, can beat one big box — they fail independently, sit close to different
antennas, and stay cheap. The cost is administering two machines (Unit 5's
monitoring and backups, twice). This "one appliance, one job" pattern is very much in
the embedded spirit; choose it when the workloads are naturally separate, and a
bigger single box when they share data.

> Rule of thumb: measure first, tune second, upgrade third. A pinned load average
> after honest tuning is the green light — and buy the next size up from what
> today's numbers need.

<div class="knowledge-check" data-quiz data-correct-msg="Right — sustained saturation after tuning, measured with real numbers, is the upgrade signal." markdown="0">
  <p class="knowledge-check__q">Quick check: what most clearly says a project has outgrown its SBC?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A newer model of the board has been announced</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Load average stays pinned at or above the core count even after tuning the workload</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The board feels warm to the touch during use</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Upgrade on **measured evidence**: pinned **load average**, real **RAM pressure**,
  or **I/O bandwidth** the board physically lacks.
- **Tune before you buy** — sample rates and channel counts often buy the margin back.
- The next rung is a **mini PC**: several times the compute, real SSDs, same Linux,
  same skills — a bigger appliance, minus the GPIO header.
- The classic climb trigger for GopherTrunk is **wideband, multi-system, record-everything**
  ambition.
- Consider **several small boards** ("one appliance, one job") when workloads are
  naturally separate.

Next up: [Flashing an OS image](/learn/embedded/flashing-an-os-image/).
