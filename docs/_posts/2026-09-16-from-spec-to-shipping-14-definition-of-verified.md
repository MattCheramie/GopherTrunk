---
title: "From Spec to Shipping, Part 14: The Definition of Verified"
description: The finale — closing an issue is a claim the problem is gone, and GopherTrunk defines exactly when that claim is earned. The issue-closing policy, the guard hook that asks a human before any close, Refs versus Closes, and the full shipping checklist from spec to verified.
category: deep-dives
keywords: when to close a bug as fixed, verified fix definition, issue closing policy, refs vs closes github, regression test plus reporter confirmation, pretooluse guard hook, protocol decoder shipping checklist, gophertrunk from spec to shipping
tags: [from-spec-to-shipping, verification, issues, process, methodology, testing]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "From Spec to Shipping"
series_part: 14
---

*Part 14 — the last — of **From Spec to Shipping**, a 14-part series on how a
protocol decoder actually gets written — from standards documents and
independent references to code you can trust on air. We have read the spec,
chosen references, pinned parsers with literal vectors, built conformance
harnesses, refereed disagreements with captures, made tests that can
disagree, gated claims on the air, failed first, and — in
[Part 13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
— instrumented the result. One act remains, and it is the one this project
got wrong in public: saying the problem is *gone*. This finale defines
"verified," shows the machinery that enforces the definition, and folds the
whole series into one shipping checklist.*

> **TL;DR:** Closing an issue is a **claim that the reported problem is
> gone**, and GopherTrunk's policy (`CLAUDE.md`, "Issue-closing policy")
> defines when that claim is earned: a **failing-first regression test now
> passes AND the reporter has confirmed it** — or you reproduced the original
> symptom yourself and showed the change resolves it. The policy exists
> because [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) was
> closed **twice** on an unverified fix while the symptom stayed live
> ([#771](https://github.com/MattCheramie/GopherTrunk/issues/771)); a
> PreToolUse guard (`.claude/hooks/guard-issue-close.py`) now asks for human
> confirmation before any close-as-completed. Until verification lands, PRs
> say **`Refs #N`, not `Closes #N`** — so a merge can't auto-close an
> unverified issue — and every reply addresses the reporter's **latest
> follow-up**, never a re-post of the original fix description.

**Key takeaways**

- **"Fixed" and "verified" are different states, and only one closes an
  issue.** A merged fix is a hypothesis with good syntax; verification is a
  failing-first test passing *plus* the symptom demonstrably gone — ideally
  in the reporter's own confirmation.
- **When you can't verify, leave it open.** A concise status comment — what
  you found, what's blocking — is a better artifact than an optimistic close,
  and it's what the policy requires.
- **Process failures get engineering fixes too.** The same project that pins
  parsers with literal vectors pins its own closing behaviour with a guard
  hook — an "ask" gate, not a hard block, on exactly the risky action.
- **The whole series is one pipeline of gates.** Constants cited, vectors
  pinned, conformance bit-identical, synthetics faithful, captures refereeing,
  regressions failing first, instruments counting — and at the end, a human
  confirming the claim. No stage substitutes for another.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The policy | close = verified claim; unverified stays open with a status comment | `CLAUDE.md` ("Issue-closing policy") |
| The guard | asks a human before any close-as-completed; `not_planned`/`duplicate` pass freely | `.claude/hooks/guard-issue-close.py` |
| PR linking | `Refs #N` until verified, so a merge can't auto-close | `CLAUDE.md`; PR convention |
| The cautionary tale | closed twice unverified; symptom still live | [#764](https://github.com/MattCheramie/GopherTrunk/issues/764) / [#771](https://github.com/MattCheramie/GopherTrunk/issues/771) |
| Follow-up discipline | answer the latest comment, not the original report | `CLAUDE.md` (policy + daily issue review) |
| The reporter loop | env-gated replay tests + verdict lines the reporter runs | [Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }}), [Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }}) |

## In this post

- **A close is a claim** — the policy, stated precisely.
- **The issue that was closed twice** — how #764 became #771, and why.
- **The guard at the gate** — a hook that makes the risky action ask.
- **Refs, not Closes — and answer the latest follow-up** — the small habits
  that carry the policy.
- **The shipping checklist** — Parts 1–13 as one table of gates.

## A close is a claim

Strip the tooling away and closing an issue is a speech act: *the problem you
reported is gone.* Everything in this series has been about not making claims
above their evidence — a constant is cited or it's a
[placeholder]({{ '/blog/solution-postmortem/from-the-issue-tracker-17-placeholder-constants/' | relative_url }});
a decoder is synthetic-green, capture-verified, or on-air-verified and never
more ([Part 10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})).
The close is simply the last claim in the chain, and it gets the same
treatment. GopherTrunk's policy defines the evidence that earns it:

