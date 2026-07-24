---
slug: fir-filters
title: FIR filters
description: Finite impulse response filters — taps, linear phase, and how to design a low-pass filter that isolates one channel from a wideband capture.
keywords: fir filter, finite impulse response, filter taps, linear phase, low pass filter, channel filter, dsp fir design, kaiser window filter
level: intermediate
status: full
prereq:
  - convolution-and-impulse-response
faq:
  - q: What does FIR stand for and mean?
    a: FIR is Finite Impulse Response. The filter's response to a single impulse lasts a finite number of samples — exactly the number of taps — then stops, because the output depends only on the current and past inputs, never on past outputs. This makes FIR filters stable and predictable.
  - q: Why are FIR filters used for isolating radio channels?
    a: FIR filters can have very sharp, precisely designed frequency responses and linear phase, meaning all frequencies are delayed equally so the signal's shape isn't distorted. That precise, distortion-free selectivity is exactly what you want to cut one narrow channel out of a wide capture before demodulating it.
---

# FIR filters

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **FIR (finite impulse response)** filter is convolution with a fixed set of
**taps** — its output depends only on current and past *inputs*, so its impulse
response has a finite length and it's always **stable**. FIR filters can be designed
with **sharp** responses and **linear phase** (no shape distortion), which makes them
the workhorse for **isolating one channel** from a wideband capture.
</div>

FIR filters are the most common filter in software radio. This lesson shows what they
are and how they carve one channel out of a crowded band.

## What makes a filter FIR

A **FIR** filter is exactly the tapped convolution from the last lesson:

```text
output[n] = tap0*x[n] + tap1*x[n-1] + ... + tapK*x[n-K]
```

The key property: the output depends only on the **input** samples, never on previous
*outputs*. So when the input impulse passes the last tap, the output goes silent —
the impulse response is **finite**, K+1 samples long. Because nothing feeds back, a
FIR filter can never run away or ring forever; it is **unconditionally stable**.

## Taps set the shape

The taps *are* the filter. Choose them well and you get a **low-pass** filter (keep
low frequencies, reject high), a **band-pass** (keep a middle band), or a **high-pass**.
More taps buy a sharper transition between "keep" and "reject" — at a higher compute
cost, since each output sample is that many multiply-adds.

Designers rarely pick taps by hand. A design method (often a windowed sinc, using the
[window functions](/learn/dsp/windows-and-leakage/) from Unit 2) turns a specification
— "flat below 6 kHz, at least 60 dB down above 8 kHz" — into the tap values.
GopherTrunk's channel filters, for instance, use a **Kaiser-windowed** design tuned
for a stopband more than 60 dB below the passband.

## Linear phase: keeping the shape

A FIR filter whose taps are symmetric has **linear phase**: every frequency is delayed
by the *same* amount of time. That matters because a digital signal's information is
in its precise shape — the transitions between symbols. A filter that delayed
different frequencies by different amounts would smear those transitions and cause
errors. Linear phase preserves the shape, which is why FIR is preferred right before
[demodulation](/learn/dsp/demodulation/).

## Isolating a channel

Here's the job FIR filters do constantly in a scanner. A capture is 2.4 MHz wide and
full of signals; your channel is 12.5 kHz somewhere in it. After
[mixing](/learn/dsp/mixing-and-downconversion/) that channel to zero, a low-pass FIR
keeps only the narrow band around zero and rejects everything else:

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 130" role="img" aria-label="A wide spectrum with several peaks; a low-pass filter response shown as a box selects only the central narrow region, and the output spectrum keeps just that one channel." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="60" x2="240" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <g stroke="currentColor" stroke-width="2"><line x1="60" y1="60" x2="60" y2="30"/><line x1="120" y1="60" x2="120" y2="20"/><line x1="180" y1="60" x2="180" y2="38"/></g>
  <rect x="100" y="15" width="40" height="45" fill="none" stroke="currentColor" stroke-dasharray="4 3"/>
  <text x="130" y="78" text-anchor="middle" font-size="9" fill="currentColor">keep this</text>
  <text x="130" y="12" text-anchor="middle" font-size="9" fill="currentColor">FIR passband</text>
  <text x="270" y="60" text-anchor="middle" font-size="16" fill="currentColor">&#8594;</text>
  <line x1="300" y1="60" x2="510" y2="60" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="405" y1="60" x2="405" y2="20" stroke="currentColor" stroke-width="2"/>
  <text x="405" y="78" text-anchor="middle" font-size="9" fill="currentColor">one clean channel</text>
</svg>
<figcaption>A low-pass FIR keeps the narrow band around zero and rejects the rest — one channel pulled cleanly out of a crowded capture.</figcaption>
</figure>

That same filter also does double duty as the **anti-aliasing** guard before the rate
is reduced — the subject of [decimation](/learn/dsp/decimation-and-resampling/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a FIR filter has no feedback, so it's always stable." markdown="0">
  <p class="knowledge-check__q">Quick check: why is a FIR filter always stable?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses very few taps</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Its output depends only on inputs, never on past outputs — no feedback</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It runs at a low sample rate</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **FIR** filter is convolution with fixed **taps**; output depends only on inputs,
  so it's **finite** and **stable**.
- The **taps** set the response — low-pass, band-pass, high-pass; more taps mean a
  sharper cutoff.
- Symmetric taps give **linear phase**, preserving a digital signal's shape.
- FIR filters **isolate a channel** and guard against aliasing before rate reduction.

Next up: IIR filters, which trade some of that predictability for efficiency.
