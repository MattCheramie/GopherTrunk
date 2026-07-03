---
title: "Crypto Lab, Part 7: CRC Recovery & the Recipe Pipeline"
description: Crypto Lab's crc recover reconstructs a protocol's CRC parameters — width, polynomial, init, reflection, and xorout — from sample frames, while recipe run chains transform and analysis operations into a CyberChef-style reusable pipeline for de-obfuscating RF payloads.
category: tutorials
keywords: crc recovery, crc parameters, reveng, polynomial recovery, cyberchef pipeline, recipe operations, xor decrypt, rc4 decrypt, des ofb, gophertrunk cryptolab, rf reverse engineering
tags: [cryptolab, crc, recipe, pipeline, security-testing, advanced]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 7
---

*Part 7 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. Two workhorses: reconstruct a protocol's CRC from samples, and chain your whole de-obfuscation sequence into one reusable recipe.*

> **TL;DR:** `cryptolab crc recover` reconstructs CRC parameters (width, poly, init, refin, refout, xorout) from `datahex,crchex` sample frames, and `crc compute` runs a CRC with explicit parameters. `cryptolab recipe run` is a CyberChef-style ordered pipeline: chain **transform** ops (xor, not, reverse-bits, hex/base64, slice, the real cipher decrypts, the descramblers, extern-decrypt) and **analysis** ops (stats, randomness), piping bytes step to step. `recipe run -list` prints the live op catalogue.

> **Authorized testing only.** Reverse-engineer framing and de-obfuscate payloads from systems you own or are licensed to assess.

**Key takeaways**

- **CRC recovery is search, not magic.** Feed varied-length sample frames and `crc recover` solves for the parameter set that fits.
- **More varied samples disambiguate.** Multiple fitting parameter sets? Add frames of differing length.
- **`recipe` encodes a workflow.** The de-obfuscation sequence you'd repeat by hand becomes one reusable JSON/YAML artifact.
- **Transforms rewrite, analyses observe.** A recipe can decrypt a known-key capture and immediately measure the result.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab crc recover -in frames.txt -widths 16,8` | Recover CRC params from `datahex,crchex` lines |
| `cryptolab crc compute -in data.bin -width 16 -poly 0x1021` | Compute a CRC with explicit parameters |
| `cryptolab recipe run -in payload.bin -recipe r.json` | Run an ordered transform/analysis pipeline |
| `cryptolab recipe run -list` | List the available operations |
| `-out final.bin` | Write the final transformed bytes |

## In this post

- **Why CRC recovery matters** — validating framing before you touch the payload.
- **`crc recover`** — from sample frames to a parameter set.
- **`crc compute`** — checking a recovered CRC.
- **The recipe model** — transforms vs analyses, and the working buffer.
- **A worked recipe** — hex-decode, decrypt, measure, in one file.

## Why CRC recovery matters

Before you can analyze a payload, you usually have to *find* it inside a frame — and framing almost always ends in a CRC. Knowing the CRC lets you validate that you've segmented the frame correctly, strip the check bytes cleanly, and confirm a decrypt produced valid framing rather than plausible garbage. But protocols rarely document their exact CRC parameters, and there are a lot of them: width, generator polynomial, initial value, input/output reflection, and a final XOR. `crc recover` searches that parameter space against your samples and tells you which combination reproduces the observed checks.

## `crc recover`

Give it sample frames — one `datahex,crchex` pair per line — and the widths to try:

```bash
gophertrunk cryptolab crc recover -in frames.txt -widths 16,8
```

```text
crc/recover — 6 sample frames -> 1 candidate parameter set(s)
  samples  6
  candidates  1
  w16 poly=0x1021 init=0xffff xor=0x0 refin=false refout=false   1
    width 16  poly 0x1021  init 0xffff  xorout 0x0  refin false  refout false
```

That's CRC-16/CCITT-FALSE, recovered from six frames. The `-widths` flag (default `16`) is a comma-separated list of widths to try — add `8`, `32`, whatever the protocol might use. The mechanics are exactly what a tool like `reveng` does, folded into the toolkit.

The two notes `crc recover` prints are the whole art of using it. If **no** parameters fit, you need more sample frames of varied length, or a wider `-widths` list. If **multiple** parameter sets fit, they're indistinguishable on your current samples — and the fix is the same: add frames of *differing length*, because init and xorout only separate once the CRC has to span different amounts of data. Ada's first recovery came back with three candidate sets; two more frames of a different length collapsed it to one. Reese's rule: **vary the length, not just the count.** Ten frames of the same length disambiguate no better than two.

## `crc compute`

Once you have parameters, `crc compute` runs the CRC forward over a file so you can verify a frame or check your reconstruction:

```bash
gophertrunk cryptolab crc compute -in data.bin -width 16 -poly 0x1021 \
    -init 0xffff -xorout 0x0
