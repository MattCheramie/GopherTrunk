---
layout: page
title: "Squelch: the conventional scanner, one gate at a time"
description: A 14-part deep dive into GopherTrunk's conventional scanner — the dwell loop with no control channel, channel power, noise squelch, offset tuning, CTCSS and DCS, AM carrier-to-noise, the call gates for P25, DMR, System Fusion, NXDN and D-STAR, bursty carriers, data decoders on the scan list, the shared narrowband front end, and the operator controls on top.
keywords: sdr squelch, noise squelch fm, ctcss decoder, dcs decoder, am airband squelch, conventional scanner sdr, scan list dmr nxdn d-star, channel power squelch, offset tuning rtl-sdr, gophertrunk squelch
nav_group: Blog
permalink: /blog/series/squelch/
---

**Squelch** is a 14-part deep dive into the conventional scanner in
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) — the scan list.
A trunked system announces every call on its control channel; a scan list
has nothing of the kind. It has to decide from the air alone whether a
channel carries a call, and it has to decide again for every channel on
every pass.

The series takes that decision apart one gate at a time. It starts with the
dwell loop, then the squelch that turned out to measure the whole SDR span
instead of the channel, the **noise squelch** that works when an AGC makes a
level meaningless, and **offset tuning**. Then the tone gates — **CTCSS**
tones 2.3 Hz apart and the real **DCS** codeword — and **AM**, which
squelches on carrier-to-noise. Then the digital channels, where a carrier is
not a call: the gates for **P25, DMR, System Fusion, NXDN and D-STAR**.
The last parts cover carriers that come in bursts, data decoders riding scan
list channels, the one front end every narrowband decoder now shares, and
the controls an operator turns.

The running thread comes from
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}): never
gate a decision on absolute dBFS when a scale-invariant statistic exists.
Almost every gate in this series began as a level threshold and was replaced
by a measurement that reads the same at any gain. The
[Cookbook]({{ '/blog/series/operator-cookbook/' | relative_url }}) gave the
scan list one recipe and
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
one episode; this series is the full depth, with the measurements behind
each default and an honest list of what has not yet been run on air.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with
[Talkgroups and scan lists]({{ '/learn/scanning/talkgroups-and-scan-lists/' | relative_url }})
in the Scanning module for the lay of the land, then come back for how the
gates work.

{%- assign parts = site.posts | where: "series", "Squelch" | sort: "series_part" -%}
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
