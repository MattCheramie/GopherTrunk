# Motorola P25 talker-alias cipher — clean-room recovery: provenance & methodology of record

**Document status:** authoritative provenance record. Supersedes the "not
cracked / UNVERIFIED" status in `p25-talker-alias-cryptanalysis.md`,
`p25-talker-alias-chosen-plaintext.md`, and
`docs/reference/motorola-talker-alias-cipher.md`.

**Subject:** the recovery, integration, and verification of the proprietary
Motorola P25 talker-alias per-byte obfuscation cipher in GopherTrunk
(`internal/radio/p25/motorola/alias.go`, `CipherVerified = true`).

**The claim this document defends:** GopherTrunk (Apache-2.0) contains **no
copyrightable expression** — no source code, no data table, no algorithm
transcription — taken from SDRTrunk (GPLv3) or any other GPL work. The cipher's
mathematical structure and its 256-byte substitution table were **derived from
observed input/output behaviour**, not read or copied from any third-party
source. Everything in the shipped tree was independently authored from that
derived specification and is reproducible from committed artifacts **with no
access to SDRTrunk at all.**

> **Not legal advice.** This is an engineering and evidentiary record written by
> the implementer. It maps the method onto well-established reverse-engineering
> doctrine so a reviewer (or counsel) can evaluate it, but it is not a legal
> opinion. Copyright and license questions are fact- and jurisdiction-specific;
> obtain professional advice before relying on the licensing conclusions for
> anything you distribute.

---

## 0. TL;DR — the boundary in one paragraph

We treated SDRTrunk as a **sealed measurement instrument**. We fed chosen
ciphertext bytes into its *public* decode accessor (`getAlias().getValue()`) and
recorded the plaintext strings that came out. We **never opened, read,
disassembled, or transcribed** the cipher's source, its lookup table, or its
algorithm. From ~22,000 (input → output) observations, GopherTrunk's own
cryptanalysis tooling plus hand analysis **reconstructed** the cipher as a set of
mathematical facts (an accumulator recurrence, a seed law, an output-assembly
rule, and a substitution table solved by majority vote across observations).
GopherTrunk's decoder was then written from scratch against that reconstructed
specification. The reconstruction is validated to reproduce the instrument's
output **byte-for-byte on a held-out set it never trained on (1242/1242
characters)** and, independently, to decode a **real over-the-air capture** to a
sensible name with a valid CRC. The throwaway Java instruments that link against
SDRTrunk are quarantined in an ephemeral scratchpad and were **never committed or
distributed**.

---

## 1. Why there is a licensing question at all

| Project | License | Relevance |
|---|---|---|
| **GopherTrunk** | Apache License 2.0 (permissive) | the work we ship |
| **SDRTrunk** (`io.github.dsheirer`) | GNU **GPLv3** (strong copyleft) | contains a *working* implementation of this cipher |

The two licenses are, for practical purposes of code re-use, one-directional:
GPLv3 source cannot be copied into an Apache-2.0 project and redistributed under
the Apache terms. A naïve "port the table and the function from SDRTrunk" would
copy GPLv3 **expression** into GopherTrunk and violate the GPL. That path was
explicitly rejected. This document exists to show that the path we *did* take
does not carry that expression across the boundary.

The distinction the whole argument turns on is **idea/method/fact vs.
expression**:

- **What copyright protects:** the specific *expression* — the literal source
  code as written, the particular arrangement of a creative work.
- **What copyright does not protect** (17 U.S.C. § 102(b); *Baker v. Selden*,
  101 U.S. 99 (1879); *Feist Publications v. Rural Telephone*, 499 U.S. 340
  (1991)): ideas, procedures, **processes, systems, methods of operation**, and
  **facts**. A cipher *is* a process/method; the value a lookup table maps an
  input to *is* a fact about that process. Facts and methods are free to
  discover and re-express.

The functional behaviour of a wire cipher — "input byte 0x24 at accumulator
state W decodes to character 'C'" — is a **fact about a communications
protocol**, the same fact that lives in Motorola's own firmware. Discovering
that fact by observation, and re-expressing it in independently written code, is
the ordinary business of protocol reverse engineering.

### Reverse-engineering precedent the method maps onto

- ***Sega Enterprises Ltd. v. Accolade, Inc.***, 977 F.2d 1510 (9th Cir. 1992) —
  intermediate copying performed **to access the unprotected functional
  elements** of a program (interoperability) can be fair use.
