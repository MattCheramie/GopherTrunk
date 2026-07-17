---
slug: managed-hosting
title: Managed hosting
entry_type: concept
category: hw-servers
description: Managed hosting is a service where the provider not only supplies the server but also handles its operating system, updates, security, and monitoring, so the customer manages only their application.
keywords: managed hosting, fully managed, server administration, patching, monitoring, unmanaged hosting, SLA
aka: [Fully managed hosting]
infobox:
  - { label: Type, value: Hosting model }
  - { label: Provider runs, value: OS, patching, security, monitoring }
  - { label: You run, value: Your application }
  - { label: Often includes, value: Response-time SLA }
  - { label: Contrast, value: Unmanaged / self-managed }
see_also: [web-hosting, dedicated-server, colocation, cloud-computing, software-as-a-service, server]
related_lessons:
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Internet_hosting_service
---

**Managed hosting** is a service in which the provider supplies the [server](/reference/server/) and also runs it for you — handling the operating system, updates, security, backups, and monitoring — so you manage only your own application.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 190" role="img" aria-label="Two responsibility stacks compared. Under unmanaged hosting the customer owns everything above the bare hardware: the operating system, patching, security, and the application. Under managed hosting the provider owns the hardware, operating system, patching, security, and monitoring, leaving the customer only the application on top." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <text x="115" y="18" text-anchor="middle" font-size="9" font-weight="600" stroke="none">UNMANAGED</text>
    <text x="345" y="18" text-anchor="middle" font-size="9" font-weight="600" stroke="none">MANAGED</text>
    <g stroke-width="1.2" font-size="8">
      <rect x="30" y="28" width="170" height="24" rx="2" fill-opacity="0.20"/><text x="115" y="44" text-anchor="middle" stroke="none">Application</text>
      <rect x="30" y="54" width="170" height="24" rx="2" fill-opacity="0.20"/><text x="115" y="70" text-anchor="middle" stroke="none">Security</text>
      <rect x="30" y="80" width="170" height="24" rx="2" fill-opacity="0.20"/><text x="115" y="96" text-anchor="middle" stroke="none">Patching &amp; updates</text>
      <rect x="30" y="106" width="170" height="24" rx="2" fill-opacity="0.20"/><text x="115" y="122" text-anchor="middle" stroke="none">Operating system</text>
      <rect x="30" y="132" width="170" height="24" rx="2" fill-opacity="0.06"/><text x="115" y="148" text-anchor="middle" stroke="none">Hardware</text>

      <rect x="260" y="28" width="170" height="24" rx="2" fill-opacity="0.20"/><text x="345" y="44" text-anchor="middle" stroke="none">Application</text>
      <rect x="260" y="54" width="170" height="24" rx="2" fill-opacity="0.06"/><text x="345" y="70" text-anchor="middle" stroke="none">Security</text>
      <rect x="260" y="80" width="170" height="24" rx="2" fill-opacity="0.06"/><text x="345" y="96" text-anchor="middle" stroke="none">Patching &amp; monitoring</text>
      <rect x="260" y="106" width="170" height="24" rx="2" fill-opacity="0.06"/><text x="345" y="122" text-anchor="middle" stroke="none">Operating system</text>
      <rect x="260" y="132" width="170" height="24" rx="2" fill-opacity="0.06"/><text x="345" y="148" text-anchor="middle" stroke="none">Hardware</text>
    </g>
    <text x="115" y="174" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">you manage all but the metal</text>
    <text x="345" y="170" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">shaded = provider runs it</text>
    <text x="345" y="181" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">you manage only the app</text>
  </g>
</svg>
<figcaption>The line between unmanaged and managed hosting is who keeps the machine healthy: unmanaged leaves everything above the hardware to you, while managed hands the operating system, patching, security, and monitoring to the provider so you tend only your application.</figcaption>
</figure>

## Overview

The line between *managed* and *unmanaged* hosting is who keeps the machine healthy. With unmanaged hosting (the typical [dedicated server](/reference/dedicated-server/) or plain [VPS](/reference/virtual-private-server/)), the provider hands you a server and you are responsible for patching the [operating system](/reference/operating-system/), configuring services, and responding to incidents.

With managed hosting, the provider does that operational work — often under a service-level agreement with guaranteed response times — and you pay a premium for it. The premium buys back time and expertise: no 3 a.m. patching, no scramble when a disk fails, and someone accountable for uptime. What you give up is some control, since you no longer touch the layers the provider now owns.

## Where it fits

Managed and unmanaged are two ends of one axis; where a model sits determines how much operations work lands on you:

| Model | Provider handles | You handle |
|-------|------------------|------------|
| [Colocation](/reference/colocation/) | Facility only | Hardware + all software |
| Unmanaged server | Hardware | OS, security, app |
| Managed hosting | Hardware, OS, security, monitoring | Application |
| [SaaS](/reference/software-as-a-service/) | Everything, incl. the app | Just using it |

Managed hosting suits teams without dedicated system administrators, or those who would rather spend effort on their product than on server upkeep. It is more hands-off than colocation but less abstract than [software as a service](/reference/software-as-a-service/), where even the application is run for you. For a hobbyist running GopherTrunk, managed hosting is rarely needed — the capture node is yours to tend — but it can simplify a back end that stores and serves decoded data.

## Sources

[^wiki]: [Internet hosting service](https://en.wikipedia.org/wiki/Internet_hosting_service) — Wikipedia, on managed versus unmanaged hosting models.
