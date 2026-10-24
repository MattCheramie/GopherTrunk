---
layout: page
title: "The Legacy Family End to End: SmartNet, EDACS, LTR, MPT 1327, NXDN, dPMR, D-STAR & YSF"
description: A 14-part deep dive into the protocols the P25, DMR and TETRA series left out — the FM-era trunking generation (Motorola Type II, EDACS, LTR, MPT 1327) and the AMBE-era narrowband and amateur modes (NXDN, dPMR Mode 3, D-STAR, System Fusion) — with each decoder's bit layouts, FEC, band plans and voice path, and an honest verification ladder showing which captures would move each one up.
keywords: motorola type ii smartnet decoder, edacs control channel decode, ltr trunking sdr, mpt 1327 decoder, nxdn decoder sdr, dpmr mode 3, d-star decoder, yaesu system fusion fich, legacy trunking protocols, gophertrunk legacy family
nav_group: Blog
permalink: /blog/series/legacy-family-end-to-end/
---

**The Legacy Family End to End** is the fourth End-to-End series on this
blog. [P25]({{ '/blog/series/p25-end-to-end/' | relative_url }}),
[DMR]({{ '/blog/series/dmr-end-to-end/' | relative_url }}) and
[TETRA]({{ '/blog/series/tetra-end-to-end/' | relative_url }}) each
followed one protocol from carrier to recorded voice. This one follows eight
at once — the pre-digital trunking generation (Motorola Type II / SmartNet,
EDACS, LTR, MPT 1327) and the AMBE-era narrowband and amateur modes (NXDN,
dPMR Mode 3, D-STAR, Yaesu System Fusion) — through the same layers: the
air interface, the control words and their FEC, the band plan that turns a
channel number into hertz, the state machine that turns a word into a grant,
and the voice path behind it.

The difference from the earlier series is the verification ladder.
[From Spec to Shipping]({{ '/blog/series/from-spec-to-shipping/' | relative_url }})
drew the line between a green synthetic test and an on-air pass; most of
this family sits below that line, pinned to published references and
independent decoders rather than to operator captures. Every part names the
rung each protocol stands on — on-air verified, capture-pinned,
reference-pinned, or placeholder — and the capture that would move it up.
The series ends with that ladder as one table and the recipe for
contributing a capture.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with
real code excerpts for the deep read. The overviews in
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
are the lighter introduction; this series goes a layer deeper into each
package.

{%- assign parts = site.posts | where: "series", "The Legacy Family End to End" | sort: "series_part" -%}
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
