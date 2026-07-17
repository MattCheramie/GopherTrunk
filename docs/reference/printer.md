---
slug: printer
title: Printer
entry_type: hardware
category: hw-personal-computers
description: A printer is an output peripheral that puts a computer's text and images onto paper, the two main consumer types being inkjet and laser, connected over USB, Wi-Fi, or a network.
keywords: printer, inkjet, laser printer, all-in-one printer, output device, USB printer, network printer, toner
infobox:
  - { label: Type, value: Output peripheral }
  - { label: Common types, value: Inkjet, laser }
  - { label: Output, value: Text and images on paper }
  - { label: Connects via, value: USB, Wi-Fi, network }
see_also: [peripheral, personal-computer, computer-monitor, usb, wi-fi]
cite_urls:
  - https://en.wikipedia.org/wiki/Printer_(computing)
---

A **printer** is an output [peripheral](/reference/peripheral/) that puts a computer's text and images onto paper, turning a digital document into a physical page.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A comparison of the two main printer types. An inkjet on the left moves a printhead across the page and sprays tiny droplets of liquid ink. A laser on the right charges a rotating drum, coats it with powdered toner, and fuses the toner onto the page with heat." xmlns="http://www.w3.org/2000/svg">
  <text x="115" y="20" fill="currentColor" stroke="none" text-anchor="middle" font-size="9" font-weight="600">inkjet</text>
  <text x="345" y="20" fill="currentColor" stroke="none" text-anchor="middle" font-size="9" font-weight="600">laser</text>
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="70" y="34" width="46" height="18" rx="2" fill="currentColor" fill-opacity="0.14"/>
    <path d="M116 52 H150" stroke-dasharray="3 2"/>
    <path d="M93 52 L90 70 M99 52 L97 70 M105 52 L104 70"/>
    <rect x="50" y="86" width="130" height="54" rx="2" fill="currentColor" fill-opacity="0.05"/>
    <path d="M62 100 H120 M62 110 H140 M62 120 H108"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7">
    <text x="123" y="47">printhead</text>
    <text x="128" y="72">droplets</text>
    <text x="115" y="156" text-anchor="middle" fill-opacity="0.85">sprays liquid ink</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <circle cx="320" cy="52" r="18" fill="currentColor" fill-opacity="0.08"/>
    <path d="M320 70 L320 86" stroke-dasharray="3 2"/>
    <rect x="360" y="42" width="24" height="20" rx="2" fill="currentColor" fill-opacity="0.14"/>
    <rect x="280" y="90" width="130" height="50" rx="2" fill="currentColor" fill-opacity="0.05"/>
    <path d="M292 104 H352 M292 114 H372 M292 124 H340"/>
  </g>
  <g fill="currentColor" stroke="none" font-size="7">
    <text x="320" y="55" text-anchor="middle">drum</text>
    <text x="372" y="55" text-anchor="middle">toner</text>
    <text x="345" y="156" text-anchor="middle" fill-opacity="0.85">fuses powder with heat</text>
  </g>
</svg>
<figcaption>The two dominant printer types work very differently: an inkjet sweeps a printhead over the page and sprays liquid-ink droplets, while a laser charges a drum, coats it with toner powder, and fuses that powder onto the paper with heat.</figcaption>
</figure>

## Overview

Two technologies dominate the consumer and office market. *Inkjet* printers spray tiny droplets of liquid ink and excel at color and photos; *laser* printers fuse powdered toner onto the page and are faster, cheaper per page, and crisper for plain text. Many units are *all-in-one* devices that also scan and copy.

A printer is a pure output device, receiving a print job from the [operating system](/reference/operating-system/) via a driver, and connects over [USB](/reference/usb/), [Wi-Fi](/reference/wi-fi/), or a wired network so it can be shared among several machines.

## Inkjet vs laser

The choice usually comes down to what and how much you print:

| Trait | Inkjet | Laser |
|-------|--------|-------|
| Marks with | Liquid ink droplets | Fused toner powder |
| Best at | Color, photos | Fast, crisp text |
| Speed | Slower | Faster |
| Cost per page | Higher | Lower |
| Up-front cost | Lower | Higher |

## Where it fits

For occasional documents an inkjet is cheap to buy; for steady text volume a laser saves money and frustration over time, since ink and toner costs often dwarf the price of the hardware itself. A networked printer lets every machine in a home or office print without a direct cable. In a software-centric setup like GopherTrunk a printer is incidental — useful for a paper frequency list or a printed log, but nothing the capture path needs.

## Sources

[^wiki]: [Printer (computing)](https://en.wikipedia.org/wiki/Printer_(computing)) — Wikipedia, on printers as output devices.
