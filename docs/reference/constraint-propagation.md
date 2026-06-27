---
slug: constraint-propagation
title: Constraint propagation
entry_type: term
category: algorithms
description: Constraint propagation narrows the possible values of unknown variables by repeatedly applying the constraints that link them — assigning one value forces others — and is the engine inside constraint solvers and a fast hand-built alternative for table-recovery problems.
keywords: constraint propagation, constraint satisfaction, CSP, forward checking, unit propagation, arc consistency, backtracking, table recovery
aka: [constraint propagation, forward checking]
autolink: true
infobox:
  - { label: Type, value: Constraint-satisfaction technique }
  - { label: Idea, value: One assignment forces others }
  - { label: Pattern, value: "Propagate → branch → backtrack" }
  - { label: Strong when, value: Dense constraints cascade }
see_also: [sat-smt-solving, algebraic-attack, brute-force-attack, known-plaintext-attack, scrambling]
cite_urls:
  - https://en.wikipedia.org/wiki/Constraint_satisfaction_problem
  - https://en.wikipedia.org/wiki/Local_consistency
---

**Constraint propagation** solves a constraint-satisfaction problem by repeatedly using the
constraints to shrink each variable's set of possible values: fixing one variable *forces*
others, whose new values force still more, cascading until the problem is solved or a
contradiction appears.[^csp][^lc] It is the propagation engine inside
[SAT/SMT solvers](/reference/sat-smt-solving/) and, hand-written for a specific problem
shape, is often dramatically faster than a general solver.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="Assigning one table cell forces several others through the constraints, which cascade until solved or contradicted." xmlns="http://www.w3.org/2000/svg">
  <circle cx="60" cy="48" r="15" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.2"/><text x="60" y="51" text-anchor="middle" font-size="8" fill="currentColor">set</text>
  <line x1="75" y1="44" x2="135" y2="32" stroke="currentColor" marker-end="url(#cpar2)"/><line x1="75" y1="52" x2="135" y2="64" stroke="currentColor" marker-end="url(#cpar2)"/>
  <circle cx="150" cy="28" r="13" fill="none" stroke="currentColor" stroke-width="1.1"/><circle cx="150" cy="68" r="13" fill="none" stroke="currentColor" stroke-width="1.1"/>
  <line x1="163" y1="26" x2="220" y2="22" stroke="currentColor" marker-end="url(#cpar2)"/><line x1="163" y1="30" x2="220" y2="44" stroke="currentColor" marker-end="url(#cpar2)"/><line x1="163" y1="68" x2="220" y2="60" stroke="currentColor" marker-end="url(#cpar2)"/>
  <g stroke="currentColor"><circle cx="234" cy="20" r="10" fill="none"/><circle cx="234" cy="48" r="10" fill="none"/><circle cx="234" cy="74" r="10" fill="none"/></g>
  <text x="300" y="51" font-size="8" fill="currentColor">… cascade → solved / conflict</text>
  <defs><marker id="cpar2" markerWidth="7" markerHeight="7" refX="5" refY="2.5" orient="auto"><path d="M0 0 L5 2.5 L0 5 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>Dense constraints make one assignment cascade through many cells, so the search either converges quickly or contradicts itself fast.</figcaption>
</figure>

## How it works

The solver keeps a partial assignment and a worklist. Each newly fixed variable triggers the
constraints that mention it, which may fix or restrict further variables (forward checking /
unit propagation); contradictions trigger backtracking. When the constraints are *dense* —
many per variable — a single seed assignment forces a long chain, so most wrong guesses die
almost immediately. This is exactly the regime where a purpose-built propagator beats a
general [SMT solver](/reference/sat-smt-solving/) on chained-lookup problems that otherwise
cause case-split blow-up.

## Relevance to SDR

Recovering a hidden byte table from observed transitions is a constraint-satisfaction problem:
each observation fixes or links table cells. In GopherTrunk's clean-room analysis of the
Motorola P25 talker-alias obfuscation (issue #773), a custom propagator over the table cells
decided in a single propagation step that a candidate structure was inconsistent — a result
the general solver could not reach because the chained lookups stalled it.

## Sources

[^csp]: [Constraint satisfaction problem](https://en.wikipedia.org/wiki/Constraint_satisfaction_problem) — Wikipedia, for variables, constraints, and propagation/backtracking.
[^lc]: [Local consistency](https://en.wikipedia.org/wiki/Local_consistency) — Wikipedia, for constraint propagation and forward checking.
