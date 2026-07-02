---
title: "Crypto Lab, Part 3: Randomness — NIST SP 800-22 & the LFSR Signature"
description: Crypto Lab's randomness battery runs a NIST SP 800-22 subset over a keystream bitstream — monobit, block frequency, runs, longest run, binary-matrix rank, spectral DFT, approximate entropy, serial, cumulative sums, and linear complexity — to tell strong keyed encryption from an exploitable LFSR scrambler.
category: tutorials
keywords: nist sp 800-22, randomness testing, linear complexity, lfsr signature, keystream analysis, monobit test, spectral dft, berlekamp massey, rf encryption testing, gophertrunk cryptolab
tags: [cryptolab, randomness, lfsr, keystream, advanced, security-testing]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 3
charts: true
---

*Part 3 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. Once you have a keystream, one question decides everything: is it strong, or is it a scrambler in disguise?*

> **TL;DR:** `cryptolab randomness battery -in keystream.bin` runs a NIST SP 800-22 subset — monobit, block frequency, runs, longest run, binary-matrix rank, spectral DFT, approximate entropy, serial, cumulative sums, and **linear complexity** — over a bitstream of **packed bytes, MSB-first**. A stream that passes is consistent with strong keyed encryption (nothing to exploit); a **failing linear-complexity or spectral test is the signature of an LFSR-based scrambler**. `randomness quick` runs the fast subset for short captures.

> **Authorized testing only.** Run this against keystreams from your own or licensed systems.

**Key takeaways**

- **The decisive RF question** is whether a recovered keystream is statistically random (strong) or structured (a keyless scrambler / short LFSR).
- **Linear complexity is the tell.** A failing linear-complexity or spectral test is the fingerprint of an LFSR.
- **Input is packed bytes, MSB-first** — the same convention as `lfsr bm`.
- **`quick` for short captures, `battery` for the full picture** — the full battery needs enough bits to be meaningful.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab randomness battery -in keystream.bin` | Full NIST SP 800-22 subset (10 tests) |
| `cryptolab randomness quick -in keystream.bin` | Fast subset (monobit / block-freq / runs / cusum) |
| `-alpha 0.01` | Significance level (default 0.01) |
| `cryptolab lfsr bm -in keystream.bin` | Recover the shortest LFSR (Berlekamp–Massey) |
| `cryptolab ks extract -pt p.bin -ct c.bin -out ks.bin` | Get a keystream to feed the battery |

## In this post

- **Why randomness is the RF question** — the difference between "encrypted" and "unbreakable."
- **The ten tests** and what each one is sensitive to.
- **The LFSR signature** — how linear complexity and spectral DFT unmask a scrambler.
- **`quick` vs `battery`** — matching the test set to how many bits you have.
- **From battery to Berlekamp–Massey** — the natural handoff to `lfsr bm`.

## Why randomness is the whole question

When you've recovered a keystream — from a known-plaintext pair, or from an LDU2 you could decrypt — you hold the raw output of whatever generator the cipher used. And here's the thing Reese drills into every new analyst: **a strong keystream and a weak one can have the same entropy.** Entropy near 8 bits/byte just means the byte values are spread out. It does *not* mean the sequence is unpredictable. A short LFSR produces a high-entropy, long-period stream that looks random to the eye and to a chi-square test — yet a few hundred bits are enough to predict the rest of it forever.

So the security question is not "is this high-entropy?" but "is this *statistically random*?" — random in the sense that no test can distinguish it from coin flips. If it passes, you're looking at strong keyed encryption (AES/DES-class) or good compression, and there is nothing to brute-force. If it fails, the stream carries exploitable structure, and the battery tells you *which kind*.

## The ten tests

```bash
gophertrunk cryptolab randomness battery -in keystream.bin
```

```text
randomness/battery — 20480 bits: 9/10 tests passed, 1 failed, 0 skipped (alpha 0.010)
  bits          20480
  looks_random  false
findings (weakest first):
  linear_complexity   0.994   p_value 0.006  passed false
  spectral_dft        0.210   p_value 0.790  passed true
  monobit             0.010   p_value 0.990  passed true
  ...
note: one or more tests failed: the stream carries exploitable structure.
  A failing linear_complexity or spectral test points at an LFSR / keyless
  scrambler — try `cryptolab lfsr bm` and `cryptolab stats period`.
