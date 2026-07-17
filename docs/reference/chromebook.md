---
slug: chromebook
title: Chromebook
entry_type: hardware
category: hw-personal-computers
description: A Chromebook is a laptop or tablet running Google's ChromeOS, a lightweight operating system built around the Chrome browser and web apps, with most data stored in the cloud.
keywords: Chromebook, ChromeOS, Chrome browser, web apps, education laptop, cloud computing, Android apps, Linux container
aka: [ChromeOS device]
autolink: true
infobox:
  - { label: Type, value: Laptop / tablet }
  - { label: OS, value: ChromeOS (Google) }
  - { label: Apps, value: Web, Android, some Linux }
  - { label: Storage, value: Small local + cloud }
  - { label: Common in, value: Education }
see_also: [laptop, operating-system, personal-computer, thin-client, cloud-computing]
related_lessons:
  - { title: "Laptops", url: /learn/intro-hardware/laptops/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Chromebook
---

A **Chromebook** is a [laptop](/reference/laptop/) or tablet that runs Google's ChromeOS, a lightweight [operating system](/reference/operating-system/) built around the Chrome web browser and cloud services.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="A Chromebook leaning on the cloud: a laptop runs a browser locally with only a small SSD, while web apps, documents, and settings live on remote cloud servers reached over the network, so most compute and storage happen off the device." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <rect x="40" y="70" width="150" height="90" rx="4"/>
    <rect x="52" y="82" width="126" height="52" fill="currentColor" fill-opacity="0.08"/>
    <path d="M30 160 H200 L210 172 H20 Z" fill="currentColor" fill-opacity="0.12"/>
    <rect x="60" y="118" width="30" height="10" rx="1"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5">
    <text x="115" y="96" font-size="8">Chrome browser</text>
    <text x="75" y="126" font-size="6.5">small SSD</text>
    <text x="115" y="152" fill-opacity="0.85">local: draw the page</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.3">
    <path d="M300 44 a26 20 0 0 1 4 -40 a30 24 0 0 1 56 6 a22 18 0 0 1 6 34 Z" fill="currentColor" fill-opacity="0.1"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7.5">
    <text x="332" y="30" font-size="8">cloud</text>
    <text x="332" y="60">web apps · files · settings</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <path d="M195 100 C250 100 250 40 300 34" stroke-dasharray="4 3"/>
    <path d="M298 40 L300 34 L294 33" stroke-dasharray="0"/>
  </g>
  <text x="250" y="118" fill="currentColor" stroke="none" text-anchor="middle" font-size="7" fill-opacity="0.85">network</text>
</svg>
<figcaption>A Chromebook keeps the hardware light and pushes the heavy lifting to the cloud: the device runs the Chrome browser and holds little locally, while apps, documents, and settings sync to remote servers over the network.</figcaption>
</figure>

## Overview

Where a traditional [personal computer](/reference/personal-computer/) installs heavyweight desktop applications, a Chromebook leans on web apps that run inside the browser, with most documents and settings synced to the cloud. Hardware is modest by design — a low-power [CPU](/reference/central-processing-unit/), modest [RAM](/reference/random-access-memory/), and a small [SSD](/reference/solid-state-drive/) or [eMMC](/reference/emmc/) — which keeps Chromebooks cheap, fast to boot, and long on battery life.

Modern models widen the software story beyond the browser: many run Android apps from the Play Store and an optional Linux container for development, so a Chromebook is no longer strictly a web-only device. It still assumes a network for most real work, and its automatic-update model and central management are part of why schools favor it.

## How it compares

A Chromebook sits between a full laptop and a pure [thin client](/reference/thin-client/):

| Machine | OS | Local compute | Runs offline | Typical cost |
|---------|----|--------------:|--------------|--------------|
| Chromebook | ChromeOS | Low | Partly | Low |
| Windows/Mac laptop | Full desktop OS | High | Fully | Medium–high |
| Thin client | Minimal firmware | Very low | No | Low |

## Where it fits

Chromebooks dominate classrooms and suit anyone whose work lives in a browser: email, documents, video calls, and SaaS tools. They behave somewhat like a self-contained thin client, leaning on remote services rather than local power, but keep enough local capability to work through a brief outage. The trade-off is limited offline software and weak local compute — heavy native workloads, including most SDR DSP, want a full [laptop](/reference/laptop/) or [desktop](/reference/desktop-computer/) instead.

## Sources

[^wiki]: [Chromebook](https://en.wikipedia.org/wiki/Chromebook) — Wikipedia, on Chromebooks and ChromeOS.
