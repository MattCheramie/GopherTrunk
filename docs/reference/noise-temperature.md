---
slug: noise-temperature
title: Noise temperature (Te)
entry_type: term
category: rf-metrics
description: Noise temperature expresses a component's added noise as an equivalent kelvin temperature; it adds linearly in cascades and underpins the satellite G/T figure.
keywords: noise temperature, equivalent noise temperature, Te, system noise temperature, G/T, antenna temperature, sky temperature, kelvin, low-noise
aka: [Te, equivalent noise temperature, effective noise temperature]
autolink: true
infobox:
  - { label: Symbol, value: "T_e" }
  - { label: Unit, value: Kelvin (K) }
  - { label: Relation, value: "T_e = 290·(F − 1)" }
see_also: [noise-figure, thermal-noise, low-noise-amplifier, receiver-sensitivity, signal-to-noise-ratio, noise-floor]
cite_urls:
  - https://en.wikipedia.org/wiki/Noise_temperature
  - https://en.wikipedia.org/wiki/Johnson%E2%80%93Nyquist_noise
---

**Noise temperature** (**T_e**, or *equivalent noise temperature*) restates a
component's self-generated noise as the physical temperature a resistor would need
to produce the same [thermal noise](/reference/thermal-noise/) power.[^wiki] Instead
of saying "this amplifier has a [noise figure](/reference/noise-figure/) of 0.5 dB,"
you say "it adds noise equivalent to a 35 K source." The two descriptions carry
identical information — T_e = 290·(F − 1) — but noise temperature resolves the tiny
noise contributions of very-low-noise systems far more clearly than fractions of a
decibel, so it is the preferred metric for satellite ground stations, deep-space
links, and radio astronomy.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A dish antenna pointed at cold sky feeding a receiver, with the total system noise temperature being the sum of antenna temperature and receiver equivalent noise temperature, and the G over T figure of merit shown." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ntar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M40 40 Q20 75 40 110" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <line x1="40" y1="75" x2="90" y2="75" stroke="currentColor" stroke-width="1.4"/>
  <text x="70" y="34" text-anchor="middle" font-size="9" fill="currentColor">T_A (sky)</text>
  <line x1="90" y1="75" x2="118" y2="75" stroke="currentColor" stroke-width="1.4" marker-end="url(#ntar)"/>
  <rect x="120" y="55" width="90" height="40" rx="4" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="165" y="72" text-anchor="middle" font-size="10" fill="currentColor">LNA</text>
  <text x="165" y="87" text-anchor="middle" font-size="9" fill="currentColor">T_e, gain G</text>
  <line x1="210" y1="75" x2="250" y2="75" stroke="currentColor" stroke-width="1.4" marker-end="url(#ntar)"/>
  <text x="360" y="60" text-anchor="middle" font-size="11" fill="currentColor">T_sys = T_A + T_e</text>
  <text x="360" y="88" text-anchor="middle" font-size="11" fill="currentColor">G/T = figure of merit</text>
  <text x="230" y="150" text-anchor="middle" font-size="9.5" fill="currentColor" fill-opacity="0.85">T_e = 290·(F − 1)   —   cascade temps add linearly</text>
</svg>
<figcaption>A cold antenna plus a low-T_e amplifier gives a low system noise temperature; the ratio of antenna gain to system temperature, G/T, is the standard figure of merit for a receive station.</figcaption>
</figure>

## How it works

Rewrite kTB as k·T_e·B and the meaning is direct: a device with equivalent noise
temperature T_e adds the same noise a matched resistor at T_e kelvin would. The
conversion to and from [noise figure](/reference/noise-figure/) uses the 290 K
reference:

**T_e = 290·(F − 1)**  and  **F = 1 + T_e/290**

So NF = 3 dB (F = 2) is T_e = 290 K; NF = 0.5 dB is about 35 K; a cryogenically
cooled amplifier at 15 K corresponds to NF ≈ 0.22 dB — a difference the decibel
scale barely shows but the temperature scale makes obvious.

The other advantage is **linear cascading**. Where noise figures combine through
the Friis ratio formula, noise temperatures simply divide-and-add:

**T_sys = T₁ + T₂/G₁ + T₃/(G₁G₂) + …**

and at the antenna the total is just **T_sys = T_A + T_e**, the antenna's own
temperature (how warm the scene it looks at is) plus the receiver's equivalent
temperature. A dish aimed at cold sky sees a low T_A; aimed near the warm horizon or
the ground it sees a high one.

## In practice: G/T

For a receive station the headline figure of merit is **G/T** — antenna
[gain](/reference/antenna-gain/) divided by system noise temperature, quoted in
dB/K. It rolls the whole receive chain into one number: raise antenna gain or lower
system temperature and G/T improves, directly improving the carrier-to-noise the
station can achieve on a given downlink. This is why satellite operators specify
earth-station performance as G/T rather than gain or noise figure alone, and why
cooling the LNA (lowering T_e) is worth the cost on faint downlinks.

## Relevance to SDR

Most terrestrial SDR work — scanning P25, DMR, TETRA, pagers, ADS-B — is limited by
man-made noise and antenna temperature far above the receiver's own T_e, so noise
*figure* is the more convenient everyday metric. Noise temperature comes into its own
in the weak-signal corners of the hobby: L-band and higher satellite reception,
NOAA/GOES and Inmarsat downlinks, radio astronomy (hydrogen-line, pulsars), and
EME/moonbounce, where every kelvin of system temperature costs link margin and cooled
or very-low-T_e preamps earn their keep. Framing the front end as "35 K of added
noise" rather than "0.5 dB" makes the trade-offs between LNA choice, feedline loss,
and antenna pointing legible.

GopherTrunk does not compute or use noise temperature — it is a decode engine, not a
station-design tool — but the concept explains why a GopherTrunk user chasing a faint
signal should think about the whole receive chain's temperature budget (cold-sky
pointing, mast-mounted low-T_e LNA, minimal feedline before it) rather than expecting
software to close a link the RF front end never had margin for.

## Sources

[^wiki]: [Noise temperature](https://en.wikipedia.org/wiki/Noise_temperature) — Wikipedia, equivalent noise temperature, cascade addition, and the G/T figure of merit.
