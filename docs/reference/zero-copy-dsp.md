---
slug: zero-copy-dsp
title: Zero-copy
entry_type: concept
category: sdr-programming
description: "Zero-copy is a DSP and I/O technique that moves sample data by sharing buffers instead of copying them, cutting memory bandwidth and latency in SDR streams."
keywords: zero-copy, zero copy DSP, buffer sharing, mmap, DMA, memory bandwidth, in-place processing, ring buffer, scatter-gather, page mapping, no-copy I/O
aka: [zero-copy, zero-copy I/O, no-copy]
autolink: true
infobox:
  - { label: Type, value: Data-movement / memory technique }
  - { label: Idea, value: Share buffers instead of copying samples }
  - { label: Mechanisms, value: "mmap, DMA, in-place, shared ring buffers" }
see_also: [ring-buffer, sample-buffer, overruns-underruns, dsp-latency, iq-recording-playback]
cite_urls:
  - https://en.wikipedia.org/wiki/Zero-copy
  - https://en.wikipedia.org/wiki/Direct_memory_access
---

**Zero-copy** is the practice of moving sample data through a system by giving each stage
access to the *same* memory rather than copying bytes from one buffer to the next.[^wiki] In
high-rate [SDR](/reference/software-defined-radio/) software, where a stream may run at tens
of megasamples per second, needless copies waste memory bandwidth, evict caches, and add
[latency](/reference/dsp-latency/); zero-copy design keeps the data still and passes
references to it, which is often the difference between keeping up with the radio and dropping
samples.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 165" role="img" aria-label="A copy-based pipeline duplicates the sample buffer at each stage, consuming memory bandwidth, while a zero-copy pipeline lets all stages read and write one shared buffer." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="16" font-size="9" fill="currentColor">copying: buffer duplicated at each hand-off</text>
  <rect x="20" y="26" width="60" height="24" fill="none" stroke="currentColor"/><text x="50" y="41" font-size="8" fill="currentColor" text-anchor="middle">buf A</text>
  <line x1="82" y1="38" x2="118" y2="38" stroke="currentColor" marker-end="url(#zcar)"/><text x="100" y="32" font-size="7" fill="currentColor" text-anchor="middle">copy</text>
  <rect x="120" y="26" width="60" height="24" fill="none" stroke="currentColor"/><text x="150" y="41" font-size="8" fill="currentColor" text-anchor="middle">buf B</text>
  <line x1="182" y1="38" x2="218" y2="38" stroke="currentColor" marker-end="url(#zcar)"/><text x="200" y="32" font-size="7" fill="currentColor" text-anchor="middle">copy</text>
  <rect x="220" y="26" width="60" height="24" fill="none" stroke="currentColor"/><text x="250" y="41" font-size="8" fill="currentColor" text-anchor="middle">buf C</text>
  <text x="20" y="92" font-size="9" fill="currentColor">zero-copy: one buffer, stages hold references</text>
  <rect x="120" y="104" width="120" height="26" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="180" y="120" font-size="8.5" fill="currentColor" text-anchor="middle">shared buffer</text>
  <line x1="70" y1="150" x2="140" y2="132" stroke="currentColor" marker-end="url(#zcar)"/><text x="70" y="160" font-size="7.5" fill="currentColor" text-anchor="middle">reader</text>
  <line x1="290" y1="150" x2="220" y2="132" stroke="currentColor" marker-end="url(#zcar)"/><text x="290" y="160" font-size="7.5" fill="currentColor" text-anchor="middle">writer</text>
  <defs><marker id="zcar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Copy-based hand-offs duplicate the sample block at every stage; a zero-copy design keeps one buffer in place and lets producers and consumers reference it directly.</figcaption>
</figure>

## How it works

The costly thing in a naive pipeline is `memcpy`: every time samples pass from the driver to
the application, or from one processing stage to the next, the bytes are read from one region
and written to another. At SDR rates that traffic can dominate the memory bus and thrash the
CPU cache. Zero-copy eliminates the copies with a few complementary mechanisms.

- **DMA (direct memory access).** The device's controller writes incoming samples straight
  into host RAM without the CPU touching each byte; the application is handed the filled
  buffer. This is how USB and PCIe SDR front-ends deliver samples efficiently.
- **Memory mapping (`mmap`).** A file or a device/kernel buffer is mapped into the process's
  address space, so reading a capture file or a kernel DMA ring means dereferencing memory,
  not issuing `read()` calls that copy through kernel buffers. Great for
  [IQ recording/playback](/reference/iq-recording-playback/) of large files.
- **In-place processing.** A stage overwrites its input buffer with its output (a mixer
  multiplying samples by a phasor, say) instead of allocating a fresh output buffer.
- **Shared queues.** A [ring buffer](/reference/ring-buffer/) between a producer and consumer
  passes ownership of a slot rather than copying its contents; the consumer reads the same
  memory the producer wrote.

The cost is discipline. Sharing a buffer means agreeing on lifetime and ownership: a stage
must not overwrite data another stage is still reading, alignment and slot boundaries must be
respected, and in a multi-reader design you need reference counting or careful hand-off so a
[sample buffer](/reference/sample-buffer/) is not recycled too early. Zero-copy trades away
the safety of private copies for bandwidth and latency, so it belongs in the hot path, not
everywhere.

## In practice

The rule of thumb is to copy at the boundaries where you genuinely need isolation and share
everywhere else on the hot path. Format conversion and deinterleaving are often the *one*
unavoidable copy — turning packed device bytes into the float layout the DSP wants — after
which the sample block flows through filtering, down-conversion, and demodulation by reference.
Overrun handling interacts closely: if a consumer stalls, a shared ring buffer must either
apply back-pressure or drop a slot, since there is no private copy to fall back on, which is
why zero-copy pipelines pair naturally with explicit
[overrun/underrun](/reference/overruns-underruns/) policies.

## Relevance to SDR

Zero-copy is a core reason SDR frameworks can run megasample streams on ordinary CPUs.
Vendor libraries expose DMA'd buffers, GNU Radio passes blocks between processing nodes
through shared circular buffers, and file replay uses `mmap` to stream captures without
per-block reads. **GopherTrunk** embodies the spirit of this in a garbage-collected language:
it streams sample blocks through its decode chain via shared buffers and ring buffers rather
than reallocating per stage, and it works hard to avoid allocation in hot loops so the Go
garbage collector stays quiet — a Go program cannot achieve C-style pointer-level zero-copy
everywhere, but reusing and sharing backing slices is the same idea applied within the
language's model. Honestly framed, true kernel-bypass zero-copy (raw DMA rings, `mmap`'d
device buffers) lives mostly in the C driver layer beneath any host application; what an SDR
program controls is not re-copying samples needlessly once they arrive, and that alone often
decides whether the chain keeps pace.

## Sources

[^wiki]: [Zero-copy](https://en.wikipedia.org/wiki/Zero-copy) — Wikipedia, on avoiding redundant data copies via DMA, mmap, and shared buffers in high-throughput I/O.
