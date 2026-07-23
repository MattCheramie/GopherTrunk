---
slug: spectrograms-and-stft
title: Spectrograms & the STFT
description: The short-time Fourier transform — sliding the FFT along a signal to see how its spectrum changes over time, the math behind every waterfall display.
keywords: spectrogram, stft, short-time fourier transform, waterfall, time-frequency, sliding fft, spectral display, hop size overlap
level: intermediate
status: full
prereq:
  - the-fft
  - windows-and-leakage
faq:
  - q: What is the difference between an FFT and an STFT?
    a: "A single FFT gives you one spectrum for a whole block of samples, with no sense of when each frequency occurred. The short-time Fourier transform runs many FFTs, one on each short window as it slides along the signal, producing a spectrum for every moment. Stack those spectra and you can see how the signal's frequency content evolves over time."
  - q: Is a waterfall display a spectrogram?
    a: "Yes. A waterfall is a spectrogram drawn in real time: each new row is one FFT of the most recent block of samples, colour-mapped by magnitude and scrolled down the screen. Reading it means reading a spectrogram — bright vertical lines are steady carriers, sloping streaks are drifting or chirping signals."
---

# Spectrograms & the STFT

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A single [FFT](/learn/dsp/the-fft/) tells you *which* frequencies are present but not
*when*. The **short-time Fourier transform (STFT)** fixes that by running an FFT on each
short **window** as it slides along the signal, giving one spectrum per moment. Stack
those spectra into a **spectrogram** — time on one axis, frequency on the other,
magnitude as brightness — and you get the **waterfall** every SDR shows.
</div>

The FFT gave us a snapshot of a signal's frequencies. But radio is alive — carriers come
and go, signals drift and chirp. To see *change*, we slide the FFT along time. This
lesson builds on [the FFT](/learn/dsp/the-fft/) and
[windows & leakage](/learn/dsp/windows-and-leakage/), and explains the display from
[FFT & waterfall](/learn/rf-sdr/fft-and-waterfall/).

## Why one FFT isn't enough

Take an FFT of a ten-second recording and you learn every frequency that appeared —
but a carrier that switched on for one second looks the same as one that was on the
whole time. All timing information is smeared into a single spectrum. For a live band
where signals appear, transmit, and vanish, that is nearly useless. We need a spectrum
*for each moment*.

## The short-time Fourier transform

The fix is simple in concept: don't transform the whole signal at once. Chop it into
short, overlapping windows and take an FFT of each one.

```text
signal:  [====================================================]
window1: [======]                    -> FFT -> spectrum @ t1
window2:     [======]                -> FFT -> spectrum @ t2
window3:         [======]            -> FFT -> spectrum @ t3
   ...   (slide by the hop size)         ...
```

Each FFT is windowed first (a Hann or Kaiser taper from the
[leakage lesson](/learn/dsp/windows-and-leakage/)) to keep its spectrum clean. The step
between successive windows is the **hop size**; windows usually **overlap** so nothing
between them is missed. The result is a two-dimensional array: frequency down one axis,
time along the other.

## Reading a spectrogram

Colour-map each spectrum's magnitude — dark for quiet, bright for strong — lay the
columns side by side, and you have a **spectrogram**.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 170" role="img" aria-label="A spectrogram with time on the horizontal axis and frequency on the vertical axis, showing a steady horizontal carrier line and a diagonal chirping streak." xmlns="http://www.w3.org/2000/svg">
  <rect x="40" y="15" width="440" height="120" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <text x="260" y="158" text-anchor="middle" font-size="11" fill="currentColor">time &#8594;</text>
  <text x="24" y="78" font-size="11" fill="currentColor" transform="rotate(-90 24 78)">frequency</text>
  <line x1="40" y1="55" x2="480" y2="55" stroke="currentColor" stroke-width="3" stroke-opacity="0.85"/>
  <text x="130" y="49" font-size="9" fill="currentColor">steady carrier</text>
  <line x1="90" y1="125" x2="300" y2="30" stroke="currentColor" stroke-width="2.5" stroke-opacity="0.7"/>
  <text x="300" y="28" font-size="9" fill="currentColor">chirp / drift</text>
  <rect x="330" y="95" width="70" height="18" fill="currentColor" fill-opacity="0.35"/>
  <text x="365" y="128" text-anchor="middle" font-size="9" fill="currentColor">burst</text>
</svg>
<figcaption>A spectrogram: horizontal lines are steady carriers, sloping streaks are drifting or chirping signals, and rectangles are bursts that start and stop.</figcaption>
</figure>

A **waterfall** is exactly this, drawn live: each incoming block becomes one FFT column,
scrolled across the display. Reading it is reading a spectrogram.

## The resolution tradeoff, again

The STFT can't escape the [time-vs-frequency tradeoff](/learn/dsp/the-fft/). A **long**
window gives fine frequency resolution but blurs *when* things happened; a **short**
window pinpoints timing but smears frequency. Choosing the window length is choosing
where on that tradeoff you want to sit — wide for spotting a fleeting burst, narrow for
separating two close carriers.

| Window length | Frequency detail | Time detail | Good for |
|---------------|------------------|-------------|----------|
| Long | fine | coarse | separating nearby carriers |
| Short | coarse | fine | catching brief bursts, fast changes |

<div class="knowledge-check" data-quiz data-correct-msg="Right — the STFT runs an FFT on each sliding window to show how the spectrum changes over time." markdown="0">
  <p class="knowledge-check__q">Quick check: what does the STFT add over a single FFT?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Higher amplitude resolution</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A spectrum for each moment, showing how frequencies change over time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Removal of quantization noise</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A single **FFT** shows which frequencies but not **when**.
- The **STFT** runs an FFT on each sliding, overlapping **window** — one spectrum per moment.
- Stacked and colour-mapped, those spectra form a **spectrogram**; drawn live it's a **waterfall**.
- Window length sets the **time-vs-frequency** tradeoff: long for detail, short for speed.

Next up: how a real signal becomes complex I/Q in the first place — the analytic signal.
