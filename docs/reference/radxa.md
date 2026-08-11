---
slug: radxa
title: Radxa
entry_type: hardware
category: hw-sbc
description: "Radxa is a Chinese maker of Rockchip-based single-board computers — the Rock 5 series (RK3588) is the fastest SBC class here, with PCIe/NVMe storage ideal for a multi-SDR GopherTrunk node."
keywords: Radxa, Rock 5B, Rock 5C, RK3588, NVMe SBC, Raspberry Pi alternative, ARM single-board computer, fast storage, multi-SDR host
aka: [Radxa, Rock 5]
autolink: true
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
see_also: [rock-pi, raspberry-pi, khadas, pine64, orange-pi, odroid, single-board-computer, nvme]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Radxa
faq:
  - q: "Can I run GopherTrunk on a Radxa Rock 5B?"
    a: "Yes. GopherTrunk is pure Go and cross-compiles to ARM64, so it runs on any Linux SBC with a USB port — no vendor toolchain. The RK3588 Rock 5B is the fastest board covered here, and its PCIe/NVMe storage suits an enthusiast multi-SDR node that records raw IQ or keeps a large call archive."
  - q: "What's the difference between 'Radxa' and 'Rock Pi'?"
    a: "Same company. Radxa is the maker; Rock Pi (and the newer Rock 5 series) is its main single-board-computer line. You'll see both names on listings — the current flagship is the Rock 5B, and the RK3588 inside it is what makes it the top performer here."
  - q: "Why pick a Radxa Rock 5B over a Raspberry Pi for GopherTrunk?"
    a: "For I/O and speed: native NVMe over PCIe sustains the write throughput a busy capture-and-archive node generates, where a Pi's microSD is slow and wears out. The extra cores and RAM give more room for decoding many channels. For a simple single-system node, a Raspberry Pi is cheaper, better documented, and the default recommendation."
  - q: "Does a faster Radxa board decode encrypted channels?"
    a: "No. No SBC changes the encryption wall — GopherTrunk cannot decode AES-protected traffic on any host. Faster hardware only lets you decode and store more unencrypted channels at once."
---

**Radxa** is a Chinese maker of Rockchip-based
[single-board computers](/reference/single-board-computer/) — its **Rock 5** series
(RK3588) is the fastest SBC class covered here, and often offers faster I/O than a
similarly priced [Raspberry Pi](/reference/raspberry-pi/).[^wiki] Radxa's boards are also
documented on this site under the [Rock Pi](/reference/rock-pi/) name, the line's original
branding.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A storage comparison. A Raspberry Pi routes its filesystem over a slow microSD card path, shown as a thin pipe. A Radxa Rock 5B adds an NVMe solid-state drive on a PCIe lane, shown as a much wider pipe, so recording raw IQ or keeping a large call archive runs far faster." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="28" width="70" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <rect x="30" y="92" width="70" height="34" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <path d="M100 45 H320" stroke-width="1.6"/>
    <path d="M100 103 L320 103 L320 121 L100 121 Z" fill-opacity="0.14" fill="currentColor"/>
    <rect x="340" y="30" width="86" height="30" rx="3" fill-opacity="0.1" fill="currentColor"/>
    <rect x="340" y="92" width="86" height="34" rx="3" fill-opacity="0.2" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="65" y="42" font-size="8" font-weight="600">Radxa</text>
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
<figcaption>Radxa's edge is I/O: an NVMe SSD on a PCIe lane moves data far faster than the microSD card a stock Pi of the same price relies on.</figcaption>
</figure>

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BRL4PCG7?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The fastest board here — enthusiast multi-SDR host.** The Radxa Rock 5B (RK3588, ~$150)
brings PCIe/[NVMe](/reference/nvme/) storage and 8 fast cores, ideal for a GopherTrunk node
that runs a multi-SDR pool and records raw IQ or keeps a big call archive — NVMe sustains
writes a Pi's microSD can't. GopherTrunk runs on it as a pure-Go ARM64 binary; Radxa's images
are decent but trail Raspberry Pi OS. For a simple single-system build, a
[Raspberry Pi](/reference/raspberry-pi/) is cheaper and the default pick. No SBC decodes
[AES encryption](/police-scanner-encryption/). See
[best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/).
</div>

