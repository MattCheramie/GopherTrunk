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
  - { title: "How an SDR receiver works", url: /learn/rf-sdr/sdr-receiver/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bias_tee
---

A **bias tee** is a small network that injects **DC power onto the coax** feeding an
antenna-mounted device — typically a [low-noise amplifier](/reference/low-noise-amplifier/)
— while passing the RF signal through to the receiver unaffected.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A bias tee injecting DC power onto the coax while passing RF through to the receiver." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="55" x2="200" y2="55" stroke="currentColor" stroke-width="1.4"/><text x="60" y="46" font-size="8" fill="currentColor">RF + DC</text>
  <line x1="200" y1="55" x2="420" y2="55" stroke="currentColor" stroke-width="1.4"/><text x="360" y="46" font-size="8" fill="currentColor">RF to RX</text>
  <line x1="200" y1="55" x2="200" y2="95" stroke="currentColor" stroke-width="1.4"/><text x="200" y="108" font-size="8" fill="currentColor" text-anchor="middle">DC supply</text>
  <circle cx="200" cy="55" r="3" fill="currentColor"/>
  <text x="230" y="30" font-size="9" fill="currentColor">powers a remote LNA over the coax</text>
</svg>
<figcaption>A bias tee feeds DC up the coax to power a mast-mounted LNA while passing the RF through.</figcaption>
</figure>

## How it works

It combines a DC path and an RF path so a single cable carries both. Many SDRs (including
some [RTL-SDR](/reference/rtl-sdr/) models) have a built-in switchable bias tee.

## Relevance to SDR

A bias tee lets you power a mast-mounted LNA without a separate cable, keeping the
amplifier close to the [antenna](/reference/antenna/) where it does the most good.

## Sources

[^wiki]: [Bias tee](https://en.wikipedia.org/wiki/Bias_tee) — Wikipedia, on the RF/DC combining network that powers antenna-mounted devices over the coax.
