---
slug: building-vs-using-ai
title: What it means to build AI into software
description: Building AI into software means calling a model from your own program so it powers a feature your users touch — not chatting with an assistant yourself. Learn what an AI feature is, its pipeline, and why it's software engineering, not machine learning.
keywords: build AI features, AI feature, LLM app development, model API, using vs building AI, AI in software, calling a model, AI engineering, software engineering, model call at runtime, AI pipeline
level: beginner
status: full
faq:
  - q: "What's the difference between using AI and building with it?"
    a: "Using AI means you sit at a chat window or a coding assistant and read the answers yourself. Building with it means a model call runs *inside* your program, at runtime, producing a result your software then uses — and your users benefit from it without ever talking to the model directly."
  - q: What is an AI feature?
    a: An AI feature is an ordinary product capability that happens to be powered by a model call. Summarizing text, pulling structured fields out of a document, answering a question from your own docs, classifying an item, or drafting a reply are all AI features. To the user it's just a button or a result; the model is an implementation detail.
  - q: "Do I need a machine-learning background to build AI features?"
    a: "No. You are integrating a model through its API, not training one. There is no math or ML theory to learn first — the work is ordinary software engineering (calling an API, handling its input and output, dealing with failure) applied to an unusual component that returns text instead of a fixed value."
gophertrunk_links:
  - title: Architecture overview
    url: /architecture.html
    note: how GopherTrunk is organized into components you could add an AI feature to.
---
# What it means to build AI into software

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Using vs building** — *using* AI is you sitting at a chat window; *building* with
it means a model call runs inside your own program. **An AI feature** is a normal
product capability (summarize, extract, classify, answer) powered by that call — the
model runs at runtime, serving your users, who never touch it directly. **It's
software engineering** — you're integrating an API, not training a model, so no ML
background is needed. If you haven't seen how models work under the hood, the sibling
path [AI in software development](/learn/ai-software-dev/what-is-ai-for-coding/)
covers that first.
</div>

You have almost certainly *used* AI by now — asked a chatbot a question, let a coding
assistant finish a line. This path is about something different: making a model a part
of the software **you** ship, so it does a job for **your** users. This first lesson
draws that line clearly and names the thing you'll be building for the rest of the
path: an *AI feature*.

## Using AI vs. building with it

When you **use** AI, you are the one in front of it. You open a chat window, type a
question, and read the reply. You paste code into a coding assistant and accept its
suggestion. In every case *you* are the user of the model, and *you* judge the output
with your own eyes before doing anything with it.

When you **build** with AI, the model moves inside your software. A model call runs at
**runtime**, as part of your program, triggered by something your users do — clicking a
button, opening a record, submitting a form. The model's answer doesn't come back to
you to read; it comes back to **your code**, which then uses it to do something useful.
Your users get the benefit, and most of them never know a model was involved at all.

That's the whole shift in one sentence: **the model runs inside your program, at
runtime, serving your users, who never see the model directly.** Everything else in
this path is about doing that well.

## What an "AI feature" actually is

An **AI feature** is not exotic. It's an ordinary product capability that happens to be
powered by a model call instead of hand-written logic. The user sees a button, a
result, or a suggestion; behind it, your code calls a model and uses what comes back.

A few concrete shapes it takes, across very different kinds of software:

- **Summarize** — turn a long support thread, article, or meeting transcript into a
  short overview.
- **Extract structured fields** — pull the vendor, date, and total out of a scanned
  invoice into tidy fields your database can store.
- **Answer from your own docs** — let a user ask a question and get an answer drawn
  from *your* product manual, not the open internet.
- **Classify** — sort an incoming message into a category, or flag whether a review is
  positive or negative.
- **Draft or rewrite** — generate a first-draft reply, or rephrase text to be shorter
  or clearer.
- **Translate** — render a message in another language on the fly.

And a **GopherTrunk** example, to keep it close to home: a feature that takes a decoded
control-channel message — normally a wall of terse fields only an expert can read — and
returns a plain-English explanation of what it means. Or one that takes a whole
scanning session and produces a short summary of which talkgroups were active and when.
In each case the user clicks once; a model call does the work; your code shows the
result.

## The anatomy of an AI feature

Every AI feature, however simple, is a small **pipeline**. It's worth seeing the shape
now, because the rest of this path is really just each stage in turn:

1. **Input** — something from your program or your user starts it off: a document, a
   message, a decoded packet, a form field.
2. **Assemble a prompt** — your code builds the text you send the model: an instruction
   plus the relevant input. (Covered in
   [prompts as code](/learn/building-ai/prompts-as-code/) and
   [context and user input](/learn/building-ai/context-and-user-input/).)
3. **Call the model** — you send that prompt to the model's API and wait for a reply.
   (Covered in [how a model API works](/learn/building-ai/how-a-model-api-works/) and
   [your first API call](/learn/building-ai/your-first-api-call/).)
4. **Parse and validate the output** — the reply comes back as text, and your code has
   to turn it into something usable and check it's sane. (Covered in
   [parsing and validating output](/learn/building-ai/parsing-and-validating-output/)
   and [structured output](/learn/building-ai/structured-output/).)
