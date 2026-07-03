---
title: "Crypto Lab, Part 6: The assess Battery — Attacking P25/DMR/TETRA"
description: Crypto Lab's assess crypto harness runs every applicable attack against captured P25, DMR, and TETRA frames and grades each — cipher-strength, a published-weakness knowledge base, IV reuse, known-plaintext, weak/default keys against the real ciphers, and reduced-keyspace brute — landing on a RESISTANT, PARTIAL, or BROKEN verdict.
category: tutorials
keywords: assess crypto, p25 encryption assessment, dmr enhanced privacy, tetra tea1 backdoor, cve-2022-24402, weak key attack, key brute force, iv reuse, adp rc4, des ofb, gophertrunk cryptolab
tags: [cryptolab, assess, p25, dmr, tetra, security-testing, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 6
---

*Part 6 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. This is the headline: one command that tries every attack and grades your deployment.*

> **TL;DR:** `cryptolab assess crypto -in frames.jsonl` takes captured encrypted frames and **actively attempts decryption by every applicable method**, grading each from 0% (held) to 100% (a fail). Methods escalate: `cipher-strength` and `known-weakness` (both PARTIAL, the latter a floor — e.g. TETRA TEA1's 80→32-bit backdoor, CVE-2022-24402), `iv-reuse` (PARTIAL), then the recovery methods `known-plaintext`, `weak-key`, `key-brute` (`-brute-bits`, capped 32), and `keystream-lfsr` (all BROKEN). The overall verdict is <span class="lab-verdict lab-verdict--ok">RESISTANT</span>, <span class="lab-verdict lab-verdict--warn">PARTIAL</span>, or <span class="lab-verdict lab-verdict--bad">BROKEN</span>.

> **Authorized testing only.** Point `assess` at captures from a system you own or are licensed to assess.

**Key takeaways**

- **One harness, every method.** `assess crypto` runs the whole ladder and reports each method's effectiveness.
- **Effectiveness is a percentage** of captured ciphertext each method recovered or exposed.
- **A verified full decrypt is <span class="lab-verdict lab-verdict--bad">BROKEN</span>.** A leak without a break is <span class="lab-verdict lab-verdict--warn">PARTIAL</span>; nothing recovered is <span class="lab-verdict lab-verdict--ok">RESISTANT</span>.
- **Real ciphers, real recoveries.** `weak-key` and `key-brute` run the actual ADP/DES/3DES/AES constructions, so a default key is truly recovered, not just flagged.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab assess crypto -in frames.jsonl` | Run every applicable attack, grade each |
| `-protocol p25\|tetra\|dmr` | Algorithm-id namespace for the weakness KB (default p25) |
| `-known-label call-12 -known-pt known.bin` | A verification oracle → verified recoveries |
| `-keys candidate-keys.txt` | Extend the weak-key dictionary (one hex key/line) |
| `-brute-bits N` | Reduced-keyspace brute over the low N bits (capped 32) |
| `-base-key HEX` | Fix the high bits for `-brute-bits` |

## In this post

- **What `assess` does** — the security test as one command.
- **The method ladder** — seven methods, what each proves, and how each grades.
- **The known-plaintext oracle** — how `-known-label`/`-known-pt` turn flags into verified breaks.
- **Cipher coverage** — which P25/DMR ciphers run natively, and which are advisory-only.
- **Reading the verdict** — RESISTANT, PARTIAL, BROKEN, and the honesty about limits.

## The security test as one command

`assess` is the toolkit's thesis made executable. Everything in the earlier parts — IV reuse, keystream analysis, weak-key checks — is a *method*, and `assess crypto` runs them all against one capture, then reports how effective each was:

```bash
gophertrunk cryptolab assess crypto -in frames.jsonl -protocol p25
```

```text
assess/crypto — PARTIAL — strongest method "iv-reuse" at 34% over 61440
  ciphertext bytes across 214 frames
  frames  214
  verdict  PARTIAL
  overall_effectiveness  0.34
findings (by effectiveness):
  iv-reuse         0.34   applicable true   exposed p1^p2 in 3 groups
  cipher-strength  0.10   applicable true   structured ciphertext in 2 frames
  known-weakness   0.00   applicable true   (advisory) no published break for algid 0x84
  known-plaintext  0.00   applicable false  needs -known-label/-known-pt
  weak-key         0.00   applicable false  no default key matched
note: PARTIAL: no full break, but information leaked...
```

**Effectiveness** is the fraction of captured ciphertext each method recovered or exposed — so the findings sort with the most successful attack on top, and 100% from a *verified* method is a complete break.

## The method ladder

Seven methods run, in escalating capability. The first three can only expose (they floor at PARTIAL); the last four can achieve a verified recovery (BROKEN).

<figure class="lab-figure">
<svg viewBox="0 0 640 250" width="640" height="250" role="img" aria-label="assess method flow from exposure methods to recovery methods to a verdict">
  <g font-size="11.5" fill="currentColor">
  <text x="10" y="18" fill="var(--fg-muted)">EXPOSURE (floor: PARTIAL)</text>
  <rect x="10" y="26" width="180" height="24" rx="4" fill="none" stroke="currentColor"/>
  <text x="20" y="42">cipher-strength</text>
  <rect x="10" y="56" width="180" height="24" rx="4" fill="none" stroke="currentColor"/>
  <text x="20" y="72">known-weakness (KB)</text>
  <rect x="10" y="86" width="180" height="24" rx="4" fill="none" stroke="currentColor"/>
  <text x="20" y="102">iv-reuse</text>
  <text x="10" y="140" fill="var(--fg-muted)">RECOVERY (verified: BROKEN)</text>
  <rect x="10" y="148" width="180" height="24" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="20" y="164">known-plaintext</text>
  <rect x="10" y="178" width="180" height="24" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="20" y="194">weak-key</text>
  <rect x="10" y="208" width="180" height="24" rx="4" fill="none" stroke="var(--accent)"/>
  <text x="20" y="224">key-brute / keystream-lfsr</text>
  <line x1="200" y1="120" x2="260" y2="120" stroke="currentColor" stroke-width="1.5"/>
  <polygon points="260,120 252,116 252,124" fill="currentColor"/>
  <rect x="266" y="80" width="150" height="34" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="341" y="101" text-anchor="middle" fill="var(--accent)">RESISTANT</text>
  <rect x="266" y="120" width="150" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="341" y="141" text-anchor="middle">PARTIAL</text>
  <rect x="266" y="160" width="150" height="34" rx="6" fill="none" stroke="currentColor"/>
  <text x="341" y="181" text-anchor="middle">BROKEN</text>
  <text x="440" y="120" fill="var(--fg-muted)">highest outcome</text>
  <text x="440" y="136" fill="var(--fg-muted)">any method reached</text>
  <text x="440" y="152" fill="var(--fg-muted)">sets the verdict</text>
  </g>
</svg>
<figcaption>Exposure methods can only leak (PARTIAL); recovery methods can verify a full break (BROKEN). The strongest outcome any method reaches sets the overall verdict.</figcaption>
</figure>

| Method | What it does | Counts toward |
|---|---|---|
| `cipher-strength` | Is the ciphertext distinguishable from random? Structured output = a weak/keyless construction | exposure (PARTIAL) |
| `known-weakness` | Look up each algorithm id in the published-weakness KB and report design flaws | advisory (floors at PARTIAL) |
| `iv-reuse` | Frames sharing an IV leak `p1⊕p2` with no key | exposure (PARTIAL) |
| `known-plaintext` | Recover the keystream from a known frame, decrypt its reuse group | recovery (BROKEN) |
| `weak-key` | Try default / supplied keys with the **real** ADP/DES/3DES/AES cipher, verified | recovery (BROKEN) |
| `key-brute` | Reduced-keyspace brute over the low `-brute-bits` bits against the oracle | recovery (BROKEN) |
| `keystream-lfsr` | Is a recovered keystream a short, predictable LFSR? | recovery (BROKEN) |

The `known-weakness` method is the knowledge-base advisory. It looks up each algorithm id in a published-weakness database keyed by protocol, and reports design flaws even when the cipher itself isn't bundled. Its headline example is **TETRA TEA1**: for `-protocol tetra`, it reports the 80→32-bit key reduction backdoor (**CVE-2022-24402**) and notes that a 2³² brute would recover the key given a TEA1 implementation. That's an actionable finding without fabricating an unverified cipher — a floor of PARTIAL, because a *published* break in a deployed algorithm is itself an exposure.

## The known-plaintext oracle

The single flag that upgrades `assess` from "flags weaknesses" to "verifies breaks" is the known-plaintext oracle: `-known-label` naming a frame and `-known-pt` giving its plaintext. With it:

- `known-plaintext` becomes possible — derive the keystream from the named frame and decrypt its whole reuse group.
- `weak-key` and `key-brute` become **definitive**: a candidate key isn't just tried, it's *verified* against the oracle, so a hit is a confirmed break rather than a guess.

```bash
gophertrunk cryptolab assess crypto -in frames.jsonl -protocol p25 \
    -known-label call-12 -known-pt known.bin -keys candidate-keys.txt
```

```text
assess/crypto — BROKEN — strongest method "weak-key" at 100% over 61440
  ciphertext bytes across 214 frames
  verdict  BROKEN
  weak-key   1.00   verified true   key 0000000000000000   sample_ascii "DISPATCH..."
note: BROKEN: a method achieved verified complete decryption — the encryption
  FAILED this test.
```

That result — an all-zero DES key recovered and verified — is the deployment failing the test. Reese's rule holds: `assess` only claims BROKEN on a *verified* recovery, never on a plausible-looking one. Without the oracle, the same weak key would be tried but couldn't be confirmed, so the harness would decline to call it a break.

The reduced-keyspace brute is the `key-brute` method: `-brute-bits N` searches the low N bits of the key against the oracle, with `-base-key` fixing the high bits. N is **capped at 32** for feasibility — this is the hashcat-analog and the exact shape of the TEA1 backdoor attack, not a claim to brute a full 128-bit key.

## Cipher coverage

The active recovery methods carry the **real keystream constructions**, so a default or small-keyspace key is genuinely recovered:

- **P25** (`-protocol p25`): ADP/RC4, DES-OFB, two- and three-key Triple-DES, AES-128/256-OFB.
- **DMR** (`-protocol dmr`): RC4 "Enhanced Privacy", DES/3DES/AES-OFB. DMR's vendor key/IV *derivation* is proprietary, so the cipher cores are keyed with the material you supply (the reconstructed key; the frame IV used directly).

Ciphers whose keystream function isn't bundled at all — **TETRA TEA1–4** and **Motorola DES-XL** — are covered by the `known-weakness` advisory instead of a fabricated implementation. For those, `assess` reports the published weakness and, via the external-cipher bridge ([Part 9]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }})), can drive an operator-supplied cipher to run the backdoor brute for real.

## Reading the verdict, and the honesty about limits

The overall verdict is the highest outcome any method reached:

- <span class="lab-verdict lab-verdict--ok">RESISTANT</span> — no applicable method recovered plaintext. The encryption held against every attack tried, with this data.
- <span class="lab-verdict lab-verdict--warn">PARTIAL</span> — information leaked: a reused IV, structured ciphertext, a recovered keystream segment, or a published break in the algorithm in use.
- <span class="lab-verdict lab-verdict--bad">BROKEN</span> — a method achieved verified complete decryption. A fail.

And the toolkit is deliberately honest about its ceiling. `assess` **cannot** brute a strong key out of a strong cipher — AES/3DES with a non-default key and rotated IVs has an infeasible keyspace, and it says so plainly, reporting <span class="lab-verdict lab-verdict--ok">RESISTANT</span>. That result *is* the security finding: the encryption is doing its job. What `assess` breaks is what fails in the field — reused IVs, default and test keys, small or backdoored keyspaces, keyless obfuscation, and weak keystreams. It is a test of operational reality, not a magic decryptor.

## Where this goes next

`assess` grades the encryption. But before you can even get to the ciphertext, you often have to peel off framing — and the CRC that validates it. [Part 7]({{ '/blog/tutorials/crypto-lab-07-crc-recovery-recipe/' | relative_url }}) covers `crc recover` and the CyberChef-style `recipe` pipeline that chains transforms and analyses into one reusable artifact. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) detail every `assess` method and flag.

## FAQ

**What does the effectiveness percentage mean?**
The fraction of captured ciphertext a method recovered or exposed. 100% from a *verified* method (one confirmed against a known-plaintext oracle) is a complete break; a lower number from an exposure method means partial leakage.

**Do I need known plaintext to run `assess`?**
No — it runs the exposure methods (cipher-strength, known-weakness, iv-reuse) without it. But `-known-label`/`-known-pt` turn the weak-key and key-brute methods into *verified* recoveries and enable known-plaintext, so an oracle is what upgrades PARTIAL findings into confirmed BROKEN ones.

**Can it break TEA1 or a properly-keyed AES system?**
It reports TEA1's published 80→32-bit backdoor (CVE-2022-24402) as an advisory and can drive an external TEA1 tool to run the 2³² brute. A properly-keyed AES system with rotated IVs it reports as RESISTANT — it will not fabricate a break it cannot verify.

**Why is `-brute-bits` capped at 32?**
Feasibility. A 32-bit search is the realistic ceiling for a brute against a known-plaintext oracle — and it's exactly the size of the TEA1 backdoor keyspace. It is not a claim to brute a full-length key.

## Series navigation

**Part 6 of 10** · ←[Part 5: Keystream Reuse & Many-Time-Pad]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }}) · Next →[Part 7: CRC Recovery & the Recipe Pipeline]({{ '/blog/tutorials/crypto-lab-07-crc-recovery-recipe/' | relative_url }})
