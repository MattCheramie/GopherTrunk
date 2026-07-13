---
slug: nanovna
title: NanoVNA
entry_type: hardware
category: test-equipment
description: "The NanoVNA is a low-cost, open-source handheld vector network analyzer that measures antenna and RF-network S-parameters from roughly 50 kHz to 1.5 GHz."
keywords: NanoVNA, handheld VNA, low-cost vector network analyzer, antenna analyzer, S11, S21, SWR, return loss, SOLT calibration, open source test equipment
aka: [NanoVNA, NanoVNA-H, NanoVNA-F]
autolink: true
infobox:
  - { label: Type, value: Handheld vector network analyzer }
  - { label: Vendor/Chip, value: "Open source; Si5351 + SA612" }
  - { label: Range, value: "~50 kHz – 1.5 GHz (up to 3–6 GHz on newer)" }
  - { label: Ports, value: "2 (S11 + S21)" }
  - { label: TX, value: "Yes (internal test source)" }
  - { label: Typical price, value: "$40 – $150" }
see_also: [vector-network-analyzer, s-parameters, standing-wave-ratio, return-loss, tinysa, antenna]
cite_urls:
  - https://en.wikipedia.org/wiki/NanoVNA
  - https://nanovna.com/
---

**The NanoVNA** is a low-cost, open-source handheld
[vector network analyzer](/reference/vector-network-analyzer/) that brought
[S-parameter](/reference/s-parameters/) measurement — antenna
[SWR](/reference/standing-wave-ratio/), [return loss](/reference/return-loss/), filter
response, and cable checks — down to a pocket-sized, sub-$100 instrument.[^wiki][^home]
Originally designed by "edy555" (Tomohiro Takahashi) and since forked into many hardware
and firmware variants, it has become the default antenna analyzer for radio amateurs and
SDR hobbyists.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A NanoVNA schematic: a Si5351 clock generator source and two receiver ports labeled CH0 for reflection and CH1 for transmission feed a small touchscreen showing an SWR curve." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="nvar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="20" y="55" width="80" height="34" rx="4" fill="none" stroke="currentColor"/>
  <text x="60" y="72" font-size="8" fill="currentColor" text-anchor="middle">Si5351</text>
  <text x="60" y="83" font-size="8" fill="currentColor" text-anchor="middle">source</text>
  <circle cx="150" cy="55" r="7" fill="none" stroke="currentColor"/>
  <text x="150" y="42" font-size="8" fill="currentColor" text-anchor="middle">CH0</text>
  <circle cx="150" cy="95" r="7" fill="none" stroke="currentColor"/>
  <text x="150" y="112" font-size="8" fill="currentColor" text-anchor="middle">CH1</text>
  <line x1="100" y1="72" x2="143" y2="58" stroke="currentColor" marker-end="url(#nvar)"/>
  <line x1="100" y1="72" x2="143" y2="92" stroke="currentColor" marker-end="url(#nvar)"/>
  <rect x="260" y="35" width="160" height="80" rx="6" fill="currentColor" fill-opacity="0.08" stroke="currentColor"/>
  <polyline points="270,105 300,90 320,60 340,55 360,72 390,98 410,104" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <text x="340" y="128" font-size="8" fill="currentColor" text-anchor="middle">SWR sweep on display</text>
  <line x1="157" y1="60" x2="260" y2="72" stroke="currentColor" stroke-opacity="0.4"/>
</svg>
<figcaption>A NanoVNA: an Si5351 source drives a two-port bridge (CH0 reflection, CH1 transmission), with results shown as a live SWR or S-parameter sweep on the built-in touchscreen.</figcaption>
</figure>

## What it is

Under the hood the NanoVNA is a full — if modest — two-port VNA. A **Si5351** clock
generator synthesizes the test tone (with harmonics extending the usable range above the
chip's fundamental limit), an **SA612** mixer and resistive bridge separate incident from
reflected waves, and a microcontroller sweeps frequency while sampling amplitude and
phase. **CH0** performs reflection ([S11](/reference/s-parameters/) — SWR, return loss,
impedance) and **CH1** performs transmission ([S21](/reference/s-parameters/) — insertion
loss or gain). Results appear on a small touchscreen and can be logged to a PC over USB
for Smith-chart and Touchstone export.

## Variants

The open design has spawned a family of boards. The **NanoVNA-H** and **-H4** are the
common improved originals (2.8" and 4" displays); the **NanoVNA-F** and **-F V2** use
metal cases and better screens; and the **NanoVNA V2 / SAA-2N** and later "LiteVNA" and
"NanoVNA-F V3" designs raise the top frequency to roughly 3–6 GHz with improved dynamic
range. The baseline units cover about **50 kHz to 1.5 GHz**, which comfortably spans the
HF, VHF, and UHF bands where most trunking and scanner antennas live.

## In practice

Like any [VNA](/reference/vector-network-analyzer/), a NanoVNA is only as good as its
**calibration**: you run a SOLT (Short-Open-Load-Through) cal at the ends of your test
leads before every serious measurement, and re-cal whenever you change the frequency span
or cabling. Its limitations relative to a bench instrument are real — modest dynamic
range, more measurement noise, and less accuracy at the top of its range — but for setting
up an [antenna](/reference/antenna/), checking a [filter](/reference/rf-filter/), or
sanity-checking [coax](/reference/coaxial-cable/) it is remarkably capable for the price.
A companion [tinySA](/reference/tinysa/) covers the spectrum-analysis side.

## Relevance to SDR

For SDR scanning the NanoVNA is the practical tool for the job the
[VNA](/reference/vector-network-analyzer/) entry describes: confirm your
[discone](/reference/discone-antenna/), [dipole](/reference/dipole-antenna/), or
[whip](/reference/whip-antenna/) is resonant on the band you care about, minimize SWR, and
verify that any inline filter passes the target frequencies and rejects out-of-band
interferers before they reach the SDR front end. GopherTrunk does not talk to a NanoVNA —
it is a receive-only decoder — but a NanoVNA is the low-cost bench aid most GopherTrunk
users reach for when building or tuning their antenna and feedline.

## Sources

[^wiki]: [NanoVNA](https://en.wikipedia.org/wiki/NanoVNA) — Wikipedia, on the NanoVNA open-source handheld vector network analyzer and its variants.
[^home]: [nanovna.com](https://nanovna.com/) — project and community hub for NanoVNA hardware, firmware, and usage guidance.
