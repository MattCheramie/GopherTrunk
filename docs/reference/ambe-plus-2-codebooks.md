---
slug: ambe-plus-2-codebooks
title: AMBE+2 codebooks
entry_type: term
category: voice-coding
description: The AMBE+2 codebooks are the fixed quantizer tables that turn a 49-bit voice frame's nine indices into model parameters — the harmonic-count table, the voicing-pattern table, the gain table, and the PRBA and HOC vector-quantizer codebooks.
keywords: AMBE+2 codebooks, quantizer tables, L table, Vuv table, gain Dg, PRBA vector quantizer, HOC codebook, 49-bit frame, mbelib ambe3600x2400
aka: [AMBE+2 quantizer tables, "AMBE+2 codebook tables"]
autolink: true
infobox:
  - { label: Frame, value: 49 information bits }
  - { label: Indices, value: "b0–b8 (nine)" }
  - { label: Codebooks, value: "L, Vuv, gain, PRBA, HOC" }
  - { label: Source, value: szechyjs/mbelib }
see_also: [ambe-plus-2, quantization, ambe-plus-2-tones, ambe-plus-2-fec, multi-band-excitation, discrete-fourier-transform, imbe-parameter-quantization]
cite_urls:
  - https://en.wikipedia.org/wiki/Multi-Band_Excitation
  - https://github.com/szechyjs/mbelib
---

The **AMBE+2 codebooks** are the fixed quantizer tables that give an
[AMBE+2](/reference/ambe-plus-2/) 2400 bps voice frame its meaning: the 49 information bits split
into nine quantization indices, and each index is a lookup into a stored codebook that returns a
piece of the [multi-band excitation](/reference/multi-band-excitation/) model — the harmonic count,
the per-band voicing pattern, the frame gain, and the spectral-amplitude vectors.[^mbe] They are
the AMBE+2 analogue of IMBE's [parameter quantization](/reference/imbe-parameter-quantization/),
but built almost entirely from vector [quantizer](/reference/quantization/) codebooks rather than
scalar formulas. GopherTrunk's tables are transcribed byte-identically from szechyjs/mbelib's
`ambe3600x2400_const.h`.[^mbelib]

<figure class="figure" markdown="0">
<svg viewBox="0 0 462 138" role="img" aria-label="The nine AMBE+2 indices b zero through b eight each address a stored codebook: b0 selects harmonic count and fundamental, b1 the voicing pattern, b2 the gain, b3 and b4 the PRBA vectors, and b5 through b8 the higher-order coefficient vectors, which together reconstruct the spectral amplitudes." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="car" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g stroke="currentColor" stroke-width="1.1" font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="14" y="20" width="70" height="18"/><text x="49" y="32">b0 → L table</text>
    <rect x="14" y="44" width="70" height="18"/><text x="49" y="56">b1 → Vuv</text>
    <rect x="14" y="68" width="70" height="18"/><text x="49" y="80">b2 → gain</text>
    <rect x="14" y="92" width="70" height="18"/><text x="49" y="104">b3,b4 → PRBA</text>
    <rect x="14" y="116" width="70" height="18"/><text x="49" y="128">b5–b8 → HOC</text>
    <rect x="300" y="52" width="148" height="46" fill="currentColor" fill-opacity="0.16"/>
    <text x="374" y="72">spectral amplitudes</text><text x="374" y="84" font-size="6.5">inverse DCT → Tl[1..L]</text>
  </g>
  <g stroke="currentColor" stroke-width="1" fill="none">
    <line x1="84" y1="29" x2="300" y2="60" marker-end="url(#car)"/>
    <line x1="84" y1="101" x2="300" y2="80" marker-end="url(#car)"/>
    <line x1="84" y1="125" x2="300" y2="90" marker-end="url(#car)"/>
  </g>
</svg>
<figcaption>Each of the nine AMBE+2 indices addresses a stored codebook; the gain, PRBA, and HOC lookups feed an inverse DCT that reconstructs the per-harmonic spectral amplitudes.</figcaption>
</figure>

## The scalar tables

