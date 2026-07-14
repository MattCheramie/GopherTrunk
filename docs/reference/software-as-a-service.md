---
slug: software-as-a-service
title: Software as a service (SaaS)
entry_type: concept
category: hw-servers
description: Software as a service (SaaS) is a model where finished applications are hosted by a provider and accessed over the internet, usually by subscription, with no software to install or servers to run.
keywords: SaaS, software as a service, cloud application, subscription software, web app, hosted software
aka: [SaaS]
infobox:
  - { label: Type, value: Cloud service model }
  - { label: Provider runs, value: The whole application }
  - { label: You provide, value: Your data and use }
  - { label: Access, value: Browser or API }
see_also: [platform-as-a-service, infrastructure-as-a-service, cloud-computing, managed-hosting, web-hosting, serverless-computing]
related_lessons:
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Software_as_a_service
---

**Software as a service** (**SaaS**) is a model in which finished applications are hosted by a provider and accessed over the internet — usually by subscription — with no software to install and no servers for the user to run.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 464 268" role="img" aria-label="A grid of who manages each layer across four cloud tiers — on-premises, IaaS, PaaS, and SaaS — with the SaaS column outlined for emphasis. Rows are application, data, runtime, operating system, virtualization, and hardware. Every cell in the SaaS column is light, meaning the provider manages the entire stack from the application down to the hardware." xmlns="http://www.w3.org/2000/svg">
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
  <rect x="371" y="44" width="86" height="180" rx="3" fill="none" stroke="currentColor" stroke-width="1.8"/>
  <g font-size="8" fill="currentColor">
    <rect x="150" y="242" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="170" y="252" text-anchor="start">you manage</text>
    <rect x="252" y="242" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"/>
    <text x="272" y="252" text-anchor="start">provider manages</text>
  </g>
</svg>
<figcaption>SaaS is the far end of the stack: the outlined column has no cells left for you. The provider runs the application, its data plumbing, the runtime, the operating system, virtualization, and hardware — you bring only your account and your data and reach it through a browser.</figcaption>
</figure>

## Overview

SaaS is the top tier of the cloud stack: the provider runs the application, the runtime, the operating system, and the hardware, and you interact only through a browser or API. Email, document suites, chat tools, and CRM systems are common examples. Billing is typically per user or per month, and updates roll out centrally, so every customer runs the same maintained version.

## Where it fits

SaaS hides the most infrastructure of the three cloud tiers, above [platform as a service](/reference/platform-as-a-service/) (where you still supply code) and [infrastructure as a service](/reference/infrastructure-as-a-service/) (where you supply the OS too). It resembles [managed hosting](/reference/managed-hosting/) taken to completion — even the application is run for you. GopherTrunk is self-hosted software rather than SaaS, because it must run next to real radio hardware; only its decoded output could be pushed to a SaaS service.

## Sources

[^wiki]: [Software as a service](https://en.wikipedia.org/wiki/Software_as_a_service) — Wikipedia, on the SaaS cloud model.
