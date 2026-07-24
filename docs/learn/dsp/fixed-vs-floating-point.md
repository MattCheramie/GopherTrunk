---
slug: fixed-vs-floating-point
title: Fixed vs floating point & performance
description: Why real-time DSP cares how numbers are stored, the tradeoffs of fixed versus floating point, and simple ways to keep a sample pipeline fast.
keywords: fixed point, floating point, dsp performance, real-time dsp, complex64, simd, dsp optimization, number representation
level: advanced
status: full
prereq:
  - dsp-in-gophertrunk
faq:
  - q: What is the difference between fixed-point and floating-point numbers?
    a: Floating point stores a number with a mantissa and an exponent, so it covers a huge range of magnitudes with automatic scaling — easy to use but historically slower on small processors. Fixed point stores numbers as scaled integers with an implied decimal position, which is fast and compact but requires the programmer to manage scaling and guard against overflow.
  - q: Why does performance matter so much in DSP?
    a: DSP runs the same operations on a relentless stream of samples — millions per second — in real time. If processing can't keep up with the sample rate, buffers overflow and data is lost. So even small per-sample costs multiply enormously, and choices like number format, buffer reuse, and avoiding needless work directly decide whether a decoder keeps up.
---

# Fixed vs floating point & performance

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
DSP runs the same math on **millions of samples a second**, so how numbers are stored
matters. **Floating point** is easy and accurate with automatic scaling;
**fixed point** (scaled integers) is compact and fast on small hardware but needs
manual scaling and overflow care. Beyond the format, **keeping up with the sample
rate** — reusing buffers, decimating early, avoiding waste — is what makes a real-time
pipeline work.
</div>

This last lesson steps back from *what* the DSP does to *how fast* it must do it — the
practical concern that shapes real decoder code.

## Two ways to store a number

| | Floating point | Fixed point |
|--|----------------|-------------|
| Stores | mantissa + exponent | scaled integer |
| Range | huge, auto-scaled | fixed, manual scaling |
| Ease of use | simple, forgiving | error-prone (overflow) |
| Speed | fast on modern CPUs/GPUs | fast on tiny/embedded hardware |
| Typical home | desktops, servers, SDR software | DSP chips, microcontrollers, FPGAs |

**Floating point** carries a scale factor with every value, so you rarely think about
magnitudes — a filter's output just works whether the signal is tiny or huge.
**Fixed point** stores values as integers with an agreed decimal position; it's leaner
and was historically much faster on cheap processors, but you must scale carefully or a
sum **overflows** and wraps into garbage.

## What GopherTrunk uses

GopherTrunk is **pure Go** and carries I/Q as **`complex64`** — a pair of 32-bit floats
— through the whole pipeline. Floating point keeps the DSP code clean and accurate, and
a modern CPU runs 32-bit float math plenty fast for the channel rates involved (tens to
hundreds of kilohertz per channel after decimation). On a tiny embedded DSP chip you'd
more likely see fixed point; in software radio on general-purpose CPUs, float wins on
simplicity with no real speed penalty.

## Keeping up with the stream

Number format aside, the dominant performance rule in DSP is blunt: **you must process
each sample before the next batch arrives.** Fall behind and buffers overflow and
samples are dropped. A few habits keep a pipeline ahead:

- **Decimate as early as possible.** Every stage after
  [downconversion](/learn/dsp/decimation-and-resampling/) runs at the low channel rate,
  not the wide capture rate — the single biggest saving.
- **Reuse buffers.** Allocating a new slice per batch creates garbage-collection
  pressure; reusing a fixed buffer avoids it (a real concern in
  [Go](/learn/programming-go/slices-maps-generics/)).
- **Do less per sample.** Precompute filter taps and NCO tables once; don't recompute
  constants in the inner loop.
- **Exploit the hardware.** Modern CPUs process several floats at once (SIMD); tight,
  regular loops let the compiler use it.

## The big picture

You've now followed a signal from a radio wave all the way to bits, and seen the
engineering that makes it run in real time. Pair this with the
[RF & SDR module](/learn/rf-sdr/) for the physics, the
[digital trunking module](/learn/digital-trunking/) for what the bits *mean*, and the
[Go module](/learn/programming-go/) to read and write the code — together they are
everything GopherTrunk is built from. Keep the
[glossary](/learn/dsp/glossary/) handy as you explore the source.

<div class="knowledge-check" data-quiz data-correct-msg="Right — decimating early means every later stage runs at the low channel rate." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the single biggest way to keep a real-time DSP pipeline fast?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Use a higher bit depth</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Decimate to the channel rate early, so later stages run on far fewer samples</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Compute the FFT more often</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- DSP runs on **millions of samples a second**, so number storage and speed matter.
- **Floating point** is simple and accurate; **fixed point** is compact and fast on
  small hardware but needs manual scaling.
- GopherTrunk uses **`complex64`** floats — simple and fast enough on modern CPUs.
- The biggest performance lever is **decimating early**; also reuse buffers and do less
  per sample.

That's the module — from a stream of samples to decoded bits, and the code that runs it
in real time. Keep exploring with the [glossary](/learn/dsp/glossary/).
