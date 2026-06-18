---
slug: shipping-and-maintaining
title: Shipping v1 & maintaining solo
description: Cut your first release with done over perfect, write release notes, gather and triage feedback, maintain sustainably, handle burnout and scope creep, and play the long game.
keywords: shipping software, release notes, gathering feedback, triage, sustainable maintenance, burnout, scope creep, saying no, open source community
level: intermediate
status: full
faq:
  - q: "When is my software ready to ship?"
    a: "When the v1 you scoped works and is genuinely useful — not when it is perfect, because it never will be. **Done is better than perfect.** Shipping an imperfect-but-useful v1 and improving it from real feedback beats polishing forever in private. The first release is a starting line, not a final exam; you can and will fix things after it is out."
  - q: "How do I handle feedback and bug reports as a solo maintainer?"
    a: "Give people one clear place to report (issues or discussions), then **triage** rather than react — sort what comes in by impact, fix the things that matter, and politely decline or defer the rest. You cannot do everything, and trying to is how solo projects burn out. A short, honest \"not right now, but noted\" is a complete and acceptable answer."
  - q: "How do solo maintainers avoid burning out long-term?"
    a: "By treating maintenance as a marathon, not a sprint — keeping scope bounded, automating the repetitive work, saying no to requests outside the project's purpose, and giving themselves permission to step back. Burnout comes from unbounded obligation, so the cure is bounded commitment: do what is sustainable, decline the rest, and protect your own energy as the project's most important resource."
---
# Shipping v1 & maintaining solo

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Done over perfect** — ship the useful v1 and improve from real feedback. **Triage, don't drown** — one place for feedback, sort by impact, say no often. **Play the long game** — sustainable pace beats heroic bursts.
</div>

Everything in this module has been building toward one act: putting your software in
front of real people. This final lesson is about crossing that line — cutting the
first release without waiting for a perfection that never arrives — and then doing the
quieter, longer work of keeping the project alive without it consuming you. Shipping is
exhilarating and a little terrifying; maintaining is where most solo projects either
find their footing or quietly fade. We will cover both: how to release, how to listen,
how to triage, and how to keep going for the long haul. And then, since this is the
end of the path, we will send you off to build something real.

## Cut the first release: done over perfect

The hardest part of v1 is not the code — it is *letting go*. There is always one more
thing to fix, one more case to handle, one more polish to apply. At some point you have
to decide the v1 you scoped is good enough and **ship it.**

Repeat the mantra: **done is better than perfect.** Not because quality does not
matter, but because:

- A useful tool in people's hands teaches you more in a week than another month of
  private polishing ever will.
- The "imperfections" you are obsessing over are often not the ones users actually
  care about — you cannot know which is which until it ships.
- A release you can improve beats a masterpiece you never finish.

So when the v1 from your [scope](/learn/intro-software-dev/scoping-a-project/) works,
passes its [guardrails](/learn/intro-software-dev/quality-when-solo/), and is
[packaged](/learn/intro-software-dev/packaging-and-distribution/) so others can run it —
tag it, publish it, and announce it. The first release is a starting line, not a final
exam.

## Write release notes people can read

Every release deserves **release notes** — a short, human-readable summary of what
changed. They are not a chore; they are how users decide whether to upgrade and how
they trust that the project is alive and cared for.

Good notes are brief and grouped:

```text
## v1.1.0
Added
- Log to CSV as well as plain text
Fixed
- Crash when the radio device disconnects mid-capture
Changed
- Faster signal detection on quiet frequencies
```

Write them for a user, not a compiler — "fixed the crash when your dongle unplugs," not
a raw commit list. Paired with the [semantic version](/learn/intro-software-dev/packaging-and-distribution/)
number, good notes turn every release into a small, trustworthy moment of
communication with the people who use your work.

## Gather and triage feedback

Once people use your software, they will tell you things — bugs, requests, confusion,
occasionally praise. Channel that into **one clear place**: GitHub **issues** for bugs
and **discussions** for questions and ideas. A single front door beats feedback
scattered across email, chat and your own memory.

