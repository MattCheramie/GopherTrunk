---
slug: what-is-an-sbc
title: What is a single-board computer?
description: A single-board computer puts a full Linux machine on one credit-card board. Learn how an SBC sits between a PC and a microcontroller, and what its GPIO pins unlock.
keywords: what is a single-board computer, SBC, Raspberry Pi, SBC vs microcontroller, SBC vs PC, GPIO pins, Linux on a small board, embedded Linux computer
level: beginner
status: full
faq:
  - q: "What is a single-board computer?"
    a: "A **single-board computer (SBC)** is a complete, working computer built onto one small circuit board — processor, memory, storage interface, and input/output all in one place. Unlike a PC, you don't assemble it from parts; unlike a microcontroller, it runs a full operating system, almost always Linux. A Raspberry Pi is the best-known example."
  - q: "Is a single-board computer the same as a microcontroller?"
    a: "No. An **SBC** runs a real operating system (usually Linux) and behaves like a small PC, while a **microcontroller** runs a single program directly on bare metal with no OS. The SBC is far more capable; the microcontroller is far cheaper, simpler, and lower-power. We cover the contrast in depth in [What is a microcontroller?](/learn/intro-hardware/what-is-a-microcontroller/)."
  - q: "What are GPIO pins for on an SBC?"
    a: "**GPIO (general-purpose input/output)** pins are the row of metal pins on the board's edge that let it sense and control the physical world — read a sensor, switch a relay, drive a light, or talk to other chips. They are the main thing that separates an SBC from an ordinary laptop, which has no easy way to touch raw electronics."
---
# What is a single-board computer?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**An SBC is a complete computer on one small board** — CPU, RAM, and I/O together, running a full OS (usually Linux). **It sits between a PC and a microcontroller** — a real operating system like a PC, plus GPIO pins to touch the physical world. **GPIO is the difference that matters** — those pins let it read sensors and control hardware directly, which a laptop can't.
</div>

This module is about a class of machine that has quietly become one of the most useful tools a maker or developer can own: the single-board computer. It's small enough to lose in a drawer, cheap enough to buy on a whim, and powerful enough to run real software for years. By the end of this lesson you'll know exactly what an SBC is, where it fits between the machines you already know, and why one specific feature — a row of pins along the edge — changes what you can build.

## A whole computer on one board

A **single-board computer (SBC)** is exactly what the name says: every part needed to make a working computer, soldered onto a single circuit board. The processor (CPU), the memory (RAM), the connections for storage, the networking, the USB ports, the video output — all of it lives on one piece of fiberglass roughly the size of a credit card.

This is different from a desktop PC, which is a collection of separate parts — motherboard, CPU, RAM sticks, graphics card, drives — that you slot together inside a case. With an SBC there's nothing to assemble. You add power and storage (usually a microSD card), and it boots.

And boot it does — into a real **operating system**. This is the headline feature. An SBC almost always runs Linux, the same kind of operating system that powers most of the world's servers. That means it has a file system, a network stack, a package manager, users, and processes. If you've used a Linux PC, you already know how to use an SBC. We'll meet the four parts every computer shares — CPU, memory, storage, and I/O — in more detail back in [CPU, memory, storage & I/O](/learn/intro-hardware/building-blocks/).

## Between a PC and a microcontroller

The best way to understand an SBC is by where it sits on the [hardware spectrum](/learn/intro-hardware/hardware-spectrum/). It lives in the gap between two machines you may already know: the full PC above it and the microcontroller below it.

- A **PC or server** has lots of power, lots of memory, and a full OS — but it's large, draws real electricity, and has no easy way to wire into raw electronics.
- A **microcontroller** is tiny, cheap, and sips power — but it runs a single bare-metal program with no operating system, and has very little memory.

An SBC takes the most useful traits of each: it has a *real operating system* like the PC, **and** it has pins to *touch the physical world* like the microcontroller. That combination is rare and powerful. You get the comfort of Linux — Python, networking, a web server, Docker — together with the ability to read a temperature sensor or flick a switch.

