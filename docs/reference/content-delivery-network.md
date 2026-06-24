---
slug: content-delivery-network
title: Content delivery network (CDN)
entry_type: concept
category: hw-servers
description: A content delivery network (CDN) is a geographically distributed group of servers that cache and serve web content from locations close to users, cutting latency and offloading origin servers.
keywords: CDN, content delivery network, edge cache, points of presence, latency, origin server, caching
aka: [CDN]
infobox:
  - { label: Type, value: Distributed caching network }
  - { label: Caches, value: Web content near users }
  - { label: Nodes, value: Edge / points of presence }
  - { label: Improves, value: Latency & origin load }
see_also: [edge-computing, reverse-proxy, load-balancer, web-hosting, scalability, data-center]
cite_urls:
  - https://en.wikipedia.org/wiki/Content_delivery_network
---

A **content delivery network** (**CDN**) is a geographically distributed group of servers that cache and serve web content from locations close to users, cutting latency and offloading the origin servers.[^wiki]

## Overview

A CDN places caching nodes — *points of presence* — in many [data centers](/reference/data-center/) around the world. When a user requests a file, they are routed to the nearest node, which serves a cached copy if it has one or fetches it from the origin and keeps it for the next request. Because most of the traffic (images, video, scripts, static pages) is served from the edge, the origin sees far less load and distant users get content over a short network hop instead of a transcontinental one. CDNs also commonly absorb traffic spikes and blunt denial-of-service attacks.

## Where it fits

A CDN is closely related to a [reverse proxy](/reference/reverse-proxy/) and a [load balancer](/reference/load-balancer/) — it is essentially a global, caching reverse proxy — and it is a form of [edge computing](/reference/edge-computing/) for content. It is a standard tool for [scalability](/reference/scalability/) in [web hosting](/reference/web-hosting/). A small GopherTrunk dashboard rarely needs one, but a public site distributing decoded archives could front them with a CDN.

## Sources

[^wiki]: [Content delivery network](https://en.wikipedia.org/wiki/Content_delivery_network) — Wikipedia, on CDN architecture and benefits.
