---
slug: amplitude-modulation
title: Amplitude modulation (AM)
entry_type: technology
category: modulation
description: Amplitude modulation (AM) encodes information by varying a carrier's amplitude; it is simple, prone to noise, and still used for shortwave broadcast and aviation voice.
keywords: amplitude modulation, AM, carrier, sidebands, aviation airband, shortwave, envelope detector, modulation index, DSB
aka: [amplitude modulation]
autolink: true
infobox:
  - { label: Type, value: Analog modulation }
  - { label: Varies, value: Carrier amplitude }
  - { label: Used for, value: Shortwave broadcast, aviation airband }
see_also: [modulation, frequency-modulation, single-sideband, carrier-wave, double-sideband, vestigial-sideband, modulation-index, amplitude-shift-keying, on-off-keying]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/rf-sdr/analog-modulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Amplitude_modulation
  - https://en.wikipedia.org/wiki/Sideband
---

**Amplitude modulation** (**AM**) encodes information by varying the
[amplitude](/reference/amplitude/) of a [carrier wave](/reference/carrier-wave/) while
its [frequency](/reference/frequency/) and [phase](/reference/phase/) stay fixed.[^wiki]
It is the oldest and simplest [modulation](/reference/modulation/) scheme, dating to the
first voice broadcasts around 1900, and it remains in daily use for shortwave and
aviation because the receiver can be trivially cheap.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A carrier whose amplitude envelope follows a slower message waveform." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 65 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0 q5 -12 10 0 q5 -8 10 0 q5 -12 10 0 q5 -22 10 0 q5 -28 10 0 q5 -22 10 0" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <path d="M20 65 C 80 18, 140 18, 200 65 S 320 112, 380 65 S 440 30, 440 65" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.5" stroke-dasharray="4 3"/>
  <text x="20" y="118" font-size="10" fill="currentColor">the envelope carries the message</text>
</svg>
<figcaption>AM varies the carrier's amplitude in step with the message; the dashed envelope is the audio.</figcaption>
</figure>

## How it works

A pure carrier at frequency *f*<sub>c</sub> carries no information — it is a single
spectral line. AM multiplies that carrier by (1 + m·s(t)), where s(t) is the message
scaled to ±1 and *m* is the [modulation index](/reference/modulation-index/). Louder
audio produces larger swings in carrier height. In the frequency domain this
multiplication shifts a copy of the message spectrum up to sit on either side of the
carrier, producing two mirror-image **sidebands**:[^sb] the upper sideband runs from
*f*<sub>c</sub> to *f*<sub>c</sub> + *f*<sub>max</sub> and the lower sideband is its
reflection below the carrier. A 3 kHz audio bandwidth therefore occupies about 6 kHz of
RF. Both sidebands carry the same information, so classic AM is inherently a
[double-sideband](/reference/double-sideband/) scheme that wastes half its bandwidth and,
because the carrier itself carries no message, most of its transmitted power.

The great virtue is demodulation: because the message rides directly on the envelope, a
receiver only needs an *envelope detector* — a diode, a capacitor, and a resistor — to
recover the audio. No frequency reference or phase lock is required, which is why the
cheapest possible radio can hear AM. The great weakness is that noise, interference, and
fading are themselves amplitude disturbances, so they add directly to the recovered
audio; AM has no inherent noise immunity the way [FM](/reference/frequency-modulation/)
does. Keeping the modulation index at or below 1 (100%) avoids *overmodulation*, where
the envelope pinches to zero and the recovered audio distorts badly and splatters energy
into adjacent channels.

## Variants

Because full-carrier double-sideband AM is spendthrift, several relatives trim it down.
[Single-sideband](/reference/single-sideband/) (SSB) suppresses the carrier and one
sideband, cutting bandwidth in half and putting all power on the information — the choice
for long-distance HF voice. Double-sideband suppressed-carrier (DSB-SC) removes only the
carrier. [Vestigial sideband](/reference/vestigial-sideband/) (VSB) transmits one full
sideband plus a filtered stub of the other, a compromise used by analog television video.
The digital cousins are [amplitude-shift keying](/reference/amplitude-shift-keying/),
which switches between discrete amplitude levels, and its two-level special case
[on-off keying](/reference/on-off-keying/), the modulation of Morse [CW](/reference/continuous-wave/)
and many low-cost remote controls.

## Relevance to SDR

AM survives where simplicity or a useful property matters. Shortwave broadcast still uses
full-carrier AM so any household set can tune it. Aviation VHF airband (108–137 MHz)
deliberately keeps AM because when two aircraft transmit at once their carriers *beat*
together, producing an audible heterodyne that warns the controller of a collision — a
safety property FM's [capture effect](/reference/frequency-modulation/) would hide. An SDR
demodulates AM by computing the magnitude of the complex [IQ](/reference/iq-data/) baseband
(a synchronous or envelope detector), which needs no [PLL](/reference/phase-locked-loop/)
and is one of the first modes any SDR tutorial implements. GopherTrunk is a digital
trunking decoder and does not target broadcast or airband AM voice, but the same envelope
math appears wherever it must measure signal amplitude.

## Sources

[^wiki]: [Amplitude modulation](https://en.wikipedia.org/wiki/Amplitude_modulation) — Wikipedia, for the definition, modulation index, sidebands, and uses of AM.
[^sb]: [Sideband](https://en.wikipedia.org/wiki/Sideband) — Wikipedia, for how modulation creates upper and lower sidebands and the resulting occupied bandwidth.
