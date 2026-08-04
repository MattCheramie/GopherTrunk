---
slug: caching-and-cdns
title: Caching & CDNs
description: Making the web fast by not doing the same work twice — browser caches, HTTP cache headers like Cache-Control and ETag, and content delivery networks that serve your site from a location close to each user.
keywords: caching, CDN, content delivery network, Cache-Control, ETag, browser cache, cache invalidation, edge server, TTL, cache busting, latency
level: intermediate
status: full
prereq:
  - http-servers-and-routing
faq:
  - q: "What's the difference between a cache and a CDN?"
    a: "A **cache** is any store of previously computed or fetched results kept so you can reuse them instead of redoing the work — the browser has one, the server can have one. A **CDN (content delivery network)** is a specific, distributed cache: a network of servers spread around the world that hold copies of your files and serve each user from a nearby one, cutting the distance data must travel."
  - q: "Why do people say cache invalidation is hard?"
    a: "Because a cache trades freshness for speed: once a copy is stored, how do you know when it's stale? Serve it too long and users see old content; expire it too soon and you lose the benefit. The usual answer is **content-hashed filenames** — when a file changes its name changes, so the new URL misses the cache while unchanged files stay cached forever. That sidesteps the hard part instead of guessing expiry times."
  - q: "Does a CDN help a dynamic app, or only static sites?"
    a: "Both, in different ways. A **static site** can be served almost entirely from the CDN. A **dynamic app** still uses a CDN for its static assets — scripts, styles, images — and often caches some API responses at the edge too, while personalised or fast-changing data goes to the origin server. Even a fully dynamic app usually has plenty that a CDN can accelerate."
---

# Caching & CDNs

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The fastest work is the work you don't repeat. **Caching** stores a result so it can
be reused instead of recomputed or refetched — the browser caches assets it already
has, guided by HTTP headers like **`Cache-Control`** and **`ETag`**. A **CDN
(content delivery network)** is a worldwide distributed cache that serves each user
from a nearby **edge** server, shrinking the distance data travels. The perennial
catch is **cache invalidation** — knowing when a stored copy is stale — usually
solved with **content-hashed filenames**. Caching underlies real
[web performance](/learn/web-dev/performance-and-web-vitals/).
</div>

By this point your app works — it authenticates users, serves data, stays secure.
The next question is *speed*, and a huge fraction of web performance comes down to a
single idea: **don't do the same work twice.** This lesson covers caching, from the
browser's own cache to a global CDN, and the one genuinely tricky part — deciding
when a cached copy has gone stale.

## Why caching matters

Every asset your page needs — scripts, styles, images, fonts — must be fetched, and
each fetch costs a round trip across the network. On a repeat visit, or across your
site's many pages, most of those files haven't changed at all. **Caching** means
keeping a copy of a result so the next request can reuse it instead of fetching or
computing it again. Done well, the browser skips whole downloads and your servers do
far less work. Caching appears at every layer — the browser, a CDN, a
[reverse proxy](/learn/deployment/reverse-proxies-and-tls/), the server, even the
[database](/learn/web-dev/backend-and-database/) — but they share the same logic.

## The browser cache & HTTP headers

The browser keeps its own cache, and your server steers it with **HTTP response
headers**. The main one is **`Cache-Control`**:

```http
Cache-Control: public, max-age=31536000, immutable
```

That tells the browser it may keep the file for a year without asking again — ideal
for an asset whose name changes when its content does. For things that change, you'd
send a short lifetime, or `no-cache` to force a check each time.

When a cached copy *might* still be good, **validation** avoids re-downloading it. An
**`ETag`** is a fingerprint of the file the server sends the first time; on the next
request the browser asks *"still this version?"* with `If-None-Match`, and the server
replies `304 Not Modified` — no body — if nothing changed:

```http
ETag: "a1b2c3"                 ← server sends the fingerprint
If-None-Match: "a1b2c3"        ← browser asks: still current?
304 Not Modified               ← server: yes, reuse your copy
```

Between `max-age` (don't even ask for a while) and `ETag` validation (ask cheaply),
you control the freshness-versus-speed tradeoff per resource.

## CDNs: caching close to the user

Even a cached-perfectly site has a physics problem on the *first* fetch: data travels
at a finite speed, and a user far from your server waits longer. A **CDN** attacks
that by putting copies of your files on **edge servers** spread across the globe.
When a user requests an asset, the CDN serves it from the nearest edge — often a few
milliseconds away instead of across an ocean.

A CDN does three things at once:

- **Cuts latency** by serving from a location physically near the user.
- **Offloads your origin** — most requests never reach your own server, so it handles
  far more traffic.
- **Absorbs spikes** — a burst of visitors hits the distributed edge, not one box.

Because [this very site](/learn/web-dev/templating-and-static-sites/) is static, a
CDN can serve essentially all of it. A dynamic app puts its **static assets** on the
CDN and sends personalised or fast-changing requests to the origin — often caching
some API responses at the edge too.

## The hard part: cache invalidation

There's an old joke that the two hardest problems in computing are naming things,
cache invalidation, and off-by-one errors. The middle one is real: once you've stored
a copy, **how do you know when it's stale?** Set a long lifetime and users get speed
but risk seeing outdated files after a deploy; set a short one and you lose most of
the benefit.

The standard trick sidesteps the guessing entirely — **content-hashed filenames**
(cache busting). The [build step](/learn/web-dev/build-tools-and-bundlers/) names
each file after a hash of its contents:

```
app.a1b2c3.js      ← change one byte and the hash — and the filename — changes
styles.9f8e7d.css
```

Now you can cache these *forever* (`max-age` of a year, `immutable`), because a
changed file gets a **new URL** the browser has never seen, so it fetches the new one
while every unchanged file stays cached. You never invalidate anything — you rename
it. The HTML that references these files gets a short cache lifetime so new filenames
are picked up promptly.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a CDN serves copies from an edge server near each user, cutting the distance and time data travels." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the main way a CDN speeds up your site?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It compresses your database so queries run faster</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It serves cached copies from an edge server physically near each user</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It rewrites your JavaScript to use fewer functions</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The core idea is **don't repeat work**: **caching** stores a result so it can be
  reused instead of recomputed or refetched.
- The **browser cache** is steered by HTTP headers — **`Cache-Control`** sets how
  long a copy is good, and an **`ETag`** lets the browser cheaply revalidate for a
  `304 Not Modified`.
- A **CDN** is a distributed cache of **edge servers** that serves each user from a
  nearby location, cutting latency, offloading your origin, and absorbing traffic
  spikes.
- **Cache invalidation** — knowing when a copy is stale — is the hard part, usually
  solved with **content-hashed filenames** so a changed file gets a brand-new URL.
- Caching applies at every layer and is a foundation of the performance work in the
  next unit.

Next up: [deploying a web app](/learn/web-dev/deploying-a-web-app/).
