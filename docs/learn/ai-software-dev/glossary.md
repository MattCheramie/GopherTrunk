---
slug: glossary
title: Glossary of AI-for-development terms
description: Plain-language definitions for every key term in the Intro to Using AI in Software Development learning path, cross-linked to the lessons that explain them.
keywords: AI glossary, LLM terms, AI coding glossary, token, context window, RLHF, embedding, RAG, MCP, vibe coding, prompt engineering
level: beginner
status: full
lesson_standalone: true
---

# Glossary of AI-for-development terms

This is a plain-language reference for the key terms used across the Intro to Using AI in Software Development path. Definitions are short on purpose; each links to the lesson that explains the idea in full. Terms are grouped by theme and roughly ordered from foundational to advanced within each group, following the shape of the path itself.

## How models work

**Artificial intelligence (AI)** — A broad term for software that performs tasks we associate with human intelligence, such as understanding language or recognising patterns, rather than following fixed rules alone. See [What is AI, for a developer?](/learn/ai-software-dev/what-is-ai-for-coding/)

**Machine learning** — An approach to AI where a system learns patterns from examples instead of being explicitly programmed for every case. See [What is AI, for a developer?](/learn/ai-software-dev/what-is-ai-for-coding/)

**Large language model (LLM)** — A machine-learning model trained on huge amounts of text to predict and generate language, and the kind of model behind most AI coding tools. See [What is AI, for a developer?](/learn/ai-software-dev/what-is-ai-for-coding/)

**Neural network** — A model loosely inspired by the brain, built from layers of simple connected units whose strengths are tuned during training. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**Parameters (weights)** — The numbers inside a neural network that are adjusted during training and that encode what the model has learned; bigger models have more of them. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**Training** — The process of adjusting a model's parameters by showing it data, so its predictions gradually improve. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**Pretraining** — The first, largest training stage, where a model learns general language and knowledge from a broad body of text before any task-specific tuning. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**Fine-tuning** — Further training of an already-pretrained model on narrower data to specialise it for a particular task or style. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**RLHF (reinforcement learning from human feedback)** — A training stage that uses human ratings of model responses to steer the model toward answers people find helpful. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**Alignment** — The effort to make a model's behaviour match human intentions and values, so it is helpful and avoids harmful output. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**Knowledge cutoff** — The date after which a model saw no training data, so it has no built-in knowledge of events or libraries newer than that point. See [How AI models are trained](/learn/ai-software-dev/how-models-are-trained/)

**Hallucination** — When a model produces confident output that is wrong or made up, such as a function or API that does not exist. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

**Token** — The unit a model reads and writes in: a chunk of text, often a word fragment, rather than a whole word or character. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

**Tokenization** — The step that splits text into tokens before the model processes it, and reassembles tokens into text on the way out. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

**Next-token prediction** — The core thing an LLM does: given the text so far, predict the most likely next token, one token at a time. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

**Autoregressive** — Describes a model that generates output one token at a time, feeding each token it produces back in to predict the next. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

**Sampling** — Choosing the next token from the model's predicted probabilities, rather than always taking the single most likely one. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

**Temperature** — A setting that controls how random sampling is: low temperature gives focused, predictable output, higher temperature gives more varied, creative output. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

**Determinism** — Whether the same input reliably gives the same output; because of sampling, model output is often not deterministic unless settings force it to be. See [How a model decides: tokens & prediction](/learn/ai-software-dev/how-models-decide/)

## Context

**Context** — Everything you give a model to work from in a single request: your question, instructions, code, and conversation history. See [The context window, in detail](/learn/ai-software-dev/context-windows/)

**Context window** — The maximum amount of text, measured in tokens, a model can consider at once; anything beyond it must be left out or summarised. See [The context window, in detail](/learn/ai-software-dev/context-windows/)

**Prompt** — The input you send to a model: the instructions and information it responds to. See [The context window, in detail](/learn/ai-software-dev/context-windows/)

**System prompt** — Background instructions, set apart from your message, that tell the model its role and how to behave for the whole conversation. See [The context window, in detail](/learn/ai-software-dev/context-windows/)

**Attention** — The mechanism that lets a model weigh which parts of its context matter most when predicting each token. See [The context window, in detail](/learn/ai-software-dev/context-windows/)

**Lost in the middle** — A tendency for models to use information at the start and end of a long context well but overlook material buried in the middle. See [The context window, in detail](/learn/ai-software-dev/context-windows/)

**Context rot** — The gradual decline in answer quality as a conversation grows long and cluttered, crowding out what actually matters. See [The context window, in detail](/learn/ai-software-dev/context-windows/)

**Prompt caching** — Reusing the processed form of a repeated chunk of context across requests, to cut cost and latency. See [Understanding the cost](/learn/ai-software-dev/understanding-cost/)

## Types of models

**Reasoning model** — A model tuned to work through problems in explicit steps before answering, trading speed for stronger results on hard, multi-step tasks. See [The types of AI models](/learn/ai-software-dev/types-of-models/)

**Embedding** — A numeric representation of text (or other data) that captures its meaning, so similar items end up close together. See [The types of AI models](/learn/ai-software-dev/types-of-models/)

**Vector** — The list of numbers that an embedding produces; comparing vectors is how systems measure how related two pieces of text are. See [The types of AI models](/learn/ai-software-dev/types-of-models/)

**Multimodal model** — A model that handles more than one kind of input or output, such as text together with images or audio. See [The types of AI models](/learn/ai-software-dev/types-of-models/)

**Diffusion model** — A model that generates output (often images) by starting from noise and refining it step by step toward a result. See [The types of AI models](/learn/ai-software-dev/types-of-models/)

