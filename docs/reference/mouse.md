---
slug: mouse
title: Mouse
entry_type: hardware
category: hw-personal-computers
description: A mouse is a hand-held pointing device that tracks motion on a surface to move an on-screen cursor, with buttons and a scroll wheel for selecting and navigating.
keywords: mouse, pointing device, optical mouse, trackball, cursor, scroll wheel, input device, trackpad
infobox:
  - { label: Type, value: Input peripheral }
  - { label: Sensing, value: Optical / laser }
  - { label: Controls, value: Buttons, scroll wheel }
  - { label: Connects via, value: USB, Bluetooth }
see_also: [peripheral, keyboard, personal-computer, desktop-computer, touchscreen]
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_mouse
---

A **mouse** is a hand-held pointing device that tracks motion across a surface to move a cursor on screen, with buttons and usually a scroll wheel for selecting and navigating.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A mouse and how it works: a top view shows the left and right buttons and a scroll wheel, a bottom callout shows the optical sensor — an LED or laser plus a tiny camera reading the surface — and an arrow maps that motion to the cursor moving on a screen." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <path d="M70 40 Q70 24 100 24 Q130 24 130 40 L130 120 Q130 140 100 140 Q70 140 70 120 Z" fill="currentColor" fill-opacity="0.05"/>
    <path d="M100 24 L100 78"/>
    <rect x="94" y="40" width="12" height="20" rx="3" fill="currentColor" fill-opacity="0.16"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="55" y="48" text-anchor="end">left</text>
    <text x="145" y="48">right</text>
    <text x="118" y="52">wheel</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <circle cx="100" cy="150" r="6"/>
    <path d="M100 140 L100 156" stroke-dasharray="0"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7" text-anchor="middle">
    <text x="100" y="170">optical sensor (LED + camera)</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <path d="M155 82 H250" stroke-dasharray="4 3"/>
    <path d="M242 76 L252 82 L242 88" stroke-dasharray="0"/>
    <rect x="270" y="34" width="160" height="96" rx="4"/>
    <rect x="278" y="42" width="144" height="80" fill="currentColor" fill-opacity="0.06"/>
    <path d="M330 66 L330 100 L340 92 L346 104 L352 100 L344 90 L356 88 Z" fill="currentColor" fill-opacity="0.9"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5">
    <text x="200" y="76">motion</text>
    <text x="350" y="118" fill-opacity="0.85">cursor on screen</text>
  </g>
</svg>
<figcaption>Move the mouse and its optical sensor reads how far it has traveled across the surface; that motion drives the on-screen cursor, while the buttons and scroll wheel send their own click and scroll events to the operating system.</figcaption>
</figure>

## Overview

A mouse is an input [peripheral](/reference/peripheral/). Modern mice sense movement optically — an LED or laser plus a tiny camera reads the surface and reports how far the device has moved — replacing the old mechanical rolling ball. The position data drives the on-screen cursor, while button clicks and wheel scrolls send their own events to the [operating system](/reference/operating-system/).

Variants trade off precision, ergonomics, and space. Connection is over [USB](/reference/usb/) or Bluetooth, and a wireless mouse adds a battery and a receiver dongle or a direct radio link.

## Types

The pointing job can be done several ways:

| Type | How it senses | Notes |
|------|---------------|-------|
| Optical mouse | LED + camera on surface | Standard, cheap |
| Laser mouse | Laser + camera | Works on more surfaces |
| Trackball | You spin a fixed ball | Stays in one spot |
| Trackpad | Finger on a touch surface | Built into laptops |
| Gaming mouse | High-precision sensor, extra buttons | Tunable sensitivity |

## Where it fits

The mouse and the [keyboard](/reference/keyboard/) are the standard input pair for any [desktop computer](/reference/desktop-computer/); a [touchscreen](/reference/touchscreen/) replaces both on phones and tablets. Choice comes down to grip, sensitivity, and whether you want extra buttons. For panning a wide spectrum waterfall or zooming into a GopherTrunk plot, a smooth scroll wheel and steady tracking are the features that actually matter.

## Sources

[^wiki]: [Computer mouse](https://en.wikipedia.org/wiki/Computer_mouse) — Wikipedia, on the mouse as a pointing input device.
