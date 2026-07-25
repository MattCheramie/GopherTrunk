# Chosen-plaintext capture for the Motorola P25 talker-alias cipher (#773)

This is a procedure for someone **with access to a Motorola P25 radio** to help
finish reverse-engineering the talker-alias obfuscation cipher. Instead of
guessing the algorithm from random captured callsigns, you choose the plaintext
(the alias) and capture the resulting ciphertext, exposing the cipher's internal
state machine one variable at a time.

Why this is high-value now: we've recovered the cipher's substitution table and
confirmed it's a length-seeded **nonlinear byte accumulator** producing one
affine byte transform per character. The one thing black-box analysis of random
real callsigns can't pin is the **hidden low byte of the accumulator** that
drives the nonlinear state update. Controlled inputs — length sweeps and
single-character/single-bit differences — drive that hidden state deliberately
and should expose it.

> See `p25-talker-alias-cryptanalysis.md` for the full clean-room findings. Key
> update: the cipher's state is an **observable memory-2/3 byte chain** (the
> accumulator high byte is readable from each even ciphertext byte), so the
> per-character transition is *nearly determined by two prior high bytes*. That
> sharpens this procedure — a short length sweep plus single-character
> differentials at a couple of positions is enough to pin the memory-2/3
> transition directly. Tier 1 plus a partial Tier 2 should suffice.

---

## ⚠️ Authorization first
Only transmit on spectrum/systems you are **licensed or explicitly authorized**
to key up on: your own licensed system, a Motorola P25 system you control, or a
**bench setup** (radio → attenuator/dummy load or a service monitor, isolated
from live infrastructure). **Do not** transmit on public-safety or any system
you don't have permission to use. This is interoperability/research on your own
equipment.

## What you need
- A Motorola P25 subscriber radio whose **talker alias / radio display alias**
  you can set (via CPS / Radio Management), on a system with **OTA aliasing
  enabled**.
- A receiver running **SDRTrunk** (or GopherTrunk) capturing that system, with
  talker-alias message logging on — the **same setup that produced the existing
  3,607-pair dataset**. For each transmission we need the **reassembled
  encoded-alias bytes (the ciphertext hex)** paired with the alias string you
  programmed.
  - **GopherTrunk emits this natively.** When a Motorola FACCH-S alias
    reassembles on a followed Phase 2 traffic channel, GopherTrunk logs a
    `p25p2 alias ciphertext` line with `rid`, `talkgroup`, and `encoded_hex`
    — the ciphertext hex this procedure needs. Grep the daemon log for
    `alias ciphertext` while capturing; each keyup produces one line, and the
    `encoded_hex` must be byte-identical across the 2–3 keyups (the sanity
    check below). No SDRTrunk required.

## Core procedure (repeat per alias)
1. In CPS, set the radio's talker alias to the next string from the list below.
   Write to radio.
2. Key up briefly so the alias is sent OTA. **Key up 2–3 times.**
3. In the receiver, capture the **encoded-alias hex** for that transmission.
4. Record a row: `alias, encoded_hex` (add `rid, talkgroup` too if easy —
   matching the existing CSV columns `rid,talkgroup,encoded_hex,alias` is ideal).
5. **Sanity check:** the ciphertext must be **byte-for-byte identical** across
   the 2–3 keyups (the cipher is deterministic). If two captures differ, a bit
   error slipped through — discard and re-capture. Keep only aliases whose
   ciphertext is stable and whose CRC validates.

## The plaintext list — what to program, in priority order

Use the literal character `A` as the "fill" unless told otherwise.
**Tier 1 alone may be enough; do it first.**

### Tier 1 — Length sweep, constant fill (pins the seed and the state evolution)
Reprogram and send each of these:
```
A
AA
AAA
AAAA
AAAAA
AAAAAA
AAAAAAA
AAAAAAAA
AAAAAAAAA
AAAAAAAAAA
AAAAAAAAAAA
AAAAAAAAAAAA
AAAAAAAAAAAAA
AAAAAAAAAAAAAA
```
Then repeat the same length sweep with a **different fill character**, `B`, and a
third with `0` (zero). Three sweeps × 14 lengths. Different fill values at each
length let us separate "position in the message" from "the byte value being
processed" — the single most valuable comparison.

### Tier 2 — First-character sweep at one fixed length (pins the position-1 substitution)
Fix length 6, keep the last 5 characters as `AAAAA`, sweep the first character:
```
AAAAAA  BAAAAA  CAAAAA  DAAAAA  EAAAAA  FAAAAA  GAAAAA  HAAAAA
IAAAAA  JAAAAA  KAAAAA  LAAAAA  MAAAAA  NAAAAA  OAAAAA  PAAAAA
0AAAAA  1AAAAA  2AAAAA  3AAAAA  4AAAAA  5AAAAA  6AAAAA  7AAAAA
8AAAAA  9AAAAA  (space)AAAAA   -AAAAA
```
More coverage of the printable range is better, but this subset is a good start.

### Tier 3 — Second/third-character sweeps + differentials (pins the per-step transition)
- Fix length 6, char 1 = `A`, sweep **char 2**: `ABAAAA, ACAAAA, ADAAAA, …`.
- Fix length 8, sweep **char 4**: `AAABAAAA, AAACAAAA, …`.
- **Single-bit-difference pairs** at the same position — send each pair:
  `A`/`C` (0x41/0x43, 1 bit), `A`/`E` (0x41/0x45), `A`/`a` (0x41/0x61),
  `@`/`A` (0x40/0x41). e.g. `AAAAAA`/`ACAAAA`, `AAAAAA`/`AaAAAA`, etc.

If you have the patience, **repeat the Tier-2 sweep at a second length**
(e.g. 10) — the cipher is length-seeded, so a second length confirms the model
generalizes.

## What to send back
A CSV (or plain list) of `alias,encoded_hex` rows — ideally in the same column
order as the existing dataset (`rid,talkgroup,encoded_hex,alias`). It feeds
straight into the solver.

## Why this works
- The **length sweep** isolates how the cipher is seeded from message length and
  how its internal state advances when the input is held constant — the
  recurrence that random callsigns hide.
- The **single-character / single-bit sweeps** hold the state fixed and vary one
  input, exposing the per-position transform and the nonlinear state update one
  variable at a time.
- Because you *know* the plaintext (you programmed it), no decoder is required —
  we only need the raw ciphertext.

Even **Tier 1 plus one Tier-2 sweep (~50 transmissions)** would likely let us pin
the seed and the per-step state machine and verify it against both the new
captures and the existing 3,607 pairs.

## Practical notes
- Reprogramming the codeplug per alias is the slow part (a few minutes each) —
  Tier 1 first; it's the highest value-per-transmission.
- If your CPS or radio supports scripting / bulk alias changes, that turns ~50
  manual writes into a batch job.
