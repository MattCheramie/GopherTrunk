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
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "NXDN (Wikipedia)", url: https://en.wikipedia.org/wiki/NXDN }
---

**NXDN** is a **narrowband** digital land-mobile radio standard jointly developed by
**Kenwood** (NEXEDGE) and **Icom** (IDAS). It uses [4FSK](/reference/frequency-shift-keying/)
in very narrow 6.25 kHz channels (or 12.5 kHz) and the
[AMBE+2](/reference/ambe-plus-2/) vocoder.

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
narrowband digital business radio.

## Deployment

Common in business, transport, and utility fleets, especially where regulators reward
6.25 kHz channelisation.

## Decoding it with GopherTrunk

GopherTrunk decodes NXDN voice and follows trunked NXDN control channels. See
[Status](/status.html).
