---
slug: correlate-access-code
title: Access-code correlation
entry_type: algorithm
category: sdr-app-building
description: Access-code correlation is a sliding cross-correlation that searches a symbol stream for a known sync word or access code, flagging positions where the match exceeds a threshold.
keywords: access code correlation, correlate access code, sync word search, sliding correlation, cross-correlation sync, frame sync detection, access code detector, preamble correlation
aka: [access code correlator, sync-word correlation, sliding correlator]
autolink: true
infobox:
  - { label: Type, value: DSP / detection algorithm }
  - { label: Finds, value: Known sync word in a stream }
  - { label: Method, value: Sliding cross-correlation }
see_also: [preamble-correlation, matched-filter, frame-synchronization, barker-code, deframing]
cite_urls:
  - https://en.wikipedia.org/wiki/Cross-correlation
  - https://en.wikipedia.org/wiki/Frame_synchronization
---

**Access-code correlation** is a sliding cross-correlation that searches a symbol stream for a
known sync word (the "access code") and flags every position where the match strength crosses a
threshold.[^xcorr] It is the standard way a receiver answers the question "where does a frame
start?" — the detection step that anchors [deframing](/reference/deframing/) and
[frame synchronization](/reference/frame-synchronization/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A fixed pattern slides along an incoming stream; at each offset an inner product is computed, and the correlation peaks sharply at the position where the pattern aligns with the embedded sync word." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="cacar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-family="monospace" font-size="9" fill="currentColor"><text x="10" y="26">stream:  + - - + …  [+ - - + - +]  … - +</text></g>
  <g font-family="monospace" font-size="9" fill="currentColor"><text x="10" y="44">pattern:          [+ - - + - +]</text></g>
  <line x1="60" y1="34" x2="240" y2="34" stroke="currentColor" stroke-width="0.7" stroke-dasharray="2 2"/>
  <text x="250" y="40" font-size="7" fill="currentColor">slide →</text>
  <line x1="30" y1="115" x2="440" y2="115" stroke="currentColor" stroke-width="1"/>
  <line x1="30" y1="115" x2="30" y2="60" stroke="currentColor" stroke-width="1"/>
  <path d="M40 112 L90 108 L140 110 L185 106 L215 62 L245 107 L300 110 L360 108 L430 112" fill="none" stroke="currentColor" stroke-width="1.4"/>
  <line x1="215" y1="62" x2="215" y2="115" stroke="currentColor" stroke-width="0.6" stroke-dasharray="2 2"/>
  <text x="215" y="74" font-size="7" fill="currentColor" text-anchor="middle">peak = frame start</text>
  <text x="36" y="72" font-size="7" fill="currentColor">corr</text>
</svg>
<figcaption>The known access code is slid along the stream; the correlation spikes at the offset where it aligns.</figcaption>
</figure>

## How it works

Let the access code be a length-`N` pattern `p[0..N−1]` of expected symbol values (for a binary
sync word, `±1`). The correlator holds the most recent `N` incoming symbols and, at every new
symbol, computes the inner product with the pattern:

```
corr = Σ_{k=0}^{N-1} s[n-N+1+k] · p[k]
```

When the incoming symbols match the pattern, every product is positive and `corr` is large;
when they don't, the products scatter and cancel toward zero. A detection is declared wherever
`corr` exceeds a threshold. Because the sum is recomputed as a sliding window, the cost is `N`
multiply-adds per input symbol — the same arithmetic as a [matched filter](/reference/matched-filter/)
whose impulse response is the reversed access code, which is exactly what this is: the matched
filter *for the sync word*.

## Variants

- **Hard vs soft symbols.** Correlating hard-decided `±1` symbols is cheapest; correlating the
  demodulator's *soft* values preserves confidence information and detects the code a couple of
  dB deeper into the noise.
- **Normalised correlation.** Dividing by the running signal energy makes the threshold
  independent of amplitude/AGC state, so one fixed threshold works across signal levels.
- **Error-tolerant matching.** For binary codes, thresholding on the number of matching bits
  (Hamming distance) lets the detector accept a sync word with a few flipped bits — essential
  at low SNR.
- **Differential correlation.** When there is a residual carrier offset, correlating
  symbol-to-symbol *transitions* rather than absolute values removes the phase rotation.

## Relevance to SDR

Access-code correlation is the front door of frame recovery in essentially every burst or
framed protocol: GNU Radio's `correlate_access_code` block, P25's Frame Sync search, DMR's SYNC
detection, and preamble detectors for ADS-B and AIS all reduce to this operation. Sync words are
chosen — often [Barker codes](/reference/barker-code/) — precisely so their autocorrelation has
one sharp peak and low sidelobes, which is what makes the threshold decision reliable.

**GopherTrunk** implements it directly. `internal/dsp/sync` provides a `Correlator` that keeps a
ring buffer of soft symbols and slides an inner product against a stored pattern, appending the
stream indices where the correlation clears a threshold; `internal/dsp/stats` adds a normalised
full cross-correlation used offline. Every GT protocol that locks a frame — P25, DMR, NXDN,
TETRA, D-STAR, YSF — leans on this sync-word search before [deframing](/reference/deframing/)
can extract a payload.

## Sources

[^xcorr]: [Cross-correlation](https://en.wikipedia.org/wiki/Cross-correlation) — Wikipedia, on the sliding inner product that measures alignment between a pattern and a signal.
[^fsync]: [Frame synchronization](https://en.wikipedia.org/wiki/Frame_synchronization) — Wikipedia, on using a known sync sequence to locate frame boundaries.
