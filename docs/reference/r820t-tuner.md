---
slug: r820t-tuner
title: R820T / R820T2 tuner
entry_type: hardware
category: hardware
description: The Rafael Micro R820T and R820T2 are the most common tuner chips paired with the RTL2832U in RTL-SDR dongles, providing the RF front-end and mixer up to ~1.7 GHz.
keywords: R820T, R820T2, R828D, Rafael Micro, tuner chip, RTL-SDR front end, mixer
aka: [R820T, R820T2]
autolink: true
infobox:
  - { label: Type, value: RF tuner IC }
  - { label: Vendor, value: Rafael Micro }
  - { label: Role, value: Front-end + mixer/LO }
  - { label: Range, value: ~24 MHz – 1.7 GHz }
see_also: [rtl-sdr, rtl2832u, superheterodyne-receiver, local-oscillator]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "RTL-SDR (Wikipedia)", url: https://en.wikipedia.org/wiki/Software-defined_radio#RTL-SDR }
---

The **R820T** and improved **R820T2** (and related R828D) from Rafael Micro are the most
common tuner chips paired with the [RTL2832U](/reference/rtl2832u/) in
[RTL-SDR](/reference/rtl-sdr/) dongles. They provide the RF front-end and
mixer/[local oscillator](/reference/local-oscillator/).

## Overview

The tuner amplifies and shifts the selected band down to a low frequency the RTL2832U
can digitise, covering roughly 24 MHz–1.7 GHz. Tuner quality affects sensitivity and
overload behaviour.

## Relevance to SDR

The tuner sets the dongle's frequency range and much of its noise performance, important
when chasing weak signals.
