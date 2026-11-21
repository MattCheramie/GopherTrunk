---
layout: page
title: "The Operator's Cookbook, Volume 2: fourteen more GopherTrunk builds"
description: A second 14-part series of copy-paste GopherTrunk recipes — an airband rig with ACARS, a paging rig, an NXDN system, fleet signalling, tuning the sound, alerts, Whisper transcription, decryption with your own keys, hold and avoid, a phone rig in Termux, a band survey, a Wireless M-Bus meter walk, the native Pluto driver, and a Raspberry Pi 4 sized honestly.
keywords: gophertrunk setup guide, airband scanner sdr acars, pocsag flex pager rtl-sdr, nxdn scanner config, scanner alerts mqtt ntfy, police scanner transcription whisper, rtl-sdr android termux, rtl_power band survey, wmbus rtl-sdr water meter, plutosdr scanner, gophertrunk cookbook
nav_group: Blog
permalink: /blog/series/operator-cookbook-volume-2/
---

**The Operator's Cookbook, Volume 2** is a second 14-part run of complete,
copy-paste [GopherTrunk](https://github.com/MattCheramie/GopherTrunk) builds —
one working rig per part. It keeps the shape of
[Volume 1]({{ '/blog/series/operator-cookbook/' | relative_url }}): the
hardware list, a working config or command line, the first-run check (the
exact log lines that mean it is working), a troubleshooting table, and
variations to grow into.

Every recipe here is built on something that landed after Volume 1 ended.
The first half is about what you can listen to: an **airband rig** with AM
voice and ACARS, a **paging rig** for POCSAG and FLEX, an **NXDN** system
both trunked and conventional, and **fleet signalling** with MDC1200,
FleetSync and tone gates. The middle is about what you do with the calls:
**tuning the sound**, **alerts** to a phone or a home-automation hub,
**transcription** with Whisper, **decryption with keys you already hold**,
and scanner-style **hold, avoid and priority**. The last part leaves the
desk: a **phone rig** in Termux, a **band survey** with `gophertrunk power`,
a **Wireless M-Bus meter walk**, the **native Pluto driver**, and a
Raspberry Pi 4 sized against real measurements — ending, like Volume 1, with
the kitchen-sink config.

The cookbook tells you **what** to do. For **why** it works, each recipe
links the deep dive behind it:
[Squelch]({{ '/blog/series/squelch/' | relative_url }}) for the scan list's
gates and
[Rebuilt From the Air]({{ '/blog/series/rebuilt-from-the-air/' | relative_url }})
for the decoders that were rebuilt to make several of these rigs possible.
Each recipe also says plainly how far it has been verified — on air by a
reporter, on a capture, or only against synthetic signals.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and the full recipe for
the build.

New here? Start with
[GopherTrunk as a scanner]({{ '/learn/scanning/gophertrunk-as-a-scanner/' | relative_url }})
in the Scanning module, then
[Volume 1, Part 1]({{ '/blog/tutorials/operator-cookbook-01-forty-dollar-p25-rig/' | relative_url }})
for a first rig.

{%- assign parts = site.posts | where: "series", "The Operator's Cookbook, Volume 2" | sort: "series_part" -%}
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