```

```text
crc/compute — CRC-16 = 0x2b1d over 240 bytes
  crc    0x2b1d
  width  16
```

The flags mirror the recovered parameters exactly — `-poly` (default `0x1021`), `-init`, `-xorout` (default `0xFFFF`), `-refin`, `-refout` — so a recovery result feeds straight into a compute call to confirm it against a held-out frame. That round-trip (recover on one set, compute on another) is how you gain confidence the parameters generalize, the same discipline the LFSR recovery in [Part 3]({{ '/blog/tutorials/crypto-lab-03-randomness-nist-lfsr/' | relative_url }}) asks for.

## The recipe model

Real de-obfuscation is rarely one step. You hex-decode, then XOR off a mask, then decrypt with a recovered key, then check the entropy to see if it worked — and then you do it again on the next capture. `recipe run` captures that whole sequence as one reusable artifact. A recipe is an ordered list of **steps**; each step is either a **transform** that rewrites the working buffer or an **analysis** that measures it without changing it. Bytes flow from one step into the next.

If you've used CyberChef, the mental model is identical: a linear stack of operations, each consuming the output of the last. What makes the `recipe` tool worth having inside GopherTrunk rather than reaching for a separate web app is that its cipher ops are the *same* constructions the rest of the toolkit uses — so a recipe isn't a toy approximation of the real decode, it *is* the real decode, reproducible from the command line and driveable over the same JSONL captures the other tools read. And because a recipe is just a file, it's the natural unit for sharing an investigation: hand a colleague your `recipe.json` and they reproduce your entire de-obfuscation on their own capture, byte for byte, without re-deriving the steps.

<figure class="lab-figure">
<svg viewBox="0 0 640 130" width="640" height="130" role="img" aria-label="Recipe pipeline: input bytes flow through transform and analysis steps to final bytes">
  <g font-size="11.5">
  <rect x="6" y="46" width="86" height="38" rx="5" fill="none" stroke="currentColor"/>
  <text x="49" y="69" text-anchor="middle" fill="currentColor">input</text>
  <rect x="116" y="46" width="96" height="38" rx="5" fill="none" stroke="currentColor"/>
  <text x="164" y="64" text-anchor="middle" fill="currentColor">hex-decode</text>
  <text x="164" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">transform</text>
  <rect x="236" y="46" width="96" height="38" rx="5" fill="none" stroke="currentColor"/>
  <text x="284" y="64" text-anchor="middle" fill="currentColor">adp-decrypt</text>
  <text x="284" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">transform</text>
  <rect x="356" y="46" width="80" height="38" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="396" y="64" text-anchor="middle" fill="var(--accent)">stats</text>
  <text x="396" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">analysis</text>
  <rect x="460" y="46" width="96" height="38" rx="5" fill="none" stroke="var(--accent)"/>
  <text x="508" y="64" text-anchor="middle" fill="var(--accent)">randomness</text>
  <text x="508" y="78" text-anchor="middle" fill="var(--fg-muted)" font-size="9">analysis</text>
  <rect x="580" y="46" width="54" height="38" rx="5" fill="none" stroke="currentColor"/>
  <text x="607" y="69" text-anchor="middle" fill="currentColor">out</text>
  <g stroke="currentColor" stroke-width="1.3" fill="currentColor">
  <line x1="92" y1="65" x2="114" y2="65"/><polygon points="116,65 108,61 108,69"/>
  <line x1="212" y1="65" x2="234" y2="65"/><polygon points="236,65 228,61 228,69"/>
  <line x1="332" y1="65" x2="354" y2="65"/><polygon points="356,65 348,61 348,69"/>
  <line x1="436" y1="65" x2="458" y2="65"/><polygon points="460,65 452,61 452,69"/>
  <line x1="556" y1="65" x2="578" y2="65"/><polygon points="580,65 572,61 572,69"/>
  </g>
  </g>
</svg>
<figcaption>Transform steps rewrite the working buffer as it flows left to right; analysis steps read it without changing it. The final buffer is written to <code>-out</code>.</figcaption>
</figure>

Run `recipe run -list` for the live catalogue. The operations, from the docs:

- **Transforms:** `xor`, `not`, `reverse-bits`, `hex-decode`, `hex-encode`, `base64-decode`, `slice`; the real cipher decrypts `rc4-decrypt` (DMR Enhanced Privacy / generic), `adp-decrypt`, `des-ofb-decrypt`, `tdes-ofb-decrypt`, `aes-ofb-decrypt`; the analog-voice descramblers `descramble-invert`, `descramble-splitband` (`split` = fraction of Nyquist), `descramble-rolling` (`frame` samples, `schedule` = split fractions or `auto`); and `extern-decrypt` (an external cipher program — **CLI only**, see [Part 9]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }})).
- **Analyses:** `stats`, `randomness`.

The cipher ops reuse the same `p25crypto` keystream constructions the `assess` harness uses, so a recipe can decrypt a known-key capture and immediately measure the result — no exporting between tools.

## A worked recipe

A recipe file is JSON or YAML — either a top-level list of steps or an object with a `steps:` list. Each step is `{op, ...params}`:

```json
[
  {"op": "hex-decode"},
  {"op": "adp-decrypt", "key": "abcdef0123", "mi": "090807060504030201"},
  {"op": "stats"},
  {"op": "randomness"}
]
```

```bash
gophertrunk cryptolab recipe run -in payload.bin -recipe recipe.json -out clear.bin
```

```text
recipe/run — ran 4 step(s): 480 → 232 bytes
  1. hex-decode    transform  bytes_in 480  bytes_out 240
  2. adp-decrypt   transform  bytes_in 240  bytes_out 240
  3. stats         analysis   entropy 4.12  ic 0.061
  4. randomness    analysis   looks_random false
  final_ascii  "UNIT 214 DISPATCH..."
  written  clear.bin
