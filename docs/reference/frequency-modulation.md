---
slug: frequency-modulation
title: Frequency modulation (FM)
entry_type: technology
category: modulation
description: Frequency modulation (FM) encodes information by varying a carrier's frequency; it resists amplitude noise and is used for broadcast and analog two-way voice.
keywords: frequency modulation, FM, deviation, capture effect, narrowband FM, broadcast, Carson bandwidth, pre-emphasis, modulation index
aka: [frequency modulation]
autolink: true
infobox:
  - { label: Type, value: Analog modulation }
  - { label: Varies, value: Carrier frequency (deviation) }
  - { label: Used for, value: FM broadcast, analog two-way voice }
see_also: [modulation, amplitude-modulation, single-sideband, frequency-shift-keying, fm-deviation, modulation-index, pre-emphasis-de-emphasis, continuous-phase-modulation]
related_lessons:
  - { title: "Analog modulation — AM, FM, SSB", url: /learn/rf-sdr/analog-modulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_modulation
  - https://en.wikipedia.org/wiki/Carson_bandwidth_rule
---

**Frequency modulation** (**FM**) encodes information by varying a
[carrier](/reference/carrier-wave/)'s [frequency](/reference/frequency/) while its
[amplitude](/reference/amplitude/) stays constant.[^wiki] The instantaneous frequency
swings above and below the resting carrier in proportion to the message, and the peak
swing is the [deviation](/reference/fm-deviation/). Edwin Armstrong demonstrated in 1933
that this trade of bandwidth for noise immunity gives dramatically cleaner audio than
[AM](/reference/amplitude-modulation/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A carrier whose cycle spacing tightens and loosens with the message, at constant amplitude." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 60 q4 -28 8 0 q4 -28 8 0 q6 -28 12 0 q7 -28 14 0 q8 -28 16 0 q7 -28 14 0 q6 -28 12 0 q4 -28 8 0 q4 -28 8 0 q4 -28 8 0 q6 -28 12 0 q7 -28 14 0 q8 -28 16 0 q7 -28 14 0 q6 -28 12 0 q4 -28 8 0 q4 -28 8 0 q4 -28 8 0 q6 -28 12 0 q7 -28 14 0 q8 -28 16 0" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="20" y="105" font-size="10" fill="currentColor">constant amplitude — information is in the spacing (frequency)</text>
</svg>
<figcaption>FM varies the carrier's frequency while amplitude stays constant, which is why it shrugs off amplitude noise.</figcaption>
</figure>

## How it works

Where AM writes the message onto the envelope, FM writes it onto the *instantaneous
frequency*: f(t) = f<sub>c</sub> + Δf·s(t), with Δf the peak deviation. Because the
information lives in frequency, not amplitude, an FM receiver can pass the signal through
a *limiter* that strips away any amplitude variation — noise, fading, ignition
sparks — before recovering the audio, and none of that amplitude noise survives. This is
the source of FM's headline advantage: above a threshold [SNR](/reference/signal-to-noise-ratio/),
each extra decibel of received signal buys more than a decibel of audio quality, and wide
deviation trades more RF bandwidth for still-quieter audio.

The bandwidth an FM signal occupies is not just twice the audio; it depends on the
*modulation index* β = Δf / f<sub>max</sub>, the ratio of peak deviation to the highest
message frequency. Carson's rule estimates the occupied bandwidth as roughly
2·(Δf + f<sub>max</sub>).[^carson] Broadcast FM in the US uses ±75 kHz deviation with
15 kHz audio, so β ≈ 5 and each station spans about 180–200 kHz — a *wideband* signal.
Two-way land-mobile radio instead uses ±2.5 to ±5 kHz *narrowband* FM to fit many
channels into a crowded band. Both add [pre-emphasis](/reference/pre-emphasis-de-emphasis/)
that boosts treble before transmission and cuts it after, because FM noise rises with
audio frequency.

FM also shows a **capture effect**: when two signals share a channel, the stronger one
by even a few decibels seizes the limiter and the weaker one vanishes, rather than the
two mixing audibly as they would in AM. This is why FM two-way traffic sounds clean but
distant AM airband stations beat together.

## Variants

FM and [phase modulation](/reference/phase-shift-keying/) (PM) are close relatives —
FM is PM applied to the integral of the message — and both belong to the broader family
of angle modulation. The digital descendant of FM is
[frequency-shift keying](/reference/frequency-shift-keying/) (FSK), which switches the
carrier among a discrete set of frequencies instead of sweeping it continuously;
its constant-envelope, phase-continuous refinements
([continuous-phase modulation](/reference/continuous-phase-modulation/),
[GMSK](/reference/gmsk/), [C4FM](/reference/c4fm/)) underlie most digital land-mobile
radio.

## Relevance to SDR

FM broadcast (wide deviation) and narrowband FM two-way voice are everywhere, and FM is
usually the first mode an SDR beginner demods. An SDR recovers FM by differentiating the
phase of the complex [IQ](/reference/iq-data/) baseband — the derivative of phase *is*
instantaneous frequency — typically as an `atan2` discriminator or the algebraically
equivalent `(I·Q' − Q·I')/(I²+Q²)`. Analog narrowband FM voice is the direct analog cousin
of the digital [4FSK](/reference/four-fsk/) voice modes GopherTrunk decodes: P25 Phase 1
[C4FM](/reference/c4fm/), [DMR](/reference/dmr/), and [NXDN](/reference/nxdn/) are all, at
the physical layer, carefully shaped forms of FM, so a good FM discriminator is the front
end of a good digital-voice symbol recovery chain.

## Sources

[^wiki]: [Frequency modulation](https://en.wikipedia.org/wiki/Frequency_modulation) — Wikipedia, for the definition, deviation, modulation index, and the capture effect.
[^carson]: [Carson bandwidth rule](https://en.wikipedia.org/wiki/Carson_bandwidth_rule) — Wikipedia, for estimating the occupied bandwidth of an FM signal from its deviation and message bandwidth.
