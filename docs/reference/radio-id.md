---
slug: radio-id
title: Radio ID
entry_type: term
category: trunked-radio
description: A radio ID (unit ID) is the unique identifier of an individual radio on a system, revealed in control-channel signalling and in-band signalling such as MDC1200.
keywords: radio ID, unit ID, ANI, subscriber ID, RID, source ID, MDC1200
aka: [radio ID, unit ID]
autolink: true
infobox:
  - { label: Type, value: Subscriber identifier }
  - { label: Seen in, value: Control-channel data, MDC1200 }
  - { label: Identifies, value: An individual radio/unit }
see_also: [talkgroup, affiliation, control-channel, mdc1200]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

A **radio ID** (unit ID) is the unique identifier of an individual radio on a system,
distinct from the [talkgroup](/reference/talkgroup/) it is using. It appears in
[control-channel](/reference/control-channel/) signalling and in analog in-band schemes
like [MDC1200](/reference/mdc1200/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A transmission burst tagged with a unique radio identifier and its talkgroup." xmlns="http://www.w3.org/2000/svg">
  <rect x="60" y="40" width="340" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="60" text-anchor="middle" font-size="10" fill="currentColor">voice transmission</text>
  <text x="120" y="30" font-size="9" fill="currentColor" text-anchor="middle">radio ID 4567</text>
  <text x="330" y="30" font-size="9" fill="currentColor" text-anchor="middle">TG 101</text>
  <g stroke="currentColor" stroke-width="1"><line x1="120" y1="33" x2="120" y2="40"/><line x1="330" y1="33" x2="330" y2="40"/></g>
  <text x="230" y="92" text-anchor="middle" font-size="9" fill="currentColor">every transmission identifies the individual radio</text>
</svg>
<figcaption>A radio ID uniquely identifies the transmitting unit, so you see who is talking, not just which group.</figcaption>
</figure>

## How it works

Each transmission and [affiliation](/reference/affiliation/) carries the source radio
ID, so a monitor can see *which unit* is talking, not just which group.

## Relevance to SDR

GopherTrunk's Radio IDs view merges live radio IDs with any alias catalogue, letting you
track individual units across a system.
