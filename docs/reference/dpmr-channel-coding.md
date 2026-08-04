---
slug: dpmr-channel-coding
title: dPMR Channel Coding
entry_type: term
category: error-correction
description: "dPMR protects its CSBK and traffic payloads with a short-block cyclic code under a rate-3/4 convolutional outer code plus interleaving. GopherTrunk parses dPMR at the post-FEC layer today, with the error-correction stage documented as an honest follow-up."
keywords: dPMR channel coding, dPMR FEC, cyclic code, rate 3/4 convolutional, interleaving, CSBK coding, ETSI TS 102 658, deferred FEC
aka: ["dPMR FEC", "dPMR channel coding"]
autolink: true
infobox:
  - { label: Outer code, value: rate-3/4 convolutional }
  - { label: Inner, value: short-block cyclic code }
  - { label: Plus, value: interleaving over the burst }
  - { label: Spec, value: ETSI TS 102 658 §6 }
see_also: [forward-error-correction, convolutional-code, dpmr, dpmr-csbk, dpmr-frame-sync, puncturing, interleaving, cyclic-redundancy-check]
cite_urls:
  - https://en.wikipedia.org/wiki/DPMR
  - https://en.wikipedia.org/wiki/Convolutional_code
---

**dPMR channel coding** is the [forward-error-correction](/reference/forward-error-correction/)
that protects a [dPMR](/reference/dpmr/) [CSBK](/reference/dpmr-csbk/) or traffic payload as it
crosses a noisy 4FSK channel.[^wiki] Per ETSI TS 102 658 §6, dPMR wraps its payload bits in a
short-block cyclic code under a rate-¾ [convolutional](/reference/convolutional-code/) outer
code, then [interleaves](/reference/interleaving/) the result across the burst so a fade turns
into scattered single errors the decoder can repair.[^conv] GopherTrunk currently parses dPMR at
the layer *above* this stage — an honest, documented deferral covered below.

<figure class="figure" markdown="0">
<svg viewBox="0 0 480 150" role="img" aria-label="The dPMR coding stack drawn as a pipeline from payload bits through a short-block cyclic code, a rate-three-quarters convolutional code, and an interleaver to the on-air burst; a dashed boundary marks that GopherTrunk implements the structured payload parsing at the top of the stack while the cyclic, convolutional, and interleave stages are a documented follow-up." xmlns="http://www.w3.org/2000/svg">
  <g font-size="7.5" fill="currentColor">
    <rect x="12" y="46" width="70" height="30" fill="currentColor" fill-opacity="0.24" stroke="currentColor" stroke-width="1"/>
    <text x="47" y="60" text-anchor="middle">payload bits</text>
    <text x="47" y="70" text-anchor="middle" font-size="6.5">(CSBK / traffic)</text>
    <path d="M82 61 L100 61" stroke="currentColor" stroke-width="1" marker-end="url(#d)"/>
    <rect x="100" y="46" width="74" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
    <text x="137" y="60" text-anchor="middle">cyclic</text>
    <text x="137" y="70" text-anchor="middle" font-size="6.5">short block</text>
    <path d="M174 61 L192 61" stroke="currentColor" stroke-width="1" marker-end="url(#d)"/>
    <rect x="192" y="46" width="74" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
    <text x="229" y="60" text-anchor="middle">conv R=3/4</text>
    <text x="229" y="70" text-anchor="middle" font-size="6.5">outer code</text>
    <path d="M266 61 L284 61" stroke="currentColor" stroke-width="1" marker-end="url(#d)"/>
    <rect x="284" y="46" width="74" height="30" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
    <text x="321" y="60" text-anchor="middle">interleave</text>
    <path d="M358 61 L376 61" stroke="currentColor" stroke-width="1" marker-end="url(#d)"/>
    <rect x="376" y="46" width="90" height="30" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="421" y="64" text-anchor="middle">on-air burst</text>
  </g>
  <text x="100" y="98" font-size="7.5" fill="currentColor">dashed = documented follow-up (GopherTrunk parses above this stage)</text>
  <defs><marker id="d" markerWidth="6" markerHeight="6" refX="5" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>dPMR's coding stack runs payload bits through a short-block cyclic code, a rate-3/4 convolutional outer code, and an interleaver before the burst goes on air. GopherTrunk implements the structured payload layer at the top; the cyclic, convolutional, and interleave stages (dashed) are a named deferral.</figcaption>
</figure>

## What the coding does

The three stages each answer a different failure mode. The short-block cyclic code adds a small
number of parity bits so the receiver can detect — and for the shortest blocks, correct — a
handful of bit errors, and it doubles as an integrity check like a
[CRC](/reference/cyclic-redundancy-check/). The rate-¾ convolutional outer code spreads each
input bit's influence across several output bits, so a Viterbi-style decoder can reconstruct the
original stream even where individual bits were lost; the ¾ rate keeps the overhead modest,
appropriate for dPMR's narrow 6.25 kHz channel where bandwidth is scarce. The interleaver then
reorders the coded bits so that a *burst* error on the channel — a fade lasting several symbols —
is spread out into isolated single-bit errors once de-interleaved, which is exactly the error
pattern the cyclic and convolutional decoders handle best. Together they let a marginal dPMR
signal deliver a correct CSBK where an uncoded block would be discarded.

## The implementation gap

GopherTrunk is deliberate about what it has and has not built. Its dPMR package ships a clean
structured surface — [frame sync](/reference/dpmr-frame-sync/) detection, CSBK parsing, the
opcode enum, the band-plan resolver, and the trunking state machine — so the engine can consume
grants end-to-end against fixtures. The CSBK parser, though, **assumes the upstream caller has
already corrected errors**: it maps 80 clean bits into fields, and the FEC that would produce
those 80 clean bits from an on-air burst is listed among the package's honest deferrals. The
source names the missing pieces explicitly: the interleaver plus the short-block-cyclic /
rate-¾-convolutional FEC over the CSBK bits, the 4FSK demodulator and symbol-clock recovery for
the 2400 sym/sec air interface, and voice-frame extraction into the AMBE+2 vocoder. Flagging
these as named follow-ups — rather than silently shipping a half-built decoder — is a deliberate
choice: the structured layer is testable now against pre-corrected fixtures, and the FEC slots
in beneath it without changing the surface above.

## Relevance to SDR

`internal/radio/dpmr/dpmr.go` documents this boundary in the package doc comment, and the
parsing in `csbk.go` operates on the post-FEC bit layer. For a live off-air dPMR control channel
the chain would be: 4FSK demod → FS3 sync → de-interleave → convolutional + cyclic decode →
80-bit CSBK → parse. Today GopherTrunk implements the sync-and-parse ends of that chain and
defers the demod-and-FEC middle, which is the correct order for a decoder built against captured
and replayed fixtures: the structured trunking logic is proven first, and the error-correction
that feeds it is added as a self-contained stage once conformance vectors are in hand.

## Sources

[^wiki]: [dPMR](https://en.wikipedia.org/wiki/DPMR) — Wikipedia, on the ETSI dPMR standard and its physical layer.
[^conv]: [Convolutional code](https://en.wikipedia.org/wiki/Convolutional_code) — Wikipedia, on the outer code family dPMR's channel coding uses.
