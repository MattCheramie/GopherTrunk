---
slug: home-server
title: Home server & self-hosting
entry_type: concept
category: hw-servers
description: A home server is a computer you run at home to host services for yourself, such as file storage, a personal site, or media — the heart of self-hosting.
keywords: home server, self-hosting, NAS, self host, port forwarding, home lab
aka: [Home server, Self-hosting]
infobox:
  - { label: Type, value: Home-run server }
  - { label: Common uses, value: NAS, media, personal site }
  - { label: Hidden costs, value: Power, uptime, backups }
  - { label: Good first box, value: Raspberry Pi or old PC }
see_also: [server, raspberry-pi, single-board-computer, dedicated-server, cloud-computing]
related_lessons:
  - { title: "Home servers", url: /learn/intro-hardware/home-servers/ }
  - { title: "Combining tiers", url: /learn/intro-hardware/combining-tiers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Self-hosting_(web_services)
---

A **home server** is a computer you run at home to host services for yourself — file storage (a NAS), a personal site, or media you stream around the house.[^wiki]

## Overview

Self-hosting means running those services on hardware you own instead of paying a provider. The appeal is control and privacy; the costs nobody mentions up front are electricity, keeping the box online 24/7, and a real backup plan. To reach it from outside your house you typically set up port forwarding on your router so requests find the [server](/reference/server/).

## Where it fits

A [Raspberry Pi](/reference/raspberry-pi/) or other [single-board computer](/reference/single-board-computer/) — or an old PC gathering dust — makes a great first server: cheap, low-power, and enough for file shares and small sites. In GopherTrunk terms a home machine can do both jobs at once: capture RF from an attached dongle *and* store and serve the decoded data, which a remote VPS cannot.

## Sources

[^wiki]: [Self-hosting (web services)](https://en.wikipedia.org/wiki/Self-hosting_(web_services)) — Wikipedia, on running services on hardware you own.
