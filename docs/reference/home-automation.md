---
slug: home-automation
title: Home automation
entry_type: concept
category: hw-sbc
description: Home automation is the control and coordination of household devices such as lights, locks, thermostats, and sensors, often run on a small always-on single-board computer acting as a local hub.
keywords: home automation, smart home, Home Assistant, IoT hub, single-board computer, sensors, Raspberry Pi smart home, local hub, always-on
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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 172" role="img" aria-label="A hub-and-spoke home automation topology. A single-board computer at the centre acts as the local hub, holding the rules and state, and connects out to household devices around it: a light, a door lock, a thermostat, and a sensor. The hub coordinates them locally and keeps working even without an internet connection." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="196" y="64" width="68" height="44" rx="5" fill-opacity="0.14" fill="currentColor"/>
    <path d="M196 86 H96" /><path d="M264 86 H364"/>
    <path d="M215 64 L150 30"/><path d="M245 64 L310 30"/>
    <path d="M215 108 L150 142"/><path d="M245 108 L310 142"/>
    <rect x="52" y="72" width="44" height="28" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <rect x="364" y="72" width="44" height="28" rx="4" fill-opacity="0.06" fill="currentColor"/>
    <circle cx="140" cy="24" r="13" fill-opacity="0.06" fill="currentColor"/>
    <rect x="298" y="12" width="26" height="24" rx="3" fill-opacity="0.06" fill="currentColor"/>
    <rect x="128" y="130" width="26" height="24" rx="3" fill-opacity="0.06" fill="currentColor"/>
    <circle cx="311" cy="142" r="13" fill-opacity="0.06" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8" text-anchor="middle">
    <text x="230" y="82" font-size="9" font-weight="600">SBC hub</text>
    <text x="230" y="95" font-size="7">rules + state</text>
    <text x="74" y="90" font-size="7.5">sensor</text>
    <text x="386" y="90" font-size="7.5">sensor</text>
    <text x="140" y="27" font-size="7.5">light</text>
    <text x="311" y="27" font-size="7.5">lock</text>
    <text x="141" y="146" font-size="7">climate</text>
    <text x="311" y="146" font-size="7.5">sensor</text>
    <text x="230" y="166" font-size="7.5" fill-opacity="0.9">coordinated locally — works with the internet down</text>
  </g>
</svg>
<figcaption>A home-automation hub — typically a small always-on SBC — sits at the centre holding the rules and state, and reaches out to lights, locks, climate, and sensors, keeping the house coordinated even when the internet link is down.</figcaption>
</figure>

## Overview

A typical setup ties together many [Internet of Things](/reference/internet-of-things/) devices through a central hub that holds the rules and state. That hub is often a small, always-on [single-board computer](/reference/single-board-computer/) such as a [Raspberry Pi](/reference/raspberry-pi/) running software like Home Assistant — cheap, low-power, and able to keep everything working locally even when the internet is down. Because the logic lives on the hub rather than in the cloud, a rule like "turn on the porch light at sunset" keeps firing regardless of whether a vendor's servers are reachable.

The devices themselves speak a patchwork of protocols — Wi-Fi, Zigbee, Z-Wave, Bluetooth, plain [GPIO](/reference/gpio/) wiring — and part of the hub's job is to translate between them and present one consistent view. Keeping that bridge on local hardware is also what makes a smart home resilient and private: sensor readings and door-lock commands never have to leave the house.

## Where it fits

Home automation is one of the most common reasons people pick up their first SBC, because the workload — talking to sensors and devices over GPIO, Wi-Fi, and other links, around the clock, at a few watts — fits the board's strengths exactly. It is also a gateway to running local machine learning: pairing the hub with [edge AI](/reference/edge-ai/) lets it recognise faces at the door or sounds in a room without uploading anything.

That same always-on, low-power profile makes such a board a natural host for other field roles. The Pi already sitting in a closet running the smart home is the same kind of box that can run GopherTrunk as a quiet capture node beside an antenna — a headless Linux machine that just needs to stay up and do one job well.

## Sources

[^wiki]: [Home automation](https://en.wikipedia.org/wiki/Home_automation) — Wikipedia, on automating and coordinating household devices.
