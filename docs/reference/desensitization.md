---
slug: desensitization
title: Desensitization (desense)
entry_type: term
category: rf-metrics
description: Desensitization is the loss of receiver sensitivity caused by a strong out-of-band signal that raises the effective noise floor, reducing the ability to hear weak signals.
keywords: desensitization, desense, receiver desense, blocking, reciprocal mixing, out-of-band interference, noise floor rise, receiver overload, front-end compression
aka: [desense, receiver desensitization, blocking]
autolink: true
infobox:
  - { label: Type, value: Receiver impairment }
  - { label: Cause, value: Strong nearby / out-of-band signal }
  - { label: Effect, value: Higher noise floor, lost sensitivity }
see_also: [receiver-sensitivity, dynamic-range, blocking-dynamic-range, intermodulation, noise-floor, third-order-intercept]
cite_urls:
  - https://en.wikipedia.org/wiki/Desensitization_(telecommunications)
  - https://en.wikipedia.org/wiki/Reciprocal_mixing
---

**Desensitization** (**desense**) is the reduction of a receiver's
[sensitivity](/reference/receiver-sensitivity/) caused by a strong signal —
usually out-of-band — that drives the front end toward compression or injects
oscillator noise, raising the effective [noise floor](/reference/noise-floor/)
so weak wanted signals disappear.[^wiki] The interfering signal need not be on
the tuned frequency at all; it simply has to be strong enough to disturb the
receiver's linear operating point or its local oscillator.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A spectrum showing a weak wanted signal that clears the noise floor until a strong nearby blocker raises the floor above it, so the wanted signal is lost." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="150" x2="430" y2="150" stroke="currentColor" stroke-opacity="0.6"/>
  <line x1="30" y1="20" x2="30" y2="150" stroke="currentColor" stroke-opacity="0.6"/>
  <line x1="30" y1="120" x2="210" y2="120" stroke="currentColor" stroke-dasharray="4 3" stroke-opacity="0.5"/>
  <text x="34" y="115" font-size="9" fill="currentColor">low floor</text>
  <path d="M120 120 L128 78 L136 120 Z" fill="currentColor" fill-opacity="0.3" stroke="currentColor" stroke-width="1.3"/>
  <text x="128" y="72" text-anchor="middle" font-size="9" fill="currentColor">wanted</text>
  <path d="M300 150 L316 30 L332 150 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor" stroke-width="1.4"/>
  <text x="316" y="24" text-anchor="middle" font-size="9" fill="currentColor">blocker</text>
  <line x1="250" y1="95" x2="430" y2="95" stroke="currentColor" stroke-dasharray="2 3" stroke-opacity="0.7"/>
  <text x="360" y="90" font-size="9" fill="currentColor">raised floor</text>
  <path d="M250 95 L250 150 L430 150 L430 95 Z" fill="currentColor" fill-opacity="0.08"/>
</svg>
<figcaption>A strong blocker lifts the effective noise floor above the wanted signal, which then no longer decodes — the hallmark of desensitization.</figcaption>
</figure>

## How it works

Two mechanisms dominate. First, **gain compression**: every amplifier and mixer is
linear only over a limited range. A strong signal pushes the low-noise amplifier or
first mixer toward its [1 dB compression point](/reference/1-db-compression-point/),
where stage gain falls. A weak wanted signal sharing that stage sees the same gain
drop, so its level relative to the internally generated noise worsens — sensitivity
is lost even though nothing changed at the wanted frequency.

Second, **reciprocal mixing**: no [local oscillator](/reference/local-oscillator/) is
spectrally pure; it carries [phase noise](/reference/phase-noise/) skirts. A strong
off-channel signal mixes with those noise skirts and lands broadband energy in the IF
passband, directly raising the noise floor around the wanted channel. This is why a
receiver can desense even when the front end is nowhere near compression: the culprit
is oscillator purity, not amplifier headroom.

Desensitization is quantified as the interfering-signal level needed to degrade
sensitivity by a fixed amount (commonly 3 dB or 6 dB), measured at a stated frequency
offset. The span between the noise floor and that blocking level is the
[blocking dynamic range](/reference/blocking-dynamic-range/).

## In practice

- A pager transmitter, broadcast-FM station, or nearby land-mobile repeater a few MHz
  away can desense a wideband scanner even though it is far outside the tuned channel.
- Software-defined radios with wide, unfiltered front ends —
  [RTL-SDR](/reference/rtl-sdr/) dongles especially — are prone to desense because the
  whole spectrum reaches the [analog-to-digital converter](/reference/analog-to-digital-converter/)
  before any channel selection.
- Cures are physical: a bandpass or notch [RF filter](/reference/rf-filter/) ahead of
  the receiver, an [attenuator](/reference/attenuator/) to pull the strong signal out of
  compression, better antenna placement, or reducing front-end gain via the
  [automatic gain control](/reference/automatic-gain-control/) settings.
- Adding gain with a [preamplifier](/reference/preamplifier/) often makes desense worse,
  not better, because it amplifies the blocker toward compression too.

## Relevance to SDR

Desensitization is one of the most common real-world reasons a trunking signal that
"should" be receivable is not. In dense RF environments — mountaintop sites,
apartment blocks near paging or cellular transmitters — a strong neighbor can lift the
noise floor across the band a wide-front-end SDR is watching. Because
[GopherTrunk](/reference/software-defined-radio/) samples a wide swath of spectrum to
follow a trunked control channel and its voice grants, a single strong out-of-band
emitter can degrade decoding of an otherwise healthy channel. The fix lives in the RF
plumbing rather than the software: front-end filtering, attenuation, and correct gain
staging restore the [receiver's sensitivity](/reference/receiver-sensitivity/). GT can
report symptoms — a rising noise floor, falling demod SNR — but it cannot undo desense
that has already corrupted the samples reaching the
[ADC](/reference/analog-to-digital-converter/).

## Sources

[^wiki]: [Desensitization (telecommunications)](https://en.wikipedia.org/wiki/Desensitization_(telecommunications)) — Wikipedia, definition and mechanisms of receiver desensitization.
