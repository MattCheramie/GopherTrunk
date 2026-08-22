---
slug: rate-limiting-and-quotas
title: Rate limiting & quotas
description: Protecting a server from its own clients — rate limits and quotas, the 429 and Retry-After protocol, and backing off with jitter like a good citizen instead of amplifying failure.
keywords: rate limiting, API quota, 429 too many requests, Retry-After, exponential backoff, jitter, token bucket, retry storm
level: intermediate
status: full
prereq:
  - methods-and-status-codes
---

# Rate limiting & quotas

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **rate limit** caps how fast a client may call (requests per second/minute); a
**quota** caps how much over a period (per day/month). Servers enforce them
because clients — usually buggy or naive ones, not malicious — can otherwise
consume everything. The protocol is **`429 Too Many Requests`** plus a
**`Retry-After`** hint; the client's duty is **exponential backoff with jitter**.
The deeper lesson generalises: any client that retries **without backing off**
turns someone else's bad moment into an outage — its own server's, or one it's
watching.
</div>

This lesson is about the traffic rules of API citizenship — the server's defence
against being loved to death, and the client etiquette that separates a good
integration from an accidental denial-of-service. It's also where a theme lands
that returns, vividly, in [Unit 6](/learn/apis/web-console-sockets/).

## Why servers must say "slower, please"

A server's capacity is finite, and its clients' appetites aren't coordinated.
The classic offenders are rarely attackers: a polling loop with the `sleep`
accidentally deleted; a retry loop with no delay hammering a failing endpoint;
a batch job someone pointed at production; one user of a shared API crowding
out the rest. **Rate limiting** is the server pricing its capacity — and doing
so *fairly*, per client (which is one more reason
[authenticated](/learn/apis/authentication-basics/) requests matter: the limit
needs someone to attach to).

The two shapes complement each other:

| | Rate limit | Quota |
|---|-----------|-------|
| Caps | burst speed (e.g. 10 req/s) | total volume (e.g. 100k req/month) |
| Protects | servers from spikes | capacity/cost budgets, fairness between accounts |
| Typical response | `429`, clears in seconds | `429`/`403`, clears at period reset |

Under the hood most implementations are a **token bucket** — tokens drip in at
the steady rate, each request spends one, and a full bucket allows a short
burst. Bursts are normal (a page load fires five API calls at once); sustained
excess is what the bucket refuses.

## The protocol of "slow down"

When a limit trips, a well-mannered server answers precisely:

```text
HTTP/1.1 429 Too Many Requests
Retry-After: 30
X-RateLimit-Limit: 600
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1755787200

{"error": {"code": "rate_limited", "message": "600 requests/min exceeded. Retry after 30s."}}
```

`429` says *you're fine, just fast* — a client fault in the
[4xx sense](/learn/apis/methods-and-status-codes/), but uniquely retryable
after a wait. **`Retry-After`** says how long; the `X-RateLimit-*` family
(conventional, near-universal) lets a client pace itself *before* hitting the
wall — the polite client reads `Remaining` and slows early.

## Backing off like a good citizen

The client's half of the bargain, and the part worth over-learning:

1. **Honour `Retry-After`** when present — the server told you the answer.
2. Otherwise **exponential backoff**: wait 1 s, then 2, 4, 8… capped at some
   maximum. Each failure buys the server more room to recover.
3. **Add jitter** — randomise each delay (e.g. ±50%). Without it, every client
   that failed together retries together, in synchronized waves that re-crush
   the recovering server on a metronome. Jitter smears the herd.
4. **Budget the retries.** After N failures, stop and surface the error —
   infinite retry is a slow-motion outage generator.
5. **Only retry what's safe**: `429`/`503` yes; `400` never (same bytes, same
   verdict); non-[idempotent](/learn/apis/methods-and-status-codes/) requests
   only with care about duplicates.

> Rule of thumb: a retry loop without backoff, jitter, and a budget isn't
> resilience — it's an attack you'll launch at your own dependencies on their
> worst day.

This discipline is bigger than 429s. *Any* repeated attempt against a failing
peer — reconnecting a dropped [WebSocket](/learn/apis/websockets/), re-polling
a dead service, re-delivering a [webhook](/learn/apis/webhooks/) — obeys the
same law. GopherTrunk's own web console once carried the scars: a reconnect
loop that reset its backoff too eagerly could hammer the daemon several times a
second, forever, from one stale browser tab.
[Unit 6 tells that story properly](/learn/apis/web-console-sockets/); this
lesson is why it matters.

## Designing limits into your own API

If you serve an API — even a LAN daemon — decide early what "too much" means,
because retrofitting limits onto consumers who've learned to be greedy is a
[breaking change](/learn/apis/api-contracts/) in spirit. Publish the numbers,
return real `429`s with `Retry-After`, and exempt cheap reads from the budgets
of expensive writes where sensible. And remember limits are load-*shaping*, not
security: they blunt abuse but don't authenticate or authorize anyone — that's
the [next-but-two lesson's](/learn/apis/api-security/) beat.

<div class="knowledge-check" data-quiz data-correct-msg="Right — jitter desynchronizes the herd so retries don't arrive in crushing synchronized waves." markdown="0">
  <p class="knowledge-check__q">Quick check: why add random jitter to exponential backoff delays?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It makes individual clients retry sooner on average</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It hides the retry pattern from rate limiters</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It spreads out clients that failed simultaneously, so they don't all retry in the same instant and re-overload the server</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Rate limits** cap speed, **quotas** cap volume; both defend finite capacity
  from uncoordinated (usually buggy, rarely malicious) clients.
- The wire protocol is **`429` + `Retry-After`**, with `X-RateLimit-*` headers
  letting polite clients pace themselves early.
- Client etiquette: **honour Retry-After, exponential backoff, jitter, a retry
  budget**, and retry only what's safe to repeat.
- The law generalises: **any** un-backed-off retry loop — HTTP, WebSocket
  reconnects, webhooks — amplifies failure into outage.
- Serving an API? **Decide and publish limits early**; they're far harder to
  add after consumers learn greed.

Next up: [API documentation](/learn/apis/api-documentation/).
