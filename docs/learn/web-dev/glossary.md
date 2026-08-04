---
slug: glossary
title: Glossary of web-development terms
description: Plain-language definitions of the terms used to build a modern web app — front end, back end, HTML, CSS, the DOM, framework, component, state, REST API, SSR, SPA, session, cookie, JWT, WebSocket, XSS, CSRF, CORS, CDN, Core Web Vitals, and more — each cross-linked to the lesson that explains it.
keywords: web development glossary, front end, back end, HTML, CSS, DOM, framework, REST API, SSR, SPA, session, cookie, JWT, WebSocket, XSS, CSRF, CORS, CDN, Core Web Vitals, caching
level: beginner
status: full
lesson_standalone: true
---

# Glossary of web-development terms

Every term used across the [Web Development](/learn/web-dev/) path, defined in plain
language and linked to the lesson where it's explained in full. Skim it as a
refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a word. Terms are
grouped by theme, roughly in the order the module introduces them.

## The web &amp; the browser

**Front end** — Everything that runs in the user's browser — the HTML, CSS, and
JavaScript that make up what they see and interact with. See
[What web development is](/learn/web-dev/what-is-web-development/)

**Back end** — The server-side half of a web app — code, data, and logic that run on
a server the user never sees directly. See
[What a back end does](/learn/web-dev/what-a-backend-does/)

**Full stack** — Working across both front end and back end, and the layers between.
See [What web development is](/learn/web-dev/what-is-web-development/)

**Browser** — The program that fetches, parses, and renders web pages, turning HTML,
CSS, and JavaScript into the page you see and interact with. See
[How the browser works](/learn/web-dev/how-the-browser-works/)

**Rendering** — The browser's process of turning parsed HTML and CSS into the actual
pixels on screen. See [How the browser works](/learn/web-dev/how-the-browser-works/)

**Client** — The party that makes a request — usually the browser — in the
client-server model. See [The client-server web](/learn/web-dev/client-server-web/)

**Server** — The party that listens for requests and returns responses. See
[The client-server web](/learn/web-dev/client-server-web/)

**Request / response** — The two halves of every web interaction: the client sends a
**request**, the server returns a **response**. See
[The client-server web](/learn/web-dev/client-server-web/)

**HTTP** — The protocol browsers and servers use to exchange requests and responses on
the web. See [The client-server web](/learn/web-dev/client-server-web/)

**Stateless** — HTTP keeps no memory between requests; each one arrives independent of
the ones before it, which is why apps need sessions. See
[Authentication & sessions](/learn/web-dev/authentication-and-sessions/)

**Static site** — A site whose pages are pre-built files served as-is, with no code
run per request. See [Static vs. dynamic sites](/learn/web-dev/static-vs-dynamic/)

**Dynamic site** — A site whose pages are built on the fly for each request, often
from a database. See [Static vs. dynamic sites](/learn/web-dev/static-vs-dynamic/)

## The front end

**HTML** — The markup language that gives a page its structure and meaning through
elements and tags. See [HTML — structure & semantics](/learn/web-dev/html-structure/)

**Element / tag** — The building blocks of HTML — a tag like `<p>` marks a piece of
content as an element with a specific meaning. See
[HTML — structure & semantics](/learn/web-dev/html-structure/)

**Semantic markup** — Using the HTML element that matches the *meaning* of the content
(a heading, a nav, a button), which aids accessibility and search. See
[HTML — structure & semantics](/learn/web-dev/html-structure/)

**CSS** — The language that styles a page — colours, spacing, and layout — separately
from its HTML structure. See [CSS — styling & layout](/learn/web-dev/css-and-layout/)

**Selector** — The part of a CSS rule that says *which* elements the styles apply to.
See [CSS — styling & layout](/learn/web-dev/css-and-layout/)

**Box model** — The rule that every element is a box of content, padding, border, and
margin, which governs sizing and spacing. See
[CSS — styling & layout](/learn/web-dev/css-and-layout/)

**Flexbox / grid** — The two modern CSS layout systems for arranging elements in one
dimension (flexbox) or two (grid). See
[CSS — styling & layout](/learn/web-dev/css-and-layout/)

**JavaScript** — The programming language of the browser, which makes pages
interactive by responding to events and changing the page. See
[JavaScript in the browser](/learn/web-dev/javascript-in-the-browser/)

