---
slug: best-practices
title: Commit hygiene & best practices
description: Habits that make Git a pleasure to work with — atomic commits, good commit messages, small PRs, a consistent merge policy, and never committing secrets or large binaries.
keywords: commit hygiene, git best practices, atomic commits, commit message, imperative mood, conventional commits, small pull requests, merge vs rebase, never commit secrets, git-lfs, gitignore
level: intermediate
status: full
prereq:
  - staging-and-commits
  - pull-requests
faq:
  - q: What makes a good commit message?
    a: 'A good commit message has a short subject line of about 50 characters written in the imperative mood ("Add login validation", not "Added" or "Adds"), a blank line, then a body that explains WHY the change was made rather than restating what the diff already shows. Many teams adopt Conventional Commits, prefixing the subject with a type like feat:, fix:, or docs: so the history is machine-readable and changelogs can be generated automatically.'
  - q: What is an atomic commit?
    a: An atomic commit makes exactly one logical change and nothing else — a single bug fix, one feature increment, or one refactor — so it stands on its own. Atomic commits make history readable, reviews easier, and tools like revert, cherry-pick, and bisect far more effective, because each commit is a clean unit you can apply or undo without dragging unrelated changes along.
  - q: I accidentally committed a secret — how do I remove it?
    a: First, treat the secret as compromised and rotate it immediately, because once pushed it may already be cached or scanned. Removing it from history means rewriting history with interactive rebase (if recent) or a tool like git filter-repo, then force-pushing — and remember the reflog and any clones or forks may still hold the old commit for a while. Prevention via .gitignore and secret-scanning is far easier than cleanup.
  - q: Should my team use merge or rebase?
    a: Either works — what matters is choosing one policy and applying it consistently. Some teams rebase feature branches for a clean linear history; others merge to preserve the true context of parallel work; many rebase locally then merge into main. Agree on the approach, document it in your contributing guide, and configure branch protection and defaults to match so the history stays predictable.
---

# Commit hygiene & best practices

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Good Git habits compound. Make **atomic commits** — one logical change each. Write
**clear messages**: a ~50-char imperative subject, a blank line, then a body explaining
**why** (consider [Conventional Commits](/learn/git/glossary/)). Keep **pull requests
small** so they're actually reviewable. Pick **merge or rebase** as a team and stay
consistent. **Never commit secrets or large binaries** — history is forever, so use
[`.gitignore`](/learn/git/gitignore/) and **git-lfs**, and rotate any secret that slips
through. Review with empathy. These are the habits that make every earlier lesson pay off.
</div>

You've learned the mechanics across this path; this final lesson is about *taste* — the
conventions that separate a repo people enjoy working in from one they dread. None are
enforced by Git itself. They're agreements a team makes and keeps.

## Atomic, focused commits

A commit should capture **one logical change** and nothing else: a single fix, one feature
increment, one refactor. Resist bundling "fix bug + rename variables + update deps" into one
commit — split them with `git add -p` to stage selectively:

```bash
$ git add -p          # stage changes hunk by hunk, keeping commits focused
```

Atomic commits make history readable and make Git's best tools work properly:
[`revert`](/learn/git/undoing-changes/),
[`cherry-pick`, and `bisect`](/learn/git/cherry-pick-bisect-reflog/) all assume each commit
is a clean, self-contained unit. A commit that does three things can't be cleanly undone or
ported.

## Writing good commit messages

The widely-followed shape, popularised by the Git community:

```text
Add rate limiting to the login endpoint

Brute-force attempts were hitting the auth service unthrottled.
Cap attempts at 5 per minute per IP and return 429 beyond that.

Closes #214
```

The rules behind it:

- **Subject ~50 characters**, in the **imperative mood** — "Add", "Fix", "Refactor", not
  "Added" or "Fixes". (It reads as *"this commit will Add…"*.)
- A **blank line** separates subject from body — many tools rely on it.
- The **body explains *why*** and wraps at ~72 columns. The diff already shows *what*
  changed; the message should capture the reasoning a future reader can't infer.

Many teams adopt **[Conventional Commits](/learn/git/glossary/)** — prefixing the subject
with a type such as `feat:`, `fix:`, `docs:`, or `refactor:`:

```text
fix: cap login attempts at 5 per minute per IP
```

