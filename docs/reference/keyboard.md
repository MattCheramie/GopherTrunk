---
slug: keyboard
title: Keyboard
entry_type: hardware
category: hw-personal-computers
description: A keyboard is the primary text-input peripheral for a computer, a grid of keys that send character and command codes, available in membrane and mechanical types and many layouts.
keywords: keyboard, mechanical keyboard, membrane keyboard, QWERTY, key switch, rubber dome, input device, keycap
infobox:
  - { label: Type, value: Input peripheral }
  - { label: Switches, value: Membrane or mechanical }
  - { label: Layouts, value: QWERTY, AZERTY, Dvorak, … }
  - { label: Connects via, value: USB, Bluetooth }
see_also: [peripheral, mouse, personal-computer, desktop-computer, usb]
cite_urls:
  - https://en.wikipedia.org/wiki/Computer_keyboard
---

A **keyboard** is the primary text-input device for a computer: a grid of keys that, when pressed, send character and command codes to the machine.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A cutaway comparing two key switches. On the left a membrane key presses a keycap onto a rubber dome that collapses to close a contact on a printed sheet. On the right a mechanical key uses a spring-loaded plastic stem in a housing that snaps a metal contact closed, giving a distinct tactile press." xmlns="http://www.w3.org/2000/svg">
  <text x="115" y="20" fill="currentColor" stroke="none" text-anchor="middle" font-size="9" font-weight="600">membrane</text>
  <text x="345" y="20" fill="currentColor" stroke="none" text-anchor="middle" font-size="9" font-weight="600">mechanical</text>
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <path d="M70 40 H160 L150 58 H80 Z" fill="currentColor" fill-opacity="0.14"/>
    <path d="M92 58 Q115 92 138 58" fill="currentColor" fill-opacity="0.08"/>
    <path d="M60 118 H170"/>
    <path d="M60 126 H170"/>
    <circle cx="115" cy="118" r="3" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="163" y="50">keycap</text>
    <text x="145" y="82">rubber dome</text>
    <text x="120" y="140">contact sheets</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <path d="M300 36 H390 L380 54 H310 Z" fill="currentColor" fill-opacity="0.14"/>
    <rect x="322" y="56" width="46" height="54" fill="currentColor" fill-opacity="0.06"/>
    <path d="M330 58 V104 M340 58 V104" stroke-width="1"/>
    <path d="M348 60 q6 8 0 16 q-6 8 0 16 q6 8 0 16" stroke-width="1"/>
    <path d="M300 118 H390"/>
    <circle cx="345" cy="118" r="3" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7.5">
    <text x="393" y="46">keycap</text>
    <text x="286" y="86" text-anchor="end">stem</text>
    <text x="360" y="86">spring</text>
    <text x="345" y="140" text-anchor="middle">metal contact</text>
  </g>
  <text x="230" y="168" fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5" fill-opacity="0.85">both close a switch on each keypress and report a code over USB or Bluetooth</text>
</svg>
<figcaption>Under every key is a switch. A membrane key collapses a rubber dome to bridge two printed contact sheets — cheap and quiet; a mechanical key uses a sprung stem that snaps a metal contact closed — pricier, but crisper and longer-lived.</figcaption>
</figure>

## Overview

A keyboard is an input [peripheral](/reference/peripheral/). Each keypress is detected and translated into a code the [operating system](/reference/operating-system/) interprets as a letter, number, or command. Two switch technologies dominate: cheap, quiet *membrane* keyboards that press a contact through a rubber dome, and *mechanical* keyboards with an individual spring-loaded switch under each key that many typists prefer for feel and durability.

Layouts vary by language and preference — QWERTY is most common, with AZERTY, Dvorak, and others in use — and boards come in sizes from full (with a numeric pad) down to compact tenkeyless and 60% designs. Keyboards connect over [USB](/reference/usb/) or wirelessly via Bluetooth.

## Membrane vs mechanical

The switch type sets most of what a keyboard feels like and costs:

| Trait | Membrane | Mechanical |
|-------|----------|-----------|
| Under each key | Shared rubber dome | Individual switch |
| Feel | Soft, mushy | Distinct, tactile/clicky |
| Noise | Quiet | Often louder |
| Durability | Lower | Very high |
| Cost | Low | Higher |

## Where it fits

A keyboard is half of the basic input pair, alongside the [mouse](/reference/mouse/), that every [desktop computer](/reference/desktop-computer/) needs and that laptops build in. The choice is mostly ergonomic: switch feel, key spacing, and whether you want a numeric pad or a compact board. For long sessions at a GopherTrunk bench, a comfortable keyboard matters more than any spec.

## Sources

[^wiki]: [Computer keyboard](https://en.wikipedia.org/wiki/Computer_keyboard) — Wikipedia, on keyboards as text-entry input devices.
