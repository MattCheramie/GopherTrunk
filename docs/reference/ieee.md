---
slug: ieee
title: IEEE
entry_type: organization
category: hw-organizations
description: The IEEE is a global professional association for electrical and electronics engineering that publishes widely used technical standards, including those behind Ethernet and Wi-Fi.
keywords: IEEE, Institute of Electrical and Electronics Engineers, standards, 802.11, 802.3, 754, engineering
aka: [Institute of Electrical and Electronics Engineers]
autolink: true
infobox:
  - { label: Type, value: Professional association and standards body }
  - { label: Founded, value: "1963" }
  - { label: HQ, value: New York / Piscataway, New Jersey, USA }
  - { label: Makes, value: Technical standards, publications }
see_also: [ethernet, wi-fi, jedec, itu, wifi-80211]
cite_urls:
  - https://www.ieee.org/
  - https://en.wikipedia.org/wiki/IEEE
---

**The IEEE** (Institute of Electrical and Electronics Engineers) is a global professional
association for electrical, electronics, and computing engineering that publishes many of
the technical standards the industry relies on.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A map of well-known IEEE standards. A central IEEE node branches to the 802 networking family, which splits into 802.3 Ethernet and 802.11 Wi-Fi, and separately to IEEE 754, the floating-point arithmetic standard that governs how computers represent decimal numbers." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="86" y1="70" x2="170" y2="45"/>
    <line x1="86" y1="70" x2="170" y2="105"/>
    <line x1="252" y1="45" x2="330" y2="28"/>
    <line x1="252" y1="45" x2="330" y2="62"/>
  </g>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.12" stroke-width="1.2">
    <rect x="20" y="55" width="66" height="30" rx="4"/>
    <rect x="170" y="30" width="82" height="30" rx="4"/>
    <rect x="170" y="90" width="82" height="30" rx="4"/>
    <rect x="330" y="14" width="112" height="28" rx="4"/>
    <rect x="330" y="48" width="112" height="28" rx="4"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="9">
    <text x="53" y="74" text-anchor="middle" font-weight="600">IEEE</text>
    <text x="211" y="49" text-anchor="middle" font-weight="600">802</text>
    <text x="211" y="109" text-anchor="middle" font-weight="600">754</text>
    <text x="340" y="32" font-size="8.5">802.3 &#8594; Ethernet</text>
    <text x="340" y="66" font-size="8.5">802.11 &#8594; Wi-Fi</text>
    <text x="211" y="121" text-anchor="middle" font-size="7.5">floating point</text>
  </g>
</svg>
<figcaption>A few IEEE standards touch almost every networked computer: the 802 family gives Ethernet (802.3) and Wi-Fi (802.11), while IEEE 754 defines how machines store floating-point numbers.</figcaption>
</figure>

## Overview

The IEEE was formed in 1963 by the merger of two older engineering societies. It is the
world's largest technical professional organization, with members across academia and
industry, and it runs conferences, journals, and education programs in addition to standards
work.[^home]

Its standards arm, the IEEE Standards Association, produces specifications used everywhere.
The IEEE 802 family is the best known: 802.3 defines [Ethernet](/reference/ethernet/) and
802.11 defines [Wi-Fi](/reference/wi-fi/). The IEEE 754 standard for floating-point
arithmetic governs how computers represent decimal numbers.

## Standards that touch SDR

A handful of IEEE numbers show up directly in a capture-and-decode setup:

| Standard | What it defines |
|----------|-----------------|
| 802.3 | Ethernet, the wired link between nodes and server |
| 802.11 | Wi-Fi, the wireless link for remote nodes |
| 754 | Floating-point numbers used throughout DSP math |

## Where it fits

IEEE standards let equipment from different vendors interoperate — the reason an Ethernet
cable or Wi-Fi connection works between any two compliant devices. The networking that ties
a fleet of GopherTrunk capture nodes back to a server runs on IEEE-standardized links, and
IEEE 754 arithmetic underlies the DSP math itself, so the floating-point samples flowing
through the decoder obey a rule the IEEE wrote.

## Sources

[^home]: [IEEE](https://www.ieee.org/) — the association's official site.
[^wiki]: [IEEE](https://en.wikipedia.org/wiki/IEEE) — Wikipedia, for the organization's history and standards work.
