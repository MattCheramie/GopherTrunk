---
slug: n-type-connector
title: N-type connector
entry_type: hardware
category: rf-front-end
description: "The N-type is a rugged, weatherproof threaded 50Ω coaxial connector with low loss to ~11 GHz, standard for outdoor antenna feedlines and base stations."
keywords: N-type connector, N connector, Type N, 50 ohm, 75 ohm, weatherproof coaxial connector, low loss, base station, LMR-400, outdoor antenna
aka: [N connector, "Type N", "N-type"]
autolink: true
affiliate: true
product:
  name: "SMA to N-type adapter kit (4 gender combinations)"
  brand: onelinkmore
  category: N-to-SMA coaxial adapter kit
  lowPrice: "9"
  highPrice: "14"
  url: https://www.amazon.com/dp/B06XPDWBPR?tag=gophertrunk-20
infobox:
  - { label: Type, value: "Threaded coaxial connector" }
  - { label: Impedance, value: "50 Ω (75 Ω variant)" }
  - { label: Range, value: "DC to ~11 GHz" }
  - { label: Coupling, value: "5/8-24 threaded, gasket-sealed" }
  - { label: TX, value: "Yes (medium/high power)" }
  - { label: Buy, value: "<a class=\"btn btn--buy\" href=\"https://www.amazon.com/dp/B06XPDWBPR?tag=gophertrunk-20\" rel=\"nofollow sponsored noopener\">View on Amazon &rarr;</a>" }
see_also: [coaxial-cable, sma-connector, bnc-connector, uhf-connector-pl259, coax-feedline, low-noise-amplifier]
cite_urls:
  - https://en.wikipedia.org/wiki/N_connector
faq:
  - q: "Which N-to-SMA adapter should I buy for an SDR?"
    a: "An SMA-to-N adapter kit that covers all four gender combinations (around $12) is the pick, because an outdoor antenna feedline almost always ends in N while the dongle's port is SMA. Buying the small kit means you have the right gender on hand whether the feedline presents an N plug or an N jack, rather than discovering the mismatch on the roof."
  - q: "Why do outdoor antennas use N instead of SMA?"
    a: "N is weatherproof and low-loss. Its gasketed, threaded shell keeps water out of the joint for years on a mast, its part-air dielectric adds only a fraction of a dB across VHF/UHF, and it handles far more power than the miniature SMA. SMA is chosen for size on a crowded dongle edge, not for a rooftop — so a good install runs N up the mast and adapts down to SMA only in the last few centimetres."
  - q: "Where should the N-to-SMA transition go in the chain?"
    a: "As close to the receiver as practical. Keep the low-loss N-terminated coax running all the way from the antenna, then step down to the dongle's SMA (or to a mast-mounted LNA) with a single adapter or short pigtail. Adapting to a lossy miniature connector at the top of the mast, before the long cable, throws away signal you cannot get back."
  - q: "Is 50 Ω or 75 Ω N right for scanning?"
    a: "Use 50 Ω N — it matches your SDR, antennas, and coax. A 75 Ω N (from broadcast/cable plant) has a different centre-pin diameter, and forcing a 75 Ω plug into a 50 Ω jack can spread and damage the contact, so the two are genuinely not interchangeable."
---

**N-type** (Type N) is a medium-size, threaded [coaxial](/reference/coaxial-cable/)
connector built for **weatherproof**, **low-loss** outdoor use, usable to roughly
**11 GHz** in its standard form.[^wiki] Its gasketed, threaded shell and air-spaced
interface make it the default choice for base-station feedlines, roof-mounted
[antennas](/reference/antenna/), and the thick low-loss coax (LMR-400 and similar) that runs
between them. The connector was designed by Paul Neill at Bell Labs in the 1940s — the "N"
is for Neill — and predates the smaller BNC and SMA that borrowed its ideas.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An N-type plug with a threaded coupling nut and a rubber gasket sealing against a jack, feeding thick low-loss coax on each side." xmlns="http://www.w3.org/2000/svg">
  <rect x="25" y="58" width="70" height="34" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.4"/>
  <line x1="95" y1="75" x2="140" y2="75" stroke="currentColor" stroke-width="3"/>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <rect x="140" y="48" width="70" height="54"/>
    <line x1="140" y1="55" x2="210" y2="55"/><line x1="140" y1="95" x2="210" y2="95"/>
    <rect x="205" y="52" width="8" height="46" fill="currentColor" fill-opacity="0.15"/>
  </g>
  <circle cx="216" cy="75" r="4" fill="currentColor"/>
  <line x1="220" y1="75" x2="246" y2="75" stroke="currentColor" stroke-width="2"/>
  <g stroke="currentColor" stroke-width="1.4" fill="none"><rect x="246" y="48" width="60" height="54"/></g>
  <line x1="306" y1="75" x2="365" y2="75" stroke="currentColor" stroke-width="3"/>
  <rect x="365" y="58" width="70" height="34" fill="currentColor" fill-opacity="0.08" stroke="currentColor" stroke-width="1.4"/>
  <g font-size="9" fill="currentColor"><text x="60" y="112" text-anchor="middle">thick coax</text><text x="175" y="128" text-anchor="middle">plug + gasket</text><text x="276" y="128" text-anchor="middle">jack</text><text x="400" y="112" text-anchor="middle">thick coax</text></g>
