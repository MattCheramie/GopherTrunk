---
slug: server
title: Server
entry_type: hardware
category: hw-servers
description: A server is a computer that provides services or data to other machines over a network, built for uptime and many simultaneous clients.
keywords: server, network server, web server, application server, uptime, clients, request response
aka: [Server]
infobox:
  - { label: Type, value: Networked computer }
  - { label: Role, value: Serves clients }
  - { label: Built for, value: Uptime & concurrency }
  - { label: Building blocks, value: CPU, RAM, storage, I/O }
  - { label: Obtained as, value: Web host, VPS, dedicated, cloud }
see_also: [web-hosting, virtual-private-server, dedicated-server, home-server, cloud-computing, network-attached-storage]
related_lessons:
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Server_(computing)
---

A **server** is a computer that provides services or data to other machines over a network — serving web pages, running an application, or storing files for clients to fetch.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 180" role="img" aria-label="Several client devices on the left send requests over a network to a single server on the right, which sends responses back. The server box is opened to show its four building blocks: CPU, RAM, storage, and input-output, and a note that it is built for uptime and many simultaneous clients." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" fill="currentColor" font-family="ui-sans-serif, sans-serif">
    <g stroke-width="1.1" fill-opacity="0.14">
      <rect x="26" y="40" width="40" height="26" rx="2"/>
      <rect x="26" y="78" width="40" height="26" rx="2"/>
      <rect x="26" y="116" width="40" height="26" rx="2"/>
    </g>
    <text x="46" y="156" text-anchor="middle" font-size="7.5" stroke="none" fill-opacity="0.85">clients</text>
    <g fill="none" stroke-width="1.3">
      <line x1="70" y1="53" x2="150" y2="80"/>
      <line x1="70" y1="91" x2="150" y2="91"/>
      <line x1="70" y1="129" x2="150" y2="102"/>
    </g>
    <path d="M150 80 l-8 0 3 -5 z" stroke-width="1"/>
    <text x="108" y="66" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.8">request</text>
    <text x="108" y="120" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.8">response</text>
    <rect x="152" y="40" width="150" height="104" rx="4" fill-opacity="0.06" stroke-width="1.5"/>
    <text x="227" y="34" text-anchor="middle" font-size="9" font-weight="600" stroke="none">Server</text>
    <g stroke-width="1.1" font-size="7.5">
      <rect x="162" y="50" width="64" height="26" rx="2" fill-opacity="0.16"/><text x="194" y="66" text-anchor="middle" stroke="none">CPU</text>
      <rect x="230" y="50" width="64" height="26" rx="2" fill-opacity="0.16"/><text x="262" y="66" text-anchor="middle" stroke="none">RAM</text>
      <rect x="162" y="80" width="64" height="26" rx="2" fill-opacity="0.16"/><text x="194" y="96" text-anchor="middle" stroke="none">storage</text>
      <rect x="230" y="80" width="64" height="26" rx="2" fill-opacity="0.16"/><text x="262" y="96" text-anchor="middle" stroke="none">I/O</text>
    </g>
    <text x="227" y="124" text-anchor="middle" font-size="7" stroke="none" fill-opacity="0.8">built for uptime &amp; concurrency</text>
    <text x="378" y="60" text-anchor="middle" font-size="8" stroke="none" font-weight="600">Same blocks,</text>
    <text x="378" y="72" text-anchor="middle" font-size="8" stroke="none" font-weight="600">different upkeep:</text>
    <g font-size="7.5" stroke="none" text-anchor="middle" fill-opacity="0.9">
      <text x="378" y="90">runs 24/7</text>
      <text x="378" y="104">many clients at once</text>
      <text x="378" y="118">redundant parts</text>
    </g>
  </g>
</svg>
<figcaption>A server answers requests from many clients at once over the network; inside it holds the same four building blocks as any computer — CPU, RAM, storage, and I/O — but is built and operated for long uptime and heavy concurrency.</figcaption>
</figure>

## Overview

Under the hood a server has the same four building blocks as any computer: a [CPU](/reference/central-processing-unit/), [RAM](/reference/random-access-memory/), [storage](/reference/data-storage/), and [I/O](/reference/input-output/). What sets it apart is how it is built and run: for long uptime and to handle many simultaneous clients without falling over, often with redundant power supplies, error-correcting memory, and hot-swappable disks.

The word describes a *role* as much as a machine. The same box can be a web server, an application server, a file server, or all three at once, depending on the software it runs. A "server" can be a rack machine in a data center, a virtual slice of one, or an old box in a closet — what makes it a server is that other machines depend on it.

## Where it fits

This is the umbrella entry for the category. You usually do not buy a bare server so much as choose *how* you get one:

| Way to get one | You manage | Best for |
|----------------|-----------|----------|
| [Web hosting](/reference/web-hosting/) | Just your site | Simple sites |
| [VPS](/reference/virtual-private-server/) | OS and up | Custom stacks, control |
| [Dedicated server](/reference/dedicated-server/) | The whole machine | Heavy, steady load |
| [Home server](/reference/home-server/) | Everything, incl. hardware | Self-hosting, learning |
| [Cloud computing](/reference/cloud-computing/) | Varies by service | On-demand, elastic |

In GopherTrunk terms a server can store and serve decoded data, run the web console, and archive recordings to [network-attached storage](/reference/network-attached-storage/), but it cannot capture RF — that still needs an antenna and a dongle on a machine with a radio front end.

## Sources

[^wiki]: [Server (computing)](https://en.wikipedia.org/wiki/Server_(computing)) — Wikipedia, on servers as machines that provide services to networked clients.
