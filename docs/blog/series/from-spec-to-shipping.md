---
layout: page
title: "From Spec to Shipping: how a protocol decoder actually gets written"
description: A 14-part series on turning standards documents and independent references into decoder code you can trust on air — reading ETSI and TIA specs, literal test vectors over round-trips, conformance harnesses, clean-room rules, the on-air verification gate, capture-driven development, and what "verified" really means.
keywords: how to write a protocol decoder, reading etsi standards, tia-102 p25 spec, reference implementation testing, conformance test harness, clean room implementation, regression test failing first, capture driven development, software verification discipline, gophertrunk engineering
nav_group: Blog
permalink: /blog/series/from-spec-to-shipping/
---

**From Spec to Shipping** is a 14-part series on how a protocol decoder
actually gets written — the method
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) follows to go from
an ETSI or TIA PDF to code that decodes real signals on air.
[From the Issue Tracker]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
taught these lessons as postmortems: the round-trip test that validates its own
bug, the placeholder constant that became a fabricated protocol, the RPC opcode
nothing ever checked. This series teaches the same discipline **forward** —
reading standards, choosing reference implementations you can trust, pinning
parsers with literal byte vectors, building bit-identical conformance
harnesses, staying clean-room, letting operator captures referee when
references disagree, and gating every claim behind on-air verification.

The recurring villain is the test that passes because both sides share the same
assumption; the recurring hero is the **independent reference** — another
implementation, a reference codec, a reporter's capture. Every part closes with
rules that transfer to any wire format, not just radio.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? The [Testing]({{ '/learn/testing/' | relative_url }}) module covers
the fundamentals this series builds on.

{%- assign parts = site.posts | where: "series", "From Spec to Shipping" | sort: "series_part" -%}
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
