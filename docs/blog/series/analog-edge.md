---
layout: page
title: "The Analog Edge: everything between the antenna and the ADC"
description: A 14-part operator's field guide to the analog half of an SDR scanner — dBFS, gain staging, clipping and intermod, phase noise, sample-rate choice, antennas, feedline, filters and LNAs, capture discipline, and diversity — because the decoder can only be as good as the samples.
keywords: sdr gain staging, dbfs meaning, sdr overload intermod, phase noise reciprocal mixing, scanner antenna guide, coax loss 800 mhz, sdr lna filter, iq capture sigmf, antenna diversity mrc, gophertrunk analog edge
nav_group: Blog
permalink: /blog/series/analog-edge/
---

**The Analog Edge** is a 14-part field guide to the half of a scanner that
software can't fix: everything between the antenna and the ADC. The
[issue tracker]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
keeps teaching the same lesson — half the hard bugs were **in the samples**, not
in the code — and this series turns those postmortems into working knowledge:
what dBFS actually measures, how to stage gain without chasing a software
threshold, what clipping, intermod and phase noise each look like from the
decoder's side, which sample rate to run, and how antennas, feedline, filters
and LNAs decide your noise floor before
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) sees a single sample.

It is the operator's companion to two concurrent series: where
[Weak-Signal Engineering]({{ '/blog/series/weak-signal-engineering/' | relative_url }})
explains the algorithms that recover a marginal signal, The Analog Edge is about
**not being marginal in the first place** — and about capturing honest evidence
(pre-combine IQ, sidecar metadata) when you are. One thread runs the length of
the series: a system that decodes cleanly on a hardware scanner but garbles in
software, and the analog-side causes eliminated one part at a time.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
measurements for the deep read.

New here? Start with
[What hardware do I need?]({{ '/what-do-i-need-for-gophertrunk/' | relative_url }})
for the shopping list, then come back for how to make it sing.

{%- assign parts = site.posts | where: "series", "The Analog Edge" | sort: "series_part" -%}
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
