---
slug: hardware-spectrum
title: The hardware spectrum
entry_type: concept
category: hw-foundations
description: The hardware spectrum is the full range of computing hardware ordered by power, cost, and control, from cloud servers down to tiny microcontrollers.
keywords: hardware spectrum, computing tiers, server to microcontroller, general-purpose vs embedded, power cost control
aka: [hardware spectrum]
infobox:
  - { label: Type, value: Conceptual range }
  - { label: Ordered by, value: Power, cost, control }
  - { label: Top end, value: Cloud and servers }
  - { label: Bottom end, value: Microcontrollers }
see_also: [server, personal-computer, single-board-computer, microcontroller, cloud-computing]
related_lessons:
  - { title: "The hardware spectrum", url: /learn/intro-hardware/hardware-spectrum/ }
  - { title: "What is hardware?", url: /learn/intro-hardware/what-is-hardware/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Classes_of_computers
---

**The hardware spectrum** is the full range of computing hardware, ordered by how much power, cost, and direct control each tier gives you.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 90" role="img" aria-label="A bar showing the hardware spectrum from cloud and servers at the high-power end to microcontrollers at the low-power end." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="60" x2="430" y2="60" stroke="currentColor" stroke-opacity="0.4"/>
  <rect x="32" y="36" width="396" height="18" rx="3" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/>
  <g font-size="8" fill="currentColor"><text x="32" y="74">cloud / server</text><text x="430" y="74" text-anchor="end">microcontroller</text></g>
  <text x="230" y="24" text-anchor="middle" font-size="10" fill="currentColor">more power, cost, scale ←→ smaller, cheaper, closer to the world</text>
</svg>
<figcaption>From a rented slice of a data center down to a chip running one job.</figcaption>
</figure>

## Overview
Running from the high-power end down, the tiers are roughly: [cloud computing](/reference/cloud-computing/) and [web hosting](/reference/web-hosting/) → [VPS](/reference/virtual-private-server/) → [dedicated server](/reference/dedicated-server/) → [home server](/reference/home-server/) → [desktop](/reference/desktop-computer/) → [laptop](/reference/laptop/) → [tablet](/reference/tablet/) and [smartphone](/reference/smartphone/) → [single-board computer](/reference/single-board-computer/) → [microcontroller](/reference/microcontroller/).

## Why it matters
A useful split cuts across the spectrum: **general-purpose** computers run many different programs and usually carry an [operating system](/reference/operating-system/), while **embedded** computers do one fixed job, often with no OS at all. Choosing a tier is a trade between power and cost on one side and size, efficiency, and physical closeness to the world on the other. GopherTrunk runs across most of it — from a full [server](/reference/server/) down to a [single-board computer](/reference/single-board-computer/) sitting by the antenna.

## Sources
[^wiki]: [Classes of computers](https://en.wikipedia.org/wiki/Classes_of_computers) — Wikipedia, on the range of computer sizes and roles.
