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

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 210" role="img" aria-label="The CPU on the left connects through three I/O controllers in the middle to a keyboard, a disk, and a network port on the right, with data flowing both into and out of the processor through those controllers." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="78" width="90" height="54" rx="6" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="63" y="102" text-anchor="middle" font-size="10" fill="currentColor" font-weight="600">CPU</text>
  <text x="63" y="117" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.85">runs code</text>
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <rect x="168" y="30" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="218" y="51">kbd controller</text>
    <rect x="168" y="88" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="218" y="109">disk controller</text>
    <rect x="168" y="146" width="100" height="34" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="218" y="167">NIC</text>
    <rect x="332" y="30" width="104" height="34" rx="4" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.1"/><text x="384" y="47">keyboard</text><text x="384" y="58" font-size="7" fill-opacity="0.8">in</text>
    <rect x="332" y="88" width="104" height="34" rx="4" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.1"/><text x="384" y="105">disk</text><text x="384" y="116" font-size="7" fill-opacity="0.8">in ⇄ out</text>
    <rect x="332" y="146" width="104" height="34" rx="4" fill="currentColor" fill-opacity="0.06" stroke="currentColor" stroke-width="1.1"/><text x="384" y="163">network</text><text x="384" y="174" font-size="7" fill-opacity="0.8">in ⇄ out</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <line x1="108" y1="105" x2="168" y2="47" marker-start="url(#io_ar)" marker-end="url(#io_ar)"/>
    <line x1="108" y1="105" x2="168" y2="105" marker-start="url(#io_ar)" marker-end="url(#io_ar)"/>
    <line x1="108" y1="105" x2="168" y2="163" marker-start="url(#io_ar)" marker-end="url(#io_ar)"/>
    <line x1="268" y1="47" x2="332" y2="47" marker-start="url(#io_ar)" marker-end="url(#io_ar)"/>
    <line x1="268" y1="105" x2="332" y2="105" marker-start="url(#io_ar)" marker-end="url(#io_ar)"/>
    <line x1="268" y1="163" x2="332" y2="163" marker-start="url(#io_ar)" marker-end="url(#io_ar)"/>
  </g>
  <text x="240" y="202" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">every device reaches the CPU through a controller — data in and out</text>
  <defs><marker id="io_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The CPU never touches devices directly — each keyboard, disk, or network port sits behind an I/O controller that the processor talks to. Input flows in (keystrokes, packets, disk reads) and output flows back out (writes, packets). A computer with no I/O is sealed off and useless.</figcaption>
</figure>

## Overview
**Input** brings data in: a keyboard, a mouse, a microphone, a sensor reading, a packet arriving on the network. **Output** sends data back out: pixels to a screen, sound to a speaker, a packet onto the network, a signal to an actuator. Many channels are bidirectional — USB and network ports carry data both ways.

A computer with no I/O is sealed off and useless; I/O is what lets it observe and act.

## Where it fits
I/O is one of the four building blocks of [computer hardware](/reference/computer-hardware/). On a desktop the channels are mostly USB, video, and network. On small devices like a [single-board computer](/reference/single-board-computer/) or a [microcontroller](/reference/microcontroller/), the most direct form of I/O is [GPIO](/reference/gpio/) — general-purpose pins wired straight to sensors and actuators.

## Sources
[^wiki]: [Input/output](https://en.wikipedia.org/wiki/Input/output) — Wikipedia, on the channels a computer uses to communicate with the outside world.
