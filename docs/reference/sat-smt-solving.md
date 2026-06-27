---
slug: sat-smt-solving
title: SAT/SMT solving
entry_type: term
category: algorithms
description: SAT and SMT solvers decide whether a set of logical or bit-vector constraints can be satisfied, and return a satisfying assignment; in cryptanalysis they recover unknown keys or tables as the unique solution to the constraints a corpus imposes.
keywords: SAT solver, SMT solver, Z3, bit-vector, constraint satisfaction, CDCL, cryptanalysis, automated reasoning
aka: [SAT solving, SMT solving, Z3]
autolink: true
infobox:
  - { label: Type, value: Automated-reasoning method }
  - { label: SAT, value: Boolean satisfiability }
  - { label: SMT, value: "SAT + theories (e.g. bit-vectors)" }
  - { label: Engine, value: "CDCL search with clause learning" }
see_also: [algebraic-attack, constraint-propagation, brute-force-attack, known-plaintext-attack, rc4-cipher]
cite_urls:
  - https://en.wikipedia.org/wiki/Satisfiability_modulo_theories
  - https://en.wikipedia.org/wiki/Boolean_satisfiability_problem
---

**A SAT solver** decides whether a Boolean formula can be made true and, if so, returns an
assignment; an **SMT solver** extends this to richer theories such as fixed-width *bit-vectors*
and arrays, which model byte-oriented ciphers directly.[^smt][^sat] In cryptanalysis the
unknown key or substitution table becomes a set of variables, every known message becomes a
constraint, and the solver searches for the assignment satisfying all of them at once.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 96" role="img" aria-label="Many per-message constraints plus unknown table variables feed an SMT solver that returns the satisfying table or reports unsat." xmlns="http://www.w3.org/2000/svg">
  <rect x="18" y="22" width="120" height="20" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="78" y="36" text-anchor="middle" font-size="8" fill="currentColor">constraints (10k)</text>
  <rect x="18" y="56" width="120" height="20" fill="none" stroke="currentColor" stroke-width="1.1"/><text x="78" y="70" text-anchor="middle" font-size="8" fill="currentColor">T2 = 256 unknowns</text>
  <line x1="138" y1="32" x2="200" y2="44" stroke="currentColor"/><line x1="138" y1="66" x2="200" y2="52" stroke="currentColor"/>
  <rect x="202" y="32" width="78" height="32" fill="currentColor" fill-opacity="0.12" stroke="currentColor" stroke-width="1.2"/><text x="241" y="51" text-anchor="middle" font-size="9" fill="currentColor">SMT</text>
  <line x1="280" y1="48" x2="338" y2="36" stroke="currentColor" marker-end="url(#smar)"/><text x="344" y="40" font-size="8" fill="currentColor">table</text>
  <line x1="280" y1="48" x2="338" y2="64" stroke="currentColor" marker-end="url(#smar)"/><text x="344" y="68" font-size="8" fill="currentColor">unsat</text>
  <defs><marker id="smar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
</svg>
<figcaption>An over-determined system (far more constraints than unknowns) collapses to a single table — or the solver proves no such table exists for the assumed structure.</figcaption>
</figure>

## How it works

Modern solvers use conflict-driven clause learning (CDCL): they guess, propagate the
consequences, and on contradiction learn a clause that prunes a large part of the search.
This makes them far smarter than [brute force](/reference/brute-force-attack/) for structured
problems — they can recover a 256-entry table that is hopeless to enumerate. Their weakness is
*chained table lookups* (a value used to index another lookup), where the case-splitting
explodes; there a hand-written [constraint propagator](/reference/constraint-propagation/) can
be faster. An `unsat` result is also informative: it proves the assumed structure cannot fit
the data.

## Relevance to SDR

When a reverse-engineering problem reduces to "find the hidden table consistent with every
message," it is a satisfiability problem. GopherTrunk's clean-room analysis of the Motorola
P25 talker-alias obfuscation (issue #773) posed the unknown internal substitution table as
bit-vector variables in Z3 and asserted the [known-plaintext](/reference/known-plaintext-attack/)
constraints; the runs returned `unsat` across the tractable structure families, narrowing the
space rather than yielding the table.

## Sources

[^smt]: [Satisfiability modulo theories](https://en.wikipedia.org/wiki/Satisfiability_modulo_theories) — Wikipedia, for SMT and bit-vector/array theories.
[^sat]: [Boolean satisfiability problem](https://en.wikipedia.org/wiki/Boolean_satisfiability_problem) — Wikipedia, for SAT and CDCL search.
