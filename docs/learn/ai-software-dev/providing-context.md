---
slug: providing-context
title: Feeding the model the right context
description: The model only knows what it can see. Learn the ways to give it the right context — pasting, @-mentioning files, letting an agent read the repo, retrieval and RAG, and tools via MCP — plus context management as a skill in a real codebase.
keywords: providing context, context management, RAG, retrieval augmented generation, embeddings, MCP, model context protocol, @-mention files, agent reading repo, relevant context, GopherTrunk
level: intermediate
prereq:
  - skills-and-config-files
  - context-windows
faq:
  - q: "What is RAG and when do I need it?"
    a: "**RAG** (retrieval-augmented generation) means fetching the most relevant chunks of a large body of text — code, docs, tickets — and pasting them into the prompt automatically before the model answers. It uses embeddings to find passages similar to your question. You need it when the material is far too big to fit in the context window, like a huge codebase or a wiki, and you want the model grounded in *your* content rather than guessing."
  - q: "What is MCP?"
    a: "**MCP**, the Model Context Protocol, is an open standard for connecting models to external sources — files, documentation, APIs, databases, issue trackers — through a common interface. Instead of every tool inventing its own way to plug in a data source, an MCP server exposes that source once and any MCP-aware client can use it. It's how an agent can, say, read your database schema or fetch a live ticket without custom glue for each tool."
  - q: "Is more context always better?"
    a: "No — relevant beats big. Stuffing the window with marginally related files dilutes the signal, can confuse the model, and costs more. Point it at the few files that matter, prune what doesn't, and start a fresh session when a thread gets muddy. Good context management is choosing what to leave out as much as what to put in."
---
# Feeding the model the right context

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**The model only knows what it can see** — getting the right material into its context is most of the battle. **Many ways in** — pasting, @-mentioning files, letting an agent read the repo, retrieval/RAG over large corpora, and tools via MCP. **Relevant beats big** — context management is a skill: prune the irrelevant, summarise long history, and start fresh when a thread gets muddy.
</div>

This is lesson 21 of the path. Back in [context windows](/learn/ai-software-dev/context-windows/) we established the central fact: a model has no hidden knowledge of your project — it answers from what's in its context plus what it absorbed during training. The previous lesson gave it *standing* context through config files. This lesson is about the rest: how you get the *right* code, docs, and data in front of the model for the task at hand, and — just as important — how you keep the junk out. By the end you'll know the main ways to supply context and why managing it well is a skill in its own right.

## The model only knows what it can see

It's worth saying plainly, because it explains nearly every disappointing answer: when a model gives a generic or wrong response about your code, it usually wasn't looking at your code. It was working from the general patterns it learned in training, because the specific file it needed wasn't in the context. The model isn't withholding effort; it literally cannot see what you didn't show it.

So "providing context" isn't a nice-to-have — it's the main thing that turns a generic assistant into one that understands *your* system. Everything below is a different mechanism for the same goal: get the relevant material into the window.

## The ways to supply context

| Method | What it is | Best for |
|---|---|---|
| **Paste / attach** | Drop code, an error, or a file straight into the chat | Quick, focused questions about a snippet you already have |
| **@-mention files** | Reference files by name in an IDE so the tool pulls them in | Working in an editor where the files are right there |
| **Agent reads the repo** | An agentic tool opens, searches, and reads files itself | Tasks spanning many files where you don't know all of them up front |
| **Retrieval / RAG** | Automatically fetch the most relevant chunks of a large corpus | Codebases or doc sets far too big to fit in the window |
| **Tools / MCP** | Connect the model to live sources — files, APIs, databases | Grounding answers in current, external data |

**Pasting and attaching** is the simplest: you hand the model exactly the text you want it to consider. It's precise but manual, and it doesn't scale past a handful of files.

**@-mentioning** files in an [IDE integration](/learn/ai-software-dev/ide-integration/) is the same idea with less friction — you name the file and the tool inserts its contents for you.

**Letting an agent read the repo** is the leap the [agentic tools](/learn/ai-software-dev/agentic-cli-tools/) made. Instead of you choosing every file, the agent searches the codebase, opens what looks relevant, and reads it — discovering context you might not have known to provide. The cost is that it spends context (and time) exploring, so a good prompt still points it roughly where to look.

