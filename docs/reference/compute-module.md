---
slug: compute-module
title: Compute Module
entry_type: hardware
category: hw-sbc
description: A Compute Module is a single-board computer stripped to a small module without ports, designed to be soldered or socketed into a custom carrier board for embedded products.
keywords: Raspberry Pi Compute Module, CM4, CM5, SO-DIMM module, carrier board, system on module, embedded SBC, custom carrier
aka: [Raspberry Pi Compute Module, CM4]
affiliate: true
product:
  name: "Raspberry Pi Compute Module (CM4/CM5)"
  brand: Raspberry Pi
  category: Single-board computer
  lowPrice: "48"
  highPrice: "62"
  url: https://www.amazon.com/s?k=Raspberry+Pi+Compute+Module+4&tag=gophertrunk-20
infobox:
  - { label: Type, value: Embeddable SBC module }
  - { label: Form, value: SO-DIMM or board-to-board }
  - { label: Needs, value: A custom carrier board }
  - { label: Runs, value: Linux }
  - { label: Best-known, value: Raspberry Pi CM4 / CM5 }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=Raspberry+Pi+Compute+Module+4&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [raspberry-pi, single-board-computer, system-on-a-chip, hat-add-on-board, gpio, embedded-system]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi#Compute_Module
faq:
  - q: "Can I run GopherTrunk on a Compute Module?"
    a: "Yes — a CM4/CM5 runs the same Raspberry Pi OS and the same pure-Go ARM64 GopherTrunk binary as a full Pi. The catch is that a Compute Module is just the module: it needs a carrier board to expose USB, Ethernet, and power before you can plug in an SDR. For most builds, a standard Raspberry Pi is far simpler and the default recommendation."
  - q: "Do I need a carrier board?"
    a: "Yes. A Compute Module has no ports of its own — it mates through an edge connector into a carrier board that provides USB, Ethernet, GPIO, and power. You either buy an off-the-shelf carrier or design your own, which is the whole reason to choose a module over a stock board."
  - q: "When is a Compute Module the right choice for GopherTrunk?"
    a: "Only when you're building a custom, embedded capture node — a sealed, antenna-mounted product with exactly the connectors it needs — and the engineering effort pays off at volume. For a normal home or hobby setup, a full Raspberry Pi is cheaper and much less work."
  - q: "Does it decode anything a full Pi can't?"
    a: "No. A CM4/CM5 is the same silicon as the matching Pi, so it decodes exactly the same signals — and no SBC changes the encryption wall: GopherTrunk cannot decode AES-protected traffic on any host."
---

**A Compute Module** is a [single-board computer](/reference/single-board-computer/) stripped down to a small module — the processor, memory, and storage — without the usual ports, meant to plug into a custom carrier board.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A Compute Module plugging into a carrier board. The small module carries only the system-on-chip, RAM, and storage, and its edge connector mates into a slot on a larger custom carrier board. The carrier is where the product designer places the actual ports — USB, Ethernet, GPIO, power — laid out exactly as the product needs." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="96" width="400" height="70" rx="6" fill-opacity="0.05" fill="currentColor"/>
    <rect x="150" y="120" width="160" height="10" rx="2" fill-opacity="0.2" fill="currentColor"/>
    <rect x="150" y="22" width="160" height="66" rx="4" fill-opacity="0.1" fill="currentColor"/>
    <rect x="164" y="34" width="30" height="24" rx="2" fill-opacity="0.2" fill="currentColor"/>
    <rect x="204" y="34" width="20" height="24" rx="2" fill-opacity="0.16" fill="currentColor"/>
    <rect x="234" y="34" width="20" height="24" rx="2" fill-opacity="0.16" fill="currentColor"/>
    <path d="M175 88 V120 M285 88 V120" stroke-width="1.1"/>
    <rect x="46" y="120" width="14" height="30" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="66" y="120" width="14" height="30" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="336" y="120" width="24" height="18" rx="2" fill-opacity="0.14" fill="currentColor"/>
    <rect x="370" y="120" width="24" height="18" rx="2" fill-opacity="0.14" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="230" y="19" text-anchor="middle" font-size="9" font-weight="600">Compute Module</text>
    <text x="179" y="50" text-anchor="middle" font-size="7.5">SoC</text>
    <text x="214" y="50" text-anchor="middle" font-size="7">RAM</text>
    <text x="244" y="50" text-anchor="middle" font-size="7">eMMC</text>
    <text x="230" y="128" text-anchor="middle" font-size="7.5" fill-opacity="0.9">edge connector</text>
    <text x="230" y="160" text-anchor="middle" font-weight="600">custom carrier board</text>
    <text x="63" y="143" text-anchor="middle" font-size="7">ports</text>
    <text x="365" y="132" text-anchor="middle" font-size="7">ports</text>
  </g>
