---
slug: types-of-models
title: The types of AI models
description: A developer's taxonomy of AI models — text-generation LLMs, reasoning models, embeddings, multimodal and vision models, image generation, speech, and code-specialized models. Learn which write code, which plan, and which act as components inside applications.
keywords: types of AI models, LLM, reasoning model, embedding model, multimodal model, vision model, diffusion model, image generation, speech model, STT, TTS, code model, classifier, local models, RAG
level: intermediate
prereq:
  - context-windows
status: full
faq:
  - q: "Which model types actually write code?"
    a: "Mainly **text-generation LLMs** and **reasoning models** — and reasoning models are best when the task needs planning or hard multi-step logic. **Code-specialized** models are LLMs tuned specifically on code, often used for fast completion. The other types (embeddings, vision, speech, image generation) support coding workflows or act as components rather than writing the code themselves."
  - q: "What is an embedding model and why would I use one?"
    a: "An **embedding model** turns a piece of text into a vector — a list of numbers — so that similar meanings sit close together in vector space. That powers **semantic search** and **retrieval-augmented generation (RAG)**: you find the most relevant documents by comparing vectors, then feed them to an LLM as context. It's a building block, not a chatbot."
  - q: "Is a reasoning model just a smarter LLM?"
    a: "It's an LLM trained and run to spend extra computation \"thinking\" — generating intermediate steps — before its final answer, which helps on planning and hard problems. That extra work usually costs more and takes longer, so you reach for reasoning models on genuinely hard tasks and use ordinary LLMs for routine ones."
---
# The types of AI models

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Not all AI is a chat LLM** — there are reasoning models, embeddings, vision, speech, image generation, and code-specialized models, each with a job. **Know which does what** — LLMs and reasoning models write code, reasoning models plan, and embeddings/classifiers/small models act as components. **Components point ahead** — using these model types *inside* your own application is a whole future skill, distinct from using AI to write code.
</div>

This is lesson 5 of the path. So far we've treated "the model" as a single chat LLM, because that's the tool you'll use most for writing software. But "AI model" covers a family of quite different systems, and knowing the family tree helps you pick the right tool and — importantly — understand what you might one day *build with*. By the end of this lesson you'll have a working taxonomy: which model types write code, which plan, and which act as components you wire into an application.

## A developer's taxonomy

Here's the landscape at a glance. We'll walk through each row below.

| Model type | What it does | Role for a developer |
|------------|--------------|----------------------|
| **Text-generation LLM** | Predicts text/code token by token | Writes and explains code |
| **Reasoning model** | LLM that "thinks" before answering | Planning and hard problems |
| **Embedding model** | Turns text into vectors | Semantic search, RAG (a component) |
| **Multimodal / vision** | Reads images alongside text | Screenshots, diagrams, UIs |
| **Image generation / diffusion** | Creates images from text | Assets, mockups, illustration |
| **Speech (STT / TTS)** | Transcribes and synthesizes speech | Voice input/output (a component) |
| **Code-specialized** | LLM tuned on code | Fast completion, code tasks |

## Text-generation LLMs

This is the workhorse and the model we've described for four lessons: a **text-generation LLM** that predicts the next token to produce text or code. When you ask an assistant to write a function, explain an error, or refactor a file, this is what answers. Everything from [How a model decides](/learn/ai-software-dev/how-models-decide/) applies directly. For most day-to-day coding, a general LLM is what you reach for, and it writes code well because, as we saw, code is highly patterned text.

## Reasoning models

A **reasoning model** is an LLM trained and configured to spend extra computation "thinking" before it commits to a final answer — generating intermediate steps, exploring approaches, sometimes checking its own work. Under the hood it's still next-token prediction; the difference is that it's been tuned to produce and use a chain of intermediate tokens, which measurably helps on tasks that need multi-step logic.

Reasoning models shine on **planning and hard problems**: designing the architecture for a new GopherTrunk decoder, untangling a subtle concurrency bug, or working through a tricky algorithm. The trade-off is that the extra thinking takes longer and usually costs more, so you save them for problems that genuinely warrant it and use ordinary LLMs for routine edits. So among code-capable models: LLMs and reasoning models both *write* code, and reasoning models are the ones you lean on to *plan*.

## Embedding models

