---
layout: page
title: "The Operator's Cookbook: complete GopherTrunk builds, one rig per part"
description: A 14-part series of copy-paste GopherTrunk recipes — a $40 P25 starter rig, DMR Tier III and two-slot conventional DMR, TETRA TMO and Direct Mode, analog FM and tone-out, multi-SDR pools, remote radios, Broadcastify streaming, a FLAC archival rig, a headless Pi appliance, a two-antenna diversity build, and the fully annotated kitchen-sink config.
keywords: gophertrunk setup guide, rtl-sdr p25 scanner setup, dmr scanner config, tetra sdr setup, raspberry pi police scanner, broadcastify feed setup, sdr trunking recipes, police scanner software config, gophertrunk cookbook
nav_group: Blog
permalink: /blog/series/operator-cookbook/
---

**The Operator's Cookbook** is a 14-part series of complete, copy-paste
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) builds — one working
rig per part, antenna to browser. Every other tutorial series on this blog is
organized by subsystem; this one is organized by **what you're trying to
build**. Each part is a full recipe: the hardware list, a working
`config.yaml`, the first-run verification (the exact log lines and web panels
that mean it's working), a troubleshooting table, and variations to grow into.

The recipes run from a **$40 P25 starter rig** through DMR Tier III, two-slot
conventional DMR, TETRA TMO and Direct Mode, analog FM with tone-out paging,
multi-dongle SDR pools, remote radios over the network, streaming to
Broadcastify and friends, a FLAC-everything archival rig, a headless
Raspberry Pi appliance, and a two-antenna diversity build — ending with the
kitchen-sink config, annotated line by line. The cookbook tells you **what**
to do; [The Hunt]({{ '/blog/series/the-hunt/' | relative_url }}),
[Running It For Real]({{ '/blog/series/running-it-for-real/' | relative_url }}),
[Recording, Composition & Streaming]({{ '/blog/series/recording-streaming/' | relative_url }})
and [The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}) tell
you **why** it works.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
configs for the deep read.

New here? Start with
[What do I need for GopherTrunk?]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }})
for the hardware basics, then pick the recipe that matches your area.

{%- assign parts = site.posts | where: "series", "The Operator's Cookbook" | sort: "series_part" -%}
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
