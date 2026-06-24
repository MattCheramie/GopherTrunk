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

## Overview

A NIC handles the lowest layers of networking: framing data into packets, putting them on the medium, and pulling incoming frames off it. Each NIC carries a globally unique **MAC address** burned in at manufacture, which identifies it on the local link. Adapters come as wired [Ethernet](/reference/ethernet/) ports, [Wi-Fi](/reference/wi-fi/) radios, or fiber interfaces, and may be a discrete card in a [PCI Express](/reference/pci-express/) slot, a [USB](/reference/usb/) dongle, or — most commonly today — circuitry integrated onto the [motherboard](/reference/motherboard/) or [system-on-a-chip](/reference/system-on-a-chip/).

## Where it fits

The NIC is where a host meets the rest of the network; an [IP address](/reference/ip-address/) is assigned to it, and a [switch](/reference/network-switch/) or [router](/reference/router/) sees it by its MAC address. In a GopherTrunk deployment a small capture node — say a [Raspberry Pi](/reference/raspberry-pi/) by the antenna — uses its built-in NIC (wired or Wi-Fi) to stream decoded calls back to a [server](/reference/server/) for storage.

## Sources

[^wiki]: [Network interface controller](https://en.wikipedia.org/wiki/Network_interface_controller) — Wikipedia, on the adapter that connects a computer to a network.
