---
slug: rock-pi
title: Rock Pi (Radxa)
entry_type: hardware
category: hw-sbc
description: Rock Pi is a family of single-board computers from Radxa of China, built mainly on Rockchip processors and often offering NVMe, faster networking, or more RAM than a comparably priced Raspberry Pi.
keywords: Rock Pi, Radxa, Rockchip, RK3588, ROCK 5, NVMe SBC, Raspberry Pi alternative, ARM single-board computer, fast storage
aka: [Radxa Rock]
affiliate: true
product:
  name: "Radxa Rock 5B (RK3588)"
  brand: Radxa
  category: Single-board computer
  lowPrice: "132"
  highPrice: "168"
  url: https://www.amazon.com/dp/B0BRL4PCG7?tag=gophertrunk-20
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: Radxa (China) }
  - { label: CPU, value: ARM (Rockchip, e.g. RK3588) }
  - { label: Runs, value: Linux / Android }
  - { label: Known for, value: NVMe, fast I/O, high RAM }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0BRL4PCG7?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [raspberry-pi, single-board-computer, odroid, orange-pi, banana-pi, nvme]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radxa
faq:
  - q: "Can I run GopherTrunk on a Rock 5B?"
    a: "Yes. GopherTrunk is pure Go and cross-compiles to ARM64, so it runs on any Linux SBC with a USB port — no vendor toolchain. The RK3588 Rock 5B is the fastest board here and its PCIe/NVMe storage suits an enthusiast multi-SDR node that also records raw IQ or keeps a large call archive."
  - q: "Why pick a Rock 5B over a Raspberry Pi for GopherTrunk?"
    a: "For its I/O and speed: native NVMe over PCIe sustains the write throughput a busy capture-and-archive node generates, where a Pi's microSD card is slow and wears out under continuous writes. The extra cores and RAM also give more room for decoding many channels. For a simpler single-system node, a Raspberry Pi is cheaper and better documented — and remains the default recommendation."
  - q: "Does its speed help with encrypted channels?"
    a: "No. No SBC changes the encryption wall — GopherTrunk cannot decode AES-protected traffic on any host. Faster hardware only helps you decode and store more unencrypted channels."
  - q: "Is the software as polished as a Pi's?"
    a: "Radxa's images are among the better-supported alternatives, but still trail Raspberry Pi OS. Any 64-bit Linux with an RK3588 image works, since GopherTrunk ships as a static ARM64 binary."
---

