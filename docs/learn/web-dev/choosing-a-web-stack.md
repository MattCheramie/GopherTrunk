---
slug: choosing-a-web-stack
title: Choosing your web stack
description: A practical framework for the endless "what should I use?" question — matching a front-end approach, back-end language, database, and hosting to the project actually in front of you, instead of chasing whatever's trending.
keywords: web stack, tech stack, choosing a framework, front-end choice, back-end language, database choice, hosting, boring technology, project requirements, full-stack
level: intermediate
status: full
prereq:
  - anatomy-of-a-web-app
faq:
  - q: "Is there a single best web stack?"
    a: "No — and anyone who says otherwise is selling something. The best stack is the one that fits *your* project's requirements, your team's existing skills, and how the app needs to run. A tiny content site and a real-time collaborative app have almost nothing in common in their ideal tooling. The skill isn't knowing the 'right' stack; it's matching tools to the job in front of you."
  - q: "Should I always use the newest, most popular framework?"
    a: "Usually not. Popularity is a weak signal and novelty is a real cost — new tools have fewer answered questions, more churn, and fewer people who know them. **Boring, proven technology** you or your team already understand ships faster and breaks less. Reach for something new when the project genuinely needs what it offers, not because it's trending."
  - q: "What matters most when choosing?"
    a: "Your **team's existing skills**, more often than not. A stack your team knows well beats a theoretically better one they'd have to learn under deadline. After that: the project's real requirements (does it even need a heavy front-end framework?), how it must run and scale, and the size of the ecosystem you can lean on when you're stuck."
---

# Choosing your web stack

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
There is **no single best stack** — only the one that fits *this* project. Choose by
**requirements** (what does the app actually need to do?), your **team's existing
skills** (the biggest real-world factor), and how it must **run and scale**. Prefer
**boring, proven technology** you understand over whatever is trending; novelty is a
cost, not a feature. A stack is a **front-end approach**, a **back-end language**, a
**database**, and **hosting** — pick each to match the job, not a hype cycle. This
lesson pulls together the whole [module](/learn/web-dev/anatomy-of-a-web-app/) into a
decision.
</div>

Every developer eventually faces the paralysing question: *what should I build this
with?* The internet answers with a thousand contradictory opinions and a new
framework every month. This lesson replaces that noise with a durable way to decide —
one that outlives any specific tool, because it's about matching technology to a
project rather than memorising a "right answer" that changes yearly.

## There is no best stack

The single most freeing thing to accept: **the best stack depends entirely on the
project.** A marketing site, an internal dashboard, a real-time app, and a
high-traffic API each have a different ideal — and a choice that's perfect for one is
wasteful or inadequate for another. The [anatomy of a web
app](/learn/web-dev/anatomy-of-a-web-app/) laid out the pieces; choosing a stack is
deciding *how* to fill each one for your specific case. So the question is never
"what's the best framework?" but "what does *this* project need?"

## Start from requirements

Good choices start from the work, not the tools. A few questions cut through most of
it:

- **How dynamic is it?** A mostly-static content site may need no
  [front-end framework](/learn/web-dev/frontend-frameworks/) at all — a
  [static generator](/learn/web-dev/templating-and-static-sites/) is simpler, faster,
  and cheaper. A highly interactive app justifies a heavier front end.
- **What's the rendering need?** Public and SEO-sensitive leans
  [server-rendered or static](/learn/web-dev/ssr-spa-static/); an app-like tool behind
  a login can be a client-rendered SPA.
- **What's the data shape?** Structured, relational data with clear relationships
  suits a relational [database](/learn/databases/); other shapes may not. Let the data
  drive the store, not fashion.
- **Real-time?** If live updates are core, you're committing to
  [WebSockets or similar](/learn/web-dev/websockets-and-realtime/) and a back end that
  supports them well.

Answering these usually narrows the field far more than any framework comparison
does.

## Weigh the real factors

With requirements in hand, weigh the factors that actually predict success:

- **Your team's existing skills.** This dominates. A stack your team knows ships
  faster and breaks less than a "better" one learned under deadline. A language you're
  fluent in is worth more than a marginally superior one you're not.
- **Ecosystem and maturity.** A large, mature ecosystem means libraries for the boring
  parts, answered questions when you're stuck, and people you can hire. A tiny or
  churning ecosystem means you build and debug more yourself.
- **How it runs and scales.** Match the [deployment](/learn/web-dev/deploying-a-web-app/)
  and scaling story to reality — most projects need far less than they fear, so don't
  architect for a scale you don't have.
- **Cost and operational burden.** Someone has to run this. A simpler stack with fewer
  moving parts is less to [monitor](/learn/web-dev/monitoring-and-analytics/), secure,
  and keep alive.

## Prefer boring technology

A principle that pays off repeatedly: **prefer boring, proven technology.** Novelty
has a real, often-hidden cost — fewer answered questions, more breaking changes, fewer
people who know it, and a higher chance the tool is abandoned in two years. Mature,
"boring" tools are boring precisely *because* they work: their sharp edges are known
and documented. Choose something new when the project genuinely needs what it uniquely
offers — not because it's trending or looks good on a résumé. The goal is a working,
maintainable product, and unexciting tools ship those more reliably than exciting ones.

## Putting it together

A stack is really four coupled choices — a front-end approach, a back-end language, a
database, and hosting — and they interact. The move is to make them **coherently**:
start from requirements, lean on what your team knows, favour proven pieces, and keep
the whole thing no more complex than the project demands. GopherTrunk's own dashboard,
the [next lesson's](/learn/web-dev/gophertrunk-web-dashboard/) worked example, is a
small, deliberate stack chosen exactly this way — Go on the back end because the whole
project is Go, a light front end because the job is a live table of calls, and no more
than that.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the best stack is the one that fits the project's requirements and your team, not whatever is newest." markdown="0">
  <p class="knowledge-check__q">Quick check: what's the soundest basis for choosing a web stack?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Whichever framework is newest and trending right now</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The project's real requirements and your team's existing skills</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Whatever has the most stars on its repository this week</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- There is **no single best stack** — the right one depends on the project in front of
  you, so ask *what does this need?*, not *what's best?*
- **Start from requirements**: how dynamic, what rendering, what data shape, and
  whether real-time is core — these narrow the field fast.
- Weigh the factors that predict success: **your team's existing skills** (the
  dominant one), ecosystem maturity, how it runs and scales, and operational cost.
- **Prefer boring, proven technology**; novelty is a hidden cost, so reach for the new
  only when the project genuinely needs it.
- A stack is four coupled choices — front end, back end, database, hosting — made
  **coherently** and no more complex than the job requires.

Next up: [the GopherTrunk web dashboard](/learn/web-dev/gophertrunk-web-dashboard/).
