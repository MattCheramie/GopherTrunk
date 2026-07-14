---
slug: digital-to-analog-converter
title: Digital-to-analog converter (DAC)
entry_type: hardware
category: hw-microcontrollers
description: A digital-to-analog converter turns a stream of numbers into a continuous analog voltage, the inverse of an ADC; it lets a microcontroller output audio, control voltages, or waveform signals.
keywords: DAC, digital to analog converter, analog output, resolution, audio, waveform, reconstruction, R-2R
aka: [DAC]
infobox:
  - { label: Type, value: Conversion device }
  - { label: Does, value: Numbers to analog voltage }
  - { label: Inverse of, value: ADC }
  - { label: Key spec, value: Resolution (bits), rate }
see_also: [analog-to-digital-converter, pulse-width-modulation, microcontroller, sensor, gpio, stm32]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Digital-to-analog_converter
---

**A digital-to-analog converter (DAC)** turns a sequence of digital numbers into a continuous analog voltage — the inverse of an [analog-to-digital converter](/reference/analog-to-digital-converter/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 168" role="img" aria-label="A signal path: a stream of digital sample values feeds a DAC, which produces a stepped staircase voltage; a low-pass filter then smooths that staircase into a continuous analog waveform." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <rect x="14" y="58" width="52" height="48" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/>
    <text x="40" y="74">80</text><text x="40" y="87">110</text><text x="40" y="100">95</text>
    <text x="40" y="122" font-size="8">samples</text>
    <rect x="80" y="64" width="52" height="36" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="106" y="86" font-size="9" font-weight="600">DAC</text>
    <rect x="286" y="64" width="66" height="36" rx="4" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="319" y="82" font-size="8.5" font-weight="600">low-pass</text><text x="319" y="93" font-size="7">filter</text>
  </g>
  <polyline points="150,116 165,116 165,98 180,98 180,70 195,70 195,52 210,52 210,56 225,56 225,78 240,78 240,104 255,104 255,118 270,118" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="210" y="138" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">staircase</text>
  <path d="M372 116 C 398 44 426 44 452 116" fill="none" stroke="currentColor" stroke-width="1.7"/>
  <text x="412" y="138" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.9">smooth analog</text>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <line x1="66" y1="82" x2="80" y2="82" marker-end="url(#dac_ar)"/>
    <line x1="132" y1="82" x2="150" y2="82" marker-end="url(#dac_ar)"/>
    <line x1="270" y1="82" x2="286" y2="82" marker-end="url(#dac_ar)"/>
    <line x1="352" y1="82" x2="372" y2="82" marker-end="url(#dac_ar)"/>
  </g>
  <defs><marker id="dac_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Feed the DAC a stream of numbers at a fixed rate and it holds each as a proportional voltage — a stepped staircase. A low-pass filter rounds those steps into a smooth continuous wave. Resolution (bits) sets how fine the steps are; update rate sets how fast new samples arrive.</figcaption>
</figure>

## Overview

A DAC takes a binary value and produces a proportional voltage on its output; feeding it a stream of values at a fixed rate reconstructs a waveform. Its two headline specs mirror the ADC's: **resolution** (bits, which sets how finely the output is quantized) and **update rate** (how fast new samples are accepted). Some [microcontrollers](/reference/microcontroller/) — such as parts of the [STM32](/reference/stm32/) line — include a true on-chip DAC; others approximate one by low-pass filtering [PWM](/reference/pulse-width-modulation/).

## What it's for

A DAC lets an MCU output genuinely analog signals: audio, control voltages for analog circuits, programmable reference levels, or arbitrary waveforms. It pairs naturally with [sensors](/reference/sensor/) and actuators that expect an analog input. Where only a rough analog level is needed, filtered PWM on an ordinary [GPIO](/reference/gpio/) pin is the cheaper substitute; a dedicated DAC wins on clean, fast, high-resolution output.

## Sources

[^wiki]: [Digital-to-analog converter](https://en.wikipedia.org/wiki/Digital-to-analog_converter) — Wikipedia, on DAC operation and specifications.
