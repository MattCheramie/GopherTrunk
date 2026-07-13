---
slug: failsoft
title: Failsoft
entry_type: term
category: trunked-radio
description: "Failsoft is a trunked system's fallback mode: when the controller or control channel fails, radios revert to fixed conventional channels so communication survives."
keywords: failsoft, fail-soft, fallback, trunking failure, controller failure, control channel loss, conventional operation, degraded mode, P25 failsoft, SmartNet failsoft
aka: [failsoft, fail-soft, fallback mode]
autolink: true
infobox:
  - { label: Type, value: Degraded-mode operation }
  - { label: Trigger, value: Controller / control-channel loss }
  - { label: Behaviour, value: Talkgroups map to fixed channels }
see_also: [trunked-radio, conventional-radio, control-channel, trunking-site, talkgroup, voice-channel]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

**Failsoft** is the fallback mode a trunked system enters when its controller or
[control channel](/reference/control-channel/) fails: rather than going silent, the site
abandons dynamic channel assignment and reverts to fixed, conventional operation so that
communication survives in a degraded form.[^wiki] Each
[talkgroup](/reference/talkgroup/) is mapped to a specific repeater channel, and radios
fall back to that channel — losing the efficiency of
[trunking](/reference/trunked-radio/) but keeping basic
[conventional](/reference/conventional-radio/) contact.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="When the control channel fails, the trunked system switches from dynamic channel grants to a fixed mapping of talkgroups onto specific conventional channels." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="130" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="85" y="39" text-anchor="middle" font-size="8.5" fill="currentColor">control channel</text>
  <line x1="85" y1="52" x2="85" y2="52" stroke="currentColor"/>
  <path d="M70 60 L100 78 M100 60 L70 78" stroke="currentColor" stroke-width="1.8"/><text x="130" y="72" text-anchor="middle" font-size="8" fill="currentColor">fails</text>
  <line x1="160" y1="69" x2="205" y2="69" stroke="currentColor" stroke-width="1.2" marker-end="url(#fsar)"/><text x="182" y="62" text-anchor="middle" font-size="7.5" fill="currentColor">failsoft</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="216" y="46" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="266" y="62">TG 101 → ch 1</text>
    <rect x="216" y="76" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="266" y="92">TG 102 → ch 2</text>
    <rect x="330" y="46" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="380" y="62">TG 205 → ch 3</text>
    <rect x="330" y="76" width="100" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="380" y="92">fixed mapping</text>
  </g>
  <defs><marker id="fsar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>In failsoft the dynamic control channel is gone; talkgroups fall back to fixed conventional channels.</figcaption>
</figure>

## How it works

Normal trunking depends on a controller assigning channels on demand through the control
channel. If the controller fails, or the control channel is lost, that coordination
disappears. In failsoft the site's repeaters revert to a pre-programmed conventional
plan: each talkgroup is pinned to one repeater, and radios programmed for that system
automatically switch to the failsoft channel for their selected talkgroup. Multiple
talkgroups may share a channel, so the tidy separation trunking provides is gone —
several groups may end up hearing each other — but the site keeps carrying voice.

Radios usually signal failsoft to the user (a distinctive tone or a display
indication) so operators know they have lost trunked features such as
[private calls](/reference/private-call/), patches, and dynamic channel access. When the
controller recovers and the control channel returns, the site resumes normal trunked
operation and radios re-affiliate.

## In practice

Failsoft is deliberately simple because it must work when the smart part of the system is
broken. The mapping of talkgroups to channels is fixed in advance, and the repeaters
often transmit a low-speed failsoft identifier so radios (and monitors) can recognize the
degraded state. It is distinct from a *site trunking* fallback, where a multi-site system
loses its network link but each site still trunks locally — failsoft is the deeper
fallback where even local trunking is unavailable and the system is effectively
conventional.

It is worth placing failsoft on the ladder of degraded modes a wide-area system can
occupy. At the top is full **wide-area trunking**, where all sites are networked and a
call reaches every site that has affiliated members. If the network backhaul fails, sites
drop to **site trunking**: each site still trunks normally on its own, assigning channels
through its own control channel, but calls no longer cross between sites. Only when a
site's own controller or control channel fails does it fall all the way to **failsoft**,
abandoning dynamic assignment entirely. Each rung sacrifices more capability for more
robustness, and failsoft is the floor — the mode designed to keep working when almost
everything smart has stopped. Radios and monitors can often tell which rung a system is on
from what the control channel is (or is not) doing and from the identifiers the site
transmits.

Failsoft is intentionally the least-loved but most-tested part of a system's design. It has
to work with no controller intelligence behind it, so its rules are static and its channel
plan is fixed in the radios and repeaters ahead of time. That simplicity is the point:
when the smart infrastructure is gone, a mode that depends on more of that same
infrastructure would be worthless. The cost is everything trunking normally buys —
efficient channel sharing, [private calls](/reference/private-call/), patches, wide-area
reach — all of which vanish until the controller returns. Well-run agencies rehearse
operating in failsoft precisely because it is the mode they will be in on the worst day, and
recognizing it quickly, both for users and for anyone monitoring, is what keeps that day
manageable.

## Relevance to SDR

Failsoft matters to a monitor because a system in failsoft no longer behaves like a
trunked system at all — there is no control channel to follow, and a trunk-tracking
scanner that keeps hunting for one will hear nothing. Recognizing the failsoft
identifier (or simply the disappearance of the control channel and the appearance of
fixed talkgroup-to-channel traffic) tells GopherTrunk to treat the site as a set of
[conventional](/reference/conventional-radio/) channels rather than as a trunk. In that
mode a monitor reverts to scanning the known [voice channels](/reference/voice-channel/)
directly, since the metadata-rich control-channel grants it normally relies on are gone.

Real systems with a defined failsoft mode include P25, Motorola Type II/SmartNet, and
EDACS. GopherTrunk's job in failsoft is diagnostic and adaptive: detect that the
[trunking site](/reference/trunking-site/) has degraded, and fall back to conventional
monitoring so the user still hears whatever traffic the site is carrying.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on failsoft and degraded-mode fallback in trunked systems.
