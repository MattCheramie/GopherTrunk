---
title: "Build in the Open, Part 10: Websites, Support Pages & GitHub Pages"
description: How to give a project a free website — enabling GitHub Pages, a Jekyll static site, a custom domain via CNAME, support and sponsor pages, and a drip-released blog.
keywords: github pages, custom domain, jekyll, static site generator, project website, landing page, github sponsors, funding.yml, blog, claude code
category: tutorials
tags: [github, claude-code, github-pages, jekyll, website, seo]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 10
---

> **TL;DR:** Your project can have a real website for free. Turn on GitHub Pages,
> point it at a static-site generator like Jekyll, and you get a fast, secure,
> CDN-backed site built straight from your repo. Add a custom domain with a
> one-line `CNAME` file, build a landing page that sells the project, add
> support and sponsor pages so people can get help and chip in, and run a blog —
> including scheduled, drip-released posts. The whole thing lives in the same
> repo as your code and deploys on every push.

**Key takeaways**

- GitHub Pages hosts a static website from your repo, free, with HTTPS and a CDN.
- A static-site generator (Jekyll) turns Markdown into a real site — no servers.
- A custom domain is a one-line `CNAME` file plus a DNS record.
- Support and sponsor pages let users get help and fund the work.
- Future-dated posts plus a daily build = a blog that drip-releases itself.

*This is Part 10 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real.*

## In this post

- **What GitHub Pages is** and why it's the easy default.
- **Static-site generators** — what Jekyll buys you over hand-written HTML.
- **Custom domains** with a `CNAME` file.
- **Landing, support, and sponsor pages** — the pages a project actually needs.
- **Running a blog** — including scheduled, drip-released posts.
- **How GopherTrunk does it**, as a concrete example you can copy.

## What is GitHub Pages?

GitHub Pages is free static-site hosting built into GitHub. You enable it in
**Settings → Pages**, choose a source (a branch, a `/docs` folder, or — better —
GitHub Actions), and GitHub serves the result over HTTPS from a CDN at
`username.github.io/repo`. Because the site is *static* — plain HTML, CSS, and
JavaScript — there's nothing to run, nothing to patch, and nothing to pay for.

"Static" sounds limiting but rarely is. Docs, landing pages, blogs, and
marketing sites are all static by nature, and a static site is the fastest and
most secure thing you can ship.

## Why use a static-site generator?

Writing raw HTML for every page gets old fast. A **static-site generator** lets
you write content in Markdown, share layouts and navigation across pages, and
build the whole site with one command. **Jekyll** is the generator GitHub Pages
supports natively — Pages can build a Jekyll site for you with no extra setup.

What a generator buys you:

- **Markdown instead of HTML.** Write content; the generator wraps it in your
  layout.
- **Shared layouts and includes.** Change the header once, every page updates.
- **Plugins.** SEO tags, sitemaps, and RSS feeds for free.
- **A blog engine.** Drop a file in a posts folder and it becomes a post.

Other generators (Hugo, Eleventy, Astro, MkDocs) work on Pages too, usually via
a GitHub Actions build. Jekyll is just the path of least resistance.

## Custom domains with a CNAME

A `username.github.io` URL works, but a real project wants a real domain. Pages
makes this a two-step affair:

1. Add a file named **`CNAME`** to your site source containing just your domain,
   e.g. `example.org`.
2. At your DNS provider, point the domain at GitHub Pages (a `CNAME` record to
   `username.github.io`, or `A`/`AAAA` records to GitHub's IPs for an apex
   domain).

GitHub then provisions a free TLS certificate, and your site serves over HTTPS
on your own domain. One file, one DNS record, done.

## The pages a project actually needs

Beyond the docs, a few pages do real work:

- **A landing page.** The pitch: what the project is, who it's for, and a
  prominent path to download or get started. This is where you convert a curious
  visitor into a user.
- **A support / community page.** Where to ask questions and report problems —
  links to issues, a Discord or forum, a FAQ. Reduces the "how do I get help?"
  friction to zero.
- **A sponsor / funding page.** If you want support for the work, make it easy.
  GitHub reads a `.github/FUNDING.yml` file and renders a **Sponsor** button on
  your repo automatically; you can list GitHub Sponsors, Ko-fi, Patreon, and
  more.
