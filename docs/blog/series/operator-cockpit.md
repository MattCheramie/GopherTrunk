---
layout: page
title: "The Operator's Cockpit: driving GopherTrunk from browser and terminal"
description: A 14-part deep dive into GopherTrunk's operator surfaces — a React SPA embedded in the Go binary and a Bubbletea TUI, both speaking the same REST + SSE API — covering live audio streaming, spectrum and constellation canvases, the map, write-mode config, and a reflect-driven form that renders in both.
keywords: sdr web console, react spa embedded go, server sent events react, audioworklet pcm streaming, live spectrum waterfall browser, bubbletea tui, reflect config form, sdr operator ui, gophertrunk operator cockpit
nav_group: Blog
permalink: /blog/series/operator-cockpit/
---

**The Operator's Cockpit** is a 14-part deep dive into how you actually *drive* a
running [GopherTrunk](https://github.com/MattCheramie/GopherTrunk) — from a phone
browser against a Raspberry Pi, or from a full-screen terminal over SSH. The
engine decodes and records on its own; this series is about the two front-ends
that let a human watch it, listen to it live, and change it safely — a **React
SPA baked into the single Go binary** and a **Bubbletea TUI**, both speaking the
*same* REST + Server-Sent-Events API.

Where [Recording, Composition & Streaming]({{ '/blog/series/recording-streaming/' | relative_url }})
covered turning a decoded call into a file and a live audio feed, this series is
the surface on the other end of that feed: the live audio cockpit with its
AudioWorklet ring buffer, the spectrum and constellation canvases, the map, and
write-mode config. And where
[RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}) ended on a "scene
cockpit," this is the cockpit pattern generalized — *one API contract, two
renderers.*

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? The [Web console]({{ '/web.html' | relative_url }}) and
[TUI]({{ '/tui.html' | relative_url }}) operator docs show the finished surfaces;
this series is how they're built.

{%- assign parts = site.posts | where: "series", "The Operator's Cockpit" | sort: "series_part" -%}
{%- if parts and parts.size > 0 -%}
<ol class="post-list series-list">
  {%- for post in parts -%}
    <li class="post-card">
      <a class="post-card__link" href="{{ post.url | relative_url }}">
        <h2 class="post-card__title">{{ post.title }}</h2>
      </a>
      <p class="post-card__meta">
        <time datetime="{{ post.date | date_to_xmlschema }}">{{ post.date | date: "%B %-d, %Y" }}</time>
        <span class="category-chip">Deep dives</span>
      </p>
      {%- if post.description -%}
        <p class="post-card__desc">{{ post.description }}</p>
      {%- endif -%}
    </li>
  {%- endfor -%}
</ol>
{%- else -%}
<p class="post-list__empty">No posts in this series yet — check back soon.</p>
{%- endif -%}

<p class="blog-feed-link">
  See all <a href="{{ '/blog/category/deep-dives/' | relative_url }}">deep dives</a>
  or subscribe via <a href="{{ '/feed.xml' | relative_url }}">RSS</a>.
</p>
