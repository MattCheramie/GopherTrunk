---
slug: imt
title: International Mobile Telecommunications (IMT)
entry_type: technology
category: cellular
description: IMT is the ITU-R umbrella under which each mobile generation is defined by minimum requirements a technology must meet to be badged — IMT-2000 for 3G, IMT-Advanced for 4G, and IMT-2020 for 5G.
keywords: IMT, International Mobile Telecommunications, IMT-2000, IMT-Advanced, IMT-2020, 3G, 4G, 5G, ITU-R, mobile generation, IMT spectrum
aka: [IMT, International Mobile Telecommunications, IMT-2000, IMT-Advanced, IMT-2020]
autolink: true
infobox:
  - { label: Type, value: ITU-R requirement framework }
  - { label: Sets, value: Minimum targets per generation }
  - { label: Milestones, value: "IMT-2000 (3G), -Advanced (4G), -2020 (5G)" }
see_also: [itu-r, lte, lte-advanced, 5g-nr, umts-wcdma, wrc, cellular-handover]
cite_urls:
  - https://www.itu.int/en/ITU-R/study-groups/rsg5/rwp5d/imt/Pages/default.aspx
  - https://en.wikipedia.org/wiki/International_Mobile_Telecommunications
---

**International Mobile Telecommunications** (**IMT**) is the [ITU-R](/reference/itu-r/)
umbrella under which each mobile **generation** is defined by the minimum requirements a
technology must meet to be badged with it.[^itu] IMT does not itself design radios: it sets
targets — peak data rate, latency, mobility, spectral efficiency — and any radio standard
that meets them qualifies as an IMT technology. The milestones map onto the familiar
generations: **IMT-2000** is 3G, **IMT-Advanced** is 4G, and **IMT-2020** is 5G.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="A timeline from IMT-2000 for 3G to IMT-Advanced for 4G to IMT-2020 for 5G, with the qualifying 3GPP technologies listed under each milestone." xmlns="http://www.w3.org/2000/svg">
  <line x1="30" y1="55" x2="430" y2="55" stroke="currentColor" stroke-opacity="0.5"/>
  <g fill="currentColor"><circle cx="90" cy="55" r="4"/><circle cx="230" cy="55" r="4"/><circle cx="370" cy="55" r="4"/></g>
  <g font-size="10" fill="currentColor" text-anchor="middle">
    <text x="90" y="40">IMT-2000</text>
    <text x="230" y="40">IMT-Advanced</text>
    <text x="370" y="40">IMT-2020</text>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle" fill-opacity="0.85">
    <text x="90" y="78">3G</text>
    <text x="230" y="78">4G</text>
    <text x="370" y="78">5G</text>
  </g>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="52" y="92" width="76" height="20" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="90" y="105">UMTS / WCDMA</text>
    <rect x="188" y="92" width="84" height="20" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="230" y="105">LTE-Advanced</text>
    <rect x="336" y="92" width="68" height="20" rx="3" fill="none" stroke="currentColor" stroke-width="1"/><text x="370" y="105">5G NR</text>
  </g>
  <text x="230" y="140" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">ITU-R sets the targets; 3GPP standards are the radios that qualify</text>
</svg>
<figcaption>Each IMT milestone names a generation by requirement; the boxes below are the 3GPP radio technologies that met the targets and earned the badge.</figcaption>
</figure>

## How it works

IMT is a framework maintained by the ITU Radiocommunication Sector, ITU-R, working through
its study groups. For each generation, ITU-R publishes a set of **minimum technical
requirements**: quantities such as peak and user-experienced data rate, control- and
user-plane latency, mobility (the speed at which a link must hold up), connection density,
and spectral efficiency. A candidate radio technology is then evaluated against those
numbers, and if it passes it is accepted into the IMT family for that generation. The scheme
deliberately separates **requirements** from **implementation**: the ITU says what a
generation must achieve, and industry decides how.

The radios that actually meet the targets come almost entirely from **3GPP**, the standards
partnership that authors the mobile air interfaces. [UMTS/WCDMA](/reference/umts-wcdma/)
qualified under IMT-2000; [LTE-Advanced](/reference/lte-advanced/) — the enhanced release of
[LTE](/reference/lte/) — met the IMT-Advanced bar for 4G; and [5G NR](/reference/5g-nr/)
satisfies IMT-2020. (Baseline LTE was marketed as 4G before it fully met IMT-Advanced, a
common gap between the marketing label and the formal badge.) Because the requirement is what
defines the generation, the same milestone can admit more than one qualifying technology, and
a technology is a "true" member of a generation only once ITU-R accepts it.

## Spectrum and the WRC

A generation needs spectrum as well as a standard. The **[World Radiocommunication
Conference](/reference/wrc/)** (WRC), the ITU treaty conference held every few years,
identifies bands as **"IMT spectrum"** — the frequency ranges administrations agree to make
available for IMT systems. These identifications, recorded in the international Radio
Regulations, are what let the same 3G/4G/5G bands be reused across borders, so a device and
its [handovers](/reference/cellular-handover/) work internationally. IMT thus ties together
three ITU workstreams: the *requirements* (IMT framework), the *evaluation* of candidate
radios, and the *spectrum* identifications made at the WRC.

## Relevance to SDR

IMT is the vocabulary behind the generation labels an SDR user meets constantly — "3G",
"4G", "5G" are shorthand for IMT-2000, IMT-Advanced, and IMT-2020 — and the WRC's IMT-band
identifications explain why cellular signals appear in the same frequency ranges worldwide.
Knowing that a badge is a *requirement*, not a single radio, clarifies why several
technologies can share a generation. GopherTrunk decodes land-mobile trunking rather than
IMT cellular systems, but IMT frames the wider mobile-spectrum landscape an SDR listener
navigates, and the ITU's role here mirrors its role in allocating the land-mobile spectrum
GopherTrunk does target.

## Sources

[^itu]: [IMT — ITU-R Working Party 5D](https://www.itu.int/en/ITU-R/study-groups/rsg5/rwp5d/imt/Pages/default.aspx) — the ITU-R group responsible for the IMT framework and the minimum requirements for each generation.
[^wiki]: [International Mobile Telecommunications](https://en.wikipedia.org/wiki/International_Mobile_Telecommunications) — Wikipedia, for the IMT-2000, IMT-Advanced, and IMT-2020 milestones and their qualifying technologies.
