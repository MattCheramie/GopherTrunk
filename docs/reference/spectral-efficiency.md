---
slug: spectral-efficiency
title: Spectral efficiency
entry_type: term
category: rf-metrics
description: Spectral efficiency is the information rate a channel carries per unit bandwidth, measured in bits per second per hertz, bounded by the Shannon capacity limit.
keywords: spectral efficiency, bits per second per hertz, bandwidth efficiency, bps/Hz, Shannon capacity, modulation order, coding rate, throughput per hertz
aka: [bandwidth efficiency, bits/s/Hz]
autolink: true
infobox:
  - { label: Symbol, value: "η" }
  - { label: Unit, value: "bits/s/Hz" }
  - { label: Bound, value: "Shannon: log₂(1+SNR)" }
see_also: [shannon-capacity, quadrature-amplitude-modulation, ofdm, bandwidth, eb-n0]
cite_urls:
  - https://en.wikipedia.org/wiki/Spectral_efficiency
  - https://en.wikipedia.org/wiki/Shannon%E2%80%93Hartley_theorem
---

**Spectral efficiency** (also **bandwidth efficiency**) is the net information rate a
communication channel carries per unit of [bandwidth](/reference/bandwidth/), measured
in bits per second per hertz (bits/s/Hz).[^wiki] It answers the question every radio
engineer faces when spectrum is scarce and expensive: how many bits can you push through
a given slice of frequency? Its ceiling is set by the
[Shannon capacity](/reference/shannon-capacity/) theorem, which no coding or modulation
scheme can exceed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="A curve of Shannon spectral-efficiency limit rising with SNR in decibels, with points for BPSK, QPSK, 16-QAM and 64-QAM placed under the curve at increasing bits per second per hertz." xmlns="http://www.w3.org/2000/svg">
  <line x1="50" y1="20" x2="50" y2="160" stroke="currentColor" stroke-opacity="0.6"/>
  <line x1="50" y1="160" x2="430" y2="160" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="6" y="30" font-size="9" fill="currentColor">bits/s/Hz</text>
  <text x="360" y="178" font-size="9" fill="currentColor">SNR (dB) →</text>
  <path d="M55 155 C120 140, 200 100, 300 55 S400 25, 425 22" fill="none" stroke="currentColor" stroke-width="1.6"/>
  <text x="330" y="45" font-size="9" fill="currentColor">Shannon limit</text>
  <circle cx="120" cy="140" r="3" fill="currentColor"/><text x="126" y="138" font-size="8" fill="currentColor">BPSK ~1</text>
  <circle cx="180" cy="120" r="3" fill="currentColor"/><text x="186" y="118" font-size="8" fill="currentColor">QPSK ~2</text>
  <circle cx="250" cy="88" r="3" fill="currentColor"/><text x="256" y="86" font-size="8" fill="currentColor">16-QAM ~4</text>
  <circle cx="320" cy="52" r="3" fill="currentColor"/><text x="300" y="46" font-size="8" fill="currentColor">64-QAM ~6</text>
</svg>
<figcaption>The Shannon curve caps spectral efficiency at each SNR; real modulations sit below it, climbing in bits/s/Hz as SNR and constellation order rise.</figcaption>
</figure>

## How it works

For a single carrier, spectral efficiency is the product of two factors: how many bits
each symbol encodes, and how many symbols per second fit in the bandwidth. A modulation
of order M carries log₂(M) bits per symbol — [BPSK](/reference/bpsk/) 1,
[QPSK](/reference/qpsk/) 2, 16-[QAM](/reference/quadrature-amplitude-modulation/) 4,
64-QAM 6, 256-QAM 8 — and the [symbol rate](/reference/symbol-rate/) that fits in a
band is limited by the [Nyquist](/reference/nyquist-theorem/) criterion and the
[pulse-shaping](/reference/pulse-shaping/) roll-off. Multiply, then subtract the
overhead spent on [forward error correction](/reference/forward-error-correction/) and
framing, and you have the *net* spectral efficiency.

There is no free lunch. Packing more bits per symbol crowds constellation points closer
together, so higher-order schemes demand a higher [SNR](/reference/signal-to-noise-ratio/)
(equivalently, higher [Eb/N0](/reference/eb-n0/)) to keep the
[bit error rate](/reference/bit-error-rate/) acceptable. The
[Shannon–Hartley theorem](/reference/shannon-capacity/) pins the absolute ceiling at
η_max = log₂(1 + SNR) bits/s/Hz — you can trade power for bandwidth and vice versa, but
never cross that line.

## In practice

- **Adaptive modulation and coding** exploits the SNR trade directly: [LTE](/reference/lte/),
  [5G NR](/reference/5g-nr/), Wi-Fi, and [DVB](/reference/dvb-t/) pick a higher-order
  constellation and lighter coding when the link is strong, dropping to robust QPSK when
  it is weak — maximizing bits/s/Hz moment to moment.
- **Multicarrier and MIMO** raise system-level efficiency: [OFDM](/reference/ofdm/) packs
  orthogonal subcarriers tightly with minimal guard band, while
  [MIMO](/reference/mimo/) reuses the same bandwidth over multiple spatial streams,
  pushing effective efficiency beyond the single-channel Shannon curve.
- Narrowband land-mobile systems optimize the opposite way — for *channel* density and
  robustness rather than peak bits/s/Hz — which is why their spectral efficiencies look
  modest next to broadband data systems.

## Relevance to SDR

Spectral efficiency explains the design choices behind the systems a scanner meets.
[DMR](/reference/dmr/) and [P25 Phase 2](/reference/p25-phase-2/) use two-slot
[TDMA](/reference/tdma/) to double voice capacity in the same
12.5 kHz channel, roughly doubling spectral efficiency over a single-carrier equivalent;
[TETRA](/reference/tetra/) fits four timeslots in 25 kHz. These are all bandwidth-efficiency
decisions traded against required SNR and equipment cost.
[GopherTrunk](/reference/software-defined-radio/) does not compute a spectral-efficiency
figure — it is a design property of the protocols GT decodes, not a runtime measurement —
but understanding it clarifies why a denser mode needs a cleaner signal to lock, and why
capacity-oriented systems demand better link quality than their robust predecessors.

## Sources

[^wiki]: [Spectral efficiency](https://en.wikipedia.org/wiki/Spectral_efficiency) — Wikipedia, definition, units, and the link to Shannon capacity and modulation order.
