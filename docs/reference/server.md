---
slug: server
title: Server
entry_type: hardware
category: hw-servers
description: A server is a computer that provides services or data to other machines over a network, built for uptime and many simultaneous clients.
keywords: server, network server, web server, application server, uptime, clients
aka: [Server]
infobox:
  - { label: Type, value: Networked computer }
  - { label: Role, value: Serves clients }
  - { label: Built for, value: Uptime & concurrency }
  - { label: Building blocks, value: CPU, RAM, storage, I/O }
see_also: [web-hosting, virtual-private-server, dedicated-server, home-server, cloud-computing]
related_lessons:
  - { title: "Web & shared hosting", url: /learn/intro-hardware/web-hosting/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Server_(computing)
---

A **server** is a computer that provides services or data to other machines over a network — serving web pages, running an application, or storing files for clients to fetch.[^wiki]

## Overview

Under the hood a server has the same four building blocks as any computer: a [CPU](/reference/central-processing-unit/), [RAM](/reference/random-access-memory/), [storage](/reference/data-storage/), and [I/O](/reference/input-output/). What sets it apart is how it is built and run: for long uptime and to handle many simultaneous clients without falling over. A "server" can be a rack machine in a data center, a virtual slice of one, or an old box in a closet.

## Where it fits

This is the umbrella entry for the category. You usually do not buy a bare server so much as choose *how* you get one: [web hosting](/reference/web-hosting/) for the simplest sites, a [virtual private server](/reference/virtual-private-server/) for more control, a [dedicated server](/reference/dedicated-server/) for the whole machine, a [home server](/reference/home-server/) you run yourself, or [cloud computing](/reference/cloud-computing/) on demand. In GopherTrunk terms a server can store and serve decoded data, but it cannot capture RF — that still needs an antenna and a dongle on a machine with a radio front end.

## Sources

[^wiki]: [Server (computing)](https://en.wikipedia.org/wiki/Server_(computing)) — Wikipedia, on servers as machines that provide services to networked clients.
