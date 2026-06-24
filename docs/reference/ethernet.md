---
slug: ethernet
title: Ethernet
entry_type: concept
category: hw-networking
description: Ethernet is the dominant family of wired local-area networking standards, defining the cables, connectors, and frame format that carry data between computers on a LAN.
keywords: Ethernet, IEEE 802.3, twisted pair, RJ45, Gigabit Ethernet, frame, wired networking
aka: [IEEE 802.3]
infobox:
  - { label: Type, value: Wired LAN standard }
  - { label: Standard, value: IEEE 802.3 }
  - { label: Common cabling, value: Twisted pair (RJ45), fiber }
  - { label: Speeds, value: 10 Mb/s – 400 Gb/s }
see_also: [network-switch, network-interface-card, fiber-optic, power-over-ethernet, wi-fi, lan-and-wan]
cite_urls:
  - https://en.wikipedia.org/wiki/Ethernet
---

**Ethernet** is the dominant family of wired local-area networking standards, defining the cabling, signaling, and frame format that carry data between computers on a [LAN](/reference/lan-and-wan/).[^wiki]

## Overview

Standardized as **IEEE 802.3**, Ethernet packages data into *frames* tagged with source and destination MAC addresses and sends them over twisted-pair copper (terminated in the familiar 8-pin RJ45 plug) or over [fiber-optic](/reference/fiber-optic/) cable for longer runs and higher speeds. Speeds have climbed from the original 10 Mb/s through Fast and Gigabit Ethernet to 10, 100, and 400 Gb/s. Devices attach through a [NIC](/reference/network-interface-card/) and connect to a [switch](/reference/network-switch/); the same cabling can also carry power via [Power over Ethernet](/reference/power-over-ethernet/).

## What it's for

Ethernet is the default for fixed, performance-sensitive connections where a cable is practical — servers, desktops, and infrastructure — while [Wi-Fi](/reference/wi-fi/) handles mobility. A wired Ethernet link gives a GopherTrunk capture node a stable, low-jitter path to stream decoded calls to a [server](/reference/server/), avoiding the contention and interference of a shared radio channel.

## Sources

[^wiki]: [Ethernet](https://en.wikipedia.org/wiki/Ethernet) — Wikipedia, on the wired LAN standard family.
