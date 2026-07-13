---
slug: radioreference
title: RadioReference
entry_type: organization
category: organizations
description: RadioReference is the largest community database of radio systems, frequencies, and talkgroups — the usual first stop for finding what to monitor.
keywords: RadioReference, frequency database, trunked systems, talkgroups, scanner database, RR, RRDB, live audio feeds
aka: [RadioReference, RR]
autolink: true
infobox:
  - { label: Type, value: Community database + forum }
  - { label: Covers, value: Systems, frequencies, talkgroups }
  - { label: Region, value: Primarily US/Canada }
see_also: [trunked-radio, control-channel, talkgroup, project-25, system-id, wacn]
related_lessons:
  - { title: "Finding & identifying systems", url: /learn/rf-sdr/finding-systems/ }
cite_urls:
  - https://www.radioreference.com/
  - https://en.wikipedia.org/wiki/RadioReference.com
---

**RadioReference** is the largest community-maintained **database of radio systems** —
frequencies, [trunked-system](/reference/trunked-radio/) types,
[control channels](/reference/control-channel/), and [talkgroups](/reference/talkgroup/),
catalogued by location. It is the usual first stop when deciding what to monitor.[^home]

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 120" role="img" aria-label="An operator looks up a local system on RadioReference and imports its control channel and talkgroup list into a scanner." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <rect x="20" y="42" width="110" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="75" y="55">RadioReference</text><text x="75" y="67" font-size="7.5">community DB</text>
    <rect x="175" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="230" y="55">control channel +</text><text x="230" y="67" font-size="7.5">talkgroup list</text>
    <rect x="330" y="42" width="110" height="32" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="385" y="55">your scanner /</text><text x="385" y="67" font-size="7.5">GopherTrunk config</text>
    <g stroke="currentColor" stroke-width="1.1"><line x1="130" y1="58" x2="174" y2="58" marker-end="url(#or_rr)"/><line x1="285" y1="58" x2="329" y2="58" marker-end="url(#or_rr)"/></g>
    <text x="230" y="96" font-size="8" fill="currentColor" fill-opacity="0.85">look up local systems → import rather than discover from scratch</text>
  </g>
  <defs><marker id="or_rr" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Operators look up local systems on RadioReference and import the control-channel and talkgroup details into their scanner.</figcaption>
</figure>

## Overview

RadioReference.com, launched in the early 2000s, is a community website combining a large,
structured **database** (often abbreviated RRDB) with an active discussion forum and a network
of live audio feeds. The database is crowdsourced and moderated: hobbyists submit and verify
the frequencies, [trunked-radio](/reference/trunked-radio/) system parameters, and talkgroup
assignments for their areas, and the result is by far the most comprehensive public catalogue
of who transmits where, especially across the United States and Canada with growing
international coverage. For a [P25](/reference/project-25/) or other trunked system, a database
entry typically lists the site frequencies, which channel is the
[control channel](/reference/control-channel/), the system's identifiers
([WACN](/reference/wacn/) and [system ID](/reference/system-id/) for P25), and a decoded list of
[talkgroups](/reference/talkgroup/) with human-readable labels — the difference between hearing
"talkgroup 4128" and knowing it is "City Fire Dispatch".

Much of the database's depth exists because it complements decoding rather than replacing it.
Someone with an SDR and a decoder first captures a system's raw parameters and talkgroup
activity, then contributes them back so the next person can simply look them up. Premium content
sits behind a paid subscription, which funds the infrastructure. The site also hosts the live
audio feeds that populate scanner-streaming apps, and its forums are a long-running knowledge
base for identifying unfamiliar signals. RadioReference is a community reference, not a
regulator — the underlying spectrum is allocated by bodies like the [FCC](/reference/fcc/) — but
for practical "what should I tune to and what will I hear" questions it is the definitive
starting point.

## Relevance to SDR

For an SDR trunk-tracker, RadioReference collapses the hardest part of getting started. In most
populated areas the local systems are already documented, so instead of blindly searching for a
control channel and reverse-engineering an unlabelled talkgroup list, you can look up a system's
control-channel frequency and its talkgroup map and be listening to identified traffic within
minutes. That head start is especially valuable for [trunked](/reference/trunked-radio/) systems,
where you must know the control channel before the decoder can follow voice grants at all.

GopherTrunk can [import](/import.html) RadioReference system details directly, turning a database
entry into a working configuration. The relationship is genuinely two-way: GopherTrunk (like any
decoder) is also a *tool for discovery*, surfacing the control channel, [system ID](/reference/system-id/),
and active talkgroups of a system that the operator can then verify and submit back to the
database. See the [finding & identifying systems](/learn/rf-sdr/finding-systems/) lesson for how
to combine RadioReference lookups with on-air discovery.

## Sources

[^home]: [RadioReference.com](https://www.radioreference.com/) — the official RadioReference site, the community database of radio systems, frequencies, and talkgroups, plus its forums and live feeds.
[^wiki]: [RadioReference.com](https://en.wikipedia.org/wiki/RadioReference.com) — Wikipedia, for the site's history and its role as a scanner and frequency database.
