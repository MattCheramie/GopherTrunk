---
slug: cases-and-cooling
title: Cases & cooling
description: Heatsinks, fans, and airflow for single-board computers — how heat limits sustained performance, passive vs active cooling, and choosing a case that keeps a 24/7 board cool in a closed box.
keywords: Raspberry Pi cooling, heatsink, fan, passive cooling, SBC case, airflow, CPU temperature, fanless case, thermal design
level: beginner
status: full
prereq:
  - picking-a-board
---

# Cases & cooling

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An SBC's speed is really a **thermal budget**: the SoC runs full speed only while it
stays cool, and slows itself down when it isn't. **Passive cooling** (a heatsink, or
a metal case that *is* a heatsink) is silent and has no moving parts to fail —
ideal for appliances when it's sized generously. **Active cooling** (a fan) removes
more heat but adds noise and a wear item. The case decides everything: a sealed
plastic box traps heat, while a vented or all-metal one sheds it. For a 24/7 decoder
running sustained CPU load, cool generously — you're buying **sustained** speed, not
peak speed.
</div>

The last purchase decisions are the box and what's inside it. Cooling sounds like an
enthusiast concern, but for a board running DSP around the clock it decides how much
of the CPU you paid for you actually get to use.

## Why does a computer slow down when hot?

The SoC monitors its own temperature. As it approaches a limit (around 80–85 °C on
most boards), the firmware progressively **reduces the clock speed** to shed heat —
protective, automatic, and silent. This is **thermal throttling**, and Unit 5's
[Thermal throttling](/learn/embedded/thermal-throttling/) lesson covers measuring and
diagnosing it. Today's point is the design consequence: a board's advertised speed is
its *cool* speed. Whether you get that speed **all day** is decided by the cooling
you bolt on. For a bursty desktop workload, throttling is a hiccup; for a real-time
decoder that must keep up with the radio every second, a throttled CPU can mean
falling behind and dropping samples.

You can watch the temperature any time:

```bash
$ vcgencmd measure_temp
temp=61.2'C
```

## What are the cooling options?

| Approach | How it works | Pros | Cons |
|----------|--------------|------|------|
| **Bare board** | Convection off the chip | Free | Throttles under sustained load |
| **Stick-on heatsink** | Metal fins spread heat into the air | Cheap, silent | Modest gain in still air |
| **Metal case as heatsink** | Whole case thermally bonded to the SoC | Silent, no moving parts, dustproof | Costs more; must be a *designed* thermal case |
| **Fan (active)** | Forced airflow over sink/board | Most cooling per dollar | Noise, dust, a part that wears out |

Two nuances. A heatsink in a **sealed box** helps far less than the same heatsink in
moving air — heat has to leave the case, not just the chip. And fans on SBCs are
small and cheap: they get loud, gather dust, and are often the *first* component to
fail on an always-on build. Many appliance builders choose a generously sized
**passive** metal case precisely to remove that failure mode; if you do use a fan,
prefer one the board can speed-control so it runs only when needed.

## What should the case do besides look good?

A case for an appliance has four jobs:

- **Shed heat** — vented, or all-metal and thermally bonded.
- **Admit the cables** — power, Ethernet, USB (your SDR), and the antenna feed
  routing near it, without strain on connectors.
- **Protect the board** — dust, knocks, curious pets, and accidental shorts from
  stray metal.
- **Mount somewhere** — shelf, wall, or rack; an appliance lives where the antenna
  cable and Ethernet want it to, often a closet or attic corner, so mounting
  points matter more than desk looks.

One radio-specific note: a metal case provides mild shielding of the board's own RF
noise (a real emission source — see
[USB SDR gotchas](/learn/embedded/usb-sdr-gotchas/)), but it also blocks the
board's built-in **Wi-Fi**. On wired Ethernet — the appliance default — that costs
nothing; if you must use Wi-Fi, check the case's antenna provisions.

## How does ambient temperature change the math?

Cooling moves heat from the chip to the surrounding air, so everything shifts with
that air's temperature. The case that's fine on a desk in a 20 °C room may throttle
in a 35 °C attic in summer — precisely where scanner appliances often live, close to
the antenna. Design for the **hottest day in the worst spot**: prefer over-sized
passive cooling, leave convection space around the case, and confirm with
measurement once deployed ([Monitoring your board](/learn/embedded/monitoring-your-board/)
puts temperature on a dashboard so summer can't surprise you).

> Rule of thumb: under your appliance's real sustained load, aim for a reported SoC
> temperature comfortably below 70 °C. That leaves margin for summer, dust, and an
> aging fan you haven't noticed stopping.

<div class="knowledge-check" data-quiz data-correct-msg="Right — cooling buys sustained speed; a hot board quietly gives its speed back." markdown="0">
  <p class="knowledge-check__q">Quick check: why does cooling matter more for a 24/7 decoder than for a desktop used in bursts?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Sustained load keeps the SoC hot, and a throttled CPU can fall behind real-time decoding</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Heat erases the SD card's contents within a few days</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It doesn't — thermal throttling only affects gaming workloads</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An SBC's advertised speed is its **cool** speed: the SoC **throttles** itself as it
  approaches its thermal limit, and sustained decoding is exactly the load that gets it there.
- **Passive cooling** (heatsink or thermal metal case) is silent with no wear items —
  the appliance favourite when sized generously; **fans** cool more but add a failure mode.
- Heat must leave the **case**, not just the chip — sealed plastic boxes trap it.
- Design for the **hottest ambient** the board will meet, and verify with
  `vcgencmd measure_temp` once deployed.
- The case's other jobs: cable access, protection, and **mounting** where the
  appliance actually lives.

Next up: [When you need more than a Pi](/learn/embedded/when-you-need-more/).
