---
slug: apis-and-integration
title: APIs & talking to other software
description: How programs talk to each other through APIs — a contract that lets one piece of software use another without knowing its internals, from in-process library calls to REST over HTTP with JSON, plus gRPC, WebSockets, and the discipline of not breaking your callers.
keywords: API, application programming interface, REST, HTTP, JSON, web API, library API, endpoint, gRPC, WebSocket, integration, webhook
level: intermediate
status: full
prereq:
  - abstraction-coupling
faq:
  - q: "What is an API?"
    a: "An API (application programming interface) is a defined set of requests one program offers to others — the inputs it accepts, the outputs it returns, and the rules for using it. It's a contract that lets you use software without knowing how it works inside, exactly like the \"program to an interface\" idea applied between programs."
  - q: "What is REST?"
    a: "REST is a common style for web APIs. It models things as resources, each with its own URL, and uses standard HTTP methods as verbs — GET to read, POST to create, PUT to update, DELETE to remove. It's popular because it leans on plumbing the web already has rather than inventing its own."
  - q: "What's the difference between a library API and a web API?"
    a: "A library API is a set of functions you call directly inside your own process — fast, in-memory, same machine. A web API is a set of requests sent over the network to another machine or service — slower, and able to fail in ways a local call can't. Same idea (a published contract), different distance."
  - q: "What format do APIs use?"
    a: "Web APIs most often exchange JSON over HTTP — a lightweight, human-readable text format for structured data. Other formats exist (XML, or the compact binary of gRPC), but JSON over HTTP is the everyday default for REST APIs."
gophertrunk_links:
  - title: Web UI
    url: /web.html
    note: GopherTrunk's browser UI is a client that talks to the running daemon over an API.
---

# APIs & talking to other software

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **API** is a contract that lets one piece of software use another **without
knowing its internals** — you depend on the published *what*, not the *how*.
It's the same "[program to an interface](/learn/intro-software-dev/abstraction-coupling/)"
idea, applied *between* programs. Most web APIs are **REST**: requests over
**HTTP**, data as **JSON**. Get the contract right and two programs written by
two teams, in two languages, on two machines, can cooperate.
</div>

You've seen that a good [abstraction](/learn/intro-software-dev/abstraction-coupling/)
lets one part of a program use another through a clean interface, ignoring the
mess behind it. An API is that exact idea stretched across a boundary — between
libraries, between programs, between whole companies. It's how your code borrows
a mapping service, how a browser page talks to a server, and how GopherTrunk's UI
drives the decoder running underneath it. Learn to read and design APIs and most
of modern software stops looking like magic and starts looking like parts that
agreed on how to talk.

## What an API is

An **API — application programming interface** — is a defined set of requests one
program offers to others. It spells out what you can ask for, what inputs each
request needs, what you'll get back, and the rules for using it. That's all a
contract is: **inputs, outputs, and rules**, with the implementation hidden behind
them.

The worn-out analogy earns its keep here: an API is a **restaurant menu**. It lists
what you can order and what each dish costs, and you order by name. You don't walk
into the kitchen, and you don't need to — the menu is the interface, the kitchen is
the hidden *how*. The chef can change recipes, swap suppliers, or rebuild the whole
kitchen, and as long as the menu still means what it said, you never notice. Break
the menu — rename a dish, quietly change what it contains — and every regular who
ordered it is surprised.

## Library APIs vs. web APIs

The word "API" covers two distances.

A **library (or module) API** is a set of functions you call **inside your own
process**. You import the library, call `parseDate(...)` or `filter.Process(...)`,
and control returns to you microseconds later. Same machine, same memory, no
network. The contract is the function signatures and their documented behavior.

A **web (or network) API** is a set of requests sent **over the network** to another
machine or service. You don't call a function; you send a message — typically an
HTTP request — and wait for a reply that has to travel there and back. Same idea, but
the distance changes everything downstream: it's slower, and it can **fail** in ways a
local call never does (the network drops, the other service is down, the reply is
late). We'll come back to that under integration.

Same concept, then — a published contract you program against — just separated by a
function call in one case and a network hop in the other.

## HTTP in one minute

Most web APIs ride on **HTTP**, the same protocol your browser uses. HTTP is a simple
**request/response** exchange: a client sends a request, the server sends one response.
A request carries a few key things:

- A **method** — the verb. `GET` reads, `POST` creates, `PUT`/`PATCH` update, `DELETE`
  removes.
- A **URL / endpoint** — the address of the thing you're acting on, e.g.
  `/systems/42/talkgroups`.
