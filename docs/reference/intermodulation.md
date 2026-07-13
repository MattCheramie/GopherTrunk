---
slug: intermodulation
title: Intermodulation Distortion (IMD)
entry_type: term
category: rf-fundamentals
description: Intermodulation distortion is the creation of spurious mixing products when two or more signals pass through a nonlinear device; third-order products fall near the originals.
keywords: intermodulation, intermodulation distortion, IMD, third-order, IM3, mixing products, nonlinearity, two-tone, spurious
aka: [IMD, intermod, IM products]
autolink: true
infobox:
  - { label: Type, value: "nonlinear distortion" }
  - { label: Worst products, value: "3rd-order (2f1−f2, 2f2−f1)" }
  - { label: Metric, value: "third-order intercept (IP3)" }
see_also: [third-order-intercept, harmonics, mixer-rf, 1-db-compression-point, desensitization, spurious-free-dynamic-range]
cite_urls:
  - https://en.wikipedia.org/wiki/Intermodulation
  - https://en.wikipedia.org/wiki/Third-order_intercept_point
---

**Intermodulation distortion (IMD)** is the generation of unwanted new frequencies
when two or more signals pass together through a device that is not perfectly
linear.[^wiki] Unlike [harmonics](/reference/harmonics/), which are integer multiples
of a single tone, intermod products are sums and differences of *multiples* of two
or more input tones. The most troublesome are the **third-order** products at
**2f₁ − f₂** and **2f₂ − f₁**, because they land right next to the original signals
and so cannot be filtered away. IMD is the fundamental reason a receiver can only
handle a limited range of signal strengths at once.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A spectrum showing two strong input tones f1 and f2 close together, flanked by smaller third-order intermodulation products at 2f1 minus f2 and 2f2 minus f1 that fall just outside the pair." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="imar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="20" x2="30" y2="135" stroke="currentColor" stroke-width="1" marker-end="url(#imar)"/>
  <line x1="30" y1="135" x2="445" y2="135" stroke="currentColor" stroke-width="1" marker-end="url(#imar)"/>
  <text x="235" y="152" font-size="9" text-anchor="middle" fill="currentColor">frequency</text>
  <line x1="190" y1="40" x2="190" y2="135" stroke="currentColor" stroke-width="2.5"/>
  <text x="190" y="34" font-size="9" text-anchor="middle" fill="currentColor">f1</text>
  <line x1="270" y1="40" x2="270" y2="135" stroke="currentColor" stroke-width="2.5"/>
  <text x="270" y="34" font-size="9" text-anchor="middle" fill="currentColor">f2</text>
  <line x1="110" y1="100" x2="110" y2="135" stroke="currentColor" stroke-width="2"/>
  <text x="110" y="94" font-size="8" text-anchor="middle" fill="currentColor">2f1−f2</text>
  <line x1="350" y1="100" x2="350" y2="135" stroke="currentColor" stroke-width="2"/>
  <text x="350" y="94" font-size="8" text-anchor="middle" fill="currentColor">2f2−f1</text>
</svg>
<figcaption>Two strong tones through a nonlinear stage breed third-order products at 2f1−f2 and 2f2−f1 — spurs that fall right beside the originals, inside the passband.</figcaption>
</figure>

## How it works

A perfectly linear device only scales and delays its input, so its output contains no
new frequencies. Real amplifiers and [mixers](/reference/mixer-rf/) have a slightly
curved transfer characteristic that can be expanded as a polynomial:
*Vout = a₁V + a₂V² + a₃V³ + …*. Feed in two tones at f₁ and f₂ and each nonlinear
term multiplies them together, producing sums and differences of their harmonics.

The order of a product is the sum of the multiplier coefficients. Second-order
products (f₁ + f₂, f₁ − f₂) usually fall far from the originals and are easy to
filter. The dangerous ones are the **third-order** products from the *V³* term at
**2f₁ − f₂** and **2f₂ − f₁**: if f₁ and f₂ are close, these spurs sit just outside
the pair, squarely inside the receiver's passband, masquerading as real signals. A
defining behaviour follows from the cubic term: a third-order product grows *three
decibels for every one decibel* that the input tones rise. This 3:1 slope is what
makes intermod explode as signal levels climb, and it is the basis of the
[third-order intercept point](/reference/third-order-intercept/) (IP3), the
extrapolated level where the third-order product would notionally equal the wanted
signal — a single figure of merit for a stage's linearity.

## In practice

Intermodulation is a symptom of driving a stage too hard, so it is tightly linked to
the [1 dB compression point](/reference/1-db-compression-point/): both describe the
onset of nonlinearity, and IP3 typically sits about 10–15 dB above the 1 dB
compression point. Higher IP3 means a device tolerates stronger signals before
intermod becomes objectionable. The usable window between the noise floor and the
level where third-order products emerge is the
[spurious-free dynamic range](/reference/spurious-free-dynamic-range/).

IMD is also a real-world nuisance beyond a single receiver. Nearby transmitters can
mix in any shared nonlinearity — a corroded connector, a rusty tower joint, even a
diode-like oxide layer (the "rusty bolt" effect) — producing intermod that radiates
and interferes on frequencies where nothing is actually transmitting. Passive
intermodulation at antenna sites is a recurring source of hard-to-trace interference.

## Relevance to SDR

An SDR's front end is a chain of nonlinear stages — [low-noise
amplifier](/reference/low-noise-amplifier/), [mixer](/reference/mixer-rf/), and ADC —
each with finite linearity. In a busy RF environment, strong out-of-band signals
(pagers, broadcast FM, cellular) can drive these into producing third-order products
that appear as phantom carriers across the tuned band, or that
[desensitize](/reference/desensitization/) the receiver by raising its effective
noise. Wideband direct-sampling SDRs are especially exposed because they present the
whole spectrum to the ADC at once, so front-end filtering and attenuation are the
usual defences. IP3 and the 1 dB compression point are the datasheet numbers that
predict how a given SDR will cope.

GopherTrunk operates on the IQ samples after this front end, so intermod that has
already occurred in the analog chain is baked into the samples it decodes — the
software cannot remove a spur that looks like a legitimate signal. The project's own
DSP notes reinforce this: symptoms that appear only at higher capture rates and
reproduce in offline replay point at front-end overload or intermod in the *captured
data*, not at GopherTrunk's steady-state processing. Recognising intermod is
therefore a diagnostic skill for GopherTrunk operators: phantom control channels or
elevated noise that vanish when an attenuator is added are the classic signature.

## Sources

[^wiki]: [Intermodulation](https://en.wikipedia.org/wiki/Intermodulation) — Wikipedia, the nonlinear-mixing definition, product orders, and the 2f1−f2 / 2f2−f1 third-order products.
