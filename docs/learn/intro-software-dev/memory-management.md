---
slug: memory-management
title: Memory — manual, GC & ownership
description: Stack vs heap, manual malloc/free and its dangers, garbage collection and its pauses, and Rust's ownership model — three approaches to managing memory safely.
keywords: memory management, stack and heap, garbage collection, manual memory, malloc free, Rust ownership, borrow checker, use after free
level: intermediate
status: full
faq:
  - q: "What is the difference between the stack and the heap?"
    a: "The **stack** holds local variables and function call frames; it is fast and freed automatically when a function returns. The **heap** is a larger pool for data whose size or lifetime is not known at compile time; it must be allocated and eventually released, and managing that is where the hard problems live."
  - q: "What is garbage collection and what is its downside?"
    a: "**Garbage collection (GC)** automatically frees memory no longer reachable by the program, removing whole categories of bugs. Its downside is the **GC pause** — the runtime periodically does collection work, which can briefly stop or slow the program at unpredictable times."
  - q: "How does Rust manage memory without a garbage collector?"
    a: "Rust uses **ownership** and a compile-time **borrow checker**. Each value has one owner; when the owner goes out of scope the memory is freed automatically. The compiler enforces rules about borrowing references, guaranteeing memory safety with no runtime GC and no manual free."
---
# Memory — manual, GC & ownership

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
**Manual** — total control, but leaks and use-after-free bugs. **Garbage collection** — automatic and safe, but unpredictable pauses. **Ownership** — Rust's compile-time safety with no GC.
</div>

Every running program needs **memory** to hold its data, and that memory has to be
handed out when the program needs it and reclaimed when it is done. How a language
handles that reclamation is one of its defining characteristics — it shapes the
language's safety, its speed, and whether it is suitable for hard real-time work.
There are three broad strategies — **manual management**, **garbage collection**,
and **ownership** — but first we need the distinction that underlies all of them:
the stack versus the heap.

## Stack vs heap

A program's memory is split into two regions that behave very differently.

The **stack** is fast, ordered storage for local variables and the bookkeeping of
function calls. When a function is called, a *frame* is pushed; when it returns,
the frame is popped and its locals vanish automatically. Allocation is essentially
free — just move a pointer. The catch: the size of everything on the stack must be
known at compile time, and it lives only as long as its function.

The **heap** is a large, flexible pool for data whose size or lifetime is not
known up front — a buffer sized by user input, an object that must outlive the
function that created it, a growing list. Heap memory must be explicitly
*allocated* and eventually *released*. **All the hard memory problems live on the
heap**, because something has to decide when each piece is no longer needed.

<figure class="figure" markdown="0">
<svg viewBox="0 0 462 188" role="img" aria-label="Two memory regions side by side: on the left a call stack of frames for main, process, and decode, with the newest decode frame just pushed and popped on return; on the right a heap of variously sized blocks, one referenced by a pointer held in a stack frame." xmlns="http://www.w3.org/2000/svg">
  <text x="104" y="26" text-anchor="middle" font-size="9.5" font-weight="600" fill="currentColor">call stack</text>
  <g font-size="8.5" text-anchor="middle" fill="currentColor">
    <rect x="44" y="40" width="120" height="30" rx="4" fill="currentColor" fill-opacity="0.1" stroke="currentColor" stroke-width="1.1"/><text x="104" y="59">main( ) frame</text>
    <rect x="44" y="72" width="120" height="30" rx="4" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/><text x="104" y="91">process( ) frame</text>
    <rect x="44" y="104" width="120" height="30" rx="4" fill="currentColor" fill-opacity="0.22" stroke="currentColor" stroke-width="1.3" stroke-dasharray="4 3"/><text x="104" y="123">decode( ) frame</text>
  </g>
  <line x1="30" y1="40" x2="30" y2="134" stroke="currentColor" stroke-width="1.1" marker-end="url(#mem_ar)"/>
  <text x="20" y="87" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.9" transform="rotate(-90 20 87)">grows each call</text>
  <text x="104" y="150" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">newest frame on top —</text>
  <text x="104" y="160" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">pushed on call, popped on return</text>
  <text x="356" y="26" text-anchor="middle" font-size="9.5" font-weight="600" fill="currentColor">heap</text>
  <rect x="250" y="34" width="196" height="104" rx="5" fill="currentColor" fill-opacity="0.03" stroke="currentColor" stroke-width="1" stroke-dasharray="3 3"/>
  <g stroke="currentColor" stroke-width="1">
    <rect x="264" y="46" width="42" height="22" rx="3" fill="currentColor" fill-opacity="0.22"/>
    <rect x="318" y="42" width="34" height="26" rx="3" fill="currentColor" fill-opacity="0.14"/>
    <rect x="364" y="48" width="30" height="20" rx="3" fill="currentColor" fill-opacity="0.26"/>
    <rect x="404" y="44" width="30" height="28" rx="3" fill="currentColor" fill-opacity="0.18"/>
    <rect x="270" y="86" width="46" height="22" rx="3" fill="currentColor" fill-opacity="0.12"/>
    <rect x="330" y="90" width="34" height="24" rx="3" fill="currentColor" fill-opacity="0.22"/>
    <rect x="382" y="88" width="42" height="20" rx="3" fill="currentColor" fill-opacity="0.16"/>
  </g>
  <text x="348" y="128" text-anchor="middle" font-size="7.5" fill="currentColor" fill-opacity="0.9">any size · any lifetime · must be freed</text>
  <line x1="164" y1="119" x2="262" y2="63" stroke="currentColor" stroke-width="1.2" marker-end="url(#mem_ar)"/>
  <text x="205" y="102" text-anchor="middle" font-size="7" fill="currentColor" fill-opacity="0.9">pointer</text>
  <defs><marker id="mem_ar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Two regions with opposite rules. The <strong>stack</strong> holds a frame per active call — pushed on call, popped on return — so its storage is automatic and fixed-size. The <strong>heap</strong> is a free-form pool for data whose size or lifetime the compiler can't predict; a stack variable often just holds a pointer into it. Every hard memory bug lives on the heap, because something must decide when each block is no longer needed.</figcaption>