| Trait | Microcontroller | Single-board computer | Desktop PC / server |
|-------|-----------------|------------------------|---------------------|
| **Operating system** | None (bare metal) | Full OS, usually Linux | Full OS |
| **Typical RAM** | Kilobytes | 512 MB – 16 GB | 8 GB – 100s of GB |
| **Power draw** | Milliwatts | A few watts | Tens to hundreds of watts |
| **Physical I/O (GPIO)** | Yes, lots | Yes | Rarely, not built in |
| **Rough cost** | $1 – $20 | $15 – $150 | $400 and up |
| **Best at** | One simple, low-power job | Small always-on Linux jobs | Heavy, fast compute |

## Why GPIO changes everything

The feature that truly sets an SBC apart from a laptop is the row of metal pins along its edge: the **GPIO header**. GPIO stands for *general-purpose input/output*, and those pins are how the computer reaches out and touches the physical world.

Each pin can be set up as an **input** (the SBC reads whether the pin is high or low — a button pressed, a sensor triggered) or an **output** (the SBC drives the pin high or low — turning on a light, switching a relay, spinning a motor through a driver). Beyond simple on/off, groups of pins speak standard hardware protocols — **I2C**, **SPI**, and **UART** — which let the board talk to sensors, displays, and other chips using just a couple of wires.

A normal laptop has no equivalent. To make a laptop read a temperature sensor you'd need extra USB hardware; an SBC just wires the sensor straight to its pins. That's why SBCs dominate robotics, home automation, and sensor projects. The computer isn't behind a screen — it's *in* the project, wired into the world.

This is also where **GopherTrunk** fits naturally. An SDR scanner is software, and an SBC runs that software while sitting right at the antenna with a USB SDR dongle plugged in — a small, quiet, always-on computer doing a real job in the field. We'll return to that example throughout the module.

## How an SBC differs from a microcontroller

It's worth drawing the SBC-versus-microcontroller line clearly now, because the rest of this path leans on it. The short version: an SBC is a *computer*, and a microcontroller is a *chip that runs one program*.

An SBC boots an operating system, so it multitasks, connects to networks, stores gigabytes of data, and runs the same languages and tools you'd use on a PC. That power costs something: it needs a few watts of steady power, takes seconds to boot, and a sudden loss of power can corrupt its storage if you're unlucky.

A microcontroller skips all of that. It has no OS — your program *is* the only thing running, directly on the silicon. That makes it astonishingly cheap, able to run on a coin battery for months, and instant to start. But it can't run Linux, can't easily host a website, and has memory measured in kilobytes. We give microcontrollers a full module of their own, starting with [What is a microcontroller?](/learn/intro-hardware/what-is-a-microcontroller/) — for now, just hold onto the contrast: *OS and power versus bare-metal and frugality*.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an SBC pairs a full operating system with GPIO pins, sitting between a PC and a microcontroller." markdown="0">
  <p class="knowledge-check__q">Quick check: What most clearly distinguishes a single-board computer from a microcontroller?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">An SBC is always physically larger and heavier</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">An SBC runs a full operating system (usually Linux); a microcontroller runs one bare-metal program with no OS</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only an SBC can read sensors or control hardware</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **An SBC is a complete computer on one board** — CPU, RAM, storage interface, and I/O together, with no assembly required.
- **It runs a real operating system** — almost always Linux, so it behaves like a small, familiar PC.
- **It sits between a PC and a microcontroller** — the OS and power of one, the physical I/O of the other.
- **GPIO is the defining feature** — pins that read sensors and control hardware, plus I2C/SPI/UART to talk to other chips.
- **Versus a microcontroller** — an SBC trades higher power draw and slower boot for the full capability of an operating system.

Next up: the board that created this whole category, and the rivals worth knowing. See [The Raspberry Pi & its alternatives](/learn/intro-hardware/raspberry-pi-and-family/).
