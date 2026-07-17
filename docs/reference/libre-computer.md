---
slug: libre-computer
title: Libre Computer
entry_type: hardware
category: hw-sbc
description: Libre Computer is a maker of single-board computers that emphasise open, mainline software support and Raspberry Pi compatibility, with boards such as Le Potato and Renegade built on Amlogic and Rockchip chips.
keywords: Libre Computer, Le Potato, AML-S905X-CC, Renegade, open source SBC, mainline Linux, Raspberry Pi compatible, ARM single-board computer, upstream drivers
infobox:
  - { label: Type, value: Single-board computer }
  - { label: Emphasis, value: Open, mainline software }
  - { label: CPU, value: ARM (Amlogic / Rockchip) }
  - { label: Runs, value: Mainline Linux, Android }
  - { label: Boards, value: Le Potato, Renegade, others }
see_also: [raspberry-pi, single-board-computer, orange-pi, odroid, rock-pi, gpio]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Libre_Computer_Project
  - https://libre.computer/
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

## Overview

Boards such as Le Potato (AML-S905X-CC) and the Renegade use Amlogic and Rockchip ARM chips and copy the Pi's footprint and 40-pin [GPIO](/reference/gpio/) header, so existing cases and add-ons often fit. On paper they look like any other Pi alternative — the difference is in how the software is delivered.

The project's distinguishing goal is upstream support: getting drivers into the mainline Linux kernel and standard bootloaders so the hardware keeps working with ordinary, current software rather than a vendor's frozen image.[^libre] Many low-cost boards ship a one-off kernel fork that works the day you buy it but never gets updated, so a few years on the board is stuck on an old, insecure OS. Mainlined support avoids that trap because the board rides along with every new kernel release.

## Where it fits

Among Pi alternatives like [Orange Pi](/reference/orange-pi/), [ODROID](/reference/odroid/), and [Rock Pi](/reference/rock-pi/), Libre Computer's pitch is longevity and trust in the software stack rather than raw specs — you may give up a little performance per dollar in exchange for a board that stays current. For an always-on GopherTrunk capture node you intend to leave in place for years, that trade favours Libre: mainline support means OS and security updates are far less likely to strand the board, so a decode node bolted up near an antenna keeps receiving patches without a risky vendor-image migration.

## Sources

[^wiki]: [Libre Computer Project](https://en.wikipedia.org/wiki/Libre_Computer_Project) — Wikipedia, on the project and its boards.
[^libre]: [Libre Computer](https://libre.computer/) — vendor site, on the boards and their open-software focus.
