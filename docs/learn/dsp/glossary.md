---
slug: glossary
title: Glossary of DSP terms
description: Plain-language definitions of digital-signal-processing terms — sample rate, Nyquist, I/Q, FFT, FIR, IIR, decimation, mixing, AGC, symbol recovery, and more — each linked to the lesson where it's explained.
keywords: dsp glossary, signal processing terms, dsp terminology, fft fir iir definition, iq nyquist decimation glossary, dsp dictionary
level: beginner
status: full
lesson_standalone: true
---

# Glossary of DSP terms

Every term used across the [DSP module](/learn/dsp/), defined in plain language and
linked to the lesson where it's explained in full. Skim it as a refresher, or use
your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are grouped by theme.

## Signals as numbers

**Digital signal processing (DSP)** — Manipulating a signal as a stream of numbers
with arithmetic, in place of analog circuits. See [What is DSP?](/learn/dsp/what-is-dsp/)

**Sample** — One measurement of a signal at an instant; DSP operates on sequences of
them. See [What is DSP?](/learn/dsp/what-is-dsp/)

**Sample rate** — How many samples are taken per second, setting the range of
frequencies that can be captured. See [Sampling & quantization](/learn/dsp/sampling-and-quantization/)

**Nyquist theorem** — You must sample above twice the highest frequency present, or
higher components alias. See [Sampling & quantization](/learn/dsp/sampling-and-quantization/)

**Aliasing** — The corruption where frequencies above half the sample rate fold down
and masquerade as lower ones. See [Sampling & quantization](/learn/dsp/sampling-and-quantization/)

**Quantization** — Rounding each sample to one of a fixed number of levels set by the
bit depth, adding quantization noise. See [Sampling & quantization](/learn/dsp/sampling-and-quantization/)

**Bit depth** — How many bits record each sample, setting amplitude precision. See
[Sampling & quantization](/learn/dsp/sampling-and-quantization/)

**I/Q (in-phase / quadrature)** — The two numbers per sample that form one complex
value carrying amplitude and phase. See [Complex signals & I/Q](/learn/dsp/complex-signals-and-iq/)

**Complex sample** — An I/Q pair treated as one number — an arrow whose length is
amplitude and angle is phase. See [Complex signals & I/Q](/learn/dsp/complex-signals-and-iq/)

**Baseband** — A signal centred on zero frequency, measured relative to the tuned
centre. See [Complex signals & I/Q](/learn/dsp/complex-signals-and-iq/)

**Negative frequency** — A component below the tuned centre; I/Q is what lets it be
told apart from one above. See [Complex signals & I/Q](/learn/dsp/complex-signals-and-iq/)

## The frequency domain

**Fourier transform** — Re-describes a signal from time to frequency, revealing which
sine waves it's made of. See [The Fourier transform](/learn/dsp/the-fourier-transform/)

**Time / frequency domain** — Two equivalent views of a signal: amplitude over time,
or energy over frequency. See [The Fourier transform](/learn/dsp/the-fourier-transform/)

**Spectrum** — The distribution of a signal's energy across frequency. See [The Fourier transform](/learn/dsp/the-fourier-transform/)

**DFT / FFT** — The Discrete Fourier Transform on sampled data, and the Fast algorithm
that computes it in ~N·log N. See [The FFT in practice](/learn/dsp/the-fft/)

**Bin** — One of the equal frequency slots an FFT divides the bandwidth into; bin width
= sample rate ÷ FFT size. See [The FFT in practice](/learn/dsp/the-fft/)

**Waterfall** — A stack of FFTs over time, showing frequency across and time down. See
[The FFT in practice](/learn/dsp/the-fft/)

**Spectral leakage** — A single tone smearing into neighbouring bins when it doesn't fit
a whole number of cycles in the FFT block. See [Windows & spectral leakage](/learn/dsp/windows-and-leakage/)

**Window function** — A taper applied to an FFT block to reduce leakage; Hann, Hamming,
Blackman, Kaiser. See [Windows & spectral leakage](/learn/dsp/windows-and-leakage/)

## Filters

**Convolution** — Sliding a filter's taps along a signal, multiplying and summing to
produce each output — how filters run in time. See [Convolution & impulse response](/learn/dsp/convolution-and-impulse-response/)

**Impulse response** — A filter's output for a single spike; it fully defines the
filter. See [Convolution & impulse response](/learn/dsp/convolution-and-impulse-response/)

