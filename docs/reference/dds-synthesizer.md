---
slug: dds-synthesizer
title: DDS synthesizer (direct digital synthesis)
entry_type: hardware
category: rf-front-end
description: "A DDS synthesizer builds an analog waveform digitally from a phase accumulator, NCO, and DAC, giving fine, fast, phase-continuous frequency control for RF sources and SDR clocks."
keywords: DDS, direct digital synthesis, DDS synthesizer, phase accumulator, NCO, numerically controlled oscillator, DAC, AD9850, AD9910, signal generator, frequency tuning word
aka: [DDS, "direct digital synthesis", "DDS synthesizer"]
autolink: true
infobox:
  - { label: Type, value: "Digital frequency synthesizer" }
  - { label: Core, value: "Phase accumulator + NCO + DAC" }
  - { label: Resolution, value: "Sub-Hz tuning steps" }
  - { label: Output, value: "Up to ~0.4x clock (Nyquist)" }
  - { label: TX, value: "Source only (low level)" }
see_also: [numerically-controlled-oscillator, local-oscillator, digital-to-analog-converter, phase-locked-loop, signal-generator]
cite_urls:
  - https://en.wikipedia.org/wiki/Direct_digital_synthesis
---

A **DDS** (direct digital synthesis) synthesizer builds an analog output waveform
**digitally**, computing successive sample values from a phase counter and converting them to
voltage with a [DAC](/reference/digital-to-analog-converter/).[^wiki] Its heart is a
[numerically controlled oscillator](/reference/numerically-controlled-oscillator/): a phase
accumulator advances by a programmable step each clock, a lookup table turns phase into a sine
amplitude, and the DAC renders it. The payoff is **very fine, very fast, phase-continuous**
frequency control from a single fixed clock — the reason DDS chips sit at the tuning heart of
signal generators and many local-oscillator and clock circuits.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A DDS signal chain: a frequency tuning word feeds a phase accumulator that drives a sine lookup table, then a DAC and a low-pass filter produce the analog output." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="ddsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="42" y="60" font-size="8" fill="currentColor" text-anchor="middle">tuning</text>
  <text x="42" y="70" font-size="8" fill="currentColor" text-anchor="middle">word</text>
  <line x1="65" y1="65" x2="90" y2="65" stroke="currentColor" stroke-width="1.6" marker-end="url(#ddsar)"/>
  <rect x="90" y="45" width="80" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="130" y="62" font-size="8" fill="currentColor" text-anchor="middle">phase</text>
  <text x="130" y="74" font-size="8" fill="currentColor" text-anchor="middle">accumulator</text>
  <line x1="170" y1="65" x2="200" y2="65" stroke="currentColor" stroke-width="1.6" marker-end="url(#ddsar)"/>
  <rect x="200" y="45" width="70" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="235" y="62" font-size="8" fill="currentColor" text-anchor="middle">sine</text>
  <text x="235" y="74" font-size="8" fill="currentColor" text-anchor="middle">LUT</text>
  <line x1="270" y1="65" x2="300" y2="65" stroke="currentColor" stroke-width="1.6" marker-end="url(#ddsar)"/>
  <rect x="300" y="45" width="50" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="325" y="70" font-size="9" fill="currentColor" text-anchor="middle">DAC</text>
  <line x1="350" y1="65" x2="380" y2="65" stroke="currentColor" stroke-width="1.6" marker-end="url(#ddsar)"/>
  <rect x="380" y="45" width="55" height="40" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="407" y="70" font-size="8" fill="currentColor" text-anchor="middle">LPF</text>
  <text x="130" y="112" font-size="8" fill="currentColor" text-anchor="middle">&#8593; fixed reference clock feeds accumulator</text>
</svg>
<figcaption>A DDS steps a phase accumulator by a tuning word each clock, maps phase to a sine amplitude, and renders it through a DAC and reconstruction filter.</figcaption>
</figure>

## Overview

Where a [phase-locked loop](/reference/phase-locked-loop/) multiplies a reference up to an
output frequency with analog feedback, a DDS *computes* the waveform outright. This gives it
frequency resolution set purely by the accumulator width: a 32-bit accumulator clocked at
100 MHz resolves steps of about 0.023 Hz. Because a new tuning word takes effect on the next
clock and the phase simply keeps accumulating, frequency changes are near-instant and
**phase-continuous** — no glitch, no settling ring like a PLL relock.

## How it works

- A **frequency tuning word** M sets how far the phase accumulator advances each clock cycle.
- The accumulator's overflow rate is the output frequency: f_out = M · f_clk / 2^N, where N is
  the accumulator width. Larger M means faster phase rotation and a higher frequency.
- A sine lookup table (and often phase truncation) maps the high bits of the accumulated phase
  to an amplitude, the DAC converts it to a voltage, and a reconstruction low-pass filter
  smooths the DAC steps.
- Usable output is limited by Nyquist to below half the clock, and in practice to roughly
  40% of it, above which images and DAC images become hard to filter.

## In practice

DDS shines at **fine resolution, fast hop, and clean phase**, but it is not free of spurs:
phase truncation, the DAC's finite resolution, and its images inject spurious tones (spurs)
that must be filtered and planned around. Its top frequency is bounded by the DAC clock, so
DDS is often used as a fine-tuning **reference** that a PLL then multiplies up to reach GHz
outputs — a hybrid that combines DDS resolution with PLL reach. Common parts include the
Analog Devices AD9850/AD9851 (hobby signal sources) and the higher-speed AD991x family.

## Relevance to SDR

DDS is the tunable core of bench [signal generators](/reference/signal-generator/), antenna
analysers, and the programmable local-oscillator/clock stages in many radios and test jigs —
anywhere a clean, precisely settable frequency must change quickly under software control. In
receivers, the same phase-accumulator idea reappears in software as the
[NCO](/reference/numerically-controlled-oscillator/) that a digital down-converter uses to
tune a channel. GopherTrunk performs exactly that kind of tuning in software when it mixes a
captured band down to a channel, so it relies on the NCO concept a hardware DDS embodies,
while the DDS chips themselves live in the RF hardware and test equipment in front of the
receiver rather than in the decode code.

## Sources

[^wiki]: [Direct digital synthesis](https://en.wikipedia.org/wiki/Direct_digital_synthesis) — Wikipedia, on the phase-accumulator/DAC architecture, tuning-word frequency relation, resolution, spurs, and Nyquist limits.
