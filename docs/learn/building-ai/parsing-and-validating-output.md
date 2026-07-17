---
slug: parsing-and-validating-output
title: Getting usable output back
description: A model returns free text, but your program needs data. Learn to parse the model's output into a value, validate that value's shape and sanity, and retry or repair on failure so an AI feature is actually reliable.
keywords: parsing LLM output, output validation, JSON parsing, retry, repair, schema validation, guardrails, robust output, well-formed but wrong, fallback
level: intermediate
status: full
prereq:
  - prompts-as-code
faq:
  - q: How do I turn a model's text reply into data my code can use?
    a: Ask for a specific, parseable format — JSON or a simple delimited format — then parse that text into a value. Wrap the parse in error handling, because the model can add stray prose or malformed syntax. The robust option is a provider's structured-output mode, which constrains the reply to your schema.
  - q: "The model returned valid JSON but the values are wrong. What went wrong?"
    a: "Parsing only proves the output is *well-formed*, not that it is *correct*. A model can return a perfectly shaped object full of wrong or nonsensical values. That is why you validate the values themselves — types, ranges, allowed choices — on top of parsing, especially before taking any action on them."
  - q: "What should I do when parsing or validation fails?"
    a: "Retry the call, or feed the bad output back and ask the model to fix it — but cap the number of attempts and have a fallback for when it still fails. Log every failure so you can see how often it happens and why. An uncapped retry loop can burn cost and latency without ever succeeding."
  - q: Does structured output mean I can skip validation?
    a: No. A structured-output or schema-constrained mode guarantees the *shape* of the reply, which removes most parse failures — but it does not guarantee the values are sane or correct. You still check values before you trust or act on them.
---

# Getting usable output back

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
The model returns **free text**, but your program needs **data**. Three steps
turn one into the other: **parse** the text into a value, **validate** that the
value's shape and content are right, and **retry or repair** when either fails.
This is the plumbing that makes an AI feature reliable rather than flaky — a model
is a [probabilistic component](/learn/building-ai/llm-as-a-component/), so its
output needs handling. The most robust way to get parseable text is a provider's
[structured-output mode](/learn/building-ai/structured-output/), covered next.
</div>

You've learned to write a prompt as [code you can maintain](/learn/building-ai/prompts-as-code/).
Now comes the other half of the round trip: getting the answer back in a form your
program can actually use. The model hands you a string; your feature needs a number, a
field, a decision. Bridging that gap — reliably, even when the model misbehaves — is
what separates a demo from something you can ship.

## Text out, data in

A model produces a **string**. That's all it ever gives you: characters. But your code
rarely wants characters — it wants a *value*. A price to store, a category to branch on,
a list of items to loop over, a yes-or-no to act on. Somewhere between the reply and the
rest of your program, that text has to become a typed value.

This is where features quietly break. The output *looks* right when you read it in a
console, and you assume it will slot into your code just as cleanly. But "looks right to
a human" and "parses cleanly in code" are different claims. A stray sentence before the
JSON, a trailing comma, a number written as words — any of these turns a plausible reply
into a parse error at exactly the wrong moment. The gap between looks-right and
is-parseable is where you do the work.

## Ask for a parseable format

The first move is to stop hoping and start instructing. Tell the model to reply in one
specific, machine-readable format — most often **JSON**, sometimes a simpler
delimited or tagged format — and to return *only* that. A prompt that says "answer in
prose" gets you prose to re-read by hand; a prompt that says "return a JSON object with
fields `category` and `confidence`, and nothing else" gets you something you can parse.

Two cautions. First, models love to be helpful, so they often wrap the JSON in extra
prose — "Sure! Here's the data:" before it, a friendly note after. Your parser will
choke on that, so either forbid it firmly in the prompt or strip it before parsing.
Second, prompt instructions are a *request*, not a guarantee — the model can still stray.
The robust, provider-supported answer is a **structured-output mode** that constrains the
reply to a schema at generation time; the [next lesson](/learn/building-ai/structured-output/)
is entirely about it. Reach for that when it's available; treat prompt-only formatting as
the fallback.

## Validate before you trust

