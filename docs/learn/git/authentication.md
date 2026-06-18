---
slug: authentication
title: "Authentication: SSH keys & tokens"
description: Authenticate with GitHub the right way — generate an SSH key, add it to the agent and GitHub, or create a personal access token (PAT) for HTTPS. Plus 2FA.
keywords: github authentication, ssh key, ssh-keygen ed25519, personal access token, github pat, fine-grained token, credential helper, ssh vs https, github 2fa
level: intermediate
status: full
prereq:
  - what-is-github
faq:
  - q: Why can't I use my password to push to GitHub anymore?
    a: GitHub disabled password authentication for Git operations over HTTPS in August 2021 for security reasons — passwords are reused, phished, and leaked far too easily. You now authenticate either with an SSH key or with a personal access token (PAT) used in place of a password. Both are revocable and scoped, so a leak does far less damage than a stolen password.
  - q: Should I use SSH keys or a personal access token?
    a: Either works well. SSH keys are convenient once set up — you authenticate silently with a key on your machine and never paste a token. Personal access tokens pair with HTTPS, work everywhere HTTPS does (handy behind restrictive firewalls), and can be scoped precisely. Many people use SSH on their own machines and tokens for scripts, CI, or locked-down networks.
  - q: What is the difference between a fine-grained and a classic personal access token?
    a: Classic tokens use broad scopes that apply to all your repositories at once. Fine-grained tokens, the newer and recommended type, let you grant access to specific repositories and specific permissions, and they always have an expiry. Prefer fine-grained tokens so a leaked token exposes as little as possible.
  - q: Does using an SSH key replace two-factor authentication?
    a: 'No. Two-factor authentication (2FA) protects your GitHub account login on the website, while SSH keys and tokens authenticate Git operations. They are separate layers and GitHub now requires 2FA for most accounts. Keep both: 2FA on the account, and a key or token for the command line.'
---

# Authentication: SSH keys & tokens

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
GitHub **no longer accepts your password** for Git operations, so you authenticate
one of two ways. **SSH keys**: generate a key with `ssh-keygen -t ed25519`, add the
**public** half to GitHub, and pushes just work. **Personal access tokens (PATs)**:
create a scoped, expiring token and use it in place of a password over HTTPS, cached
by a **credential helper**. Prefer **fine-grained** tokens scoped to specific repos.
You can switch a repo between SSH and HTTPS by changing its remote URL. And none of
this replaces **two-factor authentication**, which secures your account login.
</div>

In the [remotes](/learn/git/remotes/) lesson, every push and clone quietly assumed
you were allowed in. This lesson is how you prove who you are. Get it set up once and
it fades into the background — but the first time, the choice between SSH and tokens
trips people up, so let's make it clear.

## Why passwords are gone

GitHub turned off password authentication for Git over HTTPS in 2021. Passwords are
the weakest link in security: people reuse them, they get phished, and breaches leak
them in bulk. The replacements — SSH keys and personal access tokens — are stronger
because they're **revocable** (kill one without changing anything else), **scoped**
(grant only the access needed), and often **time-limited** (they expire).

You have two main options, and you can use both:

| Method | Pairs with | Best for |
|--------|-----------|----------|
| **SSH key** | SSH remote URLs | Your own machines — set once, silent thereafter |
| **Personal access token** | HTTPS remote URLs | Scripts, CI, restrictive networks |

## Generating and adding an SSH key

An SSH key comes in two halves: a **private** key that never leaves your machine,
and a **public** key you give to GitHub. Anyone with your public key can verify you
hold the private one, but can't impersonate you.

Generate a modern Ed25519 key, labelling it with your email for easy identification:

```bash
$ ssh-keygen -t ed25519 -C "you@example.com"
Generating public/private ed25519 key pair.
Enter file in which to save the key (/home/you/.ssh/id_ed25519):
Enter passphrase (empty for no passphrase):
Your identification has been saved in /home/you/.ssh/id_ed25519
Your public key has been saved in /home/you/.ssh/id_ed25519.pub
```

