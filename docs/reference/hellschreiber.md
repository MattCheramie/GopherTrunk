---
slug: hellschreiber
title: Hellschreiber
entry_type: protocol
category: amateur-digital
description: "Hellschreiber is a facsimile-style text mode that paints each character as a column-by-column dot pattern the operator reads by eye, making it a robust fuzzy mode on noisy HF."
keywords: Hellschreiber, Feld-Hell, Hell mode, fuzzy mode, on-off keying facsimile, Rudolf Hell, HF digital text, PSKHell, FMHell
aka: [Hellschreiber, Feld-Hell, Hell]
autolink: true
infobox:
  - { label: Type, value: HF facsimile text mode }
  - { label: Standards body, value: Amateur convention }
  - { label: Introduced, value: 1929 (Rudolf Hell) }
  - { label: Access, value: Simplex, one QSO per channel }
  - { label: Channel spacing, value: ~75–350 Hz (variant-dependent) }
  - { label: Modulation, value: On-off keyed dot facsimile }
  - { label: GopherTrunk support, value: Not decoded }
see_also: [rtty, morse-code, on-off-keying, frequency-shift-keying, single-sideband]
cite_urls:
  - https://en.wikipedia.org/wiki/Hellschreiber
---

**Hellschreiber** (German for "Hell writer," after inventor Rudolf Hell) is a text mode
that transmits characters not as data bits to be decoded, but as a **facsimile of the
printed glyph** — each letter is scanned as a grid of dots and sent column by column, so the
receiver simply paints the dots and the operator reads the words by eye.[^wiki] Because no
character-recognition step can fail, it is one of the classic **fuzzy modes**: noise sprinkles
speckle across the page but the human eye still resolves the letters, much as it copes with
weak [RTTY](/reference/rtty/) or faint [Morse](/reference/morse-code/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Hellschreiber scans each character as a column of dots and keys the carrier on for dark pixels and off for light ones, so the receiver reprints the glyph shape directly." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="hlar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g fill="currentColor">
    <rect x="30" y="30" width="10" height="10"/><rect x="30" y="42" width="10" height="10"/><rect x="30" y="54" width="10" height="10"/><rect x="30" y="66" width="10" height="10"/><rect x="30" y="78" width="10" height="10"/>
    <rect x="42" y="30" width="10" height="10"/><rect x="42" y="54" width="10" height="10"/>
    <rect x="54" y="30" width="10" height="10"/><rect x="54" y="54" width="10" height="10"/>
    <rect x="66" y="42" width="10" height="10"/>
  </g>
  <text x="52" y="100" font-size="8" fill="currentColor" text-anchor="middle">glyph as dot grid</text>
  <line x1="90" y1="60" x2="150" y2="60" stroke="currentColor" marker-end="url(#hlar)"/>
  <text x="120" y="52" font-size="8" fill="currentColor" text-anchor="middle">scan columns</text>
  <g stroke="currentColor" stroke-width="1.3" fill="none"><path d="M170 80 L185 80 L185 45 L200 45 L200 80 L215 80 L215 55 L235 55 L235 80 L255 80 L255 40 L275 40 L275 80 L430 80"/></g>
  <text x="300" y="100" font-size="8" fill="currentColor" text-anchor="middle">carrier keyed on/off per pixel →</text>
</svg>
<figcaption>Each glyph is scanned into a dot grid; the carrier is keyed on for dark pixels and off for light ones, so the receiver reprints the letter shape rather than decoding a code.</figcaption>
</figure>

## Overview

The classic variant, **Feld-Hell**, sends 245 characters per minute using a 7-pixel-high
font scanned in columns. A "dark" pixel keys the carrier **on**; a "light" pixel keys it
**off** — plain [on-off keying](/reference/on-off-keying/). At the receiver each received
pulse blackens the corresponding cell of a scrolling raster, and successive columns build up
the letters. There is deliberately **no forward error correction and no synchronisation to
lock**: the display just scrolls, and the reader's brain does the pattern recognition. To
counter timing drift, most software prints two stacked copies of the text so at least one row
is always readable.

## Variants

Modern sound-card software has spawned a family of Hell modes on the same visual principle:

- **Feld-Hell** — the original on-off-keyed mode, ~35 baud, ~75 Hz wide.
- **PSK-Hell / Hell-PSK** — replaces keying with phase-shift signalling for a narrower,
  cleaner signal.
- **FM-Hell (MT-Hell)** — shifts dot rows to different tones (multi-tone), improving
  sensitivity and tolerance to selective fading.
- **Slow-Hell** — very low speeds for extreme weak-signal work.

## History

Rudolf Hell patented the Hellschreiber in 1929 as a rugged, low-cost teleprinter for
landline and radio press circuits; it saw wide use through the mid-20th century, including
military and news traffic, precisely because a simple electromechanical machine could send
and print it without complex decoders.[^wiki] Amateurs revived it in the sound-card era,
where "fuzzy mode" ruggedness on HF is its main appeal.

## Deployment

Today Hellschreiber is an amateur curiosity and weak-signal mode, run on HF phone/data
sub-bands via Fldigi and similar programs. It has no commercial deployment but retains a
loyal following for QRP and marginal-path text contacts.

## Decoding it with GopherTrunk

**GopherTrunk does not decode Hellschreiber.** It is a visual facsimile mode read by eye,
with no framed data output to recover, so it sits well outside GopherTrunk's trunking and
data-decoding scope. Fldigi is the standard tool. The underlying
[on-off keying](/reference/on-off-keying/) is a primitive GopherTrunk understands, but the
Hell raster display is not something GopherTrunk produces.

## Sources

[^wiki]: [Hellschreiber](https://en.wikipedia.org/wiki/Hellschreiber) — Wikipedia, for the facsimile dot-scan principle, Feld-Hell timing, mode variants, and Rudolf Hell's invention.
