---
layout: page
title: "RF Scope: Wireshark for the airwaves"
description: A 10-part hands-on series on GopherTrunk's rfscope — protocol-agnostic RF network analysis. Point it at any band with no prior knowledge and get a structured Scene: an RF protocol hierarchy, per-channel I/O graph, TDMA timing, an emitter/conversation graph, an encryption triage, and expert-info anomalies.
keywords: rfscope, rf scope, wireshark for rf, rf network analysis, protocol-agnostic sdr, rf protocol hierarchy, burst detection, tdma period, frequency hopper detection, encryption triage, gophertrunk
nav_group: Blog
permalink: /blog/series/rf-scope/
---

**RF Scope** is a 10-part tutorial series on
[rfscope]({{ '/rfscope/' | relative_url }}) — GopherTrunk's protocol-agnostic
RF network analyzer, **"Wireshark for the RF physical layer."** Point it at any
band, a recorded capture or a live SDR, with **no prior knowledge of the
technology, modulation, framing, or encryption**, and it produces a structured
**Scene**: what's on the air and how it behaves. We start from your first scene
summary and build to segmentation tuning, live cockpit analysis, and scripting
the whole thing into a reverse-engineering pipeline.

A note up front: despite the name, RF Scope is **not** a waterfall or
spectrum-scope UI — it is an *analyzer* that turns raw IQ into tables, trees,
and graphs. If you want live constellations and eye diagrams, that's
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}).

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose for the deep
read. **Ada** (new operator) and **Reese** (RF veteran) walk it with you.

This is one leg of the **Lab Bench trilogy**:
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}),
[RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}) (this one), and
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}). A mystery signal —
*Mercury* — runs through all three.

{%- assign parts = site.posts | where: "series", "RF Scope" | sort: "series_part" -%}
{%- if parts and parts.size > 0 -%}
<ol class="post-list series-list">
  {%- for post in parts -%}
    <li class="post-card">
      <a class="post-card__link" href="{{ post.url | relative_url }}">
        <h2 class="post-card__title">{{ post.title }}</h2>
      </a>
      <p class="post-card__meta">
        <time datetime="{{ post.date | date_to_xmlschema }}">{{ post.date | date: "%B %-d, %Y" }}</time>
        <span class="category-chip">Tutorials</span>
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
  See all <a href="{{ '/blog/category/tutorials/' | relative_url }}">tutorials</a>
  or subscribe via <a href="{{ '/feed.xml' | relative_url }}">RSS</a>.
</p>
