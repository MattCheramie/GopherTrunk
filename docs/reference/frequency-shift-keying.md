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

## How it works

Each frequency offset (deviation) represents a symbol. **4FSK** carries 2 bits per
symbol and is the workhorse of digital land-mobile voice — [C4FM](/reference/c4fm/) in
[P25](/reference/project-25/), and the modulation of [DMR](/reference/dmr/) and
[NXDN](/reference/nxdn/) — because it tolerates efficient, non-linear amplifiers.

## Relevance to SDR

An FSK demodulator tracks instantaneous frequency; the resulting symbol levels appear
on the [symbol scope](/reference/eye-diagram/) and as clusters on a
[constellation](/reference/constellation-diagram/).
