---
slug: sbc-vs-microcontroller-vs-pc
title: SBC vs microcontroller vs PC
description: Raspberry Pi, Arduino, and desktop PCs solve different problems. Learn the capability ladder — microcontroller, single-board computer, full PC — and how to pick the right rung for a project.
keywords: SBC vs microcontroller, Raspberry Pi vs Arduino, single-board computer, microcontroller, when to use a Raspberry Pi, capability ladder, choosing hardware
level: beginner
status: full
prereq:
  - what-is-embedded
---

# SBC vs microcontroller vs PC

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Small computing comes in three rungs. A **microcontroller** (Arduino, ESP32) is a
single chip running one program with no operating system — instant-on, tiny power,
perfect timing. A **single-board computer** (Raspberry Pi) is a full Linux computer
on one board — files, networking, and real software, at the cost of a boot sequence
and more power. A **PC** brings serious CPU and expandability when the job outgrows
both. The rule: pick the **lowest rung that comfortably fits the job**. GopherTrunk's
real-time decoding needs Linux and USB bandwidth — an SBC job, not a microcontroller one.
</div>

Lesson 1 said embedded systems are defined by dedication and constraints. This lesson
gives you the map of *what to dedicate*: three families of hardware, each excellent at
different jobs and wrong for others. Getting this choice right up front saves whole
projects.

## What is a microcontroller?

A **microcontroller** (MCU) is a complete computer on a single chip: CPU, a small
amount of RAM (kilobytes, not gigabytes), flash storage for one program, and pins that
connect directly to electronics. Boards like the **Arduino Uno**, **ESP32**, and
**Raspberry Pi Pico** exist to make that chip easy to use.

The defining fact: **there is no operating system**. Your program is the only thing
running. That sounds primitive, but it buys three superpowers:

- **Instant-on** — code runs milliseconds after power arrives; nothing to boot.
- **Perfect timing** — with no OS scheduling anything else, your code can toggle a pin
  with microsecond precision, which matters for motors, LEDs, and precise sensors.
- **Tiny power draw** — microamps asleep, milliamps awake; months on a battery.

The costs are the mirror image: no files, no networking stack to speak of (the ESP32's
Wi-Fi being the famous exception), no running two programs at once, and development
means compiling on a PC and flashing the chip. The
[Computer Hardware module](/learn/intro-hardware/what-is-a-microcontroller/) goes
deeper on MCUs if this rung is your interest.

## What is a single-board computer?

A **single-board computer** (SBC) is everything a small computer needs — processor,
RAM, storage slot, USB, networking, video — on one board the size of a credit card.
The **Raspberry Pi** is the canonical example. The defining fact here is the opposite
one: **it runs a real operating system**, almost always Linux.

That changes the character of everything you do:

- You get **files, users, packages, and a shell** — the whole Linux toolbox from the
  [Linux &amp; CLI module](/learn/linux-cli/what-is-linux/).
- You get **real networking**: SSH in from your laptop, serve a web page, mount a
  network drive.
- You can run **many programs at once** — a decoder, a web server, and a monitoring
  script side by side.
- You can use **big software** — Python, Go binaries, databases, GopherTrunk itself.

The trade-offs: it takes tens of seconds to boot, draws watts rather than milliwatts,
its timing is "usually fast" rather than guaranteed (Linux may schedule something else
at any microsecond), and its storage — typically an SD card — must be treated with care,
as Unit 5 will explain.

## When do you need a full PC?

Some jobs outgrow an SBC: heavy video transcoding, machine learning, compiling large
codebases, or — closer to home — decoding *many* wideband radio channels at once. A
mini PC or desktop brings faster cores, more of them, far more RAM and I/O bandwidth,
and proper SSD storage. It costs more, draws more power, and takes more space. Lesson
[When you need more than a Pi](/learn/embedded/when-you-need-more/) covers the signs
your project has crossed that line.

## How do the three rungs compare?

| | Microcontroller | Single-board computer | PC / mini PC |
|---|---|---|---|
| **Example** | Arduino, ESP32, Pi Pico | Raspberry Pi and rivals | NUC-style mini PC, desktop |
| **Operating system** | None (your code only) | Full Linux | Full Linux/Windows |
| **RAM** | KB | 1–16 GB | 16 GB+ |
| **Boot time** | Milliseconds | Tens of seconds | Tens of seconds |
| **Power draw** | mW | 3–15 W | 15–200+ W |
| **Timing precision** | Microseconds, guaranteed | Best-effort | Best-effort |
| **Typical price** | $3–$15 | $15–$100 | $150+ |
| **Sweet spot** | Sensors, motors, battery devices | Network services, decoding, cameras | Heavy compute, many workloads |

> Rule of thumb: if the job needs an OS feature — files, networking, USB devices,
> multiple programs — start at the SBC rung. If it needs battery life or microsecond
> timing, start at the MCU rung. Only climb to a PC when measurement says you must.

## Which rung is a radio scanner?

Work the GopherTrunk appliance through the ladder. It must: read a continuous stream of
samples from a **USB SDR** (needs a real USB stack and sustained bandwidth), run a
**multi-channel DSP engine** (needs a capable CPU and an OS to schedule it), write
**recordings to storage** (needs a filesystem), and serve a **web console** (needs a
network stack). Every one of those is an operating-system feature — far beyond any
microcontroller, and comfortably within a modern Pi for a sensible number of channels.
That's why Unit 6 builds on an SBC, and why
[Tuning for small CPUs](/learn/embedded/tuning-for-small-cpus/) exists: an SBC fits,
but not with infinite headroom.

<div class="knowledge-check" data-quiz data-correct-msg="Exactly — no operating system means instant-on, guaranteed timing, and tiny power draw." markdown="0">
  <p class="knowledge-check__q">Quick check: what is the defining difference between a microcontroller and a single-board computer?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">A microcontroller runs your one program with no operating system; an SBC runs full Linux</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A microcontroller is always faster than an SBC</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">An SBC has no networking, while a microcontroller does</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The small-computer ladder has three rungs: **microcontroller**, **single-board
  computer**, **PC** — each right for different jobs.
- A **microcontroller** runs one program with **no OS**: instant-on, microsecond
  timing, battery-friendly, but no files or real networking.
- An **SBC** runs **full Linux**: files, networking, USB, many programs at once —
  at the cost of boot time, watts, and best-effort timing.
- A **PC** is the escape hatch for jobs that outgrow an SBC's CPU, RAM, or I/O.
- Pick the **lowest rung that comfortably fits**; GopherTrunk's USB SDR + DSP + web
  console workload lands squarely on the SBC rung.

Next up: [ARM and the system-on-chip](/learn/embedded/arm-and-socs/).
