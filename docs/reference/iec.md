---
slug: iec
title: International Electrotechnical Commission (IEC)
entry_type: organization
category: organizations
description: The IEC, the International Electrotechnical Commission, is the international standards body for electrical and electronic technologies, co-running ISO/IEC JTC 1 and publishing standards such as IEC 61162 behind AIS.
keywords: IEC, International Electrotechnical Commission, ISO/IEC JTC 1, IEC 61162, NMEA, electrical standards, electronic standards, Geneva
aka: [IEC, International Electrotechnical Commission]
autolink: true
infobox:
  - { label: Type, value: International standards body }
  - { label: Focus, value: Electrical & electronic technologies }
  - { label: Founded, value: 1906 }
  - { label: Standards, value: "IEC 61162, ISO/IEC JTC 1" }
see_also: [itu, nist, etsi, ieee, ais]
cite_urls:
  - https://www.iec.ch/
  - https://en.wikipedia.org/wiki/International_Electrotechnical_Commission
---

**IEC** (the **International Electrotechnical Commission**) is the international standards
organisation for electrical, electronic, and related technologies. It partners with ISO in the
joint committee **ISO/IEC JTC 1** for information technology, and it publishes standards that
reach radio and data gear — including IEC 61162, the interface behind
**[AIS](/reference/ais/)** data.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="The IEC and ISO jointly run the ISO/IEC JTC 1 committee for IT standards, while the IEC separately publishes electrical, electronic, and interface standards." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="20" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="65" y="39">IEC</text>
    <rect x="20" y="88" width="90" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="65" y="107">ISO</text>
    <rect x="200" y="54" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="255" y="69">ISO/IEC JTC 1</text><text x="255" y="80" font-size="7.5">IT standards</text>
    <rect x="355" y="18" width="95" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="402" y="35" font-size="7.5">electrical / safety</text>
    <rect x="355" y="94" width="95" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="402" y="111" font-size="7.5">IEC 61162 · AIS</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="110" y1="40" x2="199" y2="64" marker-end="url(#rel_iec)"/><line x1="110" y1="100" x2="199" y2="74" marker-end="url(#rel_iec)"/><line x1="110" y1="32" x2="354" y2="30" marker-end="url(#rel_iec)"/><line x1="110" y1="45" x2="354" y2="106" marker-end="url(#rel_iec)"/></g>
  </g>
  <defs><marker id="rel_iec" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The IEC co-runs JTC 1 with ISO for IT standards and separately publishes the electrical and interface standards that touch radio hardware.</figcaption>
</figure>

## Overview

The IEC was founded in 1906 and, from its headquarters in Geneva, prepares and publishes
international standards for all electrical, electronic, and related technologies — collectively
"electrotechnology." Its national committees represent member countries, and its standards
range from the fundamentals of quantities and units to product safety, connectors, and
electromagnetic compatibility. The IEC also administers conformity-assessment schemes that let
equipment certified in one country be accepted in others.

Where the IEC's work meets the SDR world is twofold. First, it partners with ISO in the joint
technical committee **ISO/IEC JTC 1**, which develops information-technology standards — from
character encodings to coding and interconnection — that underpin the digital systems radios
increasingly embed. Second, the IEC publishes standards that directly touch radio and
navigation data: connector and safety standards for the hardware, and maritime interface
standards such as **IEC 61162**, the internationally standardised counterpart of the NMEA 0183
serial protocol used to move **[AIS](/reference/ais/)**, GPS, and other navigation data between
shipboard equipment. It is worth distinguishing the IEC from three neighbours it is easily
confused with: the [ITU](/reference/itu/) governs spectrum and telecommunications, the US-based
[NIST](/reference/nist/) maintains measurement standards, and the [IEEE](/reference/ieee/) is a
US-based professional body whose engineering standards (such as the 802 networking series) are
distinct from the IEC's, while in Europe [ETSI](/reference/etsi/) handles telecommunications
standards.

## Relevance to SDR

The IEC rarely defines an over-the-air waveform directly — that is the province of the ITU,
ETSI, and industry bodies — but its standards shape the gear and the data around a receiver.
The IEC 61162 / NMEA interface is the format in which a decoded **[AIS](/reference/ais/)** or
GNSS stream is passed from a receiver into charting and logging software, so an SDR user who
feeds AIS sentences into a plotter is handling IEC-standardised data. IEC safety, connector,
and electromagnetic-compatibility standards also govern the physical hardware — power supplies,
cabling, and shielding — that determines how clean a receive chain is.

Those standards sit alongside rather than inside GopherTrunk's trunking decode path, so
GopherTrunk does not implement IEC standards directly. The reference stands as context for the
wider ecosystem of standards bodies whose work an SDR user encounters, and it clarifies where
the IEC fits relative to the ITU, IEEE, and ETSI.

## Sources

[^home]: [International Electrotechnical Commission](https://www.iec.ch/) — the IEC's official site, for its standards catalogue, JTC 1 role, and conformity schemes.
[^wiki]: [International Electrotechnical Commission](https://en.wikipedia.org/wiki/International_Electrotechnical_Commission) — Wikipedia, for the IEC's history, structure, and standards role.
