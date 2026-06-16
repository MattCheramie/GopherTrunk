---
slug: project-25
title: Project 25 (P25)
entry_type: protocol
category: protocols
description: Project 25 (P25) is a suite of open standards for digital land-mobile radio used by public-safety agencies in North America, defining Phase 1 (FDMA) and Phase 2 (TDMA) air interfaces.
keywords: P25, Project 25, APCO-25, public safety radio, digital trunking, C4FM, IMBE, AMBE+2
aka: [P25, Project 25, APCO-25, APCO P25]
autolink: true
infobox:
  - { label: Type, value: Digital land-mobile radio (voice + data) }
  - { label: Standards body, value: TIA / APCO }
  - { label: Introduced, value: "1995 (Phase 1)" }
  - { label: Region, value: Primarily North America (public safety) }
  - { label: Access, value: "FDMA (Phase 1), TDMA (Phase 2)" }
  - { label: Channel spacing, value: 12.5 kHz (6.25 kHz equivalent in Phase 2) }
  - { label: Modulation, value: "C4FM / CQPSK (P1), H-CPM/H-DQPSK (P2)" }
  - { label: Vocoder, value: "IMBE (P1), AMBE+2 (P2)" }
  - { label: GopherTrunk support, value: "Phase 1 and Phase 2 — see Status" }
see_also: [p25-phase-1, p25-phase-2, c4fm, imbe, ambe-plus-2, trunked-radio, control-channel, tia, apco-international]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/what-is-trunking/ }
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "Project 25 (Wikipedia)", url: https://en.wikipedia.org/wiki/Project_25 }
  - { title: "GopherTrunk decoder status", url: /status.html }
---

**Project 25** (**P25**, also **APCO-25**) is a suite of open standards for
**digital land-mobile radio** developed for public-safety and government users in
North America. It defines how radios, repeaters, and trunked systems carry digital
voice and data, with the explicit goal of interoperability between equipment from
different manufacturers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="P25 Phase 1 shown as FDMA channels and Phase 2 as two-slot TDMA." xmlns="http://www.w3.org/2000/svg">
  <text x="115" y="22" text-anchor="middle" font-size="10" fill="currentColor">Phase 1 — FDMA</text>
  <g fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"><rect x="40" y="34" width="150" height="20"/><rect x="40" y="60" width="150" height="20"/><rect x="40" y="86" width="150" height="20"/></g>
  <text x="345" y="22" text-anchor="middle" font-size="10" fill="currentColor">Phase 2 — TDMA</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="270" y="50" width="150" height="40" fill="none"/><line x1="345" y1="50" x2="345" y2="90"/></g>
  <g font-size="9" fill="currentColor" text-anchor="middle"><text x="307" y="74">slot 1</text><text x="382" y="74">slot 2</text></g>
  <text x="230" y="128" text-anchor="middle" font-size="9" fill="currentColor">Phase 2 doubles capacity in the same 12.5 kHz</text>
</svg>
<figcaption>P25 spans Phase 1 (FDMA, one call per channel) and Phase 2 (TDMA, two calls per channel).</figcaption>
</figure>

## Overview

P25 was created so that police, fire, and emergency-medical agencies could replace
incompatible analog systems with a common digital standard. It specifies the air
interface (how bits travel over the radio link), the [vocoder](/reference/vocoder/)
used for voice, the trunking signalling, and inter-system interfaces. P25 comes in
two air-interface generations: [Phase 1](/reference/p25-phase-1/), which uses
[FDMA](/reference/fdma/), and [Phase 2](/reference/p25-phase-2/), which uses
[TDMA](/reference/tdma/) to double channel capacity.

## Technical characteristics

| Property | Phase 1 | Phase 2 |
|----------|---------|---------|
| Access | FDMA | TDMA (2 slots) |
| Channel | 12.5 kHz | 12.5 kHz / 2 = 6.25 kHz equiv. |
| Modulation | [C4FM](/reference/c4fm/) (and CQPSK) | H-CPM / H-DQPSK |
| Vocoder | [IMBE](/reference/imbe/) | [AMBE+2](/reference/ambe-plus-2/) |
| Symbol rate | 4800 baud (9600 bps) | — |

Both phases can be deployed conventionally or as a trunked system coordinated by a
[control channel](/reference/control-channel/).

## History

P25 standardisation began in the late 1980s under [APCO](/reference/apco-international/),
with Phase 1 documents published by the [TIA](/reference/tia/) from the mid-1990s.
Phase 2 followed to address spectrum-efficiency mandates, introducing TDMA so two
voice conversations could share one 12.5 kHz channel.

## Deployment

P25 is the dominant digital standard for U.S. state and federal public safety, and
is used by many large metropolitan systems. Systems are catalogued in databases such
as RadioReference, which list their [control-channel](/reference/control-channel/)
frequencies and [talkgroups](/reference/talkgroup/).

## Decoding it with GopherTrunk

GopherTrunk decodes both P25 Phase 1 and Phase 2: it locks the control channel,
follows channel grants to voice channels, and runs the matching vocoder to produce
audio. See the [protocol landscape lesson](/learn/protocol-landscape/) for how P25
compares with other systems, and the [Status](/status.html) page for current
coverage.
