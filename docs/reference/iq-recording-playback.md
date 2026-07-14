---
slug: iq-recording-playback
title: IQ recording & playback
entry_type: concept
category: sdr-data-streaming
description: "IQ recording and playback captures an SDR's raw baseband samples to a file, then replays them through the DSP chain to decode, debug, and test offline without the radio."
keywords: IQ recording, IQ playback, SDR capture and replay, record and replay, offline decoding, regression fixture, file source, cfile replay, deterministic testing
aka: [record and replay, IQ capture and replay, offline replay]
autolink: true
infobox:
  - { label: Type, value: Capture-then-replay workflow }
  - { label: Idea, value: Freeze a signal to a file, decode it repeatedly }
  - { label: Payoff, value: Deterministic, hardware-free reproduction }
see_also: [file-source-sink, simulation-driven-sdr, cfile-format, iq-file-format, testing-dsp-without-hardware, iq-data]
cite_urls:
  - https://wiki.gnuradio.org/index.php/File_Source
  - https://pysdr.org/content/iq_files.html
---

**IQ recording and playback** is the workflow of writing an SDR's raw baseband
[IQ samples](/reference/iq-data/) to a file, then feeding that file back through the DSP chain later —
turning a fleeting radio signal into a frozen dataset you can decode, inspect, and re-decode as many
times as you like.[^src] It is the single most useful habit in SDR development: a recorded capture is
reproducible where a live signal never is, so it converts a "sometimes it fails on air" problem into a
deterministic one anybody can re-run.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A radio's IQ stream is recorded to a file once, then replayed many times through the same demodulate-and-decode chain to reproduce a result deterministically." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <path d="M34 52 v-16 m-6 0 l6 -8 l6 8" fill="none" stroke="currentColor" stroke-width="1.5"/>
    <rect x="52" y="38" width="66" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="85" y="55">radio IQ</text>
    <rect x="150" y="38" width="70" height="28" rx="4" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.3"/><text x="185" y="51">IQ file</text><text x="185" y="61" font-size="7">.cfile</text>
    <line x1="118" y1="52" x2="149" y2="52" stroke="currentColor" stroke-width="1.2" marker-end="url(#rpar)"/><text x="134" y="32" font-size="7">record</text>
    <rect x="150" y="88" width="70" height="28" rx="4" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.3"/><text x="185" y="105">IQ file</text>
    <rect x="264" y="88" width="80" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="304" y="105">DDC → decode</text>
    <rect x="360" y="88" width="70" height="28" rx="4" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="395" y="105">result</text>
    <line x1="220" y1="102" x2="263" y2="102" stroke="currentColor" stroke-width="1.2" marker-end="url(#rpar)"/>
    <line x1="344" y1="102" x2="359" y2="102" stroke="currentColor" stroke-width="1.2" marker-end="url(#rpar)"/>
    <text x="185" y="80" font-size="7">replay ×N</text>
    <path d="M185 66 v20" stroke="currentColor" stroke-width="1" stroke-dasharray="3 2"/>
  </g>
  <defs><marker id="rpar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Record the IQ once; replay it through the same decode chain as often as needed to reproduce a result deterministically.</figcaption>
</figure>

## How it works

**Recording** taps the sample stream immediately after the radio — before any lossy processing — and
writes it to disk as a raw [IQ file](/reference/iq-file-format/): interleaved I, Q values in some
[sample format](/reference/sample-format/), most often the GNU Radio
[cfile](/reference/cfile-format/) (complex float32) or an 8-bit `cu8` capture. Because the file has no
header, the sample rate and centre frequency must be recorded alongside it, in the filename, a
[SigMF](/reference/sigmf/) sidecar, or documentation.

**Playback** reads those bytes back at the recorded rate and injects them into the DSP chain at the
same point the live radio would — a GNU Radio [File Source](/reference/file-source-sink/) standing in
for the hardware source, or a decoder's replay mode. The key property is that replay is
**deterministic**: the same file plus the same code always yields the same intermediate samples and
the same decode, so any bug that manifests in the capture is reproducible on demand. Replay can also
run *faster than real time* — the file has no clock — which is what makes it practical to sweep
parameters or run a regression suite over hundreds of captures quickly.

## In practice

For this to be meaningful the replay path must be the **same** path the live system uses, or a lock in
replay would not imply a lock on air. Capturing at the raw-IQ tap, not after demodulation, is also
essential: once symbols are sliced, the information needed to debug the front end is gone. A recorded
control-channel burst that fails to decode is the ideal bug report — it lets a developer reproduce the
exact failure without the reporter's radio, antenna, or RF environment.

## Relevance to SDR

Record-and-replay is the foundation of serious SDR software work: it enables
[testing without hardware](/reference/testing-dsp-without-hardware/), golden-capture regression suites,
and the [simulation-driven](/reference/simulation-driven-sdr/) development loop where a captured or
synthesised signal is the fixture. It is how a decoding project accumulates a library of known-good and
known-bad signals over time.

GopherTrunk is built around exactly this. Its `gophertrunk replay` subcommand reads a capture file —
`-format u8`, `f32` (a [cfile](/reference/cfile-format/)), or `wav` — and drives it through the same
production receiver, down-converter, and control-channel pipelines the live daemon runs, so a replay
lock implies an on-air lock and a replay failure makes the capture a reproducible fixture. This is why
the project's guidance repeatedly asks for the raw `.cfile` when an on-air problem is reported: the
capture, replayed offline, is what turns a symptom into something a regression test can pin down and a
fix can be verified against.

## Sources

[^src]: [File Source](https://wiki.gnuradio.org/index.php/File_Source) — GNU Radio wiki, on replaying a recorded raw-IQ file into a flowgraph as a stand-in for a live radio source, the basis of offline decoding.
