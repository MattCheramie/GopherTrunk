---
slug: m17-project
title: M17 Project
entry_type: organization
category: organizations
description: The M17 Project is an open-source community developing the royalty-free M17 amateur digital-voice protocol and associated open hardware and software.
keywords: M17 Project, open source, amateur radio, Codec 2, royalty-free, patent-free, digital voice, 4-FSK
aka: [M17 Project]
autolink: true
infobox:
  - { label: Type, value: Open-source community project }
  - { label: Develops, value: M17 protocol + open tools }
  - { label: Ethos, value: Royalty-free, patent-free }
see_also: [m17, codec2, vocoder, arrl, iaru, four-fsk]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
cite_urls:
  - https://m17project.org/
  - https://en.wikipedia.org/wiki/M17_(amateur_radio)
---

The **M17 Project** is an open-source community that develops the royalty-free
**[M17](/reference/m17/)** amateur digital-voice protocol along with open hardware and
software implementations.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="The open-source M17 Project maintains the royalty-free M17 protocol built on the open Codec 2 vocoder." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="55">M17 Project</text><text x="70" y="67" font-size="7.5">volunteers</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="55">open spec +</text><text x="225" y="67" font-size="7.5">Codec 2 vocoder</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">M17 (open)</text><text x="385" y="67" font-size="7.5">4-FSK, 9600 bps</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_m17-project)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_m17-project)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">no patents, no royalties → anyone can build compatible gear</text>
  </g>
  <defs><marker id="rel_m17-project" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The M17 Project is the open-source community that maintains the royalty-free M17 protocol, built on the open Codec 2 vocoder.</figcaption>
</figure>

## Overview

Founded around 2019, the M17 Project is a volunteer, community-driven effort — closer in spirit
to an open-source software project than to a formal standards body like [ETSI](/reference/etsi/)
or the [TIA](/reference/tia/). Its members are amateur-radio operators and engineers who set out
to build a modern digital-voice and data protocol that is completely free of licensing
constraints, in reaction to a land-mobile world where the [vocoder](/reference/vocoder/) is
almost always the patented [AMBE](/reference/ambe/) family from [DVSI](/reference/dvsi/). By
choosing the open, patent-free **[Codec 2](/reference/codec2/)** vocoder instead, M17 can be
implemented, redistributed, and built into hardware by anyone without paying royalties — which
is not possible for AMBE-based systems like [DMR](/reference/dmr/) or [P25](/reference/project-25/).

The project develops far more than a paper specification. The **[M17](/reference/m17/)** air
interface is a 9600 bit/s [4-FSK](/reference/four-fsk/) waveform in a 9 kHz channel carrying
Codec 2 voice with convolutional [FEC](/reference/forward-error-correction/), plus optional
stream encryption and a reflector system for internet linking. The community also produces open
reference firmware, the "Module17" and OpenRTX-based radio hardware, gateway software, and
gateways to and from other digital modes. Everything — the protocol document, the source code,
and the hardware designs — is published under open licences. Because M17 lives in the amateur
bands, it operates within the allocations that national regulators (the [FCC](/reference/fcc/) in
the US, [Ofcom](/reference/ofcom/) in the UK) grant the amateur service under the international
[ITU](/reference/itu/) plan, and amateur-service advocacy for those allocations comes from
bodies such as the [ARRL](/reference/arrl/) and the [IARU](/reference/iaru/).

## Relevance to SDR

The M17 Project's radical openness is exactly what makes it interesting from an SDR and
decoding standpoint: because both the air interface *and* the vocoder are open, a fully free,
end-to-end M17 receiver is possible with no patent shadow and no need for an external AMBE
dongle. That is a genuine rarity in digital land-mobile radio, where the vocoder patent almost
always forces a compromise. For anyone learning how digital voice works, M17 is also the most
transparent case study available — you can read every stage from the 4-FSK symbols to the
decoded Codec 2 audio in open source.

GopherTrunk includes M17 link-layer support, decoding the frame structure and stream metadata,
and because the stack is open there is no licensing barrier to doing so. This makes M17 a clean
example of the honest end of the spectrum: an open community protocol that a pure-Go decoder can
handle completely, in contrast to proprietary systems where at least the vocoder is closed. See
the [digital protocol landscape](/learn/rf-sdr/protocol-landscape/) lesson for how M17 compares
with the commercial digital standards.

## Sources

[^home]: [M17 Project](https://m17project.org/) — the project's official site, home of the royalty-free M17 protocol, open hardware, and reference software.
[^wiki]: [M17 (amateur radio)](https://en.wikipedia.org/wiki/M17_(amateur_radio)) — Wikipedia, for the M17 protocol, its use of Codec 2, and the project's open ethos.
