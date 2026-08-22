---
title: "The $4 Million Side Project: What GopherTrunk Would Cost Without AI"
description: We audited every line of GopherTrunk — 425k lines of code, 2.7 million words of docs — and priced what it would cost a human team to build. The answer says something bigger about what AI makes viable.
keywords: cost to build software, ai software development, claude code, solo developer, software cost estimation, person-years, open source economics, ai pair programming, sdr scanner, gophertrunk
category: announcements
tags: [meta, claude-code, ai, open-source, milestones, economics]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
charts: true
---

> **TL;DR:** We measured everything in the GopherTrunk repository — 370k lines
> of Go, five TypeScript web apps, a clean-room ACELP vocoder, and a 2.7
> million-word documentation site — and estimated what it would take a human
> team to build it: roughly **35,000 hours (~17 person-years)** of coding,
> engineering, and writing, or about **$4.3 million** at market rates. The
> repository's actual commit history spans **17 calendar days**. That gap is
> the point: AI didn't just make this project cheaper — it made a project
> *exist* that no company would ever have funded and no individual could ever
> have finished.

**Key takeaways**

- A full audit of the repo counts ~425k lines of code (44% of the Go is
  tests) and ~2.78 million words of prose across 2,300+ pages.
- Priced bottom-up at specialist rates, the human-equivalent effort is
  **~20,000 hours of coding, ~8,000 of engineering, and ~7,400 of writing** —
  a defensible range of **$3.0M–$5.5M**, centered near **$4.3M**.
- The commit history covers 17 days. Nobody compresses 17 person-years into
  17 days by typing faster — the compression *is* the story.
- AI didn't remove the engineering. The judgment calls — field captures,
  failing-first tests, on-air verification — stayed human. What changed is
  that one person's judgment now goes 100× further.

*GopherTrunk is built in the open with [Claude
Code](https://claude.ai/code) — the whole process is documented in the
[Build in the Open]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})
series, and there's a full [learning module on AI-assisted
development]({{ '/learn/ai-software-dev/' | relative_url }}) if you want to
try the workflow yourself. This post is the accounting.*

## In this post

- **The census** — what is actually in this repository, measured, not
  estimated.
- **The hours** — how long a human team would need, split into coding,
  engineering, and writing.
- **The invoice** — what those hours cost at real market rates.
- **The 17 days** — what the calendar says actually happened.
- **The real lesson** — why "cheaper" is the wrong word, and "viable" is the
  right one.

## The census: what's actually in the repo

Cost estimates are only as honest as their inputs, so we started by counting
everything (excluding `node_modules`, `.git`, and build output):

| What | How much |
|---|---|
| Go source | **370,833 lines** across 1,784 files in 178 packages |
| …of which tests | **163,746 lines** (~44% of all Go) |
| TypeScript | **~39,200 lines** across five separate SPAs |
| SQL / shell / proto / config | **~15,400 lines** |
| Documentation & articles | **2,336 markdown pages, ~2.68 million words** |
| Changelog + engineering notes | **~80,000 words** more |

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="600" height="300" role="img"
        aria-label="Lines of code by layer: 207 thousand lines of production Go, 164 thousand of Go tests, 39 thousand of TypeScript, 15 thousand of other code"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["Go (production)","Go (tests)","TypeScript (5 SPAs)","SQL/scripts/other"],
"values":[207,164,39,15],
"ylabel":"thousands of lines" }
</script>
<figcaption>The codebase by layer. Nearly half the Go is tests — the failing-first regression discipline documented across the issue tracker is expensive, and it shows up in the line counts.</figcaption>
</figure>

And it's not generic CRUD code. The hard core is real-time DSP and
protocol work — the kind of software where a single wrong constant costs a
week: symbol-timing recovery, blind CMA and trained LMS equalizers,
maximal-ratio diversity combining, four trunking protocol families (P25,
DMR, NXDN, TETRA including Direct Mode), AMBE/IMBE voice decoders, and a
**clean-room ACELP vocoder validated bit-identically against the ETSI
reference codec**. Around that core sit a daemon, gRPC and REST APIs, five
web frontends, installers for three platforms, and a documentation site
bigger than most publishers' annual output.

## The hours: what a human team would need

We estimated bottom-up, using delivered-lines-per-hour rates appropriate to
each layer — slow for DSP and protocol conformance (~10 lines/hour of
*finished, tested* code is normal in that world), faster for
infrastructure, APIs, and frontend work.

**Coding — ~20,000 hours.** DSP/protocol/vocoder core (~80k lines at DSP
pace): ~8,000 hours. Daemon, APIs, storage, config, TUI (~127k lines):
~5,000 hours. The test suite — much of it capture-driven regression
harnesses, not unit boilerplate: ~5,500 hours. Five SPAs plus glue:
~2,400 hours.

