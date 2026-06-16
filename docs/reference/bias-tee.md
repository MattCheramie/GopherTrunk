---
slug: bias-tee
title: Bias tee
entry_type: hardware
category: hardware
description: A bias tee injects DC power onto the coax feeding an antenna-mounted device such as an LNA, while passing the RF signal through to the receiver.
keywords: bias tee, bias-T, DC injection, LNA power, phantom power, coax
aka: [bias tee, bias-T]
autolink: true
infobox:
  - { label: Type, value: RF/DC combining network }
  - { label: Function, value: DC power up the coax, RF through }
  - { label: Powers, value: Antenna-mounted LNA / active antenna }
see_also: [low-noise-amplifier, antenna, rtl-sdr]
related_lessons:
  - { title: "How an SDR receiver works", url: /learn/sdr-receiver/ }
external:
  - { title: "Bias tee (Wikipedia)", url: https://en.wikipedia.org/wiki/Bias_tee }
---

A **bias tee** is a small network that injects **DC power onto the coax** feeding an
antenna-mounted device — typically a [low-noise amplifier](/reference/low-noise-amplifier/)
— while passing the RF signal through to the receiver unaffected.

## How it works

It combines a DC path and an RF path so a single cable carries both. Many SDRs (including
some [RTL-SDR](/reference/rtl-sdr/) models) have a built-in switchable bias tee.

## Relevance to SDR

A bias tee lets you power a mast-mounted LNA without a separate cable, keeping the
amplifier close to the [antenna](/reference/antenna/) where it does the most good.
