---
slug: github-actions
title: CI/CD with GitHub Actions
description: Automate testing and deployment with GitHub Actions — workflow files, on/jobs/steps, actions/checkout, a real CI workflow, matrix builds, secrets, and PR status checks.
keywords: github actions, ci cd, workflow file, actions/checkout, runs-on, matrix build, github secrets, workflow_dispatch, pull request status checks, continuous integration
level: advanced
status: full
prereq:
  - pull-requests
faq:
  - q: Where do GitHub Actions workflow files go?
    a: In a directory called .github/workflows/ at the root of your repository, with a .yml or .yaml extension — for example .github/workflows/ci.yml. GitHub automatically discovers every workflow file in that folder and runs the ones whose trigger events occur. You can have many workflow files; each is independent and has its own triggers and jobs.
  - q: What is the difference between uses and run in a step?
    a: A step with uses runs a prebuilt action — a packaged, reusable unit such as actions/checkout@v4 that clones your code — optionally configured with a with block. A step with run executes shell commands directly on the runner, like "npm install" or "pytest". A job's steps mix both freely; uses brings in shared building blocks and run does your project-specific work.
  - q: How do secrets work in GitHub Actions?
    a: You store sensitive values (API keys, tokens) under the repository or organisation Settings → Secrets and variables → Actions. In a workflow you read them through the secrets context, for example ${{ secrets.NPM_TOKEN }}. GitHub masks secret values in logs and never exposes them to workflows triggered by forked pull requests by default, which limits the blast radius if a fork is malicious.
  - q: How does a workflow become a required status check on a pull request?
    a: When a workflow runs on the pull_request event, each of its jobs reports a pass/fail status check on the PR. In branch protection rules (or a ruleset) you mark specific checks as required, so the Merge button stays disabled until those jobs pass. This is how Actions and branch protection combine to stop broken code from reaching the default branch.
---
{% raw %}
# CI/CD with GitHub Actions

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**GitHub Actions** runs automated jobs in response to events in your repo — the
foundation of **CI/CD** (continuous integration and delivery). A **workflow** is a
YAML file in **`.github/workflows/`** with three core parts: **`on:`** (what triggers
it), **`jobs:`** (units that run on a **`runs-on:`** machine), and **`steps:`** (each
either `uses:` a prebuilt action like **`actions/checkout`** or `run:`s shell
commands). A workflow on the **`pull_request`** event becomes a **status check**, which
[branch protection](/learn/git/branch-protection/) can require before merging. **Matrix**
builds run the same job across many versions, and **`${{ secrets.* }}`** injects
credentials safely.
</div>

Once you're opening [pull requests](/learn/git/pull-requests/), the obvious next wish
is "run the tests automatically on every PR so I don't merge something broken." That's
**continuous integration**, and GitHub Actions is the built-in tool for it. This lesson
breaks down a workflow file part by part, then builds a real one.

## What CI/CD is, and why

**Continuous integration (CI)** means automatically building and testing your code
every time it changes, so problems surface in minutes instead of after a release.
**Continuous delivery/deployment (CD)** extends that to automatically shipping the
code that passes — to a staging server, a package registry, or production.

The payoff: humans stop being the thing that remembers to run the tests. Every push
and every PR gets the same checks, results are visible to everyone, and a green build
becomes a precondition for merging. GitHub Actions provides hosted machines
(**runners**) and a YAML format to describe what should run.

## The anatomy of a workflow file

A workflow lives in `.github/workflows/` and has a predictable shape. Read it
top-down:

```yaml
name: CI                      # shown in the Actions tab
on: [push]                    # what triggers this workflow
jobs:                         # one or more jobs
  build:                      # a job id
    runs-on: ubuntu-latest    # the machine it runs on
    steps:                    # ordered steps in the job
      - uses: actions/checkout@v4   # a prebuilt action
      - run: echo "Hello, Actions"  # a shell command
```

The four nouns to know:

| Key | Meaning |
|-----|---------|
| `on:` | The **events** that start the workflow |
| `jobs:` | Named units of work; by default they run in **parallel** |
| `runs-on:` | The **runner** OS image (`ubuntu-latest`, `windows-latest`, `macos-latest`) |
| `steps:` | The ordered actions inside a job |

