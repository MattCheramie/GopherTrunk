---
slug: stream-vs-message-passing
title: Stream vs message passing
entry_type: concept
category: sdr-programming
description: "Stream vs message passing is the distinction between continuous sample streams between DSP blocks and asynchronous, self-contained messages used for control and events."
keywords: stream vs message passing, PMT, GNU Radio messages, message ports, sample stream, asynchronous messaging, control plane, data plane, async PDU, polymorphic type
aka: [stream vs message passing, streams and messages]
autolink: true
infobox:
  - { label: Type, value: Two block communication modes }
  - { label: Streams, value: Continuous synchronous samples }
  - { label: Messages, value: Asynchronous discrete PMTs }
see_also: [stream-tags, signal-processing-block, flowgraph, packet-framing, async-programming]
cite_urls:
  - https://wiki.gnuradio.org/index.php/Message_Passing
  - https://en.wikipedia.org/wiki/Message_passing
---

**Stream vs message passing is the distinction between the continuous stream of samples that
flows between DSP [blocks](/reference/signal-processing-block/) and the asynchronous,
self-contained messages used for control and discrete events.**[^msg] A stream is the data
plane — an unbroken sequence of samples clocked through the [flowgraph](/reference/flowgraph/)
at the sample rate. A message is the control plane — an occasional, standalone object (a
retune command, a decoded packet, a "carrier detected" event) delivered when it happens, not
tied to any particular sample instant. Real radios need both, and knowing which to use for a
given piece of information is a core design decision in block-based SDR.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Two blocks connected two ways: a solid line carrying a continuous stream of evenly spaced samples along the bottom, and a dashed line carrying occasional discrete message packets along the top." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="smar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="20" y="60" width="70" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="55" y="84">block A</text>
    <rect x="370" y="60" width="70" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="405" y="84">block B</text>
    <text x="230" y="118" font-size="8">stream: continuous samples</text>
    <text x="230" y="30" font-size="8">messages: occasional events</text>
  </g>
  <line x1="90" y1="92" x2="369" y2="92" stroke="currentColor" stroke-width="1.3" marker-end="url(#smar)"/>
  <g fill="currentColor"><circle cx="120" cy="92" r="2"/><circle cx="150" cy="92" r="2"/><circle cx="180" cy="92" r="2"/><circle cx="210" cy="92" r="2"/><circle cx="240" cy="92" r="2"/><circle cx="270" cy="92" r="2"/><circle cx="300" cy="92" r="2"/><circle cx="330" cy="92" r="2"/></g>
  <line x1="90" y1="45" x2="369" y2="45" stroke="currentColor" stroke-width="1.2" stroke-dasharray="4 4" marker-end="url(#smar)"/>
  <g fill="none" stroke="currentColor" stroke-width="1.2"><rect x="150" y="40" width="10" height="10"/><rect x="300" y="40" width="10" height="10"/></g>
</svg>
<figcaption>Streams carry evenly clocked samples (the data plane); messages carry occasional, self-contained events (the control plane) on a separate asynchronous port.</figcaption>
</figure>

## How it works

A **stream** connection is synchronous and rate-locked. The upstream block produces items and
the downstream block consumes them in lockstep with the sample clock; the
[scheduler](/reference/block-scheduler/) meters exactly how many items move each time a block
runs. Streams are the right tool for anything that exists at every sample instant: IQ, filtered
baseband, a demodulated soft-symbol waveform.

A **message** connection is asynchronous and decoupled from the clock. A block posts a discrete
message object to a named message port; the runtime delivers it to whatever ports are connected,
whenever the receiver next services its queue. There is no rate, no per-sample alignment, and no
back-pressure in the streaming sense — a message is a whole thing that either arrives or does
not. In GNU Radio, messages are **PMTs** (Polymorphic Types): tagged, serializable containers
that can hold a number, a string, a dictionary, or a "PDU" pairing metadata with a vector of
bytes.

The dividing line is *cardinality and timing*. Information that is present continuously and
must stay sample-aligned belongs on a stream. Information that is sparse, event-driven, or
addressed to the flowgraph as a whole — "tune to 851.0125 MHz," "here is a completed frame,"
"squelch opened" — belongs in a message.

## In practice

A middle ground exists: [stream tags](/reference/stream-tags/) attach metadata to a specific
sample index *within* a stream, so you get message-like annotation that stays exactly aligned
to the data (a burst start, a detected preamble, a rate change). The rule of thumb: use a tag
when the metadata must travel with a known sample, and a message when it is independent of any
sample — a tag rides the data plane, a message rides the control plane.

Messages also break the strictly-downstream shape of a pure stream flowgraph. Because a message
port can connect a downstream block back to an upstream one, a decoder can send a retune request
back to the source, or a control GUI can push parameter changes into a running graph — feedback
loops that a one-directional sample stream cannot express.

## Relevance to SDR

Nearly every real receiver mixes the two: a high-rate sample stream carrying the signal, plus a
trickle of control and result messages — retunes, decoded packets, lock/unlock events. Getting
the split right keeps the hot DSP path free of branchy control logic and keeps events from being
forced awkwardly into a fixed-rate stream.

[GopherTrunk](/reference/software-defined-radio/) is pure Go and does not use GNU Radio's PMT
machinery, but it embodies exactly this separation: the [IQ](/reference/iq-data/) and demodulated
symbols move as high-rate streams through the decode chain, while control-channel results —
channel grants, talkgroup and radio IDs, tuning decisions for the voice follower — are discrete
events passed as Go structs over channels, the language's native asynchronous
[message-passing](/reference/async-programming/) primitive. The framework vocabulary differs, but
the data-plane/control-plane split is the same, and recognizing it is what makes a trunking
decoder's architecture legible.

## Sources

[^msg]: [Message Passing](https://wiki.gnuradio.org/index.php/Message_Passing) — GNU Radio Wiki, on asynchronous message ports, PMTs, and PDUs versus synchronous stream connections.
