---
slug: grpc-and-protobuf
title: gRPC & Protocol Buffers
description: Define messages and services in a .proto file, generate typed client and server code, and stream in both directions — how gRPC and Protocol Buffers implement RPC as practiced today.
keywords: gRPC tutorial, protocol buffers, protobuf, proto file, gRPC streaming, RPC framework, field numbers, gRPC Go
level: intermediate
status: full
prereq:
  - what-is-rpc
  - data-formats
---

# gRPC & Protocol Buffers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**gRPC** is today's mainstream RPC framework: you write your **messages** and
**services** once in a **`.proto`** file, a compiler **generates typed client and
server code** in your languages, and calls travel as compact **Protocol Buffers**
binary over HTTP/2. Protobuf fields carry **numbers**, not names, on the wire —
which makes messages small *and* makes the numbers sacred for compatibility. gRPC's
standout feature is **streaming**: server-, client-, and bidirectional streams are
first-class call types, which is why it fits live audio so naturally.
</div>

The previous lesson gave you RPC the idea; this one gives you RPC as actually
practiced. gRPC (from Google, now open) and its wire format Protocol Buffers are
the pairing you'll meet everywhere from microservices to — in
[Unit 6](/learn/apis/streaming-audio-with-grpc/) — a scanner daemon streaming
call audio.

## The .proto file: one contract, written once

Everything starts from an interface definition. Here's a miniature of a
scanner-flavoured service:

```proto
syntax = "proto3";

package scanner.v1;

message CallRequest {
  uint32 call_id = 1;
}

message AudioChunk {
  uint32 call_id   = 1;
  bytes  pcm       = 2;  // raw audio samples
  uint64 timestamp = 3;
}

service AudioService {
  // One request in, a stream of chunks back.
  rpc StreamAudio(CallRequest) returns (stream AudioChunk);
}
```

Read it as a contract in the [Unit 1 sense](/learn/apis/api-contracts/), made
machine-checkable: **messages** are typed structures, **services** are sets of
**rpc** methods with request and response types. The `= 1`, `= 2` are **field
numbers**, and they're the heart of protobuf — more on them in a moment.

From this file, `protoc` (the protobuf compiler) generates code: Go structs and
interfaces, Python classes, TypeScript, whatever you target. The generated
**stub** is the one from [last lesson's diagram](/learn/apis/what-is-rpc/), now
concrete: your Go client gets a method with the exact signature the `.proto`
promised, [context](/learn/programming-go/context-and-cancellation/)-first for
deadlines and cancellation.

## Protobuf on the wire: numbers, not names

JSON spells out `"talkgroup"` in every message. Protobuf writes each field as
its **number** plus a compactly-encoded value — `talkgroup: 1201` costs about
three bytes. The consequences cut both ways:

- **Small and fast.** No field-name bytes, no text-to-number parsing; encoding
  and decoding are near-memcpy for much of a message.
- **Opaque.** The bytes are unreadable without the schema — the
  [text-vs-binary trade](/learn/apis/text-vs-binary-protocols/) in its purest
  form.
- **The numbers are the contract.** Renaming a field is *free* (names never hit
  the wire!), but **reusing or changing a field number is the protobuf cardinal
  sin** — old messages would silently decode into the wrong field: the
  worst-case "meaning change" from the
  [contracts lesson](/learn/apis/api-contracts/). Deleted fields' numbers are
  retired forever (protobuf even has a `reserved` keyword for it). Unknown
  fields are skipped by decoders, so **adding fields is safe** — protobuf bakes
  the additive-evolution discipline into the format itself.

## Four call shapes, streaming built in

Where classic RPC had only request/response, gRPC methods come in four shapes:

| Shape | Signature sketch | Example |
|-------|------------------|---------|
| **Unary** | request → response | Fetch one talkgroup's details |
| **Server streaming** | request → **stream** of responses | `StreamAudio` above: subscribe once, chunks flow until the call ends |
| **Client streaming** | **stream** of requests → response | Upload samples, get one analysis back |
| **Bidirectional** | stream ↔ stream | Live session: commands up, telemetry down, interleaved |

Streams ride HTTP/2 framing underneath, with flow control — the transport-level
half of the [backpressure story](/learn/apis/streaming-and-backpressure/) — plus
per-call **deadlines**, **cancellation**, and a standard **status code**
vocabulary. This is the "sharp edges made explicit" promise kept: the failure
modes RPC hides are surfaced as API you're forced to touch.

## When gRPC, when REST?

gRPC shines where both ends are programs you (or your ecosystem) control:
service-to-service traffic, streaming, tight typed contracts, performance-
sensitive paths. Its friction is at the edges: browsers can't speak native gRPC
(proxies like gRPC-Web exist), and you lose curl-ability — debugging means
`grpcurl` and schema-aware tooling rather than eyeballs on text. Hence the
by-now-familiar split, which GopherTrunk itself follows: **REST for records and
public surface, gRPC for the audio stream** where binary framing and server
streaming pay their way.

> Rule of thumb: gRPC between services, REST at the edge — and never change a
> protobuf field number.

<div class="knowledge-check" data-quiz data-correct-msg="Right — names never touch the wire, but the numbers are the wire format, so reusing one silently corrupts meaning." markdown="0">
  <p class="knowledge-check__q">Quick check: which change to a stable .proto message is safe?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Reusing a deleted field's number for a new field</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Changing an existing field's number to tidy the ordering</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Renaming a field while keeping its number</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The **`.proto` file** defines messages and services once; **generated stubs**
  give every language a typed client and server — the schema-first workflow.
- Protobuf encodes **field numbers, not names**: compact and fast, opaque
  without the schema.
- **Field numbers are sacred** — rename freely, never renumber or reuse; adding
  fields is safe by design.
- gRPC offers **unary, server-, client-, and bidirectional streaming** calls
  with deadlines, cancellation, and flow control built in.
- Use **gRPC between services and for streams, REST at the public edge** — the
  split real systems, GopherTrunk included, actually practice.

Next up: [Text vs binary protocols](/learn/apis/text-vs-binary-protocols/).
