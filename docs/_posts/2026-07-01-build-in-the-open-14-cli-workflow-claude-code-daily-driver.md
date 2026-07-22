---
title: "Build in the Open, Part 14: The CLI Workflow & Claude Code as Your Daily Driver"
description: Tie the whole idea-to-release journey into one repeatable loop using the gh CLI and Claude Code — CLAUDE.md, slash commands, hooks, MCP servers, and web sessions.
keywords: gh cli, claude code, claude code cli, claude.md, slash commands, hooks, mcp servers, developer workflow, command line workflow, github automation
category: tutorials
tags: [github, cli, claude-code, workflow, gh, mcp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Build in the Open"
series_part: 14
---

> **TL;DR:** The fastest builders live close to the command line and treat an AI
> agent as a full member of the workflow. The `gh` CLI puts issues, PRs,
> releases, and CI runs a keystroke away without leaving your terminal. Claude
> Code — in the CLI *or* on the web — turns plain-English intent into branches,
> commits, and PRs, anchored by a `CLAUDE.md` memory file, sharpened with slash
> commands and hooks, and extended with MCP servers. Stitch Parts 1–13 together
> and you get one repeatable daily loop: idea → issue → branch → build with
> Claude → PR → CI → merge → release.

**Key takeaways**

- The `gh` CLI lets you create issues, open PRs, watch CI runs, and cut releases
  without touching the browser.
- Claude Code reads `CLAUDE.md` as project memory, so it learns your conventions
  once and applies them every session.
- Slash commands package repeatable prompts; hooks automate checks; MCP servers
  give the agent new tools.
- Claude Code on the web matches the exact same repo setup — branches, PRs,
  conventions — as the CLI.
- The whole series collapses into one loop you can run every day on any project.

*This is Part 14 of **Build in the Open**, a 14-part series on taking a software
project from a blank idea to a public release using GitHub and Claude Code. Each
post teaches a technique you can apply to any project in any language, then shows
how the open-source [GopherTrunk](https://github.com/MattCheramie/GopherTrunk)
scanner does it for real. This is the finale.*

## In this post

- **Why the CLI wins** — staying in one place beats tab-switching.
- **The `gh` CLI** — issues, PRs, releases, and CI runs from the terminal.
- **Claude Code as a daily driver** — CLI and web, and how they share a setup.
- **`CLAUDE.md`, slash commands, hooks, and MCP** — making the agent yours.
- **The daily loop** — Parts 1–13 as one repeatable cycle.
- **How GopherTrunk does it**, and a send-off.

## Why work from the command line?

Every context switch costs focus. Bouncing between editor, browser tabs for
issues and PRs, and a terminal for git fragments your attention. Pulling all of
it into the command line — or into a single agent conversation — keeps you in
flow. It's also scriptable and repeatable in a way clicking never is: a workflow
you can type is a workflow you can automate.

## The `gh` CLI: GitHub without the browser

[`gh`](https://cli.github.com/) is GitHub's official command-line tool. It covers
most of what you'd otherwise open a browser for:

- **Issues:** `gh issue create`, `gh issue list`, `gh issue view 698`.
- **Pull requests:** `gh pr create`, `gh pr view`, `gh pr checks`,
  `gh pr merge --squash`.
- **CI runs:** `gh run list`, `gh run watch`, `gh run view --log` to tail a
  workflow without opening the Actions tab.
- **Releases:** `gh release create v0.99.0 ...` to cut a release (or just push
  the tag and let your release workflow from
  [Part 11]({{ '/blog/tutorials/build-in-the-open-11-releases-prerelease-semver-changelogs/' | relative_url }})
  do the rest).

Authenticate once with `gh auth login` and the whole GitHub side of the workflow
lives at your prompt.

## Claude Code as your daily driver

Claude Code is an AI agent that works *in* your project — reading files, running
commands, editing code, and opening PRs. The shift is from "ask a chatbot,
copy-paste answers" to "delegate a task and review the result." You describe
intent ("add a `/api/v1/sites` endpoint and a regression test"), and the agent
does the multi-step work: explore the code, make the change, run the tests, and
prepare a commit.

<figure class="lab-figure">
<svg viewBox="0 0 660 156" width="660" height="156" role="img" aria-label="The Claude Code daily-driver loop as a cycle: a plain-English prompt leads the agent to explore the code, edit it, run the tests, commit, and open a PR; from the PR the loop returns to the next prompt, with review feeding back in.">
  <rect x="14" y="52" width="92" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="60" y="70" text-anchor="middle" fill="var(--accent)" font-size="10">prompt</text>
  <text x="60" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">plain intent</text>
  <line x1="106" y1="72" x2="132" y2="72" stroke="currentColor"/><polygon points="132,68 142,72 132,76" fill="currentColor"/>
  <rect x="142" y="52" width="92" height="40" rx="6" fill="none" stroke="currentColor"/><text x="188" y="70" text-anchor="middle" fill="currentColor" font-size="10">explore</text><text x="188" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">read files</text>
  <line x1="234" y1="72" x2="260" y2="72" stroke="currentColor"/><polygon points="260,68 270,72 260,76" fill="currentColor"/>
  <rect x="270" y="52" width="92" height="40" rx="6" fill="none" stroke="currentColor"/><text x="316" y="70" text-anchor="middle" fill="currentColor" font-size="10">edit</text><text x="316" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">make change</text>
  <line x1="362" y1="72" x2="388" y2="72" stroke="currentColor"/><polygon points="388,68 398,72 388,76" fill="currentColor"/>
  <rect x="398" y="52" width="92" height="40" rx="6" fill="none" stroke="currentColor"/><text x="444" y="70" text-anchor="middle" fill="currentColor" font-size="10">test</text><text x="444" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">run suite</text>
  <line x1="490" y1="72" x2="516" y2="72" stroke="currentColor"/><polygon points="516,68 526,72 516,76" fill="currentColor"/>
  <rect x="526" y="52" width="118" height="40" rx="6" fill="none" stroke="var(--accent)"/><text x="585" y="70" text-anchor="middle" fill="var(--accent)" font-size="10">commit → PR</text><text x="585" y="83" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gh pr create</text>
  <line x1="585" y1="92" x2="585" y2="122" stroke="currentColor"/><line x1="585" y1="122" x2="60" y2="122" stroke="currentColor"/><line x1="60" y1="122" x2="60" y2="92" stroke="currentColor"/><polygon points="56,100 60,90 64,100" fill="currentColor"/>
  <text x="322" y="138" text-anchor="middle" fill="var(--fg-muted)" font-size="9">review the result → next prompt</text>
</svg>
<figcaption>Delegate, don't copy-paste: one prompt drives the agent through explore, edit, test, and commit to an opened PR — then you review and start the loop again.</figcaption>
</figure>

It comes in two forms that share the same setup:

- **The CLI** — `claude` in your terminal, right next to `gh` and `git`.
- **The web** — Claude Code on the web runs against the same repository, the
  same branch conventions, the same `CLAUDE.md`. Start a task from your phone or
  a browser, and it operates exactly as the CLI would.

Because both read the same project configuration, the agent behaves
consistently no matter where you launch it.

### CLAUDE.md: project memory

`CLAUDE.md` is a file in your repo that Claude Code reads automatically every
session. It's where you encode the conventions you'd otherwise repeat constantly:
how to run tests, your commit-message style, branch-naming rules, "don't touch
the generated `pb/` directory by hand," and so on. Write it once and every future
session — CLI or web — starts already knowing how your project works. We
introduced it back in
[Part 3]({{ '/blog/tutorials/build-in-the-open-03-brainstorm-with-claude-readme-roadmap/' | relative_url }});
by now it's the agent's onboarding doc.

### Slash commands, hooks, and MCP servers

Three ways to bend Claude Code to your workflow:

- **Slash commands** package a repeatable prompt behind a short name — e.g. a
  `/code-review` command that runs your standard review pass. Stop re-typing the
  same instructions.
- **Hooks** run your own commands at lifecycle points — for example, auto-running
  a formatter or test suite after the agent edits a file, so quality gates fire
  without you asking.
- **MCP servers** (Model Context Protocol) give the agent new tools — a GitHub
  MCP server to manage issues and PRs, a database server to run queries, a Canva
  server to make graphics. MCP is how you extend what the agent can *reach*.

## The daily loop: Parts 1–13 as one cycle

Everything in this series collapses into a single repeatable loop you can run on
any project, in any language:

1. **Decide what to build** ([Part 1]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})) and **pick the stack** (Part 2) — once per project.
2. **Capture the idea** as an issue (Part 6), brainstormed and scoped with Claude (Part 3).
3. **Branch** off `main` (Parts 4–5): `git switch -c claude/issue-NNN-desc`.
4. **Build with Claude Code**, guided by `CLAUDE.md`, with tests (Part 8) and docs (Part 9) as you go.
5. **Open a PR** with `gh pr create`; CI runs your required checks (Part 7).
6. **Review, resolve conflicts** (Part 13), and **merge** with the right strategy (Part 5).
7. **Release** when there's something worth shipping (Part 11), on a secured, optimized repo (Part 12), surfaced on your site (Part 10).

Then back to step 2. That loop, run consistently, is how a blank idea becomes a
released, maintained project.

<figure class="lab-figure">
<svg viewBox="0 0 660 176" width="660" height="176" role="img" aria-label="The full daily loop as a cycle of seven stages: capture an idea as an issue, branch off main, build with Claude Code, open a PR, let CI run required checks, merge to main, and release when there is something worth shipping — then the arrow returns from release to the next issue.">
  <rect x="16" y="20" width="118" height="38" rx="6" fill="none" stroke="var(--accent)"/><text x="75" y="38" text-anchor="middle" fill="var(--accent)" font-size="10">idea → issue</text><text x="75" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">scoped w/ Claude</text>
  <line x1="134" y1="39" x2="176" y2="39" stroke="currentColor"/><polygon points="176,35 186,39 176,43" fill="currentColor"/>
  <rect x="186" y="20" width="118" height="38" rx="6" fill="none" stroke="currentColor"/><text x="245" y="38" text-anchor="middle" fill="currentColor" font-size="10">branch</text><text x="245" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">claude/issue-NNN</text>
  <line x1="304" y1="39" x2="346" y2="39" stroke="currentColor"/><polygon points="346,35 356,39 346,43" fill="currentColor"/>
  <rect x="356" y="20" width="118" height="38" rx="6" fill="none" stroke="var(--accent)"/><text x="415" y="38" text-anchor="middle" fill="var(--accent)" font-size="10">build w/ Claude</text><text x="415" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">CLAUDE.md · tests</text>
  <line x1="474" y1="39" x2="516" y2="39" stroke="currentColor"/><polygon points="516,35 526,39 516,43" fill="currentColor"/>
  <rect x="526" y="20" width="118" height="38" rx="6" fill="none" stroke="currentColor"/><text x="585" y="38" text-anchor="middle" fill="currentColor" font-size="10">open PR</text><text x="585" y="51" text-anchor="middle" fill="var(--fg-muted)" font-size="8">gh pr create</text>
  <line x1="585" y1="58" x2="585" y2="88" stroke="currentColor"/><polygon points="581,88 585,98 589,88" fill="currentColor"/>
  <rect x="526" y="98" width="118" height="38" rx="6" fill="none" stroke="currentColor"/><text x="585" y="116" text-anchor="middle" fill="currentColor" font-size="10">CI checks</text><text x="585" y="129" text-anchor="middle" fill="var(--fg-muted)" font-size="8">required gates</text>
  <line x1="526" y1="117" x2="484" y2="117" stroke="currentColor"/><polygon points="484,113 474,117 484,121" fill="currentColor"/>
  <rect x="356" y="98" width="118" height="38" rx="6" fill="none" stroke="currentColor"/><text x="415" y="116" text-anchor="middle" fill="currentColor" font-size="10">merge</text><text x="415" y="129" text-anchor="middle" fill="var(--fg-muted)" font-size="8">squash to main</text>
  <line x1="356" y1="117" x2="314" y2="117" stroke="currentColor"/><polygon points="314,113 304,117 314,121" fill="currentColor"/>
  <rect x="186" y="98" width="118" height="38" rx="6" fill="none" stroke="var(--accent)"/><text x="245" y="116" text-anchor="middle" fill="var(--accent)" font-size="10">release</text><text x="245" y="129" text-anchor="middle" fill="var(--fg-muted)" font-size="8">push vX.Y.Z</text>
  <line x1="186" y1="117" x2="75" y2="117" stroke="currentColor"/><line x1="75" y1="117" x2="75" y2="58" stroke="currentColor"/><polygon points="71,68 75,58 79,68" fill="currentColor"/>
  <text x="130" y="90" text-anchor="middle" fill="var(--fg-muted)" font-size="9">back to the next issue</text>
</svg>
<figcaption>Parts 1–13 collapse into one repeatable cycle: issue → branch → build with Claude → PR → CI → merge → release, then round again — the loop GopherTrunk's own history runs on.</figcaption>
</figure>

## How GopherTrunk does it

[GopherTrunk](https://github.com/MattCheramie/GopherTrunk) *is* this loop, made
visible in its own history. The git log reads like a tour of the workflow:
branches named `claude/issue-698-features-ytrap7` and
`claude/release-workflow-docs-sync-bdhhjz`, each landing as a squashed
`Merge pull request #NNN` — the exact `claude/...` branch-naming and PR-driven
convention this series taught. Commit `72c7ef3 feat: expose P25 site identity in
grant events and via /api/v1/sites (#698)` is a single logical change, tested and
merged through a PR, then promoted into `CHANGELOG.md`'s `## [Unreleased]`
section, ready for the next tag to ship it.

Releases are tag-driven (`make release-dry-run` to rehearse, then push `vX.Y.Z`
or use `gh release` / the Actions button), CI gates every PR with required checks
including `govulncheck`, and the daily Pages cron drips this very blog series out
one post at a time. The same project setup that the CLI uses is exactly what
Claude Code on the web operates against — same branches, same conventions, same
`CLAUDE.md`. The work that produced these 14 posts ran through that loop too.

## The journey, end to end

Look back at where we started: a blank idea and the advice to scratch your own
itch. From there we chose a language and platforms, brainstormed features and
wrote the README as a roadmap, learned Git and GitHub from the web UI, mastered
branching and the three ways to merge, planned work and invited contributors,
wired up CI with GitHub Actions, built a real test suite, documented the project
properly, published a website, cut SemVer releases with changelogs, optimized and
secured the repo, leveled up with advanced Git, and finally tied it all together
at the command line with Claude Code.

That's the whole arc — **idea to release** — and none of it required being an
expert at the start. It required picking one real problem and turning the crank
on the loop, one PR at a time.

So: now go build something. Pick the itch that's been bugging you, write the
one-sentence problem statement, open the repo, and start the loop. The tools are
free, the workflow is proven, and the only missing ingredient is the thing only
you can supply — the project worth building.

Thanks for reading **Build in the Open**.

## FAQ

**Do I need the CLI, or is the web enough?**
Either works, and they share the same setup. Many people use the CLI at their
desk and Claude Code on the web for kicking off tasks from a phone or browser.
Because both read your repo's `CLAUDE.md` and conventions, the experience is
consistent.

**What goes in CLAUDE.md?**
Anything you'd otherwise repeat every session: how to run tests and lint, commit
and branch conventions, files the agent should leave alone, and pointers to key
docs. Treat it as the agent's onboarding guide.

**What's the difference between a slash command and a hook?**
A slash command is something *you* invoke to run a packaged prompt. A hook fires
*automatically* at a lifecycle point (like after a file edit), without you asking
— ideal for formatters and test gates.

**What is an MCP server?**
A Model Context Protocol server extends what Claude Code can reach — issues and
PRs via a GitHub server, a database, design tools, and more. It's how you give
the agent new capabilities beyond editing files and running shell commands.

**Can I really run this whole loop on a non-Go project?**
Yes — that's the point of the series. The principles (SemVer, CI gates,
PR-driven branches, a `CLAUDE.md`, tag-triggered releases) are language-agnostic.
GopherTrunk is just one worked example; swap Go for Python, JS, or Rust and the
loop is identical.

## Series navigation

**Part 14 of 14** · ←
[Part 13: Advanced Git & GitHub Features]({{ '/blog/tutorials/build-in-the-open-13-advanced-git-and-github-features/' | relative_url }})

That's a wrap on **Build in the Open** — fourteen parts from a blank idea to a
secured, released, documented project. Revisit
[Part 1: Picking What to Build]({{ '/blog/tutorials/build-in-the-open-01-picking-what-to-build/' | relative_url }})
to start a new project from scratch, or browse the whole journey on the
[series index]({{ '/blog/series/build-in-the-open/' | relative_url }}). Now go
build something.