**Event** — A signal that something happened (a click, a keypress, a load) that
JavaScript can run code in response to. See
[JavaScript in the browser](/learn/web-dev/javascript-in-the-browser/)

**DOM (Document Object Model)** — The live tree the browser builds from your HTML;
JavaScript reads and changes it to update the page without a reload. See
[The DOM](/learn/web-dev/the-dom/)

**Form** — The HTML mechanism for collecting user input and submitting it to a server.
See [Forms & user input](/learn/web-dev/forms-and-user-input/)

**Validation** — Checking that user input is well-formed and acceptable — on the
client for feedback, and always on the server for safety. See
[Forms & user input](/learn/web-dev/forms-and-user-input/)

**Accessibility (a11y)** — Building so everyone can use your site, including people
using keyboards, screen readers, or low vision — largely achieved with semantic HTML.
See [Accessibility basics](/learn/web-dev/accessibility-basics/)

**ARIA** — Extra attributes that describe custom UI to assistive technology when plain
semantic HTML isn't enough. See [Accessibility basics](/learn/web-dev/accessibility-basics/)

## Frameworks &amp; the front end at scale

**Framework** — A library like React that keeps a complex UI in sync with data, so you
describe *what* the UI should look like rather than hand-writing DOM updates. See
[Front-end frameworks](/learn/web-dev/frontend-frameworks/)

**Component** — A reusable, self-contained piece of UI — its markup, styling, and
behaviour together — the building block of modern front ends. See
[Components & state](/learn/web-dev/components-and-state/)

**State** — The data a component holds that drives what it shows; when state changes,
the UI updates to match. See [Components & state](/learn/web-dev/components-and-state/)

**Fetch** — The browser API for making HTTP requests from JavaScript to load data
without reloading the page. See [Fetching data from the front end](/learn/web-dev/fetching-data/)

**JSON** — A lightweight text format for structured data, the common language between a
front end and an API. See [Fetching data from the front end](/learn/web-dev/fetching-data/)

**Async** — Doing work (like a network request) without freezing the page, handling the
result when it arrives. See [Fetching data from the front end](/learn/web-dev/fetching-data/)

**Bundler** — A build tool that combines and optimises your source files into what the
browser downloads. See [Build tools & bundlers](/learn/web-dev/build-tools-and-bundlers/)

**Transpiling** — Converting modern or non-browser code into a form browsers
understand, as part of the build step. See
[Build tools & bundlers](/learn/web-dev/build-tools-and-bundlers/)

**Responsive design** — Building one site that works on any screen size using fluid
layouts and media queries, usually mobile-first. See
[Responsive design](/learn/web-dev/responsive-design/)

**Media query** — A CSS rule that applies styles only under certain conditions, such as
a minimum screen width. See [Responsive design](/learn/web-dev/responsive-design/)

## Back end &amp; APIs

**Business logic** — The app-specific rules and work that must run on the server, out
of the user's reach — the reason a back end exists. See
[What a back end does](/learn/web-dev/what-a-backend-does/)

**Routing** — Matching an incoming request's URL and method to the code (a handler)
that should answer it. See [HTTP servers & routing](/learn/web-dev/http-servers-and-routing/)

**Handler** — The function on the server that runs for a matched route and produces the
response. See [HTTP servers & routing](/learn/web-dev/http-servers-and-routing/)

**HTTP method (verb)** — The action word on a request — GET, POST, PUT, DELETE — that
says what to do with a resource. See
[HTTP servers & routing](/learn/web-dev/http-servers-and-routing/)

**REST API** — A common style of web API built around resources, HTTP verbs, status
codes, and JSON — the contract a front end consumes. See
[Building a REST API](/learn/web-dev/building-a-rest-api/)

**Resource** — A thing your API exposes (a user, a call, an order), addressed by a URL.
See [Building a REST API](/learn/web-dev/building-a-rest-api/)

**Status code** — The number in an HTTP response saying how it went — 200 OK, 404 Not
Found, 500 error, and so on. See [Building a REST API](/learn/web-dev/building-a-rest-api/)

**Endpoint** — A specific URL (plus method) your API answers, mapping to one
operation. See [Building a REST API](/learn/web-dev/building-a-rest-api/)

**Database** — The system a back end reads and writes to persist data between requests.
See [The back end & its database](/learn/web-dev/backend-and-database/)

**Query** — A request to the database for data, or to change it. See
[The back end & its database](/learn/web-dev/backend-and-database/)

