---
slug: streaming
title: Streaming responses
description: Streaming delivers a model's output token-by-token as it's generated, so users see progress immediately instead of waiting for the whole reply. Learn how streamed responses work, why they cut perceived latency, and what gets harder.
keywords: streaming, server-sent events, token streaming, perceived latency, incremental output, chunks, typing indicator, deltas, cancellation, finish signal
level: intermediate
status: full
prereq:
  - how-a-model-api-works
faq:
  - q: What does streaming a model response actually do?
    a: "Instead of waiting for the full reply, the API sends the output in small incremental chunks — deltas — as the model generates each token. Your code accumulates the deltas and renders them as they arrive, so the user sees text appear progressively rather than all at once."
  - q: Does streaming make the model faster or cheaper?
    a: "Neither. The model generates the same tokens in the same time and you pay for the same usage. What streaming improves is perceived latency — the user sees output immediately instead of staring at a blank screen, so the wait feels far shorter even though the total time is unchanged."
  - q: What's harder when you stream?
    a: "You can't fully validate a structured result until the stream finishes, so parsing has to wait for the end. You also have to handle mid-stream errors and user cancellation, and cope with tool calls that interleave with the text. Rendering live is easy; acting on a half-finished response is where bugs creep in."
  - q: When should I not bother streaming?
    a: "When nothing shows the output to a person as it arrives — a background job, a batch pipeline, or a call whose result you only parse once it's complete. Streaming is a UX win for interactive chat and long outputs; for machine-to-machine work it just adds handling for no benefit."
---
# Streaming responses

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Streaming** delivers a model's output token-by-token as it's generated, so users see
progress the instant the model starts writing instead of waiting for the whole reply.
It's a **perceived latency** win, not a cost or speed win — the same tokens take the
same time and cost the same. Under the hood the
[API call](/learn/building-ai/how-a-model-api-works/) returns a sequence of small chunks
(deltas) that your code accumulates and renders live. See also
[cost, latency & limits](/learn/building-ai/cost-latency-and-limits/).
</div>

A model writes one token at a time, so a long answer genuinely takes several seconds to
finish. You have two choices for what the user sees during those seconds: a blank screen
until the whole reply lands, or the words appearing as they're written. Streaming is the
second option, and it's the difference between an app that feels sluggish and one that
feels alive.

## What streaming is

By default a model API call is **all-or-nothing**: you send the request, wait, and get
the complete response back in one piece. **Streaming** changes the shape of the response.
Instead of one big payload at the end, the API sends a series of **incremental chunks** —
often called **deltas** — as the model generates each piece of text. Your code reads
those chunks one by one and **accumulates** them into the growing answer.

The final text is identical either way. What differs is *when* the bytes arrive: spread
across the generation instead of dumped at the finish line.

## Why it matters

Because output is generated **token-by-token**, a paragraph-long answer takes real time
to produce — often several seconds. Without streaming, the user stares at nothing for
that entire stretch and only then sees the result. With streaming, the first words appear
almost immediately and the rest flows in as it's written.

Nothing about the total time changed. The model isn't faster and you don't pay less — you
saw in [cost, latency & limits](/learn/building-ai/cost-latency-and-limits/) that usage
and wall-clock time are set by the tokens involved. What streaming reduces is **perceived
latency**: how long the wait *feels*. Immediate, visible progress reads as responsive,
while an identical wait behind a blank screen reads as broken. For chat interfaces and any
long output, that difference is essential.

## How it works

A streamed response is a single **HTTP response that stays open** and delivers data in
pieces — commonly as **server-sent events**, a simple format where the server pushes a
sequence of small events down one connection. Each event carries a slice of the reply.

Your code does three things as the stream runs:

1. **Read** each chunk as it arrives.
2. **Append** the text delta to what you've collected so far.
3. **Render** the updated text — on screen, into a buffer, wherever it's going.

The stream ends with a **finish signal** — an event marking that generation is done,
usually alongside the **usage** report (tokens consumed) and a finish reason. Only once
you've seen that signal do you have the complete answer in hand.

## What gets harder

Streaming buys a better feel, but it costs you some simplicity.

- **You can't validate until it's done.** A partial JSON object isn't valid JSON, so you
  can't reliably [parse and validate output](/learn/building-ai/parsing-and-validating-output/)
  or enforce a [structured-output](/learn/building-ai/structured-output/) schema until the
  stream completes. Render the raw text live if you like, but act on the parsed result
  only after the finish signal.
- **Mid-stream failures need handling.** A connection can drop after the third chunk. Your
  code has to cope with a half-delivered response rather than assuming a clean success or a
  clean error.
- **Cancellation is a real state.** Users stop long answers. Stopping the stream should
  halt generation and leave your app in a sane state, not a stuck one.
- **Tool calls interleave.** When the model calls a tool, that request arrives *within* the
  stream, mixed in with text. You have to route each chunk to the right handler rather than
  treating everything as prose.

## UX patterns

The whole point is the interface, so a few habits make streaming feel right:

- **Render tokens live** as they arrive, so progress is visible from the first word.
- **Show a typing indicator** before the first chunk and while generation continues, so the
  wait reads as "working," not "frozen."
- **Offer a cancel/stop control** — long outputs are exactly where users change their mind.
- **Only act on the final result once complete.** Display the live text, but save, submit,
  or parse only after the finish signal.

## A short example

The shape of consuming a stream is the same everywhere: loop over chunks, append each
text delta, and stop at the finish signal.

```python
# Pseudo-code: consume a streamed response and print deltas as they arrive.
answer = ""
for chunk in client.stream(request):
    if chunk.type == "text_delta":
        answer += chunk.text        # accumulate
        print(chunk.text, end="")   # render immediately
    elif chunk.type == "finish":
        usage = chunk.usage         # tokens, finish reason — now it's complete
        break

# Only now is `answer` whole — safe to parse or validate.
```

The user watched the reply appear word by word; your code ended up with exactly the same
final string it would have received without streaming.

<div class="knowledge-check" data-quiz data-correct-msg="Right — streaming cuts perceived latency; the user sees output as it's generated, but total time and cost are unchanged." markdown="0">
  <p class="knowledge-check__q">Quick check: The main reason to stream a response is to reduce…</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Token cost — streaming makes the call cheaper</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Perceived latency — the user sees output as it's generated (it doesn't reduce cost)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Total generation time — the model finishes sooner</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Streaming** delivers output token-by-token as it's generated, instead of one payload at the end.
- It's a **perceived-latency** win: the same tokens take the same time and cost the same, but the wait *feels* far shorter.
- The response is a single open HTTP stream (often **server-sent events**); you read chunks, append **deltas**, and stop at the **finish signal** with usage.
- It makes some things harder: you can't validate structured output until the stream ends, and you must handle mid-stream errors, cancellation, and interleaved tool calls.
- Good UX renders tokens live, shows a **typing indicator**, offers a stop control, and only acts on the result once complete.

Next up: [multimodal — images, audio & documents](/learn/building-ai/multimodal-inputs/).