**Engineering — ~8,000 hours.** The non-typing work this domain demands:
reading ETSI and TIA specifications, cross-checking independent reference
decoders, root-causing on-air failures from field captures, conformance
validation against reference implementations, CI/release/installer
engineering, and review. For specialist communications work this reliably
runs 35–45% of total technical effort — and GopherTrunk's own issue
tracker, with its multi-week capture-verified investigations, is the
evidence.

**Writing — ~7,400 hours.** 2.68 million words of researched guides and
articles at a professional pace of ~400 finished words/hour, plus the
technical docs, changelog, and clean-room provenance write-ups at a slower
~250 words/hour.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="600" height="300" role="img"
        aria-label="Estimated human hours by category: coding 20,000, engineering 8,000, writing 7,400"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["coding","engineering","writing"],
"values":[20000,8000,7400],
"ylabel":"estimated human hours" }
</script>
<figcaption>~35,400 hours total — about 17 person-years of blended effort. For scale: SDRTrunk's ~500k lines of Java represent roughly a decade of its lead developer's time, and it doesn't include a clean-room vocoder, five SPAs, or a 2.7M-word docs site.</figcaption>
</figure>

## The invoice: pricing it at market rates

Using fully-loaded US contractor rates — and note that the engineering
column commands a premium, because communications-DSP specialists who can
take a vocoder through ETSI conformance are genuinely rare:

| Category | Hours | Rate | Cost |
|---|---|---|---|
| Coding (senior Go/TS engineer) | 20,000 | $120/hr | **$2,400,000** |
| Engineering (DSP/SDR specialist) | 8,000 | $180/hr | **$1,440,000** |
| Writing (content + technical writer) | 7,400 | $60/hr | **$444,000** |
| **Total** | **~35,400** | | **~$4.28M** |

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="600" height="300" role="img"
        aria-label="Labor cost by category in millions of dollars: coding 2.4, engineering 1.44, writing 0.44"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["coding","engineering","writing"],
"values":[2.40,1.44,0.44],
"ylabel":"labor cost ($ millions)" }
</script>
<figcaption>The human-equivalent invoice. With ±30% uncertainty on the productivity assumptions, the defensible range is $3.0M–$5.5M.</figcaption>
</figure>

Two honest caveats. First, productivity heuristics are heuristics — hence
the range, not a point. Second, part of the docs site follows repeated
templates, so a human content operation with editors might do the writing
in ~4,500 hours instead of 6,700; that alone moves the total by ~$130k.
Neither caveat changes the order of magnitude.

## The 17 days

Here is the number that reframes all the others. The repository's entire
commit history — 273 commits — spans **August 5 to August 21, 2026.
Seventeen calendar days.** Of those commits, 205 were authored by Claude
and 68 by human collaborators.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="600" height="300" role="img"
        aria-label="Actual calendar days, 17, versus equivalent human working days, about 4,400"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["actual calendar days","human-equivalent working days"],
"values":[17,4425],
"colors":["#2f9e44","#155799"],
"ylabel":"days" }
</script>
<figcaption>35,400 hours ÷ 8 = ~4,400 human working days of equivalent effort, delivered in 17 calendar days. The green bar is not a rendering error.</figcaption>
</figure>

Nobody types 260× faster than a professional team. The compression comes
from somewhere else: an AI collaborator that can hold four protocol
specifications, a 178-package codebase, and last Tuesday's field-capture
findings in working memory at once — and that never gets tired of writing
the regression test.

## The real lesson: not cheaper — *viable*

It's tempting to read this as a cost-savings story: "$4.3M of work for the
price of a Claude subscription." But that framing quietly assumes the $4.3M
version was ever going to happen. It wasn't.

No company funds a $4.3M, multi-year build for a **free, open-source
radio scanner**. The market wouldn't return the investment, so the project
simply doesn't get built — not badly, not slowly, *not at all*. And no solo
developer self-funds 17 person-years of nights and weekends; the honest
projects in this space that come closest are decade-long labors of love by
extraordinary individuals, and they are rare precisely because the cost is
so brutal.

