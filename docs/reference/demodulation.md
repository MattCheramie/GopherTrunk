---
slug: demodulation
title: Demodulation
entry_type: term
category: sdr-dsp
description: Demodulation recovers the original modulating information from a carrier; for digital signals it produces the symbol stream that decoding then turns into bits.
keywords: demodulation, demodulator, FM PSK FSK, symbol recovery, decoding, pipeline, coherent, non-coherent, discriminator
aka: [demodulation]
autolink: true
infobox:
  - { label: Type, value: DSP stage }
  - { label: Recovers, value: Modulating signal from carrier }
  - { label: Followed by, value: Symbol recovery, decoding }
see_also: [modulation, quadrature-demodulation, clock-recovery, costas-loop, soft-decision, frame-synchronization, constellation-diagram, software-defined-radio]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 6: Demodulation", url: /blog/deep-dives/sdr-internals-06-demodulation/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Demodulation
  - https://en.wikipedia.org/wiki/Modulation
---

**Demodulation** recovers the original modulating information from a
[carrier](/reference/carrier-wave/) — the inverse of
[modulation](/reference/modulation/).[^wiki] For FM/[FSK](/reference/frequency-shift-keying/) it
tracks instantaneous frequency; for [PSK](/reference/phase-shift-keying/) it tracks
[phase](/reference/phase/); for AM it tracks the envelope. In a digital receiver the demodulator
does not output bits directly — it outputs a continuous, noisy waveform whose value at each
symbol instant encodes the transmitted symbol.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A modulated waveform entering a demodulator block and the recovered message waveform leaving it." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 55 q5 -16 10 0 q5 -22 10 0 q5 -16 10 0 q5 -8 10 0 q5 -16 10 0 q5 -22 10 0 q5 -16 10 0 q5 -8 10 0 q5 -16 10 0 q5 -22 10 0" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <rect x="200" y="38" width="74" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="237" y="59" font-size="9" fill="currentColor" text-anchor="middle">demod</text>
  <line x1="120" y1="55" x2="199" y2="55" stroke="currentColor" stroke-width="1.1"/>
  <path d="M290 55 Q 330 30 370 55 T 440 55" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <line x1="274" y1="55" x2="289" y2="55" stroke="currentColor" stroke-width="1.1" marker-end="url(#dmar)"/>
  <text x="365" y="92" font-size="9" fill="currentColor">recovered message</text>
  <defs><marker id="dmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Demodulation recovers the original modulating signal from the carrier — the step before decoding bits.</figcaption>
</figure>

## How it works

The demodulator undoes whatever operation the transmitter's modulator performed. Which quantity
it estimates depends on where the information lives:

- **Frequency modulation** (FM, FSK, C4FM) — recover the *instantaneous frequency*, the rate of
  change of phase. In SDR this is usually a [quadrature demodulator](/reference/quadrature-demodulation/):
  the phase difference between successive [IQ](/reference/iq-data/) samples.
- **Phase modulation** (PSK, QPSK, π/4-DQPSK) — recover the carrier phase. This needs a phase
  reference, typically a [Costas loop](/reference/costas-loop/) or a differential comparison
  against the previous symbol.
- **Amplitude modulation** (AM, ASK, and the amplitude axis of QAM) — recover the envelope.

The output is a continuous, noisy stream that *contains* the
[symbols](/reference/symbol-rate/); [clock recovery](/reference/clock-recovery/) then slices it
at the right instants into discrete symbols, and decoding turns those symbols into bits.
Demodulation handles the *waveform*; decoding handles the *data*.

## Variants

- **Coherent vs non-coherent.** A coherent demodulator reconstructs the carrier's phase and
  frequency and mixes against it — optimal at low SNR but requiring a tracking loop. A
  non-coherent demodulator (an FM discriminator, an envelope detector, a differential PSK
  slicer) avoids carrier recovery at a modest sensitivity cost. Robustness and simplicity often
  make the non-coherent path the practical choice for narrowband land-mobile signals.
- **Hard vs soft output.** A hard-decision demodulator emits the nearest symbol; a
  [soft-decision](/reference/soft-decision/) demodulator emits a confidence value per bit, which
  a downstream [forward-error-correction](/reference/forward-error-correction/) decoder can use
  to recover several dB of coding gain.

## In practice

A full digital receive chain is a pipeline: tune and filter to [baseband](/reference/baseband/),
demodulate to a symbol-bearing waveform, recover symbol timing, correct any residual carrier
offset, slice to symbols, find the [frame boundary](/reference/frame-synchronization/), then FEC
decode and de-interleave to bits. Demodulation sits early in that chain, and its output quality
sets a ceiling on everything after it: a [constellation diagram](/reference/constellation-diagram/)
or [eye diagram](/reference/eye-diagram/) of the demodulated signal is the standard way to see
whether that ceiling is high enough to decode.

## Relevance to SDR

Choosing the matching demodulator for a signal's modulation is the core of recovering it. Every
mode GopherTrunk decodes — the C4FM of P25 and DMR, the four-level FSK of NXDN, the GFSK of
paging — runs through a demodulator sized to that modulation before symbol and frame recovery.
The [constellation](/reference/constellation-diagram/) visualises this stage, and a demodulator
mismatched to the signal produces a smear that no amount of downstream decoding can rescue.

## Sources

[^wiki]: [Demodulation](https://en.wikipedia.org/wiki/Demodulation) — Wikipedia, on recovering the modulating signal as the inverse of modulation.
[^mod]: [Modulation](https://en.wikipedia.org/wiki/Modulation) — Wikipedia, on the amplitude, frequency, and phase carriers of information that demodulation recovers.
