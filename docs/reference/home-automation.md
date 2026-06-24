---
slug: home-automation
title: Home automation
entry_type: concept
category: hw-sbc
description: Home automation is the control and coordination of household devices such as lights, locks, thermostats, and sensors, often run on a small always-on single-board computer acting as a local hub.
keywords: home automation, smart home, Home Assistant, IoT hub, single-board computer, sensors, Raspberry Pi smart home, local hub
aka: [smart home]
infobox:
  - { label: Type, value: Application area }
  - { label: Controls, value: Lights, locks, climate, sensors }
  - { label: Often runs on, value: An always-on SBC }
  - { label: Relies on, value: IoT devices and sensors }
  - { label: Goal, value: Local, scheduled, reactive control }
see_also: [single-board-computer, raspberry-pi, internet-of-things, sensor, edge-ai, home-server]
related_lessons:
  - { title: "SBC use cases and limits", url: /learn/intro-hardware/sbc-use-cases-and-limits/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Home_automation
---

**Home automation** is the control and coordination of household devices — lights, locks, thermostats, [sensors](/reference/sensor/) — so they run on schedules or react to conditions instead of being operated by hand.[^wiki]

## Overview

A typical setup ties together many [Internet of Things](/reference/internet-of-things/) devices through a central hub that holds the rules and state. That hub is often a small, always-on [single-board computer](/reference/single-board-computer/) such as a [Raspberry Pi](/reference/raspberry-pi/) running software like Home Assistant — cheap, low-power, and able to keep everything working locally even when the internet is down.

## Where it fits

Home automation is one of the most common reasons people pick up their first SBC, because the workload — talking to sensors and devices over [GPIO](/reference/gpio/), Wi-Fi, and other links, around the clock — fits the board's strengths. The same always-on, low-power profile makes such a board a natural host for other field roles, like running GopherTrunk as a quiet capture node beside the antenna.

## Sources

[^wiki]: [Home automation](https://en.wikipedia.org/wiki/Home_automation) — Wikipedia, on automating and coordinating household devices.
