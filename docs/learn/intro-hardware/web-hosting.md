---
slug: web-hosting
title: Web & shared hosting
description: Shared web hosting is the cheapest way online — many sites on one managed server. Learn what it runs, what it hides, and when you outgrow it.
keywords: web hosting, shared hosting, managed hosting, cPanel, WordPress hosting, static site hosting, JAMstack, PHP MySQL hosting, when to upgrade from shared hosting
level: beginner
status: full
faq:
  - q: "What is shared web hosting?"
    a: "It's a service where many customers' websites live on a single physical server, and the hosting company manages everything underneath — the operating system, the web server, security updates, and backups. You upload your files (often through a control panel like **cPanel**) and your site is online. You don't get root access or a machine of your own; you rent a slice of a shared one for a few dollars a month."
  - q: "What can I run on shared hosting?"
    a: "Mostly websites built from **static HTML**, or **PHP with a MySQL database** — which is why WordPress is everywhere on shared hosting. Some managed platforms also support Python, Node.js, or Ruby in a controlled way. What you generally *cannot* run is arbitrary software, background daemons, or long-running processes, because those need control over the machine that shared hosting deliberately withholds."
  - q: "When should I move off shared hosting?"
    a: "When you hit its walls: you need to run a custom background service, install system packages, get root access, handle real traffic, or stop competing with noisy neighbors for resources. At that point a **[VPS](/learn/intro-hardware/vps/)** gives you your own slice of a server with full control, for a modest step up in price and responsibility."
---
# Web & shared hosting

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Shared hosting puts many sites on one managed server** — the host runs everything, you just upload files. **It's the cheapest, simplest way online** and perfect for blogs, small sites, and WordPress. **You give up control in exchange** — no root, no arbitrary software, no long-running processes. **You outgrow it** the moment you need real control or steady performance.
</div>

Now that we know what a computer is, the first place most software actually lands is the web — and the cheapest door into the web is shared hosting. It's where countless blogs, small business sites, and first projects begin, precisely because it asks almost nothing of you. This lesson covers what shared and managed web hosting really gives you, the kinds of sites it suits, the languages it runs, and the point where its convenience turns into a ceiling.

## What shared hosting actually is

**Shared hosting** is the arrangement where a hosting company packs many customers' websites onto a single physical server and manages that server for all of them. You don't see the machine. You get an account, a control panel, and somewhere to put your files.

The defining trait is that *the host manages everything underneath you*: the operating system, the web server (usually Apache or Nginx), the database engine, security patches, and often backups. Your job shrinks to uploading content and maybe clicking through a one-button WordPress installer. Most accounts come with a control panel — **cPanel** is the classic — that turns tasks like adding an email address, setting up a database, or installing an SSL certificate into point-and-click operations.

**Managed hosting** is the same idea taken further: platforms tuned for one job, like managed WordPress hosts or static-site and JAMstack platforms (think Netlify-style services), that handle even more for you in exchange for staying inside their lane.

## What it's for

Shared hosting fits any site where the content matters more than the machine:

- **Blogs and personal sites** — often WordPress, often a few visitors at a time.
- **Small business and brochure sites** — a handful of pages that rarely change.
- **Static sites** — plain HTML, CSS, and images with no server-side code at all, which are fast and cheap to host anywhere.
- **JAMstack sites** — static front-ends that pull dynamic data from external APIs, a popular pattern on platforms like Netlify or similar static hosts.

If your project is "get this site in front of people without becoming a system administrator," shared hosting is built for exactly that.

## The languages and stacks it runs

Shared hosting grew up around a specific, durable stack:

- **Static HTML/CSS/JS** — the simplest case, just files served as-is.
- **PHP + MySQL** — the workhorse of shared hosting and the foundation of WordPress, Joomla, and most classic content systems. Most cheap plans assume this combination.
- **Python, Node.js, or Ruby** — supported on *some* managed platforms, but usually in a constrained way rather than the free-for-all you'd get on your own server.

The pattern to notice: shared hosting is excellent at the stacks it was designed for and awkward or impossible for anything outside them.

## Strengths and drawbacks

The trade is straightforward — you swap control for simplicity. Here is the balance:

| Strengths | Drawbacks |
|-----------|-----------|
| **Cheap** — often a few dollars a month | **No root access** — you can't install system software |
| **Zero sysadmin** — host handles OS, updates, backups | **Limited control** — you live inside the host's rules |
| **Easy** — control panel, one-click installers | **Noisy neighbors** — other sites on the box can hog resources |
| **Fast to start** — online in minutes | **No long-running processes** — no custom daemons, bots, or services |
| **Predictable** — flat monthly price | **Hard limits** — caps on CPU, memory, and processes you can't lift |

The strengths are real and the drawbacks are not flaws so much as the *deal*: the host keeps control so you don't have to manage anything, and that same control is exactly what you can't have.

## Where you outgrow it

The ceiling shows up the moment your project wants something the host won't allow. You want to run a background worker, install a system package, schedule a long-lived process, or simply stop sharing a crowded machine with strangers' traffic spikes.

Consider **GopherTrunk**: it isn't a website, it's a long-running program that processes a stream of radio data and exposes a dashboard. Shared hosting flatly can't run that — there's no place for a persistent custom process and no control over the environment. That kind of workload is the signal that you've outgrown shared hosting and want your own slice of a server instead.

<div class="knowledge-check" data-quiz data-correct-msg="Right — shared hosting runs many managed sites on one server, trading control for simplicity." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the defining trade-off of shared hosting?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">You get full root control but must manage the whole machine yourself</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">You give up control and arbitrary software in exchange for a cheap, fully managed place to put a site</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">You get a dedicated physical machine that no one else can use</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Many sites, one managed server** — shared hosting packs customers onto a shared machine the host runs for everyone.
- **The host manages everything** — OS, web server, updates, and backups, usually through a control panel like cPanel.
- **Built for content sites** — blogs, small sites, static pages, WordPress, and JAMstack platforms.
- **PHP/MySQL and static files** — the native stack, with some managed support for Python or Node.
- **Cheap and effortless, but capped** — no root, no arbitrary software, no long-running processes, and noisy neighbors.
- **You outgrow it** when you need real control or to run something like a persistent GopherTrunk process.

Next up: your own slice of a server, with root access and the freedom (and responsibility) that comes with it. See [Virtual private servers (VPS)](/learn/intro-hardware/vps/).
