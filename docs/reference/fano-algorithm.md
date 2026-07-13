---
slug: fano-algorithm
title: Fano algorithm (sequential decoding)
entry_type: algorithm
category: error-correction
description: The Fano algorithm is a depth-first sequential decoder for convolutional codes that follows one path through the code tree under a running threshold, backing up when the path score falls.
keywords: Fano algorithm, sequential decoding, convolutional code, code tree, Fano metric, running threshold, stack algorithm, Viterbi alternative, long constraint length
aka: [Fano algorithm, Fano sequential decoding, sequential decoding]
autolink: true
infobox:
  - { label: Type, value: Sequential decoder }
  - { label: Decodes, value: Convolutional codes }
  - { label: Effort, value: Variable (SNR-dependent) }
see_also: [convolutional-code, viterbi-algorithm, forward-error-correction, maximum-likelihood-sequence-estimation, trellis-coded-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Sequential_decoding
  - https://ieeexplore.ieee.org/document/1054469
---

The **Fano algorithm** is a *sequential decoding* method for
[convolutional codes](/reference/convolutional-code/): it explores the code's
decision tree one path at a time, moving forward while a running score stays above
a moving **threshold** and backing up to try alternatives when the score drops.[^wiki]
Introduced by Robert Fano in 1963, it was for years the practical way to decode the
long-constraint-length codes used on deep-space and early data links, before the
[Viterbi algorithm](/reference/viterbi-algorithm/) became dominant for short codes.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 170" role="img" aria-label="A binary code tree with one bold path advancing to the right, a dashed branch where the decoder backed up after its running metric fell below a threshold line, then resumed." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="fanoar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <line x1="30" y1="150" x2="440" y2="150" stroke="currentColor" stroke-width="0.8" stroke-dasharray="3 3" stroke-opacity="0.6"/>
  <text x="34" y="163" font-size="8" fill="currentColor" fill-opacity="0.8">running threshold T</text>
  <g stroke="currentColor" fill="none" stroke-width="1">
    <line x1="40" y1="90" x2="120" y2="60"/><line x1="40" y1="90" x2="120" y2="120"/>
    <line x1="120" y1="60" x2="200" y2="45"/><line x1="120" y1="60" x2="200" y2="85"/>
    <line x1="120" y1="120" x2="200" y2="110"/><line x1="120" y1="120" x2="200" y2="135"/>
  </g>
  <g stroke="currentColor" stroke-width="2.4" fill="none">
    <line x1="40" y1="90" x2="120" y2="60"/><line x1="120" y1="60" x2="200" y2="45"/><line x1="200" y1="45" x2="290" y2="55" marker-end="url(#fanoar)"/>
  </g>
  <line x1="120" y1="60" x2="200" y2="85" stroke="currentColor" stroke-width="1.4" stroke-dasharray="4 3"/>
  <text x="215" y="88" font-size="8" fill="currentColor" fill-opacity="0.8">back up</text>
  <g fill="currentColor"><circle cx="40" cy="90" r="3"/><circle cx="120" cy="60" r="3"/><circle cx="200" cy="45" r="3"/></g>
  <text x="300" y="59" font-size="9" fill="currentColor">best path so far</text>
</svg>
<figcaption>Fano's decoder advances along one tree path while its Fano metric stays above a threshold, and backtracks to explore siblings when the metric drops.</figcaption>
</figure>

## How it works

A convolutional encoder can be viewed as walking a **binary tree**: each information
bit chooses a branch, and each branch emits a block of coded bits. Decoding is the
search for the tree path whose emitted bits best match the received sequence. The
optimal search would examine the whole tree, but that is exponential in message
length. Sequential decoding instead follows a single **best-guess path**, guided by
the **Fano metric** — a per-branch score that rewards agreement with the received
symbols and subtracts a bias so that *longer* paths are not automatically penalised
against shorter ones. The bias makes the metric of the correct path tend to *rise*
with depth while wrong paths tend to *fall*.

The Fano algorithm turns this into a low-memory procedure driven by a single
**running threshold** `T`:

- **Move forward** to the better branch while the accumulated metric stays at or
  above `T`. When it comfortably clears `T`, *tighten* `T` upward by a step `Δ`.
- **Back up** when every branch ahead drops below `T`: retreat toward the root,
  trying not-yet-explored siblings.
- If backing up also fails (the whole neighbourhood is below `T`), **lower `T`** by
  `Δ` and try again, admitting paths that were previously rejected.

Because it keeps only the current path and the threshold — not a full trellis — the
Fano algorithm needs very little memory, which is precisely why it scaled to
constraint lengths far beyond what a Viterbi decoder could afford in the 1960s–70s.

## Variants

The **stack (or ZJ) algorithm** is the other classic sequential decoder: it keeps an
ordered stack of partial paths and always extends the best one, avoiding repeated
back-and-forth but demanding memory to hold the stack. Fano's method trades that
memory for occasional re-traversal of the same branches. Both share the same
statistical behaviour, including the defining weakness below.

## In practice

Sequential decoding's effort is **variable and data-dependent**. On a clean channel
the correct path clears every threshold and the decoder races to the end with almost
no backtracking. As the signal-to-noise ratio falls toward the **computational
cutoff rate** `R₀`, the number of tree nodes visited per decoded bit follows a
heavy-tailed (Pareto) distribution: a single bad noise burst can trigger an
avalanche of backtracking whose expected work is unbounded. Real systems therefore
cap the computation and declare an **erasure** or request a retransmit when the
budget is exhausted. This unpredictable, bursty workload — versus Viterbi's fixed,
constraint-length-bounded cost — is the central engineering trade-off between the
two, and the main reason short-constraint-length codes migrated to Viterbi while
sequential decoding survived where long codes and large coding gains mattered more
than latency.

## Relevance to SDR

Fano-style sequential decoding was the workhorse of early deep-space telemetry
(the Pioneer missions used long convolutional codes decoded sequentially) and
appeared in HF and satellite data modems where its large coding gain justified the
variable latency. In modern trunked-radio and consumer wireless links, the
short-constraint-length convolutional codes in [P25](/reference/project-25/),
[DMR](/reference/dmr/), Wi-Fi and LTE are decoded with the fixed-cost
[Viterbi algorithm](/reference/viterbi-algorithm/) instead, so **GopherTrunk** does
not run a Fano decoder in its own chain. The algorithm remains important as the
conceptual bridge between brute-force tree search and the
[maximum-likelihood sequence estimation](/reference/maximum-likelihood-sequence-estimation/)
that Viterbi made tractable, and it is the historical answer to "how do you decode a
code too long to trellis?"

## Sources

[^wiki]: [Sequential decoding](https://en.wikipedia.org/wiki/Sequential_decoding) — Wikipedia, for the Fano algorithm, the Fano metric, the running threshold, the stack algorithm, and the computational-cutoff-rate behaviour.
