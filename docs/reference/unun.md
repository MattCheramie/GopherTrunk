---
slug: unun
title: Unun
entry_type: hardware
category: rf-front-end
description: "An unun is a transformer that matches impedance between two unbalanced lines, such as a 9:1 unun feeding a high-impedance end-fed wire antenna from 50-ohm coax."
keywords: unun, unbalanced unbalanced transformer, impedance transformer, 9:1 unun, 49:1 unun, 4:1 unun, end-fed antenna, random wire, transmission line transformer, autotransformer, matching
aka: [unun, "unbalanced-to-unbalanced transformer"]
autolink: true
infobox:
  - { label: Type, value: "Unbalanced-to-unbalanced impedance transformer" }
  - { label: Common ratios, value: "4:1, 9:1, 49:1" }
  - { label: Key spec, value: "Impedance ratio, bandwidth, power" }
  - { label: TX, value: "Yes (power-rated types)" }
  - { label: Typical price, value: "$10–$90" }
see_also: [balun, impedance, feedpoint-impedance, antenna-tuner]
cite_urls:
  - https://en.wikipedia.org/wiki/Balun
  - https://en.wikipedia.org/wiki/Impedance_matching
---

An **unun** (from **un**balanced–**un**balanced) is a transformer that changes
[impedance](/reference/impedance/) between two *unbalanced* lines, keeping both the
input and output referenced to ground.[^wiki] Where a [balun](/reference/balun/)
converts between a balanced and an unbalanced line, an unun stays unbalanced on both
sides and does one job: match a source impedance (typically 50 Ω coax) to a very
different load impedance, most famously the high feedpoint impedance of an end-fed
wire antenna.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="An unun transforms 50-ohm coax on the left up to a high impedance on the right that feeds an end-fed wire antenna, with both sides ground-referenced." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="120" y2="70" stroke="currentColor" stroke-width="3"/>
  <text x="75" y="55" text-anchor="middle" font-size="9" fill="currentColor">50 Ω coax</text>
  <rect x="120" y="45" width="80" height="50" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.8"/>
  <text x="160" y="68" text-anchor="middle" font-size="10" fill="currentColor">unun</text>
  <text x="160" y="82" text-anchor="middle" font-size="8" fill="currentColor">9:1</text>
  <line x1="200" y1="60" x2="420" y2="45" stroke="currentColor" stroke-width="1.8"/>
  <text x="330" y="35" text-anchor="middle" font-size="9" fill="currentColor">end-fed wire (~450 Ω)</text>
  <line x1="150" y1="95" x2="150" y2="115" stroke="currentColor" stroke-width="1.4"/>
  <line x1="142" y1="115" x2="158" y2="115" stroke="currentColor" stroke-width="1.4"/>
  <line x1="145" y1="120" x2="155" y2="120" stroke="currentColor" stroke-width="1.2"/>
  <text x="150" y="112" text-anchor="start" font-size="7" fill="currentColor"> gnd</text>
</svg>
<figcaption>A 9:1 unun steps 50-ohm coax up to the high impedance of an end-fed wire, both sides ground-referenced.</figcaption>
</figure>

## Overview

Like its balun cousin, an unun is usually built as a transmission-line transformer
or an autotransformer wound on a ferrite toroid. The turns ratio sets the impedance
ratio, which goes as the square of the turns: a 3:1 turns ratio gives a 9:1
impedance transformation. Because both ports share a common ground, an unun does not
provide the common-mode suppression a current balun does — so an unun is often
paired with a separate [ferrite choke](/reference/ferrite-choke/) or a short
counterpoise to keep RF off the coax shield. Key specs are the transformation ratio,
the usable bandwidth (broadband transmission-line designs span many octaves), and
the power rating set by the core.

## Variants

- **9:1 unun** — the classic match for random-wire and end-fed antennas whose
  [feedpoint impedance](/reference/feedpoint-impedance/) hovers around a few hundred
  ohms; brings ~450 Ω toward 50 Ω so an [antenna tuner](/reference/antenna-tuner/)
  can finish the job.
- **49:1 unun** — used with resonant end-fed half-wave (EFHW) antennas, whose
  feedpoint impedance is very high (~2000–3000 Ω).
- **4:1 unun** — a moderate step for antennas or networks near 200 Ω, and for
  interfacing some verticals and matching sections.
- **1:1 isolation unun** — no impedance change, used mainly to place a defined
  reference between sections.

## Relevance to SDR

Unnuns matter to SDR listeners who use long-wire or end-fed receive antennas, a
popular choice for wideband HF and general coverage because a single wire covers
enormous bandwidth. A 9:1 unun brings such a wire's erratic, high impedance closer
to the 50 Ω the receiver expects, raising the [return loss](/reference/return-loss/)
and delivering more of the captured signal into the coax instead of reflecting it.
The improvement is broadband, which suits the "listen everywhere" use case of an SDR
better than a narrowband resonant match.

Because an end-fed wire relies on the feedline and ground as its counterpoise, an
unun installation almost always benefits from a common-mode choke to stop the coax
from becoming a noise-collecting radiator — the same [noise-floor](/reference/noise-floor/)
concern that drives balun and ferrite-choke use.

GopherTrunk is a receive-only software decoder with no analog hardware, so it never
contains an unun; the device lives in the antenna system feeding the SDR. Its value
to a GopherTrunk user is indirect but real: on the HF and low-VHF wire antennas
sometimes used for [POCSAG](/reference/pocsag/), utility, and other monitoring, a
proper impedance match delivers a stronger, cleaner signal to the demodulator,
improving the signal-to-noise ratio the decoder depends on.

## Sources

[^wiki]: [Balun](https://en.wikipedia.org/wiki/Balun) — Wikipedia, which covers ununs as unbalanced-to-unbalanced impedance transformers alongside baluns.
