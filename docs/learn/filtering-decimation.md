---
slug: filtering-decimation
title: Filtering & decimation
description: Digital filtering and decimation for SDR explained — how software zooms in on a single channel, removes everything else with low-pass and band-pass filters, and lowers the sample rate to keep the CPU manageable.
keywords: digital filter, decimation, channelization, low-pass filter, band-pass filter, downsampling, SDR DSP, filter then decimate, channel filter
level: intermediate
status: full
prereq:
  - sample-rate-nyquist
faq:
  - q: What is decimation in DSP?
    a: Decimation is reducing a signal's sample rate by keeping only every Nth sample after filtering out the frequencies that would otherwise alias. It's how an SDR narrows a wide capture down to just one channel's worth of bandwidth, which dramatically cuts the amount of data the rest of the pipeline has to process.
  - q: Why filter before decimating?
    a: Decimation lowers the sample rate, which shrinks the bandwidth that can be represented. Any energy outside that new, smaller bandwidth would fold back as aliasing and corrupt the signal. Filtering first removes that out-of-band energy, so decimation only throws away samples you no longer need. Filter-then-decimate is the standard order for this reason.
  - q: What does a digital filter actually do?
    a: A digital filter passes some frequencies and attenuates others, working on the stream of samples with arithmetic. A low-pass filter keeps low frequencies and removes high ones; a band-pass filter keeps a chosen band and rejects everything outside it. In an SDR, a channel filter isolates one narrow signal from a wide capture.
  - q: How does an SDR follow many channels from one capture?
    a: It channelises — for each channel of interest, it digitally shifts that channel to centre, applies a filter to isolate it, and decimates to a low sample rate suited to that one signal. Because this is just math on the shared IQ stream, the SDR can run several of these in parallel, which is how GopherTrunk follows a control channel and multiple voice channels at once.
gophertrunk_links:
  - title: Architecture
    url: /architecture.html
    note: how GopherTrunk channelises one IQ stream into many decoders.
  - title: Mixer
    url: /mixer.html
    note: the frequency-shift step that centres a channel before filtering.
---

# Filtering & decimation

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A wide [IQ capture](/learn/sample-rate-nyquist/) contains far more than the one channel
you want. **Digital filtering** keeps a chosen slice of spectrum and rejects the rest;
**decimation** then lowers the sample rate to match that narrow slice, slashing the data
the rest of the pipeline must handle. The order matters — **filter first, then
decimate** — or out-of-band energy [aliases](/learn/sample-rate-nyquist/) back in. Doing
this per channel ("channelising") is how one SDR feeds **many decoders at once**.
</div>

You've captured a couple of megahertz of [IQ](/learn/iq-data/). Now you need one
12.5 kHz channel out of it, running cheaply enough to do several at a time. That's
filtering and decimation — the first real DSP step in the
[demodulation pipeline](/learn/demodulation-pipeline/).

## What a digital filter does

A **digital filter** is just arithmetic on the sample stream that **passes some
frequencies and attenuates others**. No physical components — it's math applied to the
IQ. The two kinds you'll meet:

- **Low-pass** — keeps frequencies below a cutoff, removes those above.
- **Band-pass** — keeps a chosen band, rejects everything outside it.

In an SDR, the workhorse is a **channel filter**: a narrow filter that isolates *one*
signal (say, a single control channel) and discards its neighbours and the noise around
them. To centre the channel first, the SDR digitally **shifts** it to zero frequency
(the [mixer](/mixer.html) step) so a simple low-pass filter can do the isolating.

## Decimation: lowering the sample rate

Once you've filtered down to a narrow channel, you no longer need millions of samples a
second to represent it — a 12.5 kHz channel needs only a small fraction of the original
rate. **Decimation** drops the rate by keeping, say, every 100th sample. The result is
a much smaller stream that still contains your channel in full.

Why bother? Because every later stage —
[demodulation](/learn/demodulation-pipeline/), [clock recovery](/learn/clock-recovery/),
decoding — does work *per sample*. Fewer samples = far less CPU.

## Why filter before decimating?

Order is everything. Decimation **reduces the representable bandwidth** (it's the
[Nyquist](/learn/sample-rate-nyquist/) limit in reverse). If any energy still sits
outside that smaller bandwidth when you decimate, it **folds back as aliasing** and
contaminates your channel — a phantom signal landing right on top of the one you want.

So you **filter first** to remove everything outside the channel, *then* decimate to
throw away the now-unnecessary samples. Filter-then-decimate is the universal pattern.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="Left: a wide spectrum with several signals and a filter window around one. Middle arrow filter. Right: only the selected channel remains, at a lower sample rate." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <line x1="20" y1="90" x2="190" y2="90" stroke="currentColor" stroke-opacity="0.3"/>
    <path d="M40 90 L46 55 L52 90 Z" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <path d="M95 90 L101 45 L107 90 Z" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <path d="M150 90 L156 65 L162 90 Z" fill="currentColor" fill-opacity="0.3" stroke="currentColor"/>
    <rect x="86" y="35" width="30" height="60" fill="none" stroke="currentColor" stroke-dasharray="3 2"/>
    <text x="101" y="110">filter window</text>
    <text x="105" y="28">wide capture</text>
    <line x1="205" y1="70" x2="245" y2="70" stroke="currentColor" stroke-width="1.5" marker-end="url(#fd)"/>
    <text x="225" y="62">filter +</text><text x="225" y="84">decimate</text>
    <line x1="270" y1="90" x2="440" y2="90" stroke="currentColor" stroke-opacity="0.3"/>
    <path d="M345 90 L351 40 L357 90 Z" fill="currentColor" fill-opacity="0.4" stroke="currentColor"/>
    <text x="351" y="28">one channel, low rate</text>
  </g>
  <defs><marker id="fd" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Isolate one channel with a filter, then decimate to a low sample rate. The rest of the pipeline now has far less data to chew on.</figcaption>
</figure>

## How this fits the demodulation pipeline

Filtering and decimation are the **front of every channel's
[demodulation pipeline](/learn/demodulation-pipeline/)**: shift the channel to centre,
filter it, decimate. Because it's all arithmetic on the shared
[IQ stream](/learn/iq-data/), GopherTrunk runs **many of these in parallel** — one for
the [control channel](/learn/what-is-trunking/) and one for each
[voice channel](/learn/antenna-to-audio/) it's following — from a single capture. That
parallel channelising is exactly how it can track several calls at once.

<div class="knowledge-check" data-quiz data-correct-msg="Right — filter first so out-of-band energy can't alias when you decimate." markdown="0">
  <p class="knowledge-check__q">Quick check: why must you filter before decimating?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To make the signal louder</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">So out-of-band energy doesn't alias back into the channel</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Decimation only works on filtered audio</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Digital filters** keep a chosen band and reject the rest (low-pass, band-pass, channel).
- **Decimation** lowers the sample rate to match a narrow channel, saving CPU.
- Always **filter then decimate** to avoid aliasing.
- **Channelising** the shared IQ lets one SDR feed **many parallel decoders**.

Next: the whole chain assembled — the demodulation pipeline from tune to decoded bits.
