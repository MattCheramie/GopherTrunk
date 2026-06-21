---
slug: conventional-radio
title: Conventional radio
entry_type: term
category: trunked-radio
description: Conventional radio assigns each user group a fixed frequency, unlike trunked radio; it is simple to scan because conversations always occur on the same channel.
keywords: conventional radio, non-trunked, fixed frequency, simplex, repeater
aka: [conventional radio]
autolink: true
infobox:
  - { label: Type, value: Radio-system architecture }
  - { label: Channel use, value: Fixed frequency per group }
  - { label: Contrast, value: Trunked radio }
see_also: [trunked-radio, voice-channel, frequency]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Two-way_radio
---

**Conventional radio** assigns each user group its own **fixed frequency**, in contrast
to [trunked radio](/reference/trunked-radio/).[^wiki] A conversation always happens on the same
channel, so there is no [control channel](/reference/control-channel/) to coordinate
assignments.

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

Groups simply transmit on their assigned simplex frequency or repeater pair. This is
simple and robust but uses spectrum inefficiently, since each channel sits idle when its
group is quiet.

## Relevance to SDR

Conventional channels are scanned directly by tuning to the known frequency — no
grant-following required. [DMR Tier II](/reference/dmr-tier-2/) is a digital example.

## Sources

[^wiki]: [Two-way radio](https://en.wikipedia.org/wiki/Two-way_radio) — Wikipedia, on fixed-frequency conventional two-way radio operation.