- Optional **headers** (metadata, like auth keys or content type) and a **body** (the
  data you're sending, on POST/PUT).

The response comes back with a **status code** that says how it went at a glance:
**2xx** success (200 OK), **4xx** the caller got it wrong (404 Not Found, 401
Unauthorized), **5xx** the server broke (500 Internal Server Error). Learn to read the
status code first — it tells you whether the problem is on your side or theirs before
you read a single line of the body.

## REST & JSON

**REST** is the most common style for arranging a web API. Two ideas carry most of it:

- **Resources are URLs.** Every "thing" the API exposes — a system, a talkgroup, a
  recording — gets its own address.
- **HTTP methods are the verbs.** You `GET` a resource to read it, `POST` to create
  one, `DELETE` to remove it. You don't invent `/getSystem` and `/deleteSystem`
  endpoints; you use `GET /systems/42` and `DELETE /systems/42`.

The data going back and forth is almost always **JSON** — JavaScript Object Notation,
a lightweight text format of key/value pairs and lists that's readable by humans and
trivial for any language to parse. A tiny exchange looks like this:

```http
GET /systems/42 HTTP/1.1
Host: api.example.org
Accept: application/json

HTTP/1.1 200 OK
Content-Type: application/json

{
  "id": 42,
  "name": "Metro P25",
  "type": "P25",
  "control_channels_hz": [851000000, 851500000],
  "active": true
}
```

That's the whole shape of most web integration: a URL, a method, and JSON in and out.
Nothing about it reveals whether the server is written in Go, Python, or Rust — which
is precisely the point.

## Other shapes

REST-over-HTTP is the default, not the only option. A few others you'll meet:

- **gRPC** — fast, strongly typed, compact binary messages defined by a shared schema.
  Reached for in **service-to-service** communication inside a system, where speed and
  a strict contract matter more than being human-readable in a browser.
- **WebSockets** — a persistent, two-way connection that stays open so either side can
  send at any time. Reached for **live/streaming** data: chat, dashboards, a UI that
  needs a running feed of updates rather than one reply per request.
- **GraphQL** — the client sends a query describing **exactly the fields it wants**, and
  gets back that shape and no more. Reached for when clients have varied data needs and
  you want to avoid a proliferation of narrow endpoints.

Each is a different trade, not a strict upgrade. Pick the one whose strengths match the
job: request/reply reads (REST), tight internal calls (gRPC), live streams (WebSockets),
flexible client-shaped reads (GraphQL).

## Integration: consuming and providing

Working with APIs runs in two directions.

**Consuming** — you call *someone else's* API. GopherTrunk might query an online radio
database to turn a bare system ID into a human name. You read their docs, send requests,
and handle the responses. Three realities come with this:

- **Auth and keys.** Many APIs require an API key or token identifying you. Treat it like
  a password — never commit it to source control.
- **Rate limits.** Providers cap how often you may call. Exceed it and you get a 429; a
  polite client backs off and retries later rather than hammering.
- **Networks fail.** A remote call is not a guaranteed function return. It can time out,
  return a 500, or hand back malformed data. Assume every call can fail and handle it —
  this is exactly the
  [defensive-programming](/learn/intro-software-dev/robustness-and-errors/) discipline,
  now aimed at the network.

**Providing** — you *expose* your own API for others to call. A local daemon might offer
an API that a web UI talks to. Now you're on the menu side of the contract: you decide the
endpoints, document them, and — crucially — keep your promises so you don't break the
programs that came to depend on you.

## Stability & versioning

An API is a **promise**, and other people build on it. The moment someone calls your
endpoint, its shape is no longer yours alone to change freely — renaming a field or
dropping an endpoint **breaks every caller** that relied on it, often silently and at a
distance.

So published APIs are managed like releases. **Backward compatibility** — adding new
things without breaking existing ones — is the default expectation. When a change truly
can't stay compatible, it goes into a new **version** (`/v1/…` alongside `/v2/…`) so old
clients keep working while new ones move over. This is the same reasoning behind
semantic versioning: a breaking change is a big deal you signal loudly, not something you
slip out quietly. The
[architectural patterns](/learn/intro-software-dev/architectural-patterns/) lesson picks
up how these contracts shape whole systems.

## A worked example

GopherTrunk is a small tour of both sides of this.

Internally, the decoder runs as a **daemon** — a long-lived process doing the DSP and
trunk-following. The [browser UI](/web.html) is a **separate program** that never touches
that code directly; it's a client that talks to the daemon over an **API**. Ask the UI to
retune, show live meters, or list active talkgroups, and under the hood it's sending
requests to the daemon and rendering the responses. The two are loosely coupled across a
contract — you could swap the UI for a script and the daemon wouldn't know or care.

Externally, the app can reach *out* to an online database to **identify systems** — turning
a raw system or talkgroup ID into a recognizable name. That's a web API it *consumes*, with
someone else's keys, rate limits, and occasional outages to handle gracefully.

Two APIs, then: one **internal** (UI ↔ daemon, a contract GopherTrunk provides) and one
**external** (GopherTrunk ↔ a public database, a contract it consumes). Same idea at both
ends — programs cooperating through a published interface instead of reaching into each
other's internals.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a WebSocket keeps a connection open so the server can push a continuous, two-way stream of updates." markdown="0">
  <p class="knowledge-check__q">Quick check: a browser UI needs live updates from a running local program. Which fits a continuous two-way stream best?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A single REST GET request</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A WebSocket connection</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Deleting the endpoint</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **API** is a contract — inputs, outputs, rules — that lets one program use another
  without knowing its internals. It's "program to an interface" between programs.
- A **library API** is functions you call in-process; a **web API** is requests over the
  network. Same idea, different distance — and the network can fail.
- **HTTP** is request/response with methods (GET/POST/PUT/DELETE), endpoints, and status
  codes (2xx/4xx/5xx). It's the backbone of web APIs.
- **REST** models resources as URLs and uses HTTP methods as verbs; **JSON** is the usual
  data format. gRPC, WebSockets, and GraphQL suit tighter, streaming, or client-shaped needs.
- **Integration** runs both ways — consuming others' APIs (keys, rate limits, error
  handling) and providing your own.
- An API is a **promise**; breaking it breaks callers, so favor backward compatibility and
  **versioning**.

Next up: the unglamorous work that keeps software alive — [documentation, tech debt & maintenance](/learn/intro-software-dev/docs-and-maintenance/).