## Events: the `on:` trigger

`on:` decides when a workflow runs. The common events:

```yaml
on:
  push:
    branches: [main]          # only pushes to main
  pull_request:               # every PR opened or updated
  workflow_dispatch:          # a manual "Run workflow" button
  schedule:
    - cron: "0 6 * * 1"       # 06:00 UTC every Monday
```

- **`push`** — code is pushed (optionally scoped to branches or paths).
- **`pull_request`** — a PR is opened or updated; this is the event that produces
  **status checks**.
- **`workflow_dispatch`** — adds a manual run button in the Actions tab.
- **`schedule`** — runs on a cron timetable, for nightly jobs and the like.

## Steps: `uses:` versus `run:`

Inside a job, each step is one of two things. A **`uses:`** step runs a packaged,
reusable **action** — and the place to find them is the **GitHub Marketplace**, which
catalogues thousands of community and official actions. The near-universal first step
is `actions/checkout`, which clones your repository onto the runner (without it, the
runner has no code):

```yaml
- uses: actions/checkout@v4
- uses: actions/setup-node@v4
  with:
    node-version: "20"        # configure an action with `with:`
```

A **`run:`** step executes shell commands directly:

```yaml
- run: npm ci
- run: npm test
```

Pin actions to a version (`@v4`) so a future change to the action can't silently break
your build.

## A real workflow: test every pull request

Putting it together — a workflow that installs dependencies and runs the test suite on
every PR and every push to `main`:

```yaml
name: Tests
on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: npm
      - run: npm ci
      - run: npm test
```

Push this file and open a PR: GitHub runs the `test` job and reports its result right
on the PR's **Checks** section. A red X blocks confidence; a green check says the suite
passed against the proposed change.

## Matrix builds, secrets, and expressions

A **matrix** runs the same job many times across a set of values — perfect for testing
multiple language versions or operating systems in parallel:

```yaml
jobs:
  test:
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
        node: ["18", "20", "22"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: ${{ matrix.node }}
      - run: npm ci && npm test
```

That single job definition expands into six runs (2 operating systems × 3 Node
versions). The `${{ ... }}` syntax is an **expression**: it reads values from contexts
like `matrix`, `github`, and `secrets`. **Secrets** are encrypted credentials stored
under **Settings → Secrets and variables → Actions** and read the same way — never
hard-code a token:

```yaml
      - run: npm publish
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

GitHub masks secret values in the logs and withholds them from forked-PR runs by
default.

## From workflow result to PR status check

This is where Actions meets the merge gate. Every job that runs on the `pull_request`
event publishes a **status check** — a named pass/fail result attached to the PR. On
its own, a failing check just shows a red X but still lets you merge. The teeth come
from [branch protection](/learn/git/branch-protection/): mark the `test` check as
**required**, and GitHub disables the Merge button until it passes. Automated tests
plus a required check is the mechanism that keeps a broken commit out of `main`.
(Unsure about runners, jobs, or contexts? The [glossary](/learn/git/glossary/) has
short definitions.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — actions/checkout clones your repo onto the runner." markdown="0">
  <p class="knowledge-check__q">Quick check: why does almost every job start with <code>uses: actions/checkout@v4</code>?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It installs Node.js on the runner</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It clones your repository's code onto the runner</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It marks the workflow as a required status check</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **GitHub Actions** runs automated **CI/CD** jobs from YAML files in
  **`.github/workflows/`**.
- A workflow has **`on:`** (events), **`jobs:`** (`runs-on:` a runner), and
  **`steps:`** (`uses:` an action or `run:` a command).
- **`actions/checkout`** clones your code; the **Marketplace** supplies reusable
  actions.
- **`pull_request`** workflows produce **status checks** on the PR.
- **`matrix`** fans a job out across versions; **`${{ secrets.* }}`** injects
  credentials safely.
- Required checks plus [branch protection](/learn/git/branch-protection/) keep broken
  code out of `main`.

Next up: tagging and shipping versions with [tags, releases & semantic versioning](/learn/git/releases-and-tags/).
{% endraw %}
