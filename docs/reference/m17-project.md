---
slug: m17-project
title: M17 Project
entry_type: organization
category: organizations
description: The M17 Project is an open-source community developing the royalty-free M17 amateur digital-voice protocol and associated open hardware and software.
keywords: M17 Project, open source, amateur radio, Codec 2, royalty-free, digital voice
aka: [M17 Project]
autolink: true
infobox:
  - { label: Type, value: Open-source community project }
  - { label: Develops, value: M17 protocol + open tools }
  - { label: Ethos, value: Royalty-free, patent-free }
see_also: [m17, codec2, vocoder]
related_lessons:
  - { title: "The digital protocol landscape", url: /learn/protocol-landscape/ }
external:
  - { title: "M17 Project", url: https://m17project.org/ }
---

The **M17 Project** is an open-source community that develops the royalty-free
**[M17](/reference/m17/)** amateur digital-voice protocol along with open hardware and
software implementations.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="The open-source M17 Project maintains the royalty-free M17 protocol." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="100" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="70" y="61">M17 Project</text>
    <rect x="170" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="225" y="61">maintains</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="61">M17 (open)</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="120" y1="58" x2="169" y2="58" marker-end="url(#rel_m17-project)"/><line x1="280" y1="58" x2="329" y2="58" marker-end="url(#rel_m17-project)"/></g>
  </g>
  <defs><marker id="rel_m17-project" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>The M17 Project is the open-source community that maintains the royalty-free M17 protocol.</figcaption>
</figure>

## Overview

Founded in 2019, the project set out to build a modern amateur protocol free of vocoder
licensing constraints, adopting the open [Codec 2](/reference/codec2/)
[vocoder](/reference/vocoder/) instead of patented alternatives.

## Relevance to SDR

The project's openness makes fully free M17 decoders possible, including GopherTrunk's
link-layer support.
