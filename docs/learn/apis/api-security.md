---
slug: api-security
title: API security
description: Authentication, authorization, TLS, input validation, and least privilege — the layered checklist that keeps an exposed API endpoint from becoming an incident.
keywords: API security, API security checklist, input validation, TLS API, least privilege, injection attacks, object-level authorization, defense in depth
level: intermediate
status: full
prereq:
  - authentication-basics
  - error-handling
---

# API security

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
An API is a **machine-facing attack surface**: every endpoint will eventually be
called by something hostile, with inputs no honest client would send. The
defensive checklist is layered — **TLS** on everything, **authentication** on
every route by default, **authorization checked per object** (the most-missed
layer), **validate all input** as data rather than trusting it, **leak nothing**
in errors, and **rate-limit** the abuse paths. No single layer suffices; the
design stance is **defense in depth** plus **least privilege**.
</div>

Unit 5's checklist lesson. You've met the pieces — auth in
[Unit 2](/learn/apis/authentication-basics/), leakage in
[error handling](/learn/apis/error-handling/), limits in
[rate limiting](/learn/apis/rate-limiting-and-quotas/) — and this lesson
assembles them into the walkthrough you run before anything listens beyond
localhost. The [cybersecurity module](/learn/cybersecurity/security-for-developers/)
extends everything here.

## Think like the traffic you'll actually get

The mental shift that produces secure APIs: your endpoints will not only be
called by your well-behaved client. They'll be called by **scanners** probing
every IP for known paths (watch any internet-facing server's logs for an hour),
**scripts** replaying and mutating captured requests, and **curious users**
editing IDs in URLs. Machine-facing surfaces invite machine-scale abuse — no
UI to nudge people toward valid input, and perfect automation of anything that
works once. Design each endpoint asking "what does this do for a hostile
caller?", not "what does the console send?"

## The checklist, layer by layer

**1. TLS, everywhere it leaves the machine.** Plain HTTP means every network
hop reads your tokens and payloads. [TLS](/learn/networking/tls-and-https/) is
table stakes the moment traffic crosses a wire you don't own — and cheap enough
to just use always.

**2. Authenticate by default.** Every route requires credentials *unless
deliberately excepted* — deny-by-default, so a forgotten route is a closed
route, not an open one. The LAN-only daemon exception from
[the auth lesson](/learn/apis/authentication-basics/) is legitimate exactly as
long as the network boundary holds; re-run this checklist the day anything is
[exposed](/learn/networking/exposing-a-service-safely/).

**3. Authorize per object — the famous gap.** Authentication says who; each
request still needs "may *this* identity touch *this* resource?" The classic
API vulnerability is exactly here: `GET /api/v1/users/1041/recordings`
authenticates fine, then serves user 1041's data to user 2077 who simply
edited the number. Every ID-bearing endpoint must check ownership/permission
against the *requested object*, not just validate the token.
([Authorization & access](/learn/cybersecurity/authorization-and-access/) goes
deeper.)

**4. Validate every input as data.** Types, ranges, lengths, formats — checked
server-side, rejected with clean
[400s](/learn/apis/error-handling/). And never *compose* raw input into
commands: concatenating a query parameter into SQL, a shell command, or a file
path is how **injection** happens — the attacker's input stops being data and
starts being code. Parameterised queries and strict path handling close the
class ([web application attacks](/learn/cybersecurity/web-application-attacks/)
tours the gallery). Bound sizes too: a 2 GB body or a
`limit=10000000` is a resource attack answered by caps, not effort.

**5. Leak nothing.** Error responses reveal your input's problems, never your
internals — stack traces, paths, versions, SQL live in server logs behind a
`request_id`. Watch list endpoints too: returning *every* field of an internal
record often over-shares by accident; emit deliberate shapes
([design lesson](/learn/apis/designing-a-good-api/), meet security).

**6. Rate-limit the abuse paths.** [Limits](/learn/apis/rate-limiting-and-quotas/)
aren't authentication, but they turn "try a million tokens per hour" into "try
sixty" — brute force, scraping, and enumeration all get dramatically less
practical with a bucket in the way.

> Rule of thumb: least privilege at every layer — tokens scoped to the minimum,
> handlers reaching only the data they serve, the process running as an
> unprivileged user. Then assume any one layer fails, and let the next one
> catch it: that's defense in depth
> ([the concept's own lesson](/learn/cybersecurity/defense-in-depth/)).

## A worked probe

What the checklist looks like from the outside — three requests an attacker
sends your scanner API within minutes of finding it:

```bash
curl -sk https://target/api/v1/calls            # unauthenticated read? (layer 2)
curl -sk https://target/api/v1/users/1/tokens   # other users' objects? (layer 3)
curl -sk "https://target/api/v1/calls?limit=99999999&q='+OR+1=1--"  # caps? injection? (layer 4)
```

Every one should die cleanly — `401`, `403`/`404`, `400` — with boring,
uninformative bodies. Running exactly these probes against your own API before
strangers do is the cheapest security testing there is, and it slots naturally
into the [test suite](/learn/apis/testing-an-api/) you're about to build.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a valid login says who you are; the object-level check decides whether call 999 is yours to fetch." markdown="0">
  <p class="knowledge-check__q">Quick check: an authenticated user fetches <code>/api/v1/recordings/999</code> — a recording belonging to a different account — and gets it. Which layer failed?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Authentication — the token should have been rejected</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Object-level authorization — identity was verified but never checked against the requested resource</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Input validation — 999 should not have parsed as an ID</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- APIs face **machine-scale hostile traffic** — design each endpoint for the
  attacker's call, not just the console's.
- The layers: **TLS** always; **authenticate by default**; **authorize per
  object** (the most-missed check); **validate input as data** — never compose
  it into commands; **leak nothing** in errors; **rate-limit** abuse paths.
- **Least privilege** everywhere; **defense in depth** because single layers
  fail.
- **Probe yourself first** — three curl commands find the classic holes before
  strangers do.

Next up: [Testing an API](/learn/apis/testing-an-api/).
