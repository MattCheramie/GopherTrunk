---
slug: emergency-call
title: Emergency call
entry_type: term
category: trunked-radio
description: An emergency call is a trunked group call flagged with an emergency bit, giving it top priority for channel access and distinct alerting on the control channel.
keywords: emergency call, emergency bit, emergency alarm, priority call, ruthless preemption, talkgroup, trunking, P25 emergency, orange button
aka: [emergency call, emergency alarm, emergency bit]
autolink: true
infobox:
  - { label: Type, value: Priority group call }
  - { label: Signalled by, value: Emergency bit / opcode }
  - { label: Effect, value: Top priority for channel access }
see_also: [group-call, talkgroup, channel-grant, control-channel, radio-id, priority-scan]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

An **emergency call** is a [group call](/reference/group-call/) that carries an
**emergency flag** — a single bit or a dedicated opcode in the control-channel
signalling — that marks it as the highest-priority traffic on the system and, on many
systems, triggers a distinct alert at the dispatcher's console.[^wiki] Pressing a
radio's orange emergency button sets this flag, so the request jumps the queue for a
[voice channel](/reference/voice-channel/) even when the system is busy.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A radio pressing its emergency button, sending an emergency-flagged request that the control channel grants ahead of ordinary queued calls." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="50" width="100" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="70" y="66" text-anchor="middle" font-size="9" fill="currentColor">radio 4567</text><text x="70" y="78" text-anchor="middle" font-size="7.5" fill="currentColor">emergency btn</text>
  <line x1="122" y1="67" x2="230" y2="67" stroke="currentColor" stroke-width="1.6" marker-end="url(#emar)"/><text x="176" y="60" text-anchor="middle" font-size="8" fill="currentColor">EMERGENCY set</text>
  <rect x="232" y="50" width="120" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="292" y="71" text-anchor="middle" font-size="9" fill="currentColor">control channel</text>
  <line x1="354" y1="67" x2="440" y2="67" stroke="currentColor" stroke-width="1.6" marker-end="url(#emar)"/><text x="397" y="60" text-anchor="middle" font-size="7.5" fill="currentColor">grant first</text>
  <text x="176" y="105" text-anchor="middle" font-size="8" fill="currentColor" opacity="0.8">jumps queue ahead of ordinary calls</text>
  <defs><marker id="emar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An emergency-flagged request is granted ahead of ordinary queued calls and alerts the dispatcher.</figcaption>
</figure>

## How it works

Trunked systems queue channel requests when all voice channels are in use. An emergency
call carries a flag that raises it to the top of that queue, and on most systems it also
enables **ruthless preemption**: if no channel is free, the controller can drop a
lower-priority call in progress to free one for the emergency. The flag rides in the
service-request message a radio sends and is echoed in the resulting
[channel grant](/reference/channel-grant/), so both the system and any monitor can see
that the call is an emergency and which [talkgroup](/reference/talkgroup/) and
[radio ID](/reference/radio-id/) it belongs to.

Two related but separable events exist on most systems:

- **Emergency alarm** — a signalling-only message that registers a unit's emergency
  status at the console without necessarily opening voice. It is essentially a flagged
  status packet on the control channel.
- **Emergency call** — the flagged voice call itself, carried on a granted voice
  channel. A radio in emergency mode typically transmits both.

The emergency condition usually persists at the console until a dispatcher clears it, so
subsequent calls from the same unit may stay flagged for a while.

## Variants

- **P25** signals emergency in the group-voice service options (an Emergency bit in the
  service-options byte of the grant and in the group affiliation/service request),
  alongside priority-level bits.
- **Motorola Type II / SmartNet** uses distinct emergency command words and an
  "Emergency" talkgroup status derived from the last status bits of the ID.
- **DMR Tier III** carries the emergency indication in the control-signalling block
  ([CSBK](/reference/csbk/)) and pre-emption is a system option.

## In practice

Emergency handling is one of the sharpest differences between a trunked system and a
plain conventional channel. On a busy conventional channel an emergency simply has to
wait for a gap; on a trunked system the emergency flag reorders the queue and, with
ruthless preemption enabled, can seize a channel that is already carrying a lower-priority
call — that call is dropped and its users hear their transmission cut off. Systems also
route the emergency to a designated *emergency talkgroup* or console position, so the
right dispatcher is alerted even if the unit's normal talkgroup is monitored elsewhere.

A subtlety worth knowing is that the emergency indication is *sticky*. Once a unit
declares an emergency, the condition typically remains asserted at the console — and may
keep tagging that unit's subsequent transmissions — until a dispatcher explicitly clears
it, not merely until the unit stops talking. For a monitor this means an emergency can
color a run of calls, not just one, and the clear event is itself a signalling message
worth capturing. Accidental activations ("hot mic" or a bumped button) are common enough
that the clear/acknowledge traffic is a normal part of the picture.

## Relevance to SDR

Emergency traffic is often the most operationally significant traffic on a system, so a
scanner that can recognize the flag can surface it to the user immediately. GopherTrunk
parses the service-options and opcode fields in control-channel grants and requests, so
it can label a call as an emergency and expose which talkgroup and unit raised it —
useful for [priority scan](/reference/priority-scan/) logic and for alerting. Because the
emergency flag lives in unencrypted control-channel signalling, GopherTrunk can detect
and log an emergency even when the associated voice is encrypted; only the audio content
depends on the traffic being in the clear.

Real systems where this applies include P25 Phase 1/Phase 2 public-safety networks,
Motorola trunking, and DMR Tier III. GopherTrunk treats the emergency indication as
call metadata it recovers from the control channel, not as anything it can originate — it
is a receiver only.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on priority and emergency handling in trunked systems.
