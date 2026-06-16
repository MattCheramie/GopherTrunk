---
slug: ffsk
title: FFSK
entry_type: technology
category: modulation
description: FFSK (fast frequency-shift keying) is a coherent audio FSK where tone frequencies are integer multiples of the bit rate; used by MDC1200, DSC, and MPT 1327 signalling.
keywords: FFSK, fast frequency shift keying, MDC1200, DSC, MPT 1327, 1200 bps signalling
aka: [FFSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (coherent audio FSK) }
  - { label: Property, value: Tones are integer multiples of bit rate }
  - { label: Used by, value: MDC1200, DSC, MPT 1327 }
see_also: [afsk, frequency-shift-keying, mdc1200, dsc, mpt-1327]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/other-signals/ }
external:
  - { title: "Frequency-shift keying (Wikipedia)", url: https://en.wikipedia.org/wiki/Frequency-shift_keying }
---

**FFSK** (fast frequency-shift keying) is a coherent form of audio
[FSK](/reference/frequency-shift-keying/) in which the mark and space tones are exact
integer multiples of the bit rate, so each bit contains a whole number of cycles. This
makes detection clean and bit timing easy.

## How it works

The phase-continuous, integer-cycle tones suit short data bursts over analog FM. FFSK
carries [MDC1200](/reference/mdc1200/) unit IDs, [DSC](/reference/dsc/) maritime calls,
and [MPT 1327](/reference/mpt-1327/) control signalling, typically at 1200 bps.

## Relevance to SDR

GopherTrunk detects FFSK bursts on analog channels to decode signalling such as PTT
IDs and trunking control data.