- **A failing-first regression test now passes** — the
  [Part 12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})
  artifact — **and the reporter has confirmed the fix**; or you reproduced
  the original symptom yourself and showed this change resolves it.
- **When you can't verify, leave it open** — post a concise status comment
  saying what you found and what's blocking (a capture, a config, the
  reporter's hardware), rather than closing.
- Closing as `not_planned` or `duplicate` is fine and ungated — those are
  claims about *scope*, not about the problem being gone.

Note what the definition refuses to accept: a merged PR, a green CI run, a
plausible root-cause narrative, or the author's confidence. Every one of
those was present, twice, in the story that created the policy.

## The issue that was closed twice

[#764](https://github.com/MattCheramie/GopherTrunk/issues/764) reported a
decode deficit on high-sample-rate captures. A fix was written, tests were
green, the issue was closed. The symptom persisted; it was reopened, fixed
again, closed again — and the symptom was *still* live, resurfacing as
[#771](https://github.com/MattCheramie/GopherTrunk/issues/771). The
mechanical cause was a twin-path trap this project has since learned to name:
`gophertrunk replay -tune-hz` uses the single-channel
`ccdecoder.Downconverter`, **not** the wideband `DDCBank` — two separate
down-converter paths, so a fix to one does not touch the other, and the #764
"fix" missed the #771 replay symptom entirely.

The deeper cause was epistemic: nobody had watched the symptom die. Both
closes re-posted the fix description as their justification — a claim about
the *code*, when a close is a claim about the *symptom*. When #764 finally
did close for good, it closed the right way: the reporter's own captures were
replayed, the deficit was shown to be **baked into the samples** (the same
~9.5 dB through an independent resampler and the proven decode path), and
`TestDownconverterSNRInvariantAcrossRate` pinned the conclusion. The
verification didn't just permit the close — it produced the actual root
cause, which the two premature closes had gotten wrong.

## The guard at the gate

Policies decay; gates don't. So the policy got what every other rule in this
series got — enforcement machinery on exactly the risky action. A PreToolUse
hook intercepts issue writes before they execute, lets everything harmless
through, and turns a close-as-completed into a question a human must answer:

```python
# .claude/hooks/guard-issue-close.py (shape)
# Stops a GitHub issue being closed as *completed* ("fixed") without a
# human in the loop — because #764 was closed twice with a comment that
# merely repeated the fix description while the symptom was still live.
is_close_completed = (
    ti.get("method") == "update"
    and ti.get("state") == "closed"
    and ti.get("state_reason") in (None, "", "completed")
)
if not is_close_completed:
    return 0  # allow: comments, labels, reopens, not_planned, duplicate
# Otherwise: emit an "ask" decision — the close proceeds only once a human
# confirms the fix is genuinely verified.
```

Three design choices are worth stealing. It gates **only** the dangerous
verb — closing as completed — so it never trains anyone to click through it
on routine work. It **asks** rather than blocks: the reason text restates the
verification bar and offers the alternative (cancel, post a status comment,
leave it open), keeping the human's judgment in the loop rather than
replacing it. And it fails open — always exits 0 on malformed input — because
*a guard must never become a hard failure that blocks work*, or it gets
removed. The parallel to Part 13 is exact: this is a WARN with a human as the
persistence gate.

## Refs, not Closes — and answer the latest follow-up

Two small habits carry the policy day to day. First, **PRs say `Refs #764`,
not `Closes #764`, until the fix is verified** — because GitHub auto-closes
on merge, and auto-close is precisely the unverified claim the policy
forbids, made by a robot on your behalf. `Closes` is earned at the same
moment the close itself is.

Second, **address the latest follow-up, not the original report**. Both bad
#764 closes re-posted the initial fix description — a reply to a comment
nobody had just made. The daily issue review bakes the discipline in: each
pass surfaces issues *updated since the prior day* and distinguishes genuine
new reporter responses from maintainer replies, so the thing being answered
is always the newest evidence. A reporter's follow-up is field data —
[Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})'s
most valuable input — and answering an old comment instead is how a thread
ships the same wrong claim twice.

## The shipping checklist

Fourteen parts, one pipeline. Each stage produces an artifact, and each
artifact has a gate it must pass before the next claim is allowed:

| Stage | Artifact | Gate |
|---|---|---|
| Read the spec ([1]({{ '/blog/deep-dives/from-spec-to-shipping-01-reading-a-radio-standard/' | relative_url }})) | named constants: sync, CRC, offsets | every constant carries its section citation |
| Choose references ([2]({{ '/blog/deep-dives/from-spec-to-shipping-02-choosing-references/' | relative_url }})) | a reference stable with known authority | proven on air beats popular |
| Pin the parser ([3]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }})) | literal byte vectors | cross-checked against an independent decoder |
| Conform ([4]({{ '/blog/deep-dives/from-spec-to-shipping-04-conformance-harness/' | relative_url }})) | skip-guarded reference harness | bit-identical output, at two layers |
| Stay clean ([5]({{ '/blog/deep-dives/from-spec-to-shipping-05-clean-room-rules/' | relative_url }})) | oracle comparisons, license hygiene | outputs compared; source never translated |
| Referee disputes ([6]({{ '/blog/deep-dives/from-spec-to-shipping-06-when-references-disagree/' | relative_url }}), [11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})) | a measurement on a real capture | the right answer wins by a wide margin — or no pick |
| Harden the tests ([7]({{ '/blog/deep-dives/from-spec-to-shipping-07-tests-that-can-disagree/' | relative_url }})) | unconditional encodes, strict fakes, faithful synthetics | the test is *able* to disagree with you |
| Pin the wire ([8]({{ '/blog/deep-dives/from-spec-to-shipping-08-smartnet-rebuild/' | relative_url }}), [9]({{ '/blog/deep-dives/from-spec-to-shipping-09-wire-protocols-without-schemas/' | relative_url }})) | framing and opcodes from upstream literals | reference-literal tests catch constant drift |
| Stage the claim ([10]({{ '/blog/deep-dives/from-spec-to-shipping-10-the-on-air-gate/' | relative_url }})) | a status: synthetic-green / capture-verified / on-air-verified | no claim above its evidence |
| Fix bugs ([12]({{ '/blog/deep-dives/from-spec-to-shipping-12-failing-first/' | relative_url }})) | a failing-first regression | fails without the fix, passes with it |
| Instrument ([13]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})) | branch-complete counters, persistent WARNs | designed for the *next* investigation |
| Close (this part) | the verified claim | regression passes **and** the reporter confirms — a human says so |

