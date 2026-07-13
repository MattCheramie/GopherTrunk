---
slug: js8call
title: JS8Call
entry_type: technology
category: amateur-digital
description: "JS8Call is an amateur-radio keyboard-to-keyboard chat mode derived from FT8's weak-signal 8-FSK waveform, adding free-form messaging, relaying, and store-and-forward."
keywords: JS8Call, JS8, weak signal, keyboard chat, FT8 derived, 8-FSK, messaging, relay, store and forward, amateur radio, Jordan Sherer
aka: [JS8Call, JS8]
autolink: true
infobox:
  - { label: Type, value: Weak-signal keyboard chat mode }
  - { label: Idea, value: FT8 waveform + free-form conversation }
  - { label: Examples, value: Text chat, relaying, store-and-forward, APRS gateways }
see_also: [ft8, m-ary-fsk, ldpc-code, joe-taylor]
cite_urls:
  - https://en.wikipedia.org/wiki/JS8Call
  - http://js8call.com/
---

**JS8Call** is an amateur-radio **keyboard-to-keyboard chat mode** that takes the
weak-signal waveform of [FT8](/reference/ft8/) and turns it into a real conversational
system. Where FT8 exchanges only rigid, pre-formatted station reports, JS8Call layers
**free-form text messaging** on top of the same [8-FSK](/reference/m-ary-fsk/) tones —
adding relaying, group calls, and store-and-forward so operators can hold slow chats, pass
messages, and reach stations they can't hear directly, all at signal levels far below the
noise floor.[^wiki] It is best understood not as a new waveform but as an application built
around a proven one.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="JS8Call reuses the FT8 8-FSK physical layer but replaces fixed station reports with free-form text, relaying, and store-and-forward messaging." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="js8ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <rect x="25" y="40" width="120" height="40" fill="currentColor" fill-opacity="0.12" stroke="currentColor"/>
  <text x="85" y="57" font-size="8.5" fill="currentColor" text-anchor="middle">FT8 8-FSK</text>
  <text x="85" y="70" font-size="7.5" fill="currentColor" text-anchor="middle">physical layer</text>
  <line x1="145" y1="60" x2="200" y2="60" stroke="currentColor" marker-end="url(#js8ar)"/>
  <rect x="200" y="30" width="235" height="60" fill="none" stroke="currentColor"/>
  <text x="317" y="46" font-size="8.5" fill="currentColor" text-anchor="middle">JS8Call application layer</text>
  <g font-size="7.5" fill="currentColor" text-anchor="middle"><text x="255" y="66">free text</text><text x="317" y="66">relay</text><text x="385" y="66">store &amp; fwd</text><text x="317" y="80">directed calls · groups</text></g>
</svg>
<figcaption>JS8Call keeps FT8's sub-noise 8-FSK physical layer and adds a messaging layer: free text, directed and group calls, relaying, and store-and-forward.</figcaption>
</figure>

## How it works

At the radio layer JS8Call sends the same Gaussian 8-tone FSK symbols as FT8 and protects
them with the same [LDPC](/reference/ldpc-code/) forward error correction, so its
sub-noise sensitivity is essentially inherited. The key changes are above the waveform:

- **Slower, selectable speeds.** JS8Call offers Normal, Fast, Turbo, and Slow sub-modes
  (roughly 10-, 6-, 4-, and 30-second frame times), trading throughput against sensitivity.
- **Free-form text.** Instead of FT8's fixed 77-bit report, JS8Call packs arbitrary
  characters into a stream of frames, so operators type real sentences that assemble across
  multiple transmissions.
- **Directed protocol.** Messages can be addressed to a specific callsign or group, can
  query a station's status or SNR, and can be **relayed** hop-by-hop through intermediate
  stations.
- **Store-and-forward.** A station can leave a message for another that isn't currently on;
  it is held and delivered when that station appears.

Because transmissions are not locked to strict UTC slots the way FT8's are, JS8Call is more
forgiving of timing, though an accurate clock still helps decoding.

## Relevance to SDR

JS8Call occupies HF amateur allocations (with common calling frequencies such as 7.078 and
14.078 MHz) and is popular for low-power, emergency, and off-grid messaging where its
weak-signal reach beats voice. Any receiver that can deliver clean SSB audio works as a
front end: a traditional HF transceiver, or an SDR such as an [RTL-SDR](/reference/rtl-sdr/)
with an [upconverter](/reference/upconverter/), an [Airspy HF+](/reference/airspy-hf-plus/),
or an [SDRplay](/reference/sdrplay-rsp1a/) piped into the JS8Call application.

[GopherTrunk](/reference/software-defined-radio/) does not decode JS8Call — it is a
trunked land-mobile scanner (P25, DMR, NXDN, TETRA and similar), not an HF weak-signal
messaging client. JS8Call is decoded by its own free software of the same name, which
descends directly from the [WSJT-X](/reference/ft8/) codebase.[^home]

## Sources

[^wiki]: [JS8Call](https://en.wikipedia.org/wiki/JS8Call) — Wikipedia, for JS8Call's derivation from FT8, its free-form messaging, relaying, store-and-forward, and sub-mode speeds.
[^home]: [JS8Call](http://js8call.com/) — the official project site, documenting the application, its operating modes, and its relationship to the FT8 waveform.
