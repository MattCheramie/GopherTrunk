---
permalink: /search/
title: Search
description: Search the GopherTrunk documentation and the RF & SDR learning path — pages, lessons, and reference, all from one box.
nav_group: Learn RF & SDR
sitemap: false
search: false
---

# Search

<div class="search-page">
  <form class="search-page__form" role="search" action="{{ '/search/' | relative_url }}" method="get">
    <input
      type="search"
      id="gt-search-input"
      name="q"
      class="search-page__input"
      placeholder="Search docs and lessons — e.g. P25, decibels, IQ, control channel"
      autocomplete="off"
      autocapitalize="off"
      spellcheck="false"
      aria-label="Search query"
      autofocus>
  </form>
  <p class="search-page__status" id="gt-search-status" aria-live="polite">Loading the search index…</p>
  <ul class="search-results" id="gt-search-results"></ul>
</div>

<noscript>
Search needs JavaScript. You can still browse the
[learning path](/learn/rf-sdr/), the [glossary](/learn/rf-sdr/glossary/), or use your
browser's find on any page.
</noscript>

<script defer src="{{ '/assets/js/search.js' | relative_url }}"></script>
