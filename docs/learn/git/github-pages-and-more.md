---
slug: github-pages-and-more
title: Pages, Gists, Discussions & wikis
description: A practical tour of GitHub's extra features — free Pages hosting, shareable Gists, Discussions for Q&A, repo wikis, the profile README, and stars, watching & sponsors.
keywords: github pages, custom domain, jekyll, github gist, github discussions, repo wiki, profile readme, github stars, watch repository, github sponsors
level: beginner
status: full
prereq:
  - what-is-github
faq:
  - q: What is GitHub Pages and is it free?
    a: GitHub Pages is free static website hosting served directly from a GitHub repository. You enable it under Settings → Pages and choose a source — a branch and folder, or a GitHub Actions build. It serves HTML, CSS, and JavaScript (and can build Jekyll sites automatically) at a username.github.io address, with optional custom domains. It's ideal for project docs, portfolios, and blogs; it does not run server-side code.
  - q: What is the difference between a Gist and a repository?
    a: A Gist is a lightweight place to share one or a few files — a snippet, a config, a script — without the ceremony of a full repo. Each Gist is actually a tiny Git repository, so it has version history and can be cloned, but it has no issues, pull requests, or folders. Use a repo for a project; use a Gist to quickly share a self-contained snippet.
  - q: When should I use Discussions instead of Issues?
    a: Use Issues for actionable, trackable work — bugs and feature requests that should be opened, assigned, and closed. Use Discussions for open-ended conversation — questions, ideas, announcements, and community Q&A that may never "close." Discussions support threaded replies and marking an answer, which suits support questions; Issues suit a task list.
  - q: What is a profile README?
    a: A profile README is a special repository named exactly after your username (for example octocat/octocat) containing a README.md. GitHub renders that README at the top of your profile page, so you can introduce yourself, highlight projects, or show stats. It's the same Markdown as any README — it just appears on your public profile.
---

# Pages, Gists, Discussions & wikis

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Beyond code, GitHub bundles handy extras. **Pages** hosts a static website free from a
repo (a branch or [Actions](/learn/git/github-actions/) build, with **custom domains**
and **Jekyll**). **Gists** are tiny git repos for sharing snippets, public or secret.
**Discussions** are for open-ended community Q&A — distinct from trackable
[issues](/learn/git/issues-and-projects/). **Wikis** hold long-form docs alongside a
repo, and a **profile README** (a repo named after you) introduces you on your profile.
Round it out with **stars**, **watching**, and **Sponsors**.
</div>

You've covered the core collaboration loop. This lesson is a quick tour of GitHub's
extra features — each useful, none complicated. Skim it now and come back when you need
one.

## GitHub Pages: free static hosting

**Pages** serves a static website straight from a repository — no servers to manage,
no cost. Turn it on under **Settings → Pages** and pick a **source**:

- a **branch and folder** (commonly `main` and `/` or `/docs`), or
- **GitHub Actions**, which runs a build workflow and deploys the output.

Your site appears at `https://<username>.github.io/<repo>/` (or
`https://<username>.github.io/` for a repo named `<username>.github.io`). You can point
a **custom domain** at it under the same settings and add a `CNAME` file. Pages also
**builds Jekyll** automatically — drop in Markdown and a `_config.yml` and GitHub
generates the HTML, which is exactly how many project documentation sites (including
this one) are published. Pages serves static files only; it won't run server-side code.

## Gists: shareable snippets

A **Gist** is the fast way to share a snippet without creating a whole project. Visit
**gist.github.com**, paste your code, give the file a name with a sensible extension
(so it's syntax-highlighted), and save. Two visibilities:

- **public** — discoverable and listed on your Gists page;
- **secret** — not listed or searchable, but anyone with the link can view it (so it's
  not private — don't put real credentials in one).

Each Gist is itself a **tiny Git repository**: it has version history, supports
comments, and can be cloned and pushed to like any repo. Reach for a Gist to share a
config file, a one-off script, or an error log.

## Discussions: community Q&A

**Discussions** is a forum built into a repository, enabled under **Settings → General →
Features**. It's for the conversations that don't fit the issue model:

| Use **Issues** for… | Use **Discussions** for… |
|---|---|
| Bugs and feature requests | Questions and how-do-I help |
| Trackable, closeable work | Open-ended ideas and brainstorming |
| Things you assign | Announcements and show-and-tell |

Discussions support **threaded replies** and let a maintainer **mark an answer**, which
makes them ideal for support Q&A. The rule of thumb: if it has a clear definition of
"done," it's an [issue](/learn/git/issues-and-projects/); if it's a conversation, it's a
discussion.

## Wikis: long-form docs

Every repo can have a **Wiki** — a separate space for documentation that doesn't belong
in the codebase, like design notes, FAQs, or onboarding guides. Enable it under
**Settings → Features**, then edit pages in Markdown right in the browser. A wiki is
backed by its own Git repository, so you can clone it (`<repo>.wiki.git`) and edit
offline. For docs that should version with the code, a `/docs` folder is often better;
for free-form, frequently edited reference material, a wiki shines.

## The profile README

A **profile README** turns your GitHub profile into a mini landing page. Create a
repository named **exactly** your username (e.g. `octocat/octocat`), add a `README.md`,
and GitHub renders it at the top of your profile:

```markdown
### Hi, I'm Octocat 

- Working on open-source CLI tools
- Ask me about Git and Go
- Reach me at octocat@example.com
```

It's ordinary Markdown — add badges, links, or contribution stats. GitHub even prompts
you to create one when you visit your own profile.

## Stars, watching & Sponsors

Three social signals worth knowing:

- **Star** — a public bookmark and a vote of appreciation; a repo's star count is a
  rough popularity gauge, and your stars form a personal reading list.
- **Watch** — subscribe to a repo's notifications. Choose *All Activity*,
  *Participating*, or *Custom* (only releases, or only issues) so you're notified about
  what matters without drowning.
- **Sponsors** — GitHub's built-in way to fund maintainers, either one-off or monthly,
  via a **Sponsor** button enabled through a `FUNDING.yml` file.

Starring costs nothing and helps maintainers; watching keeps you informed; sponsoring
keeps the projects you depend on alive. (Any unfamiliar term here is in the
[glossary](/learn/git/glossary/).)

<div class="knowledge-check" data-quiz data-correct-msg="Right — Pages serves a static site straight from your repo." markdown="0">
  <p class="knowledge-check__q">Quick check: which GitHub feature hosts a free static website from a repository?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Gists</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Discussions</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">GitHub Pages</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Pages** hosts a free static site from a branch or [Actions](/learn/git/github-actions/)
  build, with custom domains and Jekyll support.
- **Gists** are tiny git repos for sharing snippets — public or secret (secret ≠
  private).
- **Discussions** suit open-ended Q&A; [issues](/learn/git/issues-and-projects/) suit
  trackable work.
- **Wikis** hold long-form docs in their own backing repo.
- A **profile README** (a repo named after you) appears on your profile.
- **Stars**, **watching**, and **Sponsors** are the social and funding layer.

Next up: locking down `main` with [branch protection, reviews & CODEOWNERS](/learn/git/branch-protection/).
