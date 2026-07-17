---
slug: proxies-and-load-balancers
title: Proxies, reverse proxies & load balancers
description: What proxies, reverse proxies, and load balancers actually do — forward proxies in front of clients, reverse proxies as a service's front door, TLS termination and routing, spreading traffic with load balancers and health checks, and how a CDN caches content close to users.
keywords: proxy, reverse proxy, forward proxy, load balancer, nginx, Caddy, Traefik, CDN, TLS termination, health checks, self hosting, front door
level: advanced
status: full
prereq:
  - http
faq:
  - q: "What is the difference between a forward proxy and a reverse proxy?"
    a: "A forward proxy sits in front of the clients and makes outbound requests on their behalf — filtering, caching, or hiding who is asking. A reverse proxy sits in front of the servers and takes incoming requests from the internet, passing them to the right backend. Same idea of a middleman relaying a request, opposite ends of the connection."
  - q: "What does a reverse proxy do?"
    a: "It is the front door for one or more servers. It receives requests from the internet, terminates TLS, routes each request to a backend by hostname or path, can cache responses, and hides the real servers behind it. nginx, Caddy, and Traefik are common reverse proxies for self-hosted services."
  - q: "What is TLS termination?"
    a: "TLS termination is when a reverse proxy holds the certificate, decrypts incoming HTTPS, and forwards plain HTTP to your app on a private network. It keeps certificate handling in one place instead of wiring TLS into every application."
  - q: "What is the difference between a load balancer and a reverse proxy?"
    a: "A reverse proxy is the front door that routes requests to backends; a load balancer specifically spreads traffic across several equivalent backends and skips unhealthy ones. In practice they are often the same box — a reverse proxy that also balances load."
---

# Proxies, reverse proxies & load balancers

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **forward proxy** sits in front of clients and makes their outbound requests for
them. A **reverse proxy** sits in front of servers and is the front door that greets
[HTTP](/learn/networking/http/) requests from the internet. A **load balancer** spreads
that traffic across several backends and skips the sick ones. A **CDN** caches content
close to users worldwide. All four are middlemen in the request path — they route,
protect, cache, and scale traffic without the client or the backend needing to know.
</div>

Once a service grows past a single machine, or simply needs to face the open internet,
raw client-to-server connections stop being enough. The fix is almost always a middleman
that sits in the path and does the tedious, security-sensitive work in one place. The
names differ, but the shape is the same.

## What a proxy is

A **proxy** is an intermediary. Instead of a client talking straight to a server, it
talks to the proxy; the proxy receives the request, **forwards it on**, waits for the
reply, and **relays the response** back. To each side it looks like an ordinary
conversation — the client thinks it's talking to the server, and the server sees the
proxy as the client.

That one indirection is surprisingly powerful. Because the proxy sits in the middle, it
can inspect, filter, cache, encrypt, redirect, or spread out traffic — all without
either end being rewritten. The differences between the terms below are mostly about
**which end** the proxy stands in front of.

## Forward proxy

A **forward proxy** sits in front of the **clients**. When a machine on the inside wants
to reach the outside world, it hands the request to the forward proxy, which makes the
outbound request **on the client's behalf** and passes the answer back.

Because every outbound request funnels through it, a forward proxy is a natural place to:

- **Filter** — block or allow which sites the clients may reach.
- **Cache** — keep a local copy of popular content so repeated fetches are fast.
- **Hide the client** — the destination server sees the proxy's address, not the
  individual machine behind it.

The classic example is a **corporate web proxy**: every employee's browser is pointed at
it, and it enforces policy and logging on the way out.

## Reverse proxy

A **reverse proxy** flips that around: it sits in front of the **servers**. It receives
requests coming *in* from the internet and passes each one to the correct backend, then
relays the backend's reply. To the outside world the reverse proxy *is* the service — the
real servers stay hidden behind it.

A reverse proxy is where a lot of production plumbing lives:

- **TLS termination** — it holds the certificate, decrypts incoming HTTPS, and forwards
  plain HTTP to the app on a private network, so certificate handling stays in one place
  (see [TLS & HTTPS](/learn/networking/tls-and-https/)).
- **Routing** — it sends each request to the right backend by **hostname or path**
  (`api.example.com` to one app, `example.com/blog` to another).
- **Caching** — it can store and serve common responses without bothering the backend.
- **Hiding the real servers** — clients only ever see the proxy's address.

**nginx**, **Caddy**, and **Traefik** are the reverse proxies you'll meet most often.
This is the standard **front door** for a self-hosted service, and it's the pattern you'll
use when [exposing a service safely](/learn/networking/exposing-a-service-safely/).

## Load balancer

A **load balancer** is a reverse proxy with a specific job: it **spreads incoming traffic
across several backend servers** instead of sending everything to one. When one backend is
busy, the next request goes to another, so a service can **scale** by simply adding more
machines behind the balancer.

It also runs **health checks** — periodic probes to see which backends are actually
answering. A server that fails its check is **skipped** until it recovers, so a single
crashed or overloaded backend doesn't take the whole service down. Traffic just flows
around it.

In small setups the load balancer is **often the same box** as the reverse proxy — the
front door both routes requests *and* balances them across a pool of equivalent backends.

## CDN

A **CDN (content delivery network)** is a global network of caching servers spread across
many locations. It keeps copies of your content **close to users**, so a visitor is served
from a nearby node instead of a round trip to your origin server — which is faster and
**absorbs traffic spikes** before they ever reach you. You point your domain at the CDN,
and it fetches from your origin only when its cache misses.

## Why you'll meet them

You don't need a data centre to run into these. Even a **single self-hosted app** usually
sits behind a **reverse proxy** — because that's the clean way to get **HTTPS** (TLS
termination) and tidy **routing** by name or path, instead of teaching every app to hold
its own certificate. It's the same story whether you're
[exposing a service safely](/learn/networking/exposing-a-service-safely/) or putting
[GopherTrunk on the network](/learn/networking/gophertrunk-on-the-network/): one small
proxy out front does the security-sensitive, fiddly work so your app doesn't have to.

<div class="knowledge-check" data-quiz data-correct-msg="Right — that's the reverse proxy's job as a service's front door." markdown="0">
  <p class="knowledge-check__q">Quick check: what sits in front of your servers to terminate TLS and route incoming requests to the right backend?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A forward proxy</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A reverse proxy</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A CDN edge node</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **proxy** is a middleman that receives a request, forwards it on, and relays the reply.
- A **forward proxy** sits in front of **clients** — filtering, caching, hiding who's asking.
- A **reverse proxy** sits in front of **servers** — TLS termination, routing, caching, and
  hiding the real backends. nginx, Caddy, Traefik.
- A **load balancer** spreads traffic across backends and skips unhealthy ones via **health
  checks**; often the same box as the reverse proxy.
- A **CDN** caches content close to users for speed and soaks up traffic spikes.
- Even one self-hosted app usually sits behind a reverse proxy for HTTPS and clean routing.

Next up: [network security basics](/learn/networking/network-security-basics/)
