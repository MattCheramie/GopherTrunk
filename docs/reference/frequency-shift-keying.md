---
slug: frequency-shift-keying
title: Frequency-shift keying (FSK)
entry_type: technology
category: modulation
description: Frequency-shift keying (FSK) is digital modulation that switches a carrier between discrete frequencies; four-level FSK (4FSK) underlies P25 C4FM, DMR, and NXDN.
keywords: FSK, frequency shift keying, 2FSK, 4FSK, C4FM, digital modulation, mark space
aka: [frequency-shift keying, FSK, 4FSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation }
  - { label: Varies, value: Carrier frequency (discrete) }
  - { label: Used by, value: P25, DMR, NXDN, paging }
see_also: [phase-shift-keying, quadrature-amplitude-modulation, c4fm, gmsk, afsk, ffsk, symbol-rate]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/digital-modulation/ }
external:
  - { title: "Frequency-shift keying (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency-shift_keying }
---

**Frequency-shift keying** (**FSK**) is digital [modulation](/reference/modulation/)
that switches a [carrier](/reference/carrier-wave/) between a fixed set of frequencies,
one per [symbol](/reference/symbol-rate/). Two frequencies gives 2FSK; four gives 4FSK.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A bit stream above, and below it a carrier that switches between a low frequency for zeros and a high frequency for ones." xmlns="http://www.w3.org/2000/svg">
  <g font-size="11" fill="currentColor" font-family="monospace"><text x="40" y="24">1</text><text x="140" y="24">0</text><text x="240" y="24">1</text><text x="340" y="24">0</text></g>
  <path d="M30 80 q6 -26 12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0
            M130 80 q12 -26 24 0 t24 0 t24 0 t24 0
            M230 80 q6 -26 12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0 t12 0
            M330 80 q12 -26 24 0 t24 0 t24 0 t24 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="30" y="115" font-size="9" fill="currentColor">each bit picks a frequency (here 2FSK; 4FSK uses four)</text>
</svg>
<figcaption>FSK switches the carrier between set frequencies; four-level 4FSK underlies P25 C4FM and DMR.</figcaption>
</figure>

## How it works

Each frequency offset (deviation) represents a symbol. **4FSK** carries 2 bits per
symbol and is the workhorse of digital land-mobile voice — [C4FM](/reference/c4fm/) in
[P25](/reference/project-25/), and the modulation of [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) — because it tolerates efficient, non-linear amplifiers.

## Relevance to SDR

An FSK demodulator tracks instantaneous frequency; the resulting symbol levels appear
on the [symbol scope](/reference/eye-diagram/) and as clusters on a
[constellation](/reference/constellation-diagram/).
