---
slug: itu
title: International Telecommunication Union (ITU)
entry_type: organization
category: organizations
description: The ITU is the United Nations agency for information and communication technologies, allocating the global radio spectrum and publishing radio regulations and standards.
keywords: ITU, International Telecommunication Union, spectrum allocation, radio regulations, United Nations, ITU-R, WRC, Region 1 2 3
aka: [ITU, International Telecommunication Union]
autolink: true
infobox:
  - { label: Type, value: UN specialised agency }
  - { label: Founded, value: "1865" }
  - { label: Role, value: Global spectrum allocation, standards }
see_also: [itu-r, wrc, fcc, etsi, ntia, icao, frequency-bands]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/rf-sdr/frequency-and-spectrum/ }
cite_urls:
  - https://www.itu.int/
  - https://en.wikipedia.org/wiki/International_Telecommunication_Union
---

The **International Telecommunication Union** (**ITU**) is the United Nations specialised
agency for information and communication technologies.[^wiki] It coordinates the **global radio
[spectrum](/reference/frequency-bands/)** and publishes the international Radio Regulations —
the treaty that every national regulator writes its own rules within.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The ITU dividing the radio spectrum into allocated service blocks across its three world regions." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="20" text-anchor="middle" font-size="9" fill="currentColor">International spectrum allocations (Radio Regulations)</text>
  <line x1="30" y1="72" x2="430" y2="72" stroke="currentColor" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="1.1">
    <rect x="40" y="54" width="70" height="18" fill="currentColor" fill-opacity="0.12"/><rect x="110" y="54" width="60" height="18" fill="currentColor" fill-opacity="0.22"/><rect x="170" y="54" width="80" height="18" fill="currentColor" fill-opacity="0.12"/><rect x="250" y="54" width="60" height="18" fill="currentColor" fill-opacity="0.22"/><rect x="310" y="54" width="110" height="18" fill="currentColor" fill-opacity="0.12"/>
  </g>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <text x="75" y="66">maritime</text><text x="140" y="66">aviation</text><text x="210" y="66">broadcast</text><text x="280" y="66">amateur</text><text x="365" y="66">cellular</text>
  </g>
  <text x="230" y="98" text-anchor="middle" font-size="8.5" fill="currentColor">Region 1 (EMEA) · Region 2 (Americas) · Region 3 (Asia-Pacific)</text>
  <text x="230" y="112" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.8">national regulators (FCC, Ofcom, …) implement within this framework</text>
</svg>
<figcaption>The ITU coordinates global spectrum allocation and radio regulations across all countries, dividing the world into three regions.</figcaption>
</figure>

## Overview

Founded in 1865 as the International Telegraph Union, the ITU is the oldest of the UN's
specialised agencies and today has more than 190 member states plus hundreds of private-sector
and academic members. Its work is carried out through three sectors. **ITU-R** (Radiocommunication)
manages the radio-frequency spectrum and satellite orbits and publishes the Radio Regulations;
its recommendations underpin systems such as [AIS](/reference/ais/) and
[DSC](/reference/dsc/). **ITU-T** (Standardization) produces telecommunication standards,
including the widely used G-series voice codecs like [G.711](/reference/g711/) and
[G.729](/reference/g729/). **ITU-D** (Development) works on bridging the connectivity gap in
developing countries.

The central document for radio is the **Table of Frequency Allocations**, which slices the
spectrum from 8.3 kHz upward into bands and assigns each to one or more radio *services*
(fixed, mobile, broadcasting, maritime, aeronautical, amateur, radiolocation, space, and
more). To manage regional differences the ITU divides the globe into three regions — Region 1
(Europe, Africa, the Middle East and northern Asia), Region 2 (the Americas) and Region 3
(most of Asia-Pacific) — each of which can carry slightly different allocations. This treaty
framework is periodically revised at the **World Radiocommunication Conference**
([WRC](/reference/wrc/)), where member states negotiate changes. National regulators such as
the [FCC](/reference/fcc/), [Ofcom](/reference/ofcom/) and the US [NTIA](/reference/ntia/)
then translate these international allocations into domestic band plans and licences. The ITU
also administers the international callsign prefix blocks and the Master International Frequency
Register, and coordinates satellite filings to avoid orbital and frequency interference.

## Relevance to SDR

The ITU's allocations are the reason specific signals live in specific bands worldwide, which
is what makes an SDR usable at all: when you see marine traffic near 156–162 MHz, aircraft
[ADS-B](/reference/ads-b/) at 1090 MHz, or the amateur bands where hobbyist digital modes
live, you are looking directly at the Table of Frequency Allocations. Understanding the region
you are in explains why a band that carries one service in North America may carry a different
one in Europe — a common source of confusion when following online tuning guides written for
another region. The three-region split also explains why some land-mobile
[trunked-radio](/reference/trunked-radio/) systems sit at different frequencies on different
continents even when they run the same protocol.

Because the [ITU-R](/reference/itu-r/) sector authors the recommendations that describe several
signals GopherTrunk and other decoders handle (AIS message formats, DSC, and the modulation
of some maritime and aeronautical systems), the ITU indirectly shapes the bitstreams a decoder
must parse. GopherTrunk itself does not implement any ITU allocation logic — it is a receiver
and decoder — but every capture you feed it sits somewhere in the ITU's global plan, and the
Radio Regulations are the ultimate authority on what a given slice of spectrum is *supposed*
to contain. For legal monitoring, remember that the ITU sets the international framework but
your own national regulator sets what you may lawfully receive.

## Sources

[^home]: [International Telecommunication Union](https://www.itu.int/) — the ITU's official site, for the Radio Regulations, the Table of Frequency Allocations, and global spectrum coordination.
[^wiki]: [International Telecommunication Union](https://en.wikipedia.org/wiki/International_Telecommunication_Union) — Wikipedia, for the agency's history, sector structure, and role.
