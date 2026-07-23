---
slug: matched-filters-and-pulse-shaping
title: Matched filters & pulse shaping
description: Root-raised-cosine filters, the Nyquist no-ISI criterion, and why a matched filter at the receiver maximizes SNR right where each symbol is decided.
keywords: matched filter, pulse shaping, root raised cosine, rrc filter, nyquist isi criterion, raised cosine, intersymbol interference, symbol filter
level: advanced
status: full
prereq:
  - fir-filters
  - convolution-and-impulse-response
faq:
  - q: What is a matched filter and why does it maximize SNR?
    a: "A matched filter is a filter whose impulse response is a time-reversed copy of the transmitted symbol pulse. Correlating the incoming signal against the exact pulse shape gathers all of that symbol's energy into a single peak at the decision instant, while noise, being uncorrelated with the pulse, does not add up the same way. The result is the highest possible signal-to-noise ratio at the moment the symbol is decided."
  - q: Why split the pulse shaping into a root-raised-cosine at each end?
    a: "The transmitter and receiver each apply a root-raised-cosine filter so that their cascade equals a full raised-cosine response. The full raised cosine satisfies the Nyquist no-ISI criterion, so symbols do not smear into their neighbours at the sampling instants, while splitting it in half also makes the receiver's filter a matched filter for the transmitted pulse — one design serves both goals at once."
---

# Matched filters & pulse shaping

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Square symbol pulses spread forever in frequency, so transmitters **shape** them. A
**raised-cosine** pulse satisfies the **Nyquist no-ISI criterion** — it is zero at every
*other* symbol's sampling instant, so neighbours don't smear together. Split it as a
**root-raised-cosine (RRC)** filter at *both* transmitter and receiver: their cascade is
a full raised cosine, and the receiver half is also a **matched filter** that maximizes
**SNR** exactly where each symbol is decided.
</div>

The [FIR](/learn/dsp/fir-filters/) lesson built filters that select frequency bands. This
lesson uses a filter for a different job: shaping the symbols themselves so they can be
read cleanly. It sits right before symbol decisions in the
[demodulation](/learn/dsp/demodulation/) chain.

## The problem with square pulses

The obvious way to send a symbol is a rectangular pulse — hold a level for one symbol
period. But a sharp-edged pulse occupies an enormous bandwidth (its spectrum is a sinc
that trails off slowly), spilling into adjacent channels. Band-limit it crudely and its
tails ring into neighbouring symbols — **inter-symbol interference (ISI)**, where each
symbol contaminates the ones around it.

## The Nyquist no-ISI criterion

Harry Nyquist's insight: a pulse can be band-limited *and* ISI-free if its shape is zero
at every symbol instant except its own. Then, sampled at the symbol centres, each pulse
contributes only to *its* sample and nothing to the others — even though the pulses
overlap in between.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A raised-cosine pulse peaking at the centre symbol instant and crossing zero exactly at the neighbouring symbol instants marked by vertical dashed lines." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="110" x2="500" y2="110" stroke="currentColor" stroke-opacity="0.3"/>
  <g stroke="currentColor" stroke-opacity="0.3" stroke-dasharray="3 3">
    <line x1="120" y1="20" x2="120" y2="120"/><line x1="200" y1="20" x2="200" y2="120"/>
    <line x1="280" y1="20" x2="280" y2="120"/><line x1="360" y1="20" x2="360" y2="120"/><line x1="440" y1="20" x2="440" y2="120"/>
  </g>
  <path d="M40 108 Q 120 118 160 112 Q 200 104 220 60 Q 260 -10 300 60 Q 320 104 360 112 Q 400 118 480 108" fill="none" stroke="currentColor" stroke-width="2"/>
  <circle cx="280" cy="35" r="3" fill="currentColor"/>
  <text x="280" y="24" text-anchor="middle" font-size="9" fill="currentColor">its own instant (peak)</text>
  <text x="150" y="132" text-anchor="middle" font-size="9" fill="currentColor">zero at neighbours</text>
</svg>
<figcaption>A raised-cosine pulse peaks at its own symbol instant and is exactly zero at every neighbour's — so overlapping pulses cause no ISI when sampled at the centres.</figcaption>
</figure>

The **raised-cosine** pulse is the classic shape that meets this criterion. A roll-off
factor sets how gently its spectrum tapers — a small roll-off is bandwidth-tight but
rings more; a larger one is gentler but wider.

## Splitting the filter: root-raised-cosine

Here is the elegant part. Rather than shaping the whole pulse at the transmitter, the
system splits the raised cosine into two **root-raised-cosine (RRC)** halves — one at the
transmitter, one at the receiver — so that:

```text
RRC(transmit)  *  RRC(receive)  =  full raised cosine  (no ISI)
```

That single choice buys two things at once:

- The **cascade** is a full raised cosine, so the **Nyquist no-ISI** condition holds at
  the decision instants.
- The receiver's RRC is a **matched filter** for the transmitted pulse — its impulse
  response is the pulse shape reversed — which maximizes SNR at the symbol centre.

## Why the matched filter wins

A matched filter correlates the incoming signal against the exact pulse it is looking
for. All of a symbol's energy piles up into one peak at the sampling instant, while
noise — uncorrelated with the pulse — does not accumulate the same way. No other linear
filter gives a better signal-to-noise ratio at that instant, which is why the RX RRC sits
right before the [symbol decision and timing loop](/learn/dsp/clock-and-symbol-recovery/).
Cleaner peaks mean cleaner symbols and a lower [bit error rate](/learn/dsp/snr-evm-and-ber/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the two RRC halves cascade into a raised cosine (no ISI) and the RX half is a matched filter." markdown="0">
  <p class="knowledge-check__q">Quick check: why is pulse shaping split into a root-raised-cosine at both ends?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To double the data rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">So their cascade is a no-ISI raised cosine and the receiver half is a matched filter</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To convert the signal to real-valued samples</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Square pulses waste bandwidth and cause **inter-symbol interference (ISI)**.
- A **raised-cosine** pulse meets the **Nyquist no-ISI criterion** — zero at every other symbol instant.
- Splitting it into **root-raised-cosine** halves at TX and RX yields no ISI *and* a matched filter.
- The **matched filter** maximizes **SNR** at the decision instant, lowering the error rate.

Next up: multiplying by an oscillator to shift a channel to baseband — mixing and downconversion.
