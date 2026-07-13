---
slug: roll-off-factor
title: Roll-off factor
entry_type: term
category: modulation
description: "Roll-off factor (alpha) sets the excess bandwidth of a raised-cosine or root-raised-cosine pulse, trading occupied spectrum against filter tail length and timing robustness."
keywords: roll-off factor, alpha, excess bandwidth, raised cosine, root raised cosine, RRC, beta, Nyquist bandwidth, 0.2 alpha, 0.35 alpha
aka: [roll-off factor, excess bandwidth factor, alpha, beta]
autolink: true
infobox:
  - { label: Symbol, value: "α (or β), 0 ≤ α ≤ 1" }
  - { label: Unit, value: "Dimensionless" }
  - { label: Relation, value: "B = (1 + α) · Rs / 2" }
see_also: [root-raised-cosine-filter, pulse-shaping, intersymbol-interference, bandwidth, symbol-rate]
cite_urls:
  - https://en.wikipedia.org/wiki/Roll-off
  - https://en.wikipedia.org/wiki/Raised-cosine_filter
---

**Roll-off factor** (usually α, sometimes β) is the parameter, between 0 and 1, that sets how
gradually a raised-cosine or [root-raised-cosine](/reference/root-raised-cosine-filter/) pulse's
spectrum tapers to zero — and therefore how much **excess bandwidth** the shaped signal uses beyond
the theoretical Nyquist minimum.[^wiki] It is the main knob a system designer turns when balancing
spectral efficiency against how forgiving the link is.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Raised-cosine spectra for three roll-off factors: a near-rectangular brick-wall shape at alpha zero, a moderate taper at alpha 0.35, and a wide gentle taper at alpha 1." xmlns="http://www.w3.org/2000/svg">
  <line x1="40" y1="125" x2="440" y2="125" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="240" y1="125" x2="240" y2="30" stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 2"/>
  <text x="240" y="145" text-anchor="middle" font-size="8" fill="currentColor">center</text>
  <path d="M130 40 H350 V125 H130 Z" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="150" y="52" font-size="8" fill="currentColor">α = 0</text>
  <path d="M110 40 Q 160 40 200 55 Q 240 80 280 105 Q 320 125 370 125 L110 125 Z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-dasharray="5 3"/>
  <text x="300" y="72" font-size="8" fill="currentColor">α = 0.35</text>
  <path d="M70 40 Q 155 42 240 82 Q 325 122 410 125 L70 125 Z" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.6"/>
  <text x="360" y="112" font-size="8" fill="currentColor">α = 1</text>
</svg>
<figcaption>Small α approaches a brick-wall (minimum bandwidth); larger α widens the spectrum but with a gentler, easier-to-build skirt.</figcaption>
</figure>

## How it works

The narrowest possible zero-[ISI](/reference/intersymbol-interference/) pulse is a brick-wall filter
occupying exactly the Nyquist bandwidth R_s/2 either side of center (α = 0), but that ideal has an
infinitely long sinc-shaped tail and cannot be built. The raised-cosine family fixes this by rounding
the band edge with a cosine skirt whose width is set by α: the occupied one-sided bandwidth is
B = (1 + α)·R_s/2, so α is literally the fractional **excess bandwidth**. At α = 0 the spectrum is the
unrealizable brick wall; at α = 1 the signal occupies twice the Nyquist bandwidth but has a smooth,
easily realized roll-off and short, well-behaved pulse tails.

The choice is a genuine trade-off. **Low α** packs the signal into less spectrum and improves
[spectral efficiency](/reference/spectral-efficiency/), but the pulse tails grow longer and larger,
which makes the eye more sensitive to sampling-timing error and raises the signal's peak-to-average
ratio (harder on the transmit amplifier). **High α** wastes spectrum but yields short tails, a wide-open
eye, and relaxed timing tolerance. Because the shaping is split as a root-raised-cosine at both ends, the
transmitter and receiver must use the **same** α for the composite to remain a proper Nyquist pulse.

## Relevance to SDR

Every digitally modulated signal an SDR touches has a defined roll-off, and matching it is part of
building the receive [matched filter](/reference/matched-filter/). GopherTrunk's demodulators implement
an RRC filter whose α matches each protocol: P25 C4FM/CQPSK uses a nominal α ≈ 0.2 root-raised-cosine
shaping, and other four-level and PSK trunking modes specify their own values. Using the wrong α
mismatches the transmit and receive filters, so the composite is no longer ISI-free and the eye partly
closes even on a clean signal — a subtle way to lose sensitivity. Knowing the roll-off also lets a
receiver predict a channel's occupied [bandwidth](/reference/bandwidth/) and set its channelizer width
accordingly.

## In practice

Common values cluster low for spectrum-tight land-mobile systems (α around 0.2) and higher in systems
that prize simple filtering (older modems used α = 0.35 or more). Halving α saves only a fraction of
bandwidth but noticeably tightens the timing budget, which is why very small α is reserved for links that
can afford precise clock recovery.

## Sources

[^wiki]: [Roll-off](https://en.wikipedia.org/wiki/Roll-off) — Wikipedia, for the excess-bandwidth definition; Raised-cosine filter for the B = (1+α)Rs/2 relation and trade-offs.
