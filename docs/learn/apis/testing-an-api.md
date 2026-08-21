---
slug: testing-an-api
title: Testing an API
description: Exercise endpoints with curl, then pin the contract with automated tests using Go's httptest — status codes, shapes, and error paths asserted so clients never get surprised.
keywords: API testing, httptest Go, curl testing, contract tests, integration tests, test error paths, regression test API
level: intermediate
status: full
prereq:
  - anatomy-of-http
  - error-handling
gophertrunk_links:
  - title: API & events reference
    url: /api-events.html
    note: the daemon's documented API surface — the kind of contract these tests exist to pin.
---

# Testing an API

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
API tests **pin the contract**: they assert status codes, response shapes, and
error behaviour so that a change that would surprise clients fails the build
first. Start manual — **curl** is exploration — then automate with **`httptest`**
(Go's standard tool for running handlers against real HTTP requests in-process).
Test the **sad paths** as thoroughly as the happy ones, and make every bug fix
start with a **failing test that reproduces it** — a green test you never saw
fail proves nothing.
</div>

Unit 5 closes where engineering discipline lives. You've designed the contract,
the errors, the limits; tests are how those promises stay true while the code
underneath churns. This lesson moves from by-hand curl checks to an automated
suite, with Go as the demonstration language
(the [testing in Go lesson](/learn/programming-go/testing-in-go/) covers the
framework itself).

## Manual first: curl as a laboratory

Exploration precedes automation. With a daemon running, interrogate the
contract by hand — happy paths, then deliberately broken ones:

```bash
curl -i http://scanner.local:8080/api/v1/talkgroups          # 200? shape?
curl -i http://scanner.local:8080/api/v1/talkgroups/999999   # 404? error shape?
curl -i "http://scanner.local:8080/api/v1/calls?limit=-5"    # 400? message helpful?
curl -i -X DELETE http://scanner.local:8080/api/v1/calls/1   # allowed? 405?
```

`-i` shows status and headers — always look at them, not just the body. This
loop is how you *learn* an API's actual behaviour; its weakness is that it
proves things only about *today*. Tomorrow's refactor can break any of it
silently. Automation is the act of writing today's observations down as
permanent assertions.

## Automating with httptest

Go's standard library makes API testing unusually pleasant: `net/http/httptest`
runs your real handlers against real HTTP requests **in-process** — no port, no
separate server, no flakiness:

```go
func TestGetTalkgroupNotFound(t *testing.T) {
    srv := httptest.NewServer(api.Handler(testStore()))
    defer srv.Close()

    resp, err := http.Get(srv.URL + "/api/v1/talkgroups/999999")
    if err != nil {
        t.Fatal(err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNotFound {
        t.Fatalf("status = %d, want 404", resp.StatusCode)
    }
    var e struct {
        Error struct{ Code string } `json:"error"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
        t.Fatalf("error body is not valid JSON: %v", err)
    }
    if e.Error.Code != "not_found" {
        t.Errorf("error.code = %q, want %q", e.Error.Code, "not_found")
    }
}
```

Note what's asserted: the **status code**, that the error body **parses as the
documented shape**, and the **stable error code** — exactly the three layers
the [error-handling lesson](/learn/apis/error-handling/) said programs depend
on. That's a **contract test**: it doesn't care how the handler finds
talkgroups, only that the promise to clients holds. Implementation refactors
sail through it; contract breaks die in CI. (Building the handlers themselves
is the web-dev module's
[REST API lesson](/learn/web-dev/building-a-rest-api/).)

## What to test, in priority order

1. **Every endpoint's happy path** — status, `Content-Type`, required fields
   present with right types.
2. **The documented sad paths** — missing auth → 401, someone else's object →
   403/404, garbage input → 400 *with the standard error shape*. Untested error
   paths are where drift hides longest, because no human exercises them daily.
3. **Contract invariants** — pagination honours `limit`, defaults apply,
   unknown query params don't 500, timestamps parse as ISO 8601.
4. **The security probes** — the three hostile curls from the
   [security lesson](/learn/apis/api-security/), automated so the holes can't
   quietly reopen.

> Rule of thumb: when a bug is reported, write the test that **fails because of
> the bug first**, then fix until it passes. A test written after the fix — 
> that you never watched fail — may be passing for the wrong reason, silently
> testing nothing. Failing-first is the only proof the test can actually catch
> the regression it exists to prevent.

That failing-first discipline is doubly vital for APIs because of a trap worth
naming: **self-consistent tests**. If your test *encodes* a request with the
same helper the server uses to *decode* it, both sides can share a bug and pass
green — you've tested that the code agrees with itself, not that it honours the
contract. Prefer asserting against **literal** expected bytes/JSON (as the
example above does), and against captured real traffic where you can get it.

## Testing what streams

Real-time surfaces are testable too, with one adjustment: give every read a
**deadline**. An SSE test connects, asserts the `text/event-stream` header,
triggers an event, and reads until the expected `data:` line — failing on a
timeout rather than hanging CI forever. WebSocket tests dial, exchange one
message pair, and assert the close behaviour. The discipline transfers
unchanged: pin the observable protocol, not the internals — a lesson
[Unit 6's client project](/learn/apis/building-your-own-client/) will apply
from the consumer's side.

<div class="knowledge-check" data-quiz data-correct-msg="Right — only a test you watched fail on the bug is proven able to detect it." markdown="0">
  <p class="knowledge-check__q">Quick check: why write the regression test before fixing the bug it covers?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It's faster — the test and fix share a commit</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Watching it fail proves it detects the bug — a test first seen green may be passing vacuously</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Test frameworks require tests to exist before code changes</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **curl explores; tests pin.** Manual checks prove today — automated
  assertions protect tomorrow.
- **`httptest`** runs real handlers against real requests in-process — fast,
  flake-free contract tests.
- Assert the **contract**: status, shape, stable error codes — and test **sad
  paths** as hard as happy ones.
- **Failing-first** bug tests are the only ones proven to catch their
  regression; beware **self-consistent** tests that check code against itself.
- Streams are testable too — **deadline every read**, assert the observable
  protocol.

Next up: [GopherTrunk's REST API](/learn/apis/the-daemon-rest-api/).
