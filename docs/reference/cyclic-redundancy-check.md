---
slug: cyclic-redundancy-check
title: Cyclic redundancy check (CRC)
entry_type: algorithm
category: algorithms
description: A cyclic redundancy check is an error-detection code that appends a checksum computed by polynomial division; it detects corrupted frames in AIS, ADS-B, AX.25, and more.
keywords: CRC, cyclic redundancy check, checksum, error detection, FCS, polynomial, CRC-24
aka: [cyclic redundancy check, CRC]
autolink: true
infobox:
  - { label: Type, value: Error-detection code }
  - { label: Method, value: Polynomial division checksum }
  - { label: Used by, value: AIS, ADS-B (CRC-24), AX.25, M17 }
see_also: [forward-error-correction, ads-b, ais, ax25]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Cyclic redundancy check (Wikipedia)", url: https://en.wikipedia.org/wiki/Cyclic_redundancy_check }
---

A **cyclic redundancy check** (**CRC**) is an error-*detection* code that appends a
checksum computed by polynomial division over the data. A mismatch on receive flags a
corrupted frame.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A data frame with a CRC value appended, and a check at the receiver that flags pass or fail." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="40" width="220" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="140" y="59" text-anchor="middle" font-size="9" fill="currentColor">data frame</text>
  <rect x="250" y="40" width="80" height="30" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.2"/><text x="290" y="59" text-anchor="middle" font-size="9" fill="currentColor">CRC</text>
  <line x1="334" y1="55" x2="370" y2="55" stroke="currentColor" marker-end="url(#crcar)"/>
  <text x="410" y="50" font-size="9" fill="currentColor">recompute</text><text x="410" y="63" font-size="9" fill="currentColor">→ pass/fail</text>
  <text x="180" y="92" text-anchor="middle" font-size="9" fill="currentColor">detects (not corrects) errors</text>
  <defs><marker id="crcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A CRC appends a checksum so the receiver can detect whether a frame arrived intact.</figcaption>
</figure>

## How it works

Unlike [forward error correction](/reference/forward-error-correction/), a CRC detects
but does not correct errors. Frames failing the CRC are discarded.
[ADS-B](/reference/ads-b/) uses CRC-24; [AIS](/reference/ais/) and
[AX.25](/reference/ax25/) use CRC-16 (FCS).

## Relevance to SDR

CRC validation ensures GopherTrunk only reports data frames that arrived intact.