</figure>

```text
stack:  fast, auto-freed, fixed-size, function-scoped
heap:   flexible, manually/automatically managed, the source of memory bugs
```

## Manual management and its dangers

In **C and C++** you manage the heap yourself. You call `malloc` (or `new`) to
allocate and `free` (or `delete`) to release, and it is entirely your
responsibility to get it right.

```c
char *buf = malloc(1024);   // ask for memory
// ... use buf ...
free(buf);                  // you must release it — exactly once
```

This gives **maximum control and speed** — no runtime overhead, deterministic
timing — and it is why C and C++ dominate operating systems, embedded work and
high-performance DSP. But it is dangerous. The classic failure modes:

- **Memory leak** — you allocate but never `free`; the program's memory use grows
  until it slows or crashes.
- **Use-after-free** — you `free` memory then keep using the pointer; behaviour is
  undefined and often exploitable.
- **Double free** — you `free` the same block twice, corrupting the allocator.
- **Buffer overflow** — you write past the end of an allocation, clobbering adjacent
  memory; a leading cause of security vulnerabilities (more in
  [language-level security](/learn/intro-software-dev/language-security/)).

These bugs are notoriously hard to find because the symptom often appears far from
the cause.

## Garbage collection

The first big answer is to take the job away from the programmer. A **garbage
collector (GC)** is part of the language runtime that automatically finds memory no
longer reachable by the program and frees it. **Java, Go, Python, C# and
JavaScript** are all garbage-collected.

The benefit is enormous: **whole classes of bugs simply disappear**. No leaks from
forgotten frees, no use-after-free, no double frees. You allocate and forget; the
GC cleans up. This is a big part of why these languages are so productive.

The cost is the **GC pause**. Periodically the collector must do work — scanning
for reachable objects, reclaiming the rest — and depending on the algorithm this
can briefly **stop the program** ("stop-the-world") or steal CPU at unpredictable
moments. Modern collectors (Go's concurrent GC, Java's ZGC and G1) have shrunk
these pauses dramatically, often to well under a millisecond, but the pauses are
**non-deterministic**: you cannot be certain exactly when one will happen.

## Ownership: a third way

**Rust** takes a different path entirely — **memory safety without a garbage
collector and without manual `free`**, enforced at compile time. The mechanism is
**ownership** plus a **borrow checker**:

- Every value has exactly one **owner** (a variable).
- When the owner goes out of scope, the value is freed automatically — predictably,
  at a known point, with no runtime collector.
- You can **borrow** references to a value, but the compiler enforces rules: many
  shared (read-only) borrows, or exactly one mutable borrow, never both at once.

