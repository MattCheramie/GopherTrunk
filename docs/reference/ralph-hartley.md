---
slug: ralph-hartley
title: Ralph Hartley
entry_type: person
category: people
description: "Ralph Hartley (1888–1970) was an American engineer who invented the Hartley oscillator and gave an early quantitative measure of information that fed the Shannon-Hartley theorem."
keywords: Ralph Hartley, Hartley oscillator, Hartley information, Shannon-Hartley theorem, information theory, transform, Bell Labs
aka: [Ralph Hartley, Ralph V. L. Hartley]
autolink: true
infobox:
  - { label: Lived, value: "1888–1970" }
  - { label: Field, value: "Electronics, information theory" }
  - { label: Known for, value: "Hartley oscillator; measure of information" }
see_also: [shannon-capacity, claude-shannon, harry-nyquist, bandwidth, carrier-wave]
cite_urls:
  - https://en.wikipedia.org/wiki/Ralph_Hartley
  - https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem
---

**Ralph Hartley** (1888–1970) was an American electronics engineer who invented the
**Hartley oscillator** and, in a landmark 1928 paper, proposed one of the first
quantitative measures of information — work that fed directly into the
**[Shannon-Hartley theorem](/reference/shannon-capacity/)** and the information theory
of [Claude Shannon](/reference/claude-shannon/).[^wiki][^sh]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A tapped inductor and capacitor forming the resonant tank of a Hartley oscillator, feeding an amplifier that sustains oscillation." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rhar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <path d="M40 60 q10 -12 20 0 q10 12 20 0 q10 -12 20 0 q10 12 20 0" fill="none" stroke="currentColor"/>
  <line x1="120" y1="60" x2="120" y2="40" stroke="currentColor"/>
  <text x="80" y="45" text-anchor="middle" font-size="9" fill="currentColor">tapped L</text>
  <line x1="150" y1="52" x2="150" y2="68" stroke="currentColor"/><line x1="162" y1="52" x2="162" y2="68" stroke="currentColor"/>
  <text x="156" y="88" text-anchor="middle" font-size="9" fill="currentColor">C</text>
  <polygon points="230,45 230,75 270,60" fill="none" stroke="currentColor"/>
  <text x="248" y="95" text-anchor="middle" font-size="9" fill="currentColor">amp</text>
  <line x1="270" y1="60" x2="330" y2="60" stroke="currentColor" marker-end="url(#rhar)"/>
  <text x="380" y="64" text-anchor="middle" font-size="10" fill="currentColor">oscillation</text>
  <line x1="120" y1="60" x2="230" y2="60" stroke="currentColor"/>
</svg>
<figcaption>The Hartley oscillator uses a single tapped inductor and one capacitor as its resonant tank, feeding an amplifier that sustains the oscillation.</figcaption>
</figure>

## Life and work

Hartley was born in Nevada, studied at the University of Utah, and won a Rhodes
Scholarship to Oxford. He joined the Western Electric research arm that became Bell
Telephone Laboratories, where he spent most of his career.[^wiki] In 1915 he devised the
oscillator circuit that bears his name, used in radio receivers and transmitters for
decades. During the First World War he worked on directional radio for aircraft, and he
later contributed to the theory of amplitude and single-sideband transmission.

## Contribution

Hartley made two lasting contributions.

The first is the **Hartley oscillator**, an LC feedback oscillator whose defining feature
is a single **tapped inductor** (or two series inductors) providing the feedback,
resonating with one capacitor to set the frequency. Its counterpart, the Colpitts
oscillator, splits the capacitance instead; both remain textbook building blocks for
generating a [carrier wave](/reference/carrier-wave/).

The second, and historically deeper, is his 1928 paper *"Transmission of Information."*
Hartley argued that the amount of information a communication system can convey should be
measured by the **logarithm of the number of distinguishable messages**, giving a precise
sense in which more symbols or more time yields more information. He related this to the
[bandwidth](/reference/bandwidth/) and duration of a signal, anticipating the fundamental
trade-off between how fast and how reliably one can communicate.[^sh] The base-unit of
information he implicitly defined — the "hartley," a decimal digit's worth — is still a
named unit.

## Legacy

Hartley's logarithmic measure was the crucial precursor to Shannon's 1948 theory. When
Shannon quantified channel [capacity](/reference/shannon-capacity/), the resulting formula
for a bandwidth-limited channel with Gaussian noise — capacity equals bandwidth times the
log of one plus signal-to-noise ratio — is called the **Shannon-Hartley theorem** in
recognition that Hartley had already tied information to bandwidth. His work, alongside
that of Bell Labs colleague [Harry Nyquist](/reference/harry-nyquist/), formed the bridge
between practical telegraphy and modern information theory.

## Relevance to SDR

Both of Hartley's contributions touch software radio. Oscillators of the Hartley and
Colpitts type generate the local oscillators and reference tones that every receiver
front end needs. More fundamentally, the Shannon-Hartley relationship sets the ceiling on
how many bits per second any modulation can push through a given bandwidth at a given
signal-to-noise ratio — the yardstick against which real waveforms like those GopherTrunk
decodes are measured. GopherTrunk does not implement Hartley's oscillator (it processes
already-digitised samples), but the information-theoretic limit he helped establish frames
the whole enterprise.

## Sources

[^wiki]: [Ralph Hartley](https://en.wikipedia.org/wiki/Ralph_Hartley) — Wikipedia, for biography, the oscillator, and the 1928 paper.
[^sh]: [Shannon–Hartley theorem](https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem) — Wikipedia, for the capacity formula and Hartley's contribution.
