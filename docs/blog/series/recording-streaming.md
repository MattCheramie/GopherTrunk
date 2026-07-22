---
layout: page
title: "Recording, Composition & Streaming: turning decoded calls into files, feeds & uploads"
description: A 14-part deep dive into GopherTrunk's output half — how a decoded call becomes a crash-safe WAV, a row in the call log, a live browser stream, and an upload to Broadcastify, RdioScanner, OpenMHz and Icecast, all from optional, event-driven, config-gated subsystems in pure Go.
keywords: sdr call recording, wav recorder, broadcastify calls upload, rdioscanner openmhz, icecast streaming, call log sqlite, recording retention, live audio stream, event bus subscriber, gophertrunk recording composition streaming
nav_group: Blog
permalink: /blog/series/recording-streaming/
---

**Recording, Composition & Streaming** is a 14-part deep dive into the *output
half* of [GopherTrunk](https://github.com/MattCheramie/GopherTrunk) — everything
that happens after the vocoders produce PCM. A decoded call has to become a
**crash-safe WAV** on disk, a **row in the call log**, a **live stream** a
browser can play, and an **upload** to the public call aggregators — and each of
those is an independent, optional subsystem that reacts to a handful of bus
events instead of calling its neighbours.

Where [SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) gave
this whole area one survey episode, and
[Voice Coding]({{ '/blog/series/voice-coding/' | relative_url }}) covered the
*codec* that produces the PCM, this series takes the **subsystem/plumbing** view:
the event contracts, the session lifecycle, file-integrity guarantees, metadata
and persistence, retention, the live-listen path, the fan-out manager, and the
real API quirks of each aggregator backend. One thread runs the length of the
series — the life of a single call, from the instant its PCM exists to the moment
it lands in a Broadcastify feed.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? Start with the
[antenna-to-audio]({{ '/learn/rf-sdr/antenna-to-audio/' | relative_url }}) lesson
for the last mile, then come back for the implementation.

{%- assign parts = site.posts | where: "series", "Recording, Composition & Streaming" | sort: "series_part" -%}
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
