---
slug: phase-noise
title: Phase Noise
entry_type: term
category: rf-fundamentals
description: Phase noise is an oscillator's short-term random phase fluctuation, spreading a carrier into skirts and causing reciprocal mixing that raises a receiver's noise floor.
keywords: phase noise, oscillator noise, dBc/Hz, reciprocal mixing, phase jitter, carrier skirts, close-in phase noise, spectral purity, local oscillator noise
aka: [phase noise, oscillator phase noise, SSB phase noise]
autolink: true
infobox:
  - { label: Type, value: Short-term random phase fluctuation }
  - { label: Unit, value: dBc/Hz (at an offset from carrier) }
  - { label: Key effect, value: Reciprocal mixing }
see_also: [local-oscillator, frequency-stability, phase-locked-loop, mixer-rf, error-vector-magnitude, noise-floor]
cite_urls:
  - https://en.wikipedia.org/wiki/Phase_noise
---

**Phase noise** is the rapid, random fluctuation in the phase of an oscillator's output,
which smears an ideally pure carrier into a band of "skirts" around its nominal
frequency.[^wiki] It is the frequency-domain face of timing jitter, and in a receiver it
is the mechanism behind **reciprocal mixing** — the effect by which a noisy
[local oscillator](/reference/local-oscillator/) lets a strong nearby signal raise the
[noise floor](/reference/noise-floor/) on the channel you are trying to hear.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="Two carrier spectra: an ideal oscillator drawn as a single infinitely thin spike, and a real oscillator drawn as a spike surrounded by broad skirts that fall off with offset from the carrier." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pnar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" fill="none" font-size="10">
    <line x1="45" y1="20" x2="45" y2="140" marker-end="url(#pnar)"/>
    <line x1="45" y1="140" x2="440" y2="140" marker-end="url(#pnar)"/>
    <line x1="140" y1="35" x2="140" y2="140" stroke-width="2"/>
    <path d="M300 140 Q330 132 350 95 Q360 45 370 45 Q380 45 390 95 Q410 132 440 140" stroke-width="1.7"/>
  </g>
  <g fill="currentColor" font-size="10" stroke="none">
    <text x="8" y="30">power</text>
    <text x="415" y="158">freq</text>
    <text x="95" y="30">ideal</text>
    <text x="330" y="34">real: skirts</text>
    <text x="122" y="152">f₀</text>
    <text x="352" y="152">f₀</text>
  </g>
</svg>
<figcaption>An ideal oscillator is a single spectral line; a real one carries phase-noise skirts that fall off with offset from the carrier, measured in dBc/Hz.</figcaption>
</figure>

## How it works

An oscillator's output can be written as *cos(2πf₀t + φ(t))*, where *φ(t)* is a small
random phase perturbation. Because that phase wanders continuously, energy that would sit
in a single line at *f₀* is instead spread into a continuous pedestal on either side. The
standard measure is **£(f), single-sideband phase noise in dBc/Hz**: the ratio of noise
power in a 1 Hz bandwidth at a given *offset* from the carrier to the total carrier power.
A quote like "−110 dBc/Hz at 10 kHz offset" means that 10 kHz away, each hertz of
bandwidth holds 110 dB less power than the carrier. Phase noise is worst *close in* and
improves as you move out, eventually flattening into the oscillator's broadband noise.

**Reciprocal mixing** is why this matters on receive. In a [mixer](/reference/mixer-rf/),
the [local oscillator](/reference/local-oscillator/) multiplies with the incoming RF. If
the LO carries phase-noise skirts, a *strong* off-channel signal mixes with those skirts
and its energy lands in your IF passband as raised noise — even though the strong signal
is nominally out of band. A clean LO keeps the skirts low so nearby strong signals stay
out.

On transmit, the same phase noise degrades the signal directly: it rotates the
constellation from symbol to symbol, inflating the
[error vector magnitude](/reference/error-vector-magnitude/) and limiting how high an
order of modulation the link can carry.

## In practice

Phase noise is set by oscillator design. A free-running crystal oscillator is good close
in; a [phase-locked loop](/reference/phase-locked-loop/) synthesizer trades close-in
noise (dominated by the reference and loop) against far-out noise (dominated by the VCO),
and the loop bandwidth is chosen to blend the two. Higher output frequencies are worse:
multiplying an oscillator up by *N* raises its phase noise by *20·log₁₀(N)* dB. This is a
distinct property from long-term [frequency stability](/reference/frequency-stability/) —
phase noise is the *short-term* jitter around the carrier, while stability is the slow
drift of the carrier's average frequency; a source can be excellent at one and poor at
the other.

## Relevance to SDR

Reciprocal mixing is a real limit for SDR monitoring in crowded RF environments. A budget
tuner with a noisy synthesizer can have its usable sensitivity destroyed near a strong
pager or broadcast transmitter, because that signal reciprocal-mixes with the LO skirts
and buries a weak trunking [control channel](/reference/control-channel/). This is often
mistaken for poor sensitivity when the real fault is LO spectral purity; better SDRs
(and a preselecting band-pass filter to keep the strong signal out) are the fix.

**GopherTrunk** is a software decoder and does not generate any RF oscillator of its own;
the phase noise that reaches its decode chain is whatever the front-end SDR contributed
before sampling. The concept is diagnostic for its users: a signal that looks strong on
the waterfall yet decodes with a stubbornly high error rate near a powerful neighbour is
a classic reciprocal-mixing signature pointing at the receiver's oscillator, not at
GopherTrunk's DSP.

## Sources

[^wiki]: [Phase noise](https://en.wikipedia.org/wiki/Phase_noise) — Wikipedia, definition of oscillator phase noise, dBc/Hz, and reciprocal mixing.
