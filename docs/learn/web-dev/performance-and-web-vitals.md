---
slug: performance-and-web-vitals
title: Performance & Core Web Vitals
description: What "fast" actually means on the web and how it's measured — loading, interactivity, and visual stability captured by Google's Core Web Vitals (LCP, INP, CLS) — plus the concrete levers that move each metric.
keywords: web performance, Core Web Vitals, LCP, largest contentful paint, INP, interaction to next paint, CLS, cumulative layout shift, page speed, perceived performance, lazy loading
level: intermediate
status: full
prereq:
  - caching-and-cdns
faq:
  - q: "What are the Core Web Vitals?"
    a: "Three user-centred metrics Google standardised. **LCP (Largest Contentful Paint)** measures *loading* — when the main content appears. **INP (Interaction to Next Paint)** measures *interactivity* — how quickly the page responds to a tap or click. **CLS (Cumulative Layout Shift)** measures *visual stability* — how much the layout jumps around as it loads. Together they approximate how fast a page *feels*, not just how fast it technically loads."
  - q: "Why does perceived performance matter more than raw load time?"
    a: "Because users experience *feel*, not milliseconds on a stopwatch. A page that shows meaningful content quickly and responds instantly to taps feels fast even if some assets are still loading in the background. Techniques like showing content progressively, reserving space for images, and prioritising visible content all improve the *perception* of speed, which is what the Web Vitals try to capture."
  - q: "Where do I even start optimising?"
    a: "Measure first — never guess. Use a tool like Lighthouse or field data from real users to find *your* actual bottleneck, then fix the biggest one. The common wins are almost always the same: send less (smaller images and JavaScript), send it from closer ([a CDN](/learn/web-dev/caching-and-cdns/)), and don't block the page while non-essential work runs."
---

# Performance & Core Web Vitals

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
"Fast" on the web is about how a page *feels*, and Google's **Core Web Vitals** put
numbers on it: **LCP** (loading — when main content appears), **INP** (interactivity —
response to taps), and **CLS** (visual stability — how much things jump). You improve
them by **sending less** (smaller images and JavaScript), **sending it from closer**
(a [CDN](/learn/web-dev/caching-and-cdns/)), and **not blocking** the page while
non-critical work runs. The rule above all: **measure before you optimise** — fix
your real bottleneck, not a guessed one.
</div>

Users abandon slow pages, search engines rank them lower, and "slow" is decided in
the first couple of seconds. But performance is easy to chase in the wrong direction —
tuning something that was never the problem. This lesson defines what fast *means* in
terms users actually feel, the standard way it's measured, and the handful of levers
that move it.

## Fast is a feeling, not a stopwatch

The trap in performance work is optimising a number no user notices. What matters is
**perceived performance** — how quickly the page seems ready and responsive. A page
that paints its headline and article instantly *feels* fast even while images and
analytics still load quietly in the background. So the goal isn't "everything loaded"
as early as possible; it's **the parts the user cares about, ready and responsive,
first**. The Core Web Vitals exist to measure exactly that feeling.

## The Core Web Vitals

Google standardised three **user-centred** metrics, each capturing a different way a
page can feel slow:

- **LCP — Largest Contentful Paint** (loading). When the biggest, main piece of
  content — usually the hero image or headline — finishes rendering. It answers *"has
  the useful content shown up yet?"* Aim for a couple of seconds or less.
- **INP — Interaction to Next Paint** (interactivity). How quickly the page visibly
  responds after the user taps, clicks, or types. A page that looks done but freezes
  when touched fails here — usually because heavy
  [JavaScript](/learn/web-dev/javascript-in-the-browser/) is hogging the main thread.
- **CLS — Cumulative Layout Shift** (visual stability). How much content jumps around
  as the page loads — the maddening moment a button shifts just as you tap it.
  Reserving space for images and ads keeps this low.

Together they approximate *loading*, *responsiveness*, and *stability* — the three
axes of how fast a page feels.

## The levers that move them

Most performance wins come from a short list, and they map cleanly onto the Vitals.

**Send less.** The biggest lever is fewer and smaller bytes:

- **Images** are usually the heaviest thing on a page — compress them, size them to
  how they're displayed, use modern formats, and **lazy-load** offscreen ones so they
  don't delay the visible content (helps **LCP**).
- **JavaScript** is expensive twice — to download *and* to run. Ship less of it, split
  it so each page loads only what it needs, and defer non-critical scripts so they
  don't block interactivity (helps **INP**).

```html
<!-- Lazy-load offscreen images; reserve space to avoid layout shift -->
<img src="photo.jpg" loading="lazy" width="800" height="600" alt="…">
```

**Send it from closer, and reuse it.** A [CDN](/learn/web-dev/caching-and-cdns/)
serves assets from near the user, and good **caching** means repeat visits fetch
almost nothing (helps **LCP**).

**Don't block, and reserve space.** Let the critical content render before secondary
work; always set image and element dimensions so the layout doesn't jump (helps
**CLS** and **INP**). The [rendering strategy](/learn/web-dev/ssr-spa-static/) matters
too — server-rendered or static HTML often shows content sooner than a client-rendered
app that must download and run JavaScript before anything appears.

## Measure first, always

The one rule that saves the most wasted effort: **measure before you optimise.** Your
intuition about the bottleneck is often wrong, and the fix for one site does nothing
for another. Two kinds of measurement matter:

- **Lab data** — a synthetic test in a tool like Lighthouse, run on demand, giving a
  reproducible score and a prioritised list of issues.
- **Field data** — real Core Web Vitals from actual users on real devices and
  networks, which is what ultimately counts.

Find *your* biggest bottleneck, fix it, measure again. Optimising by guesswork —
shaving milliseconds off code no one waits on — is the classic way to spend real
effort for no felt improvement. This measure-and-verify loop is the same discipline
the [monitoring](/learn/web-dev/monitoring-and-analytics/) lesson applies to the rest
of a running app.

<div class="knowledge-check" data-quiz data-correct-msg="Right — LCP measures loading (when the main content paints), one of the three Core Web Vitals." markdown="0">
  <p class="knowledge-check__q">Quick check: which aspect of the experience does LCP (Largest Contentful Paint) measure?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">How secure the page's HTTPS connection is</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Loading — when the main content of the page finishes rendering</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">How many database queries the back end runs</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Web performance is about **perceived speed** — how ready and responsive a page
  *feels*, not a raw stopwatch time.
- The **Core Web Vitals** measure three axes: **LCP** (loading — main content
  appears), **INP** (interactivity — response to input), and **CLS** (visual
  stability — how much the layout jumps).
- The main levers are **send less** (compress and lazy-load images, ship and defer
  less JavaScript), **send it from closer** (a CDN plus caching), and **don't block
  or shift** (render critical content first, reserve space for elements).
- Rendering strategy matters: static or server-rendered HTML often paints content
  sooner than a client-rendered app.
- Above all, **measure before you optimise** — use lab and field data to fix your real
  bottleneck, not a guessed one.

Next up: [monitoring & analytics](/learn/web-dev/monitoring-and-analytics/).
