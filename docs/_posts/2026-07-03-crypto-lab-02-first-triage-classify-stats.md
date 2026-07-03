---
title: "Crypto Lab, Part 2: First Triage — classify auto and stats scan"
description: Crypto Lab's classify auto emits a ranked verdict for an unknown RF payload — plaintext, substitution, repeating-XOR, periodic scrambler, LFSR, or strong-encrypted — each with the exact next command, backed by the stats scan and stats period statistical triage.
category: tutorials
keywords: cryptolab classify, entropy analysis, index of coincidence, chi-square, xor key length, autocorrelation period, payload triage, rf security testing, cipher identification, gophertrunk
tags: [cryptolab, triage, entropy, getting-started, security-testing, rf]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 2
charts: true
---

*Part 2 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. Before you attack anything, you triage — and Crypto Lab's front door tells you what you're holding and what to run next.*

> **TL;DR:** `cryptolab classify auto -in unknown.bin` runs the cheap measurements the other tools rely on — entropy, index of coincidence, chi-square, XOR key-length triage, autocorrelation period, and a randomness battery — and emits a **ranked verdict** (`plaintext`, `substitution-or-shift`, `repeating-xor`, `periodic-scrambler`, `lfsr-or-keyless-scrambler`, or `strong-encrypted`), each with the **exact recommended next command**. Under it, `stats scan` and `stats period` give you the raw numbers.

> **Authorized testing only.** Everything here assumes the payload came from a system you own or are licensed to assess.

**Key takeaways**

- **`classify auto` is the front door** for any unknown byte payload — it never leaves you guessing which tool to reach for next.
- **Every verdict carries a next command.** The output is a routing decision, not just a label.
- **`stats scan` gives the raw triage numbers** — entropy, IC, chi-square, and the top XOR key-length candidates.
- **`stats period` finds the repeat** — autocorrelation peaks and repeated n-grams that reveal a cipher's period or a scrambler's frame size.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab classify auto -in unknown.bin` | Ranked verdict + recommended next command |
| `-max-keylen N` | Cap the repeating-XOR key length considered (default 40) |
| `cryptolab stats scan -in payload.bin` | Entropy, IC, chi-square, top XOR key lengths |
| `cryptolab stats period -in payload.bin` | Autocorrelation period + repeated-n-gram histogram |
| `-max-lag N` | Max autocorrelation lag for `stats period` (default 256) |
| `-ngram N` | N-gram length for the repeat histogram (default 3) |
| `-format json` (global) | Emit the verdict as JSON for scripting |

## In this post

- **Why triage first** — and how a five-second classify saves an hour of wrong turns.
- **Reading `classify auto`** — the six verdict classes and what each recommends.
- **`stats scan`** — the four numbers that separate strong encryption from a weak cipher.
- **`stats period`** — finding the period that sets your key length or frame size.
- **A worked triage** — from raw bytes to a concrete next command.

## Why triage before you attack

Ada's instinct, holding her first unknown payload, was to start brute-forcing XOR keys. Reese stopped her: *"You don't know it's XOR yet. You don't even know it's a cipher. Triage first."* That's the discipline the classifier encodes. A payload can be plaintext you failed to recognize, a shift cipher, a repeating-key cipher, a periodic analog-style scrambler, a short-LFSR keystream, or genuinely strong encryption with nothing to attack. Each of those wants a *different* tool, and running the wrong one wastes time and — worse — can produce a plausible-looking false positive.

`classify auto` runs all the cheap measurements at once and routes you. It is the RF-framed analogue of the "auto-detect" step in tools like Ciphey or CrypTool's cipher identifier, but it knows about the things RF payloads actually are: LFSR scramblers and reused keystreams, not just book ciphers.

## Reading `classify auto`

```bash
gophertrunk cryptolab classify auto -in unknown.bin
```

```text
classify/auto — 4096 bytes: likely repeating-xor (confidence 78%)
  bytes                 4096
  entropy_bits          6.41
  index_of_coincidence  0.0389
  chi_square_uniform    812.5
  printable_fraction    0.31
  recommended  cryptolab brute xor -in <file> -keylen 7

findings (ranked):
  repeating-xor          0.78   repeating-XOR key length ~7 (normalized Hamming 1.42)
  periodic-scrambler     0.34   autocorrelation peak at lag 7 (z=4.1)
