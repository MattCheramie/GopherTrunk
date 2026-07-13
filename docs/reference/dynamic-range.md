---
slug: dynamic-range
title: Dynamic range
entry_type: term
category: rf-metrics
description: Dynamic range is the span in decibels between a receiver's noise floor and the point where strong signals overload it; SFDR and blocking are its two flavours.
keywords: dynamic range, receiver dynamic range, SFDR, blocking dynamic range, noise floor, overload, third-order intermodulation, ADC dynamic range, headroom
aka: [receiver dynamic range]
autolink: true
infobox:
  - { label: Symbol, value: "DR" }
  - { label: Unit, value: Decibels (dB) }
  - { label: Spans, value: "noise floor → overload" }
see_also: [spurious-free-dynamic-range, blocking-dynamic-range, third-order-intercept, 1-db-compression-point, noise-floor, intermodulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Dynamic_range
  - https://en.wikipedia.org/wiki/Spurious-free_dynamic_range
---

**Dynamic range** is the span, in [decibels](/reference/decibel/), between the
weakest signal a receiver can usefully detect — its
[noise floor](/reference/noise-floor/) — and the strongest it can handle before
distortion or overload makes it useless.[^wiki] It measures the receiver's ability
to hear a faint signal *while a much stronger one is present*, which is the real
test in a crowded band. A radio can have superb sensitivity yet poor dynamic range,
collapsing into a mess of spurious products the moment a strong nearby transmitter
appears. Two figures pin down the top end: the
[spurious-free dynamic range](/reference/spurious-free-dynamic-range/) (SFDR), set by
intermodulation, and the [blocking dynamic range](/reference/blocking-dynamic-range/),
set by gain compression and desensitization.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 185" role="img" aria-label="A vertical power scale from the noise floor at the bottom to the overload point at top, with dynamic range spanning between, the SFDR ceiling set by spurious products and the blocking ceiling set by the 1 dB compression point." xmlns="http://www.w3.org/2000/svg">
  <line x1="70" y1="20" x2="70" y2="165" stroke="currentColor" stroke-width="1.4"/>
  <text x="60" y="18" text-anchor="end" font-size="9" fill="currentColor">stronger</text>
  <line x1="70" y1="35" x2="300" y2="35" stroke="currentColor" stroke-width="1.6"/>
  <text x="308" y="39" font-size="10" fill="currentColor">1 dB compression → overload</text>
  <line x1="70" y1="75" x2="300" y2="75" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/>
  <text x="308" y="79" font-size="10" fill="currentColor">spurs rise above floor (SFDR top)</text>
  <line x1="70" y1="150" x2="300" y2="150" stroke="currentColor" stroke-width="1.6"/>
  <text x="308" y="154" font-size="10" fill="currentColor">noise floor</text>
  <path d="M120 145 L120 40" stroke="currentColor" stroke-width="1.4" marker-end="url(#drar)"/>
  <path d="M120 40 L120 145" stroke="currentColor" stroke-width="1.4" marker-end="url(#drar)"/>
  <defs><marker id="drar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="128" y="98" font-size="11" fill="currentColor">dynamic range</text>
  <path d="M180 145 L180 78" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
  <text x="186" y="116" font-size="9.5" fill="currentColor" fill-opacity="0.85">SFDR</text>
</svg>
<figcaption>Dynamic range is the window from noise floor to overload; the SFDR ceiling (where spurs emerge) is usually lower than the blocking/compression ceiling.</figcaption>
</figure>

## How it works

The bottom of the window is fixed by physics: the noise floor is kTB raised by the
receiver's [noise figure](/reference/noise-figure/) over the measurement bandwidth.
The top is set by nonlinearity. As input power rises, an ideal receiver would keep
amplifying linearly, but every real amplifier and mixer eventually saturates. Two
distinct ceilings result:

- **Spurious-free dynamic range (SFDR).** As two or more strong signals mix in the
  non-linear front end they create [intermodulation](/reference/intermodulation/)
  products — most damaging the third-order ones that fall in-band. SFDR is the range
  from the noise floor up to the input level at which those spurs just rise above the
  floor. It is governed by the
  [third-order intercept](/reference/third-order-intercept/) (IP3).
- **Blocking dynamic range.** A single very strong signal can compress the gain and
  raise the noise floor (reciprocal mixing through
  [phase noise](/reference/phase-noise/)), desensitizing the receiver to the wanted
  signal. This ceiling is tied to the
  [1 dB compression point](/reference/1-db-compression-point/) and shows up as
  [desensitization](/reference/desensitization/).

Because a two-tone third-order spur grows three times faster (in dB) than the tones
that create it, SFDR relates to IP3 by the compact rule **SFDR = ⅔·(IP3 − noise
floor)**. SFDR is usually the narrower, more limiting number in a busy spectrum.

## In practice

For an analog receiver the front-end amplifier and mixer set dynamic range. For a
digital or software-defined receiver the
[analog-to-digital converter](/reference/analog-to-digital-converter/) is often the
bottleneck: its dynamic range is bounded by quantization noise at the bottom and
full-scale clipping at the top, roughly 6.02·N + 1.76 dB for N effective bits, and
in practice by [ENOB](/reference/enob/) and available headroom. Gain staging is the
art of positioning the whole chain inside this window — enough gain that the wanted
signal clears the noise floor, not so much that a strong neighbour drives the ADC or
front end into overload.

## Relevance to SDR

Dynamic range is where cheap SDRs struggle most. An RTL-SDR's 8-bit ADC gives only
about 48 dB of raw quantization-limited range, so a strong FM broadcast or pager
transmitter a few hundred kHz away can generate intermod spurs and desensitization
that bury a weak trunking control channel — a classic SFDR-limited failure that no
amount of software can undo once the samples are corrupted. Higher-end receivers
(Airspy, SDRplay, USRP) with 12–16-bit ADCs and better front ends buy tens of dB more
range, and external front-end filters (a cavity or bandpass filter ahead of the
radio) attack the problem directly by removing the strong out-of-band signals before
they reach the non-linear stages.

For GopherTrunk this is a capture-side concern: the decoder can only work with the
dynamic range the front end preserved. If a control channel decodes cleanly in
isolation but falls apart when a strong local signal is on, the culprit is dynamic
range in the RF chain — add front-end filtering or reduce gain — not the DSP.

## Sources

[^wiki]: [Dynamic range](https://en.wikipedia.org/wiki/Dynamic_range) — Wikipedia, definition of dynamic range and its receiver-specific SFDR and blocking variants.
