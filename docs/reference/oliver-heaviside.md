---
slug: oliver-heaviside
title: Oliver Heaviside
entry_type: person
category: people
description: Oliver Heaviside (1850–1925) was an English engineer who recast Maxwell's equations, formulated the telegrapher's equations, and predicted the ionospheric Heaviside layer.
keywords: Oliver Heaviside, telegrapher's equations, Heaviside layer, ionosphere, vector calculus, operational calculus, impedance, Maxwell's equations
aka: [Oliver Heaviside, Heaviside]
autolink: true
infobox:
  - { label: Lived, value: "1850–1925" }
  - { label: Field, value: Electrical engineering / mathematics }
  - { label: Known for, value: "Telegrapher's equations, Heaviside layer, vector calculus" }
see_also: [james-clerk-maxwell, ionospheric-propagation, impedance, coaxial-cable]
cite_urls:
  - https://en.wikipedia.org/wiki/Oliver_Heaviside
---

**Oliver Heaviside** (1850–1925) was a self-taught English engineer and mathematician
who reformulated [Maxwell's equations](/reference/james-clerk-maxwell/) into the compact
vector form used today, derived the telegrapher's equations governing signals on
transmission lines, and predicted the conducting atmospheric layer that makes long-range
[ionospheric propagation](/reference/ionospheric-propagation/) possible.[^wiki] Working
largely alone and outside academia, he introduced ideas — [impedance](/reference/impedance/),
operational calculus, and much of modern vector notation — that remain fundamental to
radio engineering.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A radio wave from a transmitter reflecting off the Heaviside ionospheric layer to reach a distant receiver over the curved Earth." xmlns="http://www.w3.org/2000/svg">
  <path d="M20 130 Q 230 150 440 130" fill="none" stroke="currentColor" stroke-width="1.3"/>
  <line x1="20" y1="45" x2="440" y2="45" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" stroke-opacity="0.7"/>
  <text x="230" y="38" text-anchor="middle" font-size="9" fill="currentColor">Heaviside (ionospheric) layer</text>
  <g stroke="currentColor" stroke-width="1.4" fill="none"><path d="M70 128 L 230 50"/><path d="M230 50 L 390 128"/></g>
  <g fill="currentColor" font-size="9" text-anchor="middle"><circle cx="70" cy="128" r="3"/><text x="70" y="145">TX</text><circle cx="390" cy="128" r="3"/><text x="390" y="145">RX</text></g>
</svg>
<figcaption>Heaviside predicted a conducting atmospheric layer that reflects radio waves back to Earth, enabling over-the-horizon communication.</figcaption>
</figure>

## Life and work

Heaviside was born in London in 1850 into modest circumstances and had almost no formal
higher education. He worked briefly as a telegraph operator, then retired in his twenties
to study electricity and mathematics privately for the rest of his life. Partial deafness
and a solitary temperament kept him at the margins of the scientific establishment, yet
he corresponded with and influenced its leading figures.[^wiki]

His most consequential mathematical work was to take the twenty coupled equations
[James Clerk Maxwell](/reference/james-clerk-maxwell/) had written for electromagnetism
and, using the vector calculus he helped invent, compress them into the four elegant
equations engineers learn today. He also developed **operational calculus**, a symbolic
method for solving the differential equations of circuits that anticipated the Laplace
transform.

## Contribution

For radio and telecommunications, three of Heaviside's results stand out. First, the
**telegrapher's equations** describe how voltage and current propagate along a
transmission line in terms of its distributed resistance, inductance, capacitance, and
conductance — the basis for understanding [coaxial cable](/reference/coaxial-cable/),
feedlines, and signal loss. Second, his analysis of line
[impedance](/reference/impedance/) and impedance matching explained how to send signals
long distances without distortion, leading to loaded telephone lines. Third, in 1902 he
proposed (independently of Arthur Kennelly) that a conducting layer in the upper
atmosphere reflects radio waves — the **Kennelly–Heaviside layer**, later confirmed as
part of the [ionosphere](/reference/ionospheric-propagation/).[^wiki]

## Legacy

The ionospheric layer that bends shortwave signals around the curve of the Earth carried
his name for decades, and the step function used throughout signal processing is the
**Heaviside step function**. His reformulation of Maxwell's equations and his notion of
impedance are so embedded in electrical engineering that they are rarely credited to him
by name. Every long-distance HF contact that bounces off the ionosphere, and every
transmission-line calculation an SDR feedline demands, rests on work Heaviside did alone
in a rented room. He died in Devon in 1925.

## Sources

[^wiki]: [Oliver Heaviside](https://en.wikipedia.org/wiki/Oliver_Heaviside) — Wikipedia, for biography, the telegrapher's equations, vector calculus, and the Heaviside layer.
