---
slug: csdr
title: csdr
entry_type: technology
category: sdr-frameworks
description: "csdr is a command-line DSP toolkit whose small programs are chained with Unix pipes to build SDR signal chains, best known as the engine inside OpenWebRX."
keywords: csdr, command-line DSP, Unix pipe, OpenWebRX, IQ processing, fmdemod, fir_decimate, Andras Retzler, stdin stdout
aka: [csdr]
autolink: true
infobox:
  - { label: Type, value: Command-line DSP toolkit }
  - { label: Idea, value: Chain small DSP programs with Unix pipes }
  - { label: Examples, value: OpenWebRX demod chains, scripted SDR }
see_also: [web-sdr, liquid-dsp, fir-filter, software-defined-radio, iq-data, demodulation]
cite_urls:
  - https://github.com/jketterl/csdr
  - https://github.com/ha7ilm/csdr
---

**csdr** is a command-line digital-signal-processing toolkit in which each DSP operation is
a small program that reads samples from standard input and writes them to standard output, so
a complete SDR signal chain is built by connecting the programs with Unix pipes.[^csdr] It is
best known as the DSP engine inside [OpenWebRX](/reference/web-sdr/), where csdr commands turn
raw [IQ](/reference/iq-data/) into demodulated audio on the server.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A csdr pipeline: raw IQ piped into a decimating FIR filter, then a shift, then an FM demodulator, then audio, each stage a separate program connected by a Unix pipe." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="csar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="7.5" fill="currentColor" text-anchor="middle">
    <rect x="12" y="45" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="47" y="63">raw IQ</text>
    <rect x="105" y="45" width="80" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="145" y="60">fir_decimate</text><text x="145" y="69">_cc</text>
    <rect x="208" y="45" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="243" y="63">shift_cc</text>
    <rect x="300" y="45" width="70" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="335" y="63">fmdemod</text>
    <rect x="392" y="45" width="60" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="422" y="63">audio</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="82" y1="60" x2="105" y2="60" marker-end="url(#csar)"/>
    <line x1="185" y1="60" x2="208" y2="60" marker-end="url(#csar)"/>
    <line x1="278" y1="60" x2="300" y2="60" marker-end="url(#csar)"/>
    <line x1="370" y1="60" x2="392" y2="60" marker-end="url(#csar)"/>
  </g>
  <text x="232" y="98" font-size="8" fill="currentColor" text-anchor="middle">each "|" is a Unix pipe carrying a raw float sample stream</text>
</svg>
<figcaption>A csdr signal chain is literally a shell pipeline: each DSP stage is its own program, and Unix pipes carry the raw sample stream from one to the next.</figcaption>
</figure>

## How it works

csdr embraces the Unix philosophy: instead of one monolithic application, it provides many tiny
tools — `fir_decimate_cc`, `shift_addition_cc`, `fmdemod_quadri_cf`, `agc_ff`, `bandpass_fir_fft_cc`,
`convert_u8_f`, and dozens more — each doing one DSP job on a stream of raw samples. Samples cross
the pipes as headerless binary: complex float pairs for IQ (`_cc`/`_cf` suffixes) or real floats
for audio (`_ff`/`_ff`). A naming convention encodes each tool's input and output type, so the
suffixes tell you what can legally connect to what. To demodulate a station you decimate to the
channel bandwidth, shift the channel to baseband, filter, demodulate, and resample to audio — each
step a separate process, all running concurrently while the OS schedules them and pipe buffers
provide flow control.

Under the hood the DSP is written in C for speed, with SIMD-optimized [FIR
filters](/reference/fir-filter/) and FFT routines. Later versions reorganized into a C++ library
with a thin command wrapper, but the pipe-friendly command interface remained the defining feature.

## Variants

Two lineages exist. The original **ha7ilm/csdr** by András Retzler established the tool set and
its use in the first OpenWebRX. The actively maintained **jketterl/csdr** fork, developed alongside
the modern OpenWebRX, refactored the internals into a reusable C++ library and a `csdr` command
front end, added modules, and improved performance while keeping the pipeline model. A companion
project, `pycsdr`, exposes the same DSP blocks to Python so chains can be built in code rather than
in a shell.

## In practice

csdr's natural habitat is server-side SDR streaming, above all [OpenWebRX](/reference/web-sdr/):
when a browser client tunes a channel, the server assembles a csdr pipeline for the requested
mode (AM, FM, SSB, digital), feeds it IQ from the receiver, and streams the resulting audio out.
The pipe model makes it trivial to prototype and reconfigure a chain from the shell and to splice
in other Unix tools. Its trade-offs are the trade-offs of processes and pipes: copying float
streams between programs costs memory bandwidth, and very tight, low-latency loops are better served
by an in-process library such as [liquid-dsp](/reference/liquid-dsp/).

## Relevance to SDR

csdr is a compact demonstration that a full receive chain — decimation, translation, filtering,
demodulation, resampling — is just a sequence of stream transforms, and that Unix pipes are a
perfectly good "flowgraph" for connecting them. That makes it an excellent teaching and scripting
tool and the reason web-SDR platforms could offer many modes without a heavyweight framework.

GopherTrunk implements the same conceptual chain, but in-process rather than across pipes: its
Go pipeline decimates, translates each channel to baseband with a downconverter, filters, and
demodulates, passing sample buffers between stages inside one program. GT is a decoder of digital
trunking rather than an analog audio streamer, and it favors a single statically compiled binary
over a shell of cooperating processes — but csdr's stage-by-stage view of the signal chain maps
directly onto what GT does internally, and both rely on the same primitives, above all the
[FIR filter](/reference/fir-filter/).

## Sources

[^csdr]: [csdr repository](https://github.com/jketterl/csdr) — the maintained csdr DSP toolkit, documenting its command/pipe interface, sample-type conventions, and use inside OpenWebRX.
