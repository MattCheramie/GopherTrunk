---
slug: windows-and-leakage
title: Windows & spectral leakage
description: Why chopping a signal into blocks smears its energy across FFT bins, and how window functions trade frequency resolution for cleaner, lower-leakage spectra.
keywords: spectral leakage, window function, hann window, hamming window, fft windowing, sidelobes, dsp windows, blackman window
level: advanced
status: full
prereq:
  - the-fft
faq:
  - q: What causes spectral leakage?
    a: An FFT assumes the block of samples it's given repeats forever. If the signal doesn't complete a whole number of cycles in that block, the ends don't line up, creating an artificial jump when the block "wraps". That discontinuity spreads the signal's energy into neighbouring bins — leakage — smearing a single tone across the spectrum.
  - q: Which window function should I use?
    a: It depends on the goal. A Hann or Hamming window is a good general default, balancing leakage suppression against resolution. A Blackman or Kaiser window suppresses leakage more strongly at the cost of a wider main peak — useful when a strong signal would otherwise drown a weak one nearby. A rectangular window (no window) gives the sharpest peak but the worst leakage.
---

# Windows & spectral leakage

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An FFT assumes its block of samples **repeats forever**; if the signal doesn't fit a
whole number of cycles, the mismatched ends create a jump that **leaks** energy into
neighbouring bins. A **window function** tapers the block's edges to zero, greatly
reducing leakage — at the cost of a slightly wider main peak. It's the standard
cleanup applied before every FFT.
</div>

The FFT is powerful but has a sharp edge — literally. This lesson explains the
artifact called leakage and the window functions that tame it.

## The hidden assumption

An FFT doesn't see your block of samples as a fragment; it treats it as **one period
of a signal that repeats forever**. If the tone in your block completes a whole
number of cycles, the end lines up perfectly with the start when it "wraps," and the
FFT shows a clean, single spike.

But real signals rarely land on a whole number of cycles per block. Then the end and
the start *don't* match, and the wrap-around creates an abrupt **discontinuity** — a
jump that was never in the real signal.

## Leakage: one tone, many bins

That artificial jump is sharp, and sharp edges contain lots of frequencies. So a
single pure tone that *should* light up one bin instead **smears** its energy across
many neighbouring bins. This is **spectral leakage**:

```text
Ideal (fits the block):   ......▁█▁......      one clean bin
Leaky (doesn't fit):    ..▁▂▃▅█▅▃▂▁..         energy spread across bins
```

Leakage is a problem because it can bury a weak signal sitting next to a strong one:
the strong signal's leakage floods the weak one's bins.

## Windowing: taper the edges

The fix is a **window function** — a smooth curve you multiply the block by before the
FFT. It's near 1 in the middle and tapers to 0 at both ends, so the block always
starts and ends at zero and there's no discontinuity to leak from.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A rectangular block with hard edges on the left, and the same block multiplied by a smooth bell-shaped window that tapers to zero at both ends on the right." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="150" height="60" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="95" y="108" text-anchor="middle" font-size="10" fill="currentColor">rectangular (hard edges)</text>
  <text x="255" y="64" text-anchor="middle" font-size="16" fill="currentColor">&#8594;</text>
  <path d="M320 90 C 360 90, 370 30, 415 30 C 460 30, 470 90, 510 90" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <line x1="320" y1="90" x2="510" y2="90" stroke="currentColor" stroke-opacity="0.3"/>
  <text x="415" y="108" text-anchor="middle" font-size="10" fill="currentColor">windowed (tapered to zero)</text>
</svg>
<figcaption>A window tapers the block's ends to zero, removing the wrap-around jump that causes leakage.</figcaption>
</figure>

## Choosing a window

Windows trade leakage suppression against how wide the main peak becomes:

| Window | Leakage suppression | Main peak | Good for |
|--------|--------------------|-----------|----------|
| Rectangular (none) | poor | narrowest | closely spaced equal signals |
| Hann / Hamming | good | moderate | general use — a solid default |
| Blackman / Kaiser | excellent | widest | a weak signal beside a strong one |

There's no free lunch: killing leakage widens the peak, blurring nearby frequencies a
little. GopherTrunk's own filter and channelization design uses a **Kaiser** window
(with a shaping parameter β) precisely because it can push leakage far below the noise
floor when isolating one channel from a crowded band — a design choice you'll see in
[FIR filters](/learn/dsp/fir-filters/) and again in
[DSP in GopherTrunk](/learn/dsp/dsp-in-gophertrunk/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a window tapers the block edges to zero, removing the discontinuity that leaks." markdown="0">
  <p class="knowledge-check__q">Quick check: what does applying a window function to an FFT block reduce?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The sample rate</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Spectral leakage into neighbouring bins</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Quantization noise</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An FFT assumes its block **repeats forever**; a signal that doesn't fit creates a
  discontinuity.
- That jump causes **spectral leakage** — one tone smearing across many bins.
- A **window function** tapers the block edges to zero, cutting leakage.
- Windows trade leakage for peak width; **Hann/Hamming** are good defaults,
  **Kaiser/Blackman** for weak-beside-strong.

Next up: Unit 3 and the operation at the heart of every filter — convolution.
