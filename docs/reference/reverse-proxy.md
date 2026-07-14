---
slug: reverse-proxy
title: Reverse proxy
entry_type: concept
category: hw-servers
description: A reverse proxy is a server that sits in front of backend servers and forwards client requests to them, presenting a single front door while handling routing, TLS, caching, and security.
keywords: reverse proxy, proxy server, TLS termination, request routing, nginx, gateway, caching
infobox:
  - { label: Type, value: Intermediary server }
  - { label: Faces, value: Clients on behalf of backends }
  - { label: Common jobs, value: TLS, routing, caching }
  - { label: Contrast, value: Forward proxy }
see_also: [load-balancer, content-delivery-network, web-hosting, server, high-availability, kubernetes]
cite_urls:
  - https://en.wikipedia.org/wiki/Reverse_proxy
---

A **reverse proxy** is a [server](/reference/server/) that sits in front of one or more backend servers and forwards client requests to them, presenting a single front door while handling routing, encryption, caching, and security on their behalf.[^wiki]

<figure class="figure" markdown="0">
<svg viewBox="0 0 448 210" role="img" aria-label="Clients on the left reach a single reverse proxy in the middle over HTTPS. The proxy terminates TLS and routes each request by its URL path over plain HTTP to one of three hidden private backends, labelled slash-app, slash-api, and slash-static." xmlns="http://www.w3.org/2000/svg">
  <text x="52" y="24" text-anchor="middle" font-size="9" fill="currentColor" font-weight="600">clients</text>
  <g fill="currentColor" font-size="8" text-anchor="middle">
    <circle cx="52" cy="60" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="52" cy="105" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
    <circle cx="52" cy="150" r="13" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.1"/>
  </g>
  <rect x="150" y="78" width="104" height="54" rx="6" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/>
  <text x="202" y="100" text-anchor="middle" font-size="9.5" fill="currentColor" font-weight="600">Reverse proxy</text>
  <text x="202" y="114" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">TLS · route by path/host</text>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.75" fill="none">
    <line x1="65" y1="62" x2="150" y2="98" marker-end="url(#rp_ar)"/>
    <line x1="65" y1="105" x2="150" y2="105" marker-end="url(#rp_ar)"/>
    <line x1="65" y1="148" x2="150" y2="112" marker-end="url(#rp_ar)"/>
  </g>
  <text x="105" y="46" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.7">https</text>
  <g fill="currentColor" font-size="8.5" text-anchor="middle">
    <rect x="346" y="20" width="92" height="32" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="392" y="40">/app</text>
    <rect x="346" y="62" width="92" height="32" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="392" y="82">/api</text>
    <rect x="346" y="104" width="92" height="32" rx="4" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="392" y="124">/static</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" stroke-opacity="0.75" fill="none">
    <line x1="254" y1="100" x2="346" y2="36" marker-end="url(#rp_ar)"/>
    <line x1="254" y1="105" x2="346" y2="78" marker-end="url(#rp_ar)"/>
    <line x1="254" y1="112" x2="346" y2="120" marker-end="url(#rp_ar)"/>
  </g>
  <text x="300" y="60" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.7">http</text>
  <text x="392" y="150" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.7">private backends</text>
  <text x="224" y="198" text-anchor="middle" font-size="8.5" fill="currentColor" fill-opacity="0.9">one public front door; the backends stay hidden and speak plain HTTP</text>
  <defs><marker id="rp_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>To the client the reverse proxy is the whole site. It terminates HTTPS at the edge, then routes each request by path or hostname to the right backend over plain internal HTTP — so several private services hide behind one public address, and TLS, caching, and security live in one place.</figcaption>
</figure>

## Overview

To the client, the reverse proxy *is* the website; it then decides which backend should handle each request. Along the way it commonly terminates TLS (so backends speak plain HTTP internally), routes by hostname or URL path, caches responses, compresses output, and shields backends from direct exposure. This contrasts with a *forward* proxy, which sits in front of clients to reach the wider internet. Popular reverse proxies include nginx, HAProxy, and Caddy.

## Where it fits

A reverse proxy overlaps heavily with a [load balancer](/reference/load-balancer/) — both front a pool of backends — and the same software often does both. It is a building block for [web hosting](/reference/web-hosting/), [high availability](/reference/high-availability/), and the edge of a [content delivery network](/reference/content-delivery-network/). For a GopherTrunk web dashboard, a reverse proxy is a tidy way to add HTTPS and a clean public address in front of the decoder's local web port.

## Sources

[^wiki]: [Reverse proxy](https://en.wikipedia.org/wiki/Reverse_proxy) — Wikipedia, on reverse proxies and their roles.
