---
slug: deploying-a-web-app
title: Deploying a web app
description: Getting your web app off your laptop and onto the internet — building the front end, running the back end on a host, pointing a domain at it, and putting a reverse proxy with TLS in front, connecting to the deployment module's toolkit.
keywords: deployment, hosting, build, static hosting, reverse proxy, TLS, HTTPS, domain, DNS, environment variables, CI/CD, production
level: intermediate
status: full
prereq:
  - what-a-backend-does
faq:
  - q: "Do the front end and back end deploy the same way?"
    a: "Usually not. A **front end** (or a fully static site) is a bundle of files that can be dropped onto static hosting or a CDN — no server process runs your code. A **back end** is a program that must *run* somewhere: a server, container, or platform that keeps the process alive, restarts it if it crashes, and gives it a database and secrets. Many apps host each separately."
  - q: "How does my code know its production settings, like the database URL?"
    a: "Through **environment variables** and secrets injected by the host at runtime — never hard-coded in the source. The same code runs in development and production; only the configuration differs. This keeps credentials out of your repository and lets one build run in any environment."
  - q: "Why do I need a reverse proxy if my app already serves HTTP?"
    a: "A **reverse proxy** sits in front of your app and handles the cross-cutting concerns you don't want in your application code — terminating **TLS** (HTTPS), routing to the right service, load-balancing across instances, and serving static files. Your app can speak plain HTTP internally while the proxy presents a single secure front door to the world. See [reverse proxies & TLS](/learn/deployment/reverse-proxies-and-tls/)."
---

# Deploying a web app

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Deploying means moving your app from *works on my laptop* to *running for users on
the internet*. A **front end** is built into static files you host or push to a
[CDN](/learn/web-dev/caching-and-cdns/); a **back end** is a process that must *run*
on a host with a database, secrets, and something to restart it. Production settings
come from **environment variables**, never hard-coded. A **reverse proxy** presents
the public HTTPS front door, terminating **TLS** and routing to your services. This
is the on-ramp to the [Containers & Deployment](/learn/deployment/what-is-deployment/)
module, which owns the full toolkit.
</div>

You've built an app. Now it has to *run somewhere other than your machine* — reachable
at a domain, over HTTPS, staying up when you close your laptop. That step is
**deployment**, and it's a big enough subject to have its own module. This lesson is
the web developer's orientation: the shape of getting a web app live, and where the
[deployment path](/learn/deployment/what-is-deployment/) takes over with the details.

## What "deploying" actually means

Locally, your app runs because *you* start it and keep the terminal open. In
production none of that holds: no one is watching the terminal, traffic arrives at
all hours, and a crash must not mean permanent downtime. **Deploying** is the work of
making your app run reliably, unattended, on infrastructure the public can reach. The
[deployment module's opener](/learn/deployment/what-is-deployment/) frames the whole
discipline; here we map it onto the front-end/back-end split you already know.

## Front end vs. back end hosting

The two halves of a web app deploy differently because they *are* different:

- **The front end** — after a [build step](/learn/web-dev/build-tools-and-bundlers/),
  it's just static files: HTML, CSS, JavaScript, images. Static files don't run code,
  so they can be dropped onto **static hosting** or a **CDN** and served fast from the
  edge. A [static site](/learn/web-dev/static-vs-dynamic/) like this one deploys
  entirely this way.
- **The back end** — a running **process**. It needs a host that keeps it alive,
  restarts it on failure, connects it to a [database](/learn/web-dev/backend-and-database/),
  and supplies its secrets. That's a server, a container platform, or a managed
  runtime — the subject of most of the deployment module.

A typical modern app hosts the front end on a CDN and the back end as a service, with
the browser calling the back end's [API](/learn/web-dev/building-a-rest-api/). A
server-rendered app deploys as one running process instead.

## The build step

Deployment starts with a **build** — the repeatable step that turns your source into
exactly what runs in production. For a front end that's bundling and minifying (with
[content-hashed filenames](/learn/web-dev/caching-and-cdns/) for caching); for a back
end it might be compiling a binary or assembling a container image. The output is an
**artifact**: a fixed, versioned thing you deploy. The key discipline is that the same
artifact you tested is the one you ship — you don't rebuild differently for
production.

```bash
# Front end: source -> static bundle
npm run build          # emits dist/  (hashed, minified files to host)

# Back end (Go, as GopherTrunk): source -> single binary
go build -o gophertrunk ./cmd/gophertrunk
```

## Configuration & secrets

The same build has to run in staging and production without code changes, so
everything that *differs* between environments — the database URL, API keys, the port
— comes from **environment variables** injected by the host at runtime. As the
[first API call](/learn/building-ai/your-first-api-call/) lesson stressed for keys,
secrets never live in source or in git; the host supplies them:

```bash
DATABASE_URL=postgres://…   API_KEY=…   PORT=8080   ./gophertrunk
```

This is what lets one tested artifact run anywhere — only its configuration changes.

## The public front door: reverse proxy & TLS

Your back end can speak plain HTTP on some internal port, but the public internet
needs one secure, stable entry point. That's a **reverse proxy** — a server that sits
in front of your app and handles the cross-cutting concerns:

- **TLS termination** — it holds the certificate and serves **HTTPS**, so browsers get
  an encrypted, trusted connection (see [TLS & HTTPS](/learn/networking/tls-and-https/)).
- **Routing** — it maps incoming paths or hostnames to the right service.
- **Load balancing** — it spreads traffic across multiple instances of your app.
- **Serving static files** — it can hand back assets without troubling your app.

A **domain name** points at the proxy via DNS, the proxy terminates TLS and forwards
to your app, and the whole thing looks like one clean `https://` site.
[Reverse proxies & TLS](/learn/deployment/reverse-proxies-and-tls/) covers this
setup in full.

## Where the deployment module takes over

This lesson is deliberately the *shape* of deployment. The mechanics — containers,
CI/CD pipelines that build and ship on every commit, orchestration, rollbacks,
monitoring — are the whole
[Containers & Deployment](/learn/deployment/what-is-deployment/) module. The bridge to
remember: a web app is a **buildable artifact** plus **runtime configuration**,
running behind a **TLS-terminating proxy** on a host that keeps it alive. Everything
else is detail on top of that spine.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a built front end is static files you can host on a CDN; a back end is a process that must run on a host." markdown="0">
  <p class="knowledge-check__q">Quick check: why do the front end and back end of an app usually deploy differently?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The front end runs on the database and the back end runs in the browser</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A built front end is static files you can host; a back end is a process that must keep running</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The front end needs a reverse proxy but the back end never does</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Deploying** makes your app run reliably and unattended on infrastructure the
  public can reach — the step from *works on my laptop* to *running for users*.
- **Front end** and **back end** host differently: a built front end is **static
  files** for a CDN; a back end is a **running process** that needs a host, database,
  and secrets.
- Every deploy starts from a repeatable **build** that emits a versioned **artifact** —
  ship the same artifact you tested.
- Production settings and secrets come from **environment variables** injected at
  runtime, never hard-coded in source.
- A **reverse proxy** presents the public front door — terminating **TLS**/HTTPS,
  routing, and load-balancing — while your app speaks plain HTTP behind it.
- The full toolkit lives in the [Containers & Deployment](/learn/deployment/what-is-deployment/)
  module; a web app is an **artifact plus configuration** behind a proxy.

Next up: [performance & Core Web Vitals](/learn/web-dev/performance-and-web-vitals/).
