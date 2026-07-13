---
slug: motorola-type-ii
title: Motorola Type II
entry_type: protocol
category: land-mobile-trunking
description: Motorola Type II is a classic analog trunked-radio system using a digital control channel to assign analog FM voice channels, widely deployed before the move to P25.
keywords: Motorola Type II, SmartNet, SmartZone, analog trunking, control channel, fleet, public safety legacy, 3600 baud, fleet-subfleet, Type I
aka: [Motorola Type II, SmartNet, SmartZone]
autolink: true
infobox:
  - { label: Type, value: Analog trunked radio (digital control) }
  - { label: Developer, value: Motorola }
  - { label: Era, value: 1980s–2000s }
  - { label: Access, value: FDMA with digital control channel }
  - { label: Voice, value: Analog FM }
  - { label: Control channel, value: 3600 bps digital }
  - { label: GopherTrunk support, value: Decoded }
see_also: [smartnet-smartzone, trunked-radio, control-channel, fdma, edacs, ltr, talkgroup, radio-id]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Motorola_Type_II
  - https://wiki.radioreference.com/index.php/Motorola_Trunking
---

**Motorola Type II** is a classic **analog trunked-radio** family (marketed as **SmartNet**
and, in its multi-site form, **SmartZone**) that pairs a **digital
[control channel](/reference/control-channel/)** with **analog FM** voice channels. It was
the dominant trunking technology for public-safety and business fleets through the 1980s,
1990s, and early 2000s, before the migration to digital [P25](/reference/project-25/).[^wiki]
The SmartNet/SmartZone product line and its signalling are covered in more depth under
[SmartNet / SmartZone](/reference/smartnet-smartzone/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 380 140" role="img" aria-label="A Motorola Type II digital control channel assigning analog FM voice channels on demand." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="20" width="300" height="26" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="190" y="37" text-anchor="middle" font-size="9" fill="currentColor">digital control channel (3600 bps)</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="40" y="80" width="90" height="34"/><rect x="150" y="80" width="90" height="34"/><rect x="260" y="80" width="80" height="34" fill="currentColor" fill-opacity="0.18"/></g>
  <text x="190" y="130" text-anchor="middle" font-size="8.5" fill="currentColor">analog FM voice channels (assigned on demand)</text>
  <line x1="190" y1="46" x2="300" y2="78" stroke="currentColor" stroke-dasharray="3 3" marker-end="url(#lg_motorola-type-ii)"/>
  <defs><marker id="lg_motorola-type-ii" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Motorola Type II pairs a digital control channel with analog FM voice channels assigned on demand.</figcaption>
</figure>

## Overview

In a Type II system, one frequency is designated the **control channel**, carrying a
continuous **3600 bit/s** digital data stream. Radios idle on this channel and listen for
[channel grants](/reference/channel-grant/): when a user keys up, the control channel tells
every radio in that [talkgroup](/reference/talkgroup/) which voice frequency to move to,
they retune, pass the analog FM conversation, and return to the control channel when it
ends. This is what makes the system **[trunked](/reference/trunked-radio/)** — a small pool
of RF channels is shared dynamically across many talkgroups — even though the voice itself
is ordinary analog FM. Each transmitting radio is identified by a [radio ID](/reference/radio-id/),
and Type II's flat talkgroup addressing (as opposed to the older Type I "fleet/subfleet"
scheme) is a defining trait. SmartZone extends the same idea across multiple sites with
automatic roaming, so a radio moving between towers stays affiliated to its talkgroup.

## Technical characteristics

| Property | Value |
|----------|-------|
| Access | [FDMA](/reference/fdma/) with a dedicated control channel |
| Control channel | 3600 bit/s digital data stream |
| Voice | Analog FM |
| Addressing | Flat talkgroups (Type II) vs fleet/subfleet (Type I) |
| Identities | [Talkgroups](/reference/talkgroup/) and [radio IDs](/reference/radio-id/) in control data |
| Multi-site | SmartZone (wide-area roaming) |
| Bands | VHF, UHF, 700/800/900 MHz |

## History

Motorola introduced trunking with **SmartNet** in the 1980s, evolving the addressing from
the earlier Type I fleet/subfleet model to the flat talkgroup model of **Type II** (and a
transitional "Type IIi hybrid").[^wiki][^rr] Wide-area **SmartZone** followed, linking
multiple sites into a single logical system. The family became ubiquitous through the 1990s
and early 2000s and defined how a generation of scanner users understood trunk tracking.
As agencies moved to digital, Motorola's own P25 systems and the mixed-mode SmartZone/OmniLink
lineage inherited much of the same operational vocabulary.

## Deployment

Type II ran a very large share of North American public-safety, transit, and commercial
trunked systems for two decades. Many have since been replaced by [P25](/reference/project-25/),
but analog SmartNet and SmartZone systems remain in service, particularly among business
and utility users who never needed to migrate. Its long dominance is why so much scanner
and SDR tooling — GopherTrunk included — treats Type II tracking as a baseline capability.

## Decoding it with GopherTrunk

GopherTrunk locks to the Type II **3600 bit/s control channel**, decodes its signalling to
recover talkgroup and radio-ID activity, and **follows channel grants to the analog FM voice
channels** to produce trunk-tracked audio — exactly the workflow a hardware trunking scanner
performs, done in software from raw IQ. Because the voice payload is plain FM, no vocoder is
needed; the work is in reliably demodulating and framing the control data and managing the
follow-the-grant state machine. See the [Status](/status.html) page for the current state of
Type II and SmartZone support.

## Sources

[^wiki]: [Motorola Type II](https://en.wikipedia.org/wiki/Motorola_Type_II) — Wikipedia, for the SmartNet/SmartZone analog trunking family, its 3600 bps digital control channel, and talkgroup/radio-ID signalling.
[^rr]: [Motorola Trunking](https://wiki.radioreference.com/index.php/Motorola_Trunking) — RadioReference Wiki, for Type I vs Type II addressing, control-channel behaviour, and SmartZone multi-site operation.
