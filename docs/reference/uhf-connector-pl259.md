---
slug: uhf-connector-pl259
title: UHF connector (PL-259)
entry_type: hardware
category: rf-front-end
description: "The UHF connector (PL-259 plug, SO-239 jack) is a rugged threaded coaxial connector with no constant impedance, common on HF and VHF ham and CB gear."
keywords: UHF connector, PL-259, SO-239, non-constant impedance, threaded coaxial connector, ham radio, CB radio, HF VHF, PL259 SO239
aka: [PL-259, SO-239, "UHF connector", "PL259"]
autolink: true
infobox:
  - { label: Type, value: "Threaded coaxial connector" }
  - { label: Impedance, value: "Non-constant (no fixed Z)" }
  - { label: Range, value: "DC to ~300 MHz (usable)" }
  - { label: Coupling, value: "5/8-24 threaded" }
  - { label: TX, value: "Yes (high power, HF/VHF)" }
see_also: [coaxial-cable, n-type-connector, sma-connector, bnc-connector, standing-wave-ratio]
cite_urls:
  - https://en.wikipedia.org/wiki/UHF_connector
---

The **UHF connector** — a threaded pair whose plug is the **PL-259** and whose jack is the
**SO-239** — is a rugged, inexpensive [coaxial](/reference/coaxial-cable/) connector that
dominates HF, CB, and lower-VHF amateur gear.[^wiki] Its defining electrical quirk is that
it is **not a constant-impedance** connector: the internal geometry does not hold a defined
50 Ω through the joint, so it introduces a reflection that grows with frequency. Despite the
"UHF" name — a 1930s marketing label from when 30 MHz counted as ultra-high — it is best
used below roughly **300 MHz**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A PL-259 plug threading onto an SO-239 jack, with a return-loss curve showing the impedance discontinuity worsening as frequency rises." xmlns="http://www.w3.org/2000/svg">
  <line x1="25" y1="55" x2="90" y2="55" stroke="currentColor" stroke-width="3"/>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <rect x="90" y="38" width="60" height="34"/>
    <line x1="90" y1="45" x2="150" y2="45"/><line x1="90" y1="65" x2="150" y2="65"/>
  </g>
  <circle cx="156" cy="55" r="3.5" fill="currentColor"/>
  <line x1="159" y1="55" x2="182" y2="55" stroke="currentColor" stroke-width="2"/>
  <rect x="182" y="38" width="50" height="34" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="232" y1="55" x2="290" y2="55" stroke="currentColor" stroke-width="3"/>
  <g font-size="9" fill="currentColor"><text x="120" y="88" text-anchor="middle">PL-259</text><text x="207" y="88" text-anchor="middle">SO-239</text></g>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <line x1="60" y1="145" x2="430" y2="145"/><line x1="60" y1="145" x2="60" y2="105"/>
    <path d="M60 112 Q200 116 320 132 T430 143" stroke-width="1.8"/>
  </g>
  <g font-size="9" fill="currentColor"><text x="245" y="158" text-anchor="middle">frequency &#8594;</text><text x="34" y="112">match</text><text x="360" y="126">worsens</text></g>
</svg>
<figcaption>The PL-259/SO-239 pair is mechanically solid but electrically non-constant-impedance: its match degrades as frequency rises, so it is an HF/VHF connector.</figcaption>
</figure>

## Overview

The UHF connector's appeal is mechanical and economic. The **5/8-24** threaded shell is
large, forgiving, and easy to solder onto thick coax with basic tools, and it handles high
transmit power. For HF and the CB/10-metre/2-metre bands, where the frequency is low enough
that the impedance discontinuity barely matters, it is a perfectly good, durable connector —
which is why generations of amateur and commercial HF/VHF equipment standardised on it.

## What it is

Unlike [N-type](/reference/n-type-connector/) or [SMA](/reference/sma-connector/), which are
engineered so the coax's 50 Ω passes cleanly through the connector body, the PL-259/SO-239
interface changes impedance internally. At HF this reflection is negligible; by the time you
reach 300–450 MHz it produces measurable extra [SWR](/reference/standing-wave-ratio/) and
loss. This is why the connector is fine for a 2-metre rig but a poor choice for 70 cm and
above, where operators switch to N or BNC. The name is a historical artifact, not a
capability claim.

## Variants

- **PL-259 plug / SO-239 jack** is the standard pair ("PL" for plug, "SO" for socket in old
  military nomenclature).
- **Reducers (adapters)** let a PL-259 sized for thick RG-8/RG-213 accept thinner RG-58
  cable.
- **Silver/PTFE versions** improve solderability and dielectric quality but do not change the
  fundamental non-constant-impedance behaviour.
- **Right-angle and bulkhead** bodies are common for panel mounting.

## Relevance to SDR

An SDR listener meets the UHF connector mostly on **HF and low-VHF antennas and radios**:
shortwave receivers, CB and ham HF gear, and many wideband discone bases present an SO-239.
Reaching a dongle then means a **PL-259/SO-239 to [SMA](/reference/sma-connector/)** or
[BNC](/reference/bnc-connector/) adapter. Below a few hundred MHz — which covers HF utility,
CB, and 2-metre traffic — the connector's impedance quirk costs almost nothing, so it is a
practical, robust interface. For the UHF land-mobile trunking bands (700/800 MHz P25 and
DMR), prefer N or SMA and keep the UHF connector out of the path. GopherTrunk is decode
software and touches no connectors, but a UHF connector used well above its comfortable range
adds reflection and loss that erode the signal-to-noise ratio reaching the receiver.

## Sources

[^wiki]: [UHF connector](https://en.wikipedia.org/wiki/UHF_connector) — Wikipedia, on the PL-259/SO-239 pair, non-constant impedance, high-power HF/VHF use, and the historical "UHF" name.
