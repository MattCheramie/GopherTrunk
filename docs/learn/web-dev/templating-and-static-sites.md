---
slug: templating-and-static-sites
title: Templating & static site generators
description: Generating HTML from templates and data instead of writing every page by hand — how server-side templating fills a layout per request, how static site generators like Jekyll build a whole site at once, and where each fits.
keywords: templating, template engine, static site generator, Jekyll, SSG, layout, template, front matter, Liquid, server-side templating, build a static site
level: intermediate
status: full
prereq:
  - ssr-spa-static
faq:
  - q: What is a template?
    a: "A template is HTML with placeholders and a little logic — loops, conditionals, variable slots — that gets filled with data to produce a finished page. Instead of writing a hundred near-identical pages by hand, you write one template and render it many times with different data, which keeps the markup consistent and the content separate."
  - q: What is a static site generator?
    a: "A static site generator (SSG) runs your templates plus content files through a build step once, producing a folder of plain HTML you can serve anywhere. Jekyll, which builds this site, is one; Hugo, Eleventy, and Astro are others. It's server-side templating moved to build time so there's no rendering work per visit."
  - q: When should I use a static site generator versus server-side templating?
    a: "Use an SSG when the content is the same for everyone and changes at a manageable pace — docs, blogs, marketing — because building ahead of time is the fastest and cheapest to serve. Use per-request server-side templating when pages must be personalised or always fresh, since those can't be baked at build time."
---

# Templating & static site generators

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Rather than hand-writing every page, you generate HTML from **templates** — layouts with
placeholders and light logic — filled with **data**. **Server-side templating** does this
**per request** (the classic [SSR](/learn/web-dev/ssr-spa-static/) path). A **static site
generator** does it **once at build time**, producing plain HTML files to serve anywhere —
the fastest, cheapest option for content that's the same for everyone. **Jekyll**, which
builds **this very site**, is an SSG: Markdown and templates in, a static folder out, then
[deployed](/learn/deployment/what-is-deployment/) as files. Same idea, different timing.
</div>

The [rendering lesson](/learn/web-dev/ssr-spa-static/) put "when is the HTML built" on a
spectrum. This lesson is the *how* for two points on it — server-side templating and static
generation — which share one mechanism: **templates plus data produce HTML.** It's how most
of the web's content pages are made, from a server rendering a profile page to the static
generator that turned the Markdown you're reading into this HTML. Understanding it demystifies
both a lot of back-end code and this site itself.

## Templates: HTML with slots and logic

Writing every page by hand doesn't scale — a hundred product pages share the same structure
and differ only in data. A **template** captures the structure once as HTML with
**placeholders** for values and a little **logic** (loops, conditionals), and you **render**
it against data to get a finished page.

```html
<!-- A template: static structure, dynamic slots. -->
<article>
  <h1>{{ call.talkgroup }}</h1>
  <p>Duration: {{ call.seconds }}s</p>
  <ul>
    {% for tag in call.tags %}
      <li>{{ tag }}</li>
    {% endfor %}
  </ul>
</article>
```

Render this with one call's data and you get that call's page; render it with another and you
get the next — same markup, different content. The win is **separation**: designers and
developers work on the template, the data comes from wherever data lives, and the two meet at
render time. Every ecosystem has template engines — Go's `html/template`, Jinja, ERB, Liquid,
Handlebars — but they all do this one job.

## Server-side templating: render per request

The traditional dynamic web renders templates **on the server, per request**. A request comes
in, the handler loads the data (often from the [database](/learn/web-dev/backend-and-database/)),
fills the template, and returns the finished HTML — the [SSR](/learn/web-dev/ssr-spa-static/)
strategy from the last lesson.

```go
// A handler renders a template with per-request data.
func callPage(w http.ResponseWriter, r *http.Request) {
    call := store.GetCall(r.PathValue("id"))   // fresh data for THIS request
    tmpl.Execute(w, call)                       // fill the template → HTML
}
```

Because it runs on each visit, server-side templating handles **personalised and always-fresh**
pages — a logged-in dashboard, a live listing — that can't be baked ahead of time. The price is
per-request rendering work, and importantly a **security responsibility**: template engines
**escape** inserted values by default so user data can't inject markup or scripts. Never disable
that escaping for untrusted data, or you open an [XSS](/learn/web-dev/what-a-backend-does/) hole
— the [web security lesson](/learn/web-dev/ssr-spa-static/) covers it, and it's why "trust
nothing from the client" extends all the way to how you render it.

