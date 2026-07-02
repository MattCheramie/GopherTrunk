---
title: "Crypto Lab, Part 8: Analog Voice Descrambling — Inversion, Split-Band, Rolling Code"
description: Crypto Lab's descramble tool undoes analog voice privacy — frequency inversion, split-band inversion, and rolling-code hopping — operating on raw 16-bit PCM audio, and shows why analog scrambling is fundamentally weaker than digital encryption.
category: tutorials
keywords: analog voice scrambling, frequency inversion, split band inversion, rolling code descrambler, spectrum inversion, voice privacy, pcm audio, self inverse, gophertrunk cryptolab
tags: [cryptolab, descramble, analog-voice, security-testing, advanced, rf]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "Crypto Lab"
series_part: 8
---

*Part 8 of **Crypto Lab**, a 10-part series on GopherTrunk's optional cryptographic-research toolkit. Not all RF privacy is digital — some is a trick played on the audio spectrum, and the descrambler undoes it.*

> **TL;DR:** `cryptolab descramble` has three modes for analog voice privacy, all operating on raw **16-bit signed little-endian mono PCM**: `invert` (full-band frequency inversion — run it twice to undo), `splitband` (`-split` fraction of Nyquist, disjoint-bin FFT, self-inverse for a given split), and `rolling` (`-frame` samples with a `-schedule` of per-frame splits, or `-schedule auto` to detect inverted frames). Analog scrambling reorders the *spectrum*; it doesn't encrypt bits, which is why it's far weaker than digital encryption.

> **Authorized testing only.** Descramble audio only from systems you own or are licensed to assess.

**Key takeaways**

- **Analog scrambling is spectral, not cryptographic.** It rearranges the audio spectrum, it doesn't encrypt data.
- **Inversion is self-inverse.** Applying the same operation again undoes it — no key needed.
- **Split-band adds a knob, not real security.** One `-split` fraction is the whole "key."
- **Rolling code hops the split** per frame — `auto` detects it from spectral energy; a known schedule replays it exactly.

## Cheat sheet

| Command / flag | What it does |
|---|---|
| `cryptolab descramble invert -in s.s16 -out c.s16` | Full-band spectral inversion (run twice to undo) |
| `cryptolab descramble splitband -in s.s16 -out c.s16 -split 0.5` | Invert low/high sub-bands about a split point |
| `cryptolab descramble rolling -in s.s16 -out c.s16 -frame 1024 -schedule auto` | Per-frame rolling inversion, auto-detected |
| `-split F` | Split point as a fraction of Nyquist (0..1) |
| `-frame N` | Frame length in samples (rolling) |
| `-schedule auto\|F1,F2,...` | Detect inverted frames, or replay a known split schedule |

## In this post

- **How analog voice privacy works** — and why it's not encryption.
- **`invert`** — full-band frequency inversion and the self-inverse trick.
- **`splitband`** — inverting sub-bands independently.
- **`rolling`** — hopping the inversion per frame, auto or scheduled.
- **The PCM contract** — what the tool reads and writes.

## Analog privacy is a spectral trick

Before digital encryption, radios kept voice "private" by scrambling the *audio spectrum*. The most common method, **frequency inversion**, flips the audio band end-for-end: low frequencies become high and high become low, about a center point. The result sounds like unintelligible Donald-Duck garble on a scanner — but nothing is encrypted. There's no key in any cryptographic sense, no keystream, no ciphertext. The information is all still there; it's just been folded over in frequency.

That's the crucial distinction Reese hammers home: **analog scrambling reorders, it doesn't encrypt.** A digital cipher transforms bits so that without the key the output is indistinguishable from random. A frequency inverter just moves energy around the spectrum in a way anyone can reverse by moving it back. The entire "secret" is *which* spectral rearrangement was used — and there are very few of them. This is why analog voice privacy is a museum piece from a security standpoint, and why the descrambler needs no key input at all: undoing the rearrangement is a deterministic signal-processing operation.

It helps to put a number on "very few." A full-band inverter has essentially *one* configuration — invert or don't. A split-band inverter's only parameter is the split fraction, and in practice deployments cluster on a handful of round values, so a few trials exhaust the space. Even a rolling-code scrambler, which sounds sophisticated, hops among a small set of split points on a repeating schedule — a keyspace measured in dozens of states, not the 2¹²⁸ of a real cipher. Contrast that with the digital ciphers in [Part 6]({{ '/blog/tutorials/crypto-lab-06-assess-battery/' | relative_url }}), where a properly-keyed AES system grades <span class="lab-verdict lab-verdict--ok">RESISTANT</span> because its keyspace is genuinely infeasible to search. Analog scrambling never had that protection; it relied entirely on the listener not bothering to reverse a known transform. Once you bother — which is exactly what `descramble` does — the privacy evaporates.

