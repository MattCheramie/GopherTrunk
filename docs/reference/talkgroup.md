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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A single talkgroup whose successive calls appear on different physical voice channels over time." xmlns="http://www.w3.org/2000/svg">
  <text x="230" y="22" text-anchor="middle" font-size="10" fill="currentColor">Talkgroup 101 (one virtual channel)</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="40" y="50" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="85" y="65">call 1</text><text x="85" y="78" font-size="8">on ch 3</text>
    <rect x="185" y="50" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="230" y="65">call 2</text><text x="230" y="78" font-size="8">on ch 7</text>
    <rect x="330" y="50" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="375" y="65">call 3</text><text x="375" y="78" font-size="8">on ch 2</text>
  </g>
  <text x="230" y="115" text-anchor="middle" font-size="9" fill="currentColor">you follow the talkgroup; the system moves it across channels</text>
</svg>
<figcaption>A talkgroup is a virtual channel: its calls hop across physical frequencies, but you follow the group.</figcaption>
</figure>

## How it works

Because the frequency changes call to call, the talkgroup provides a stable identity.
Operators lock, prioritise, or mute *talkgroups*, and the system handles the
frequency-hopping underneath.

## Relevance to SDR

In GopherTrunk you follow talkgroups, not frequencies; each is shown with the
transmitting [radio ID](/reference/radio-id/).
