---
slug: radioreference
title: RadioReference
entry_type: organization
category: organizations
description: RadioReference is the largest community database of radio systems, frequencies, and talkgroups — the usual first stop for finding what to monitor.
keywords: RadioReference, frequency database, trunked systems, talkgroups, scanner database, RR
aka: [RadioReference, RR]
autolink: true
see_also: [trunked-radio, control-channel, talkgroup, project-25]
related_lessons:
  - { title: "Finding & identifying systems", url: /learn/rf-sdr/finding-systems/ }
external:
  - { title: "RadioReference.com", url: https://www.radioreference.com/ }
---

**RadioReference** is the largest community-maintained **database of radio systems** —
frequencies, [trunked-system](/reference/trunked-radio/) types,
[control channels](/reference/control-channel/), and [talkgroups](/reference/talkgroup/),
catalogued by location. It is the usual first stop when deciding what to monitor.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 100" role="img" aria-label="RadioReference cataloguing systems that an operator looks up to configure a scanner." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="110" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="75" y="61">RadioReference</text>
    <rect x="175" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="61">system + TG data</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="61">your scanner config</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="130" y1="58" x2="174" y2="58" marker-end="url(#or_rr)"/><line x1="285" y1="58" x2="329" y2="58" marker-end="url(#or_rr)"/></g>
  </g>
  <defs><marker id="or_rr" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Operators look up local systems on RadioReference and import the details into their scanner.</figcaption>
</figure>


## Overview

For most populated areas, the local systems are already documented on RadioReference, so
you can look up a system's control-channel frequency and talkgroup list rather than
discovering them from scratch. GopherTrunk can [import](/import.html) these details
directly.
