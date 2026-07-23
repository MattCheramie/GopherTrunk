---
slug: the-fourier-transform
title: The Fourier transform
description: The one idea that any signal is a sum of sine waves — what the Fourier transform computes, and why the frequency domain makes filtering and detection easy.
keywords: fourier transform, frequency domain, dft, sine wave decomposition, spectrum, fourier analysis, dsp frequency
level: intermediate
status: full
prereq:
  - complex-signals-and-iq
faq:
  - q: What does the Fourier transform actually do?
    a: It takes a signal described over time (a list of samples) and re-describes it over frequency — telling you how much of each frequency is present. The remarkable underlying fact is that any signal can be built by adding up sine waves of different frequencies, amplitudes, and phases, and the transform finds that recipe.
  - q: What is the difference between the time domain and the frequency domain?
    a: The time domain is the signal as it varies moment to moment — the raw samples. The frequency domain is the same signal expressed as its component frequencies. They contain the same information in two views; some operations (like removing an interfering tone) are trivial in one view and hard in the other.
---

# The Fourier transform

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **Fourier transform** re-describes a signal from the **time domain** (samples
over time) into the **frequency domain** (how much of each frequency is present). Its
foundation: **any signal is a sum of sine waves**. In the frequency domain, jobs like
isolating a channel or spotting a carrier become easy — which is why the transform is
DSP's most-used tool.
</div>

This lesson introduces the single most powerful idea in signal processing. Don't
worry about the equations — the *concept* is what unlocks everything after it.

## Any signal is a sum of sines

Here's the surprising fact underneath all of it: **every signal, however complicated,
can be built by adding together plain sine waves** of different frequencies,
strengths, and phases. A square wave is a sum of sines; a voice is a sum of sines; a
digital transmission is a sum of sines.

The **Fourier transform** runs that idea in reverse: given the signal, it works out
*which* sines, and *how much* of each, are in the mix. That recipe — amount of energy
at each frequency — is the signal's **spectrum**.

## Two views of the same thing

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="On the left, a complicated wiggly waveform in the time domain; an arrow labelled Fourier transform points to the right, where two sharp peaks appear on a frequency axis." xmlns="http://www.w3.org/2000/svg">
  <text x="90" y="18" text-anchor="middle" font-size="11" fill="currentColor">time domain</text>
  <path d="M10 75 C 40 40, 55 110, 85 70 S 130 30, 160 80 S 180 110, 200 75" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="10" y1="120" x2="200" y2="120" stroke="currentColor" stroke-opacity="0.3"/>
  <text x="105" y="135" text-anchor="middle" font-size="9" fill="currentColor">amplitude vs time</text>
  <line x1="225" y1="75" x2="300" y2="75" stroke="currentColor" stroke-width="1.5" marker-end="url(#f1)"/>
  <text x="262" y="66" text-anchor="middle" font-size="9" fill="currentColor">Fourier</text>
  <text x="410" y="18" text-anchor="middle" font-size="11" fill="currentColor">frequency domain</text>
  <line x1="320" y1="120" x2="510" y2="120" stroke="currentColor" stroke-opacity="0.3"/>
  <line x1="370" y1="120" x2="370" y2="55" stroke="currentColor" stroke-width="2"/>
  <line x1="450" y1="120" x2="450" y2="80" stroke="currentColor" stroke-width="2"/>
  <text x="415" y="135" text-anchor="middle" font-size="9" fill="currentColor">energy vs frequency</text>
  <defs><marker id="f1" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The same signal, two views. That messy waveform on the left is just two tones — obvious as two peaks on the right.</figcaption>
</figure>

The waveform and its spectrum hold the *same information* — but some questions are
trivial in one view and painful in the other. "Is there a carrier near 851 MHz?" is
almost impossible to eyeball in the time domain and blindingly obvious as a peak in
the frequency domain.

## Why radio lives in the frequency domain

Nearly every radio task is naturally a frequency question:

- **Finding a signal** — a control channel is a peak at a known frequency.
- **Isolating a channel** — keep a band of frequencies, discard the rest (that's
  [filtering](/learn/dsp/fir-filters/)).
- **Tuning** — shift the whole spectrum so your channel sits at zero (that's
  [mixing](/learn/dsp/mixing-and-downconversion/)).

There's even a deep shortcut the transform reveals: **filtering in the time domain
(convolution) is just multiplication in the frequency domain** — the link you'll meet
in [convolution & impulse response](/learn/dsp/convolution-and-impulse-response/).

## From idea to computation

The Fourier transform as pure math runs over a continuous signal. On sampled data we
use the **Discrete Fourier Transform (DFT)**, which takes a block of samples and
returns the energy in a set of frequency **bins**. Computed naively it's slow — but a
clever algorithm, the **[FFT](/learn/dsp/the-fft/)**, makes it fast enough to run in
real time on a live radio stream. That's the next lesson, and it's what powers the
waterfall display you may have seen.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the Fourier transform converts time-domain samples into a frequency-domain spectrum." markdown="0">
  <p class="knowledge-check__q">Quick check: the Fourier transform turns a signal described over time into one described over…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">amplitude</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">frequency</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">distance</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Any signal is a sum of sine waves**; the **Fourier transform** finds the recipe.
- It converts the **time domain** (samples) into the **frequency domain** (a
  **spectrum**).
- Radio tasks — finding, isolating, tuning — are naturally frequency questions.
- On sampled data we use the **DFT**, made fast by the **FFT** — coming next.

Next up: the FFT, and what a waterfall display is really showing you.
