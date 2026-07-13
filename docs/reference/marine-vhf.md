---
slug: marine-vhf
title: Marine VHF Radio
entry_type: technology
category: aviation-marine
description: Marine VHF is the internationally standardised band of FM voice and data channels near 156-162 MHz used for ship-to-ship, ship-to-shore, safety, and DSC calling at sea.
keywords: marine VHF, VHF marine band, channel 16, channel 70, DSC, 156 MHz, 162 MHz, FM voice, distress calling, GMDSS, dual watch, ship radio
aka: [marine VHF, VHF marine band]
autolink: true
infobox:
  - { label: Type, value: Maritime voice + data radio service }
  - { label: Idea, value: Standard FM simplex/duplex channel plan at sea }
  - { label: Band, value: ~156–162 MHz VHF }
see_also: [frequency-modulation, dsc, ais, frequency-bands]
cite_urls:
  - https://en.wikipedia.org/wiki/Marine_VHF_radio
  - https://www.itu.int/
---

**Marine VHF** is the internationally standardised set of radio channels near
**156–162 MHz** used for communication at sea. It carries [FM](/reference/frequency-modulation/)
voice for ship-to-ship and ship-to-shore calling, distress and safety traffic, port and
navigation coordination, and — on dedicated channels — digital data such as
[DSC](/reference/dsc/) calling and [AIS](/reference/ais/) position reporting.[^wiki] Its
fixed, worldwide **channel plan** means a vessel can raise another ship, a marina, or a
coast station almost anywhere using the same numbered channels.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="The marine VHF band divided into numbered channels, highlighting channel 16 for distress and calling and channel 70 for digital selective calling among FM voice channels." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="55" width="400" height="26" fill="none" stroke="currentColor"/>
  <g stroke="currentColor" stroke-opacity="0.4">
    <line x1="90" y1="55" x2="90" y2="81"/><line x1="150" y1="55" x2="150" y2="81"/>
    <line x1="210" y1="55" x2="210" y2="81"/><line x1="270" y1="55" x2="270" y2="81"/>
    <line x1="330" y1="55" x2="330" y2="81"/><line x1="390" y1="55" x2="390" y2="81"/>
  </g>
  <rect x="150" y="55" width="60" height="26" fill="currentColor" fill-opacity="0.25"/>
  <rect x="330" y="55" width="60" height="26" fill="currentColor" fill-opacity="0.15"/>
  <text x="180" y="72" text-anchor="middle" font-size="8" fill="currentColor">Ch 16</text>
  <text x="360" y="72" text-anchor="middle" font-size="8" fill="currentColor">Ch 70</text>
  <text x="30" y="48" font-size="8" fill="currentColor">156 MHz</text>
  <text x="430" y="48" text-anchor="end" font-size="8" fill="currentColor">162 MHz</text>
  <text x="230" y="105" text-anchor="middle" font-size="8.5" fill="currentColor">Ch 16 = distress &amp; calling (FM voice) · Ch 70 = DSC (digital)</text>
</svg>
<figcaption>Marine VHF divides the band into standard numbered channels, with Channel 16 for FM distress and calling and Channel 70 reserved for DSC.</figcaption>
</figure>

## How it works

Marine VHF uses **narrowband FM** with channels spaced 25 kHz apart across roughly
156–162 MHz. Channels are numbered in an internationally agreed plan and split into
**simplex** channels (transmit and receive on one frequency, for ship-to-ship) and
**duplex** channels (separate ship and shore frequencies, for public correspondence and
port operations). Two channels are effectively universal: **Channel 16** (156.800 MHz) is
the distress, safety, and calling channel that every vessel monitors, and **Channel 70**
(156.525 MHz) is reserved entirely for [DSC](/reference/dsc/) digital calling rather than
voice.

Sets typically offer a **dual-watch** feature that keeps Channel 16 monitored while the
crew works another channel, and transmit power is switchable between about 25 W for range
and 1 W for close-in or port use to limit congestion. Because VHF propagates roughly
line-of-sight, practical range is set by antenna height — a handheld reaches a few miles,
while a masthead antenna talking to a tall coast station can span tens of miles.

## Relevance to SDR

The marine VHF band is an easy and rewarding target for a
[software-defined radio](/reference/software-defined-radio/): FM voice channels demodulate
with standard narrowband FM, while the same slice of spectrum carries the digital signals
covered elsewhere in this guide — DSC bursts on Channel 70 and
[AIS](/reference/ais/) near 162 MHz. **GopherTrunk** is built for land-mobile trunking
rather than marine voice, but it *does* decode the marine data signals AIS and DSC that
share this band; plain FM voice on the voice channels is outside its trunking focus and is
better handled by a general scanner.

## Sources

[^wiki]: [Marine VHF radio](https://en.wikipedia.org/wiki/Marine_VHF_radio) — Wikipedia, for the marine VHF channel plan, FM voice operation, Channel 16 distress/calling and Channel 70 DSC assignments, and simplex/duplex usage.
