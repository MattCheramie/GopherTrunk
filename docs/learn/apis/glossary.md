---
slug: glossary
title: Glossary of API & protocol terms
description: Plain-language definitions of the terms used across the APIs & Protocols module — API, endpoint, REST, HTTP method, status code, token, WebSocket, SSE, webhook, RPC, protobuf, framing, backpressure, rate limit, and more — each cross-linked to the lesson that explains it.
keywords: API glossary, protocol terms, REST glossary, HTTP terms, WebSocket definition, SSE, webhook, gRPC, protobuf, backpressure, rate limiting, idempotent
level: beginner
status: full
lesson_standalone: true
---

# Glossary of API & protocol terms

Every term used across the [APIs &amp; Protocols](/learn/apis/) module, defined in
plain language and linked to the lesson where it's explained in full. Skim it as a
refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are
grouped by theme, roughly in the order the module introduces them.

> Looking for radio and scanning terms instead? The site's
> [Field Guide reference](/reference/) covers the trunking, DSP, and SDR
> vocabulary GopherTrunk itself is built on.

## Foundations

**API (application programming interface)** — A published promise about how one
piece of software may use another: the requests it accepts and the answers it
gives. See [What is an API?](/learn/apis/what-is-an-api/)

**Protocol** — Rules two parties agree on in advance — format, sequence, meaning,
error handling — so exchanged bytes mean the same thing to both. See
[What is a protocol?](/learn/apis/what-is-a-protocol/)

**Client** — The party that initiates: it opens the connection and sends the
request when it wants something. See
[Clients and servers](/learn/apis/clients-and-servers/)

**Server** — The party that waits at a known address and answers whatever
arrives; a role per conversation, not a kind of machine. See
[Clients and servers](/learn/apis/clients-and-servers/)

**Peer-to-peer** — A design where every participant can both initiate and
answer, with no fixed server role. See
[Clients and servers](/learn/apis/clients-and-servers/)

**Serialization** — Flattening an in-memory data structure into an agreed byte
format for the trip between programs; parsing reverses it. See
[Data formats: JSON and friends](/learn/apis/data-formats/)

**JSON** — The web's default data format: human-readable text built from
objects, arrays, strings, numbers, booleans, and null. See
[Data formats: JSON and friends](/learn/apis/data-formats/)

**Schema** — The field-level agreement about a message's structure and types —
which a format like JSON does not enforce by itself. See
[Data formats: JSON and friends](/learn/apis/data-formats/)

**Contract** — Everything a consumer may rely on: endpoints, shapes, meanings,
and error behaviour; it outlives the implementation behind it. See
[API contracts](/learn/apis/api-contracts/)

**Breaking change** — Any change that can make a correct existing client stop
working — removal, renaming, type change, or (worst) a silent meaning change.
See [API contracts](/learn/apis/api-contracts/)

**Additive change** — The safe evolution direction: adding optional fields and
new endpoints without removing or repurposing anything. See
[API contracts](/learn/apis/api-contracts/)

**Tolerant reading** — The client-side half of additive evolution: ignore
fields and event types you don't recognise. See
[API contracts](/learn/apis/api-contracts/)

## HTTP & REST

**HTTP** — The web's request/response protocol: a request line, headers, a
blank line, and a body — mirrored by a status line on responses. See
[Anatomy of an HTTP request](/learn/apis/anatomy-of-http/)

**Header** — A `Name: value` line carrying metadata about an HTTP exchange —
format, size, credentials, negotiation. See
[Anatomy of an HTTP request](/learn/apis/anatomy-of-http/)

**Body** — The payload bytes after an HTTP message's blank line, in the format
its `Content-Type` header declares. See
[Anatomy of an HTTP request](/learn/apis/anatomy-of-http/)

**REST** — The convention of modelling an API as resources named by URLs,
manipulated with the small fixed set of HTTP methods. See
[REST fundamentals](/learn/apis/rest-fundamentals/)

**Resource** — A "thing" in an API's domain — a talkgroup, a call — addressed
by its own URL. See [REST fundamentals](/learn/apis/rest-fundamentals/)

