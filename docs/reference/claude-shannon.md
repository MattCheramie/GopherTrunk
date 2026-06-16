---
slug: claude-shannon
title: Claude Shannon
entry_type: person
category: people
description: Claude Shannon (1916–2001) was an American mathematician who founded information theory, defining channel capacity and the limits of reliable communication over noisy channels.
keywords: Claude Shannon, information theory, channel capacity, Shannon limit, entropy, Bell Labs
aka: [Claude Shannon, Shannon]
autolink: true
infobox:
  - { label: Lived, value: "1916–2001" }
  - { label: Field, value: Mathematics / engineering }
  - { label: Known for, value: Information theory }
see_also: [nyquist-theorem, harry-nyquist, signal-to-noise-ratio, forward-error-correction]
related_lessons:
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/sample-rate-nyquist/ }
external:
  - { title: "Claude Shannon (Wikipedia)", url: https://en.wikipedia.org/wiki/Claude_Shannon }
---

**Claude Shannon** (1916–2001) was an American mathematician and engineer who founded
**information theory**, defining how much information a noisy channel can carry.

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

His 1948 paper "A Mathematical Theory of Communication" introduced channel capacity,
entropy, and the limits that bound any communication system — including the role of
[SNR](/reference/signal-to-noise-ratio/).

## Contribution

Shannon's capacity theorem sets the ultimate target that
[forward error correction](/reference/forward-error-correction/) strives toward, and his
work with [Nyquist](/reference/harry-nyquist/)'s underlies the
[sampling theorem](/reference/nyquist-theorem/).

## Legacy

Information theory governs the design of every modern digital radio.