```

The tool emits **one finding per plausible class, ranked by confidence**, and promotes the top one into a headline and a `recommended` field. Each finding is a class plus a rationale plus the *exact command to run next* — filled in with the numbers it just measured (note the `-keylen 7` above came straight from the key-length guess). The six classes it can emit:

| Verdict class | What it means | It recommends |
|---|---|---|
| `plaintext` | Already readable (high printable fraction, low entropy) | no decryption needed |
| `substitution-or-shift` | IC near English, mostly letters | `brute substitution` (or `brute caesar`) |
| `repeating-xor` | A short repeating-XOR key length stands out | `brute xor -keylen N` |
| `periodic-scrambler` | Strong autocorrelation peak at some lag | `stats period`, then `descramble rolling` or `ks mtp` |
| `lfsr-or-keyless-scrambler` | High entropy *but* failing a randomness test | `lfsr bm` |
| `strong-encrypted` | ~8 bits/byte entropy and passes the battery | no weak structure — needs key material or IV reuse |

The `lfsr-or-keyless-scrambler` verdict is the clever one. High entropy usually screams "strong encryption," but if the payload *fails* the linear-complexity or spectral test in the randomness battery, that high entropy is coming from a **structured generator**, not a real cipher — the signature of an LFSR scrambler. The classifier catches exactly that case and routes you to `lfsr bm` instead of letting you write the payload off as unbreakable. We go deep on that distinction in [Part 3]({{ '/blog/tutorials/crypto-lab-03-randomness-nist-lfsr/' | relative_url }}).

One honest boundary the tool prints itself: `classify` only triages *byte payloads*. Analog voice scrambling (spectral inversion) operates on PCM audio, not bytes, so for `.s16` clips you go straight to the `descramble` tool ([Part 8]({{ '/blog/tutorials/crypto-lab-08-analog-voice-descrambling/' | relative_url }})).

## `stats scan`: the four numbers

When you want the raw triage numbers rather than a routing decision, `stats scan` gives them:

```bash
gophertrunk cryptolab stats scan -in payload.bin
```

```text
stats/scan — 4096 bytes: entropy 6.410 bits/byte, IC 0.0389, chi-square 812.5
  keylen=7    0.41   normalized_hamming 1.42
  keylen=14   0.33   normalized_hamming 1.71
  keylen=21   0.29   normalized_hamming 1.98
note: intermediate statistics: possibly a polyalphabetic / repeating-key
  cipher — try the top key lengths above with `cryptolab brute`.
```

Four measurements do most of the work:

- **Shannon entropy** (bits/byte). Near 8 means strong encryption or compression — no weak structure to exploit. Well below 8 means there's exploitable regularity.
- **Index of coincidence.** English text sits around 0.066; uniform random sits near 0.0039. An elevated IC (`> 0.045`) points at a monoalphabetic or shift cipher, or plaintext.
- **Chi-square vs uniform.** How far the byte distribution is from flat — a companion to entropy.
- **XOR key-length triage.** A normalized-Hamming test over candidate key lengths; the lowest-scoring lengths are the likeliest repeating-XOR periods.

`stats scan` prints a note that interprets the numbers for you — high entropy ("looks like strong encryption"), elevated IC ("try the brute tool"), or intermediate ("try the top key lengths"). The `-max-keylen` flag (default 40) caps how long a repeating-XOR key it will consider.

Here is the key-length triage as a chart — entropy stays flat while the normalized-Hamming score dips at the true period and its multiples:

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="560" height="300" role="img"
        aria-label="Normalized Hamming distance by candidate XOR key length, dipping at 7, 14, and 21"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["2","3","4","5","6","7","8","9","10","14","21"],
"values":[2.9,3.0,2.8,3.1,2.7,1.42,2.6,3.0,2.8,1.71,1.98],
"pass":[false,false,false,false,false,true,false,false,false,true,true],
"ylabel":"normalized Hamming (lower = likelier period)" }
</script>
<figcaption>The normalized-Hamming score dips sharply at key length 7 and again at its multiples 14 and 21 — the fingerprint of a 7-byte repeating key.</figcaption>
</figure>

## `stats period`: finding the repeat

Where `stats scan` guesses a key length from Hamming distance, `stats period` measures the repeat directly, with byte autocorrelation and a repeated-n-gram histogram:

```bash
gophertrunk cryptolab stats period -in payload.bin -max-lag 256 -ngram 3
```

```text
stats/period — 4096 bytes: 2 period candidate(s) within lag 256
  period=7    5.10   coincidence 0.061  z_score 5.1
  period=14   3.20   coincidence 0.048  z_score 3.2
  ngram=556e49  4     count 4  kind repeated_ngram