- **A blog.** For release notes, tutorials, and build-in-the-open posts like this
  one — which doubles as SEO and a reason for people to come back.

## Running a blog (and drip-releasing posts)

A static blog is just dated files in a posts folder. The clever part is
*scheduling*. Most generators, Jekyll included, will **exclude a post dated in
the future** from the build. Pair that with a **daily scheduled rebuild** and you
get a powerful pattern:

1. Write a whole series at once, dating the posts on consecutive future days.
2. Commit them all in one pull request.
3. A daily build job rebuilds the site; each day, the posts that have "come due"
   appear automatically.

No draft branches, no daily commits, no manual publishing — the calendar does
the releasing for you. (This very series is published exactly that way.)

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk)'s entire website —
[gophertrunk.org](https://gophertrunk.org) — is a Jekyll site on GitHub Pages,
built from the repo's `docs/` folder. The pieces line up with everything above:

- **Custom domain via `CNAME`.** `docs/CNAME` contains exactly one line —
  `gophertrunk.org` — and that's what serves the site on its own domain over
  HTTPS.
- **The landing page is synthesized from the README at build time.** The Pages
  workflow (`.github/workflows/pages.yml`) has a "Synthesize landing page from
  README.md" step that transforms `README.md` into `docs/index.md` during the
  build — rewriting asset paths and cross-links and stripping the duplicate hero.
  The homepage and the README can never drift apart, because there's only one
  source (the doc-organization payoff from Part 9).
- **SEO comes from plugins.** `docs/_config.yml` enables `jekyll-seo-tag`
  (titles, meta descriptions, Open Graph/Twitter cards), `jekyll-sitemap`
  (`sitemap.xml`), and `jekyll-feed` (an Atom feed at `/feed.xml`) — so search
  engines and readers get structured metadata for free.
- **Support and sponsor links are real files.** There's a `docs/support.md`
  page, and `.github/FUNDING.yml` lists **GitHub Sponsors** (`github: MattCheramie`)
  and **Ko-fi** (`ko_fi: Mrcheramie`), which GitHub turns into the repo's Sponsor
  button.
- **The blog drip-releases itself.** `docs/_config.yml` sets `future: false` (a
  future-dated post stays out of the build until its date), and
  `pages.yml` runs a daily `cron: "0 16 * * *"` rebuild. A whole series is
  committed at once on consecutive future dates and goes live one post per day —
  with a `scripts/schedule-series.py` helper to assign the dates. That's the
  mechanism publishing the post you're reading now.

You don't need a radio project to copy this. Any repo can flip on Pages, drop a
`CNAME`, point Jekyll at a `/docs` folder, add a `FUNDING.yml`, and schedule a
blog — all for the price of a push.

## FAQ

**Is GitHub Pages really free?**
Yes, for public repositories, including HTTPS and CDN delivery, within generous
soft bandwidth and size limits. For a project site, docs, or blog you will almost
certainly never hit them.

**Do I have to use Jekyll?**
No. Jekyll is the generator Pages builds natively with zero config, but you can
deploy any static site — Hugo, Eleventy, Astro, plain HTML — by building it in a
GitHub Actions workflow and publishing the output. Jekyll is just the lowest-
friction option.

**How do I add a custom domain to GitHub Pages?**
Add a `CNAME` file containing your domain to the site source, then create a DNS
record at your registrar pointing the domain at GitHub Pages. GitHub provisions
a free TLS certificate automatically once DNS resolves.

**How do I schedule blog posts to publish on a future date?**
Date the post in the future and configure your generator to exclude future-dated
content (`future: false` in Jekyll), then run a daily scheduled build. Each
build reveals the posts whose date has arrived — so you can write a series now and
have it release one post per day.

**How do I add a Sponsor button to my repo?**
Add a `.github/FUNDING.yml` listing your funding platforms (GitHub Sponsors,
Ko-fi, Patreon, etc.). GitHub reads it and shows a Sponsor button on the repo
automatically — no other setup required.

## Series navigation

**Part 10 of 14** · ←
[Part 9]({{ '/blog/tutorials/build-in-the-open-09-documentation-done-right/' | relative_url }})
· Next →
[Part 11: Releases — Pre-release, SemVer & Changelogs]({{ '/blog/tutorials/build-in-the-open-11-releases-prerelease-semver-changelogs/' | relative_url }})
