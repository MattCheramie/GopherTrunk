---
slug: mixer-rf
title: Mixer (RF)
entry_type: hardware
category: rf-front-end
description: "An RF mixer multiplies two signals to shift a frequency up or down, producing sum and difference products used for frequency conversion in every superheterodyne receiver."
keywords: RF mixer, frequency mixer, frequency conversion, heterodyne, downconversion, upconversion, image frequency, IF, LO, double-balanced mixer, Gilbert cell
aka: [mixer, RF mixer, frequency mixer]
autolink: true
infobox:
  - { label: Type, value: "Frequency-conversion device" }
  - { label: Function, value: "Multiplies RF and LO to shift frequency" }
  - { label: Products, value: "Sum and difference (f_RF ± f_LO)" }
  - { label: Key spec, value: "Conversion loss, IP3, isolation" }
see_also: [superheterodyne-receiver, image-frequency, local-oscillator, intermodulation, intermediate-frequency]
cite_urls:
  - https://en.wikipedia.org/wiki/Frequency_mixer
  - https://en.wikipedia.org/wiki/Heterodyne
---

A **mixer** is a three-port nonlinear or switching device that **multiplies** an incoming
radio-frequency signal by a [local-oscillator](/reference/local-oscillator/) tone to shift
it to a new frequency.[^wiki] Multiplying two sinusoids produces components at the **sum
and difference** of their frequencies, so a mixer converts a signal up or down in frequency
without changing the information it carries — the heart of every
[superheterodyne receiver](/reference/superheterodyne-receiver/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An RF signal and a local-oscillator tone enter a mixer, which outputs sum and difference frequencies; the difference becomes the intermediate frequency." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="mxrar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <circle cx="200" cy="70" r="26" fill="none" stroke="currentColor" stroke-width="1.4"/>
    <line x1="182" y1="52" x2="218" y2="88" stroke="currentColor" stroke-width="1.3"/>
    <line x1="218" y1="52" x2="182" y2="88" stroke="currentColor" stroke-width="1.3"/>
    <line x1="60" y1="70" x2="172" y2="70" stroke="currentColor" stroke-width="1.2" marker-end="url(#mxrar)"/>
    <text x="95" y="61">RF (f_RF)</text>
    <line x1="200" y1="135" x2="200" y2="100" stroke="currentColor" stroke-width="1.2" marker-end="url(#mxrar)"/>
    <text x="200" y="148">LO (f_LO)</text>
    <line x1="228" y1="70" x2="400" y2="70" stroke="currentColor" stroke-width="1.2" marker-end="url(#mxrar)"/>
    <text x="320" y="55" font-size="8">f_RF − f_LO  (IF, kept)</text>
    <text x="320" y="86" font-size="8">f_RF + f_LO  (filtered out)</text>
  </g>
</svg>
<figcaption>A mixer multiplies RF by the LO, producing sum and difference frequencies; a filter keeps one (the intermediate frequency) and rejects the other.</figcaption>
</figure>

## Overview

Mixers realise the *heterodyne* principle: combining two frequencies to create new ones at
their sum and difference. In a receiver the difference product is usually kept as the
[intermediate frequency](/reference/intermediate-frequency/) (downconversion); in a
transmitter the mixer moves baseband up to the carrier (upconversion). Because the process
is linear in the information but nonlinear in frequency, the modulation on the RF signal is
transferred intact to the new frequency.

## How it works

Multiplying two cosines uses the identity
cos(A)·cos(B) = ½[cos(A−B) + cos(A+B)], which is exactly the sum-and-difference behaviour a
mixer exploits. Real mixers approximate multiplication in one of several ways:

- **Diode (passive) mixers** switch one or more diodes on and off at the LO rate. A
  single-diode mixer is simple but leaks the LO and RF to the output; **balanced** and
  **double-balanced** designs (e.g. a diode ring) cancel those leakage terms, giving good
  **port-to-port isolation** and rejecting even-order products.
- **Active mixers** such as the **Gilbert cell** use transistors to both multiply and add
  gain, so they can have *conversion gain* instead of the ~6–7 dB *conversion loss* typical
  of passive diode mixers.

Because mixing keeps both the sum and difference, a filter after the mixer selects the wanted
product. The unwanted one — plus the LO feedthrough — must be suppressed.

## Variants

- **Downconverting vs upconverting** — the same device, distinguished by whether the wanted
  output is below or above the input.
- **Single-balanced / double-balanced** — increasing port isolation and spur suppression.
- **Image-reject and I/Q mixers** — pairs of mixers fed 90° apart cancel the image response
  in hardware, the basis of quadrature and zero-IF architectures.
- **Subharmonic mixers** — driven at half the LO frequency, useful at high frequencies.

## Relevance to SDR

Every superheterodyne front end contains at least one mixer, and it is the source of two
classic problems SDR users must manage. First, the **image frequency**: a signal spaced from
the LO by the same difference as the wanted signal — but on the opposite side — also lands on
the IF, so it must be rejected by a preselector filter or an image-reject/quadrature design.
See [image frequency](/reference/image-frequency/). Second, because mixers are inherently
nonlinear, strong signals generate **[intermodulation](/reference/intermodulation/)**
products and spurs; a mixer's third-order intercept point sets how gracefully it handles
crowded bands.

Most consumer SDRs — RTL-SDR tuners, SDRplay, Airspy — implement mixing inside their tuner
chips (often as quadrature/zero-IF mixers), so the "mixer" is a block on a die rather than a
part you handle. GopherTrunk operates entirely on the digital I/Q stream those chips produce;
it performs further frequency translation in software with a
[numerically controlled oscillator](/reference/numerically-controlled-oscillator/) and a
digital down-converter — mathematically the same multiply-and-filter operation, done in DSP
rather than in a physical mixer.

## Sources

[^wiki]: [Frequency mixer](https://en.wikipedia.org/wiki/Frequency_mixer) — Wikipedia, on mixer operation, balanced topologies, conversion loss, and image response.
