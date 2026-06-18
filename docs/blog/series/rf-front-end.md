---
layout: page
title: "RF Front End: driving RTL-SDR, Airspy & HackRF in pure Go"
description: A 14-part deep-dive series on GopherTrunk's RF source layer — how RTL-SDR, Airspy, and HackRF dongles are driven in pure Go with no libusb, the USB transport, per-radio register bring-up, IQ conversion, and the real bugs that were hit and fixed along the way.
keywords: rtl-sdr driver, airspy driver, hackrf driver, pure go usb, libusb alternative, rtl2832u, r820t tuner, iq streaming, sdr device driver, GopherTrunk
nav_group: Blog
permalink: /blog/series/rf-front-end/
---

**RF Front End** is a 14-part series that zooms all the way in on the layer the
[SDR Internals]({{ '/blog/series/sdr-internals/' | relative_url }}) series only
skimmed: the **RF source**. It is the story of how
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) turns three families
of cheap USB dongles — **RTL-SDR, Airspy (R2/Mini and HF+), and HackRF** — into a
clean stream of IQ samples, in **pure Go**, with **no `libusb`, no `librtlsdr`,
no CGO at all**.

Where SDR Internals asked *what is the pipeline?*, this series asks *how do you
actually talk to the metal?* — the `Device` contract, the self-registering
driver registry, hand-rolled USB transport on three operating systems, the
RTL2832U register dance, the R820T tuner, real-to-complex sample conversion, and
the hotplug-aware device pool. And because driver work is where theory meets
hardware reality, **every part carries a "problem we hit" section**: a real bug
(RTL-SDR Blog V4 deafness, GC-churn IQ loss, a silent USB reaper deadlock) and
the Go code that fixed it.

New to radio first? Start with the
[Learn RF &amp; SDR]({{ '/learn/rf-sdr/' | relative_url }}) path, then come back here for
the implementation.

{%- assign parts = site.posts | where: "series", "RF Front End" | sort: "series_part" -%}
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