<figure class="lab-figure">
<svg viewBox="0 0 640 170" width="640" height="170" role="img" aria-label="Frequency inversion flips the audio spectrum end for end about a center point">
  <g font-size="11">
  <text x="10" y="20" fill="var(--fg-muted)">clear speech spectrum (energy sloping down with frequency)</text>
  <line x1="20" y1="70" x2="300" y2="70" stroke="currentColor"/>
  <polygon points="30,68 30,40 40,42 40,68" fill="currentColor" opacity="0.85"/>
  <polygon points="45,68 45,48 55,50 55,68" fill="currentColor" opacity="0.7"/>
  <polygon points="60,68 60,56 70,57 70,68" fill="currentColor" opacity="0.55"/>
  <polygon points="75,68 75,60 85,61 85,68" fill="currentColor" opacity="0.4"/>
  <polygon points="90,68 90,63 100,64 100,68" fill="currentColor" opacity="0.3"/>
  <text x="20" y="86" fill="var(--fg-muted)" font-size="9">low</text>
  <text x="280" y="86" fill="var(--fg-muted)" font-size="9">high</text>
  <line x1="320" y1="55" x2="360" y2="55" stroke="var(--accent)" stroke-width="1.5"/>
  <polygon points="360,55 352,51 352,59" fill="var(--accent)"/>
  <text x="340" y="45" text-anchor="middle" fill="var(--accent)" font-size="10">invert</text>
  <text x="380" y="20" fill="var(--fg-muted)">inverted: low ↔ high, garbled to the ear</text>
  <line x1="380" y1="70" x2="620" y2="70" stroke="currentColor"/>
  <polygon points="390,68 390,63 400,64 400,68" fill="currentColor" opacity="0.3"/>
  <polygon points="405,68 405,60 415,61 415,68" fill="currentColor" opacity="0.4"/>
  <polygon points="420,68 420,56 430,57 430,68" fill="currentColor" opacity="0.55"/>
  <polygon points="435,68 435,48 445,50 445,68" fill="currentColor" opacity="0.7"/>
  <polygon points="450,68 450,40 460,42 460,68" fill="currentColor" opacity="0.85"/>
  <text x="20" y="130" fill="var(--fg-muted)">self-inverse: invert again → back to clear</text>
  <line x1="230" y1="126" x2="270" y2="126" stroke="currentColor" stroke-width="1.5"/>
  <polygon points="270,126 262,122 262,130" fill="currentColor"/>
  <text x="300" y="130" fill="var(--accent)" font-size="11">descramble invert (twice = identity)</text>
  </g>
</svg>
<figcaption>Frequency inversion flips the spectrum end-for-end. It is its own inverse: applying it a second time restores the original audio.</figcaption>
</figure>

## `invert`: full-band inversion

The simplest mode flips the whole band:

```bash
gophertrunk cryptolab descramble invert -in scrambled.s16 -out clear.s16
```

```text
descramble/invert — spectrally inverted 48000 samples -> clear.s16
  samples  48000
note: spectral inversion is its own inverse: run again to undo.
```

The key property is right there in the note: **spectral inversion is its own inverse.** The scrambler inverted the spectrum to garble it; you invert it again to restore it. There's no separate "encrypt" and "decrypt" — it's the same operation both directions, which is the clearest possible demonstration that no real secret is involved. Both `-in` and `-out` are required.

## `splitband`: inverting sub-bands

A slightly fancier scrambler splits the band at some point and inverts the low and high halves *independently*. `splitband` undoes it, using a disjoint-bin FFT so the operation stays exactly self-inverse for a given split:

```bash
gophertrunk cryptolab descramble splitband -in scrambled.s16 -out clear.s16 -split 0.5
```

The `-split` flag is the split point as a fraction of Nyquist (0..1); `0.5` splits the band in the middle. This split fraction is the entire "key" for a split-band scrambler — and the note reminds you it's self-inverse: re-run with the *same* `-split` to undo. In practice you recover the split by trying a few values and listening (or checking the output's spectral balance) for intelligible speech. A handful of candidate splits covers essentially every deployment, which again underlines how little security is on offer.

