---
slug: ffsk
title: FFSK
entry_type: technology
category: modulation
description: FFSK (fast frequency-shift keying) is a coherent audio FSK where tone frequencies are integer multiples of the bit rate; used by MDC1200, DSC, and MPT 1327 signalling.
keywords: FFSK, fast frequency shift keying, MDC1200, DSC, MPT 1327, 1200 bps signalling, coherent FSK, continuous phase, POCSAG
aka: [FFSK]
autolink: true
infobox:
  - { label: Type, value: Digital modulation (coherent audio FSK) }
  - { label: Property, value: Tones are integer multiples of bit rate }
  - { label: Used by, value: MDC1200, DSC, MPT 1327 }
see_also: [afsk, frequency-shift-keying, mdc1200, dsc, mpt-1327, minimum-shift-keying, continuous-phase-modulation, packet-radio]
related_lessons:
  - { title: "Other signals you'll meet", url: /learn/rf-sdr/other-signals/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency-shift_keying
  - https://en.wikipedia.org/wiki/Minimum-shift_keying
---

**FFSK** (fast frequency-shift keying) is a coherent form of audio
[FSK](/reference/frequency-shift-keying/) in which the mark and space tones are exact
integer multiples of the bit rate, so each bit contains a whole number of cycles.[^wiki]
This makes detection clean, bit timing self-evident, and the transmitted spectrum
compact — the properties that made FFSK the standard for short control bursts on analog
land-mobile channels.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A fast two-tone FSK waveform with phase-continuous transitions between mark and space tones." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q10 -26 20 0 t20 0 t20 0 t20 0
            M120 60 q15 -26 30 0 t30 0 t30 0
            M240 60 q10 -26 20 0 t20 0 t20 0 t20 0 t20 0
            M360 60 q15 -26 30 0 t30 0 t20 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="20" y="100" font-size="9" fill="currentColor">phase-continuous mark/space tones (e.g. 1200/1800 Hz)</text>
</svg>
<figcaption>FFSK (fast FSK) is phase-continuous tone signalling used by MDC-1200 and DSC.</figcaption>
</figure>

## How it works

FFSK's defining trick is choosing the two tones so that each represents a whole number of
cycles per bit period. A common form keys 1200 Hz for a mark and 1800 Hz for a space at
1200 baud: at 1200 bps a bit lasts 1/1200 s, in which 1200 Hz completes exactly one cycle
and 1800 Hz exactly 1.5 cycles. Because both tones cross zero at the bit boundary, the
transmitter can switch between them with **no phase discontinuity**, giving a continuous-
phase, constant-envelope waveform — closely related to
[minimum-shift keying](/reference/minimum-shift-keying/) — that a saturated FM transmitter
passes cleanly. The integer-cycle property also means a receiver can count zero crossings
or correlate against the two tones with the bit clock aligned automatically, so bit and
symbol timing fall out of the signal itself rather than needing a separate recovery loop.

Those traits — clean detection, easy timing, narrow spectrum, robustness to the
non-linearities of an FM channel — suit **short data bursts** far more than sustained
throughput, which is why FFSK shows up in signalling systems that piggyback brief control
messages onto otherwise-analog voice channels.

## Variants

FFSK is closely tied to the same continuous-phase, integer-index ideas behind MSK; the
difference from generic [AFSK](/reference/afsk/) is precisely the coherent, integer-cycle
tone choice, which AFSK (e.g. Bell 202's 1200/2200 Hz at 1200 baud, where 2200 Hz is not
an integer number of cycles per bit) does not enforce. Different systems pick different
tone pairs and bit rates, but the coherent principle is shared across
[MDC1200](/reference/mdc1200/), maritime [DSC](/reference/dsc/), and
[MPT 1327](/reference/mpt-1327/).

## In practice

FFSK carries [MDC1200](/reference/mdc1200/) unit-ID and emergency signalling on analog
two-way channels, [DSC](/reference/dsc/) digital selective calling on marine VHF/HF (the
distress-alert layer of the GMDSS), and [MPT 1327](/reference/mpt-1327/) trunking control
signalling — typically at 1200 bps. In each the FFSK burst is short: an MDC PTT-ID is a
fraction of a second at the start or end of a transmission, and an MPT 1327 control channel
interleaves signalling frames with call setup.

## Relevance to SDR

GopherTrunk detects FFSK bursts on analog channels to decode signalling such as PTT IDs
and trunking control data: it FM-demodulates the channel, then correlates the audio against
the mark/space tones with bit timing derived from the integer-cycle structure. Because the
bursts are short and self-clocking, reliable capture depends mostly on getting clean FM
audio in the first place.

## Sources

[^wiki]: [Frequency-shift keying](https://en.wikipedia.org/wiki/Frequency-shift_keying) — Wikipedia, for coherent/fast FSK and continuous-phase tone signalling.
[^msk]: [Minimum-shift keying](https://en.wikipedia.org/wiki/Minimum-shift_keying) — Wikipedia, for the integer-index, phase-continuous basis FFSK shares with MSK.
