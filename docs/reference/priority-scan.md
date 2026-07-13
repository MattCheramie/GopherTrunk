---
slug: priority-scan
title: Priority scan
entry_type: term
category: trunked-radio
description: Priority scan is following a chosen set of talkgroups and preferring the higher-priority ones, so an important call preempts a lower one already being monitored.
keywords: priority scan, priority monitor, scan list, talkgroup priority, preempt, trunk tracking, control channel, trunking, scanner priority
aka: [priority scan, priority monitor, priority scanning]
autolink: true
infobox:
  - { label: Type, value: Monitoring strategy }
  - { label: Driven by, value: Control-channel grants }
  - { label: Rule, value: Higher priority wins the receiver }
see_also: [talkgroup, control-channel, channel-grant, group-call, emergency-call, voice-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Scanner_(radio)
---

**Priority scan** is a monitoring strategy in which a receiver follows a chosen set of
[talkgroups](/reference/talkgroup/) and, when more than one is active, prefers the
higher-priority group — so an important call can **preempt** a lower-priority one that is
already being listened to.[^wiki] On a trunked system this is driven entirely by the
[control channel](/reference/control-channel/): the scanner reads every
[channel grant](/reference/channel-grant/), compares the granted talkgroup against its
priority list, and points its limited receiver capacity at the most important active
call.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A scanner listening to a low-priority talkgroup switches to a higher-priority talkgroup when the control channel grants it a call." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="150" height="28" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="95" y="38" text-anchor="middle" font-size="8.5" fill="currentColor">control channel grants</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="30" y="64" width="120" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="90" y="81">TG 402 (P3) active</text>
    <rect x="30" y="98" width="120" height="26" rx="4" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/><text x="90" y="115">TG 101 (P1) active</text>
  </g>
  <line x1="152" y1="111" x2="250" y2="90" stroke="currentColor" stroke-width="1.4" marker-end="url(#psar)"/>
  <rect x="258" y="72" width="120" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="318" y="88" text-anchor="middle" font-size="8.5" fill="currentColor">receiver follows</text><text x="318" y="100" text-anchor="middle" font-size="8.5" fill="currentColor">TG 101 (higher)</text>
  <defs><marker id="psar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>When a higher-priority talkgroup becomes active, the receiver leaves the lower-priority call to follow it.</figcaption>
</figure>

## How it works

The listener assigns each talkgroup on the scan list a priority level. As the scanner
decodes the [control channel](/reference/control-channel/), each grant tells it that some
talkgroup has gone active on a [voice channel](/reference/voice-channel/). The scanner
then applies its rule:

- If nothing is currently being followed, tune to the newly granted call (if it is on the
  scan list).
- If a call is already being followed and a **higher-priority** talkgroup is granted,
  preempt the current call and switch to the higher one.
- If a **lower-or-equal-priority** talkgroup is granted, note it but keep the current
  call.

On conventional (non-trunked) radios, priority scan means periodically interrupting a
lower-priority channel to *sample* the priority channel for activity. On a trunked
system the logic is cleaner because the control channel *announces* activity — the
scanner never has to leave a call to check whether a priority group is busy; it simply
watches the grants. This makes trunked priority scanning both faster and more reliable.

## In practice

Priority scan is what lets a monitor with only one or two receivers cover a system that
may have dozens of active talkgroups. Priorities are usually arranged so that
dispatch/primary channels outrank tactical or administrative ones, and
[emergency calls](/reference/emergency-call/) are commonly given an implicit top priority
so they always break through. A well-tuned priority list is the difference between
hearing the important traffic and being stuck on a long, low-value transmission while the
call that matters plays out unheard on another channel.

A second decision is what to do when a higher-priority call *ends*. A common policy is
**resume**: if the preempting call finishes while the earlier, lower-priority call is
still up, the receiver falls back to whatever is still active — so the listener does not
lose the lower call entirely, only the portion that overlapped. Systems and scanners also
provide a *priority hold* or *hold* option that pins the receiver to one talkgroup
regardless of what else is granted, which is the opposite behaviour: it is useful when a
listener wants to stay with a developing incident and deliberately ignore everything else.
The right choice depends on whether the user is monitoring broadly or focused on one
event, and a good implementation makes both easy to express.

Priority scan is also where the number of available receivers shows through. With a single
tuner, priority is strictly one-at-a-time and preemption really does mean abandoning one
call for another. With several tuners, "priority" becomes an allocation problem — assign
the scarce receivers to the highest-value active calls and let lower ones go unheard only
when there are more calls than tuners — which is a richer and more forgiving version of the
same idea.

## Relevance to SDR

Priority scan is a scanner behaviour rather than a system feature, and it maps directly
onto how GopherTrunk allocates its receivers. GopherTrunk parses every
[channel grant](/reference/channel-grant/) on the control channel and can weight
[talkgroups](/reference/talkgroup/) so that a limited pool of tuners is always assigned
to the highest-value active calls, preempting a lower-priority call when a more important
one begins — including surfacing [emergency](/reference/emergency-call/)-flagged traffic
first. Because the decision is driven by control-channel metadata, GopherTrunk can make
it before committing a receiver to demodulate any audio, which is what keeps a
software-defined scanner responsive across a large channel pool.

Real systems where this applies include P25 Phase 1/Phase 2, DMR Tier III, and Motorola
trunking. The priorities are the user's policy; GopherTrunk supplies the control-channel
awareness that makes the policy actionable.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on following talkgroups by monitoring control-channel grants.
