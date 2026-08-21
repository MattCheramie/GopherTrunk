---
slug: picking-a-board
title: Picking a board
description: How to read an SBC spec sheet — RAM, CPU cores, USB bandwidth, networking, and storage interfaces — and match a board to the job you're actually giving it instead of buying on hype.
keywords: choosing a single-board computer, SBC spec sheet, Raspberry Pi RAM, USB bandwidth, gigabit Ethernet SBC, how much RAM Raspberry Pi, board selection
level: beginner
status: full
prereq:
  - sbc-vs-microcontroller-vs-pc
faq:
  - q: How much RAM does a single-board computer need?
    a: It depends entirely on the workload. A bare headless Linux system idles in a few hundred megabytes; a network service or decoder like GopherTrunk is comfortable in 2 GB; 4 GB gives real headroom for recordings, a database, and OS caches; 8 GB is for running several serious workloads at once. RAM is soldered on SBCs, so buy the size you'll want in a year, not the minimum that boots today.
  - q: What spec matters most for an SDR project?
    a: After a mid-range or better CPU, the sleeper spec is USB. A software-defined radio streams samples continuously, so the board needs enough real, sustained USB throughput — and on some boards several ports (or the Ethernet) share one internal USB controller, so the SDR competes with other traffic. A board with USB 3 ports on their own controller, plus 4 GB of RAM, is a comfortable GopherTrunk host.
  - q: Are cheap Raspberry Pi alternatives worth it?
    a: "Sometimes — the better ones offer more RAM or CPU per dollar. The price you pay is software: shorter-lived OS images, thinner documentation, and a smaller community when something breaks. For a first board or an appliance you want to trust for months, the mainstream choice usually costs less in total time."
gophertrunk_links:
  - title: Best single-board computer for GopherTrunk
    url: /best-single-board-computer-for-gophertrunk/
    note: current board-by-board recommendations against this lesson's checklist.
  - title: Hardware guide
    url: /hardware.html
    note: the SDR side of the shopping list — dongles the board will host.
---

# Picking a board

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A spec sheet answers four questions in order: **CPU** (how many cores, what
generation — decides real-time decode capacity), **RAM** (soldered forever — 4 GB is
the comfortable middle), **USB and I/O bandwidth** (the sleeper spec — an SDR streams
samples continuously and some boards bottleneck all USB through one controller), and
**networking** (wired **gigabit Ethernet** beats Wi-Fi for an appliance). Match the
board to the job's *sustained* load, and leave **headroom** — a board that fits
exactly today is a board that's too small next year.
</div>

Unit 2 is the shopping unit: board, storage, power, cooling. This first lesson gives
you the durable skill — reading any board's spec sheet against your actual workload —
so the recommendations in the [board guide](/best-single-board-computer-for-gophertrunk/)
make sense instead of being magic.

## What job are you buying for?

Write the job down before opening a shop page. For this module's destination build:
*continuously decode one or more trunked-radio channels from a USB SDR, record audio
to disk, and serve a web console — 24/7*. That one sentence already implies: a CPU
that can run DSP in real time, storage that survives constant writing, sustained USB
throughput, and a reliable network path. A different job — a network-wide ad blocker,
a weather display — implies a much smaller board. The spec sheet only means something
relative to the job.

## How much CPU is enough?

SBC spec sheets lead with cores and clock speed, but the useful reading is
generational: each Pi generation's cores are substantially faster *per core* than the
last, and for real-time DSP per-core speed matters as much as core count. Practical
guidance:

- **Real-time decoding is a sustained load**, not a burst. The CPU must keep up with
  the radio every second; there is no "it'll finish eventually." Falling behind means
  dropped samples and broken audio.
- **More channels, more CPU.** Each simultaneously monitored channel adds decode work
  — [Tuning for small CPUs](/learn/embedded/tuning-for-small-cpus/) turns this into a
  budget you can measure.
- **Prefer the current generation.** The jump in per-core speed between SBC
  generations is usually far larger than between PC generations.

## How much RAM — and why is it forever?

RAM on an SBC is **soldered** — the number you buy is the number you keep. Rough tiers:

| RAM | What it comfortably runs |
|-----|--------------------------|
| 1 GB | Bare headless Linux plus one light service |
| 2 GB | A modest decoder or network service; little headroom |
| **4 GB** | **The comfortable middle: GopherTrunk, recordings, OS caches, room to grow** |
| 8 GB+ | Several serious workloads, big databases, containers |

Unused RAM isn't wasted — Linux uses spare memory as disk cache, which on slow SD
storage is a genuine performance feature. Buy for next year's project, not today's
minimum.

## Why is USB the sleeper spec?

Here's the one that catches SDR builders. An RTL-SDR **streams samples continuously**
— megabytes every second, without pause, for as long as the scanner runs. Two spec
details decide whether that goes smoothly:

- **USB generation.** USB 2.0's real-world sustained throughput (~30 MB/s shared) is
  enough for one RTL-SDR, but tight once other traffic joins. USB 3 ports give an
  order of magnitude of headroom.
- **Controller topology.** On some boards (famously, older Pis) *all* USB ports —
  sometimes the Ethernet jack too — share **one internal controller**, so a storage
  stick's burst of writes steals bandwidth from the SDR mid-stream. Newer boards give
  USB 3 its own lanes. A spec sheet that says "4 × USB" without saying how they're
  wired is hiding the number that matters.

Insufficient USB bandwidth shows up later as mysterious dropped samples —
[USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/) is devoted to those failure modes.

## What about networking, storage interfaces, and the rest?

- **Ethernet.** For an always-on appliance, **wired gigabit Ethernet** wins: no
  dropouts, no Wi-Fi credentials to break, and — bonus for radio work — no 2.4 GHz
  transmitter sitting next to your SDR. Wi-Fi is the fallback, not the plan
  ([Networking your board](/learn/embedded/networking-your-board/)).
- **Storage interface.** All boards take SD cards; better ones can boot from a USB
  SSD or (best) an NVMe drive — a major reliability upgrade the
  [next lesson](/learn/embedded/storage-and-sd-cards/) weighs.
- **The GPIO header** matters if you'll attach electronics (Unit 4); all mainstream
  boards have one.
- **Form factor and power** feed the next lessons — a faster board eats more watts
  and needs more cooling.

> Rule of thumb: size the board so your workload uses **half** its CPU. The other
> half is headroom for busy hours, OS updates, and the feature you'll add next month.

<div class="knowledge-check" data-quiz data-correct-msg="Right — sustained USB throughput (and how ports share controllers) is the spec SDR builds live or die on." markdown="0">
  <p class="knowledge-check__q">Quick check: which under-advertised spec matters most when a board will host a streaming USB SDR?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The number of HDMI outputs</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The colour and material of the official case</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Sustained USB bandwidth and whether ports share one internal controller</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Start from the **job**, written down — a spec only means something against a
  workload.
- **CPU**: real-time decoding is a *sustained* load; prefer current-generation cores
  and budget per channel.
- **RAM is soldered** — 4 GB is the comfortable middle for a GopherTrunk appliance;
  spare RAM becomes useful disk cache.
- **USB is the sleeper spec**: an SDR streams continuously, and shared controllers
  can starve it — check topology, not just port count.
- Prefer **wired gigabit Ethernet** for an appliance, and size for **50% CPU
  headroom**.

Next up: [Storage &amp; SD cards](/learn/embedded/storage-and-sd-cards/).
