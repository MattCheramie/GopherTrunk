---
slug: what-is-an-api
title: What is an API?
description: An API is a promise about how one piece of software may use another. Learn what an application programming interface really is, why the contract matters more than the code behind it, and where you meet APIs every day.
keywords: what is an API, application programming interface, API meaning, API for beginners, API contract, web API, REST API basics
level: beginner
status: full
faq:
  - q: What does API stand for?
    a: API stands for Application Programming Interface. It is the set of operations one piece of software offers to others — the functions you may call, the requests you may send, and the answers you can expect back. The word "interface" is the key part; an API describes the surface of a program, not its insides.
  - q: Is an API the same thing as a website?
    a: No, but they are close cousins. A website returns pages meant for humans to read in a browser; a web API returns structured data (usually JSON) meant for other programs to consume. Many services offer both over the same protocol — HTTP — and the same server often powers each. GopherTrunk's daemon is an example, serving a web console for people and a REST API for programs.
  - q: Do I need to know how to program to understand APIs?
    a: Not to understand them. The core idea — a published promise about how to ask for things and what you'll get back — is plain English. To *use* an API you eventually issue requests, but tools like curl let you do that from a terminal with no programming, and this module starts there before any code appears.
---

# What is an API?

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An **API** (application programming interface) is a **promise about how one piece of
software may use another**: which requests it accepts, in what format, and what it
sends back. The promise — the **contract** — is the important part; the code behind
it can change completely as long as the contract holds. A **web API** makes that
promise over the network, which is how a scanner daemon, a weather service, and a
payment processor can all be driven by programs that have never seen their source
code.
</div>

This is lesson 1, and its job is to make the word "API" concrete before we touch any
protocol. By the end you'll recognise APIs everywhere — in libraries, in operating
systems, and above all on the network — and you'll understand why the *interface* is
a bigger deal than the implementation behind it.

## What problem does an API solve?

Imagine GopherTrunk's scanner daemon: it tracks trunked radio systems, logs every
call, and records audio. Now imagine you want to build something on top of it — a
phone notification when your fire department keys up, a spreadsheet of the day's
busiest talkgroups. You shouldn't have to read the daemon's source code, and you
certainly shouldn't reach into its database files directly.

Instead, the daemon publishes an **interface**: "send me an HTTP request shaped like
*this*, and I will answer with call history shaped like *that*." Your program only
needs to honour that shape. The daemon's authors can rewrite the internals — new
database, new language, new algorithms — and your program keeps working, because the
promise at the boundary never changed. That separation of *what you can ask* from
*how it's done* is the entire point of an API.

## An API is a contract, not code

The most useful mental model is a **contract**:

- The provider promises: "if you ask like this, I'll answer like that."
- The consumer promises: "I'll only ask in the ways you documented."

Neither side needs to know anything else about the other. A contract is also what
makes an API *stable*: breaking it — renaming a field, changing what a request
means — breaks every consumer at once, which is why mature APIs treat their published
shape as nearly sacred. We give that idea a whole lesson in
[API contracts](/learn/apis/api-contracts/).

> Rule of thumb: if you can swap out the implementation without consumers noticing,
> you have a real API. If consumers depend on how it works inside, you have a leak.

## Where do APIs live?

The word covers several layers of software, and they're all the same idea at
different distances:

| Kind of API | Example | Who calls it |
|-------------|---------|--------------|
| **Library API** | Go's `net/http` package functions | Code in the same program |
| **Operating-system API** | "open a file", "create a socket" | Programs on the same machine |
| **Web API** | `GET /api/v1/calls` on a daemon | Programs anywhere on a network |

This module is mostly about the last row — **web APIs**, where the request travels
over a network using a protocol like HTTP. But keep the other rows in mind: when you
call a function in a Go package, you're trusting a contract in exactly the same way a
web client trusts a server.

## What does using a web API actually look like?

Here's a taste of what's coming. This is a real-shaped exchange with a scanner
daemon's API — a program asking for the list of talkgroups it knows about:

```text
GET /api/v1/talkgroups HTTP/1.1
Host: scanner.local:8080

HTTP/1.1 200 OK
Content-Type: application/json

{"talkgroups": [{"id": 1201, "label": "County Fire Dispatch"}]}
```

A request goes out with a **method** (`GET`) and a **path** (`/api/v1/talkgroups`);
a response comes back with a **status** (`200 OK`) and structured **data** (JSON).
Every piece of that line-by-line anatomy gets its own lesson in
[Unit 2](/learn/apis/anatomy-of-http/) — for now, notice how readable it is. Nothing
magic, just a conversation with rules.

## Who's on each side of an API?

An API always has two roles: something **exposing** the interface (the server, the
library, the service) and something **using** it (the client, the caller, the
consumer). One program routinely plays both parts — GopherTrunk's daemon *serves* an
API to its web console while *consuming* other interfaces underneath (the SDR
driver's, the operating system's). The next two lessons sharpen these roles: first
what a [protocol](/learn/apis/what-is-a-protocol/) is, then how
[clients and servers](/learn/apis/clients-and-servers/) split the work.

<div class="knowledge-check" data-quiz data-correct-msg="Exactly — the API is the published promise at the boundary, independent of the code behind it." markdown="0">
  <p class="knowledge-check__q">Quick check: a service rewrites its internals completely but keeps every documented request and response identical. What happened to its API?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The API changed, because the code changed</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The API is now a new version and old clients must update</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Nothing — the API is the contract, and the contract held</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- An **API** is a published **promise** about how one piece of software may use
  another — the requests it accepts and the answers it gives.
- The **contract** matters more than the implementation: internals can change freely
  as long as the interface holds.
- APIs exist at every layer — **library**, **operating system**, and **web** — and
  this module focuses on web APIs, where requests travel over a network.
- A web API exchange is just a structured conversation: method and path out, status
  and data back.
- Every API has a **provider** and a **consumer**, and most real programs play both
  roles at once.

Next up: [What is a protocol?](/learn/apis/what-is-a-protocol/).