## Overview

Radxa builds its computers around Rockchip SoCs — the powerful **RK3588** drives the
higher-end [Rock 5](/reference/rock-pi/) boards — and frequently adds features the Pi lacks at
the same price: an [NVMe](/reference/nvme/) slot, faster Ethernet, or more RAM. The NVMe slot
is the headline: instead of running the whole system off a microSD card, a Rock 5 can boot and
store data on a proper solid-state drive over PCIe, which transforms any workload that reads or
writes a lot of data — exactly what a capture-and-archive GopherTrunk node does.

Boards run Linux or Android and keep a Pi-style [GPIO](/reference/gpio/) header, so much of the
Pi accessory ecosystem carries over mechanically. As with most Pi alternatives, software polish
trails the [Raspberry Pi](/reference/raspberry-pi/) — vendor images and community depth are not
quite at Pi levels — but Radxa's are generally among the better-supported of the alternatives.

## The Rock 5 lineup

| Board | SoC | Rough role | Notable |
|-------|-----|-----------|---------|
| Rock 5B / 5B+ | RK3588 (8-core) | Flagship | PCIe 3.0 NVMe, up to 16–32 GB RAM, 8K, 2.5 GbE |
| Rock 5C | RK3588S2 (8-core) | Value flagship | Compact, PCIe, HDMI 8K |
| Rock 3 / 4 series | RK3566 / RK3399 | Mid-range | Cheaper, still Pi-class or better |

The Rock 5B is the one to reach for as a GopherTrunk host: its RK3588 has the most real-time
DSP headroom of any board here, and its PCIe/NVMe storage is the reason to pick it over a Pi.

## Running GopherTrunk on a Rock 5B

- **Architecture** — the RK3588 is ARM64 (aarch64), so the standard static `linux/arm64`
  GopherTrunk binary from the [downloads page](/downloads.html) runs directly, no vendor
  toolchain required.
- **CPU** — the full 8-core RK3588 (4× Cortex-A76 up to 2.4 GHz + 4× Cortex-A55) has the most
  headroom here: comfortably a wideband, multi-SDR pool with many demodulators running at
  once, not just a single channel.
- **RAM** — 4 GB up to 16/32 GB, ample for large recording buffers, the web console, and
  following many systems concurrently.
- **USB** — multiple USB 3.0 ports (plus USB-C) carry several SDR dongles with bandwidth to
  spare for wideband [Airspy](/reference/airspy/) capture; a powered hub keeps a bank of
  [dongles](/multi-dongle-sdr-setup/) stable.
- **Storage** — the headline: an M.2 [NVMe](/reference/nvme/) slot on PCIe means continuous IQ
  recording and a growing call database run on a proper SSD, sustaining write throughput a
  microSD card can't and won't wear out under constant writes.
- **Power / thermals** — the RK3588 runs hot under sustained decode load, so a heatsink and fan
  are essential for a 24/7 node; power draw stays modest for the performance.
- **OS / networking** — any 64-bit Linux with an RK3588 image (Armbian, Radxa's Debian/Ubuntu,
  DietPi); support is good but trails Raspberry Pi OS. 2.5 GbE (and gigabit) plus optional
  Wi-Fi let a headless node move recorded IQ and serve the
  [web console](/what-do-i-need-for-gophertrunk/) without bottlenecking.

**Bottom line:** a Rock 5B comfortably runs a wideband, multi-site SDR pool with fast NVMe
recording — the enthusiast build when both decode count and storage throughput matter.

## Where to buy

The Radxa Rock 5B is the enthusiast pick — the fastest board here, with PCIe/NVMe for a
multi-SDR GopherTrunk node that records raw IQ or archives calls. If you don't need that speed
or storage, a [Raspberry Pi](/reference/raspberry-pi/) is cheaper, simpler, and the default
recommendation. For the full walkthrough of this board see [Rock Pi](/reference/rock-pi/), and
to compare alternatives — [Khadas](/reference/khadas/), [PINE64](/reference/pine64/),
[Orange Pi](/reference/orange-pi/), [ODROID](/reference/odroid/) — see
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0BRL4PCG7?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Radxa](https://en.wikipedia.org/wiki/Radxa) — Wikipedia, on Radxa and its Rock single-board computers.
