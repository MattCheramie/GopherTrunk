---
slug: baseband
title: Baseband
entry_type: term
category: sdr-dsp
description: Baseband is a signal centred at (or near) zero frequency, after a receiver has mixed the carrier away. SDRs deliver IQ samples at baseband for software to process.
keywords: baseband, zero-IF, complex baseband, IQ, downconversion, DC, passband, negative frequency
aka: [baseband, "complex baseband"]
autolink: true
infobox:
  - { label: Type, value: Signal frequency band }
  - { label: Centre, value: At or near 0 Hz }
  - { label: SDR form, value: Complex (IQ) baseband }
see_also: [iq-data, intermediate-frequency, local-oscillator, dc-offset, direct-conversion-receiver, iq-imbalance, zero-if]
related_lessons:
  - { title: "IQ data & complex signals", url: /learn/rf-sdr/iq-data/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Baseband
  - https://en.wikipedia.org/wiki/In-phase_and_quadrature_components
---

**Baseband** is a signal centred at (or near) **zero frequency**, after the carrier has
been mixed away.[^wiki] An SDR delivers **complex baseband** [IQ samples](/reference/iq-data/) —
the channel shifted down to 0 Hz — which is the natural form for software to filter and
demodulate. The term contrasts with *passband*, where the same information rides on a carrier
somewhere up the spectrum.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A signal at a high carrier frequency shifted down to sit centred on zero hertz at baseband." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="85" x2="430" y2="85" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="120" y1="85" x2="120" y2="30" stroke="currentColor" stroke-dasharray="3 3" stroke-opacity="0.5"/><text x="120" y="100" text-anchor="middle" font-size="8" fill="currentColor">0 Hz</text>
  <path d="M340 85 L350 50 L360 85 Z" fill="none" stroke="currentColor"/><text x="350" y="42" text-anchor="middle" font-size="8" fill="currentColor">carrier</text>
  <path d="M110 85 L120 45 L130 85 Z" fill="currentColor" fill-opacity="0.25" stroke="currentColor"/>
  <path d="M340 60 q-110 -25 -215 -12" fill="none" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#bbar)"/>
  <text x="240" y="108" text-anchor="middle" font-size="8.5" fill="currentColor">mixed down to baseband (0 Hz)</text>
  <defs><marker id="bbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Baseband is the channel shifted down to 0 Hz; SDRs output complex (IQ) baseband samples.</figcaption>
</figure>

## How it works

A receiver mixes the wanted channel down until its centre sits at 0 Hz. A *real* baseband signal
(one number per sample) would fold the spectrum onto itself — a component just above 0 Hz would
be indistinguishable from one just below. Radios avoid this by using **complex baseband**: two
sampled streams, in-phase (I) and quadrature (Q), that together represent both positive and
negative frequencies around the centre.[^iq] With I and Q, a signal 10 kHz above the tuned
frequency and one 10 kHz below occupy genuinely different parts of the complex spectrum, so no
information is lost. This is why an SDR's native output is a stream of complex samples rather than
a single audio-like waveform.

Working at baseband is a matter of convenience and rate. Once a channel is at 0 Hz, only its own
bandwidth needs to be represented, so it can be [decimated](/reference/decimation/) to a low
sample rate and processed cheaply. All the receiver's remaining
[filtering](/reference/digital-filter/) and demodulation happen here.

## In practice

Producing baseband directly in the analog front end — mixing RF straight to 0 Hz — is
**[direct conversion](/reference/direct-conversion-receiver/)** (zero-IF). It is simple and
cheap, but it puts the sensitive part of the spectrum right where two hardware artefacts live: a
**[DC offset](/reference/dc-offset/)** spike from LO leakage at the exact centre, and
**[IQ imbalance](/reference/iq-imbalance/)** — gain or phase mismatch between the I and Q paths
that leaks a mirror image of each signal across 0 Hz. Both are corrected in software or sidestepped
by tuning slightly off-centre. Architectures that mix to a small
[low-IF](/reference/low-if/) instead keep the wanted channel just off 0 Hz to dodge the DC spike,
then shift it the last step digitally.

## Relevance to SDR

Baseband is where nearly all SDR software actually operates. GopherTrunk's digital
down-converter translates each control or voice channel to complex baseband, filters and
decimates it to the per-protocol channel rate, and hands that stream to the demodulator. Reading
a signal as [IQ](/reference/iq-data/) at baseband — with its symmetric positive and negative
frequency axes — is the foundation every later stage builds on.

## Sources

[^wiki]: [Baseband](https://en.wikipedia.org/wiki/Baseband) — Wikipedia, on signals centred near zero frequency and the passband contrast.
[^iq]: [In-phase and quadrature components](https://en.wikipedia.org/wiki/In-phase_and_quadrature_components) — Wikipedia, on the I/Q representation that lets complex baseband carry positive and negative frequencies.
