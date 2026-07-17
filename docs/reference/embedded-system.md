---
slug: embedded-system
title: Embedded system
entry_type: concept
category: hw-microcontrollers
description: An embedded system is a computer built into a larger product to perform a dedicated function, usually around a microcontroller with fixed firmware rather than a general-purpose operating system, sensing the world and acting on it in a closed loop.
keywords: embedded system, dedicated function, firmware, real-time, microcontroller, appliance, deeply embedded, sensor, actuator, closed loop
infobox:
  - { label: Type, value: Dedicated-purpose computer }
  - { label: Built around, value: Microcontroller or SoC }
  - { label: Software, value: Firmware, often real-time }
  - { label: Found in, value: Appliances, vehicles, radios }
see_also: [microcontroller, firmware, real-time-operating-system, bare-metal-programming, internet-of-things, sensor]
related_lessons:
  - { title: "What is a microcontroller?", url: /learn/intro-hardware/what-is-a-microcontroller/ }
  - { title: "MCU programming and trade-offs", url: /learn/intro-hardware/mcu-programming-and-tradeoffs/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Embedded_system
---

**An embedded system** is a computer built into a larger product to carry out a dedicated function, rather than a general-purpose machine you load arbitrary programs onto.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 148" role="img" aria-label="Signal chain of an embedded system. Sensors on the left feed readings into a central microcontroller running fixed firmware. The microcontroller decides what to do and drives actuators on the right, such as a motor or relay. A feedback arrow runs from the actuators back to the sensors, closing the sense-decide-act loop." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="24" y="46" width="96" height="48" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <rect x="182" y="40" width="108" height="60" rx="4" fill="currentColor" fill-opacity="0.14"/>
    <rect x="352" y="46" width="96" height="48" rx="4" fill="currentColor" fill-opacity="0.08"/>
    <line x1="120" y1="70" x2="182" y2="70"/>
    <path d="M182 70 L172 65 M182 70 L172 75" stroke-width="1.2"/>
    <line x1="290" y1="70" x2="352" y2="70"/>
    <path d="M352 70 L342 65 M352 70 L342 75" stroke-width="1.2"/>
    <path d="M400 94 L400 126 L72 126 L72 94" stroke-dasharray="3 3"/>
    <path d="M72 100 L67 108 M72 100 L77 108" stroke-width="1.2"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="72" y="66" font-size="8.5" font-weight="600">Sensors</text>
    <text x="72" y="80" font-size="7.5" fill-opacity="0.85">temp, motion,</text>
    <text x="72" y="90" font-size="7.5" fill-opacity="0.85">light&#8230;</text>
    <text x="236" y="64" font-size="9" font-weight="600">Microcontroller</text>
    <text x="236" y="78" font-size="7.5" fill-opacity="0.85">fixed firmware</text>
    <text x="236" y="90" font-size="7.5" fill-opacity="0.85">sense &#8594; decide &#8594; act</text>
    <text x="400" y="66" font-size="8.5" font-weight="600">Actuators</text>
    <text x="400" y="80" font-size="7.5" fill-opacity="0.85">motor, relay,</text>
    <text x="400" y="90" font-size="7.5" fill-opacity="0.85">display</text>
    <text x="236" y="140" font-size="7.5" fill-opacity="0.85">feedback closes the loop</text>
  </g>
</svg>
<figcaption>An embedded system reads the physical world through sensors, a microcontroller running fixed firmware decides what to do, and it drives actuators in response. Because the output changes the world the sensors measure, the whole thing is a closed sense-decide-act loop — one job, done cheaply and reliably.</figcaption>
</figure>

## Overview

Most embedded systems are built around a [microcontroller](/reference/microcontroller/) or [system-on-chip](/reference/system-on-a-chip/) running fixed [firmware](/reference/firmware/). They are everywhere — in appliances, cars, medical devices, industrial controllers, and radios — usually invisible to the user. Many have hard real-time constraints, met with [interrupts](/reference/interrupt/) and a [real-time operating system](/reference/real-time-operating-system/), while simpler ones run [bare metal](/reference/bare-metal-programming/).

They read the physical world through [sensors](/reference/sensor/) and act on it through actuators, closing the loop between measuring and doing. The term shades from "deeply embedded" 8-bit controllers with kilobytes of memory up to Linux-class systems-on-chip, but the common thread is a device dedicated to one purpose rather than a computer you install arbitrary apps onto.

## Embedded versus general-purpose

The distinction is specialization, not size:

| Trait | Embedded system | General-purpose computer |
|-------|-----------------|--------------------------|
| Purpose | One dedicated function | Runs arbitrary programs |
| Software | Fixed firmware | Installable apps + OS |
| Resources | Kilobytes to a few MB | Gigabytes |
| Timing | Often hard real-time | Best-effort |
| Visible? | Usually hidden inside a product | The product itself |

## Where it fits

The defining trade-off is specialization: an embedded system does one job extremely well, cheaply, and reliably, at the cost of flexibility. When a device also needs network connectivity it becomes part of the [Internet of Things](/reference/internet-of-things/). In radio terms, the small transmitters that fill the airwaves GopherTrunk listens to are embedded systems — far too small to run GopherTrunk itself, which lives on a general-purpose computer with an SDR front end. A low-power capture node feeding IQ upstream is one honest exception: it is an embedded-style appliance built for a single job.

## Sources

[^wiki]: [Embedded system](https://en.wikipedia.org/wiki/Embedded_system) — Wikipedia, on dedicated-function computers.