**Taps / coefficients** — The weights that define a filter's response. See [FIR filters](/learn/dsp/fir-filters/)

**FIR (finite impulse response)** — A filter with no feedback — always stable, can have
linear phase; the channel-isolation workhorse. See [FIR filters](/learn/dsp/fir-filters/)

**Linear phase** — Delaying all frequencies equally, so a signal's shape isn't
distorted; a property of symmetric FIR filters. See [FIR filters](/learn/dsp/fir-filters/)

**IIR (infinite impulse response)** — A filter with feedback — sharp and cheap but can
be unstable and lacks linear phase. See [IIR filters](/learn/dsp/iir-filters/)

**Poles and zeros** — The feedback and feed-forward terms of an IIR filter; misplaced
poles cause instability. See [IIR filters](/learn/dsp/iir-filters/)

**Decimation** — Lowering the sample rate by filtering then keeping every Nth sample.
See [Decimation & resampling](/learn/dsp/decimation-and-resampling/)

**Interpolation** — Raising the sample rate by inserting and smoothing samples. See
[Decimation & resampling](/learn/dsp/decimation-and-resampling/)

**Resampling** — Changing the sample rate by a rational factor (interpolate then
decimate), often via a polyphase filter. See [Decimation & resampling](/learn/dsp/decimation-and-resampling/)

## From signal to bits

**Mixing** — Multiplying by an oscillator's sinusoid to shift the spectrum in
frequency. See [Mixing & downconversion](/learn/dsp/mixing-and-downconversion/)

**NCO (numerically controlled oscillator)** — Software that generates a complex sine
wave at a set frequency — the digital local oscillator. See [Mixing & downconversion](/learn/dsp/mixing-and-downconversion/)

**Downconversion** — Mixing a channel down to zero (baseband). See [Mixing & downconversion](/learn/dsp/mixing-and-downconversion/)

**DDC (digital downconverter)** — Mix + filter + decimate combined into one channel-
extraction stage. See [Mixing & downconversion](/learn/dsp/mixing-and-downconversion/)

**Demodulation** — Recovering the message from a carrier; AM = magnitude, FM = phase-
change rate, PM = angle. See [Demodulation in code](/learn/dsp/demodulation/)

**Discriminator** — The block that FM-demodulates by measuring phase change between
samples. See [Demodulation in code](/learn/dsp/demodulation/)

**C4FM** — The four-level FM used by P25 Phase 1; each symbol carries two bits at 4800
symbols/second. See [Demodulation in code](/learn/dsp/demodulation/)

**Symbol** — One transmitted unit carrying one or more bits, read at its centre. See
[Clock & symbol recovery](/learn/dsp/clock-and-symbol-recovery/)

**Clock / symbol recovery** — Finding where each symbol begins without a shared clock,
via a timing loop. See [Clock & symbol recovery](/learn/dsp/clock-and-symbol-recovery/)

**Timing error detector** — Measures whether the current sampling instant is early or
late. See [Clock & symbol recovery](/learn/dsp/clock-and-symbol-recovery/)

**Matched filter** — A filter shaped to the transmitted pulse that maximizes SNR at
symbol centres. See [Clock & symbol recovery](/learn/dsp/clock-and-symbol-recovery/)

**AGC (automatic gain control)** — A feedback loop that holds a signal's amplitude near
a target as it fades. See [Gain & automatic gain control](/learn/dsp/gain-and-agc/)

## In practice

**Channel rate** — The fixed sample rate a DDC resamples every channel to (48 kHz C4FM,
144 kHz TETRA in GopherTrunk). See [DSP in GopherTrunk](/learn/dsp/dsp-in-gophertrunk/)

**Rate-invariance** — The property that a decoder behaves the same at any capture rate
because the DDC normalizes to the channel rate. See [DSP in GopherTrunk](/learn/dsp/dsp-in-gophertrunk/)

**Fixed point / floating point** — Two ways to store numbers; GopherTrunk uses 32-bit
float `complex64` throughout. See [Fixed vs floating point & performance](/learn/dsp/fixed-vs-floating-point/)

**`complex64`** — Go's 64-bit complex type (a pair of 32-bit floats) carrying I/Q
through GopherTrunk. See [Fixed vs floating point & performance](/learn/dsp/fixed-vs-floating-point/)
