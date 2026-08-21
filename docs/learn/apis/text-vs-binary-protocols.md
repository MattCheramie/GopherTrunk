---
slug: text-vs-binary-protocols
title: Text vs binary protocols
description: Human-readable JSON versus compact binary framing — what each family costs in bytes, speed, and debuggability, measured honestly, and how to choose per situation.
keywords: text protocol, binary protocol, JSON vs protobuf, wire format comparison, debuggability, encoding overhead, protocol efficiency
level: intermediate
status: full
prereq:
  - data-formats
  - grpc-and-protobuf
---

# Text vs binary protocols

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Text protocols** (HTTP/1.1, JSON, SSE) spend bytes to stay **human-readable** —
any terminal, proxy log, or pair of eyes can inspect the traffic. **Binary
protocols** (protobuf, HTTP/2 framing, digital radio bursts) spend **tooling** to
save bytes and CPU. The honest comparison is situational: for low-volume APIs the
efficiency gap is noise and **debuggability wins**; for high-rate streams the gap
is the product. Compression narrows the *size* difference but not the *parsing*
one — and the modern winning pattern is **text at the edges, binary in the core**.
</div>

You've now used both families — curl-able JSON in Units 2–3, opaque protobuf in
the last lesson. This lesson puts the trade-off itself under the lamp, because
you'll make this choice (or inherit it) in every system you touch.

## What the same data costs

One call record, two honest encodings:

```text
JSON — 107 bytes, readable by anyone:
{"call_id":48213,"talkgroup":1201,"duration_seconds":8.4,
 "encrypted":false,"units":[70233,70281]}

Protobuf — ~25 bytes, readable by no one without the schema:
08 95 f8 02 10 b1 09 19 cd cc cc cc cc cc 20 40 28 00 32 06 ...
```

A ~4× size gap is typical for structured records: JSON pays for field names in
every message, decimal digits for every number, and quoting punctuation. Parsing
shows a similar spread — turning `"48213"` back into an integer costs real CPU;
reading a binary varint barely registers. Whether any of that *matters* is
purely a question of volume:

| Traffic | JSON overhead | Verdict |
|---------|---------------|---------|
| A config fetch at startup | microseconds, ~80 wasted bytes | Irrelevant — optimise for eyes |
| 100 req/s dashboard API | still small vs network latency | Irrelevant, usually |
| 50 audio chunks/s × hours | megabytes/hour, constant parse load | Binary earns its keep |
| Millions of msgs/s between services | dominant cost | Binary, no debate |

> Rule of thumb: the encoding matters in proportion to message rate × message
> count × lifetime. Measure before believing either camp — and remember the
> network round trip usually dwarfs both encodings at low volume.

## What text buys: the whole world can read it

The underrated asset of text protocols is that **every generic tool understands
them**. `curl -v` shows the exchange; grep works on logs; a proxy dump is
legible; a colleague can diagnose your API over chat from a pasted transcript.
When something misbehaves at 2 a.m., "I can read the actual traffic" shortens
incidents dramatically — the [packet-capture lesson](/learn/networking/packet-capture-basics/)
becomes self-service instead of an archaeology project.

Binary traffic *is* debuggable, but only through **schema-aware tooling**
(`grpcurl`, protobuf decoders, protocol dissectors) that must be present,
current, and pointed at the right schema. That's a maintained dependency, and in
a small project it's often the first thing that rots.

## What binary buys: the machine barely notices

Binary's wins compound at rate: fewer bytes per message, cheaper
encode/decode, **native binary payloads** (audio samples in JSON means base64 —
+33% size *plus* a copy), and precise types (no "is 2⁵³+1 still exact?"
worries). There's also framing honesty: binary protocols state lengths up front
instead of scanning for delimiters — the topic the
[next lesson](/learn/apis/message-framing/) treats in full.

Digital radio is the extreme case study: a P25 or TETRA burst is binary framing
under hard bandwidth physics — every bit on the air costs spectrum and time, so
the protocols pack fields to the bit and spend the savings on error correction.
Nobody proposes JSON over the air; when bytes are truly scarce, the debate ends.

## Compression: half an answer

"Just gzip the JSON" is the standard rejoinder, and it's half right. Compressed
JSON often lands within ~2× of protobuf's size — field names repeat, and
compressors feast on repetition. But compression adds its own CPU on both ends,
adds latency per message, and does nothing for **parsing** cost: after
decompression you still parse text. It also compresses *streams* better than
individual small messages. Compression is a fine tool for shrinking a text API's
bandwidth; it is not a refutation of binary formats where encode/decode rate is
the constraint.

## The pattern that wins in practice

Real systems increasingly refuse to choose globally, splitting instead by
audience: **human-facing and third-party edges speak text** (JSON REST APIs, SSE
feeds — curl-able, self-describing), while **machine-to-machine cores speak
binary** (gRPC between services, binary WebSocket frames for telemetry). Even
HTTP itself did this: HTTP/2 kept HTTP/1.1's readable *semantics* and swapped
the wire to binary framing — the concepts you debug in text, the bytes travel in
binary. GopherTrunk's daemon is a small mirror of the same split: JSON REST and
SSE at the edge, gRPC for the audio firehose, as
[Unit 6](/learn/apis/the-daemon-rest-api/) tours.

<div class="knowledge-check" data-quiz data-correct-msg="Right — compression shrinks the bytes but you still pay full text-parsing cost after decompressing." markdown="0">
  <p class="knowledge-check__q">Quick check: gzipping a JSON API's responses mostly closes the size gap with protobuf. What gap remains?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">None — compressed JSON and protobuf are then equivalent</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Parsing cost — after decompression the receiver still parses text, and both ends paid the compression CPU</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Compressed JSON can no longer be read by generic tools, so it loses text's advantages too</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Text** spends bytes for **readability**: generic tools, legible logs,
  debugging with eyes — worth the most exactly where traffic is light.
- **Binary** spends tooling for **efficiency**: ~4× smaller records, cheap
  parsing, native binary payloads — worth the most at high rate and volume.
- The choice matters **in proportion to volume** — measure before optimising,
  and remember round-trip latency dwarfs encoding at low rates.
- **Compression** narrows size, not parsing cost — half an answer.
- The winning pattern is **text at the edges, binary in the core** — HTTP/2 and
  GopherTrunk alike.

Next up: [Message framing](/learn/apis/message-framing/).
