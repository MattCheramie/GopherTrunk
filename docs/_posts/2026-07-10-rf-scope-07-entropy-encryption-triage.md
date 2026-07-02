---
title: "RF Scope, Part 7: Entropy & Encryption Triage — The Crypto Lab Bridge"
description: rfscope's entropy analyzer triages unidentified digital emitters — blind-demodulating a representative payload, running the cryptolab randomness battery and classify-auto measurements, and returning a verdict from plaintext to strong-encrypted with a recommended cryptolab command, plus a -frames-out file that hands the bytes to Crypto Lab.
category: tutorials
keywords: rfscope, entropy, encryption detection, cryptolab bridge, shannon entropy, index of coincidence, chi-square, xor key length, nist sp 800-22, frames-out, keystream reuse, gophertrunk
tags: [rfscope, cryptolab, rf-analysis, advanced, gophertrunk, dsp]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "RF Scope"
series_part: 7
charts: true
---

*Part 7 of **RF Scope**, GopherTrunk's protocol-agnostic RF network analyzer.
Topology found an unknown digital emitter; this is where RF Scope opens its payload
and decides what to do with it.*

> **TL;DR:** The `entropy` analyzer (which depends on `topology`) takes every
> unidentified digital emitter, blind-demodulates a representative payload (at least
> **32 bytes**, at most **16 emitters** per scene), and runs the cryptolab randomness
> battery plus the same measurements `classify auto` uses — Shannon entropy, index of
> coincidence, chi-square, XOR key-length, period, NIST SP 800-22. It returns a
> **verdict** — plaintext, substitution-or-shift, repeating-xor, periodic-scrambler,
> lfsr-or-keyless-scrambler, strong-encrypted, or inconclusive — each with a recommended
> cryptolab command. `-frames-out` writes a cryptolab `ks` frames file. **Mercury's
> payload is handed off here.**

**Key takeaways**

- **It triages only unknowns.** Emitters that decode as a known protocol are skipped —
  triage is for what `siglab` could not name.
- **The measurements are cryptolab's own**, called directly, so RF Scope and the
  toolkit agree on the verdict.
- **Seven verdicts, each with a next command** you can paste straight into cryptolab.
- **`-frames-out` closes the loop** — it emits `{label, iv, ct}` JSONL that feeds
  `cryptolab classify auto`, `ks reuse`/`mtp`, and `brute`.

## Cheat sheet

| Verdict | Recommended cryptolab command |
|---|---|
| `plaintext` | already plaintext — no decryption needed |
| `substitution-or-shift` | `cryptolab brute substitution -in <file>` |
| `repeating-xor` | `cryptolab brute xor -in <file> -keylen N` |
| `periodic-scrambler` | `cryptolab stats period` then descramble / `ks mtp` |
| `lfsr-or-keyless-scrambler` | `cryptolab lfsr bm -in <file>` |
| `strong-encrypted` | `cryptolab ks reuse` if IVs repeat, else key material needed |
| `-frames-out frames.jsonl` | Emit the `ks` frames file for all triaged payloads |

## In this post

- **What the entropy analyzer does** — and what it deliberately skips.
- **The measurement battery** — entropy, IC, chi-square, XOR, period, NIST.
- **The seven verdicts** and the command each one points at.
- **`-frames-out`** — the Crypto Lab bridge format.
- **Mercury's hand-off** to [Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}).

## What it does

Wireshark cannot tell you whether a flow is encrypted, but analysts eyeball payload
entropy to guess. RF Scope's `entropy` analyzer makes that rigorous for RF. For each
digital emitter that `siglab` could **not** identify, it recovers a byte payload and
asks: is this plaintext, lightly obfuscated, scrambled, or genuinely encrypted? It is
the "is this flow encrypted?" question, answered with statistics rather than a hunch.

It depends on `topology` because it works per **emitter**, not per burst — it wants the
longest, cleanest representative of each source. The loop is bounded on both ends:

```go
// internal/rfscope/entropy.go
if triaged >= maxEntropyEmitters { break }        // ≤ 16 emitters
if e.Fingerprint.ClassFamily != "digital" { continue }
// skip emitters that decode as a known protocol — triage is for unknowns
if ident, err := siglab.IdentifyIQ(...); err == nil && ident != nil && !ident.Inconclusive {
    continue
}
payload := survey.RecoverPayload(iq, rate, sc.Bursts[bidx].Features)
if len(payload) < minPayloadBytes { continue }    // ≥ 32 bytes
```

`survey.RecoverPayload` **blind-demodulates** the burst — using the classifier's
estimated symbol rate and features, with no protocol framing — into a raw byte stream.
That stream is not a decode; it is the bits on the air, sliced at the estimated symbol
boundaries. Below 32 bytes there is not enough to measure, so the emitter is skipped.

## The measurement battery

`classifyPayload` runs the payload through cryptolab's *own* engine packages —
imported directly, so it works in the default binary with no `-tags cryptolab` build:

