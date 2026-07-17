---
slug: esp32
title: ESP32
entry_type: hardware
category: hw-microcontrollers
description: The ESP32 is a low-cost dual-core microcontroller with built-in Wi-Fi and Bluetooth from Espressif, the workhorse of DIY smart-home gadgets and Internet of Things devices.
keywords: ESP32, ESP8266, ESP-IDF, Wi-Fi, Bluetooth, dual core, IoT microcontroller, MicroPython, Espressif, Xtensa
aka: [ESP32]
autolink: true
infobox:
  - { label: Type, value: Wireless microcontroller }
  - { label: Core, value: Dual-core }
  - { label: Wireless, value: Wi-Fi + Bluetooth }
  - { label: Typical price, value: ~$2–5 }
  - { label: Language, value: C/C++ (ESP-IDF, Arduino), MicroPython }
see_also: [microcontroller, internet-of-things, arduino, gpio, firmware, sensor]
related_lessons:
  - { title: "ESP32", url: /learn/intro-hardware/esp32/ }
cite_urls:
  - https://en.wikipedia.org/wiki/ESP32
---

**The ESP32** is a low-cost, dual-core [microcontroller](/reference/microcontroller/) with built-in Wi-Fi and Bluetooth, often around a couple of dollars.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 168" role="img" aria-label="Block diagram of an ESP32 chip. Two processor cores share on-chip SRAM and connect to an integrated radio block for Wi-Fi and Bluetooth, which feeds an external antenna. A peripheral bus links the cores to GPIO pins, an analog-to-digital converter, and serial interfaces used to attach sensors." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="30" y="34" width="70" height="30" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <rect x="30" y="72" width="70" height="30" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <rect x="128" y="50" width="70" height="36" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="230" y="44" width="96" height="48" rx="4" fill="currentColor" fill-opacity="0.10"/>
    <rect x="128" y="112" width="198" height="32" rx="4" fill="currentColor" fill-opacity="0.06"/>
    <line x1="100" y1="49" x2="128" y2="60"/>
    <line x1="100" y1="87" x2="128" y2="76"/>
    <line x1="198" y1="68" x2="230" y2="68"/>
    <line x1="227" y1="128" x2="128" y2="128"/>
    <line x1="163" y1="86" x2="163" y2="112"/>
    <path d="M326 60 L360 52 L360 44 M360 52 L392 44" stroke-width="1.1"/>
    <line x1="360" y1="52" x2="360" y2="84"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="65" y="53" font-size="8">Core 0</text>
    <text x="65" y="91" font-size="8">Core 1</text>
    <text x="163" y="66" font-size="7.5" font-weight="600">SRAM</text>
    <text x="163" y="78" font-size="7.5" fill-opacity="0.85">shared</text>
    <text x="278" y="64" font-size="8.5" font-weight="600">Radio</text>
    <text x="278" y="78" font-size="7.5" fill-opacity="0.85">Wi-Fi + BT</text>
    <text x="376" y="98" font-size="7.5">antenna</text>
    <text x="227" y="126" font-size="8" font-weight="600">Peripherals</text>
    <text x="227" y="138" font-size="7.5" fill-opacity="0.85">GPIO &#183; ADC &#183; UART/SPI/I&#178;C &#8594; sensors</text>
  </g>
</svg>
<figcaption>What sets the ESP32 apart is an integrated radio: two cores share SRAM and drive an on-chip Wi-Fi and Bluetooth block out to an antenna, while a peripheral bus wires GPIO, an ADC, and serial ports to sensors. Radio plus compute for a couple of dollars is why it dominates hobby IoT.</figcaption>
</figure>

## Overview

That combination of price and wireless connectivity makes the ESP32 the workhorse of DIY smart-home gadgets and the [Internet of Things](/reference/internet-of-things/). It comes from Espressif Systems in Shanghai, and common variants include the S3 and C3, alongside the older and simpler ESP8266 that preceded it.

Under the hood the classic ESP32 pairs two 32-bit cores with a few hundred kilobytes of SRAM and, crucially, a built-in 2.4 GHz radio doing both Wi-Fi and Bluetooth — the feature that separates it from a plain [microcontroller](/reference/microcontroller/). Newer parts swap the original Xtensa cores for [RISC-V](/reference/risc-v/).

## ESP32 versus its predecessor

The ESP8266 put a Wi-Fi microcontroller on the map; the ESP32 broadened it:

| Feature | ESP8266 | ESP32 (classic) |
|---------|---------|-----------------|
| Cores | Single | Dual |
| Wireless | Wi-Fi | Wi-Fi + Bluetooth |
| GPIO | Few usable | Many, plus ADC/DAC/touch |
| SRAM | ~80 KB | ~520 KB |
| Role | Cheap Wi-Fi node | General wireless MCU |

## Where it fits

You can program an ESP32 in C/C++ — using either Espressif's ESP-IDF or the [Arduino](/reference/arduino/) toolchain — or in MicroPython. Wired to [sensors](/reference/sensor/) over its [GPIO](/reference/gpio/) pins and talking over Wi-Fi, it is exactly the kind of cheap wireless node that crowds the airwaves GopherTrunk listens to, even though it is far too small to run GopherTrunk itself. As an SDR helper it makes a fine remote GPIO or telemetry board — reading a sensor and reporting over Wi-Fi beside a capture station.

## Sources

[^wiki]: [ESP32](https://en.wikipedia.org/wiki/ESP32) — Wikipedia, on the dual-core design, wireless features, and variants.
