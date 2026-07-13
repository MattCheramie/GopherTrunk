---
slug: attenuation
title: Attenuation
entry_type: term
category: rf-fundamentals
description: Attenuation is the reduction in signal strength as it passes through a medium, cable, or obstacle, expressed in decibels.
keywords: attenuation, loss, dB, coax loss, signal weakening, insertion loss, absorption
aka: [attenuation, loss]
autolink: true
infobox:
  - { label: Type, value: Signal loss }
  - { label: Unit, value: Decibels (dB) }
  - { label: Causes, value: Distance, cable, obstacles, filters }
see_also: [path-loss, decibel, radio-propagation, antenna, free-space-path-loss, link-budget, coaxial-cable]
related_lessons:
  - { title: "How signals travel", url: /learn/rf-sdr/propagation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Attenuation
  - https://en.wikipedia.org/wiki/Atmospheric_attenuation
---

**Attenuation** is the reduction of signal strength as energy passes through a medium,
cable, connector, or obstacle.[^wiki] It is expressed in [decibels](/reference/decibel/)
and subtracts directly from a power budget — a loss of 6 dB means only a quarter of the
power survives.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A sine wave whose amplitude shrinks steadily from left to right as it is attenuated travelling from transmitter to receiver." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="65" x2="440" y2="65" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 65 q15 -40 30 0 t30 0 q15 -30 30 0 t30 0 q15 -20 30 0 t30 0 q15 -12 30 0 t30 0 q15 -7 30 0 t30 0 q15 -4 30 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="30" y="115" font-size="10" fill="currentColor">transmitter</text>
  <text x="430" y="115" font-size="10" fill="currentColor" text-anchor="end">weaker at receiver</text>
  <text x="230" y="130" font-size="9" fill="currentColor" text-anchor="middle">−6 dB = ×¼ power · −3 dB = ×½ power</text>
</svg>
<figcaption>Attenuation is the loss of signal strength as energy spreads out and is absorbed along the path.</figcaption>
</figure>

## How it works

Attenuation comes from two mechanisms: **spreading**, where a fixed amount of power is
diluted over a larger area or volume, and **absorption**, where the medium converts RF
energy into heat. Both are conventionally logged in decibels because losses then simply
add to the gains and other losses in a chain, and a total is a running sum.

The main contributors in a receive path are:

- **Cable loss.** Coax loss rises with frequency and length. A cheap RG-58 run might lose
  several dB per 10 m at 400 MHz; low-loss cable and short runs cut this. Because a lossy
  cable ahead of the amplifier also degrades [noise figure](/reference/noise-figure/), the
  loss before the first amplifier is doubly harmful.
- **Connector and component insertion loss.** Every connector, filter, splitter, and
  switch adds a fraction of a dB to a few dB. These accumulate.
- **Obstacles.** Walls, foliage, and terrain absorb and scatter energy; loss through
  material generally worsens at higher frequencies, which is why UHF penetrates buildings
  less well than VHF.
- **Atmospheric absorption.** Above ~10 GHz, oxygen and water vapour absorb RF, and rain
  adds "rain fade"; below UHF this is negligible.[^atm]

The frequency dependence is the recurring theme: for most of these mechanisms, higher
frequencies attenuate more. That single fact shapes band choice, cable choice, and site
planning.

## In practice

Attenuation is not always the enemy. A deliberate **attenuator** is inserted to protect a
receiver from strong signals that would otherwise overload the front end and drive the ADC
past [full scale](/reference/dbfs/), spraying
[intermodulation](/reference/intermodulation/) across the band. Trading a few dB of wanted
signal for headroom against a nearby transmitter is often a net win. Filters attenuate
out-of-band energy for the same protective reason.

In a [link budget](/reference/link-budget/), every loss is entered as a negative dB term:
transmit power, plus antenna gains, minus [path loss](/reference/path-loss/), minus cable
and connector attenuation, yields the received power compared against
[receiver sensitivity](/reference/receiver-sensitivity/). Knowing where the dB are going
tells you where to spend effort — usually shortening the cable run and adding a preamp at
the antenna.

## Relevance to SDR

Free-space spreading is a specific kind of attenuation called
[path loss](/reference/path-loss/) (or [free-space path loss](/reference/free-space-path-loss/)
in the idealised case). For the operator, controllable attenuation lives in the feedline:
keeping [coax](/reference/coaxial-cable/) runs short and connectors clean minimises loss
between [antenna](/reference/antenna/) and receiver, preserving
[SNR](/reference/signal-to-noise-ratio/). GopherTrunk sees only what survives the path and
the cable — attenuation upstream of the SDR is loss it cannot recover.

## Sources

[^wiki]: [Attenuation](https://en.wikipedia.org/wiki/Attenuation) — Wikipedia, the general definition, spreading-versus-absorption mechanisms, and causes of signal loss.
[^atm]: [Atmospheric attenuation](https://en.wikipedia.org/wiki/Atmospheric_attenuation) — Wikipedia, absorption by oxygen, water vapour, and rain at higher frequencies.
