---
slug: solas
title: SOLAS
entry_type: concept
category: aviation-marine
description: SOLAS is the IMO's core maritime safety treaty; it mandates the GMDSS, which is why the DSC, NAVTEX, EPIRB, and AIS signals an SDR listener hears exist and are standardized.
keywords: SOLAS, Safety of Life at Sea, GMDSS, IMO, maritime safety convention, DSC, NAVTEX, EPIRB, AIS, distress alerting
aka: [SOLAS, Safety of Life at Sea, GMDSS]
autolink: true
infobox:
  - { label: Full name, value: "Int'l Convention for the Safety of Life at Sea" }
  - { label: Adopted by, value: "IMO" }
  - { label: Mandates, value: GMDSS radio systems }
see_also: [ais, dsc, navtex, epirb-406, cospas-sarsat, imo, marine-vhf]
cite_urls:
  - https://en.wikipedia.org/wiki/SOLAS_Convention
  - https://en.wikipedia.org/wiki/Global_Maritime_Distress_and_Safety_System
---

**SOLAS** — the **International Convention for the Safety of Life at Sea** — is the core
maritime safety treaty, adopted and maintained by the [IMO](/reference/imo/).[^wiki] For an
SDR listener its importance is indirect but decisive: SOLAS is the regulation that *requires*
ships to carry particular radio equipment, so the maritime signals you hear on the bands exist
and are standardized because a treaty says they must. Above set tonnage thresholds, SOLAS
mandates the **Global Maritime Distress and Safety System** (**GMDSS**) — the coordinated suite
of radio systems that handles distress alerting, safety broadcasts, and position reporting at
sea.[^gmdss]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="A ship carrying the GMDSS radio systems SOLAS mandates: AIS for position reporting, DSC for digital distress alerting, a NAVTEX receiver for safety broadcasts, and an EPIRB satellite distress beacon." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <path d="M120 150 L340 150 L320 178 L140 178 Z" fill="currentColor" fill-opacity="0.1"/>
    <line x1="230" y1="150" x2="230" y2="95"/>
    <line x1="210" y1="120" x2="250" y2="120"/>
  </g>
  <g font-size="8.5" fill="currentColor" text-anchor="middle" stroke="none">
    <text x="230" y="168">SOLAS ship — GMDSS fit</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="150" y1="150" x2="70" y2="60" marker-end="url(#so_a)"/>
    <line x1="200" y1="150" x2="175" y2="55" marker-end="url(#so_a)"/>
    <line x1="270" y1="150" x2="300" y2="55" marker-end="url(#so_a)"/>
    <line x1="320" y1="150" x2="400" y2="60" marker-end="url(#so_a)"/>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle" stroke="none">
    <text x="60" y="48">AIS</text><text x="60" y="38" font-size="7">position</text>
    <text x="172" y="48">DSC</text><text x="172" y="38" font-size="7">distress alert</text>
    <text x="303" y="48">NAVTEX</text><text x="303" y="38" font-size="7">safety info</text>
    <text x="405" y="48">EPIRB</text><text x="405" y="38" font-size="7">sat beacon</text>
  </g>
  <defs><marker id="so_a" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>SOLAS obliges ships to carry a GMDSS fit; each mandated system is a signal an SDR user can receive.</figcaption>
</figure>

## What SOLAS requires

SOLAS is organised into chapters covering construction, fire protection, life-saving
appliances, and — the part that matters here — **radiocommunications**. Its radio chapter
requires ships on international voyages, above defined tonnages, to carry GMDSS equipment
appropriate to the sea areas they operate in. Rather than specify one radio, GMDSS layers
several systems so that a distress can always be raised and safety information always
received, whether a ship is near shore or mid-ocean. The treaty sets the *requirement*; the
technical formats are standardized internationally so that equipment from any
manufacturer interoperates worldwide.

That mandate is why a handful of maritime signals are so consistent and widespread:

- **[Digital Selective Calling](/reference/dsc/)** (DSC) — the digital distress-alerting layer
  on [marine VHF](/reference/marine-vhf/) channel 70 and on MF/HF, used to send an automated,
  addressed distress or safety call that includes the ship's identity and position.
- **[NAVTEX](/reference/navtex/)** — the 518 kHz maritime safety broadcast service delivering
  navigational warnings, weather, and search-and-rescue notices to a printer or screen aboard.
- **[EPIRB](/reference/epirb-406/)** — the 406 MHz emergency position-indicating radio beacon
  that, when a vessel sinks, transmits a distress signal to the
  [Cospas-Sarsat](/reference/cospas-sarsat/) satellite network.
- **[AIS](/reference/ais/)** — the Automatic Identification System, broadcasting each ship's
  identity, position, course, and speed on VHF for collision avoidance and traffic monitoring.

Larger ships also carry satellite terminals for long-range distress and communications, closing
the coverage gap beyond terrestrial range.

## Why it matters to the listener

The practical takeaway is that SOLAS is the **regulatory driver** behind nearly everything an
SDR user hears on the marine bands. AIS is dense and continuous precisely because every SOLAS
ship must transmit it; DSC calls appear on channel 70 because the treaty requires the watch;
NAVTEX runs on schedule on 518 kHz because ships must be able to receive safety broadcasts.
These are open, standardized, unencrypted signals — designed for interoperability and safety —
which is exactly what makes them so accessible to a hobbyist receiver.

## Relevance to SDR

Maritime signals fall outside GopherTrunk's land-mobile trunking focus, so GopherTrunk does not
decode AIS, DSC, NAVTEX, or EPIRB itself; dedicated tools and a general-coverage or VHF receiver
handle those. SOLAS is documented here as the *why* behind them: it explains why the maritime RF
landscape looks the way it does and why its systems are so uniform across the world's oceans — a
single treaty, administered by the [IMO](/reference/imo/), obliges every qualifying ship to carry
the same standardized GMDSS radios. Understanding that connection turns a scattered set of marine
signals into a coherent, purpose-built safety network.

## Sources

[^wiki]: [SOLAS Convention](https://en.wikipedia.org/wiki/SOLAS_Convention) — Wikipedia, for SOLAS's status as the core maritime safety treaty and its radiocommunications requirements.
[^gmdss]: [Global Maritime Distress and Safety System](https://en.wikipedia.org/wiki/Global_Maritime_Distress_and_Safety_System) — Wikipedia, for the GMDSS systems (DSC, NAVTEX, EPIRB, AIS, satellite) SOLAS mandates.
