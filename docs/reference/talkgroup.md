---
slug: talkgroup
title: Talkgroup
entry_type: term
category: trunked-radio
description: A talkgroup is a virtual channel in a trunked system identifying a group of users; members hear each other regardless of which physical frequency a call is assigned.
keywords: talkgroup, TGID, virtual channel, trunking, dispatch, fleet
aka: [talkgroup, talk group]
autolink: true
infobox:
  - { label: Type, value: Virtual user channel }
  - { label: Identified by, value: Talkgroup ID (TGID) }
  - { label: You follow, value: Talkgroups, not frequencies }
see_also: [trunked-radio, control-channel, voice-channel, radio-id, affiliation]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Talkgroup (Wikipedia)", url: https://en.wikipedia.org/wiki/Talkgroup }
---

A **talkgroup** is a virtual channel in a [trunked radio](/reference/trunked-radio/)
system — a numbered label identifying a group of users such as "Police Dispatch."
Members hear each other no matter which physical [voice channel](/reference/voice-channel/)
the system assigns to a given call.

## How it works

Because the frequency changes call to call, the talkgroup provides a stable identity.
Operators lock, prioritise, or mute *talkgroups*, and the system handles the
frequency-hopping underneath.

## Relevance to SDR

In GopherTrunk you follow talkgroups, not frequencies; each is shown with the
transmitting [radio ID](/reference/radio-id/).
