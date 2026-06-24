---
slug: web-hosting
title: Web & shared hosting
entry_type: concept
category: hw-servers
description: Web hosting is a service that runs your website on someone else's servers so it is reachable on the internet, with shared hosting as the cheapest tier.
keywords: web hosting, shared hosting, website hosting, cPanel, hosting provider
aka: [Web hosting, Shared hosting]
infobox:
  - { label: Type, value: Hosting service }
  - { label: Cheapest tier, value: Shared hosting }
  - { label: Control, value: Limited / managed }
  - { label: Runs, value: PHP, Python, static sites }
see_also: [server, virtual-private-server, dedicated-server, cloud-computing, php-language]
related_lessons:
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
  - { title: "Virtual private server (VPS)", url: /learn/intro-hardware/vps/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Web_hosting_service
---

**Web hosting** is a service that runs your website on someone else's [servers](/reference/server/) so it is reachable on the internet.[^wiki]

## Overview

Shared hosting is the cheapest tier: many customers' sites live on one server, sharing its CPU, memory, and disk. It is managed for you — you upload files and the provider keeps the machine running — but you get limited control in exchange. It comfortably runs the common cases: [PHP](/reference/php-language/) apps, [Python](/reference/python-language/) sites, and plain static pages.

## Where it stops being enough

Shared hosting hits a wall when you need custom software, background services, or root access to the operating system. At that point you step up to a [virtual private server](/reference/virtual-private-server/), which gives you your own slice with full control, or a [dedicated server](/reference/dedicated-server/) for the whole machine.

## Sources

[^wiki]: [Web hosting service](https://en.wikipedia.org/wiki/Web_hosting_service) — Wikipedia, on hosting tiers including shared hosting.
