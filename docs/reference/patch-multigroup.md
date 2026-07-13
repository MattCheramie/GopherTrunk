---
slug: patch-multigroup
title: Patch & multigroup
entry_type: term
category: trunked-radio
description: A patch or multigroup is a dispatcher-created link that joins several talkgroups so one transmission is heard by all of them, announced on the control channel.
keywords: patch, multigroup, multi-group, megagroup, talkgroup patch, dispatcher patch, simulselect, control channel, trunking, P25 patch
aka: [patch, multigroup, multi-group, megagroup, simulselect]
autolink: true
infobox:
  - { label: Type, value: Talkgroup grouping }
  - { label: Created by, value: Dispatcher / console }
  - { label: Announced on, value: Control channel }
see_also: [talkgroup, group-call, channel-grant, control-channel, private-call, radio-id]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

A **patch** (or **multigroup**) is a dispatcher-created link that temporarily joins two or
more [talkgroups](/reference/talkgroup/) so that a single transmission is heard by all of
them at once.[^wiki] The trunking system announces the patch on the
[control channel](/reference/control-channel/), and thereafter a
[group call](/reference/group-call/) to any member talkgroup is delivered to every group
in the patch. It is how a dispatcher glues separate units — say fire and EMS, or two
adjacent districts — into one temporary net without reprogramming any radio.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A dispatcher console linking talkgroups 101, 102 and 205 into one patch, so a call to any of them reaches all three." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="48" width="110" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="75" y="69" text-anchor="middle" font-size="9" fill="currentColor">dispatcher</text>
  <line x1="132" y1="65" x2="180" y2="65" stroke="currentColor" stroke-width="1.1" marker-end="url(#pmar)"/><text x="156" y="58" text-anchor="middle" font-size="7.5" fill="currentColor">patch</text>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <ellipse cx="320" cy="65" rx="120" ry="46" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3"/>
    <rect x="212" y="52" width="54" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="239" y="69">TG 101</text>
    <rect x="293" y="52" width="54" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="320" y="69">TG 102</text>
    <rect x="374" y="52" width="54" height="26" rx="4" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="401" y="69">TG 205</text>
  </g>
  <text x="320" y="120" text-anchor="middle" font-size="8" fill="currentColor" opacity="0.8">one call reaches all three</text>
  <defs><marker id="pmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A patch links several talkgroups so one transmission is delivered to every member group.</figcaption>
</figure>

## How it works

A dispatcher builds a patch at the console by selecting the talkgroups to join. The
console tells the trunking controller, which distributes the patch definition to the
sites and advertises it on the control channel. When any member group keys up, the
controller issues [channel grants](/reference/channel-grant/) so that all member groups —
which may be on different [voice channels](/reference/voice-channel/) at different sites —
receive the same audio. The patch persists until the dispatcher tears it down; it is a
temporary, console-controlled grouping, not a change to any radio's programming.

A closely related construct is the **multigroup** (sometimes *megagroup*): a
pre-defined "supergroup" ID that expands to a fixed set of member talkgroups. A patch is
typically ad-hoc and dispatcher-built on the fly, whereas a multigroup is configured in
the system and invoked by a single ID, but both produce the same on-air effect — one
transmission, many groups.

## In practice

For a listener, patches are important because they change *who is talking to whom*. A
call that appears on TG 205 may actually be part of a conversation that started on TG
101; without knowing the patch is in place, the two look like unrelated traffic. Systems
announce patch/multigroup membership on the control channel precisely so radios (and
consoles) stay consistent, which means a monitor can reconstruct the grouping too.
Patches are common during large incidents, mutual-aid operations, and events that pull
together agencies that normally run on separate talkgroups.

A patch can also span audio types and even systems. A console may patch a trunked
talkgroup to a [conventional](/reference/conventional-radio/) channel, to a telephone
line, or to a talkgroup on a neighboring system, acting as a bridge so users who could
never otherwise hear each other share one net. When that happens, the identity a monitor
sees on the trunked side is only part of the conversation — the other participants live
outside the trunked signalling entirely. This is why patches are as much an
interoperability tool as a convenience: during a multi-agency incident, a single patch can
be the only thing letting fire, police, and EMS on three different radio systems talk on
one virtual channel.

A further wrinkle is **dynamic regrouping**, a related console power in which the system
temporarily reassigns radios to a new group — sometimes forcing selected units onto a
special talkgroup and even locking their selector so they cannot leave it. Where a patch
links existing groups so their traffic is shared, dynamic regrouping moves units into a
newly imposed group. Both are announced over the control channel so the affected radios
comply, and both are temporary states a dispatcher sets up and later tears down. For a
monitor the two can look similar on the air, and telling them apart depends on reading the
specific opcodes involved.

## Relevance to SDR

Because patch and multigroup definitions are broadcast on the control channel,
GopherTrunk can parse them and understand that several
[talkgroups](/reference/talkgroup/) are temporarily linked. That lets it present a call
correctly — showing that traffic on one talkgroup is being delivered to the others in the
patch — rather than as unrelated events. It also helps a scanner avoid double-tasking
receivers: if three patched groups share the same audio, the monitor need only follow one
[voice channel](/reference/voice-channel/) to hear the whole net.

Real systems with these features include P25 (patch/regroup and dynamic regrouping) and
Motorola trunking (dispatcher patch and multigroup). GopherTrunk reads the grouping
passively from control-channel signalling; it recovers the *structure* of the
conversation as metadata, and the audio it can play back still depends on the traffic
being unencrypted.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on dispatcher patches and multigroup/regroup features in trunked systems.
