---
slug: glossary
title: Glossary of AI-engineering terms
description: Plain-language definitions of the terms used to build AI into software — model API, token, system prompt, structured output, tool use, embedding, RAG, agent, MCP, eval, prompt injection, and more — each cross-linked to the lesson that explains it.
keywords: AI engineering glossary, LLM terms, RAG, embeddings, tool use, function calling, MCP, prompt injection, eval, structured output, agent, context window
level: beginner
status: full
lesson_standalone: true
---

# Glossary of AI-engineering terms

Every term used across the [Building AI Into Your Software](/learn/building-ai/)
path, defined in plain language and linked to the lesson where it's explained in
full. Skim it as a refresher, or use your browser's find (Ctrl/Cmd-F) to jump to a
word. Terms are grouped by theme, roughly in the order the path introduces them.

## Foundations

**AI feature** — A capability in your product powered by a model call at runtime —
summarizing, extracting, answering, classifying — where the user never touches the
model directly. See [What it means to build AI into software](/learn/building-ai/building-vs-using-ai/)

**Model API** — The HTTP interface your code uses to send a request to a language
model and get a response. See [How a model API call works](/learn/building-ai/how-a-model-api-works/)

**Token** — The chunk of text a model reads and writes in; both input and output
are counted in tokens, and tokens drive cost and latency. See
[How a model API call works](/learn/building-ai/how-a-model-api-works/)

**Context window** — The maximum number of tokens a model can consider at once —
your prompt plus its reply must fit inside it. See
[How a model API call works](/learn/building-ai/how-a-model-api-works/)

**Message roles (system / user / assistant)** — The labels on each message in a
request: **system** for standing instructions, **user** for input, **assistant**
for the model's replies. See [How a model API call works](/learn/building-ai/how-a-model-api-works/)

**Stateless** — The API remembers nothing between calls; to continue a conversation
your code resends the prior messages each time. See
[How a model API call works](/learn/building-ai/how-a-model-api-works/)

**Probabilistic / non-deterministic** — The same input can produce different output,
because the model samples its next token; features must be designed for variance.
See [The model as a probabilistic component](/learn/building-ai/llm-as-a-component/)

**Hallucination** — Confident, plausible-sounding output that is actually wrong;
the reason you must ground and verify. See
[Handling hallucinations & failure](/learn/building-ai/handling-hallucinations-and-failure/)

**API key** — The secret credential that authenticates and bills your API calls;
keep it in an environment variable, never in code. See
[Your first API call](/learn/building-ai/your-first-api-call/)

**SDK** — A provider's client library that wraps the raw HTTP API in your language.
See [Your first API call](/learn/building-ai/your-first-api-call/)

**Latency** — How long a call takes; dominated by how many tokens are generated,
which is why long outputs feel slow. See
[Cost, latency & rate limits by design](/learn/building-ai/cost-latency-and-limits/)

**Rate limit (HTTP 429)** — A provider cap on requests or tokens per minute; hitting
it returns a 429, handled with retry-and-backoff. See
[Cost, latency & rate limits by design](/learn/building-ai/cost-latency-and-limits/)

## Prompting

**System prompt** — The standing instructions, persona, rules, and output-format
spec that shape every response, kept separate from per-request data. See
[Prompts as code](/learn/building-ai/prompts-as-code/)

**Prompt template** — A fixed prompt scaffold with slots filled in at runtime, kept
in source and versioned like code. See [Prompts as code](/learn/building-ai/prompts-as-code/)

**Context assembly** — Building the right context for each call — user input, app
data, retrieved documents, prior turns — safely and within budget. See
[Feeding the model context & user input](/learn/building-ai/context-and-user-input/)

**Zero-shot / few-shot** — Prompting with instructions only (zero-shot) versus
including a handful of worked examples (few-shot) to steer the output. See
[Few-shot examples & steering output](/learn/building-ai/few-shot-and-steering/)

**In-context learning** — The model adapting to examples given in the prompt,
without any retraining. See [Few-shot examples & steering output](/learn/building-ai/few-shot-and-steering/)

## Structured output & tools

**Structured output** — Making the model return machine-readable JSON that matches a
schema you define — the bridge from text generator to programmable component. See
[Structured output & JSON](/learn/building-ai/structured-output/)

**JSON Schema** — A description of the fields, types, and required values you want
back; a structured-output call is constrained to match it. See
[Structured output & JSON](/learn/building-ai/structured-output/)

**Tool use / function calling** — Describing functions to the model so it can
request a call with arguments; your code runs it and returns the result. See
[Tool use & function calling](/learn/building-ai/tool-use-and-function-calling/)

**Streaming** — Delivering output token-by-token as it's generated, cutting
*perceived* latency (not cost). See [Streaming responses](/learn/building-ai/streaming/)

**Multimodal** — A model that accepts more than text — images, audio, or documents
— as input. See [Multimodal — images, audio & documents](/learn/building-ai/multimodal-inputs/)

## Retrieval (RAG)

**Grounding** — Putting the facts the model needs into the prompt so it answers from
them rather than from memory. See
[Why retrieval — grounding the model in your data](/learn/building-ai/grounding-and-retrieval/)

**RAG (retrieval-augmented generation)** — Fetching the most relevant pieces of your
data for each query and adding them to the prompt before the model answers. See
[Why retrieval](/learn/building-ai/grounding-and-retrieval/)

