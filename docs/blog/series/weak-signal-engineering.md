---
layout: page
title: "Weak-Signal Engineering: decoding the calls at the edge"
description: A 14-part deep dive into how GopherTrunk roughly doubled its decode yield on marginal signals — blind and trained equalization, frozen-snapshot taps ahead of differential decoders, soft-decision FEC, diversity combining with coherence-gated calibration, and the metrics that lie along the way.
keywords: weak signal decoding, blind equalizer cma, lms equalizer training sequence, soft decision viterbi llr, diversity combining mrc, coherence cross correlation, evm trap crc yield, differential decoder phase, isi multipath equalization, gophertrunk weak signal
nav_group: Blog
permalink: /blog/series/weak-signal-engineering/
---

**Weak-Signal Engineering** is a 14-part deep dive into the marginal regime —
the space between full quieting and no signal, where a receiver **locks but
doesn't decode**, and where most real traffic actually lives. Across one year of
issue-tracker work, [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
roughly **doubled** its decode yield there with four levers: blind equalization
(CMA with frozen snapshots), trained equalization (LMS on known midambles),
soft-decision FEC, and diversity combining. This series is the cross-protocol
theory and engineering of those levers — what each one can and cannot fix, the
traps between them, and the tests that keep them honest.

Where [SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) gave
equalization and diversity one survey episode, this series goes lever by lever.
The concurrent
[TETRA End to End]({{ '/blog/series/tetra-end-to-end/' | relative_url }}) series
is its flagship case study, and
[The Analog Edge]({{ '/blog/series/analog-edge/' | relative_url }}) covers the
other side of the ADC — because the first weak-signal question is always whether
the deficit is in the channel or baked into the samples. One discipline runs the
length of the series: **CRC-valid frames are the only trustworthy verdict**;
EVM, SNR and constellation beauty are advisory at best and traps at worst.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with the
[DSP]({{ '/learn/dsp/' | relative_url }}) learning module for the vocabulary,
then come back for the engineering.

{%- assign parts = site.posts | where: "series", "Weak-Signal Engineering" | sort: "series_part" -%}
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