**Parameterised query** — A query where user data is passed as separate parameters, not
concatenated into the query string — the core defence against SQL injection. See
[The back end & its database](/learn/web-dev/backend-and-database/)

**SSR (server-side rendering)** — Building the page's HTML on the server for each
request, so content arrives ready to display. See
[SSR vs. SPA vs. static](/learn/web-dev/ssr-spa-static/)

**SPA (single-page application)** — An app that loads once and then renders and
re-renders pages in the browser with JavaScript. See
[SSR vs. SPA vs. static](/learn/web-dev/ssr-spa-static/)

**Templating** — Generating HTML by filling a template with data, on the server or at
build time. See [Templating & static site generators](/learn/web-dev/templating-and-static-sites/)

**Static site generator** — A tool (like the Jekyll behind this site) that builds a set
of static HTML pages from templates and content ahead of time. See
[Templating & static site generators](/learn/web-dev/templating-and-static-sites/)

## Auth &amp; sessions

**Authentication** — Verifying *who* a user is, typically at login with a password or
token. See [Authentication & sessions](/learn/web-dev/authentication-and-sessions/)

**Authorization** — Deciding *what* an authenticated user is allowed to do — checked on
every request, not just at login. See
[Authentication & sessions](/learn/web-dev/authentication-and-sessions/)

**Session** — Server-side state about one logged-in user, referenced by an opaque
session ID the browser sends back on each request. See
[Authentication & sessions](/learn/web-dev/authentication-and-sessions/)

**Password hashing** — Storing only a one-way, salted hash of a password (via bcrypt,
scrypt, or Argon2), never the password itself. See
[Authentication & sessions](/learn/web-dev/authentication-and-sessions/)

**Salt** — A unique random value mixed into each password hash so identical passwords
don't hash alike and precomputed attacks fail. See
[Authentication & sessions](/learn/web-dev/authentication-and-sessions/)

**Cookie** — A small named value the browser stores and sends *automatically* on
matching requests — the classic home of a session ID. See
[Cookies, tokens & JWTs](/learn/web-dev/cookies-tokens-jwt/)

**HttpOnly / Secure / SameSite** — Cookie flags that harden a session: **HttpOnly**
hides it from JavaScript, **Secure** sends it only over HTTPS, **SameSite** limits
cross-site sending. See [Cookies, tokens & JWTs](/learn/web-dev/cookies-tokens-jwt/)

**Bearer token** — A credential the client sends *deliberately* in an `Authorization`
header, common for APIs and non-browser clients. See
[Cookies, tokens & JWTs](/learn/web-dev/cookies-tokens-jwt/)

**JWT (JSON Web Token)** — A signed (not encrypted) token carrying its own claims, so
the server can trust it without a lookup — powerful but hard to revoke before it
expires. See [Cookies, tokens & JWTs](/learn/web-dev/cookies-tokens-jwt/)

**Refresh token** — A longer-lived credential used to mint fresh short-lived access
tokens, so tokens can expire quickly without forcing constant logins. See
[Cookies, tokens & JWTs](/learn/web-dev/cookies-tokens-jwt/)

## Real-time

**WebSocket** — A single persistent, bidirectional connection either side can send over
at any time, letting a server push data the instant it has it. See
[WebSockets & real-time updates](/learn/web-dev/websockets-and-realtime/)

**Polling** — Repeatedly asking the server "anything new?" over ordinary HTTP — simple
but wasteful compared with a push channel. See
[WebSockets & real-time updates](/learn/web-dev/websockets-and-realtime/)

**Server-sent events (SSE)** — A one-way (server → browser) live stream over ordinary
HTTP that reconnects automatically — simpler than WebSockets when the browser needn't
talk back. See [WebSockets & real-time updates](/learn/web-dev/websockets-and-realtime/)

## Security

**XSS (cross-site scripting)** — An attack that runs an attacker's JavaScript in your
users' browsers; defended by encoding output and a content security policy. See
[Web security essentials](/learn/web-dev/web-security-essentials/)

**CSRF (cross-site request forgery)** — An attack that tricks a logged-in browser into
sending an unintended request; defended with anti-CSRF tokens and `SameSite` cookies.
See [Web security essentials](/learn/web-dev/web-security-essentials/)

**CORS (cross-origin resource sharing)** — The browser rules that control which *other*
origins may read your responses, relaxing the same-origin policy — not a server-side
access control. See [Web security essentials](/learn/web-dev/web-security-essentials/)

