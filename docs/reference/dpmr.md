---
slug: dpmr
title: dPMR
entry_type: protocol
category: land-mobile-trunking
description: dPMR (digital private mobile radio) is an ETSI narrowband 4FSK standard using 6.25 kHz FDMA channels, closely related to NXDN, for licence-free and licensed business radio.
keywords: dPMR, digital private mobile radio, ETSI, 6.25 kHz, narrowband, FDMA, 4FSK, TS 102 490, TS 102 658, AMBE+2, PMR446
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
see_also: [nxdn, four-fsk, frequency-shift-keying, ambe-plus-2, fdma, etsi, trunked-radio]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/DPMR
  - https://www.etsi.org/technologies/digital-mobile-radio
---

**dPMR** (**digital private mobile radio**) is an [ETSI](/reference/etsi/) narrowband
digital land-mobile standard that carries voice and data in **6.25 kHz**
[FDMA](/reference/fdma/) channels using [4FSK](/reference/four-fsk/) modulation and the
[AMBE+2](/reference/ambe-plus-2/) [vocoder](/reference/vocoder/). It is technically very
close to [NXDN](/reference/nxdn/) — both are 6.25 kHz FDMA 4FSK systems — and serves the
same low-cost, spectrum-efficient niche as an open, multi-vendor alternative to
proprietary narrowband radio.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 360 150" role="img" aria-label="Narrow 6.25 kHz FDMA channels for dPMR, one call per channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="135" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4" marker-end="url(#fa_dpmr)"/>
  <text x="22" y="80" font-size="9" fill="currentColor" transform="rotate(-90 22 80)">frequency</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1">
    <rect x="50" y="28" width="260" height="22"/><rect x="50" y="60" width="260" height="22"/><rect x="50" y="92" width="260" height="22"/>
  </g>
  <g font-size="8.5" fill="currentColor"><text x="180" y="43" text-anchor="middle">one call per channel (6.25 kHz)</text><text x="180" y="75" text-anchor="middle">one call per channel</text><text x="180" y="107" text-anchor="middle">one call per channel</text></g>
  <defs><marker id="fa_dpmr" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>dPMR uses 6.25 kHz FDMA channels — one call per channel — like its close relative NXDN.</figcaption>
</figure>

## Overview

dPMR splits into tiers by intended use. **Mode 1** is a licence-free, direct-mode
(peer-to-peer) tier operating on the shared digital PMR446 channels in Europe — the
digital counterpart to analogue walkie-talkies, with no infrastructure. **Mode 2** is
licensed conventional operation through repeaters, and **Mode 3** adds trunking with a
control channel, talkgroups, and radio IDs. Across all tiers the physical layer is the
same: 6.25 kHz channels, 4FSK at 2400 symbols per second, and AMBE+2 speech coding. That
6.25 kHz channelisation is the headline feature — it packs twice as many talk paths into
a band as a 12.5 kHz system, and, like NXDN, it reaches this efficiency with plain
[FDMA](/reference/fdma/) rather than the two-slot [TDMA](/reference/tdma/) of
[DMR](/reference/dmr/).

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) |
| Channel spacing | 6.25 kHz |
| Modulation | [4FSK](/reference/four-fsk/), 2400 baud |
| Vocoder | [AMBE+2](/reference/ambe-plus-2/) |
| Tiers | Mode 1 (licence-free direct), Mode 2 (licensed conventional), Mode 3 (trunked) |
| Standards | ETSI TS 102 490 (Mode 1), TS 102 658 (Modes 2/3) |
| Voice/data | Digital voice, short data, status, GPS |

## History

dPMR was **standardised by ETSI** during the late-2000s narrowbanding push, published as
the TS 102 490 and TS 102 658 series, as an open specification any manufacturer could
implement.[^wiki][^etsi] It arrived alongside DMR from the same standards body: DMR
targeted the two-slot TDMA route to 6.25 kHz-equivalent efficiency, while dPMR took the
simpler FDMA route. Its near-identical relationship to Japan's NXDN reflects convergent
engineering — both settled on 6.25 kHz 4FSK with AMBE+2 as the natural low-cost narrowband
digital design of the era.

## Deployment

dPMR is used mainly in **European and international business radio** — retail, security,
hospitality, transport, and light industry — where its licence-free Mode 1 makes it an
easy analogue-to-digital upgrade for small users. It is less common in North America than
[DMR](/reference/dmr/) or [NXDN](/reference/nxdn/), reflecting regional vendor support and
regulatory history. Networked and trunked dPMR systems exist but are far less widespread
than DMR trunking.

## Decoding it with GopherTrunk

Because dPMR shares NXDN's narrowband 4FSK physical layer, GopherTrunk decodes it through
much the same receiver chain — symbol timing, discriminator, and frame sync — differing
mainly in frame layout and higher-layer signalling. Clear voice can be recovered where an
AMBE+2 vocoder is available; keyed encryption is out of scope for GopherTrunk's
receiver-only design. See the [Status](/status.html) page for the current state of dPMR
support.

## Sources

[^wiki]: [dPMR](https://en.wikipedia.org/wiki/DPMR) — Wikipedia, for the ETSI 6.25 kHz FDMA 4FSK standard, its Mode 1/2/3 tiers, and its relationship to NXDN.
[^etsi]: [ETSI — Digital Mobile Radio](https://www.etsi.org/technologies/digital-mobile-radio) — ETSI, the standards body, for the dPMR (TS 102 490 / TS 102 658) and DMR narrowband digital land-mobile specifications.