**Representation** — The serialized snapshot of a resource that actually
travels — usually JSON, negotiated via `Content-Type` and `Accept`. See
[REST fundamentals](/learn/apis/rest-fundamentals/)

**Statelessness** — The REST discipline that each request carries everything
the server needs, with no conversation memory between requests. See
[REST fundamentals](/learn/apis/rest-fundamentals/)

**Endpoint** — One addressable operation of an API — a path (plus method) a
client can call, like `GET /api/v1/calls`. See
[REST fundamentals](/learn/apis/rest-fundamentals/)

**HTTP method** — The verb of a request — GET, POST, PUT, PATCH, DELETE — each
carrying a promise about what kind of action it is. See
[Methods & status codes](/learn/apis/methods-and-status-codes/)

**Safe (method)** — A method promising no change to server state; GET is the
canonical example. See
[Methods & status codes](/learn/apis/methods-and-status-codes/)

**Idempotent** — Repeatable without additional effect — the property that makes
retries safe; GET, PUT, and DELETE have it, POST does not. See
[Methods & status codes](/learn/apis/methods-and-status-codes/)

**Status code** — The three-digit verdict on a response: 2xx success, 3xx
redirection, 4xx client fault, 5xx server fault. See
[Methods & status codes](/learn/apis/methods-and-status-codes/)

**Query string** — The `?key=value&…` portion of a URL carrying options —
filters, sorting, paging — as opposed to identity. See
[URLs, query strings, and bodies](/learn/apis/urls-queries-and-bodies/)

**Percent-encoding** — Escaping structural characters (`%20`, `%26`) so
arbitrary data can travel inside a URL. See
[URLs, query strings, and bodies](/learn/apis/urls-queries-and-bodies/)

**Pagination** — Fetching a large collection in pages via query parameters —
offset-based or cursor-based. See
[URLs, query strings, and bodies](/learn/apis/urls-queries-and-bodies/)

**Authentication** — Establishing who is calling; distinct from authorization,
which establishes what they may do. See
[API authentication](/learn/apis/authentication-basics/)

**API key** — A long random string identifying one client or account,
presented on every request. See
[API authentication](/learn/apis/authentication-basics/)

**Bearer token** — A credential for which possession is proof — whoever
presents it is treated as its owner, hence TLS-only handling. See
[API authentication](/learn/apis/authentication-basics/)

**Versioning** — Running incompatible contracts side by side under version
markers (like `/api/v1/`) when a breaking change is unavoidable. See
[API versioning](/learn/apis/api-versioning/)

**Deprecation** — The polite retirement of an old API version: announce,
warn in-band, measure, then remove on the promised date. See
[API versioning](/learn/apis/api-versioning/)

## Real-time

**Polling** — Repeatedly asking the server whether anything changed; simple
and stateless, but wasteful and stale by up to one interval. See
[Polling vs push](/learn/apis/polling-vs-push/)

**Push** — Delivery initiated by the server the moment an event exists —
via an open connection or a callback. See
[Polling vs push](/learn/apis/polling-vs-push/)

**Long polling** — A hybrid where the server withholds its answer until it has
news, giving near-push latency over plain HTTP. See
[Polling vs push](/learn/apis/polling-vs-push/)

**WebSocket** — A persistent, full-duplex message pipe created by upgrading an
HTTP connection (`101 Switching Protocols`). See
[WebSockets](/learn/apis/websockets/)

**Full duplex** — Both sides may send at any time without waiting for the
other — the WebSocket's defining capability. See
[WebSockets](/learn/apis/websockets/)

**Keepalive (ping/pong)** — Periodic frames proving a long-lived connection is
still alive, and detecting silent death when the answer stops. See
[WebSockets](/learn/apis/websockets/)

**Reconnect backoff** — Waiting longer after each failed connection attempt —
and resetting that delay only when data actually arrives, never on a mere
handshake. See [WebSockets](/learn/apis/websockets/)

**Server-sent events (SSE)** — A one-way event stream carried as a single HTTP
response that never ends, in a simple text format. See
[Server-sent events](/learn/apis/server-sent-events/)

