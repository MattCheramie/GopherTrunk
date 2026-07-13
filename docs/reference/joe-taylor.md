---
slug: joe-taylor
title: Joe Taylor (K1JT)
entry_type: person
category: people
description: "Joe Taylor (K1JT, born 1941) is a Nobel-winning radio astronomer and ham who created the WSJT weak-signal modes including FT8, JT65 and WSPR."
keywords: Joe Taylor, K1JT, WSJT, WSJT-X, FT8, JT65, WSPR, weak signal, radio astronomy, binary pulsar, Nobel Prize, amateur radio
aka: [Joe Taylor, K1JT, Joseph Hooton Taylor Jr.]
autolink: true
infobox:
  - { label: Lived, value: "1941–" }
  - { label: Field, value: Radio astronomy / amateur radio }
  - { label: Known for, value: "WSJT modes (FT8, JT65, WSPR)" }
see_also: [ft8, wspr, jt65, jt9, ft4, forward-error-correction]
cite_urls:
  - https://en.wikipedia.org/wiki/Joseph_Hooton_Taylor_Jr.
  - https://wsjt.sourceforge.io/
---

**Joe Taylor** (amateur call sign **K1JT**, born 1941) is an American radio astronomer,
Nobel laureate, and ham radio operator who designed the **WSJT** family of weak-signal
digital modes — including [FT8](/reference/ft8/), [JT65](/reference/jt65/),
[JT9](/reference/jt9/), [FT4](/reference/ft4/), and the propagation-sensing beacon mode
[WSPR](/reference/wspr/).[^wiki] His software let ordinary amateurs make contacts at
signal levels far below the noise floor, changing what a small station and a modest
antenna can accomplish.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A very weak signal buried below the noise floor is recovered by heavy forward error correction to yield a decoded message, illustrating the WSJT weak-signal approach." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="jtar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1" fill="none" opacity="0.5">
    <path d="M20 70 L40 62 L55 78 L70 60 L85 80 L100 64 L115 76 L130 58 L145 82 L160 66 L175 74 L190 60"/>
  </g>
  <line x1="20" y1="88" x2="200" y2="88" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3"/>
  <text x="20" y="102" font-size="8" fill="currentColor">noise floor</text>
  <line x1="205" y1="70" x2="255" y2="70" stroke="currentColor" stroke-width="1.2" marker-end="url(#jtar)"/>
  <rect x="258" y="52" width="90" height="36" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/>
  <text x="303" y="68" font-size="8.5" fill="currentColor" text-anchor="middle">strong FEC</text>
  <text x="303" y="80" font-size="8.5" fill="currentColor" text-anchor="middle">+ 15 s frame</text>
  <line x1="348" y1="70" x2="398" y2="70" stroke="currentColor" stroke-width="1.2" marker-end="url(#jtar)"/>
  <text x="428" y="73" font-size="9" fill="currentColor" text-anchor="middle">decoded</text>
</svg>
<figcaption>WSJT modes pull messages out of signals well below the noise floor by pairing dense forward error correction with long, precisely timed frames.</figcaption>
</figure>

## Life and work

Joseph Hooton Taylor Jr. earned his doctorate in astronomy and spent his career at the
University of Massachusetts and then Princeton University. In 1974, with his student
Russell Hulse, he discovered the first **binary pulsar**, PSR B1913+16 — a pair of
neutron stars whose orbit decays exactly as Einstein's general relativity predicts,
providing the first indirect evidence for gravitational waves. The discovery earned the
two the **1993 Nobel Prize in Physics**.[^wiki]

Taylor has been a licensed amateur since his teens. His scientific work — detecting and
timing faint pulsar signals against cosmic noise — is essentially a weak-signal
detection problem, and that expertise carried directly into his later contributions to
amateur radio.

## Contribution

Beginning around 2001, Taylor released **WSJT** ("Weak Signal communication, by K1JT"),
a suite of digital modes engineered to complete contacts when the signal is at or below
the [noise floor](/reference/noise-floor/). Each mode trades speed for sensitivity by
combining three ideas: rigid time synchronization (transmit and receive slots aligned to
the computer clock), narrow tone-based modulation, and heavy
[forward error correction](/reference/forward-error-correction/) that lets the decoder
reconstruct a message from fragments.

- **JT65** (2003) was built for Earth–Moon–Earth ("moonbounce") and meteor-scatter work,
  using 65-tone frequency-shift keying and a Reed–Solomon code.
- **WSPR** (Weak Signal Propagation Reporter, 2008) is a low-power beacon mode: stations
  transmit a tiny structured message, and a global network logs who heard whom, mapping
  live propagation.
- **FT8** (2017), developed with Steven Franke (K9AN), packs a contact into 15-second
  slots and became the most popular mode on the amateur bands within a couple of years.
  **FT4** is a faster contest-oriented variant, and **JT9** targets the LF/MF/HF bands.

These modes are distributed in the open-source **WSJT-X** application, which decodes the
audio from a receiver — including a [software-defined radio](/reference/software-defined-radio/) —
and displays decodes on a waterfall.

## Legacy

FT8 alone transformed amateur operating practice: it made worldwide contacts routine for
stations running a few watts into a compromise antenna, and it filled the HF bands with
machine-precise 15-second exchanges. Taylor's design philosophy — squeeze the last
decibel out of a channel with disciplined timing and strong coding — is the same
principle that governs deep-space telemetry and modern digital radio generally, echoing
the capacity limits framed by [Claude Shannon](/reference/claude-shannon/). WSJT-X
remains actively developed by a volunteer team, with Taylor as its guiding figure.

## Sources

[^wiki]: [Joseph Hooton Taylor Jr.](https://en.wikipedia.org/wiki/Joseph_Hooton_Taylor_Jr.) — Wikipedia, for his biography, the binary-pulsar discovery, the 1993 Nobel Prize, and the WSJT modes.
