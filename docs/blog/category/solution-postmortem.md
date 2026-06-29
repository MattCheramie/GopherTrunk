---
layout: page
title: "Blog: Solution Postmortem"
description: Post-incident write-ups — what broke, how it was diagnosed, and the fix.
nav_group: Blog
permalink: /blog/category/solution-postmortem/
category: solution-postmortem
---

{% assign posts = site.categories[page.category] %}
{% if posts and posts.size > 0 %}
<ul class="post-list">
  {% for post in posts %}
    <li class="post-card">
      <a class="post-card__link" href="{{ post.url | relative_url }}">
        <h2 class="post-card__title">{{ post.title }}</h2>
      </a>
      <p class="post-card__meta">
        <time datetime="{{ post.date | date_to_xmlschema }}">{{ post.date | date: "%B %-d, %Y" }}</time>
      </p>
      {% if post.description %}
        <p class="post-card__desc">{{ post.description }}</p>
      {% elsif post.excerpt %}
        <p class="post-card__desc">{{ post.excerpt | strip_html | truncate: 200 }}</p>
      {% endif %}
    </li>
  {% endfor %}
</ul>
{% else %}
<p class="post-list__empty">No posts in this category yet.</p>
{% endif %}

<p class="blog-feed-link">
  See <a href="{{ '/blog/' | relative_url }}">all posts</a> or subscribe via <a href="{{ '/feed.xml' | relative_url }}">RSS</a>.
</p>
