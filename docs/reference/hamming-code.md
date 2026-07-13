---
slug: hamming-code
title: Hamming code
entry_type: algorithm
category: error-correction
description: Hamming codes are perfect single-error-correcting block codes that locate one flipped bit from a parity-check syndrome; the extended version also detects two errors (SECDED).
keywords: Hamming code, single error correction, SECDED, parity check matrix, syndrome, Hamming(7,4), block code, Richard Hamming, FEC
aka: [Hamming code, "Hamming(7,4)", SECDED]
autolink: true
infobox:
  - { label: Type, value: Single-error-correcting linear block code }
  - { label: Corrects, value: 1 error (2-error detect when extended) }
  - { label: Named for, value: Richard Hamming }
see_also: [forward-error-correction, golay-code, reed-muller-code, bch-code, richard-hamming, cyclic-redundancy-check]
related_lessons:
  - { title: "The demodulation pipeline", url: /learn/rf-sdr/demodulation-pipeline/ }
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Hamming_code
  - https://en.wikipedia.org/wiki/Hamming(7,4)
---

**Hamming codes** are the simplest family of single-error-correcting linear block
[error-correction](/reference/forward-error-correction/) codes: they add a small number of
parity bits so the receiver can pinpoint and flip **one** erroneous bit per block, and, when
extended by one overall parity bit, also **detect** (but not correct) a second error.[^wiki]
They are named for [Richard Hamming](/reference/richard-hamming/), who devised them at Bell
Labs around 1950 out of frustration with punched-card readers that could flag an error but
not fix it. The canonical example is the **Hamming(7,4)** code, which protects 4 data bits
with 3 parity bits.[^h74]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="Seven code positions labelled with three parity bits at powers of two and four data bits, with brackets showing each parity bit covering an overlapping set of positions." xmlns="http://www.w3.org/2000/svg">
  <g font-family="monospace" font-size="11" fill="currentColor" text-anchor="middle">
    <text x="40" y="52">p1</text><text x="100" y="52">p2</text><text x="160" y="52">d1</text><text x="220" y="52">p4</text><text x="280" y="52">d2</text><text x="340" y="52">d3</text><text x="400" y="52">d4</text>
  </g>
  <path d="M40 62 q30 16 60 0" fill="none" stroke="currentColor" stroke-opacity="0.55"/>
  <path d="M100 66 q60 20 120 0" fill="none" stroke="currentColor" stroke-opacity="0.55"/>
  <path d="M220 70 q90 22 180 0" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="230" y="108" text-anchor="middle" font-size="9" fill="currentColor">each parity bit checks the positions whose index shares that power-of-two bit</text>
</svg>
<figcaption>In Hamming(7,4), parity bits sit at positions 1, 2 and 4; the three failed/passed checks form a 3-bit number that is the position of the bad bit.</figcaption>
</figure>

## How it works

The trick is that parity bits cover **overlapping** subsets of positions chosen so the
pattern of failed checks — the **syndrome** — reads out, in binary, the index of the flipped
bit. Number the code positions 1..n. Parity bit *p1* covers every position whose index has
its 1's bit set (1, 3, 5, 7, …); *p2* covers positions with the 2's bit set (2, 3, 6, 7, …);
*p4* covers positions with the 4's bit set (4, 5, 6, 7); and so on for each power of two.
On receive, each parity check either passes (0) or fails (1). Read the failed checks as a
binary number: if it is zero, the block is (probably) clean; if it is, say, `101` = 5, then
position 5 is wrong and the decoder flips it. A single bit — data or parity — is corrected
identically, because the construction is symmetric across all positions.

Formally, a Hamming code is a linear code defined by a **parity-check matrix H** whose
columns are *all* the nonzero binary vectors of length *m*. A received word *r* is valid when
`H·rᵀ = 0`; otherwise the product `H·rᵀ` — the syndrome — is exactly the column of *H*
corresponding to the error position. Because every column of *H* is distinct and nonzero, any
single error yields a unique, decodable syndrome. This also fixes the code's shape: with *m*
parity bits the block length is `n = 2ᵐ − 1` and it carries `k = 2ᵐ − 1 − m` data bits, giving
the family (7,4), (15,11), (31,26), and so on. Hamming codes have minimum distance 3, so they
correct 1 error *or* detect 2, and they are **perfect** — every possible received word lies
within distance 1 of exactly one codeword, wasting no redundancy.

## Variants

- **Extended Hamming (SECDED).** Adding one overall parity bit raises the minimum distance to
  4, giving **Single-Error-Correct, Double-Error-Detect**. The syndrome now distinguishes "no
  error", "one correctable error", and "two errors, uncorrectable — flag it." This is the form
  used in ECC computer memory, e.g. Hamming(72,64).
- **Shortened Hamming.** Dropping some data positions yields non-power-of-two lengths tailored
  to a protocol field, such as the **Hamming(17,12)**, **Hamming(15,11)**, and **Hamming(13,9)**
  variants used inside DMR bursts, or **Hamming(10,6)** and **(16,11)** seen in other link-control
  words.
- **Relation to other codes.** Hamming codes are the distance-3 members of the broader
  [BCH](/reference/bch-code/) family, and their duals are the simplex codes — punctured
  [Reed–Muller](/reference/reed-muller-code/) codes. The [Golay](/reference/golay-code/) code
  can be seen as a much stronger cousin (correcting 3 errors) in the same "perfect code"
  category.

## In practice

Hamming's low overhead and trivial decoder make it ideal for small, high-value control fields
where a single flipped bit would otherwise corrupt a channel grant or address. It is weak
against **bursts** — two adjacent errors exceed its correcting power — so systems pair it with
[interleaving](/reference/interleaving/) to scatter bursts into isolated single errors, or nest
Hamming codes into a two-dimensional product code (see [BPTC](/reference/bptc/)) so row and
column passes cover each other's misses.

## Relevance to SDR

Hamming coding is woven through digital land-mobile radio. [DMR](/reference/dmr/) protects the
link-control and slot-type fields inside its bursts with shortened Hamming codes, and the
[BPTC(196,96)](/reference/bptc/) scheme that shields DMR data and control payloads is literally
a product of Hamming codes across rows and columns. Similar short Hamming variants appear in
[NXDN](/reference/nxdn/) and other control-signalling paths. GopherTrunk implements these
decoders directly: when it de-interleaves a DMR burst and recovers the link-control word, the
Hamming syndrome check is what lets it correct a bit hit rather than discard the whole message,
which is often the difference between reporting a talkgroup and dropping it.

## Sources

[^wiki]: [Hamming code](https://en.wikipedia.org/wiki/Hamming_code) — Wikipedia, for the parity-check construction, syndrome decoding, perfect-code property, and Richard Hamming's origin.
[^h74]: [Hamming(7,4)](https://en.wikipedia.org/wiki/Hamming(7,4)) — Wikipedia, for the worked (7,4) example, parity-check matrix, and the SECDED extension.
