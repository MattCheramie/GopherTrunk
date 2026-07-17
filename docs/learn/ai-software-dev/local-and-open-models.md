---
slug: local-and-open-models
title: Running local & open models
description: Some capable open-weight models can be downloaded and run on your own hardware — private, offline, and free of per-token bills — but you supply the compute, and there are real quality and hardware trade-offs to weigh.
keywords: local LLM, open-weight models, Ollama, llama.cpp, LM Studio, quantization, VRAM, GPU requirements, offline AI, self-hosted model, open source models
level: intermediate
status: full
prereq:
  - the-provider-landscape
  - agentic-cli-tools
faq:
  - q: "Can I run a coding model on my own computer?"
    a: "Yes. Tools like Ollama and LM Studio download an **open-weight** model and run it entirely on your machine, exposing a local API that many editors and CLIs can point at. Whether it runs *well* depends on the model's size and your hardware — a small model is comfortable on a laptop, a large one needs a serious GPU or it runs slowly on CPU and system RAM."
  - q: "Are local models as good as cloud models for coding?"
    a: "For everyday completion, boilerplate and offline Q&A, a good local model is genuinely useful. But at the top end there is still a gap: the hardest agentic coding and large-context work generally favour the frontier hosted models today. That gap has been closing steadily and open-weight releases keep gaining, so treat any specific comparison as a snapshot — but don't expect a laptop model to match the best cloud model on a gnarly multi-file task just yet."
  - q: "What hardware do I need?"
    a: "The main limit is **memory** — specifically **VRAM** on your GPU. A model's weights have to fit in memory to run at usable speed, and bigger models need more of it. **Quantization** shrinks a model to fit in less memory at some cost to quality, which is what lets modest hardware run useful models at all. Roughly: small models fit in a laptop's RAM or a modest GPU; large ones want a high-VRAM card, or they fall back to slow CPU inference."
  - q: "Why run a model locally at all?"
    a: "Four common reasons: **privacy** (confidential code never leaves your machine), **offline or air-gapped** operation, **cost** (no per-token bills or rate limits once you own the hardware), and **control** (the model can't be changed or retired out from under you). It's also a great way to learn how these systems actually work."
---
# Running local & open models

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Open-weight** models publish their trained parameters, so you can download and run them **locally** — on your own machine, with nothing leaving it and no per-token bill. The catch is that **you** supply the compute: a model's size is measured in billions of parameters, and it has to fit in memory — mainly your GPU's **VRAM** — to run at usable speed. **Quantization** shrinks a model to fit at some quality cost. Small models run on a laptop and are fine for everyday help; the hardest agentic coding still favours frontier hosted models. See [the provider landscape](/learn/ai-software-dev/the-provider-landscape/) for where these sit in the market.
</div>

The [previous lesson](/learn/ai-software-dev/agentic-cli-tools/) reached models running on someone else's servers. This one flips that: some of the most capable models can be downloaded and run on hardware you own. That buys privacy, offline use and freedom from usage bills — but it moves the compute bill onto your desk, and there are honest trade-offs to weigh. This lesson covers what "local" means, the tools that make it easy, the hardware reality, and when running local is actually the right call.

## Open-weight vs. proprietary

A quick recap from [the provider landscape](/learn/ai-software-dev/the-provider-landscape/). A **proprietary** model lives only behind an API — the weights (the trained numbers that *are* the model) stay on the provider's servers and you never possess them. An **open-weight** model publishes those weights for download, so anyone can run it on their own hardware or a rented server.

"Open-weight" is not the same as "open-source" in the full sense. You get the weights, but not necessarily the training data, the training code, or an unrestricted licence — some releases carry usage limits. Read the licence before you build on one.

Rather than rank this week's releases, it's safer to name families generically. You'll see open-weight lineages like Llama, Qwen, Mistral, DeepSeek and gpt-oss-style releases, offered in several sizes and refreshed often. Which one leads on a given task shifts constantly, so treat any specific claim as a snapshot.

## What "local" means

"Local" means the model runs **on your machine**. You download the weights once, load them into memory, and every request is answered by your own hardware. Nothing is sent to a provider; no network call leaves your computer. The flip side of that independence is that **you supply the compute** — the speed and the ceiling on model size are set by your hardware, not by a data centre.

## The tools

You don't have to wire this up by hand. A few tools have made local models genuinely easy to run:

- **Ollama** — the easiest on-ramp. One command pulls a model and runs it; it manages the download, the format and the memory for you.
- **llama.cpp** — the efficient inference engine that sits *under* many of these tools. It's lower-level, but it's what makes running models on ordinary hardware (including CPUs) practical.
- **LM Studio** — a graphical app for browsing, downloading and chatting with models, for people who'd rather not touch a terminal.

