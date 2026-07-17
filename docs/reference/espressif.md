---
slug: espressif
title: Espressif Systems
entry_type: organization
category: hw-organizations
description: Espressif Systems is a Chinese semiconductor company best known for its low-cost ESP8266 and ESP32 Wi-Fi and Bluetooth microcontrollers used widely in IoT projects.
keywords: Espressif, ESP32, ESP8266, Wi-Fi microcontroller, IoT, Bluetooth, RISC-V, SoC
aka: [Espressif]
autolink: true
infobox:
  - { label: Type, value: Public semiconductor company }
  - { label: Founded, value: "2008" }
  - { label: HQ, value: Shanghai, China }
  - { label: Makes, value: Wi-Fi/Bluetooth microcontrollers (ESP series) }
see_also: [esp32, microcontroller, internet-of-things, wi-fi, bluetooth]
related_lessons:
  - { title: "ESP32", url: /learn/intro-hardware/esp32/ }
cite_urls:
  - https://www.espressif.com/
  - https://en.wikipedia.org/wiki/Espressif_Systems
---

**Espressif Systems** is a Chinese semiconductor company, founded in 2008, best known for
its low-cost [ESP32](/reference/esp32/) family of [Wi-Fi](/reference/wi-fi/) and
[Bluetooth](/reference/bluetooth/) [microcontrollers](/reference/microcontroller/).[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A diagram of an Espressif ESP chip. A single package outline contains three integrated blocks: a microcontroller CPU core, a Wi-Fi and Bluetooth radio, and on-chip memory. Below, a short timeline notes the 2014 ESP8266 that added Wi-Fi, the 2016 ESP32 that added Bluetooth and more power, and later ESP32 variants including RISC-V cores." xmlns="http://www.w3.org/2000/svg">
  <rect x="120" y="16" width="220" height="66" rx="6" stroke="currentColor" fill="currentColor" fill-opacity="0.06" stroke-width="1.4"/>
  <g stroke="currentColor" fill="currentColor" fill-opacity="0.14" stroke-width="1.1">
    <rect x="134" y="30" width="60" height="38" rx="3"/>
    <rect x="200" y="30" width="60" height="38" rx="3"/>
    <rect x="266" y="30" width="60" height="38" rx="3"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8.5">
    <text x="164" y="46">MCU</text>
    <text x="164" y="58">core</text>
    <text x="230" y="46">Wi-Fi /</text>
    <text x="230" y="58">BT radio</text>
    <text x="296" y="46">memory</text>
    <text x="230" y="93" font-size="8" fill-opacity="0.9">one low-cost package</text>
  </g>
  <line x1="40" y1="118" x2="420" y2="118" stroke="currentColor" stroke-width="1.2"/>
  <g stroke="currentColor" fill="currentColor" stroke-width="1.1">
    <circle cx="90" cy="118" r="4" fill-opacity="0.15"/>
    <circle cx="230" cy="118" r="5" fill="currentColor"/>
    <circle cx="370" cy="118" r="4" fill-opacity="0.15"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="8">
    <text x="90" y="108">2014 ESP8266</text>
    <text x="90" y="140">Wi-Fi</text>
    <text x="230" y="108" font-weight="600">2016 ESP32</text>
    <text x="230" y="140">+ Bluetooth, more power</text>
    <text x="370" y="108">later variants</text>
    <text x="370" y="140">incl. RISC-V cores</text>
  </g>
</svg>
<figcaption>Espressif's trick is integration: the ESP series packs a microcontroller, a Wi-Fi/Bluetooth radio, and memory into one cheap chip, evolving from the Wi-Fi-only ESP8266 to the Bluetooth-capable ESP32 and its later RISC-V variants.</figcaption>
</figure>

## Overview

Espressif first drew wide attention with the ESP8266, a cheap chip that put Wi-Fi within
reach of hobbyist projects. Its successor, the ESP32, added Bluetooth, more processing
power, and a richer set of peripherals while staying inexpensive.[^home]

The chips integrate a microcontroller, radio, and memory in one package, and Espressif
backs them with open development frameworks and broad community support. That combination
made the ESP series a default choice for connected sensors and devices across the
[Internet of Things](/reference/internet-of-things/).

## The ESP family

A few generations trace Espressif's climb from a Wi-Fi add-on to a full IoT platform:

| Chip | Year | Adds |
|------|------|------|
| ESP8266 | 2014 | Low-cost Wi-Fi |
| ESP32 | 2016 | Bluetooth, dual core, more peripherals |
| ESP32-C / -S | 2020+ | RISC-V cores, USB, security features |

## Where it fits

By making a wireless-capable microcontroller cheap and easy to program, Espressif helped
make networked embedded projects routine. An ESP32 makes a tidy little wireless sensor or
telemetry node — for example, reporting status or environmental data from a remote
GopherTrunk antenna site back over Wi-Fi — while the heavy radio decoding stays on a
larger host.

## Sources

[^home]: [Espressif](https://www.espressif.com/) — the company's official site, for its chips and SDKs.
[^wiki]: [Espressif Systems](https://en.wikipedia.org/wiki/Espressif_Systems) — Wikipedia, for the company's history and products.