note: strongest period is 7 byte(s): for a repeating-XOR cipher try
  `cryptolab brute xor -keylen 7`; for a periodic scrambler this is the
  likely frame size.
```

The autocorrelation peaks (ranked by z-score) tell you the dominant period; the repeated n-grams show you *where* the structure repeats. This is the measurement that sets two important knobs downstream: the **key length** for a repeating-XOR or Vigenère brute, and the **frame size** for a rolling scrambler in `descramble rolling`. The flags — `-max-lag` (how far out to look, default 256), `-ngram` (block length for the repeat histogram, default 3), and `-top` (how many candidates to report) — trade runtime for reach.

If no autocorrelation peak stands out, `stats period` says so, and that absence is itself a finding: no short repeating period is consistent with strong encryption or a long, aperiodic keystream.

## A worked triage

Ada's actual first payload ran like this. `classify auto` came back `repeating-xor` at 78%, recommending `brute xor -keylen 7`. She confirmed the period with `stats period`, which put a clean autocorrelation peak at lag 7 (z = 5.1). One `brute xor -keylen 7` later ([Part 4]({{ '/blog/tutorials/crypto-lab-04-classical-ciphers-brute/' | relative_url }})) and the plaintext fell out. Total time from raw bytes to plaintext: under a minute — because she let the classifier route her instead of guessing.

Reese's version of the lesson: the classifier isn't magic, it's *bookkeeping*. It runs the same handful of statistics you'd run by hand, then does the tedious part — remembering which statistic implies which tool — so you don't misread a high-entropy LFSR as unbreakable or waste a substitution solve on a repeating-XOR cipher.

Every one of these tools honors the global `-format` flag, so `classify auto -in unknown.bin` with a global `-format json` gives you a machine-readable verdict you can pipe into a script that runs the recommended command automatically.

## Where this goes next

The classifier's most interesting call — separating a genuinely strong keystream from a structured scrambler that merely *looks* random — leans entirely on the randomness battery. [Part 3]({{ '/blog/tutorials/crypto-lab-03-randomness-nist-lfsr/' | relative_url }}) opens up that battery: the NIST SP 800-22 subset, why a failing linear-complexity or spectral test is the unmistakable signature of an LFSR, and how that one test distinguishes keyed encryption from a keyless scrambler. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) list every flag if you want the reference alongside.

## FAQ

**Do I always have to run `classify` first?**
No, but it's the cheapest possible first move and it routes you correctly. If you already know the payload class, jump straight to the tool — `classify` just automates the "which tool?" decision.

**What's the difference between `stats scan` and `classify auto`?**
`stats scan` gives you the raw numbers (entropy, IC, chi-square, key lengths); `classify auto` runs those *plus* the randomness battery and turns them into a ranked verdict with a recommended command. Use `classify` to decide, `stats` to inspect.

**Why does a high-entropy payload sometimes get flagged as an LFSR scrambler, not strong encryption?**
Because entropy alone can't tell a real cipher from a structured generator. If the payload has ~8-bit entropy but *fails* the linear-complexity or spectral randomness test, that's an LFSR — high entropy, but predictable. The classifier catches it and points you at `lfsr bm`.

**Can I script the recommended command?**
Yes. Run with the global `-format json` and parse the `recommended` field. The classifier fills in the measured key length or lag, so the command is ready to run.

## Series navigation

**Part 2 of 10** · ←[Part 1: Breaking It Is the Test]({{ '/blog/tutorials/crypto-lab-01-breaking-it-is-the-test/' | relative_url }}) · Next →[Part 3: Randomness — NIST SP 800-22 & the LFSR Signature]({{ '/blog/tutorials/crypto-lab-03-randomness-nist-lfsr/' | relative_url }})
