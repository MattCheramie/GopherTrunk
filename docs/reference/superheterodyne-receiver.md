---
slug: superheterodyne-receiver
title: Superheterodyne receiver
entry_type: term
category: sdr-dsp
description: A superheterodyne receiver uses a mixer and local oscillator to shift a desired band down to a fixed intermediate frequency for easier filtering and detection; SDR front-ends apply the same principle.
keywords: superheterodyne, superhet, mixer, local oscillator, intermediate frequency, IF, downconversion
aka: [superheterodyne receiver, superhet]
autolink: true
infobox:
  - { label: Type, value: Receiver architecture }
  - { label: Key parts, value: Mixer + local oscillator }
  - { label: Produces, value: Intermediate frequency / baseband }
see_also: [local-oscillator, digital-down-converter, analog-to-digital-converter, software-defined-radio]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "Superheterodyne receiver (Wikipedia)", url: https://en.wikipedia.org/wiki/Superheterodyne_receiver }
---

A **superheterodyne receiver** uses a mixer driven by a
[local oscillator](/reference/local-oscillator/) to shift a chosen band **down** to a
fixed intermediate frequency (IF) — or to baseband — where it is easier to filter and
detect.

## How it works

Tuning is just changing the local-oscillator frequency so the wanted band lands at the
fixed IF. SDR quadrature front-ends apply the same idea, mixing to baseband as
[IQ](/reference/iq-data/) before the [ADC](/reference/analog-to-digital-converter/).

## Relevance to SDR

Understanding the mixer/LO explains why "tuning" an SDR is simply setting a number, and
how a [digital down-converter](/reference/digital-down-converter/) does it in software.
