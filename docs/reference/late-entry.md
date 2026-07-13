---
slug: late-entry
title: Late entry
entry_type: term
category: trunked-radio
description: Late entry is joining a trunked call already in progress, made possible because the control channel periodically re-broadcasts the active channel grant.
keywords: late entry, late-entry, call in progress, grant update, group voice update, control channel, channel grant, trunking, P25 late entry
aka: [late entry, late-entry, grant update]
autolink: true
infobox:
  - { label: Type, value: Call-following behaviour }
  - { label: Enabled by, value: Repeated grant updates }
  - { label: Benefit, value: Radios join a call mid-transmission }
see_also: [control-channel, channel-grant, group-call, talkgroup, voice-channel, busy-idle]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

**Late entry** is joining a trunked call that is *already in progress* rather than at its
start.[^wiki] It works because the [control channel](/reference/control-channel/) does not
announce a call only once: while a call is active it periodically re-broadcasts a
**grant update** naming the [talkgroup](/reference/talkgroup/) and the
[voice channel](/reference/voice-channel/) it is using, so a radio (or monitor) that
missed the original [channel grant](/reference/channel-grant/) can still discover the
call and tune to it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A timeline of a call showing the initial grant followed by repeated grant updates, with a late radio joining at one of the updates rather than at the start." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="55" x2="440" y2="55" stroke="currentColor" stroke-width="1.2" marker-end="url(#lear)"/><text x="235" y="45" text-anchor="middle" font-size="8" fill="currentColor">control channel over time</text>
  <line x1="60" y1="48" x2="60" y2="62" stroke="currentColor" stroke-width="1.6"/><text x="60" y="76" text-anchor="middle" font-size="7.5" fill="currentColor">grant</text>
  <line x1="150" y1="49" x2="150" y2="61" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/><text x="150" y="76" text-anchor="middle" font-size="7" fill="currentColor">update</text>
  <line x1="240" y1="49" x2="240" y2="61" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/><text x="240" y="76" text-anchor="middle" font-size="7" fill="currentColor">update</text>
  <line x1="330" y1="49" x2="330" y2="61" stroke="currentColor" stroke-width="1.2" stroke-dasharray="3 2"/><text x="330" y="76" text-anchor="middle" font-size="7" fill="currentColor">update</text>
  <circle cx="240" cy="55" r="4.5" fill="currentColor"/><text x="240" y="102" text-anchor="middle" font-size="8" fill="currentColor">late radio joins here</text>
  <line x1="240" y1="108" x2="240" y2="62" stroke="currentColor" stroke-width="1" marker-end="url(#lear)"/>
  <defs><marker id="lear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Repeated grant updates let a radio join a call at a later point, not only at the initial grant.</figcaption>
</figure>

## How it works

A call is announced once by an initial [channel grant](/reference/channel-grant/). If a
radio was busy, out of range, or scanning another system at that instant, it would
otherwise never learn the call exists. To prevent that, the controller re-issues the
grant as a **grant update** (P25 calls this a *Group Voice Channel Grant Update*) at a
regular interval for as long as the call runs. Any radio affiliated to the
[talkgroup](/reference/talkgroup/) that sees an update while the call is still active
retunes to the named [voice channel](/reference/voice-channel/) and picks up the audio
mid-transmission — hence "late entry."

The same updates carry the current [busy/idle](/reference/busy-idle/) picture of the
system: they tell the fleet which channels are occupied by which groups right now. For a
listener, late entry is why a call can be followed even if monitoring started after the
call began — the necessary information keeps repeating on the control channel.

## In practice

The update interval is a system trade-off. Frequent updates make late entry fast and
robust but consume control-channel airtime; sparse updates conserve the control channel
but lengthen the worst-case delay before a late joiner (or a scanner) locks onto the
call. On busy systems updates for many simultaneous calls are interleaved on the single
control channel, so a monitor may see a rotating stream of "TG 101 on ch 3, TG 204 on ch
7, TG 101 on ch 3 …" as the controller keeps every active call advertised.

Late entry also interacts with encryption and vocoder framing. A voice call carries its
own embedded signalling — algorithm and key identifiers, the talkgroup, sometimes the
source unit — repeated within the traffic channel itself, so a radio (or monitor) that
joins late can recover the call's identity and cryptographic parameters without having
seen the setup on the control channel. That in-band repetition is the traffic-channel
analogue of the control channel's grant updates: both exist so that a receiver arriving
mid-call has everything it needs to make sense of what it is hearing.

## Relevance to SDR

Late entry is not just a radio feature — it is precisely what makes a trunk-following
scanner robust. A monitor that only reacted to the *initial* grant would miss every call
already underway when it tuned in, and would lose a call if it dropped a single message.
Because GopherTrunk parses grant updates as well as initial grants on the
[control channel](/reference/control-channel/), it can begin following a call in progress
the moment it starts decoding a system, and it can re-acquire a call if it briefly loses
a grant. The updates also give it a running map of which
[voice channels](/reference/voice-channel/) are active, which it uses to schedule
receivers efficiently.

Real systems that broadcast these updates include P25 Phase 1/Phase 2 (Group Voice
Channel Grant Update messages) and DMR Tier III. As with all trunking signalling,
GopherTrunk reads late-entry updates passively from the control channel; the audio it
recovers still depends on the traffic being unencrypted.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on control-channel signalling that lets radios follow active calls.
