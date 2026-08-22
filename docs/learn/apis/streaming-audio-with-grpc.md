---
slug: streaming-audio-with-grpc
title: Streaming audio with gRPC
description: Live call audio rides a gRPC server stream — why binary RPC fits real-time voice, how a streaming call actually flows, and the deadline, teardown, and backpressure care streams demand.
keywords: gRPC audio streaming, live audio API, server streaming RPC, binary streaming, real-time audio, StreamAudio, gRPC deadlines
level: advanced
status: full
prereq:
  - grpc-and-protobuf
  - streaming-and-backpressure
gophertrunk_links:
  - title: API & events reference
    url: /api-events.html
    note: where the daemon's streaming interfaces are documented alongside the event surface.
---

# Streaming audio with gRPC

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Live call audio is the daemon's interface where **every Unit 4 argument lands at
once**: continuous **binary** data (base64-in-JSON would bloat and burn CPU), a
**server-streaming** shape (subscribe once, chunks flow until the call ends),
and machine-to-machine consumers — so it rides **gRPC**, not REST. Consuming a
stream well means honouring the **stream lifecycle**: read in a loop until the
server ends it, set **deadlines**, **cancel** what you stop consuming, and keep
up — or decide, per the backpressure lesson, what happens when you don't.
</div>

The REST API serves *records* of calls; the event stream says a call is
happening *now*. The third interface delivers the call itself — audio, live,
as it decodes. This lesson is Unit 4 made audible: why this surface is
binary RPC, and the client disciplines streaming demands.

## Why not just REST this?

Run the checklist from [text vs binary](/learn/apis/text-vs-binary-protocols/)
against live audio, and every line points one way. Audio is **continuous** — 
not a document with an end, so request/response is the wrong shape and
[SSE](/learn/apis/server-sent-events/) (a text transport) would mean base64
bloat on every chunk. It's **binary and relentless** — thousands of samples per
second for as long as someone holds a transmit button, where JSON encoding
costs real CPU on both ends forever. And it's **machine-to-machine** — the
consumer is an archiver, a re-streamer, an audio player, never human eyes on a
terminal. High-rate + binary + program-to-program is precisely the
[gRPC quadrant](/learn/apis/grpc-and-protobuf/); curl-ability, the strongest
REST argument, is worth nothing to a stream of PCM samples.

## The shape of the call

The interface is the `.proto` pattern you've already read — a **server
streaming** RPC: the client sends one subscribe request, the server responds
with a stream of audio chunks that flows until the call ends. In Go, the
generated client makes the lifecycle explicit:

```go
stream, err := client.StreamAudio(ctx, &pb.CallRequest{CallId: 48214})
if err != nil {
    return err
}
for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break // the call ended; server closed the stream cleanly
    }
    if err != nil {
        return err // mid-stream failure: network, deadline, server
    }
    player.Write(chunk.Pcm)
}
```

Read that loop as a protocol, because it is one: **`Recv` until `EOF`** is the
stream's contract — `EOF` here is not an error but the server saying "complete."
Note also what each chunk carries besides samples: identity and a timestamp,
because in streaming, **metadata rides with the data** — a chunk must be
self-describing enough to survive being buffered, logged, or split from its
neighbours.

## Streams demand different manners

Three disciplines separate a robust stream consumer from a demo:

- **Deadlines and cancellation.** That `ctx` is doing load-bearing work: it
  carries the call's **deadline** and your **cancellation** — hang up the
  moment you stop consuming (user closed the player, your buffer strategy gave
  up), because an abandoned-but-open stream makes the server encode and send
  audio nobody hears. gRPC propagates the cancel to the server, which stops
  work — the [RPC lesson's](/learn/apis/what-is-rpc/) "sharp edges made
  explicit" promise, and Go's
  [context machinery](/learn/programming-go/context-and-cancellation/) is
  exactly how it's plumbed.
- **Keep up, or choose your loss.** Audio is the
  [backpressure lesson's](/learn/apis/streaming-and-backpressure/) *perishable
  sample* class in its purest form: a chunk that arrives late for playback is
  worthless, and a client that reads slowly pushes backpressure up HTTP/2's
  flow control into the server. Decide the policy before shipping: for live
  listening, skip ahead (drop-oldest) and note the gap; for archiving, buffer
  generously — completeness beats latency there, the classification working
  as designed.
- **Mid-stream failure is normal.** A stream that dies at chunk 400 delivered
  399 real chunks first — reconnect-and-resume logic, and knowing whether your
  consumer is idempotent about overlapping audio, is part of the client, not
  an exception path.

> Rule of thumb: every stream you open, you own — deadline it, cancel it when
> you stop caring, and know your keep-up policy. Streams don't clean up after
> inattentive clients; they bill them, and their servers, quietly.

## One daemon, three doors — by design

Step back and Unit 6's architecture lesson is visible: **REST** for records
(text, cacheable, curl-able — its strengths), **SSE/webhooks** for event
notification (push, lightweight, text), **gRPC** for bulk real-time binary
(streams, deadlines, efficiency). Not indecision — *fit*: each surface uses
the protocol whose strengths its traffic actually exercises, which is the
mature answer to every "REST vs gRPC" argument you'll meet. Match the door to
the traffic, and let one system speak several protocols without apology.

<div class="knowledge-check" data-quiz data-correct-msg="Right — cancelling tells the server to stop producing; abandoning the stream leaves it encoding audio nobody consumes." markdown="0">
  <p class="knowledge-check__q">Quick check: your player window closes mid-call. What should your gRPC audio client do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — the stream ends on its own when the call does</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Keep reading chunks and discard them until EOF, to be polite</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Cancel the context — the server stops encoding and sending immediately instead of feeding a stream nobody hears</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Live audio is **continuous, binary, machine-to-machine** — the traffic
  profile gRPC **server streaming** exists for; REST/SSE would pay base64 and
  parsing forever.
- The client contract is a **`Recv`-until-`EOF` loop**; `EOF` means "complete,"
  and chunks carry **self-describing metadata**.
- **Deadline and cancel** every stream — abandonment without cancellation
  wastes server work on audio nobody hears.
- Audio is **perishable**: pick your keep-up policy (skip for live, buffer for
  archive) before the first drop, not after.
- One daemon, **three doors** — REST, events, gRPC — each protocol where its
  strengths meet the traffic: the architecture lesson of the whole unit.

Next up: [The web console's sockets](/learn/apis/web-console-sockets/).