## `rolling`: hopping the split per frame

The most elaborate analog scramblers change the inversion **per frame** on a hopping schedule, so the split point moves over time. `descramble rolling` handles this two ways. With a known schedule, you replay the exact hop sequence:

```bash
gophertrunk cryptolab descramble rolling -in scrambled.s16 -out clear.s16 \
    -frame 1024 -schedule 0.5,0.4,0.6,0.5
```

`-frame` sets the frame length in samples, and `-schedule` is a comma-separated list of split fractions applied frame by frame (cycling through the list). Re-running with the same `-frame` and `-schedule` undoes a known rolling inverter exactly.

When you *don't* know the schedule, `-schedule auto` detects it:

```bash
gophertrunk cryptolab descramble rolling -in scrambled.s16 -out clear.s16 \
    -frame 1024 -schedule auto
```

```text
descramble/rolling — auto-descrambled 47 frames (29 inverted) -> clear.s16
  frames  47
  frames_inverted  29
note: auto mode is a heuristic full-band inversion detector (energy balance);
  a known schedule is exact.
```

Auto mode is a heuristic: for each frame it decides whether that frame was inverted by looking at its spectral energy balance, and flips the ones that look inverted. The tool is honest that this is a heuristic full-band detector — good for a first pass, but a *known* schedule is exact. It reports how many of the frames it judged inverted, which is a useful sanity check: if it flips nearly all or nearly none, your `-frame` size is probably wrong.

Ada's first rolling capture came out garbled at `-frame 512` — the frame size didn't align with the scrambler's hop period. Bumping to `-frame 1024` snapped the energy-balance detector into agreement and the speech came through. Reese's note: **the frame size is the parameter that matters most in rolling mode** — get it aligned to the scrambler's hop and auto-detection usually does the rest.

## The PCM contract

Every `descramble` mode reads and writes the same format: **raw 16-bit signed little-endian mono PCM** (`.s16`), samples normalized internally to [-1, 1). The input length must be a whole number of 16-bit samples, and the tool errors clearly if it isn't. Because it's raw PCM, you produce these files from your decoder's audio path or a recording, and you can pipe the descrambled output straight into any audio tool to listen. Note that this is *not* the byte-payload world of the rest of the toolkit — which is exactly why `classify auto` ([Part 2]({{ '/blog/tutorials/crypto-lab-02-first-triage-classify-stats/' | relative_url }})) explicitly declines to triage audio and points you here. The same three descramblers are also available as `recipe` ops (`descramble-invert`, `descramble-splitband`, `descramble-rolling`, [Part 7]({{ '/blog/tutorials/crypto-lab-07-crc-recovery-recipe/' | relative_url }})), so an inversion step can be chained inline with byte transforms.

## Where this goes next

We've now covered the whole CLI toolkit. [Part 9]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }}) steps back to the surfaces *around* it: the browser console that renders a form for every tool automatically, the live-capture bridge that feeds `ks` and `assess` straight from a running decoder, and the external-cipher mechanism — with its deliberate CLI-only safety boundary. The [cryptolab docs]({{ '/cryptolab.html' | relative_url }}) detail the descramble internals.

## FAQ

**Why doesn't `descramble` take a key?**
Because analog scrambling isn't encryption — there's no cryptographic key. The "secret" is just which spectral rearrangement was used (full-band, a split fraction, a hop schedule), and undoing it is a deterministic DSP operation, not a decryption.

**What does "self-inverse" mean here?**
The scrambling and descrambling operations are identical. Frequency inversion applied twice returns the original audio; split-band inversion with the same `-split` undoes itself. There's no separate decrypt step.

**How do I find the right `-split` or `-schedule`?**
Try a few candidate splits and check for intelligible speech or balanced output spectrum. For rolling code, get `-frame` aligned to the hop period first, then let `-schedule auto` detect the per-frame inversions.

**What audio format does it expect?**
Raw 16-bit signed little-endian mono PCM (`.s16`). The input must be a whole number of 16-bit samples; the tool errors if not.

## Series navigation

**Part 8 of 10** · ←[Part 7: CRC Recovery & the Recipe Pipeline]({{ '/blog/tutorials/crypto-lab-07-crc-recovery-recipe/' | relative_url }}) · Next →[Part 9: The Web Console, Live-Capture Bridge & External Ciphers]({{ '/blog/tutorials/crypto-lab-09-web-console-bridge-external/' | relative_url }})
