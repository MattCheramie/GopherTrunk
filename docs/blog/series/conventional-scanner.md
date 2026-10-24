---
layout: page
title: "The Conventional Scanner: analog FM, air-band AM and in-band data, one dwell at a time"
description: A 14-part operator's tutorial on GopherTrunk's conventional (non-trunked) scanner — the dwell loop, the scan list as config, offset tuning, in-channel squelch, CTCSS and DCS done right, the FM voice chain, AM for the air band with a carrier tracker, ACARS, MDC1200 and FleetSync on scan-list channels, priority interleave, persistent lockouts, the scanner's log lines, and a fully annotated config.
keywords: sdr conventional scanner, rtl-sdr analog fm scanner, ctcss dcs squelch sdr, air band am sdr, acars decoder sdr, mdc1200 fleetsync scanner, scanner priority scan lockout, gophertrunk conventional scanner config
nav_group: Blog
permalink: /blog/series/conventional-scanner/
---

**The Conventional Scanner** is a 14-part operator's tutorial on the half of
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) that has no
control channel: `scanner.conventional`. The
[Cookbook]({{ '/blog/series/operator-cookbook/' | relative_url }}) gave
analog FM one recipe;
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }})
explained the samples; [Beyond Voice]({{ '/blog/series/beyond-voice/' | relative_url }})
decoded the bursts. This series is the manual for the dwell between them:
what each scan-list key configures, what the code does with it, what the log
prints, and what the September 2026 field reports changed.

That month rebuilt most of this subsystem against real radios — a Kenwood
lab, an aircraft radio, a hangar full of hum. The LO now sits off the channel
so a clipped front end's products miss the audio; squelch measures the
channel, not the whole SDR span; CTCSS uses exact Goertzel bins and DCS the
on-air bit order; AM got envelope detection, a carrier-to-noise squelch and a
carrier tracker; and ACARS, MDC1200 and FleetSync ride the scanner's own
channels instead of a dedicated dongle. Each part says which of those is
on-air verified and which still waits on a capture.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
config and log excerpts for the deep read. The postmortems behind several of
these parts are in
[From the Issue Tracker, Season 2]({{ '/blog/series/from-the-issue-tracker-s2/' | relative_url }}).

{%- assign parts = site.posts | where: "series", "The Conventional Scanner" | sort: "series_part" -%}
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