Three of the codebooks are simple indexed arrays. The **L table** (`AmbePlusLtable`, 126 entries)
maps the 7-bit fundamental index `b_0` directly to the harmonic count `L`, which climbs from 9 at
the high-pitch end to 56 at the low-pitch end. The **gain table** (`AmbePlusDg`, 64 entries) maps
the 6-bit index `b_2` to a gain delta — AMBE+2 codes gain differentially, so the synthesizer adds
half the previous frame's gain to this delta. The **voicing-pattern table** (`AmbePlusVuv`,
16 rows of 8) maps the 4-bit `b_1` to one of sixteen voiced/unvoiced patterns across eight bands;
the decoder picks a band per harmonic as `⌊l · 16 · f₀⌋` and reads that column:

```go
var AmbePlusVuv = [16][8]int{
    {0, 0, 0, 0, 0, 0, 0, 0}, // b1 = 0: all unvoiced
    {0, 0, 0, 0, 0, 0, 1, 1},
    // … fourteen further rows, up to …
    {1, 1, 1, 1, 1, 1, 1, 1}, // b1 = 15: all voiced
}
```

## The vector-quantizer codebooks

The spectral envelope uses true vector quantizers — each index selects a short vector, not a
scalar. The two **PRBA** codebooks carry the coarse gain shape: `AmbePlusPRBA24` (512 entries of
3 values, indexed by the 9-bit `b_3`) supplies `Gm[2..4]`, and `AmbePlusPRBA58` (128 entries of
4 values, indexed by the 7-bit `b_4`) supplies `Gm[5..8]`. The four **HOC** (higher-order
coefficient) codebooks (`AmbePlusHOCb5`–`HOCb8`, each 16 entries of 4 values, indexed by `b_5`–`b_8`)
fill in the finer per-band detail.

| Codebook | Dimensions | Index | Supplies |
|----------|-----------|-------|----------|
| `AmbePlusLtable` | 126 | b0 (7 bit) | harmonic count L |
| `AmbePlusVuv` | 16 × 8 | b1 (4 bit) | voicing pattern |
| `AmbePlusDg` | 64 | b2 (6 bit) | gain delta |
| `AmbePlusPRBA24` | 512 × 3 | b3 (9 bit) | PRBA Gm[2..4] |
| `AmbePlusPRBA58` | 128 × 4 | b4 (7 bit) | PRBA Gm[5..8] |
| `AmbePlusHOCb5..b8` | 16 × 4 each | b5..b8 | HOC per band |

## From codebooks to a spectrum

The lookups are only the front end. The eight PRBA values feed an inverse 8-point DCT-II to
produce band DC terms; those plus the HOC vectors form the per-band coefficient sets `Cik`, sized
to the per-band harmonic counts `Ji` read from an L-indexed table (`AmbePlusLmprbl`). A final
per-band inverse [DCT](/reference/discrete-fourier-transform/) expands each band's coefficients into
the log-amplitude residuals `Tl[1..L]`, whose lengths sum to `L` by construction of the table.
This is where the stored codebooks become a continuous spectral envelope.

## Relevance to SDR

These codebooks are what GopherTrunk consults for every AMBE+2 frame in P25 Phase 2, DMR, and NXDN
voice. Because they are transcribed 1:1 from mbelib and the surrounding control flow mirrors the C
reference, any drift between the two is detectable — a mis-copied codebook entry would decode to a
slightly wrong amplitude rather than a crash. The table *values* are facts about the published
algorithm and ISC-licensed via mbelib; the AMBE+2 algorithm that drives them carries a separate
patent status, discussed in the [AMBE+2](/reference/ambe-plus-2/) entry.

## Sources

[^mbe]: [Multi-Band Excitation](https://en.wikipedia.org/wiki/Multi-Band_Excitation) — Wikipedia, on the MBE model parameters the AMBE+2 codebooks quantize.
[^mbelib]: [szechyjs/mbelib](https://github.com/szechyjs/mbelib) — the ISC-licensed reference whose `ambe3600x2400_const.h` codebook values GopherTrunk reproduces.
