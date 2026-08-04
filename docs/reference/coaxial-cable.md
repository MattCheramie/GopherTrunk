---
slug: coaxial-cable
title: Coaxial cable
entry_type: hardware
category: rf-front-end
description: "Coaxial cable carries RF from antenna to receiver on a shielded centre conductor; every metre and connector adds loss, more at higher frequencies, so keep runs short."
keywords: coaxial cable, coax, feedline, RG-58, RG-6, LMR-400, cable loss, shield, impedance, velocity factor, characteristic impedance
aka: [coax, "coaxial cable", feedline]
autolink: true
affiliate: true
product:
  name: "NooElec SMA coaxial cable connectivity kit"
  brand: NooElec
  category: SDR coaxial cable kit
  lowPrice: "15"
  highPrice: "21"
  url: https://www.amazon.com/dp/B077H87LTS?tag=gophertrunk-20
see_also: [attenuation, path-loss, antenna, standing-wave-ratio, low-noise-amplifier, sma-connector, n-type-connector]
related_lessons:
  - { title: "Antennas 101", url: /learn/rf-sdr/antennas/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Coaxial_cable
  - https://en.wikipedia.org/wiki/Characteristic_impedance
faq:
  - q: "What coax should I buy for an SDR?"
    a: "For short bench and patch leads with the SMA connectors SDRs use, a NooElec SMA cable connectivity kit (around $18) covers the common lengths and adapters. For any run over a few metres, especially at UHF or 800 MHz, step up to low-loss RG8X or LMR-400 rather than thin RG58, which throws away too much signal over distance."
  - q: "How much does cable loss matter for scanning?"
    a: "A lot on the higher bands. Loss grows with frequency, so a long thin cable that is fine at HF can cost several dB at 1 GHz — and 6 dB of feedline loss throws away three-quarters of the signal before it reaches the SDR. Keep runs short and thick, or mount an LNA at the antenna so its gain is applied before the cable loss."
  - q: "RG58 or RG8X for an SDR feedline?"
    a: "RG58 (or thin RG174) is fine only for short patch leads. For a real feedline run use RG8X for moderate lengths or LMR-400/RG213 for masthead installs — the extra thickness cuts loss sharply at VHF/UHF. RG6 (75 Ω TV cable) is also excellent and cheap for receive-only use if you accept the minor impedance mismatch."
  - q: "Do connectors and adapters add loss too?"
    a: "Yes — each junction adds a small insertion loss and a potential mismatch, so use the fewest transitions you can. Weatherproof any outdoor connectors, since water in coax raises loss dramatically."
---

**Coaxial cable** ("coax") carries RF between the [antenna](/reference/antenna/) and
receiver. A centre conductor runs inside a tubular **shield**, separated by a dielectric,
which keeps the signal contained and the [impedance](/reference/impedance/) constant
(commonly 50 Ω).[^wiki] Every metre and every connector adds
[loss](/reference/attenuation/) — and, crucially, that loss grows with frequency, so the
same cable that is fine at 30 MHz can be a poor choice at 1 GHz.

<figure class="figure" markdown="0">
<svg viewBox="0 0 320 130" role="img" aria-label="A cutaway of coaxial cable showing the centre conductor, dielectric, braided shield, and outer jacket as concentric layers." xmlns="http://www.w3.org/2000/svg">
  <ellipse cx="160" cy="65" rx="120" ry="48" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <ellipse cx="160" cy="65" rx="92" ry="36" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <ellipse cx="160" cy="65" rx="58" ry="22" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-opacity="0.6"/>
  <circle cx="160" cy="65" r="6" fill="currentColor"/>
  <g font-size="8" fill="currentColor"><text x="160" y="58" text-anchor="middle">core</text><text x="160" y="95" text-anchor="middle">dielectric</text><text x="160" y="118" text-anchor="middle">shield + jacket</text></g>
</svg>
<figcaption>Coax carries RF on a centre conductor inside a shield; keep runs short and quality high to limit loss.</figcaption>
</figure>

## How it works

The defining property of coax is that the signal travels as a field between the inner
conductor and the surrounding shield, fully enclosed. Two geometric facts follow. First, the
ratio of the shield's inner diameter to the core's diameter (and the dielectric constant
between them) fixes the **characteristic impedance** — the impedance a matched line presents
regardless of length.[^z] Radio gear standardises on 50 Ω for transmit-capable systems and
75 Ω for video/broadcast reception; mixing them causes a mismatch and reflections. Second,
the shield keeps external interference out and the signal in, which is why coax outperforms
open wire in noisy environments.

Loss comes from two mechanisms: **conductor loss** (resistance in the copper, which rises
with the square root of frequency as current crowds into the skin) and **dielectric loss**
(the insulator absorbing energy, rising roughly linearly with frequency). The combined
effect is that cable attenuation, quoted in dB per 100 ft or per 100 m, climbs steeply with
frequency. A thin RG-174 patch cable might lose a fraction of a dB at HF but several dB at
1 GHz over the same length. The cable's **velocity factor** (typically 0.66–0.85) also means
the signal travels slower than in free space, which matters when cutting phasing lines or
stubs.

## Variants

- **RG-58 / RG-174** — thin, flexible, lossy; fine for short patch leads only.
- **RG-6** — 75 Ω, low-loss, cheap (TV cable); excellent for receive-only VHF/UHF if you
  accept the impedance mismatch, which is minor for reception.
- **LMR-400 / RG-213** — thick, low-loss 50 Ω runs for masthead installs; the go-to for any
  cable over a few metres at UHF.
- **Hardline / semi-rigid** — solid copper shield, lowest loss, used for long or
  high-frequency feeds.
- Cable terminates in connectors — [SMA](/reference/sma-connector/) on most SDR dongles,
  the larger [N-type](/reference/n-type-connector/) on antennas and low-loss runs — and each
  junction adds its own small loss and potential mismatch.

## In practice

A long or low-grade cable can quietly undo a good antenna: 6 dB of feedline loss throws away
three-quarters of the signal power before it ever reaches the SDR. Operators counter this two
ways — keep the feedline **short and thick**, or mount a
[low-noise amplifier](/reference/low-noise-amplifier/) at the antenna (fed via a
[bias tee](/reference/bias-tee/)) so the LNA's gain is applied *before* the cable loss and
suppresses it, per the Friis budget. A poor match at either end also raises
[SWR](/reference/standing-wave-ratio/) and, for receive, wastes signal; weatherproofing
outdoor connectors matters because water in coax raises loss dramatically.

## Relevance to SDR

Feedline choice is the most common overlooked variable in an SDR install: a hobbyist adds a
better antenna but leaves 15 m of thin RG-58 in place and sees no improvement. GopherTrunk
decodes whatever the SDR captures and has no view of the cable, but the coax is where a
weak trunking signal is most often lost before digitisation. On a UHF or 800 MHz P25 site,
using proper low-loss cable — or moving the SDR to the antenna and streaming
[IQ](/reference/iq-data/) back over the network — is frequently the single biggest
improvement to GopherTrunk's decode reliability.

## Where to buy

For the short SMA-terminated patch leads an SDR bench setup needs, a **NooElec SMA
coaxial cable connectivity kit** (around $18) bundles common lengths and adapters in
one box. For an actual feedline run of more than a few metres — especially at UHF or
800 MHz P25 — buy proper low-loss **RG8X** or **LMR-400** by the length instead of
relying on thin RG58.

<a class="btn btn--buy" href="https://www.amazon.com/dp/B077H87LTS?tag=gophertrunk-20" rel="nofollow sponsored noopener">Check price on Amazon &rarr;</a>

For connectors, adapters, and full cable choices, see the
[SDR cables and connectors guide](/sdr-cables-and-connectors/).

*As an Amazon Associate, GopherTrunk earns from qualifying purchases — at no extra
cost to you. It never changes what we recommend.*

## Sources

[^wiki]: [Coaxial cable](https://en.wikipedia.org/wiki/Coaxial_cable) — Wikipedia, on coax construction, characteristic impedance, and frequency-dependent loss.
[^z]: [Characteristic impedance](https://en.wikipedia.org/wiki/Characteristic_impedance) — Wikipedia, on the length-independent impedance set by a line's geometry and why matched lines avoid reflections.
