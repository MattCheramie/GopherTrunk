---
slug: coax-pigtail
title: Coax pigtail
entry_type: hardware
category: rf-front-end
description: "A coax pigtail is a short, flexible RG316 jumper — SMA to BNC, UHF, N, or SMA — that connects an SDR dongle to a fixed antenna or feedline without stressing the jack."
keywords: coax pigtail, RG316 pigtail, SMA pigtail, SMA jumper, flexible SDR cable, SMA to BNC pigtail, dongle to antenna cable, short coax jumper
aka: [pigtail, RG316 pigtail, SMA jumper, patch lead]
autolink: true
affiliate: true
product:
  name: "RTL-SDR Blog RG316 SMA pigtail kit"
  brand: RTL-SDR Blog
  category: RG316 coaxial pigtail kit
  lowPrice: "13"
  highPrice: "18"
  url: https://www.amazon.com/dp/B0132N1DM0?tag=gophertrunk-20
infobox:
  - { label: Type, value: "Flexible RF jumper" }
  - { label: Cable, value: "RG316 (thin, flexible)" }
  - { label: Ends, value: "SMA + BNC/UHF/N as needed" }
  - { label: Length, value: "Short (inches to a few feet)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B0132N1DM0?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [coaxial-cable, sma-connector, sma-adapter-kit, bnc-connector, coax-feedline, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Coaxial_cable
faq:
  - q: "Which coax pigtail should I buy for an SDR?"
    a: "An RG316 SMA pigtail kit (around $15) is the flexible-lead answer: short jumpers terminated in SMA plus the common far-end connectors, so you can connect a dongle to a fixed antenna or feedline without hanging the SDR's weight on a rigid barrel adapter. RG316 is thin and very flexible, ideal for tight bends and bench use over the foot or two a pigtail spans."
  - q: "What is the difference between a pigtail and an adapter?"
    a: "An adapter is a rigid brass barrel that changes connector type in place; a pigtail is a short flexible cable that changes type over a few inches. Use an adapter when the mating parts can sit directly together; use a pigtail when you need to route around a corner, relieve strain on the dongle's SMA jack, or reach a panel-mounted antenna without the whole radio dangling from the connector."
  - q: "Is RG316 lossy?"
    a: "RG316 has relatively high loss per foot, but a pigtail is only inches to a couple of feet long, so the total loss is negligible even at UHF. Its whole point is flexibility over a short span. For any real distance — across a room or up a mast — switch to lower-loss RG58, RG8X, or LMR-240 feedline rather than extending a thin pigtail."
  - q: "Can I use a pigtail to move my antenna to the window?"
    a: "For a short reach, yes — a two-to-three-foot pigtail gets a dongle off a cramped USB port and onto a window-mounted antenna. For a longer reach, do not chain pigtails; either run a proper low-loss feedline or, better, move the whole SDR to the window on a USB extension and keep the RF run short."
---

A **coax pigtail** is a short, very flexible [coaxial](/reference/coaxial-cable/) jumper —
typically thin **RG316** — with an [SMA](/reference/sma-connector/) connector on one end and
whatever your antenna or feedline uses on the other. Where a rigid barrel
[adapter](/reference/sma-adapter-kit/) changes connector type *in place*, a pigtail changes
it over a few inches of bendable cable, so it **connects an SDR to a fixed antenna or
feedline without hanging the radio's weight on the jack** or forcing an awkward straight-line
mate.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A short flexible coax pigtail curving from an SDR dongle's SMA jack to a panel-mounted antenna connector, relieving strain on the dongle." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="52" width="70" height="34" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.4"/>
  <text x="65" y="73" font-size="10" fill="currentColor" text-anchor="middle">SDR</text>
  <circle cx="104" cy="69" r="3" fill="currentColor"/>
  <path d="M104 69 C 160 40, 230 100, 300 69" fill="none" stroke="currentColor" stroke-width="2.4"/>
  <text x="205" y="52" font-size="9" fill="currentColor" text-anchor="middle">flexible RG316</text>
  <circle cx="300" cy="69" r="3" fill="currentColor"/>
  <line x1="360" y1="30" x2="360" y2="108" stroke="currentColor" stroke-width="2"/>
  <rect x="303" y="60" width="57" height="18" fill="none" stroke="currentColor" stroke-width="1.2"/>
  <text x="405" y="73" font-size="9" fill="currentColor" text-anchor="middle">antenna</text>
</svg>
<figcaption>A pigtail's flexibility routes around corners and relieves strain the way a rigid adapter cannot.</figcaption>
</figure>

## What it is

RG316 is a thin PTFE-insulated coax prized for flexibility and temperature tolerance rather
than low loss. Over the short span a pigtail spans — inches to a couple of feet — its
higher loss per foot is negligible, even at the UHF frequencies trunked systems use, so you
get the bend radius and strain relief without paying meaningfully in signal. A pigtail kit
usually bundles several jumpers terminated in SMA plus common far-end connectors
([BNC](/reference/bnc-connector/), [UHF](/reference/uhf-connector-pl259/),
[N](/reference/n-type-connector/), SMA-to-SMA), covering the same mismatches an
[adapter kit](/reference/sma-adapter-kit/) does but as flexible leads.

The practical reason pigtails exist is mechanical. An SDR's SMA jack is a small solder joint
on a circuit board; hanging a heavy antenna, a stack of adapters, or a stiff cable directly
off it stresses that joint and can crack it. A short flexible pigtail takes the load and the
motion instead, and lets you route the connection around a corner or down to a panel-mounted
socket that a straight barrel could never reach.

## Relevance to SDR

On the bench, a pigtail is how you get a dongle's port comfortably away from a crowded USB
hub and onto a [window-mounted antenna](/best-sdr-antenna/) or a length of feedline. It is
the flexible complement to the rigid adapter kit: the adapter changes the connector, the
pigtail gives you the slack and the strain relief. For an [RTL-SDR](/reference/rtl-sdr/) sat
in a USB port with a fixed antenna a foot away, a single pigtail is often the entire cable
budget.

What a pigtail is *not* is a feedline. RG316's loss climbs steeply with length, so chaining
pigtails to reach across a room throws away signal fast. For any real distance, run proper
low-loss [coax](/reference/coax-feedline/) — or, better, keep the RF run short by moving the
whole SDR to the antenna on a [USB extension](/reference/usb-extension-cable/) and sending
data the long way. GopherTrunk decodes whatever the front end captures and has no view of
the cable; the pigtail's only job is to deliver the antenna's signal to the SDR cleanly and
without mechanical stress.

## Where to buy

An **RG316 SMA pigtail kit** (around $15) is the flexible-lead staple — a set of short
jumpers in SMA plus the common terminations, so a dongle can reach a fixed antenna or
feedline without a rigid barrel taking the strain. It pairs naturally with an
[SMA adapter kit](/reference/sma-adapter-kit/): the adapters handle the odd in-place
transition, the pigtails handle everything that needs to bend.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B0132N1DM0?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For a real distance run, step up to low-loss [feedline](/reference/coax-feedline/); for the
full connector map, see the [SDR cables and connectors guide](/sdr-cables-and-connectors/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Coaxial cable](https://en.wikipedia.org/wiki/Coaxial_cable) — Wikipedia, on coax construction and the frequency-dependent, per-length loss that makes thin RG316 suitable only for short jumpers.
