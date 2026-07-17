---
slug: peripheral
title: Peripheral
entry_type: concept
category: hw-personal-computers
description: A peripheral is any device connected to a computer that extends its capabilities — input, output, or storage — sitting outside the core CPU and memory and communicating over a bus such as USB.
keywords: peripheral, input device, output device, I/O device, USB device, external device, storage peripheral
infobox:
  - { label: Type, value: Connected device class }
  - { label: Categories, value: Input, output, storage }
  - { label: Examples, value: Keyboard, mouse, monitor, printer }
  - { label: Connects via, value: USB, Bluetooth, PCIe, … }
see_also: [input-output, keyboard, mouse, computer-monitor, printer, usb]
cite_urls:
  - https://en.wikipedia.org/wiki/Peripheral
---

A **peripheral** is any device connected to a computer that extends what it can do, sitting outside the core [CPU](/reference/central-processing-unit/) and memory and communicating with them over a bus.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A hub-and-spoke map of peripherals around a computer. The computer sits at the center; input peripherals like a keyboard, mouse, and webcam feed data in on the left, output peripherals like a monitor, printer, and speakers send results out on the right, and a storage peripheral hangs below — all connected over buses such as USB, Bluetooth, or PCIe." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.5">
    <rect x="190" y="82" width="80" height="42" rx="4" fill="currentColor" fill-opacity="0.12"/>
  </g>
  <text x="230" y="100" fill="currentColor" stroke="none" text-anchor="middle" font-size="8" font-weight="600">computer</text>
  <text x="230" y="113" fill="currentColor" stroke="none" text-anchor="middle" font-size="6.5" fill-opacity="0.85">CPU + memory</text>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="30" y="26" width="66" height="20" rx="3"/>
    <rect x="30" y="64" width="66" height="20" rx="3"/>
    <rect x="30" y="102" width="66" height="20" rx="3"/>
    <rect x="364" y="26" width="70" height="20" rx="3"/>
    <rect x="364" y="64" width="70" height="20" rx="3"/>
    <rect x="364" y="102" width="70" height="20" rx="3"/>
    <rect x="197" y="156" width="66" height="20" rx="3"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1" stroke-dasharray="4 3">
    <path d="M96 36 C150 40 160 90 190 96"/>
    <path d="M96 74 C150 78 165 96 190 100"/>
    <path d="M96 112 C150 108 170 106 190 106"/>
    <path d="M364 36 C310 40 300 90 270 96"/>
    <path d="M364 74 C310 78 295 96 270 100"/>
    <path d="M364 112 C310 108 290 108 270 108"/>
    <path d="M230 124 V156"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" text-anchor="middle">
    <text x="63" y="39">keyboard</text>
    <text x="63" y="77">mouse</text>
    <text x="63" y="115">webcam</text>
    <text x="399" y="39">monitor</text>
    <text x="399" y="77">printer</text>
    <text x="399" y="115">speakers</text>
    <text x="230" y="169">external drive</text>
    <text x="63" y="17" font-weight="600" fill-opacity="0.85">INPUT</text>
    <text x="399" y="17" font-weight="600" fill-opacity="0.85">OUTPUT</text>
    <text x="230" y="192" font-size="7" fill-opacity="0.85">bus: USB · Bluetooth · PCIe</text>
  </g>
</svg>
<figcaption>Peripherals ring the computer and fall into three groups — input devices feed data in, output devices send results out, and storage devices hold data — each attaching over a bus such as USB, Bluetooth, or PCIe.</figcaption>
</figure>

## Overview

Peripherals are how a computer talks to the world, and they fall into three rough groups. *Input* peripherals feed data in — the [keyboard](/reference/keyboard/), [mouse](/reference/mouse/), [webcam](/reference/webcam/), and scanners. *Output* peripherals send results out — the [monitor](/reference/computer-monitor/), [printer](/reference/printer/), and speakers. *Storage* peripherals hold data, like external drives.

All of them attach to the machine through an [input/output](/reference/input-output/) interface such as [USB](/reference/usb/), Bluetooth, or an expansion slot, and rely on driver software so the [operating system](/reference/operating-system/) can use them.

## Categories

The same bus carries devices from every category; what differs is the direction data flows:

| Category | Data direction | Examples |
|----------|----------------|----------|
| Input | Into the computer | Keyboard, mouse, webcam, scanner |
| Output | Out of the computer | Monitor, printer, speakers |
| Storage | Both ways | External drive, USB stick |
| Combined | Both ways | Touchscreen, all-in-one printer |

## Where it fits

The term draws the line between the computer proper and everything bolted onto it: a part inside the case on the core bus is a component, while a device hanging off an external port is a peripheral. The boundary is fuzzy — an internal drive and an external one differ mainly in where they plug in. For a software-defined radio, the [RTL-SDR](/reference/rtl-sdr/) dongle is just another USB peripheral: the computer treats the radio front end as an input device streaming [IQ data](/reference/iq-data/).

## Sources

[^wiki]: [Peripheral](https://en.wikipedia.org/wiki/Peripheral) — Wikipedia, on peripherals as devices that extend a computer.