An **embedding model** does something that doesn't look like chatting at all. It takes a chunk of text and outputs a **vector** — a fixed-length list of numbers — positioned so that texts with similar *meaning* land near each other in that numeric space. Two descriptions of the same idea sit close; unrelated text sits far apart.

That single trick powers **semantic search** and **retrieval-augmented generation (RAG)**. To find the most relevant docs for a question, you embed the question and compare its vector to the embedded documents, retrieving the closest ones by meaning rather than keyword. Then you hand those documents to an LLM as [context](/learn/ai-software-dev/context-windows/) — which is exactly the "give the model the right material" move we keep returning to, and which we'll build out in [Providing context](/learn/ai-software-dev/providing-context/). Embeddings don't talk to you; they're a **component** other systems are built from.

<figure class="figure" markdown="0">
<svg viewBox="0 0 440 240" role="img" aria-label="A two-dimensional sketch of a vector space in which texts about similar things cluster together: protocol terms in one group, antenna terms in another, memory terms in a third, with an embedded query point retrieving its two nearest neighbours." xmlns="http://www.w3.org/2000/svg">
  <rect x="28" y="34" width="384" height="182" rx="4" fill="currentColor" fill-opacity="0.03" stroke="currentColor" stroke-width="1" stroke-opacity="0.4"/>
  <g stroke="currentColor" stroke-width="0.7" stroke-opacity="0.15">
    <line x1="124" y1="34" x2="124" y2="216"/><line x1="220" y1="34" x2="220" y2="216"/><line x1="316" y1="34" x2="316" y2="216"/>
    <line x1="28" y1="94" x2="412" y2="94"/><line x1="28" y1="155" x2="412" y2="155"/>
  </g>
  <text x="220" y="230" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.85">a 2-D sketch of a high-dimensional vector space</text>
  <g fill="currentColor">
    <text x="110" y="52" text-anchor="middle" font-size="8" font-weight="600">'P25' · 'DMR' · 'trunking'</text>
    <circle cx="100" cy="74" r="4" fill-opacity="0.5"/><circle cx="122" cy="68" r="4" fill-opacity="0.5"/><circle cx="112" cy="94" r="4" fill-opacity="0.5"/><circle cx="132" cy="86" r="4" fill-opacity="0.5"/>
    <text x="330" y="78" text-anchor="middle" font-size="8" font-weight="600">'antenna' · 'coax' · 'SWR'</text>
    <circle cx="316" cy="100" r="4" fill-opacity="0.5"/><circle cx="340" cy="104" r="4" fill-opacity="0.5"/><circle cx="324" cy="124" r="4" fill-opacity="0.5"/><circle cx="346" cy="118" r="4" fill-opacity="0.5"/>
    <text x="190" y="206" text-anchor="middle" font-size="8" font-weight="600">'malloc' · 'GC pause'</text>
    <circle cx="178" cy="176" r="4" fill-opacity="0.5"/><circle cx="202" cy="184" r="4" fill-opacity="0.5"/><circle cx="186" cy="192" r="4" fill-opacity="0.5"/>
  </g>
  <circle cx="128" cy="112" r="38" fill="none" stroke="currentColor" stroke-width="1.1" stroke-dasharray="4 3" stroke-opacity="0.7"/>
  <circle cx="128" cy="112" r="5.5" fill="currentColor" fill-opacity="0.9"/>
  <text x="128" y="130" text-anchor="middle" font-size="7.5" fill="currentColor" font-weight="600">query</text>
  <g stroke="currentColor" stroke-width="1.1" fill="none">
    <line x1="128" y1="112" x2="114" y2="96" marker-end="url(#tom_ar)"/>
    <line x1="128" y1="112" x2="132" y2="90" marker-end="url(#tom_ar)"/>
  </g>
  <text x="150" y="150" text-anchor="start" font-size="7" fill="currentColor" fill-opacity="0.9">retrieve nearest neighbours</text>
  <defs><marker id="tom_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An embedding model places each piece of text at a point in a high-dimensional space (sketched here in 2-D) so that texts about similar things land near each other — protocol terms cluster apart from antenna terms. Embedding your question puts it in the same space; retrieving its nearest neighbours returns the most relevant chunks, which is the engine under semantic search and RAG.</figcaption>
</figure>

## Multimodal and vision models

