---
slug: raspberry-pi
title: Raspberry Pi
entry_type: hardware
category: hw-sbc
description: Raspberry Pi is a popular, low-cost single-board computer used for learning, hobby projects, home servers, and edge devices, running Linux and programmed in Python, C, or Go.
keywords: Raspberry Pi, Pi Zero 2 W, Pi 4, Pi 5, Compute Module, HAT, Raspberry Pi OS, single-board computer, 40-pin header, SDR host
autolink: true
infobox:
  - { label: Type, value: Single-board computer }
  - { label: CPU, value: ARM (Broadcom SoC) }
  - { label: RAM, value: ~512 MB – 16 GB }
  - { label: Runs, value: Raspberry Pi OS / Linux }
  - { label: Typical price, value: ~$15 – $80 }
see_also: [single-board-computer, gpio, nvidia-jetson, beaglebone, home-server, software-defined-radio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "What is an SBC?", url: /learn/intro-hardware/what-is-an-sbc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi
---

**Raspberry Pi** is the popular, low-cost [single-board computer](/reference/single-board-computer/) that defined the category — used for learning, hobby projects, home servers, and edge devices.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="Top view of a Raspberry Pi board. The 40-pin GPIO header runs along the top edge, the Broadcom system-on-chip sits in the centre with RAM beside it, USB and Ethernet jacks line the right edge, HDMI and power connectors line the bottom, and a microSD card slot sits on the left. This layout is the de-facto template that most Pi-compatible boards copy." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="36" y="30" width="388" height="116" rx="7" fill-opacity="0.05" fill="currentColor"/>
    <g stroke-width="1">
      <rect x="70" y="36" width="230" height="11" rx="2"/>
      <line x1="78" y1="36" x2="78" y2="47"/><line x1="90" y1="36" x2="90" y2="47"/>
      <line x1="102" y1="36" x2="102" y2="47"/><line x1="114" y1="36" x2="114" y2="47"/>
      <line x1="126" y1="36" x2="126" y2="47"/><line x1="138" y1="36" x2="138" y2="47"/>
      <line x1="150" y1="36" x2="150" y2="47"/><line x1="162" y1="36" x2="162" y2="47"/>
      <line x1="174" y1="36" x2="174" y2="47"/><line x1="186" y1="36" x2="186" y2="47"/>
      <line x1="198" y1="36" x2="198" y2="47"/><line x1="210" y1="36" x2="210" y2="47"/>
    </g>
    <rect x="150" y="72" width="56" height="46" rx="4" fill-opacity="0.16" fill="currentColor"/>
    <rect x="220" y="80" width="30" height="30" rx="3" fill-opacity="0.1" fill="currentColor"/>
    <rect x="392" y="52" width="32" height="26" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="392" y="90" width="32" height="26" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="70" y="118" width="34" height="20" rx="2" fill-opacity="0.12" fill="currentColor"/>
    <rect x="120" y="118" width="26" height="20" rx="2" fill-opacity="0.12" fill="currentColor"/>
    <rect x="30" y="86" width="10" height="24" rx="2" fill-opacity="0.18" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="185" y="26" text-anchor="middle" font-size="7.5">40-pin GPIO header</text>
    <text x="178" y="99" text-anchor="middle" font-size="9" font-weight="600">SoC</text>
    <text x="235" y="98" text-anchor="middle" font-size="6.5">RAM</text>
    <text x="408" y="46" text-anchor="middle" font-size="7.5">USB</text>
    <text x="408" y="122" text-anchor="middle" font-size="7">Ethernet</text>
    <text x="87" y="132" text-anchor="middle" font-size="6.5">HDMI</text>
    <text x="133" y="132" text-anchor="middle" font-size="6.5">pwr</text>
    <text x="35" y="80" text-anchor="middle" font-size="6.5">SD</text>
  </g>
</svg>
<figcaption>The Raspberry Pi's layout — a 40-pin GPIO header along one edge, a Broadcom SoC and RAM in the middle, and USB, Ethernet, HDMI, power, and a microSD slot around the sides — became the de-facto template that most Pi-compatible boards copy.</figcaption>
</figure>

## Overview

The range runs from the tiny Pi Zero 2 W through the Pi 4 and Pi 5 to the [Compute Module](/reference/compute-module/) for embedding in custom hardware. A Raspberry Pi runs Raspberry Pi OS (a Linux distribution) and is programmed in ordinary languages such as [Python](/reference/python-language/), [C](/reference/c-language/), and [Go](/reference/go-language/) — the same tools and workflow as a desktop Linux machine, which is a large part of why it became the default teaching board.

What sets it apart from a sealed PC is the 40-pin [GPIO](/reference/gpio/) header, which lets code talk directly to electronics, and the *HAT* — an add-on board that stacks onto that header to add hardware. Its real advantage over faster rivals, though, is the ecosystem: an enormous body of documentation, a huge community, and well-maintained OS images mean most projects "just work," which is worth more than raw specs for a board you want to set up once and leave running.

## The lineup

| Model | Rough role | RAM | Notable |
|-------|-----------|-----|---------|
| Pi Zero 2 W | Tiny, low-power | 512 MB | Smallest, cheapest, Wi-Fi |
| Pi 4 | General-purpose | 1–8 GB | Dual HDMI, USB 3, gigabit |
| Pi 5 | Fastest flagship | 2–16 GB | PCIe, much higher performance |
| Compute Module | Embedded module | 1–16 GB | Needs a custom carrier board |

## Where it fits

For most projects the Pi is the default choice: cheap, well documented, and broadly supported. A Raspberry Pi by the antenna can run GopherTrunk as a small, low-power [SDR](/reference/software-defined-radio/) capture node, hosting an RTL-SDR or similar dongle and decoding locally while drawing only a few watts — headless, fanless, and easy to leave in place. When you need GPU compute at the edge, the [NVIDIA Jetson](/reference/nvidia-jetson/) is an alternative; when you need stronger real-time I/O, look at the [BeagleBone](/reference/beaglebone/); and when you want more of it as a [home server](/reference/home-server/), the Pi handles that too.

## Sources

[^wiki]: [Raspberry Pi](https://en.wikipedia.org/wiki/Raspberry_Pi) — Wikipedia, on the models and uses of the Raspberry Pi.
