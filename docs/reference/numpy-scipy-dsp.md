---
slug: numpy-scipy-dsp
title: NumPy & SciPy for DSP
entry_type: technology
category: sdr-frameworks
description: "NumPy and SciPy give Python vectorized arrays and scipy.signal DSP routines for offline SDR work: filtering, FFT spectra, resampling, and IQ-file analysis."
keywords: NumPy, SciPy, scipy.signal, Python DSP, offline SDR, complex64 IQ, FFT, filter design, firwin, resample, IQ file analysis
aka: [NumPy, SciPy, numpy scipy dsp, scipy.signal]
autolink: true
infobox:
  - { label: Type, value: Python numerical / DSP libraries }
  - { label: Idea, value: Vectorized arrays + scipy.signal for offline DSP }
  - { label: Examples, value: IQ-file analysis, filter design, FFT spectra }
see_also: [python-language, fast-fourier-transform, fir-filter, iq-file-format, iq-data, software-defined-radio]
cite_urls:
  - https://numpy.org/doc/stable/
  - https://docs.scipy.org/doc/scipy/reference/signal.html
---

**NumPy** and **SciPy** are the pair of Python libraries that make the language a practical
workbench for signal processing: NumPy supplies fast vectorized N-dimensional arrays (including
complex types), and SciPy's `scipy.signal` and `scipy.fft` modules add ready-made DSP
routines — filter design, convolution, [FFTs](/reference/fast-fourier-transform/), resampling,
and spectral estimation.[^np][^sp] Together they are the default tools for *offline*
[software-defined-radio](/reference/software-defined-radio/) work: loading a recorded
[IQ file](/reference/iq-file-format/), inspecting it, prototyping an algorithm, and checking the
result — all in a few lines of [Python](/reference/python-language/).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="An offline SDR workflow in Python: an IQ file is read with NumPy fromfile into a complex64 array, processed by scipy.signal filtering and an FFT, and the resulting spectrum is plotted." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="npar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="14" y="50" width="80" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="54" y="65">IQ file</text><text x="54" y="76">(complex64)</text>
    <rect x="118" y="50" width="90" height="34" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="163" y="65">np.fromfile</text><text x="163" y="76">→ ndarray</text>
    <rect x="232" y="50" width="100" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="282" y="65">scipy.signal</text><text x="282" y="76">filter / FFT</text>
    <rect x="356" y="50" width="90" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="401" y="65">spectrum /</text><text x="401" y="76">plot</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="94" y1="67" x2="118" y2="67" marker-end="url(#npar)"/>
    <line x1="208" y1="67" x2="232" y2="67" marker-end="url(#npar)"/>
    <line x1="332" y1="67" x2="356" y2="67" marker-end="url(#npar)"/>
  </g>
  <text x="230" y="115" font-size="8" fill="currentColor" text-anchor="middle">whole arrays processed at once — no per-sample Python loop</text>
</svg>
<figcaption>A typical offline SDR session: NumPy reads an IQ recording into a complex array, scipy.signal filters and transforms it, and the spectrum is plotted — all on whole arrays, avoiding slow per-sample loops.</figcaption>
</figure>

## How it works

The foundation is NumPy's `ndarray`: a contiguous, typed buffer that operations run over as a
whole. Crucially for SDR it has native **complex** types — `complex64` (two 32-bit floats) matches
the interleaved float IQ that most SDR tools write, so `np.fromfile(path, dtype=np.complex64)`
loads an entire recording into a baseband array in one call. Arithmetic, mixing (multiplying by a
complex exponential to shift frequency), and magnitude/phase all become vectorized expressions,
and the interpreter overhead of Python's per-element loops is avoided because the work happens in
compiled C and BLAS underneath.

SciPy layers the DSP algorithms on top. `scipy.signal` designs and applies filters (`firwin`,
`butter`, `lfilter`, `sosfilt`, `filtfilt` — see [FIR filters](/reference/fir-filter/)), resamples
(`resample`, `resample_poly`, `decimate`), correlates, and estimates spectra (`welch`,
`spectrogram`); `scipy.fft` (and `numpy.fft`) provide the [FFT](/reference/fast-fourier-transform/)
for turning a block of samples into a spectrum. A handful of these calls reproduces the core of a
receive chain — decimate, shift, filter, demodulate — well enough to validate an idea before it is
reimplemented in a fast language.

## In practice

The everyday use is analysis and prototyping, not real-time reception. Paired with Matplotlib for
plotting (and often Jupyter notebooks), NumPy/SciPy let you open a capture, plot its spectrum and
[spectrogram](/reference/fast-fourier-transform/), measure SNR, design a filter and see its response,
try a demodulator on a slice of samples, and generate reference test signals. The main caveats are
speed and streaming: pure-Python/NumPy is fine for megabyte-scale files but not for sustained
real-time multi-megasample throughput, and the array model wants the whole signal in memory rather
than an endless stream — which is exactly why production receivers are written in C, C++, Rust, or Go.

## Relevance to SDR

NumPy and SciPy are the lingua franca for *reasoning about* SDR signals: nearly every tutorial,
research paper, and algorithm prototype in the field is expressed in them, and they are the standard
way to generate the golden reference vectors against which a fast implementation is checked. Their
role is upstream of the receiver — design and verify here, then port to a real-time language.

That upstream/downstream split is exactly GopherTrunk's relationship to them. GT is the real-time
receiver written in Go for a single dependency-free binary, so it does not run NumPy or SciPy in its
decode path. But the *analysis* around GT lives comfortably in this world: a recorded
[IQ file](/reference/iq-file-format/) that GT replays can be loaded with `np.fromfile` and examined
with `scipy.signal` to measure SNR and EVM, confirm a channel's center offset, or independently
resample a capture — the kind of cross-check GT's own DSP notes rely on when deciding whether a
decode failure lives in the samples or in the code.

## Sources

[^np]: [NumPy documentation](https://numpy.org/doc/stable/) — the ndarray, complex dtypes, and FFT routines used for array-based signal processing.
[^sp]: [scipy.signal reference](https://docs.scipy.org/doc/scipy/reference/signal.html) — SciPy, the filter-design, resampling, correlation, and spectral-estimation routines for DSP.
