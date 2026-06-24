---
slug: peripheral
title: Peripheral
entry_type: concept
category: hw-personal-computers
description: A peripheral is any device connected to a computer that extends its capabilities — input, output, or storage — sitting outside the core CPU and memory and communicating over a bus such as USB.
keywords: peripheral, input device, output device, I/O device, USB device, external device
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

## Overview

Peripherals are how a computer talks to the world, and they fall into three rough groups. *Input* peripherals feed data in — the [keyboard](/reference/keyboard/), [mouse](/reference/mouse/), [webcam](/reference/webcam/), and scanners. *Output* peripherals send results out — the [monitor](/reference/computer-monitor/), [printer](/reference/printer/), and speakers. *Storage* peripherals hold data, like external drives. All of them attach to the machine through an [input/output](/reference/input-output/) interface such as [USB](/reference/usb/), Bluetooth, or an expansion slot, and rely on driver software so the [operating system](/reference/operating-system/) can use them.

## Where it fits

The term draws the line between the computer proper and everything bolted onto it: a part inside the case on the core bus is a component, while a device hanging off an external port is a peripheral. The boundary is fuzzy — an internal drive and an external one differ mainly in where they plug in. For a software-defined radio, the [RTL-SDR](/reference/rtl-sdr/) dongle is just another USB peripheral: the computer treats the radio front end as an input device streaming [IQ data](/reference/iq-data/).

## Sources

[^wiki]: [Peripheral](https://en.wikipedia.org/wiki/Peripheral) — Wikipedia, on peripherals as devices that extend a computer.