- ***Sony Computer Entertainment, Inc. v. Connectix Corp.***, 203 F.3d 596 (9th
  Cir. 2000) — reverse engineering, including observing a program's behaviour to
  build an independent, interoperable implementation, is fair use where the
  functional elements sought are not otherwise accessible.
- ***Google LLC v. Oracle America, Inc.***, 141 S. Ct. 1183 (2021) —
  re-implementing an API's *functional* declarations independently is
  permissible; naming another program's public methods to invoke them is not
  copying its creative expression.
- **Industry clean-room practice** (e.g. the Phoenix/IBM-BIOS re-implementation;
  ReactOS/Wine): reconstruct behaviour from a specification of *what* a component
  does, keeping the *how* (the original's code) out of the implementer's hands.

Our method sits squarely in the *Sega/Connectix/Google* black-box lane: we ran
the program and observed its behaviour to extract unprotected functional facts,
then wrote our own code. We did **not** copy expression.

---

## 2. The boundary we held: what we did NOT do

Enumerated so a reviewer can check each one:

1. **We never read the cipher source.** SDRTrunk's talker-alias cipher lives in
   its `…message.lc.motorola` classes. We never opened, printed, decompiled, or
   read the body of the method that performs the per-byte transform, nor the
   class that holds the substitution table. No SDRTrunk source file was open in
   any editor, `cat`-ed, or grepped for cipher logic during the recovery.
2. **We never copied the table.** The 256-byte substitution table now in
   `alias.go` was **solved** from observations (Section 6.3), not transcribed. It
   is expressed in our own Go, in an affine frame we chose (Section 8), which is
   not even the same byte values a naïve transcription would produce.
3. **We never copied any code.** No SDRTrunk method, class, constant block, or
   comment appears in GopherTrunk. The GopherTrunk decoder is a from-scratch
   implementation of the reconstructed specification.
4. **We did not distribute the instrument.** The Java probes that link SDRTrunk
   (Section 5) are throwaway, live only in an ephemeral per-session scratchpad,
   and were **never** added to the GopherTrunk tree, committed, or published.

What we *did* do — and state plainly, because a document that hides it would not
survive scrutiny — is **execute** SDRTrunk (a lawful act; running a program is
not copying it) and **call its public accessor** on inputs of our choosing to
observe outputs. Observing outputs yields facts, not expression.

---

## 3. The starting point (what pre-existed this recovery)

This was not a cold start. GopherTrunk already contained a clean-room
cryptanalysis lab, `internal/cryptolab` (build-tag gated), and a prior
investigation (issue **#773**, documented in
`research/p25-talker-alias-cryptanalysis.md`) that had, **purely from a
ground-truth CSV of decoded outputs**:

- established the message framing (`WACN | System | RadioID | encoded-alias |
  CRC-16`, encoded field exactly `2n` bytes for an `n`-character alias);
- recovered a candidate 256-byte substitution table and the **per-byte decode
  primitive** `out = Modd·(LUT[e] − Hodd)` with `Modd = inv((accum)|1)`,
  `Hodd = accum >> 8`;
- **correctly localised the two open problems**, on the record: (a) "the high
  byte is not a function of any 2-byte combination of the readable inputs — its
  true dependency is on the hidden low byte," and (b) "no LCG family reproduces
  the accumulator's high byte."

That prior work was already data-driven and license-clean, and it was **gated
off** (`CipherVerified = false`) precisely because it was unproven. The recovery
described here resolved the two localized walls and then *proved* the result.

---

## 4. Strategy overview

Six phases, each independently checkable:

| Phase | Goal | Output | SDRTrunk touched? |
|---|---|---|---|
| 1 | Build a behavioural **oracle** and prove it faithful | throwaway Java probe (quarantined) | yes — run, public API only |
| 2 | Design a **chosen-ciphertext** experiment | 22,380 train + 300 holdout (input→output) | yes — run, public API only |
| 3 | **Reconstruct** the cipher from the observations | a mathematical spec + solved table | **no** — pure analysis of our data |
| 4 | **Validate** the reconstruction on held-out data | 1242/1242 byte-exact | **no** |
| 5 | **Implement** independently in Go from the spec | `alias.go` decoder + tests | **no** |
| 6 | **Corroborate** against real over-the-air data | RID 200062 → "CRIO 0062", CRC 0x6A96 | **no** |

The line between "yes" and "**no**" is the clean-room line. Everything from
Phase 3 onward — every artifact that ends up in the shipped tree — was produced
with **no** access to SDRTrunk, from data that is (input bytes we chose → output
string the instrument emitted). Phases 1–2 produced only *observations*, never
code that ships.

---

## 5. The instruments (throwaway, quarantined, GPL-linked)

To observe behaviour we wrote small Java programs that link SDRTrunk's jar and
call its public API. **These are derivative works of SDRTrunk (they import its
classes) and are therefore kept strictly out of GopherTrunk** — they live only in
the ephemeral session scratchpad (`/private/tmp/…/scratchpad/`, auto-deleted) and
were never committed or distributed. They are measurement instruments, not
product.

Files (scratchpad only): `AliasOracleProbe.java` (feasibility),
`AliasOracleProbe2–4.java` (sweeps), `Probe5.java` (anchor-invariance test),
`VerifyOracle.java` (fidelity proof). The decisive property — that they read
**only** the public decoded output — is visible in the harness itself:

```java
// From AliasOracleProbe.java — our throwaway instrument, not shipped.
// Header comment, verbatim: "Treats MotorolaTalkerAliasComplete.getAlias()
//   as an opaque oracle. … No cipher internals are inspected."
MotorolaTalkerAliasComplete complete = new MotorolaTalkerAliasComplete(
        m, APCO25Talkgroup.create(1), 0, TimeslotMessage.TIMESLOT_0, 0L, Protocol.APCO25);
String s = complete.getAlias().getValue();   // <-- the ONLY thing we read: the output string
```

We construct the public message object from bytes **we** chose, ask it for its
decoded string, and record `(bytes_in → string_out)`. The cipher's table and
transform are never referenced. Naming `getAlias()`/`MotorolaTalkerAliasComplete`
to *invoke* them is calling a public API (cf. *Google v. Oracle*), not copying
the cipher's expression.

> **Rule of record:** GPL-linked instruments never enter the Apache tree. If any
> `*.java` under `io.github.dsheirer` imports ever appears in a GopherTrunk
> commit, that is a provenance breach and must be reverted. See the quarantine
> ledger, Section 11.

---

## 6. Method, step by step

### 6.1 Build the oracle and prove it faithful (Phase 1)

Before trusting the instrument, we proved two things (`VerifyOracle.java`), so
the observations we would later fit against were known to be genuine SDRTrunk
behaviour, not an artifact of how we constructed inputs:

- **(A) Real captured vector.** A genuine over-the-air alias vector (the seven
  44-bit fragments from SDRTrunk's own header-class documentation example) was
  reassembled through SDRTrunk's **production assembler path**; its **CRC-16
  validated on the genuine OTA data** and the header ISSI matched — i.e. the
  instrument decodes real captures correctly.
- **(B) Path equivalence.** For ~120 synthetic inputs, the production assembler
  path and the direct-constructor path we used for bulk sweeping produced
  **byte-identical** decoded strings with valid CRCs. So the fast direct path is
  a faithful stand-in for the real decode path.

Only after (A) and (B) did we generate the training corpus.

### 6.2 Design the chosen-ciphertext experiment (Phase 2)

A stream cipher's internals are exposed fastest by **controlling its input and
watching the output move**. The corpus was designed, not scraped:

- **Baselines:** all-zero ciphertext at every length `L` (isolates the pure
  keystream / seed trajectory).
- **Single-byte sweeps:** vary one ciphertext byte through all 256 values at each
  position and length (isolates the per-byte transform and the accumulator
  update — one variable at a time).
- **Randoms:** full-width random ciphertext at each length (stress / holdout).
- **Anchor-invariance control (`Probe5.java`):** the same encoded bytes under
  five different WACN/System/RID anchors. Result: **decode is independent of the
  anchor** — it depends only on the encoded bytes and their count. This told us
  the seed is keyed on **length**, not identity, and let us ignore three fields.

Totals: **22,380** training rows + **300** held-out rows, each a JSON record of
`{len, pos, enc_hex, dec_bytes_hex, …}` — pure (input → output).

### 6.3 Reconstruct the cipher (Phase 3 — no SDRTrunk)

From the observations alone (GopherTrunk's `cryptolab` differential modes plus
hand analysis), the cipher was reconstructed as a 16-bit **additive accumulator**
stream cipher. The two walls fell as follows:

- **High byte depends on the *hidden low byte*.** Differential sweeps showed each
  character's high byte equals the even output byte **unless** the low byte is a
  negative `int8` (≥ 0x80), in which case it is forced to `0xFF`. It is a
  per-character **output-assembly rule** keyed on the low byte's sign — not a
  state lookup, which is why earlier "wiring" searches found nothing.
- **The accumulator is additive in the right frame.** In the naïve frame the
  16-bit state is *not* a clean LCG. Sweeping the substitution table's affine
  gauge freedom, gauge `a = 117` (inverse 221) **linearises** the accumulator to
  the clean additive law `W ← W + 0x0125` (293). The apparent period-7 wobble was
  just carry from the low byte (`293 & 0xFF = 37`).
- **Seed law from the zero-input chain:** `seed(n) = 0xC433 + 586·(n−1)`
  (586 = 2·293), i.e. every length starts one character-step along one shared
  trajectory.
- **The 256-byte table** was solved **per index by majority vote** across every
  observation that exercised it: each `(row, position)` gives one equation for
  one table entry; thousands of redundant, agreeing equations pin all 256 values.
  This is statistical recovery of facts from data — the antithesis of copying.

The full reconstructed specification is in Section 8.

### 6.4 Validate on held-out data (Phase 4 — no SDRTrunk)

The reconstruction was fit on the training sweeps only, then run against the
**300 held-out rows it never saw.** A held-out reproduction is a genuine recovery
test, not a memorised round-trip. It reproduces the instrument's output
**byte-for-byte**. Real command and output, run just now against the **committed**
holdout with **no SDRTrunk present** (Section 7).

### 6.5 Implement independently in Go (Phase 5 — no SDRTrunk)

`internal/radio/p25/motorola/alias.go` was written from the Section 8
specification. It is original Go: our variable names, our control flow, our
chosen affine frame for the table, our documentation. Nothing was transliterated
from any Java source (which we never read). It is gated behind
`CipherVerified = true` only because a committed regression fixture (the held-out
oracle vectors) now stands behind it, per the repo's issue-closing policy.

### 6.6 Corroborate against real air (Phase 6 — no SDRTrunk)

The strongest possible check is ground truth that is **not** the oracle: a real
Motorola system's own output over the air. The `#376` capture (Victorian MMR, RID
**200062**) reassembles to `SUID | cipher(18B) | CRC(2B) | pad(6B)`. Run through
the recovered cipher:

- the 18-byte cipher region decodes to **`CRIO 0062`** — clean printable ASCII,
  and `0062` is the tail of the radio's own ID (200062). It is the **only**
  length/alignment that yields *any* clean ASCII; a wrong cipher gives zero.
- **CRC-16/GSM over `SUID | cipher` = `0x6A96`**, exactly the on-air CRC bytes (a
  1-in-65,536 coincidence — and it matches).

Two independent, unrelated checks (a printable name containing the RID; a
matching 16-bit CRC) both pass on genuine Motorola firmware output. That is
correctness confirmed against reality, not against our own instrument.

---

## 7. Reproduction transcript (real commands, real output)

Everything below is reproducible from **committed** repo artifacts with **no
SDRTrunk**. Outputs are the actual results captured while writing this document.

**7.1 — The reconstruction reproduces the held-out oracle 100%** (self-contained
Go validator + the committed holdout fixture):

```
$ go run alias_cipher_recovered.go \
      internal/radio/p25/motorola/testdata/alias_cipher_oracle_holdout.jsonl
holdout rows:            300
clean-char match:        1242/1242 = 100.000%  (FFFD chars excluded: 1)
full-row match (clean):  299/299 = 100.00%
charset-truncated rows:  0 (excluded: no ground truth)
RESULT: cipher fully recovered (100% clean-char match).
```

(FFFD = one character the instrument's own lossy UTF-16 charset destroyed; it
carries no recoverable ground truth and is excluded. See Section 9.)

**7.2 — The shipped decoder passes its committed regression suite:**

```
$ go test ./internal/radio/p25/motorola/ -run 'TestRecoveredCipher|TestRealSampleDecodes' -v
=== RUN   TestRecoveredCipherHoldout
=== RUN   TestRecoveredCipherFixtures
=== RUN   TestRealSampleDecodesAliasWithValidCRC
--- PASS: TestRealSampleDecodesAliasWithValidCRC (0.00s)
--- PASS: TestRecoveredCipherFixtures (0.00s)
    alias_recovered_test.go:88: holdout: 300 rows, 1242/1242 clean chars byte-identical (FFFD excluded 1, charset-truncated rows 0)
--- PASS: TestRecoveredCipherHoldout (0.00s)
PASS
ok      github.com/MattCheramie/GopherTrunk/internal/radio/p25/motorola 0.257s
```

- `TestRecoveredCipherHoldout` replays the 300 committed held-out vectors through
  the **shipped** `DecodeAliasBytes` and requires every clean character byte-exact.
- `TestRecoveredCipherFixtures` pins independent oracle vectors (the decoded bytes
  are the instrument's output, not self-generated) plus readable round-trips.
- `TestRealSampleDecodesAliasWithValidCRC` is the real-air `CRIO 0062` / `0x6A96`
  check of Section 6.6.

**7.3 — Full repository gate green with all changes:**

```
$ make vet test
… (170 packages) …
ok      github.com/MattCheramie/GopherTrunk/internal/radio/p25/motorola
ok      github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase1
ok      github.com/MattCheramie/GopherTrunk/internal/radio/p25/phase2
# 0 failures
```

---

## 8. The recovered specification (self-contained)

Included in full so this document is independently sufficient — a reader could
re-implement the cipher from this section alone, which is itself the point: the
knowledge is a compact set of facts, now expressed here in our own words. All
byte arithmetic is mod 256; `W` is mod 2¹⁶.

```
n   = len(encoded) / 2                       # output char count; odd trailing byte dropped
W   = 0xC433 + 586*(n-1)                     # length seed (586 = 2*293)
for i in 0 .. 2n-1:                          # one step per encoded byte
    Hodd   = W >> 8                          # accumulator high byte
    Modd   = inverse_mod_256((W & 0xFF) | 1) # odd multiplier (low byte forced odd)
    out[i] = Modd * (LUT[encoded[i]] - Hodd) # per-byte substitution
    W     += 293 * (encoded[i] + 1)          # accumulator update

for k in 0 .. n-1:                           # pack into n UTF-16BE characters
    low_k  = out[2k+1]
    high_k = 0xFF        if low_k >= 0x80     # low byte is a negative int8
           = out[2k]     otherwise
    char_k = (high_k << 8) | low_k
```

`LUT` is the recovered 256-byte bijection. It is committed in
`internal/radio/p25/motorola/alias.go` (`motorolaAliasLUT`), expressed in the
affine frame that makes the accumulator update the clean additive law above. The
outer framing that delimits `encoded` is `SUID(7B) | cipher(2n) | CRC-16/GSM |
zero-pad`, with the cipher region **self-delimited by its CRC** (the shortest
even span whose trailing CRC-16/GSM over `SUID|cipher` matches and after which the
frame is all zero). CRC-16/GSM parameters: poly `0x1021`, init `0x0000`, xorout
`0xFFFF` — confirmed against the `#376` capture (`0x6A96`).

---

## 9. Charset loss (disclosed)

The instrument serialises through a lossy UTF-16 charset. Two effects are handled
honestly rather than hidden:

- **U+FFFD** (~2 % of rows) marks a character the charset destroyed; excluded from
  comparison (its true bytes are unrecoverable). The holdout has exactly 1 such
  character, excluded — hence "1242, FFFD excluded 1."
- **Length-truncated rows** (27 of 22,380 training rows; **none** in the holdout)
  where the instrument recorded fewer bytes than the true cipher output. Excluded
  for lack of comparable ground truth. Our decoder is causally correct on them
  regardless (byte *i* never affects an earlier character).

Neither effect inflates the score: excluded characters are removed from *both*
numerator and denominator, and the holdout — the number that matters — has zero
truncated rows.

---

## 10. Provenance ledger — every shipped artifact

Each item that entered the GopherTrunk tree, with its origin:

| Shipped artifact | What it is | Origin | Clean because |
|---|---|---|---|
| `internal/radio/p25/motorola/alias.go` | the decoder + `motorolaAliasLUT` + framing | authored from the Section 8 spec | our own Go; table solved from observations, not copied |
| `…/motorola/alias_test.go`, `alias_recovered_test.go` | regression tests | authored | our own Go |
| `…/motorola/testdata/alias_cipher_oracle_holdout.jsonl` | 300 held-out vectors | **observations**: `(bytes_in → string_out)` | pure I/O facts; contains no SDRTrunk code, only inputs we chose and outputs we measured |
| `…/phase1/*`, `…/phase2/talker_alias.go`, `sigfollow/dispatcher.go` | framing/reassembly + wiring | authored | our own Go |
| this document + the three status-updated docs | prose | authored | our own words |

**Not shipped (quarantined):** all `*.java` probes, the 22,380-row training
dataset, the CSV conversions, the `cryptolab` solver scratch outputs. These live
only in the ephemeral scratchpad.

The single artifact that is *derived from running SDRTrunk* — the holdout JSONL —
contains **no SDRTrunk expression**. It is a list of (input bytes we selected →
output string the program emitted). Those outputs are functional facts about the
cipher (identical to what Motorola's own firmware produces); they are not
SDRTrunk's creative expression, and committing a measurement of a program's
factual output is not committing the program.

---

## 11. Quarantine ledger (GPL-linked instruments)

| Instrument | Links GPL SDRTrunk? | Location | Committed / distributed? |
|---|---|---|---|
| `AliasOracleProbe.java` (+2/3/4) | yes | scratchpad only | **no** |
| `Probe5.java` | yes | scratchpad only | **no** |
| `VerifyOracle.java` | yes | scratchpad only | **no** |
| 22,380-row training dataset | derived by running it | scratchpad only | **no** |

Standing rule: no file importing `io.github.dsheirer` may ever be committed to
GopherTrunk. A CI/grep check for that import string in the tree is a cheap,
auditable guard (recommended follow-up).

---

## 12. Why this stands up to scrutiny

1. **We can produce the entire shipped result without SDRTrunk.** Section 7.1 runs
   on committed artifacts with SDRTrunk absent and reproduces the oracle 100%. If
   the result required copied SDRTrunk expression, it could not.
2. **The table is solved, not transcribed, and lives in a different frame.** Even
   the *byte values* in `motorolaAliasLUT` differ from a naïve transcription (we
   chose the affine gauge that linearises the accumulator). A diff against any
   SDRTrunk table would not match.
3. **The output-vs-instrument match is on held-out data** (1242/1242) — recovery,
   not memorisation.
4. **Independent ground truth agrees.** A real over-the-air capture decodes to a
   sensible name with a matching CRC (Section 6.6) — correctness confirmed against
   Motorola firmware, wholly independent of the instrument.
5. **The method is the sanctioned one.** Black-box observation to extract
   unprotected functional facts, then independent re-implementation, is exactly
   *Sega/Connectix/Google* territory. We ran the program; we did not copy it.
6. **The boundary is documented and auditable.** What we read (only public
   outputs), what we avoided (all cipher internals), and what is quarantined (all
   GPL-linked instruments) are enumerated and checkable.

---

## 13. Honest caveats

- **Not legal advice** (restated). The doctrine summary is context for review, not
  an opinion; distribution decisions warrant counsel.
- **Validation breadth.** Verified on the 300-row held-out oracle set **and** one
  real over-the-air capture (a Phase 2 FACCH-S frame, RID 200062). The
  voice-channel (Phase 1 LDU1/TDULC) carrier now shares the same verified decode
  by construction but has not been exercised against a *Phase 1* alias captured on
  air (none available). More real captures would broaden confidence; none are
  required for the license argument.
- **The instruments are GPL derivatives.** Stated plainly: the throwaway Java
  probes link SDRTrunk and are derivative works of it. They are never distributed,
  so no GPL distribution obligation attaches, and — the point of this document —
  none of their expression, or SDRTrunk's, is in GopherTrunk.

---

## 14. References

- Companion (now status-updated): `research/p25-talker-alias-cryptanalysis.md`
  (the data-driven findings), `research/p25-talker-alias-chosen-plaintext.md`
  (capture procedure), `docs/reference/motorola-talker-alias-cipher.md`.
- Code: `internal/radio/p25/motorola/alias.go` and its tests;
  `internal/radio/p25/phase2/talker_alias.go`; `internal/sigfollow/dispatcher.go`.
- Doctrine: 17 U.S.C. § 102(b); *Baker v. Selden*, 101 U.S. 99 (1879); *Feist v.
  Rural*, 499 U.S. 340 (1991); *Sega v. Accolade*, 977 F.2d 1510 (9th Cir. 1992);
  *Sony v. Connectix*, 203 F.3d 596 (9th Cir. 2000); *Google v. Oracle*, 141 S.
  Ct. 1183 (2021).
- Issue: [#773](https://github.com/MattCheramie/GopherTrunk/issues/773).
