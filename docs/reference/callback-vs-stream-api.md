---
slug: callback-vs-stream-api
title: Callback vs stream API
entry_type: concept
category: sdr-programming
description: "Callback vs stream API is the choice between a driver pushing sample buffers into your function versus your code pulling samples on demand, a core SDR I/O pattern."
keywords: callback API, stream API, pull model, push model, blocking read, async callback, SDR driver API, sample delivery, read model, event-driven I/O
aka: [push vs pull API, callback vs blocking read]
autolink: true
infobox:
  - { label: Type, value: SDR I/O programming model }
  - { label: Callback, value: "Driver pushes buffers into your function" }
  - { label: Stream, value: "You pull samples with a read call" }
see_also: [librtlsdr, soapysdr, ring-buffer, overruns-underruns, back-pressure]
cite_urls:
  - https://en.wikipedia.org/wiki/Callback_(computer_programming)
  - https://en.wikipedia.org/wiki/Push_technology
---

**Callback vs stream API** is the choice between two ways an SDR driver hands samples to your
program: a **callback** (push) model, where the driver calls a function you registered each time
a buffer of samples is ready, versus a **stream** (pull) model, where your code calls a
blocking `read`-style function to fetch the next samples when it wants them.[^cb] Every
[SDR](/reference/software-defined-radio/) library exposes one, the other, or both, and the
pattern you build around shapes threading, buffering, and how you handle overruns.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="In the callback model the driver thread invokes the application's function whenever samples arrive, while in the stream model the application thread calls read to pull samples from the driver's buffer." xmlns="http://www.w3.org/2000/svg">
  <text x="20" y="16" font-size="9" fill="currentColor">callback (push): driver drives the timing</text>
  <rect x="20" y="26" width="70" height="24" fill="none" stroke="currentColor"/><text x="55" y="41" font-size="8" fill="currentColor" text-anchor="middle">driver</text>
  <line x1="92" y1="38" x2="150" y2="38" stroke="currentColor" marker-end="url(#cbar)"/><text x="121" y="32" font-size="7.5" fill="currentColor" text-anchor="middle">calls you</text>
  <rect x="152" y="26" width="90" height="24" fill="none" stroke="currentColor"/><text x="197" y="41" font-size="8" fill="currentColor" text-anchor="middle">your callback</text>
  <text x="20" y="88" font-size="9" fill="currentColor">stream (pull): your code drives the timing</text>
  <rect x="20" y="98" width="90" height="24" fill="none" stroke="currentColor"/><text x="65" y="113" font-size="8" fill="currentColor" text-anchor="middle">your loop</text>
  <line x1="112" y1="110" x2="170" y2="110" stroke="currentColor" marker-end="url(#cbar)"/><text x="141" y="104" font-size="7.5" fill="currentColor" text-anchor="middle">read()</text>
  <rect x="172" y="98" width="70" height="24" fill="none" stroke="currentColor"/><text x="207" y="113" font-size="8" fill="currentColor" text-anchor="middle">driver</text>
  <line x1="170" y1="120" x2="114" y2="120" stroke="currentColor" marker-end="url(#cbar)"/><text x="141" y="134" font-size="7.5" fill="currentColor" text-anchor="middle">samples</text>
  <defs><marker id="cbar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>In the push model the driver's own thread invokes your callback whenever a buffer fills; in the pull model your thread blocks in a read call and the driver hands back samples on request.</figcaption>
</figure>

## How it works

In the **callback** model you register a function and start the stream; the driver spawns (or
borrows) a thread that, each time a USB or network buffer of samples arrives, invokes your
function with a pointer to that buffer. Control is *inverted* — the driver decides when your
code runs, and it runs in the driver's thread context. The iron rule is that a callback must
return fast: it runs on the thread that also has to service the next buffer, so any heavy DSP
done inline stalls the driver and causes an [overrun](/reference/overruns-underruns/). The
correct pattern is to copy or hand off the samples into a [ring buffer](/reference/ring-buffer/)
and return immediately, letting your own worker threads do the real processing.

In the **stream** model you own the loop. Your thread calls a blocking `readStream` /
`read_samples` function that returns the next *N* samples (or blocks until they are available),
and you process them and loop. Control stays with your code, which many people find easier to
reason about — it reads like ordinary sequential I/O — and it composes naturally with a
threading model you design yourself. The driver still buffers internally, and if your loop is
too slow to keep calling `read`, that buffer overflows and you drop samples, so the same
overrun discipline applies, just detected at the read call instead of inside a callback.

Key differences that drive the design:

- **Who controls timing.** Callback: the driver. Stream: your loop. This decides where
  back-pressure and pacing live.
- **Thread context.** Callback code runs on the driver's thread and must be short and
  non-blocking; stream code runs on your own thread and may block.
- **Error surfacing.** Streams return an overrun status from the read call; callbacks usually
  signal it out-of-band, so you must watch for it.
- **Composability.** A stream API drops cleanly into a language's normal blocking-I/O and
  concurrency idioms; a callback often needs an adapter to bridge into them.

## In practice

The two are easy to convert between, and robust programs usually normalize to one internal
model. A common bridge is to wrap a callback API in a small shim that pushes each delivered
buffer into a ring buffer, then expose a blocking `read` on the other side — turning push into
pull so the rest of the application sees a clean stream. This is exactly why libraries differ:
[librtlsdr](/reference/librtlsdr/) is fundamentally callback-driven (`rtlsdr_read_async` calls
your function per buffer), while [SoapySDR](/reference/soapysdr/) presents a unified
`readStream` pull interface across many devices. Whichever the hardware offers, the goal is the
same — get samples off the driver's thread and into your own buffered pipeline before doing
real work, so a slow DSP stage applies [back-pressure](/reference/back-pressure/) or drops
deliberately rather than corrupting the driver's timing.

## Relevance to SDR

This choice is the first thing you meet when writing SDR software, because it is the seam
between the vendor driver and your DSP. **GopherTrunk** faces it at every source: some device
back-ends are callback-based and others stream-based, so it adapts each into a consistent
internal sample-stream feeding its decode chain, and its file/replay source is naturally a pull
stream. Framing this honestly, the pattern is not GopherTrunk-specific — it is universal to SDR
applications, and understanding it explains why nearly every real receiver puts a ring buffer
immediately behind the source: it decouples the driver's delivery model from the program's
processing model, so a callback that must return in microseconds and a decode loop that may
pause for milliseconds can coexist without dropping samples.

## Sources

[^cb]: [Callback (computer programming)](https://en.wikipedia.org/wiki/Callback_(computer_programming)) — Wikipedia, on inversion of control and registered functions invoked by a driver or framework.