**Speech-to-text / text-to-speech** — Models that convert spoken audio into written text, or written text into spoken audio. See [The types of AI models](/learn/ai-software-dev/types-of-models/)

**Code-specialized model** — A model trained or tuned especially for programming tasks, so it handles code and related instructions better than a general model. See [Coding models compared](/learn/ai-software-dev/coding-models-compared/)

**Open-weight model** — A model whose trained parameters are published, so anyone can download and run it themselves. See [The provider & model landscape](/learn/ai-software-dev/the-provider-landscape/)

**Proprietary model** — A model kept private by its maker and offered only as a service, not as downloadable weights. See [The provider & model landscape](/learn/ai-software-dev/the-provider-landscape/)

## Providers, access & cost

**Provider** — A company that builds or hosts AI models and offers access to them, through apps, APIs, or both. See [The provider & model landscape](/learn/ai-software-dev/the-provider-landscape/)

**API (application programming interface)** — A defined way for your own programs to send requests to a model and receive its responses, without a human in the loop. See [Interfaces: app, API, CLI & IDE](/learn/ai-software-dev/interfaces-and-access/)

**API key** — A secret credential that identifies you to a provider's API, authorising requests and tying them to your account and billing. See [Interfaces: app, API, CLI & IDE](/learn/ai-software-dev/interfaces-and-access/)

**Chat app** — A provider's own web or desktop application where you talk to a model through a conversation interface. See [Interfaces: app, API, CLI & IDE](/learn/ai-software-dev/interfaces-and-access/)

**IDE integration** — AI built into your code editor, so it can read your project and suggest or change code where you already work. See [Interfaces: app, API, CLI & IDE](/learn/ai-software-dev/interfaces-and-access/)

**Agent** — An AI system that does more than answer: it can take steps, use tools, and act on your project to pursue a goal. See [Interfaces: app, API, CLI & IDE](/learn/ai-software-dev/interfaces-and-access/)

**Agentic** — Describes a tool or workflow where the AI plans and carries out multi-step tasks on its own, rather than just responding to each message. See [Agentic & command-line tools](/learn/ai-software-dev/agentic-cli-tools/)

**CLI tool** — A command-line program that puts an AI agent in your terminal, where it can run commands and edit files in your project. See [Agentic & command-line tools](/learn/ai-software-dev/agentic-cli-tools/)

**Rate limit** — A cap on how many requests or tokens you may use in a given period, set by the provider to manage load. See [Usage limits & tiers](/learn/ai-software-dev/usage-limits-and-tiers/)

**Usage tier** — A level of access, often raised as your account matures or spending grows, that sets your limits and available features. See [Usage limits & tiers](/learn/ai-software-dev/usage-limits-and-tiers/)

**Subscription vs pay-as-you-go** — Two billing models: a flat recurring fee for app access versus paying per token of API use as you consume it. See [Understanding the cost](/learn/ai-software-dev/understanding-cost/)

**Input vs output tokens** — Billing usually counts tokens you send (input) and tokens the model generates (output) separately, with output often priced higher. See [Understanding the cost](/learn/ai-software-dev/understanding-cost/)

## Using AI to code

**Code completion** — AI that suggests the next part of the code as you type, finishing a line or block in place. See [Code & "sentence" completion](/learn/ai-software-dev/code-completion/)

**Fill-in-the-middle** — A completion style where the model writes code to fit between existing code before and after the cursor, not just at the end. See [Code & "sentence" completion](/learn/ai-software-dev/code-completion/)

**Chat-assisted coding** — Working with an AI by conversing about your code, asking questions and pasting snippets while you stay in control of the edits. See [Chat-assisted coding](/learn/ai-software-dev/chat-assisted-coding/)

**Vibe coding** — Building software by describing what you want in natural language and letting the AI generate and change the code, with little hands-on editing. See [Agentic coding & "vibe coding"](/learn/ai-software-dev/agentic-and-vibe-coding/)

**Spectrum of delegation** — The range of how much you hand to the AI, from typing every line yourself to letting an agent do whole tasks, with everything in between. See [Choosing where on the spectrum to work](/learn/ai-software-dev/the-spectrum/)

**Prompt engineering** — The craft of writing clear, well-structured prompts that reliably get good output from a model. See [Prompting for code](/learn/ai-software-dev/prompting-for-code/)

**Skill** — A reusable, named instruction set that teaches an AI tool how to do a particular task the way you want it done. See [Skills & AI config files](/learn/ai-software-dev/skills-and-config-files/)

**Config file (CLAUDE.md / AGENTS.md)** — A file in your project that gives an AI tool standing instructions and context about how to work in that codebase. See [Skills & AI config files](/learn/ai-software-dev/skills-and-config-files/)

**Rules file** — A configuration file that sets persistent guidelines an AI tool should follow, such as conventions to keep or things to avoid. See [Skills & AI config files](/learn/ai-software-dev/skills-and-config-files/)

**RAG (retrieval-augmented generation)** — A technique that fetches relevant documents and feeds them into the prompt, so the model answers from real, current sources instead of memory alone. See [Feeding the model the right context](/learn/ai-software-dev/providing-context/)

**MCP (Model Context Protocol)** — An open standard that lets AI tools connect to external data sources and tools through a common interface. See [Feeding the model the right context](/learn/ai-software-dev/providing-context/)

**Router** — A layer that picks which model handles a given request, sending each task to the model best suited to it. See [One model, or a combination?](/learn/ai-software-dev/one-model-vs-many/)

**Verification** — Checking that AI-written code is correct, safe, and does what you intended, rather than trusting it on sight. See [Verifying AI-written code](/learn/ai-software-dev/verification-and-trust/)

**Prompt injection** — An attack where hidden instructions in data the model reads try to hijack its behaviour against your intent. See [Security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/)
