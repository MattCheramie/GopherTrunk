---
slug: platform-as-a-service
title: Platform as a service (PaaS)
entry_type: concept
category: hw-servers
description: Platform as a service (PaaS) is a cloud model where the provider runs the servers, operating system, and runtime, and the customer just deploys application code without managing the underlying infrastructure.
keywords: PaaS, platform as a service, cloud platform, application runtime, deploy code, managed runtime
aka: [PaaS]
infobox:
  - { label: Type, value: Cloud service model }
  - { label: Provider runs, value: Servers, OS, runtime }
  - { label: You provide, value: Application code }
  - { label: Sits between, value: IaaS and SaaS }
see_also: [infrastructure-as-a-service, software-as-a-service, cloud-computing, serverless-computing, container, web-hosting]
related_lessons:
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Platform_as_a_service
---

**Platform as a service** (**PaaS**) is a [cloud computing](/reference/cloud-computing/) model in which the provider runs the servers, operating system, and application runtime, and you simply deploy code without managing the infrastructure underneath.[^wiki]

## Overview

PaaS is the middle tier of the cloud stack. You push an application — a web app, an API, a background worker — and the platform builds, runs, scales, and patches it for you, often using [containers](/reference/container/) behind the scenes. You do not log in to servers, install an [operating system](/reference/operating-system/), or configure load balancing; those are the platform's job. In exchange you accept its conventions about how apps are packaged and deployed.

## Where it fits

PaaS sits above [infrastructure as a service](/reference/infrastructure-as-a-service/) (raw VMs you must configure) and below [software as a service](/reference/software-as-a-service/) (finished applications you just use). It overlaps with [serverless computing](/reference/serverless-computing/), which pushes the abstraction further to per-request execution. For deploying a GopherTrunk web dashboard, PaaS removes server upkeep — but it cannot host the RF capture itself, which needs real radio hardware.

## Sources

[^wiki]: [Platform as a service](https://en.wikipedia.org/wiki/Platform_as_a_service) — Wikipedia, on the PaaS cloud model.
