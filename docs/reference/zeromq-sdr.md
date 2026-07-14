---
slug: zeromq-sdr
title: ZeroMQ for SDR
entry_type: technology
category: sdr-data-streaming
description: "ZeroMQ for SDR uses GNU Radio's ZMQ source and sink blocks to move IQ streams between processes or machines over a message-passing socket library."
keywords: ZeroMQ SDR, ZMQ, GNU Radio ZMQ blocks, inter-process IQ, PUB SUB PUSH PULL, message passing, distributed flowgraph, zmq source sink, streaming samples
aka: [ZeroMQ, ZMQ, 0MQ]
autolink: true
infobox:
  - { label: Type, value: Message-passing transport for IQ }
  - { label: In GNU Radio, value: ZMQ PUB/SUB/PUSH/PULL source & sink }
  - { label: Idea, value: Split a flowgraph across processes or hosts }
see_also: [network-iq-streaming, stream-vs-message-passing, gnuradio, flowgraph, file-source-sink]
cite_urls:
  - https://wiki.gnuradio.org/index.php/ZMQ_PUB_Sink
  - https://en.wikipedia.org/wiki/ZeroMQ
---

**ZeroMQ for SDR** is the use of the ZeroMQ (ZMQ) message-passing library — via GNU Radio's ZMQ
**source and sink blocks** — to carry [IQ](/reference/iq-data/) streams between separate processes or
machines.[^zmq] It lets a single [flowgraph](/reference/flowgraph/) be cut into pieces that run
independently: a sink block in one process publishes samples onto a ZMQ socket, and a source block in
another subscribes to them, with ZeroMQ handling the framing, buffering, and reconnection.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="One GNU Radio flowgraph ends in a ZMQ PUB sink that publishes IQ; two other flowgraphs each contain a ZMQ SUB source that receives the same stream over a socket." xmlns="http://www.w3.org/2000/svg">
  <g font-size="8.5" fill="currentColor" text-anchor="middle">
    <rect x="24" y="50" width="120" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="84" y="65">source flowgraph</text><text x="84" y="77" font-size="7.5">… → ZMQ PUB sink</text>
    <rect x="316" y="24" width="120" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="376" y="39">SUB source → …</text><text x="376" y="49" font-size="7">subscriber A</text>
    <rect x="316" y="82" width="120" height="30" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="376" y="97">SUB source → …</text><text x="376" y="107" font-size="7">subscriber B</text>
    <line x1="144" y1="63" x2="315" y2="40" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#zmar)"/>
    <line x1="144" y1="71" x2="315" y2="96" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" marker-end="url(#zmar)"/>
    <text x="232" y="60" font-size="7.5">ZMQ socket (tcp://)</text>
  </g>
  <defs><marker id="zmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>A ZMQ PUB sink publishes IQ from one flowgraph; SUB sources in other processes subscribe to the same stream over a socket.</figcaption>
</figure>

## How it works

ZeroMQ is a socket library that sits a level above raw TCP: it moves discrete **messages** rather than
a byte stream, and builds in patterns for how endpoints relate. GNU Radio wraps four of these as block
pairs:

- **PUB / SUB** — one publisher fans a stream out to any number of subscribers, each getting a copy;
  subscribers can come and go, and a slow one is dropped rather than stalling the publisher. Ideal for
  broadcasting one radio's IQ to several consumers.
- **PUSH / PULL** — a pipeline/load-balancing pair: pushed messages are distributed round-robin across
  the pullers, useful for spreading heavy DSP across worker processes.

A GNU Radio ZMQ sink serialises the samples of its input stream (with optional
[stream tags](/reference/stream-tags/)) into ZMQ messages and binds or connects a socket at a
`tcp://host:port` address; the matching source on the other side deserialises them back into a GNU
Radio stream. Because the transport is just a URL, the two halves can be threads, separate processes on
one box, or programs on different machines, with no change to the flowgraph logic. ZeroMQ's internal
queues absorb jitter, and its automatic reconnection means a restarted consumer rejoins without
tearing down the producer.

## Relevance to SDR

ZeroMQ is GNU Radio's standard tool for **inter-process and inter-host** IQ movement, and the reason it
matters is architectural: it lets you decompose a monolithic flowgraph. A stable, expensive front end
(tune, filter, decimate) can run as one long-lived process publishing baseband IQ, while experimental
decoders attach and detach as SUB subscribers without disturbing it — a much more flexible topology
than a single graph. It also bridges GNU Radio to non-GNU-Radio programs, since ZeroMQ has bindings in
most languages, so a Python or C++ analysis tool can consume the same published stream. Compared with
the dedicated [network IQ streaming](/reference/network-iq-streaming/) servers, ZMQ is lower-level and
more general: it does not know it is carrying IQ, which is exactly what makes it composable. It embodies
the [message-passing](/reference/stream-vs-message-passing/) style of connecting DSP components, as
opposed to a single shared in-process stream.

GopherTrunk does not use ZeroMQ — it is a self-contained pure-Go application with an internal streaming
pipeline rather than a GNU Radio flowgraph, and it has no GNU Radio dependency. ZeroMQ is relevant here
as the wider ecosystem's answer to a problem GopherTrunk solves internally with Go channels and ring
buffers: getting sample streams from one processing stage to the next, and occasionally to another
process, without losing or blocking on data. Where GNU Radio reaches for a ZMQ block, GopherTrunk moves
the same samples over an in-process bus.

## Sources

[^zmq]: [ZMQ PUB Sink](https://wiki.gnuradio.org/index.php/ZMQ_PUB_Sink) — GNU Radio wiki, documenting the ZeroMQ sink/source blocks and the PUB/SUB and PUSH/PULL patterns for carrying streams between flowgraphs.
