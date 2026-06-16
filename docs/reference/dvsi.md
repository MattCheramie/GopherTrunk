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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="DVSI develops and licenses the AMBE and IMBE vocoders used by digital systems." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="61">DVSI</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="61">licenses</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="61">AMBE · IMBE</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_dvsi)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_dvsi)"/></g>
  </g>
  <defs><marker id="rel_dvsi" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>DVSI develops and licenses the patented AMBE and IMBE vocoders.</figcaption>
</figure>

## Overview

DVSI's codecs are used by [P25](/reference/project-25/), [DMR](/reference/dmr/),
[NXDN](/reference/nxdn/), and [D-STAR](/reference/d-star/). Their patented, licensed nature
is why open alternatives like [Codec 2](/reference/codec2/) exist.

## Relevance to SDR

GopherTrunk implements the relevant vocoders in pure Go to decode digital voice, while
respecting the patent/licensing landscape DVSI created.
