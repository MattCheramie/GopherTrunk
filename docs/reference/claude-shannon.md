---
slug: claude-shannon
title: Claude Shannon
entry_type: person
category: people
description: Claude Shannon (1916–2001) was an American mathematician who founded information theory, defining channel capacity and the limits of reliable communication over noisy channels.
keywords: Claude Shannon, information theory, channel capacity, Shannon limit, entropy, Bell Labs, bit, sampling theorem
aka: [Claude Shannon, Shannon]
autolink: true
infobox:
  - { label: Lived, value: "1916–2001" }
  - { label: Field, value: Mathematics / engineering }
  - { label: Known for, value: Information theory }
see_also: [nyquist-theorem, harry-nyquist, ralph-hartley, robert-gallager, signal-to-noise-ratio, forward-error-correction, shannon-capacity]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Claude_Shannon
  - https://web.archive.org/web/19980715013250/http://cm.bell-labs.com/cm/ms/what/shannonday/shannon1948.pdf
---

**Claude Shannon** (1916–2001) was an American mathematician and engineer who founded
**information theory**, defining how much information a noisy channel can carry and setting
the ultimate limit that every communication system, radio included, works against.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="An information source passing through a noisy channel to a destination, with the channel-capacity formula." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="30" y="40" width="60" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="59">source</text>
    <rect x="150" y="40" width="80" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="190" y="53">noisy</text><text x="190" y="64" font-size="8">channel</text>
    <rect x="290" y="40" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="330" y="59">destination</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="90" y1="55" x2="149" y2="55" marker-end="url(#shar)"/><line x1="230" y1="55" x2="289" y2="55" marker-end="url(#shar)"/></g>
  </g>
  <text x="230" y="98" text-anchor="middle" font-size="10" fill="currentColor">C = B · log₂(1 + SNR)</text>
  <defs><marker id="shar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Shannon founded information theory, defining the maximum error-free data rate of a noisy channel.</figcaption>
</figure>

## Life and work

Claude Elwood Shannon was born in 1916 in Petoskey, Michigan, and grew up in nearby
Gaylord, an inveterate tinkerer who built model planes and a telegraph line to a friend's
house. He took degrees in both electrical engineering and mathematics at the University of
Michigan, then went to MIT, where his 1937 master's thesis — often called the most
important master's thesis of the century — showed that the algebra of logic developed by
George Boole could describe and simplify relay switching circuits. That single insight is
the theoretical foundation of all digital hardware. After a doctorate applying mathematics
to genetics, he joined Bell Labs in 1941, working during the war on fire-control systems and
on cryptography, where he crossed paths with Alan Turing and proved the theoretical
unbreakability of the one-time pad.

His playful, eclectic mind never stopped: he juggled while riding a unicycle down the halls
of Bell Labs, built a mechanical mouse that could solve a maze, designed an early
chess-playing analysis, and constructed a machine whose only function was to switch itself
off. Behind the whimsy was the most consequential body of work in twentieth-century
communications.

## Contribution

In 1948 Shannon published *A Mathematical Theory of Communication*, and the field of
information theory sprang into existence essentially complete.[^paper] He proposed measuring
information in **bits** and quantified the uncertainty of a source by its entropy. His
central and most startling result — the noisy-channel coding theorem — established that
every channel has a definite **capacity**, and that as long as one transmits below that
capacity, error-correcting codes exist that make the probability of error as small as
desired, no matter how noisy the channel. For the common case of a bandlimited channel with
Gaussian noise, capacity is *C = B · log₂(1 + SNR)*: raise the bandwidth or the
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) and more error-free bits per
second become possible. This drew a hard line — the [Shannon capacity](/reference/shannon-capacity/)
or "Shannon limit" — that no coding scheme can cross, and it turned reliable communication
from an art into a target with a known optimum. Shannon also stated the sampling theorem in
the general form now taught, building explicitly on the earlier signalling analysis of
[Harry Nyquist](/reference/harry-nyquist/) and [Ralph Hartley](/reference/ralph-hartley/).

## Legacy

Shannon's capacity theorem promised that near-optimal codes existed but did not say how to
build them, and much of the next half-century of coding theory was a quest to reach the
limit he had drawn. [Forward error correction](/reference/forward-error-correction/)
schemes — convolutional codes, then the turbo and LDPC codes that
[Robert Gallager](/reference/robert-gallager/) helped originate — now approach the Shannon
limit within a fraction of a decibel, which is why modern 5G and Wi-Fi extract so much
throughput from so little signal. For a scanner like GopherTrunk the theory is the
explanation of what it sees: whether a P25 or DMR frame can be recovered at a given
[SNR](/reference/signal-to-noise-ratio/) is, at bottom, a question about operating below the
channel's Shannon capacity, and the FEC in those protocols is engineering built to squeeze
toward the boundary Shannon defined. Information theory now governs the design of every
digital radio, storage medium, and data link on Earth.

## Sources

[^wiki]: [Claude Shannon](https://en.wikipedia.org/wiki/Claude_Shannon) — Wikipedia, for biography and his founding of information theory.
[^paper]: [A Mathematical Theory of Communication](https://web.archive.org/web/19980715013250/http://cm.bell-labs.com/cm/ms/what/shannonday/shannon1948.pdf) — C. E. Shannon, Bell System Technical Journal (1948), the founding paper defining entropy and channel capacity.
