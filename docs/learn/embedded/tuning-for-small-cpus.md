---
slug: tuning-for-small-cpus
title: Tuning for small CPUs
description: Sample rates, channel counts, and decode load — how to measure a real-time DSP workload on an SBC, find the knobs that actually move CPU cost, and fit a scanner inside a small compute budget with headroom.
keywords: gophertrunk cpu usage, sbc tuning, sample rate cpu cost, channel count, real-time dsp load, htop, decode overrun, headroom, small cpu optimization
level: advanced
status: full
prereq:
  - installing-gophertrunk-on-a-pi
gophertrunk_links:
  - title: Best single-board computer for GopherTrunk
    url: /best-single-board-computer-for-gophertrunk/
    note: which boards carry which workloads comfortably, measured.
---

# Tuning for small CPUs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Real-time decoding is a **hard deadline**: samples arrive continuously, and a CPU
that falls behind doesn't finish late — it **drops samples** and breaks decodes.
The cost drivers, in order: **captured bandwidth** (sample rate — every captured
hertz is DSP whether or not anything interesting is there), **simultaneous
decode/record count**, and per-call extras. Tuning is measurement-driven: watch
**load average and per-core use** under real traffic, watch the daemon's own
**can't-keep-up warnings**, then cut the biggest cost that buys the least. Ship
at **≤50% sustained CPU** — busy hours are exactly when you can't afford to
drop.
</div>

The daemon decodes — but a small CPU holds the whole thing to a budget, and this
lesson is the budgeting skill: what real-time work costs, which knobs matter (and
which are noise), and how to leave the headroom that keeps the busy hour clean.

## Why is real time unforgiving?

A batch job on a slow CPU finishes late; nobody dies. A real-time decoder has no
"late": the SDR delivers its samples every second regardless, and processing
them slower than they arrive means buffers fill and samples **drop** — and with
them, chunks of the control channel and the voice you were recording. The
symptoms read as radio problems (missed grants, choppy audio, lost lock), which
is why CPU starvation is worth ruling out *first* — it's the one cause you can
diagnose entirely from the shell. The daemon says it plainly when it's
struggling: warnings that decode **can't keep up with real time** in the journal
are the CPU asking for help, not a decoder bug.

## What actually costs CPU?

Three drivers, in the order they dominate:

**Captured bandwidth.** The front of the DSP chain — filtering and channelizing
the raw stream ([the pipeline](/learn/rf-sdr/demodulation-pipeline/)) — scales
with the **sample rate** you configure, *not* with how much of that spectrum is
interesting. Capturing 2.4 MHz to watch one control channel pays the wideband
tax for silence. Match the captured width to the system you're actually
following; on a small board, the difference between "covers the whole band" and
"covers this system's channels" is often the difference between overload and
comfort.

**Simultaneous decode/recording count.** Each concurrently followed voice
channel adds demodulation, error correction, and a vocoder. A quiet system costs
little; the county-wide storm hour costs the maximum. Budget for the busy hour
— and cap it deliberately: a scan-list that follows *your* talkgroups instead of
everything ([talkgroups &amp; scan lists](/learn/scanning/talkgroups-and-scan-lists/))
is also a CPU policy.

**Per-call extras.** Audio post-processing, transcoding, streaming — each
nice-to-have rides on every call. On a Pi 5 with one modest system, irrelevant;
on a Zero-class board, decisive.

## How do you measure instead of guess?

Under **real traffic** (a quiet Sunday morning undersells the budget):

```bash
$ htop                 # per-core bars: sustained %, and whether one core is pinned
$ uptime               # load average vs core count — the headline number
$ journalctl -u gophertrunk | grep -ic "keep up"   # the daemon's own verdict
```

Three readings to interpret honestly. **Sustained CPU** across the busy hour —
peaks are fine, plateaus near the ceiling are not. **Load average vs cores**
([When you need more](/learn/embedded/when-you-need-more/)'s test — 4.0 on four
cores is saturation). And **one pinned core with idle siblings**: pipeline
stages aren't infinitely parallel, so a single hot stage can bottleneck a board
whose *average* looks healthy. Cross-check the physical layer while you're
there: [throttling](/learn/embedded/thermal-throttling/) shrinks the budget
you're tuning within — `vcgencmd get_throttled` first, or you'll tune against a
moving target.

> Rule of thumb: tune to **≤50% sustained CPU during the busy hour**, throttling
> flags clean. That headroom absorbs traffic spikes, OS chores, and summer — the
> moments recordings are worth the most.

## Which knob do you turn first?

Cheapest confirmed win first:

1. **Stop background waste.** The Lite image is nearly clean, but check `htop`
   for stowaways — a desktop stack, an abandoned experiment, a debug-verbosity
   log flood ([SD-card wear](/learn/embedded/sd-card-wear/)'s friend).
2. **Cut captured bandwidth** to what the followed system needs — the biggest
   single lever, usually costing nothing you wanted.
3. **Cap concurrency** — scan-list scope and simultaneous-recording limits
   sized to the busy hour you measured.
4. **Shed per-call extras** you added on a whim.
5. **Then** conclude the board is too small — with measurements in hand, either
   a [bigger board](/learn/embedded/when-you-need-more/) or split systems
   across two small ones. The [board guide](/best-single-board-computer-for-gophertrunk/)
   maps workloads to boards so the next purchase is sized by data.

Change **one knob at a time** and re-measure the busy hour; a tuning change
whose effect you didn't measure is a superstition, not a fix. Keep the
measurement ritual in your [monitoring](/learn/embedded/monitoring-your-board/)
trend — load drift after a config change is the early warning that you spent
your headroom.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the front-end DSP cost scales with captured bandwidth, interesting or not." markdown="0">
  <p class="knowledge-check__q">Quick check: why does a wider captured sample rate cost CPU even when the extra spectrum is empty?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Empty spectrum compresses poorly, filling RAM with noise</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The filtering and channelizing front end must process every captured sample, signal or silence</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It doesn't — CPU use depends only on how many calls are active</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Real-time decoding has a **hard deadline**: a CPU that falls behind **drops
  samples**, and the symptoms masquerade as radio problems.
- Costs ranked: **captured bandwidth** (pay per hertz, interesting or not),
  **simultaneous decode count** (budget the busy hour), per-call extras.
- **Measure under real traffic**: sustained %, load vs cores, single-core
  pinning, the daemon's *can't-keep-up* warnings — and clear throttling flags
  before tuning.
- Turn knobs in order — waste, bandwidth, concurrency, extras — **one at a
  time, re-measuring**; upgrade only with data in hand.
- Ship at **≤50% sustained busy-hour CPU**: headroom is what keeps the storm
  hour recorded.

Next up: [USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/).
