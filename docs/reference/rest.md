---
slug: rest
title: REST
entry_type: concept
category: testing-delivery
description: REST is an architectural style for web APIs in which clients act on named resources, identified by URLs, using standard HTTP verbs over a stateless protocol.
keywords: REST, RESTful, web API, HTTP verbs, resources, stateless, GET POST PUT DELETE, endpoints, JSON, Roy Fielding
aka: [REST "representational state transfer"]
autolink: true
infobox:
  - { label: Category, value: "Web API architectural style" }
  - { label: Stands for, value: "Representational State Transfer" }
  - { label: Built on, value: "HTTP" }
  - { label: Verbs, value: "GET, POST, PUT, PATCH, DELETE" }
  - { label: Key trait, value: "Stateless, resource-oriented" }
see_also: [api, error-handling, semantic-versioning, end-to-end-testing, integration-testing, version-control]
related_lessons:
  - { title: "Errors, edge cases & defensive programming", url: /learn/intro-software-dev/robustness-and-errors/ }
cite_urls:
  - https://en.wikipedia.org/wiki/REST
---

**REST** (Representational State Transfer) is an architectural style for web
[APIs](/reference/api/) in which clients act on named *resources*, identified by URLs,
using standard HTTP verbs over a stateless protocol.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A client sends HTTP verbs GET, POST, PUT, DELETE to a server's resource URLs." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="80" height="40" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="69">client</text>
    <line x1="100" y1="50" x2="340" y2="50" stroke="currentColor" stroke-width="1"/><text x="220" y="44" font-size="8">GET /messages</text>
    <line x1="100" y1="63" x2="340" y2="63" stroke="currentColor" stroke-width="1"/><text x="220" y="59" font-size="8">POST /messages</text>
    <line x1="100" y1="76" x2="340" y2="76" stroke="currentColor" stroke-width="1"/><text x="220" y="72" font-size="8">PUT /messages/42</text>
    <line x1="100" y1="89" x2="340" y2="89" stroke="currentColor" stroke-width="1"/><text x="220" y="85" font-size="8">DELETE /messages/42</text>
    <rect x="340" y="45" width="100" height="40" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.3"/><text x="390" y="63">server</text><text x="390" y="76" font-size="8">resources</text>
  </g>
</svg>
<figcaption>A REST client acts on resource URLs using standard HTTP verbs.</figcaption>
</figure>

## The core ideas

REST, described by Roy Fielding, organizes an API around a handful of constraints:[^wiki]

- **Resources and URLs.** Everything is a resource — a message, a user, a device — each
  identified by a URL like `/messages/42`. The URL names the *thing*, not the action.
- **Standard verbs.** The HTTP method says what to do: `GET` reads, `POST` creates, `PUT`
  / `PATCH` updates, `DELETE` removes. Reusing the protocol's verbs keeps the surface
  small and predictable.
- **Stateless.** Each request carries everything the server needs; the server keeps no
  per-client session between requests, which makes REST services easy to scale.
- **Representations.** A resource is transferred in some representation, today most often
  JSON, that the client and server agree on.

REST also leans on HTTP status codes (200, 404, 500) as its
[error-handling](/reference/error-handling/) contract, telling the client at the protocol
level whether a request succeeded.

## Why it's common

REST became the default web-API style because it builds on HTTP that every client and
server already speaks, requires no special tooling, and maps cleanly onto how people think
about data ("the message at this address"). Its statelessness makes services simpler to
cache, load-balance, and scale. Alternatives like GraphQL and gRPC exist and fit some
problems better, but REST remains the lingua franca of web services.

## In practice

A REST API is still an [API](/reference/api/), so its stability is a promise to callers:
breaking changes warrant a new major version under
[semantic versioning](/reference/semantic-versioning/), often expressed as a versioned path
like `/v2/messages`. Because REST endpoints are where many systems meet, they are a prime
target for [integration tests](/reference/integration-testing/) and
[end-to-end tests](/reference/end-to-end-testing/) that exercise real requests against the
running service.

## Sources

[^wiki]: [REST](https://en.wikipedia.org/wiki/REST) — Wikipedia, for the architectural style, its constraints, and Roy Fielding's role in defining it.
