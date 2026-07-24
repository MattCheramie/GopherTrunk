---
slug: convolution-and-impulse-response
title: Convolution & impulse response
description: The operation at the heart of every filter — what an impulse response is, and how sliding-and-summing in time equals multiplying in the frequency domain.
keywords: convolution, impulse response, filter dsp, convolution explained, time domain filtering, impulse response filter, dsp convolution
level: advanced
status: full
prereq:
  - the-fourier-transform
faq:
  - q: What is convolution in simple terms?
    a: Convolution slides one short sequence (the filter) along a signal, and at each position multiplies the overlapping values and adds them up to produce one output sample. Repeat down the whole signal and you get the filtered result. It sounds abstract, but a moving average — replacing each sample with the average of it and its neighbours — is exactly a convolution.
  - q: What is an impulse response?
    a: A filter's impulse response is what comes out when you feed in a single spike (an impulse) and nothing else. Remarkably, that one response completely defines the filter — convolving any input with the impulse response produces the filtered output. So "the filter" and "its impulse response" are two names for the same thing.
---

# Convolution & impulse response

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Convolution** slides a short sequence along a signal, multiplying and summing the
overlap to produce each output sample — it's how filters work in the time domain. A
filter is fully described by its **impulse response** (what it outputs for a single
spike). And there's a deep shortcut: **convolution in time equals multiplication in
frequency**, tying filters back to the Fourier view.
</div>

Every filter you'll meet — FIR, IIR, the anti-alias filter in a downconverter — is
built on one operation. This lesson demystifies it.

## A moving average is a filter

Start with something familiar: smoothing a noisy signal by replacing each sample with
the average of it and its neighbours.

```text
input:   4  8  6  2  9  3
average each with its two neighbours (÷3):
output:     6  5.3  5.7  4.7 ...
```

You just **filtered** the signal — a low-pass filter, in fact, because averaging
smooths out fast wiggles (high frequencies) and keeps slow trends (low ones). Slide
the averaging window along, compute one output per position: that sliding-and-summing
*is* convolution.

## Convolution, generally

A general filter doesn't weight all neighbours equally — it uses a set of weights
called **taps** (or **coefficients**). At each position, line the taps up against the
signal, multiply each overlapping pair, and sum:

```text
output[n] = tap0*x[n] + tap1*x[n-1] + tap2*x[n-2] + ...
```

Different taps make different filters. Choosing them to keep the frequencies you want
and reject the rest is **filter design**, the subject of the next two lessons.

## The impulse response defines the filter

Here's the elegant part. Feed a filter a single **impulse** — one spike, `1, 0, 0, 0,
…` — and whatever comes out is its **impulse response**. For a tapped filter, the
output *is* the taps themselves.

That impulse response completely characterizes the filter: convolve *any* input with
it and you get the correctly filtered output. So the taps, the impulse response, and
"the filter" are all the same thing seen three ways.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A single input spike on the left passes through a filter box and emerges on the right as a spread-out impulse response shape." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="90" x2="120" y2="90" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="55" y1="90" x2="55" y2="35" stroke="currentColor" stroke-width="2"/>
  <text x="60" y="108" text-anchor="middle" font-size="9" fill="currentColor">impulse in</text>
  <rect x="150" y="55" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="195" y="76" text-anchor="middle" font-size="11" fill="currentColor">filter</text>
  <line x1="120" y1="72" x2="150" y2="72" stroke="currentColor" stroke-width="1.5" marker-end="url(#c1)"/>
  <line x1="240" y1="72" x2="270" y2="72" stroke="currentColor" stroke-width="1.5" marker-end="url(#c1)"/>
  <line x1="300" y1="90" x2="510" y2="90" stroke="currentColor" stroke-opacity="0.3"/>
  <g stroke="currentColor" stroke-width="2">
    <line x1="330" y1="90" x2="330" y2="70"/><line x1="355" y1="90" x2="355" y2="50"/><line x1="380" y1="90" x2="380" y2="40"/><line x1="405" y1="90" x2="405" y2="52"/><line x1="430" y1="90" x2="430" y2="72"/>
  </g>
  <text x="400" y="108" text-anchor="middle" font-size="9" fill="currentColor">impulse response out</text>
  <defs><marker id="c1" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Feed a filter one spike and its output — the impulse response — fully describes what the filter does to any signal.</figcaption>
</figure>

## The great shortcut

Recall from [the Fourier transform](/learn/dsp/the-fourier-transform/) that time and
frequency are two views of a signal. There's a beautiful theorem linking them:

> **Convolution in the time domain = multiplication in the frequency domain.**

Filtering a signal (convolving it with the taps) is the same as multiplying its
spectrum by the filter's frequency response. This is why filters are so often
described by their *frequency response* — "pass below 6 kHz, reject above" — even
though they *run* as convolution in time. It also means a filter's job is easiest to
picture in the frequency domain and easiest to *compute* in the time domain.

<div class="knowledge-check" data-quiz data-correct-msg="Right — convolution in time is multiplication in frequency, the link filters rely on." markdown="0">
  <p class="knowledge-check__q">Quick check: convolving a signal with a filter in the time domain is equivalent to what in the frequency domain?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Adding the two spectra</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Multiplying the two spectra</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — they're unrelated</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Convolution** slides a set of **taps** along a signal, multiplying and summing —
  how filters run in time.
- A moving average is the simplest example: a low-pass filter.
- A filter is fully defined by its **impulse response** (its output for a single
  spike).
- **Convolution in time = multiplication in frequency** — filters are designed in
  frequency, run in time.

Next up: FIR filters, the workhorse for isolating a channel.
