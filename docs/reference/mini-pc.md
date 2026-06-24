---
slug: mini-pc
title: Mini PC
entry_type: hardware
category: hw-personal-computers
description: A mini PC is a small-form-factor desktop computer, often the size of a paperback, that uses low-power laptop-class parts to deliver a full PC in a fraction of the space and power of a tower.
keywords: mini PC, small form factor, NUC, SFF PC, low-power desktop, fanless PC
aka: [Mini PC, SFF PC, NUC]
infobox:
  - { label: Type, value: Small-form-factor PC }
  - { label: Size, value: Paperback to lunchbox }
  - { label: Internals, value: Laptop-class CPU/RAM }
  - { label: Power, value: Low (often 10–65 W) }
see_also: [personal-computer, desktop-computer, all-in-one-computer, thin-client, single-board-computer]
related_lessons:
  - { title: "Desktop computers", url: /learn/intro-hardware/desktop-computers/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Nettop
---

A **mini PC** is a small-form-factor [desktop computer](/reference/desktop-computer/) — often about the size of a paperback book — that packs a full PC into a fraction of the volume and power of a conventional tower.[^wiki]

## Overview

To shrink the box, a mini PC uses low-power laptop-class parts: an efficient [CPU](/reference/central-processing-unit/) with integrated graphics, soldered or compact [RAM](/reference/random-access-memory/), and an M.2 [SSD](/reference/solid-state-drive/). It still runs a full desktop [operating system](/reference/operating-system/) and connects to an external [monitor](/reference/computer-monitor/), [keyboard](/reference/keyboard/), and [mouse](/reference/mouse/). Intel's NUC line popularized the format; many vendors now ship similar units, some fanless and silent.

## Where it fits

A mini PC suits anywhere a tower is too big or loud: media boxes, digital signage, small offices, and always-on home services. It sits between a richer [single-board computer](/reference/single-board-computer/) and a full desktop — more powerful and more standard than a Pi, smaller and quieter than a tower. A small fanless mini PC makes a tidy GopherTrunk node: it can sit by the antenna running a capture daemon around the clock and sip power doing it.

## Sources

[^wiki]: [Nettop](https://en.wikipedia.org/wiki/Nettop) — Wikipedia, on small-form-factor desktop PCs.
