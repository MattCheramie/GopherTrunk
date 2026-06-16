---
slug: dvsi
title: Digital Voice Systems (DVSI)
entry_type: organization
category: organizations
description: Digital Voice Systems, Inc. (DVSI) is the company that develops and licenses the IMBE and AMBE vocoder families used by P25, DMR, NXDN, and D-STAR.
keywords: DVSI, Digital Voice Systems, IMBE, AMBE, AMBE+2, vocoder licensing, patents
aka: [DVSI, Digital Voice Systems]
autolink: true
infobox:
  - { label: Type, value: Company }
  - { label: Products, value: IMBE, AMBE, AMBE+2 vocoders }
  - { label: Note, value: Patented/licensed codecs }
see_also: [vocoder, imbe, ambe, ambe-plus-2, multi-band-excitation]
related_lessons:
  - { title: "Vocoders — IMBE & AMBE+2", url: /learn/vocoders/ }
external:
  - { title: "Digital Voice Systems (Wikipedia)", url: https://en.wikipedia.org/wiki/Multi-Band_Excitation }
---

**Digital Voice Systems, Inc.** (**DVSI**) is the company that develops and licenses the
[IMBE](/reference/imbe/) and [AMBE](/reference/ambe/) / [AMBE+2](/reference/ambe-plus-2/)
[vocoder](/reference/vocoder/) families based on
[multi-band excitation](/reference/multi-band-excitation/).

## Overview

DVSI's codecs are used by [P25](/reference/project-25/), [DMR](/reference/dmr/),
[NXDN](/reference/nxdn/), and [D-STAR](/reference/d-star/). Their patented, licensed nature
is why open alternatives like [Codec 2](/reference/codec2/) exist.

## Relevance to SDR

GopherTrunk implements the relevant vocoders in pure Go to decode digital voice, while
respecting the patent/licensing landscape DVSI created.
