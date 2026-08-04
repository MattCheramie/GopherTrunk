---
slug: tetra-aach
title: TETRA AACH
entry_type: term
category: trunked-radio
description: "The TETRA Access Assignment Channel (AACH) is a 14-bit ACCESS-ASSIGN field, RM(30,14)-coded and scrambled, carried in every downlink slot; its downlink usage marker tells a receiver, slot by slot, whether the slot carries control signalling or traffic."
keywords: TETRA AACH, access assignment channel, ACCESS-ASSIGN, downlink usage marker, RM 30 14, EN 300 392-2 21.4.7, TETRA slot usage, dynamic MCCH sharing
aka: [AACH, "Access Assignment Channel", ACCESS-ASSIGN]
autolink: true
infobox:
  - { label: Payload, value: 14 type-1 bits }
  - { label: Coding, value: "RM(30,14) + colour-code scramble" }
  - { label: Carried in, value: Every downlink slot }
  - { label: Spec, value: "ETSI EN 300 392-2 §21.4.7 / §8.3.1.1" }
see_also: [tetra, tetra-rm-30-14, tetra-scrambler, tetra-logical-channels, tetra-burst-formats, tetra-traffic-slot-mapping, control-channel, reed-muller-code]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Muller_code
---

The **TETRA AACH** (**Access Assignment Channel**) is the short signalling field carried in
the centre of *every* [TETRA](/reference/tetra/) downlink slot, whatever else the slot is
doing.[^tetra] It carries a 14-bit **ACCESS-ASSIGN** PDU that names how the slot is being
used — idle, control signalling, or a traffic call — and, on the uplink side, how random
access to the slot is granted. Because it is present in all four TDMA slots regardless of
their content, decoding the AACH frame by frame is how a receiver maps which slot is
currently the [control channel](/reference/control-channel/) and which slots hold calls.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A TETRA downlink slot with its two AACH half-fields sitting either side of the training sequence, together carrying a 14-bit ACCESS-ASSIGN PDU that is RM(30,14) coded and scrambled into 30 type-5 bits; a 2-bit header and two 6-bit fields make up the 14 bits, and field 1 yields the downlink usage marker." xmlns="http://www.w3.org/2000/svg">
  <rect x="16" y="24" width="150" height="24" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="91" y="40" text-anchor="middle" font-size="8.5" fill="currentColor">BKN1 · 216 bits</text>
  <rect x="166" y="24" width="34" height="24" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="183" y="40" text-anchor="middle" font-size="7" fill="currentColor">AACH</text>
  <rect x="200" y="24" width="44" height="24" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="222" y="40" text-anchor="middle" font-size="7" fill="currentColor">train</text>
  <rect x="244" y="24" width="34" height="24" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="261" y="40" text-anchor="middle" font-size="7" fill="currentColor">AACH</text>
  <rect x="278" y="24" width="176" height="24" rx="3" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="366" y="40" text-anchor="middle" font-size="8.5" fill="currentColor">BKN2 · 216 bits</text>
  <g font-size="8" fill="currentColor">
    <rect x="60" y="80" width="40" height="22" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="80" y="95" text-anchor="middle" font-size="7">hdr 2</text>
    <rect x="100" y="80" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="130" y="95" text-anchor="middle" font-size="7">field1 6</text>
    <rect x="160" y="80" width="60" height="22" fill="none" stroke="currentColor" stroke-width="1"/>
    <text x="190" y="95" text-anchor="middle" font-size="7">field2 6</text>
  </g>
  <path d="M225 102 L225 116 L300 116" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="308" y="119" font-size="8" fill="currentColor">RM(30,14) + scramble → 30 bits</text>
  <text x="60" y="138" font-size="8" fill="currentColor">field1 (header ≠ 0) = downlink usage marker: control / traffic / idle</text>
</svg>
<figcaption>The AACH occupies two half-fields either side of a slot's training sequence; a 2-bit header and two 6-bit fields carry 14 information bits that are RM(30,14) coded and scrambled to 30 type-5 bits, and field 1 yields the downlink usage marker.</figcaption>
</figure>

## How it works

The 14 information bits split into a **2-bit access-assignment header** followed by two
**6-bit fields**. The header selects how the fields are read. Header value 0 leaves the
downlink as common control by definition; the other three header values (`DLF1_ULCA`,
`DLF1_ULAO`, `DLF1_ULF1`) all place the *downlink usage marker* in field 1, while field 2
carries uplink-access information a downlink monitor does not need.

Those 14 bits are protected by a short block code rather than the convolutional chain the
other signalling channels use. Per §8.3.1.1 the AACH skips RCPC and interleaving entirely:
the [RM(30,14)](/reference/tetra-rm-30-14/) Reed–Muller code takes the 14 type-1 bits
straight to 30 type-5 bits, which are then [scrambled](/reference/tetra-scrambler/) with the
cell colour code. The receiver reverses that — descramble, then the maximum-likelihood
RM(30,14) decode — and reports the corrected 14 bits plus a Hamming-distance metric that
doubles as a confidence gate. GopherTrunk also runs a soft-decision variant of the decode
that recovers the marker on marginal bursts the hard decoder mis-corrects.

## The downlink usage marker

The field-1 value is the AACH's most useful output. It is a per-slot enumeration:

| Marker | Meaning |
| --- | --- |
| 0 | Unallocated (idle) |
| 1 | Assigned control |
| 2 | Common control |
| 3 | Reserved |
| ≥ 4 | Traffic — the value itself identifies the call occupying the slot |

A marker of 1 or 2 means the slot is carrying control signalling; a marker of 4 or greater
means it holds a call, and the marker *value* is the identifier that call was granted. That
makes the AACH the demux key a voice follower routes by: the AACH decodes in every downlink
slot, and a granted call's usage marker matches the marker carried in its
[grant](/reference/channel-grant/) — the reliable way to keep concurrent same-carrier calls
apart (see [traffic slot mapping](/reference/tetra-traffic-slot-mapping/)). It is also what
lets a receiver follow a Single Carrier Base Station running dynamic MCCH sharing, where any
of the four TDMA slots can act as the control channel at a given moment rather than a fixed
slot 1.

## Relevance to SDR

`internal/radio/tetra/aach.go` decodes the 14 recovered bits into an `AccessAssign` PDU and
exposes `DownlinkUsage`, `IsControlChannel`, and `IsTraffic`; the coding chain lives in
`channel_coding.go` (`EncodeAACH` / `DecodeAACH` / `DecodeAACHSoft`). Because the AACH sits
in the centre of the slot next to the training sequence, a receiver that has already
correlated the burst has the AACH dibits in hand for free, so reading the usage marker adds
almost nothing to the per-slot cost while giving the trunk-following engine a truthful,
slot-by-slot picture of what the carrier is carrying. The normal-frame interpretation here
covers frames 1–17; frame 18 carries the broadcast block and can reinterpret the fields,
which does not affect the steady-state downlink usage marker.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA air interface and its TDMA slot structure.
[^rm]: [Reed–Muller code](https://en.wikipedia.org/wiki/Reed%E2%80%93Muller_code) — Wikipedia, on the block-code family that protects the AACH.
