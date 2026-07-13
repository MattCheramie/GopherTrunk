---
slug: bandwidth
title: Bandwidth
entry_type: term
category: rf-fundamentals
description: Bandwidth is the width of the frequency range a signal occupies or a receiver captures, measured in hertz; it bounds data rate and sets capture requirements.
keywords: bandwidth, Hz, channel width, occupied bandwidth, capture bandwidth, Shannon, channel capacity
infobox:
  - { label: Symbol, value: B }
  - { label: Unit, value: Hertz (Hz) }
  - { label: Determines, value: Data capacity, capture needs }
see_also: [sample-rate, nyquist-theorem, frequency, signal-to-noise-ratio, occupied-bandwidth, shannon-capacity]
related_lessons:
  - { title: "Anatomy of a signal", url: /learn/rf-sdr/signal-anatomy/ }
  - { title: "Sample rate, bandwidth & Nyquist", url: /learn/rf-sdr/sample-rate-nyquist/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Bandwidth_(signal_processing)
  - https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem
---

**Bandwidth** is the width, in hertz, of the [frequency](/reference/frequency/) range a
signal occupies or that a receiver captures.[^wiki] A narrowband voice channel may be
~12.5 kHz wide; an FM broadcast station ~200 kHz; a Wi-Fi channel tens of megahertz.
Bandwidth is one of the most consequential numbers in radio: it caps how fast information
can flow, it decides how much spectrum a signal consumes, and — on the receive side — it
sets how much of the spectrum an SDR must digitise at once.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A spectrum bump occupying a span of frequency, with the width of the occupied span labelled bandwidth and frequency increasing to the right." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="100" x2="440" y2="100" stroke="currentColor" stroke-opacity="0.4"/>
  <path d="M150 100 C 190 100, 195 35, 230 35 C 265 35, 270 100, 310 100" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.6"/>
  <line x1="150" y1="55" x2="310" y2="55" stroke="currentColor" marker-start="url(#bws)" marker-end="url(#bwe)"/>
  <text x="230" y="48" text-anchor="middle" font-size="11" fill="currentColor">bandwidth</text>
  <text x="230" y="118" text-anchor="middle" font-size="10" fill="currentColor">frequency →</text>
  <defs>
    <marker id="bws" markerWidth="8" markerHeight="8" refX="2" refY="3" orient="auto"><path d="M6 0 L0 3 L6 6 z" fill="currentColor"/></marker>
    <marker id="bwe" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker>
  </defs>
</svg>
<figcaption>Bandwidth is the span of frequency a signal occupies — wider signals carry more data but use more spectrum.</figcaption>
</figure>

## How it works

Bandwidth arises because information cannot be sent instantaneously. A steady, unchanging
[carrier](/reference/carrier-wave/) is a single spectral line of zero width;
[modulating](/reference/modulation/) it to carry a message spreads its energy into
sidebands, and the faster the modulation, the wider those sidebands. The exact width a
signal genuinely uses is its [occupied bandwidth](/reference/occupied-bandwidth/), usually
defined as the span containing 99% of the transmitted power. Regulators allocate each
service a channel of a given width and demand that emissions stay within it, leaving a
[guard band](/reference/guard-band/) of empty spectrum between channels to prevent
interference.

There is a hard link between bandwidth and how much data a channel can carry. The
Shannon–Hartley theorem gives the theoretical ceiling as
*C = B · log₂(1 + SNR)*, where *C* is capacity in bits per second, *B* is bandwidth, and
SNR is the [signal-to-noise ratio](/reference/signal-to-noise-ratio/) — see
[Shannon capacity](/reference/shannon-capacity/). Capacity scales linearly with
bandwidth but only logarithmically with SNR, which is why more bandwidth is the most
direct way to raise data rate, and why crowded spectrum is such a valuable resource.
Bandwidth cuts the other way too: a wider receiver passband admits more noise
(noise power is proportional to bandwidth), so opening the filter beyond what the signal
needs only lowers SNR.

## In practice

- **Channel width sets the standard.** Land-mobile digital voice fits inside narrow
  channels — DMR packs two TDMA slots into 12.5 kHz, P25 Phase 1 uses 12.5 kHz, NXDN
  offers a 6.25 kHz mode — driving the choice of modulation and symbol rate.
- **Match the filter to the signal.** A receiver's channel filter should be about as wide
  as the [occupied bandwidth](/reference/occupied-bandwidth/): wide enough to pass the
  signal undistorted, narrow enough to reject neighbours and noise.
- **Capture versus channel.** An SDR's capture (or instantaneous) bandwidth is far wider
  than any one channel, letting it see a whole trunked system's control and voice
  channels simultaneously.

## Relevance to SDR

An SDR's capture bandwidth is roughly its [sample rate](/reference/sample-rate/): by the
[Nyquist theorem](/reference/nyquist-theorem/) a complex (IQ) stream sampled at *fs*
represents a band *fs* wide. To monitor a trunking system, GopherTrunk captures a wide
slice — several megahertz — then uses
[filtering and decimation](/reference/decimation/), and a
[digital down-converter](/reference/digital-down-converter/), to carve out each narrow
channel at its own centre frequency and reduce it to the per-channel rate the decoder
needs (48 kHz for the 4800-baud C4FM family). The wide capture is what lets one receiver
follow a control channel and its voice grants at once; the narrow per-channel bandwidth
is what keeps noise out of each individual decode.

## Sources

[^wiki]: [Bandwidth (signal processing)](https://en.wikipedia.org/wiki/Bandwidth_(signal_processing)) — Wikipedia, definition and measures of signal bandwidth.
[^shannon]: [Shannon–Hartley theorem](https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem) — Wikipedia, on the channel-capacity limit set by bandwidth and signal-to-noise ratio.
