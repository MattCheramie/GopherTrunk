---
slug: oliver-lodge
title: Oliver Lodge
entry_type: person
category: people
description: "Oliver Lodge (1851–1940) was a British physicist and radio pioneer who demonstrated Hertzian waves and patented tuning (syntony) for wireless."
keywords: Oliver Lodge, Oliver Joseph Lodge, syntony, tuning, wireless telegraphy, coherer, Hertzian waves, radio pioneer, resonance
aka: [Oliver Lodge, Oliver Joseph Lodge, Sir Oliver Lodge]
autolink: true
infobox:
  - { label: Lived, value: "1851–1940" }
  - { label: Field, value: Physics }
  - { label: Known for, value: "Syntonic tuning; early radio" }
see_also: [resonance, radio-wave, heinrich-hertz, guglielmo-marconi, q-factor]
cite_urls:
  - https://en.wikipedia.org/wiki/Oliver_Lodge
---

**Oliver Lodge** (1851–1940) was a British physicist and one of the earliest
demonstrators of wireless, best remembered for inventing **syntony** — the tuning of a
transmitter and receiver to the same frequency so they respond to each other and reject
interference.[^wiki] His work turned the crude spark-and-spark experiments of the 1890s
into something that could carry [radio waves](/reference/radio-wave/) selectively, a
principle that survives in every tuned circuit today.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A transmitter and receiver each with a tuned resonant circuit set to the same frequency, so the receiver responds strongly only to that frequency, illustrating syntonic tuning." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="olar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="25" y="45" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <text x="65" y="62">TX</text><text x="65" y="76" font-size="8">tuned f₀</text>
    <rect x="355" y="45" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/>
    <text x="395" y="62">RX</text><text x="395" y="76" font-size="8">tuned f₀</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none" opacity="0.8">
    <path d="M120 65 q15 -12 30 0 q15 12 30 0 q15 -12 30 0 q15 12 30 0 q15 -12 30 0 q15 12 30 0" marker-end="url(#olar)"/>
  </g>
  <text x="230" y="42" font-size="9" fill="currentColor" text-anchor="middle">only f₀ resonates</text>
</svg>
<figcaption>Syntonic tuning: matching the resonant frequency of the receiver to the transmitter lets it pick out one signal and ignore others.</figcaption>
</figure>

## Life and work

Lodge was a professor of physics at University College Liverpool and later the first
principal of the University of Birmingham. In 1894, shortly after the death of
[Heinrich Hertz](/reference/heinrich-hertz/), Lodge gave a widely noted lecture and
demonstration in which he detected Hertzian waves across a lecture hall using a
**coherer** — a glass tube of loose metal filings that changes resistance in the presence
of a radio-frequency spark. His public demonstrations predated
[Guglielmo Marconi](/reference/guglielmo-marconi/)'s commercial systems by a year or
more.[^wiki]

Lodge was also a prominent figure in Victorian science more broadly, active in the study
of lightning protection, and — controversially — in spiritualism late in life.

## Contribution

Lodge's central contribution to radio was **syntony**, patented in 1897. Early spark
transmitters radiated across a broad band of frequencies, so any receiver picked up every
transmitter at once. By adding a tuned resonant circuit — an inductor and capacitor sized
to a chosen [resonance](/reference/resonance/) — to both ends of the link, Lodge made a
receiver respond strongly only to its matched transmitter. This is exactly the sharpness
now measured by a circuit's [Q factor](/reference/q-factor/), and it is what makes
multiple stations able to share the spectrum.

His syntonic patents were valuable enough that the Marconi Company eventually bought
them, acknowledging tuning as essential to practical wireless.

## Legacy

Every radio that selects one station from many does so by Lodge's principle. The tuned
front end of a receiver, the resonant matching network of an antenna, and the channel
filters inside a [software-defined radio](/reference/software-defined-radio/) are all
descendants of syntony. Lodge is remembered as the physicist who made wireless
*selective*, turning a curiosity into a communication medium.

## Sources

[^wiki]: [Oliver Lodge](https://en.wikipedia.org/wiki/Oliver_Lodge) — Wikipedia, for his biography, the 1894 Hertzian-wave demonstration, and the syntonic tuning patents.
