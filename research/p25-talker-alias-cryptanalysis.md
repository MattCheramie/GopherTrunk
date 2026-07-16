# P25 Motorola talker-alias cipher — clean-room cryptanalysis findings (#773)

Status: **not cracked.** This documents what has been *established* about the
Motorola FACCH-S talker-alias obfuscation cipher from the 3,607-pair ground-truth
dataset (`er-imagery`, tag `gt-773-alias-groundtruth`), and the structural
hypotheses that have been *ruled out*, so the next investigation starts ahead and
doesn't repeat dead ends.

All derivation here is **data-driven only** — fitted to and validated against the
decoded-log *output* in the ground-truth CSV. No SDRTrunk source (GPLv3) was read
or ported; GopherTrunk is Apache-2.0 and its cipher stays gated
(`CipherVerified = false`) until a real alias decodes end-to-end.

These findings were also posted as a status update on
[#773](https://github.com/MattCheramie/GopherTrunk/issues/773), which is where any
confirmation from the SDRTrunk decoder author or follow-up chosen-plaintext
captures will be tracked.

## Framing (verified)

The reassembled message is `WACN | System | RadioID | encoded-alias | CRC-16`.
For an alias of `n` characters the **encoded-alias field is exactly `2n` bytes**,
followed by a **2-byte CRC-16** — i.e. the `encoded_hex` column is `2n + 2` bytes
and its **last 2 bytes are the header CRC, not cipher output.**

- Proof: the alias `P18` appears on two different RIDs with `encoded_hex`
  `956AB19AE437`**`D7FB`** and `956AB19AE437`**`99BA`** — identical first 6 bytes
  (the cipher-encoded alias), differing last 2 bytes (the RID-dependent CRC).
- GopherTrunk's framing already accounts for this (`alias.go`: `bits 56..end-16`
  are the encoded alias, `last 16 bits` are CRC-16/GSM). **The cipher operates on
  the first `2n` bytes only.**