<figure class="lab-figure">
<svg viewBox="0 0 680 200" width="680" height="200" role="img" aria-label="The shipping pipeline as five gates in sequence: spec-cited constants, reference-pinned parser, faithful synthetics green, capture verified, on-air verified with reporter confirmation — then a human-confirmation diamond guarding the final closed state. A side note marks not-planned and duplicate closes as bypassing the guard, and a label under each gate names the claim allowed at that stage.">
  <rect x="14" y="52" width="102" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="65" y="69" text-anchor="middle" fill="currentColor" font-size="9">spec-cited</text>
  <text x="65" y="82" text-anchor="middle" fill="currentColor" font-size="9">constants</text>
  <line x1="116" y1="72" x2="136" y2="72" stroke="currentColor"/><polygon points="134,68 142,72 134,76" fill="currentColor"/>
  <rect x="144" y="52" width="102" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="195" y="69" text-anchor="middle" fill="currentColor" font-size="9">reference-pinned</text>
  <text x="195" y="82" text-anchor="middle" fill="currentColor" font-size="9">parser + vectors</text>
  <line x1="246" y1="72" x2="266" y2="72" stroke="currentColor"/><polygon points="264,68 272,72 264,76" fill="currentColor"/>
  <rect x="274" y="52" width="102" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="325" y="69" text-anchor="middle" fill="currentColor" font-size="9">faithful synthetics</text>
  <text x="325" y="82" text-anchor="middle" fill="currentColor" font-size="9">green</text>
  <line x1="376" y1="72" x2="396" y2="72" stroke="currentColor"/><polygon points="394,68 402,72 394,76" fill="currentColor"/>
  <rect x="404" y="52" width="102" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="455" y="69" text-anchor="middle" fill="var(--accent)" font-size="9">capture-verified</text>
  <text x="455" y="82" text-anchor="middle" fill="var(--accent)" font-size="9">(replay A/B)</text>
  <line x1="506" y1="72" x2="526" y2="72" stroke="currentColor"/><polygon points="524,68 532,72 524,76" fill="currentColor"/>
  <rect x="534" y="52" width="130" height="40" rx="6" fill="none" stroke="var(--accent)" stroke-width="2"/>
  <text x="599" y="69" text-anchor="middle" fill="var(--accent)" font-size="9">on-air-verified +</text>
  <text x="599" y="82" text-anchor="middle" fill="var(--accent)" font-size="9">reporter confirms</text>
  <text x="65" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">claim: "transcribed"</text>
  <text x="195" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">claim: "matches refs"</text>
  <text x="325" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">claim: "should work"</text>
  <text x="455" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">claim: "works on this capture"</text>
  <text x="599" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="8">claim: "the problem is gone"</text>
  <line x1="599" y1="92" x2="599" y2="128" stroke="var(--accent)"/><polygon points="595,126 599,134 603,126" fill="var(--accent)"/>
  <path d="M 599 134 L 641 154 L 599 174 L 557 154 Z" fill="none" stroke="var(--accent)"/>
  <text x="599" y="151" text-anchor="middle" fill="var(--accent)" font-size="8">guard hook:</text>
  <text x="599" y="162" text-anchor="middle" fill="var(--accent)" font-size="8">human confirms</text>
  <line x1="557" y1="154" x2="500" y2="154" stroke="currentColor"/><polygon points="502,150 494,154 502,158" fill="currentColor"/>
  <rect x="404" y="140" width="90" height="28" rx="6" fill="none" stroke="currentColor"/>
  <text x="449" y="158" text-anchor="middle" fill="currentColor" font-size="9">CLOSED</text>
  <text x="290" y="158" text-anchor="middle" fill="var(--fg-muted)" font-size="8">not_planned / duplicate: ungated (claims about scope, not the symptom)</text>
