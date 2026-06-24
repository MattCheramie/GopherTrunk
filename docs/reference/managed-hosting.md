---
slug: managed-hosting
title: Managed hosting
entry_type: concept
category: hw-servers
description: Managed hosting is a service where the provider not only supplies the server but also handles its operating system, updates, security, and monitoring, so the customer manages only their application.
keywords: managed hosting, fully managed, server administration, patching, monitoring, unmanaged hosting
aka: [Fully managed hosting]
infobox:
  - { label: Type, value: Hosting model }
  - { label: Provider runs, value: OS, patching, security, monitoring }
  - { label: You run, value: Your application }
  - { label: Contrast, value: Unmanaged / self-managed }
see_also: [web-hosting, dedicated-server, colocation, cloud-computing, software-as-a-service, server]
related_lessons:
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Internet_hosting_service
---

**Managed hosting** is a service in which the provider supplies the [server](/reference/server/) and also runs it for you — handling the operating system, updates, security, backups, and monitoring — so you manage only your own application.[^wiki]

## Overview

The line between *managed* and *unmanaged* hosting is who keeps the machine healthy. With unmanaged hosting (the typical [dedicated server](/reference/dedicated-server/) or plain [VPS](/reference/virtual-private-server/)), the provider hands you a server and you are responsible for patching the [operating system](/reference/operating-system/), configuring services, and responding to incidents. With managed hosting, the provider does that operational work — often with guaranteed response times — and you pay a premium for it.

## Where it fits

Managed hosting suits teams without dedicated system administrators, or those who would rather spend effort on their product than on server upkeep. It is more hands-off than [colocation](/reference/colocation/) but less abstract than [software as a service](/reference/software-as-a-service/), where even the application is run for you. For a hobbyist running GopherTrunk, managed hosting is rarely needed — the capture node is yours to tend — but it can simplify a back end that stores and serves decoded data.

## Sources

[^wiki]: [Internet hosting service](https://en.wikipedia.org/wiki/Internet_hosting_service) — Wikipedia, on managed versus unmanaged hosting models.
