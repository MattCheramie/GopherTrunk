---
slug: private-call
title: Private call
entry_type: term
category: trunked-radio
description: A private call is a one-to-one unit-to-unit call on a trunked system, granted by the control channel to a pair of radio IDs rather than to a talkgroup.
keywords: private call, individual call, unit-to-unit call, unit call, selective call, radio ID, trunking, P25 unit call, DMR private call
aka: [private call, individual call, unit call, unit-to-unit call]
autolink: true
infobox:
  - { label: Type, value: One-to-one voice call }
  - { label: Addressed to, value: A pair of radio IDs }
  - { label: Granted by, value: Control channel }
see_also: [group-call, radio-id, channel-grant, control-channel, talkgroup, emergency-call]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

A **private call** is a one-to-one, unit-to-unit voice call on a trunked system:
instead of addressing a [talkgroup](/reference/talkgroup/), one radio calls a single
other radio, and the [control channel](/reference/control-channel/) issues a
[channel grant](/reference/channel-grant/) naming the two
[radio IDs](/reference/radio-id/) rather than a group.[^wiki] It is the trunked
equivalent of a phone call between two handsets, distinct from the many-to-many
[group call](/reference/group-call/) that most dispatch traffic uses.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A control channel granting a private call between radio 4567 and radio 8890, with both radios retuning to the same voice channel while other radios are not addressed." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="20" width="150" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="95" y="39" text-anchor="middle" font-size="9" fill="currentColor">control channel</text>
  <line x1="95" y1="50" x2="95" y2="80" stroke="currentColor" stroke-width="1.1" marker-end="url(#pcar)"/><text x="200" y="66" text-anchor="middle" font-size="8.5" fill="currentColor">grant: 4567 → 8890 (private)</text>
  <rect x="40" y="88" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="95" y="107" text-anchor="middle" font-size="9" fill="currentColor">radio 4567</text>
  <rect x="300" y="88" width="110" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="355" y="107" text-anchor="middle" font-size="9" fill="currentColor">radio 8890</text>
  <line x1="152" y1="103" x2="298" y2="103" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3"/><text x="225" y="98" text-anchor="middle" font-size="8" fill="currentColor">both retune, no group</text>
  <defs><marker id="pcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A private call is granted to two radio IDs, not a talkgroup, and only those two radios follow the call.</figcaption>
</figure>

## How it works

A private call begins when one radio requests a connection to a specific target
[radio ID](/reference/radio-id/). On some systems the target's radio first
acknowledges (an alert or "call setup" handshake) before audio flows; on others the
call sets up immediately. Either way the trunking controller allocates a
[voice channel](/reference/voice-channel/) and sends a
[channel grant](/reference/channel-grant/) that carries **two** unit addresses — the
source and the destination — rather than a single group address. Both radios retune to
the granted channel (and timeslot, on [TDMA](/reference/tdma/) systems), and no other
radio in the fleet un-mutes because the call is not addressed to any group they
monitor.

The distinguishing feature, from a monitor's point of view, is the addressing. A
[group call](/reference/group-call/) grant reads "talkgroup 101 → channel 3"; a private
call grant reads "unit 4567 → unit 8890 → channel 3". That difference in the grant's
opcode and fields is exactly what lets a decoder classify the call before any audio is
even present.

## Variants

- **P25** carries private calls as *unit-to-unit voice service* messages on the control
  channel, with the grant naming a source and target Source/Destination Unit ID. A
  separate answer-request/response handshake can precede the grant.
- **DMR Tier III** and Motorola/EF Johnson systems support **individual calls** the same
  way, distinguishing them from group calls by the call-type bits in the signalling
  block ([CSBK](/reference/csbk/) on DMR).
- Analog SmartNet/[Motorola Type II](/reference/motorola-type-ii/) systems also carry
  private (interconnect and unit-to-unit) calls, flagged by the command word.

Private calls are frequently short — a supervisor reaching one unit, a status query —
and on many systems they can be encrypted end-to-end even when group traffic is in the
clear.

## In practice

The setup handshake is what distinguishes the two common private-call styles. In an
*availability-checked* private call, the system first pages the target unit and waits for
it to answer before allocating a voice channel; if the target does not respond, the call
is never granted, so a monitor may see the request and the page but no grant. In a
*direct* private call, the system grants a channel immediately and the target simply
un-mutes when it hears its own ID, at the cost of possibly keying up a channel for a unit
that is switched off. Systems also differ in whether a private call ties up a full
[voice channel](/reference/voice-channel/) for two units — an inefficient use of a shared
resource — which is one reason some agencies restrict or disable the feature and route
one-to-one traffic through telephone interconnect or data messaging instead.

For a listener, the practical consequence is that private calls are sparse, bursty, and
easy to miss: they are not tied to a talkgroup on any scan list, they can be brief, and
they may be encrypted. Catching them at all depends on parsing the private-call opcodes
rather than only following group traffic, which is why a trunk-tracking scanner treats
them as a distinct call class.

## Relevance to SDR

For a scanner, private calls matter because they are the traffic that *isn't* on the
talkgroup list. A monitor watching only group calls will miss unit-to-unit
conversations entirely unless it also parses private-call grants. GopherTrunk reads the
grant opcodes on the control channel and classifies each call as group, private, or
[emergency](/reference/emergency-call/); when it sees a private-call grant it can task a
receiver to the assigned channel and log both radio IDs involved, populating its unit
activity view with who called whom. This is the same mechanism it uses for group calls —
follow the grant, tune the [voice channel](/reference/voice-channel/) — but the metadata
recovered is a pair of units rather than a group.

As always, GopherTrunk decodes what is transmitted in the clear or with recoverable
scrambling; a private call that is encrypted end-to-end will still be *detected and
logged* (the grant is unencrypted control-channel signalling), but its audio will not be
intelligible. Real systems that carry private calls include P25 Phase 1 and Phase 2,
DMR Tier III, and legacy Motorola trunking.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on unit-to-unit (private/individual) calls versus group calls on trunked systems.
