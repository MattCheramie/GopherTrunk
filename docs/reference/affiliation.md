---
slug: affiliation
title: Affiliation
entry_type: term
category: trunked-radio
description: Affiliation is the process by which a radio registers with a trunked system over the control channel, letting the system route calls and track which units are active.
keywords: affiliation, registration, control channel, radio ID, trunking
aka: [affiliation]
autolink: true
infobox:
  - { label: Type, value: Trunking registration event }
  - { label: Carried on, value: Control channel }
  - { label: Reveals, value: Active radios and talkgroups }
see_also: [control-channel, radio-id, talkgroup, trunked-radio]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

**Affiliation** is the process by which a radio **registers** with a
[trunked radio](/reference/trunked-radio/) system over the
[control channel](/reference/control-channel/) when it powers on or changes
[talkgroup](/reference/talkgroup/), so the system can route calls efficiently.

## How it works

Affiliation messages name the [radio ID](/reference/radio-id/) and talkgroup, giving the
control channel a constant stream of information about which units and groups are
active.

## Relevance to SDR

Affiliation data lets GopherTrunk populate its Radio IDs and activity views even before
a call begins, showing who is on the system.
