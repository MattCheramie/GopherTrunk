---
slug: secrets-and-configuration
title: Secrets & configuration management
description: Keeping API keys and passwords out of images and repositories — environment injection, secret stores, and the difference between configuration and secrets.
keywords: secrets management, secret store, api keys, environment variables secrets, config vs secrets, do not commit secrets, secret injection, deployment security
level: intermediate
status: full
prereq:
  - environments-and-config
faq:
  - q: What is the difference between configuration and a secret?
    a: "Configuration is any setting that changes how software behaves — ports, paths, log levels — and is generally safe to store in version control. A secret is configuration that must stay confidential, like an API key, password, or token. Secrets need extra handling: never committed to a repo or baked into an image, and injected only at runtime."
  - q: Why shouldn't I put secrets in my Docker image or Git repository?
    a: Anything in an image or a repo is copyable and often widely shared — pushed to registries, cloned by collaborators, kept in history forever. A secret placed there leaks to everyone with access and can't be truly removed from history. Secrets must be supplied separately at runtime so the image and repo stay safe to share.
---

# Secrets & configuration management

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Configuration** (ports, paths, log levels) is safe to keep in version control;
**secrets** (API keys, passwords, tokens) are not. Never bake secrets into an **image**
or commit them to a **repo** — they leak and can't be un-leaked. Instead **inject** them
at runtime via environment variables, a mounted file, or a **secret store**. GopherTrunk
supports pulling its API token from a separate file for exactly this reason.
</div>

Some configuration must stay confidential. This lesson is about handling secrets
without leaking them — one of the most common and costly deployment mistakes.

## Config vs secrets

Not all settings are equal:

| | Configuration | Secret |
|--|---------------|--------|
| Examples | ports, paths, log level, feature flags | API keys, passwords, tokens, certificates |
| Sensitive? | no | yes |
| Version control? | yes — track it | **never** |
| In the image? | fine | **never** |

The line matters because they're handled differently. Ordinary
[config](/learn/deployment/environments-and-config/) belongs in version control so you
know what each environment runs. Secrets must be kept *out* of everything that gets
copied around.

## Why secrets can't live in images or repos

Two places feel convenient and are both traps:

- **In the container image** — images get pushed to
  [registries](/learn/deployment/images-and-registries/) and pulled by many machines;
  a secret inside travels to all of them.
- **In the Git repo** — repositories are cloned, shared, and keep **history forever**.
  A committed secret is exposed to everyone with access and *stays* in history even
  after you delete it, so it must be rotated (changed) once leaked.

The [cybersecurity module](/learn/cybersecurity/secrets-management/) covers this failure
mode in depth; the deployment rule is simple: **secrets are injected at runtime, never
built in.**

## Injecting secrets safely

Common ways to supply a secret only when the program runs:

- **Environment variables** — set the secret in the environment the process starts in;
  the app reads it at startup. Convenient and container-friendly.
- **A mounted secret file** — put the secret in a file with tight permissions and point
  the app at it. GopherTrunk's systemd unit shows this pattern: an optional
  `EnvironmentFile` supplies a bearer token, referenced from the config as
  `api.auth.token_file` — so the token lives in a locked-down file, not in
  `config.yaml`.
- **A secret store** — a dedicated service (Docker/Kubernetes secrets, a cloud secrets
  manager, Vault) that holds secrets encrypted and hands them to authorized services at
  runtime. This is the scalable answer for larger deployments.

```ini
# GopherTrunk's systemd unit — keep the token out of config.yaml:
# EnvironmentFile=-/etc/gophertrunk/env
```

## A safety net: keep secrets out by default

Two habits prevent most leaks:

- Add secret files to [`.gitignore`](/learn/git/gitignore/) so they can't be committed
  accidentally.
- Reference secrets *by path or variable* in your committed config (like
  `token_file:`), so the config is safe to share while the secret itself stays local.

<div class="knowledge-check" data-quiz data-correct-msg="Right — secrets are injected at runtime, never baked into images or committed to repos." markdown="0">
  <p class="knowledge-check__q">Quick check: where should an API key be stored?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Hard-coded in the container image</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Committed to the Git repository for convenience</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Injected at runtime via an environment variable, mounted file, or secret store</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Configuration** is safe to version; **secrets** (keys, passwords, tokens) are not.
- Never place secrets in an **image** or a **repo** — they leak widely and stay in
  history.
- **Inject** secrets at runtime via environment variables, a mounted file, or a
  **secret store**.
- GopherTrunk reads its API token from a separate file (`token_file`), keeping it out of
  `config.yaml`.

Next up: Unit 4 — automating build, test, and release with CI/CD.
