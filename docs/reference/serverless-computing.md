---
slug: serverless-computing
title: Serverless computing
entry_type: concept
category: hw-servers
description: Serverless computing is a cloud model where code runs in short-lived, provider-managed containers triggered by events, scaling automatically and billing only for actual execution, with no servers to provision.
keywords: serverless, functions as a service, FaaS, lambda, event-driven, autoscaling, pay per use, cold start
aka: [FaaS, Functions as a service]
infobox:
  - { label: Type, value: Cloud execution model }
  - { label: Unit, value: Function / event handler }
  - { label: Scaling, value: Automatic, to zero }
  - { label: Billing, value: Per execution }
  - { label: Weakness, value: Cold starts, time limits }
see_also: [platform-as-a-service, cloud-computing, container, infrastructure-as-a-service, scalability, software-as-a-service]
cite_urls:
  - https://en.wikipedia.org/wiki/Serverless_computing
---

**Serverless computing** is a [cloud computing](/reference/cloud-computing/) model in which code runs in short-lived, provider-managed environments triggered by events; it scales automatically and bills only for actual execution, with no servers to provision.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="An event-driven flow. Events such as an HTTP request, a queue message, and a timer arrive on the left. Each fires a function that the platform runs in a freshly spun-up container. Under a burst of events several containers run in parallel, and when events stop the platform scales back to zero so nothing is billed while idle." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <g font-size="7.5" stroke="none" text-anchor="middle">
      <rect x="20" y="34" width="72" height="20" rx="3" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="56" y="48">HTTP request</text>
      <rect x="20" y="72" width="72" height="20" rx="3" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="56" y="86">queue msg</text>
      <rect x="20" y="110" width="72" height="20" rx="3" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="56" y="124">timer</text>
    </g>
    <text x="56" y="24" text-anchor="middle" font-size="8" stroke="none" font-weight="600">events</text>
    <g stroke-width="1.3" fill="none">
      <line x1="92" y1="44" x2="180" y2="60"/>
      <line x1="92" y1="82" x2="180" y2="82"/>
      <line x1="92" y1="120" x2="180" y2="104"/>
    </g>
    <text x="255" y="24" text-anchor="middle" font-size="8" stroke="none" font-weight="600">functions spun up on demand</text>
    <g stroke-width="1.2" fill-opacity="0.16">
      <rect x="182" y="46" width="46" height="30" rx="3"/>
      <rect x="234" y="46" width="46" height="30" rx="3"/>
      <rect x="182" y="84" width="46" height="30" rx="3"/>
      <rect x="234" y="84" width="46" height="30" rx="3"/>
    </g>
    <g font-size="7" stroke="none" text-anchor="middle" fill-opacity="0.9">
      <text x="205" y="65">fn</text><text x="257" y="65">fn</text><text x="205" y="103">fn</text><text x="257" y="103">fn</text>
    </g>
    <line x1="288" y1="80" x2="344" y2="80" stroke-width="1.3" fill="none"/>
    <path d="M344 80 l-8 -3 v6 z" stroke-width="1"/>
    <text x="316" y="72" text-anchor="middle" font-size="6.5" stroke="none" fill-opacity="0.75">idle</text>
    <rect x="346" y="66" width="96" height="28" rx="3" fill-opacity="0.04" stroke-width="1.3" stroke-dasharray="4 3"/>
    <text x="394" y="84" text-anchor="middle" font-size="8" stroke="none">scale to zero</text>
    <text x="255" y="150" text-anchor="middle" font-size="8" stroke="none" fill-opacity="0.85">you pay only while a function is actually running</text>
  </g>
</svg>
<figcaption>Each event fires a function that the platform runs in a freshly created container; a burst spins up many in parallel, and when events stop the platform scales to zero — so idle time costs nothing.</figcaption>
</figure>

## Overview

The name is a slight misnomer — servers still exist, but the developer never sees or manages them. You upload a function; the platform runs it on demand when an event fires (an HTTP request, a queue message, a timer), spins up [containers](/reference/container/) to handle load, and tears them down afterward. Because idle functions cost nothing, serverless can scale to zero between bursts.

The trade-offs are real: a *cold start* delay when a new container must be created for the first request, hard limits on how long a single invocation may run, and statelessness — each call starts fresh, so anything that must persist lives in an external store. Long-running or stateful work fits poorly, but bursty, event-driven tasks fit beautifully.

## Trade-offs

Serverless sits at the top of the cloud abstraction ladder, giving up control for convenience:

| Property | Serverless | Traditional server |
|----------|-----------|--------------------|
| Provisioning | None | You size the machine |
| Scaling | Automatic, to zero | Manual or scripted |
| Billing | Per execution | Per hour, even idle |
| Startup latency | Cold-start delay | Always warm |
| Long / stateful jobs | Poor fit | Fine |

Where the workload is spiky and stateless, paying only for execution is a big win; where it is continuous, a server that stays warm is both simpler and cheaper.

## Where it fits

Serverless is the most abstract way to run your own code, sitting above [platform as a service](/reference/platform-as-a-service/) by removing even the notion of a running process you manage. Its automatic [scalability](/reference/scalability/) suits bursty, event-driven workloads. GopherTrunk's decode loop is continuous and tied to live RF, so it is a poor fit for serverless, but occasional tasks — sending an alert when a talkgroup appears — map naturally onto a function.

## Sources

[^wiki]: [Serverless computing](https://en.wikipedia.org/wiki/Serverless_computing) — Wikipedia, on the serverless and FaaS model.
