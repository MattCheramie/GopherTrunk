---
slug: tetra-mac-pdu
title: TETRA MAC PDU
entry_type: term
category: trunked-radio
description: "The TETRA lower-MAC PDUs — MAC-RESOURCE, MAC-FRAG/END, MAC-BROADCAST and MAC-SUPPLEMENTARY — frame the Layer-3 TM-SDU inside a decoded slot; MAC-RESOURCE also carries the channel-allocation element that assigns a call its traffic carrier and timeslot."
keywords: TETRA MAC PDU, MAC-RESOURCE, MAC-BROADCAST, SYSINFO, channel allocation element, TM-SDU, MAC-FRAG, MAC-END, EN 300 392-2 21.4
aka: [MAC-RESOURCE, "TETRA lower MAC", "MAC PDU"]
autolink: true
infobox:
  - { label: PDU type field, value: 2 bits (opens every MAC PDU) }
  - { label: Types, value: "RESOURCE, FRAG/END, BROADCAST, SUPPLEMENTARY" }
  - { label: Frames, value: "TM-SDU (LLC/L3 PDU) + optional grant" }
  - { label: Spec, value: "ETSI EN 300 392-2 §21.4" }
see_also: [tetra, tetra-logical-channels, tetra-llc, tetra-cmce-mle-pdu, channel-grant, control-channel, tetra-aach, tetra-traffic-slot-mapping]
cite_urls:
  - https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio
  - https://en.wikipedia.org/wiki/Medium_access_control
---

The **TETRA MAC PDU** is the unit the lower Medium Access Control layer emits once the
physical layer has channel-decoded a slot.[^tetra] The type-1 bits recovered from a
signalling [logical channel](/reference/tetra-logical-channels/) (SCH/F or SCH/HD) are *not*
a Layer-3 message directly — they are a MAC PDU that frames the **TM-SDU** (the
[LLC](/reference/tetra-llc/) / Layer-3 payload) and, for the resource PDU, carries the
**channel-allocation element** that assigns a call its physical traffic resource.[^mac] A
[CMCE](/reference/tetra-cmce-mle-pdu/) voice grant is therefore reached only by parsing the
MAC header, lifting out the TM-SDU, and reading the allocation element beside it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 470 150" role="img" aria-label="A decoded slot's bits open with a 2-bit MAC PDU type; when it is MAC-RESOURCE the PDU carries header flags, an address element, an optional channel-allocation element giving carrier and timeslot, and then the TM-SDU that continues up to the LLC and CMCE layers." xmlns="http://www.w3.org/2000/svg">
  <rect x="14" y="34" width="52" height="26" fill="currentColor" fill-opacity="0.28" stroke="currentColor" stroke-width="1.1"/>
  <text x="40" y="47" text-anchor="middle" font-size="7" fill="currentColor">type 2</text>
  <text x="40" y="56" text-anchor="middle" font-size="6.5" fill="currentColor">RESOURCE</text>
  <rect x="66" y="34" width="70" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="101" y="50" text-anchor="middle" font-size="7" fill="currentColor">hdr flags</text>
  <rect x="136" y="34" width="70" height="26" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <text x="171" y="50" text-anchor="middle" font-size="7" fill="currentColor">address</text>
  <rect x="206" y="34" width="96" height="26" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.1"/>
  <text x="254" y="47" text-anchor="middle" font-size="7" fill="currentColor">chan-alloc</text>
  <text x="254" y="56" text-anchor="middle" font-size="6.5" fill="currentColor">carrier+slot</text>
  <rect x="302" y="34" width="154" height="26" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/>
  <text x="379" y="50" text-anchor="middle" font-size="7.5" fill="currentColor">TM-SDU → LLC → CMCE</text>
  <path d="M254 60 L254 84" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="254" y="98" text-anchor="middle" font-size="7.5" fill="currentColor">physical resource of the granted call</text>
  <path d="M379 60 L379 84" fill="none" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2"/>
  <text x="379" y="98" text-anchor="middle" font-size="7.5" fill="currentColor">the call's parties + type</text>
  <text x="235" y="128" text-anchor="middle" font-size="8" fill="currentColor">a grant = allocation element (MAC) + party SSIs (CMCE), same PDU</text>
