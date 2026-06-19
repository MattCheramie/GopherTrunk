---
slug: api
title: API
entry_type: concept
category: principles-quality
description: An API (application programming interface) is a defined contract through which one piece of software exposes its capabilities to another, hiding implementation details behind a stable set of operations.
keywords: API, application programming interface, contract, interface, library API, web API, REST, endpoints, abstraction, backward compatibility
aka: [API "application programming interface"]
autolink: true
infobox:
  - { label: Category, value: "Software interface / contract" }
  - { label: Stands for, value: "Application Programming Interface" }
  - { label: Kinds, value: "Library, OS, web/HTTP APIs" }
  - { label: Web style, value: "REST, GraphQL, gRPC" }
  - { label: Key property, value: "Stable contract, hidden implementation" }
see_also: [rest, abstraction, error-handling, semantic-versioning, package-manager, coupling-and-cohesion, solid]
related_lessons:
  - { title: "Errors, edge cases & defensive programming", url: /learn/intro-software-dev/robustness-and-errors/ }
external:
  - { title: "API — Wikipedia", url: https://en.wikipedia.org/wiki/API }
---

**An API** (application programming interface) is a defined contract through which one
piece of software exposes its capabilities to another, hiding its implementation behind
a stable set of operations.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="A client calls an API, which is a contract in front of a provider's hidden implementation." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="45" width="80" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.3"/><text x="60" y="66">client</text>
    <line x1="100" y1="63" x2="180" y2="63" stroke="currentColor" stroke-width="1.1"/><text x="140" y="55" font-size="8">calls</text>
    <rect x="180" y="35" width="70" height="56" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.4"/><text x="215" y="60">API</text><text x="215" y="74" font-size="8">contract</text>
    <line x1="250" y1="63" x2="330" y2="63" stroke="currentColor" stroke-width="1.1"/>
    <rect x="330" y="40" width="110" height="46" rx="5" fill="none" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/><text x="385" y="60">implementation</text><text x="385" y="74" font-size="8">(hidden)</text>
  </g>
</svg>
<figcaption>An API is the published contract a client calls; the provider's implementation stays hidden behind it.</figcaption>
</figure>

## What an API is

An API is a form of [abstraction](/reference/abstraction/): it specifies *what* a
component does — the operations, their inputs and outputs, and the rules for using them
— without revealing *how* it does it. That separation lets the provider change the
implementation freely as long as the contract holds, and lets the consumer build against
a stable surface. APIs come in several flavors:

- **Library / package APIs** — the public functions, types, and methods a code library
  exposes to programs that import it.
- **Operating-system APIs** — the system calls a program uses to talk to the kernel.
- **Web APIs** — operations a service exposes over the network, commonly in the
  [REST](/reference/rest/) style over HTTP, or via GraphQL or gRPC.

A good API is small and role-focused (the [interface segregation](/reference/solid/)
idea), consistent, and clear about how it reports failures — its
[error-handling](/reference/error-handling/) contract is part of the interface.

## Why APIs matter

APIs are the seams that let large systems be built from independently developed parts.
Because callers depend only on the contract, an API reduces
[coupling](/reference/coupling-and-cohesion/) between components and teams. That stability
is also a promise: once others build on your API, changing it can break them, which is
why backward compatibility and [semantic versioning](/reference/semantic-versioning/)
matter so much — a major version bump signals a breaking change to the contract. The
same discipline governs the public surface of a reusable library distributed through a
[package manager](/reference/package-manager/).
