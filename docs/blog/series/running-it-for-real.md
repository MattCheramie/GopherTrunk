---
layout: page
title: "Running It For Real: deploying and hardening a 24/7 SDR service"
description: A 14-part deep dive into taking GopherTrunk from a laptop demo to a hardened, always-on service — auth posture and TLS, Prometheus metrics, structured logs, the diagnostics reporter and SDR doctor, the opt-in feature matrix, broadcast backends, Docker and systemd, all in pure Go.
keywords: sdr daemon deployment, gophertrunk hardening, prometheus metrics sdr, structured logging, sdr doctor preflight, broadcastify openmhz rdioscanner, docker rtl-sdr usb passthrough, systemd hardening, gophertrunk running it for real
nav_group: Blog
permalink: /blog/series/running-it-for-real/
---

**Running It For Real** is a 14-part deep dive into everything between "it decodes
on my laptop" and "it has been feeding a public
[Broadcastify](https://www.broadcastify.com/) channel for six months without me
touching it." [GopherTrunk](https://github.com/MattCheramie/GopherTrunk) ships as
a single static binary; this series is about running that binary as a real
service — the auth posture you pick before it leaves your LAN, the metrics and
logs that tell you it's healthy, the diagnostics that catch a failing dongle
before it costs you a call, the feature flags that keep optional subsystems off
until you want them, the aggregator uploads, and the Docker/systemd plumbing that
keeps it up.

Where [RF Front End]({{ '/blog/series/rf-front-end/' | relative_url }}) ended on
the diagnostics-and-metrics *payoff* for the radio layer, this series is that
same instinct applied to the whole daemon. And where
[Build in the Open]({{ '/blog/series/build-in-the-open/' | relative_url }})
covered CI, releases, and securing the repository, this is the operational
half — securing and running the thing the repository produces.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose with real
code for the deep read.

New here? The [Hardening]({{ '/hardening.html' | relative_url }}) and
[Opt-in features]({{ '/opt-in-features.html' | relative_url }}) docs are the
operator reference; this series is the design behind them, plus the
[Containers & Deployment]({{ '/learn/deployment/' | relative_url }}) module for
first principles.

{%- assign parts = site.posts | where: "series", "Running It For Real" | sort: "series_part" -%}
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
