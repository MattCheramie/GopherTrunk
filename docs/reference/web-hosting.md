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

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 214" role="img" aria-label="Four hosting tiers side by side — shared, VPS, dedicated, and cloud — each drawn as a bar split into a provider-managed portion on top and a you-managed portion below. Moving from shared to dedicated the you-managed portion grows, and an arrow marks that control and responsibility increase across those three. Cloud is drawn with a dashed border as a separate elastic option." xmlns="http://www.w3.org/2000/svg">
  <text x="167" y="13" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">more control &amp; responsibility &#8594;</text>
  <line x1="63" y1="22" x2="271" y2="22" stroke="currentColor" stroke-width="1.2" stroke-opacity="0.7" marker-end="url(#wh_ar)"/>
  <g fill="currentColor" font-size="9" text-anchor="middle" font-weight="600">
    <text x="63" y="36">Shared</text>
    <text x="167" y="36">VPS</text>
    <text x="271" y="36">Dedicated</text>
    <text x="375" y="36">Cloud</text>
  </g>
  <g stroke="currentColor">
    <rect x="18" y="42" width="90" height="106" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="18" y="148" width="90" height="22" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="122" y="42" width="90" height="68" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="122" y="110" width="90" height="60" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="226" y="42" width="90" height="28" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="226" y="70" width="90" height="100" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="330" y="42" width="90" height="73" rx="2" fill="currentColor" fill-opacity="0.04" stroke-width="0.8" stroke-opacity="0.5"/>
    <rect x="330" y="115" width="90" height="55" rx="2" fill="currentColor" fill-opacity="0.22" stroke-width="1"/>
    <rect x="330" y="42" width="90" height="128" rx="2" fill="none" stroke-width="1.2" stroke-dasharray="5 4"/>
  </g>
  <g fill="currentColor" font-size="7.5" text-anchor="middle">
    <text x="63" y="98" fill-opacity="0.75">provider</text>
    <text x="63" y="162">you</text>
    <text x="167" y="79" fill-opacity="0.75">provider</text>
    <text x="167" y="143">you</text>
    <text x="271" y="59" fill-opacity="0.75">provider</text>
    <text x="271" y="123">you</text>
    <text x="375" y="82" fill-opacity="0.75">provider</text>
    <text x="375" y="145">you</text>
  </g>
  <text x="375" y="184" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.7">elastic · scales on demand</text>
  <g font-size="8" fill="currentColor">
    <rect x="40" y="192" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1"/>
    <text x="60" y="202" text-anchor="start">you manage</text>
    <rect x="150" y="192" width="14" height="12" rx="2" fill="currentColor" fill-opacity="0.04" stroke="currentColor" stroke-width="0.8" stroke-opacity="0.5"/>
    <text x="170" y="202" text-anchor="start">provider manages</text>
  </g>
  <defs><marker id="wh_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Hosting tiers trade convenience for control. Shared hosting is almost entirely managed for you but gives little control; a VPS hands you your own slice, and a dedicated server the whole machine — each step down puts more of the stack in your hands. Cloud sits apart, billed on demand and scaling elastically.</figcaption>
</figure>

## Overview

Shared hosting is the cheapest tier: many customers' sites live on one server, sharing its CPU, memory, and disk. It is managed for you — you upload files and the provider keeps the machine running — but you get limited control in exchange. It comfortably runs the common cases: [PHP](/reference/php-language/) apps, [Python](/reference/python-language/) sites, and plain static pages.

## Where it stops being enough

Shared hosting hits a wall when you need custom software, background services, or root access to the operating system. At that point you step up to a [virtual private server](/reference/virtual-private-server/), which gives you your own slice with full control, or a [dedicated server](/reference/dedicated-server/) for the whole machine.

## Sources

[^wiki]: [Web hosting service](https://en.wikipedia.org/wiki/Web_hosting_service) — Wikipedia, on hosting tiers including shared hosting.
