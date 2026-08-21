---
slug: building-your-own-client
title: Build your own client
description: Put the whole module to work — a small Go program that queries GopherTrunk's REST API for context and follows the live event stream, with the reconnect, tolerance, and liveness disciplines built in.
keywords: build an API client, Go API client, consume SSE in Go, REST client Go, event stream client, API client best practices, capstone project
level: advanced
status: full
prereq:
  - the-daemon-rest-api
  - live-events-and-webhooks
gophertrunk_links:
  - title: API & events reference
    url: /api-events.html
    note: keep it open while you build — it is the contract your client codes against.
  - title: Web console guide
    url: /web.html
    note: the reference client — when unsure how to fetch something, watch the console do it.
---

# Build your own client

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The capstone: a small program that **GETs context from the REST API** (the
talkgroup catalogue), **follows the SSE event stream**, and announces watched
calls — pull and push combined, the shape of nearly every real integration.
The credit is earned in the disciplines: **typed but tolerant parsing**,
**dispatch-and-ignore** on event types, **reconnect with backoff that resets
only on data**, and **graceful shutdown**. Build it once against a real daemon
and the whole module's ideas become muscle memory.
</div>

Thirty-one lessons of theory; now the synthesis. The project is deliberately
small — a "call announcer" that prints a line whenever a watched talkgroup
keys up, with human labels — because every real integration you'll ever write
(alerts, dashboards, exporters, archivers) is this same skeleton wearing
different output.

## The plan

1. **Fetch context once** from `GET /api/v1/talkgroups` — ID → label mapping.
2. **Subscribe** to `GET /api/v1/events` and read SSE frames forever.
3. **On `call.start`**, look up the label and announce watched talkgroups.
4. **Survive reality**: reconnect on drop, tolerate unknown events, shut down
   cleanly on Ctrl-C.

Steps 1–3 are an afternoon. Step 4 is the module.

## The skeleton, in Go

Abbreviated to the load-bearing parts — the full version is yours to grow:

```go
func fetchTalkgroups(ctx context.Context, base string) (map[int]string, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/v1/talkgroups", nil)
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("talkgroups: unexpected status %s", resp.Status)
    }
    var body struct {
        Talkgroups []struct {
            ID    int    `json:"id"`
            Label string `json:"label"`
        } `json:"talkgroups"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
        return nil, err
    }
    labels := make(map[int]string, len(body.Talkgroups))
    for _, tg := range body.Talkgroups {
        labels[tg.ID] = tg.Label
    }
    return labels, nil
}
```

Details that are doctrine, not style: the **status check before decoding** (a
[4xx/5xx](/learn/apis/methods-and-status-codes/) body is an error shape, not a
talkgroup list); the **envelope decoded explicitly** — `{"talkgroups": [...]}`
spelled out, because the [History-panel bug](/learn/apis/the-daemon-rest-api/)
showed exactly what a guessed envelope key costs; and struct tags naming each
field, so a contract mismatch surfaces as a visible zero value rather than a
typo'd map lookup.

For the stream, read lines, assemble events, dispatch:

```go
func followEvents(ctx context.Context, base string, handle func(typ, data string)) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", base+"/api/v1/events", nil)
    req.Header.Set("Accept", "text/event-stream")
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var typ, data string
    sc := bufio.NewScanner(resp.Body)
    for sc.Scan() {
        line := sc.Text()
        switch {
        case strings.HasPrefix(line, "event: "):
            typ = strings.TrimPrefix(line, "event: ")
        case strings.HasPrefix(line, "data: "):
            data = strings.TrimPrefix(line, "data: ")
        case line == "": // blank line = event complete
            if typ != "" || data != "" {
                handle(typ, data)
            }
            typ, data = "", ""
        }
    }
    return sc.Err() // stream ended: caller decides to reconnect
}
```

That switch is the [SSE wire format](/learn/apis/server-sent-events/) parsed
by hand — twenty lines, no dependency, and now you *know* it. The handler
dispatches on type and **ignores what it doesn't recognise**:

```go
handle := func(typ, data string) {
    switch typ {
    case "call.start":
        var ev struct {
            Talkgroup int    `json:"talkgroup"`
            System    string `json:"system"`
        }
        if err := json.Unmarshal([]byte(data), &ev); err != nil {
            log.Printf("bad call.start payload: %v", err)
            return
        }
        if watched[ev.Talkgroup] {
            fmt.Printf("[%s] %s — call started\n", ev.System, labels[ev.Talkgroup])
        }
    default: // unknown event types are future features, not errors
    }
}
```

## Making it survive: the checklist

The skeleton runs; these turn it into software. Work through them as
self-assessment — each is one lesson, cashed in:

- **Reconnect with backoff.** `followEvents` returning is normal life. Wrap it
  in a loop that waits 1 s, 2 s, 4 s… capped, **resetting only after events
  flow again** — you know [exactly why](/learn/apis/web-console-sockets/), and
  jitter is [why your neighbours thank you](/learn/apis/rate-limiting-and-quotas/).
- **Refresh context.** Labels fetched at startup go stale; re-fetch on
  reconnect (a reconnect is a new world) or periodically.
- **Distinguish silence from death.** Track time-since-last-event and log a
  warning past a threshold — the [liveness discipline](/learn/apis/live-events-and-webhooks/).
- **Shut down cleanly.** Cancel the context on SIGINT and let both requests
  unwind — Go's [context plumbing](/learn/programming-go/context-and-cancellation/)
  makes this nearly free, and daemons appreciate closed connections.
- **Pin your parsing with a test.** Feed `followEvents` a canned stream and
  assert the dispatches — [failing-first](/learn/apis/testing-an-api/) when
  you find your first parsing bug.

## Where to take it

Grow this skeleton in any direction the module opened: swap the announcement
for a webhook *receiver* and compare the two integration styles from the
receiving end; add the [gRPC audio stream](/learn/apis/streaming-audio-with-grpc/)
and play the calls you announce; or give it a web face — at which point you're
building what the console is, and the
[web development module](/learn/web-dev/) is the natural next path (its
[GopherTrunk dashboard lesson](/learn/web-dev/gophertrunk-web-dashboard/)
meets you from the other side). If you'd rather run your creation as a proper
always-on service, the [deployment module](/learn/deployment/) covers turning
a program into infrastructure.

<div class="knowledge-check" data-quiz data-correct-msg="Right — unknown event types are additive evolution arriving; a tolerant client skips them and keeps working across upgrades." markdown="0">
  <p class="knowledge-check__q">Quick check: your client receives an event type it has never seen. What should it do?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Exit with an error — the contract has clearly been broken</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Reconnect, in case the stream is corrupted</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Ignore it and continue — new types are additive contract growth, and tolerance is the client's half of that bargain</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The capstone shape — **REST for context, SSE for the moment, dispatch on
  event type** — is the skeleton of nearly every real integration.
- Parsing doctrine: **check status before decoding, spell out envelopes,
  name fields explicitly, ignore unknown event types**.
- Survival doctrine: **reconnect with jittered backoff (reset on data),
  refresh context on reconnect, track liveness, shut down via context**.
- **Test the parser** with canned streams — your client has a contract too.
- You've consumed text and binary, pull and push, contract and failure — the
  module, end to end, in one small program.

That completes the APIs & Protocols path. From here, build the interfaces
people see with the [Web Development module](/learn/web-dev/), or make your
client a permanent resident with the
[Deployment & Operations module](/learn/deployment/).