Accept the default path and set a passphrase if you want extra protection. Now start
the SSH agent and load the key so you don't retype the passphrase each time:

```bash
$ eval "$(ssh-agent -s)"
Agent pid 12345
$ ssh-add ~/.ssh/id_ed25519
Identity added: /home/you/.ssh/id_ed25519 (you@example.com)
```

Next, copy the **public** key (the `.pub` file — never the private one) and add it to
GitHub. On the website: **Settings → SSH and GPG keys → New SSH key**, paste the
contents of `id_ed25519.pub`, give it a title, and save.

Finally, confirm it works:

```bash
$ ssh -T git@github.com
Hi alice! You've successfully authenticated, but GitHub does not provide shell access.
```

That "does not provide shell access" line is success, not an error — it just means
SSH worked and there's nothing to log into.

## Creating and using a personal access token

If you'd rather use HTTPS, create a **personal access token** to use in place of a
password. On GitHub: **Settings → Developer settings → Personal access tokens**.

Choose **fine-grained** tokens (the recommended type) over classic ones. When
creating one you'll set:

- **Expiry** — how long it lives (shorter is safer; 90 days is common).
- **Repository access** — all repos, or just selected ones (prefer selected).
- **Permissions / scopes** — for normal pushing and pulling you want **Contents:
  Read and write**. Grant nothing more than you need.

Copy the token *immediately* — GitHub shows it only once. When you next push over an
HTTPS remote, paste the token where Git asks for a password:

```bash
$ git push
Username for 'https://github.com': alice
Password for 'https://alice@github.com': <paste your token here>
```

So you don't paste it every time, enable a **credential helper** to cache it:

```bash
$ git config --global credential.helper store    # saves to a plaintext file
$ git config --global credential.helper cache    # keeps it in memory, then forgets
```

On macOS use `osxkeychain` and on Windows `manager` for secure OS-backed storage.

## Switching a remote between SSH and HTTPS

The same repository can be reached either way; only the remote URL differs:

- SSH: `git@github.com:alice/myproject.git`
- HTTPS: `https://github.com/alice/myproject.git`

If you cloned over HTTPS but now prefer SSH (or vice versa), just point the remote at
the other URL with `git remote set-url`:

```bash
$ git remote set-url origin git@github.com:alice/myproject.git
$ git remote -v
origin  git@github.com:alice/myproject.git (fetch)
origin  git@github.com:alice/myproject.git (push)
```

No re-cloning needed — your history and branches stay exactly as they were.

## Two-factor authentication

One more layer, and it's a different one. **Two-factor authentication (2FA)** protects
your *account login on github.com* — the password plus a second factor like an
authenticator app or passkey. SSH keys and tokens, by contrast, protect *Git
operations* from the command line. They are not substitutes for each other.

GitHub now **requires 2FA** for most accounts that contribute code. Set it up under
**Settings → Password and authentication**, and save your recovery codes somewhere
safe so a lost phone doesn't lock you out. (Fuzzy on a term here? The
[glossary](/learn/git/glossary/) defines the key ones.)

<div class="knowledge-check" data-quiz data-correct-msg="Right — you give GitHub the public key and keep the private one." markdown="0">
  <p class="knowledge-check__q">Quick check: when setting up SSH authentication, which half of the key pair do you add to GitHub?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The private key (id_ed25519)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The public key (id_ed25519.pub)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Both halves, pasted together</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- GitHub **dropped password auth** for Git; use an SSH key or a personal access token.
- Generate an SSH key with **`ssh-keygen -t ed25519`**, add it to the agent, and put the
  **public** half on GitHub; test with `ssh -T git@github.com`.
- A **personal access token** stands in for a password over HTTPS; prefer **fine-grained**,
  scoped, expiring tokens and cache them with a **credential helper**.
- Switch a repo between SSH and HTTPS with **`git remote set-url`** — no re-clone needed.
- **2FA** secures your account login and is separate from (and required alongside) keys and tokens.

Next up: contributing to projects you don't own with [forking](/learn/git/forking/).