> ⚠️ Earlier passive analysis (including a prior session's "16-bit nonlinear,
> ~13.6% non-deterministic high byte" conclusion) mistakenly fed the trailing
> CRC bytes into the cipher fit — ~11% contamination per message. **Strip the
> last 2 bytes before any cryptanalysis.** Doing so drops the high-byte
> non-determinism from 13.6% to 3.2% and exposes the structure below.

## Per-byte decode model (verified)

The decoded stream is `UTF-16BE(alias) + 0x00` padding (so even byte positions
decode to `0x00` for ASCII aliases). Each byte decodes as:

```
decoded[i] = int8( M_i · LUT[ encoded[i] ] + c_i )
```

- `LUT` is a fixed 256-entry int8 **substitution table, fully recovered** — it
  round-trips every byte of all 3,607 aliases given the per-character keystream
  (0 inconsistencies). (Recovered in gauge coordinates; see "gauge" below.)
- `(M_i, c_i)` is the per-character keystream: `M_i` odd (a multiplier; carries 7
  bits of the accumulator low byte via a mod-256 inverse), `c_i` additive
  (carries the accumulator high byte).
- At **even** positions `decoded = 0`, so `M·LUT[enc] + c = 0`, which means the
  accumulator **high byte is readable directly from the ciphertext**:
  `H_k = LUT[ encoded[2k] ]`. This gives 29,601 direct state observations
  (CRC-stripped) with no decoding required.

## The key structural finding: an observable memory-2/3 byte chain

With the CRC stripped, the cipher is **not** an opaque 16-bit black box — its
state is *observable* through the even-position high bytes `H_k = LUT[enc[2k]]`,
and the recurrence has short memory:

| Relation (all inputs readable from ciphertext) | Determinism |
|---|---|
| `H[k+1] ~ (H[k-1], H[k], eo[k])` (high-byte recurrence) | **functional — 4 conflicts / 10,080 keys** (memory-2) |
| `char[k] ~ (H[k-1], H[k], eo[k])` (the decode itself) | **2 conflicts / 13,416 keys** |
| keystream `(M,c) ~ (H[k-2], H[k-1], H[k])` | **12 inconsistent / 1,692** (memory-3) |

(`eo[k] = enc[2k+1]` is the odd ciphertext byte.) So the hidden accumulator low
byte is effectively **recovered from the previous high byte(s)** — two prior high
bytes pin the keystream almost completely. The residual handful of conflicts are
consistent with bit-errors plus the single unobservable low-bit.

This reframes the problem: it is a **memory-3 nonlinear byte recurrence over
observable high bytes**, not a search for a hidden 16-bit register.

## Product-form decode and the two component maps

The verified decode factors (the additive `c` is `−M·Hbyte`, so even positions
vanish because `LUT[enc] = Hbyte` there):

```
decoded[i] = M_i · ( LUT[ encoded[i] ] − Hbyte_i )      (mod 256, signed)
```

where `Hbyte_i` is the accumulator high byte at position `i` and `M_i` the
multiplier. This lets the per-character keystream be split into **two separate
observables** recovered from an affine fit of `char` vs `LUT[eo]` per index:
the multiplier `M_odd` and the odd-position high byte `Hbyte_odd` (the high byte
*between* two readable even high bytes — recovering it doubles the resolution of
the accumulator trajectory). Both are **clean 2-byte functions of the two
preceding even high bytes, with no ciphertext input**:

| Component map | Determinism |
|---|---|
| `Hbyte_odd ~ (H[k-1], H[k])` | 5 conflicts / 1,294 keys |
| `M_odd ~ (H[k-1], H[k])` | 19 conflicts / 1,294 keys |
| `H[k+1] ~ (Hbyte_odd, enc[2k+1])` (odd step) | 27 / 1,987 |

A closed form for `f1 = Hbyte_odd(H[k-1],H[k])` and `f2 = M_odd(H[k-1],H[k])`
would give a fully readable decoder
`char = f2(Hp,H) · (LUT[eo] − f1(Hp,H))`. **Neither fits** any tested closed form
(affine, quadratic, cross-term, LUT-composition, modular-inverse, GF(2⁸)
multiply — best ≈ 1,267/1,294, i.e. random), despite both being functional.

This *functional-but-never-closed-form* pattern (it recurs at every level —
high-byte recurrence, keystream, and now both component maps) is the signature of
a **second internal substitution table** distinct from the recovered output
`LUT`, applied inside the update. Testing that hypothesis (e.g. `Hbyte_odd =
T[H] ⊕ mask(Hp)`) is blocked by coverage: only one pair of `Hp`-groups shares
≥ 8 common `H` values in the passive corpus — far too sparse. **Chosen-plaintext
captures would supply exactly the dense, controlled `(Hp,H)` coverage needed to
recover such an internal table or rule it out.**

## A table decoder does NOT generalize (so a closed form is required)

Training the empirical map `G(H[k-1], H[k], eo[k]) → char` on 80% of messages and
decoding the held-out 20%: only **7.2% of held-out messages decode fully**, **70%
of characters correct**, **30% hit unseen `(H,H,eo)` contexts.** Coverage gaps
would garble live aliases, so a lookup table is not shippable. Only a **closed
form** for the recurrence (or per-character keystream) yields a general decoder.

## Structural classes ruled out (all on CRC-stripped clean data)

Each was tested by simulating the per-byte recurrence from the length seed and
checking the even-position high byte against the corpus through an auto-solved
affine output gauge (and, where noted, by direct algebraic fit):

- **Linear LCG** `s' = A·s + B (+input)` — ruled out two independent ways:
  brute over all `A,B` (both `s·A+B` and `(s+in)·A+B` orders, both seed models),
  **and** a direct 2-step algebraic fit `V' = A·V + S·input + B (mod 2^16)` in
  four state-coordinate handles. Both **ciphertext** and **plaintext** feedback.
- **Multiplicative recurrence mod a prime** `s' = (g·s + input + b) mod p` — 15
  moduli (65537, 65521, …, 32749, 32719), both seed/extraction modes.
- **Multiply-with-carry (MWC)** — 6 carry/input/output variants. (Tested because
  the memory-2/3 byte-coupling is the MWC signature; still negative.)
- **Xorshift, rotate-mix, xor-then-multiply, multiply-then-xor** full-state forms.
- **Low-degree polynomial** maps (linear/quadratic/cross-term) over Z/2^16 in
  four coordinate handles; and per-byte affine / quadratic over Z/256.
- **NLFSR / single unknown S-box of a linear tap**: `target = T[ g(inputs) ]`
  with `g` a linear (or xor) byte-combine — for the high-byte recurrence and for
  the keystream `M`,`c`. A single byte-projection cannot determine the target
  (best is the degenerate `g = H` baseline; adding taps only adds conflicts).
- Earlier (pre-CRC-fix, still informative): GF(2)-linear / LFSR / convolutional
  encode, GF(2⁸)-affine across 8 field polynomials, affine-group and S-box
  compositions, modular-inverse update families.

The per-character update is therefore a genuine **multi-input nonlinear function**
(note the verified decode itself contains a state×input product `M·LUT[eo]`, which
no "S-box-of-linear" form can express) that is none of the standard parameterized
generators.

## Gauge note

`LUT`, `M`, and `c` are recovered only up to a global affine gauge
(`M_t = M_g·Mt0`, `c_t = c_g + M_g·ct0`), and the cipher's *internal* substitution
table need not equal the recovered output `LUT`. Closed-form fitting in gauge
coordinates is therefore handicapped for nonlinear forms — the simulate-from-seed
brute (which works in true coordinates and applies the gauge only at the output
comparison) is the more reliable test and is what most of the ruling-out used.

## The 16-bit state machine, and why the constraint-solve attack stalls

Pursuing a global constraint/SMT solve for the suspected internal table sharpened the
target further. Define the per-character state `s[k] = (H_odd[k], M_odd[k])` (both
recovered per memory-3 context). Then the cipher is a **clean 16-bit Markov state
machine**:

```
(H_odd, M_odd, eo)  ->  (H_odd', M_odd')      EXACTLY FUNCTIONAL: 0 conflicts / 423 keys
```

one input byte `eo` per character, deterministic; and `M_odd` barely affects
propagation (`(H_odd, eo) -> next` is already 2-conflict). This is the tightest
characterization of the cipher to date.

The internal-table hypothesis was then solved for directly (Z3 + a custom
constraint propagator), both on the sparse 423-transition state machine **and** on the
**dense** even-H recurrence `H[k+1] = F(H[k-1], H[k], eo)` (~10,076 readable keys —
massively over-determined, no overfit possible). Result: **negative across the whole
tractable family** — single-table 1/2/3-lookup networks, two-table networks, balanced
Feistel (incl. a gauge-aware variant solving the table jointly with a free low-byte
permutation). A precise structural reason emerged from the dense solve: within any
merged `Hp ⊕ eo` group the same `H` maps to different `H[k+1]`, so **`H[k+1]` is not a
function of any merged 2-byte index** — which rules out every table-network ending in a
single lookup of a combined index.

Net: the update is a genuine multi-input nonlinear function whose true dependency is the
**hidden low byte**; the dense even-H relation only shadows it, and the full-state
machine that carries the low byte is sparse (423) and gauge-tangled on exactly that
byte. The blocker is **not** constraint count — it is that the one variable that matters
is under-observed in any passive corpus.

## Differential / dense-corpus results (2026-06, #773)

The reporter supplied a **dense differential dataset** on top of the 3,607-pair
corpus — sequential, prefix-sharing aliases (AGP 6300–8810, BALWKSTN, IRS 1–30,
P-series) whose shared plaintext prefixes force shared state trajectories, giving
the controlled `(H_prev, H)` coverage that random callsigns can't. Combined corpus:
~4,250 labeled messages. This is the dense-coverage data the "two paths" below
called for, short of true chosen-plaintext. Running it through the recovery
toolkit (the committed `alias propagate` mode) establishes:

- **The decode model is exactly right.** Every *fully-covered* message round-trips
  to its known plaintext under `char = M·(LUT[eo] − Hodd)`; the only mismatches are
  isolated **single-byte** errors that are the signature of bit errors in the
  capture, not a model error.
- **The per-character state machine is functional.** Harvesting the transition
  `T:(state, eo) → next-state` from the pinned states gives ~1,150 keys with only
  ~8 conflicting observations (bit-error level) — independently reconfirming the
  deterministic 16-bit machine from a second, denser corpus.
- **The length seed is a clean function of message length** (0 conflicts across the
  observed lengths) — `seed(n)` is pinned directly.
- **But coverage does not saturate.** Pinned contexts and distinct transition keys
  grow **~linearly** with corpus size (≈706/1,325/1,810/2,240 pinned and
  ≈250/605/896/1,179 keys at 25/50/75/100% of the corpus) rather than plateauing.
  A clean 80/20 split decodes only **~1.6% of held-out messages fully** (~52% of
  characters). The reachable state space is therefore large: a *passive* corpus,
  even a dense differential one, cannot reach full coverage by intersection +
  propagation alone.

**Net (answering the standing question):** GopherTrunk's *live* decode
(`DecodeAliasBytes`) is an **algebraic LCG placeholder** (`accum·293 + 0x72E9` + a
single LUT) — it does **not** model a second internal table, and that algebraic
family is exactly what the analysis above rules out. The real update is a
**deterministic, functional, but closed-form-free** per-character transition,
consistent with a **second internal substitution table** distinct from the output
LUT. The dense differential data **sharpens** the target (confirms the machine,
pins the seed, validates the decode) but does **not close** it, because the one
variable that matters — the hidden accumulator low byte folded through that
internal table — stays under-observed in any passive capture. The cipher stays
gated (`CipherVerified = false`).

## Chosen-plaintext results (2026-07, #773)

A chosen-plaintext corpus arrived on #773 (`moto_alias_t.zip`, from a third-party
working encoder/decoder — I/O logs only, consumed as data, no source read).
299 round-trip-verified records against the synthetic SUID `BEE00.ABC.123456`:
a length sweep of all-`A` (lengths 1–14), a first-character sweep at length 6
(`?AAAAA`, chars `0x20`–`0x7E`), a second-character sweep at length 6 (`A?AAAA`),
and a fourth-character sweep at length 8 (`AAA?AAAA`). The `Alias Encoded` field
is the pure `2n`-byte cipher output with **no CRC**, cleaner than the corpus.

