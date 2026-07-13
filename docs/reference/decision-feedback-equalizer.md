---
slug: decision-feedback-equalizer
title: Decision-feedback equalizer (DFE)
entry_type: algorithm
category: equalization
description: A decision-feedback equalizer (DFE) combines a feedforward filter with feedback of past symbol decisions to cancel post-cursor intersymbol interference without noise enhancement.
keywords: decision feedback equalizer, DFE, feedforward filter, feedback filter, post-cursor ISI, error propagation, nonlinear equalizer, intersymbol interference
aka: [DFE, decision-feedback equalizer]
autolink: true
infobox:
  - { label: Type, value: Nonlinear equalizer }
  - { label: Structure, value: Feedforward + decision feedback }
  - { label: Caveat, value: Error propagation on wrong decisions }
see_also: [zero-forcing-equalizer, mmse-equalizer, adaptive-filter, maximum-likelihood-sequence-estimation, constellation-diagram, signal-to-noise-ratio]
cite_urls:
  - https://en.wikipedia.org/wiki/Decision_feedback_equalizer
  - https://en.wikipedia.org/wiki/Intersymbol_interference
---

A **decision-feedback equalizer (DFE)** is a nonlinear equalizer that pairs a
**feedforward filter** with a **feedback filter** fed by the receiver's own past symbol
**decisions**, using those already-detected symbols to subtract off the interference their
tails leave on later symbols.[^wiki] Because it cancels ISI with *known* (decided) symbols
rather than by inverting the channel, a DFE avoids the
[noise enhancement](/reference/zero-forcing-equalizer/) that plagues linear equalizers on
channels with deep nulls.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 175" role="img" aria-label="A feedforward filter feeds a summing junction and a decision device; the decisions pass through a feedback filter that is subtracted back at the summing junction." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="dfear" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="55" y="35" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="95" y="49">feedforward</text><text x="95" y="59">filter</text>
    <circle cx="185" cy="50" r="12" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="185" y="54" font-size="13">+</text>
    <rect x="235" y="35" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="270" y="49">decision</text><text x="270" y="59">device</text>
    <rect x="180" y="128" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="220" y="142">feedback</text><text x="220" y="152">filter</text>
    <line x1="15" y1="50" x2="54" y2="50" stroke-width="1.1" marker-end="url(#dfear)"/><text x="30" y="43">rx</text>
    <line x1="135" y1="50" x2="172" y2="50" stroke-width="1.1" marker-end="url(#dfear)"/>
    <line x1="197" y1="50" x2="234" y2="50" stroke-width="1.1" marker-end="url(#dfear)"/>
    <line x1="305" y1="50" x2="440" y2="50" stroke-width="1.1" marker-end="url(#dfear)"/><text x="360" y="43">decisions →</text>
    <path d="M330 50 V 143 H 261" fill="none" stroke-width="1.1" marker-end="url(#dfear)"/>
    <path d="M180 143 H 185 V 63" fill="none" stroke-width="1.1" marker-end="url(#dfear)"/><text x="150" y="100" text-anchor="end" font-size="8">subtract</text>
  </g>
</svg>
<figcaption>The DFE: a feedforward filter handles pre-cursor ISI, past decisions run through a feedback filter, and their reconstructed interference is subtracted before the decision device.</figcaption>
</figure>

## How it works

The channel's smeared impulse response has a *cursor* (the main symbol) with **pre-cursor**
tails from future symbols and **post-cursor** tails from past symbols. A DFE splits the job:

- The **feedforward filter** operates on the received samples and, like a linear MMSE
  equalizer, cleans up pre-cursor ISI and shapes the response — but it is deliberately not
  asked to invert deep nulls, so it enhances little noise.
- The **decision device** slices the summed signal to the nearest constellation point,
  producing a hard symbol decision.
- The **feedback filter** takes those *past decisions*, reconstructs the post-cursor
  interference they contribute to the current symbol, and subtracts it at the summing
  junction. Since decisions are noise-free symbol values, this cancellation adds no noise.

Both filters are typically adapted together by an [LMS](/reference/lms-algorithm/) or
[RLS](/reference/rls-algorithm/) rule under an [MMSE](/reference/mmse-equalizer/) criterion,
trained on a known sequence and then run decision-directed.

## Error propagation

The DFE's strength — feeding decisions back — is also its weakness. If the decision device
slices a symbol *wrong* (likely at low [SNR](/reference/signal-to-noise-ratio/)), the
feedback filter subtracts the *wrong* interference, which corrupts the next few symbols and
can trigger a short **burst of errors** before the loop recovers. This **error propagation**
is the DFE's defining caveat; it bounds how aggressively feedback can be used and is why the
gap to an [MLSE](/reference/maximum-likelihood-sequence-estimation/) receiver widens on the
harshest channels. Mitigations include precoding (e.g. Tomlinson–Harashima, which moves the
feedback to the transmitter) and reduced-state or soft-decision feedback.

## Relevance to SDR

The DFE is the workhorse equalizer for severe-ISI channels where a purely linear equalizer
would either leave too much interference or enhance too much noise — it appears in digital
TV (ATSC), voiceband and cable modems, DSL, and many microwave and HF links. It tightens the
[constellation](/reference/constellation-diagram/) on multipath channels that would defeat a
[zero-forcing](/reference/zero-forcing-equalizer/) design. GopherTrunk's narrowband
land-mobile decoders (P25, DMR, NXDN) rely on [matched filtering](/reference/matched-filter/)
and synchronisation rather than a DFE, so it is described here as a standard equalization
architecture from the broader RF world.

## Sources

[^wiki]: [Decision feedback equalizer](https://en.wikipedia.org/wiki/Decision_feedback_equalizer) — Wikipedia, on the feedforward/feedback structure, post-cursor cancellation, and error propagation.
