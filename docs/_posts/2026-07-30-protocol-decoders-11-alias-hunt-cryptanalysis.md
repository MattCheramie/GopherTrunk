---
title: "Protocol Decoders, Part 11: The Alias Hunt II — The Keystream, Dead Ends & Ethics"
description: The clean-room cryptanalysis of the Motorola P25 talker-alias cipher — the per-character affine keystream, every ruled-out hypothesis, the low-byte observability floor that keeps it uncracked, and the ethics that gate it all.
category: deep-dives
keywords: motorola alias cryptanalysis, p25 talker alias cipher, chosen plaintext attack, observability floor, clean room reverse engineering, nonlinear state machine, cipher verified false, gophertrunk crypto lab mercury
tags: [cryptanalysis, p25, motorola, clean-room, ethics, cipher]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Protocol Decoders"
series_part: 11
---

*Part 11 of **Protocol Decoders**, and the second half of the capstone the series
has built toward since Part 1. Part 10 framed the Motorola talker-alias message
and recovered its substitution table. This post attacks the cipher underneath —
and the honest answer, stated plainly, is that it is **not cracked.** This is the
same emitter the [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }})
series calls **Mercury**; here we meet it from the decoder's side. Everything below
is clean-room, data-driven, and authorized-use-only (issue #773).*

> **TL;DR:** The alias cipher is a **length-seeded nonlinear byte accumulator** that
> emits one affine transform per character: `char = M·(LUT[value] − H)`. Its state
> is *observable* — the accumulator high byte reads straight off even ciphertext
> bytes — which reduces it to a functional 16-bit state machine. But every closed
> form (LCG, MWC, GF(2⁸), Feistel, table networks) is **ruled out**, a lookup table
> **doesn't generalize**, and chosen-plaintext hits an **observability floor**: the
> hidden low byte drives the update and is never observed. `CipherVerified` stays
> **false**.

**Key takeaways**

- The per-character keystream factors into a clean product form
  `char = M·(LUT[value] − H)`, and the state is a functional 16-bit machine.
- A long list of standard generators (LCG, mod-prime, MWC, xorshift, GF(2⁸),
  Feistel, single/multi-table networks) is **exhausted** — all fit at random.
- A trained lookup table decodes only ~7% of held-out messages; passive coverage
  grows **linearly** and never closes.
- Chosen-plaintext pins the seed and three exact states but hits an
  **observability floor** — zero observed low-byte transitions. Not cracked.

## Cheat sheet

| Result | Status |
|---|---|
| Framing + SUID + CRC | verified (Part 10) |
| 256-entry output `LUT` | fully recovered |
| Decode model `char = M·(LUT[value] − H)` | verified |
| 16-bit state machine `(H,M,eo) → (H′,M′)` | functional, 0 conflicts / 423 |
| Closed form for the update | **none found** (all ruled out) |
| Low-byte update `Lnext` | **unobservable** — the floor |
| `CipherVerified` | **false** |

## In this post

- **The keystream** — the per-character affine model and the observable state.
- **The dead ends** — the structural classes that were ruled out.
- **Why a table won't ship** — coverage that never closes.
- **The observability floor** — the low byte you can't see.
- **The chosen-plaintext procedure** and the ethics that gate all of it.

## The keystream: an observable 16-bit machine

Part 10 left us with `decoded[i] = int8(M·LUT[enc[i]] + c)` and one gift: because
aliases are UTF-16BE, even positions decode to zero, so the accumulator high byte is
readable straight off the ciphertext — `H_k = LUT[enc[2k]]`. That gives **29,601
direct state observations** across the corpus with no decoding required, and it lets
the additive term be rewritten so the decode factors into a product form:

```
char = M · ( LUT[value] − H )       // value = the ciphertext/encode byte; H = high byte
```

Split that per-character keystream into two observables — the multiplier `M_odd` and
the odd-position high byte `Hbyte_odd` — and both turn out to be clean functions of
the two preceding even high bytes, *with no ciphertext input*. The recurrence has
short memory. In fact, defining the per-character state `s[k] = (H_odd[k], M_odd[k])`
gives a **clean 16-bit Markov state machine**:

```
(H_odd, M_odd, eo)  →  (H_odd′, M_odd′)     exactly functional: 0 conflicts / 423 keys
```

one input byte per character, deterministic. This is the tightest characterization
of the cipher to date. It reframes the whole problem: it is *not* an opaque 16-bit
black box to brute — it is a **functional, observable, nonlinear byte recurrence**,
and the only thing missing is a closed form for the update.

<figure class="lab-figure">
<svg viewBox="0 0 680 172" width="680" height="172" role="img" aria-label="The alias cipher as a functional 16-bit state machine: a length seed feeds a per-character state of high byte and multiplier, updated by the odd ciphertext byte, with the accumulator high byte observable from even ciphertext bytes but the low byte hidden">
  <rect x="14" y="66" width="96" height="44" rx="6" fill="none" stroke="var(--fg-muted)"/>
  <text x="62" y="84" text-anchor="middle" fill="var(--fg-muted)" font-size="11">seed(n)</text>
  <text x="62" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="10">length-keyed</text>
  <line x1="110" y1="88" x2="146" y2="88" stroke="currentColor"/>
  <polygon points="146,84 156,88 146,92" fill="currentColor"/>
  <rect x="156" y="52" width="150" height="72" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="231" y="76" text-anchor="middle" fill="var(--accent)" font-size="12">state s[k]</text>
  <text x="231" y="94" text-anchor="middle" fill="currentColor" font-size="10">H (observable)</text>
  <text x="231" y="110" text-anchor="middle" fill="#c0392b" font-size="10">L (hidden)</text>
  <line x1="306" y1="88" x2="360" y2="88" stroke="currentColor"/>
  <polygon points="360,84 370,88 360,92" fill="currentColor"/>
  <text x="333" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="10">eo[k]</text>
  <rect x="370" y="60" width="130" height="56" rx="6" fill="none" stroke="currentColor"/>
  <text x="435" y="82" text-anchor="middle" fill="currentColor" font-size="11">update U</text>
  <text x="435" y="98" text-anchor="middle" fill="var(--fg-muted)" font-size="10">nonlinear</text>
  <path d="M 435 116 L 435 140 L 231 140 L 231 124" fill="none" stroke="var(--accent)"/>
  <polygon points="227,128 231,118 235,128" fill="var(--accent)"/>
  <text x="333" y="154" text-anchor="middle" fill="var(--accent)" font-size="10">s[k+1] — functional: 0 conflicts / 423</text>
  <line x1="500" y1="88" x2="560" y2="88" stroke="currentColor"/>
  <polygon points="560,84 570,88 560,92" fill="currentColor"/>
  <rect x="570" y="66" width="96" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="618" y="84" text-anchor="middle" fill="currentColor" font-size="11">char</text>
  <text x="618" y="99" text-anchor="middle" fill="var(--fg-muted)" font-size="9">M·(LUT[v]−H)</text>
</svg>
<figcaption>The cipher reduces to a functional 16-bit state machine — high byte observable, low byte hidden — updated by the odd ciphertext byte. The machine is deterministic; only the closed form of the update is missing.</figcaption>
</figure>

## The dead ends

Knowing the machine is functional doesn't tell you its update rule. An enormous
amount of the work was spent *ruling out* the standard generators, each tested by
simulating the recurrence from the length seed and checking the observable high byte
against the corpus through an auto-solved output gauge (and, where possible, by
direct algebraic fit). All negative:

- **Linear LCG** `s′ = A·s + B (±input)` — brute over all `A,B` in both operand
  orders and both seed models, *and* a direct 2-step algebraic fit. Both ciphertext
  and plaintext feedback.
- **Multiplicative recurrence mod a prime** — 15 moduli (65537, 65521, …, 32719),
  both extraction modes.
- **Multiply-with-carry (MWC)** — 6 variants (tested precisely *because* the
  memory-2/3 coupling is the MWC signature; still negative).
- **Xorshift, rotate-mix, xor-then-multiply, multiply-then-xor** full-state forms.
- **Low-degree polynomials** over Z/2¹⁶ and per-byte affine/quadratic over Z/256.
- **NLFSR / single S-box of a linear tap** — a single byte-projection can't
  determine the target; adding taps only adds conflicts.
- **GF(2⁸)-affine** across 8 field polynomials, **Feistel** (including a gauge-aware
  variant), and **single/two-table lookup networks** — solved directly with Z3 on
  the *dense* even-H recurrence (~10,076 over-determined keys, no overfit possible).

The dense solve even produced a precise structural *reason* the table networks fail:
within any merged `Hp ⊕ eo` group the same `H` maps to different `H[k+1]`, so the
next high byte **is not a function of any merged 2-byte index** — which rules out
every table-network that ends in a single lookup of a combined index. The best
closed-form fit anywhere is ≈ 1,267 / 1,294, i.e. random. The update is a genuine
multi-input nonlinear function — note the verified decode itself contains a
state×input product `M·LUT[value]`, which no "S-box-of-linear" form can even express.

## Why a table won't ship

The obvious shortcut — skip the algorithm, just learn the empirical map
`G(H[k−1], H[k], eo[k]) → char` from the corpus — was tried and **fails to
generalize.** Train on 80% of messages, decode the held-out 20%:

| Metric | Result |
|---|---|
| Held-out messages fully decoded | **7.2%** |
| Characters correct | **70%** |
| Characters hitting an unseen `(H,H,eo)` context | **30%** |

And it gets worse the more you look: on the denser differential corpus, distinct
transition keys grow **~linearly** with corpus size (≈250 → 605 → 896 → 1,179 keys at
25/50/75/100%) instead of plateauing. The reachable state space is large; a passive
corpus, even a dense one, can't reach full coverage by intersection alone. A lookup
table would garble live aliases at unseen contexts — so it is **not shippable**, and
only a closed form yields a general decoder. This is why `CipherVerified` can't be
flipped on a table: a partial table produces *confident wrong names*.

## The observability floor

The chosen-plaintext data (a length sweep and single-character sweeps, from a
third-party working encoder consumed as I/O logs only — no source read) sharpened the
target dramatically. It **confirmed the memory-2 machine** on new data (0 conflicts /
1,408), **pinned the length seed** as a function of length, and **pinned three exact
states** by differential state-fixing. It even recovered most of the high-byte update
law:

```
Hnext@state(value) = H_state + g(value) + s·139
```

where `g` is a state-independent function covered for **197 / 256** values, and `s`
is a two-way branch (~40% of values) governed by the state's **low byte**. That last
clause is the wall. A decoder also needs the *low-byte* update
`Lnext = LowUpdate(H, L, value)` to propagate the accumulator — and the chosen
plaintext exposes `L` only as `L|1` at **three non-consecutive** positions. The number
of observed `(L_in → L_out)` transitions is therefore **zero**, and bit 0 of `L` is
never observable at all (the decode multiplier forces `L|1`).

This is an **observability floor, not an effort limit.** No amount of cryptanalysis
recovers an update with zero observed transitions; the branch selector `s(L, value)`
is only ~80% predictable, the residual being exactly that hidden low bit. Two
independent fingerprints confirm there's a real hidden variable: the state map is
**non-bijective** (lengths 7 and 9 share a high-byte trajectory — an LCG/MWC/odd
multiply *cannot* merge trajectories), and a single global 16-bit multiply is ruled
out because two pinned states differ by 139, not a carry of 1. The signature points
to a **second internal substitution table** distinct from the recovered output LUT.

<figure class="lab-figure">
<svg viewBox="0 0 680 168" width="680" height="168" role="img" aria-label="The observability floor: the high byte update law is recovered with g covered for 197 of 256 values, but the low byte update has zero observed transitions because chosen plaintext only exposes the low byte at three non-consecutive positions, leaving the branch selector unresolved">
  <rect x="20" y="26" width="300" height="52" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="170" y="48" text-anchor="middle" fill="var(--accent)" font-size="12">HIGH byte — recovered</text>
  <text x="170" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="10">Hnext = H + g(value) + s·139 · g covers 197/256</text>
  <rect x="360" y="26" width="300" height="52" rx="6" fill="none" stroke="#c0392b"/>
  <text x="510" y="48" text-anchor="middle" fill="#c0392b" font-size="12">LOW byte — unobservable</text>
  <text x="510" y="66" text-anchor="middle" fill="var(--fg-muted)" font-size="10">0 observed (L_in → L_out) · bit 0 never seen</text>
  <line x1="340" y1="94" x2="340" y2="150" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <text x="340" y="110" text-anchor="middle" fill="var(--fg-muted)" font-size="10">the floor</text>
  <rect x="60" y="104" width="220" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="170" y="122" text-anchor="middle" fill="currentColor" font-size="10">3 non-consecutive swept positions</text>
  <text x="170" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="10">exposes L only as L|1</text>
  <rect x="400" y="104" width="220" height="40" rx="5" fill="none" stroke="currentColor"/>
  <text x="510" y="122" text-anchor="middle" fill="currentColor" font-size="10">branch selector s(L, value)</text>
  <text x="510" y="136" text-anchor="middle" fill="var(--fg-muted)" font-size="10">~80% predictable — residual = hidden bit</text>
</svg>
<figcaption>The high-byte update is well-determined; the low-byte update has zero observed transitions. The one variable that decides the branch is exactly the one no passive or three-position capture reveals.</figcaption>
</figure>

## The chosen-plaintext procedure — and the ethics

The decisive missing input is precise: single-character sweeps at **consecutive**
positions (so `L_in → L_out` becomes observable) across more positions and a second
length. Because you *know* the plaintext you programmed, no decoder is needed — only
the raw ciphertext. The procedure is documented for anyone with a programmable
Motorola radio: a **Tier 1** length sweep with a constant fill (pins the seed), a
**Tier 2** first-character sweep at fixed length (pins the position-1 transform), and
**Tier 3** differentials (pin the per-step transition).

That capability comes with a hard boundary, stated first in the procedure and
repeated here:

> **Authorization first.** Only transmit on spectrum and systems you are licensed or
> explicitly authorized to key up on — your own licensed system, a Motorola system
> you control, or a **bench setup** (radio → attenuator/dummy load or a service
> monitor, isolated from live infrastructure). Do not transmit on public-safety or
> any system you don't have permission to use.

The cryptanalysis itself is **clean-room** and stays that way on purpose:

- **Data-driven only.** Every result is fitted to and validated against decoded-log
  *output* — captured ciphertext paired with known plaintext. The one chosen-plaintext
  tranche (`moto_alias_t.zip`) was consumed as **I/O logs, no source read**.
- **No GPL source consulted.** SDRTrunk has a working implementation, but it is
  GPLv3 and GopherTrunk is **Apache-2.0**, so its table and decode are intentionally
  *not* read or ported. GopherTrunk's decode must be an independent, permissively
  licensed derivation — the algorithm is treated as a fact about Motorola's wire
  protocol, discovered from data.
- **Gated until verified.** `CipherVerified = false`, and it flips only with a
  committed regression fixture mapping real bytes to correct plaintext — never on
  inference. The live decoder ships an algebraic placeholder that the analysis above
  has itself *ruled out*, so it can't accidentally present a fabricated name.

The recovery toolkit lives under `internal/cryptolab/subjects/motorola` (build tag
`cryptolab`) with modes `gauge`, `structure`, `cells`, `fromseed`, `propagate`, and
`sweep` — every technique above is reproducible from a corpus CSV. This is the same
workbench the [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) series
drives against Mercury; the two series meet exactly here.

### The honest conclusion

The cipher is **not cracked.** What's established: the framing, the CRC boundary, the
output LUT, the product-form decode, and a functional 16-bit state machine with the
high-byte update law recovered over most of its domain. What's missing: the low-byte
update, blocked by an observability floor that only a denser, consecutive-position
chosen-plaintext capture can lift. Calling this "solved" would violate the same rule
that keeps `CipherVerified` false — and that rule is the point.

## Where this goes next

[Part 12]({{ '/blog/deep-dives/protocol-decoders-12-testing-decoders-without-radios/' | relative_url }})
closes the series by answering the question every post has quietly relied on: how do
you *test* a decoder — including a gated one — without a radio, a live system, or a
lucky capture? Synthesized control channels, a decoder registry, golden fixtures, and
regression discipline are what let a claim like "the framing is verified" actually
mean something.

## FAQ

**Is the Motorola talker-alias cipher cracked?**
No. The framing, CRC boundary, and output substitution table are recovered, and the
cipher reduces to a functional 16-bit state machine — but no closed form for its
update has been found, and the hidden low byte that drives it is under-observed in
every capture obtained so far. `CipherVerified` remains false.

**What is the observability floor?**
It's the point where more analysis can't help because the needed data was never
observed. The cipher's low-byte update has *zero* observed input→output transitions
in the available chosen-plaintext (which only exposes the low byte at three
non-consecutive positions), so the branch it controls can't be resolved. The fix is a
denser capture, not a cleverer solver.

**Why not just use SDRTrunk's implementation?**
Licensing. SDRTrunk is GPLv3; GopherTrunk is Apache-2.0. Porting the GPL table or
decode would contaminate the license, so the algorithm is derived independently from
captured data only — a true clean-room reconstruction — and gated until a real frame
verifies it.

**Is doing chosen-plaintext captures legal?**
Only on systems and spectrum you're authorized to transmit on — your own licensed
system, equipment you control, or an RF-isolated bench setup with a dummy load or
service monitor. The procedure states this first; transmitting on public-safety or
any unauthorized system is out of bounds.

## Series navigation

**Part 11 of 12** · ←
[Part 10: The Alias Hunt I]({{ '/blog/deep-dives/protocol-decoders-10-alias-hunt-framing/' | relative_url }})
· Next →
[Part 12: Testing Decoders Without Radios]({{ '/blog/deep-dives/protocol-decoders-12-testing-decoders-without-radios/' | relative_url }})