```

The full battery runs a ten-test subset of NIST SP 800-22, each sensitive to a different kind of non-randomness:

| Test | Detects |
|---|---|
| monobit | An imbalance of ones vs zeros overall |
| block frequency | Local bias within fixed-size blocks |
| runs | Too many / too few consecutive-bit runs |
| longest run | An improbably long run of ones in a block |
| binary-matrix rank | Linear dependence among bit sub-blocks |
| spectral DFT | Periodic patterns via a frequency-domain peak |
| approximate entropy | Repeating overlapping patterns |
| serial | Non-uniform frequency of overlapping m-bit patterns |
| cumulative sums | Drift in the running one/zero sum |
| **linear complexity** | Whether a short LFSR reproduces the sequence |

Each test yields a **p-value**; a test *passes* when its p-value clears the significance level `alpha` (default 0.01). The report's `looks_random` field is the roll-up: it's true only when nothing failed. Findings are sorted weakest-first, so the most non-random test floats to the top.

Here is a battery result as a chart — nine tests clear the 0.01 threshold, but linear complexity dives under it, coloring red:

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="600" height="320" role="img"
        aria-label="NIST SP 800-22 p-values with a 0.01 threshold; linear complexity fails"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["monobit","block-freq","runs","longest-run","matrix-rank","spectral","approx-ent","serial","cusum","lin-cplx"],
"values":[0.99,0.61,0.44,0.72,0.38,0.79,0.51,0.66,0.90,0.006],
"threshold":0.01,
"pass":[true,true,true,true,true,true,true,true,true,false],
"ylabel":"p-value (pass ≥ 0.01)" }
</script>
<figcaption>Nine tests pass comfortably; linear complexity's p-value of 0.006 falls below the 0.01 threshold and fails — the sequence is not random. A single red bar is enough to change the verdict.</figcaption>
</figure>

## The LFSR signature

The reason **linear complexity** gets bolded everywhere is that it is the test purpose-built to separate keyed encryption from a linear scrambler. Linear complexity is the length of the shortest LFSR that generates the sequence, computed by the Berlekamp–Massey algorithm. For a truly random stream of *n* bits, that length hovers around *n/2*. For an LFSR-based scrambler, it collapses to a small constant — the register length — no matter how many bits you feed it.

So the battery's linear-complexity test asks: *is the shortest LFSR suspiciously short?* When it is, the test fails, and you have your answer — this is not a strong cipher, it is a scrambler you can reconstruct and predict. The **spectral DFT** test is the companion signature: an LFSR's periodicity shows up as a peak in the frequency domain, which the DFT test flags. When either fails, the classifier and the battery both point you at the same next tool: `lfsr bm`, which we saw the toolkit recommend in [Part 2]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }}).

This is exactly the distinction that makes a high-entropy verdict *not* mean "give up." Ada learned it the hard way on a scrambled payload: entropy 7.98, "must be AES," she thought — until the battery failed linear complexity and `lfsr bm` recovered a 17-bit register that regenerated the whole keystream. Structured, not strong.

## Input format: packed bytes, MSB-first

One detail that trips people up: the battery reads its input as a **bitstream of packed bytes, MSB-first**. Each byte contributes 8 bits, most-significant first, in file order. This is the same convention `lfsr bm` uses, which is deliberate — a keystream you've extracted flows straight from one tool to the other without any re-packing. If you feed it text or a hex dump instead of raw packed bytes, the bits won't line up and the tests will be meaningless, so extract to raw bytes first (`ks extract -out ks.bin`, [Part 5]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }})).

## `quick` vs `battery`

The full battery is only meaningful with enough bits — several of the tests (binary-matrix rank, spectral DFT, linear complexity, approximate entropy) need a substantial sample to produce a reliable p-value, and with too few bits they get skipped and reported as such. For short captures, `randomness quick` runs the fast, low-data subset — monobit, block frequency, runs, and cumulative sums:

```bash
gophertrunk cryptolab randomness quick -in short-keystream.bin
```

The classifier itself uses this rule internally: it runs the full `Battery` when it has at least a couple thousand bits and falls back to `Quick` below that. As a human analyst you make the same call — `quick` for a first look at a short keystream, `battery` once you have enough material for the heavy tests to fire. The `-alpha` flag tunes the significance level on either; lowering it makes the tests more permissive (fewer false failures), raising it makes them stricter.

## From battery to Berlekamp–Massey

The battery *detects* structure; `lfsr bm` *recovers* it. When linear complexity fails, run:

```bash
gophertrunk cryptolab lfsr bm -in keystream.bin
```

```text
lfsr/bm — 20480 bits: linear complexity 17
  linear_complexity  17
  feedback_taps      [17 14 0]
note: short LFSR recovered: feed more bits to confirm the taps generalize
  beyond the observed sequence.
```

A linear complexity of 17 over 20,480 bits is a screaming result — the entire keystream is the output of a 17-stage register, and the reported feedback taps let you regenerate and predict it. Reese's caution, which the tool prints for you: confirm the taps generalize by feeding more bits. A short LFSR fit to too little data can be a coincidence; a short LFSR that holds across thousands of bits is a break. If you have a plaintext/ciphertext pair rather than a raw keystream, `lfsr keystream -pt p.bin -ct c.bin` XORs them and runs Berlekamp–Massey in one step.

## Where this goes next

Not every weak payload is an LFSR. Some are classical ciphers — repeating XOR, a Caesar shift, a Vigenère key, a substitution alphabet — and Crypto Lab has a dedicated `brute` tool with an embedded English language model to recover them. [Part 4]({{ '/blog/tutorials/crypto-lab-04-classical-ciphers-brute/' | relative_url }}) walks all four modes and when each applies. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) have the full test list if you want the reference.

## FAQ

**What input format does the battery expect?**
Raw packed bytes, MSB-first — the same convention as `lfsr bm`. Don't feed it hex text or a base64 string; extract the keystream to raw bytes first.

**My payload has entropy near 8 but the battery fails. What does that mean?**
High entropy with a failing test — especially linear complexity or spectral DFT — is the signature of an LFSR or keyless scrambler. It looks random byte-for-byte but is fully predictable from a short register. Run `lfsr bm` to recover it.

**When should I use `quick` instead of `battery`?**
When you have a short keystream. Several battery tests need a lot of bits to be reliable and will be skipped otherwise; `quick` runs only the tests that work on small samples.

**Does passing the battery prove the encryption is unbreakable?**
No — it proves the *keystream* is statistically indistinguishable from random at this alpha, which is consistent with strong keyed encryption. There may still be operational weaknesses (reused IVs, default keys); the battery only speaks to the keystream's structure.

## Series navigation

**Part 3 of 10** · ←[Part 2: First Triage]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }}) · Next →[Part 4: Classical Ciphers — Brute XOR, Caesar, Vigenère, Substitution]({{ '/blog/tutorials/crypto-lab-04-classical-ciphers-brute/' | relative_url }})
