---
layout: learn-hub
learn_module: dsp
permalink: /learn/dsp/
title: Learn DSP — from radio samples to decoded bits
description: A free, structured path through the digital signal processing behind software radio — sampling, I/Q, the Fourier transform, filters, downconversion, demodulation, and clock recovery, mapped onto a real decoder.
keywords: learn dsp, digital signal processing tutorial, dsp for sdr, fft tutorial, fir filter, iir filter, downconversion, demodulation, clock recovery, i/q samples
---

Software-defined radio is really just digital signal processing wearing an
antenna. The [RF & SDR path](/learn/rf-sdr/) shows you what a radio wave *is*;
this path shows you what your computer *does* with it once the samples start
arriving — the arithmetic that turns a stream of numbers into voice, data, and
trunking control messages.

**Who this is for.** Anyone comfortable with the idea that an SDR hands you a
list of numbers and curious about what happens next. No advanced maths is
assumed — if you can follow "add these up" and "slide this along", you can
follow every lesson here. A little from the RF & SDR path helps, and we link
back to it constantly, but you can start cold. Already know the DFT and just
want the applied chain? Jump to [DSP in
GopherTrunk](/learn/dsp/dsp-in-gophertrunk/).

**How the path works.** Five units build the pipeline one stage at a time.
Unit 1 turns a wave into an array of numbers. Unit 2 shows you the frequency
domain — the single most useful way to look at a signal. Unit 3 is filters:
keeping the signal you want and discarding the rest. Unit 4 tunes,
demodulates, and recovers the data hidden in the wave. Unit 5 puts the whole
chain together the way a real decoder does and maps every stage onto
GopherTrunk's actual code. Mark lessons complete as you go — your progress is
saved in your browser. New here? **[Start with lesson 1: What is
DSP?](what-is-dsp/)**
