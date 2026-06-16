---
slug: software-defined-radio
title: Software-defined radio (SDR)
entry_type: technology
category: sdr-dsp
description: Software-defined radio (SDR) is radio technology in which traditionally hardware functions — tuning, filtering, demodulation — are implemented in software operating on digitised IQ samples.
keywords: software defined radio, SDR, IQ, digital radio, RTL-SDR, flexibility
aka: [software-defined radio, SDR]
autolink: true
infobox:
  - { label: Type, value: Radio architecture }
  - { label: Idea, value: Move tuning/demod into software }
  - { label: Hardware emits, value: IQ samples }
  - { label: Examples, value: RTL-SDR, HackRF, Airspy }
see_also: [iq-data, analog-to-digital-converter, superheterodyne-receiver, rtl-sdr, demodulation]
related_lessons:
  - { title: "What is software-defined radio?", url: /learn/what-is-sdr/ }
external:
  - { title: "Software-defined radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Software-defined_radio }
---

**Software-defined radio** (**SDR**) moves the functions that were once fixed hardware —
tuning, filtering, [demodulation](/reference/demodulation/) — into **software** operating
on digitised [IQ samples](/reference/iq-data/). The hardware does only enough to convert a
slice of spectrum into numbers.

## How it works

An SDR front-end amplifies, mixes, and digitises a band into IQ; software then does
everything else. Because the differences between systems live in code, one device can
decode many protocols.

## Relevance to SDR

GopherTrunk is the software half of an SDR, specialised for digital trunked radio. The
hardware (e.g. [RTL-SDR](/reference/rtl-sdr/)) is almost interchangeable.
