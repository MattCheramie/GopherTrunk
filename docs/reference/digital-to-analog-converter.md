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

## Overview

A DAC takes a binary value and produces a proportional voltage on its output; feeding it a stream of values at a fixed rate reconstructs a waveform. Its two headline specs mirror the ADC's: **resolution** (bits, which sets how finely the output is quantized) and **update rate** (how fast new samples are accepted). Some [microcontrollers](/reference/microcontroller/) — such as parts of the [STM32](/reference/stm32/) line — include a true on-chip DAC; others approximate one by low-pass filtering [PWM](/reference/pulse-width-modulation/).

## What it's for

A DAC lets an MCU output genuinely analog signals: audio, control voltages for analog circuits, programmable reference levels, or arbitrary waveforms. It pairs naturally with [sensors](/reference/sensor/) and actuators that expect an analog input. Where only a rough analog level is needed, filtered PWM on an ordinary [GPIO](/reference/gpio/) pin is the cheaper substitute; a dedicated DAC wins on clean, fast, high-resolution output.

## Sources

[^wiki]: [Digital-to-analog converter](https://en.wikipedia.org/wiki/Digital-to-analog_converter) — Wikipedia, on DAC operation and specifications.
