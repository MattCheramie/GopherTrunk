---
slug: zero-crossing-detection
title: Zero-crossing detection
entry_type: algorithm
category: synchronization
description: Zero-crossing detection estimates a signal's frequency or symbol timing from the instants it crosses zero — a simple, low-cost basis for FSK demodulation and clock recovery.
keywords: zero-crossing detection, zero crossing, frequency estimation, FSK demodulation, clock recovery, symbol timing, discriminator, low-cost demod, noise sensitivity
aka: [zero-crossing detector, zero-crossing counter]
autolink: true
infobox:
  - { label: Type, value: Timing / frequency estimator }
  - { label: Measures, value: Zero-crossing instants & spacing }
  - { label: Used for, value: FSK demod, clock recovery }
see_also: [clock-recovery, frequency-shift-keying, quadrature-demodulation, afsk, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Zero_crossing
  - https://en.wikipedia.org/wiki/Frequency-shift_keying
---

**Zero-crossing detection** estimates a signal's frequency and timing by watching
the instants at which it crosses zero.[^wiki] The spacing between successive
crossings is a direct, if coarse, measure of instantaneous frequency, and the
crossing edges themselves provide a timing reference — which makes zero-crossing
methods a cheap route to [FSK](/reference/frequency-shift-keying/) demodulation and
symbol [clock recovery](/reference/clock-recovery/). The appeal is simplicity: no
multiplies, no phase loop, just detect sign changes and time them.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A sine wave crossing a horizontal zero axis, with the upward-going crossings marked; wider spacing between crossings indicates a lower frequency, narrower spacing a higher frequency." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="20" y1="70" x2="440" y2="70" stroke="currentColor" stroke-width="1"/>
    <path d="M20 70 Q 55 15 90 70 Q 125 125 160 70 Q 195 15 230 70 Q 258 25 285 70 Q 312 115 340 70 Q 367 25 395 70 Q 422 115 440 70" fill="none" stroke="currentColor" stroke-width="1.4"/>
    <circle cx="90" cy="70" r="3" fill="currentColor"/>
    <circle cx="160" cy="70" r="3" fill="currentColor"/>
    <circle cx="230" cy="70" r="3" fill="currentColor"/>
    <circle cx="285" cy="70" r="3" fill="currentColor"/>
    <circle cx="340" cy="70" r="3" fill="currentColor"/>
    <circle cx="395" cy="70" r="3" fill="currentColor"/>
    <text x="125" y="100">wide gap = low f</text>
    <text x="340" y="100">narrow gap = high f</text>
  </g>
</svg>
<figcaption>The time between successive zero crossings measures the half-period, so widening or narrowing crossing spacing signals a lower or higher instantaneous frequency — the basis of zero-crossing FSK demodulation.</figcaption>
</figure>

## How it works

Two things fall out of the crossing instants:

- **Frequency.** The interval between two consecutive zero crossings is half the
  waveform's period, so counting crossings in a fixed window (or timing the gap
  between them) estimates instantaneous frequency. For a binary FSK signal the two
  tones produce two distinct crossing rates; thresholding the estimate recovers the
  bits — this is a **zero-crossing discriminator**, historically a common
  low-complexity FM/FSK demodulator.
- **Timing.** In a baseband symbol stream the transitions between symbol levels
  create zero crossings (after removing any DC bias). Those edges mark where the
  data clock ticks; a loop can align a local symbol clock to them, recovering timing
  much as a digital PLL would but from edge events rather than a matched-filter peak.

Practically, one interpolates between the two samples straddling a sign change to
locate the crossing to sub-sample accuracy, since the true crossing rarely lands
exactly on a sample.

## Limitations under noise

Zero crossings are attractively cheap but fragile:

- **Noise creates false crossings.** Near a genuine crossing the signal moves slowly
  through zero, so even modest noise flips the sign repeatedly, spawning spurious
  edges that corrupt both frequency and timing estimates. A hysteresis band
  (Schmitt-trigger behaviour) or pre-filtering to the signal bandwidth mitigates this
  but does not eliminate it.
- **Quantized resolution.** With few samples per cycle the crossing time — and thus
  the frequency estimate — is coarse unless interpolated.
- **Amplitude/DC sensitivity.** A DC offset or slow fade shifts where the waveform
  crosses zero, biasing the timing; the input usually must be centred first.

For these reasons high-performance receivers prefer coherent or matched-filter
approaches — a [Costas loop](/reference/costas-loop/) or Gardner timing detector — and
reserve zero-crossing methods for low-cost or coarse-acquisition roles. It contrasts
with [quadrature demodulation](/reference/quadrature-demodulation/), which recovers
instantaneous frequency from the differentiated arctangent of I/Q rather than from
sign changes and degrades far more gracefully in noise.

## Relevance to SDR

Zero-crossing FSK/FM discriminators show up in undemanding links — pager and telemetry
receivers, [AFSK](/reference/afsk/) modems, DTMF and tone detectors, and simple RTTY
decoders — where their negligible cost outweighs their noise penalty. In trunked-radio
work the underlying data is FSK-family, but production decoders (GopherTrunk included)
use quadrature/discriminator demodulation with proper matched filtering and closed-loop
timing rather than raw zero-crossing counting, because control-channel reliability at
low SNR demands it. Zero-crossing detection remains a useful mental model and a handy
first-pass frequency estimate.

## Sources

[^wiki]: [Zero crossing](https://en.wikipedia.org/wiki/Zero_crossing) — Wikipedia, on zero-crossing instants for frequency/timing estimation; see also [Frequency-shift keying](https://en.wikipedia.org/wiki/Frequency-shift_keying) for the demod context.
