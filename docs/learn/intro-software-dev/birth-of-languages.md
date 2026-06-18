---
slug: birth-of-languages
title: From machine code to high-level languages
description: How programming climbed from raw machine code through assembly to high-level languages like FORTRAN, COBOL, LISP, and C — and what a compiler actually does.
keywords: machine code, assembly language, high-level languages, FORTRAN, COBOL, LISP, ALGOL, C programming language, compiler, abstraction in programming
level: beginner
status: full
faq:
  - q: "What is the difference between machine code and a high-level language?"
    a: "**Machine code** is the raw numbers a processor executes directly — fast for the chip, nearly unreadable for humans. A **high-level language** lets you write something close to English or math, like `total = price * quantity`, which a tool then translates down to machine code. High-level languages trade a little control for an enormous gain in readability and productivity."
  - q: "What does a compiler do?"
    a: "A **compiler** is a program that translates source code written in a high-level language into machine code the processor can run. It reads your whole program, checks it, and produces an executable. This is the key idea that makes high-level languages practical — you write for humans, and the compiler handles the translation to the machine."
  - q: "Why is C still used for radio and DSP code?"
    a: "**C** (1972) gives you direct, predictable control over memory and hardware while still being far more readable than assembly. Digital signal processing needs to move large streams of samples quickly with tight, well-understood performance, and C delivers that. Decades of mature radio and DSP libraries are written in C, so it remains the backbone of low-level signal code."
---
# From machine code to high-level languages

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Abstraction climbs** — programming rose from raw machine code to assembly to human-friendly high-level languages. **Compilers translate** — they turn readable source code into machine code so you write for humans, not the chip. **Languages have purposes** — FORTRAN for numbers, COBOL for business, LISP for AI, C for systems.
</div>

A computer only understands one thing: numbers that mean "do this operation." Yet nobody writes serious software in those numbers anymore. This lesson is the story of how programming climbed away from the raw machine — a ladder of **abstraction**, where each rung let people say more while typing less. By the end you'll know what machine code, assembly, and high-level languages are, what a compiler does, and why a handful of languages from the 1950s and 70s still shape the code behind today's radios. This builds directly on [the hardware story](/learn/intro-software-dev/computing-hardware-story/) — now we make that hardware usable.

## The bottom rung: machine code

At the very bottom is **machine code** — the numeric instructions the processor reads and executes directly. Every chip family has its own set of these instructions, encoded as raw numbers. Written out, a program in machine code is just a wall of values like:

```text
10110000 01100001
```

That happens to mean "put the number 97 into a register" on one kind of processor. It is fast for the chip and almost impossible for a human to read, write, or debug. Early programmers really did work at this level, toggling switches or punching cards — slow, error-prone, and unforgiving.

## Assembly: names instead of numbers

The first step up was **assembly language**. Instead of writing the number for "add," you write a short mnemonic like `ADD`; instead of a raw memory address, you write a name. A small program called an **assembler** translates those mnemonics back into the exact machine-code numbers.

```text
MOV  AL, 97   ; put 97 into register AL
ADD  AL, 1    ; add 1, AL now holds 98
```

Assembly is a thin, human-readable skin over machine code — one assembly line usually maps to one machine instruction. It made programming far more bearable, but it's still tied to one specific processor, and you still think in terms of registers and addresses rather than the actual problem you're solving.

## Why abstraction was needed

Assembly solved readability but not the deeper problem: you were still thinking like the machine, not like the problem. Writing a payroll calculation or a physics simulation meant translating your idea into thousands of tiny register operations by hand, and a program written for one processor wouldn't run on another.

The goal of **abstraction** is to let you express *what you want* without spelling out every machine step. Compare:

```text
; assembly: load, add, store, by hand
LOAD  R1, price
LOAD  R2, qty
MUL   R3, R1, R2
STORE R3, total
```

```text
high-level: one readable line
total = price * qty
```

Both end up as the same machine code. But the second is shorter, clearer, harder to get wrong, and — crucially — *portable*: the same line can be translated for many different processors. That trade is the heart of high-level languages: give up a little fine control to gain enormous productivity and portability.

## Compilers: the translators

What turns that readable line into machine code is a **compiler** — a program whose job is translation. It reads your entire source file, checks it for errors, and produces an executable file of machine code the processor can run directly.

