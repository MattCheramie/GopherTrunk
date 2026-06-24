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

## Overview

SaaS is the top tier of the cloud stack: the provider runs the application, the runtime, the operating system, and the hardware, and you interact only through a browser or API. Email, document suites, chat tools, and CRM systems are common examples. Billing is typically per user or per month, and updates roll out centrally, so every customer runs the same maintained version.

## Where it fits

SaaS hides the most infrastructure of the three cloud tiers, above [platform as a service](/reference/platform-as-a-service/) (where you still supply code) and [infrastructure as a service](/reference/infrastructure-as-a-service/) (where you supply the OS too). It resembles [managed hosting](/reference/managed-hosting/) taken to completion — even the application is run for you. GopherTrunk is self-hosted software rather than SaaS, because it must run next to real radio hardware; only its decoded output could be pushed to a SaaS service.

## Sources

[^wiki]: [Software as a service](https://en.wikipedia.org/wiki/Software_as_a_service) — Wikipedia, on the SaaS cloud model.
