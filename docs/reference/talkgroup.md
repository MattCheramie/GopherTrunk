---
slug: talkgroup
title: Talkgroup
entry_type: term
category: trunked-radio
description: A talkgroup is a virtual channel in a trunked system identifying a group of users; members hear each other regardless of which physical frequency a call is assigned.
keywords: talkgroup, TGID, virtual channel, trunking, dispatch, fleet, group call, patch, priority
aka: [talkgroup, talk group]
autolink: true
infobox:
  - { label: Type, value: Virtual user channel }
  - { label: Identified by, value: Talkgroup ID (TGID) }
  - { label: You follow, value: Talkgroups, not frequencies }
see_also: [trunked-radio, control-channel, voice-channel, radio-id, affiliation, group-call, patch-multigroup, priority-scan, emergency-call]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Talkgroup
  - https://en.wikipedia.org/wiki/Project_25
---

A **talkgroup** is a virtual channel in a [trunked radio](/reference/trunked-radio/)
system — a numbered label identifying a group of users such as "Police Dispatch."[^wiki]
Members hear each other no matter which physical [voice channel](/reference/voice-channel/)
the system assigns to a given call, because the talkgroup, not the frequency, is the thing
they belong to.

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

Every radio is programmed with a set of talkgroups it can select and monitor. When a user
keys up on a talkgroup, the radio requests a call carrying that talkgroup ID; the control
channel [grants](/reference/channel-grant/) a voice channel and announces the talkgroup so
every affiliated radio retunes together. Because the frequency changes call to call, the
talkgroup ID provides the stable identity that a human — or a scanner — tracks. Operators
lock, prioritise, or mute *talkgroups*, and the system handles the frequency-hopping
underneath.

Talkgroup IDs are numeric and system-specific. On P25 a TGID is a 16-bit value scoped to
the system's [WACN](/reference/wacn/) and [system ID](/reference/system-id/); DMR uses its
own numbering; and hobbyist databases publish human-readable alias tables mapping IDs to
agency and function. A talkgroup usually carries [group calls](/reference/group-call/), but
a system can also flag one as an [emergency](/reference/emergency-call/) talkgroup for
priority handling.

## Variants

- **Dispatch vs. tactical** — some talkgroups are announcement/dispatch groups; others are
  short-lived tactical channels for an incident.
- **Patched / multigroup** — a dispatcher can temporarily merge several talkgroups into a
  [patch or multigroup](/reference/patch-multigroup/) so they all hear one another.
- **Announcement group** — a supergroup that reaches every subordinate talkgroup at once,
  used for system-wide broadcasts.

## In practice

Talkgroup planning defines how an agency is organised on the air: a mid-size city might
run separate talkgroups for police districts, fire, EMS, public works, and events. Because
a scanner following talkgroups can filter to just the ones of interest, monitoring by
talkgroup is far more useful than watching raw frequencies. [Priority
scan](/reference/priority-scan/) lets a listener interrupt a lower-priority talkgroup when
a chosen one becomes active.

## Relevance to SDR

In GopherTrunk you follow talkgroups, not frequencies; each active call is shown with its
talkgroup ID, any configured alias, and the transmitting [radio ID](/reference/radio-id/).
This lets you build allow/deny lists, prioritise groups, and log activity per talkgroup
across a whole trunked system captured with one SDR.

## Sources

[^wiki]: [Talkgroup](https://en.wikipedia.org/wiki/Talkgroup) — Wikipedia, on the talkgroup as a virtual channel in trunked systems.
[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 talkgroup addressing within a WACN/system ID.
