---
slug: robert-gallager
title: Robert Gallager
entry_type: person
category: people
description: "Robert Gallager (b. 1931) is an American information theorist who invented low-density parity-check (LDPC) codes and shaped modern coding theory."
keywords: Robert Gallager, LDPC codes, low-density parity-check, information theory, coding theory, error-exponent, data networks
aka: [Robert Gallager, Robert G. Gallager]
autolink: true
infobox:
  - { label: Lived, value: "b. 1931" }
  - { label: Field, value: "Information theory" }
  - { label: Known for, value: "LDPC codes; information theory" }
see_also: [ldpc-code, forward-error-correction, claude-shannon, turbo-code, shannon-capacity]
cite_urls:
  - https://en.wikipedia.org/wiki/Robert_G._Gallager
  - https://en.wikipedia.org/wiki/Low-density_parity-check_code
---

**Robert Gallager** (born 1931) is an American electrical engineer and information
theorist who invented **[low-density parity-check (LDPC) codes](/reference/ldpc-code/)**
and made foundational contributions to the theory of reliable communication.[^wiki][^ldpc]
His LDPC codes, introduced in his 1960 doctoral thesis, are among the most powerful
[forward error correction](/reference/forward-error-correction/) schemes known and are
used across modern wireless and storage systems.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A sparse bipartite Tanner graph connecting variable nodes to check nodes, the structure of a low-density parity-check code." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor"><circle cx="60" cy="40" r="5"/><circle cx="140" cy="40" r="5"/><circle cx="220" cy="40" r="5"/><circle cx="300" cy="40" r="5"/><circle cx="380" cy="40" r="5"/></g>
  <g fill="none" stroke="currentColor"><rect x="110" y="115" width="14" height="14"/><rect x="230" y="115" width="14" height="14"/><rect x="330" y="115" width="14" height="14"/></g>
  <g stroke="currentColor" stroke-opacity="0.5">
    <line x1="60" y1="45" x2="117" y2="115"/><line x1="140" y1="45" x2="117" y2="115"/><line x1="220" y1="45" x2="117" y2="115"/>
    <line x1="140" y1="45" x2="237" y2="115"/><line x1="300" y1="45" x2="237" y2="115"/><line x1="380" y1="45" x2="237" y2="115"/>
    <line x1="220" y1="45" x2="337" y2="115"/><line x1="300" y1="45" x2="337" y2="115"/><line x1="60" y1="45" x2="337" y2="115"/>
  </g>
  <text x="20" y="44" font-size="9" fill="currentColor">bits</text>
  <text x="10" y="127" font-size="9" fill="currentColor">checks</text>
</svg>
<figcaption>An LDPC code is defined by a sparse parity-check matrix, drawn here as a Tanner graph linking code bits to a few parity checks each.</figcaption>
</figure>

## Life and work

Gallager earned his doctorate from MIT in 1960 and spent his career there, becoming a
professor of electrical engineering and a leading figure in the information-theory
community. His thesis introduced LDPC codes together with the iterative decoding idea
that makes them work; the codes were so far ahead of the hardware of the day that they
were largely set aside for three decades before being rediscovered in the 1990s.[^wiki]
He also wrote influential texts on information theory and on data networks, and won the
Claude E. Shannon Award and the U.S. National Medal of Science.

## Contribution

An **LDPC code** is a linear block code defined by a **parity-check matrix that is
sparse** — each parity equation involves only a handful of the code's bits, and each bit
participates in only a few equations. That sparsity is the whole trick. It lets the code
be decoded by **iterative message passing** (belief propagation) over the bipartite
"Tanner graph" of bits and checks: each check node and bit node repeatedly exchanges its
best estimate of the bit values, and the estimates converge toward a valid codeword after
a few rounds.[^ldpc] Because the graph is sparse, each iteration is cheap, so very long,
very strong codes become decodable in practice.

Gallager also contributed the theory of **error exponents**, quantifying how the
probability of a decoding error falls off as block length grows for a channel operated
below its [Shannon capacity](/reference/shannon-capacity/) — a sharpening of
[Claude Shannon](/reference/claude-shannon/)'s existence results into rate-versus-reliability
trade-offs.

## Legacy

For decades LDPC codes were a mathematical curiosity, too expensive to decode with the
electronics of the 1960s. Their rediscovery — spurred by the arrival of
[turbo codes](/reference/turbo-code/) and cheap iterative-decoding hardware — made them
the error-correction method of choice for systems that must operate close to the Shannon
limit. Gallager's iterative-decoding idea, once exotic, is now standard engineering.

## Relevance to SDR

LDPC codes are everywhere in modern digital radio: Wi-Fi (802.11n/ac/ax), 5G NR data
channels, DVB-S2 and DVB-T2 broadcasting, 10-Gigabit Ethernet, and flash-memory
controllers all rely on them to squeeze reliable throughput out of noisy channels. They
sit alongside turbo codes and polar codes as the capacity-approaching family that
displaced older convolutional schemes for high-rate links. The land-mobile trunking
protocols GopherTrunk targets (P25, DMR, NXDN, TETRA) predate this family and use simpler
block and convolutional codes, so GopherTrunk itself does not decode LDPC, but the codes
are foundational to the broader software-radio landscape.

## Sources

[^wiki]: [Robert G. Gallager](https://en.wikipedia.org/wiki/Robert_G._Gallager) — Wikipedia, for biography, awards, and the 1960 thesis.
[^ldpc]: [Low-density parity-check code](https://en.wikipedia.org/wiki/Low-density_parity-check_code) — Wikipedia, for the sparse parity-check structure and iterative decoding.
