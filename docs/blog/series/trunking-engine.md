---
layout: page
title: "Trunking Engine: from control-channel grant to recorded call"
description: A 12-part deep dive into GopherTrunk's trunking engine — the event-driven state machine that turns control-channel grants into tuned, recorded calls, with voice-pool allocation, priority preemption, RID and affiliation tracking, patches, multi-site roaming, and encrypted-mode handling.
keywords: trunking engine, p25 trunking, control channel grant, voice pool preemption, source rid recovery, affiliation tracking, talkgroup patch, multi-site roaming, encrypted mode, event bus, GopherTrunk
nav_group: Blog
permalink: /blog/series/trunking-engine/
---

**Trunking Engine** is a 12-part deep dive into the "brain" of
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) — the component that
turns a stream of control-channel **grants** into tuned, recorded **calls**. The
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) series gave
this engine a single survey episode and flagged it as "worth its own series."
This is that series.

Each part explains *what* the engine does with one class of event — a grant, a
priority conflict, an affiliation, a patch, an encrypted call — *how* it's built
in pure Go around a single-writer `select` loop and an in-process event bus, and
the **software-design principle** that keeps the core testable without a radio.
Several parts carry a **"problem we hit"** war story from real field captures
(the source-RID recovery work behind issue #915, encrypted-mode tuner
starvation).

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New to trunked radio first? Start with the
[Digital Trunking]({{ '/learn/digital-trunking/' | relative_url }}) path, then
come back for the implementation.

{%- assign parts = site.posts | where: "series", "Trunking Engine" | sort: "series_part" -%}
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
