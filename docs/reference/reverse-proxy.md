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

## Overview

To the client, the reverse proxy *is* the website; it then decides which backend should handle each request. Along the way it commonly terminates TLS (so backends speak plain HTTP internally), routes by hostname or URL path, caches responses, compresses output, and shields backends from direct exposure. This contrasts with a *forward* proxy, which sits in front of clients to reach the wider internet. Popular reverse proxies include nginx, HAProxy, and Caddy.

## Where it fits

A reverse proxy overlaps heavily with a [load balancer](/reference/load-balancer/) — both front a pool of backends — and the same software often does both. It is a building block for [web hosting](/reference/web-hosting/), [high availability](/reference/high-availability/), and the edge of a [content delivery network](/reference/content-delivery-network/). For a GopherTrunk web dashboard, a reverse proxy is a tidy way to add HTTPS and a clean public address in front of the decoder's local web port.

## Sources

[^wiki]: [Reverse proxy](https://en.wikipedia.org/wiki/Reverse_proxy) — Wikipedia, on reverse proxies and their roles.
