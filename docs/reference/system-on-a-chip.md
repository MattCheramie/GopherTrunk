---
slug: system-on-a-chip
title: System on a chip (SoC)
entry_type: hardware
category: hw-mobile
description: A system on a chip (SoC) integrates a computer's major subsystems — CPU, GPU, memory controller, radios, and accelerators — onto a single piece of silicon, the building block of nearly every phone and small device.
keywords: system on a chip, SoC, application processor, integrated CPU GPU, mobile chip, Snapdragon, Apple Silicon, SoC vs discrete
aka: [SoC]
autolink: true
infobox:
  - { label: Type, value: Integrated circuit }
  - { label: Integrates, value: CPU, GPU, memory & I/O controllers }
  - { label: Common in, value: Phones, tablets, SBCs }
  - { label: Examples, value: Snapdragon, Apple Silicon, Broadcom }
see_also: [mobile-operating-system, arm-architecture, central-processing-unit, soc-vs-discrete, integrated-circuit, cellular-modem]
cite_urls:
  - https://en.wikipedia.org/wiki/System_on_a_chip
---

A **system on a chip (SoC)** is an integrated circuit that combines a computer's major subsystems — processor cores, graphics, memory and I/O controllers, and often radios and accelerators — onto a single die.[^wiki]

## Overview

Where a desktop spreads its parts across a [motherboard](/reference/motherboard/), an SoC packs them into one chip: one or more [CPU](/reference/central-processing-unit/) cores, a [GPU](/reference/graphics-processing-unit/), a memory controller, and blocks such as a [cellular modem](/reference/cellular-modem/), [GPS receiver](/reference/gps-receiver/), and an [NPU](/reference/neural-processing-unit/) for on-device machine learning. Most are built on the [Arm architecture](/reference/arm-architecture/). Familiar examples include Qualcomm's Snapdragon, Apple Silicon, and the Broadcom parts inside the [Raspberry Pi](/reference/raspberry-pi/).

## Where it fits

Integration is what makes a [smartphone](/reference/smartphone/) small, cheap, and power-efficient: shorter traces and shared silicon cut size and energy use, at the cost of the upgradability you get from [discrete](/reference/soc-vs-discrete/) parts. The same logic puts SoCs in tablets, single-board computers, and embedded gear. A capture node running GopherTrunk on a Pi leans on its Broadcom SoC for everything but the radio front end, which still needs an external SDR dongle.

## Sources

[^wiki]: [System on a chip](https://en.wikipedia.org/wiki/System_on_a_chip) — Wikipedia, on SoC integration and uses.
