---
slug: arm-and-socs
title: ARM and the system-on-chip
description: What a system-on-chip is, why one chip carries the CPU, GPU, memory controller, and I/O, and how ARM's licensing model put its processor cores in nearly every small board and phone.
keywords: ARM, system on chip, SoC, ARM architecture, ARM vs x86, Raspberry Pi SoC, CPU cores, instruction set, ARM licensing, aarch64
level: beginner
status: full
prereq:
  - sbc-vs-microcontroller-vs-pc
---

# ARM and the system-on-chip

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **system-on-chip (SoC)** packs the CPU cores, GPU, memory controller, and I/O
controllers that a PC spreads across a motherboard into **one chip** — which is what
makes a credit-card computer possible. Nearly every SoC uses **ARM** CPU cores,
because ARM **licenses designs** instead of selling chips, letting hundreds of
companies build their own SoCs around the same cores. ARM's design tradition favours
**performance per watt**, which is why it owns phones and SBCs. For you it means one
practical thing: software must be built for the **ARM architecture** — which is why
GopherTrunk ships ARM Linux builds.
</div>

You've placed the SBC on the capability ladder. This lesson opens the lid: what is
actually on that little board, and why does the word "ARM" appear on nearly all of
them? Understanding the SoC explains both the Pi's price and the one gotcha you'll hit
when installing software on it.

## What does a PC motherboard do that a Pi doesn't?

Open a desktop PC and you'll find the computer spread out: a CPU in a socket, RAM in
slots, a chipset shepherding I/O, a graphics card, and controllers dotted around the
board. Each part is separately made, separately replaceable, and connected by long
copper traces.

An SBC collapses almost all of that into a single **system-on-chip**: one piece of
silicon containing the CPU cores, the graphics processor, the memory controller, video
encoders/decoders, and the controllers for USB, network, display, and the low-level
pins. The board around it is mostly just connectors and power regulation.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 240" role="img" aria-label="Block diagram of a system-on-chip: one outlined chip containing CPU cores, GPU, memory controller, and I/O controllers, with external RAM, SD card, USB, and Ethernet connected around it." xmlns="http://www.w3.org/2000/svg">
  <rect x="90" y="20" width="340" height="160" fill="none" stroke="currentColor" stroke-width="2" rx="8"/>
  <text x="260" y="14" text-anchor="middle" font-size="12" fill="currentColor">one chip — the SoC</text>
  <rect x="105" y="35" width="145" height="60" fill="none" stroke="currentColor" stroke-width="1.5" rx="4"/>
  <text x="177" y="60" text-anchor="middle" font-size="12" fill="currentColor">CPU cores</text>
  <text x="177" y="78" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">(ARM)</text>
  <rect x="270" y="35" width="145" height="60" fill="none" stroke="currentColor" stroke-width="1.5" rx="4"/>
  <text x="342" y="70" text-anchor="middle" font-size="12" fill="currentColor">GPU / video</text>
  <rect x="105" y="110" width="145" height="55" fill="none" stroke="currentColor" stroke-width="1.5" rx="4"/>
  <text x="177" y="142" text-anchor="middle" font-size="12" fill="currentColor">memory controller</text>
  <rect x="270" y="110" width="145" height="55" fill="none" stroke="currentColor" stroke-width="1.5" rx="4"/>
  <text x="342" y="136" text-anchor="middle" font-size="12" fill="currentColor">I/O controllers</text>
  <text x="342" y="152" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">USB · SD · GPIO · net</text>
  <line x1="177" y1="180" x2="177" y2="210" stroke="currentColor" stroke-width="1.5"/>
  <text x="177" y="226" text-anchor="middle" font-size="12" fill="currentColor">RAM</text>
  <line x1="342" y1="180" x2="342" y2="210" stroke="currentColor" stroke-width="1.5"/>
  <text x="342" y="226" text-anchor="middle" font-size="12" fill="currentColor">SD · USB · Ethernet</text>
</svg>
<figcaption>A <strong>system-on-chip</strong> integrates what a PC spreads across a motherboard. The board around it is mostly connectors and power.</figcaption>
</figure>

