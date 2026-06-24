---
slug: compute-module
title: Compute Module
entry_type: hardware
category: hw-sbc
description: A Compute Module is a single-board computer stripped to a small module without ports, designed to be soldered or socketed into a custom carrier board for embedded products.
keywords: Raspberry Pi Compute Module, CM4, CM5, SO-DIMM module, carrier board, system on module, embedded SBC
aka: [Raspberry Pi Compute Module, CM4]
infobox:
  - { label: Type, value: Embeddable SBC module }
  - { label: Form, value: SO-DIMM or board-to-board }
  - { label: Needs, value: A custom carrier board }
  - { label: Runs, value: Linux }
  - { label: Best-known, value: Raspberry Pi CM4 / CM5 }
see_also: [raspberry-pi, single-board-computer, system-on-a-chip, hat-add-on-board, gpio, embedded-system]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi#Compute_Module
---

**A Compute Module** is a [single-board computer](/reference/single-board-computer/) stripped down to a small module — the processor, memory, and storage — without the usual ports, meant to plug into a custom carrier board.[^wiki]

## Overview

The best-known example is the Raspberry Pi Compute Module (CM4, CM5), but the idea is general: take the [system on a chip](/reference/system-on-a-chip/) and memory of a board like the [Raspberry Pi](/reference/raspberry-pi/) and put it on a compact, breakable-out module. A product designer then lays out a *carrier board* that routes the [GPIO](/reference/gpio/), USB, Ethernet, and power exactly as the product needs, rather than working around a fixed consumer layout.

## Where it fits

A Compute Module is the choice when an [embedded system](/reference/embedded-system/) needs the brains of an SBC but its own enclosure, connectors, and shape — a step beyond bolting a [HAT](/reference/hat-add-on-board/) onto a stock board. In a GopherTrunk-style product, a Compute Module on a custom carrier could host the SDR front end and storage in a sealed, antenna-mounted capture node. The trade-off is engineering effort: you design and manufacture the carrier yourself.

## Sources

[^wiki]: [Raspberry Pi Compute Module](https://en.wikipedia.org/wiki/Raspberry_Pi#Compute_Module) — Wikipedia, on the modular, carrier-board form of the Raspberry Pi.
