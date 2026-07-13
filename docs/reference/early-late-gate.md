---
slug: early-late-gate
title: Early-late gate timing recovery
entry_type: algorithm
category: synchronization
description: Early-late gate timing recovery locks a symbol clock by comparing matched-filter samples taken slightly early and slightly late, steering the sampler toward the symbol peak.
keywords: early-late gate, early late gate, timing recovery, symbol synchronization, clock recovery, matched filter, timing error detector, Gardner, Mueller-Muller
aka: [early-late gate, early/late gate, ELG]
autolink: true
infobox:
  - { label: Type, value: Timing-error detector / loop }
  - { label: Recovers, value: Symbol sampling instant }
  - { label: Needs, value: Matched-filter output, ~2 samples/symbol }
see_also: [clock-recovery, gardner-timing-recovery, mueller-muller-timing-recovery, matched-filter, symbol-rate, root-raised-cosine-filter]
cite_urls:
  - https://en.wikipedia.org/wiki/Early-late_gate
  - https://en.wikipedia.org/wiki/Symbol_synchronization
---

**Early-late gate** timing recovery is a symbol-synchronization technique that
finds the best instant to sample each symbol by exploiting the symmetry of the
[matched-filter](/reference/matched-filter/) pulse: it takes one sample slightly
**early** and one slightly **late** relative to the current estimate, and if the
two differ in magnitude the sampler is off-centre and must be nudged.[^wiki] It is
one of the classic [clock-recovery](/reference/clock-recovery/) methods that lets a
receiver lock its [symbol clock](/reference/symbol-rate/) to a transmitter with no
shared timing reference.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A symmetric pulse peak with three sample points: an early sample and a late sample equal in height when centred, and unequal when the on-time sample is off the peak." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <path d="M40 130 Q 130 -5 220 130" fill="none" stroke="currentColor" stroke-width="1.4"/>
    <line x1="130" y1="20" x2="130" y2="135" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3"/><text x="130" y="150">on-time (peak)</text>
    <circle cx="100" cy="66" r="3" fill="currentColor"/><text x="86" y="58">early</text>
    <circle cx="160" cy="66" r="3" fill="currentColor"/><text x="176" y="58">late</text>
    <text x="130" y="14">|early| = |late| ⇒ locked</text>
    <path d="M300 130 Q 390 -5 480 130" fill="none" stroke="currentColor" stroke-width="1.4"/>
    <line x1="372" y1="20" x2="372" y2="135" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3"/>
    <circle cx="345" cy="52" r="3" fill="currentColor"/><text x="331" y="44">early</text>
    <circle cx="405" cy="92" r="3" fill="currentColor"/><text x="420" y="86">late</text>
    <text x="382" y="150">off-centre ⇒ error</text>
  </g>
</svg>
<figcaption>On a symmetric matched-filter pulse the early and late samples are equal only when the middle sample sits on the peak; any imbalance is a signed timing error that steers the clock.</figcaption>
</figure>

## How it works

After [matched filtering](/reference/matched-filter/) (e.g. a
[root-raised-cosine](/reference/root-raised-cosine-filter/) receive filter), each
symbol appears as a pulse whose peak marks the correct sampling instant. Because that
pulse is symmetric about its peak, a sample taken τ *before* the peak and one taken τ
*after* it have equal magnitude — **when** you are sampling on the peak. The loop
exploits this:

- Form a **timing error** ≈ |*y*(early)| − |*y*(late)| (an energy/magnitude
  comparison, so it works regardless of the symbol's polarity).
- If early > late, the true peak is earlier than assumed → advance the clock; if
  late > early, retard it. At the peak the two are equal and the error is zero.
- Feed this error through a loop filter to a numerically-controlled clock or
  interpolator, forming a closed timing loop that converges and then tracks slow
  drift.

Early-late gate typically wants roughly **two samples per symbol** (or an
interpolator to synthesize the early/late points) and a reasonably symmetric pulse
for the equal-magnitude assumption to hold.

## Contrast with Gardner and Mueller-Muller

Early-late gate is the intuitive ancestor of the detectors used most today:

- **[Gardner](/reference/gardner-timing-recovery/)** also uses samples around the
  symbol but forms its error from the *midpoint* sample between two symbols times
  their difference; it needs 2 samples/symbol, is non-data-aided, and is notably
  **carrier-phase independent**, which is why it dominates modern PSK/QAM modems.
- **[Mueller-Muller](/reference/mueller-muller-timing-recovery/)** is a
  *decision-directed*, **one-sample-per-symbol** detector — cheaper in samples but it
  needs reliable symbol decisions and is more sensitive to residual carrier offset.

Early-late gate is simple and robust but its explicit early/late samples cost extra
computation or interpolation compared with Gardner, and like all these detectors it
degrades under low SNR where the pulse shape is buried in noise.

## Relevance to SDR

Symbol-timing recovery is mandatory in every digital receiver: without it the sampler
drifts off the symbol peaks and error rates collapse. Early-late gate and its variants
appear in modem chipsets, satellite and telemetry receivers, and SDR toolkits.
Trunked-radio decoders recover a 4800 sym/s clock for C4FM/π-4-DQPSK modes this way.
GopherTrunk performs symbol-timing recovery in its demodulators using
Gardner/Mueller-Muller-style detectors; early-late gate is the foundational scheme
those refine.

## Sources

[^wiki]: [Early-late gate](https://en.wikipedia.org/wiki/Early-late_gate) — Wikipedia, on the early/late magnitude-comparison timing-error detector; see also symbol-synchronization loop background.