<figure class="lab-figure">
<svg viewBox="0 0 664 150" width="664" height="150" role="img" aria-label="Diagram: one person's judgment plus an AI collaborator combine into work that was previously not viable — the same inputs that used to end at 'not viable: 17 person-years, 4.3 million dollars' now produce a shipped open-source project in 17 days.">
  <rect x="10" y="14" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="33" text-anchor="middle" fill="currentColor" font-size="10">one person's judgment</text>
  <text x="85" y="48" text-anchor="middle" fill="var(--fg-muted)" font-size="8">taste, verification, field work</text>
  <rect x="10" y="86" width="150" height="46" rx="6" fill="none" stroke="currentColor"/>
  <text x="85" y="105" text-anchor="middle" fill="currentColor" font-size="10">AI collaborator</text>
  <text x="85" y="120" text-anchor="middle" fill="var(--fg-muted)" font-size="8">breadth, stamina, recall</text>
  <line x1="160" y1="37" x2="220" y2="66" stroke="currentColor"/><polygon points="215,60 226,69 213,71" fill="currentColor"/>
  <line x1="160" y1="109" x2="220" y2="80" stroke="currentColor"/><polygon points="213,75 226,77 215,86" fill="currentColor"/>
  <rect x="228" y="50" width="170" height="46" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="313" y="69" text-anchor="middle" fill="var(--accent)" font-size="10">judgment × leverage</text>
  <text x="313" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="8">every decision human, every hour amplified</text>
  <line x1="398" y1="73" x2="440" y2="73" stroke="currentColor"/><polygon points="440,69 450,73 440,77" fill="currentColor"/>
  <rect x="450" y="14" width="204" height="52" rx="6" fill="none" stroke="var(--fg-muted)" stroke-dasharray="5 4"/>
  <text x="552" y="34" text-anchor="middle" fill="var(--fg-muted)" font-size="10">before: not viable</text>
  <text x="552" y="50" text-anchor="middle" fill="var(--fg-muted)" font-size="8">17 person-years · ~$4.3M · never built</text>
  <rect x="450" y="82" width="204" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="552" y="102" text-anchor="middle" fill="var(--accent)" font-size="10">now: shipped, open source</text>
  <text x="552" y="118" text-anchor="middle" fill="var(--fg-muted)" font-size="8">17 days · free to everyone · improving daily</text>
</svg>
<figcaption>The change isn't a discount on existing projects — it's a change in which projects can exist at all.</figcaption>
</figure>

What AI changes is the *viability threshold*. A whole category of software
— too niche for venture capital, too large for a hobbyist, too valuable to
its small community to stay unbuilt — suddenly becomes something one
motivated person can actually ship. GopherTrunk sits squarely in that
category: scanner hobbyists, volunteer firefighters, storm chasers, and
railfans were never going to be a $4.3M market. They now have a modern,
free, cross-platform trunking scanner anyway.

And crucially, the human didn't leave the loop — the *role* changed. Every
verified fix in this project traces to a human decision: which capture to
record, which symptom is real, whether a green test actually proves
anything (the project's own
[issue-closing policy](https://github.com/MattCheramie/GopherTrunk/blob/main/CONTRIBUTING.md)
exists because it once didn't). The AI wrote most of the lines; the
engineering discipline that makes those lines trustworthy is human, and it
is the part that doesn't compress. That division of labor — human
judgment, machine leverage — is the whole model, and it's teachable:
the [Build in the Open]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})
series walks through it from a blank repo.

If you have a project like this — the one you've told yourself is too big,
too niche, too much — the math just changed. The itch you've been ignoring
because it would take a decade might now take a season.

## FAQ

**How were the hour estimates calculated?**
Bottom-up, from measured line and word counts, using delivered-productivity
rates appropriate to each layer: ~10 lines/hour for finished DSP and
protocol-conformance code, ~25 for infrastructure, ~30 for tests, ~20 for
frontend, and 250–400 finished words/hour for prose. Engineering
(specification study, field debugging, conformance validation) is estimated
at 35–45% of technical effort, standard for specialist communications
software. The ±30% band on the assumptions gives the $3.0M–$5.5M range.

**Isn't AI-generated code lower quality, making the comparison unfair?**
Judge it by the artifacts: 44% of the Go is tests, fixes land with
failing-first regressions, the vocoder is validated bit-for-bit against the
ETSI reference codec, and on-air behavior is verified against real
captures before an issue closes. Those are stricter standards than most
commercial codebases meet. Quality here isn't a property of who typed the
code — it's a property of the verification discipline around it.

**Did AI really do everything?**
No, and that's central to the story. Humans chose what to build, recorded
the field captures, judged which symptoms were real, and enforced the
verification discipline. Roughly three-quarters of commits are
AI-authored, but every one of them traces back to a human decision about
what mattered. The right mental model is a specialist team of one, with
leverage.

**What does this mean for my own project idea?**
That the viability line has moved. Projects that were previously
"technically possible but economically absurd" — niche tools, deep
rewrites, well-documented open source for small communities — are now
within reach of a single motivated person. Start the way any good project
starts (a real problem, ruthlessly scoped — see
[Build in the Open, Part 1]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})),
then bring the leverage.
