---
slug: dpmr
title: dPMR
entry_type: protocol
category: protocols
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
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "dPMR (Wikipedia)", url: https://en.wikipedia.org/wiki/DPMR }
---

**dPMR** (**digital private mobile radio**) is an [ETSI](/reference/etsi/) narrowband
standard using **6.25 kHz** [FDMA](/reference/fdma/) channels with
[4FSK](/reference/frequency-shift-keying/) modulation. It is technically close to
[NXDN](/reference/nxdn/) and serves the same low-cost, spectrum-efficient niche.

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
open alternative for low-tier business radio.

## Deployment

Used in European and international business radio; less common in North America than
[DMR](/reference/dmr/) or [NXDN](/reference/nxdn/).

## Decoding it with GopherTrunk

dPMR decodes similarly to NXDN given its shared narrowband 4FSK design. See
[Status](/status.html).