**Embedding** — A vector (list of numbers) representing the meaning of a piece of
text, so similar meanings sit close together. See
[Embeddings & vector search](/learn/building-ai/embeddings-and-vector-search/)

**Vector search / vector database** — Searching by meaning by finding the nearest
embedding vectors, stored in an index built for fast nearest-neighbor lookup. See
[Embeddings & vector search](/learn/building-ai/embeddings-and-vector-search/)

**Chunking** — Splitting documents into passages for embedding and retrieval; chunk
size and overlap strongly affect retrieval quality. See
[Chunking & retrieval quality](/learn/building-ai/chunking-and-retrieval-quality/)

**Re-ranking** — Retrieving a larger candidate set, then reordering it with a
stronger model and keeping only the best few. See
[Chunking & retrieval quality](/learn/building-ai/chunking-and-retrieval-quality/)

**Fine-tuning** — Further-training a model on your data to change its behavior,
style, or format; a poor tool for injecting facts (use RAG for those). See
[RAG vs. long context vs. fine-tuning](/learn/building-ai/rag-vs-long-context-vs-finetuning/)

## Agents

**Agent** — A model running in a loop with tools, driving a multi-step task itself
rather than answering in a single call. See
[Agents — letting the model act in a loop](/learn/building-ai/what-is-an-ai-agent/)

**Agent loop (plan–act–observe)** — The cycle an agent repeats: plan, call a tool,
observe the result, decide the next step, until done. See
[Agents — letting the model act in a loop](/learn/building-ai/what-is-an-ai-agent/)

**Tool** — A function you expose to a model or agent — reading data, calling an API,
taking an action — described by a name and a parameter schema. See
[Giving agents tools (and MCP)](/learn/building-ai/giving-agents-tools-and-mcp/)

**MCP (Model Context Protocol)** — An open standard for exposing tools, data, and
resources to models, so one tool server works across compatible apps. See
[Giving agents tools (and MCP)](/learn/building-ai/giving-agents-tools-and-mcp/)

**Least privilege** — Giving a tool or agent only the minimum access it needs, so a
mistake or an attack can do little harm. See
[Giving agents tools (and MCP)](/learn/building-ai/giving-agents-tools-and-mcp/)

## Evaluation & safety

**Eval / eval set** — A curated set of representative inputs plus a way to score
outputs, run on every change to measure quality and catch regressions. See
[Evaluating AI features](/learn/building-ai/evaluating-ai-features/)

**LLM-as-judge** — Using a model to grade outputs against criteria when there's no
single correct answer to compare against. See
[Evaluating AI features](/learn/building-ai/evaluating-ai-features/)

**Regression** — A change (to a prompt, model, or pipeline) that improves some cases
while quietly breaking others; evals catch it. See
[Evaluating AI features](/learn/building-ai/evaluating-ai-features/)

**Guardrail** — A check or limit that keeps a feature safe — validation, filters,
step caps, or required human approval. See
[Handling hallucinations & failure](/learn/building-ai/handling-hallucinations-and-failure/)

**Human-in-the-loop** — Requiring a person to confirm before a high-stakes or
irreversible action is taken. See
[Handling hallucinations & failure](/learn/building-ai/handling-hallucinations-and-failure/)

**Prompt injection** — Untrusted text containing instructions that hijack the
model's behavior, because it can't reliably tell instructions from data. See
[Prompt injection & data safety](/learn/building-ai/prompt-injection-and-security/)

**Indirect prompt injection** — Injection hidden in content the model later reads —
a retrieved document, a web page, a tool result — rather than typed by the user. See
[Prompt injection & data safety](/learn/building-ai/prompt-injection-and-security/)

**Lethal trifecta** — The dangerous combination of access to private data, exposure
to untrusted content, and the ability to exfiltrate or act externally. See
[Prompt injection & data safety](/learn/building-ai/prompt-injection-and-security/)

## Shipping & operations

**Right-sizing** — Choosing the smallest, cheapest model that still passes your
evals, rather than defaulting to the biggest. See
[Choosing a model for a feature](/learn/building-ai/choosing-a-model-for-a-feature/)

**Model routing / cascade** — Sending a request to a cheap model first and escalating
to a stronger one only when needed. See
[Optimizing cost & latency](/learn/building-ai/optimizing-cost-and-latency/)

**Prompt caching** — Reusing a cached, stable prompt prefix so repeated calls are
cheaper and faster. See [Optimizing cost & latency](/learn/building-ai/optimizing-cost-and-latency/)

**PII (personally identifiable information)** — Data that identifies a person;
minimize, redact, and protect it before sending anything to a model API. See
[Privacy, data & compliance](/learn/building-ai/privacy-data-and-compliance/)

**Data retention** — How long a provider keeps the data you send, and whether it may
train on it — read the terms before you ship. See
[Privacy, data & compliance](/learn/building-ai/privacy-data-and-compliance/)

**Observability** — Logging and tracing every model call — inputs, outputs, tokens,
latency, cost, errors — so you can debug and monitor in production. See
[Observability & monitoring](/learn/building-ai/observability-and-monitoring/)

**Quality drift** — A gradual fall in a shipped feature's quality as inputs or models
change, caught by ongoing monitoring and online evals. See
[Observability & monitoring](/learn/building-ai/observability-and-monitoring/)
