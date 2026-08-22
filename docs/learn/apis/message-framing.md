---
slug: message-framing
title: Message framing
description: A byte stream has no natural boundaries — length prefixes, delimiters, and fixed frames compared, what happens when framing drifts, and why sync recovery separates good protocols from fragile ones.
keywords: message framing, length prefix, delimiter, TCP byte stream, frame synchronization, framing errors, protocol design, sync word
level: advanced
status: full
prereq:
  - what-is-a-protocol
  - text-vs-binary-protocols
---

# Message framing

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
TCP delivers a **byte stream**, not messages: your three writes can arrive as one
read, or five. **Framing** is the receiver's rule for finding message boundaries,
and there are three families: **delimiters** (scan for a marker, escape it in
data), **length prefixes** (read N, then N bytes), and **fixed-size frames**.
When framing **drifts** — one boundary misjudged — everything after parses as
garbage, so serious protocols add **resynchronization**: a way to find the next
true boundary. Radio protocols, which lose bits routinely, are the masterclass in
framing done defensively.
</div>

Here's the level below everything this module has used so far. HTTP, SSE,
WebSockets, gRPC — each one had to answer the same primitive question first:
*in an endless river of bytes, where does one message end and the next begin?*
This lesson is that question, and it's where API engineering meets the problems
radio engineers have fought for a century.

## The stream has no seams

A TCP connection ([TCP & UDP](/learn/networking/tcp-and-udp/)) promises bytes
in order — and *nothing* about grouping. Write `{"a":1}` then `{"b":2}`, and the
receiver may read `{"a":1}{"b":` now and `2}` later. Any code that assumes "one
read = one message" works flawlessly on localhost, then shatters under real
network conditions — one of the classic bugs of network programming. The
protocol must define boundaries; the bytes won't.

<figure class="figure" markdown="0">
<svg viewBox="0 0 520 150" role="img" aria-label="A continuous byte stream shown as one long bar; framing marks divide it into three messages; a misplaced divider makes every later message wrong." xmlns="http://www.w3.org/2000/svg">
  <text x="10" y="18" font-size="12" fill="currentColor" font-weight="bold">What TCP delivers</text>
  <rect x="10" y="26" width="500" height="22" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="260" y="42" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.7">…a continuous, seamless run of bytes…</text>
  <text x="10" y="80" font-size="12" fill="currentColor" font-weight="bold">What framing recovers</text>
  <rect x="10" y="88" width="160" height="22" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="170" y="88" width="200" height="22" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <rect x="370" y="88" width="140" height="22" fill="none" stroke="currentColor" stroke-width="1.5"/>
  <text x="90" y="104" text-anchor="middle" font-size="11" fill="currentColor">message 1</text>
  <text x="270" y="104" text-anchor="middle" font-size="11" fill="currentColor">message 2</text>
  <text x="440" y="104" text-anchor="middle" font-size="11" fill="currentColor">message 3</text>
  <line x1="170" y1="84" x2="170" y2="114" stroke="currentColor" stroke-width="2"/>
  <line x1="370" y1="84" x2="370" y2="114" stroke="currentColor" stroke-width="2"/>
  <text x="260" y="140" text-anchor="middle" font-size="11" fill="currentColor" fill-opacity="0.8">misplace one divider, and every message after it parses as garbage</text>
</svg>
<figcaption>Framing turns a seamless byte stream back into messages — and one wrong boundary poisons everything downstream until resync.</figcaption>
</figure>

## The three families

| Strategy | How it works | Example protocols | Weakness |
|----------|--------------|-------------------|----------|
| **Delimiter** | Scan for a marker byte/sequence | HTTP headers (CRLF + blank line), SSE (blank line), newline-delimited JSON | The marker must never appear in data — so data must be escaped or forbidden |
| **Length prefix** | Read a fixed-size length field, then exactly that many bytes | HTTP bodies (`Content-Length`), WebSocket frames, gRPC messages | A corrupted length desynchronizes everything after it |
| **Fixed size** | Every frame is exactly N bytes | Audio sample blocks, TDMA radio bursts, cell-based transports | Rigid — every message pays for the largest case |

