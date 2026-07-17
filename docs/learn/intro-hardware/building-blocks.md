---
slug: building-blocks
title: CPU, memory, storage & I/O
description: Every computer is built from four parts — a processor, working memory, long-term storage, and ways to talk to the world. Learn what each does and how they scale from servers to chips.
keywords: cpu memory storage io, what is a cpu, ram vs storage, what is volatile memory, gpio and io, ssd vs flash vs eeprom, computer building blocks, cores and clock speed
level: beginner
status: full
faq:
  - q: "What is the difference between memory and storage?"
    a: "**Memory (RAM)** is fast, volatile working space — it holds whatever the computer is actively using right now and is wiped when the power goes off. **Storage** is slower but non-volatile — it keeps files, programs, and data when the machine is unplugged. A rough analogy: memory is your desk (what you're working on now), storage is the filing cabinet (everything kept for later)."
  - q: "What do cores and clock speed actually mean?"
    a: "A **core** is an independent processing unit inside a CPU; more cores let the chip work on more things at once. **Clock speed** (measured in GHz) is roughly how many basic steps each core takes per second. A 4-core, 3 GHz chip can do four streams of work, each taking about three billion simple steps a second — but real-world speed depends on far more than these two numbers."
  - q: "What is I/O on a small device like a microcontroller?"
    a: "On a microcontroller, **I/O** usually means **GPIO** — general-purpose input/output pins you wire directly to sensors, buttons, LEDs, and motors. Larger machines do I/O through USB ports, network cards, and displays instead. It is the same idea at every scale: the channels a computer uses to read from and act on the outside world."
  - q: "Why does a microcontroller have so little memory?"
    a: "Because it is built for one small job at low cost and low power. A microcontroller might have a few kilobytes of RAM where a server has hundreds of gigabytes — a difference of millions of times. It does not need more: blinking an LED or reading a sensor takes almost nothing, and shrinking memory keeps the chip cheap and power-thrifty."
---
# CPU, memory, storage & I/O

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Four parts, every machine** — a CPU executes instructions, memory holds active work, storage keeps data when powered off, and I/O connects to the world. **Memory is volatile, storage is not** — RAM is fast and forgetful; storage is slower but permanent. **Scale is the only difference** — a server and a microcontroller have the same four parts, separated by a factor of millions in size and speed.
</div>

In the last lesson we said every computer is anything that fetches instructions and runs them. This lesson opens the box. Whether you're looking at a data-center server or a chip the size of a fingernail, you'll find the same four building blocks doing the same four jobs. Once you can name them and know what each one limits, the whole rest of this path — every platform, every trade-off — becomes much easier to reason about.

## The CPU: where instructions run

The **CPU** (central processing unit, or just "the processor") is the part that actually does the work. Its entire job is the fetch-execute loop: pull the next instruction from memory, carry it out, repeat — billions of times a second. Each instruction is tiny: add two numbers, compare them, move data, jump somewhere else. Everything your computer does is composed from these primitive steps.

Two numbers usually describe a CPU. **Clock speed**, measured in gigahertz (GHz), is roughly how many basic steps a core takes per second — 3 GHz means about three billion. **Cores** are independent processing units on the chip; a four-core CPU can run four streams of work at the same time. More cores and a higher clock generally mean more capacity, but they're far from the whole story: cache size, instruction design, and memory speed all matter, which is why you can't compare two chips on GHz alone.

The range is staggering. A server CPU might have 64 cores running at high speed; a microcontroller often has a single core at a few hundred megahertz or less. Both run the same kind of loop — one just does vastly more of it.

## Memory: fast, volatile working space

**Memory** — almost always meaning **RAM** (random-access memory) — is the CPU's working space. It holds the program currently running and the data it's actively using, because RAM is far faster to read and write than storage. When you open an app, it's loaded from storage into memory so the CPU can work with it quickly.

The defining trait of RAM is that it's **volatile**: cut the power and everything in it vanishes. That's fine, because it's only ever holding work-in-progress. Think of it as a desk — large enough to spread out what you're using now, but cleared off every night.

Memory is often the real ceiling on what a machine can do. If a program needs more RAM than the device has, it either runs painfully slowly or won't run at all. A server with 256 GB of RAM can juggle thousands of tasks; a microcontroller with 2 KB can hold only a few hundred variables. The job has to fit in the space available.

## Storage: keeping data when the power is off

**Storage** is the **non-volatile** counterpart to memory: it keeps its contents with the power off. This is where your operating system, programs, files, and saved data actually live. It's slower than RAM but vastly larger and permanent. Storage comes in several forms depending on the machine:

| Type | Where you find it | Notes |
|------|-------------------|-------|
| **SSD** (solid-state drive) | Laptops, desktops, servers | Fast flash-based storage, no moving parts |
| **HDD** (hard disk drive) | Servers, older PCs, bulk archives | Spinning magnetic platters; cheap per gigabyte, slower |
| **Flash / eMMC** | Phones, tablets, single-board computers | Compact flash storage soldered onto the board |
| **SD / microSD card** | Raspberry Pi, cameras, many SBCs | Removable flash; convenient but wears out over time |
| **EEPROM / on-chip flash** | Microcontrollers | A small amount of storage built into the chip for the program itself |

The pattern is the same everywhere: a slower, permanent place to keep things, sized to the machine. A microcontroller might store its entire program in a few hundred kilobytes of on-chip flash; a server may hold dozens of terabytes across many drives.

## I/O: how a computer touches the world

A processor with memory and storage but no way to communicate would be useless. **I/O** (input/output) is everything that lets a computer take in information and act on it. The forms it takes depend heavily on the machine:

