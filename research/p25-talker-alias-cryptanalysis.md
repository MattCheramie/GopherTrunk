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

## Two paths to finish

1. **Functional spec from the reference author** (`@ilyacodes`,
   `MotorolaTalkerAliasComplete.java`): the exact accumulator seed, the per-byte
   state update, and the state→`(M,c)` extraction. Given the memory-3 observation
   above, even a prose description of the update would close it immediately.
2. **Chosen-plaintext captures** from a programmable Motorola radio — see
   `p25-talker-alias-chosen-plaintext.md`. The memory-3 finding *sharpens* this:
   a short length sweep plus single-character differentials at a couple of
   positions is enough to pin the memory-2/3 transition directly, because the
   state is now observable rather than hidden.

## Reproducing

Scratch tooling used for these results (not committed): even-H extraction and
determinism sweeps, the LUT/keystream recovery, the simulate-from-seed brutes
(linear / mod-prime / MWC / nonlinear / both-feedback), the train/test table
decoder, and the NLFSR/S-box search. All read the ground-truth CSV and strip the
trailing 2 CRC bytes before analysis.
