---
slug: stream-tags
title: Stream tags
entry_type: concept
category: sdr-programming
description: "A stream tag is a key/value metadata item attached to a specific sample index in a DSP stream, used to mark bursts, rate changes, timestamps, and other in-band events."
keywords: stream tags, GNU Radio tags, sample metadata, tagged stream, tag propagation, burst tagging, rx_time tag, sob eob, PMT tag, in-band metadata
aka: [tags, tagged stream, stream tags]
autolink: true
infobox:
  - { label: Type, value: In-stream metadata }
  - { label: Anchored to, value: A specific sample index }
  - { label: Used in, value: "GNU Radio tagged streams" }
see_also: [stream-vs-message-passing, signal-processing-block, sigmf, packet-framing, iq-recording-playback]
cite_urls:
  - https://wiki.gnuradio.org/index.php/Stream_Tags
  - https://en.wikipedia.org/wiki/GNU_Radio
---

**A stream tag is a key/value metadata item pinned to a specific sample index inside a DSP
stream.**[^tags] Where an ordinary stream carries only sample values, a tag lets a
[block](/reference/signal-processing-block/) attach information — a timestamp, a burst start,
a new sample rate, a detected frame boundary — to *one exact sample* and have it travel with
the data downstream. Tags bridge the gap between raw sample streams and asynchronous
[messages](/reference/stream-vs-message-passing/): the metadata stays perfectly aligned to the
sample it describes, no matter how the stream is buffered or delayed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A horizontal row of stream samples with two tags anchored to specific samples: one labeled rx_time on an early sample and one labeled sob (start of burst) on a later sample, each drawn as a flag attached to its sample index." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="95" x2="440" y2="95" stroke="currentColor" stroke-width="1.2"/>
  <g fill="currentColor"><circle cx="60" cy="95" r="2.5"/><circle cx="100" cy="95" r="2.5"/><circle cx="140" cy="95" r="2.5"/><circle cx="180" cy="95" r="2.5"/><circle cx="220" cy="95" r="2.5"/><circle cx="260" cy="95" r="2.5"/><circle cx="300" cy="95" r="2.5"/><circle cx="340" cy="95" r="2.5"/><circle cx="380" cy="95" r="2.5"/><circle cx="420" cy="95" r="2.5"/></g>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="140" y1="95" x2="140" y2="45"/>
    <line x1="300" y1="95" x2="300" y2="45"/>
  </g>
  <g font-size="8" fill="currentColor">
    <rect x="112" y="34" width="56" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1"/><text x="140" y="45" text-anchor="middle">rx_time</text>
    <rect x="274" y="34" width="52" height="16" rx="2" fill="none" stroke="currentColor" stroke-width="1"/><text x="300" y="45" text-anchor="middle">sob</text>
    <text x="235" y="120" text-anchor="middle">sample index increases →</text>
  </g>
</svg>
<figcaption>Each tag is anchored to one sample index and carries a key/value pair (here a receive timestamp and a start-of-burst marker) that propagates downstream with the stream.</figcaption>
</figure>

## How it works

A tag has three parts: the **absolute sample offset** it is attached to (counted from the start
of the stream), a **key** identifying what it means, and a **value**. In GNU Radio both key and
value are PMTs, so a value can be a number, a string, or a structured dictionary. Any block can
add a tag at the sample it is currently producing, read tags on its input, and — crucially —
the runtime **propagates** tags automatically: as samples flow downstream, their tags move with
them, and blocks that change the sample rate shift each tag's offset so it still points at the
right sample.

That propagation is the whole value of tags. Because the metadata is bound to a sample rather
than delivered out of band, it survives buffering, threading, and rate changes without drifting
out of alignment. A tag placed on the first sample of a burst is still on the first sample of
that burst after the stream has been filtered, decimated, and requeued several blocks later.

Propagation policy is configurable per block: *all-to-all* (default, copy input tags to
outputs), *one-to-one* (for rate-preserving blocks), or *none* (the block manages tags itself,
typical for a resampler or a framer that re-derives its own markers).

## In practice

Common uses include:

- **Timestamps** — an `rx_time` tag on the first sample from a device pins wall-clock or
  GPS-disciplined time to the stream, so downstream events can be time-stamped precisely.
- **Bursty / packetized data** — `sob`/`eob` (start/end of burst) tags, or a length tag on a
  *tagged stream block*, mark where a [packet](/reference/packet-framing/) begins and ends so a
  framer can operate on variable-length chunks within a continuous stream.
- **Rate and frequency changes** — a tag announces a new sample rate or a retune so downstream
  filters can adapt at the exact sample the change takes effect.

Tags are also the natural in-flowgraph counterpart to file-level metadata formats like
[SigMF](/reference/sigmf/): SigMF annotations describe events at sample offsets in a recording,
which map almost directly onto stream tags when that recording is played back through a
flowgraph.

## Relevance to SDR

Tags solve a recurring SDR problem: some facts about a signal are only known at one instant
(when the burst started, when the radio was tuned, what time it was) and must stay welded to
that instant through the rest of the pipeline. Doing this with side channels invites off-by-N
misalignment as buffers and rates change; tags make correct-by-construction alignment the
runtime's job.

[GopherTrunk](/reference/software-defined-radio/) is pure Go and does not use GNU Radio's tag
system, so it has no `rx_time`/`sob` PMT tags as such. It achieves the same alignment guarantees
structurally instead — burst and frame boundaries are carried by the framing stage that produced
them, and timestamps travel alongside the decoded events rather than as separate stream
annotations. The concept is still the right lens for understanding metadata-that-must-follow-a-
sample, and it maps directly onto how annotated
[IQ recordings](/reference/iq-recording-playback/) are described and replayed.

## Sources

[^tags]: [Stream Tags](https://wiki.gnuradio.org/index.php/Stream_Tags) — GNU Radio Wiki, on tags anchored to sample offsets, PMT key/value pairs, and tag propagation policies.
