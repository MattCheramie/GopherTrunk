---
title: "Crypto Lab, Part 4: Classical Ciphers — Brute XOR, Caesar, Vigenère, Substitution"
description: Crypto Lab's brute tool recovers classical ciphers — repeating XOR, Caesar shift, Vigenère, and monoalphabetic substitution — using an embedded English trigram language model and optional cribs, with lfsr recovering the keystream of a linear scrambler.
category: tutorials
keywords: brute force cipher, repeating xor, vigenere cipher, caesar cipher, substitution cipher, hill climbing, english trigram model, crib dragging, berlekamp massey, gophertrunk cryptolab
tags: [cryptolab, brute-force, classical-ciphers, security-testing, advanced, rf]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 4
charts: true
---

*Part 4 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. When triage says "classical cipher," this is the tool that recovers the key.*

> **TL;DR:** `cryptolab brute` has four modes — `xor`, `caesar`, `vigenere`, and `substitution` — each recovering a classical cipher by scoring candidate plaintexts against an **embedded English trigram language model** (optionally boosted with a `-crib`). Use `xor` for repeating-XOR, `caesar` for a single shift, `vigenere` for a polyalphabetic key, and `substitution` for a monoalphabetic alphabet. `lfsr bm` handles the linear-scrambler case, recovering the keystream as `pt ⊕ ct`.

> **Authorized testing only.** These attacks belong on ciphertext from systems you own or are licensed to assess.

**Key takeaways**

- **Four modes, four cipher families** — match the mode to what triage found.
- **English scoring drives recovery.** An embedded trigram model ranks candidate plaintexts; a `-crib` sharpens it.
- **XOR and Vigenère auto-detect key length** from the same periodicity test `stats` uses.
- **Substitution is a hill-climb** with random restarts — trade `-restarts` for recovery on short ciphertexts.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab brute xor -in cipher.bin` | Recover a single- or repeating-byte XOR key (auto key length) |
| `cryptolab brute xor -in cipher.bin -keylen 7` | Solve a known repeating-XOR key length |
| `cryptolab brute caesar -in cipher.txt` | Recover a single additive (Caesar/ROT) shift |
| `cryptolab brute vigenere -in cipher.txt -keylen 5` | Recover a repeating additive (Vigenère) key |
| `cryptolab brute substitution -in cipher.txt -restarts 40` | Solve a monoalphabetic substitution (hill-climb) |
| `-crib "UNIT "` | Boost scoring with a known-plaintext substring |
| `cryptolab lfsr bm -in keystream.bin` | Recover a linear scrambler's LFSR |

## In this post

- **When each mode applies** — routing from the triage verdict to the right brute.
- **`brute xor`** — single-byte and repeating-key, with auto key-length detection.
- **`brute caesar` and `brute vigenere`** — additive ciphers over the alphabet.
- **`brute substitution`** — hill-climbing a full alphabet against English.
- **`lfsr`** — the recovery path for a linear keystream.

## Matching the mode to the cipher

The classifier from [Part 2]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }}) already routes you here with a specific recommendation. This is the map it's using:

| Triage says | Cipher family | Mode |
|---|---|---|
| `repeating-xor` | Bytewise repeating XOR | `brute xor` |
| `substitution-or-shift` (single shift) | Caesar / ROT-n | `brute caesar` |
| `substitution-or-shift` (polyalphabetic) | Vigenère | `brute vigenere` |
| `substitution-or-shift` (monoalphabetic) | Substitution alphabet | `brute substitution` |
| `lfsr-or-keyless-scrambler` | Linear keystream | `lfsr bm` |

What every mode has in common is **scoring against English**. The engine carries an English trigram language model (`internal/cryptolab/engine/lang`) and a corpus; each candidate key produces a candidate plaintext, and the model scores how English-like that plaintext reads. The highest-scoring candidate wins. If you already know a fragment of the plaintext — a unit ID, a call sign, a common word — pass it as a `-crib` and the scorer weights any candidate that contains it, which dramatically sharpens recovery on short or noisy ciphertext.

Why does English scoring work at all? Because natural-language plaintext is enormously non-random at the level of letter triples: `THE`, `AND`, `ING`, `ION` are common; `QZX`, `JQV`, `WKZ` essentially never occur. A trigram model turns that regularity into a single number — the log-probability of a candidate plaintext under English — and the correct key is the one that produces text scoring far above every wrong key, because a wrong key smears the letter statistics toward uniform noise. This is the same statistical lever the classifier's index-of-coincidence check pulls in [Part 2]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }}); `brute` just uses it to *rank keys* rather than to *classify a payload*. The practical consequence for you as an analyst: the more ciphertext you feed, the sharper the separation between the right key and the runners-up, and the shorter the message, the more a `-crib` earns its keep.

## `brute xor`

Repeating-XOR is the workhorse of weak RF obfuscation, so it gets the most machinery. Give it a key length and it solves directly:

```bash
gophertrunk cryptolab brute xor -in cipher.bin -keylen 7
```

Leave the key length off and it **auto-detects**. It first solves the single-byte case, then scores the top few key lengths guessed from the same normalized-Hamming periodicity test `stats scan` uses, and keeps whichever the English scorer likes best — a guard against overfitting a genuinely single-byte cipher to a noisy long-key guess:

```bash
gophertrunk cryptolab brute xor -in cipher.bin -crib "UNIT "
```

```text
brute/xor — recovered 7-byte XOR key
  keylen                  7
  auto_keylen_candidates  [7 14 5]
  key=52414449 4f31       0.83   key_ascii "RADIO1"
    plaintext_prefix "UNIT 214 DISPATCH TO SECTOR 7, ACKNOWLEDGE..."