Notice HTTP uses **two** of them: delimiters for the header section (human-typed
text, unbounded) and a length prefix for the body (arbitrary binary data that no
delimiter could survive). That hybrid is a sensible general design: delimiters
where content is constrained text, lengths where content is arbitrary bytes.

Delimiters carry a subtle tax — **escaping**. If `\n` ends a message, a `\n`
*inside* a message must be encoded (which is partly why newline-delimited JSON
works: JSON string escapes already guarantee no raw newlines). Forget the
escaping rule on either side and you've built a protocol that fails only on the
data that happens to contain the delimiter — a bug that hides for months.

## When framing drifts

Framing errors are uniquely vicious because they **cascade**. Misread one length
field — a corrupted byte, a bug, a version mismatch — and the receiver starts
the "next message" mid-payload. That message parses as garbage *or, worse,
parses as valid nonsense*; its misread length points somewhere else random, and
the connection is now permanently desynchronized, producing errors unrelated to
the original cause. The symptom appears far from the fault, which is what makes
framing bugs so miserable to diagnose.

Protocol designers answer with **resynchronization** strategies:

- **Tear down and reconnect.** TCP protocols mostly punt: on any framing
  anomaly, kill the connection; the fresh one starts cleanly framed. Crude,
  correct, and exactly what gRPC and WebSocket implementations do on a malformed
  frame.
- **Sync markers.** Put a distinctive pattern at each frame start; on
  confusion, scan forward for it. This is how radio lives: a P25 or DMR receiver
  finds a **frame sync word** — a bit pattern chosen for unmistakability — in a
  stream that has no connections to reset, and simply *hunts* for the next sync
  after any corruption. GopherTrunk's decoders spend serious engineering on
  exactly this: on the air, framing drift is not an edge case but the weather.
- **Self-terminating encodings** whose boundaries survive local damage —
  rarer, costlier, used where neither reset nor markers fit.

> Rule of thumb: never build your own framing on TCP if you can borrow one —
> newline-delimited JSON, WebSocket frames, or gRPC give you battle-tested
> boundaries. If you must build it, length-prefix it, bound the length
> (a 2 GB "message" is an attack, not a message), and decide the resync story
> *before* the first byte ships.

## Why API people should care

Mostly you'll consume framing others built — but the abstraction leaks upward
exactly when things break: a proxy that buffers an SSE stream is a framing
middleman misbehaving; a "connection reset" after a malformed chunk is a resync
policy firing; a hand-rolled TCP protocol between two of your own services is
you, on the hook for everything above. And if you ever look inside the digital
radio protocols GopherTrunk decodes, you'll find this lesson is their whole
lower half.

<div class="knowledge-check" data-quiz data-correct-msg="Right — after a bad length the receiver reads boundaries at wrong places indefinitely; that cascade is what makes framing errors so costly." markdown="0">
  <p class="knowledge-check__q">Quick check: a receiver misreads one length prefix in a stream. What's the consequence?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">One message is lost; the next parses normally</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Desynchronization — every subsequent "message" starts at a wrong boundary until some resync mechanism fires</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Nothing — TCP's checksums prevent misread lengths</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- TCP is a **byte stream with no message boundaries**; "one write = one read" is
  a localhost illusion.
- Framing comes in three families: **delimiters** (escape the marker!),
  **length prefixes**, and **fixed-size frames** — HTTP hybridizes the first
  two.
- Framing errors **cascade**: one wrong boundary poisons everything after it,
  with symptoms far from the cause.
- Serious protocols plan **resynchronization** — connection reset on TCP,
  **sync words** on radio, where drift is routine.
- **Borrow framing** (WebSocket, gRPC, NDJSON) rather than building it; if you
  build, bound your lengths and design the resync first.

Next up: [Schemas & code generation](/learn/apis/schemas-and-codegen/).
