---
slug: affiliation
title: Affiliation
entry_type: term
category: trunked-radio
description: Affiliation is the process by which a radio registers with a trunked system over the control channel, letting the system route calls and track which units are active.
keywords: affiliation, registration, control channel, radio ID, trunking, group affiliation, roaming
aka: [affiliation]
autolink: true
infobox:
  - { label: Type, value: Trunking registration event }
  - { label: Carried on, value: Control channel }
  - { label: Reveals, value: Active radios and talkgroups }
see_also: [control-channel, radio-id, talkgroup, trunked-radio, registration, roaming, group-call, neighbor-site]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Trunked_radio_system
  - https://en.wikipedia.org/wiki/Project_25
---

**Affiliation** is the process by which a radio **registers** with a
[trunked radio](/reference/trunked-radio/) system over the
[control channel](/reference/control-channel/) when it powers on or changes
[talkgroup](/reference/talkgroup/), so the system can route calls efficiently.[^wiki]
Affiliation is closely related to [registration](/reference/registration/): registration
tells the system *which site* a radio is on, while group affiliation tells it *which
talkgroup* the radio currently wants.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A radio sending a registration request to the system, which records it as affiliated." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="45" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="75" y="66" text-anchor="middle" font-size="9" fill="currentColor">radio 4567</text>
  <rect x="340" y="45" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="385" y="66" text-anchor="middle" font-size="9" fill="currentColor">system</text>
  <line x1="122" y1="56" x2="338" y2="56" stroke="currentColor" stroke-width="1.1" marker-end="url(#afar)"/><text x="230" y="50" text-anchor="middle" font-size="8.5" fill="currentColor">"register on TG 101"</text>
  <line x1="338" y1="70" x2="122" y2="70" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#afar)"/><text x="230" y="84" text-anchor="middle" font-size="8.5" fill="currentColor">acknowledged · affiliated</text>
  <defs><marker id="afar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Affiliation is a radio registering with the system over the control channel so calls can be routed to it.</figcaption>
</figure>

## How it works

Affiliation messages name the source [radio ID](/reference/radio-id/) and the talkgroup,
giving the control channel a constant stream of information about which units and groups
are active. The controller keeps a live table of affiliated units per talkgroup, which it
uses to decide whether a group call is worth granting a channel (if nobody at the site is
affiliated to a talkgroup, the system need not assign a channel there) and, on
[multisite](/reference/multisite-trunking/) networks, which sites to key up so that
[roaming](/reference/roaming/) members hear the call.

A radio typically affiliates when it powers on, when the user turns the talkgroup
selector, and again after moving to a new site's control channel — the affiliation and
[registration](/reference/registration/) exchange is what makes a call follow the user
around the system.

## Variants

- **Group affiliation** — announces the talkgroup the radio has selected.
- **Unit registration** — announces the radio's presence/location at a site, independent
  of any talkgroup.
- **De-registration** — some systems signal when a radio leaves or powers down, so stale
  entries expire cleanly.

## In practice

Affiliation traffic is a rich passive intelligence source: even before anyone speaks, the
control channel reveals which radio IDs are on the air and which talkgroups they favour.
Analysts use affiliation patterns to map a fleet's structure and estimate how many units a
site is carrying. Because affiliation is unencrypted control signalling, it is visible even
on systems whose voice traffic is [encrypted](/reference/encryption/).

## Relevance to SDR

Affiliation data lets GopherTrunk populate its radio-ID and activity views even before a
call begins, showing who is on the system and which groups they are watching. Tracking
affiliations over time also helps GopherTrunk distinguish active talkgroups worth
following from dormant ones.

## Sources

[^wiki]: [Trunked radio system](https://en.wikipedia.org/wiki/Trunked_radio_system) — Wikipedia, on radio registration/affiliation over the control channel.
[^p25]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on P25 unit registration and group affiliation signalling.
