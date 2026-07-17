---
slug: data-structures-algorithms
title: Data structures & algorithms, briefly
description: A plain-language tour of the handful of data structures worth knowing, how to pick the right container for a job, and how Big-O notation describes the way work grows as your input gets bigger.
keywords: data structures, algorithms, Big-O notation, time complexity, array, linked list, hash map, hash table, stack, queue, binary tree, ring buffer
level: intermediate
status: full
prereq:
  - clean-code
faq:
  - q: "Do I need to memorize sorting algorithms?"
    a: "No. You almost never write a sort by hand — every language ships a fast, well-tested one. What matters is knowing the *tradeoffs*: that sorting costs about O(n log n), that a sorted list lets you binary-search in O(log n), and when reaching for the standard library's sort is the right move. Understand the shape of the cost, not the line-by-line recipe."
  - q: "What is Big-O notation?"
    a: "Big-O is shorthand for how the work an algorithm does grows as its input grows. O(1) means constant — the size doesn't matter. O(n) means the work scales in step with the input. O(n^2) means doubling the input roughly quadruples the work. It ignores constant factors and focuses on the shape of the curve, which is what decides whether code stays fast at scale."
  - q: "What data structure should I use?"
    a: "Start from the operation you do most. Need fast lookup by a key? A hash map. Need order and index access? An array or list. Need to test membership? A set. Need a fixed rolling window of recent samples? A ring buffer. Pick the container whose cheap operation matches your hot path, and the rest of the code tends to fall into place."
  - q: "Why do data structures matter if the language has built-ins?"
    a: "Because the built-ins *are* the data structures — the language hands you arrays, maps, and sets, but it can't tell you which one fits your problem. Choosing wrong doesn't fail to compile; it just runs slowly and reads awkwardly. Knowing what each container is good at is how you pick correctly."
gophertrunk_links:
  - title: Architecture
    url: /architecture.html
    note: how GopherTrunk's real-time pipeline is organized — ring buffers carry samples between stages so no stage blocks another.
---

# Data structures & algorithms, briefly

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
A **data structure** is a way to organize data so the operations you care about are
cheap. An **algorithm** is the recipe that acts on it. **Big-O** describes how the cost
grows as the input grows. The craft isn't implementing these from scratch — you rarely
do — it's **choosing the right container** and knowing how its cost scales. Pick well
and the code stays simple and fast; pick badly and it turns slow and tangled.
</div>

This is a short overview, not a computer-science course. Whole textbooks and semesters
are devoted to this topic; the goal here is to give you a working mental model — enough
to recognize the handful of structures you'll actually use, choose between them with
confidence, and read the Big-O labels that libraries and colleagues throw around. Once
you have written [readable code](/learn/intro-software-dev/clean-code/), the next lever
on quality is making sure the shape of your data fits the work you ask of it.

## What a data structure is

A data structure is simply a way of organizing data in memory so that certain operations
are cheap. That's the whole idea. Every structure makes some things fast and other things
slow, and the trade it makes is the reason it exists.

The practical consequence is that the *right* structure makes code both simple and fast,
while the *wrong* one makes it slow and tangled. If you find yourself writing loops inside
loops to hunt for an item, or shuffling elements around to keep something in order, that's
often a sign the data is in the wrong container — a better-fitting structure would make the
awkward operation disappear. Choosing well is a design decision, closely related to
[abstraction and coupling](/learn/intro-software-dev/abstraction-coupling/): the structure
you pick shapes the code that uses it.

## The essential few

You can go a long way with a small vocabulary of structures. Here are the ones worth
knowing on sight.

**Arrays and lists** hold items in order, one after another, reachable by index. Reading
the *i*-th element is instant, and iterating in order is natural. *Reach for it when* you
need a sequence you'll walk through or index into — the default container for "a bunch of
things in order."

**Maps and hash tables** store key→value pairs and let you look a value up by its key
almost instantly, no matter how many entries there are. *Reach for it when* you need to
answer "what's associated with this key?" quickly — counting occurrences, caching results,
indexing records by ID.

**Sets** store unique items and answer one question fast: "is this in here?" They're a hash
map without the values. *Reach for it when* you need membership or de-duplication — "have I
seen this one already?"

**Stacks and queues** control the *order* things come back out. A stack is LIFO
(last-in-first-out — like a stack of plates); a queue is FIFO (first-in-first-out — like a
line at a counter). *Reach for a stack when* you need to undo or backtrack; *reach for a
queue when* you need to process items in arrival order.

**Trees** arrange items in a hierarchy or keep them sorted for fast search. A binary search
tree, for instance, lets you find, insert, and remove while staying ordered, all in about
O(log n) time. *Reach for it when* you need hierarchy (a file system, a parse tree) or an
ordered collection you search often.

**Ring buffers** (circular buffers) are fixed-size arrays that wrap around: once full, each
new item overwrites the oldest. They never grow and never allocate after setup, which makes
them ideal for streaming a continuous flow of data through a fixed window — exactly the job
in an audio or IQ-sample pipeline, where a producer keeps handing samples to a consumer that
has to keep up. *Reach for it when* you need a bounded rolling window of the most recent data.
This is a core building block in real-time systems; see
[concurrency and pipelines](/learn/intro-software-dev/concurrency-and-pipelines/).

