---
slug: chat-assisted-coding
title: Chat-assisted coding
description: Chat-assisted coding uses conversation as the interface — you ask for a function, explanation, refactor, or test and review each result before using it. Learn why human-in-the-loop review is the safety mechanism.
keywords: chat-assisted coding, AI chat coding, human in the loop, prompt for code, refactor with AI, explain code, table-driven test, code review, conversational coding, AI pair programming
level: beginner
prereq:
  - code-completion
faq:
  - q: "What is chat-assisted coding?"
    a: "It's using a back-and-forth **conversation** to get coding help: you describe what you want — a function, an explanation, a refactor, a test, a fix — and the model replies with code or prose that you then review before using. A human reviews every result, which is what keeps it safe."
  - q: "When should I use chat instead of inline completion?"
    a: "Use [completion](/learn/ai-software-dev/code-completion/) for the obvious next line you'd type anyway. Use **chat** when you need to *describe* what you want, ask *why* something behaves a certain way, request a larger or multi-step change, or get an explanation. Chat handles anything that doesn't fit in a single grey suggestion."
  - q: "Should I keep one long chat thread or start fresh?"
    a: "Iterate in the **same thread** while you're refining one piece of work — the model keeps the context. Start a **fresh thread** when you switch tasks, because stale context from the old task can confuse the model and waste your [context window](/learn/ai-software-dev/context-windows/). A good rule: one thread per problem."
---
# Chat-assisted coding

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Conversation as interface** — you ask in words for a function, explanation, refactor, or test, and the model answers. **Human-in-the-loop** — you review every result before using it, and that review is the safety mechanism, not a formality. **Prompts and context** — clear requests plus the right surrounding code are what make answers useful.
</div>

This is lesson 16 of the path, and the second mode of AI-assisted coding. In the [previous lesson](/learn/ai-software-dev/code-completion/) the model finished your lines as you typed. Here it does much more, but only when you ask, and only with your review in between. By the end you'll understand why chat is the everyday workhorse of AI-assisted coding, why the review step is what keeps it safe, and how good prompts and good context decide whether the answers are worth anything.

## Conversation as the interface

In chat-assisted coding, the interface is a dialogue. You type a request in plain language — possibly with some code pasted in — and the model replies with code, an explanation, or both. You read the reply, decide what to do with it, and either use it, adjust it, or ask again. The unit of work is no longer a single line (as in completion) but a *request and its answer*.

That makes chat enormously flexible. The same interface handles a sweep of different jobs:

| You ask for… | Example request |
|--------------|-----------------|
| **An explanation** | "Explain what this function does, step by step." |
| **New code** | "Write a function that parses a P25 network status broadcast." |
| **A refactor** | "Refactor this to remove the duplicated error handling." |
| **Tests** | "Write a table-driven test for this decoder function." |
| **A fix** | "This panics on an empty slice — why, and how do I fix it?" |
| **A review** | "Are there edge cases this loop doesn't handle?" |

Notice these aren't different tools; they're different *questions* to the same conversational partner. That breadth is why chat is the **everyday workhorse** mode — the one most people reach for most of the time. It sits between the token-by-token assistance of [completion](/learn/ai-software-dev/code-completion/) and the hands-off delegation of [agentic coding](/learn/ai-software-dev/agentic-and-vibe-coding/), and for a huge range of tasks it's exactly the right altitude.

## The human stays in the loop

The defining feature of chat-assisted coding is that **a human reviews every result before it's used**. The model proposes; you dispose. Nothing the model writes touches your project until you decide to put it there.

This "human-in-the-loop" review is not a courtesy or a slowdown — it is *the safety mechanism*. Recall from earlier in this path ([How a model decides what to write](/learn/ai-software-dev/how-models-decide/)) that a language model generates fluent, statistically-likely text. It does not run your code, does not know your project's invariants unless you tell it, and can produce confident answers that are subtly or completely wrong — the phenomenon we call **hallucination**, where the model invents a plausible-looking API, function, or fact that doesn't exist. Fluent and correct are different properties, and only you can check the second one.

So the review is where correctness actually enters the process. When you read a chat reply, you're doing the job the model can't: checking it against what you know about the problem, the codebase, and reality. Does this function exist? Does this handle the empty case? Is this the polynomial *this* protocol uses? That judgment is the value you add, and it's why chat is far safer than modes where the human reviews less. Pairing your review with the project's [tests](/learn/intro-software-dev/testing/) and [version control](/learn/intro-software-dev/version-control-collab/) gives you a second and third net beneath your eyes.

The practical upshot: never paste a chat answer in and move on without reading it. The moment you stop reviewing is the moment chat stops being chat-assisted coding and starts being something riskier.

## Good prompts and good context

