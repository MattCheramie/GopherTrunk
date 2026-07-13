---
slug: sma-connector
title: SMA connector
entry_type: hardware
category: rf-front-end
description: "SMA is a compact 50Ω threaded coaxial connector rated to 18 GHz, the near-universal RF port on SDR dongles, LNAs, and small antennas."
keywords: SMA connector, SubMiniature version A, 50 ohm, RP-SMA, reverse polarity SMA, coaxial connector, SDR antenna port, 18 GHz
aka: [SMA, "SubMiniature version A", RP-SMA]
autolink: true
infobox:
  - { label: Type, value: "Threaded coaxial connector" }
  - { label: Impedance, value: "50 Ω (75 Ω variant rare)" }
  - { label: Range, value: "DC to 18 GHz" }
  - { label: Coupling, value: "1/4-36 threaded" }
  - { label: TX, value: "Yes (low power)" }
see_also: [bnc-connector, n-type-connector, coaxial-cable, rtl-sdr, antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/SMA_connector
---

**SMA** (SubMiniature version A) is a small threaded [coaxial](/reference/coaxial-cable/)
connector with a nominal **50 Ω** impedance, usable from DC to about **18 GHz**.[^wiki] It
is the de-facto RF port on software-defined-radio receivers, low-noise amplifiers, filters,
and compact [antennas](/reference/antenna/), so almost every SDR accessory mates to it. The
male plug carries a captive pin and an internal thread; the female jack has a centre socket
and an external thread that the plug screws onto.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An SMA plug with a threaded coupling nut and centre pin mating to an SMA jack, with the 50-ohm coax feeding each side." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="smaar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="30" y="55" width="70" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="100" y1="75" x2="150" y2="75" stroke="currentColor" stroke-width="2"/>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <rect x="150" y="45" width="60" height="60"/>
    <line x1="150" y1="52" x2="210" y2="52"/><line x1="150" y1="98" x2="210" y2="98"/>
  </g>
  <circle cx="215" cy="75" r="4" fill="currentColor"/>
  <line x1="219" y1="75" x2="245" y2="75" stroke="currentColor" stroke-width="2"/>
  <g stroke="currentColor" stroke-width="1.4" fill="none"><rect x="245" y="45" width="55" height="60"/></g>
  <line x1="300" y1="75" x2="360" y2="75" stroke="currentColor" stroke-width="2"/>
  <rect x="360" y="55" width="70" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <g font-size="9" fill="currentColor"><text x="65" y="115" text-anchor="middle">coax</text><text x="180" y="128" text-anchor="middle">plug (thread)</text><text x="272" y="128" text-anchor="middle">jack</text><text x="395" y="115" text-anchor="middle">coax</text></g>
</svg>
<figcaption>An SMA plug screws onto a jack; the threaded coupling holds a repeatable 50 Ω interface up to 18 GHz.</figcaption>
</figure>

## Overview

SMA was developed in the 1960s for miniature semi-rigid cable and has since become one of
the most common RF interfaces in test equipment and hobby radio. The thread is a
**1/4-36 UNS** form, and the coupling nut is torqued (about 0.5–1 N·m on a precision joint)
to guarantee a consistent, low-reflection contact. Because the connector is physically
small, it suits the crowded edge of an SDR board where a bulkier
[BNC](/reference/bnc-connector/) or [N-type](/reference/n-type-connector/) would not fit.

## What it is

Mechanically, SMA trades quick handling for bandwidth and repeatability. Screwing the nut
takes longer than the bayonet twist of a BNC, but the threaded interface stays put under
vibration and holds its match at microwave frequencies where a BNC has long since become
lossy and reflective. Standard SMA uses a PTFE dielectric and is specified to roughly 500
mating cycles — far fewer than a BNC — so it is meant for connections you make and then
leave alone, not for constant swapping.

## Variants

- **RP-SMA (reverse polarity SMA)** swaps the genders of the centre contacts: the plug that
  would carry a pin instead carries a socket, and vice versa. The shell and thread are
  unchanged, so an RP-SMA plug will thread onto a standard SMA jack but the centre
  conductors never touch and no signal passes. RP-SMA was mandated on much consumer Wi-Fi
  gear to discourage swapping to higher-gain antennas; it is electrically identical to SMA,
  just keyed differently. **Mixing RP-SMA and standard SMA is the most common cause of a
  cable that mates mechanically yet passes no RF** — always check the pin/socket, not just
  the thread.
- **3.5 mm and 2.92 mm (K)** connectors are precision air-dielectric relatives that mate
  with SMA but extend usable range to 26–40 GHz.
- **75 Ω SMA** exists but is rare and not interchangeable with the 50 Ω type.

## Relevance to SDR

The SMA jack is the antenna port on nearly every SDR that matters to a trunking listener:
[RTL-SDR](/reference/rtl-sdr/) V3/V4 dongles, [Airspy](/reference/airspy/),
[HackRF](/reference/hackrf/), and the bias-tee-fed LNAs and filters between antenna and
radio. Adapters to [BNC](/reference/bnc-connector/), [N-type](/reference/n-type-connector/),
and older ports are cheap, but each adapter adds a small reflection and insertion loss, so a
clean run uses the fewest transitions possible. GopherTrunk is decode software and touches
no connectors directly, but the quality and correctness of the SMA joints in front of the
receiver set the signal-to-noise ratio it has to work with — a loose nut or an RP-SMA
mismatch shows up as missing or weak captures long before any DSP tuning matters.

## Sources

[^wiki]: [SMA connector](https://en.wikipedia.org/wiki/SMA_connector) — Wikipedia, on SMA construction, 50 Ω impedance, 18 GHz rating, thread form, and the RP-SMA variant.