</svg>
<figcaption>Each gate licenses exactly one claim — and the last claim, "the problem is gone," passes through a human before it becomes a closed issue.</figcaption>
</figure>

The through-line, one last time: the running villain of this series was the
test that passes because both sides share the same assumption, and every gate
above is a way of importing a fact from *outside* the loop — a spec section,
an independent decoder, a reference codec, a capture, an operator, and
finally a reporter saying "it works now." Verification is not a phase at the
end. It is the habit of never letting your own assumption referee itself.

## Where to go from here

This series taught the method forward; its source material runs the other
way. [From the Issue Tracker]({{ '/blog/series/from-the-issue-tracker/' | relative_url }})
tells the same lessons as postmortems — the fabricated framing, the
self-consistent traps, the two-pipelines drift — with the scars visible.
[P25 End to End Part 13]({{ '/blog/deep-dives/p25-end-to-end-13-testing-p25/' | relative_url }})
applies this discipline to P25's twin decode paths, literal vectors to
capture metrics. The
[testing learn module]({{ '/learn/testing/' | relative_url }}) covers the
underlying craft — including capture-gated verification — from first
principles. And if you'd rather *run* the thing all this rigor produced, the
[Operator's Cookbook]({{ '/blog/series/operator-cookbook/' | relative_url }})
starts from a config file instead of a spec. Best of all: contribute a
capture. Half the open items in this project are blocked on evidence, not
effort, and [Part 11]({{ '/blog/deep-dives/from-spec-to-shipping-11-capture-driven-development/' | relative_url }})
shows exactly what to record.

## FAQ

**Why not close an issue as soon as the fix is merged?**
Because merging proves the code changed, not that the symptom died — and
those came apart, twice, on the same issue. The #764 fix was merged, green,
and wrong about the root cause; only replaying the reporter's captures
produced the real one. `Refs` on the PR plus a verified close later costs one
extra click and prevents shipping a claim you can't back.

**What if the reporter never responds to confirm?**
The policy's second branch covers it: reproduce the original symptom yourself
and show the change resolves it — a capture replayed through a failing-first
harness is a reporter-independent verification. If you can do neither, the
issue stays open with a status comment; an open issue that's honest beats a
closed one that's optimistic.

**Isn't a guard hook overkill for a process rule?**
It's proportionate to the failure it prevents: the same mistake made twice on
one issue by people who knew better. Rules that live in documents decay under
deadline pressure; the hook re-states the bar at the exact moment of the
risky action, gates nothing else, and asks rather than blocks — the cheapest
intervention that actually changes behaviour.

**Does the gate apply to closing as not_planned or duplicate?**
No — those pass through freely, by design. They claim "we won't do this" or
"this is tracked elsewhere," which requires judgment but not verification.
Only `completed` claims the reported problem is gone, and only that claim
needs evidence of the symptom's death.

**What's the single habit to adopt from this series?**
Before any confident claim — this parser is correct, this fix works, this
issue is done — ask *what fact from outside my own loop says so?* A cited
section, an independent decoder, a capture, a reporter. If the answer is
"my test agrees with my code," you have the series villain, not evidence.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Instruments, Not Logs]({{ '/blog/deep-dives/from-spec-to-shipping-13-instruments-not-logs/' | relative_url }})
· [Back to the series index]({{ '/blog/series/from-spec-to-shipping/' | relative_url }})
