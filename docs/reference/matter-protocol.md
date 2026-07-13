---
slug: matter-protocol
title: Matter
entry_type: technology
category: wireless-data-iot
description: "Matter is an IP-based application layer for smart-home devices that runs over Thread and Wi-Fi, giving cross-vendor interoperability above the radio rather than a new air interface."
keywords: Matter, smart home, Connectivity Standards Alliance, CHIP, Thread, Wi-Fi, Bluetooth LE commissioning, IPv6, interoperability, IoT application layer
aka: [Matter, Project CHIP, Connected Home over IP]
autolink: true
infobox:
  - { label: Type, value: Smart-home application layer }
  - { label: Idea, value: One IP app layer over Thread & Wi-Fi }
  - { label: Examples, value: "Lights, locks, sensors, thermostats" }
see_also: [thread-protocol, wifi-80211, bluetooth-le, connectivity-standards-alliance, internet-of-things, home-automation]
cite_urls:
  - https://en.wikipedia.org/wiki/Matter_(standard)
  - https://csa-iot.org/all-solutions/matter/
---

**Matter** is a royalty-free application-layer standard for smart-home devices that lets
products from different vendors interoperate over a common IP foundation.[^wiki] Rather
than defining a new radio, Matter rides on existing air interfaces —
[Thread](/reference/thread-protocol/) for low-power devices and
[Wi-Fi](/reference/wifi-80211/) for higher-bandwidth ones — with
[Bluetooth LE](/reference/bluetooth-le/) used only briefly to commission a new device
onto the network. It is stewarded by the
[Connectivity Standards Alliance](/reference/connectivity-standards-alliance/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A layer stack showing Matter as a common application layer sitting on IPv6, which runs over either the Thread 802.15.4 radio or Wi-Fi, with Bluetooth LE used only for initial commissioning." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" font-size="10" fill="currentColor" text-anchor="middle">
    <rect x="90" y="20" width="280" height="28" rx="3" fill="currentColor" fill-opacity="0.28"/><text x="230" y="38">Matter application layer</text>
    <rect x="90" y="56" width="280" height="24" rx="3" fill="currentColor" fill-opacity="0.14"/><text x="230" y="72">IPv6</text>
    <rect x="90" y="90" width="135" height="34" rx="3" fill="none"/><text x="157" y="105">Thread</text><text x="157" y="118" font-size="8">(802.15.4 radio)</text>
    <rect x="235" y="90" width="135" height="34" rx="3" fill="none"/><text x="302" y="105">Wi-Fi</text><text x="302" y="118" font-size="8">(802.11 radio)</text>
    <rect x="90" y="134" width="280" height="22" rx="3" fill="none" stroke-dasharray="4 3"/><text x="230" y="149" font-size="8">Bluetooth LE — commissioning only</text>
  </g>
</svg>
<figcaption>Matter is an application layer over IPv6 that runs on either Thread or Wi-Fi, with Bluetooth LE used only to onboard a new device.</figcaption>
</figure>

## How it works

Matter standardises the layers *above* the radio: how a device describes itself (as a
"data model" of clusters and attributes — an on/off light, a temperature sensor), how
commands and status are exchanged, and how devices are securely commissioned and
controlled. Because every Matter device is an IPv6 host, a light on a Thread mesh and a
camera on Wi-Fi share one addressing and security model, and any Matter controller can
drive both. Onboarding uses a QR/numeric setup code and a brief BLE link to hand the
device its network credentials, after which it joins Thread or Wi-Fi and speaks Matter
over IP. Security rests on per-device certificates and AES-secured sessions.

This "one app layer, several radios" design is Matter's whole point: it removes the need
for each ecosystem (Apple, Google, Amazon, Samsung) to invent its own device protocol,
while leaving the physical layer to proven standards.

## Relevance to SDR

Matter defines *nothing* new at the RF level, so there is no distinct Matter waveform to
capture — a Matter device is simply a Thread (802.15.4 OQPSK/DSSS) or Wi-Fi (802.11
OFDM) transmitter, plus a short burst of BLE during setup. For a software-defined-radio
operator, Matter is best understood as the reason a growing number of 2.4 GHz Thread and
Wi-Fi endpoints exist, not as a signal in itself. **GopherTrunk** does not decode any of
these underlying radios; it targets land-mobile voice trunking and aeronautical data, so
Matter and its transports are out of scope. Their only practical relevance here is added
2.4 GHz band congestion near an antenna site.

## Sources

[^wiki]: [Matter (standard)](https://en.wikipedia.org/wiki/Matter_(standard)) — Wikipedia, on the Matter smart-home application layer, its IPv6 foundation, Thread/Wi-Fi transports, BLE commissioning, and CSA stewardship.
