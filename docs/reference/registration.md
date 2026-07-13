---
slug: registration
title: Registration
entry_type: term
category: trunked-radio
description: Registration is a radio announcing its presence and identity to a trunked site over the control channel so the system knows where each unit is and can route calls.
keywords: registration, unit registration, RREQ, RREG, deregistration, affiliation, control channel, radio ID, trunking site, P25 registration
aka: [registration, unit registration, RREG]
autolink: true
infobox:
  - { label: Type, value: Trunking control event }
  - { label: Carried on, value: Control channel }
  - { label: Reveals, value: Which units are on which site }
see_also: [affiliation, trunking-site, radio-id, control-channel, roaming, talkgroup]
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

**Registration** is the process by which a radio announces its presence and identity to
a [trunking site](/reference/trunking-site/) over the
[control channel](/reference/control-channel/), so the system knows the unit is on the
air and which site it is using.[^wiki] It is closely related to
[affiliation](/reference/affiliation/): registration tells the network *where a unit is*,
while affiliation tells it *which [talkgroup](/reference/talkgroup/) that unit wants to
follow* — and on many systems a radio does both when it powers on.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A radio sending a registration request to a trunking site, which records the unit as present and returns a registration acknowledgement." xmlns="http://www.w3.org/2000/svg">
  <rect x="25" y="43" width="95" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="72" y="64" text-anchor="middle" font-size="9" fill="currentColor">radio 4567</text>
  <rect x="340" y="43" width="95" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="387" y="60" text-anchor="middle" font-size="8.5" fill="currentColor">site 3</text><text x="387" y="71" text-anchor="middle" font-size="7.5" fill="currentColor">registry</text>
  <line x1="122" y1="54" x2="338" y2="54" stroke="currentColor" stroke-width="1.1" marker-end="url(#rgar)"/><text x="230" y="48" text-anchor="middle" font-size="8.5" fill="currentColor">register: unit 4567</text>
  <line x1="338" y1="68" x2="122" y2="68" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#rgar)"/><text x="230" y="82" text-anchor="middle" font-size="8.5" fill="currentColor">acknowledged</text>
  <defs><marker id="rgar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Registration records a unit as present on a specific site so the system can route calls to it.</figcaption>
</figure>

## How it works

When a radio powers on, moves to a new site, or has been idle, it sends a
**registration request** on the control channel carrying its
[radio ID](/reference/radio-id/) (and often a system/WACN identifier so the site can
confirm the unit belongs to this network). The site validates the unit — it may accept,
deny, or refuse it — and returns a **registration response**. Once registered, the unit
is tracked at that site; the system's home-site knowledge is what lets it deliver a call
to a unit that has moved. When a radio leaves or powers down cleanly it may send a
**deregistration**, though many just drop off and time out of the registry.

Registration and affiliation are often bundled: a P25 radio typically performs a *unit
registration* and then a *group affiliation* in quick succession, so a monitor sees both
events shortly after a radio comes up. On a multi-site network, the sequence repeats as
the unit moves between sites — see [roaming](/reference/roaming/).

## Variants

- **P25** defines explicit Unit Registration Request/Response and Deregistration
  messages on the control channel, keyed on the unit's Source ID and the system's
  WACN/System ID.
- **Motorola Type II / SmartNet** performs registration implicitly through its affiliation
  and "radio check" signalling rather than a formally named registration transaction.
- **DMR Tier III** carries registration in its control-signalling blocks
  ([CSBK](/reference/csbk/)).

## In practice

Registration is the quiet backbone of trunking mobility. On a wide-area network the
system must always know which site a unit is on so it can deliver that unit's calls to the
right place; registration is how it learns this, and re-registration as a unit
[roams](/reference/roaming/) is how it keeps the knowledge current. A registration is also
a point at which the system can *deny* a unit — an out-of-service or stolen radio can be
refused, and a monitor will then see a denial rather than an acceptance, which is itself
telling.

The timing matters, too. Radios re-register periodically even when stationary (a
keep-alive so the system does not age them out), and they register on any event that
might have moved them — power-on, coming out of a coverage gap, or switching to a new
selected talkgroup. The result is that a lightly used system still has a steady trickle of
registration traffic on its control channel, independent of any voice calls, and that
trickle is exactly what reveals the fleet's composition. Deregistration, where a radio
announces it is leaving, is less reliably transmitted than registration — many radios just
drop off and let the site's timer expire — so a monitor generally learns about arrivals
more crisply than departures.

## Relevance to SDR

Registration traffic is a rich, continuous source of unit-presence metadata that a
scanner can harvest without any voice call ever occurring. By parsing registration and
deregistration messages on the control channel, GopherTrunk can build a live roster of
which [radio IDs](/reference/radio-id/) are active on a
[trunking site](/reference/trunking-site/) and when they came and went — the same data
that populates its unit-activity and site views. Because registration is unencrypted
control-channel signalling, this works regardless of whether the system's voice traffic
is encrypted.

On a multi-site network, watching registrations across sites is how a monitor infers
[roaming](/reference/roaming/): the same unit ID registering at a new site indicates it
has physically moved. GopherTrunk treats all of this as read-only control-channel
information — it observes registrations, it does not perform them. Real systems where
this applies include P25 Phase 1/Phase 2 and DMR Tier III public-safety and utility
networks.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on unit registration and affiliation over the control channel.