```

The finding reports the key in hex, its printable rendering, and a plaintext preview. Here the crib `"UNIT "` both confirmed the solve and pulled the right key length to the top. This chart shows why: across candidate key lengths, the English score of the best recovered plaintext spikes at the true period.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="line" width="560" height="300" role="img"
        aria-label="English score of the best recovered plaintext by candidate XOR key length, peaking at 7"></canvas>
<script type="application/json" class="lab-chart-data">
{ "series":[{"label":"English score","dots":true,"points":[[2,0.31],[3,0.34],[4,0.29],[5,0.41],[6,0.33],[7,0.83],[8,0.36],[9,0.30],[10,0.38],[14,0.62]]}],
"xlabel":"candidate key length","ylabel":"English score","xmin":2,"xmax":14,"ymin":0,"ymax":1 }
</script>
<figcaption>English score by candidate XOR key length: a sharp peak at 7 (the true period) and a secondary bump at its multiple 14. The scorer keeps the peak.</figcaption>
</figure>

## `brute caesar` and `brute vigenere`

These two handle additive ciphers over the alphabet. `caesar` recovers a single shift — the classic ROT-n — by trying every shift and scoring:

```bash
gophertrunk cryptolab brute caesar -in cipher.txt
```

```text
brute/caesar — recovered Caesar shift 13
  shift  13
  key=0d   0.79   plaintext_prefix "MERIDIAN CONTROL HOLDING ON CHANNEL..."
```

`vigenere` extends that to a repeating additive key. Like `xor`, it auto-detects the key length from the periodicity test if you don't supply one, then solves each key position independently:

```bash
gophertrunk cryptolab brute vigenere -in cipher.txt -keylen 5
gophertrunk cryptolab brute vigenere -in cipher.txt          # auto key length
```

The distinction between `xor` and `vigenere` is the alphabet: `xor` operates bytewise over all 256 values (right for binary RF payloads), while `caesar`/`vigenere` operate additively over the letter alphabet (right for text-like ciphertext). Triage's `substitution-or-shift` verdict with a strong periodicity peak points at Vigenère; a flat periodicity with elevated IC points at Caesar or substitution.

## `brute substitution`

A monoalphabetic substitution — every letter mapped to another, consistently — has an astronomically large keyspace (26! alphabets), far too many to enumerate. So `brute substitution` doesn't enumerate; it **hill-climbs**. Starting from a frequency-seeded guess, it repeatedly swaps letter mappings, keeping any swap that improves the English trigram score, and restarts from fresh random alphabets to escape local optima:

```bash
gophertrunk cryptolab brute substitution -in cipher.txt -restarts 40
```

```text
brute/substitution — recovered substitution key QWERTYUIOPASDFGHJKLZXCVBNM
  key    QWERTYUIOPASDFGHJKLZXCVBNM
  score  -3821.4
  plaintext  0.  key ...  plaintext_prefix "THE REPEATER SITE IS OFFLINE FOR..."
