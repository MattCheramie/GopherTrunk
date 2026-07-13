---
slug: nist
title: NIST (National Institute of Standards and Technology)
entry_type: organization
category: organizations
description: "NIST is the US federal metrology and standards agency, setting the AES and FIPS cryptographic standards and maintaining national time and frequency references."
keywords: NIST, National Institute of Standards and Technology, NBS, AES, FIPS, cryptography standards, time and frequency, WWV, metrology
aka: [NIST, National Institute of Standards and Technology, NBS]
autolink: true
infobox:
  - { label: Type, value: US federal standards agency }
  - { label: Founded, value: "1901" }
  - { label: Region, value: United States }
see_also: [advanced-encryption-standard, frequency-stability, data-encryption-standard, frequency-counter, cryptography]
cite_urls:
  - https://www.nist.gov/
  - https://en.wikipedia.org/wiki/National_Institute_of_Standards_and_Technology
---

**NIST** (the **National Institute of Standards and Technology**) is the United States
federal agency responsible for measurement science, standards, and technology, operating
within the Department of Commerce.[^home] For radio and signal work its most visible outputs
are the [AES](/reference/advanced-encryption-standard/) and FIPS cryptographic standards it
selects and publishes, and the national time-and-[frequency](/reference/frequency-stability/)
references it maintains and broadcasts.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 112" role="img" aria-label="NIST publishes cryptographic standards such as AES and maintains the national time and frequency references broadcast on WWV and WWVB." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="nist_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="180" y="42" width="100" height="34" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.2"/><text x="230" y="63">NIST</text>
    <rect x="330" y="8" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="390" y="24">AES / FIPS</text>
    <rect x="330" y="46" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="390" y="62">Time &amp; frequency</text>
    <rect x="330" y="84" width="120" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="390" y="100">WWV / WWVB</text>
    <rect x="10" y="46" width="130" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="75" y="62">Metrology (SI units)</text>
    <g stroke="currentColor" stroke-width="1"><line x1="140" y1="59" x2="178" y2="59" marker-end="url(#nist_ar)"/><line x1="280" y1="52" x2="329" y2="24" marker-end="url(#nist_ar)"/><line x1="280" y1="59" x2="329" y2="58" marker-end="url(#nist_ar)"/><line x1="280" y1="66" x2="329" y2="94" marker-end="url(#nist_ar)"/></g>
  </g>
</svg>
<figcaption>NIST underpins US measurement, publishes cryptographic standards, and keeps the national time and frequency references.</figcaption>
</figure>

## Overview

NIST was founded in 1901 as the National Bureau of Standards (NBS) and renamed NIST in 1988.
Its core mission is metrology — realizing and disseminating the SI units of measurement so
that a second, a meter, a volt, or a hertz means the same thing everywhere in US commerce
and science. It operates major laboratories, maintains reference materials and calibration
services, and its atomic-clock ensemble contributes to the international definition of the
second.

Two strands of NIST's work reach directly into radio and communications. First, in
cryptography, NIST runs the open competitions and processes that select US federal
standards: it standardized the [Data Encryption Standard](/reference/data-encryption-standard/)
in the 1970s, ran the competition that produced the [Advanced Encryption
Standard](/reference/advanced-encryption-standard/) (AES) in 2001, and continues to
standardize hash functions and post-quantum algorithms through its FIPS and Special
Publication series. Second, in time and frequency, NIST operates the atomic clocks behind
UTC(NIST) and broadcasts time signals on the shortwave stations **WWV** and **WWVH** and the
60 kHz longwave station **WWVB**, which disciplines radio-controlled clocks across North
America.

## Relevance to SDR

NIST's work anchors two things SDR practitioners care about: encryption and accurate time.
The [AES](/reference/advanced-encryption-standard/) cipher that NIST standardized is the same
algorithm used to encrypt digital land-mobile voice in systems like P25 and DMR — which is
precisely why encrypted traffic is opaque to a receiver. On the timing side, NIST's
frequency references define [frequency stability](/reference/frequency-stability/) and are a
practical calibration target: WWV and WWVB are classic signals for testing an SDR's tuning,
and a [frequency counter](/reference/frequency-counter/) or reference oscillator traces its
accuracy back to NIST.

GopherTrunk does not implement NIST standards as such, but it lives inside their
consequences. It decodes clear and scrambled traffic; when a system uses AES keyed
encryption — a NIST standard — the payload is unrecoverable by design, and GopherTrunk
honestly reports that rather than attempting to break it. NIST is included here as the US
authority behind the cryptographic and timing standards that repeatedly shape what an SDR
can and cannot do.

## Sources

[^home]: [NIST](https://www.nist.gov/) — the institute's official site, for its metrology, cryptographic standards, and time-and-frequency services.
[^wiki]: [National Institute of Standards and Technology](https://en.wikipedia.org/wiki/National_Institute_of_Standards_and_Technology) — Wikipedia, for the agency's history, role, and standards.