This makes history machine-readable and lets tools generate changelogs and version bumps
automatically.

## Small, reviewable pull requests

A 1,000-line [pull request](/learn/git/pull-requests/) gets a rubber-stamp; a 100-line one
gets a real review. Keep PRs **small and single-purpose** — one feature or fix per PR. Big
work is better as a series of small, stacked PRs than one giant one. Small PRs review
faster, merge sooner, and are far easier to revert if something's wrong. Write a clear
description and link the [issue](/learn/git/issues-and-projects/) it addresses.

## Merge vs rebase: pick a policy

There's no universally right answer to [merge vs rebase](/learn/git/rebasing/) — and that's
exactly why the team must **choose one and apply it consistently**:

| Approach | History | Common with |
|----------|---------|-------------|
| Merge commits | Preserves parallel context | Teams valuing true history |
| Rebase / linear | Clean straight line | Teams valuing readability |
| Rebase locally, merge to `main` | Tidy branches + recorded integration | A popular middle ground |

The inconsistency is what hurts, not the choice. Document your policy in a contributing
guide and align your [branch protection](/learn/git/branch-protection/) and Git defaults to
match.

## Never commit secrets or large binaries

This is the one with no easy undo: **Git history is forever.** A pushed commit can be
cloned, forked, cached, and scanned within seconds.

- **Secrets** — API keys, passwords, tokens — must never be committed. Keep them in
  environment variables or a secrets manager, and list secret files in
  [`.gitignore`](/learn/git/gitignore/). If one slips through, **rotate it immediately**
  (assume it's compromised), then scrub it from history with [interactive
  rebase](/learn/git/interactive-rebase/) (if recent) or `git filter-repo`, and force-push.
  Even then, the [reflog](/learn/git/cherry-pick-bisect-reflog/) and any existing clones or
  forks may retain the old commit — which is why rotation, not removal, is the real fix.
- **Large binaries** — videos, datasets, build artifacts — bloat the repo permanently
  because every clone fetches all of history. Use **Git LFS** (Large File Storage), which
  stores big files outside normal history and keeps only lightweight pointers in the repo:

```bash
$ git lfs track "*.psd"   # store .psd files via LFS instead of in core history
$ git add .gitattributes
```

Prevention beats cleanup every time — and most hosts offer secret-scanning that warns you
before a leaked key spreads.

## A disciplined .gitignore

A well-tended [`.gitignore`](/learn/git/gitignore/) is quiet best practice. Ignore build
output, dependencies (`node_modules/`), editor files, OS cruft (`.DS_Store`), and local
config from the start — a clean `git status` keeps you from accidentally committing junk or
secrets. Start from a language-appropriate template (GitHub's `gitignore` repo has good
ones) and keep it current as the project grows.

## Review etiquette

Code review is a team's quality backbone, so make it humane:

- **Review the code, not the coder** — comment on the change, never the person.
- **Be specific and kind**: suggest, explain *why*, and offer alternatives rather than just
  flagging faults.
- **Distinguish blocking issues from nitpicks** — prefix optional suggestions ("nit:") so
  authors know what actually gates the merge.
- **As an author**, keep PRs small, respond to every comment, and don't take feedback
  personally — it's about the code.

Good etiquette makes review something the team looks forward to rather than dreads, and that
culture is worth more than any single rule above.

<div class="knowledge-check" data-quiz data-correct-msg="Right — the subject is short and imperative, and the body explains why." markdown="0">
  <p class="knowledge-check__q">Quick check: which is the best-formed commit subject line?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">"Fixed a whole bunch of stuff in the login code and also tweaked some other things"</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">"Add rate limiting to the login endpoint"</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">"changes"</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Make **atomic commits** — one logical change each — so revert, cherry-pick, and bisect work cleanly.
- Write good messages: **~50-char imperative subject**, blank line, body explaining **why**; consider **Conventional Commits**.
- Keep **pull requests small** and single-purpose so reviews are real.
- Pick **merge or rebase** as a team and apply it **consistently**.
- **Never commit secrets** (rotate if leaked) **or large binaries** (use **git-lfs**) — history is forever.
- Maintain a disciplined **[`.gitignore`](/learn/git/gitignore/)** and review code with empathy.

Next up: every term from this path, defined in one place — the [Git & GitHub glossary](/learn/git/glossary/).