Chat's answers are only as good as two things you control: the **prompt** (what you ask) and the **context** (what the model can see). Both get their own lessons later in this path — [Prompting for code](/learn/ai-software-dev/prompting-for-code/) and [Providing context](/learn/ai-software-dev/providing-context/) — but the short version matters here because it determines whether chat is useful at all.

A weak prompt produces a weak answer. "Make this better" gives the model nothing to aim at. A strong prompt states the goal, the constraints, and the form you want the answer in: *"Refactor this function to return an error instead of panicking, keep the existing signature for the happy path, and don't change the public API."* The more precisely you describe the target, the less the model has to guess.

Context is the other half. A model can only reason about what's in its [context window](/learn/ai-software-dev/context-windows/) — the working memory of the conversation. If you ask it to refactor a function but don't show the function, or the types it depends on, it will fill the gaps with assumptions, and those assumptions are where bugs come from. Pasting the relevant code, the error message, the type definitions, and a sentence about what the code is *for* turns a guessing game into a grounded one.

## A worked GopherTrunk example

Suppose you've written a small decoder helper in GopherTrunk and you want a solid test for it. The function:

```go
// parseTSBK decodes a P25 trunking signaling block and returns the opcode
// and its payload. It returns an error if the block is too short.
func parseTSBK(block []byte) (opcode byte, payload []byte, err error) {
    if len(block) < 12 {
        return 0, nil, fmt.Errorf("tsbk too short: %d bytes", len(block))
    }
    return block[0] & 0x3F, block[1:12], nil
}
```

A good chat prompt gives the goal, the context, and the form:

> Write a table-driven Go test for this function. Cover: a valid 12-byte block, a too-short block (should error), and a block longer than 12 bytes (extra bytes ignored). Use `t.Run` subtests and check both the returned values and the error.

A solid reply would scaffold something like this, which you then read and verify against what `parseTSBK` actually promises:

```go
func TestParseTSBK(t *testing.T) {
    tests := []struct {
        name       string
        block      []byte
        wantOpcode byte
        wantLen    int
        wantErr    bool
    }{
        {"valid", make([]byte, 12), 0x00, 11, false},
        {"too short", make([]byte, 5), 0x00, 0, true},
        {"extra bytes ignored", make([]byte, 20), 0x00, 11, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            op, payload, err := parseTSBK(tt.block)
            if (err != nil) != tt.wantErr {
                t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
            }
            if err == nil {
                if op != tt.wantOpcode {
                    t.Errorf("opcode = %#x, want %#x", op, tt.wantOpcode)
                }
                if len(payload) != tt.wantLen {
                    t.Errorf("payload len = %d, want %d", len(payload), tt.wantLen)
                }
            }
        })
    }
}
```

The model did the tedious scaffolding; your job is the review. Is `0x3F` masking handled in a case with a non-zero opcode byte? (Here, no — every input byte is zero, so you might ask for a case with `block[0] = 0xC1` to prove the mask works.) That follow-up is the loop in action: you spotted a gap, so you ask again in the *same thread*, where the model still has the function and the existing test in view.

## Iterating versus starting fresh

Chat is iterative by nature, and knowing when to keep going versus when to reset is a real skill.

**Stay in the same thread** while you're working on one piece. The model retains the conversation, so each follow-up — "now add the missing mask case," "now make the error message match our style" — builds on what came before without you re-explaining.

**Start a fresh thread** when you switch to an unrelated task. Old context doesn't just sit there harmlessly; it competes for room in the [context window](/learn/ai-software-dev/context-windows/) and can bias the model toward the previous problem. A thread that started about CRC code and drifted into UI layout is one that'll give muddier answers to both. The simple discipline — *one thread per problem* — keeps the context relevant and the answers sharp.

<div class="knowledge-check" data-quiz data-correct-msg="Right — in chat-assisted coding, the human reviews every result, and that review is what keeps it safe." markdown="0">
  <p class="knowledge-check__q">Quick check: In chat-assisted coding, what is the main safety mechanism?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The model runs and verifies its own code before answering</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A human reviews every result before it's used</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The chat tool blocks any answer that contains a bug</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Conversation as interface** — you ask in plain language for explanations, code, refactors, tests, or fixes, and review the reply.
- **Everyday workhorse** — chat sits between completion and agentic coding and handles the widest range of day-to-day tasks.
- **Human-in-the-loop** — reviewing every result before using it is the safety mechanism, because models produce fluent text that can still be wrong.
- **Prompts matter** — a precise request that states the goal, constraints, and desired form gets a far better answer than a vague one.
- **Context matters** — the model can only reason about what's in its context window, so showing the relevant code and errors grounds the answer.
- **One thread per problem** — iterate in the same thread while refining one task; start fresh when you switch, to keep context relevant.

Next up: what happens when you hand the model the goal instead of the question, and let it plan, edit, and run code on its own — see [Agentic coding & "vibe coding"](/learn/ai-software-dev/agentic-and-vibe-coding/).
