---
slug: cellular-modem
title: Cellular modem
entry_type: hardware
category: hw-mobile
description: A cellular modem is the radio subsystem that connects a device to mobile networks (4G LTE, 5G), handling the RF, modulation, and protocols needed to carry data and calls over licensed cellular bands.
keywords: cellular modem, baseband, LTE, 5G, 4G, mobile broadband, baseband processor, modem chip, SIM, eSIM
infobox:
  - { label: Type, value: Wireless radio modem }
  - { label: Connects to, value: 4G/5G cellular networks }
  - { label: Also called, value: Baseband processor }
  - { label: Pairs with, value: SIM / eSIM }
see_also: [system-on-a-chip, esim, gps-receiver, smartphone, mobile-operating-system, modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Baseband_processor
---

A **cellular modem** is the radio subsystem that connects a device to mobile networks — 4G LTE, 5G, and their predecessors — handling the RF, [modulation](/reference/modulation/), and protocols that carry calls and data over licensed cellular bands.[^wiki]

## Overview

Often called the *baseband processor*, a cellular modem runs its own real-time firmware and manages the complex dance of attaching to a tower, negotiating bandwidth, and hopping between cells. It is paired with a subscriber identity from a SIM or an [eSIM](/reference/esim/). In a phone the modem is usually a block inside the main [SoC](/reference/system-on-a-chip/) (or a companion chip), wired to its own antennas separate from Wi-Fi and [GPS](/reference/gps-receiver/).

## Where it fits

The cellular modem is what makes a [smartphone](/reference/smartphone/) "mobile" in the connectivity sense — always-on data anywhere there is coverage. For a remote GopherTrunk capture node out of Wi-Fi range, a cellular modem (as a USB stick or HAT) is the practical backhaul, letting the node upload decoded calls over the mobile network. The modem speaks the carrier's protocols; it is not a general SDR and does not expose raw RF the way an [RTL-SDR](/reference/rtl-sdr/) does.

## Sources

[^wiki]: [Baseband processor](https://en.wikipedia.org/wiki/Baseband_processor) — Wikipedia, on the modem subsystem in mobile devices.
