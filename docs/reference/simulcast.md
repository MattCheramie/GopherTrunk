---
slug: simulcast
title: Simulcast
entry_type: term
category: trunked-radio
description: "Simulcast is a coverage technique where several transmitters send the same signal on one frequency in tight time and frequency lock, filling in coverage but adding self-interference."
keywords: simulcast, simulcast distortion, overlapping coverage, time delay, launch time, GPS disciplined, multipath, trunking, P25 simulcast, distortion
aka: [simulcast, simulcast distortion]
autolink: true
infobox:
  - { label: Type, value: Multi-transmitter coverage scheme }
  - { label: Requires, value: Tight time + frequency sync (GPS) }
  - { label: Side effect, value: Simulcast distortion (self-multipath) }
see_also: [trunking-site, multisite-trunking, control-channel, intersymbol-interference, multipath-propagation, c4fm]
cite_urls:
  - https://en.wikipedia.org/wiki/Simulcast
---

**Simulcast** is a coverage technique in which several transmitters radiate the *same*
modulated signal on the *same* frequency at nearly the same instant, so that a wide area
is served by what looks to a radio like one big transmitter.[^wiki] The transmitters at
each [trunking site](/reference/trunking-site/) are locked to a common time and frequency
reference — typically GPS-disciplined — and fed identical bits, so their carriers add
constructively over most of the region instead of interfering as separate signals would.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="Two synchronized towers transmitting the same signal; in the overlap region a receiver sees two copies at slightly different delays, causing distortion." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="2">
    <line x1="70" y1="40" x2="70" y2="110"/><line x1="390" y1="40" x2="390" y2="110"/>
  </g>
  <path d="M70 40 L58 110 M70 40 L82 110 M390 40 L378 110 M390 40 L402 110" stroke="currentColor" stroke-width="1" fill="none"/>
  <text x="70" y="128" text-anchor="middle" font-size="9" fill="currentColor">TX A</text>
  <text x="390" y="128" text-anchor="middle" font-size="9" fill="currentColor">TX B</text>
  <circle cx="230" cy="150" r="4" fill="currentColor"/><text x="230" y="172" text-anchor="middle" font-size="9" fill="currentColor">receiver (overlap zone)</text>
  <line x1="78" y1="112" x2="222" y2="148" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3" marker-end="url(#simar)"/>
  <line x1="382" y1="112" x2="238" y2="148" stroke="currentColor" stroke-width="1" stroke-dasharray="4 3" marker-end="url(#simar)"/>
  <text x="150" y="128" text-anchor="middle" font-size="8" fill="currentColor">same bits, delay τ₁</text>
  <text x="315" y="128" text-anchor="middle" font-size="8" fill="currentColor">same bits, delay τ₂</text>
  <text x="230" y="30" text-anchor="middle" font-size="9" fill="currentColor">τ₁ ≠ τ₂ → the two copies smear the symbols</text>
  <defs><marker id="simar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>In the overlap zone a receiver hears the same waveform twice at different delays — self-inflicted multipath called simulcast distortion.</figcaption>
</figure>

## How it works

Away from the boundaries a listener is dominated by one transmitter and hears a clean
signal. In the **overlap region**, though, two or more copies arrive at slightly
different times because the path lengths differ. The delay spread between them acts
exactly like [multipath](/reference/multipath-propagation/): the symbols smear into one
another, producing [intersymbol interference](/reference/intersymbol-interference/) —
the effect operators call **simulcast distortion**. To keep this manageable, all sites must
share a common clock so the launch times differ by only microseconds, and system
planners tune per-site "launch time" delays so the overlap zones fall where few users
are.

Simulcast is a property of one wide **simulcast cell**, distinct from
[multisite trunking](/reference/multisite-trunking/), where separate sites use separate
frequencies and radios roam between them. A simulcast system can itself be one of many
sites in a larger multisite network.

## Relevance to SDR

Simulcast is one of the harder demodulation cases for a software receiver: the
composite waveform in the overlap zone has deep frequency-selective fades and a spread
delay that a simple detector cannot equalise, so decode quality can collapse even
though signal *strength* is high. [C4FM](/reference/c4fm/) and other narrowband
trunking modulations are especially sensitive because their symbol period is short
relative to the delay spread.

**GopherTrunk** decodes the [control channel](/reference/control-channel/) of simulcast
systems where the signal is clean, but in the worst overlap zones the self-interference
can defeat the demodulator — moving the antenna, or favouring one dominant site, is
often the only fix, since the distortion is baked into the received samples rather than
being something the DSP can undo. GopherTrunk treats a simulcast cell as a single site:
it does not try to separate the contributing transmitters.

## Sources

[^wiki]: [Simulcast](https://en.wikipedia.org/wiki/Simulcast) — Wikipedia, on synchronized multi-transmitter broadcasting and its overlap-zone distortion.
