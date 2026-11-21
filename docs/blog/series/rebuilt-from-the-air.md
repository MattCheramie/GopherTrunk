---
layout: page
title: "Rebuilt From the Air: the decoders that passed every test and never heard a radio"
description: A 14-part postmortem series on the GopherTrunk decoders whose framing was invented — MDC1200, CTCSS and DCS, FLEX, LoRa, System Fusion, NXDN, D-STAR and P25 Phase 2 — each diagnosed from a reference implementation or a reporter's capture, rebuilt against independent decoders, and pinned to real air where a capture exists. Ends with the audit checklist and the errata for this blog.
keywords: sdr decoder postmortem, self-consistent test trap, mdc1200 decoder, flex pager decoder, lora bit layer, ysf fich, nxdn decoder, d-star decoder, p25 phase 2 tdma, reference implementation testing, gophertrunk rebuilt from the air
nav_group: Blog
permalink: /blog/series/rebuilt-from-the-air/
---

**Rebuilt From the Air** is a 14-part postmortem series about the
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) decoders that passed
every test they had and never matched a real radio. In each one the encoder
in the test suite and the decoder under test were written from the same
layout — a sync word, a bit order, an interleaver, a line code — and the
layout was wrong. Round trips stayed green for months. The first real
capture decoded nothing.

[From the Issue Tracker]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
named that failure the
[self-consistent trap]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}),
and [From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }})
told the first full rebuild, SmartNet. This series is everything that came
after: **MDC1200**, whose tones were sliced as the data; the **CTCSS and DCS**
gates; **FLEX**, **LoRa** and **System Fusion**; three parts on **NXDN** —
the frame, the control channel on real air, and the voice scrambler;
**D-STAR**; and two parts on **P25 Phase 2**. Each part gives the symptom as
it was reported, the theories that were wrong, the reference or capture that
settled it, and the test that now fails without the fix.

It is also a correction. Several earlier posts on this blog described the
formats these decoders used to implement. Part 1 lists them, each part says
what it overturns, and the finale is the errata — together with the
checklist for telling a real decoder from an invented one, and a ledger of
what is on-air verified, what is capture-verified, and what still waits for
a recording.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers and tables** for the medium read, and the full investigation
narrative for the deep read.

New here? Start with
[Troubleshooting a decode]({{ '/learn/digital-trunking/troubleshooting-a-decode/' | relative_url }})
in the Digital Trunking module, then come back for the rebuilds.

{%- assign parts = site.posts | where: "series", "Rebuilt From the Air" | sort: "series_part" -%}
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