## Choosing the right one

The trick is to start from the operation you do most often, then pick the structure that
makes *that* operation cheap. A few concrete cases:

- **Need fast lookup by key?** → a **hash map**. Retrieving a value by its key is roughly
  constant time however large the map gets.
- **Need order plus index access?** → an **array** (or list). You get the sequence and
  instant access to any position.
- **Need to test membership repeatedly?** → a **set**. "Have I seen this before?" answered
  in near-constant time.
- **Need a fixed rolling window of samples?** → a **ring buffer**. A bounded window of the
  latest data with no per-item allocation.

None of these are exotic. The skill is matching the shape of your problem to the container
whose cheap operation is the one you lean on.

## Algorithms & Big-O

An algorithm is a recipe — a sequence of steps that turns some input into a result. Sorting
a list, searching for a name, finding the shortest route: each is an algorithm, and each has
a *cost* that grows as the input gets bigger. **Big-O notation** is the shorthand for that
growth. It ignores constant factors and machine speed and focuses on the one thing that
decides whether code survives at scale: the shape of the curve.

The common shapes, from best to worst, with an everyday analogy:

- **O(1) — constant.** The work is the same no matter how big the input. *Looking up a word
  in a dictionary by flipping straight to a known page number.* A hash-map lookup is O(1).
- **O(log n) — logarithmic.** Each step throws away half of what's left. *Guessing a number
  between 1 and 1,000 by always halving the range — you're done in about ten guesses.* Binary
  search over sorted data is O(log n).
- **O(n) — linear.** The work grows in step with the input. *Reading every name on a
  guest list once.* Scanning a list to find a match is O(n).
- **O(n log n) — linearithmic.** A little worse than linear; the practical ceiling for good
  sorting algorithms. *Sorting that guest list well.*
- **O(n^2) — quadratic.** Double the input and the work roughly quadruples. *Comparing every
  guest against every other guest.* Two nested loops over the same data is the classic O(n^2).

The one to watch is O(n^2). It hides easily — a loop inside another loop, both walking the
same collection — and stays invisible in testing because on ten items it's instant. Then the
input grows to a hundred thousand items in production and the same code grinds to a halt. A
lot of "why is this suddenly so slow?" turns out to be a quadratic loop that nobody noticed
until the data got big.

## You don't build these from scratch

Here's the reassuring part: you almost never implement these yourself. Every mainstream
language ships them in its standard library — Go has slices, maps, and `container` types;
Python has lists, dicts, and sets; Java, Rust, C++ and the rest all provide the same
essentials, tested and tuned far better than a hand-rolled version. Different
[language families](/learn/intro-software-dev/language-families/) name and package them
differently, but the ideas carry across all of them.

Your job is not to reinvent a hash table. It's to **pick the right one** for the problem and
**know what it costs**. That's the reusable skill: the containers change names from language
to language, but "fast lookup by key wants a hash map" is true everywhere.

## Where it shows up in real-time code

The stakes rise in real-time and streaming programs. A DSP pipeline like GopherTrunk
receives a relentless flow of samples from the radio and *cannot fall behind* — if a
processing stage takes too long, samples pile up or get dropped and the decode fails. That
constraint shapes the structures it uses.

Ring buffers carry samples between stages so a fast producer and a slower consumer don't have
to move in lockstep. The hot loops — the code that runs on every sample — are kept at O(n) or
better, because anything quadratic in the per-sample path would instantly fall behind the
incoming data. And because the samples never stop, the code has to handle overruns and gaps
gracefully rather than crash; that's the discipline of
[robustness and error handling](/learn/intro-software-dev/robustness-and-errors/) meeting the
[concurrency and pipelines](/learn/intro-software-dev/concurrency-and-pipelines/) that keep a
stream flowing. Choosing the right structure isn't academic here — it's the difference between
keeping up with the radio and losing the signal.

<div class="knowledge-check" data-quiz data-correct-msg="Right — a hash set (or hash map) answers 'have I seen this?' in about O(1), so it stays fast no matter how many IDs you've checked." markdown="0">
  <p class="knowledge-check__q">Quick check: you need to check "have I seen this radio ID before?" millions of times. Best structure?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">A plain array you scan from the start each time (O(n) per check)</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">A hash set / hash map (O(1) lookup)</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">A stack, because IDs arrive in order</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- A **data structure** organizes data so the operations you care about are cheap; the right
  one makes code simple and fast, the wrong one makes it slow and tangled.
- The **essential few** — arrays/lists, maps, sets, stacks/queues, trees, and ring buffers —
  cover most everyday needs.
- **Choose from your hot path**: fast lookup by key → hash map; order plus index → array;
  membership → set; rolling window of samples → ring buffer.
- **Big-O** describes how cost grows with input size; watch for **O(n^2)** hidden in nested
  loops — it bites only once the data gets big.
- You **don't build these from scratch** — the standard library provides them; your job is to
  pick correctly and know the cost.
- **Real-time code** leans on ring buffers and O(n)-or-better hot loops because it can't fall
  behind the incoming samples.

Next up: Module 4 turns to named, reusable solutions to recurring problems — [what is a design pattern?](/learn/intro-software-dev/what-are-patterns/).
