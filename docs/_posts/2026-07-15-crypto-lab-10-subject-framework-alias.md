---
title: "Crypto Lab, Part 10: The Subject Framework, alias & the Mercury Reveal"
description: Crypto Lab's pluggable subject framework and its alias tool recover length-seeded, keyless byte obfuscators through four incremental, resumable modes — gauge, structure, cells, and fromseed — and finally reveal that the Mercury signal was never encryption at all, graded BROKEN, with the moral that obfuscation is not encryption.
category: tutorials
keywords: subject framework, alias recovery, byte obfuscator, length seeded, talker alias, affine gauge, berlekamp massey, z3 structural search, keyless obfuscation, gophertrunk cryptolab, mercury
tags: [cryptolab, alias, subject-framework, obfuscation, security-testing, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 10
---

*Part 10 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. The finale: a framework for keyless obfuscators, and the answer to the mystery that ran through the whole Lab Bench trilogy.*

> **TL;DR:** Crypto Lab's pluggable **subject framework** studies length-seeded, keyless byte obfuscators (per-char decode `char = int8(Modd·(LUT[eo] − Hodd))`). The `alias` tool recovers one through four incremental, logging, resumable modes over a ground-truth corpus: `gauge` (brute all 32,768 affine gauges), `structure` (enumerate merged-index wirings; writes `high-transitions.csv` for the optional Z3 script), `cells` (intersect per-context state candidates — monotone, resumable via `-resume`), and `fromseed` (simulate from the length seed and auto-solve the gauge). **The Mercury reveal:** it was never strong encryption — it was a keyless, length-seeded byte obfuscator, recovered by `alias` and graded <span class="lab-verdict lab-verdict--bad">BROKEN</span>. Obfuscation is not encryption.

> **Authorized testing only.** The corpus and captures here are your own or licensed research material.

**Key takeaways**

- **The subject framework is pluggable.** Each "subject" is a specific byte obfuscator with its own recovery modes, registered like any tool.
- **`alias` recovers a keyless obfuscator** — no key exists; the structure *is* the secret, and structure can be reconstructed.
- **The modes are incremental and resumable.** Coverage only grows as you feed more captures — even ciphertext-only ones.
- **Mercury was obfuscation, not encryption** — the moral that ties the trilogy together.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab alias gauge -csv corpus.csv` | Brute all 32,768 affine gauges |
| `cryptolab alias structure -csv corpus.csv` | Enumerate merged-index wirings; export transitions |
| `cryptolab -out ./out alias cells -csv corpus.csv` | Intersect per-context state candidates (writes checkpoint) |
| `cryptolab -resume ./out/checkpoint.json alias cells -csv more.csv` | Resume and grow coverage |
| `cryptolab alias fromseed -csv corpus.csv` | Simulate from the length seed, auto-gauge |
| `-csv corpus.csv` | Ground-truth corpus (`rid,talkgroup,encoded_hex,alias`) |

## In this post

- **What a "subject" is** — the pluggable framework for byte obfuscators.
- **The four `alias` modes** — gauge, structure, cells, fromseed.
- **What the data supports** — why the effort is directed, logged, and resumable.
- **The optional Z3 search** — richer structural forms outside the binary.
- **The Mercury reveal** — and the moral that closes the trilogy.

## What a "subject" is

Everything up to now attacked *ciphers* — things with keys and keystreams. But a whole class of RF obfuscation has no key at all. Instead, the sender runs the data through a fixed, keyless transform: a lookup table and a per-character state update seeded only by the message length. There's nothing to brute-force because there's nothing secret except the *structure* of the transform itself. Recover the structure and the obfuscation is gone, permanently, for everyone.

The **subject framework** is Crypto Lab's pluggable home for studying these. Each subject is a specific byte-oriented obfuscator, registered like any other tool (the toolkit ships subjects under `internal/cryptolab/subjects/`). The flagship is the Motorola-style talker-alias obfuscator, exposed as the `alias` tool. Its decode model is *established* — the output substitution table and the per-character decode `char = int8(Modd·(LUT[eo] − Hodd))` are known — but the per-character *state update* is not, and recovering it is the research problem the four modes chip at.

Unlike every other tool, `alias` takes a **`-csv` corpus**, not an `-in` payload: rows of `rid,talkgroup,encoded_hex,alias` (a trailing 2-byte CRC on the encoded field is stripped automatically). This is a ground-truth dataset — encoded values paired with their decoded aliases — because recovering a keyless obfuscator is a fitting problem against known input/output pairs.

## The four modes

<figure class="lab-figure">
<svg viewBox="0 0 640 150" width="640" height="150" role="img" aria-label="Alias recovery pipeline: gauge, structure, cells, fromseed feeding a recovered obfuscator">
  <g font-size="11.5">
  <rect x="8" y="20" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="41" text-anchor="middle" fill="currentColor">gauge</text>
  <rect x="8" y="62" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="83" text-anchor="middle" fill="currentColor">structure</text>
  <rect x="8" y="104" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="125" text-anchor="middle" fill="currentColor">cells (resumable)</text>
  <rect x="160" y="62" width="120" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="220" y="83" text-anchor="middle" fill="currentColor">fromseed</text>
  <rect x="330" y="55" width="150" height="48" rx="8" fill="none" stroke="var(--accent)"/>
  <text x="405" y="76" text-anchor="middle" fill="var(--accent)">recovered</text>
  <text x="405" y="92" text-anchor="middle" fill="var(--accent)">obfuscator</text>
  <rect x="512" y="55" width="120" height="48" rx="8" fill="none" stroke="currentColor"/>
  <text x="572" y="76" text-anchor="middle" fill="currentColor">verdict</text>
  <text x="572" y="92" text-anchor="middle" fill="var(--fg-muted)" font-size="10">BROKEN</text>
  <g stroke="currentColor" stroke-width="1.2" fill="currentColor">
  <line x1="128" y1="37" x2="328" y2="70"/><polygon points="330,71 321,68 324,76"/>
  <line x1="128" y1="79" x2="158" y2="79"/><polygon points="160,79 152,75 152,83"/>
  <line x1="280" y1="79" x2="328" y2="79"/><polygon points="330,79 322,75 322,83"/>
  <line x1="128" y1="121" x2="328" y2="90"/><polygon points="330,89 321,90 324,98"/>
  <line x1="480" y1="79" x2="510" y2="79"/><polygon points="512,79 504,75 504,83"/>
  </g>
  </g>
</svg>
<figcaption>The four incremental modes attack the state update from different angles; together they reconstruct the keyless obfuscator, which grades BROKEN.</figcaption>
</figure>

**`gauge`** brute-forces all **32,768 affine gauges** — coordinate frames of the form `Modd·x − Modd·Hodd` — looking for one in which the odd high byte becomes a clean function of a simple merged index:

```bash
gophertrunk cryptolab alias gauge -csv alias_ground_truth.csv
```

```text
alias/gauge — swept 32768 gauges; best conflict floor 6/214 contexts
  best_mg 173  best_cg 44  clean_gauges 0
note: no gauge makes the odd high byte a clean function of the XOR merged
  index — consistent with the hidden internal table; the high byte's true
  dependency is on the unobserved low byte.
```

**`structure`** enumerates merged-index table wirings over the dense, plaintext-free high-byte recurrence `H[k+1] = F(H[k-1], H[k], eo[k])` and reports each wiring's conflict floor. With a global `-out` directory it also writes `high-transitions.csv` — the input for the optional Z3 search:

```bash
gophertrunk cryptolab -out ./out alias structure -csv alias_ground_truth.csv
```

**`cells`** intersects per-context state candidates across the corpus. It is **monotone and resumable**: coverage only grows as you feed it more captures — even ciphertext-only ones, which still feed the high-byte recurrence. You resume with `-resume` and point it at more data:

```bash
gophertrunk cryptolab -out ./out alias cells -csv batch1.csv
gophertrunk cryptolab -resume ./out/checkpoint.json alias cells -csv batch2.csv
```

```text
alias/cells — pinned 141/214 contexts; 1876/3204 chars decodable; 12 messages fully decoded
note: monotone & resumable: re-run with -resume and more captures (even
  ciphertext-only) and coverage only grows.
```

**`fromseed`** simulates candidate accumulator updates from the length seed, auto-solves the affine output gauge, and culls families by high-byte mismatch:

```bash
gophertrunk cryptolab alias fromseed -csv alias_ground_truth.csv
```

Every mode logs its progress, writes survivor artifacts to the `-out` directory, and ends its report with exactly what additional data would close the remaining gap — the effort is always directed, logged, and resumable, never a blind brute.

## What the data supports

The honest framing matters here, because this is live research, not a solved cipher. The two state bytes form 256×256 = 65,536-cell functions; a passive corpus typically exercises only a few hundred cells of each. But the geometry is favorable in a specific way: the **high byte is readable from ciphertext alone**, so `structure` and `fromseed` keep chipping at it from passive captures with no transmitter. The **multiplier byte** is only touched by labeled rows, so `cells` improves *monotonically* as more labeled data arrives. That's the whole design philosophy of the framework — a passive analyst with a growing capture set makes steady, resumable progress toward full recovery, and the tool tells you which kind of data (more ciphertext vs. more labeled pairs) moves which frontier.

## The optional Z3 search

The in-binary propagator explores single-lookup merged-index wirings. Richer forms — multi-round updates, two internal tables, small Feistel families — live in an **optional** Python/Z3 search under `internal/cryptolab/smt/`, outside the Go build. It consumes only the `high-transitions.csv` that `alias structure` exports:

```bash
gophertrunk cryptolab -out ./out alias structure -csv ground_truth.csv
python solve_structure.py --transitions ./out/high-transitions.csv
```

It's not required to use the toolkit, and it ships no cipher — it's a research aid that explores structural hypotheses as a constraint system. The clean-room findings from this line of work are written up in [`research/p25-talker-alias-cryptanalysis.md`](https://github.com/MattCheramie/GopherTrunk/blob/main/research/p25-talker-alias-cryptanalysis.md), which documents what's been established and — just as usefully — the dead ends already ruled out, so the next investigation starts ahead.

## The Mercury reveal

And so we come back to Mercury. Ada first caught it near 453 MHz in [Signal Lab Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }}) — an intermittent burst that blind signal-ID named a candidate (~4800 sym/s, 4-level FSK-like) but couldn't lock. [RF Scope Part 7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }}) flagged it unknown-protocol, triaged its payload as *not obviously strong*, and exported frames. In [Part 5]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }}), `ks reuse` found no keystream collisions — because, Reese suspected, there was no keystream at all.

He was right. Mercury was never encryption. Running the frames through `classify auto` and then the subject framework, `alias` recovered it as a **length-seeded, keyless byte obfuscator** — a talker-alias-style transform with a fixed lookup table and a length-seeded state update, no key anywhere. Ada decoded the aliases end-to-end, and `assess` graded it <span class="lab-verdict lab-verdict--bad">BROKEN</span>. Not because a strong cipher fell, but because there was never a cipher to begin with. The "encryption" that had eluded three tools across three series was obfuscation wearing a costume.

That's the moral, and it's the whole point of Crypto Lab: **obfuscation is not encryption.** A transform with no key protects nothing once its structure is known — and structure, unlike a good key over a rotated IV, can always be recovered. The verdict scale from Part 1 exists precisely to tell these apart: real encryption with a good key and rotated IVs grades <span class="lab-verdict lab-verdict--ok">RESISTANT</span> and stays that way; obfuscation grades <span class="lab-verdict lab-verdict--bad">BROKEN</span> the moment someone reconstructs it. Mercury looked like the former and was the latter. That gap — between looking secure and being secure — is exactly what a security test is for.

## Where this goes next

This closes the Crypto Lab series and the Lab Bench trilogy. If you want to retrace Mercury from the start, [Signal Lab Part 8]({{ '/blog/tutorials/signal-lab-08-naming-the-unknown/' | relative_url }}) is where it was named and [RF Scope Part 7]({{ '/blog/tutorials/rf-scope-07-entropy-encryption-triage/' | relative_url }}) is where it was triaged and its frames exported. From here, the natural next step is to point the toolkit at your *own* systems: build with `TAGS=cryptolab`, run `assess crypto` against a capture you're authorized to test, and see whether your encryption grades RESISTANT — or reveals a reused IV, a default key, or an obfuscator hiding where a cipher should be. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) are the full reference.

## FAQ

**What makes `alias` different from the cipher tools?**
It recovers a *keyless* obfuscator — there's no key to find, only the structure of the transform. It fits that structure against a ground-truth corpus (`-csv`), and once recovered it decodes every message, for good.

**Why does `alias` take a `-csv` corpus instead of an `-in` file?**
Because recovering a keyless obfuscator is a fitting problem against known input/output pairs. The corpus is `rid,talkgroup,encoded_hex,alias`; the trailing CRC is stripped automatically.

**Are the `alias` modes a finished attack?**
They're an active, incremental research framework. Each mode is monotone/resumable and reports what data would close the remaining gap. `cells` in particular only improves as you add captures — including ciphertext-only ones for the high-byte recurrence.

**What was Mercury, in the end?**
A keyless, length-seeded byte obfuscator (talker-alias style) — not strong encryption at all. `alias` recovered it and `assess` graded it BROKEN. The lesson: obfuscation is not encryption, and a security test is what tells the difference.

## Series navigation

**Part 10 of 10** · ←[Part 9: Web Console, Live Bridge & External Ciphers]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }}) · Back to the [Crypto Lab series hub]({{ '/blog/series/crypto-lab/' | relative_url }})
