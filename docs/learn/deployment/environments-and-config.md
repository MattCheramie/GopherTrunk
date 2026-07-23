---
slug: environments-and-config
title: Environments & configuration
description: Development, staging, and production, why the same build must run in all of them, and how environment variables and config files keep settings out of code.
keywords: environments, dev staging production, configuration, environment variables, config file, twelve factor, externalized config, deployment config
level: beginner
status: full
prereq:
  - what-is-deployment
faq:
  - q: What are dev, staging, and production environments?
    a: They are separate places the same software runs. Development is your machine, where you build and experiment. Staging is a production-like environment for final testing. Production is the real thing your users actually touch. Keeping them separate lets you test changes safely before they reach users.
  - q: Should configuration go in the code or outside it?
    a: Outside. Anything that differs between environments — ports, file paths, database URLs, feature flags — should come from configuration the program reads at startup, not values baked into the build. That way one identical build runs everywhere, and you change behaviour by changing config, not by rebuilding.
---

# Environments & configuration

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Software runs in several **environments** — **development**, **staging**,
**production** — and the *same build* should run in all of them. What differs between
them lives in **configuration**, supplied by **environment variables** or a **config
file** the program reads at startup — never hard-coded. Build once, configure per
environment.
</div>

The same program has to behave slightly differently depending on where it runs. This
lesson is about handling that difference cleanly, without rebuilding.

## The environments

Most software passes through a few standard environments on its way to users:

| Environment | What it's for | Who touches it |
|-------------|---------------|----------------|
| **Development** | building and experimenting | you |
| **Staging** | final, production-like testing | your team |
| **Production** | the real service | your users |

They differ in the obvious ways — production has real data, real traffic, real
consequences — so you test in dev and staging first. But the *code* should be identical
across them. What changes is configuration.

## Build once, configure per environment

A cardinal rule of deployment: **the same artifact runs everywhere; only its
configuration changes.** If dev and production run different *builds*, you can never be
sure you tested what you shipped. So anything environment-specific gets pulled *out* of
the code:

- ports and network addresses
- file and directory paths
- database connection strings
- feature flags and log levels
- credentials (with extra care — see
  [secrets](/learn/deployment/secrets-and-configuration/))

## Two ways to supply config

**Environment variables** are key/value settings the operating system hands the program
— the standard way to inject a single value, especially in containers (they're covered
in the [Linux CLI module](/learn/linux-cli/environment-variables/)):

```bash
export GOPHERTRUNK_CONFIG=/etc/gophertrunk/config.yaml
gophertrunk run
```

**Config files** suit richer, structured settings. GopherTrunk uses a YAML config that
you point it at with a flag, keeping every environment-specific choice in one editable
file rather than in the binary:

```bash
gophertrunk run -config /etc/gophertrunk/config.yaml
```

The two combine well: a config file for the bulk of settings, environment variables to
override a few per environment (and to inject secrets the file shouldn't contain).

## Keep environments honest

Two habits prevent painful surprises:

- **Make staging resemble production.** The closer they are, the more bugs staging
  catches before users do.
- **Never edit production config by hand and forget.** Track config in version control
  (minus secrets) so you know exactly what each environment is running — the same
  [Git](/learn/git/) discipline you use for code.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the same build runs everywhere; configuration is what changes per environment." markdown="0">
  <p class="knowledge-check__q">Quick check: what should differ between your development and production environments?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The compiled build itself</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The configuration, while the build stays identical</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The programming language</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Software runs in **development**, **staging**, and **production** — separate places,
  same build.
- Environment-specific settings live in **configuration**, not in code.
- Supply config via **environment variables** and/or a **config file** read at startup.
- Keep **staging like production** and track config in version control (secrets aside).

Next up: what a build actually produces — artifacts and versioning.
