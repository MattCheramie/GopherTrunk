---
slug: conventional-radio
title: Conventional radio
entry_type: term
category: trunked-radio
description: Conventional radio assigns each user group a fixed frequency, unlike trunked radio; it is simple to scan because conversations always occur on the same channel.
keywords: conventional radio, non-trunked, fixed frequency, simplex, repeater, CTCSS, DCS, two-way radio
aka: [conventional radio]
autolink: true
infobox:
  - { label: Type, value: Radio-system architecture }
  - { label: Channel use, value: Fixed frequency per group }
  - { label: Contrast, value: Trunked radio }
see_also: [trunked-radio, voice-channel, frequency, failsoft, control-channel, ctcss]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Two-way_radio
  - https://en.wikipedia.org/wiki/Continuous_Tone-Coded_Squelch_System
---

**Conventional radio** assigns each user group its own **fixed frequency**, in contrast
to [trunked radio](/reference/trunked-radio/).[^wiki] A conversation always happens on the
same channel, so there is no [control channel](/reference/control-channel/) to coordinate
assignments and no [channel grant](/reference/channel-grant/) to follow — you simply tune
to the frequency you want.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="On the left, conventional radio with each group on its own fixed channel; on the right, trunked radio sharing a pool." xmlns="http://www.w3.org/2000/svg">
  <text x="115" y="20" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">conventional</text>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="40" y="34" width="150" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="115" y="49">Police → ch A (always)</text>
    <rect x="40" y="60" width="150" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="115" y="75">Fire → ch B (always)</text>
    <rect x="40" y="86" width="150" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="115" y="101">Public works → ch C</text>
  </g>
  <text x="345" y="20" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">trunked</text>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="270" y="34" width="150" height="22" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="345" y="49">shared pool of channels</text>
    <rect x="270" y="60" width="70" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="305" y="75">any group</text>
    <rect x="350" y="60" width="70" height="22" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="75">on demand</text>
  </g>
  <text x="230" y="135" text-anchor="middle" font-size="9" fill="currentColor">fixed assignment vs. assigned-per-call</text>
</svg>
<figcaption>Conventional radio gives each group a permanent frequency; trunking shares a pool on demand.</figcaption>
</figure>

## How it works

Groups transmit on their assigned simplex frequency or through a repeater pair (mobiles
transmit on an input frequency, the repeater retransmits on an output frequency to extend
range). To keep unrelated users off the same channel, conventional systems commonly layer
a sub-audible squelch tone or code — [CTCSS](/reference/ctcss/) or
[DCS](/reference/dcs/) — so a receiver only opens its speaker for its own group. Digital
conventional modes achieve the same selectivity with a color code, network access code,
or radio access number.

The architecture is simple and robust because there is no central controller to fail: if
one channel's repeater goes down, only that channel is affected. The cost is spectrum
efficiency — each frequency sits idle whenever its group is quiet, so serving *N* groups
requires close to *N* channels regardless of how little any one of them talks.

## Variants

- **Simplex** — radios transmit and receive on one frequency, talking directly unit to
  unit with no infrastructure.
- **Repeater (duplex)** — a fixed repeater rebroadcasts traffic to cover a wider area;
  the input/output pair is still one logical channel.
- **Analog vs. digital** — analog FM with CTCSS/DCS, or digital modes like
  [DMR Tier II](/reference/dmr-tier-2/) and [NXDN](/reference/nxdn/) conventional, which
  add unit IDs and text without a trunking controller.

## In practice

Conventional operation remains common for amateur radio, business licensees, aviation,
marine [VHF](/reference/marine-vhf/), and smaller public-safety agencies that do not need
the capacity of a trunked network. It is also the fallback for trunked systems: when a
site loses its controller, [failsoft](/reference/failsoft/) mode reverts each channel to
conventional repeater operation so local users keep talking. A monitor that knows a
system's frequency list can cover it with a simple priority-ordered scan.

## Relevance to SDR

Conventional channels are scanned directly by tuning to the known frequency — no
grant-following required, which makes them the easiest traffic to capture with an SDR.
GopherTrunk can log and play conventional analog and digital channels alongside the
trunked systems it tracks; the two coexist because a monitored region usually mixes both.
[DMR Tier II](/reference/dmr-tier-2/) is a widely deployed digital conventional example.

## Sources

[^wiki]: [Two-way radio](https://en.wikipedia.org/wiki/Two-way_radio) — Wikipedia, on fixed-frequency conventional two-way radio operation.
[^ctcss]: [Continuous Tone-Coded Squelch System](https://en.wikipedia.org/wiki/Continuous_Tone-Coded_Squelch_System) — Wikipedia, on the sub-audible tones that keep conventional channels selective.
