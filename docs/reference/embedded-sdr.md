---
slug: embedded-sdr
title: Embedded SDR
entry_type: concept
category: sdr-app-building
description: "Embedded SDR is running software-defined radio on small, low-power computers such as a Raspberry Pi or SBC, within tight CPU, memory, thermal, and power budgets."
keywords: embedded SDR, SDR on Raspberry Pi, SBC SDR, low-power SDR, headless scanner, ARM SDR, thermal limits, USB bandwidth, real-time SDR, portable SDR decoder
aka: [embedded SDR, SDR on a Pi, headless SDR]
autolink: true
infobox:
  - { label: Type, value: Deployment constraint / practice }
  - { label: Idea, value: SDR within a small power/CPU budget }
  - { label: Used in, value: "Field scanners, remote receivers, IoT gateways" }
see_also: [single-board-computer, raspberry-pi, arm-architecture, real-time-dsp, cross-compilation, cooling-and-thermals]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://en.wikipedia.org/wiki/Single-board_computer
---

**Embedded SDR** is software-defined radio deployed on a small, low-power computer — a
[Raspberry Pi](/reference/raspberry-pi/), another [single-board computer](/reference/single-board-computer/),
or an embedded module — rather than a desktop or server.[^sbc] The signal-processing math is
identical to what runs on a workstation, but the surrounding constraints are not: a handful
of [ARM](/reference/arm-architecture/) cores, a gigabyte or two of RAM, passive or minimal
cooling, a few watts of power budget, and a USB bus that must keep up with a firehose of
samples. Embedded SDR is less a distinct technology than the discipline of making a real-time
receiver fit — and stay reliable — inside that envelope.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An antenna and USB SDR feed a small single-board computer whose CPU, memory, and thermal budget are the bottleneck; it outputs decoded data over the network, all within a low power budget." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="esar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <path d="M28 34 L28 62 M18 34 L38 34 M21 28 L35 28" stroke="currentColor" stroke-width="1.2" fill="none"/><text x="28" y="76">antenna</text>
    <rect x="62" y="40" width="60" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="92" y="56">USB SDR</text>
    <rect x="152" y="26" width="150" height="86" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="227" y="20">SBC (few ARM cores, ~2 GB)</text>
    <rect x="164" y="44" width="58" height="22" rx="4" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1"/><text x="193" y="58">DSP + decode</text>
    <rect x="232" y="44" width="58" height="22" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="261" y="58">thermal</text>
    <text x="227" y="88" font-size="7">CPU / RAM / heat = the budget</text>
    <text x="227" y="100" font-size="7">a few watts total</text>
    <rect x="336" y="52" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="371" y="64">decoded</text><text x="371" y="75">over net</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="28" y1="62" x2="60" y2="55" marker-end="url(#esar)"/>
    <line x1="122" y1="53" x2="150" y2="60" marker-end="url(#esar)"/>
    <line x1="302" y1="66" x2="334" y2="66" marker-end="url(#esar)"/>
  </g>
</svg>
<figcaption>Embedded SDR puts the whole receive chain on a small board — the CPU, memory, USB bandwidth, and heat, not the algorithm, are the limiting factors.</figcaption>
</figure>

## How it works

The pipeline is the same as on any host: a USB front end delivers IQ, the software
down-converts and channelizes it, and decoders turn symbols into traffic. What changes is
that every stage now competes for scarce resources. Three limits dominate:

- **CPU throughput.** [Real-time DSP](/reference/real-time-dsp/) must keep pace with the
  sample rate or the input buffer overruns and data is lost. On a few modest cores this
  forces careful budgeting — decimate early to the narrowest rate that preserves the signal,
  avoid redundant filtering, and lean on the ARM NEON [SIMD](/reference/arm-architecture/)
  units where the math allows.
- **Thermal and power.** A small board in a sealed case
  [heats up](/reference/cooling-and-thermals/), and once it hits its limit the firmware
  throttles the clock — so a decoder that passes on the bench can start dropping samples
  after twenty minutes in a hot enclosure. Sustained, not peak, load is what matters, and
  the whole system may need to live inside a few watts.
- **I/O bandwidth.** High sample rates saturate USB and memory bandwidth; a Pi's shared USB
  bus in particular can bottleneck a wideband capture long before the CPU does. Embedded
  designs favour narrower captures and zero-copy sample handling to avoid moving data more
  than necessary.

Because these machines are usually **headless** and unattended, robustness matters as much
as speed: the software should recover from a USB glitch, bound its memory, and keep running
for weeks. Software is typically built with [cross-compilation](/reference/cross-compilation/)
from a fast desktop to the target's ARM architecture, or shipped as a portable binary so no
toolchain is needed on the board itself.

## In practice

Typical embedded-SDR roles are a fixed monitoring receiver at a remote antenna, a portable
field scanner, an ADS-B or AIS feeder, or a networked front end that streams IQ to a bigger
machine. The design instinct is to do just enough on the board — capture, channelize, decode
the target — and push anything heavy (wideband search, machine learning, archival) upstream.

## Relevance to SDR

Embedded SDR is where a great deal of real-world receiving actually happens: the low cost and
low power of an [SBC](/reference/single-board-computer/) plus a cheap dongle make it easy to
leave a receiver running permanently. The constraints shape the software — efficient DSP,
predictable memory, graceful overrun handling — more than any single algorithm does.

**GopherTrunk is built for exactly this environment.** It is a pure-Go decoder that compiles
to a single [static binary](/reference/static-binary/) with no runtime dependencies, so
deploying to a Pi or other ARM board is a file copy, not a dependency hunt, and
[cross-compiling](/reference/cross-compilation/) for `linux/arm64` is a one-line build. Its
decode chain normalizes each channel to a fixed per-protocol rate and sizes the receiver from
that output, keeping the steady-state CPU cost low and independent of the capture rate — the
property that lets it hold real-time on modest cores. GopherTrunk runs headless and is a
natural fit for unattended embedded deployments; the usual practical limits are the host's
sustained CPU and thermal headroom and the USB front end's bandwidth, not the decoder itself.
It does not, and does not need to, offload work to a GPU or accelerator to run in this class
of hardware.

## Sources

[^sbc]: [Single-board computer](https://en.wikipedia.org/wiki/Single-board_computer) — Wikipedia, on the low-power, integrated boards that host embedded SDR. See also [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) for the receive-chain stages these boards must run in real time.
