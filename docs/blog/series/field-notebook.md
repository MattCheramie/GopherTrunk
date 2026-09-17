---
layout: page
title: "The Field Notebook: reading GopherTrunk's log, one line at a time"
description: A 14-part operator's tutorial on GopherTrunk's debug.log — startup and lock lines, the TETRA and DMR decode-status lines, the carrier-offset WARN, DMO counters, overruns, the MRC health line, capture and recorder lines, voice-chain teardown reasons, web symptoms that are really log stories, and how to turn a log into a bug report that gets fixed.
keywords: gophertrunk log, sdr scanner troubleshooting, read debug log, decode status line, host_drops overruns, mrc coherence log, tetra decode status, dmr sync_hits, scanner not decoding, gophertrunk field notebook
nav_group: Blog
permalink: /blog/series/field-notebook/
---

**The Field Notebook** is a 14-part operator's tutorial that reads the
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) `debug.log` one
line family at a time. The
[Cookbook]({{ '/blog/series/operator-cookbook/' | relative_url }}) told you
which lines prove a rig is alive;
[From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }})
argued that the log is an **instrument**, not a diary. This series is the
instrument's manual: for each line — startup WARNs, lock and hunt lines, the
TETRA and DMR decode-status counters, the carrier-offset WARN, the DMO noise
meter versus its traffic counter, overruns and `host_drops`, the MRC health
line, capture and recorder lines, the voice chain's teardown reasons — it
answers the same three questions. **What does each field measure? What does a
healthy rig print? Which deep dive do you open when it doesn't?**

The running thread is that almost every field report in the
[issue tracker]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
already contained its own diagnosis in a line nobody read: a counter that was
a noise meter mistaken for traffic, a success-only line that carried no
information, a WARN firing on one sample of a ten-second condition. The
series ends with the one-page field card and with how to turn a log into a
bug report that gets fixed — what to paste, which capture to make, and what
"verified" means before anyone closes the issue.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
log excerpts for the deep read.

New here? Start with
[When decoding fails]({{ '/learn/scanning/when-decoding-fails/' | relative_url }})
in the Scanning module for the big picture, then come back for the lines.

{%- assign parts = site.posts | where: "series", "The Field Notebook" | sort: "series_part" -%}
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
