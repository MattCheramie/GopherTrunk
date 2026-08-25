---
layout: page
title: Site map
description: A complete, browsable map of every page on the GopherTrunk site — guides, learning paths, the Field Guide, technical docs, and the blog. Looking for the machine-readable sitemap for search engines? It lives at /sitemap.xml.
permalink: /sitemap/
nav_group: About
---

# Site map

Every page on GopherTrunk, grouped by section. Search engines should use the
machine-readable [XML sitemap](/sitemap.xml) — that's the one to submit in
Google Search Console.

- [Home]({{ '/' | relative_url }})

## Guides &amp; documentation

<ul>
{%- assign pages = site.html_pages | sort: "title" -%}
{%- for p in pages -%}
  {%- if p.url == "/" -%}{%- continue -%}{%- endif -%}
  {%- if p.sitemap == false -%}{%- continue -%}{%- endif -%}
  {%- comment -%} `unlisted: true` pages stay in sitemap.xml for search engines
      but are excluded from every on-site listing surface. {%- endcomment -%}
  {%- if p.unlisted -%}{%- continue -%}{%- endif -%}
  {%- if p.url contains "/learn/" or p.url contains "/reference/" or p.url contains "/blog/" -%}{%- continue -%}{%- endif -%}
  <li><a href="{{ p.url | relative_url }}">{{ p.title | default: p.name }}</a></li>
{%- endfor -%}
</ul>

## Learn

Structured learning content. See [all learning]({{ '/learn/' | relative_url }}).

### Learning paths

Optional guided routes through several modules toward competence in a domain.

<ul>
{%- for lp in site.data.learn.learning_paths -%}
  <li><a href="{{ '/learn/paths/' | append: lp.id | append: '/' | relative_url }}">{{ lp.title }}</a></li>
{%- endfor -%}
</ul>

{%- for path in site.data.learn.modules %}
### {{ path.title }}

<ul>
{%- assign path_prefix = "/learn/" | append: path.id | append: "/" -%}
{%- assign learn_pages = site.html_pages | where_exp: "p", "p.url contains path_prefix" | sort: "url" -%}
{%- for p in learn_pages -%}
  {%- if p.sitemap == false -%}{%- continue -%}{%- endif -%}
  <li><a href="{{ p.url | relative_url }}">{{ p.title | default: p.name }}</a></li>
{%- endfor -%}
</ul>
{%- endfor %}

## Field Guide

Long-form reference articles. See the [Field Guide overview]({{ '/reference/' | relative_url }}).

<ul>
{%- assign ref_pages = site.html_pages | where_exp: "p", "p.url contains '/reference/'" | sort: "title" -%}
{%- for p in ref_pages -%}
  {%- if p.sitemap == false -%}{%- continue -%}{%- endif -%}
  {%- if p.url == "/reference/" -%}{%- continue -%}{%- endif -%}
  <li><a href="{{ p.url | relative_url }}">{{ p.title | default: p.name }}</a></li>
{%- endfor -%}
</ul>

## Blog

Index &amp; archive pages, then every post.

<ul>
{%- assign blog_pages = site.html_pages | where_exp: "p", "p.url contains '/blog/'" | sort: "url" -%}
{%- for p in blog_pages -%}
  {%- if p.sitemap == false -%}{%- continue -%}{%- endif -%}
  <li><a href="{{ p.url | relative_url }}">{{ p.title | default: p.name }}</a></li>
{%- endfor -%}
{%- for post in site.posts -%}
  <li><a href="{{ post.url | relative_url }}">{{ post.title }}</a> <small>({{ post.date | date: "%Y-%m-%d" }})</small></li>
{%- endfor -%}
</ul>
