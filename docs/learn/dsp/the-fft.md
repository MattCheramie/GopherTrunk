---
slug: the-fft
title: The FFT in practice
description: How the Fast Fourier Transform makes the DFT computable in real time, what bin resolution means, the time-versus-frequency tradeoff, and what a waterfall display really shows.
keywords: fft, fast fourier transform, dft, fft bins, frequency resolution, waterfall display, fft size, spectrum analyzer
level: intermediate
status: full
prereq:
  - the-fourier-transform
faq:
  - q: What is an FFT bin?
    a: An FFT splits the sampled bandwidth into a fixed number of equal-width slots called bins, each reporting the energy in a small range of frequencies. The number of bins equals the FFT size. More bins over the same bandwidth means finer frequency resolution — but each FFT then needs more samples, which means it covers a longer slice of time.
  - q: What is a waterfall display?
    a: A waterfall stacks many FFTs computed over successive slices of time, drawing each as a horizontal line where colour or brightness shows energy per frequency bin, and scrolling downward. It lets you watch signals appear, disappear, and move across the spectrum — the standard way SDR software visualizes activity.
---

# The FFT in practice

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The **FFT (Fast Fourier Transform)** is a fast algorithm for the DFT, quick enough to
run live on a radio stream. It splits the bandwidth into **bins** — more bins mean
finer **frequency resolution**, but each FFT then spans more **time**. A **waterfall**
is just many FFTs stacked over time. The FFT is how software radio *sees* the
spectrum.
</div>

The Fourier transform is the idea; the FFT is how you actually compute it fast enough
to matter. This lesson covers the practical knobs and what they trade off.

## Fast enough to be useful

A direct DFT of N samples costs on the order of N² operations — hopeless for the
millions of samples a second an SDR produces. The **FFT** computes the identical
result in about N·log N operations. For a 4,096-point transform that's the difference
between ~16 million operations and ~50 thousand — the leap that makes real-time
spectrum analysis possible on ordinary hardware.

## Bins and resolution

An FFT of size N divides the sampled bandwidth into **N bins** of equal width:

```text
bin width  =  sample rate / FFT size
```

At 2.4 MS/s with a 4,096-point FFT, each bin is about 586 Hz wide. Want to resolve
signals closer together? Use more bins. But there's a catch built into physics.

## The time–frequency tradeoff

To get finer frequency resolution you need a **bigger** FFT, and a bigger FFT needs
**more samples**, which cover a **longer** stretch of time. So you can know *precisely
what* frequency or *precisely when* — but not both at once:

| FFT size (at 2.4 MS/s) | Bin width | Time span |
|------------------------|-----------|-----------|
| 1,024 | ~2.3 kHz | ~0.43 ms |
| 4,096 | ~586 Hz | ~1.7 ms |
| 16,384 | ~146 Hz | ~6.8 ms |

Fine frequency detail costs time resolution, and vice versa. Choosing the FFT size is
choosing where on that trade you want to sit — a short FFT to catch brief bursts, a
long one to separate close carriers.

## The waterfall

Compute an FFT, draw its bins as a row of colours (bright = strong), drop down a line,
compute the next FFT, and repeat. That scrolling picture is a **waterfall** — time
running down the screen, frequency across it.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A waterfall diagram: stacked horizontal rows representing successive FFTs, with vertical streaks marking signals persisting across time and frequency." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="10" width="480" height="90" fill="none" stroke="currentColor" stroke-opacity="0.4"/>
  <line x1="150" y1="10" x2="150" y2="100" stroke="currentColor" stroke-width="6" stroke-opacity="0.5"/>
  <line x1="300" y1="10" x2="300" y2="100" stroke="currentColor" stroke-width="3" stroke-opacity="0.8"/>
  <line x1="420" y1="40" x2="420" y2="100" stroke="currentColor" stroke-width="4" stroke-opacity="0.5"/>
  <text x="260" y="115" text-anchor="middle" font-size="10" fill="currentColor">frequency &#8594;</text>
  <text x="12" y="55" font-size="10" fill="currentColor" transform="rotate(-90 12 55)">time &#8595;</text>
</svg>
<figcaption>Each row is one FFT; each vertical streak is a signal holding a frequency over time. This is what SDR software shows you.</figcaption>
</figure>

The RF path's [FFT & waterfall](/learn/rf-sdr/fft-and-waterfall/) lesson looks at
reading these displays; here the point is that the picture is *literally* a stack of
FFTs, and its sharpness in frequency versus time is the tradeoff above. Windowing —
the next lesson — cleans up how each FFT row looks.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a larger FFT gives finer frequency resolution but spans more time." markdown="0">
  <p class="knowledge-check__q">Quick check: you double the FFT size at a fixed sample rate. What happens?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="correct">Finer frequency resolution, but each FFT covers more time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Coarser frequency resolution and less time</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing changes</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **FFT** computes the DFT in ~N·log N operations — fast enough for live radio.
- It splits bandwidth into **bins**; **bin width = sample rate ÷ FFT size**.
- Finer frequency resolution needs a **bigger FFT**, which spans **more time** — a
  fundamental tradeoff.
- A **waterfall** is many FFTs stacked over time — how SDR software shows the spectrum.

Next up: why real FFTs smear energy, and how window functions fix it.
