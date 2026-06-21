---
slug: nxdn
title: NXDN
entry_type: protocol
category: protocols
description: NXDN is a narrowband digital land-mobile radio standard by Kenwood and Icom, using 4FSK in 6.25 or 12.5 kHz channels with the AMBE+2 vocoder, in conventional and trunked forms.
keywords: NXDN, NEXEDGE, IDAS, narrowband digital, 6.25 kHz, Kenwood, Icom, 4FSK
aka: [NXDN, NEXEDGE, IDAS]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Developers, value: Kenwood & Icom }
  - { label: Access, value: FDMA (conventional or trunked) }
  - { label: Channel spacing, value: 6.25 kHz or 12.5 kHz }
  - { label: Modulation, value: 4FSK (2400/4800 baud) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded }
see_also: [dpmr, frequency-shift-keying, ambe-plus-2, trunked-radio, control-channel]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/NXDN
---

**NXDN** is a **narrowband** digital land-mobile radio standard jointly developed by
**Kenwood** (NEXEDGE) and **Icom** (IDAS). It uses [4FSK](/reference/frequency-shift-keying/)
in very narrow 6.25 kHz channels (or 12.5 kHz) and the
[AMBE+2](/reference/ambe-plus-2/) vocoder.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="Narrow 6.25 kHz FDMA channels for NXDN." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="135" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#fa_nxdn)"/>
  <text x="22" y="80" font-size="9" fill="currentColor" transform="rotate(-90 22 80)">frequency</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1">
    <rect x="50" y="28" width="260" height="22"/><rect x="50" y="60" width="260" height="22"/><rect x="50" y="92" width="260" height="22"/>
  </g>
  <g font-size="8.5" fill="currentColor"><text x="180" y="43" text-anchor="middle">one call per channel (6.25 kHz)</text><text x="180" y="75" text-anchor="middle">one call per channel</text><text x="180" y="107" text-anchor="middle">one call per channel</text></g>
  <defs><marker id="fa_nxdn" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>NXDN packs calls into very narrow 6.25 kHz (or 12.5 kHz) FDMA channels.</figcaption>
</figure>

## Overview

NXDN emphasises spectrum efficiency, fitting a channel into as little as 6.25 kHz —
half the width of typical [DMR](/reference/dmr/) or [P25](/reference/project-25/)
channels. It supports both conventional operation and trunking with a
[control channel](/reference/control-channel/).

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Channel | 6.25 kHz (2400 baud) or 12.5 kHz (4800 baud) |
| Modulation | 4FSK |
| Vocoder | AMBE+2 |

## History

Introduced in the late 2000s as Kenwood and Icom's common air interface for
narrowband digital business radio.[^wiki]

## Deployment

Common in business, transport, and utility fleets, especially where regulators reward
6.25 kHz channelisation.

## Decoding it with GopherTrunk

GopherTrunk decodes NXDN voice and follows trunked NXDN control channels. See
[Status](/status.html).

## Sources

[^wiki]: [NXDN](https://en.wikipedia.org/wiki/NXDN) — Wikipedia, for the Kenwood/Icom narrowband 4FSK air interface, channel widths, and the AMBE+2 vocoder.
