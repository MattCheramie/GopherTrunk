---
layout: page
title: "Signal Lab: from first capture to lab-grade VSA"
description: A 10-part hands-on series on GopherTrunk's SigLab — the offline signal-analysis workbench. Replay a capture with no radio attached, read the decode dashboard, visualize constellations and eye diagrams, synthesize impaired references, measure VSA modulation quality, identify unknown signals, dissect P25 PDUs, and benchmark demodulators.
keywords: siglab, gophertrunk signal lab, offline iq analysis, replay iq capture, constellation diagram, eye diagram, evm measurement, vsa, p25 decode, sdr signal analysis, demodulator benchmark
nav_group: Blog
permalink: /blog/series/signal-lab/
---

**Signal Lab** is a 10-part tutorial series on
[SigLab]({{ '/siglab.html' | relative_url }}), GopherTrunk's standalone
signal-analysis workbench — the tool that runs **entirely offline against a
recorded IQ capture**, no SDR and no daemon required. We start from your very
first replay and climb, one post at a time, to lab-grade modulation
measurement, blind signal identification, P25 PDU dissection, and demodulator
regression benchmarking.

Every post is built for three ways of reading: a **TL;DR + cheat-sheet** up top
for skimmers, **bold section headers, tables, and diagrams** for the medium
read, and full prose with real command output for the deep read. You'll meet
**Ada**, a brand-new operator working her first capture, and **Reese**, the RF
veteran who explains *why* each number matters.

This is one leg of the **Lab Bench trilogy** — three concurrent series on
GopherTrunk's analysis consoles:
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}) (this one),
[RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}), and
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}). A single
mystery signal — *Mercury* — runs through all three.

New to radio first? Start with the
[Learn RF &amp; SDR]({{ '/learn/rf-sdr/' | relative_url }}) path, then come back.

{%- assign parts = site.posts | where: "series", "Signal Lab" | sort: "series_part" -%}
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
