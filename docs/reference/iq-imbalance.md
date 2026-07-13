---
slug: iq-imbalance
title: IQ imbalance
entry_type: term
category: sdr-dsp
description: "IQ imbalance is the gain and phase mismatch between the I and Q branches of a quadrature receiver, producing a mirror image of the wanted signal."
keywords: IQ imbalance, I/Q imbalance, gain mismatch, phase mismatch, quadrature error, image rejection, mirror image, spurious image, direct conversion, zero-IF, SDR calibration
aka: [I/Q imbalance, quadrature imbalance, gain and phase mismatch]
autolink: true
infobox:
  - { label: Type, value: Receiver impairment }
  - { label: Cause, value: Unequal I/Q gain & non-90° phase }
  - { label: Symptom, value: Image at mirror frequency }
see_also: [iq-data, image-rejection, dc-offset, iq-modulation, quadrature-demodulation, direct-conversion-receiver]
cite_urls:
  - https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation
  - https://en.wikipedia.org/wiki/Direct-conversion_receiver
---

**IQ imbalance** is the gain and phase mismatch between the in-phase (I) and
quadrature (Q) branches of a [quadrature receiver](/reference/quadrature-demodulation/),
which corrupts the [complex baseband](/reference/iq-data/) and raises a mirror-frequency
image of the wanted signal.[^wiki] Ideally the two branches have identical gain and are
exactly 90° apart; real mixers, filters, and amplifiers never match perfectly, so a signal
at +f leaks a weaker copy to −f. It is one of the two headline impairments of
[direct-conversion](/reference/direct-conversion-receiver/) SDR front ends, alongside
[DC offset](/reference/dc-offset/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A spectrum showing a strong wanted tone at positive frequency and a weaker unwanted image tone at the mirror negative frequency caused by IQ imbalance." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="130" x2="440" y2="130" stroke="currentColor" stroke-width="1.2"/>
  <line x1="235" y1="30" x2="235" y2="140" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <text x="235" y="152" font-size="9" fill="currentColor" text-anchor="middle">0</text>
  <text x="360" y="152" font-size="9" fill="currentColor" text-anchor="middle">+f</text>
  <text x="110" y="152" font-size="9" fill="currentColor" text-anchor="middle">−f (image)</text>
  <line x1="360" y1="130" x2="360" y2="45" stroke="currentColor" stroke-width="3"/>
  <text x="360" y="38" font-size="9" fill="currentColor" text-anchor="middle">wanted</text>
  <line x1="110" y1="130" x2="110" y2="102" stroke="currentColor" stroke-width="3"/>
  <text x="110" y="95" font-size="9" fill="currentColor" text-anchor="middle">image</text>
  <path d="M120 102 Q 235 80 350 47" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2" marker-end="url(#iqiar)"/>
  <text x="245" y="72" font-size="8.5" fill="currentColor">IRR = wanted/image</text>
  <defs><marker id="iqiar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Gain/phase mismatch spills a scaled, conjugated copy of a tone at +f into the mirror bin at −f; the ratio of the two is the image-rejection ratio (IRR).</figcaption>
</figure>

## How it works

A quadrature receiver multiplies the incoming RF by two [local-oscillator](/reference/local-oscillator/)
copies that should be identical in amplitude and exactly 90° apart. Model the mismatch as an
amplitude error `g` (the I and Q gains differ) and a phase error `φ` (the two LO copies are not
exactly quadrature). The recovered baseband is then not the clean complex signal `x(t)` but a
weighted sum of `x(t)` and its complex conjugate `x*(t)`. Because conjugation flips the sign of
frequency, that `x*(t)` term places a scaled copy of every component at its mirror frequency —
the image. The suppression is quantified by the **image-rejection ratio (IRR)**, the power ratio
of the wanted component to its image. Small errors already limit rejection sharply: a 1° phase
error or a 0.1 dB gain error each cap the IRR near 40 dB, and the two combine.

The image is damaging for two reasons. A strong out-of-band signal at −f can drop an image
directly on top of a weak wanted signal at +f, and in a wideband SDR a bright carrier throws a
spur into an otherwise empty part of the passband, which can masquerade as a real emission on a
[waterfall display](/reference/waterfall-display/).

## Variants and correction

- **Frequency-independent (narrowband) imbalance** — a single complex gain error across the
  channel, from the mixer and LO phase splitter. It is corrected by estimating the two error
  terms and applying a 2×2 real matrix (or an equivalent complex-plus-conjugate combination) to
  the I/Q stream. Blind methods exploit the fact that a proper complex signal has zero
  correlation between `x` and `x*`; forcing that correlation to zero removes the imbalance
  without a test tone.
- **Frequency-dependent imbalance** — the I and Q analog filters have slightly different
  responses, so `g` and `φ` vary across the band. This needs a short adaptive FIR filter on one
  branch rather than a single complex multiply.
- **Factory calibration** — many SDRs inject a known tone and measure the resulting image to
  solve for the correction coefficients once, storing them for reuse.

Correcting IQ imbalance improves [image rejection](/reference/image-rejection/) and cleans up the
[constellation](/reference/constellation-diagram/), lowering the error-vector magnitude before
demodulation.

## In practice

The imbalance is easy to see and measure. Feed the receiver a single clean test tone offset from
the tuned center: a perfect quadrature receiver shows one line on a spectrum, while an imbalanced
one shows a second, weaker line at the mirror offset, and the dB gap between them is the IRR
directly. On a [waterfall](/reference/waterfall-display/) the tell-tale is a faint "ghost"
carrier that moves *opposite* to the real one as you retune — a real signal and its image slide
in mirror directions about DC. Two practical points follow. First, IQ correction and
[DC-offset](/reference/dc-offset/) removal are usually done together, since both are artefacts of
the same zero-IF architecture and both live right around the center bin. Second, correction is
only as stable as the estimate: temperature drift and AGC gain changes shift the coefficients, so
a good corrector adapts slowly and continuously rather than calibrating once and freezing.

## Relevance to SDR

Direct-conversion and low-IF tuners — the R820T/RTL-SDR chain, Airspy, HackRF, and similar — all
exhibit IQ imbalance because their quadrature mixers are analog. Popular SDR applications
(SDR#, SDRangel, GQRX) include automatic IQ-correction that runs the blind conjugate-nulling
estimator continuously. For a trunking decoder the practical stakes are modest: the control and
voice channels GopherTrunk tunes are narrowband and the [downconverter](/reference/digital-down-converter/)
selects a single channel, so a residual image usually lands off-channel. GopherTrunk relies on
the front-end/driver correction rather than implementing its own IQ-balancer, but a large
uncorrected image can still appear as a phantom channel on a spectrum view, which is worth
recognising when diagnosing a capture.

## Sources

[^wiki]: [Quadrature amplitude modulation](https://en.wikipedia.org/wiki/Quadrature_amplitude_modulation) — Wikipedia, on I/Q representation and the effect of gain/phase mismatch between branches.
