---
slug: api-versioning
title: API versioning
description: How APIs evolve without breaking their clients — additive change first, URL and header versioning when a break is unavoidable, and deprecation done politely with real timelines.
keywords: API versioning, breaking changes, v1 v2 API, URL versioning, deprecation policy, backward compatibility, semantic versioning API
level: intermediate
status: full
prereq:
  - api-contracts
---

# API versioning

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Versioning is the escape hatch for the day a **breaking change** becomes
unavoidable. The best version bump is the one you never make — **additive change**
handles most evolution — but when a break must happen, a **version marker** (most
visibly a `/v1/` path segment) lets old and new contracts run **side by side**.
**Deprecation** is the polite retirement of the old one: announce, warn in-band,
give a real timeline, then remove. A version number is a promise of stability
*within* that version.
</div>

[API contracts](/learn/apis/api-contracts/) established that breaking changes are
the cardinal sin — and that sometimes they're necessary anyway. Versioning is how
mature APIs commit that sin without taking their consumers down with them.

## First resort: don't version, add

Most evolution needs no version bump at all. New optional field? Add it — tolerant
clients ignore what they don't know. New capability? New endpoint. New input
option? Optional, with a default matching the old behaviour. An API can grow for
years this way, and the best ones do: staying on `v1` for a decade is a sign of
good initial design and disciplined evolution, not stagnation.

> Rule of thumb: a version bump is for changes that would break a correct
> existing client — nothing else. Additive growth is free.

## When you must break: mark the version

The moment two incompatible contracts must coexist, clients need a way to say
which one they mean. Three conventions exist:

| Style | Looks like | Trade-off |
|-------|------------|-----------|
| **URL path** | `/api/v1/calls` vs `/api/v2/calls` | Visible, curl-able, cacheable, log-friendly — the pragmatic default |
| **Header** | `Api-Version: 2026-08-01` | Keeps URLs "pure"; invisible in logs and pasted URLs |
| **Query param** | `/api/calls?version=2` | Easy but mixes versioning into the options namespace |

The URL style dominates in practice for a reason you already know from
[URLs, queries, and bodies](/learn/apis/urls-queries-and-bodies/): everything in
the URL is visible, shareable, and debuggable. GopherTrunk's daemon serves
`/api/v1/…` in exactly this spirit — the `v1` is a namespace for one contract,
and a hypothetical incompatible redesign would live at `/api/v2/` while `v1`
kept answering.

Note what a major version marker means: **stability within, freedom between.**
Inside `v1`, only additive change; between `v1` and `v2`, anything. This is the
API cousin of semantic versioning's major number, minus the minor/patch
machinery — most APIs find one number is enough.

## Running two versions is the real cost

The version marker is the easy part. The bill arrives afterwards: every request
now enters one of *two* contracts that both must be correct, tested, documented,
and supported — usually backed by one implementation translating at the edges.
That ongoing tax is why seasoned designers treat `v2` as a last resort, and why
"let's just clean up the field names in v2" is how APIs end up with a v1 that
lives forever anyway (its clients had no reason to move) *plus* a v2 to maintain.
Breaks need to buy something substantial.

## Deprecation: retiring a contract politely

Old versions can't live forever, and turning one off is a process, not an event:

1. **Announce** — changelog, docs, direct channels — with a concrete end date.
   Months, not weeks; consumers have their own release cycles.
2. **Warn in-band.** Responses from deprecated endpoints can carry a
   `Deprecation`/`Sunset` header or a documented warning field — the only channel
   *guaranteed* to reach the people actually calling.
3. **Measure.** Traffic on the old version tells you who hasn't moved; if you can
   identify callers (via [authentication](/learn/apis/authentication-basics/)),
   contact them.
4. **Remove on the announced date** — returning a clear error (`410 Gone` is the
   status made for it), not a silent 404 or, worse, wrong answers.

A deprecation done this way is a feature of a trustworthy API; done abruptly,
it's the event that teaches your consumers to fear every update.

## Version the contract, not the software

One last distinction: the API version is not the software's release number.
GopherTrunk the program ships releases continuously; the daemon's API stays `v1`
throughout, because the *contract* hasn't broken. Coupling the two — bumping the
API because the app hit 3.0 — inflicts migration work on clients who gain
nothing. Let the marketing version and the contract version live separate lives.

<div class="knowledge-check" data-quiz data-correct-msg="Right — additive changes don't break clients, so no new version is needed." markdown="0">
  <p class="knowledge-check__q">Quick check: you're adding a new optional <code>site</code> field to call records. What does versioning require?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A v2, since the response shape is changing</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A deprecation notice for the v1 call record</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Nothing — it's additive, and tolerant clients ignore unknown fields</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Additive change first** — most evolution never needs a version bump, and a
  long-lived `v1` is a compliment.
- A **version marker** (usually a `/v1/` path segment) namespaces one contract so
  incompatible ones can run **side by side**.
- Major version = **stability within, freedom between**; one number usually
  suffices.
- The true cost of `v2` is **maintaining both** — break only for substantial
  gains.
- **Deprecate politely**: announce with dates, warn in-band, measure stragglers,
  then remove with a clear `410`.
- The API version tracks the **contract**, not the software release.

Next up: [Polling vs push](/learn/apis/polling-vs-push/).
