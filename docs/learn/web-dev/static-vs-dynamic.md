---
slug: static-vs-dynamic
title: Static vs. dynamic sites
description: The spectrum from a file served as-is to a page built on the fly — what static and dynamic mean, the tradeoffs in speed, cost, and freshness, why this learning site is static, and how modern approaches blur the line.
keywords: static site, dynamic site, static vs dynamic, static site generator, server-rendered, CDN, prerendered HTML, when to use static, web app vs website, Jekyll
level: beginner
status: full
prereq:
  - anatomy-of-a-web-app
faq:
  - q: What is the difference between a static and a dynamic site?
    a: "A **static** site serves pre-built files exactly as they are — every visitor gets the same file. A **dynamic** site builds each page on request, often personalised and drawn from a database. Static is faster and cheaper to serve; dynamic can show fresh, per-user content. Most real sites mix both."
  - q: Is a static site the same as a simple site?
    a: "Not necessarily. \"Static\" describes how pages are served — as fixed files — not how sophisticated they are. A static site can have rich design and interactivity through JavaScript running in the browser; what makes it static is that the server hands over the same file rather than generating it per request."
  - q: Why would you choose a static site?
    a: "For speed, cost, security, and reliability. Serving a fixed file is extremely fast and cheap, has almost no attack surface, and rarely breaks. If your content is the same for everyone and doesn't change every second — docs, blogs, marketing, this learning site — static is often the best choice."
---

# Static vs. dynamic sites

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **static** site serves pre-built files unchanged — every visitor gets the same
HTML. A **dynamic** site builds each page on request, often personalised and
pulled from a **database**. Static wins on **speed, cost, and security**; dynamic
wins when content must be **fresh or per-user**. It's a spectrum, not a binary, and
modern tools deliberately blur the line. This very site is **static** — generated
ahead of time and served as plain files — which is why it loads fast and rarely
breaks.
</div>

The last lesson mapped the four parts of a web app and noted that real sites use
different amounts of each. This lesson turns that observation into a spectrum. At
one end a page is a **file served as-is**; at the other it's **assembled fresh for
every request**. Understanding where a project sits on that line drives real
decisions about speed, cost, and complexity — including the choice behind the pages
you're reading now.

## What "static" means

A **static** site is a set of files — HTML, CSS, JavaScript, images — sitting on a
server. When a request comes in, the server finds the matching file and sends it
back **unchanged**. There's no logic, no database lookup, no per-visitor assembly:
everyone who asks for `/about` gets the exact same `about.html`.

That doesn't mean *boring*. A static page can still be beautifully designed and can
run JavaScript in the browser to be interactive — animations, menus, even fetching
data from some other API after it loads. What makes it static is only this: the
**server did no work to build it**. It reached for a finished file and handed it
over.

## What "dynamic" means

A **dynamic** site builds the page **on request**. When a request arrives, the
back end runs code — checks who you are, queries the database, applies logic — and
**generates** the HTML (or data) for that specific request, then sends it back.

This is what lets a page be different for every visitor and every moment: your
name in the corner, your account's data, the current contents of a constantly
changing list. A social feed, a bank dashboard, an email inbox — all dynamic,
because the whole point is that the page reflects *live, personalised* state that
couldn't have been baked into a file in advance.

## The tradeoffs

Neither is better in the abstract; they trade different things.

| | Static | Dynamic |
|---|--------|---------|
| **Speed** | Very fast — just hand over a file | Slower — must build each page |
| **Cost** | Cheap — minimal server work | Higher — servers, database, logic |
| **Freshness** | Fixed until rebuilt | Live, up-to-the-second |
| **Personalisation** | Same for everyone | Per-user |
| **Security** | Tiny attack surface | More to defend |
| **Complexity** | Simple to run | More moving parts |

The pattern is clear: static is **fast, cheap, and simple** but shows the **same
fixed content** to everyone; dynamic is **flexible and fresh** but **costs more**
in servers, complexity, and things that can break. The right choice follows from
what the content actually needs to do.

## Why this site is static

This learning site is **static**, and it's a good illustration of why you'd
choose that. The content — these lessons — is the same for every reader and
changes only when an author edits it, not second by second. So there's no reason
to build each page on every request. Instead the pages are generated **ahead of
time** by a **static site generator** (Jekyll, in this case) and served as plain
files.

The payoff is exactly the static column above: pages load fast, hosting is cheap,
there's almost nothing to attack because there's no live back end or database in
the request path, and it's very hard to break. When your content fits — docs,
blogs, marketing, reference — static is often the *right* engineering call, not a
lesser one. We cover how these generators work in
[templating & static site generators](/learn/web-dev/templating-and-static-sites/).

## A spectrum, and the blur

In practice, the line is deliberately fuzzy, and most real sites live in the
middle:

- A mostly **static** site can fetch dynamic data in the browser after it loads —
  static shell, live contents.
- A **static site generator** can pre-build thousands of pages from a database at
  build time, so pages that *look* dynamic are actually served as files.
- A **dynamic** site can **cache** its generated pages so repeat requests are served
  fast, like static ones — the subject of
  [caching & CDNs](/learn/web-dev/caching-and-cdns/).

The deeper question — render on the server, in the browser, or ahead of time — is a
lesson of its own, [SSR vs. SPA vs. static](/learn/web-dev/ssr-spa-static/). For
now, hold the spectrum in mind: not "which one," but "how much of each, and where."

<div class="knowledge-check" data-quiz data-correct-msg="Right — a static site serves the same pre-built file to everyone; the server does no per-request work to build it." markdown="0">
  <p class="knowledge-check__q">Quick check: what defines a static site?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It has no JavaScript and can't be interactive at all</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The server sends pre-built files unchanged, doing no per-request work</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It stores its pages in a database instead of files</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **static** site serves pre-built files unchanged; a **dynamic** site builds
  each page on request from logic and a database.
- Static means the **server did no work to build the page** — it can still be
  designed and interactive in the browser.
- Dynamic means the page is **generated per request**, enabling fresh and
  per-user content.
- The tradeoffs: static is **faster, cheaper, simpler, and safer**; dynamic is
  **fresher and personalised** but more complex.
- **This site is static**, generated ahead of time by a static site generator and
  served as files — the right call for content that's the same for everyone.
- It's a **spectrum**: static shells fetch live data, generators pre-build many
  pages, and dynamic sites cache — the line is intentionally blurred.

Next up: [HTML — structure & semantics](/learn/web-dev/html-structure/).