**EventSource** — The browser API for SSE, with automatic reconnection and
`Last-Event-ID` resume built in. See
[Server-sent events](/learn/apis/server-sent-events/)

**Last-Event-ID** — The header a reconnecting SSE client presents so the
server can replay events missed during the gap. See
[Server-sent events](/learn/apis/server-sent-events/)

**Webhook** — An HTTP callback: you register a URL and the service POSTs
events to it — the server becomes a client of you. See
[Webhooks](/learn/apis/webhooks/)

**At-least-once delivery** — The delivery guarantee retries produce: every
event arrives, but possibly more than once — so receivers must deduplicate.
See [Webhooks](/learn/apis/webhooks/)

**Signature (webhook)** — An HMAC over the delivery's raw body, proving the
event came from the real sender and not a forger. See
[Webhooks](/learn/apis/webhooks/)

**Backpressure** — A consumer's slowness pushed back up the pipeline; the
excess must be buffered, blocked on, or dropped — there is no fourth option.
See [Streaming & backpressure](/learn/apis/streaming-and-backpressure/)

**Bounded buffer** — A queue with a fixed capacity and an explicit policy for
when it fills; the unbounded alternative is a deferred crash. See
[Streaming & backpressure](/learn/apis/streaming-and-backpressure/)

**Drop-oldest / conflate** — Freshness-preserving drop policies for live data:
discard the stalest queued items, or keep only the latest state. See
[Streaming & backpressure](/learn/apis/streaming-and-backpressure/)

## RPC & binary

**RPC (remote procedure call)** — Making a network exchange look like a
function call, with generated stubs hiding the plumbing. See
[What is RPC?](/learn/apis/what-is-rpc/)

**Stub** — The generated local stand-in for a remote procedure, handling
serialization and transport on each side. See
[What is RPC?](/learn/apis/what-is-rpc/)

**Partial failure** — The remote call's third outcome — silence — where the
caller cannot know whether the operation executed. See
[What is RPC?](/learn/apis/what-is-rpc/)

**gRPC** — The mainstream RPC framework: `.proto`-defined services, generated
typed code, Protocol Buffers over HTTP/2, streaming built in. See
[gRPC & Protocol Buffers](/learn/apis/grpc-and-protobuf/)

**Protocol Buffers (protobuf)** — gRPC's compact binary encoding, writing
field numbers rather than names — which makes those numbers sacred. See
[gRPC & Protocol Buffers](/learn/apis/grpc-and-protobuf/)

**Field number** — A protobuf field's on-the-wire identity; renaming a field
is free, renumbering or reusing a number silently corrupts meaning. See
[gRPC & Protocol Buffers](/learn/apis/grpc-and-protobuf/)

**Server streaming** — A gRPC call shape: one request in, a stream of
responses back until the server ends it. See
[gRPC & Protocol Buffers](/learn/apis/grpc-and-protobuf/)

**Text protocol** — A protocol whose bytes are human-readable (HTTP/1.1,
JSON, SSE) — inspectable by any generic tool, at a cost in bytes. See
[Text vs binary protocols](/learn/apis/text-vs-binary-protocols/)

**Binary protocol** — A protocol trading readability for compactness and
cheap parsing — requiring schema-aware tooling to inspect. See
[Text vs binary protocols](/learn/apis/text-vs-binary-protocols/)

**Framing** — The receiver's rule for finding message boundaries in a raw
byte stream — delimiters, length prefixes, or fixed-size frames. See
[Message framing](/learn/apis/message-framing/)

**Length prefix** — Framing by stating each message's size up front: read N,
then read N bytes. See [Message framing](/learn/apis/message-framing/)

**Desynchronization** — The cascading failure after one misjudged boundary:
every later message parses at the wrong offset until resync. See
[Message framing](/learn/apis/message-framing/)

**Sync word** — A distinctive marker at each frame start that lets a receiver
re-find boundaries after corruption — radio protocols' framing lifeline. See
[Message framing](/learn/apis/message-framing/)

**OpenAPI** — The standard machine-readable schema language for REST APIs,
from which clients, servers, docs, and validators are generated. See
[Schemas & code generation](/learn/apis/schemas-and-codegen/)

