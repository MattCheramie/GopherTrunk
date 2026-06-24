---
slug: esim
title: eSIM
entry_type: hardware
category: hw-mobile
description: An eSIM is a SIM card built directly into a device as a reprogrammable chip, letting users download and switch carrier profiles over the air instead of inserting a physical card.
keywords: eSIM, embedded SIM, eUICC, SIM card, carrier profile, remote provisioning, virtual SIM, dual SIM
autolink: true
aka: [embedded SIM, eUICC]
infobox:
  - { label: Type, value: Embedded SIM chip }
  - { label: Replaces, value: Physical SIM card }
  - { label: Provisioning, value: Over the air }
  - { label: Standard, value: GSMA eUICC }
see_also: [cellular-modem, system-on-a-chip, smartphone, smartwatch, mobile-operating-system, near-field-communication]
cite_urls:
  - https://en.wikipedia.org/wiki/ESIM
---

An **eSIM** is a SIM built directly into a device as a small, reprogrammable chip, letting a user download and switch carrier profiles over the air instead of swapping a physical card.[^wiki]

## Overview

Technically an *eUICC* (embedded Universal Integrated Circuit Card) standardized by the GSMA, an eSIM solders the SIM function onto the board and holds one or more downloadable *profiles*. Activating a plan means scanning a QR code or following a carrier flow; switching networks or running a second line no longer requires opening a tray. Freeing a device from a card slot saves space and improves water resistance, and lets tiny products — [smartwatches](/reference/smartwatch/), trackers — carry cellular service at all.

## Where it fits

The eSIM is the connectivity counterpart to a device's [cellular modem](/reference/cellular-modem/): the modem provides the radio, the eSIM the identity. For a fleet of remote GopherTrunk capture nodes on cellular backhaul, eSIM provisioning means carriers and data plans can be assigned and changed remotely, without anyone visiting each node to insert a card.

## Sources

[^wiki]: [eSIM](https://en.wikipedia.org/wiki/ESIM) — Wikipedia, on embedded SIM technology and remote provisioning.
