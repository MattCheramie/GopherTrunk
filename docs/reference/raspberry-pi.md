---
slug: raspberry-pi
title: Raspberry Pi
entry_type: hardware
category: hw-sbc
description: Raspberry Pi is a popular, low-cost single-board computer used for learning, hobby projects, home servers, and edge devices, running Linux and programmed in Python, C, or Go.
keywords: Raspberry Pi, Pi Zero 2 W, Pi 4, Pi 5, Compute Module, HAT, Raspberry Pi OS, single-board computer, 40-pin header, SDR host
autolink: true
affiliate: true
product:
  name: "Raspberry Pi 5 (8GB)"
  brand: Raspberry Pi
  category: Single-board computer
  lowPrice: "70"
  highPrice: "90"
  url: https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20
infobox:
  - { label: Type, value: Single-board computer }
  - { label: CPU, value: ARM (Broadcom SoC) }
  - { label: RAM, value: ~512 MB – 16 GB }
  - { label: Runs, value: Raspberry Pi OS / Linux }
  - { label: Typical price, value: ~$15 – $80 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [single-board-computer, gpio, nvidia-jetson, beaglebone, home-server, software-defined-radio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
  - { title: "What is an SBC?", url: /learn/intro-hardware/what-is-an-sbc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi
faq:
  - q: "Which Raspberry Pi should I run GopherTrunk on?"
    a: "A Raspberry Pi 4 (4GB) or Pi 5 (8GB) is the recommended, best-supported GopherTrunk host. GopherTrunk is pure Go and cross-compiles to ARM64, so it runs on Raspberry Pi OS with nothing but a USB port for your SDR dongle — no vendor toolchain. The Pi 5 has the most headroom for decoding several channels or a small SDR pool; the Pi 4 is the value pick for one or two systems."
  - q: "Can a Pi Zero 2 W run GopherTrunk?"
    a: "For a single control channel, yes — but it is RAM- and CPU-limited, so it is not the board for wideband capture or a multi-SDR pool. Treat the Zero 2 W as a tiny, low-power node for one system and step up to a Pi 4 or Pi 5 for anything busier."
  - q: "Does a faster Pi let me decode encrypted channels?"
    a: "No. No single-board computer changes the encryption wall — GopherTrunk (like every scanner) cannot decode AES-protected traffic on any host. A faster Pi only helps you follow more unencrypted channels at once."
  - q: "Do I need Raspberry Pi OS specifically?"
    a: "No — any 64-bit Linux for the Pi works, since GopherTrunk ships as a static ARM64 binary. Raspberry Pi OS is simply the most documented and turnkey choice, which is why it is the default recommendation."
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

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The recommended GopherTrunk host.** The Raspberry Pi is the best-documented,
best-supported [SDR](/reference/software-defined-radio/) capture node — GopherTrunk is
pure Go and runs as a static ARM64 binary on Raspberry Pi OS with just a USB port for
your dongle. **A Pi 4 (4GB, ~$55) or Pi 5 (8GB, ~$80) is the mainstream pick;** the Pi 5
has the most headroom for multiple channels. The **Pi Zero 2 W (~$18)** works for a single
control channel but is RAM-limited — not for wideband or multi-SDR. Unless you have a
specific reason for another board, **start with a Pi.** No Pi decodes
[AES encryption](/police-scanner-encryption/) — no host does. See
[best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/) and the
[Raspberry Pi SDR scanner guide](/raspberry-pi-sdr-scanner/).
</div>

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

## Running GopherTrunk on a Raspberry Pi

The Raspberry Pi is the reference GopherTrunk host — the combination of capable hardware and
the most turnkey software makes it the board every setup guide assumes. Here is what each
part of the Pi brings to a decode node:

- **Architecture** — the Pi 4, Pi 5, Zero 2 W (and the [Compute Modules](/reference/compute-module/)) are all ARM64 (aarch64). GopherTrunk ships as a static ARM64 Go binary you drop on the board and run — no cross-compiler, no vendor toolchain, no SoapySDR. Grab the `linux/arm64` build from the [downloads page](/downloads.html).
- **CPU** — the Pi 5's quad Cortex-A76 at 2.4 GHz has ample headroom for real-time DSP across several demodulators; the Pi 4's quad Cortex-A72 at 1.8 GHz comfortably handles a couple of channels. The Zero 2 W's quad Cortex-A53 is enough for a single control channel but not for a multi-SDR pool or [wideband channelizing](/reference/software-defined-radio/).
- **RAM** — the Zero 2 W's 512 MB suits one channel; a Pi 4 or Pi 5 with 4–8 GB has room for recording, the web console, and following several systems at once.
- **USB** — the Pi 4 and Pi 5 expose two USB 3.0 and two USB 2.0 ports, plenty for one or more SDR dongles; USB 3.0 bandwidth matters for wideband Airspy capture. Run several dongles from a powered hub so the board isn't back-powering them. The Zero 2 W has only a micro-USB (OTG) port, so you'll need an adapter and it's really a single-dongle board. See [running several dongles](/multi-dongle-sdr-setup/).
- **Storage** — every model boots from microSD; the Pi 4 can add a USB SSD and the Pi 5 a proper [NVMe](/reference/nvme/) drive over its PCIe connector, which is the better choice for continuous IQ recording or a growing call database than a wear-prone SD card.
- **Power / thermals** — a Pi 4/5 draws only a few watts and is fine 24/7; the Pi 5 runs hot under sustained load, so fit the active cooler for an always-on node. The Zero 2 W sips power and needs no cooling.
- **OS / networking** — Raspberry Pi OS (64-bit) is the most turnkey Linux for GopherTrunk, with Ubuntu also well supported. The Pi 4/5 have gigabit Ethernet plus Wi-Fi and the Zero 2 W has Wi-Fi, so you can run the board headless by the antenna and reach the [web console](/what-do-i-need-for-gophertrunk/) from your desk.

**Bottom line:** a Pi 4 or Pi 5 comfortably runs a couple of SDRs with recording and the web
console — the mainstream GopherTrunk build — while a Pi Zero 2 W is a tiny, low-power host
for a single control channel.

## Where to buy

The Raspberry Pi is the recommended GopherTrunk host, and three models cover almost every
build. The Pi 4 and Pi 5 are the mainstream picks; the Zero 2 W is a tiny, low-power option
for a single control channel only. See the
[Raspberry Pi SDR scanner guide](/raspberry-pi-sdr-scanner/) for a full walkthrough and
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/)
to compare the alternatives.

| Model | Best for | Approx. price | Buy |
|-------|----------|--------------|-----|
| Raspberry Pi 5 (8GB) | Flagship — most headroom, multiple channels / small SDR pool | ~$80 | <a class="btn btn--buy" href="https://www.amazon.com/dp/B0CK2FCG1K?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a> |
| Raspberry Pi 4 Model B (4GB) | Value mainstream host, huge community | ~$55 | <a class="btn btn--buy" href="https://www.amazon.com/dp/B09TTNF8BT?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a> |
| Raspberry Pi Zero 2 W | Tiny/low-power, single control channel only (RAM-limited) | ~$18 | <a class="btn btn--buy" href="https://www.amazon.com/s?k=Raspberry+Pi+Zero+2+W&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a> |

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Raspberry Pi](https://en.wikipedia.org/wiki/Raspberry_Pi) — Wikipedia, on the models and uses of the Raspberry Pi.
