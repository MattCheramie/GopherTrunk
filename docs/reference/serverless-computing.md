---
slug: serverless-computing
title: Serverless computing
entry_type: concept
category: hw-servers
description: Serverless computing is a cloud model where code runs in short-lived, provider-managed containers triggered by events, scaling automatically and billing only for actual execution, with no servers to provision.
keywords: serverless, functions as a service, FaaS, lambda, event-driven, autoscaling, pay per use
aka: [FaaS, Functions as a service]
infobox:
  - { label: Type, value: Cloud execution model }
  - { label: Unit, value: Function / event handler }
  - { label: Scaling, value: Automatic, to zero }
  - { label: Billing, value: Per execution }
see_also: [platform-as-a-service, cloud-computing, container, infrastructure-as-a-service, scalability, software-as-a-service]
cite_urls:
  - https://en.wikipedia.org/wiki/Serverless_computing
---

**Serverless computing** is a [cloud computing](/reference/cloud-computing/) model in which code runs in short-lived, provider-managed environments triggered by events; it scales automatically and bills only for actual execution, with no servers to provision.[^wiki]

## Overview

The name is a slight misnomer — servers still exist, but the developer never sees or manages them. You upload a function; the platform runs it on demand when an event fires (an HTTP request, a queue message, a timer), spins up [containers](/reference/container/) to handle load, and tears them down afterward. Because idle functions cost nothing, serverless can scale to zero between bursts. The trade-offs are cold-start latency, execution time limits, and statelessness — long-running or stateful work fits poorly.

## Where it fits

Serverless is the most abstract way to run your own code, sitting above [platform as a service](/reference/platform-as-a-service/) by removing even the notion of a running process you manage. Its automatic [scalability](/reference/scalability/) suits bursty, event-driven workloads. GopherTrunk's decode loop is continuous and tied to live RF, so it is a poor fit for serverless, but occasional tasks — sending an alert when a talkgroup appears — map naturally onto a function.

## Sources

[^wiki]: [Serverless computing](https://en.wikipedia.org/wiki/Serverless_computing) — Wikipedia, on the serverless and FaaS model.
