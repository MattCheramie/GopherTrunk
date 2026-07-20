---
slug: modulation-and-coding-scheme
title: Modulation and coding scheme (MCS)
entry_type: concept
category: modulation
description: A modulation and coding scheme is an index that jointly selects a modulation order and a forward-error-correction code rate; link adaptation picks the highest MCS the channel's SNR can carry.
keywords: MCS, modulation and coding scheme, modulation and coding index, link adaptation, adaptive modulation and coding, AMC, CQI, LTE, 5G NR, Wi-Fi, throughput
aka: [MCS, Modulation and coding scheme, modulation and coding index]
autolink: true
infobox:
  - { label: Type, value: Link-adaptation index }
  - { label: Selects, value: "Modulation order + FEC code rate" }
  - { label: Driven by, value: "Measured SNR / CQI" }
  - { label: Used by, value: "LTE, 5G NR, Wi-Fi" }
see_also: [quadrature-amplitude-modulation, forward-error-correction, signal-to-noise-ratio, spectral-efficiency, lte, wifi-80211, shannon-capacity]
cite_urls:
  - https://en.wikipedia.org/wiki/Link_adaptation
  - https://en.wikipedia.org/wiki/Adaptive_modulation_and_coding
---

A **modulation and coding scheme** (**MCS**) is an index that *jointly* selects a
modulation order — [QPSK](/reference/qpsk/), 16-, 64-, or 256-level
[QAM](/reference/quadrature-amplitude-modulation/) — and a
[forward-error-correction](/reference/forward-error-correction/) code rate.[^link] One MCS
number therefore pins down both how many bits ride on each symbol and how much redundancy
protects them, so a transmitter can trade robustness for throughput with a single choice.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 200" role="img" aria-label="A ladder in which rising SNR selects a higher MCS index, mapping to a higher modulation order and code rate and thus higher throughput." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2"><line x1="40" y1="175" x2="40" y2="20" marker-end="url(#rel_mcs)"/><line x1="40" y1="175" x2="440" y2="175" marker-end="url(#rel_mcs)"/></g>
  <text x="18" y="100" font-size="9" fill="currentColor" transform="rotate(-90 18 100)" text-anchor="middle">throughput →</text>
  <text x="240" y="195" font-size="9" fill="currentColor" text-anchor="middle">measured SNR / CQI →</text>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="60" y="140" width="96" height="26" rx="4" fill="currentColor" fill-opacity="0.10" stroke="currentColor" stroke-width="1.1"/><text x="108" y="151">MCS low</text><text x="108" y="161" font-size="7">QPSK, rate 1/3</text>
    <rect x="182" y="104" width="96" height="26" rx="4" fill="currentColor" fill-opacity="0.13" stroke="currentColor" stroke-width="1.1"/><text x="230" y="115">MCS mid</text><text x="230" y="125" font-size="7">16QAM, rate 1/2</text>
    <rect x="304" y="68" width="96" height="26" rx="4" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.1"/><text x="352" y="79">MCS high</text><text x="352" y="89" font-size="7">64QAM, rate 3/4</text>
    <rect x="340" y="30" width="96" height="26" rx="4" fill="currentColor" fill-opacity="0.20" stroke="currentColor" stroke-width="1.1"/><text x="388" y="41">MCS top</text><text x="388" y="51" font-size="7">256QAM, rate 5/6</text>
  </g>
</svg>
<figcaption>Link adaptation climbs the ladder: the better the measured channel, the higher the MCS index — a higher modulation order and lighter coding — and the more bits per second the link carries.</figcaption>
</figure>

## How it works

The point of an MCS table is **link adaptation**, also called **adaptive modulation and
coding (AMC)**.[^amc] The receiver measures channel quality — a
[signal-to-noise ratio](/reference/signal-to-noise-ratio/) estimate, reported in cellular
systems as a Channel Quality Indicator (CQI) — and the transmitter selects the *highest* MCS
index that the channel can still decode reliably. When the link is strong, a high MCS packs
many bits per symbol with light coding, maximising
[spectral efficiency](/reference/spectral-efficiency/). As the channel degrades — the user
moves away, fades, or picks up interference — the scheduler steps down to a lower MCS: a
smaller modulation order and a stronger code, fewer bits per symbol but a lower error rate.
The system is, in effect, chasing the [Shannon capacity](/reference/shannon-capacity/) of the
instantaneous channel, holding the error rate roughly constant while throughput floats up and
down with conditions.

Each MCS index is a fixed row in a standard's table, so both ends agree on exactly what
modulation and code rate an index means. Higher index means more bits per symbol but demands
higher SNR to decode; there is no free lunch — the ladder simply lets the link ride as high as
the moment allows.

## In practice

MCS tables are central to modern wireless. **[LTE](/reference/lte/)** defines an MCS index
(0–28) that the eNodeB assigns per user per subframe based on reported CQI. **5G NR** extends
the idea with multiple MCS tables, including a 256QAM table for strong links and a
low-spectral-efficiency table for coverage edges. **[Wi-Fi](/reference/wifi-80211/)** (802.11n
onward) has its own MCS index spanning BPSK through 256QAM/1024QAM and several code rates, with
the rate-control algorithm picking an index per frame. In every case the same principle holds:
one index, jointly chosen, tracking the channel.

## Relevance to SDR

For an SDR observer, the MCS carried in a system's control signalling is a live readout of link
quality — a high index means the base station judges the channel excellent, a low index means it
is fighting fades or range. Decoding those control fields reveals how a network is adapting
without ever touching the payload. GopherTrunk's land-mobile trunking targets (P25, DMR, NXDN,
TETRA) use fixed modulation rather than a per-burst MCS ladder, so MCS is documented here as the
adaptive-link concept an SDR user meets in cellular and Wi-Fi monitoring.

## Sources

[^link]: [Link adaptation](https://en.wikipedia.org/wiki/Link_adaptation) — Wikipedia, for the joint selection of modulation and code rate by an MCS index and its role in matching the channel.
[^amc]: [Adaptive modulation and coding](https://en.wikipedia.org/wiki/Adaptive_modulation_and_coding) — Wikipedia, for AMC, CQI feedback, and the MCS tables used in LTE, 5G NR, and Wi-Fi.
