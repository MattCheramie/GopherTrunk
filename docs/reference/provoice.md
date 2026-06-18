---
slug: provoice
title: ProVoice
entry_type: protocol
category: protocols
description: ProVoice is the digital-voice option for EDACS trunked systems, using an AMBE-family vocoder over the otherwise analog-trunking EDACS air interface.
keywords: ProVoice, EDACS digital voice, M/A-COM, AMBE, digital trunking
aka: [ProVoice]
autolink: true
see_also: [edacs, ambe, vocoder, trunked-radio]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/rf-sdr/protocol-landscape/ }
external:
  - { title: "Enhanced Digital Access Communications System (Wikipedia)", url: https://en.wikipedia.org/wiki/Enhanced_Digital_Access_Communications_System }
---

**ProVoice** is the **digital-voice option for [EDACS](/reference/edacs/)** trunked
systems (GE/Ericsson, later M/A-COM). It replaces EDACS's analog FM voice with a digital
[AMBE](/reference/ambe/)-family [vocoder](/reference/vocoder/) while keeping the EDACS
control-channel signalling.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 110" role="img" aria-label="An EDACS control channel assigning a voice channel that carries digital ProVoice AMBE frames instead of analog FM." xmlns="http://www.w3.org/2000/svg">
  <rect x="30" y="24" width="400" height="22" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="230" y="39" text-anchor="middle" font-size="8.5" fill="currentColor">EDACS control channel</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none"><rect x="120" y="64" width="60" height="26"/><rect x="180" y="64" width="60" height="26"/><rect x="240" y="64" width="60" height="26"/><rect x="300" y="64" width="60" height="26"/></g>
  <text x="240" y="105" text-anchor="middle" font-size="8" fill="currentColor">ProVoice = digital AMBE frames (vs analog FM)</text>
  <line x1="230" y1="46" x2="240" y2="62" stroke="currentColor" stroke-dasharray="3 3"/>
</svg>
<figcaption>ProVoice carries digital AMBE voice over EDACS, in place of the system's analog FM voice channels.</figcaption>
</figure>

## Overview

Because ProVoice uses a proprietary vocoder, decoding it has historically required
licensed components; the EDACS trunking layer itself is followed the same way regardless
of whether the voice is analog or ProVoice.
