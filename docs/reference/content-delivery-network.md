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

<figure class="figure" markdown="0">
<svg viewBox="0 0 452 220" role="img" aria-label="A single origin server on the left feeds three geographically distributed edge caches in the middle, each in a different region. Each edge cache serves the nearby user on the right over a short hop, while the dashed links back to the origin are only used to fill a cache on a miss." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="88" width="78" height="44" rx="5" fill="currentColor" fill-opacity="0.18" stroke="currentColor" stroke-width="1.3"/>
  <text x="57" y="108" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">Origin</text>
  <text x="57" y="122" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">one source</text>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="188" y="24" width="98" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/><text x="237" y="42">Edge PoP</text><text x="237" y="55" font-size="7">region A</text>
    <rect x="188" y="90" width="98" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/><text x="237" y="108">Edge PoP</text><text x="237" y="121" font-size="7">region B</text>
    <rect x="188" y="156" width="98" height="40" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/><text x="237" y="174">Edge PoP</text><text x="237" y="187" font-size="7">region C</text>
  </g>
  <g fill="currentColor" font-size="7.5" text-anchor="middle">
    <circle cx="392" cy="44" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="392" y="47">user</text>
    <circle cx="392" cy="110" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="392" y="113">user</text>
    <circle cx="392" cy="176" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/><text x="392" y="179">user</text>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1" stroke-opacity="0.4" stroke-dasharray="4 3">
    <line x1="96" y1="106" x2="188" y2="44"/>
    <line x1="96" y1="110" x2="188" y2="110"/>
    <line x1="96" y1="114" x2="188" y2="176"/>
  </g>
  <g stroke="currentColor" fill="none" stroke-width="1.2" stroke-opacity="0.8">
    <line x1="286" y1="44" x2="378" y2="44" marker-end="url(#cdn_ar)"/>
    <line x1="286" y1="110" x2="378" y2="110" marker-end="url(#cdn_ar)"/>
    <line x1="286" y1="176" x2="378" y2="176" marker-end="url(#cdn_ar)"/>
  </g>
  <text x="140" y="140" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.6">fill on miss</text>
  <text x="226" y="212" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">each user is served from the nearest edge cache, not the distant origin</text>
  <defs><marker id="cdn_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The origin is copied out to caching nodes — points of presence — spread around the world. A user is routed to the closest one and served over a short hop; the origin is touched only on a cache miss (the dashed links). Most traffic never reaches the origin, so it stays fast and lightly loaded even far away.</figcaption>
</figure>

## Overview

A CDN places caching nodes — *points of presence* — in many [data centers](/reference/data-center/) around the world. When a user requests a file, they are routed to the nearest node, which serves a cached copy if it has one or fetches it from the origin and keeps it for the next request. Because most of the traffic (images, video, scripts, static pages) is served from the edge, the origin sees far less load and distant users get content over a short network hop instead of a transcontinental one. CDNs also commonly absorb traffic spikes and blunt denial-of-service attacks.

## Where it fits

A CDN is closely related to a [reverse proxy](/reference/reverse-proxy/) and a [load balancer](/reference/load-balancer/) — it is essentially a global, caching reverse proxy — and it is a form of [edge computing](/reference/edge-computing/) for content. It is a standard tool for [scalability](/reference/scalability/) in [web hosting](/reference/web-hosting/). A small GopherTrunk dashboard rarely needs one, but a public site distributing decoded archives could front them with a CDN.

## Sources

[^wiki]: [Content delivery network](https://en.wikipedia.org/wiki/Content_delivery_network) — Wikipedia, on CDN architecture and benefits.