5. **Use it** — display it, store it, or act on it, like any other value in your
   program.
6. **Handle failure** — plan for the call being slow, costing money, or coming back
   wrong. (Covered in
   [handling hallucinations and failure](/learn/building-ai/handling-hallucinations-and-failure/).)

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="A left-to-right pipeline: input, then assemble a prompt, then call the model, then parse and validate, then use the result. A small loop underneath marks handling failure." xmlns="http://www.w3.org/2000/svg">
  <g text-anchor="middle" fill="currentColor">
    <rect x="6" y="20" width="78" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="45" y="41" font-size="9">input</text>
    <rect x="98" y="20" width="78" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="137" y="38" font-size="8.5">assemble</text><text x="137" y="49" font-size="8.5">prompt</text>
    <rect x="190" y="20" width="78" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="229" y="38" font-size="8.5">call the</text><text x="229" y="49" font-size="8.5">model</text>
    <rect x="282" y="20" width="78" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="321" y="38" font-size="8.5">parse &amp;</text><text x="321" y="49" font-size="8.5">validate</text>
    <rect x="374" y="20" width="78" height="34" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="413" y="41" font-size="9">use it</text>
  </g>
  <g stroke="currentColor" stroke-width="1.4" fill="none">
    <path d="M84 37 L96 37" marker-end="url(#pipe_ar)"/>
    <path d="M176 37 L188 37" marker-end="url(#pipe_ar)"/>
    <path d="M268 37 L280 37" marker-end="url(#pipe_ar)"/>
    <path d="M360 37 L372 37" marker-end="url(#pipe_ar)"/>
  </g>
  <text x="229" y="80" text-anchor="middle" font-size="8" fill="currentColor" fill-opacity="0.85">handle failure at every step</text>
  <defs><marker id="pipe_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Every AI feature is the same small pipeline: take an input, assemble a prompt, call the model, parse and validate what comes back, then use it — with failure handling wrapped around the whole thing.</figcaption>
</figure>

Don't worry about the details of each stage yet. The point is that an AI feature is not
one magic step — it's a handful of ordinary ones, most of which are plain code you
write.

## What changes when you build it

The moment the model moves inside your software, a pile of responsibilities becomes
*yours* that were the chatbot's problem before. When you used a chat window, someone
else wrote the prompt scaffolding, handled the wrong answers, and paid the bill. Now you
do.

| When you *use* AI | When you *build* with it |
|-------------------|--------------------------|
| You write the prompt each time | **Your code** assembles the prompt from your users' input |
| You read and judge the answer | **Your code** parses, validates, and uses the answer |
| A wrong answer is yours to shrug off | A wrong answer reaches your users unless you catch it |
| You wait as long as you like | **Latency** is part of your product's responsiveness |
| Someone else pays per message | **You** pay per call, and it adds up at scale |
| You handle the odd glitch by retrying | **You** design for failure — timeouts, bad output, outages |

None of this is a reason not to build. It's simply the job. Owning the prompt, the
inputs, the output handling, the cost, the latency, the failures, and the user
experience *is* what building an AI feature means — and each of those gets its own
lesson later in this path.

## It's software engineering, not machine learning

Here is the reassuring part. Building AI features is **software engineering**, not
machine learning. You are not training a model, tuning weights, or doing statistics. You
are calling an API that someone else trained, the same way you'd call a payments API or
a maps API — send a request, get a response, handle it.

That means you need **no** ML or math background to start. The skills are the ordinary
ones you already use: making an API call, shaping its input, parsing its output,
validating results, handling errors, watching cost. What's unusual is only the
*component* itself: this API returns fluent text and doesn't always give the same answer
twice, so you handle it a little differently than a function that returns a fixed value.
That one difference — treating the model as a **probabilistic component** rather than a
deterministic one — is the mindset the next lesson is built around
([the model as a component](/learn/building-ai/llm-as-a-component/)).

<div class="knowledge-check" data-quiz data-correct-msg="Right — an AI feature runs the model call inside your software, serving your users, who never interact with the model directly." markdown="0">
  <p class="knowledge-check__q">Quick check: What best distinguishes an AI <em>feature</em> from using a chatbot?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It uses a bigger, more capable model than a chatbot does</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The model runs inside your program, serving your users, who never touch it directly</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It requires training your own model rather than calling an existing one</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Using AI** is you at a chat window or coding assistant; **building with it** puts a
  model call inside your own software, at runtime, serving your users.
- An **AI feature** is an ordinary product capability — summarize, extract, answer,
  classify, draft, translate — powered by a model call the user never sees.
- Every AI feature is the same small **pipeline**: input, assemble a prompt, call the
  model, parse and validate, use it, and handle failure.
- When you build it, the prompt, inputs, output handling, cost, latency, failures, and
  UX all become **your** responsibility.
- It's **software engineering, not machine learning** — you integrate an API, no ML or
  math background required.

Next up: [the model as a probabilistic component](/learn/building-ai/llm-as-a-component/) — the mindset shift that makes everything else work.
