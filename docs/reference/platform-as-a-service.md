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

<figure class="figure" markdown="0">
<svg viewBox="0 0 464 268" role="img" aria-label="A grid of who manages each layer across four cloud tiers — on-premises, IaaS, PaaS, and SaaS — with the PaaS column outlined for emphasis. Rows are application, data, runtime, operating system, virtualization, and hardware. In the PaaS column only the application and data cells are filled as managed by you; the runtime and everything below are light, managed by the provider." xmlns="http://www.w3.org/2000/svg">
  <g fill="currentColor" font-size="9" text-anchor="middle">
    <text x="144" y="36" font-weight="600">On-prem</text>
    <text x="234" y="36" font-weight="600">IaaS</text>
    <text x="324" y="36" font-weight="600">PaaS</text>
    <text x="414" y="36" font-weight="600">SaaS</text>
  </g>
  <g fill="currentColor" font-size="8.5" text-anchor="end">
    <text x="94" y="63">Application</text>
    <text x="94" y="93">Data</text>
    <text x="94" y="123">Runtime</text>
    <text x="94" y="153">OS</text>
    <text x="94" y="183">Virtualization</text>
    <text x="94" y="213">Hardware</text>
  </g>
  <g stroke="currentColor">
    <rect x="103" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="103" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="193" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="193" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="283" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="283" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="283" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="46" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="76" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="106" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="136" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="166" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="373" y="196" width="82" height="26" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
  </g>
  <rect x="281" y="44" width="86" height="180" rx="3" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g font-size="8" fill="currentColor">
    <rect x="150" y="242" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="170" y="252" text-anchor="start">you manage</text>
    <rect x="252" y="242" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"/>
    <text x="272" y="252" text-anchor="start">provider manages</text>
  </g>
</svg>
<figcaption>PaaS draws the line higher than IaaS. In the outlined PaaS column only your application and data stay yours; the platform runs the runtime, operating system, virtualization, and hardware beneath. You push code and the provider operates everything under it.</figcaption>
</figure>

## Overview

PaaS is the middle tier of the cloud stack. You push an application — a web app, an API, a background worker — and the platform builds, runs, scales, and patches it for you, often using [containers](/reference/container/) behind the scenes. You do not log in to servers, install an [operating system](/reference/operating-system/), or configure load balancing; those are the platform's job. In exchange you accept its conventions about how apps are packaged and deployed.

## Where it fits

PaaS sits above [infrastructure as a service](/reference/infrastructure-as-a-service/) (raw VMs you must configure) and below [software as a service](/reference/software-as-a-service/) (finished applications you just use). It overlaps with [serverless computing](/reference/serverless-computing/), which pushes the abstraction further to per-request execution. For deploying a GopherTrunk web dashboard, PaaS removes server upkeep — but it cannot host the RF capture itself, which needs real radio hardware.

## Sources

[^wiki]: [Platform as a service](https://en.wikipedia.org/wiki/Platform_as_a_service) — Wikipedia, on the PaaS cloud model.
