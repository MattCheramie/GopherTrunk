---
slug: zadig
title: Zadig
entry_type: hardware
category: hardware
description: Zadig is a Windows utility that installs the generic WinUSB driver onto an SDR dongle, replacing the default TV-tuner driver so SDR software can access the device.
keywords: Zadig, WinUSB, RTL-SDR driver, Windows, libusb, DVB-T driver, USB driver
aka: [Zadig, WinUSB]
autolink: true
see_also: [rtl-sdr, rtl2832u, rtl-tcp, soapysdr]
related_lessons:
  - { title: "SDR hardware — RTL-SDR, HackRF, Airspy", url: /learn/rf-sdr/sdr-hardware/ }
external:
  - { title: "Zadig", url: https://zadig.akeo.ie/ }
---

**Zadig** is a small Windows utility that installs the generic **WinUSB** driver onto an
SDR dongle. Out of the box, Windows binds an [RTL-SDR](/reference/rtl-sdr/) to its
TV-tuner (DVB-T) driver; Zadig replaces that with WinUSB so SDR software can talk to the
device directly.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A dongle's default TV-tuner driver being replaced by the WinUSB driver via Zadig." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="44" width="90" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 3"/><text x="75" y="63" text-anchor="middle" font-size="8.5" fill="currentColor">DVB-T driver</text>
  <line x1="125" y1="59" x2="185" y2="59" stroke="currentColor" marker-end="url(#zar)"/><text x="155" y="51" text-anchor="middle" font-size="8" fill="currentColor">Zadig</text>
  <rect x="195" y="44" width="90" height="30" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.2"/><text x="240" y="63" text-anchor="middle" font-size="8.5" fill="currentColor">WinUSB</text>
  <line x1="290" y1="59" x2="350" y2="59" stroke="currentColor" marker-end="url(#zar)"/>
  <rect x="360" y="44" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="400" y="63" text-anchor="middle" font-size="8.5" fill="currentColor">SDR app</text>
  <defs><marker id="zar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Zadig swaps the dongle's default Windows driver for WinUSB so SDR software can access it.</figcaption>
</figure>

## Overview

GopherTrunk's Windows installer bundles Zadig to automate this step. On Linux and macOS
the equivalent access is handled by libusb/IOKit without a separate tool.
