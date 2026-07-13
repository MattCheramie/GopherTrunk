---
slug: eye-diagram
title: Eye diagram
entry_type: term
category: modulation
description: An eye diagram overlays many symbol periods of a demodulated signal against time; a wide-open "eye" indicates good timing and noise margin for decoding.
keywords: eye diagram, eye pattern, symbol timing, noise margin, jitter, 4FSK, intersymbol interference, eye opening
aka: [eye diagram, eye pattern]
autolink: true
infobox:
  - { label: Type, value: Signal-quality display }
  - { label: Axes, value: Amplitude vs. time (overlaid) }
  - { label: Open eye, value: Good timing/noise margin }
see_also: [constellation-diagram, clock-recovery, symbol-rate, intersymbol-interference, c4fm, pulse-shaping]
related_lessons:
  - { title: "Digital modulation & constellations", url: /learn/rf-sdr/digital-modulation/ }
  - { title: "Tuning for a clean lock", url: /learn/rf-sdr/tuning-with-scopes/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Eye_pattern
  - https://en.wikipedia.org/wiki/Intersymbol_interference
---

An **eye diagram** overlays many short segments of a demodulated signal, each one
symbol period long (or two), so they stack on the same time axis into the characteristic
"eye" shapes between the symbol levels.[^wiki] It is the time-domain companion to the
[constellation diagram](/reference/constellation-diagram/): where the constellation shows
symbol quality on the IQ plane, the eye shows how cleanly the signal transitions between
levels and how much margin the decoder has at the sampling instant.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 150" role="img" aria-label="An eye diagram with an open eye shape and a dashed vertical line marking the ideal sampling instant at its widest point." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2" stroke-opacity="0.85">
    <path d="M40 30 C120 30 120 120 200 120 C280 120 280 30 360 30"/>
    <path d="M40 120 C120 120 120 30 200 30 C280 30 280 120 360 120"/>
    <path d="M40 30 C120 30 120 120 200 120"/><path d="M200 30 C280 30 280 120 360 120"/>
  </g>
  <line x1="200" y1="20" x2="200" y2="130" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="200" y="146" text-anchor="middle" font-size="10" fill="currentColor">sample here (eye widest)</text>
</svg>
<figcaption>An eye diagram overlays symbol periods; a wide-open eye means good timing and noise margin.</figcaption>
</figure>

## How it works

Imagine slicing the demodulated waveform into pieces exactly one symbol long and laying them all
on top of one another, triggered on the symbol clock — like a repeatedly retriggered
oscilloscope. Wherever the signal *could* be at that phase, a trace passes; the region no trace
enters is the open "eye." The decoder samples once per symbol, ideally at the instant the eye is
widest, and slices the amplitude to the nearest level. Reading the eye:

- **Vertical opening** is the amplitude margin — the gap between the highest a low symbol reaches
  and the lowest a high symbol reaches. Noise eats into it from both sides; when it closes, low
  and high symbols become indistinguishable and errors appear.
- **Horizontal opening** is the timing margin — the span of sampling instants that still fall
  inside the eye. Timing [jitter](/reference/clock-recovery/) and
  [intersymbol interference](/reference/intersymbol-interference/) narrow it. A wide horizontal
  opening means the [clock-recovery](/reference/clock-recovery/) loop can be slightly off and
  still sample correctly.
- **Thickness of the traces** reflects noise and residual ISI; a fuzzy crossing point at the eye
  edges signals timing jitter.

The best sampling phase is the vertical line through the widest part of the eye, which is exactly
what timing recovery tries to find and hold.

## Variants

An M-level signal shows M−1 stacked eyes: a two-level scheme has a single eye, while a four-level
[C4FM](/reference/c4fm/) or [4FSK](/reference/frequency-shift-keying/) signal shows three eyes,
one between each adjacent pair of the four levels. Eye diagrams are most natural for real
(one-dimensional) baseband signals — amplitude or frequency modulations — and are drawn
separately for the I and Q rails of a complex signal. A well-designed
[pulse-shaping](/reference/pulse-shaping/) filter, such as a
[root-raised-cosine](/reference/root-raised-cosine-filter/), is built specifically to force the
ISI to zero at the sampling instant, which is what keeps the eye open in a bandlimited channel.

## In practice

Because it exposes timing and amplitude margin at once, the eye diagram is the standard bench and
software test for whether a link has usable margin or is riding the edge of failure. A signal
that decodes but shows a nearly closed eye is fragile — a little more noise or multipath will tip
it into errors — whereas a wide, clean eye has headroom. Watching the eye while adjusting gain,
tuning, or filter settings gives immediate feedback that a constellation alone (which hides
timing information) does not.

## Relevance to SDR

For frequency-modulated trunking signals the eye diagram is often more diagnostic than the
constellation, since the information lives in instantaneous frequency and forms clean stacked
eyes after FM demodulation. GopherTrunk's eye-diagram panel shows the three-eye pattern of the
4800-baud C4FM family, letting an operator see timing and noise margin at a glance and confirm
that symbol timing is locked to the widest opening before trusting a marginal decode.

## Sources

[^wiki]: [Eye pattern](https://en.wikipedia.org/wiki/Eye_pattern) — Wikipedia, for the overlaid-symbol-period display and what an open eye indicates.
[^isi]: [Intersymbol interference](https://en.wikipedia.org/wiki/Intersymbol_interference) — Wikipedia, for how ISI narrows the eye and why pulse shaping keeps it open.
