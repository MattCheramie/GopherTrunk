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

## How it works

Each transmission and [affiliation](/reference/affiliation/) carries the source radio
ID, so a monitor can see *which unit* is talking, not just which group.

## Relevance to SDR

GopherTrunk's Radio IDs view merges live radio IDs with any alias catalogue, letting you
track individual units across a system.
