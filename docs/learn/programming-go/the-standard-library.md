---
slug: the-standard-library
title: The standard library
description: A tour of the batteries Go ships with — io, bytes, encoding, net, and more — and why in Go you reach for an external dependency far less often than in other ecosystems.
keywords: go standard library, go stdlib, io reader, net http, encoding json, bufio, go packages, batteries included
level: intermediate
status: full
prereq:
  - packages-and-modules
faq:
  - q: Why does Go have so few dependencies compared to other languages?
    a: Its standard library is unusually complete and stable — HTTP servers, JSON, cryptography, compression, and more are all built in and maintained alongside the language. So the common building blocks are already there, and Go projects tend to have short dependency lists. Fewer dependencies means less to audit, update, and break.
  - q: What is io.Reader and why does it matter?
    a: io.Reader is a one-method interface for "something you can read bytes from." Files, network connections, byte buffers, and compressed streams all satisfy it, so any function written against io.Reader works with all of them. It's the connective tissue of the standard library and the best example of Go's small-interfaces philosophy.
---

# The standard library

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Go ships a large, stable **standard library** — `io`, `bytes`, `bufio`,
`encoding/json`, `net/http`, `os`, `time`, and much more — so you reach for an
external dependency far less than in most ecosystems. Its small interfaces (like
**`io.Reader`**) tie everything together. Knowing what's already there saves you
writing — and depending on — code you don't need.
</div>

A language is only as pleasant as its libraries. Go's standard library is
famously complete, and leaning on it is a core Go skill. This lesson maps the
pieces you'll use most.

## The packages you'll reach for constantly

| Package | Gives you |
|---------|-----------|
| `fmt` | Formatted printing and string building |
| `io` / `bufio` | Streaming reads and writes, buffering |
| `bytes` / `strings` | Working with byte slices and text |
| `os` | Files, environment, arguments |
| `encoding/json` | Turn structs to JSON and back |
| `net/http` | A full HTTP client *and* server |
| `time` | Durations, timers, timestamps |
| `errors` | Wrapping and inspecting errors |

That `net/http` line is worth dwelling on: a production-grade web server is *in the
standard library*. GopherTrunk's web interface and API build on it directly — no
framework required.

## Small interfaces, everywhere

The standard library is glued together by tiny interfaces from
[the interfaces lesson](/learn/programming-go/interfaces/). The star is `io.Reader`:

```go
func countBytes(r io.Reader) (int, error) {
    buf := make([]byte, 4096)
    total := 0
    for {
        n, err := r.Read(buf)
        total += n
        if err != nil {
            return total, err   // io.EOF marks the clean end
        }
    }
}
```

Because a file, a network socket, a `bytes.Buffer`, a gzip stream, and an SDR sample
source *all* satisfy `io.Reader`, `countBytes` works with every one of them
unchanged. Learn the handful of these interfaces — `io.Reader`, `io.Writer`,
`io.Closer` — and huge swaths of Go code suddenly read the same way.

## Fewer dependencies, on purpose

Because so much is built in and kept stable across versions, Go projects typically
have short dependency lists. That's not just tidiness: every dependency is code you
must trust, audit, and update (see the
[security](/learn/cybersecurity/secure-coding/) and
[licensing](/learn/software-licensing/auditing-dependencies/) angles). Reaching for
the standard library first is faster *and* safer.

<div class="knowledge-check" data-quiz data-correct-msg="Right — files, sockets, and buffers all satisfy io.Reader, so one function handles them all." markdown="0">
  <p class="knowledge-check__q">Quick check: why can one function that takes an <code>io.Reader</code> read from files, network connections, and byte buffers alike?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Go converts them all to files first</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">They all satisfy the io.Reader interface</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses reflection to detect each type</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- Go's **standard library** is large and stable — HTTP, JSON, crypto, and more are
  built in.
- Small interfaces like **`io.Reader`** let one function work across files, sockets,
  and buffers.
- Reaching for the stdlib first means **fewer dependencies** to audit and maintain.
- GopherTrunk's web server and API are built on `net/http` — no framework needed.

Next up: put it all together by reading a real Go codebase.
