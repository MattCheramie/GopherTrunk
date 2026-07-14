---
slug: throttle-block
title: Throttle
entry_type: concept
category: sdr-programming
description: A throttle block rate-limits a flowgraph that has no hardware clock, sleeping so samples flow at a set rate — essential for replaying files or running simulations without pegging a CPU.
keywords: throttle, throttle block, rate limit, sample rate pacing, no hardware, file replay, simulation, GNU Radio throttle, real time pacing, sleep
aka: [throttle, throttle block]
autolink: true
infobox:
  - { label: Type, value: Rate-limiting pipeline block }
  - { label: Job, value: Pace samples to a wall-clock rate }
  - { label: Use, value: Flowgraphs with no hardware source }
see_also: [file-source-sink, real-time-dsp, simulation-driven-sdr, back-pressure, block-scheduler]
cite_urls:
  - https://wiki.gnuradio.org/index.php/Throttle
  - https://en.wikipedia.org/wiki/Rate_limiting
---

**A throttle** is a pipeline block that limits how fast samples move through a flowgraph, sleeping as needed so the average throughput matches a specified sample rate.[^gr] It exists to solve one problem: a flowgraph whose source is a **file** or a **generator** has no clock of its own and will otherwise run as fast as the CPU can go, so a throttle re-imposes the wall-clock pace that real hardware would have provided.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A file source feeds a throttle block that meters samples at a fixed rate into a demod and sink; without the throttle the pipeline would run as fast as the CPU allows." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="thar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="12" fill="currentColor">
    <rect x="8" y="45" width="86" height="40" fill="none" stroke="currentColor"/><text x="20" y="63">file</text><text x="20" y="77" font-size="9">source</text>
    <rect x="140" y="45" width="96" height="40" fill="currentColor" fill-opacity="0.15" stroke="currentColor"/><text x="158" y="69">throttle</text>
    <rect x="282" y="45" width="86" height="40" fill="none" stroke="currentColor"/><text x="300" y="69">demod</text>
    <line x1="94" y1="65" x2="138" y2="65" stroke="currentColor" marker-end="url(#thar)"/>
    <line x1="236" y1="65" x2="280" y2="65" stroke="currentColor" marker-end="url(#thar)"/>
    <g stroke="currentColor" stroke-opacity="0.6"><line x1="150" y1="95" x2="150" y2="112"/><line x1="176" y1="95" x2="176" y2="112"/><line x1="202" y1="95" x2="202" y2="112"/></g>
    <text x="120" y="128" font-size="11">meters to N samples/s</text>
  </g>
</svg>
<figcaption>With no ADC to set the tempo, a throttle paces the stream to a chosen sample rate so a file-driven flowgraph runs in real time instead of flat out.</figcaption>
</figure>

## How it works

A throttle passes samples through unchanged; its only effect is timing. It records a start time and a running count of samples emitted, and before releasing the next batch it computes how long that many samples *should* have taken at the target rate. If it is ahead of that schedule, it sleeps the difference; if it is behind, it passes data straight through. Over time the emitted count tracks wall-clock time multiplied by the rate, so the stream averages the requested samples per second.

Because a throttle only *slows* things, it applies [back-pressure](/reference/back-pressure/) upstream: while it is sleeping it is not reading its input, so the [file source](/reference/file-source-sink/) blocks and the whole chain idles at the set rate instead of busy-spinning a core to 100%. That single property — turning a CPU-bound loop into a paced, low-load stream — is the practical reason it exists.

## In practice

- **Exactly one per flowgraph.** Two throttles fighting over the same stream produce erratic timing; the convention is a single throttle just after the source.
- **Never alongside real hardware.** A live SDR source already sets the clock. Adding a throttle then makes two clocks compete, causing the hardware to overflow — so a throttle is strictly a *hardware-less* tool.
- **Pacing, not precision.** Because it works by sleeping in chunks, a throttle's instantaneous rate is lumpy; it guarantees the long-run average, not sample-accurate timing. For anything needing true determinism you drop the throttle and let the consumer set the pace directly.

## Relevance to SDR

The throttle is the canonical bridge between offline data and a pipeline built to run in [real time](/reference/real-time-dsp/). When you develop against a recorded capture or a [simulation-driven](/reference/simulation-driven-sdr/) signal generator, the throttle lets you watch a spectrum or constellation update at a lifelike rate, and keeps a demo or GUI from saturating the CPU. Every block-based SDR framework ships one for this reason.

**GopherTrunk** takes a different but related approach to the same need. For deterministic testing and its `.cfile` replay path, GT does not pace to wall-clock at all — it lets the decoder consume the file as fast as it can, which makes offline runs both reproducible and quick, and its decode chain is rate-invariant so the result matches a live capture. Where GT does want lifelike pacing (for example driving a live-style visual from a file), the same throttle idea applies: meter the [file source](/reference/file-source-sink/) to the capture's sample rate. The distinction to keep honest is that throttling buys *realistic timing*, whereas GT's test path deliberately gives that up in exchange for speed and determinism.

## Sources

[^gr]: [Throttle block](https://wiki.gnuradio.org/index.php/Throttle) — GNU Radio Wiki, on pacing a flowgraph to a sample rate when no hardware clock is present.
