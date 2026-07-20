---
layout: page
title: "Voice Coding: turning digital voice frames into audio, in pure Go"
description: A 12-part deep dive into GopherTrunk's vocoders — the pure-Go IMBE and AMBE+2 decoders, their shared MBE synthesis core, the per-mode composer chain, the call-startup acquisition squelch, enhancement and loudness, recording, and calibration.
keywords: vocoder, imbe decoder, ambe+2 decoder, mbe synthesis, p25 voice, dmr voice, digital voice pure go, acquisition squelch, loudness normalization, voice calibration, GopherTrunk
nav_group: Blog
permalink: /blog/series/voice-coding/
---

**Voice Coding** is a 12-part deep dive into how
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) turns a few kilobits
per second of digital voice frames back into audio — in **pure Go**, with no DSP
libraries and no CGO. Digital voice isn't recorded sound; it's a *model* of a
voice. This series walks the two vocoders GopherTrunk implements — **IMBE** (P25
Phase 1) and **AMBE+2** (P25 Phase 2, DMR, NXDN) — their shared **MBE synthesis
core**, the per-mode **composer** that wires frames to the right decoder, and
everything after: enhancement, loudness, recording, encoding, and calibration.

[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) gave the
vocoders one episode; here each gets the full treatment, including a **"problem
we hit"** post on the v0.7.1 **call-startup acquisition squelch** — why fresh P25
recordings opened with a full-scale burst of noise, and the speech-signature
heuristic that fixed it.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New to digital voice first? See the
[digital voice]({{ '/learn/rf-sdr/digital-voice/' | relative_url }}) lesson, then
come back for the implementation.

{%- assign parts = site.posts | where: "series", "Voice Coding" | sort: "series_part" -%}
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