The common thread: each one pulls a model and exposes a **local API** on your machine — typically one that mimics a familiar API shape — so many editors, [agentic CLIs](/learn/ai-software-dev/agentic-cli-tools/) and chat front-ends can be pointed straight at it instead of at a cloud provider.

## The hardware reality

This is where local models get honest with you. A model's size is measured in **parameters** — billions of them — and to run at usable speed, those parameters have to fit in fast memory. The single biggest limit is **VRAM**, the memory on your GPU. Fit the model in VRAM and it's quick; spill over and it falls back to system RAM and the CPU, where it still *runs* but often crawls.

**Quantization** is the lever that makes modest hardware viable. It stores each parameter at lower precision — think of it as compressing the model — so a model that wouldn't otherwise fit now does, in exchange for a modest, usually acceptable quality loss. Heavier quantization means a smaller footprint and a bigger quality hit.

Put together, the rule of thumb is simple:

| Model size | Roughly what it needs |
|---|---|
| **Small** (a few billion params) | A laptop's RAM or a modest GPU — runs comfortably |
| **Medium** | A single strong consumer GPU with plenty of VRAM |
| **Large** (frontier-class open weights) | A high-VRAM card (or several), or slow CPU/RAM fallback |

There's no way around the physics: bigger, more capable models need more memory. Quantization buys you headroom; it doesn't repeal the trade-off.

## The quality gap for coding

Be honest about where local models stand for coding, and durable about *why*. Small local models are genuinely good at **code completion**, **boilerplate**, explaining a snippet, and answering questions **offline** — the everyday help that doesn't need the absolute strongest model. For a lot of routine work, a local model is plenty.

At the top end, though, a gap remains today. The hardest **agentic coding** — long multi-step tasks, large-context reasoning across a whole repository, subtle debugging — still generally favours the frontier hosted models. That's not a knock on open weights; it reflects the enormous compute that goes into training and serving the biggest closed models. The gap has narrowed release by release and keeps narrowing, so this is a snapshot, not a law. Match the model to the task and you'll rarely be disappointed.

## Why run local

Even with that gap, running local earns its place for real reasons:

- **Privacy.** Confidential or proprietary code never leaves your machine — nothing to send, nothing to leak.
- **Offline / air-gapped.** No internet, no problem. The model runs on a plane, in the field, or on a machine that's deliberately disconnected.
- **No usage bills or rate limits.** Once you own the hardware, inference is "free" — no per-token meter, no throttling on a busy afternoon.
- **Control and reproducibility.** The exact model stays put; no provider can change it, price it up, or retire it out from under you.
- **Learning and experimentation.** Running a model yourself is one of the best ways to understand how these systems actually behave.

## When it makes sense (and when it doesn't)

A short decision guide. Choose by the task in front of you rather than by principle.

**Reach for local when:**

- The code is sensitive and can't leave your network (see [security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/)).
- You need to work offline or on an air-gapped machine.
- The work is routine — completion, boilerplate, quick questions — and doesn't need the strongest model.
- You want predictable cost with no per-token bill (see [understanding cost](/learn/ai-software-dev/understanding-cost/)).

**Reach for a hosted frontier model when:**

- The task is a hard, novel, multi-file agentic job where the top of the capability range earns its keep.
- You don't have the hardware to run a large model at usable speed.

Many developers don't choose once — they run a **hybrid** setup: a local model for private or simple work, a cloud model for the hard problems, switching by task. That's exactly the mix-and-match habit covered in [one model vs. many](/learn/ai-software-dev/one-model-vs-many/), and choosing well per task is the theme of [coding models compared](/learn/ai-software-dev/coding-models-compared/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the model's weights must fit in fast memory (ideally GPU VRAM) to run at usable speed; that's the main constraint." markdown="0">
  <p class="knowledge-check__q">Quick check: the single biggest limit on which local model you can run at usable speed is usually…?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">Which operating system you use</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">How much (V)RAM you have</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">Your internet connection speed</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Open-weight** models publish their weights so you can download and run them yourself; that's not the same as an open-source licence, so check the terms.
- **Local** means the model runs on your machine — nothing leaves it, and you supply the compute.
- **Ollama, llama.cpp and LM Studio** pull a model and expose a local API your editors and CLIs can point at.
- The main hardware limit is **memory, mostly GPU VRAM**; **quantization** shrinks a model to fit at some quality cost.
- Small local models handle completion, boilerplate and offline Q&A well; the hardest agentic coding still favours frontier hosted models, though the gap keeps closing.
- Run local for privacy, offline use, cost and control — and many people run a **hybrid** of local and cloud, choosing by task.

Next up: with every option on the table, how do you actually choose — [app, IDE, agent — or all three?](/learn/ai-software-dev/app-vs-ide-vs-both/).
