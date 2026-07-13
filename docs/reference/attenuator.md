---
slug: attenuator
title: Attenuator
entry_type: hardware
category: rf-front-end
description: "An attenuator is a passive network that reduces signal power by a fixed or adjustable number of decibels, used to prevent overload and protect a receiver's ADC."
keywords: attenuator, RF attenuator, pad, dB pad, fixed attenuator, step attenuator, variable attenuator, overload protection, ADC protection, matched pad
aka: [attenuator, RF attenuator, pad]
autolink: true
infobox:
  - { label: Type, value: "Passive RF network" }
  - { label: Function, value: "Reduce signal power by a set dB" }
  - { label: Topologies, value: "Pi, T, bridged-T" }
  - { label: Common values, value: "3, 6, 10, 20, 30 dB" }
see_also: [decibel, dynamic-range, analog-to-digital-converter, standing-wave-ratio, dbm]
cite_urls:
  - https://en.wikipedia.org/wiki/Attenuator_(electronics)
  - https://en.wikipedia.org/wiki/Decibel
---

An **attenuator** (or **pad**) is a passive network that **reduces** a signal's power by a
fixed or adjustable number of [decibels](/reference/decibel/) while keeping the source and
load impedance-matched.[^wiki] Where an amplifier adds gain, an attenuator deliberately throws
gain away — most often to stop a strong signal from overloading a sensitive stage such as a
receiver's [analog-to-digital converter](/reference/analog-to-digital-converter/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A resistive Pi-network attenuator with series and shunt resistors between an input and output, reducing signal amplitude by a fixed number of decibels while staying matched to 50 ohms." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <line x1="40" y1="45" x2="150" y2="45"/>
    <rect x="150" y="38" width="70" height="14" fill="none"/>
    <line x1="220" y1="45" x2="420" y2="45"/>
    <line x1="150" y1="45" x2="150" y2="70"/><rect x="143" y="70" width="14" height="34"/><line x1="150" y1="104" x2="150" y2="112"/>
    <line x1="220" y1="45" x2="220" y2="70"/><rect x="213" y="70" width="14" height="34"/><line x1="220" y1="104" x2="220" y2="112"/>
    <line x1="120" y1="112" x2="250" y2="112"/>
  </g>
  <g font-size="9" fill="currentColor">
    <text x="60" y="36">in</text>
    <text x="390" y="36">out (−N dB)</text>
    <text x="240" y="30">series R</text>
    <text x="120" y="95">shunt R</text>
  </g>
</svg>
<figcaption>A resistive Pi pad drops the signal by a set number of decibels while presenting a matched impedance at both ports.</figcaption>
</figure>

## Overview

An attenuator is defined by three things: its **attenuation** in dB, its **characteristic
impedance** (usually 50 Ω), and its **power handling**. Ideally it reduces every frequency by
the same amount (flat response) and stays well matched, so it can be dropped anywhere in a
50 Ω line without causing reflections. Because it is purely resistive, the "lost" power is
dissipated as heat inside the pad.

## How it works

A fixed pad is a small resistor network — typically a **Pi** (π) or **T** arrangement of three
resistors — chosen so that looking in from either port the impedance is still 50 Ω while a set
fraction of the power reaches the output. The dB value maps directly to a power ratio: 3 dB
halves the power, 10 dB cuts it to one-tenth, 20 dB to one-hundredth. See
[decibel](/reference/decibel/) for the ratio math. Because the network is symmetric and
matched, attenuators can be cascaded (a 10 dB and a 20 dB pad give 30 dB) and reversed.

## Variants

- **Fixed pads** — a single value (3, 6, 10, 20, 30 dB), the workhorse form.
- **Step attenuators** — switchable in discrete steps (e.g. 1 dB steps to 31 dB) for bench and
  automated use.
- **Variable / continuously adjustable** — including voltage-controlled and digital step
  attenuators used in automatic-gain-control loops.
- **Power attenuators** — heat-sinked units rated to absorb transmitter-level power.

## In practice — protecting the receiver

The most common SDR use is **overload protection**. A receiver has a finite
[dynamic range](/reference/dynamic-range/): its noise floor at the bottom and, at the top, the
point where its front end compresses or its ADC clips (full-scale, expressed in
[dBFS](/reference/dbfs/)). In a strong-signal environment a modest pad brings the whole
spectrum down out of the clipping region, trading a little sensitivity for freedom from the
[intermodulation](/reference/intermodulation/) spurs that overload creates. An attenuator is
also handy for calibrating levels, protecting a spectrum analyser input, and setting the
optimum drive into a mixer.

## Relevance to SDR

Attenuators are one of the simplest and most useful accessories for SDR reception. Many RTL-SDR
and other dongle users add a switchable pad or a step attenuator when listening near powerful
transmitters, because a receiver driven into clipping produces phantom signals across the band
that no amount of software can remove. Some SDRs include a built-in switchable attenuator or
a variable-gain front end that serves the same purpose under software control.

**GopherTrunk** contains no attenuator — it is software downstream of the ADC. But attenuation
choices upstream directly shape the samples it decodes: too little and strong-signal clipping
corrupts the I/Q stream; too much and weak signals sink into the noise floor. GT works best
when the front-end gain and any external pad are set so the wanted signal sits comfortably
inside the ADC's range.

## Sources

[^wiki]: [Attenuator (electronics)](https://en.wikipedia.org/wiki/Attenuator_(electronics)) — Wikipedia, on resistive pad topologies, dB values, matching, and power handling.
