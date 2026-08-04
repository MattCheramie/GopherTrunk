---
slug: libre-computer
title: Libre Computer
entry_type: hardware
category: hw-sbc
description: Libre Computer makes low-cost single-board computers with open, mainline Linux support and Raspberry Pi compatibility, such as Le Potato and Renegade, built on Amlogic and Rockchip chips.
keywords: Libre Computer, Le Potato, AML-S905X-CC, Renegade, open source SBC, mainline Linux, Raspberry Pi compatible, ARM single-board computer, upstream drivers
affiliate: true
product:
  name: "Libre Computer Le Potato (AML-S905X-CC)"
  brand: Libre Computer
  category: Single-board computer
  lowPrice: "31"
  highPrice: "39"
  url: https://www.amazon.com/dp/B074P6BNGZ?tag=gophertrunk-20
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Emphasis, value: Open, mainline software }
  - { label: CPU, value: ARM (Amlogic / Rockchip) }
  - { label: Runs, value: Mainline Linux, Android }
  - { label: Boards, value: Le Potato, Renegade, others }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B074P6BNGZ?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [raspberry-pi, single-board-computer, orange-pi, odroid, rock-pi, gpio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Libre_Computer_Project
  - https://libre.computer/
faq:
  - q: "Can I run GopherTrunk on a Le Potato?"
    a: "Yes. GopherTrunk is pure Go and cross-compiles to ARM64, so it runs on any Linux SBC with a USB port — no vendor toolchain. Le Potato is modest hardware, so treat it as a light single-channel decode node; it's the cheapest Pi alternative and its mainline-Linux support helps an always-on box stay current."
  - q: "Why choose Le Potato over a Raspberry Pi?"
    a: "Two reasons: price — it's the cheapest board here at ~$35 — and Libre Computer's focus on mainline (upstream) Linux support, so an always-on node is less likely to be stranded on a frozen vendor image years later. You give up some performance per dollar, and a Raspberry Pi is still the better-documented default for most builds."
  - q: "Is it powerful enough for multiple channels?"
    a: "Not really — it's modest hardware best suited to a light, single-control-channel decoder. For a multi-SDR pool or wideband work, step up to a Raspberry Pi 4/5 or an RK3588 board."
  - q: "Does it change what can be decoded?"
    a: "No. No SBC changes the encryption wall — GopherTrunk cannot decode AES-protected traffic on any host. Le Potato decodes exactly what any other Linux host does, just with less headroom."
---

**Libre Computer** is a maker of [single-board computers](/reference/single-board-computer/) that emphasise open, mainline software support and broad [Raspberry Pi](/reference/raspberry-pi/) compatibility.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="Two software stacks compared as layers. The vendor-image path stacks your app on top of a frozen vendor kernel and a one-off board patch, which strands the board when the OS moves on. The Libre Computer path stacks your app on a standard OS on the mainline Linux kernel with upstreamed drivers, so ordinary current software keeps running." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="34" y="30" width="170" height="24" rx="3" fill-opacity="0.06" fill="currentColor"/>
    <rect x="34" y="58" width="170" height="24" rx="3" fill-opacity="0.06" fill="currentColor"/>
    <rect x="34" y="86" width="170" height="24" rx="3" fill-opacity="0.16" fill="currentColor" stroke-dasharray="4 3"/>
    <rect x="34" y="114" width="170" height="24" rx="3" fill-opacity="0.06" fill="currentColor"/>
    <rect x="256" y="30" width="170" height="24" rx="3" fill-opacity="0.06" fill="currentColor"/>
    <rect x="256" y="58" width="170" height="24" rx="3" fill-opacity="0.06" fill="currentColor"/>
    <rect x="256" y="86" width="170" height="52" rx="3" fill-opacity="0.16" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="119" y="21" font-weight="600" font-size="8.5">vendor image</text>
    <text x="119" y="46">your app</text>
    <text x="119" y="74">frozen vendor OS</text>
    <text x="119" y="102" font-size="7.5">one-off board patch</text>
    <text x="119" y="130">hardware</text>
    <text x="341" y="21" font-weight="600" font-size="8.5">Libre Computer</text>
    <text x="341" y="46">your app</text>
    <text x="341" y="74">standard current OS</text>
    <text x="341" y="108" font-weight="600">mainline Linux kernel</text>
    <text x="341" y="122" font-size="7.5">upstreamed drivers + hardware</text>
    <text x="119" y="156" font-size="7.5" fill-opacity="0.9">strands when the OS moves on</text>
    <text x="341" y="156" font-size="7.5" fill-opacity="0.9">keeps working with current software</text>
  </g>
</svg>
<figcaption>Many cheap boards ship a frozen vendor image with a one-off patch that strands them when the OS moves on; Libre Computer's pitch is upstream support — drivers merged into the mainline Linux kernel so standard, current software keeps running.</figcaption>
</figure>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The cheapest Pi alternative — fine for light use.** Le Potato (AML-S905X-CC, ~$35) is
modest hardware, well suited to a light, single-control-channel GopherTrunk decoder.
GopherTrunk runs on it as a pure-Go ARM64 binary, and Libre Computer's mainline-Linux focus
helps an always-on node stay patched for years. It's **not for multi-SDR or wideband** work
— step up to a [Raspberry Pi](/reference/raspberry-pi/) 4/5 for that, which is also the
better-documented default. No SBC decodes
[AES encryption](/police-scanner-encryption/). Compare on
[best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/).
</div>

## Overview

Boards such as Le Potato (AML-S905X-CC) and the Renegade use Amlogic and Rockchip ARM chips and copy the Pi's footprint and 40-pin [GPIO](/reference/gpio/) header, so existing cases and add-ons often fit. On paper they look like any other Pi alternative — the difference is in how the software is delivered.

The project's distinguishing goal is upstream support: getting drivers into the mainline Linux kernel and standard bootloaders so the hardware keeps working with ordinary, current software rather than a vendor's frozen image.[^libre] Many low-cost boards ship a one-off kernel fork that works the day you buy it but never gets updated, so a few years on the board is stuck on an old, insecure OS. Mainlined support avoids that trap because the board rides along with every new kernel release.

## Where it fits

Among Pi alternatives like [Orange Pi](/reference/orange-pi/), [ODROID](/reference/odroid/), and [Rock Pi](/reference/rock-pi/), Libre Computer's pitch is longevity and trust in the software stack rather than raw specs — you may give up a little performance per dollar in exchange for a board that stays current. For an always-on GopherTrunk capture node you intend to leave in place for years, that trade favours Libre: mainline support means OS and security updates are far less likely to strand the board, so a decode node bolted up near an antenna keeps receiving patches without a risky vendor-image migration.

## Running GopherTrunk on Le Potato

Le Potato is modest, mainline-supported hardware — a good fit for a small, always-on
single-channel decoder rather than a busy pool. What it brings to a GopherTrunk node:

- **Architecture** — the Amlogic S905X is ARM64 (aarch64), so it runs the standard static `linux/arm64` GopherTrunk binary from the [downloads page](/downloads.html) with no vendor toolchain, and its mainline-kernel support keeps that binary working across OS updates.
- **CPU** — a quad Cortex-A53 at up to 1.5 GHz handles the light real-time DSP of a single RTL-SDR control channel comfortably, but it will run short for a multi-SDR pool or [wideband channelizing](/reference/software-defined-radio/) — treat it as a one-decoder board.
- **RAM** — up to 2 GB, which covers one channel plus the web console and modest logging; not the board for large in-memory recording buffers.
- **USB** — four USB 2.0 ports, fine for one RTL-SDR dongle (~2.4 MS/s is light); there's no USB 3.0, so heavy wideband Airspy capture and multiple busy dongles are out of scope. Use a powered hub for anything beyond a single dongle — see [multi-dongle setups](/multi-dongle-sdr-setup/).
- **Storage** — microSD plus an optional eMMC module, enough for logs and a modest call database; keep continuous IQ recording off the SD card to avoid wear.
- **Power / thermals** — very low draw and no cooling required, so it's well suited to a fanless, leave-it-running node.
- **OS / networking** — Libre Computer ships mainline-based 64-bit Linux images (their Raspbian/Ubuntu builds) so it stays current; Fast (100 Mbps) Ethernet on board, Wi-Fi via a USB adapter, both plenty for the headless [web console](/what-do-i-need-for-gophertrunk/) and one system's traffic.

**Bottom line:** Le Potato comfortably runs a single control channel as a cheap, low-power,
always-on decoder — step up to a [Raspberry Pi](/reference/raspberry-pi/) for anything busier.

## Where to buy

Le Potato is the budget pick — the cheapest board here — and a fine host for a light,
single-channel GopherTrunk decoder that you want to leave running for years on mainline
Linux. For anything busier, or for the smoothest setup, a
[Raspberry Pi](/reference/raspberry-pi/) is the default recommendation — see
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/dp/B074P6BNGZ?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Libre Computer Project](https://en.wikipedia.org/wiki/Libre_Computer_Project) — Wikipedia, on the project and its boards.
[^libre]: [Libre Computer](https://libre.computer/) — vendor site, on the boards and their open-software focus.
