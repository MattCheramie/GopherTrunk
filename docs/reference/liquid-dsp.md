---
slug: liquid-dsp
title: liquid-dsp
entry_type: technology
category: sdr-frameworks
description: "liquid-dsp is a portable, dependency-light C library of DSP building blocks for software radio — filters, modems, resamplers, FEC, and synchronizers with a plain C API."
keywords: liquid-dsp, DSP library, C DSP, digital signal processing, software radio library, FIR filter, resampler, modem, forward error correction, Joseph Gaeddert
aka: [liquid-dsp, liquid]
autolink: true
infobox:
  - { label: Type, value: Portable C DSP library }
  - { label: Idea, value: Reusable software-radio primitives }
  - { label: Deps, value: Minimal (optional FFTW) }
see_also: [fftw, gnuradio, fir-filter, volk, resampler, root-raised-cosine-filter]
cite_urls:
  - https://liquidsdr.org/
  - https://github.com/jgaeddert/liquid-dsp
---

**liquid-dsp** is a portable, dependency-light C library of digital-signal-processing
primitives aimed at [software-defined radio](/reference/software-defined-radio/): filters,
modulators and demodulators, resamplers, [forward error correction](/reference/forward-error-correction/),
and synchronization loops, all exposed through a plain C API.[^liquid] Its goal is to give
radio developers a toolbox of proven, self-contained components so they can build a modem or a
decoder without reimplementing the same FIR filter, symbol synchronizer, or Reed-Solomon coder
from scratch. It compiles almost anywhere with a C compiler and has no mandatory external
dependencies.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 140" role="img" aria-label="A receive chain assembled from liquid-dsp objects: a resampler feeds an FIR filter, then a symbol synchronizer, then a demodulator, then a forward-error-correction decoder producing bytes." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="lqar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="6" y="50" width="76" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="44" y="62">resampler</text><text x="44" y="72">(msresamp)</text>
    <rect x="98" y="50" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="133" y="62">FIR filter</text><text x="133" y="72">(firfilt)</text>
    <rect x="184" y="50" width="76" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="222" y="62">symsync</text><text x="222" y="72">(timing)</text>
    <rect x="276" y="50" width="70" height="30" rx="5" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="311" y="62">demod</text><text x="311" y="72">(modemcf)</text>
    <rect x="362" y="50" width="86" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="405" y="62">FEC decode</text><text x="405" y="72">→ bytes</text>
  </g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="82" y1="65" x2="96" y2="65" marker-end="url(#lqar)"/>
    <line x1="168" y1="65" x2="182" y2="65" marker-end="url(#lqar)"/>
    <line x1="260" y1="65" x2="274" y2="65" marker-end="url(#lqar)"/>
    <line x1="346" y1="65" x2="360" y2="65" marker-end="url(#lqar)"/>
  </g>
  <text x="230" y="28" text-anchor="middle" font-size="9" fill="currentColor">a receiver assembled from liquid-dsp objects</text>
</svg>
<figcaption>liquid-dsp provides the reusable pieces of a receiver — resampling, filtering, timing recovery, demodulation, and error correction — each a small self-contained C object.</figcaption>
</figure>

## How it works

liquid-dsp is structured as **modules**, each a family of objects created, run, and destroyed
through a consistent C interface (`_create`, an execute call, `_destroy`). A program instantiates
the pieces it needs and feeds samples through them. The catalogue spans the whole receive and
transmit path:

- **Filtering** — [FIR](/reference/fir-filter/) and IIR filters, arbitrary-rate
  [resamplers](/reference/resampler/), polyphase filter banks, and pulse-shaping filters
  including the [root-raised-cosine](/reference/root-raised-cosine-filter/) used by most digital
  radios.
- **Modems** — linear modulators/demodulators (PSK, QAM, ASK, APSK) plus continuous-phase and
  analog schemes, all sharing one `modem` object interface.
- **Synchronization** — symbol-timing recovery, carrier phase/frequency loops, frame detectors,
  and equalizers, the components that let a receiver lock onto a real signal.
- **Coding** — [forward error correction](/reference/forward-error-correction/) (convolutional,
  Reed-Solomon, Hamming), interleaving, CRCs, and scramblers.
- **Math and utilities** — windowing, FFT (using [FFTW](/reference/fftw/) when available, an
  internal transform otherwise), complex-number helpers, and a fixed-point option.

The design emphasizes portability and independence: the core builds with just a C toolchain, and
[FFTW](/reference/fftw/) is an optional accelerator rather than a requirement. Because the objects
are small and orthogonal, developers compose them freely — the library supplies the DSP
primitives and leaves the protocol logic to the application.

## Relevance to SDR

liquid-dsp is a common foundation for custom SDR modems and decoders where pulling in a full
framework would be too heavy. It powers homebrew data links, experimental waveforms, and the DSP
cores of applications that want C-level control without writing every filter by hand. It also
appears inside larger ecosystems — for example as the basis of a GNU Radio out-of-tree module —
so a [GNU Radio](/reference/gnuradio/) flowgraph can call liquid's synchronizers and coders. Its
niche is complementary to a SIMD kernel library like [VOLK](/reference/volk/): liquid-dsp supplies
whole DSP *objects* (a symbol synchronizer, a Reed-Solomon coder), while VOLK optimizes the
vector math those objects run on.

**GopherTrunk** does not link liquid-dsp. Being a pure-Go project, GopherTrunk implements its
resamplers, matched filters, timing and carrier recovery, and FEC in Go so the whole decoder
stays a single static binary with no C dependency. liquid-dsp is nonetheless the closest
open-source analogue to what GopherTrunk's DSP layer *is* — a curated set of software-radio
primitives — and it is a valuable reference when validating a Go implementation against a
well-established C one, or when prototyping a receiver structure before porting it.

## Sources

[^liquid]: [liquidsdr.org](https://liquidsdr.org/) — the liquid-dsp project site and API documentation, describing the module catalogue (filters, resamplers, modems, synchronizers, FEC), the object lifecycle, and the optional FFTW dependency.
