---
slug: what-is-rpc
title: What is RPC?
description: Remote procedure calls make a network call look like a function call — the oldest idea in distributed computing, why it's seductive, and the sharp edges (partial failure, latency) the abstraction hides.
keywords: RPC, remote procedure call, RPC vs REST, stubs, network transparency, partial failure, distributed systems basics
level: intermediate
status: full
faq:
  - q: What does RPC stand for?
    a: RPC stands for Remote Procedure Call. It is the idea of invoking a function that runs on another machine as if it were local — your code calls something like GetTalkgroup(1201), and behind the scenes the arguments are serialized, sent over the network, executed remotely, and the result is sent back and returned to your code.
  - q: What is the difference between RPC and REST?
    a: REST models an API as resources (nouns) named by URLs and manipulated with a fixed set of HTTP verbs; RPC models an API as procedures (verbs) you call with arguments. REST leans on HTTP's shared conventions — caching, status codes, uniform methods — while RPC gives you freedom to define any operation. Actions that aren't naturally create/read/update/delete on a thing often fit RPC better.
  - q: Why is RPC considered to have sharp edges?
    a: "Because a network call only *looks* like a function call. A local call either runs or the whole program has crashed; a remote call can fail in a third way — you sent a request and got no answer, and you cannot know whether it executed. Remote calls are also thousands of times slower and can fail independently. Code written as if remote calls were local breaks in exactly these gaps."
---

# What is RPC?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**RPC (remote procedure call)** makes a network exchange look like an ordinary
**function call**: `client.GetTalkgroup(1201)` serializes arguments, ships them to
a server, runs the real function there, and returns the result — with generated
**stubs** hiding the plumbing. The seduction is real productivity; the danger is
that the network never fully disappears: **partial failure** (no answer — did it
run?), **latency** (a million times a local call), and **independent failure** are
sharp edges the function-call syntax hides. Modern RPC keeps the syntax and makes
the edges explicit.
</div>

Unit 4 changes vocabulary. REST asked "what resource, which verb?"; RPC asks the
older, more direct question: "what if calling code on another machine just looked
like… calling code?" That idea predates the web itself, powers
[gRPC](/learn/apis/grpc-and-protobuf/) today, and carries a famous lesson about
leaky abstractions.

## The idea: verbs, not nouns

Some operations are naturally procedures. GopherTrunk's daemon has plenty:
"retune the SDR to 460.4 MHz," "start hunting for control channels," "stream me
this call's audio." Squeezing those into
[resource semantics](/learn/apis/rest-fundamentals/) gets awkward — is a retune a
`PATCH` on a radio resource? a `POST` to `/retunes`? RPC drops the pretense: an
API is a set of **named operations with typed parameters and results**, like a
library's interface that happens to live across a network. If you've written Go
[interfaces](/learn/programming-go/interfaces/), an RPC service definition will
feel immediately familiar.

## How the trick works

Neither your code nor the server's handler touches a socket. Generated glue does:

1. Your code calls a **client stub** — a local function with the right signature.
2. The stub **serializes** the arguments ([Unit 1's machinery](/learn/apis/data-formats/)),
   sends them over the wire, and waits.
3. A **server stub** deserializes, calls the *real* implementation, and
   serializes the return value back.
4. Your stub returns the result, and your code continues, apparently having made
   a plain function call.

The stubs are typically generated from a shared interface definition — the
subject of [schemas & code generation](/learn/apis/schemas-and-codegen/) — which
is how both sides stay in agreement about signatures.

## The abstraction leaks — famously

The dream version of RPC was called **network transparency**: remote calls
indistinguishable from local ones. Decades of distributed-systems experience
turned that dream into a cautionary tale, for three reasons that no syntax can
hide:

- **Partial failure.** A local call either returns or your whole process is
  gone. A remote call has a third outcome: **silence**. Request sent, no reply —
  and you *cannot know* whether the server executed it before the network
  failed. Retry a non-[idempotent](/learn/apis/methods-and-status-codes/)
  operation on silence and you may run it twice; don't retry and it may never
  have run. Every robust RPC client is built around this ambiguity — timeouts,
  retry budgets, idempotent designs.
- **Latency.** A local call costs nanoseconds; a same-city network round trip
  costs milliseconds — a factor of ~a million. Code that innocently calls a
  remote function in a loop, as function-call syntax invites, performs
  catastrophically. Remote calls want batching and streaming.
- **Independent failure.** The server can crash, restart, or upgrade while your
  process lives on — mid-call. Versioning ([Unit 2's lesson](/learn/apis/api-versioning/)
  applies fully) and reconnection are permanent concerns.

> Rule of thumb: enjoy RPC's syntax, but never *believe* it. Every remote call
> is a network message wearing a function costume — give it a timeout, decide
> its retry story, and never put it in a tight loop.

Modern frameworks internalized the lesson: gRPC calls carry explicit deadlines,
status codes, and cancellation (Go programmers will recognise
[context](/learn/programming-go/context-and-cancellation/) plumbed through every
generated signature) — the sharp edges surfaced in the API instead of hidden.

## RPC vs REST: coexistence, not war

| Fits RPC | Fits REST |
|----------|-----------|
| Actions and commands ("retune", "start hunt") | Records and collections (talkgroups, call history) |
| Streaming, low-latency internal traffic | Public APIs where curl-ability and caching matter |
| Typed contracts between services you control | Loose coupling with unknown third-party clients |

Most real systems use both, split along exactly these lines — GopherTrunk serves
REST for its records and gRPC for audio streaming, as
[Unit 6](/learn/apis/streaming-audio-with-grpc/) shows in the flesh.

<div class="knowledge-check" data-quiz data-correct-msg="Right — silence is the third outcome, and the caller cannot know whether the remote side executed." markdown="0">
  <p class="knowledge-check__q">Quick check: what failure mode does a remote call have that a local function call does not?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It can return a wrong answer, which local calls never do</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It can produce silence — no reply, with no way to know whether it executed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It can throw an exception the caller must catch</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **RPC** models an API as **named procedures** — verbs with typed parameters —
  rather than resources and uniform methods.
- Generated **stubs** on both sides hide serialization and transport, keeping
  application code looking like plain function calls.
- The abstraction leaks: **partial failure** (silence — did it run?),
  **latency** (~10⁶× a local call), and **independent failure** are irreducible.
- Robust RPC means **timeouts, deadlines, retry policies, and idempotent
  designs** — modern frameworks make these explicit rather than hidden.
- RPC and REST **coexist**: commands and streams lean RPC; records and public
  surfaces lean REST.

Next up: [gRPC & Protocol Buffers](/learn/apis/grpc-and-protobuf/).