What the controlled data establishes (reconfirming the machine, and pinning
states the passive corpus couldn't):

- **The memory-2 machine holds exactly on new data.** `(H[k-1], H[k], eo[k]) →
  H[k+1]` is functional with **0 conflicts / 1408 observations** across the sweeps.
- **LCG stays dead; the generic cell-solver doesn't close it.** `fromseed`
  mismatch ≈ 0.996; `cells` pins only 3/859 contexts on the chosen-plaintext,
  because the generic *intersection* method doesn't exploit the controlled
  structure — the differential method below is the right tool for this data.
- **Differential state pinning works cleanly.** At a swept position the pre-char
  state is fixed, so `FWD[C] = char·(L|1) + H` fits **95/95** and pins the
  odd-position state directly: byte 1 (len 6) `H=56, L|1=161` (Modd 97); byte 3
  (len 6, prefix `A`) `H=117, L|1=225` (Modd 33); byte 7 (len 8, prefix `AAA`)
  `H=206, L|1=11` (Modd 163). This is exactly the low-byte observability the
  passive corpus lacked.
- **The update transfer function is genuinely nonlinear.** `U(state, input).high`
  at a fixed state is not affine in the char, the value `FWD[C]`, or `C`, nor an
  `FWD`-composition of any of them (best 3/95) — the input hits the high byte
  through a substitution/multiply, not an add.
- **The update folds the ciphertext byte / value `FWD[C]`, not the plaintext char.**
  Matching the two pinned states by plaintext char, the next-high-byte delta
  `Hnext_b − Hnext_a` takes **73** distinct values; matching by value (≡ by
  ciphertext byte — they are bijective) it collapses to **2**. So the per-step
  update folds the ciphertext/value; the plaintext enters only via the encode map
  `value = char·(L|1)+H`. This pins `eo` as the *sole* input driver in
  `H[k+1] ~ (H[k-1],H[k],eo)`.
- **`Hnext = H_state + g(value)` with a low-byte branch — the sharpest lead.**
  Across all three pinned states the delta `Hnext_b(v) − Hnext_a(v)` takes exactly
  **two** values, and one is *always* the state high-byte difference `H_b − H_a`:
  `A1↔A3 {61,200} (Hdiff 61)`, `A1↔A7 {33,150} (Hdiff 150)`, `A3↔A7 {89,206}
  (Hdiff 89)`. So `Hnext@state(v) = H_state + g(v)` for a **state-independent**
  `g(v)`, except on a two-way branch (≈40 % of values) that shifts the result by a
  fixed ±139 and is governed by the state's low byte. No single-bit / threshold
  predicate on the value selects the branch (best 22/33 ≈ random) — i.e. the
  selector is the hidden low byte folded through an internal table, a concrete,
  quantified form of the "second internal table" hypothesis.
- **A single global 16-bit multiply is ruled out.** If the update were linear in
  the state (`(state + T[input])·MUL + ADD`), the two pinned states would force
  `Hnext_b − Hnext_a` to two *consecutive* values (a carry). They differ by 139,
  not 1 — the state does not enter linearly, and `value·Q+K` / `value·L_state +
  H_state·256` closed forms both fail at a fixed state (≤2/95).
- **The seed is itself nonlinear in length.** `H₀(n)` steps by −22 with irregular
  +95 jumps not explained by carries — `seed(n)` runs through the same machinery,
  not a `base + n·K` line.
- **The state map is non-bijective.** In the length sweep, lengths 7 and 9 share
  the high-byte trajectory from position 2 (`H(7)[2:] == H(9)[2:7]`). A bijective
  LCG/MWC/odd-multiply cannot merge trajectories — this is the fingerprint of a
  **non-invertible internal substitution table** inside the update, matching the
  "second internal table" hypothesis above.

**The low-byte attack hits an observability floor (not an effort limit).** The
high-byte update law `Hnext = H + g(value) + s·139` is well-determined, and `g`
is covered for 197/256 values. But a *decoder* also needs the **low-byte** update
`Lnext = LowUpdate(H, L, value)` to propagate the accumulator, and this data
exposes `L` only as `L|1` at the **three** swept positions — which are
non-consecutive (byte 1 and byte 3 straddle byte 2's unobserved `L`; byte 7 is a
different length/seed). So the number of observed `(L_in → L_out)` transitions is
**zero**, and `L`'s bit 0 is never observable at all (the decode multiplier forces
`L|1`). No amount of cryptanalysis recovers an update with zero observed
transitions — the branch bit `s(L, value)` is correspondingly only ~80 %
predictable by a multiply-carry (best 27/33, different multiplier per state-pair),
the residual being exactly that hidden low bit.

