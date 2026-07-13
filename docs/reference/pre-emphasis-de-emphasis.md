---
slug: pre-emphasis-de-emphasis
title: Pre-emphasis & de-emphasis
entry_type: term
category: modulation
description: "Pre-emphasis boosts high audio frequencies before FM transmission and de-emphasis cuts them back at the receiver, trading flat response for lower high-frequency noise."
keywords: pre-emphasis, de-emphasis, FM audio, 75 microsecond, 50 microsecond, 750 microsecond, high-frequency boost, noise reduction, emphasis time constant
aka: [pre-emphasis, de-emphasis, emphasis]
autolink: true
infobox:
  - { label: Symbol, value: "τ (time constant)" }
  - { label: Unit, value: "Microseconds (75/50/750 µs)" }
  - { label: Relation, value: "Corner f = 1/(2πτ)" }
see_also: [frequency-modulation, broadcast-fm, signal-to-noise-ratio, digital-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Emphasis_(telecommunications)
  - https://en.wikipedia.org/wiki/FM_broadcasting
---

**Pre-emphasis and de-emphasis** are a complementary filter pair used with
[frequency modulation](/reference/frequency-modulation/): the transmitter boosts high audio
frequencies before modulating (pre-emphasis), and the receiver applies an exactly inverse cut
after demodulating (de-emphasis).[^wiki] The two cancel to leave a flat overall response, but
because the receiver's cut also attenuates the noise that FM concentrates at high frequencies,
the net effect is a quieter, cleaner signal.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Two response curves: a rising pre-emphasis curve at the transmitter and a falling de-emphasis curve at the receiver that are mirror images, plus a shaded high-frequency noise region that de-emphasis suppresses." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="130" x2="440" y2="130" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="40" y1="130" x2="40" y2="20" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="34" y="18" text-anchor="end" font-size="8" fill="currentColor">dB</text>
  <text x="440" y="145" text-anchor="end" font-size="8" fill="currentColor">frequency →</text>
  <path d="M60 100 Q 200 100 440 40" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="300" y="70" font-size="8" fill="currentColor">pre-emphasis (boost)</text>
  <path d="M60 100 Q 200 100 440 120" fill="none" stroke="currentColor" stroke-width="1.6" stroke-dasharray="4 3"/>
  <text x="300" y="118" font-size="8" fill="currentColor">de-emphasis (cut)</text>
  <path d="M300 130 L440 130 L440 90 L300 128 Z" fill="currentColor" fill-opacity="0.15"/>
  <text x="360" y="128" font-size="7" fill="currentColor">HF noise removed</text>
</svg>
<figcaption>Pre-emphasis lifts highs before transmission; the receiver's inverse de-emphasis restores flatness and drops high-frequency noise with it.</figcaption>
</figure>

## How it works

An FM demodulator produces output noise whose power rises with the square of frequency — a
"triangular" noise spectrum — so without correction the high end of recovered audio is much
noisier than the low end. The fix is to pre-distort at the source: a simple first-order
high-pass shelving network boosts treble before the audio modulates the carrier, characterized
by a **time constant** τ that sets the corner frequency f = 1/(2πτ). Broadcast FM uses τ = 75 µs
in the Americas and Korea and 50 µs elsewhere; narrowband land-mobile FM commonly uses 750 µs.

At the receiver a matching low-pass network with the same τ cuts the treble back by the identical
amount. On the wanted audio the boost and cut cancel exactly, so the listener hears flat response.
But the receiver's cut is applied *after* the demodulator, so it also attenuates that rising
high-frequency noise — and the parts of the spectrum where FM noise is worst are exactly where
de-emphasis attenuates most. The result is a meaningful improvement in high-frequency
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) for free, at the cost of a little
headroom (loud treble transients deviate the carrier further, which the pre-emphasis stage must
be limited to control).

## Relevance to SDR

Any software FM demodulator that wants to sound right must apply de-emphasis with the correct time
constant — a [digital filter](/reference/digital-filter/) is a one-pole IIR — or the audio sounds
harsh and bright. Getting τ wrong (using 75 µs on a 50 µs signal, say) leaves a mild treble tilt.
For [broadcast FM](/reference/broadcast-fm/) the de-emphasis is applied to the recovered mono/stereo
audio, not to the composite baseband, so a receiver must de-emphasize after stereo decoding. In
land-mobile digital modes the concept mostly falls away: P25 and DMR carry digitized, vocoded audio
rather than analog FM audio, so there is no analog emphasis stage in GopherTrunk's digital decode
path. Emphasis matters to GopherTrunk-adjacent work only when demodulating conventional analog FM
voice, where the standard land-mobile 750 µs de-emphasis should be used.

## Sources

[^wiki]: [Emphasis (telecommunications)](https://en.wikipedia.org/wiki/Emphasis_(telecommunications)) — Wikipedia, for the pre/de-emphasis pairing, time constants, and FM noise-reduction rationale.