A **multimodal** model accepts more than text — most usefully for developers, images. A **vision**-capable model can take a screenshot of a broken UI, a photo of a whiteboard architecture sketch, a diagram, or an error dialog, and reason about it alongside your words. You can paste a picture of a misaligned layout and ask what's wrong, or hand it a flowchart and ask for the code. Many leading chat models are now multimodal, so this is often a capability *of* an LLM rather than a wholly separate model.

## Image generation and speech

Two more families round out the picture, less central to writing code but worth knowing.

**Image-generation models**, typically built on **diffusion** (a technique that starts from noise and refines it into an image guided by your text prompt), create pictures from descriptions — handy for mockups, placeholder assets, or illustrations, though not for code itself.

**Speech models** come in two directions: **speech-to-text (STT)** transcribes spoken audio into text, and **text-to-speech (TTS)** turns text into spoken audio. They power voice input to a coding tool and spoken output from one. Like embeddings, these are usually **components** you plug in rather than something you converse with to write software.

## Code-specialized models

Finally, **code-specialized** models are LLMs whose training leaned especially heavily on code, sometimes optimized to be small and fast for a specific job — most commonly the inline autocomplete in your editor, where low latency matters more than deep reasoning. They write code (they're LLMs), but they're tuned for the coding context specifically. We compare code models in [Coding models compared](/learn/ai-software-dev/coding-models-compared/).

## Which writes, which plans, which is a component

Pulling the taxonomy together by *role*:

- **Write code** — text-generation LLMs and reasoning models (and code-specialized LLMs).
- **Plan and tackle hard problems** — reasoning models especially.
- **Act as components** — embedding models, classifiers, speech models, and small/local models, which you wire *into* a larger system rather than chat with.

That last category is worth dwelling on, because it points at something this path deliberately *doesn't* cover yet.

## Looking ahead: building software with embedded AI

Be clear about the scope of this path. **This path is about using AI to write software** — you, the developer, working with a model to produce code. That's one whole skill.

There is a different, equally large skill: **building software that has AI embedded in it** — applications that use these model types as *features inside your own product*. An embedding model powering search in your app. A classifier (a small model that sorts inputs into categories) routing support tickets. A small, local model running on the user's machine for privacy or offline use. A vision model reading documents your users upload. In that world the model types in this lesson stop being your assistant and become *components in your architecture* — things you call from code, exactly like the libraries and layers we met in [What is software?](/learn/intro-software-dev/what-is-software/).

The [Building AI Into Your Software](/learn/building-ai/) path covers that "building with embedded AI" discipline. To make it concrete in GopherTrunk's domain: one could imagine training a small **classifier** on captured radio characteristics to help identify a signal's protocol — P25 versus DMR versus NXDN — or using an **embedding** model to search a corpus of signal descriptions by similarity. Those would be AI *inside* GopherTrunk, a different project from using AI to *write* GopherTrunk. For now, keep the two ideas separate, and recognize the model types here so the later path lands easily.

<div class="knowledge-check" data-quiz data-correct-msg="Right — embedding models turn text into vectors so similar meanings sit close, which powers semantic search and RAG." markdown="0">
  <p class="knowledge-check__q">Quick check: What does an embedding model produce, and what is it mainly used for?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Polished prose answers, used to chat with the user directly</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Vectors that place similar meanings close together, used for semantic search and RAG</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Images from text prompts, used to generate UI mockups</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Text-generation LLMs** — the workhorse that writes and explains code, predicting tokens as we've described throughout this module.
- **Reasoning models** — LLMs that spend extra computation thinking, best for planning and hard problems, at higher cost and latency.
- **Embedding models** — turn text into vectors so similar meanings cluster, powering semantic search and RAG as a component, not a chatbot.
- **Multimodal, image, and speech** — vision models read screenshots and diagrams, diffusion models generate images, and STT/TTS handle speech, often as components.
- **Code-specialized models** — LLMs tuned on code, frequently small and fast for editor autocomplete.
- **Using vs building** — this path uses AI to write software; a future path will cover building software with AI embedded in it, using embeddings, classifiers, and small/local models as features in your own application.

Next up: who actually makes and serves these models, and how the open-versus-closed split shapes your choices. See [The provider landscape](/learn/ai-software-dev/the-provider-landscape/).
