---
slug: frequency-bands
title: Frequency bands (HF/VHF/UHF)
entry_type: term
category: rf-fundamentals
description: Frequency bands are conventional divisions of the radio spectrum — VLF, LF, MF, HF, VHF, UHF, SHF — each with characteristic propagation and uses.
keywords: frequency bands, HF, VHF, UHF, SHF, band plan, spectrum allocation
aka: [VHF, UHF, SHF]
autolink: true
infobox:
  - { label: Type, value: Spectrum divisions }
  - { label: HF, value: "3–30 MHz" }
  - { label: VHF, value: "30–300 MHz" }
  - { label: UHF, value: "300 MHz – 3 GHz" }
see_also: [electromagnetic-spectrum, frequency, radio-propagation, ionospheric-propagation]
related_lessons:
  - { title: "Frequency, bands & the spectrum", url: /learn/frequency-and-spectrum/ }
external:
  - { title: "Radio spectrum (Wikipedia)", url: https://en.wikipedia.org/wiki/Radio_spectrum }
---

**Frequency bands** are the conventional decade-wide divisions of the radio
[spectrum](/reference/electromagnetic-spectrum/): VLF, LF, MF, **HF** (3–30 MHz),
**VHF** (30–300 MHz), **UHF** (300 MHz–3 GHz), and SHF (3–30 GHz). Each behaves
differently and is allocated to different uses.

## How it works

HF can refract off the [ionosphere](/reference/ionospheric-propagation/) for worldwide
"skip", while VHF and UHF are essentially line-of-sight. Regulators publish band plans
assigning slices to broadcast, aviation, marine, public safety, and amateur use.

## Relevance to SDR

Most trunked-radio scanning happens in VHF, UHF, and the 700/800 MHz bands. Matching
an SDR's tuning range to the target band is the first hardware decision.
