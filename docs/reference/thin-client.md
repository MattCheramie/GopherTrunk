---
slug: thin-client
title: Thin client
entry_type: hardware
category: hw-personal-computers
description: A thin client is a minimal computer that does little work itself, instead connecting to a server or virtual desktop that runs the applications and stores the data centrally.
keywords: thin client, zero client, VDI, remote desktop, virtual desktop, terminal, central server
aka: [Thin client, Zero client]
infobox:
  - { label: Type, value: Minimal networked PC }
  - { label: Does, value: Display a remote session }
  - { label: Relies on, value: Central server / VDI }
  - { label: Local power, value: Very low }
see_also: [personal-computer, server, virtualization, chromebook, mini-pc]
cite_urls:
  - https://en.wikipedia.org/wiki/Thin_client
---

A **thin client** is a minimal computer that does little processing on its own, instead connecting over a network to a [server](/reference/server/) or virtual desktop that runs the applications and holds the data.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 185" role="img" aria-label="A thin-client topology: three small thin clients on the left, each just a screen with minimal hardware, connect over a network to one central server on the right that runs the virtual desktops and stores all the data. Screen images flow back to the clients while keystrokes flow to the server." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="none" stroke-width="1.2">
    <rect x="30" y="26" width="60" height="40" rx="3" fill="currentColor" fill-opacity="0.06"/>
    <rect x="30" y="74" width="60" height="40" rx="3" fill="currentColor" fill-opacity="0.06"/>
    <rect x="30" y="122" width="60" height="40" rx="3" fill="currentColor" fill-opacity="0.06"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle" font-size="7">
    <text x="60" y="49">thin</text>
    <text x="60" y="59">client</text>
    <text x="60" y="97">thin</text>
    <text x="60" y="107">client</text>
    <text x="60" y="145">thin</text>
    <text x="60" y="155">client</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.1" stroke-dasharray="4 3">
    <path d="M90 46 L210 90"/>
    <path d="M90 94 L210 94"/>
    <path d="M90 142 L210 98"/>
  </g>
  <text x="150" y="80" fill="currentColor" stroke="none" text-anchor="middle" font-size="7" fill-opacity="0.85">network</text>
  <g stroke="currentColor" fill="none" stroke-width="1.4">
    <rect x="330" y="46" width="100" height="96" rx="4" fill="currentColor" fill-opacity="0.1"/>
    <path d="M340 66 H420 M340 82 H420 M340 98 H420 M340 114 H420"/>
    <circle cx="348" cy="58" r="2" fill="currentColor"/>
  </g>
  <g fill="currentColor" stroke="none" text-anchor="middle">
    <text x="380" y="158" font-size="8" font-weight="600">server / VDI</text>
    <text x="380" y="172" font-size="7" fill-opacity="0.85">runs the apps · holds the data</text>
    <text x="255" y="112" font-size="6.5" fill-opacity="0.85">screen &#8592; · keys &#8594;</text>
  </g>
</svg>
<figcaption>A thin client is little more than a screen on the network: many of them share one central server that runs the actual applications and stores the data, streaming each user a picture back while sending keystrokes and clicks the other way.</figcaption>
</figure>

## Overview

A thin client is essentially a screen, [keyboard](/reference/keyboard/), and [mouse](/reference/mouse/) bolted to just enough hardware to draw a remote session — a low-power [CPU](/reference/central-processing-unit/), a little [RAM](/reference/random-access-memory/), and a network port, with no spinning disk and often no local [storage](/reference/data-storage/) at all. The real computing happens centrally, frequently on a [virtualized](/reference/virtualization/) desktop infrastructure (VDI), and the thin client simply streams the picture back and forth.

A *zero client* takes this further, with firmware fixed to a single remote protocol and essentially nothing to configure. The appeal in both cases is that the valuable state lives on the server, not the device on the desk.

## How it compares

Where the computing and the data actually live is the whole distinction:

| Machine | Runs apps | Stores data | Works standalone |
|---------|-----------|-------------|------------------|
| Thin client | On the server | On the server | No |
| Chromebook | Browser + cloud | Mostly cloud | Partly |
| Full PC | Locally | Locally | Yes |

## Where it fits

Thin clients suit large organizations — call centers, hospitals, schools — where central management, security, and low per-seat cost matter more than local flexibility. Because nothing important lives on the device, a failed unit is swapped in minutes and stolen hardware leaks no data. A [Chromebook](/reference/chromebook/) is a consumer cousin of the idea. The limits are the same: take away the network or the server and a thin client can do almost nothing, so it is a poor fit for standalone or offline compute like local SDR processing.

## Sources

[^wiki]: [Thin client](https://en.wikipedia.org/wiki/Thin_client) — Wikipedia, on minimal client computers that rely on central servers.