## Static site generators: render once at build

A **static site generator (SSG)** takes the same templates-plus-data idea and moves it to
**build time**. You run a build step **once**; it renders every page and writes a folder of plain
**HTML** files. There's no application server rendering per visit — the output is just files, so
it's the fastest and cheapest to serve, and it drops straight onto a
[CDN](/learn/web-dev/responsive-design/) or any static host.

The trade is that the content is fixed until the **next build**, so SSGs fit content that's the
same for everyone and changes at a manageable pace: documentation, blogs, marketing sites — and
learning modules like this one. When the content changes, you **rebuild and redeploy**. Popular
generators include **Jekyll**, **Hugo**, **Eleventy**, and **Astro**; they differ in language and
speed but share the model — content and templates in, static HTML out.

## Jekyll: how this site is built

This site runs on **Jekyll**, so it's a concrete example you're literally reading. The lesson
content is **Markdown** with a block of **front matter** at the top — the `slug`, `title`, and
`description` metadata you'd see if you viewed the source of this file. Jekyll's build:

```text
Markdown + front matter  ─┐
Liquid templates/layouts ─┼─▶  jekyll build  ─▶  _site/  (plain HTML)  ─▶  served as files
_data/*.yml (site data)  ─┘
```

- **Front matter** — per-page metadata (title, description, keywords) the templates read.
- **Layouts and includes** — Liquid templates wrapping your content in the shared page chrome.
- **`_data` files** — structured data (like the module's lesson list) the templates loop over to
  build navigation.

Jekyll renders all of that **once** into a `_site/` folder of static HTML, which is then
[deployed](/learn/deployment/what-is-deployment/) as plain files — no server-side code runs when
you load a page, which is why it's fast and simple to host. This is the payoff of
[static vs. dynamic](/learn/web-dev/static-vs-dynamic/) made real: a rich, navigable site that is,
under the hood, just files a [reverse proxy](/learn/deployment/reverse-proxies-and-tls/) serves.

## Choosing between them

The decision mirrors the [rendering lesson](/learn/web-dev/ssr-spa-static/):

- **Static generation** when content is **shared and reasonably stable** — docs, blogs,
  marketing, this module. Build once, serve files, rebuild on change.
- **Server-side templating** when pages must be **personalised or always fresh** — anything that
  depends on who's asking or on data that changes every second.
- **Both, on one site** — statically generate the content pages and server-render (or hydrate
  into a [SPA](/learn/web-dev/ssr-spa-static/)) the interactive parts.

Whichever you use, the core skill is the same: express a page as a **template**, keep the
**content as data**, and let the render step — at build or per request — put them together. Master
that and a huge swath of web development, from a Go server's HTML to this Jekyll site, works the
same way.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a static site generator like Jekyll renders templates and content into plain HTML once at build time, then serves those files." markdown="0">
  <p class="knowledge-check__q">Quick check: when does a static site generator like Jekyll produce the HTML?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">On the server, freshly for each visitor's request</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Once at build time, writing plain HTML files that are then served as-is</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">In the browser, assembled by JavaScript after the page loads</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Templating** generates pages from **templates** (HTML with placeholders and light logic)
  filled with **data**, instead of hand-writing every page.
- **Server-side templating** renders **per request**, handling **personalised and fresh** pages;
  template engines **escape** inserted values to prevent XSS — don't turn that off for untrusted
  data.
- A **static site generator** renders the same way but **once at build time**, producing plain
  HTML files that are the fastest and cheapest to serve.
- **Jekyll builds this site**: Markdown + front matter + Liquid templates + `_data` →
  `jekyll build` → static HTML, then [deployed](/learn/deployment/what-is-deployment/) as files.
- **Choose static** for shared, stable content and **server-side templating** for personalised or
  always-fresh pages — or mix both on one site.
- The durable skill is the same everywhere: **template + data → HTML**, at build or per request.

Next up: [Authentication &amp; sessions](/learn/web-dev/authentication-and-sessions/).
