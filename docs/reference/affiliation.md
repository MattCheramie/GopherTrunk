---
slug: affiliation
title: Affiliation
entry_type: term
category: trunked-radio
description: Affiliation is the process by which a radio registers with a trunked system over the control channel, letting the system route calls and track which units are active.
keywords: affiliation, registration, control channel, radio ID, trunking
aka: [affiliation]
autolink: true
infobox:
  - { label: Type, value: Trunking registration event }
  - { label: Carried on, value: Control channel }
  - { label: Reveals, value: Active radios and talkgroups }
see_also: [control-channel, radio-id, talkgroup, trunked-radio]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
external:
  - { title: "Trunked radio system (Wikipedia)", url: https://en.wikipedia.org/wiki/Trunked_radio_system }
---

**Affiliation** is the process by which a radio **registers** with a
[trunked radio](/reference/trunked-radio/) system over the
[control channel](/reference/control-channel/) when it powers on or changes
[talkgroup](/reference/talkgroup/), so the system can route calls efficiently.

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

Affiliation messages name the [radio ID](/reference/radio-id/) and talkgroup, giving the
control channel a constant stream of information about which units and groups are
active.

## Relevance to SDR

Affiliation data lets GopherTrunk populate its Radio IDs and activity views even before
a call begins, showing who is on the system.
