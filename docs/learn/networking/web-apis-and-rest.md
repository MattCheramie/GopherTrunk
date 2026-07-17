---
slug: web-apis-and-rest
title: Web APIs & REST
description: What a web API is and how REST works — resources addressed by URL endpoints, HTTP methods as verbs, stateless requests, JSON request and response bodies, API keys and rate limits, and how GopherTrunk's web interface talks to the daemon over an API.
keywords: web API, REST, JSON, endpoint, HTTP API, API key, request response, integration, HTTP methods, rate limit
level: intermediate
status: full
prereq:
  - http
faq:
  - q: What is a web API?
    a: "A web API is an interface a service exposes over HTTP so other programs — not just browsers — can use it. It's the same request/response you already know, but the client is code instead of a person clicking links. A program sends a request to a defined address and gets back structured data, usually JSON, it can act on."
  - q: What is REST?
    a: "REST is the most common style for building web APIs. It models everything as resources addressed by URLs (endpoints), uses HTTP methods as verbs — GET to read, POST to create, PUT to update, DELETE to remove — and keeps each request stateless, meaning the request carries everything the server needs. It's popular because it reuses HTTP as-is."
  - q: How does a web API authenticate callers?
    a: "Most web APIs identify the caller with an API key or token sent in a request header, such as Authorization. The server checks it, decides what the caller may do, and often counts requests against a rate limit. Keep keys in headers rather than the URL, where they can leak into logs and browser history."
  - q: What data format do web APIs use?
    a: "JSON is the usual format for both request and response bodies — a lightweight, text-based structure of keys and values that any language can read and write. The API describes which fields it expects and which it returns, so client and server agree on the shape of the data."
gophertrunk_links:
  - title: Web interface
    url: /web.html
    note: GopherTrunk's web UI is a client talking to the running daemon over an API.
---

# Web APIs & REST

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **web API** lets programs talk to each other over
[HTTP](/learn/networking/http/) — the same request/response you already know, but
the client is code instead of a browser. **REST** is the common style for doing it:
**resources** addressed by URL **endpoints**, HTTP methods as verbs, and stateless
requests. The data going back and forth is almost always **JSON**.
</div>

You've seen how a browser fetches a page over HTTP. A web API is the same machinery
pointed at a different consumer: another program. Instead of returning HTML for a
human to read, it returns structured data for code to act on.

## What a web API is

A **web API** (Application Programming Interface) is an interface a service exposes
over [HTTP](/learn/networking/http/) so other programs can use it. Nothing new is
happening at the network level — it's the same request goes out, response comes back
cycle you learned. The difference is who's on the client end: not a person clicking
links, but software making calls and reading the reply.

That reply is designed for machines. A web page hands you HTML dressed up for the eye;
an API hands you clean, structured data — a list of records, a status, an object —
that a program can loop over, store, or display however it likes.

## REST

**REST** (Representational State Transfer) is the most common style for building web
APIs. It doesn't invent a new protocol — it uses HTTP the way HTTP was meant to be
used, which is why it's so widespread. Three ideas carry most of it:

- **Resources addressed by URLs.** Every thing the API deals with — a user, a system,
  a talkgroup — has an address called an **endpoint**, like `/systems` or
  `/systems/42`.
- **HTTP methods as verbs.** The method says what you want to do: **GET** to read,
  **POST** to create, **PUT** to update, **DELETE** to remove. The same URL behaves
  differently depending on the verb.
- **Stateless requests.** Each request carries everything the server needs to handle
  it; the server doesn't rely on remembering your previous call.

A couple of example endpoints make the pattern clear:

```
GET  /systems          → list all systems
GET  /systems/42       → read the system with id 42
POST /systems          → create a new system
```

Read the endpoint plus the verb together and the intent is obvious — that readability
is the point.

## JSON

**JSON** (JavaScript Object Notation) is the usual data format for both the request
and the response body. It's lightweight, text-based, and every language can read and
write it, which is why it won. A response from `GET /systems/42` might look like this:

```json
{
  "id": 42,
  "name": "Metro P25",
  "control_channel_hz": 851000000,
  "active": true
}
```

Keys name the fields, values hold the data, and the structure nests as needed. The API
documents which fields it expects on the way in and which it returns on the way out, so
client and server agree on the shape.

## Auth & limits

Most APIs need to know who's calling. The common approach is an **API key** or token
sent in a request **header** — often `Authorization` — that the server checks before
doing anything. Put the key in a header, never in the URL, where it can leak into
server logs, browser history, and bookmarks.

Two more things you'll meet quickly:

- **Rate limits.** Many APIs cap how many requests you may send per minute or per day.
  Exceed it and you're told to slow down rather than served.
- **Errors via status codes.** APIs lean on the same
  [HTTP status codes](/learn/networking/http/) as the web: `200` for success, `400`
  for a bad request, `401` when your key is missing or wrong, `429` when you've hit a
  rate limit, `500` when the server itself failed.

## In practice

You'll sit on both sides of this. Sometimes you **consume** someone else's API — for
example, looking up a system or talkgroup in an online radio-reference database and
pulling back JSON to fill in names. Other times you **expose** your own — a web UI in
the browser calling a backend to fetch and update data.

It's the same idea the software-development path covers from the builder's angle in
its [APIs & integration](/learn/intro-software-dev/apis-and-integration/) lesson: a
defined contract that lets two programs work together without knowing each other's
internals.

## GopherTrunk example

GopherTrunk's [web interface](/web.html) is a web API client. The page you open in the
browser is the front end; the running daemon is the back end, and the two talk over an
API. When the UI shows live activity or lets you change a setting, it's making requests
to the daemon and rendering the JSON that comes back — exactly the client/server split
described above.

GopherTrunk can also act as an API consumer, querying online databases to identify
systems and talkgroups so decoded traffic shows names instead of bare numbers.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in REST, each resource lives at its own URL, called its endpoint." markdown="0">
  <p class="knowledge-check__q">Quick check: in a REST API, what typically identifies the resource you're acting on?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The JSON field named "id" only</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Its URL / endpoint</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The API key in the header</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **web API** lets programs talk to each other over HTTP; the client is code, not a
  browser.
- **REST** models everything as **resources** at URL **endpoints** and uses HTTP
  methods as verbs — GET, POST, PUT, DELETE — with stateless requests.
- **JSON** is the usual format for request and response bodies.
- Authenticate with an **API key** in a header, mind **rate limits**, and read
  **status codes** for success and errors.
- You both consume others' APIs and expose your own — GopherTrunk's web UI is a client
  talking to the daemon over an API.

Next up: [WebSockets & real-time connections](/learn/networking/websockets-and-realtime/)
