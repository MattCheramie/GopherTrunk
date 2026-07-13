---
slug: fred-gardner
title: Floyd M. Gardner
entry_type: person
category: people
description: Floyd M. Gardner was an engineer and author whose work on phase-locked loops and the Gardner timing-error detector shaped modern digital symbol-timing recovery.
keywords: Floyd Gardner, Gardner timing recovery, phaselock techniques, symbol timing, PLL, timing-error detector
aka: ["Floyd Gardner", "Floyd M. Gardner", "Fred Gardner"]
autolink: true
infobox:
  - { label: Field, value: Electrical engineering }
  - { label: Known for, value: Gardner timing detector; PLL theory }
  - { label: Author of, value: "Phaselock Techniques" }
see_also: [gardner-timing-recovery, clock-recovery, mueller-muller-timing-recovery, phase-locked-loop, john-costas, symbol-rate]
related_lessons:
  - { title: "Clock recovery & symbol timing", url: /learn/rf-sdr/clock-recovery/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Floyd_M._Gardner
  - https://onlinelibrary.wiley.com/doi/book/10.1002/0471732699
---

**Floyd M. Gardner** was an engineer and author whose work on phase-locked loops — and the
**[Gardner timing-error detector](/reference/gardner-timing-recovery/)** — underpins modern
digital **symbol-timing recovery**.[^wiki] His textbook *Phaselock Techniques* is a standard
reference used by generations of communications engineers.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 120" role="img" aria-label="A symbol waveform sampled twice per symbol at the midpoint and peak, the basis of the Gardner detector." xmlns="http://www.w3.org/2000/svg">
  <path d="M30 85 C 90 85 90 35 150 35 C 210 35 210 85 270 85 C 330 85 330 35 390 35" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <g fill="currentColor"><circle cx="90" cy="60" r="3"/><circle cx="150" cy="35" r="3"/><circle cx="210" cy="60" r="3"/><circle cx="270" cy="85" r="3"/></g>
  <text x="220" y="110" text-anchor="middle" font-size="8.5" fill="currentColor">two samples per symbol (midpoint + peak) estimate timing</text>
</svg>
<figcaption>Gardner's detector uses a midpoint and a peak sample per symbol to drive timing recovery.</figcaption>
</figure>

## Life and work

Floyd Martin Gardner was an American communications engineer and consultant who spent much of
his career working on synchronisation for satellite and digital communication systems and
teaching the subject. He is best known to the wider field as an author: his book *Phaselock
Techniques*, first published in 1966 and revised across several editions, became the
canonical practical treatment of the [phase-locked loop](/reference/phase-locked-loop/),
covering loop order, damping, acquisition, noise, and the trade-offs an engineer actually
faces when designing a real loop.[^book] For decades it was the first reference an engineer
reached for when a receiver would not lock, and it did much to turn PLL design from folklore
into disciplined practice. Alongside the textbook, Gardner published a series of influential
technical papers on timing recovery for digital receivers, the most cited of which appeared
in the *IEEE Transactions on Communications* in 1986.

## Contribution

That 1986 paper introduced what is now universally called the **Gardner timing-error
detector**. The problem it solves is fundamental to any digital receiver: the transmitter's
symbol clock is not known exactly at the receiver, so the receiver must estimate the correct
instant to sample each symbol and continuously track any drift. Gardner's detector operates on
just two samples per symbol — one near the symbol's peak and one near the transition midpoint
between symbols — and forms an error term from their product that is positive or negative
depending on whether the sampling instant is early or late. Fed to a loop filter and a
controlled interpolator, this drives the sampling phase toward the correct point. Its decisive
practical virtue, which sets it apart from earlier decision-directed schemes, is that it works
**without** knowledge of the carrier phase: timing can be recovered before, or independently
of, carrier lock. That property makes it well suited to feedforward and non-coherent
architectures and to the burst signals common in trunked radio, and it complements
carrier-recovery methods such as the [Costas loop](/reference/costas-loop/) of
[John Costas](/reference/john-costas/).

## Legacy

The Gardner detector is one of the most widely used timing-recovery methods in digital
communications, sitting alongside the [Mueller and Müller](/reference/mueller-muller-timing-recovery/)
detector as a default choice in modem and SDR design. Because it needs only two samples per
symbol and no carrier phase, it maps cleanly onto software implementations and appears in
many open-source and commercial demodulators. *Phaselock Techniques*, meanwhile, continues to
shape how loop filters and acquisition are designed across radar, instrumentation, and
communications. For GopherTrunk the connection is concrete and current: recovering the symbol
clock of a C4FM or π/4-DQPSK signal is exactly the job Gardner's detector was built for, and
Gardner-style timing recovery features in several of GopherTrunk's decoders, letting them
sample each symbol at the right instant even as the transmitter's clock drifts relative to the
receiver.

## Sources

[^wiki]: [Floyd M. Gardner](https://en.wikipedia.org/wiki/Floyd_M._Gardner) — Wikipedia, for biography and his work on phase-locked loops and timing recovery.
[^book]: [Phaselock Techniques](https://onlinelibrary.wiley.com/doi/book/10.1002/0471732699) — F. M. Gardner, Wiley (3rd ed., 2005), the standard reference on PLL design underlying the Gardner timing detector.
