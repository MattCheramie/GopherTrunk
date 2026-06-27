#!/usr/bin/env python3
"""Optional Z3 structural search for a byte-obfuscator's state update.

Consumes only the high-byte transitions the Go toolkit exports
(Hprev,Hcur,eo,Hnext per row) and searches a few small structural families
for a single internal 256-byte table T2 that reproduces them.

This is a research scaffold, not part of the Go build. See README.md.
"""

import argparse
import csv
import sys


def load_transitions(path):
    rows = []
    with open(path, newline="") as f:
        for rec in csv.reader(f):
            if not rec or rec[0].strip().lower().startswith("hprev"):
                continue
            if len(rec) < 4:
                continue
            rows.append(tuple(int(x) & 0xFF for x in rec[:4]))
    return rows


def merged_index_functional(rows, combine):
    """Fast pre-check (no solver): is Hnext a function of combine(inputs)?

    Returns the conflict count. Zero means a single lookup of that merged
    index already reproduces the data — no solver needed.
    """
    table = {}
    conflicts = 0
    for hp, hc, eo, hn in rows:
        idx = combine(hp, hc, eo)
        if idx in table:
            if table[idx] != hn:
                conflicts += 1
        else:
            table[idx] = hn
    return conflicts, len(table)


COMBINES = {
    "hc": lambda hp, hc, eo: hc,
    "hc^eo": lambda hp, hc, eo: hc ^ eo,
    "hp^hc": lambda hp, hc, eo: hp ^ hc,
    "hp^hc^eo": lambda hp, hc, eo: hp ^ hc ^ eo,
    "(hc+eo)": lambda hp, hc, eo: (hc + eo) & 0xFF,
}


def solver_search(rows):
    try:
        from z3 import Array, BitVec, BitVecSort, Select, Solver, sat
    except ImportError:
        print("z3 not installed; run: pip install -r requirements.txt", file=sys.stderr)
        return

    bv8 = BitVecSort(8)
    # Family: Hnext = T2[Hcur ^ eo] ^ Hprev  (one table, 2 lookups-worth of mix)
    T2 = Array("T2", bv8, bv8)
    s = Solver()
    for hp, hc, eo, hn in rows:
        s.add(Select(T2, BitVec(hc ^ eo, 8)) ^ BitVec(hp, 8) == BitVec(hn, 8))
    print("family T2[hc^eo]^hp:", "SAT" if s.check() == sat else "UNSAT")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--transitions", required=True,
                    help="CSV of Hprev,Hcur,eo,Hnext rows (decimal 0..255)")
    args = ap.parse_args()

    rows = load_transitions(args.transitions)
    if not rows:
        print("no transitions loaded", file=sys.stderr)
        sys.exit(1)
    print(f"loaded {len(rows)} transitions")

    print("merged-index pre-checks (conflicts / distinct keys):")
    for name, fn in COMBINES.items():
        conf, keys = merged_index_functional(rows, fn)
        print(f"  {name:12s} {conf:6d} / {keys}")

    solver_search(rows)


if __name__ == "__main__":
    main()
