---
slug: csbk
title: CSBK
entry_type: term
category: trunked-radio
description: A CSBK (control signalling block) is the single-block control message in DMR, carrying call setup, channel grants, and system data on a Tier III control channel.
keywords: CSBK, control signalling block, DMR control channel, Tier III, channel grant, signalling
aka: [CSBK, "control signalling block", "control signaling block"]
autolink: true
see_also: [control-channel, channel-grant, dmr-tier-3, dmr, tsbk]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
external:
  - { title: "Digital mobile radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Digital_mobile_radio }
---

A **CSBK** (**control signalling block**) is the single-block control message of
[DMR](/reference/dmr/). On a [Tier III](/reference/dmr-tier-3/) control channel, CSBKs
carry call requests, [channel grants](/reference/channel-grant/), and system data — the
DMR counterpart of P25's [TSBK](/reference/tsbk/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A DMR control channel carrying CSBK blocks, one of which grants a traffic channel and slot." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="400" height="24" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="230" y="46" text-anchor="middle" font-size="8.5" fill="currentColor">DMR Tier III control channel</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="66" width="80" height="26" fill="none"/><rect x="130" y="66" width="90" height="26" fill="currentColor" fill-opacity="0.22"/><rect x="230" y="66" width="80" height="26" fill="none"/><rect x="320" y="66" width="80" height="26" fill="none"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="80" y="83">CSBK</text><text x="175" y="80">CSBK</text><text x="175" y="90" font-size="7">(grant)</text><text x="270" y="83">CSBK</text><text x="360" y="83">CSBK</text></g>
</svg>
<figcaption>DMR Tier III coordinates the system with CSBKs; a grant CSBK assigns a traffic channel and timeslot.</figcaption>
</figure>

## Overview

CSBKs are protected by [BPTC](/reference/bptc/) coding. Following them is how a decoder
tracks trunked DMR, just as TSBKs are followed on P25.
