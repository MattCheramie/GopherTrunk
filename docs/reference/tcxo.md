---
slug: tcxo
title: TCXO (temperature-compensated crystal oscillator)
entry_type: hardware
category: rf-front-end
description: "A TCXO is a crystal oscillator with built-in temperature compensation, holding a few ppm of frequency accuracy — the reference upgrade that lets SDRs tune reliably."
keywords: TCXO, temperature-compensated crystal oscillator, ppm, frequency accuracy, frequency reference, crystal oscillator, XO, SDR clock, RTL-SDR TCXO
aka: [TCXO, "temperature-compensated crystal oscillator"]
autolink: true
infobox:
  - { label: Type, value: "Compensated crystal oscillator" }
  - { label: Stability, value: "±0.5 to ±2 ppm typical" }
  - { label: Compensation, value: "Analog/digital vs temperature" }
  - { label: Power, value: "Low (milliwatts)" }
  - { label: TX, value: "N/A (reference)" }
see_also: [frequency-stability, ppm-frequency-correction, ocxo, local-oscillator, rtl-sdr]
cite_urls:
  - https://en.wikipedia.org/wiki/Crystal_oscillator#Temperature-compensated_crystal_oscillator
---

A **TCXO** (temperature-compensated crystal oscillator) is a quartz-crystal oscillator with
a compensation circuit that cancels the crystal's drift with temperature, holding its output
frequency to typically **±0.5 to ±2 [ppm](/reference/ppm-frequency-correction/)**.[^wiki] It
is the small but decisive upgrade over a plain crystal (XO) that gives a
software-defined radio a stable, predictable tuning [reference](/reference/local-oscillator/),
so channels land where they should and stay there as the board warms up. Ppm — parts per
million — is the natural unit here: 1 ppm at 100 MHz is 100 Hz of error.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="Two frequency-versus-temperature curves: a plain crystal drifts widely in an S-shape while a TCXO stays inside a narrow few-ppm band." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="55" y1="20" x2="55" y2="140"/><line x1="55" y1="80" x2="430" y2="80"/>
  </g>
  <path d="M60 45 C120 55 150 130 240 80 C330 30 360 105 425 118" fill="none" stroke="currentColor" stroke-width="1.6" stroke-dasharray="5 4"/>
  <path d="M60 74 C160 78 260 82 425 78" fill="none" stroke="currentColor" stroke-width="2.2"/>
  <g stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"><line x1="55" y1="70" x2="430" y2="70"/><line x1="55" y1="90" x2="430" y2="90"/></g>
  <g font-size="9" fill="currentColor">
    <text x="30" y="26">+ppm</text><text x="30" y="140">-ppm</text>
    <text x="245" y="158" text-anchor="middle">temperature &#8594;</text>
    <text x="150" y="40">plain crystal (drifts)</text>
    <text x="300" y="98">TCXO (compensated)</text>
  </g>
</svg>
<figcaption>A plain crystal's frequency swings in an S-curve with temperature; a TCXO's compensation flattens it into a narrow few-ppm band.</figcaption>
</figure>

## Overview

Every quartz crystal has a characteristic frequency-versus-temperature curve, usually a
gentle cubic (S-shaped) response. A plain oscillator follows that curve and can drift tens of
ppm across a room's temperature swing — enough to pull a narrowband channel off frequency and
break demodulation. A TCXO measures temperature and applies an equal-and-opposite correction
so the net output stays nearly flat, trading a modest cost and a little extra power for a
one- to two-order-of-magnitude improvement in [frequency
stability](/reference/frequency-stability/).

## How it works

- A temperature sensor (often a thermistor network) tracks the crystal's environment.
- A compensation network converts that reading into a small voltage that pulls a varactor,
  nudging the oscillator to counter the crystal's known drift. Analog TCXOs use a shaped
  resistor/thermistor network; digital (DCXO/MCXO) types store a correction curve in a lookup
  table.
- The result is a fixed reference frequency held to a few ppm from roughly -20 °C to +70 °C,
  where the raw crystal alone might wander 20–50 ppm.

## In practice

TCXOs do not fix a **fixed** offset — a small constant error and unit-to-unit variation
remains, which is why SDR software still applies a
[ppm correction](/reference/ppm-frequency-correction/) constant. What the TCXO buys is that
the offset stays *put*: once you calibrate the ppm value it does not wander as the dongle
heats up. They also warm up far faster and draw far less power than an
[OCXO](/reference/ocxo/), which achieves better stability by holding the crystal in a heated
oven at the cost of size, current, and warm-up time.

## Relevance to SDR

The TCXO is the single most visible reference upgrade in the SDR hobby: the difference
between a bargain [RTL-SDR](/reference/rtl-sdr/) that drifts as it warms and a "TCXO" version
that locks a P25 or DMR control channel and holds it. Because trunking decode depends on
tracking a control channel over minutes to hours, a stable reference directly determines
whether the receiver stays on frequency. GopherTrunk decodes the samples the radio delivers
and does not drive the oscillator, but it benefits directly from a TCXO-equipped front end:
less residual drift means fewer lost control-channel messages and less reliance on the
software's own automatic frequency correction. For the best absolute accuracy, operators step
up to an [OCXO](/reference/ocxo/) or a [GPSDO](/reference/gpsdo/).

## Sources

[^wiki]: [Temperature-compensated crystal oscillator](https://en.wikipedia.org/wiki/Crystal_oscillator#Temperature-compensated_crystal_oscillator) — Wikipedia, on TCXO compensation, ppm stability figures, and comparison to plain crystals and OCXOs.
