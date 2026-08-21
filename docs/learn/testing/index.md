---
layout: learn-hub
learn_module: testing
permalink: /learn/testing/
title: Learn Testing & Software Quality — from newbie to expert
description: A free, structured module on software testing and quality — why software breaks, unit and integration testing in Go, regression tests that fail first, linters, CI, methodical debugging, and the capture-gated verification discipline GopherTrunk itself runs on.
keywords: learn software testing, unit testing go, go test tutorial, regression test, test pyramid, code coverage, continuous integration, flaky tests, debugging, git bisect, root cause analysis, failing-first test
---

Every module on this site teaches you to *build* something. This one teaches you to
know whether what you built actually **works** — and to keep it working while it
changes. Testing is the difference between "it seemed fine when I ran it" and "I can
show you it's correct, and I'll know within minutes if it ever stops being correct."
That skill compounds: it makes you faster, not slower, because you spend your time
building instead of re-breaking and re-fixing the same things.

**Who this is for.** Anyone who writes code — or wants to — and has felt the sting of
something breaking that used to work. It pairs naturally with the
[Programming in Go]({{ '/learn/programming-go/' | relative_url }}) module (the examples
here are Go) and the [Git &amp; GitHub]({{ '/learn/git/' | relative_url }}) module
(reviews, checks, and bisecting all live there too), but the first units assume
nothing beyond basic programming. By the end you'll be reading and writing tests the
way a working engineer does.

**How the path works.** Six units climb from *why* to *how* to *for real*. The first
covers **why software breaks** and what quality thinking looks like. The second and
third teach the tests themselves — **unit tests** in Go, then **integration,
regression, and generative testing**. The fourth is the **tooling** that guards a
codebase between releases: linters, formatters, review, and CI. The fifth is
**debugging** — reproducing, reading evidence, bisecting, and finding root causes.
The last unit applies all of it to [GopherTrunk]({{ '/' | relative_url }}) itself:
replay tests that decode recorded radio captures, the traps a self-consistent test
can hide, and the failing-first discipline under which no fix lands without a test
that failed before it. Mark lessons complete as you go — your progress is saved in
your browser. New here?
**[Start with lesson 1: Why does software break?](/learn/testing/why-software-breaks/)**