Integration is why the whole board can cost $35: one chip to make, one chip to place,
short traces, small board. It's also why nothing on an SBC is upgradeable — the "CPU"
and "chipset" are the same object, soldered down, RAM often stacked right on top.

## What is ARM, and why is it everywhere?

**ARM** is a company that designs CPU cores — and, unusually, **doesn't manufacture
chips**. It licenses the designs (and the underlying **instruction set**, the
vocabulary of operations a CPU understands) to anyone who pays. Broadcom licenses ARM
cores for the Pi's SoC; Apple, Qualcomm, Samsung, Rockchip, and Allwinner do the same
for theirs. Hundreds of different SoCs, one shared architecture.

Contrast that with **x86**, the architecture of desktop PCs, where Intel and AMD design
*and* build the chips themselves. The licensing model is why you can't buy a $35 x86
board from a dozen vendors, but can buy a $35 ARM board from twenty.

ARM's design tradition, born in low-power devices, optimises **performance per watt**
— how much computing you get per unit of electricity. That's the currency that matters
in a phone (battery) and an SBC (small, fanless, cheap power supply), which is why ARM
conquered both. Modern ARM cores are genuinely fast; the difference from x86 today is
less "slow vs fast" than "designed to a power budget vs designed to a socket."

## What does the architecture mean for your software?

Here's the practical payoff of this lesson. A compiled program is machine code for one
instruction set: a binary built for x86 **will not run** on an ARM board, and vice
versa. So on an SBC you must either:

- install software from your OS's package repository (already built for ARM),
- download a vendor's **ARM build** — GopherTrunk publishes ARM Linux binaries for
  exactly this reason, which Unit 6 installs — or
- compile from source on the board (or cross-compile from your PC).

One more wrinkle: ARM comes in **32-bit** (`armhf`) and **64-bit** (`arm64` /
`aarch64`) flavours. Modern Pi OS images are 64-bit, and you should match the binary to
the OS: check with `uname -m` (an `aarch64` result wants an `arm64` download). You'll
do this for real in [Install GopherTrunk on a Pi](/learn/embedded/installing-gophertrunk-on-a-pi/).

```bash
$ uname -m
aarch64        # 64-bit ARM — download arm64 builds
```

## What else lives on the SoC that you'll care about?

Three residents of the SoC show up later in this module:

- **The video/GPU block** handles display output and video encoding — mostly idle on a
  headless appliance, which is fine.
- **The I/O controllers** set hard limits: how many USB ports share how much bandwidth,
  whether Ethernet is gigabit, how fast the SD interface runs. Two boards with the same
  CPU can differ hugely here — this is next unit's
  [Picking a board](/learn/embedded/picking-a-board/).
- **The thermal sensor and clock governor**: the SoC measures its own temperature and
  slows itself down when hot — [thermal throttling](/learn/embedded/thermal-throttling/),
  Unit 5's opening topic.

<div class="knowledge-check" data-quiz data-correct-msg="Right — ARM licenses core designs, so many companies build their own SoCs around them." markdown="0">
  <p class="knowledge-check__q">Quick check: why do so many different companies' SoCs all contain ARM cores?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">ARM manufactures all the chips itself and sells them cheaply</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">ARM licenses its core designs, so any chipmaker can build an SoC around them</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Governments require ARM cores in small devices</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **system-on-chip** integrates CPU, GPU, memory controller, and I/O into one chip —
  the reason SBCs are small, cheap, and non-upgradeable.
- **ARM licenses CPU designs** rather than selling chips, so hundreds of vendors build
  SoCs on one shared architecture.
- ARM optimises **performance per watt** — the currency of phones and small boards.
- Compiled software is architecture-specific: SBCs need **ARM builds** (and 64-bit
  `arm64` vs 32-bit matters — check `uname -m`). GopherTrunk ships ARM Linux builds.
- The SoC's **I/O controllers** and **thermal behaviour** set the real-world limits
  you'll meet in Units 2 and 5.

Next up: [Why the Raspberry Pi?](/learn/embedded/why-the-raspberry-pi/).
