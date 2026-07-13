---
slug: polar-code
title: Polar code
entry_type: algorithm
category: error-correction
description: Polar codes exploit channel polarization to split a channel into reliable and useless bit-channels, freezing the bad ones and decoding by successive cancellation; used for 5G NR control channels.
keywords: polar code, channel polarization, frozen bits, successive cancellation, SC decoding, SC list decoding, CRC-aided, Arikan, 5G NR control channel, capacity-achieving
aka: [polar code, Arikan code]
autolink: true
infobox:
  - { label: Type, value: Capacity-achieving block code }
  - { label: Key idea, value: Channel polarization + frozen bits }
  - { label: Used by, value: 5G NR control channels }
see_also: [forward-error-correction, ldpc-code, turbo-code, reed-muller-code, convolutional-code, viterbi-algorithm]
cite_urls:
  - https://en.wikipedia.org/wiki/Polar_code_(coding_theory)
  - https://ieeexplore.ieee.org/document/5075875
---

A **polar code** is a linear block code that, as the block length grows, provably **achieves
channel capacity** using a strikingly simple construction: combine many copies of a channel so
they *polarize* into some that become almost perfectly reliable and some that become almost
useless, then send data only on the reliable ones.[^wiki] Introduced by Erdal Arıkan in 2009,
polar codes were the first codes with a rigorous proof of capacity-achievement and a
low-complexity decoder, and they were adopted for the **control channels** of 5G NR.[^arikan]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="Combining channels through butterfly transforms polarizes them into a set of highly reliable bit-channels that carry data and a set of unreliable bit-channels that are frozen to fixed values." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="pcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <text x="70" y="18" text-anchor="middle" font-size="9" fill="currentColor">identical channels</text>
  <text x="390" y="18" text-anchor="middle" font-size="9" fill="currentColor">polarized bit-channels</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="40" y1="40" x2="150" y2="40"/><line x1="40" y1="70" x2="150" y2="70"/><line x1="40" y1="100" x2="150" y2="100"/><line x1="40" y1="130" x2="150" y2="130"/>
    <line x1="150" y1="40" x2="150" y2="70"/><line x1="150" y1="100" x2="150" y2="130"/>
    <line x1="150" y1="55" x2="250" y2="55"/><line x1="150" y1="115" x2="250" y2="115"/>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="300" y1="35" x2="360" y2="35" marker-end="url(#pcar)"/>
    <line x1="300" y1="65" x2="360" y2="65" marker-end="url(#pcar)"/>
    <line x1="300" y1="100" x2="360" y2="100" marker-end="url(#pcar)"/>
    <line x1="300" y1="130" x2="360" y2="130" marker-end="url(#pcar)"/>
  </g>
  <g font-size="8" fill="currentColor">
    <text x="368" y="38">reliable → data</text>
    <text x="368" y="68">reliable → data</text>
    <text x="368" y="103">weak → frozen (0)</text>
    <text x="368" y="133">weak → frozen (0)</text>
  </g>
</svg>
<figcaption>Recursive combining polarizes identical channels: data rides the bit-channels that became reliable, while the weak ones are frozen to known values.</figcaption>
</figure>

## How it works

Take two copies of a channel and combine their inputs with a single XOR before transmission.
Decoded jointly, one of the two resulting *bit-channels* comes out **better** than the
original and the other **worse**. Apply this butterfly recursively across N = 2ⁿ copies and
the effect compounds: the bit-channels split into two extremes — a fraction essentially
noiseless and the rest essentially useless — with almost nothing in between. That is **channel
polarization**.

The code then writes itself:

- Rank the N synthetic bit-channels by reliability (computed for the target channel and SNR).
- Put the K information bits on the K most reliable bit-channels.
- **Freeze** the remaining N−K bit-channels to fixed, known values (usually 0). Encoder and
  decoder agree on which indices are frozen.

Decoding is **successive cancellation (SC)**: process bit-channels in order, and for each one
either read off the frozen value (known in advance) or make a soft decision using the channel
observations *and* all previously decided bits. Because each decision leans on the earlier
ones, SC is sequential, with complexity O(N log N).

## Variants

Plain SC only approaches capacity at very long block lengths and makes irreversible early
mistakes. Two refinements fixed that for practical short blocks:

- **SC List (SCL)** keeps the L most likely partial-decode paths instead of committing to one,
  choosing the best survivor at the end — much stronger at a modest cost.
- **CRC-aided SCL** appends a [CRC](/reference/cyclic-redundancy-check/) to the data; the
  decoder picks the list path whose CRC checks out, which sharply lowers the error rate and is
  the form standardized in 5G.

Polar codes are also closely related to [Reed–Muller codes](/reference/reed-muller-code/),
which use the same recursive transform but choose the information positions differently.

## Relevance to SDR

Polar codes protect the 5G NR control channels (PBCH broadcast, and the PDCCH/PUCCH
downlink/uplink control), where messages are short and reliability is paramount; the data
channels use [LDPC](/reference/ldpc-code/) instead, and earlier 3G/4G used
[turbo codes](/reference/turbo-code/). They are a landmark in
[forward error correction](/reference/forward-error-correction/) as the first provably
capacity-achieving construction. The land-mobile and aviation formats GopherTrunk decodes do
not use polar coding, so GT does not implement a polar decoder; it is documented here as a
cornerstone of modern cellular coding theory.

## Sources

[^wiki]: [Polar code (coding theory)](https://en.wikipedia.org/wiki/Polar_code_(coding_theory)) — Wikipedia, for channel polarization, frozen bits, and successive-cancellation decoding.
[^arikan]: [Channel polarization: a method for constructing capacity-achieving codes](https://ieeexplore.ieee.org/document/5075875) — E. Arıkan, IEEE Trans. Information Theory (2009), the founding paper.
