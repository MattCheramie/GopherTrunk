---
slug: tsbk
title: TSBK
entry_type: term
category: trunked-radio
description: A TSBK (trunking signalling block) is the unit of control-channel signalling in P25 — the message that announces channel grants, registrations, and system parameters.
keywords: TSBK, trunking signalling block, P25 control channel, channel grant, single block, signalling
aka: [TSBK, "trunking signalling block", "trunking signaling block"]
autolink: true
see_also: [control-channel, channel-grant, project-25, p25-phase-1, csbk]
related_lessons:
  - { title: "What is trunked radio?", url: /learn/rf-sdr/what-is-trunking/ }
related_reading:
  - { title: "SDR Internals, Part 11: The trunking engine & event bus", url: /blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
---

A **TSBK** (**trunking signalling block**) is the unit of control-channel signalling in
[P25](/reference/project-25/).[^wiki] Each TSBK is a short, error-protected message on the
[control channel](/reference/control-channel/) carrying one piece of system business — a
[channel grant](/reference/channel-grant/), a registration, an affiliation, or a system
parameter.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="A control channel carrying a stream of TSBK blocks, one of which is a channel grant." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="30" width="400" height="24" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="230" y="46" text-anchor="middle" font-size="8.5" fill="currentColor">P25 control channel</text>
  <g stroke="currentColor" stroke-width="1.1"><rect x="40" y="66" width="80" height="26" fill="none"/><rect x="130" y="66" width="80" height="26" fill="currentColor" fill-opacity="0.22"/><rect x="220" y="66" width="80" height="26" fill="none"/><rect x="310" y="66" width="80" height="26" fill="none"/></g>
  <g font-size="8" fill="currentColor" text-anchor="middle"><text x="80" y="83">TSBK</text><text x="170" y="80">TSBK</text><text x="170" y="90" font-size="7">(grant)</text><text x="260" y="83">TSBK</text><text x="350" y="83">TSBK</text></g>
</svg>
<figcaption>The P25 control channel is a stream of TSBKs; one announces a grant pointing radios to a voice channel.</figcaption>
</figure>

## Overview

Decoding TSBKs is exactly how a scanner follows a P25 system: read the grant TSBKs and
retune to the assigned [voice channel](/reference/voice-channel/) in step with the radios.
P25 Phase 2 uses *MAC PDU* messages for the same role. DMR's equivalent is the
[CSBK](/reference/csbk/).

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its control-channel signalling.
