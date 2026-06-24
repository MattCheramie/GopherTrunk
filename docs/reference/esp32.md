---
slug: esp32
title: ESP32
entry_type: hardware
category: hw-microcontrollers
description: The ESP32 is a low-cost dual-core microcontroller with built-in Wi-Fi and Bluetooth, the workhorse of DIY smart-home gadgets and Internet of Things devices.
keywords: ESP32, ESP8266, ESP-IDF, Wi-Fi, Bluetooth, dual core, IoT microcontroller, MicroPython
aka: [ESP32]
autolink: true
infobox:
  - { label: Type, value: Wireless microcontroller }
  - { label: Core, value: Dual-core }
  - { label: Wireless, value: Wi-Fi + Bluetooth }
  - { label: Typical price, value: ~$2–5 }
  - { label: Language, value: C/C++ (ESP-IDF, Arduino), MicroPython }
see_also: [microcontroller, internet-of-things, arduino, gpio, firmware]
related_lessons:
  - { title: "ESP32", url: /learn/intro-hardware/esp32/ }
cite_urls:
  - https://en.wikipedia.org/wiki/ESP32
---

**The ESP32** is a low-cost, dual-core [microcontroller](/reference/microcontroller/) with built-in Wi-Fi and Bluetooth, often around a couple of dollars.[^wiki]

## Overview

That combination of price and wireless connectivity makes the ESP32 the workhorse of DIY smart-home gadgets and the [Internet of Things](/reference/internet-of-things/). Common variants include the S3 and C3, alongside the older and simpler ESP8266 that preceded it.

## Where it fits

You can program an ESP32 in C/C++ — using either Espressif's ESP-IDF or the [Arduino](/reference/arduino/) toolchain — or in MicroPython. Wired to sensors over its [GPIO](/reference/gpio/) pins and talking over Wi-Fi, it is exactly the kind of cheap wireless node that crowds the airwaves GopherTrunk listens to, even though it is far too small to run GopherTrunk itself.

## Sources

[^wiki]: [ESP32](https://en.wikipedia.org/wiki/ESP32) — Wikipedia, on the dual-core design, wireless features, and variants.