</svg>
<figcaption>A Compute Module holds only the brains — SoC, RAM, and storage — and mates through an edge connector into a carrier board that the product designer lays out with exactly the ports and shape the product requires.</figcaption>
</figure>

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**A module — it needs a carrier board.** The Raspberry Pi Compute Module (CM4/CM5, ~$55) is
the same silicon as a full Pi but with no ports of its own: it plugs into a
[carrier board](/reference/hat-add-on-board/) that provides USB, Ethernet, and power. It runs
the same pure-Go ARM64 GopherTrunk binary, but it's aimed at **custom, embedded capture
nodes** built at volume, not everyday setups. For a normal build, a full
[Raspberry Pi](/reference/raspberry-pi/) is far simpler and the default recommendation. No
SBC decodes [AES encryption](/police-scanner-encryption/). Compare on
[best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/).
</div>

## Overview

The best-known example is the Raspberry Pi Compute Module (CM4, CM5), but the idea is general: take the [system on a chip](/reference/system-on-a-chip/) and memory of a board like the [Raspberry Pi](/reference/raspberry-pi/) and put it on a compact module that breaks all of its signals out to an edge connector. A product designer then lays out a *carrier board* that routes the [GPIO](/reference/gpio/), USB, Ethernet, and power exactly as the product needs, rather than working around a fixed consumer layout.

This separation lets one well-supported compute module serve many different products. The same module drops into a digital-signage player, an industrial gateway, or a camera, and each carrier exposes only the connectors that product uses. It also decouples the two design cadences: the module vendor handles the tricky high-speed SoC-to-memory routing, while your team designs the comparatively simple carrier.

## Module vs stock board

| | Stock SBC | Compute Module |
|---|-----------|----------------|
| Ports | Fixed, on the board | You choose, on the carrier |
| Shape / size | Fixed | Whatever the product needs |
| Effort to deploy | Plug in and go | Design + build a carrier |
| Best for | Prototypes, hobby, general use | Volume embedded products |
| Extending | [HAT](/reference/hat-add-on-board/) on the header | Integrated into the carrier |

## Where it fits

A Compute Module is the choice when an [embedded system](/reference/embedded-system/) needs the brains of an SBC but its own enclosure, connectors, and shape — a step beyond bolting a [HAT](/reference/hat-add-on-board/) onto a stock board. In a GopherTrunk-style product, a Compute Module on a custom carrier could host the SDR front end, timing, and storage in a sealed, antenna-mounted capture node with only the connectors it actually uses. The trade-off is engineering effort: you design, lay out, and manufacture the carrier yourself, which only pays off past a certain volume.

## Running GopherTrunk on a Compute Module

A Compute Module is the same silicon as the matching [Raspberry Pi](/reference/raspberry-pi/),
so its GopherTrunk abilities equal that Pi's — with the crucial caveat that the module has no
ports of its own and depends on a carrier board to expose them:

- **Architecture** — the CM4 and CM5 are ARM64 (aarch64), running the same static `linux/arm64` GopherTrunk binary from the [downloads page](/downloads.html) as a full Pi — no vendor toolchain.
- **CPU** — a CM4 carries the Pi 4's quad Cortex-A72 (1.5 GHz); a CM5 carries the Pi 5's quad Cortex-A76 (2.4 GHz). Real-time DSP capacity is identical to those boards: a couple of channels on the CM4, comfortable multi-channel work on the CM5.
- **RAM** — the same options as the matching Pi (up to 8 GB on the CM4, up to 16 GB on the CM5), enough for recording, the web console, and several systems.
- **USB** — the module breaks USB out to the carrier, so the number and speed (USB 2.0 vs 3.0) of ports for your SDR dongle(s) is whatever the carrier board provides — check that it offers USB 3.0 if you plan on wideband Airspy capture or [several dongles](/multi-dongle-sdr-setup/).
- **Storage** — most modules include onboard eMMC (a "Lite" variant uses microSD on the carrier), and the carrier can route PCIe to an [NVMe](/reference/nvme/) drive for continuous IQ recording and the call database.
- **Power / thermals** — draw matches the equivalent Pi; a CM5-class module under sustained load wants a heatsink, and cooling is designed into the carrier/enclosure since the module has none.
- **OS / networking** — runs Raspberry Pi OS (64-bit), the most turnkey Linux for GopherTrunk; Ethernet and Wi-Fi come from the carrier, so a well-chosen carrier gives you the same headless [web console](/what-do-i-need-for-gophertrunk/) access as a stock Pi.

**Bottom line:** a Compute Module runs exactly the workload of the Pi it's based on — a
couple of SDRs with recording (CM4) up through comfortable multi-channel work (CM5) — once
it's on a carrier board that supplies the USB, storage, and networking it needs.

## Where to buy

A Compute Module makes sense only for a custom, embedded GopherTrunk build — remember it's a
*module* and needs a carrier board to expose USB, Ethernet, and power before you can plug in
an SDR. For a normal setup, a full [Raspberry Pi](/reference/raspberry-pi/) is cheaper, needs
no carrier, and is the default recommendation — see
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/s?k=Raspberry+Pi+Compute+Module+4&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Raspberry Pi Compute Module](https://en.wikipedia.org/wiki/Raspberry_Pi#Compute_Module) — Wikipedia, on the modular, carrier-board form of the Raspberry Pi.
