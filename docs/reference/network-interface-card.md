---
slug: network-interface-card
title: Network interface card (NIC)
entry_type: hardware
category: hw-networking
description: A network interface card is the hardware that connects a computer to a network, converting the machine's data into signals on the wire or air and carrying its hardware (MAC) address.
keywords: network interface card, NIC, network adapter, Ethernet card, MAC address, wireless adapter
aka: [NIC, network adapter, network card]
infobox:
  - { label: Type, value: Network adapter }
  - { label: Connects, value: Computer to network }
  - { label: Identifies, value: By MAC address }
  - { label: Forms, value: Wired (Ethernet) or wireless }
see_also: [ethernet, wi-fi, network-switch, router, ip-address, motherboard]
cite_urls:
  - https://en.wikipedia.org/wiki/Network_interface_controller
---

A **network interface card** (**NIC**, or network adapter) is the hardware that connects a computer to a network, turning the machine's data into electrical, optical, or radio signals and back again.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 188" role="img" aria-label="A host's data passes to its network interface card, which frames it and drives the bits onto the network cable — and pulls incoming frames back off the wire — identified on the link by the MAC address burned into the card." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="52" width="118" height="84" rx="6" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  <text x="79" y="88" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">host</text>
  <text x="79" y="104" text-anchor="middle" font-size="8" fill="currentColor">operating system</text>
  <rect x="186" y="44" width="120" height="100" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="246" y="70" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">NIC</text>
  <text x="246" y="88" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.9">MAC 00:1B:44:11:3A:B7</text>
  <line x1="196" y1="98" x2="296" y2="98" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.4"/>
  <text x="246" y="120" text-anchor="middle" font-size="8" fill="currentColor">frames ⇄ signals</text>
  <rect x="354" y="62" width="90" height="64" rx="6" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.2" stroke-dasharray="5 3"/>
  <text x="399" y="90" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">network</text>
  <text x="399" y="105" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">the wire</text>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <line x1="138" y1="82" x2="186" y2="82" marker-end="url(#nic_ar)"/>
    <line x1="186" y1="104" x2="138" y2="104" marker-end="url(#nic_ar)"/>
    <line x1="306" y1="82" x2="354" y2="82" marker-end="url(#nic_ar)"/>
    <line x1="354" y1="104" x2="306" y2="104" marker-end="url(#nic_ar)"/>
  </g>
  <text x="246" y="176" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">the card frames data and puts it on (and takes it off) the wire</text>
  <defs><marker id="nic_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The NIC is the bridge between a host and the medium: it frames the machine's outgoing data and drives it onto the cable, and pulls incoming frames back off it. Its burned-in MAC address is what names the card on the local link, so a switch or router can find it.</figcaption>
</figure>

## Overview

A NIC handles the lowest layers of networking: framing data into packets, putting them on the medium, and pulling incoming frames off it. Each NIC carries a globally unique **MAC address** burned in at manufacture, which identifies it on the local link. Adapters come as wired [Ethernet](/reference/ethernet/) ports, [Wi-Fi](/reference/wi-fi/) radios, or fiber interfaces, and may be a discrete card in a [PCI Express](/reference/pci-express/) slot, a [USB](/reference/usb/) dongle, or — most commonly today — circuitry integrated onto the [motherboard](/reference/motherboard/) or [system-on-a-chip](/reference/system-on-a-chip/).

## Where it fits

The NIC is where a host meets the rest of the network; an [IP address](/reference/ip-address/) is assigned to it, and a [switch](/reference/network-switch/) or [router](/reference/router/) sees it by its MAC address. In a GopherTrunk deployment a small capture node — say a [Raspberry Pi](/reference/raspberry-pi/) by the antenna — uses its built-in NIC (wired or Wi-Fi) to stream decoded calls back to a [server](/reference/server/) for storage.

## Sources

[^wiki]: [Network interface controller](https://en.wikipedia.org/wiki/Network_interface_controller) — Wikipedia, on the adapter that connects a computer to a network.
