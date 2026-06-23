---
slug: input-output
title: Input/output (I/O)
entry_type: concept
category: hw-foundations
description: Input/output (I/O) is the set of channels through which a computer takes in and sends out data, from keyboards and screens to network ports and GPIO pins.
keywords: input output, I/O, peripherals, USB, network ports, GPIO, keyboard screen
aka: [I/O, input/output, input output]
infobox:
  - { label: Type, value: Data channels }
  - { label: Input, value: Keyboard, sensors, network }
  - { label: Output, value: Screen, actuators, network }
  - { label: On small devices, value: GPIO pins }
see_also: [computer-hardware, gpio, central-processing-unit, single-board-computer, microcontroller]
related_lessons:
  - { title: "The building blocks", url: /learn/intro-hardware/building-blocks/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Input/output
---

**Input/output (I/O)** is the set of channels through which a computer takes in and sends out data — everything that connects the [CPU](/reference/central-processing-unit/) to the world beyond the chip.[^wiki]

## Overview
**Input** brings data in: a keyboard, a mouse, a microphone, a sensor reading, a packet arriving on the network. **Output** sends data back out: pixels to a screen, sound to a speaker, a packet onto the network, a signal to an actuator. Many channels are bidirectional — USB and network ports carry data both ways.

A computer with no I/O is sealed off and useless; I/O is what lets it observe and act.

## Where it fits
I/O is one of the four building blocks of [computer hardware](/reference/computer-hardware/). On a desktop the channels are mostly USB, video, and network. On small devices like a [single-board computer](/reference/single-board-computer/) or a [microcontroller](/reference/microcontroller/), the most direct form of I/O is [GPIO](/reference/gpio/) — general-purpose pins wired straight to sensors and actuators.

## Sources
[^wiki]: [Input/output](https://en.wikipedia.org/wiki/Input/output) — Wikipedia, on the channels a computer uses to communicate with the outside world.