**Rock Pi** is a family of [single-board computers](/reference/single-board-computer/) from Radxa of China, built mainly on Rockchip chips and often offering faster I/O than a similarly priced [Raspberry Pi](/reference/raspberry-pi/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A storage comparison. A Raspberry Pi routes its filesystem over a slow microSD card path, shown as a thin pipe. A Rock Pi adds an NVMe solid-state drive on a PCIe lane, shown as a much wider pipe, so recording raw IQ or keeping a large call archive runs far faster." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="28" width="70" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <rect x="30" y="92" width="70" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <path d="M100 45 H320" stroke-width="1.6"/>
    <path d="M100 103 L320 103 L320 121 L100 121 Z" fill-opacity="0.14" fill="currentColor"/>
    <rect x="340" y="30" width="86" height="30" rx="3" fill-opacity="0.1" fill="currentColor"/>
    <rect x="340" y="92" width="86" height="34" rx="3" fill-opacity="0.2" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="65" y="42" font-size="8" font-weight="600">Rock Pi</text>
    <text x="65" y="54" font-size="6.5">RK3588</text>
    <text x="65" y="106" font-size="8" font-weight="600">Rasp. Pi</text>
    <text x="65" y="118" font-size="6.5">Pi 4</text>
    <text x="210" y="40" font-size="7">PCIe lane — wide, fast</text>
    <text x="210" y="115" font-size="7">microSD — narrow, slow</text>
    <text x="383" y="49" font-size="8" font-weight="600">NVMe SSD</text>
    <text x="383" y="106" font-size="7.5">microSD</text>
    <text x="383" y="118" font-size="7.5">card</text>
  </g>
</svg>
<figcaption>The Rock Pi's edge is I/O: an NVMe SSD on a PCIe lane moves data far faster than the microSD card a stock Pi of the same price relies on — the difference that matters when a node records raw IQ or keeps a large archive.</figcaption>
</figure>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The fastest board here — enthusiast multi-SDR host.** The Radxa Rock 5B (RK3588, ~$150)
brings PCIe/NVMe storage and 8 fast cores, ideal for a GopherTrunk node that runs a multi-SDR
pool and records raw IQ or keeps a big call archive — NVMe sustains writes a Pi's microSD
can't. GopherTrunk runs on it as a pure-Go ARM64 binary; Radxa's images are decent but trail
Raspberry Pi OS. For a simple single-system build, a
[Raspberry Pi](/reference/raspberry-pi/) is cheaper and the default pick. No SBC decodes
[AES encryption](/police-scanner-encryption/). See
[best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/).
</div>

## Overview

The line is built around Rockchip SoCs — the powerful RK3588 powers the higher-end ROCK 5 boards — and frequently adds features the Pi lacks at the same price, such as an [NVMe](/reference/nvme/) slot, faster Ethernet, or more RAM. The NVMe slot is the headline: instead of running the whole system off a microSD card, a Rock Pi can boot and store data on a proper solid-state drive over PCIe, which transforms any workload that reads or writes a lot of data.

Boards run Linux or Android and keep a Pi-style [GPIO](/reference/gpio/) header, so much of the Pi accessory ecosystem carries over mechanically. As with most Pi alternatives, software polish trails the Raspberry Pi — vendor images and community depth are not quite at Pi levels — but Radxa's boards are generally among the better-supported of the alternatives.

## Rock Pi vs a stock Pi

| | [Raspberry Pi](/reference/raspberry-pi/) 4 | Rock Pi / ROCK 5 |
|---|-------------|------------------|
| SoC | Broadcom quad-core | Rockchip (RK3588, 8-core) |
| Fast storage | microSD (optional USB SSD) | Native NVMe over PCIe |
| RAM ceiling | Up to 8 GB | Up to 16–32 GB |
| Networking | Gigabit | Gigabit / 2.5 GbE on some |
| Software polish | Excellent | Good, trails the Pi |

## Where it fits

Rock Pi competes with [ODROID](/reference/odroid/), [Orange Pi](/reference/orange-pi/), and [Banana Pi](/reference/banana-pi/), and is a natural pick when fast local storage matters. A GopherTrunk node that records raw IQ or keeps days of decoded calls benefits from NVMe rather than an SD card, which is where a Rock Pi can clearly edge out a stock Pi: SD cards are both slow and prone to wear out under continuous writes, while an NVMe drive sustains the throughput a busy capture-and-archive node generates. The extra RAM and cores also give more headroom for decoding several channels at once.

## Running GopherTrunk on the Rock 5B

The Rock 5B is the fastest board covered here, and its NVMe storage makes it the natural host
for a GopherTrunk node that both decodes a lot and records a lot. What it offers:

- **Architecture** — the RK3588 is ARM64 (aarch64), so the standard static `linux/arm64` GopherTrunk binary from the [downloads page](/downloads.html) runs directly, no vendor toolchain required.
- **CPU** — the full 8-core RK3588 (4× Cortex-A76 up to 2.4 GHz + 4× Cortex-A55) has the most real-time DSP headroom of any board here: comfortably a wideband, multi-SDR pool with many demodulators running at once, not just a single channel.
- **RAM** — 4 GB up to 16/32 GB, ample for large recording buffers, the web console, and following many systems concurrently.
- **USB** — multiple USB 3.0 ports (plus USB-C) carry several SDR dongles with plenty of bandwidth for wideband Airspy capture; a powered hub keeps a bank of [dongles](/multi-dongle-sdr-setup/) stable.
- **Storage** — the headline feature: an M.2 [NVMe](/reference/nvme/) slot on PCIe means continuous IQ recording and a growing call database run on a proper SSD, which sustains write throughput a microSD card can't and won't wear out under constant writes. microSD and eMMC are also available for boot.
- **Power / thermals** — the RK3588 runs hot under sustained decode load, so a heatsink and fan are essential for a 24/7 node; power draw stays modest for the performance.
- **OS / networking** — any 64-bit Linux with an RK3588 image (Armbian, Radxa's Debian/Ubuntu, DietPi); support is good but trails Raspberry Pi OS. It has 2.5 GbE (and gigabit) plus optional Wi-Fi, so a headless node moves recorded IQ and serves the [web console](/what-do-i-need-for-gophertrunk/) without bottlenecking.

**Bottom line:** a Rock 5B comfortably runs a wideband, multi-site SDR pool with fast NVMe
recording — the enthusiast build when both decode count and storage throughput matter.

## Where to buy

The Radxa Rock 5B is the enthusiast pick — the fastest board here, with PCIe/NVMe for a
multi-SDR GopherTrunk node that records raw IQ or archives calls. If you don't need that
speed or storage, a [Raspberry Pi](/reference/raspberry-pi/) is cheaper, simpler, and the
default recommendation — see
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BRL4PCG7?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Radxa](https://en.wikipedia.org/wiki/Radxa) — Wikipedia, on Radxa and its Rock Pi single-board computers.
