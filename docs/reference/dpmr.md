---
slug: dpmr
title: dPMR
entry_type: protocol
category: land-mobile-trunking
description: dPMR (digital private mobile radio) is an ETSI narrowband 4FSK standard using 6.25 kHz FDMA channels, closely related to NXDN, for licence-free and licensed business radio.
keywords: dPMR, digital private mobile radio, ETSI, 6.25 kHz, narrowband, FDMA, 4FSK
aka: [dPMR]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio }
  - { label: Standards body, value: ETSI }
  - { label: Access, value: FDMA }
  - { label: Channel spacing, value: 6.25 kHz }
  - { label: Modulation, value: 4FSK (2400 baud) }
  - { label: Vocoder, value: AMBE+2 }
  - { label: GopherTrunk support, value: Decoded }
see_also: [nxdn, frequency-shift-keying, ambe-plus-2, etsi, fdma]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/DPMR
---

**dPMR** (**digital private mobile radio**) is an [ETSI](/reference/etsi/) narrowband
standard using **6.25 kHz** [FDMA](/reference/fdma/) channels with
[4FSK](/reference/frequency-shift-keying/) modulation. It is technically close to
[NXDN](/reference/nxdn/) and serves the same low-cost, spectrum-efficient niche.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="Narrow 6.25 kHz FDMA channels for dPMR." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="135" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#fa_dpmr)"/>
  <text x="22" y="80" font-size="9" fill="currentColor" transform="rotate(-90 22 80)">frequency</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1">
    <rect x="50" y="28" width="260" height="22"/><rect x="50" y="60" width="260" height="22"/><rect x="50" y="92" width="260" height="22"/>
  </g>
  <g font-size="8.5" fill="currentColor"><text x="180" y="43" text-anchor="middle">one call per channel (6.25 kHz)</text><text x="180" y="75" text-anchor="middle">one call per channel</text><text x="180" y="107" text-anchor="middle">one call per channel</text></g>
  <defs><marker id="fa_dpmr" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>dPMR uses 6.25 kHz FDMA channels, like its close relative NXDN.</figcaption>
</figure>

## Overview

dPMR comes in licence-free (Mode 1) and licensed conventional/trunked variants. Its
6.25 kHz channelisation packs more users into a band than 12.5 kHz systems.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | FDMA |
| Channel | 6.25 kHz |
| Modulation | 4FSK, 2400 baud |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) |

## History

Standardised by ETSI in parallel with the narrowbanding push of the late 2000s as an
open alternative for low-tier business radio.[^wiki]

## Deployment

Used in European and international business radio; less common in North America than
[DMR](/reference/dmr/) or [NXDN](/reference/nxdn/).

## Decoding it with GopherTrunk

dPMR decodes similarly to NXDN given its shared narrowband 4FSK design. See
[Status](/status.html).

## Sources

[^wiki]: [dPMR](https://en.wikipedia.org/wiki/DPMR) — Wikipedia, for the ETSI 6.25 kHz FDMA 4FSK standard and its relationship to NXDN.
