---
slug: ccitt
title: CCITT
entry_type: organization
category: organizations
description: CCITT, the International Telegraph and Telephone Consultative Committee, was the ITU body renamed ITU-T in 1993; its names still label standards such as ITA2/Baudot, the V-series modems, and the G-series codecs.
keywords: CCITT, ITU-T, International Telegraph and Telephone Consultative Committee, ITA2, Baudot, V-series, G-series, G.711, telegraph alphabet
aka: [CCITT, International Telegraph and Telephone Consultative Committee]
autolink: true
infobox:
  - { label: Type, value: Former ITU standards body }
  - { label: Focus, value: Telegraph & telephone standards }
  - { label: Renamed, value: "ITU-T (1993)" }
  - { label: Standards, value: "ITA2, V-series, G-series" }
see_also: [itu, itu-r, g711, rtty, modem]
cite_urls:
  - https://www.itu.int/en/ITU-T/
  - https://en.wikipedia.org/wiki/ITU-T
---

**CCITT** (the **International Telegraph and Telephone Consultative Committee**, French
*Comité Consultatif International Téléphonique et Télégraphique*) was the
[ITU](/reference/itu/) body renamed **ITU-T** in 1993. Its names still label standards behind
**[RTTY](/reference/rtty/)**, telephone **[modems](/reference/modem/)**, and speech
codecs.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A lineage arrow from CCITT, active until 1993, to ITU-T from 1993 to the present, with example standards each produced." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="40" width="150" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="105" y="56">CCITT</text><text x="105" y="68" font-size="7.5">to 1993</text>
    <rect x="280" y="40" width="150" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="355" y="56">ITU-T</text><text x="355" y="68" font-size="7.5">1993 – present</text>
    <line x1="180" y1="58" x2="279" y2="58" stroke="currentColor" stroke-width="1.4" marker-end="url(#rel_ccitt)"/>
    <text x="230" y="50" font-size="7.5">renamed</text>
    <text x="105" y="98" font-size="7.5" fill-opacity="0.85">ITA2 · V-series · G.711</text>
    <text x="355" y="98" font-size="7.5" fill-opacity="0.85">continues the numbering</text>
  </g>
  <defs><marker id="rel_ccitt" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>CCITT became ITU-T in 1993; standards it issued are still cited under their CCITT names.</figcaption>
</figure>

## Overview

The CCITT was, for most of the 20th century, the standards arm of the
[ITU](/reference/itu/) responsible for telegraphy and telephony. Meeting in study groups and
plenary assemblies, it produced the "Recommendations" that made national telephone and
telegraph networks interconnect across borders. In the 1992–1993 ITU reorganisation the CCITT
was renamed the **ITU Telecommunication Standardization Sector (ITU-T)**, which carries on the
same work and the same Recommendation numbering; its sister sector, [ITU-R](/reference/itu-r/),
handles radiocommunication. Because the renaming preserved the numbering, a great many
standards are still known and cited by their original CCITT names.

Several of those touch signals an SDR user meets. The **CCITT No. 2 telegraph alphabet** —
also called **ITA2** or Baudot — is the 5-bit character code that underlies
**[RTTY](/reference/rtty/)** (radioteletype), still heard on the HF bands. The **V-series**
Recommendations defined the telephone-line **[modem](/reference/modem/)** standards (V.21,
V.22, V.32, V.34, and beyond) whose modulation and handshaking ideas echo through modern data
modes. And the **G-series** defined the digital speech codecs of the telephone network, most
famously **[G.711](/reference/g711/)** pulse-code modulation and its relatives, which set the
template for how voice is digitised. Presenting CCITT as the historical predecessor of ITU-T is
the accurate framing: the organisation did not disappear so much as change its name and keep
publishing.

## Relevance to SDR

CCITT-lineage standards describe the letters and tones of some of the oldest decodable digital
signals on the air. When an SDR user copies **[RTTY](/reference/rtty/)**, the decoder is
mapping mark/space tones back into ITA2 characters exactly as the CCITT No. 2 alphabet defines
them. Older HF and phone-patch data modes trace their modulation lineage to the V-series, and
the G-series codecs describe how sampled voice is represented in countless digital systems.
Knowing that these carry CCITT names — now formally ITU-T — helps a listener follow
documentation that predates the 1993 rename.

These legacy modes sit outside GopherTrunk's land-mobile trunking focus, so GopherTrunk does
not itself decode RTTY or telephone modems; dedicated tools handle those. The reference stands
as context for the wider RF landscape an SDR user explores, and it resolves the common
confusion between the historical CCITT name and today's ITU-T.

## Sources

[^home]: [ITU Telecommunication Standardization Sector (ITU-T)](https://www.itu.int/en/ITU-T/) — the official site of the sector that continues the CCITT's work and Recommendation numbering.
[^wiki]: [ITU-T](https://en.wikipedia.org/wiki/ITU-T) — Wikipedia, for the CCITT's history and its 1993 renaming to ITU-T.
