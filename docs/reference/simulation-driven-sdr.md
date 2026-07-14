---
slug: simulation-driven-sdr
title: Simulation-driven SDR development
entry_type: concept
category: sdr-app-building
description: "Simulation-driven SDR development builds and verifies radio signal processing offline against synthesized or recorded signals with known parameters, before or instead of testing over the air."
keywords: simulation-driven SDR, offline SDR development, synthetic signal testing, channel model, IQ simulation, modulate demodulate loopback, additive noise, known parameters, DSP verification
aka: ["simulation-first SDR", "offline SDR development", "synthetic-signal development"]
autolink: true
infobox:
  - { label: Type, value: Development methodology }
  - { label: Idea, value: "Generate the signal you need, with known truth" }
  - { label: Used in, value: "DSP design, edge-case testing, CI" }
see_also: [testing-dsp-without-hardware, file-source-sink, iq-recording-playback, golden-test-vectors]
cite_urls:
  - https://en.wikipedia.org/wiki/Software-defined_radio
  - https://en.wikipedia.org/wiki/Additive_white_Gaussian_noise
---

**Simulation-driven SDR development** is the practice of designing and verifying
radio signal processing against signals whose parameters you control — synthesized
in software or replayed from a recording — rather than only against whatever
happens to be on the air.[^sdr] You generate the exact waveform you need, at the
exact SNR, frequency offset, or bit pattern you want to test, and because you know
the *ground truth* that produced it, you can assert the decoder recovered the
right answer, not merely a plausible one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Known bits are modulated to IQ, passed through a channel model that adds noise and a frequency offset, then decoded; the recovered bits are compared against the known bits to compute a bit error rate." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="sdsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="14" y="20" width="70" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="49" y="37">known bits</text>
    <line x1="84" y1="33" x2="118" y2="33" stroke="currentColor" stroke-width="1.1" marker-end="url(#sdsar)"/>
    <rect x="120" y="20" width="66" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="153" y="37">modulate</text>
    <line x1="186" y1="33" x2="220" y2="33" stroke="currentColor" stroke-width="1.1" marker-end="url(#sdsar)"/>
    <rect x="222" y="20" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/><text x="262" y="33">channel</text><text x="262" y="43" font-size="7">noise + offset</text>
    <line x1="262" y1="46" x2="262" y2="78" stroke="currentColor" stroke-width="1.1" marker-end="url(#sdsar)"/>
    <rect x="222" y="80" width="80" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="262" y="97">demodulate</text>
    <line x1="222" y1="93" x2="188" y2="93" stroke="currentColor" stroke-width="1.1" marker-end="url(#sdsar)"/>
    <rect x="118" y="80" width="70" height="26" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="153" y="97">recovered</text>
    <line x1="118" y1="93" x2="88" y2="93" stroke="currentColor" stroke-width="1.1" marker-end="url(#sdsar)"/>
    <rect x="14" y="80" width="70" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="49" y="93">compare</text><text x="49" y="102" font-size="7">→ BER</text>
    <line x1="49" y1="46" x2="49" y2="78" stroke="currentColor" stroke-width="1.1" stroke-dasharray="2 2" marker-end="url(#sdsar)"/>
  </g>
</svg>
<figcaption>A modulate → channel → demodulate loop with known transmit bits yields a measurable bit error rate — impossible when the truth is unknown off-air.</figcaption>
</figure>

## How it works

The core is a *loopback* through a channel model. You start from known transmit
bits, run them through the same modulator the target system uses, then deliberately
degrade the resulting IQ with a model of the channel: additive white Gaussian
noise scaled to a chosen SNR,[^awgn] a carrier frequency and phase offset, a timing
offset, and — for harder cases — multipath fading. That impaired IQ is exactly the
kind of stream a real receiver sees, so you feed it to the demodulator and framing
code under test.

Because you kept the original bits, you can compute concrete correctness metrics:
[bit error rate](/reference/bit-error-rate/) versus SNR, how much frequency offset
the acquisition tolerates, whether the
[forward-error-correction](/reference/forward-error-correction/) fixes the number
of errors it should. This is how DSP is designed rather than merely spot-checked:
you can sweep a parameter and see the performance curve, then compare it against
theory.

Synthesis and capture are complements, not rivals. Synthetic signals give you
perfect ground truth and arbitrary edge cases; recorded
[captures](/reference/iq-recording-playback/) give you the messy real
imperfections — a specific transmitter's phase noise, a real fading environment —
that no simple channel model reproduces. Mature projects use both.

## In practice

Synthetic signals shine for the cases you cannot summon on demand: a burst sitting
right at threshold, a decoder input crafted to force a particular
error-correction path, a precise ±kHz offset to test acquisition range. You dial
the impairment in exactly and get a deterministic, checked-in test.

Captures shine for realism and for reproducing field bugs. When a user reports a
decode failure, the fix starts by getting their raw
[recording](/reference/cfile-format/) and replaying it through a
[file source](/reference/file-source-sink/) until the failure reproduces on the
bench — after which a [golden vector](/reference/golden-test-vectors/) locks the
fix in place. The whole cycle runs with no radio attached, which is the point of
[hardware-free testing](/reference/testing-dsp-without-hardware/).

The discipline's caveat is that a simulation is only as honest as its channel
model. A decoder that passes against clean AWGN can still fail on real hardware
impairments the model omits (front-end nonlinearity, reciprocal mixing, ADC
artifacts), so a green simulation suite is necessary but not sufficient — real
captures remain the final word.

## Relevance to SDR

Simulation-driven work is standard practice across the field: MATLAB/Simulink,
GNU Radio, and NumPy/SciPy pipelines all make it easy to modulate, add a channel,
and demodulate in a loop, and every published BER-vs-Eb/N0 curve is a simulation
result. It lets a modem be designed and its performance bounded before any RF
hardware exists.

**GopherTrunk** leans on the capture-replay half of this heavily and uses
synthetic inputs where they pin behavior precisely. A concrete example from its own
history: a field decode failure was diagnosed by replaying the reporter's captures
offline, then *independently* resampling a high-rate capture with a separate
resampler and replaying it through the proven low-rate path — reproducing the same
degraded SNR and proving the deficit was baked into the captured samples (front-end
phase noise), not GT's downconverter. That is simulation-driven debugging: control
one variable at a time, offline, against known-good references. GT does not model
transmit channels (it is a receiver), so its simulation surface is receive-side —
degraded captures and synthesized decoder inputs — rather than a full TX/RX
loopback.

## Sources

[^sdr]: [Software-defined radio](https://en.wikipedia.org/wiki/Software-defined_radio) — Wikipedia, on processing digitized baseband in software, which lets signals be synthesized and replayed identically to live reception.
[^awgn]: [Additive white Gaussian noise](https://en.wikipedia.org/wiki/Additive_white_Gaussian_noise) — Wikipedia, on the standard channel-noise model used to set a target SNR in link simulations.
