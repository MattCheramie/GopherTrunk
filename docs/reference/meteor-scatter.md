---
slug: meteor-scatter
title: Meteor scatter
entry_type: term
category: propagation
description: Meteor scatter is a VHF mode that reflects signals off the brief ionized trails left by meteors, giving fleeting long-distance bursts used by digital modes like FT8/MSK144.
keywords: meteor scatter, meteor burst, MSK144, meteor trail, ionized trail, VHF propagation, ping, underdense trail, overdense trail, meteor shower
aka: [meteor scatter, meteor burst, MS]
autolink: true
infobox:
  - { label: Type, value: Burst VHF scatter mode }
  - { label: Mechanism, value: Reflection off ionized meteor trails }
  - { label: Best band, value: Low VHF (~30–150 MHz) }
see_also: [sporadic-e, sky-wave, radio-propagation, ft8, doppler-shift]
cite_urls:
  - https://en.wikipedia.org/wiki/Meteor_burst_communications
  - https://en.wikipedia.org/wiki/Meteor_scatter
---

**Meteor scatter** (meteor-burst communication) is a
[VHF](/reference/frequency-bands/) propagation mode that reflects
[radio waves](/reference/radio-wave/) off the short-lived columns of ionised gas that
meteors leave as they burn up in the upper atmosphere.[^wiki] Each trail supports a
usable path for only a fraction of a second to a few seconds, so meteor scatter delivers
long-distance contact in brief "pings" and "bursts" rather than a steady signal.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A transmitter aiming a VHF ray at a diagonal glowing meteor trail high in the atmosphere, which briefly reflects the signal down to a distant receiver hundreds of kilometres away." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="140" x2="440" y2="140" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="180" y1="42" x2="250" y2="62" stroke="currentColor" stroke-width="3" stroke-opacity="0.55"/><text x="188" y="36" font-size="8" fill="currentColor">meteor trail (~85–110 km)</text>
  <line x1="55" y1="138" x2="55" y2="120" stroke="currentColor" stroke-width="2"/><text x="55" y="153" text-anchor="middle" font-size="8" fill="currentColor">TX</text>
  <line x1="405" y1="138" x2="405" y2="120" stroke="currentColor" stroke-width="2"/><text x="405" y="153" text-anchor="middle" font-size="8" fill="currentColor">RX</text>
  <path d="M55 120 L210 52 L405 120" fill="none" stroke="currentColor" stroke-width="1.5" marker-end="url(#msar)"/>
  <text x="150" y="120" font-size="8" fill="currentColor" fill-opacity="0.7">reflection lasts a fraction of a second</text>
  <defs><marker id="msar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A meteor's ionised trail momentarily forms a reflector at E-layer height, bouncing a VHF signal to a station hundreds of kilometres away before it dissipates.</figcaption>
</figure>

## How it works

Billions of tiny meteoroids strike the atmosphere every day, most no larger than a
grain of sand. As one ablates around 85–110 km up, it ionises a long, thin column of
air. That column of free electrons can reflect or scatter VHF signals for as long as it
persists before winds and recombination disperse it. Trails fall into two classes:

- **Underdense trails** — the common, faint ones. They act like a cloud of independent
  scatterers; the reflection rises and decays exponentially within a few tenths of a
  second (a "ping").
- **Overdense trails** — from larger meteoroids. They are dense enough to reflect like a
  metallic column and can sustain a usable path for several seconds, sometimes with
  strong signal and audible [Doppler](/reference/doppler-shift/) as the trail drifts.

Because any one trail is fleeting, meteor-scatter systems transmit in short, repeated,
high-rate bursts and wait for a trail to bridge the path. The reflecting region sits at
E-layer height, giving typical single-hop ranges of roughly 1,000–2,000 km. Activity
follows the meteor flux: a baseline of sporadic meteors runs all the time, peaking
before local dawn, and rises sharply during annual meteor showers.

## In practice

Historically, "meteor-burst" data networks (such as remote environmental telemetry)
exploited the mode for low-rate, store-and-forward messaging over long distances without
satellites. Today it is best known in amateur radio through digital modes purpose-built
for it — [FT8](/reference/ft8/)'s fast cousin MSK144 packs a complete message into a
short, repeated frame so a single ping can carry a whole exchange. Its ion supply
overlaps with [sporadic E](/reference/sporadic-e/), and the two often coexist on the low
VHF bands.

## Relevance to SDR

Meteor scatter lives squarely in VHF, so it is accessible to ordinary SDR hardware — an
[RTL-SDR](/reference/rtl-sdr/) or [Airspy](/reference/airspy/) can capture the pings,
and the decoding is done in software. For a trunking scanner like **GopherTrunk** the
mode is neither a target nor a normal interference source; it is an illustration of how
even transient, exotic reflectors briefly extend VHF reach far past the
[radio horizon](/reference/radio-horizon/) that otherwise bounds a scanner's coverage.
GopherTrunk models none of this and simply decodes whatever the front end receives.

## Sources

[^wiki]: [Meteor burst communications](https://en.wikipedia.org/wiki/Meteor_burst_communications) — Wikipedia, on ionised meteor trails, underdense vs overdense reflection, burst timing, and VHF ranges.
