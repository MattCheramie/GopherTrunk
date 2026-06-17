---
layout: page
title: "Build in the Open: GitHub + Claude Code from idea to release"
description: A 14-part tutorial series on taking a software project from a blank idea to a public release using GitHub and Claude Code — a generic, transferable workflow, with the open-source GopherTrunk scanner as the worked example.
keywords: github tutorial, claude code, build software with ai, open source workflow, git branching, github actions, semantic versioning, github pages, repository security
nav_group: Blog
permalink: /blog/series/build-in-the-open/
---

**Build in the Open** is a 14-part series on taking a software project from a
blank idea all the way to a public release — using **GitHub** and **Claude
Code** — **one stage per post**. Each part teaches a technique you can apply to
*any* project in *any* language, then shows how the open-source
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) scanner does it for
real, so every lesson comes with a concrete worked example you can copy.

New to the project? Grab a build from the
[downloads page]({{ '/downloads.html' | relative_url }}) and follow along, or
read the [GopherTrunk README](https://github.com/MattCheramie/GopherTrunk) to
see the patterns in their natural habitat.

{%- assign parts = site.posts | where: "series", "Build in the Open" | sort: "series_part" -%}
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
