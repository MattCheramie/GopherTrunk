---
slug: secrets-management
title: Managing secrets & keys
description: How software stores secrets — API keys, tokens, passwords, and private keys — and why keeping them out of code, using a secrets manager, applying least privilege, and rotating on a schedule decides whether one leak becomes a breach.
keywords: secrets management, API key, credentials, environment variables, secrets manager, vault, key rotation, secret scanning, private key, least privilege
level: intermediate
status: full
prereq:
  - authorization-and-access
faq:
  - q: What counts as a secret in software?
    a: A secret is any value that grants access if someone knows it — API keys, access tokens, passwords, private keys, and database connection strings all qualify. If leaking a value would let an attacker act as you or reach your data, treat it as a secret and keep it out of code.
  - q: Where should I store secrets instead of in my code?
    a: Load them at runtime from environment variables or a dedicated secrets manager, and never commit them to git. A secrets manager or vault stores them encrypted, controls who can read them, and audits access — far safer than scattering them across config files or source code.
  - q: What should I do if a secret leaks?
    a: Rotate or revoke it immediately. Assume anything committed or exposed was already captured, so deleting the commit is not enough — the old value must be made useless by replacing it with a new one and invalidating the old one.
  - q: How often should I rotate secrets?
    a: Rotate on a regular schedule, and immediately whenever you suspect exposure. Where possible use short-lived credentials that expire on their own, so a leaked value stops working within minutes or hours instead of lasting indefinitely.
---

# Managing secrets & keys

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Secrets (keys, tokens, passwords)** are the values that grant access if someone
knows them. **Keep them out of code** — load them from environment variables or a
secrets manager, never commit them — scope each one to only what it needs, and
**rotate** them on a schedule. If one ever leaks, **rotate a leaked secret
immediately** rather than just deleting the commit. Software is full of secrets,
and how you store them decides whether one leak becomes a breach.
</div>

Almost every program talks to something else — a payment provider, a database, a
cloud API — and proving "it's really me" means holding a secret. Get the storage of
those secrets right and a stray copy is a shrug; get it wrong and a single exposed
key hands an attacker the keys to everything. This builds directly on
[authorization and access](/learn/cybersecurity/authorization-and-access/): a secret
is what a system checks before it grants that access.

## What counts as a secret

A **secret** is any value that grants access if it is known. The common ones:

- **API keys** — identify your app to a service and often carry its full permissions.
- **Access tokens** — short-lived proof of a completed login or authorization.
- **Passwords** — for user accounts, service accounts, and databases.
- **Private keys** — the half of a key pair you must never share (TLS, SSH, signing).
- **Connection strings** — often bundle a host, username, and password in one line.

The test is simple: **if knowing the value lets someone in, it's a secret.** Treat it
that way even if it feels low-stakes, because attackers chain small footholds into big
ones.

## Keep secrets out of code and git

The most common mistake is the most damaging: writing a secret directly into source
code. Never **hard-code** a secret, and never **commit** one to git. Source code gets
copied, forked, shared, and pushed to public repositories — and git remembers
everything, so a secret committed once lives in the history even after you "delete" it.

Instead, keep secrets outside the code and load them at runtime:

- Read them from
  [environment variables](/learn/linux-cli/environment-variables/), which the program
  pulls in when it starts (this is how most SDKs expect their keys — see
  [your first API call](/learn/building-ai/your-first-api-call/)).
- Or fetch them from a secrets manager (below).
- Add any local secret file to your **`.gitignore`** so it can never be committed by
  accident.

The code then references a *name* like `API_KEY`, and the actual value lives somewhere
git never sees.

## Secrets managers & vaults

For anything beyond a hobby project, a **secrets manager** or **vault** is the right
home for secrets. These are dedicated stores — cloud secret managers, HashiCorp Vault,
and similar tools — built to do three things a config file cannot:

- **Hold secrets encrypted** at rest, not in plain text.
- **Control access**, so only the services and people who need a secret can read it.
- **Audit use**, recording who read which secret and when.

This beats scattering secrets across config files, spreadsheets, or chat messages,
where you lose track of how many copies exist and who can see them. With a manager,
there is one authoritative copy, guarded and logged.

## Least privilege and rotation

Two habits keep a leak small. First, **least privilege**: scope each secret to only
what it needs. An API key that just reads one report should not also be able to delete
data — the narrower a key's power, the less damage a leak does. This is the same idea
as [authorization and access](/learn/cybersecurity/authorization-and-access/), applied
to machines instead of people.

Second, **rotation**: replace secrets with fresh values on a regular schedule, so any
copy an attacker may have quietly grabbed stops working. Better still, use
**short-lived credentials** where possible — tokens that expire on their own after
minutes or hours, so there is only a brief window in which a leaked value is useful.

## When a secret leaks

When a secret is exposed — pushed to a public repo, pasted in a ticket, printed in a
log — the instinct is to delete the commit or message and move on. That is **not** the
fix. Assume anything committed or exposed was already **captured**: bots scrape public
repositories for keys within seconds, and git history and backups keep copies you
cannot reach.

The correct response is to **rotate or revoke** the secret — issue a new value and
invalidate the old one — so the exposed copy becomes worthless no matter who has it.
Deleting the commit only hides the evidence; it does not close the door.

GitHub's **secret scanning** can catch some leaks automatically, flagging known key
formats when they are pushed, and many providers can revoke a leaked key on your
behalf. Treat those alerts as urgent. See
[Git: CLI, LFS & security](/learn/git/cli-lfs-and-security/) for how secrets sneak into
a repository in the first place.

<div class="knowledge-check" data-quiz data-correct-msg="Right — rotate or revoke first; the exposed copy must be made useless." markdown="0">
  <p class="knowledge-check__q">Quick check: you accidentally commit an API key to a public repo. What must you do first?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Delete the commit and force-push over the history</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Rotate/revoke the key — deleting the commit isn't enough, since it may already be copied</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Make the repository private and hope nobody saw it</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **secret** is any value — API key, token, password, private key, connection
  string — that grants access if known.
- **Keep secrets out of code and git**; load them from environment variables or a
  secrets manager and add secret files to `.gitignore`.
- A **secrets manager or vault** stores secrets encrypted, controls access, and audits
  use — better than scattered config files.
- Apply **least privilege** so each secret can do only what it needs, and **rotate** on
  a schedule with short-lived credentials where possible.
- If a secret leaks, **rotate or revoke it immediately** — deleting the commit doesn't
  help, because it may already be copied.

Next up: [social engineering & the human layer](/learn/cybersecurity/social-engineering/)