This is a profound idea, and it's only possible because of the stored-program design from lesson one: a compiler is just software that reads other software as data and writes new software out. Programs writing programs. (Some languages translate-and-run line by line instead of compiling ahead of time — that distinction gets its own treatment in [compiled vs interpreted](/learn/intro-software-dev/compiled-vs-interpreted/).)

The first compilers were a hard sell — many programmers doubted a machine could generate code as good as a human's hand-tuned assembly. They were proven wrong, and the productivity gain was so large there was no going back.

## The first great languages

Once compilers worked, languages bloomed — each shaped by the problem its creators cared about. A few from the early era still echo through computing today:

| Language | Year | Built for | Why it mattered |
|----------|------|-----------|-----------------|
| **FORTRAN** | 1957 | Scientific & numeric computing | First widely used high-level language; made fast math practical |
| **LISP** | 1958 | Symbolic / AI computing | Pioneered functional ideas; foundational to AI research |
| **COBOL** | 1959 | Business data processing | English-like syntax; still runs banks and governments |
| **ALGOL** | late 1950s | Algorithms & language design | Introduced block structure that shaped nearly every later language |
| **C** | 1972 | Systems programming | Close to the metal yet portable; still the backbone of low-level code |

**FORTRAN** (FORmula TRANslation) was built so scientists and engineers could write formulas almost as they'd write them on paper. That focus on fast numerical computation is directly relevant to signal work — the math behind filtering and demodulating a radio signal is exactly the kind of number-crunching FORTRAN was designed to make easy, and its descendants still dominate scientific computing.

**COBOL**, shaped by the work of **Grace Hopper** and a committee, used deliberately English-like syntax so business processes would be readable by non-specialists. Astonishingly, huge amounts of banking and government software still run on COBOL today.

**LISP** took a different path entirely — treating code and data as the same kind of list, and pioneering the **functional** style of programming. It became the language of early artificial intelligence research and seeded ideas that mainstream languages only adopted decades later.

**ALGOL** is the quiet influencer. Few people used it directly, but its **block structure** — grouping statements with clear nesting — became the template for almost every language that followed, from C to Python.

## C: the language radio code still speaks

If one language deserves special attention here, it's **C** (1972), created by **Dennis Ritchie** at Bell Labs alongside the Unix operating system. C hits a rare sweet spot: it's high-level enough to be readable and portable, yet low-level enough to give you direct, predictable control over memory and hardware. You can write fast, tight code without dropping all the way to assembly.

That combination made C the default language for operating systems, embedded devices, and — importantly for us — **digital signal processing**. Crunching a continuous stream of radio samples in real time demands exactly what C offers: speed, tight memory control, and predictable performance. A great deal of the SDR and DSP code running underneath GopherTrunk and its peers is written in C or its close relative C++, and decades of mature, battle-tested signal libraries exist in C. When you tune a software radio, somewhere underneath, C is doing the heavy math. The pioneers behind C and these other languages get their own profiles in [the next lesson](/learn/intro-software-dev/the-pioneers/).

<div class="knowledge-check" data-quiz data-correct-msg="Right — a compiler translates high-level source code into the machine code the processor runs." markdown="0">
  <p class="knowledge-check__q">Quick check: What is the job of a compiler?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">To run the computer's hardware directly without any software</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">To translate high-level source code into machine code the processor can run</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">To turn machine code back into the original English description</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **Machine code** — the raw numeric instructions a chip executes; fast for the processor, brutal for humans.
- **Assembly** — readable mnemonics that map one-to-one onto machine code, but still tied to one processor.
- **Abstraction** — high-level languages let you express *what* you want, gaining readability and portability for a little lost control.
- **Compilers** — programs that translate source code into machine code, making high-level languages practical.
- **The first greats** — FORTRAN for science, COBOL for business, LISP for AI, ALGOL for design influence.
- **C** — close to the metal yet portable, and still the backbone of the DSP and radio code under tools like GopherTrunk.

Next up: behind every one of these breakthroughs was a person. We meet the people who built programming — from Ada Lovelace to Margaret Hamilton — in [The people who built programming](/learn/intro-software-dev/the-pioneers/).