Parsing tells you the output is **well-formed**. It does not tell you the output is
**right**. A model can hand you syntactically perfect JSON whose contents are nonsense —
a confidence of 1.7 on a 0-to-1 scale, a category you never offered, a date in the far
future, an empty string where you needed a name. Right shape is not the same as right
content, and a model — being a [probabilistic component](/learn/building-ai/llm-as-a-component/)
— produces plenty of the former without the latter.

So validation has two layers. First check the **shape**: are the expected fields present,
and are their types what you expect? Then check the **sanity** of the values: is the
number in range, is the choice one of the allowed options, is the string non-empty? Only
output that passes both earns the right to flow into the rest of your program.

This matters most right before you *do* something with the output. If the value only gets
displayed, a bad one is a cosmetic glitch. If it triggers an action — a database write, a
tuning command, a message sent — a bad value causes real damage, so validate hardest
there. When the model's output drives a [tool or function call](/learn/building-ai/tool-use-and-function-calling/),
this validation step is your guardrail.

## The retry / repair loop

When parsing or validation fails, you have options better than crashing. There are two
common recoveries:

- **Retry** — call the model again. Because output is probabilistic, a fresh attempt
  often succeeds where the last one didn't, especially with a slightly firmer prompt.
- **Repair** — feed the bad output back to the model along with the error ("this failed
  to parse / this field was out of range — return corrected JSON") and let it fix its own
  mistake.

Two rules keep this from going wrong. **Cap the attempts** — two or three, not an
unbounded loop that can burn cost and latency forever. And **have a fallback** for when
all attempts fail: a default value, a graceful error to the user, a handoff to a human —
anything but an unhandled crash. Finally, **log every failure**, so you can see how often
this happens and why; that record is the raw material for
[observability and monitoring](/learn/building-ai/observability-and-monitoring/) later.

## A robust pattern

Put the pieces in order and the shape of reliable output handling falls out: call,
extract the text, parse it, validate it, and on any failure repair or retry up to a
capped number of times — then fall back.

```python
for attempt in range(MAX_ATTEMPTS):
    reply = call_model(prompt)          # 1. call
    text  = extract_text(reply)         # 2. get the string
    try:
        value = parse(text)             # 3. parse -> a value
        validate(value)                 # 4. shape + sanity checks
        return value                    # success
    except (ParseError, ValidationError) as err:
        log_failure(attempt, text, err) # record what went wrong
        prompt = repair_prompt(text, err)  # ask it to fix, then retry

return fallback()                       # 5. all attempts failed
```

The exact library and syntax vary, but every reliable AI feature has this skeleton
underneath: parse, validate, bounded retry, fallback.

## Don't over-trust structured output

Structured-output modes are a genuine upgrade — they make malformed JSON largely go away
by constraining generation to your schema. But they solve the *shape* problem, not the
*correctness* problem. A schema-constrained reply is guaranteed to have the right fields
and types; it is **not** guaranteed to have sensible values. The model can still return a
confidence of 2.0 or a category that's valid JSON but wrong for the situation.

So keep the value-level checks even when the format is guaranteed. Format guarantees are
not correctness guarantees. Structured output removes one whole class of failures; your
validation still has to catch the rest.

<div class="knowledge-check" data-quiz data-correct-msg="Right — parse and validate, retry or repair with a cap and a fallback, and prefer a structured-output mode where you can." markdown="0">
  <p class="knowledge-check__q">Quick check: parsing the model's JSON occasionally throws in production. What's the best design?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Wrap the parse in a try/catch and silently ignore failures</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Validate and retry/repair with a capped fallback — and prefer a structured-output mode</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Assume valid JSON always parses and remove the error handling</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- The model returns a **string**; your code needs a **value**. Parsing bridges that gap,
  and it's where features quietly break.
- **Ask for a parseable format** (JSON or a delimited format) and only that — and beware
  extra prose wrapped around the data.
- **Validate before you trust**: check the shape *and* the sanity of the values, because
  well-formed is not the same as correct. Validate hardest before taking any action.
- **On failure, retry or repair** — but cap the attempts, keep a fallback, and log every
  failure.
- **Structured output** removes most parse failures but not value-level errors, so keep
  the sanity checks even with a guaranteed schema.

Next up: Module 3 makes output reliable by design — [structured output & JSON](/learn/building-ai/structured-output/).
