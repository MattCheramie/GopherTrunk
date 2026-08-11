---
slug: khadas
title: Khadas
entry_type: hardware
category: hw-sbc
description: "Khadas makes compact, well-built VIM and Edge single-board computers on Amlogic and Rockchip SoCs — capable ARM64 GopherTrunk hosts with a cloud OS installer, at a premium over a Raspberry Pi."
keywords: Khadas, Khadas VIM4, Khadas Edge2, Amlogic A311D2, RK3588S, ARM single-board computer, Raspberry Pi alternative, OOWOW, compact SBC
aka: [Khadas, Khadas VIM, Khadas Edge]
autolink: true
affiliate: true
product:
  name: "Khadas VIM4 (Amlogic A311D2)"
  brand: Khadas
  category: Single-board computer
  lowPrice: "180"
  highPrice: "230"
  url: https://www.amazon.com/dp/B09ZTLLZLZ?tag=gophertrunk-20
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Maker, value: Khadas }
  - { label: CPU, value: "ARM (Amlogic A311D2 / Rockchip RK3588S)" }
  - { label: Runs, value: Linux / Android }
  - { label: Known for, value: "Compact build, OOWOW cloud installer, M.2" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B09ZTLLZLZ?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [raspberry-pi, radxa, pine64, rock-pi, odroid, orange-pi, single-board-computer, nvme]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Single-board_computer
faq:
  - q: "Can I run GopherTrunk on a Khadas VIM4 or Edge2?"
    a: "Yes. GopherTrunk is pure Go and cross-compiles to ARM64, so it runs on any Linux SBC with a USB port — no vendor toolchain. Both the Amlogic A311D2 VIM4 and the RK3588S Edge2 are 64-bit boards with plenty of headroom for a multi-channel GopherTrunk node; add an M.2 SSD for continuous IQ recording."
  - q: "Khadas VIM4 or Edge2 for GopherTrunk?"
    a: "The Edge2's Rockchip RK3588S is the faster, more DSP-capable SoC and the better pick for a busy multi-SDR node; the VIM4 (Amlogic A311D2) is compact and capable for a couple of channels. Either works — pick on price, form factor, and whether you want the extra RK3588S cores."
  - q: "Why choose Khadas over a Raspberry Pi?"
    a: "For build quality, a compact form factor, an M.2 slot, and the OOWOW cloud OS installer that flashes images without a card reader. The trade-off is price and ecosystem: Khadas costs more than a Pi and its community and documentation are smaller. For a simple, cheap, well-documented node a Raspberry Pi is still the default."
  - q: "Does a Khadas board help with encrypted channels?"
    a: "No. No SBC changes the encryption wall — GopherTrunk cannot decode AES-protected traffic on any host. Faster hardware only lets you decode and store more unencrypted channels."
---

**Khadas** makes compact, well-built [single-board computers](/reference/single-board-computer/)
in two lines — the **VIM** series (Amlogic SoCs) and the **Edge** series (Rockchip) — that
serve as capable ARM64 GopherTrunk hosts, at a premium over a
[Raspberry Pi](/reference/raspberry-pi/) but with a distinctive OS-flashing workflow and a
tidy, dense board layout.[^wiki]

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Compact, premium ARM64 hosts.** Khadas VIM4 (Amlogic A311D2) and Edge2 (Rockchip
[RK3588S](/reference/rock-pi/)) are 64-bit boards that run GopherTrunk as a pure-Go ARM64
binary, with an **M.2 slot** for fast [NVMe](/reference/nvme/) recording and the **OOWOW**
cloud installer that flashes an OS with no card reader. The **Edge2's RK3588S** is the faster
pick for a busy multi-SDR node. They cost **more than a [Raspberry Pi](/reference/raspberry-pi/)**
and have a smaller community, so a Pi stays the default for a simple node. No SBC decodes
[AES encryption](/police-scanner-encryption/). See
[best SBC for GopherTrunk](/best-single-board-computer-for-gophertrunk/).
</div>

## Overview

Khadas positions its boards a notch above the budget crowd: solid PCB build, integrated
heatsink options, an M.2 slot on the higher models, and **OOWOW** — a built-in cloud service
that downloads and flashes an OS image directly to the board, so you don't need a separate SD
card reader and a PC imaging tool. Two lines matter for a GopherTrunk host:

- **VIM series** — Amlogic-based. The **VIM4** uses the octa-core Amlogic A311D2 (4× Cortex-A73
  + 4× Cortex-A53) with 8 GB RAM, Wi-Fi 6, and an [M.2](/reference/nvme/) slot.
- **Edge series** — Rockchip-based. The **Edge2** uses the RK3588S (the same-family SoC as the
  [Radxa Rock 5](/reference/radxa/) line), the more DSP-capable choice for many channels at once.

Boards run Linux or Android and keep a [GPIO](/reference/gpio/) header. As with most Pi
alternatives, software polish and community depth trail the
[Raspberry Pi](/reference/raspberry-pi/), and the price is higher — you pay for build quality and
the compact form factor.

## Running GopherTrunk on a Khadas board

- **Architecture** — the A311D2 and RK3588S are both ARM64 (aarch64), so the static
  `linux/arm64` GopherTrunk binary from the [downloads page](/downloads.html) runs directly, no
  vendor toolchain required.
- **CPU** — the VIM4's A311D2 comfortably handles a couple of channels; the Edge2's RK3588S has
  meaningfully more real-time DSP headroom for a wideband, multi-SDR pool.
- **RAM** — 8 GB on both flagship configs, ample for recording buffers, the web console, and
  following several systems concurrently.
- **USB** — USB 3.0 ports carry SDR dongles with bandwidth for wideband [Airspy](/reference/airspy/)
  capture; run several from a powered hub so the board isn't back-powering them (see
  [running several dongles](/multi-dongle-sdr-setup/)).
- **Storage** — the [M.2](/reference/nvme/) slot takes an SSD for continuous IQ recording or a
  growing call database, far better than a wear-prone microSD card; eMMC and microSD are also
  available for boot.
- **Power / thermals** — the RK3588S runs hot under sustained decode load; Khadas sells fitted
  heatsinks and fans, which a 24/7 node needs.
- **OS / networking** — flash via OOWOW or write your own image (Ubuntu/Debian, Armbian);
  gigabit Ethernet plus Wi-Fi let a headless node serve the
  [web console](/what-do-i-need-for-gophertrunk/) from by the antenna.

**Bottom line:** a Khadas Edge2 or VIM4 is a compact, well-made GopherTrunk host for a
multi-channel node with M.2 recording — a premium alternative when a
[Raspberry Pi](/reference/raspberry-pi/)'s form factor or storage doesn't fit.

## Where to buy

Khadas boards are sold on Amazon and through Khadas directly. For a GopherTrunk node, the
**Edge2** (RK3588S) is the stronger performer for many channels; the **VIM4** is the compact,
capable pick for a couple of systems. If you want the cheapest, best-documented option instead,
a [Raspberry Pi](/reference/raspberry-pi/) remains the default — compare all the boards in
[best single-board computer for GopherTrunk](/best-single-board-computer-for-gophertrunk/), or
look at the [Radxa](/reference/radxa/) Rock 5 for native PCIe/NVMe.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B09ZTLLZLZ?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra cost
to you. It never changes what we recommend.*

## Sources

[^wiki]: [Single-board computer](https://en.wikipedia.org/wiki/Single-board_computer) — Wikipedia, background on ARM single-board computers including Khadas VIM and Edge boards.
