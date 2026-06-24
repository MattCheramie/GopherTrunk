---
slug: thin-client
title: Thin client
entry_type: hardware
category: hw-personal-computers
description: A thin client is a minimal computer that does little work itself, instead connecting to a server or virtual desktop that runs the applications and stores the data centrally.
keywords: thin client, zero client, VDI, remote desktop, virtual desktop, terminal
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

## Overview

A thin client is essentially a screen, [keyboard](/reference/keyboard/), and [mouse](/reference/mouse/) bolted to just enough hardware to draw a remote session — a low-power [CPU](/reference/central-processing-unit/), a little [RAM](/reference/random-access-memory/), and a network port, with no spinning disk and often no local [storage](/reference/data-storage/) at all. The real computing happens centrally, frequently on a [virtualized](/reference/virtualization/) desktop infrastructure (VDI), and the thin client simply streams the picture back and forth. A *zero client* takes this further, with firmware fixed to a single remote protocol.

## Where it fits

Thin clients suit large organizations — call centers, hospitals, schools — where central management, security, and low per-seat cost matter more than local flexibility. Because nothing important lives on the device, a failed unit is swapped in minutes and stolen hardware leaks no data. A [Chromebook](/reference/chromebook/) is a consumer cousin of the idea. The limits are the same as that idea's: take away the network or the server and a thin client can do almost nothing, so it is a poor fit for standalone or offline compute like local SDR processing.

## Sources

[^wiki]: [Thin client](https://en.wikipedia.org/wiki/Thin_client) — Wikipedia, on minimal client computers that rely on central servers.
