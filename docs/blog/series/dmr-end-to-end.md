---
layout: page
title: "DMR End to End: from a 4FSK carrier to two calls, direct mode & decrypted voice"
description: A 14-part deep dive into GopherTrunk's DMR stack — 4FSK and sync polarity, the two-slot TDMA cadences, BPTC/Golay/RS FEC and the forged terminator, link control and late entry, conventional IPSC as two calls, the idle beacon, Tier III trunking, direct-mode carrier gating, wideband bin edges, AMBE+2 silence frames, Enhanced Privacy RC4, and the testing that holds it together.
keywords: dmr decoder, dmr sdr scanner, dmr tier 2 tier 3, dmr direct mode simplex decode, dmr bptc golay, dmr embedded lc late entry, dmr ipsc two slots, dmr enhanced privacy rc4, ambe+2 vocoder, gophertrunk dmr
nav_group: Blog
permalink: /blog/series/dmr-end-to-end/
---

**DMR End to End** is a 14-part deep dive into the
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) DMR stack — the
world's most widely deployed digital PMR protocol, followed from a raw
4800-symbol-per-second 4FSK carrier all the way to two simultaneous recorded
calls, direct-mode handhelds and decrypted Enhanced Privacy voice. Where
[Protocol Decoders]({{ '/blog/series/protocol-decoders/' | relative_url }})
gave DMR one episode and the
[Cookbook]({{ '/blog/series/operator-cookbook/' | relative_url }}) two recipes,
this series walks every layer at full depth: bursts, sync words and polarity,
the two TDMA cadences, the **BPTC/Golay/Reed-Solomon** FEC stack, link control
and **late entry**, conventional IPSC repeaters decoded as two calls, the idle
beacon, Tier III trunking, **direct mode** and the gaps that blinded the
receiver, wideband decoding and the deaf-heal, AMBE+2 voice, **Enhanced
Privacy**, and the testing discipline behind all of it.

The running thread is that one DMR carrier carries **two of everything** —
two timeslots, two sync polarities, two tiers of trunking, two decode
cadences — and the receiver only works when it knows which one it is looking
at. Almost every DMR bug in the
[issue tracker]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
was the decoder assuming the wrong twin: a repeater's back-to-back bursts
where a handheld leaves gaps, a voice burst read as a data slot type, two
talkgroups folded into one call. The series ends honestly, with the field
runs and captures still open.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with the
[DMR Tier 2/3 lesson]({{ '/learn/digital-trunking/dmr-tier-2-3/' | relative_url }})
in the Digital Trunking module for the lay of the land, then come back for how
GopherTrunk actually decodes it.

{%- assign parts = site.posts | where: "series", "DMR End to End" | sort: "series_part" -%}
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
