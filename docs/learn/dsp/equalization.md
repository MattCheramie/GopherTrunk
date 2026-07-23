---
slug: equalization
title: Equalization & multipath
description: Why reflections smear symbols together and how an adaptive equalizer undoes a channel's distortion — the filter that cleans up inter-symbol interference.
keywords: equalization, multipath, inter-symbol interference, adaptive equalizer, channel distortion, isi, delay spread, equalizer filter
level: advanced
status: full
prereq:
  - matched-filters-and-pulse-shaping
  - fir-filters
faq:
  - q: What is multipath and why does it cause errors?
    a: "Multipath is the arrival of a signal by several paths at once — a direct ray plus reflections off buildings and terrain — each delayed by a different amount. The delayed copies overlap the direct signal, so energy from one symbol lands on top of the next. That overlap is inter-symbol interference, and it blurs the constellation until symbols become impossible to tell apart."
  - q: How does an equalizer fix a distorted channel?
    a: "An equalizer is a filter whose response is the inverse of the channel's. Where the channel spread and delayed the signal, the equalizer applies an opposite correction that collapses the echoes back onto the direct path. Because the channel changes as the receiver or reflectors move, the equalizer is adaptive — it continuously adjusts its coefficients to track the channel using the recovered symbols as a reference."
---

# Equalization & multipath

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A radio signal reaches the antenna by many paths — a direct ray plus **reflections** — each
delayed differently. Those overlapping copies spill one symbol's energy onto the next:
**multipath-induced inter-symbol interference (ISI)**. An **equalizer** is a filter that
applies the **inverse** of the channel, collapsing the echoes back together. Because the
channel changes as things move, the equalizer is **adaptive**, retuning itself continuously.
</div>

[Matched filtering](/learn/dsp/matched-filters-and-pulse-shaping/) kept a clean channel
ISI-free — but the air is not clean. This lesson is about the distortion the *channel*
adds and the filter that undoes it. It builds on
[FIR filters](/learn/dsp/fir-filters/) and connects to [propagation](/learn/rf-sdr/propagation/).

## Multipath: many copies, many delays

A transmitted signal rarely takes one path. It goes direct, and it also bounces off
buildings, hills, and vehicles, so several delayed copies pile up at the receiver. When
the spread of those delays approaches a symbol period, a reflection of *this* symbol
arrives at the same time as the *next* one — inter-symbol interference, this time caused
by the channel rather than by pulse shaping.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A transmitter and receiver with a direct path and a longer reflected path off a building, and a small plot showing the direct pulse plus a delayed echo overlapping the next symbol." xmlns="http://www.w3.org/2000/svg">
  <circle cx="40" cy="60" r="10" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="40" y="88" text-anchor="middle" font-size="9" fill="currentColor">TX</text>
  <circle cx="290" cy="60" r="10" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="290" y="88" text-anchor="middle" font-size="9" fill="currentColor">RX</text>
  <line x1="50" y1="60" x2="280" y2="60" stroke="currentColor" stroke-width="1.5"/>
  <text x="165" y="54" text-anchor="middle" font-size="8" fill="currentColor">direct</text>
  <rect x="150" y="8" width="30" height="16" fill="none" stroke="currentColor" stroke-opacity="0.6"/>
  <text x="165" y="20" text-anchor="middle" font-size="7" fill="currentColor">bldg</text>
  <path d="M50 55 L160 24 L280 55" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.6"/>
  <text x="120" y="34" text-anchor="middle" font-size="8" fill="currentColor">reflected (delayed)</text>
  <g>
    <line x1="360" y1="110" x2="510" y2="110" stroke="currentColor" stroke-opacity="0.3"/>
    <path d="M370 110 L390 110 L390 60 L410 60 L410 110 L430 110" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <path d="M390 110 L410 110 L410 85 L430 85 L430 110 L450 110" fill="none" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.6" stroke-dasharray="3 2"/>
    <text x="435" y="128" text-anchor="middle" font-size="8" fill="currentColor">echo spills into next symbol</text>
  </g>
</svg>
<figcaption>Multipath: a delayed reflection overlaps the following symbol, causing inter-symbol interference the equalizer must remove.</figcaption>
</figure>

On the [constellation](/learn/dsp/constellations-and-symbol-mapping/) this shows as points
smeared and pulled off their ideal spots — not random fuzz (that's noise) but a
structured distortion the channel imposed.

## The equalizer: inverting the channel

If multipath is a filter the channel applied to your signal, then undoing it is applying
the **inverse filter**. That is an **equalizer** — usually an FIR filter whose taps are
set so that channel-then-equalizer together look like a clean, flat, delay-only path. It
gathers the scattered echo energy back onto the direct symbol, reopening the
[eye](/learn/dsp/the-eye-diagram/) and tightening the constellation.

```text
distorted:  channel  ->  smeared symbols  ->  errors
equalized:  channel  ->  EQUALIZER (inverse)  ->  clean symbols
```

## Why it must be adaptive

The catch: the channel is not fixed. Move the receiver, and the reflectors, or a passing
truck, and the set of paths changes moment to moment. A one-time inverse would be wrong
seconds later. So the equalizer is **adaptive** — it continuously nudges its taps to keep
the output clean, judging its own success against the recovered symbols (which sit on a
known grid) and error-driven update rules. It is the same feedback idea as the
[timing](/learn/dsp/clock-and-symbol-recovery/) and
[carrier](/learn/dsp/carrier-and-frequency-recovery/) loops, applied to channel shape.

## Where it fits — and where it doesn't

Not every system equalizes. Narrowband voice modes with modest data rates often have a
symbol period long enough that typical delay spread causes little ISI, so they skip an
explicit equalizer. Higher-rate and wideband systems, where delay spread is a real
fraction of a symbol, lean on equalization heavily. Either way, understanding multipath
explains a signal that is strong on the meter yet decodes poorly — a hallmark the
[troubleshooting](/learn/digital-trunking/troubleshooting-a-decode/) guide flags, since
strength alone can't overcome a smeared channel.

<div class="knowledge-check" data-quiz data-correct-msg="Right — an adaptive equalizer applies the inverse of the changing channel to undo multipath ISI." markdown="0">
  <p class="knowledge-check__q">Quick check: why must an equalizer be adaptive?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To slowly increase the sample rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Because the multipath channel keeps changing as the receiver and reflectors move</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Because floating-point math drifts over time</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Multipath** delivers delayed copies of the signal; overlap causes channel **ISI**.
- It smears the constellation in a **structured** way, distinct from random noise.
- An **equalizer** applies the channel's **inverse**, collapsing echoes back onto the symbol.
- The channel changes, so the equalizer is **adaptive**, retuning its taps continuously.

Next up: the three numbers that measure a link's health — SNR, EVM, and BER.