```go
// internal/rfscope/entropy.go
ent := stats.ShannonEntropy(data)
ic := stats.IndexOfCoincidence(data)
chi := stats.ChiSquareUniform(data)
keylens := stats.GuessXORKeyLength(data, 40)
periods := stats.DetectPeriod(data, 256, 4)
bits := lfsr.BitsFromBytes(data)
rep := randomness.Battery(bits, randomness.DefaultAlpha) // Quick() under 2048 bits
```

Each measurement probes for a different kind of structure:

- **Shannon entropy** (bits/byte, max 8) — the headline randomness number. Near 8 is
  high-entropy; well below is structured.
- **Index of coincidence** — the probability two random bytes match. High IC survives a
  substitution cipher, so it is the tell for substitution/shift.
- **Chi-square vs uniform** — how far the byte distribution is from flat.
- **XOR key length** — `GuessXORKeyLength` scores candidate repeating-XOR key lengths;
  a low score at length > 1 means a repeating key.
- **Period** — `DetectPeriod` finds a repeating structure (a scrambler cycle) in the
  byte stream — the byte-domain twin of Part 5's channel-occupancy period.
- **NIST SP 800-22** — the `randomness.Battery` (or `Quick` for short payloads) runs the
  standardized randomness tests; `LooksRandom()` is the aggregate verdict, and specific
  tests like `linear_complexity` and `spectral_dft` failing points at an LFSR/scrambler.

These are the *exact* thresholds and recommendations cryptolab's `classify auto`
applies, so RF Scope's triage and a full cryptolab run never disagree.

## The seven verdicts

`classifyPayload` collects every verdict whose evidence fires, sorts by confidence, and
returns the top one. The taxonomy, roughly from least to most protected:

| Class | Fires when | Meaning |
|---|---|---|
| `plaintext` | >95% printable, entropy < 4.8 | Readable bytes — no crypto |
| `substitution-or-shift` | IC > 0.045, letters > 0.6 | Classical substitution |
| `repeating-xor` | XOR key length > 1, low score | Repeating-key XOR |
| `periodic-scrambler` | A period detected | Cyclic scrambler |
| `lfsr-or-keyless-scrambler` | entropy > 7.0 + a NIST test fails | LFSR / keyless scrambler |
| `strong-encrypted` | entropy > 7.9 and `LooksRandom` | No exploitable structure |
| `inconclusive` | none of the above | Not enough signal |

The crucial distinction is between **obfuscation** and **encryption**. A
`repeating-xor`, `periodic-scrambler`, or `lfsr-or-keyless-scrambler` verdict says
*there is exploitable structure* — cryptolab can very likely recover the plaintext
without a key. Only `strong-encrypted` says the payload has *no weak structure* and
needs key material (or an IV/MI reuse) to go further. That "obfuscation is not
encryption" line is the moral the whole trilogy is building toward.

<figure class="lab-figure">
<canvas class="lab-chart" data-chart="bars" width="560" height="300" role="img"
        aria-label="NIST randomness battery pass/fail for a scrambled payload"></canvas>
<script type="application/json" class="lab-chart-data">
{ "categories":["monobit","runs","block-freq","longest-run","fft/spectral","linear-complexity","approx-entropy","serial"],
"values":[0.42,0.31,0.55,0.28,0.004,0.006,0.19,0.24],
"threshold":0.01,
"pass":[true,true,true,true,false,false,true,true],
"ylabel":"p-value" }
</script>
<figcaption>A payload that passes most NIST tests but fails <code>spectral_dft</code> and <code>linear_complexity</code> — the signature of an LFSR/scrambler, not strong encryption.</figcaption>
</figure>

## Why the battery, not a single number

It would be tempting to reduce all this to one number — Shannon entropy — and call
anything above 7.9 bits "encrypted." RF Scope deliberately does not, because entropy
alone is *ambiguous*. A repeating-XOR ciphertext, an LFSR-scrambled stream, and a
genuinely AES-encrypted payload can all sit at 7.9+ bits per byte; entropy cannot tell
them apart. The other measurements exist precisely to break that tie:

- Two payloads at entropy 7.95 look identical until you check `linear_complexity`. The
  one that **fails** it has a short LFSR behind it — recoverable with
  `cryptolab lfsr bm`. The one that passes everything has no such shortcut.
- A payload at entropy 6.5 might be plaintext in an unusual encoding, or a substitution
  cipher. The **index of coincidence** separates them: a substitution cipher preserves
  the language's letter-frequency structure, so its IC stays high even though the bytes
  are scrambled.
- A scrambler with a short cycle can push entropy high while leaving a **period** that
  `DetectPeriod` finds immediately — the byte-domain echo of Part 5's channel period.

So each verdict is a *conjunction* of evidence, and `classifyPayload` collects every
verdict whose conditions fire, then sorts by confidence. That is why the result carries
not just a class but the raw `EntropyBits`, `IndexOfCoincidence`, `ChiSquareUniform`, and
`KeyLen` alongside it — you can see *which* structure earned the verdict, and overrule it
if you know something the statistics do not. Reese's rule again: *"one number is a
horoscope; a battery is a diagnosis."*

## A note on payload length

