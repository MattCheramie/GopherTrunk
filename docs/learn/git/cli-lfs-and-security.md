---
slug: cli-lfs-and-security
title: The GitHub CLI, Git LFS & repo security
description: Round out day-to-day GitHub with three practical features — driving GitHub from the terminal with the gh CLI, handling big binaries with Git LFS, and letting Dependabot, secret scanning and code scanning guard your repo.
keywords: GitHub CLI, gh command, Git LFS, large file storage, Dependabot, secret scanning, push protection, security advisories, CodeQL, code scanning
level: intermediate
status: full
prereq:
  - pull-requests
  - remotes
faq:
  - q: What is the gh CLI and why use it?
    a: "gh is GitHub's official command-line tool. It lets you do GitHub-side work — open and review pull requests, manage issues, clone repos, watch Actions runs — without leaving the terminal or round-tripping through a browser. Because it's a command, it also scripts cleanly, so common chores can be automated. It complements git rather than replacing it: git handles commits and history, gh handles the GitHub platform around them."
  - q: What is Git LFS for?
    a: Git Large File Storage keeps big binary files (audio, video, datasets, trained models) out of your repository's history. Git normally stores every version of every file forever, so committing large binaries bloats the repo permanently. LFS replaces those files with tiny text pointers and stores the real blobs separately, so a clone only pulls the versions it needs.
  - q: How does GitHub stop me committing a secret?
    a: Two features work together. Secret scanning recognises the patterns of known credentials — API keys, tokens, private keys — in your commits. Push protection goes further and blocks the push outright when it spots one, before the secret ever reaches the remote. If a secret does leak, the correct fix is to rotate (revoke and reissue) it, not just delete the commit, because anyone may already have copied it.
  - q: What is Dependabot?
    a: Dependabot watches the dependencies your project declares and alerts you when one has a known vulnerability. It can also open pull requests that bump the affected dependency to a fixed version automatically, so keeping libraries patched becomes a review-and-merge chore rather than manual tracking.
---

# The GitHub CLI, Git LFS & repo security

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three practical features round out day-to-day GitHub. The **gh CLI** drives the
platform from your terminal — open [pull requests](/learn/git/pull-requests/), review,
watch [Actions](/learn/git/github-actions/) runs — without browser round-trips, and it
scripts cleanly. **Git LFS** keeps big binaries (audio, video, datasets, models) out of
your history by storing tiny **pointer files** in Git and the blobs separately. And
GitHub can **guard the repo itself**: **Dependabot** flags vulnerable dependencies,
**secret scanning** plus push protection stop leaked credentials, and code scanning
(CodeQL) runs static analysis in CI. All three are opt-in tools you turn on as a repo
grows.
</div>

You've learned the everyday flow — commit, push, open a PR, run CI. This lesson covers
three features that sit alongside that flow and make working on GitHub faster and safer:
a command-line tool, a way to handle large files, and the platform's built-in security
guards. None of them is essential on day one, but each earns its place as a project
matures.

## The GitHub CLI (gh)

**`gh`** is GitHub's official command-line tool. Everything you'd normally click through
the website for — pull requests, issues, releases, Actions — has a `gh` equivalent, so
you can stay in the terminal where your code already is.

First, authenticate once:

```sh
gh auth login          # interactive: pick GitHub.com, HTTPS/SSH, log in
```

After that, the common commands read almost like English:

```sh
gh repo clone owner/name       # clone without typing the full URL
gh pr create                   # open a PR from the current branch
gh pr checkout 42              # check out someone else's PR locally
gh pr view --web               # jump to a PR in the browser
gh issue list                  # browse open issues
gh run watch                   # follow the latest Actions run live
```

Two things make `gh` worth adopting. First, it **saves round-trips**: opening a PR or
checking a review no longer means switching to the browser, finding the tab, and
clicking around. Second, because each command is just a command, it **scripts nicely** —
you can wire `gh` into shell scripts or Actions to automate repetitive chores.

Note that `gh` **complements git, it doesn't replace it**. `git` still handles your
commits, branches, and history locally; `gh` handles the GitHub platform layered on top —
the PRs, issues, and runs that live on the server.

## Git LFS — large files without bloat

Git was built for source code, and it stores **every version of every file forever**.
That's exactly what you want for text: diffs are small and history is cheap. It's a
problem for **big binaries** — audio captures, video, datasets, trained models. Commit a
100 MB file, change it ten times, and your repository now carries a gigabyte of history
that every clone must download, permanently, even after you delete the file.