**Net:** the chosen-plaintext confirms the 16-bit machine, pins the seed and three
exact states, determines the high-byte update law, and recovers `g` over most of
its domain — but the low-byte update and the branch table are *observationally*
out of reach from sweeps at only three positions. `CipherVerified` stays
**false**. The decisive remaining capture is Tier-3-style and now has a precise
justification: single-character sweeps at **consecutive** positions (so
`L_in → L_out` becomes observable) across **more** positions and a second length —
see `p25-talker-alias-chosen-plaintext.md`.

> **Data-access note (also unblocks #813).** The gated GitHub attachments
> (`user-attachments/...`, fork release assets) that this session's network policy
> 403s can be retrieved by letting `WebFetch` resolve the `github.com` link — it
> returns the signed `objects.githubusercontent.com` redirect — then `curl`-ing
> that signed URL directly (that host is reachable; the 5-minute signature window
> is ample).

## Two paths to finish

1. **Chosen-plaintext captures** from a programmable Motorola radio — see
   `p25-talker-alias-chosen-plaintext.md`. A first tranche arrived (see the
   2026-07 results above) and confirmed the machine + pinned the seed, but was not
   dense enough to recover the internal table; a wider positional/length sweep is
   the specific follow-up. The memory-3 finding *sharpens* this:
   a short length sweep plus single-character differentials at a couple of
   positions is enough to pin the memory-2/3 transition directly, because the
   state is now observable rather than hidden. The dense-corpus result above is
   the strongest evidence yet that this is the decisive missing input: passive
   coverage grows linearly and never closes, whereas a controlled sweep hits each
   `(state, eo)` context deliberately.
2. **Harder clean-room cryptanalysis** of the recovered 16-bit state machine. The
   straightforward constraint/SMT solve for a small internal-table network is
   **exhausted** (see above), so the remaining clean-room directions are
   research-grade: latent-bit recovery of the hidden low byte to complete the
   full-state machine, large-compute program synthesis over a richer operation set,
   or matching the FSM fingerprint to a catalogued primitive. All derived solely from
   the dataset — the only known reference implementation is GPLv3 and is intentionally
   **not** consulted; this project's decode must be an independent, permissively
   licensed derivation.

## Reproducing

The recovery toolkit is committed under `internal/cryptolab/subjects/motorola`
(build tag `cryptolab`). Point any mode at a `rid,talkgroup,encoded_hex,alias`
corpus — the trailing 2 CRC bytes are stripped on load:

```
gophertrunk cryptolab alias gauge     -csv corpus.csv   # affine gauge sweep
gophertrunk cryptolab alias structure -csv corpus.csv   # merged-index wiring enumeration
gophertrunk cryptolab alias cells     -csv corpus.csv   # per-context state pinning (resumable)
gophertrunk cryptolab alias fromseed  -csv corpus.csv   # simulate candidate update families
gophertrunk cryptolab alias propagate -csv corpus.csv   # harvest (state,eo)->state transition + seed; coverage
```

The `propagate` mode produced the dense-corpus results above. Earlier scratch
tooling (not committed) covered even-H extraction and determinism sweeps, the
LUT/keystream recovery, the simulate-from-seed brutes (linear / mod-prime / MWC /
nonlinear / both-feedback), the train/test table decoder, and the NLFSR/S-box
search.