### Retrieval and RAG for large corpora

When the material is far bigger than any context window — a million-line codebase, years of documentation — you can't paste it and an agent can't read all of it. **Retrieval-augmented generation (RAG)** solves this by fetching only the most relevant pieces.

The trick relies on **embeddings**, which we met in [types of models](/learn/ai-software-dev/types-of-models/): a model turns each chunk of text into a vector — a list of numbers — positioned so that passages about similar things sit near each other. Your question gets embedded the same way, and the system retrieves the chunks whose vectors are closest to it. Those few chunks get pasted into the prompt, and the model answers grounded in *your* content rather than its general training. That's the whole shape of RAG: embed everything once, retrieve the nearest matches per question, generate from them. It's how a tool can answer about a codebase it could never hold in memory all at once.

<figure class="figure" markdown="0">
<svg viewBox="0 0 476 214" role="img" aria-label="A retrieval-augmented-generation pipeline: offline, the corpus is embedded into a vector store; per query, the question is embedded, its nearest chunks are retrieved from the store, injected into the prompt alongside the question, and passed to the model to generate an answer." xmlns="http://www.w3.org/2000/svg">
  <text x="150" y="14" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">index once · offline</text>
  <g text-anchor="middle" fill="currentColor" font-size="8">
    <rect x="18" y="24" width="78" height="40" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="57" y="42" font-weight="600">corpus</text><text x="57" y="54" font-size="7">code · docs</text>
    <rect x="120" y="24" width="56" height="40" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/><text x="148" y="48" font-weight="600">embed</text>
    <rect x="206" y="18" width="92" height="52" rx="5" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.3"/><text x="252" y="40" font-weight="600">vector</text><text x="252" y="52">store</text>
    <rect x="250" y="86" width="120" height="32" rx="5" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.1"/><text x="310" y="106" font-weight="600">nearest chunks</text>
  </g>
  <text x="57" y="142" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">per query</text>
  <g text-anchor="middle" fill="currentColor" font-size="8">
    <rect x="18" y="150" width="78" height="40" rx="5" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="57" y="168" font-weight="600">question</text><text x="57" y="180" font-size="7">'control channel'</text>
    <rect x="120" y="150" width="56" height="40" rx="5" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/><text x="148" y="174" font-weight="600">embed</text>
    <rect x="250" y="150" width="140" height="40" rx="5" fill="currentColor" fill-opacity="0.16" stroke="currentColor" stroke-width="1.3"/><text x="320" y="168" font-weight="600">prompt</text><text x="320" y="180" font-size="7">question + chunks</text>
    <rect x="398" y="150" width="70" height="40" rx="5" fill="currentColor" fill-opacity="0.2" stroke="currentColor" stroke-width="1.3"/><text x="433" y="168" font-weight="600">model</text><text x="433" y="180" font-size="7">answer</text>
  </g>
  <g stroke="currentColor" stroke-width="1.3" fill="none">
    <line x1="96" y1="44" x2="120" y2="44" marker-end="url(#rag_ar)"/>
    <line x1="176" y1="44" x2="206" y2="44" marker-end="url(#rag_ar)"/>
    <line x1="252" y1="70" x2="300" y2="86" marker-end="url(#rag_ar)"/>
    <line x1="310" y1="118" x2="320" y2="150" marker-end="url(#rag_ar)"/>
    <line x1="96" y1="170" x2="120" y2="170" marker-end="url(#rag_ar)"/>
    <line x1="176" y1="170" x2="240" y2="74" marker-end="url(#rag_ar)"/>
    <line x1="390" y1="170" x2="398" y2="170" marker-end="url(#rag_ar)"/>
  </g>
  <text x="196" y="118" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.9">retrieve</text>
  <text x="330" y="136" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.9">inject</text>
  <defs><marker id="rag_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>RAG has two phases. Once, <strong>offline</strong>, every chunk of the corpus is embedded into a vector store. Then <strong>per question</strong> the query is embedded too, the nearest chunks are retrieved by vector similarity, and those few chunks are injected into the prompt so the model answers grounded in your content — never holding the whole corpus in context at once.</figcaption>