**Git LFS (Large File Storage)** fixes this. Instead of putting the binary in Git, LFS
stores a tiny **pointer file** — a few lines of text naming the real content — and keeps
the actual blob on a separate LFS store. Git history stays small; the heavy data is
fetched only when a checkout needs it.

You tell LFS which files to manage by pattern:

```sh
git lfs install                 # one-time setup per machine
git lfs track "*.wav"           # manage all .wav files with LFS
git add .gitattributes          # tracking rules live here — commit them
git add capture.wav
git commit -m "Add sample capture via LFS"
```

`git lfs track` writes the pattern into **`.gitattributes`**, which you commit so
everyone on the project inherits the same rules.

A few caveats:

- **Quotas.** LFS storage and bandwidth have limits on GitHub's free tier; large or busy
  repos may need a paid allowance.
- **Everyone needs LFS installed.** A collaborator without Git LFS sees the pointer files,
  not the real content.
- **It's not for source code.** LFS is for large binaries; plain text belongs in normal
  Git, where diffs and merges work.

LFS is also a decision point next to your [`.gitignore`](/learn/git/gitignore/) choices:
some big files should be tracked in LFS, but many build artifacts and generated blobs
shouldn't be committed at all — those simply go in `.gitignore`.

## Repo security features

GitHub can actively watch your repository for common security problems. These are opt-in,
found under **Settings → Security**, and cost nothing to enable on most repos.

### Dependabot

**Dependabot** reads the dependencies your project declares (via `package.json`,
`go.mod`, `requirements.txt`, and the like) and cross-checks them against a database of
known vulnerabilities. When a dependency you use has a published flaw, it raises an
**alert**. Better still, it can open **automated pull requests** that bump the affected
package to a fixed version — so patching becomes a matter of reviewing and merging a PR,
not tracking advisories by hand.

### Secret scanning & push protection

Credentials committed by accident — an API key, a cloud token, a private key — are one
of the most common ways secrets leak. GitHub's **secret scanning** recognises the
patterns of hundreds of known credential types in your commits. **Push protection** takes
it a step earlier: it **blocks the push** the moment it detects a secret, so the credential
never reaches the remote in the first place.

If a secret does get out, understand the real fix. Deleting the commit or rewriting
history is **not enough** — anyone (or any bot scraping public pushes) may already have a
copy. The only safe response is to **rotate the secret**: revoke the leaked key and issue
a new one, so the exposed value becomes worthless.

### Security advisories & code scanning (CodeQL)

For maintainers, GitHub adds two more tools. **Security advisories** let you draft and
discuss a vulnerability **privately**, prepare a fix, and only then publish the advisory
and coordinate disclosure — rather than fixing a bug in the open where it tips off
attackers. And **code scanning** with **CodeQL** runs automated static analysis over your
code as part of [CI](/learn/git/github-actions/), flagging suspicious patterns (injection
risks, unsafe deserialisation, and so on) directly on the pull request that introduced
them.

## When to reach for each

These are tools you switch on as the need appears, not upfront:

- Install **`gh`** as soon as you're opening PRs regularly — it pays for itself the first
  day.
- Turn on **Git LFS** the moment you need to version a large binary. Do it *before* the
  first big commit; retrofitting LFS onto existing history is painful.
- Enable **Dependabot** as soon as your project has real dependencies.
- Enable **secret scanning and push protection** early — a leaked key is far cheaper to
  prevent than to clean up.
- Reach for **advisories and CodeQL** when the project is used by others and security
  matters enough to formalise.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a leaked key may already be copied, so you must revoke it and issue a new one." markdown="0">
  <p class="knowledge-check__q">Quick check: you accidentally push an API key. Besides removing it from history, what MUST you do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — deleting the commit is enough</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Rotate/revoke the leaked key and issue a new one</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Add the key to a .gitignore file</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **`gh`** is GitHub's official CLI — PRs, issues, clones, and Actions runs from the
  terminal; it scripts well and **complements**, not replaces, git.
- **Git LFS** keeps large binaries out of history via small **pointer files** and a
  `.gitattributes` tracking list; mind quotas, and remember everyone needs LFS installed.
- **Dependabot** alerts on vulnerable dependencies and can open update PRs automatically.
- **Secret scanning + push protection** detect and block committed credentials; the real
  fix for a leak is to **rotate** the secret, not just delete the commit.
- **Advisories** let you handle vulnerabilities privately, and **CodeQL code scanning**
  runs static analysis in [CI](/learn/git/github-actions/).
- All of these are **opt-in** — turn them on as the repo grows.

Next up: Module 6 opens the power tools — rewriting history precisely with [interactive rebase](/learn/git/interactive-rebase/).
