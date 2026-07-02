---
layout: page
title: "Crypto Lab: breaking radio encryption, ethically"
description: A 10-part hands-on series on GopherTrunk's cryptolab — a byte-oriented cryptographic-research toolkit for security-testing RF encryption. Triage unknown payloads, run the NIST randomness battery, recover classical ciphers, exploit keystream reuse, and grade a deployment RESISTANT, PARTIAL, or BROKEN. For authorized security testing only.
keywords: cryptolab, rf encryption, keystream reuse, many-time-pad, nist sp 800-22, p25 encryption, tea1 backdoor, cryptanalysis, radio security testing, gophertrunk, message indicator iv
nav_group: Blog
permalink: /blog/series/crypto-lab/
---

**Crypto Lab** is a 10-part tutorial series on
[cryptolab]({{ '/cryptolab/' | relative_url }}) — GopherTrunk's byte-oriented
cryptographic-research toolkit for **security-testing RF encryption**. Its
governing idea is blunt: *attempting decryption is the test.* Point it at
captured ciphertext and it attacks by every applicable method, then grades how
far each got — <span class="lab-verdict lab-verdict--ok">RESISTANT</span>,
<span class="lab-verdict lab-verdict--warn">PARTIAL</span>, or
<span class="lab-verdict lab-verdict--bad">BROKEN</span>. We start from first
triage and climb to keystream-reuse recovery, the `assess` battery, and the
resumable subject framework.

> **Authorized testing only.** These posts are for security researchers,
> licensed operators testing their own systems, and CTF/educational use.
> Intercepting or decrypting communications you are not authorized to access is
> illegal in most jurisdictions. Crypto Lab is deliberately build-tag-gated
> (`-tags cryptolab`) and excluded from the default install.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers, tables, and diagrams** for the medium read, and full prose for the deep
read. **Ada** (new analyst) and **Reese** (the veteran) work each attack with
you.

This is one leg of the **Lab Bench trilogy**:
[Signal Lab]({{ '/blog/series/signal-lab/' | relative_url }}),
[RF Scope]({{ '/blog/series/rf-scope/' | relative_url }}), and
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) (this one). The
mystery signal *Mercury*, captured in Signal Lab and mapped in RF Scope, is
finally cracked here.

{%- assign parts = site.posts | where: "series", "Crypto Lab" | sort: "series_part" -%}
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