```rust
fn main() {
    let buf = vec![0u8; 1024]; // buf owns the heap allocation
    process(&buf);             // borrow it, read-only
}                              // buf goes out of scope here → freed automatically
```

The **borrow checker** rejects use-after-free, double-free, and data races *at
compile time*. There is no GC pause because there is no GC — frees happen at
deterministic points the compiler inserts. The trade-off is a steeper learning
curve: you must satisfy the borrow checker, which can feel like fighting the
compiler until the rules click. The payoff is C-like performance with
memory safety guaranteed.

| Approach | Languages | Safety | Runtime cost | Predictability |
|----------|-----------|--------|--------------|----------------|
| Manual | C, C++ | unsafe (your job) | none | fully deterministic |
| Garbage collection | Java, Go, Python | safe | GC pauses | non-deterministic |
| Ownership | Rust | safe (compile-time) | none | deterministic |

<div class="knowledge-check" data-quiz data-correct-msg="Right — Rust frees memory at compile-time-determined scope exits via ownership, so there's no garbage collector and no GC pause." markdown="0">
  <p class="knowledge-check__q">Quick check: how does Rust achieve memory safety without garbage-collection pauses?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">It runs a faster garbage collector in the background</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">Compile-time ownership rules free memory at known scope exits — no GC at all</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">It keeps everything on the stack and never uses the heap</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Reference counting and RAII

Between fully manual and fully garbage-collected sits a family of techniques worth
knowing, because you will meet them constantly.

**Reference counting** tracks how many references point at a heap object. Each new
reference bumps a counter; each one dropped decrements it; when it hits zero, the
object is freed immediately and deterministically. Python uses reference counting
as its primary mechanism (with a backup cycle collector), and C++'s
`shared_ptr` and Rust's `Rc`/`Arc` offer it on demand. It is simple and gives
prompt cleanup, but it has two costs: the bookkeeping overhead on every reference
change, and an inability to free **reference cycles** (two objects pointing at each
other keep each other alive), which is why Python needs a separate cycle collector.

**RAII** — *Resource Acquisition Is Initialisation* — is the C++ and Rust idiom of
tying a resource's lifetime to an object's scope. You acquire the resource in a
constructor and release it in a destructor, which the language runs automatically
when the object goes out of scope. RAII handles not just memory but files, locks
and sockets — anything that must be released — with the same deterministic,
scope-based rule that underpins Rust's ownership model. It is a big reason
deterministic languages can be safe *and* free of a garbage collector: cleanup is
guaranteed by the structure of the code, not by a periodic sweep.

## Why GC pauses matter for real-time radio

Here is where these abstractions bite in practice. A software-defined radio
delivers a relentless stream of [IQ samples](/learn/rf-sdr/iq-data/) — at 2.4
million samples per second, a new buffer arrives roughly every few milliseconds,
and the program must consume each one before the next arrives or the hardware's
buffer overflows and **samples are dropped**. Dropped samples mean corrupted
audio, missed packets, decode failures.

Now suppose a garbage-collected runtime decides to pause for a collection just as a
buffer needs processing. Even a sub-millisecond pause, at the wrong moment and
multiplied across a long capture, can cause overruns. This is the core reason hard
real-time DSP gravitates toward **manual** (C/C++) or **ownership-based** (Rust)
languages, where memory reclamation is **deterministic** and you control exactly
when it happens. Go is GC'd but is often used in radio tooling for the control and
orchestration layers — where occasional sub-millisecond pauses are harmless —
while the tightest sample-crunching stays in C or Rust. Picking the right tool per
layer is exactly the judgement in
[choosing a language for the domain](/learn/intro-software-dev/language-for-the-domain/).

## Recap

- **Stack vs heap** — the stack is fast and auto-freed; the heap is flexible and the
  source of memory bugs.
- **Manual management** (C/C++) — total control and speed, but risks leaks,
  use-after-free, double-frees and buffer overflows.
- **Garbage collection** (Java, Go, Python) — automatic and safe, eliminating whole
  bug classes, at the cost of unpredictable pauses.
- **Ownership** (Rust) — compile-time borrow checking gives memory safety with no GC
  and deterministic frees.
- **Determinism is the key axis for real-time** — manual and ownership models give
  predictable timing; GC does not.
- **GC pauses can drop radio samples** — which is why tight DSP avoids garbage
  collection.

Next up: doing many things at once without tripping over yourself — [concurrency
and parallelism](/learn/intro-software-dev/concurrency-models/).
