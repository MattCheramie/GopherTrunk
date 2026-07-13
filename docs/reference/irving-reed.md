---
slug: irving-reed
title: Irving S. Reed
entry_type: person
category: people
description: "Irving S. Reed (1923–2012) was an American mathematician who co-invented Reed-Solomon and Reed-Muller error-correcting codes used across digital radio and storage."
keywords: Irving Reed, Irving S. Reed, Reed-Solomon code, Reed-Muller code, error correction, coding theory, RX detector, radar
aka: [Irving Reed, Irving S. Reed]
autolink: true
infobox:
  - { label: Lived, value: "1923–2012" }
  - { label: Field, value: "Mathematics, engineering" }
  - { label: Known for, value: "Reed-Solomon and Reed-Muller codes" }
see_also: [reed-solomon-code, reed-muller-code, gustave-solomon, forward-error-correction, richard-hamming]
cite_urls:
  - https://en.wikipedia.org/wiki/Irving_S._Reed
  - https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction
---

**Irving S. Reed** (1923–2012) was an American mathematician and engineer who
co-invented two of the most important families of error-correcting codes: the
**[Reed-Muller code](/reference/reed-muller-code/)** in 1954 and, with
[Gustave Solomon](/reference/gustave-solomon/), the
**[Reed-Solomon code](/reference/reed-solomon-code/)** in 1960.[^wiki][^rs] The
Reed-Solomon code in particular became one of the most widely deployed pieces of
[forward error correction](/reference/forward-error-correction/) in history.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A block of data symbols followed by appended parity symbols, with a burst of errors spanning several symbols that Reed-Solomon decoding corrects." xmlns="http://www.w3.org/2000/svg">
  <g fill="none" stroke="currentColor">
    <rect x="30" y="50" width="30" height="34"/><rect x="60" y="50" width="30" height="34"/><rect x="90" y="50" width="30" height="34"/><rect x="120" y="50" width="30" height="34"/><rect x="150" y="50" width="30" height="34"/><rect x="180" y="50" width="30" height="34"/>
    <rect x="230" y="50" width="30" height="34"/><rect x="260" y="50" width="30" height="34"/><rect x="290" y="50" width="30" height="34"/><rect x="320" y="50" width="30" height="34"/>
  </g>
  <g fill="currentColor" fill-opacity="0.35"><rect x="90" y="50" width="30" height="34"/><rect x="120" y="50" width="30" height="34"/></g>
  <text x="120" y="40" text-anchor="middle" font-size="10" fill="currentColor">data symbols</text>
  <text x="290" y="40" text-anchor="middle" font-size="10" fill="currentColor">parity</text>
  <text x="120" y="108" text-anchor="middle" font-size="9" fill="currentColor">burst of errors</text>
  <text x="230" y="130" text-anchor="middle" font-size="9" fill="currentColor">corrected using parity symbols</text>
</svg>
<figcaption>Reed-Solomon appends parity symbols so that a burst of corrupted symbols can be located and corrected, the code Reed co-invented in 1960.</figcaption>
</figure>

## Life and work

Reed earned his PhD in mathematics from Caltech in 1949 and worked at MIT's Lincoln
Laboratory and later the RAND Corporation before joining the University of Southern
California as a professor.[^wiki] He was involved in some of the earliest digital
computing — he helped develop register-transfer notation used in machine design — but he
is remembered chiefly for coding theory. Beyond codes, his name also attaches to the
**Reed–Xiaoli (RX) detector** in hyperspectral imaging and to work on radar signal
processing.

## Contribution

Reed's two great codes attack error correction from complementary angles.

The **Reed-Muller code**, introduced in 1954, is a binary block code built from
multivariate Boolean polynomials. It offers a tunable trade-off between rate and error
protection and has an elegant, easily analysed structure; a simple first-order variant
was later used to protect telemetry from NASA's Mariner deep-space probes.

The **Reed-Solomon code**, published in 1960 with Gustave Solomon, works over larger
alphabets — it treats data as *symbols* (groups of bits) rather than single bits — and
is defined by evaluating a polynomial at many points. Its defining strength is
**burst-error correction**: because errors are counted per symbol, a run of consecutive
corrupted bits damages only a few symbols, which the parity symbols can then locate and
repair.[^rs] A Reed-Solomon code that adds 2*t* parity symbols can correct any *t*
symbol errors, a clean and powerful guarantee.

## Legacy

Reed-Solomon coding is one of the most successful algorithms ever fielded. Its
resistance to burst errors made it the natural choice for physical media and channels
where damage comes in clumps — scratches on a disc, fades on a radio link. Practical
decoding was made feasible by the Berlekamp–Massey algorithm and related work, building
on the same block-code foundations as [Richard Hamming](/reference/richard-hamming/)'s
earlier codes.

## Relevance to SDR

Reed-Solomon is pervasive in digital radio and storage: CDs, DVDs, Blu-ray, and
QR codes; DVB and ATSC digital television; deep-space telemetry; and many data-link
protocols all use it. In land-mobile radio it appears inside P25, where Reed-Solomon
codes protect critical trunking and header data. GopherTrunk, which decodes P25 and
related trunked systems, therefore relies on Reed-Solomon decoding in its control-channel
and header paths — a direct, everyday use of Reed's 1960 invention.

## Sources

[^wiki]: [Irving S. Reed](https://en.wikipedia.org/wiki/Irving_S._Reed) — Wikipedia, for biography, Reed-Muller code, and the RX detector.
[^rs]: [Reed–Solomon error correction](https://en.wikipedia.org/wiki/Reed%E2%80%93Solomon_error_correction) — Wikipedia, for the symbol-based construction and burst-error correction.
