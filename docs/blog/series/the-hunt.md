---
layout: page
title: "The Hunt: discovering trunked systems you didn't know were there"
description: A 14-part deep dive into GopherTrunk's discovery engine — wideband sweeps, peak and occupancy detection, signal classification, control-channel hunting, DMR LCN correlation, multi-site P25, alias harvesting, and exporting your finds, all in pure Go.
keywords: control channel hunting, wideband sweep sdr, signal discovery, occupancy detection, dmr lcn correlation, p25 site discovery, talker alias harvest, sigmf export, radioreference export, gophertrunk the hunt
nav_group: Blog
permalink: /blog/series/the-hunt/
---

**The Hunt** is a 14-part deep dive into the part of
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) that finds systems you
didn't know were there. Where the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) series
started from a *known* control channel and turned its grants into calls, this
series starts one step earlier — from a **blank band** — and works forward:
sweep the spectrum, pull carriers out of the noise, classify what each one is,
lock a control channel, reconstruct a system's channel map and its sites, harvest
the aliases riding on its traffic, and export the whole find so someone else can
tune it in seconds.

It is the detective arc. Where
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}) taught you to
*name the unknown* from a single capture on a workbench, The Hunt does it **live
and at band scale** — the same measurements, driving a search. One thread runs
the length of the series: a stray carrier that shows up in a routine survey, and
everything it takes to turn it into a named, mapped, exportable system.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with the
[Digital Trunking]({{ '/learn/digital-trunking/' | relative_url }}) module for
how a trunked system is laid out, then come back for how GopherTrunk finds one.

{%- assign parts = site.posts | where: "series", "The Hunt" | sort: "series_part" -%}
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