note: transform steps rewrite the working buffer; analysis steps (stats,
  randomness) observe it without changing it.
```

Notice the payoff: steps 1–2 transform (hex-decode, then decrypt with a known key and MI), and steps 3–4 *measure* the result in place — entropy dropped to 4.12 and the randomness check reports structure, both confirming the decrypt produced real plaintext rather than noise. The de-obfuscation sequence an analyst repeats by hand becomes one file you can rerun on every capture, and the per-step report shows exactly where the byte count changes.

The report is non-nil even on a failing step, so if step 3 errors you still see what steps 1–2 did — useful when you're tuning a recipe and a parameter is wrong. The `-out` flag writes the final buffer; the web Recipe Builder ([Part 9]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }})) assembles the same pipeline interactively.

## Where this goes next

So far every tool has been byte-oriented. But some RF privacy isn't digital at all — it's **analog voice scrambling**, which operates on the audio waveform, not bytes. [Part 8]({{ '/blog/tutorials/crypto-lab-08-analog-voice-descrambling/' | relative_url }}) covers the `descramble` tool's three modes (inversion, split-band, rolling code) and how analog voice privacy differs fundamentally from digital encryption. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) list the full recipe op set.

## FAQ

**Why does `crc recover` sometimes return several candidate parameter sets?**
Because your samples don't distinguish them — different init/xorout combinations can produce the same checks over frames of identical length. Add sample frames of *differing* length and the ambiguous sets collapse.

**Can a recipe decrypt and then check whether the decrypt worked?**
Yes — that's the point of mixing transforms and analyses. Put a cipher decrypt transform, then a `stats` or `randomness` analysis after it; a drop in entropy or a failing randomness check confirms real plaintext emerged.

**What's the difference between a transform and an analysis op?**
A transform rewrites the working buffer (the bytes that flow to the next step); an analysis reads the buffer and reports on it without changing it. Analyses are how you measure progress mid-pipeline.

**Are recipe files JSON or YAML?**
Either — JSON is valid YAML, so one parser handles both. Use a top-level list of steps, or an object with a `steps:` key.

## Series navigation

**Part 7 of 10** · ←[Part 6: The `assess` Battery]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }}) · Next →[Part 8: Analog Voice Descrambling]({{ '/blog/tutorials/crypto-lab-08-analog-voice-descrambling/' | relative_url }})
