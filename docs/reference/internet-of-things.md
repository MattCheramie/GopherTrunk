---
slug: internet-of-things
title: Internet of Things (IoT)
entry_type: concept
category: hw-microcontrollers
description: The Internet of Things is the network of everyday physical objects with embedded computing and connectivity that send and receive data, often over cheap wireless microcontrollers.
keywords: Internet of Things, IoT, smart home, embedded, wireless sensors, ESP32, connected devices
aka: [IoT, Internet of Things]
autolink: true
infobox:
  - { label: Type, value: Networked embedded devices }
  - { label: Typical node, value: Cheap wireless microcontroller }
  - { label: Connectivity, value: Wi-Fi, Bluetooth, cellular }
  - { label: Reports to, value: Server or cloud }
see_also: [esp32, microcontroller, cloud-computing, firmware, server]
related_lessons:
  - { title: "ESP32", url: /learn/intro-hardware/esp32/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Internet_of_things
---

**The Internet of Things (IoT)** is the network of everyday physical objects — sensors, appliances, wearables — with embedded computing and connectivity that lets them send and receive data.[^wiki]

## Overview

IoT devices are typically built on cheap wireless [microcontrollers](/reference/microcontroller/) like the [ESP32](/reference/esp32/), each reporting its readings to a [server](/reference/server/) or the [cloud](/reference/cloud-computing/). Individually each node is tiny; collectively they number in the billions.

## Where it fits

The appeal of IoT is putting just enough computing into ordinary objects to make them measurable and controllable from afar. Those same low-power wireless nodes are part of the radio landscape: their transmissions are among the signals filling the airwaves GopherTrunk listens to, though the devices themselves are far too small to run it.

## Sources

[^wiki]: [Internet of things](https://en.wikipedia.org/wiki/Internet_of_things) — Wikipedia, on networked physical objects with embedded computing and connectivity.
