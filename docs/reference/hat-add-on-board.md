---
slug: hat-add-on-board
title: HAT add-on board
entry_type: hardware
category: hw-sbc
description: A HAT is an add-on board that stacks onto a Raspberry Pi's GPIO header to add hardware such as displays, sensors, real-time clocks, or radios, following a standard mechanical and electrical specification.
keywords: HAT, Hardware Attached on Top, Raspberry Pi HAT, GPIO add-on, daughterboard, expansion board, EEPROM ID, 40-pin header
aka: [HAT, Hardware Attached on Top]
infobox:
  - { label: Type, value: SBC add-on board }
  - { label: Connects via, value: 40-pin GPIO header }
  - { label: Standard, value: Raspberry Pi HAT spec }
  - { label: Adds, value: Displays, sensors, radios, I/O }
  - { label: Stands for, value: Hardware Attached on Top }
see_also: [raspberry-pi, gpio, single-board-computer, compute-module, sensor, i2c]
related_lessons:
  - { title: "Raspberry Pi and family", url: /learn/intro-hardware/raspberry-pi-and-family/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Raspberry_Pi#HATs
---

**A HAT** (Hardware Attached on Top) is an add-on board that stacks onto a [Raspberry Pi](/reference/raspberry-pi/)'s [GPIO](/reference/gpio/) header to add hardware.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 176" role="img" aria-label="An exploded side view of a HAT stacking onto a Raspberry Pi. The Pi board sits at the bottom with its 40-pin header standing up; the HAT board hovers above, its matching socket lining up with the header, and mounting standoffs at the corners. A small EEPROM chip on the HAT lets the Pi identify the board and auto-configure its pins." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="70" y="122" width="320" height="30" rx="4" fill-opacity="0.05" fill="currentColor"/>
    <g stroke-width="1.1">
      <line x1="150" y1="98" x2="150" y2="122"/><line x1="162" y1="98" x2="162" y2="122"/>
      <line x1="174" y1="98" x2="174" y2="122"/><line x1="186" y1="98" x2="186" y2="122"/>
      <line x1="198" y1="98" x2="198" y2="122"/><line x1="210" y1="98" x2="210" y2="122"/>
      <line x1="222" y1="98" x2="222" y2="122"/><line x1="234" y1="98" x2="234" y2="122"/>
      <line x1="246" y1="98" x2="246" y2="122"/><line x1="258" y1="98" x2="258" y2="122"/>
    </g>
    <rect x="70" y="44" width="320" height="28" rx="4" fill-opacity="0.1" fill="currentColor"/>
    <rect x="144" y="72" width="122" height="12" rx="2" fill-opacity="0.2" fill="currentColor"/>
    <rect x="330" y="52" width="22" height="14" rx="2" fill-opacity="0.24" fill="currentColor"/>
    <line x1="82" y1="72" x2="82" y2="122" stroke-dasharray="3 3"/>
    <line x1="378" y1="72" x2="378" y2="122" stroke-dasharray="3 3"/>
    <circle cx="82" cy="122" r="3" fill-opacity="0.3" fill="currentColor"/>
    <circle cx="378" cy="122" r="3" fill-opacity="0.3" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="8.5">
    <text x="230" y="38" text-anchor="middle" font-weight="600">HAT board</text>
    <text x="341" y="49" text-anchor="middle" font-size="6.5">EEPROM</text>
    <text x="205" y="82" text-anchor="middle" font-size="6.5">socket</text>
    <text x="118" y="112" text-anchor="middle" font-size="7.5">40-pin header</text>
    <text x="230" y="168" text-anchor="middle" font-weight="600">Raspberry Pi</text>
    <text x="60" y="98" text-anchor="end" font-size="7.5" fill-opacity="0.9">standoffs</text>
  </g>
</svg>
<figcaption>A HAT drops onto the Pi's 40-pin header with corner standoffs holding it rigid; an onboard EEPROM identifies the board so the Pi can auto-configure the right pins the moment it powers up.</figcaption>
</figure>

## Overview

The HAT specification fixes the board size, mounting-hole positions, and 40-pin connector so add-ons fit predictably, and a small EEPROM on the HAT can identify itself so the Pi auto-configures the right pins at boot. That self-identification is the part that makes a HAT more than a generic daughterboard: the Pi reads the EEPROM, learns which pins the HAT drives, and applies the correct device-tree overlay without the user editing config files.

Typical HATs add displays, [sensors](/reference/sensor/), real-time clocks, motor drivers, power management, GPS, or radios — often talking to the Pi over [I2C](/reference/i2c/) or SPI so they consume only a couple of the header's pins for data. Other SBCs use similar stackable add-ons under different names (capes on the BeagleBone, for example), but the Raspberry Pi HAT is the most widely supported ecosystem.

## Anatomy of the standard

| Element | What the spec fixes | Why it matters |
|---------|---------------------|----------------|
| Form factor | Board outline, mounting holes | HATs and cases fit any Pi |
| Connector | 40-pin GPIO header | Consistent electrical pinout |
| EEPROM | ID + pin configuration | Pi auto-sets up the board |
| Data link | Usually I2C / SPI | Few pins, many peripherals |
| Power | Feeds from the Pi's rails | No separate supply needed |

## Where it fits

A HAT is the quickest way to extend a board without designing your own electronics — a middle ground before committing to a custom carrier for a [Compute Module](/reference/compute-module/). For GopherTrunk, a HAT can supply the parts a bare Pi lacks at the antenna: a real-time clock for accurate timestamps when the network is down, a fan and power controller for a sealed enclosure, or a GPS HAT that feeds a pulse-per-second signal in for precise timing on a capture node. Because HATs stack, one can carry several of these at once.

## Sources

[^wiki]: [Raspberry Pi HATs](https://en.wikipedia.org/wiki/Raspberry_Pi#HATs) — Wikipedia, on the HAT add-on board standard.
