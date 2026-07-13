---
slug: balun
title: Balun
entry_type: hardware
category: rf-front-end
description: "A balun is a transformer that connects a balanced line or antenna to an unbalanced one such as coax, blocking common-mode current and optionally matching impedance."
keywords: balun, balanced unbalanced, current balun, voltage balun, choke balun, 1:1 balun, 4:1 balun, transmission line transformer, common-mode, feedline, dipole feed, coax
aka: [balun, "balanced-to-unbalanced transformer"]
autolink: true
infobox:
  - { label: Type, value: "Balanced-to-unbalanced transformer" }
  - { label: Common ratios, value: "1:1, 4:1, 9:1" }
  - { label: Key spec, value: "Common-mode rejection, power, bandwidth" }
  - { label: TX, value: "Yes (power-rated types)" }
  - { label: Typical price, value: "$8–$120" }
see_also: [feedpoint-impedance, dipole-antenna, unun, coaxial-cable, ferrite-choke]
cite_urls:
  - https://en.wikipedia.org/wiki/Balun
  - https://en.wikipedia.org/wiki/Transmission_line
---

A **balun** (from **bal**anced–**un**balanced) is a component that joins a
*balanced* line or antenna — one where two conductors carry equal and opposite
currents — to an *unbalanced* line such as coax, where signal rides on a center
conductor against a grounded shield.[^wiki] Connecting the two directly lets current
flow on the outside of the coax shield, unbalancing the antenna and turning the
feedline into an unwanted radiator; a balun forces balance and can, with a turns
ratio, also transform [impedance](/reference/impedance/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A balun connects unbalanced coax on the left to a balanced two-conductor antenna feed on the right, blocking common-mode current on the shield." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="70" x2="120" y2="70" stroke="currentColor" stroke-width="3"/>
  <line x1="30" y1="70" x2="120" y2="70" stroke="currentColor" stroke-opacity="0.35" stroke-width="8"/>
  <text x="75" y="55" text-anchor="middle" font-size="9" fill="currentColor">coax (unbalanced)</text>
  <rect x="120" y="45" width="70" height="50" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.8"/>
  <text x="155" y="74" text-anchor="middle" font-size="10" fill="currentColor">balun</text>
  <line x1="190" y1="58" x2="300" y2="35" stroke="currentColor" stroke-width="1.8"/>
  <line x1="190" y1="82" x2="300" y2="105" stroke="currentColor" stroke-width="1.8"/>
  <circle cx="305" cy="34" r="4" fill="currentColor"/>
  <circle cx="305" cy="106" r="4" fill="currentColor"/>
  <text x="360" y="72" text-anchor="middle" font-size="9" fill="currentColor">balanced feed</text>
</svg>
<figcaption>A balun mates unbalanced coax to a balanced antenna feed, keeping current off the coax shield.</figcaption>
</figure>

## Overview

Balance matters because a balanced antenna, like a center-fed
[dipole](/reference/dipole-antenna/), assumes its two halves carry symmetric
currents. Feed it with coax directly and the shield offers a third path: common-mode
current flows on the outside of the braid, so the feedline both radiates (distorting
the pattern) and receives (bringing household noise to the receiver). A **current
balun** presents a high impedance to that common-mode current while passing the
differential signal, restoring symmetry. A **voltage balun** instead forces equal
and opposite voltages at its output; current baluns are generally preferred for
pattern cleanliness.

## Variants

- **1:1 current (choke) balun** — many turns of coax on a
  [ferrite](/reference/ferrite-choke/) core, or a coiled-coax "ugly balun." No
  impedance change; its job is purely common-mode suppression at the
  [feedpoint](/reference/feedpoint-impedance/).
- **4:1 balun** — transforms 200 Ω down to 50 Ω (or 300 Ω to 75 Ω), matching
  folded dipoles and off-center-fed antennas. Built as a transmission-line
  (Guanella/Ruthroff) transformer or a wound autotransformer.
- **9:1 and other ratios** — used for high-impedance wire antennas such as random-
  wire and end-fed designs.
- **Half-wave coax (bazooka) and sleeve baluns** — narrowband baluns made from a
  measured length of line rather than a ferrite core.

A closely related device, the [unun](/reference/unun/), transforms impedance
between two *unbalanced* lines; a balun differs in that at least one side is
balanced.

## Relevance to SDR

Any listener feeding a balanced antenna — a dipole, folded dipole, loop, or T2FD —
with coax should use a balun at the feedpoint. The receive-side payoff mirrors the
transmit-side one: a current balun keeps common-mode noise off the coax shield,
which for weak-signal SDR work often lowers the [noise floor](/reference/noise-floor/)
by several dB and cleans up the [radiation pattern](/reference/radiation-pattern/)
so nulls and gain behave as designed. Ratio baluns additionally correct the
[impedance](/reference/impedance/) mismatch that would otherwise raise
[SWR](/reference/standing-wave-ratio/) and cost signal.

GopherTrunk is software and includes no analog hardware, so a balun is entirely part
of the antenna install ahead of the SDR. Its benefit to GopherTrunk is the same as
any front-end improvement: a properly balanced, low-noise feed delivers a cleaner
signal to the demodulator, improving the signal-to-noise ratio the trunking decoder
relies on to hold a control-channel lock. Baluns are common as small in-line
modules or as the potted transformer inside a receive antenna's feedpoint box.

## Sources

[^wiki]: [Balun](https://en.wikipedia.org/wiki/Balun) — Wikipedia, on balanced-to-unbalanced transformers, current versus voltage types, and their impedance ratios.
