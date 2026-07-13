---
slug: project-25
title: Project 25 (P25)
entry_type: protocol
category: land-mobile-trunking
description: Project 25 (P25) is a suite of open standards for digital land-mobile radio used by public-safety agencies in North America, defining Phase 1 (FDMA) and Phase 2 (TDMA) air interfaces.
keywords: P25, Project 25, APCO-25, public safety radio, digital trunking, C4FM, CQPSK, IMBE, AMBE+2, TIA-102, common air interface, NAC, WACN, System ID, RFSS, TSBK
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
see_also: [p25-phase-1, p25-phase-2, p25-cai, c4fm, cqpsk, imbe, ambe-plus-2, network-access-code, wacn, system-id, rfss, trunked-radio, control-channel, tsbk, tia, apco-international]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
related_reading:
  - { title: "SDR Internals, Part 10: Protocol decoders & state machines", url: /blog/deep-dives/sdr-internals-10-protocol-decoders-state-machines/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://www.cisa.gov/safecom/p25
external:
  - { title: "GopherTrunk decoder status", url: /status.html }
---

**Project 25** (**P25**, also **APCO-25**) is a suite of open standards for
**digital land-mobile radio** developed for public-safety and government users in
North America. It defines how radios, repeaters, and trunked systems carry digital
voice and data, with the explicit goal of interoperability between equipment from
different manufacturers.[^wiki] The standards are published by the
[TIA](/reference/tia/) as the TIA-102 series and were driven by the requirements of
[APCO International](/reference/apco-international/), the association of public-safety
communications officials.[^cisa]

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
incompatible analog and proprietary systems with a common, openly documented digital
standard. Rather than a single document, P25 is a *suite*: it specifies the
[Common Air Interface](/reference/p25-cai/) (how bits travel over the radio link),
the [vocoder](/reference/vocoder/) used for voice, the trunking control signalling,
inter-RF-subsystem interfaces (ISSI), console and network interfaces, and encryption.
Because every layer is published, radios from Motorola, L3Harris, Tait, Kenwood, and
others can interoperate on the same system.

P25 comes in two air-interface generations that share the same signalling and
addressing: [Phase 1](/reference/p25-phase-1/), which uses
[FDMA](/reference/fdma/) with one call per 12.5 kHz channel, and
[Phase 2](/reference/p25-phase-2/), which uses two-slot [TDMA](/reference/tdma/) to
carry two calls in the same 12.5 kHz for doubled spectrum efficiency. A single system
often mixes both: a Phase 1 [C4FM](/reference/c4fm/) control channel directs traffic
to Phase 2 voice channels.

## Technical characteristics

| Property | Phase 1 | Phase 2 |
|----------|---------|---------|
| Access | FDMA | TDMA (2 slots) |
| Channel | 12.5 kHz | 12.5 kHz / 2 = 6.25 kHz equiv. |
| Modulation | [C4FM](/reference/c4fm/) / [CQPSK](/reference/cqpsk/) | H-CPM / H-DQPSK |
| Vocoder | [IMBE](/reference/imbe/) | [AMBE+2](/reference/ambe-plus-2/) |
| Symbol rate | 4800 baud (9600 bps) | 6000 symbols/s per slot |
| Voice FEC | Golay, Hamming, Reed–Solomon, [trellis](/reference/trellis-coded-modulation/) | as Phase 2 profile |
| Control signalling | [TSBK](/reference/tsbk/) on the control channel | Phase 1 control channel |

Every P25 unit and system carries a layered set of identifiers that the control
channel broadcasts and the decoder reads: the [Network Access Code](/reference/network-access-code/)
(NAC), a 12-bit "digital squelch" that separates co-channel systems; the
[WACN](/reference/wacn/) and [System ID](/reference/system-id/), which together
uniquely name a network worldwide; the [RFSS](/reference/rfss/) and site number,
which locate a specific RF subsystem; and per-call fields such as source
[radio ID](/reference/radio-id/) and destination [talkgroup](/reference/talkgroup/).
Both phases can be deployed conventionally (fixed frequency) or as a trunked system
coordinated by a [control channel](/reference/control-channel/).

## History

P25 standardisation began in the late 1980s under [APCO](/reference/apco-international/),
which set user requirements, with the [TIA](/reference/tia/) drafting the TIA-102
documents; Phase 1 was published from the mid-1990s.[^wiki] The suite deliberately
reused the [IMBE](/reference/imbe/) vocoder and a spectrally efficient 4-level FSK
([C4FM](/reference/c4fm/)) modulation so that legacy 12.5 kHz channel plans could be
kept. Phase 2 followed in the 2000s to meet FCC narrowbanding and spectrum-efficiency
mandates, adopting two-slot [TDMA](/reference/tdma/) and the half-rate
[AMBE+2](/reference/ambe-plus-2/) vocoder so that two conversations could share one
12.5 kHz channel.[^cisa]

## Deployment

P25 is the dominant digital standard for U.S. state, county, and federal public
safety, and is used by many large metropolitan systems as well as by users in
Australia, Canada, and elsewhere. Statewide interoperability networks (for example
large state radio systems serving thousands of agencies) are typically P25 trunked
systems with dozens of sites tied together by a wide-area network. Systems are
catalogued in databases such as [RadioReference](/reference/radioreference/), which
list their [control-channel](/reference/control-channel/) frequencies, NACs, System
IDs, and [talkgroups](/reference/talkgroup/). Much public-safety traffic is now
encrypted (commonly AES-256), which is audible to a scanner only as the system
metadata, not as clear voice.

## Decoding it with GopherTrunk

GopherTrunk decodes both P25 Phase 1 and Phase 2. It locks the control channel,
reads the [TSBK](/reference/tsbk/) grants and the
[NAC](/reference/network-access-code/) / [WACN](/reference/wacn/) /
[System ID](/reference/system-id/) / [RFSS](/reference/rfss/) identity broadcasts,
follows [channel grants](/reference/channel-grant/) to the assigned voice channel
(and, in Phase 2, the assigned slot), and runs the matching
[IMBE](/reference/imbe/) or [AMBE+2](/reference/ambe-plus-2/) vocoder to produce
audio. It renders clear and (with a known key) some scrambled traffic, but it does
not defeat keyed encryption such as AES or DES — encrypted calls are logged with
their metadata only. See the [protocol landscape lesson](/learn/rf-sdr/protocol-landscape/)
for how P25 compares with other systems, and the [Status](/status.html) page for
current coverage.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, for P25 history, the Phase 1/Phase 2 air interfaces, the identifier hierarchy, and the TIA/APCO standardisation.
[^cisa]: [P25 (Project 25)](https://www.cisa.gov/safecom/p25) — CISA SAFECOM, for the public-safety interoperability rationale, the TIA-102 standards suite, and the Phase 1 to Phase 2 spectrum-efficiency progression.
