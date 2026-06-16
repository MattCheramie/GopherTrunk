---
slug: polarization
title: Polarization
entry_type: term
category: antennas-propagation
description: Polarization is the orientation of a radio wave's electric field, set by the transmitting antenna; matching receive and transmit polarization avoids signal loss.
keywords: polarization, vertical, horizontal, circular, cross-polarization, electric field
aka: [polarization, polarisation]
autolink: true
infobox:
  - { label: Type, value: Wave/antenna property }
  - { label: Common kinds, value: Vertical, horizontal, circular }
  - { label: Mismatch loss, value: Up to ~20 dB }
see_also: [antenna, dipole-antenna, radio-propagation, antenna-gain]
related_lessons:
  - { title: "Antennas 101", url: /learn/antennas/ }
external:
  - { title: "Polarization (waves) (Wikipedia)", url: https://en.wikipedia.org/wiki/Polarization_(waves) }
---

**Polarization** is the orientation of a [radio wave](/reference/radio-wave/)'s
electric field, determined by how the transmitting [antenna](/reference/antenna/) is
mounted — vertical, horizontal, or circular.

## How it works

A receive antenna should match the transmitter's polarization; a full mismatch can
cost on the order of 20 dB. Most land-mobile and public-safety radio is vertically
polarized, while FM broadcast is often horizontal or circular.

## Relevance to SDR

A vertical antenna is the safe default for scanning land-mobile and trunked systems,
matching their vertical polarization.
