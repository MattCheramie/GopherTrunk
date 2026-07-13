---
slug: tia
title: Telecommunications Industry Association (TIA)
entry_type: organization
category: organizations
description: The TIA is a US standards organization that, with APCO, develops the Project 25 (P25) suite of public-safety digital radio standards.
keywords: TIA, Telecommunications Industry Association, P25, Project 25, TIA-102, standards, public safety, ANSI
aka: [TIA, Telecommunications Industry Association]
autolink: true
infobox:
  - { label: Type, value: Standards organization }
  - { label: Region, value: United States }
  - { label: Standards, value: P25 (TIA-102, with APCO) }
see_also: [project-25, apco-international, etsi, p25-phase-1, p25-phase-2, tsbk]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
cite_urls:
  - https://www.tiaonline.org/
  - https://en.wikipedia.org/wiki/Telecommunications_Industry_Association
---

The **Telecommunications Industry Association** (**TIA**) is a US standards organization
that, together with [APCO International](/reference/apco-international/), develops the
**[Project 25](/reference/project-25/)** suite of public-safety digital radio standards.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="TIA publishes the TIA-102 standards suite that defines the P25 air interface, vocoder, and inter-system interfaces." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="61">TIA</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="61">TIA-102 suite</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">P25 Phase 1/2</text><text x="385" y="67">multi-vendor</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_tia)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_tia)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">air interface · C4FM/CQPSK · IMBE/AMBE+2 · inter-RF-subsystem</text>
  </g>
  <defs><marker id="rel_tia" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The TIA publishes the TIA-102 standards suite that defines Project 25, from the air interface to inter-system interfaces.</figcaption>
</figure>

## Overview

The TIA is an ANSI-accredited standards developer representing the US information and
communications technology industry, formed in 1988 from the merger of earlier trade bodies and
long associated with the Electronic Industries Alliance (EIA) — which is why older documents
carry "TIA/EIA" designations. Its engineering committees publish standards across cabling,
mobile devices, satellite, and land-mobile radio. In the scanner world its defining work is
the **TIA-102** document series, which specifies [Project 25](/reference/project-25/) end to
end. That suite covers far more than the over-the-air waveform: it defines the Common Air
Interface (the [P25 CAI](/reference/p25-cai/)), the [IMBE](/reference/imbe/) and
[AMBE+2](/reference/ambe-plus-2/) [vocoders](/reference/vocoder/), trunking control signalling,
encryption, key management ([OTAR](/reference/otar/)), and the inter-RF-subsystem and console
interfaces that let equipment from different manufacturers work together on one network.

TIA-102 splits P25 into phases. **[P25 Phase 1](/reference/p25-phase-1/)** uses
[C4FM](/reference/c4fm/)/[CQPSK](/reference/cqpsk/) [FDMA](/reference/fdma/) in a 12.5 kHz
channel at 9600 bit/s. **[P25 Phase 2](/reference/p25-phase-2/)** adds two-slot
[TDMA](/reference/tdma/) for 6.25 kHz-equivalent efficiency using the AMBE+2 vocoder. This
multi-vendor, open-standard approach is exactly what public-safety agencies wanted: the
requirements came from user representatives at [APCO International](/reference/apco-international/),
the engineering was standardised by the TIA, and the result — sometimes called "APCO-25" — can
be built by Motorola, L3Harris, Tait, and others while remaining interoperable. The
Compliance Assessment Program tests real equipment against the TIA-102 documents. The TIA works
within US spectrum rules set by the [FCC](/reference/fcc/), whose narrowbanding push made the
efficient P25 waveforms attractive; internationally, the analogous professional standards come
from [ETSI](/reference/etsi/) ([DMR](/reference/dmr/), [TETRA](/reference/tetra/)).

## Relevance to SDR

The TIA's TIA-102 standards underpin the single most commonly monitored family of digital
systems in North America. Because the P25 air interface is a *published* standard, decoder
authors can implement its frame structure, [Golay](/reference/golay-code/) and
[Reed–Solomon](/reference/reed-solomon-code/) [FEC](/reference/forward-error-correction/),
[network access code](/reference/network-access-code/), and trunking control blocks
([TSBK](/reference/tsbk/)) directly from the specification rather than by guesswork. That
openness is why robust open-source P25 receivers exist at all. The one deliberately closed
piece is the vocoder — IMBE and AMBE+2 are licensed from [DVSI](/reference/dvsi/) — which is a
separate concern from the freely readable air interface.

GopherTrunk decodes P25 Phase 1 and Phase 2 control and voice channels built to TIA-102, so in
a very direct sense the TIA's documents define the bitstreams GopherTrunk parses. What
GopherTrunk cannot do is recover traffic protected by keyed [encryption](/reference/encryption/)
(P25 supports AES and DES options in the TIA-102 security services) — it is a receiver for
clear and scrambled traffic, not a key-recovery tool. For how P25 sits alongside the
[ETSI](/reference/etsi/) digital family, see the
[digital protocol landscape](/learn/rf-sdr/protocol-landscape/) lesson.

## Sources

[^home]: [Telecommunications Industry Association](https://www.tiaonline.org/) — the TIA's official site, publisher of the TIA-102 (P25) standards suite and its Compliance Assessment Program.
[^wiki]: [Telecommunications Industry Association](https://en.wikipedia.org/wiki/Telecommunications_Industry_Association) — Wikipedia, for the TIA's history, structure, and its P25 work with APCO.
