---
slug: computer-monitor
title: Computer monitor
entry_type: hardware
category: hw-personal-computers
description: A computer monitor is the display that shows a computer's visual output, characterized by its panel technology, size, resolution, and refresh rate, connected over HDMI, DisplayPort, or USB-C.
keywords: monitor, display, LCD, LED, IPS, VA, TN, OLED, resolution, refresh rate, HDMI, DisplayPort, USB-C
infobox:
  - { label: Type, value: Display / output device }
  - { label: Panel, value: LCD (IPS/VA/TN), OLED }
  - { label: Key specs, value: Size, resolution, refresh rate }
  - { label: Connects via, value: HDMI, DisplayPort, USB-C }
see_also: [peripheral, personal-computer, desktop-computer, all-in-one-computer, graphics-processing-unit, waterfall-display]
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_monitor
---

A **computer monitor** is the display that shows a computer's visual output — the screen you read and interact with, as opposed to the box that does the computing.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A computer monitor labelled with its key specifications: the diagonal measures its size, a pixel grid in the corner stands for resolution such as 1920 by 1080, a hertz label marks the refresh rate, and a row of HDMI, DisplayPort, and USB-C ports on the back carry the video signal from the GPU." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <rect x="60" y="24" width="230" height="130" rx="4"/>
    <rect x="70" y="34" width="210" height="110" fill="currentColor" fill-opacity="0.07"/>
    <path d="M150 154 L200 154 M175 154 L175 170 M150 170 H200"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1" stroke-dasharray="4 3">
    <path d="M70 34 L280 144"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="0.8">
    <path d="M240 44 H274 M240 54 H274 M240 64 H274 M250 44 V64 M260 44 V64"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="120" y="96" font-size="8" fill-opacity="0.85">size (diagonal)</text>
    <text x="238" y="80">resolution</text>
    <text x="238" y="90" font-size="6.5">1920&#215;1080 px</text>
    <text x="95" y="130">refresh: 60&#8211;144 Hz</text>
    <text x="175" y="184" text-anchor="middle" font-size="7">stand</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1">
    <rect x="340" y="40" width="90" height="18" rx="2"/>
    <rect x="340" y="66" width="90" height="18" rx="2"/>
    <rect x="340" y="92" width="90" height="18" rx="2"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5" text-anchor="middle">
    <text x="385" y="52">HDMI</text>
    <text x="385" y="78">DisplayPort</text>
    <text x="385" y="104">USB-C</text>
    <text x="385" y="128" font-size="7" fill-opacity="0.85">video in from GPU</text>
  </g>
</svg>
<figcaption>A monitor is defined by three numbers — size, resolution, and refresh rate — and fed a video signal from the computer's GPU over an HDMI, DisplayPort, or USB-C input, which it turns into light.</figcaption>
</figure>

## Overview

Most monitors are flat-panel displays built on LCD technology (with IPS, VA, or TN variants) lit by LEDs, while higher-end models use OLED for deeper contrast and true blacks. The main specifications are *size* (measured diagonally), *resolution* (the pixel count, such as 1920×1080 or 3840×2160), and *refresh rate* (how many times per second the image updates, in hertz).

A monitor is an output [peripheral](/reference/peripheral/): it receives a video signal from the computer's [GPU](/reference/graphics-processing-unit/) over HDMI, DisplayPort, or USB-C and turns it into light. It does no computing of its own — swap the box behind it and the same panel shows a different machine.

## Panel types

The panel technology sets the trade-off between color, contrast, speed, and price:

| Panel | Strength | Weakness | Typical use |
|-------|----------|----------|-------------|
| IPS | Wide viewing angles, color | Lower contrast | General, creative |
| VA | High contrast | Slower pixels | Media, general |
| TN | Fast, cheap | Poor angles/color | Budget, e-sports |
| OLED | Deep blacks, fast | Cost, burn-in risk | Premium, HDR |

## Where it fits

A monitor pairs with any [desktop computer](/reference/desktop-computer/), [mini PC](/reference/mini-pc/), or laptop, while an [all-in-one computer](/reference/all-in-one-computer/) builds the same panel into the system. Picking one is a trade-off: office and coding work reward resolution and screen area, gaming rewards high refresh rate and low latency, and photo or video work rewards color accuracy. For a GopherTrunk bench, screen area pays off — a wide or tall monitor keeps a [waterfall](/reference/waterfall-display/), constellation plot, and call log all visible at once.

## Sources

[^wiki]: [Computer monitor](https://en.wikipedia.org/wiki/Computer_monitor) — Wikipedia, on display devices for computers.
