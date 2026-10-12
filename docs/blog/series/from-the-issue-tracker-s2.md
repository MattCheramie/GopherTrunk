---
layout: page
title: "From the Issue Tracker, Season 2: fourteen more bugs that fought back"
description: A 14-part postmortem series mined from GopherTrunk's September and October 2026 issues — MDC1200's precoded line code, zero-IF clipping tones, a CTCSS gate calibrated at the wrong sample rate, the TETRA DMO scramble seed solved in GF(2), spliced neighbour broadcasts, a FLAC sample-rate table, a scanner squelched by the whole span, and a P25 opcode parsed with the wrong layout.
keywords: sdr debugging postmortem, mdc1200 decode bug, ctcss sample rate, dcs codeword, tetra dmo scramble seed, flac sample rate 880029, tetra neighbour cells, cgo static binary termux, conventional scanner squelch, p25 opcode 0x03, gophertrunk issues
nav_group: Blog
permalink: /blog/series/from-the-issue-tracker-s2/
---

**From the Issue Tracker, Season 2** picks up where
[Season 1]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
left off. Season 1 closed with three meta-lessons — the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}),
[census everything]({{ '/blog/solution-postmortem/from-the-issue-tracker-21-census-everything/' | relative_url }})
and
[two pipelines, one symptom]({{ '/blog/solution-postmortem/from-the-issue-tracker-22-two-pipelines/' | relative_url }}) —
and September's reports promptly produced fourteen new bugs that fit those
lessons exactly. Every post is again a true story with receipts: the symptom
as the operator reported it, the plausible explanations that were wrong, the
instrument that settled it, the fix, and the regression that fails without it.

This season leans toward the conventional and analog side of the project —
a line code that was never decoded, a tone at four times the carrier offset,
a detector threshold in the wrong units, an invented DCS codeword, a scanner
gated by every carrier in its span — and toward the TETRA Direct Mode seed
solve that ended months of "colour code" guessing. It closes on a static
binary that was not static and a P25 opcode that shared a layout it never had.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers and tables** for the medium read, and the full investigation narrative
for the deep read. The companion operator tutorial is
[The Conventional Scanner]({{ '/blog/series/conventional-scanner/' | relative_url }}),
which explains the subsystem most of these bugs lived in.

{%- assign parts = site.posts | where: "series", "From the Issue Tracker, Season 2" | sort: "series_part" -%}
{%- if parts and parts.size > 0 -%}
<ol class="post-list series-list">
  {%- for post in parts -%}
    <li class="post-card">
      <a class="post-card__link" href="{{ post.url | relative_url }}">
        <h2 class="post-card__title">{{ post.title }}</h2>
      </a>
      <p class="post-card__meta">
        <time datetime="{{ post.date | date_to_xmlschema }}">{{ post.date | date: "%B %-d, %Y" }}</time>
        <span class="category-chip">Solution Postmortem</span>
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
  See all <a href="{{ '/blog/category/solution-postmortem/' | relative_url }}">solution postmortems</a>
  or subscribe via <a href="{{ '/feed.xml' | relative_url }}">RSS</a>.
</p>
