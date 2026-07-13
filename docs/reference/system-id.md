---
slug: system-id
title: System ID
entry_type: term
category: trunked-radio
description: "A System ID is a 12-bit P25 code identifying a trunked system within a WACN; paired with the WACN and RFSS it uniquely names the network a radio is on."
keywords: system id, sysid, P25 system id, 12-bit, WACN, RFSS, system identity, trunking network id, radioreference sysid
aka: [system id, sysid, system identity]
autolink: true
infobox:
  - { label: Type, value: P25 system identifier }
  - { label: Size, value: 12 bits (0x000–0xFFF) }
  - { label: Scope, value: Unique within its WACN }
see_also: [wacn, rfss, project-25, trunking-site, network-access-code, roaming]
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **System ID** (often written *SysID*) is a 12-bit code that identifies a specific
trunked system within a [WACN](/reference/wacn/), the middle level of the
[P25](/reference/project-25/) identity hierarchy.[^wiki] It is not globally unique on
its own — a given SysID value can appear under many different WACNs — but the pair
`WACN + System ID` names one operator's system unambiguously, and adding the
[RFSS](/reference/rfss/) number pins down a specific subsystem within it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="Three identifier fields shown as a bit ruler: a 20-bit WACN, a 12-bit System ID, and an 8-bit RFSS, concatenated into one full system address." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="20" y="40" width="200" height="34" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="120" y="57">WACN — 20 bits</text><text x="120" y="69" font-size="7.5">0xBEE00</text>
    <rect x="224" y="40" width="130" height="34" rx="4" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.2"/><text x="289" y="57">System ID — 12 bits</text><text x="289" y="69" font-size="7.5">0x2A7</text>
    <rect x="358" y="40" width="82" height="34" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="399" y="57">RFSS — 8b</text><text x="399" y="69" font-size="7.5">0x01</text>
  </g>
  <text x="230" y="100" text-anchor="middle" font-size="9" fill="currentColor">WACN + System ID + RFSS = one system's globally unique address</text>
</svg>
<figcaption>The System ID is the 12-bit middle field; only together with the WACN and RFSS does it uniquely name a system.</figcaption>
</figure>

## How it works

The System ID is broadcast continuously on the [control channel](/reference/control-channel/)
in the network-status messages, so radios and monitors learn it without configuration.
A subscriber radio compares an advertised WACN and System ID against its home network
when deciding whether to affiliate: a match means "this is my system, register here," a
mismatch means "this is someone else's network, keep scanning." This is the mechanism
behind [roaming](/reference/roaming/) across [sites](/reference/trunking-site/) of the
same system versus rejecting a foreign one.

Because the 12-bit field allows only 4096 values, the same System ID is reused freely
across the world; it is meaningful only *inside* its parent WACN. That is why database
listings quote the WACN and System ID together as the canonical identifier of a P25
network. The System ID is distinct from the per-frame
[Network Access Code](/reference/network-access-code/), which is a fast squelch filter
rather than a registered identity.

## In practice

- The System ID is written as a three-hex-digit value (12 bits), e.g. `0x2A7`, and is
  quoted with the WACN as the canonical identifier of a P25 network.
- Because values repeat across WACNs, a System ID quoted on its own is ambiguous — a
  common source of confusion when comparing systems from different operators.
- A subscriber radio's home configuration includes its WACN and System ID; matching both
  is the precondition for affiliating to any site it hears.

## Relevance to SDR

For a monitor the System ID, read together with the WACN, is the key that matches a
control channel to a named system in reference databases and distinguishes it from
neighbouring networks. **GopherTrunk** decodes the System ID from the P25 control
channel and reports it next to the WACN and RFSS, giving each tracked control channel a
concrete network identity. It is descriptive metadata — knowing the System ID does not
unlock encrypted voice.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 system-identity hierarchy including the System ID.
