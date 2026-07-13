---
slug: root-raised-cosine-filter
title: Root-raised-cosine filter (RRC)
entry_type: algorithm
category: filtering-multirate
description: A root-raised-cosine (RRC) filter is a pulse-shaping filter split between transmitter and receiver so the combined link limits bandwidth with zero intersymbol interference.
keywords: root raised cosine, RRC, pulse shaping, intersymbol interference, ISI, matched filter, roll-off factor, Nyquist filter, raised cosine, excess bandwidth
aka: [root-raised-cosine filter, RRC filter, square-root raised cosine, SRRC]
autolink: true
infobox:
  - { label: Type, value: Pulse-shaping filter }
  - { label: Goal, value: Limit bandwidth, zero ISI }
  - { label: Used, value: TX and RX (matched pair) }
see_also: [matched-filter, pulse-shaping, c4fm, symbol-rate, eye-diagram, digital-filter]
related_lessons:
  - { title: "Filtering & decimation", url: /learn/rf-sdr/filtering-decimation/ }
related_reading:
  - { title: "SDR Internals, Part 4: DSP foundations — filters, NCO & AGC", url: /blog/deep-dives/sdr-internals-04-dsp-foundations-filters-nco-agc/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Root-raised-cosine_filter
  - https://en.wikipedia.org/wiki/Raised-cosine_filter
---

A **root-raised-cosine** (**RRC**) filter is a [pulse-shaping](/reference/pulse-shaping/)
[filter](/reference/digital-filter/) applied at **both** ends of a digital radio link — half
the shaping at the transmitter, half at the receiver.[^wiki] Placed at only one end it does
not, by itself, satisfy the zero-intersymbol-interference condition; but because the transmit
RRC and the receive RRC multiply in the frequency domain, the two square-root halves combine
into a full **raised-cosine** response that limits bandwidth while producing **zero
intersymbol interference (ISI)** at the ideal sampling instants.[^rc]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A root-raised-cosine pulse: a tall central lobe with small symmetric side ripples that pass through zero at each neighbouring symbol time, so a symbol does not smear into its neighbours." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="95" x2="440" y2="95" stroke="currentColor" stroke-opacity="0.3"/>
  <path d="M20 95 Q 70 100 110 95 Q 150 86 170 95 Q 200 108 230 40 Q 260 108 290 95 Q 310 86 350 95 Q 390 100 440 95" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g fill="currentColor" fill-opacity="0.55"><circle cx="110" cy="95" r="2.4"/><circle cx="170" cy="95" r="2.4"/><circle cx="230" cy="40" r="2.4"/><circle cx="290" cy="95" r="2.4"/><circle cx="350" cy="95" r="2.4"/></g>
  <g stroke="currentColor" stroke-opacity="0.25" stroke-dasharray="3 3"><line x1="110" y1="40" x2="110" y2="108"/><line x1="170" y1="40" x2="170" y2="108"/><line x1="290" y1="40" x2="290" y2="108"/><line x1="350" y1="40" x2="350" y2="108"/></g>
  <text x="230" y="128" text-anchor="middle" font-size="9" fill="currentColor">peak at own symbol time, zero at every neighbouring symbol time → no ISI</text>
</svg>
<figcaption>A raised-cosine pulse peaks at its own sampling instant and crosses exactly zero at every other symbol time, so neighbouring symbols do not interfere.</figcaption>
</figure>

## How it works

The **raised-cosine** filter is a member of the Nyquist family: its impulse response is zero
at every integer symbol period except the centre, so a stream of such pulses can be sampled
at the symbol clock with no contribution from adjacent symbols. Splitting that response into
two identical square-root ("root") halves — one at the transmitter, one at the receiver —
keeps the composite Nyquist property while giving the receive filter a second job (below).

The single design knob is the **roll-off factor** β (0 to 1). It sets the **excess
bandwidth**: the occupied bandwidth is `(1 + β)` times the Nyquist minimum of half the
[symbol rate](/reference/symbol-rate/). A small β (say 0.2) is spectrally tight but the
pulse tails ring long, so timing error is punishing and the peak-to-average power ratio
climbs; a large β (0.35–0.5) is more forgiving of timing jitter but eats more spectrum.
Practical systems pick β as a compromise — TETRA uses β ≈ 0.35, for instance.

## Variants: RRC vs. raised-cosine, and the matched pair

The distinction that trips people up: the **raised-cosine** is the *end-to-end* channel
response that has zero ISI, while the **root-raised-cosine** is what you actually load into
each individual filter so that TX × RX = raised-cosine. Crucially, an RRC pulse on its own
is **not** ISI-free — its zero crossings do not land on the symbol grid — which is why you
must have the matching RRC at the far end. That pairing is not a coincidence: the receive
RRC is precisely the [matched filter](/reference/matched-filter/) for an RRC-shaped transmit
pulse, so the same filter that completes the Nyquist shaping also maximises signal-to-noise
ratio at the sampling instant. One filter, two payoffs — zero ISI and optimal SNR — which is
exactly why the square root is split symmetrically across the link.

## In practice

Correctly applying the receive RRC is a required step in demodulating any linearly-modulated
digital signal that specifies it. Related C4FM/CQPSK systems such as
[P25](/reference/p25-phase-1/) use a raised-cosine / shaped-Gaussian family rather than a
pure RRC, whereas π/4-DQPSK systems like [TETRA](/reference/tetra/) and NADC specify RRC
shaping outright. A clean, wide-open [eye diagram](/reference/eye-diagram/) after the receive
RRC is the visual confirmation that the matched pair is doing its job.

## Relevance to SDR

Linear digital modes carried by [SDR](/reference/software-defined-radio/) receivers rely on
the receiver reconstructing the correct RRC to recover symbols cleanly. GopherTrunk applies
the appropriate receive pulse-shaping / matched filter for the linear modes it decodes,
sharpening [symbol](/reference/symbol-rate/) decisions before the slicer; the C4FM family it
handles most often uses a shaped-Gaussian variant of the same Nyquist idea rather than a
textbook RRC, but the goal — bandwidth-limited pulses that don't smear into one another — is
identical.

## Sources

[^wiki]: [Root-raised-cosine filter](https://en.wikipedia.org/wiki/Root-raised-cosine_filter) — Wikipedia, on split RRC pulse shaping and its role as the receive matched filter.
[^rc]: [Raised-cosine filter](https://en.wikipedia.org/wiki/Raised-cosine_filter) — Wikipedia, on the Nyquist zero-ISI response, roll-off factor, and excess bandwidth.
