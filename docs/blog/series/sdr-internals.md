---
layout: page
title: "SDR Internals: building a software-defined radio in pure Go"
description: A 14-part deep-dive series walking the entire GopherTrunk software-defined-radio pipeline — RF to audio — one component per post, with the Go implementation and the software-design principle behind each piece.
keywords: software defined radio, SDR in Go, pure Go DSP, P25 DMR decoder, IQ data, demodulation, FEC, vocoder, trunking, GopherTrunk internals
nav_group: Blog
permalink: /blog/series/sdr-internals/
---

**SDR Internals** is a 14-part series that walks the complete software-defined
radio pipeline behind [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
— from raw RF samples all the way to recorded audio — **one component per post**.
Each part explains *what* the component is, *how* it's implemented in pure Go
(`CGO_ENABLED=0`, single static binary), the **software-design principle** behind
it, and how that principle shaped the code. Together they form an overview of the
whole engine; each is also a doorway to a deeper, per-component series.

New to radio first? Start with the
[Learn RF &amp; SDR]({{ '/learn/rf-sdr/' | relative_url }}) path, then come back here for
the implementation.

{%- assign parts = site.posts | where: "series", "SDR Internals" | sort: "series_part" -%}
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