- **Networking** — Ethernet and Wi-Fi, how a machine reaches the internet and other devices. Essential for a server, optional for a sensor.
- **USB and peripherals** — keyboards, mice, drives, and the SDR dongles GopherTrunk uses to pull in radio signals.
- **Displays** — screens, from a phone's touchscreen to a server's complete absence of one (servers are usually "headless").
- **GPIO** — general-purpose input/output pins on single-board computers and microcontrollers, wired directly to buttons, LEDs, motors, and sensors. This is how a tiny device senses and controls the physical world.

I/O is often what actually decides which machine fits a job. A Raspberry Pi running GopherTrunk right at an antenna needs a USB port for the SDR dongle and a network connection to send data back; a microcontroller monitoring a temperature sensor needs GPIO and almost nothing else. Same four building blocks, very different I/O.

## Putting it together across the spectrum

Each block scales independently, but they tend to move together: powerful machines have lots of everything, tiny ones have little of anything.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A block diagram of the four building blocks. Storage on the left loads programs into RAM in the middle, which the CPU on the right fetches and runs; the CPU also exchanges data with input and output below it. Storage is non-volatile and keeps its contents with the power off, while RAM is volatile and clears when the power is cut." xmlns="http://www.w3.org/2000/svg">
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="20" y="52" width="112" height="58" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.3"/>
    <text x="76" y="78" font-weight="600">Storage</text>
    <text x="76" y="94" font-size="8">non-volatile</text>
    <rect x="174" y="52" width="112" height="58" rx="6" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
    <text x="230" y="78" font-weight="600">RAM</text>
    <text x="230" y="94" font-size="8">volatile · working</text>
    <rect x="328" y="52" width="112" height="58" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1.3"/>
    <text x="384" y="78" font-weight="600">CPU</text>
    <text x="384" y="94" font-size="8">runs instructions</text>
    <rect x="328" y="140" width="112" height="42" rx="6" fill="currentColor" fill-opacity="0.05" stroke="currentColor" stroke-width="1.2"/>
    <text x="384" y="160" font-weight="600" font-size="9">I/O</text>
    <text x="384" y="173" font-size="8">USB · net · GPIO</text>
  </g>
  <g stroke="currentColor" fill="none">
    <line x1="132" y1="74" x2="174" y2="74" stroke-width="1.6" marker-end="url(#bb_ar)"/>
    <line x1="286" y1="74" x2="328" y2="74" stroke-width="1.6" marker-end="url(#bb_ar)"/>
    <line x1="384" y1="110" x2="384" y2="140" stroke-width="1.4" marker-end="url(#bb_ar)"/>
    <line x1="374" y1="140" x2="374" y2="110" stroke-width="1.4" marker-end="url(#bb_ar)"/>
  </g>
  <g fill="currentColor" text-anchor="middle" font-size="7.5" fill-opacity="0.9">
    <text x="153" y="68">load</text>
    <text x="307" y="68">fetch</text>
  </g>
  <g fill="currentColor" text-anchor="middle" font-size="8" fill-opacity="0.75">
    <text x="76" y="126">kept when power off</text>
    <text x="230" y="126">wiped when power off</text>
  </g>
  <defs><marker id="bb_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The four parts and how data moves between them. A program lives permanently in storage, is loaded into fast volatile RAM to run, and the CPU fetches and executes it while trading data with I/O. Cut the power and RAM empties; storage keeps everything — that volatile-versus-permanent split is the whole reason both exist.</figcaption>
</figure>

Here's the rough shape of it, from largest to smallest:

| Platform | Typical RAM | Typical storage | Typical I/O |
|----------|-------------|-----------------|-------------|
| **Data-center server** | 64 GB – 1 TB+ | Terabytes (SSD/HDD) | Fast networking, no display |
| **Laptop / desktop** | 8 – 64 GB | 256 GB – 2 TB SSD | USB, Wi-Fi, display, keyboard |
| **Smartphone** | 4 – 16 GB | 64 GB – 1 TB flash | Touchscreen, cellular, Wi-Fi |
| **Single-board computer (e.g. Raspberry Pi)** | 1 – 8 GB | microSD or eMMC | USB, Ethernet/Wi-Fi, GPIO |
| **Microcontroller** | 2 KB – 512 KB | a few KB – a few MB flash | GPIO, simple serial buses |

Read down the columns and the scale of the difference is clear: a server can have a million times the memory of a microcontroller. Yet every row has the same four parts. Knowing how much of each a machine has tells you most of what it can — and can't — do, which is exactly the map the next lesson draws.

<div class="knowledge-check" data-quiz data-correct-msg="Right — RAM is fast, volatile working space; storage is the slower, permanent place data lives when the power is off." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the key difference between memory (RAM) and storage?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Memory keeps data permanently; storage is wiped when the power is off</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Memory is fast, volatile working space; storage keeps data permanently when powered off</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">They are two names for the same thing</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The CPU runs instructions** — its fetch-execute loop is the heartbeat of every computer; cores and clock speed describe its capacity, but only roughly.
- **Memory is volatile working space** — RAM holds what's running now and is wiped on power-off; it's often the real ceiling on what a machine can do.
- **Storage is permanent** — SSDs, HDDs, flash, SD cards, and on-chip EEPROM all keep data with the power off, sized to the machine.
- **I/O connects to the world** — networking, USB, displays, and GPIO are how a computer senses and acts, and often the deciding factor in fit.
- **Same parts, different scale** — a server and a microcontroller share all four blocks, separated by a factor of millions.

Next up: those four blocks assume a general-purpose CPU — but some jobs reach for [GPUs, FPGAs & accelerators](/learn/intro-hardware/specialized-compute/) instead.
