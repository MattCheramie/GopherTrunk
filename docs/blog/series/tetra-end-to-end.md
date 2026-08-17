---
layout: page
title: "TETRA End to End: from a π/4-DQPSK carrier to clear recorded voice"
description: A 14-part deep dive into GopherTrunk's TETRA stack — bursts and slot grids, RCPC channel coding, scrambling and colour codes, a clean-room ACELP vocoder conformed bit-identical against the ETSI reference, soft-decision TCH/S, the equalizers, and the full Direct Mode (DMO) saga.
keywords: tetra decoder, pi/4-dqpsk, tetra burst slot grid, rcpc viterbi, tetra scrambling colour code, acelp vocoder, etsi en 300 395-2, tetra dmo direct mode, soft decision tch/s, gophertrunk tetra
nav_group: Blog
permalink: /blog/series/tetra-end-to-end/
---

**TETRA End to End** is a 14-part deep dive into the
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) TETRA stack — the
whole path from a raw 25 kHz π/4-DQPSK carrier to clear voice in a WAV file.
Where [Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
gave TETRA one survey episode, this series walks every layer at full depth:
bursts and the slot grid, RCPC channel coding and the CRC that isn't an LFSR,
scrambling and colour codes, the traffic channel, a **clean-room ACELP vocoder**
conformed bit-identical against the ETSI reference codec, the soft-decision and
equalizer upgrades that roughly doubled marginal-signal yield, and the
three-part **Direct Mode (DMO)** saga — including the "encrypted" verdict that
turned out to be a descramble bug.

It is also a series about verification discipline. The recurring villain is the
self-consistent synthetic test — the round-trip that validates its own bug — and
the recurring hero is the reporter's capture. Where
[Weak-Signal Engineering]({{ '/blog/series/weak-signal-engineering/' | relative_url }})
owns the cross-protocol theory of equalizers, soft decisions and diversity, this
series is its flagship case study, protocol by protocol layer.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with the
[Digital Trunking]({{ '/learn/digital-trunking/' | relative_url }}) module for
how a trunked system is laid out, then come back for the TETRA specifics.

{%- assign parts = site.posts | where: "series", "TETRA End to End" | sort: "series_part" -%}
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