note: monoalphabetic substitution solving assumes English plaintext and
  enough ciphertext; very short or rare-letter-heavy messages may leave a
  few letters unresolved.
```

The `-restarts` flag (default 25) is the key runtime/recovery trade-off: more restarts give the hill-climb more chances to find the global optimum, which matters most on short ciphertexts where the statistics are thin. `-seed` fixes the RNG for reproducible runs. The tool is honest about its assumption — it needs English plaintext and enough ciphertext; a very short message or one heavy on rare letters may leave a few mappings unresolved.

Reese's field note: substitution solving is where the crib is worth the most. If you know one word appears — a town name on a repeater ID, say — even without a `-crib` flag the recovered alphabet usually snaps into place around it once enough of the surrounding text resolves.

## `lfsr`: the linear-scrambler path

When triage says `lfsr-or-keyless-scrambler` rather than a classical cipher, `brute` isn't the tool — `lfsr` is. It recovers the shortest LFSR for a bit sequence via Berlekamp–Massey ([Part 3]({{ '/blog/tutorials/crypto-lab-03-randomness-nist-lfsr/' | relative_url }}) covers the theory):

```bash
gophertrunk cryptolab lfsr bm -in keystream.bin
```

If you have a plaintext/ciphertext pair instead of a bare keystream, `lfsr keystream` does the whole job in one step — it computes the keystream as `pt ⊕ ct` and immediately runs Berlekamp–Massey on it:

```bash
gophertrunk cryptolab lfsr keystream -pt known.bin -ct cipher.bin
```

```text
lfsr/keystream — 4096 keystream bytes; linear complexity 23
  keystream_bytes    4096
  linear_complexity  23
  keystream 5f3a91c2...   taps [23 18 0]
```

A low linear complexity relative to the number of bits means the "encryption" was a linear scrambler, and the recovered taps regenerate the entire keystream. That's a keyless break — no key was ever needed, only the register structure. It's a preview of the deeper theme in [Part 10]({{ '/blog/tutorials/crypto-lab-10-subject-framework-alias/' | relative_url }}): obfuscation is not encryption.

## Where this goes next

Classical ciphers are the tutorial cases. The break that actually happens against modern digital RF encryption is subtler and more common: **keystream reuse**. When a stream cipher reuses its key and IV, two ciphertexts XOR to the XOR of their plaintexts — no key recovery required. [Part 5]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }}) is the core real-world break, and it's where the Mercury frames from RF Scope finally get analyzed. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) list every mode's flags.

## FAQ

**What's the difference between `brute xor` and `brute vigenere`?**
`xor` is bytewise over all 256 values — right for binary RF payloads. `vigenere` is additive over the letter alphabet — right for text-like ciphertext. Both auto-detect key length from the same periodicity test.

**How does the tool "know" it found the right key?**
It scores each candidate plaintext against an embedded English trigram language model and keeps the highest scorer. A `-crib` adds weight for candidates containing a known substring, which is decisive on short ciphertext.

**Why does substitution use random restarts?**
The substitution keyspace (26!) is far too large to enumerate, so the solver hill-climbs from a frequency-seeded guess. Restarts let it escape local optima; more restarts improve recovery on short messages at the cost of runtime.

**My payload is high-entropy — should I still try `brute`?**
Probably not for a classical cipher. High entropy with a failing randomness test points at an LFSR — use `lfsr bm`. `brute`'s classical modes expect exploitable low-entropy structure.

## Series navigation

**Part 4 of 10** · ←[Part 3: Randomness & the LFSR Signature]({{ '/blog/tutorials/crypto-lab-03-randomness-nist-lfsr/' | relative_url }}) · Next →[Part 5: Keystream Reuse — Many-Time-Pad & Crib-Dragging]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }})
