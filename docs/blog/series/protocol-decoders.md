---
layout: page
title: "Protocol Decoders: control channels, state machines & the alias hunt"
description: A 12-part deep dive into GopherTrunk's protocol decoders — how P25 Phase 1/2, DMR, NXDN, TETRA, and the EDACS/LTR/MPT-1327 legacy family turn symbols into grants, culminating in a clean-room cryptanalysis of the unsolved Motorola talker-alias cipher.
keywords: p25 decoder, dmr decoder, tetra decoder, nxdn, edacs ltr mpt1327, control channel decode, tsbk, p25 phase 2 mac, talker alias cryptanalysis, motorola alias cipher, GopherTrunk
nav_group: Blog
permalink: /blog/series/protocol-decoders/
---

**Protocol Decoders** is a 12-part deep dive into how
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) turns a demodulated
symbol stream into structured **control-channel messages** — the grants that the
[Trunking Engine]({{ '/blog/series/trunking-engine/' | relative_url }}) acts on.
Where [SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) gave the
decoders one survey episode, this series walks each standard in turn: **P25 Phase
1 and Phase 2, DMR Tier 2/3, NXDN, dPMR, TETRA**, and the **EDACS / LTR /
MPT-1327** legacy family — framing, FEC, PDUs, opcodes, and the state machines
that hold it together.

A single mystery runs through the whole series: an intercepted Motorola
**talker-alias** field that decodes to garbage. We plant it in Part 1 and pay it
off in Parts 10–11 with a **clean-room cryptanalysis** (issue #773) — the
recovered substitution table, the affine keystream model, the ruled-out dead
ends, and the honest verdict: **not yet cracked.** It's the same emitter the
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) series calls
*Mercury*.

> **Authorized use only.** The alias-hunt posts are for security researchers,
> licensed operators, and educational/CTF use. GopherTrunk's alias decode stays
> gated (`CipherVerified = false`) and clean-room (Apache-2.0, no GPL source
> read) until a real alias decodes end-to-end.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

{%- assign parts = site.posts | where: "series", "Protocol Decoders" | sort: "series_part" -%}
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
