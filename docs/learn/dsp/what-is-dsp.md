---
slug: what-is-dsp
title: What is DSP?
description: Why we process signals with numbers instead of analog circuits, what a digital signal processor does, and where DSP sits inside a software-defined radio.
keywords: what is dsp, digital signal processing, dsp explained, dsp vs analog, software radio dsp, signal processing basics
level: beginner
status: full
faq:
  - q: What is digital signal processing in simple terms?
    a: Digital signal processing is doing to a signal — with arithmetic on numbers — what used to be done with analog electronic circuits. Once a signal is a stream of numbers, filtering, tuning, and decoding all become math a computer performs. That flexibility is what makes software-defined radio possible.
  - q: Do I need to be good at maths to learn DSP?
    a: Not to start. The core ideas are intuitive — averaging numbers to smooth a signal, sliding one list along another, looking at a signal by its frequencies. This module keeps the maths light and visual; you can understand the whole pipeline with arithmetic and pictures, and pick up the deeper formulas later if you want them.
---

# What is DSP?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Digital signal processing (DSP)** is manipulating a signal as a stream of
**numbers** using arithmetic, instead of with analog circuits. Filtering, tuning,
and demodulation all become math a computer runs. DSP is the engine inside a
**software-defined radio**: the hardware just digitizes the airwaves, and DSP turns
those numbers into voice, data, and control messages.
</div>

This is lesson 1 of the DSP path. Its job is to answer *what* DSP is and *why* it
matters before we get into the how. By the end you'll see where every later lesson —
sampling, filters, the FFT, demodulation — fits in the chain.

## From circuits to numbers

For most of radio's history, a signal was processed by physical circuits: a coil and
capacitor tuned to a station, a diode recovered the audio. Each function needed
dedicated hardware. **DSP replaces those circuits with arithmetic.** Once a signal is
a list of numbers, "tune to this frequency" or "keep only these components" is a
calculation, and one general-purpose processor can do the work of a rack of analog
gear.

That shift is the whole idea behind [software-defined radio](/learn/rf-sdr/what-is-sdr/):
the radio hardware does as little as possible — just convert the antenna's signal
into numbers — and *software* does the rest. Change the software and the same
hardware decodes a different system.

## What a signal looks like as numbers

A signal is just a value that changes over time — a voltage, a pressure, a radio
wave's amplitude. Sample it regularly and you get a sequence:

```text
time:    0    1    2    3    4    5    6   ...
value:  0.0  0.6  0.9  0.6  0.0 -0.6 -0.9  ...   (a sampled sine wave)
```

Everything DSP does operates on sequences like this. Smoothing noise is averaging
neighbouring values; tuning is multiplying by another sequence; detecting a tone is
asking how much of a given frequency is present. The next lesson,
[sampling & quantization](/learn/dsp/sampling-and-quantization/), covers exactly how
a continuous wave becomes this list.

## Where DSP sits in a software radio

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 90" role="img" aria-label="Signal chain from antenna to decoded data: antenna, analog-to-digital converter, DSP block, decoded output." xmlns="http://www.w3.org/2000/svg">
  <rect x="6" y="30" width="80" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="46" y="51" text-anchor="middle" font-size="11" fill="currentColor">antenna</text>
  <rect x="112" y="30" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="157" y="46" text-anchor="middle" font-size="10" fill="currentColor">ADC</text>
  <text x="157" y="58" text-anchor="middle" font-size="9" fill="currentColor">(digitize)</text>
  <rect x="228" y="30" width="120" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="288" y="46" text-anchor="middle" font-size="11" fill="currentColor">DSP</text>
  <text x="288" y="58" text-anchor="middle" font-size="9" fill="currentColor">filter, tune, demod</text>
  <rect x="374" y="30" width="120" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="434" y="51" text-anchor="middle" font-size="10" fill="currentColor">decoded data</text>
  <line x1="86" y1="47" x2="112" y2="47" stroke="currentColor" stroke-width="1.5"/>
  <line x1="202" y1="47" x2="228" y2="47" stroke="currentColor" stroke-width="1.5"/>
  <line x1="348" y1="47" x2="374" y2="47" stroke="currentColor" stroke-width="1.5"/>
</svg>
<figcaption>The radio front-end digitizes the airwaves; DSP — the whole rest of this path — turns those numbers into decoded data.</figcaption>
</figure>

Everything between "digitize" and "decoded data" is DSP, and it's what this module
builds up stage by stage. By [the final lesson](/learn/dsp/dsp-in-gophertrunk/) you'll
map each stage onto GopherTrunk's real code.

## Why numbers win

Processing in software buys you three things analog can't match easily:

- **Flexibility** — a new protocol is new code, not new hardware.
- **Precision** — arithmetic is exact and repeatable; components drift with heat and age.
- **Complexity for free** — filters and detectors that would need dozens of parts are
  a few lines of math.

The price is that you need enough compute to keep up with the sample rate — which is
why [performance](/learn/dsp/fixed-vs-floating-point/) is a real concern in DSP code.

<div class="knowledge-check" data-quiz data-correct-msg="Right — DSP works on a signal represented as a stream of numbers." markdown="0">
  <p class="knowledge-check__q">Quick check: what does DSP operate on?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Tuned analog circuits</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A signal represented as a stream of numbers</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Only pre-recorded audio files</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **DSP** manipulates a signal as a stream of **numbers** with arithmetic, replacing
  analog circuits.
- It's the engine of **software-defined radio**: hardware digitizes, software decodes.
- Every DSP operation is math on a **sequence of samples**.
- Numbers buy flexibility, precision, and complexity — at the cost of needing compute.

Next up: how a continuous wave actually becomes that stream of numbers.
