---
slug: interfaces-and-access
title: "Interfaces: app, API, CLI & IDE"
description: The four ways an AI model reaches you — the chat app, the API that underlies everything, command-line agent tools, and IDE plugins — with what each is for, what it's good and bad at, and the basics of accounts versus API keys.
keywords: AI interfaces, chat app, AI API, API key, command-line AI agent, CLI coding tool, IDE plugin, AI code extension, authentication, accounts vs API keys, how to access an LLM
level: beginner
status: full
prereq:
  - the-provider-landscape
faq:
  - q: "What's the difference between using the chat app and the API?"
    a: "The **chat app** is a finished product — a website or desktop/mobile application you log into and type at, with no code required. The **API** is the programmatic interface underneath: you send requests from your own code and get responses back. The app is built *on* the API. Use the app to work directly with a model; use the API to build the model into software of your own."
  - q: "Do I need an API key just to chat with a model?"
    a: "Usually not. The consumer **chat apps** authenticate with an ordinary account — email and password — and bill through a subscription. An **API key** is a secret credential for programmatic access, used by code, CLI agents and IDE plugins. You only need a key when a tool talks to the model on your behalf rather than you typing into a hosted app."
  - q: "Which interface should I start with for coding?"
    a: "Start with whichever gets you a working loop fastest — for most people that's the **chat app** or an **IDE plugin**, because neither needs you to write integration code. This lesson is the map of the options; Module 3 walks through actually using the app, the IDE and a terminal agent, so you can pick deliberately rather than by habit."
---
# Interfaces: app, API, CLI & IDE

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Four ways in** — the chat app, the API, command-line agents, and IDE plugins. **The API underlies them all** — every other interface is software built on the same programmatic access. **Authentication splits two ways** — accounts for hosted apps, API keys for anything that calls the model with your code.
</div>

This is lesson 8 of the path. You've met the providers and how their models differ; now: how does a model actually *reach you*? There are really just four doorways, and knowing them keeps you from confusing the model (the engine) with the interface (the way you drive it). By the end you'll know what each interface is, who it suits, what it's good and bad at, and the basics of authenticating with accounts versus API keys. This lesson is the **map**; the next module, starting with [The provider's app](/learn/ai-software-dev/the-provider-app/), is the **territory** — actually using the app, the IDE and a terminal agent in anger.

## The four doorways

The same underlying model can reach you through any of these. They differ not in the model but in *where you sit* relative to it.

**1. The chat app.** A finished application — a website, desktop app, or phone app — that you log into and type at. This is how most people first meet a model: a text box, a conversation, maybe file uploads and image input. No code, no setup beyond an account. It's the friendliest front door and a fine place to think out loud, ask questions, and paste snippets, but it sits *outside* your project — copy-pasting code back and forth is the friction you'll feel.

**2. The API.** The programmatic interface: your code sends a request to the provider's servers and gets a response back. This is the foundation — **the app, the CLI tools, and the IDE plugins are all software built on top of the API.** You use it directly when you're building AI *into* your own software (a feature, a script, a service), which means you handle the requests, the keys, the error handling and the cost. Powerful and unlimited in shape, but it asks you to be a programmer.

Here's the shape of a single API request, illustratively, with the heavy lifting trimmed away:

```text
POST https://api.<provider>.example/v1/messages
Authorization: Bearer <YOUR_API_KEY>

{
  "model": "<a-model-from-the-family>",
  "messages": [
    { "role": "user", "content": "Explain this Go function: ..." }
  ]
}
```

Every chat app and coding tool is, underneath, sending something like this on your behalf.

**3. Command-line / agent tools in the terminal.** Programs you run in your terminal that wrap the API into an **agent** — software that can read your files, run commands, edit code, run tests, and loop until a task is done, all in your actual project directory. This is where AI stops *suggesting* and starts *doing*. It's the most powerful way to hand off real multi-step work, and the one that most demands supervision — it acts on your files. We dig into these in [Agentic CLI tools](/learn/ai-software-dev/agentic-cli-tools/).

