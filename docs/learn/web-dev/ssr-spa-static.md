---
slug: ssr-spa-static
title: SSR vs. SPA vs. static
description: The three ways to produce the HTML a browser shows — rendered on the server per request (SSR), built in the browser by JavaScript (SPA), or generated ahead of time as static files — what each is good at, their trade-offs, and how to choose.
keywords: SSR, SPA, static site, server-side rendering, single-page app, static site generation, SSG, hydration, rendering strategy, time to first byte, SEO, isomorphic
level: intermediate
status: full
prereq:
  - fetching-data
faq:
  - q: What's the difference between SSR and a SPA?
    a: "In server-side rendering (SSR) the server builds the full HTML for each request and sends a ready-to-show page. In a single-page app (SPA) the server sends a near-empty shell plus JavaScript, and the browser builds the page and fetches data itself, then updates in place without full reloads. SSR is faster to first paint and better for SEO; a SPA feels more app-like after it loads."
  - q: What does 'static' mean here?
    a: "A static site is HTML generated ahead of time — at build, not per request — and served as plain files. There's no server rendering work per visit, so it's the fastest and cheapest to serve. It's ideal for content that's the same for everyone, like docs, blogs, or marketing pages. This site is static."
  - q: Do I have to pick just one?
    a: "No. Modern frameworks mix them — statically generate some pages, server-render others, and hydrate into an interactive SPA on the client. The three are points on a spectrum of when the HTML is built (ahead of time, per request, or in the browser), and real apps often use more than one."
---

# SSR vs. SPA vs. static

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
There are three ways to produce the HTML the browser shows, differing in **when the HTML
is built**. **Static** generates it **ahead of time** at build and serves plain files —
fastest and cheapest, best for content that's the same for everyone (this site).
**SSR** builds the HTML **on the server per request** — great for first paint, SEO, and
personalised pages. A **SPA** ships a shell plus JavaScript and builds the page **in the
browser**, then updates in place with [fetch](/learn/web-dev/fetching-data/) — the most
app-like feel, but slower to start. Real frameworks **mix** all three.
</div>

You've seen the front end ([components and fetch](/learn/web-dev/components-and-state/))
and the back end ([servers and APIs](/learn/web-dev/http-servers-and-routing/)). This
lesson is the question that sits between them: **where and when does the HTML get built?**
The answer — on the server per request, in the browser, or ahead of time — is the
**rendering strategy**, and it shapes performance, SEO, and how the app feels. The three
options are really points on one spectrum of *timing*, so understanding that axis matters
more than memorising labels.

## Static: built ahead of time

The simplest strategy: generate the HTML **once, at build time**, and serve the resulting
files as-is to every visitor. No per-request rendering, no application server in the hot
path — just files. That makes static sites the **fastest and cheapest** to serve and the
easiest to put on a [CDN](/learn/web-dev/responsive-design/) close to users.

The catch is that the content is the **same for everyone** until the next build, so it
suits pages that don't change per user or per request: documentation, blogs, marketing
sites, this very learning module. When the data does change, you **rebuild** and redeploy.
This is exactly what a [static site generator](/learn/web-dev/templating-and-static-sites/)
does, and it connects to [static vs. dynamic](/learn/web-dev/static-vs-dynamic/) from Unit 1
— static here means the *rendering is done up front*.

## SSR: built on the server per request

**Server-side rendering** builds the full HTML **on the server, fresh for each request**,
and sends a **complete, ready-to-show page**. The browser gets meaningful content on the
very first response — no waiting for JavaScript to construct the page — which is good for:

- **First paint / time-to-content** — the page shows immediately, even on slow devices.
- **SEO and link previews** — crawlers and social scrapers see real HTML, not an empty
  shell they'd have to run JavaScript to fill.
- **Personalised or always-fresh pages** — each request can render the current data for
  *this* user, which static can't.

The cost is that the server does rendering work on **every request**, so it needs compute
and is slower and pricier to serve than static files. SSR is the classic model behind
[templating](/learn/web-dev/templating-and-static-sites/) — a server building HTML from a
template and data per request — and it's making a strong comeback in modern frameworks.

## SPA: built in the browser

A **single-page app** flips it: the server sends a near-**empty HTML shell** plus a bundle
of JavaScript, and **the browser builds the page**. From then on the app **fetches data**
([the fetch lesson](/learn/web-dev/fetching-data/)) and updates the page **in place** —
navigating between "pages" without a full reload, swapping content instantly. That's the
snappy, app-like feel of a dashboard or editor.

```text
SPA first load:  server → tiny HTML shell + JS bundle
                 browser runs JS → builds UI → fetch()es data → renders
Later actions:   browser fetch()es data → updates the page in place (no reload)
```

The trade-off is the **start**: the browser must download and run the JavaScript before
anything meaningful appears, so first paint is slower, and a naive SPA is weak for SEO
because the initial HTML is nearly empty. SPAs shine **after** they load — for highly
interactive, stateful apps where the user stays a while and the up-front cost is worth it.
GopherTrunk's live [dashboard](/architecture.html) leans this way: load once, then stream
updates in place.

## Hydration: the hybrid

The two dynamic approaches combine. A page can be **server-rendered (or statically
generated) for a fast, SEO-friendly first paint**, and then the same JavaScript **hydrates**
it in the browser — attaching event handlers and taking over so it behaves like a SPA from
there. You get the best of both: instant meaningful content *and* rich interactivity. This
"render on the server, hydrate on the client" pattern is the default in frameworks like
Next.js, Nuxt, and SvelteKit, which is why the SSR-vs-SPA argument has softened into "use
each where it fits." The mental model to keep is the timing axis: **build ahead of time,
per request, or in the browser** — hydration simply uses more than one point on it.

## Choosing

Match the strategy to the page, not to fashion:

- **Static** — content that's the same for everyone and changes rarely: docs, blog,
  marketing, this site. Fastest, cheapest, simplest.
- **SSR** — content that must be fresh, personalised, or SEO-critical on first load: a
  news feed, a product page, a logged-in home screen.
- **SPA** — highly interactive, stateful apps where users stay a while: dashboards,
  editors, tools. Often SSR/static for the first paint, then hydrated.

You don't have to pick one for the whole app — statically generate the marketing pages,
server-render the product pages, and hydrate the dashboard into a SPA. The question is
always the same: **when should this page's HTML be built, and by whom?**

<div class="knowledge-check" data-quiz data-correct-msg="Right — a static site's HTML is generated ahead of time at build and served as files, which is why it's the fastest and cheapest to serve." markdown="0">
  <p class="knowledge-check__q">Quick check: when is the HTML built for a static site?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">On the server, freshly for each incoming request</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Ahead of time at build, then served as plain files</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">In the browser by JavaScript after the shell loads</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The three strategies differ in **when the HTML is built**: ahead of time, per request,
  or in the browser.
- **Static** builds HTML at build time and serves files — fastest and cheapest, for
  content that's the same for everyone (like this site).
- **SSR** builds HTML on the server per request — best for first paint, SEO, and
  personalised or always-fresh pages, at the cost of per-request compute.
- A **SPA** ships a shell plus JavaScript and builds the page in the browser, updating in
  place with [fetch](/learn/web-dev/fetching-data/) — app-like, but slower to start and
  weaker for SEO.
- **Hydration** combines them: server-render or statically generate for first paint, then
  hydrate into a SPA — the modern default.
- **Choose per page**, matching the strategy to whether the content is shared, fresh, or
  highly interactive.

Next up: [Templating &amp; static site generators](/learn/web-dev/templating-and-static-sites/).