</svg>
<figcaption>An N-type plug threads onto a jack and its gasket seals the joint against weather; it carries higher power and lower loss than BNC or SMA.</figcaption>
</figure>

## Overview

Where [SMA](/reference/sma-connector/) is chosen for size and [BNC](/reference/bnc-connector/)
for quick handling, N-type is chosen for **ruggedness and sealing**. The **5/8-24** threaded
coupling clamps a captive rubber gasket that keeps water out of the joint — essential for a
connector that lives on a mast for years. The internal geometry is partly air-dielectric,
giving low insertion loss and letting the connector handle far more transmit power than the
miniature types.

## What it is

An N joint on good coax adds only a fraction of a dB of loss across the VHF/UHF land-mobile
range, which is why serious antenna installs terminate in N rather than adapting down to a
lossy miniature connector at the top of the mast. Its size is the price: N is too big for a
crowded SDR board edge, so the last few centimetres before the receiver usually transition
to SMA or BNC. Standard N is rated to about 11 GHz; precision variants push higher.

## Variants

- **50 Ω N** is the RF and land-mobile standard.
- **75 Ω N** is used in some broadcast and cable-TV plant; the centre-pin diameter differs,
  and forcing a 75 Ω plug into a 50 Ω jack can spread and damage the contact, so the two are
  genuinely not interchangeable.
- **Reverse-polarity N (RP-N)** appears on some Wi-Fi and licence-exempt gear to discourage
  antenna swaps, mirroring the RP-SMA idea.
- **Weatherproofing** is still finished in the field with self-amalgamating tape over the
  gasketed shell for a fully sealed outdoor joint.

## Relevance to SDR

For a fixed listening post, the antenna and its feedline almost always use N-type at the
mast and the radio end, because minimising feedline loss and keeping water out of the joint
directly protect the weak signals a scanner cares about. A short adapter or pigtail then
steps N down to the [SMA](/reference/sma-connector/) port on the dongle or the
[low-noise amplifier](/reference/low-noise-amplifier/) mounted near the antenna. GopherTrunk
is decode software and never touches hardware, but its results ride on that feedline: a
well-made, sealed N joint on low-loss [coax](/reference/coaxial-cable/) preserves the
signal-to-noise ratio that the decoder ultimately depends on, while a corroded or
water-ingressed N connector degrades every capture no matter how the DSP is tuned.

## Where to buy

For a fixed listening post, run **N-terminated** low-loss
[coax](/reference/coax-feedline/) all the way from the rooftop antenna and step down to
the dongle's **[SMA](/reference/sma-connector/)** port only at the very end. The cheap
enabler is an **SMA-to-N adapter kit** covering all four gender combinations (around $12),
so you have the right part whether the feedline ends in an N plug or an N jack. If you also
juggle BNC or UHF gear, the all-in-one **[SMA adapter kit](/reference/sma-adapter-kit/)**
bundles those too.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B06XPDWBPR?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For the outdoor feedline itself and a mast-mounted [LNA](/best-sdr-lna/), see the
[SDR cables and connectors guide](/sdr-cables-and-connectors/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [N connector](https://en.wikipedia.org/wiki/N_connector) — Wikipedia, on the weatherproof gasketed design, 5/8-24 thread, air dielectric, power handling, and ~11 GHz range.