**4. IDE plugins and extensions.** Integrations that live inside your code editor (VS Code, a JetBrains IDE, and others). They put completion, chat, and edits right where you already work, with the editor feeding the model context about the file and project around your cursor. Low friction because there's no context-switch out of your editor, at the cost of being tied to that editor's capabilities. Covered in [IDE integration](/learn/ai-software-dev/ide-integration/).

## What each is good and bad at

| Interface | What it is | Who it's for | Good at | Weak at |
|---|---|---|---|---|
| **Chat app** | Hosted website/desktop/mobile app you log into | Anyone; total beginners | Zero setup, thinking out loud, explanations, quick snippets | Sits outside your project — copy-paste friction; no direct file access |
| **API** | Programmatic access from your own code | Developers building AI into software | Full control, automation, embedding AI in products | Requires coding; you handle keys, errors, cost |
| **CLI / agent** | Terminal tool that reads files, runs commands, edits code | Developers doing real multi-step work in a project | Autonomous tasks across many files; runs tests and tools | Acts on your files — needs supervision; steeper to set up |
| **IDE plugin** | Extension inside your code editor | Developers who live in an editor | Completion and chat with project context, no context switch | Tied to that editor; less autonomous than a full agent |

A useful way to read the table: the four interfaces form a rough spectrum from *least integrated, least autonomous* (the chat app, where you ferry text by hand) to *most integrated, most autonomous* (the terminal agent, which acts directly on your code). The API sits to the side as the foundation the others are built on. None is "best" — they suit different moments, and most developers use two or three. We compare the app, the IDE and the agent head-to-head in [App vs IDE vs both](/learn/ai-software-dev/app-vs-ide-vs-both/).

## Authentication basics: accounts vs API keys

Two kinds of credential, matching two kinds of access.

**Accounts** authenticate the **hosted apps**. You sign in with an email and password (often with two-factor authentication), and the provider bills you through a subscription. This is how the chat app and most consumer products work — no secret to manage beyond your login.

**API keys** authenticate **programmatic access** — anything where software calls the model on your behalf: your own code on the API, a CLI agent, or an IDE plugin you've connected to a provider. A key is a long secret string the provider issues to you; every request carries it, and the provider meters and bills usage against it.

Because a key *is* the credential, treat it like a password:

- **Never commit a key to a repository** or paste it into a public place. Leaked keys get found and abused fast — and you pay for the abuse.
- **Keep keys out of source code.** Put them in environment variables or a secrets manager (see [version control & collaboration](/learn/intro-software-dev/version-control-collab/) for why secrets must stay out of git).
- **Rotate a key** if it might have leaked, and scope or limit it where the provider allows.

You don't always face raw keys: many CLI and IDE tools handle authentication through a guided login flow, and some let you sign in with your provider account instead of pasting a key. Either way, the principle holds — a credential that can spend money or read your code deserves the same care as any password. Tooling and key handling intersect with privacy and safety, which we cover in [Security, privacy & ethics](/learn/ai-software-dev/security-privacy-ethics/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — the chat app, CLI agents and IDE plugins are all software built on top of the API." markdown="0">
  <p class="knowledge-check__q">Quick check: which interface underlies all the others?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The IDE plugin</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">The API — the chat app, CLI agents and IDE plugins are all built on it</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The command-line agent</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Four interfaces** — the chat app, the API, terminal CLI/agent tools, and IDE plugins, each a different place to sit relative to the same model.
- **The API is the foundation** — every other interface is software built on top of programmatic API access.
- **A spectrum of integration** — from the copy-paste chat app to the terminal agent that acts directly on your files, with IDE plugins in between.
- **None is best** — each suits different work, and most developers use several.
- **Accounts vs API keys** — hosted apps use account logins; anything that calls the model with your code uses an API key.
- **Guard your keys** — a key spends money and can read your code, so keep it out of repositories and treat it like a password.

Next up: why those interfaces meter your usage — rate limits, caps and tiers, and what to do when you hit them — in [Usage limits & tiers](/learn/ai-software-dev/usage-limits-and-tiers/).