The `randomness.Battery` call has a length dependency that matters in practice. For a
payload of at least **2048 bits** (256 bytes) it runs the full NIST battery; below that
it falls back to `randomness.Quick`, a lighter subset, because the heavy tests are not
statistically meaningful on short samples. This is the same reason the analyzer skips
anything under 32 bytes outright — you cannot say much about the randomness of a handful
of bytes.

The consequence for an operator: a **longer captured burst gives a more confident
verdict.** A hopper that only ever fires 40-byte bursts will be triaged, but on the
`Quick` battery, and a `strong-encrypted` call on it is weaker than the same call on a
256-byte payload. When a verdict matters and the emitter transmits longer bursts
elsewhere, capture a window that catches one — the entropy analyzer always picks the
*longest* burst of the emitter to triage, so a longer capture directly buys a better
measurement.

## The Crypto Lab bridge: -frames-out

A verdict is useful; a payload cryptolab can *act on* is better. `-frames-out` writes
every triaged payload as a cryptolab `ks` **frames file** — one JSON object per line,
`{label, iv, ct}`, with `iv` and `ct` hex-encoded:

```go
// internal/rfscope/bridge.go
rec := frameRec{
    Label: fmt.Sprintf("emitter-%d@%.4fMHz", r.EmitterID, float64(r.FreqHz)/1e6),
    IV:    hex.EncodeToString(r.iv),
    CT:    hex.EncodeToString(r.payload),
}
```

The field names and encoding match exactly what cryptolab's `ks` loader parses, so the
file feeds `cryptolab classify auto`, `ks reuse`/`mtp`, and `brute` with no
conversion. In practice:

```bash
# Triage a band and emit frames for every unknown payload
gophertrunk rfscope analyze -in wide.cfile -sample-rate 2400000 -frames-out frames.jsonl

# Hand them to Crypto Lab
gophertrunk cryptolab classify auto -in frames.jsonl     # needs -tags cryptolab
```

If any frames share an IV/MI, RF Scope even points you straight at
[keystream reuse]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }}):
`DetectKeystreamReuse` reports the reuse groups and prints a `cryptolab ks reuse`
suggestion. That IV-as-crib workflow is the subject of
[Crypto Lab Part 5]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }}),
and the broader "breaking it is the test" philosophy is
[Crypto Lab Part 1]({{ '/blog/tutorials/crypto-lab-01-breaking-it-is-the-test/' | relative_url }}).

## Mercury's hand-off

Here is where Mercury leaves RF Scope. Topology gave us one hopping digital emitter
that `siglab` never named — a textbook triage candidate. The entropy analyzer
blind-demodulates its longest burst into a payload comfortably over 32 bytes and runs
the battery. The result is telling: **not** `strong-encrypted`. Its entropy is high but
not maxed, and — as in the chart above — it fails the structural NIST tests rather than
passing everything. RF Scope grades it as *not-obviously-strong* obfuscation and writes
its bytes to the frames file:

```text
rfscope: wrote 1 cryptolab frame(s) → frames.jsonl
```

That single line is the hand-off. Mercury is now a `ct` in a JSONL file, waiting for
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }}) to finish the job. The
Part 10 reveal — that Mercury was never encryption at all, but a keyless, length-seeded
byte obfuscator — starts from exactly this frame.

## Where this goes next

The entropy verdicts do not just sit in a table — they become **anomalies**. [Part
8]({{ '/blog/tutorials/rf-scope-08-expert-info-anomalies/' | relative_url }}) covers
the `expert` analyzer, which turns `strong-encrypted` into an `encrypted` alert and the
obfuscation verdicts into `obfuscated` warnings, alongside its own rules for hoppers,
intermittent emitters, and abnormally wide or narrow carriers.

## FAQ

**Does the triage need the cryptolab build tag?**
No. `classifyPayload` calls cryptolab's untagged engine packages directly, so the
in-binary triage runs in the default build. Only the full `ks reuse` hand-off wants a
`-tags cryptolab` build.

**Why skip emitters that `siglab` identifies?**
Triage is for *unknowns*. If a signal decodes as a named protocol, GopherTrunk already
understands it; the entropy analyzer's job is the traffic nothing else can name.

**What's the difference between obfuscated and encrypted here?**
`obfuscated` verdicts (repeating-xor, scrambler, LFSR) have exploitable structure —
cryptolab can usually recover plaintext without a key. `strong-encrypted` has no weak
structure and needs key material or IV reuse. The distinction is the whole point.

**What exactly is in a frames file line?**
One JSON object: `{"label","iv","ct"}`, with `iv` (recovered IV/MI, possibly empty) and
`ct` (the recovered payload) hex-encoded — the format `cryptolab ks` parses directly.

## Series navigation

**Part 7 of 10** · ←
[Part 6: Topology]({{ '/blog/tutorials/rf-scope-06-topology-emitters-conversations/' | relative_url }})
· Next →
[Part 8: Expert Info — Anomalies]({{ '/blog/tutorials/rf-scope-08-expert-info-anomalies/' | relative_url }})
