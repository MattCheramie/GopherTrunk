# cryptolab SMT structural search (optional)

This directory holds an **optional** Python/Z3 search for the per-character
state update of the length-seeded byte-obfuscator studied by the `alias`
subject. It explores richer structural forms than the in-binary merged-index
propagator (`engine/propagate`): single/double internal tables, 1–3 lookups,
and a small balanced-Feistel family, solved as a constraint system over the
recovered high-byte transitions.

It is **not** part of the Go build and is not required to use the toolkit. It
consumes only the high-byte transitions the Go side exports, and runs in the
cloud environment.

## Setup

```
pip install -r requirements.txt
```

## Input

Export the dense high-byte transitions from the Go side, then point the
solver at them:

```
# (planned) export helper writes one "Hprev,Hcur,eo,Hnext" line per transition
gophertrunk cryptolab -out ./out alias structure -csv ground_truth.csv
python solve_structure.py --transitions ./out/high-transitions.csv
```

`solve_structure.py` reads a CSV of `Hprev,Hcur,eo,Hnext` rows (decimal
0..255) and searches the structural families for a single internal table
that reproduces them. It prints `UNSAT` for families ruled out and the
recovered table for any family that fits.

## Status

The single-merged-index and 2-round single-table forms are UNSAT on the full
transition set: the high byte is not a function of any merged 2-byte index,
because its true dependency is on the hidden accumulator low byte. This solver
is the place to extend the search to ≥3-round / two-table forms.
