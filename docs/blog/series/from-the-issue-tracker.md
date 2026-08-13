---
layout: page
title: "From the Issue Tracker: postmortems of the bugs that fought back"
description: A 22-part postmortem series distilled from GopherTrunk's closed issues — the misleading symptoms, wrong first fixes, and hard-won root causes behind the trickiest bugs, from the first P25 lock to the diagnostic habits that ended guess-and-retest debugging.
keywords: sdr debugging postmortem, p25 decoder bugs, rtl-sdr usb troubleshooting, airspy sample rate, dsp root cause analysis, self-consistent test trap, gophertrunk issues
nav_group: Blog
permalink: /blog/series/from-the-issue-tracker/
---

**From the Issue Tracker** is a postmortem series mined from
[GopherTrunk](https://github.com/MattCheramie/GopherTrunk)'s closed GitHub issues.
Every post is a true story with receipts: the symptom as a user actually reported
it, the plausible explanations that turned out to be wrong, the diagnostic that
finally cracked it, and the fix — linked back to the original issue thread so you
can check the work.

Where the other series on this blog explain how GopherTrunk *works*, this one
explains how it *broke*, and what each breakage taught us. The recurring villains:
round-trip tests that validate their own bugs, success-only log lines whose silence
means nothing, hardware that lies politely, and "smoking gun" evidence that was an
artifact of the instrument. The series closes with three meta-lesson posts that pull
those threads together.

Every post reads three ways: a **TL;DR + cheat-sheet** for skimmers, **bold
headers and tables** for the medium read, and the full investigation narrative for
the deep read.

Reference companions to this series live in the Field Guide's
[Field Notes domain]({{ '/reference/#domain-field-notes' | relative_url }}) — the same
knowledge, condensed into look-up form:
[hardware quirks]({{ '/reference/#fn-hardware' | relative_url }}),
[configuration gotchas]({{ '/reference/#fn-config' | relative_url }}),
[diagnostic signatures]({{ '/reference/#fn-diagnostics' | relative_url }}), and
[protocol facts the specs hide]({{ '/reference/#fn-protocol' | relative_url }}).

{%- assign parts = site.posts | where: "series", "From the Issue Tracker" | sort: "series_part" -%}
{%- if parts and parts.size > 0 -%}
<ol class="post-list series-list">
  {%- for post in parts -%}
    <li class="post-card">
      <a class="post-card__link" href="{{ post.url | relative_url }}">
        <h2 class="post-card__title">{{ post.title }}</h2>
      </a>
      <p class="post-card__meta">
        <time datetime="{{ post.date | date_to_xmlschema }}">{{ post.date | date: "%B %-d, %Y" }}</time>
        <span class="category-chip">Solution Postmortem</span>
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
  See all <a href="{{ '/blog/category/solution-postmortem/' | relative_url }}">solution postmortems</a>
  or subscribe via <a href="{{ '/feed.xml' | relative_url }}">RSS</a>.
</p>