</figure>

### Tools and MCP for live data

Some context isn't a file at all — it's the current state of a database, a live API response, or today's open issues. **Tools** let a model reach out and fetch such things during a conversation. The **Model Context Protocol (MCP)** is an open standard for these connections: an MCP *server* exposes a source — files, docs, an API, a database, an issue tracker — through a common interface, and any MCP-aware client can use it. The point of a standard is leverage: expose your data source once, and every MCP-capable tool can read it, instead of writing custom glue for each tool. This is increasingly how an agent grounds itself in current, external reality rather than a static snapshot.

## Context management is a skill

Having many ways *in* creates a new problem: it's easy to over-fill the window. And more context is not automatically better. **Relevant beats big.** A window padded with marginally related files dilutes the signal the model needs, can actively mislead it toward the wrong file, and costs more on every turn. Managing context well is mostly about restraint:

- **Prune the irrelevant.** Remove files and attachments that aren't pulling their weight. If a file isn't load-bearing for the task, it's noise.
- **Summarise long history.** A sprawling conversation eventually crowds out the actual code. Condense the decisions so far into a short summary and drop the back-and-forth that produced them.
- **Start fresh when the thread gets muddy.** Once a session is full of abandoned directions and contradictions, the cleanest fix is a new session seeded with just the relevant state. A clean window is easier to steer than a cluttered one — the same advice as [refining versus restarting a prompt](/learn/ai-software-dev/prompting-for-code/).

The mental model: you are the editor of the model's attention. Every token of context competes with every other for the model's focus, so curating *down* to what matters is as much the job as gathering material in the first place.

## In a real codebase: point, don't dump

GopherTrunk makes the practical lesson concrete. Suppose you're fixing a bug in the control-channel decoder. The instinct might be to give the model "the whole repo" and let it sort things out. Resist it. The repo spans DSP, multiple protocol decoders, a daemon, and replay tooling — most of which is irrelevant to your bug and would only dilute the model's attention.

Instead, point it at the right neighbourhood: the `internal/scanner/ccdecoder` package, plus the one or two types your change touches and the test file you'll extend. That focused context lets the model match the package's existing error style and conventions — exactly what the [config file](/learn/ai-software-dev/skills-and-config-files/) and a precise prompt set up — without wading through unrelated DSP code. If it turns out the bug reaches into the down-converter, you add `ddc.go` then; you don't front-load the entire tree on the off chance. Start narrow, widen only when the task proves it needs more.

This also keeps you out of a known trap: GopherTrunk has two separate down-conversion paths (a single-channel `Downconverter` used by replay, and a wideband `DDCBank` used live). A model handed the whole repo can easily edit the wrong one. Pointing it at the specific path the task concerns prevents that class of mistake — context discipline is correctness, not just efficiency.

<div class="knowledge-check" data-quiz data-correct-msg="Right — RAG uses embeddings to fetch the most relevant chunks of a large corpus so the model can answer from your content without holding all of it in context." markdown="0">
  <p class="knowledge-check__q">Quick check: how does retrieval-augmented generation (RAG) handle a codebase far too large for the context window?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It permanently retrains the model on the whole codebase</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">It uses embeddings to fetch only the most relevant chunks and pastes those into the prompt</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It compresses the entire codebase so it fits in the window at once</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **The model only knows what it can see** — generic answers usually mean the relevant code wasn't in the context, not that the model didn't try.
- **Many ways in** — paste/attach, @-mention files in an IDE, let an agent read the repo, retrieve with RAG, or connect live sources with tools.
- **RAG uses embeddings** — it fetches the chunks most similar to your question so the model can ground its answer in a corpus too big to fit.
- **MCP standardises connections** — an open protocol so one data source, exposed once, works with any MCP-aware tool.
- **Relevant beats big** — prune irrelevant files, summarise long history, and start fresh when a thread gets muddy.
- **Point, don't dump** — aim the model at the specific package and types a task needs, and widen only when it proves it must.

Next up: whether to commit to a single model and provider or combine several — the trade-offs, and how multi-model setups actually work — in [One model, or a combination?](/learn/ai-software-dev/one-model-vs-many/).
