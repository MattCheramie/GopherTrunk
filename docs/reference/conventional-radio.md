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
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Two-way radio (Wikipedia)", url: https://en.wikipedia.org/wiki/Two-way_radio }
---

**Conventional radio** assigns each user group its own **fixed frequency**, in contrast
to [trunked radio](/reference/trunked-radio/). A conversation always happens on the same
channel, so there is no [control channel](/reference/control-channel/) to coordinate
assignments.

## How it works

Groups simply transmit on their assigned simplex frequency or repeater pair. This is
simple and robust but uses spectrum inefficiently, since each channel sits idle when its
group is quiet.

## Relevance to SDR

Conventional channels are scanned directly by tuning to the known frequency — no
grant-following required. [DMR Tier II](/reference/dmr-tier-2/) is a digital example.
