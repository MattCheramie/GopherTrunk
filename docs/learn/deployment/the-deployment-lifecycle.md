---
slug: the-deployment-lifecycle
title: The deployment lifecycle
description: "Build, test, release, deploy, operate — the repeating loop every change moves through, and how the rest of this module maps onto each stage."
keywords: deployment lifecycle, build test release deploy operate, software delivery, release cycle, deployment stages, continuous delivery loop
level: beginner
status: full
prereq:
  - what-is-deployment
faq:
  - q: What are the stages of the deployment lifecycle?
    a: "A common way to name them is build, test, release, deploy, and operate. Build turns source into an artifact, test proves it works, release packages and versions it, deploy puts it where it runs, and operate keeps it healthy and feeds problems back into the next build. It's a loop, not a line — every change goes around it again."
  - q: Is the deployment lifecycle the same as CI/CD?
    a: "CI/CD is the automation that runs the build, test, and release stages for you on every change. The lifecycle is the bigger picture — it also includes deploying to a server and operating it in production. CI/CD automates the left-hand stages; you still design the deploy and operate ones."
---

# The deployment lifecycle

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Shipping software is a repeating **loop**, not a one-off event. A change moves through
**build** (source to artifact), **test** (prove it works), **release** (package and
version), **deploy** (put it where it runs), and **operate** (keep it healthy) — then
what you learn operating feeds the next build. Every lesson in this module lives at one
of these stages.
</div>

[What is deployment?](/learn/deployment/what-is-deployment/) framed the problem: getting
code from your machine to a running service other people use. This lesson gives you the
map — the five stages a change repeats forever — so the rest of the module has a place
to hang.

## Why think in a lifecycle at all?

A single deploy feels like a straight line: write code, ship it, done. But real software
is never done. You fix a bug, add a feature, patch a security hole — and each change
retraces the same path. Naming that path as a **cycle** does two things: it makes every
step deliberate instead of ad-hoc, and it makes the **operate** stage feed back into the
next **build** so you actually learn from production.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 120" role="img" aria-label="A loop of five stages: build, test, release, deploy, operate, with operate feeding back to build." xmlns="http://www.w3.org/2000/svg">
  <g font-size="9" fill="currentColor" text-anchor="middle">
  <rect x="8" y="45" width="72" height="30" rx="4" fill="none" stroke="currentColor"/><text x="44" y="64">build</text>
  <rect x="118" y="45" width="72" height="30" rx="4" fill="none" stroke="currentColor"/><text x="154" y="64">test</text>
  <rect x="228" y="45" width="72" height="30" rx="4" fill="none" stroke="currentColor"/><text x="264" y="64">release</text>
  <rect x="338" y="45" width="72" height="30" rx="4" fill="none" stroke="currentColor"/><text x="374" y="64">deploy</text>
  <rect x="448" y="45" width="64" height="30" rx="4" fill="none" stroke="currentColor" stroke-width="1.7"/><text x="480" y="64">operate</text>
  </g>
  <g stroke="currentColor" fill="none"><line x1="80" y1="60" x2="118" y2="60"/><line x1="190" y1="60" x2="228" y2="60"/><line x1="300" y1="60" x2="338" y2="60"/><line x1="410" y1="60" x2="448" y2="60"/>
  <path d="M480 45 C 480 12, 44 12, 44 45" stroke-dasharray="4 3"/></g>
  <text x="262" y="20" text-anchor="middle" font-size="8.5" fill="currentColor">what you learn operating feeds the next build</text>
</svg>
<figcaption>The loop: each change goes build → test → release → deploy → operate, and operating feeds the next change.</figcaption>
</figure>

## What happens at each stage?

Each stage has a clear job and a matching lesson later in this module:

| Stage | The job | Where this module covers it |
|-------|---------|------------------------------|
| Build | Turn source into a single, reproducible artifact | [Build artifacts & versioning](/learn/deployment/build-artifacts-and-versioning/), [Writing a Dockerfile](/learn/deployment/writing-a-dockerfile/) |
| Test | Prove the artifact works before anyone relies on it | [CI/CD pipelines](/learn/deployment/ci-cd-pipelines/) |
| Release | Version, package, and publish it so it can be pulled | [Images & registries](/learn/deployment/images-and-registries/), [Release strategies](/learn/deployment/release-strategies/) |
| Deploy | Put it where it runs and start it | [Cloud & VPS](/learn/deployment/deploying-to-cloud-and-vps/), [Zero-downtime deploys](/learn/deployment/zero-downtime-deploys/) |
| Operate | Keep it healthy, watch it, fix it, feed back | [Observability](/learn/deployment/observability/), [Incident response](/learn/deployment/incident-response/) |

## Where does automation fit?

The first three stages — build, test, release — are exactly what a
[CI/CD pipeline](/learn/deployment/ci-cd-pipelines/) automates. A push or a version tag
triggers a fresh machine to build the artifact, run the tests, and publish the result,
identically every time. GopherTrunk does this with GitHub Actions: every pull request is
built and tested, and a version tag builds and publishes the downloadable release
binaries. Automating the left of the loop is what makes going around it fast and boring —
which is what you want.

## Why does operate loop back to build?

The last stage is the one beginners skip. Once a version is live you have to **operate**
it: read its logs, watch its metrics, respond when it breaks. Everything you learn there
— a slow endpoint, a crash under load, a confusing config — becomes a bug report or a
feature request that starts the loop again with a new **build**. A deployment you can't
observe is a loop with the feedback wire cut; you'll keep shipping the same problems.

<div class="knowledge-check" data-quiz data-correct-msg="Right — operate feeds what you learn back into the next build, closing the loop." markdown="0">
  <p class="knowledge-check__q">Quick check: what makes the deployment lifecycle a loop rather than a line?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Every deploy must be undone before the next one</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">What you learn operating a release feeds the next build</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The stages can run in any order</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Deployment is a repeating **loop**: build, test, release, deploy, operate.
- **Build** makes an artifact, **test** proves it, **release** versions it, **deploy**
  runs it, **operate** keeps it healthy.
- **CI/CD** automates the build-test-release stages so going around the loop is fast.
- **Operate** feeds what you learn back into the next **build** — that feedback is the
  point of the loop.

Next up: the environments a build runs in, and keeping config out of code.
