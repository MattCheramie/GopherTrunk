---
slug: cfar-detection
title: CFAR detection
entry_type: algorithm
category: estimation-array
description: CFAR detection sets a detection threshold from a local estimate of the surrounding noise so the false-alarm rate stays constant as the noise floor varies, used in radar and spectral sensing.
keywords: CFAR, constant false alarm rate, CA-CFAR, cell averaging, OS-CFAR, ordered statistic, guard cells, radar detection, adaptive threshold, energy detection, spectral occupancy
aka: [CFAR, constant false alarm rate detection, CA-CFAR, OS-CFAR]
autolink: true
infobox:
  - { label: Type, value: Adaptive-threshold detector }
  - { label: Holds constant, value: False-alarm probability }
  - { label: Variants, value: CA-CFAR, OS-CFAR, GO/SO }
see_also: [energy-detection, noise-floor, signal-to-noise-ratio, matched-filter, welch-method]
cite_urls:
  - https://en.wikipedia.org/wiki/Constant_false_alarm_rate
  - https://en.wikipedia.org/wiki/Detection_theory
---

**CFAR** (constant false alarm rate) detection sets a target-present threshold not from a
fixed number but from a running estimate of the noise and clutter *around* each cell under
test, so that as the [noise floor](/reference/noise-floor/) rises and falls the probability
of a false alarm stays fixed.[^wiki] A fixed threshold either drowns in false alarms when
the noise climbs or goes deaf when it drops; CFAR keeps the false-alarm rate pinned by
making the threshold track the local background.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A sliding window over range or frequency cells: a cell under test flanked by guard cells and reference cells; the reference cells estimate the noise, which is scaled to form a threshold that the test cell must exceed to be declared a detection." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <g stroke="currentColor" stroke-width="1" fill="none">
      <rect x="30" y="70" width="22" height="30"/><rect x="52" y="70" width="22" height="30"/><rect x="74" y="70" width="22" height="30"/>
      <rect x="96" y="70" width="18" height="30"/><rect x="114" y="70" width="18" height="30"/>
      <rect x="132" y="60" width="24" height="40" stroke-width="1.6"/>
      <rect x="156" y="70" width="18" height="30"/><rect x="174" y="70" width="18" height="30"/>
      <rect x="192" y="70" width="22" height="30"/><rect x="214" y="70" width="22" height="30"/><rect x="236" y="70" width="22" height="30"/>
    </g>
    <text x="63" y="118">reference (noise est.)</text>
    <text x="114" y="55">guard</text>
    <text x="144" y="52" font-size="9">CUT</text>
    <text x="225" y="118">reference (noise est.)</text>
    <line x1="30" y1="135" x2="258" y2="135" stroke="currentColor" stroke-width="1" marker-end="url(#cfar)"/><text x="144" y="150">range / frequency cells →</text>
    <rect x="300" y="55" width="140" height="60" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/>
    <text x="370" y="72" font-size="9">T = α · noise_est</text>
    <text x="370" y="88" font-size="9">CUT &gt; T ?</text>
    <text x="370" y="104" font-size="9">→ detection</text>
    <line x1="258" y1="80" x2="299" y2="82" stroke="currentColor" stroke-width="1.1" marker-end="url(#cfar)"/>
  </g>
  <defs><marker id="cfar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A CFAR window estimates local noise from reference cells (guard cells excluded), scales it by a factor α set by the desired false-alarm rate, and declares a detection when the cell under test exceeds that adaptive threshold.</figcaption>
</figure>

## How it works

Slide a window across the range, Doppler, or frequency cells. The **cell under test (CUT)**
is the sample being judged. Immediately around it sit a few **guard cells**, excluded so a
strong target's energy leaking into neighbours cannot inflate its own threshold. Beyond the
guards are the **reference cells**, which are assumed to be target-free and are used to
estimate the local noise/clutter power. Multiply that estimate by a scale factor `α` — chosen
analytically from the desired false-alarm probability and the number of reference cells — to
form the threshold. If the CUT exceeds it, declare a detection. Because the threshold rides
on the measured background, the false-alarm rate is (ideally) constant everywhere.

## Variants

- **CA-CFAR (cell-averaging).** Averages all reference cells. Optimal in uniform noise, best
  false-alarm control, but loses targets near a clutter edge and suffers *masking* when a
  second target sits in the reference window.
- **GO-/SO-CFAR (greatest-of / smallest-of).** Average each side separately and take the
  larger (GO, tames clutter edges) or the smaller (SO, keeps closely spaced targets).
- **OS-CFAR (ordered-statistic).** Sort the reference cells and pick the k-th value instead
  of the mean. Robust to interfering targets and clutter edges at a small SNR penalty and
  higher compute cost.

## Relevance to SDR

CFAR is the classical radar target-detection stage, and the exact same idea transfers to
receivers: deciding whether a bin in a [Welch-averaged](/reference/welch-method/)
spectrum is occupied, driving squelch, or feeding a spectrum-occupancy/energy-detection map
all benefit from a threshold that tracks the local floor rather than a hand-tuned constant.
It pairs naturally with a [matched filter](/reference/matched-filter/) (which maximises SNR)
followed by CFAR (which decides). **GopherTrunk** does not implement a formal CFAR detector —
its signal presence and squelch logic use simpler power/quality thresholds — but CFAR is the
principled generalisation of that thresholding and the direct sibling of
[energy detection](/reference/energy-detection/) in spectrum sensing.

## Sources

[^wiki]: [Constant false alarm rate](https://en.wikipedia.org/wiki/Constant_false_alarm_rate) — Wikipedia, on adaptive thresholding from a local noise estimate (CA-CFAR, OS-CFAR).