**Same-origin policy** — The browser default that JavaScript on one origin can't read
responses from another, which CORS selectively relaxes. See
[Web security essentials](/learn/web-dev/web-security-essentials/)

**Content Security Policy (CSP)** — An HTTP header restricting which scripts the browser
will run, blunting XSS even if injection slips through. See
[Web security essentials](/learn/web-dev/web-security-essentials/)

## Speed &amp; caching

**Caching** — Storing a result so it can be reused instead of recomputed or refetched —
the foundation of a fast site. See [Caching & CDNs](/learn/web-dev/caching-and-cdns/)

**Cache-Control / ETag** — HTTP headers that steer the browser cache: **Cache-Control**
sets how long a copy is good, an **ETag** lets the browser cheaply revalidate for a
`304 Not Modified`. See [Caching & CDNs](/learn/web-dev/caching-and-cdns/)

**CDN (content delivery network)** — A worldwide network of edge servers that cache
copies of your files and serve each user from a nearby one, cutting latency. See
[Caching & CDNs](/learn/web-dev/caching-and-cdns/)

**Cache invalidation** — The hard problem of knowing when a cached copy is stale,
usually sidestepped with content-hashed filenames so a changed file gets a new URL. See
[Caching & CDNs](/learn/web-dev/caching-and-cdns/)

**Core Web Vitals** — Google's three user-centred performance metrics: **LCP** (loading),
**INP** (interactivity), and **CLS** (visual stability). See
[Performance & Core Web Vitals](/learn/web-dev/performance-and-web-vitals/)

**Perceived performance** — How fast a page *feels* to a user, which matters more than
raw load time and is what the Web Vitals try to capture. See
[Performance & Core Web Vitals](/learn/web-dev/performance-and-web-vitals/)

**Lazy loading** — Deferring the load of offscreen or non-critical assets (like images)
so they don't delay the content the user can see. See
[Performance & Core Web Vitals](/learn/web-dev/performance-and-web-vitals/)

## Shipping &amp; running

**Deployment** — Making your app run reliably and unattended on infrastructure the
public can reach — the step from *works on my laptop* to *running for users*. See
[Deploying a web app](/learn/web-dev/deploying-a-web-app/)

**Build / artifact** — The repeatable step that turns source into exactly what runs in
production, producing a fixed, versioned **artifact** you deploy. See
[Deploying a web app](/learn/web-dev/deploying-a-web-app/)

**Environment variable** — Configuration and secrets injected by the host at runtime
rather than hard-coded, so one build runs in any environment. See
[Deploying a web app](/learn/web-dev/deploying-a-web-app/)

**Reverse proxy** — A server in front of your app that presents the public HTTPS front
door — terminating TLS, routing, and load-balancing — while your app speaks plain HTTP
behind it. See [Deploying a web app](/learn/web-dev/deploying-a-web-app/)

**TLS / HTTPS** — The encryption that secures data in transit; a reverse proxy usually
terminates it to serve your site over HTTPS. See
[Deploying a web app](/learn/web-dev/deploying-a-web-app/)

**Monitoring** — Watching a live app's health — uptime, errors, latency — so you learn
about problems proactively, before users report them. See
[Monitoring & analytics](/learn/web-dev/monitoring-and-analytics/)

**Health check** — A small endpoint that returns "OK" when the app is alive, pinged
constantly so a crash or bad deploy is caught in minutes. See
[Monitoring & analytics](/learn/web-dev/monitoring-and-analytics/)

**Error tracking** — Capturing and grouping exceptions from production (server and
browser) so you see how many users a bug affects. See
[Monitoring & analytics](/learn/web-dev/monitoring-and-analytics/)

**Analytics** — Measuring user behaviour to guide product decisions, ideally with
privacy-respecting tools that count aggregates instead of profiling people. See
[Monitoring & analytics](/learn/web-dev/monitoring-and-analytics/)

**Web stack** — The coherent set of choices — front-end approach, back-end language,
database, hosting — matched to a project's real requirements and the team's skills. See
[Choosing your web stack](/learn/web-dev/choosing-a-web-stack/)

**Boring technology** — The principle of preferring mature, proven tools over trendy
ones, because novelty is a hidden cost. See
[Choosing your web stack](/learn/web-dev/choosing-a-web-stack/)