</svg>
<figcaption>A MAC-RESOURCE PDU frames the TM-SDU and, when a call is assigned, carries a channel-allocation element giving the traffic carrier and timeslot; the grant a follower acts on is assembled from that element plus the CMCE parties riding in the same PDU.</figcaption>
</figure>

## The PDU types

Every downlink MAC PDU opens with a 2-bit **MAC PDU type** (§21.4.1):

| Value | PDU | Purpose |
| --- | --- | --- |
| 0 | MAC-RESOURCE | Carries a TM-SDU and, optionally, the channel-allocation element (a grant) |
| 1 | MAC-FRAG / MAC-END | Continuation of a TM-SDU too long for one slot; subtype 0 is a fragment, subtype 1 the final piece |
| 2 | MAC-BROADCAST | SYSINFO (cell frequency parameters) or ACCESS-DEFINE, selected by a 2-bit sub-type |
| 3 | MAC-SUPPLEMENTARY | MAC-U-SIGNAL / supplementary signalling |

**MAC-RESOURCE** is the workhorse. After the type field it carries fill-bit and
grant-position flags, a 2-bit encryption mode, a random-access flag, a 6-bit length
indication, and then an **address element** whose 3-bit type selects its width — NULL (no
address, and no TM-SDU), a 24-bit SSI, a 10-bit event label, or combinations up to 34 bits.
Reserved length values flag a start fragment (the TM-SDU continues in a following
MAC-FRAG/MAC-END) or a stolen half-slot.

## The channel-allocation element

When a MAC-RESOURCE PDU assigns a call, a channel-allocation element follows the address
(§21.5.2). It carries a 2-bit allocation type, a 4-bit assigned **timeslot**, a 2-bit
up/downlink field, CLCH-permission and cell-change flags, and a 12-bit main **carrier
number** that resolves to an absolute frequency via the band plan. An extended-carrier flag
adds frequency band, offset, duplex spacing and reverse-operation fields; a monitoring
pattern and, when the up/downlink field is 0, an augmented block of QAM/bandwidth/power
fields extend it further. Parsing must walk all of these branches so the cursor lands exactly
at the TM-SDU that follows — the offset at which the LLC and CMCE layers begin. That physical
resource (carrier + timeslot + [usage marker](/reference/tetra-aach/)) is one half of a
[voice grant](/reference/channel-grant/); the party SSIs and emergency flag are the other
half, read from the CMCE TM-SDU riding in the same PDU.

## Relevance to SDR

`internal/radio/tetra/mac.go` implements the lower MAC: `ParseMACResource` decodes the
header, address and (when present) allocation element and reports `TMSDUBitOffset`, the bit
index where the TM-SDU begins; `ParseSysInfo` decodes the MAC-BROADCAST SYSINFO PDU into the
cell's carrier and frequency parameters; and `macFragmentPayload` lifts the continuation
bytes out of MAC-FRAG/MAC-END. The bit widths follow §21.4.3.1 and §21.5.2 and are
cross-checked against osmo-tetra's decoder, because a single wrong field width shifts every
downstream Layer-3 field and corrupts the decoded ISSI/GSSI — the class of bug that only
surfaces against real off-air PDUs, not synthetic round-trips.

## Sources

[^tetra]: [Terrestrial Trunked Radio](https://en.wikipedia.org/wiki/Terrestrial_Trunked_Radio) — Wikipedia, on the TETRA protocol stack.
[^mac]: [Medium access control](https://en.wikipedia.org/wiki/Medium_access_control) — Wikipedia, on the MAC sublayer's role between the physical and link layers.
