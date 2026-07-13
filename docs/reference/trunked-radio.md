---
slug: trunked-radio
title: Trunked radio
entry_type: term
category: trunked-radio
description: Trunked radio is a system that shares a small pool of frequencies among many user groups by assigning a channel to each call on demand, coordinated by a control channel.
keywords: trunked radio, trunking, control channel, talkgroup, channel pool, public safety, P25, DMR Tier III, TETRA, queuing theory
aka: [trunked radio, trunking]
autolink: true
infobox:
  - { label: Type, value: Radio-system architecture }
  - { label: Coordinated by, value: Control channel }
  - { label: User identity, value: Talkgroup }
  - { label: Examples, value: P25, DMR Tier III, TETRA }
see_also: [conventional-radio, control-channel, voice-channel, talkgroup, channel-grant, fdma, tdma, multisite-trunking, trunking-site, group-call, registration]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Erlang_(unit)
---

**Trunked radio** is a system architecture in which many user groups share a small pool
of frequencies, with a computer assigning a free channel to each call for its duration
and reclaiming it afterward.[^wiki] A [control channel](/reference/control-channel/)
coordinates the whole system, so a handful of radio channels can serve hundreds of
[talkgroups](/reference/talkgroup/) that would each need a permanent frequency under
[conventional radio](/reference/conventional-radio/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A control channel at the top issuing an assignment, and a pool of voice channels below with one assigned to a talkgroup." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="20" width="400" height="34" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  <text x="230" y="35" text-anchor="middle" font-size="11" fill="currentColor" font-weight="600">control channel (data)</text>
  <text x="230" y="48" text-anchor="middle" font-size="9" fill="currentColor">"TG 101 → channel 3"</text>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="100" width="86" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="73" y="120">voice 1</text>
    <rect x="135" y="100" width="86" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="178" y="120">voice 2</text>
    <rect x="240" y="100" width="86" height="32" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="283" y="116">voice 3</text><text x="283" y="128" font-size="8">TG 101</text>
    <rect x="345" y="100" width="86" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="388" y="120">voice 4</text>
  </g>
  <line x1="230" y1="54" x2="283" y2="98" stroke="currentColor" stroke-dasharray="4 3" marker-end="url(#trar)"/>
  <text x="230" y="155" text-anchor="middle" font-size="9" fill="currentColor">a free channel is assigned per call, then released</text>
  <defs><marker id="trar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A trunked system shares a pool of channels; the control channel assigns a free one to each call.</figcaption>
</figure>

## How it works

When a user keys up, their radio sends a call request on the control channel. The site
controller picks a free [voice channel](/reference/voice-channel/) from the pool and
broadcasts a [channel grant](/reference/channel-grant/) that points the target talkgroup
to it; every radio [affiliated](/reference/affiliation/) to that group — plus any monitor
following the system — retunes to the named channel and slot. When the talker unkeys and
a hang-timer expires, the channel is released back to the pool, ready for the next call.

The efficiency comes from statistics. Real voice traffic is bursty: any one group
transmits only a small fraction of the time, and groups rarely all key up at once.
Trunking exploits this by pooling demand, a form of the same [queuing
theory](/reference/trunked-radio/) (measured in *erlangs* of offered load) that lets a
telephone exchange serve many subscribers with fewer trunk lines than customers.[^erlang]
A well-engineered system sizes its channel count so that the probability of a call
finding every channel busy — the *grade of service* — stays low. When it is exceeded, the
system either queues the request or returns a busy tone.

## Variants

- **FDMA vs. TDMA.** [FDMA](/reference/fdma/) systems (P25 Phase 1, NXDN) give each call
  its own frequency; [TDMA](/reference/tdma/) systems (P25 Phase 2, DMR Tier III, TETRA)
  divide each frequency into timeslots so one RF channel carries several calls at once.
- **Dedicated vs. distributed control.** Most systems dedicate one frequency as the
  control channel; lighter DMR modes rotate control around the pool using a
  [rest channel](/reference/rest-channel/).
- **Single- vs. multi-site.** A lone [trunking site](/reference/trunking-site/) covers one
  cell; [multisite trunking](/reference/multisite-trunking/) links many sites into a
  wide-area network, sometimes with [simulcast](/reference/simulcast/) transmitters and
  automatic [roaming](/reference/roaming/) between [neighbor sites](/reference/neighbor-site/).

## In practice

Public-safety and commercial fleets are the dominant users: police, fire, transit,
utilities, and campuses. The largest deployments — statewide P25 systems, TETRA networks
in Europe — carry tens of thousands of subscribers across hundreds of sites, tied together
by [registration](/reference/registration/) and [group-call](/reference/group-call/)
signalling. When the network backbone fails, sites can drop into
[failsoft](/reference/failsoft/), reverting affected channels to conventional repeater
operation so local traffic survives.

## Relevance to SDR

To monitor a trunked system you decode the control channel first, then follow the grants
it issues — exactly what GopherTrunk does for [P25](/reference/project-25/),
[DMR Tier III](/reference/dmr-tier-3/), and others. Because a monitor only has to keep up
with one data channel plus whatever calls are active, a single wideband SDR capture can
reconstruct dozens of concurrent conversations, tagged by talkgroup and
[radio ID](/reference/radio-id/). This is the core reason trunk-following is worth the
extra decoding complexity over simply scanning fixed frequencies.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on trunking architecture and control-channel coordination.
[^erlang]: [Erlang (unit)](https://en.wikipedia.org/wiki/Erlang_(unit)) — Wikipedia, on the traffic-engineering measure that underlies channel-pool sizing.
