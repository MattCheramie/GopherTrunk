---
slug: 1-db-compression-point
title: 1 dB compression point (P1dB)
entry_type: term
category: rf-metrics
description: The 1 dB compression point is the input or output power where an amplifier's gain has dropped 1 dB from linear, marking the onset of saturation and overload.
keywords: 1 dB compression point, P1dB, gain compression, saturation, compression point, amplifier overload, input P1dB, output P1dB, blocking
aka: [P1dB, 1-dB compression point, gain compression point]
autolink: true
infobox:
  - { label: Symbol, value: "P1dB (IP1dB / OP1dB)" }
  - { label: Unit, value: dBm }
  - { label: Marks, value: onset of gain saturation }
see_also: [third-order-intercept, dynamic-range, blocking-dynamic-range, desensitization, power-amplifier, low-noise-amplifier]
cite_urls:
  - https://en.wikipedia.org/wiki/Gain_compression
  - https://en.wikipedia.org/wiki/1_dB_compression_point
---

**1 dB compression point** (**P1dB**) is the power level at which an amplifier's gain
has fallen **1 dB below** its ideal linear value.[^wiki] Below it the device is
essentially linear — output tracks input dB for dB; at P1dB the amplifier is starting
to saturate and can no longer keep up, so P1dB marks the practical top of a device's
linear operating range and the onset of **overload**. It complements the
[third-order intercept](/reference/third-order-intercept/): where IP3 predicts
intermodulation spurs, P1dB pins down single-signal gain compression and the
[blocking dynamic range](/reference/blocking-dynamic-range/) ceiling.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A plot of output power versus input power where the actual gain curve tracks an ideal straight line at low power then bends over, the 1 dB compression point marked where the curve falls one decibel below the ideal line." xmlns="http://www.w3.org/2000/svg">
  <line x1="55" y1="18" x2="55" y2="158" stroke="currentColor" stroke-width="1.3"/>
  <line x1="55" y1="158" x2="440" y2="158" stroke="currentColor" stroke-width="1.3"/>
  <text x="48" y="16" text-anchor="end" font-size="9" fill="currentColor">P_out</text>
  <text x="438" y="174" text-anchor="end" font-size="9" fill="currentColor">P_in</text>
  <line x1="55" y1="150" x2="360" y2="28" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/>
  <text x="300" y="40" font-size="9" fill="currentColor">ideal linear</text>
  <path d="M55 150 L250 72 Q310 48 400 44" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <text x="360" y="60" font-size="9" fill="currentColor">actual gain</text>
  <circle cx="272" cy="72" r="4" fill="currentColor"/>
  <line x1="272" y1="72" x2="272" y2="63" stroke="currentColor" stroke-width="1"/>
  <text x="278" y="88" font-size="9.5" fill="currentColor">1 dB down = P1dB</text>
  <line x1="272" y1="72" x2="272" y2="158" stroke="currentColor" stroke-width="0.9" stroke-dasharray="2 2"/>
  <text x="272" y="172" text-anchor="middle" font-size="9" fill="currentColor">IP1dB</text>
</svg>
<figcaption>The gain curve tracks the ideal line at low power, then bends; the 1 dB compression point is where actual output falls one decibel short of linear — the practical edge of clean operation.</figcaption>
</figure>

## How it works

Every amplifier has a finite supply voltage and current, so the output cannot swing
past a hard limit. As input power climbs, the output approaches that ceiling and the
incremental gain shrinks — **gain compression**. P1dB is a simple, agreed-upon
threshold for "meaningfully compressed": the input (or output) power at which the
measured gain is exactly 1 dB less than the small-signal gain.

- **Input-referred (IP1dB)** describes how strong a signal the device can accept.
- **Output-referred (OP1dB)** describes how much clean power it can deliver; the two
  differ by the compressed gain (OP1dB ≈ IP1dB + G − 1 dB).

Beyond P1dB the device continues into hard saturation, where output barely rises with
input and the signal is grossly distorted. For a receiver, driving the front end past
compression is *overload*: gain collapses, harmonics and intermod explode, and a
strong signal can **desensitize** the radio to everything else — see
[desensitization](/reference/desensitization/). P1dB, together with the noise floor,
therefore brackets the usable [dynamic range](/reference/dynamic-range/).

## In practice

For a receive [low-noise amplifier](/reference/low-noise-amplifier/), a high P1dB
means it stays linear even when a strong local transmitter appears — essential for
scanning near paging or broadcast sites. For a transmit
[power amplifier](/reference/power-amplifier/), P1dB (and its output-referred form)
roughly marks the top of clean, low-distortion output; digital modes with high
[crest factor](/reference/crest-factor-papr/) must be **backed off** several dB below
P1dB to keep their peaks out of compression and preserve
[EVM](/reference/error-vector-magnitude/) and spectral cleanliness. Engineers also use
P1dB as a quick linearity sanity check, since it needs only a single-tone sweep,
whereas IP3 requires a two-tone setup.

## Relevance to SDR

For SDR reception, P1dB is really a statement about the front end ahead of the
[analog-to-digital converter](/reference/analog-to-digital-converter/) — the LNA,
mixer, and IF chain. Set gain too high and a strong nearby signal pushes some stage
(often the ADC's full-scale, an equivalent compression) into overload, wiping out the
weak channels you actually wanted. This is the mechanism behind the common advice to
*reduce gain* or add an [attenuator](/reference/attenuator/) or front-end filter when
a scanner degrades in a strong-signal environment: you are keeping the whole chain
comfortably below its compression point.

GopherTrunk sees only the digitized result. Once a stage has compressed and clipped,
the distortion is in the samples and no decoder can remove it; a signal that decodes
in isolation but fails when a strong carrier is present is compression/overload
limited, fixed by gain staging and filtering upstream, not by software.

## Sources

[^wiki]: [Gain compression](https://en.wikipedia.org/wiki/Gain_compression) — Wikipedia, definition of the 1 dB compression point and its role in amplifier saturation.