**Code generation** — Producing typed clients, server scaffolding, and docs
from a schema, so the contract cannot silently drift from the code. See
[Schemas & code generation](/learn/apis/schemas-and-codegen/)

**Contract-first** — Authoring the schema as the design artifact and
generating code from it — rather than exporting a schema from whatever the
code does. See [Schemas & code generation](/learn/apis/schemas-and-codegen/)

## Designing & operating

**Surface area** — Everything an API publishes — each element a promise to
support forever, hence "when in doubt, leave it out." See
[Designing a good API](/learn/apis/designing-a-good-api/)

**Principle of least surprise** — The design goal that a user can correctly
guess the next endpoint from the last one, because conventions never vary.
See [Designing a good API](/learn/apis/designing-a-good-api/)

**Error code** — A short, stable, documented string in an error body that
programs branch on — never the prose message. See
[Error handling](/learn/apis/error-handling/)

**Problem details** — The standardised JSON error-body shape (RFC 9457) that
layers status, code, and human message. See
[Error handling](/learn/apis/error-handling/)

**Rate limit** — A cap on how fast a client may call, enforced with `429 Too
Many Requests`; a quota caps total volume per period. See
[Rate limiting & quotas](/learn/apis/rate-limiting-and-quotas/)

**Retry-After** — The response header telling a rate-limited or unavailable
client how long to wait before retrying. See
[Rate limiting & quotas](/learn/apis/rate-limiting-and-quotas/)

**Exponential backoff with jitter** — Retry etiquette: double the wait after
each failure and randomise it, so failed clients don't retry in crushing
synchronized waves. See
[Rate limiting & quotas](/learn/apis/rate-limiting-and-quotas/)

**Documentation drift** — Docs describing last year's API; defeated
structurally by generating reference from the schema and testing examples.
See [API documentation](/learn/apis/api-documentation/)

**Object-level authorization** — Checking that *this* identity may touch
*this* resource — the most commonly missed API security layer. See
[API security](/learn/apis/api-security/)

**Injection** — The attack class where composed-in user input stops being
data and starts being code (SQL, shell, path). See
[API security](/learn/apis/api-security/)

**Contract test** — A test asserting status codes, shapes, and error codes —
what clients rely on — independent of implementation. See
[Testing an API](/learn/apis/testing-an-api/)

**httptest** — Go's standard tooling for exercising real HTTP handlers
in-process, without ports or external servers. See
[Testing an API](/learn/apis/testing-an-api/)

**Failing-first test** — A regression test watched to fail on the bug before
the fix — the only proof it can catch that regression. See
[Testing an API](/learn/apis/testing-an-api/)

## GopherTrunk's APIs

**/api/v1/** — The daemon's versioned REST namespace: talkgroups, radio IDs,
call history, systems. See
[GopherTrunk's REST API](/learn/apis/the-daemon-rest-api/)

**Dogfooding** — Building the official client (the web console) on the public
API, guaranteeing the contract is complete and constantly exercised. See
[GopherTrunk's REST API](/learn/apis/the-daemon-rest-api/)

**Event stream** — The daemon's SSE feed of call and system events, which the
console's live panels and your scripts subscribe to. See
[Live events & webhooks](/learn/apis/live-events-and-webhooks/)

**Event type** — The contract vocabulary of a stream (`call.start`,
`call.end`, `system.status`) — dispatch on the known, ignore the new. See
[Live events & webhooks](/learn/apis/live-events-and-webhooks/)

**Audio streaming (gRPC)** — The daemon's server-streaming interface carrying
live call audio as binary chunks — Unit 4's arguments applied to voice. See
[Streaming audio with gRPC](/learn/apis/streaming-audio-with-grpc/)

**Reconnect storm** — The failure mode where clients whose backoff resets on
handshake hammer a refusing server at full speed forever — and the design
lessons that prevent it. See
[The web console's sockets](/learn/apis/web-console-sockets/)

**Liveness** — Distinguishing a quiet stream from a dead one — by connection
state, keepalives, and time-since-last-event thresholds. See
[Build your own client](/learn/apis/building-your-own-client/)
