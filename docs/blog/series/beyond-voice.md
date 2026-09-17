---
layout: page
title: "Beyond Voice: everything else GopherTrunk decodes on the band"
description: A 14-part deep dive into GopherTrunk's non-voice decoders — AFSK/FFSK signalling (MDC1200, FleetSync), two-tone and POCSAG/FLEX paging, APRS and the location layer, ADS-B, AIS, DSC, LoRa, M17 and the D-STAR/YSF/dPMR family — and the one eleven-place wiring pattern that carries each from a burst on the air to a row in the web console.
keywords: sdr data decoder, mdc1200 decoder, fleetsync decoder, pocsag flex decoder, aprs ax.25 decode, ads-b mode s decoder, ais gmsk decoder, dsc decoder, lora sdr receiver, m17 decoder, gophertrunk beyond voice
nav_group: Blog
permalink: /blog/series/beyond-voice/
---

**Beyond Voice** is a 14-part deep dive into everything
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) decodes that is
*not* a trunked voice call. Every earlier series on this blog followed a voice
call from carrier to WAV; this one follows the rest of the band — the
**AFSK/FFSK signalling** that rides under analog fleets (MDC1200, Kenwood
FleetSync), **two-tone** fire paging, **POCSAG and FLEX** pager networks,
**APRS** and the location layer that feeds the map, **ADS-B** aircraft
squitters, **AIS** ship reports, marine **DSC**, a pure-Go **LoRa** chirp
receiver, the open **M17** link layer, and the amateur digital-voice family
(**D-STAR, System Fusion, dPMR**).

The running thread is a single shape. Each decoder is the same eleven-place
pattern — front end, framer, bus event, storage table, REST route, web panel,
config builder, field help, doctor preflight, `config.example.yaml`, and the
tests that police them — so the series teaches the pattern once (with the
FleetSync landing as the worked example) and then walks it through twelve
different physical layers. Along the way it collects the traps that cost real
rounds: a WAV reader that assumed 16-bit PCM, a carrier estimator that only
looked at the first few milliseconds, and a slicer threshold that drifted
toward a run of space tone.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with
[Other signals]({{ '/learn/rf-sdr/other-signals/' | relative_url }}) in the
RF & SDR module for what else lives on the band, then come back for how each
one is decoded.

{%- assign parts = site.posts | where: "series", "Beyond Voice" | sort: "series_part" -%}
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
