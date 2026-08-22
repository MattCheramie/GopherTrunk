---
slug: rest-fundamentals
title: REST fundamentals
description: REST names resources with URLs and manipulates them with a small set of verbs. Learn the convention that makes unfamiliar APIs feel familiar — resources, representations, statelessness, and predictable URL design.
keywords: REST API, what is REST, RESTful API, resources and representations, REST conventions, stateless API, REST URL design
level: beginner
status: full
prereq:
  - anatomy-of-http
---

# REST fundamentals

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**REST** is a convention for shaping HTTP APIs: model your data as **resources**,
name each one with a **URL**, and manipulate them all with the same **small set of
verbs** (GET, POST, PUT, PATCH, DELETE). What travels back and forth is a
**representation** — usually JSON — of the resource, and each request is
**stateless**, carrying everything the server needs. The payoff is predictability:
learn one REST API and the next one already feels familiar.
</div>

You can build an HTTP API any way you like — and early web APIs did, each
inventing its own vocabulary of commands. **REST** won because it replaced
invention with convention. This lesson covers the handful of ideas that make a
"RESTful" API predictable, using a scanner daemon's API as the running example.

## Nouns, not verbs

The central REST move is to organise an API around **resources** — the *things*
in your domain — rather than actions. A trunking scanner's things are systems,
talkgroups, radio IDs, calls. Each gets a URL, and URLs follow a two-level rhythm:

```text
/api/v1/talkgroups          the collection of talkgroups
/api/v1/talkgroups/1201     one talkgroup
/api/v1/calls               the call history
/api/v1/calls/48213         one recorded call
```

Compare the non-REST alternative: `/getTalkgroup?id=1201`, `/deleteCall.php`,
`/fetch_radio_list`. Every such API must be memorised from scratch. With
resource-shaped URLs, the verb moves out of the path and into the HTTP **method**,
where it's one of a closed set — which is the next lesson's whole subject. The
combination is expressive with almost no vocabulary:

| Request | Meaning |
|---------|---------|
| `GET /api/v1/talkgroups` | List the talkgroups |
| `GET /api/v1/talkgroups/1201` | Fetch talkgroup 1201 |
| `PATCH /api/v1/talkgroups/1201` | Update part of it (its label, say) |
| `DELETE /api/v1/talkgroups/1201` | Remove it |

## Representations, not the thing itself

What the server sends is not the resource — it's a **representation** of it, a
JSON snapshot serialized for the trip (that's the
"[data formats](/learn/apis/data-formats/)" machinery at work). The distinction
sounds philosophical but has practical teeth: the same resource can have several
representations (JSON for programs, CSV for a spreadsheet export), and updating a
resource means sending back a modified representation, not remote-controlling the
server's memory. `Content-Type` and `Accept` headers are how the two sides agree
on which representation flows.

## Statelessness: every request stands alone

REST asks that each request carry **everything the server needs** — identity,
parameters, context. The server keeps no memory of a "conversation" between
requests: there is no "and now the next page" request, only "give me page 3,
explicitly." Why accept that discipline?

- **Any server can answer.** With no per-client conversation state, requests can
  be load-balanced freely and servers restarted mid-day — the practical scaling
  point from [clients and servers](/learn/apis/clients-and-servers/).
- **Requests are replayable and debuggable.** A stateless request pasted into
  `curl` behaves identically to the one your program sent, because nothing hidden
  differs.

Statelessness is why credentials travel *on every request* (an `Authorization`
header, not a login "session" the API remembers) — a design you'll meet properly
in [API authentication](/learn/apis/authentication-basics/).

## What REST is not

Two calibrations worth making early. First, REST is a *style*, not a standard:
there's no compliance test, and real APIs sit on a spectrum from strictly
resource-shaped to loosely "JSON over HTTP." The conventions in this lesson are
the widely-agreed core, and they're what people mean by "a REST API" in practice.
Second, REST isn't always the right shape: actions that aren't naturally
create/read/update/delete on a thing — "retune the SDR," "start a hunt" — fit
awkwardly, and [RPC](/learn/apis/what-is-rpc/) exists for exactly that shape of
problem. A well-designed system often uses both; GopherTrunk serves REST for its
records and gRPC for streaming audio, as Unit 6 shows.

The [web-dev module's REST lesson](/learn/web-dev/building-a-rest-api/) approaches
this same material from the builder's side — worth a look when you get there.

<div class="knowledge-check" data-quiz data-correct-msg="Right — REST puts the thing in the URL and the action in the method." markdown="0">
  <p class="knowledge-check__q">Quick check: which URL is the most RESTful way to expose radio ID 70233?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>/api/v1/getRadio?rid=70233</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong"><code>/api/v1/radios/fetch/70233/do</code></button></li>
    <li><button type="button" class="quiz__option" data-answer="correct"><code>/api/v1/radios/70233</code></button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- REST models an API as **resources** (nouns) named by **URLs**, in a
  collection/item rhythm like `/talkgroups` and `/talkgroups/1201`.
- Actions come from the **small closed set of HTTP methods**, not from invented
  verbs in the path.
- Requests and responses carry **representations** — serialized snapshots,
  usually JSON — negotiated via `Content-Type` and `Accept`.
- **Statelessness** means each request stands alone, which buys scalability and
  replayable debugging at the cost of carrying context every time.
- REST is a **convention, not a standard** — and not every operation fits it,
  which is why RPC still exists.

Next up: [Methods & status codes](/learn/apis/methods-and-status-codes/).
