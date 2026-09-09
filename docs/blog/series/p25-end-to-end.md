---
layout: page
title: "P25 End to End: from a C4FM carrier to recorded, named, multi-site voice"
description: A 14-part deep dive into GopherTrunk's P25 stack — C4FM and the NID, TSBKs and Multi-Block Trunking, band plans, the CQPSK/LSM linear path, Phase 2 TDMA, IMBE voice, encryption signalling, multi-site roaming, wideband decoding, and the weak-signal gap that still needs your captures.
keywords: p25 decoder, c4fm demodulation, p25 tsbk, multi-block trunking ambt, p25 phase 2 tdma, imbe vocoder, p25 band plan, cqpsk lsm simulcast, p25 trunking scanner, gophertrunk p25
nav_group: Blog
permalink: /blog/series/p25-end-to-end/
---

**P25 End to End** is a 14-part deep dive into the
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) P25 stack — the
protocol GopherTrunk locks more often than any other, followed from a raw
4800-symbol-per-second C4FM carrier all the way to recorded, named,
multi-site-tracked voice. Where
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
surveyed P25 across three episodes, this series walks every layer at full
depth: frame sync and the NID, TSBKs and the trellis, **Multi-Block Trunking**
(the PDU that used to be dropped as noise), channel identifiers and band
plans, the **CQPSK/LSM** linear path, Phase 2 TDMA, IMBE voice, encryption
signalling and policy, sites and roaming, and the wideband engine that watches
a whole system at once.

The running thread is that P25 in GopherTrunk is a family of **twin paths** —
Phase 1 and Phase 2, C4FM and CQPSK, single-channel and wideband, live and
replay — and every twin pair is a place where a fix can land on one side and
silently miss the other, the lesson the
[Two Pipelines postmortem]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }})
taught the hard way. The series ends honestly: Part 12 is about the
**weak-signal gap** in the Phase 1 C4FM voice path and the capture
contributions that would close it.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with the
[Digital Trunking]({{ '/learn/digital-trunking/' | relative_url }}) module for
how a trunked system is laid out, then come back for the P25 specifics.

{%- assign parts = site.posts | where: "series", "P25 End to End" | sort: "series_part" -%}
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
