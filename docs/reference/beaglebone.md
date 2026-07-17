---
slug: beaglebone
title: BeagleBone
entry_type: hardware
category: hw-sbc
description: BeagleBone is an open-source single-board computer known for strong real-time I/O, with many GPIO pins and onboard programmable real-time units, favored for industrial control.
keywords: BeagleBone, BeagleBone Black, PRU, programmable real-time unit, real-time I/O, industrial control, open-source SBC, GPIO, deterministic timing
autolink: true
infobox:
  - { label: Type, value: Single-board computer }
  - { label: CPU, value: ARM (TI SoC) + PRUs }
  - { label: RAM, value: ~512 MB – 4 GB }
  - { label: Runs, value: Linux }
  - { label: Noted for, value: Real-time I/O, dual expansion headers }
  - { label: Typical price, value: ~$50 – $150 }
see_also: [single-board-computer, gpio, raspberry-pi, nvidia-jetson, input-output, microcontroller]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/BeagleBoard#BeagleBone
---

**BeagleBone** is an open-source [single-board computer](/reference/single-board-computer/) known for strong real-time I/O — many [GPIO](/reference/gpio/) pins and onboard programmable real-time units (PRUs).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 178" role="img" aria-label="Block diagram of a BeagleBone. A central ARM CPU runs Linux, but two small programmable real-time units sit beside it on the same chip, each wired to the board's two long expansion pin headers. The PRUs execute deterministic timing-critical code independently of the Linux scheduler, driving the many GPIO pins directly." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="30" width="400" height="118" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="150" y="52" width="90" height="44" rx="4" fill-opacity="0.14" fill="currentColor"/>
    <rect x="266" y="52" width="46" height="20" rx="3" fill-opacity="0.2" fill="currentColor"/>
    <rect x="266" y="80" width="46" height="20" rx="3" fill-opacity="0.2" fill="currentColor"/>
    <g stroke-width="1">
      <rect x="52" y="126" width="164" height="9" rx="1.5"/>
      <rect x="244" y="126" width="164" height="9" rx="1.5"/>
    </g>
    <path d="M240 62 H266" stroke-width="1.1"/>
    <path d="M240 90 H266" stroke-width="1.1"/>
    <path d="M289 100 V126" stroke-width="1.1"/>
    <path d="M195 96 V126" stroke-width="1.1"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="195" y="78" text-anchor="middle" font-size="9" font-weight="600">ARM CPU</text>
    <text x="195" y="90" text-anchor="middle" font-size="7.5" fill-opacity="0.85">Linux</text>
    <text x="289" y="66" text-anchor="middle" font-size="8" font-weight="600">PRU 0</text>
    <text x="289" y="94" text-anchor="middle" font-size="8" font-weight="600">PRU 1</text>
    <text x="134" y="147" text-anchor="middle">header P8</text>
    <text x="326" y="147" text-anchor="middle">header P9</text>
    <text x="230" y="24" text-anchor="middle" font-size="8" fill-opacity="0.9">deterministic real-time units alongside the Linux CPU</text>
  </g>
</svg>
<figcaption>Beyond the Linux-running ARM CPU, a BeagleBone carries two programmable real-time units on the same chip; because they run outside the OS scheduler, they can bit-bang protocols and sample inputs with cycle-accurate timing across the board's two long expansion headers.</figcaption>
</figure>

## Overview

The PRUs are small, deterministic processors alongside the main ARM CPU, which lets a BeagleBone handle precise, timing-critical signalling that a general-purpose Linux board struggles with. On an ordinary SBC the OS scheduler can preempt your code at any moment, so software-driven waveforms jitter; a PRU runs a tight loop with no operating system underneath it, so its timing is repeatable to the clock cycle.

Combined with a generous pin count spread across two long expansion headers, this makes the BeagleBone a favourite for industrial control, motor drivers, and electronics-heavy projects. It runs Linux like other SBCs and is fully open-source down to the board design, which appeals to product developers who want to fork the hardware itself rather than just the software.

## How it works

The split of duties between the Linux CPU and the PRUs is the whole point of the board:

| Job | Runs on | Why |
|-----|---------|-----|
| Networking, filesystem, apps | ARM CPU (Linux) | Needs a full OS and libraries |
| Cycle-accurate pin timing | PRU | No scheduler jitter, deterministic loops |
| Bulk GPIO / bus signalling | PRU + headers | Direct hardware access, hundreds of pins |
| Coordination / data hand-off | Shared memory | CPU sets up work, PRU executes it |

## Where it fits

The BeagleBone is the SBC alternative to the [Raspberry Pi](/reference/raspberry-pi/) when [I/O](/reference/input-output/) and determinism matter more than raw cost or community size. For general-purpose use a Pi is simpler and cheaper; for GPU work at the edge see the [NVIDIA Jetson](/reference/nvidia-jetson/). In a GopherTrunk context the BeagleBone is rarely the decode host itself, but its real-time pins suit the jobs *around* a capture node — precise PPS/GPS timing, antenna-relay switching, or driving a rotator — where jitter would otherwise creep in.

## Sources

[^wiki]: [BeagleBone](https://en.wikipedia.org/wiki/BeagleBoard#BeagleBone) — Wikipedia, on the BeagleBoard family and its real-time I/O.
