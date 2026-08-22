---
slug: authentication-basics
title: API authentication
description: How a server knows who's calling — API keys, bearer tokens, and sessions compared, the Authorization header, why keys never belong in URLs, and why authentication without TLS is theatre.
keywords: API authentication, API key, bearer token, Authorization header, token vs session, API security, 401 vs 403, TLS
level: intermediate
status: full
prereq:
  - anatomy-of-http
  - urls-queries-and-bodies
---

# API authentication

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Authentication** answers "who is calling?"; **authorization** answers "what may
they do?" — keep the words separate. APIs mostly authenticate with a credential on
**every request** (statelessness at work): an **API key** or **bearer token** in
the `Authorization` header. Bearer credentials are exactly what the name says —
**whoever bears one, is you** — so they must travel only over **TLS**, never in
URLs, and be revocable. Sessions and cookies are the browser-world sibling of the
same idea.
</div>

Every request so far assumed the server answers anyone. Real APIs need to know who
is asking — to protect data, attribute actions, and enforce limits. This lesson
covers the mechanics; its sibling in Unit 5,
[API security](/learn/apis/api-security/), covers the wider defensive checklist,
and the [cybersecurity module](/learn/cybersecurity/authentication-basics/) treats
authentication as its own discipline.

## Two words that must not blur

- **Authentication** ("authn") — establishing identity: this request comes from
  Matt's monitoring script.
- **Authorization** ("authz") — establishing permission: Matt's script may read
  call history but not delete recordings.

HTTP's status codes encode the difference: `401 Unauthorized` really means
*unauthenticated* (who are you? — often fixable by supplying credentials), while
`403 Forbidden` means *unauthorized* (I know who you are; the answer is no).
A client that treats them identically will re-prompt for credentials that were
never the problem.

## The credential travels on every request

Recall [statelessness](/learn/apis/rest-fundamentals/): the server keeps no
conversation memory, so identity must ride along each time, in the header built
for it:

```text
GET /api/v1/calls HTTP/1.1
Host: scanner.local:8080
Authorization: Bearer gtk_9f2c81aa74e04d5c
```

The common schemes, in ascending sophistication:

| Scheme | What the credential is | Typical use |
|--------|------------------------|-------------|
| **API key** | A long random string identifying one client/account | Server-to-server, hobby scripts, per-client attribution |
| **Bearer token** | A string obtained by logging in or a grant flow; often expiring | Most modern APIs; OAuth-style delegated access |
| **Session cookie** | A random ID referencing login state kept server-side | Browsers and web apps ([sessions in web-dev](/learn/web-dev/authentication-and-sessions/)) |

The lines blur — an API key is really a bearer token that never expires — and
some tokens (JWTs) carry signed claims inside them, a rabbit hole the
[cookies, tokens & JWT lesson](/learn/web-dev/cookies-tokens-jwt/) explores. The
operational differences that matter here: **expiry** (short-lived tokens limit the
damage window of a leak) and **revocability** (can the server kill a stolen
credential without killing all of them?).

## "Bearer" means what it says

A bearer credential is like cash: possession *is* proof. No signature, no
challenge — whoever presents the string is treated as its owner. Three rules
follow directly:

1. **TLS always.** Over plain HTTP, every hop on the network can read the header
   and become you. Authentication without
   [TLS](/learn/networking/tls-and-https/) is theatre.
2. **Never in URLs.** `?api_key=...` puts the credential in server logs, proxy
   logs, and browser history — the [previous lesson's](/learn/apis/urls-queries-and-bodies/)
   rule, now with its full justification. Headers are not logged by default; URLs
   are.
3. **Never in source control.** Keys live in environment variables or secret
   stores, not code — a habit the
   [secrets-management lesson](/learn/cybersecurity/secrets-management/) turns
   into a system.

> Rule of thumb: treat a bearer token exactly like the password it replaces —
> because to the server, it is one.

## What does the server do with it?

On each request the server looks up the credential, resolves it to an identity,
then applies authorization — which of its endpoints and resources this identity
may touch. Good APIs scope credentials narrowly: a key for your dashboard widget
needs *read calls*, not *administer the daemon*. When you eventually mint keys for
your own services, make read-only the default and privileged scopes explicit —
the principle of least privilege, in API clothing
([authorization & access](/learn/cybersecurity/authorization-and-access/) goes
deeper).

A local daemon on a trusted LAN — a GopherTrunk box on your shelf — often runs
with no authentication at all, and that's a *deliberate* trade: the network
boundary is the control. The moment such a service is exposed beyond the LAN, the
calculus flips; the
[exposing a service safely lesson](/learn/networking/exposing-a-service-safely/)
is the right companion before any port-forwarding adventure.

<div class="knowledge-check" data-quiz data-correct-msg="Right — 403 means authentication succeeded but this identity lacks permission; new credentials won't change the answer." markdown="0">
  <p class="knowledge-check__q">Quick check: your script's token is valid, but a DELETE returns <code>403 Forbidden</code>. What's going on?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The token expired — log in again and retry</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">You're authenticated but not authorized — this identity may not delete</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The server is down — retry later with backoff</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Authentication = who; authorization = what** — and `401` vs `403` encode
  exactly that split.
- Stateless APIs carry the credential **on every request**, in the
  `Authorization` header.
- **API keys, bearer tokens, sessions** are one family; expiry and revocability
  are the operational differences that matter.
- **Bearer = possession is proof**: TLS always, never in URLs, never in source
  control, treat like a password.
- Scope credentials **narrowly** (least privilege), and treat "no auth on a
  trusted LAN" as a boundary decision that must be revisited before exposure.

Next up: [API versioning](/learn/apis/api-versioning/).