Then **triage** rather than react. You cannot do everything, so sort what arrives:

- **Real bugs that hurt users** — fix these first.
- **Good ideas that fit the project** — note them, schedule the worthwhile ones.
- **Out-of-scope requests** — politely decline or defer; "not right now, but noted" is
  a complete answer.
- **Duplicates and noise** — close kindly, point to the existing thread.

Triage is the skill that keeps feedback from becoming an avalanche. The goal is not to
satisfy every request — it is to keep the project's purpose intact while genuinely
listening.

<div class="knowledge-check" data-quiz data-correct-msg="Right — done over perfect means shipping a genuinely useful v1 and improving it from real feedback, rather than polishing forever." markdown="0">
  <p class="knowledge-check__q">Quick check: what does "done is better than perfect" mean for your first release?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Quality does not matter, so ship anything that compiles</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Ship a genuinely useful v1 now and improve it from real feedback, rather than polishing forever in private</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Never release until every possible feature and edge case is finished</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Maintain sustainably and protect your energy

A project does not end at v1 — that is when maintenance begins, and maintenance is a
**marathon, not a sprint.** Your most precious resource is not time or skill; it is your
own sustained energy, and the whole art is spending it carefully:

- **Bound your scope.** **Scope creep** — saying yes to feature after feature until the
  project sprawls beyond what one person can carry — is the quiet killer. Hold the line
  on what the project is *for*.
- **Learn to say no.** Most requests are reasonable in isolation and unsustainable in
  aggregate. A polite, honest "this isn't where I'm taking the project" protects both
  you and the project's coherence. Saying no is a maintenance skill, not rudeness.
- **Automate the repetitive.** Let CI build and test releases; let bots handle the
  mechanical. Every chore you automate is energy saved for the work only you can do.
- **Permit yourself to step back.** It is fine to slow down, take a break, or say a
  project is in low-maintenance mode. Burnout comes from *unbounded obligation*; the
  cure is **bounded commitment.** A maintainer who paces themselves outlasts one who
  sprints and quits.

This is the discipline the whole module has pointed at: give yourself the structure —
and the limits — a healthy team would. The limits are not a weakness. They are what let
you still be here, and still enjoying it, a year from now.

## Build a small community and play the long game

You do not have to stay solo forever in spirit, even if you write all the code. A
little community makes the long game far easier and far more rewarding:

- **Be welcoming.** A friendly reply to a first-time bug reporter costs a minute and
  earns goodwill that compounds.
- **Write things down.** Good docs and clear issues let users help each other and turn
  some of them into contributors who lighten your load.
- **Celebrate use.** People building things with your software is the whole point —
  notice it, thank them, and let it refuel you.

Over time, a useful, well-maintained tool with a kind maintainer attracts exactly the
help that eases the bus-factor-of-one problem we opened the module with. That is the
long game: not a heroic launch, but a sustainable rhythm of small releases, honest
communication, and steady care — the same pattern that lets a one-person radio project
like GopherTrunk keep serving hobbyists from the [downloads](/downloads.html) page
release after release.

## Recap

- **Done over perfect** — ship the useful v1 and improve from real feedback instead of
  polishing forever.
- **Release notes** — short, human, grouped, paired with a semver number, so users
  trust and understand each update.
- **One door, then triage** — funnel feedback to issues and discussions, then sort by
  impact rather than reacting to everything.
- **Maintain sustainably** — bound your scope, automate the repetitive, and protect
  your energy as the project's key resource.
- **Say no on purpose** — declining out-of-scope work keeps both you and the project
  healthy.
- **Play the long game** — a welcoming maintainer and steady small releases beat a
  heroic burst that burns out.

You have reached the end of the path. You started at "what is software?" and arrived
here with a stack you can maintain, a workflow that keeps you fast, a way to keep
quality high alone, and a strategy to package, ship and sustain real software. That is
a complete solo developer toolkit — congratulations on finishing. Keep the
[glossary](/learn/intro-software-dev/glossary/) handy for any term you want to revisit,
and then do the only thing that turns all of this into skill: go build something real,
scope it small, and ship it.
