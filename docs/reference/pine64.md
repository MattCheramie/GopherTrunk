---
slug: pine64
title: PINE64
entry_type: hardware
category: hw-sbc
description: "PINE64 makes low-cost, community-driven single-board computers — the Quartz64 (RK3566) and ROCK64 (RK3328) — that run GopherTrunk as an ARM64 binary, sold mostly direct with intermittent Amazon stock."
keywords: PINE64, Quartz64, ROCK64, RK3566, RK3328, open source SBC, Raspberry Pi alternative, ARM single-board computer, community SBC
aka: [PINE64, Pine64]
autolink: true
affiliate: true
product:
  name: "PINE64 single-board computer (Quartz64 / ROCK64)"
  brand: PINE64
  category: Single-board computer
  lowPrice: "35"
  highPrice: "90"
  url: https://www.amazon.com/s?k=PINE64+single+board+computer&tag=gophertrunk-20
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: PINE64 }
  - { label: CPU, value: "ARM (Rockchip RK3566 / RK3328)" }
  - { label: Runs, value: Linux / BSD }
  - { label: Known for, value: "Low cost, open-source community focus" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/s?k=PINE64+single+board+computer&tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [raspberry-pi, radxa, khadas, rock-pi, orange-pi, odroid, single-board-computer, nvme]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Pine64
faq:
  - q: "Can I run GopherTrunk on a PINE64 board?"
    a: "Yes. GopherTrunk is pure Go and cross-compiles to ARM64, so it runs on any 64-bit Linux SBC with a USB port — including the Quartz64 (RK3566) and ROCK64 (RK3328). These are modest boards, best for a single system or a couple of channels rather than a big wideband pool."
  - q: "Quartz64 or ROCK64 for GopherTrunk?"
    a: "The Quartz64 (RK3566, newer) is the better pick — more RAM options, an eMMC socket, and a PCIe footprint on some models. The ROCK64 (RK3328) is older and cheaper, fine for a low-power single-channel node. Neither has the DSP headroom of an RK3588 board like the Radxa Rock 5."
  - q: "Where do I buy PINE64 boards?"
    a: "Mostly direct from the PINE64 store (pine64.com) and resellers like ameriDroid; some models (notably the older PINE A64) appear on Amazon, but Quartz64/ROCK64 stock there is intermittent and third-party. The button is a tagged Amazon search that resolves to whatever's currently listed."
  - q: "Does a PINE64 board decode encrypted channels?"
    a: "No. No SBC changes the encryption wall — GopherTrunk cannot decode AES-protected traffic on any host. A faster board only lets you follow more unencrypted channels at once."
---

**PINE64** makes low-cost, community-driven
[single-board computers](/reference/single-board-computer/) — the **Quartz64** (Rockchip
RK3566) and **ROCK64** (RK3328) among them — that run GopherTrunk as an ARM64 binary. The
project's focus is open, affordable hardware and a strong Linux/BSD community rather than raw
performance, and its boards are sold mostly direct with only intermittent
[Amazon](/reference/raspberry-pi/) stock.[^wiki]

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Low-cost, open-community boards for a modest node.** PINE64's Quartz64 (RK3566) and ROCK64
(RK3328) run GopherTrunk as a pure-Go ARM64 binary and suit a single system or a couple of
channels — not a big wideband pool, where an RK3588 board like the
[Radxa Rock 5](/reference/radxa/) has far more DSP headroom. **Sold mostly direct**
(pine64.com / ameriDroid); Amazon stock is intermittent and third-party, so the button is a
tagged search. Software support trails the [Raspberry Pi](/reference/raspberry-pi/), which
stays the default for a simple node. No SBC decodes
[AES encryption](/police-scanner-encryption/). See
[best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/).
</div>

## Overview

PINE64 grew out of a community around cheap ARM hardware and now spans single-board computers,
the PinePhone, and the Pinebook laptops. Its SBCs are built on Rockchip and Allwinner SoCs and
priced aggressively, with the trade-off that vendor software is thinner than the
[Raspberry Pi](/reference/raspberry-pi/)'s — you lean more on the community (Armbian, mainline
Linux, the PINE64 wiki) to get an image running. Two boards are the relevant GopherTrunk hosts:

- **Quartz64** — Rockchip **RK3566** (quad Cortex-A55), 4–8 GB LPDDR4, eMMC socket, microSD,
  and a PCIe footprint on some models. The newer, more capable choice.
- **ROCK64** — Rockchip **RK3328** (quad Cortex-A53), up to 4 GB. Older and cheaper, a fine
  low-power single-channel node.

Both keep a [GPIO](/reference/gpio/) header and run 64-bit Linux; both are ARM64, so
GopherTrunk's static binary drops on without a toolchain.

## Running GopherTrunk on a PINE64 board

- **Architecture** — the RK3566 and RK3328 are ARM64 (aarch64), so the static `linux/arm64`
  GopherTrunk binary from the [downloads page](/downloads.html) runs directly.
- **CPU** — enough for a single [control channel](/reference/control-channel/) or a couple of
  demodulators; these are Cortex-A55/A53 class parts, so they don't have the real-time headroom
  of an RK3588 board for a wideband, multi-SDR pool.
- **RAM** — a Quartz64 with 8 GB has room for recording buffers, the web console, and following
  a few systems; the ROCK64's 4 GB ceiling is tighter.
- **USB** — USB ports carry an SDR dongle; for more than one, use a powered hub so the board
  isn't back-powering them (see [running several dongles](/multi-dongle-sdr-setup/)).
- **Storage** — boot from microSD or eMMC; a Quartz64 with a PCIe/[NVMe](/reference/nvme/)
  option is the better host for continuous IQ recording than a wear-prone SD card.
- **OS / networking** — Armbian or a community image; gigabit Ethernet (plus Wi-Fi on some) lets
  a headless node serve the [web console](/what-do-i-need-for-gophertrunk/) from by the antenna.

**Bottom line:** a Quartz64 or ROCK64 is a cheap, low-power GopherTrunk node for one system or a
couple of channels — reach for an RK3588 board like the [Radxa Rock 5](/reference/radxa/) when
you want a bigger wideband pool.

## Where to buy

PINE64 boards are sold mostly **direct from the PINE64 store and resellers like ameriDroid**;
some models turn up on Amazon (the older PINE A64 has a stable listing), but Quartz64/ROCK64
stock there is intermittent and third-party, so the button is a tagged Amazon search that
resolves to whatever's currently listed. For a plug-and-play, best-documented node a
[Raspberry Pi](/reference/raspberry-pi/) is the simpler pick; for more speed and native
PCIe/NVMe, see [Radxa](/reference/radxa/) or [Khadas](/reference/khadas/). Compare them all in
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/).

<a class="btn btn--buy" href="https://www.amazon.com/s?k=PINE64+single+board+computer&tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Pine64](https://en.wikipedia.org/wiki/Pine64) — Wikipedia, on PINE64 and its single-board computers including the Quartz64 and ROCK64.
