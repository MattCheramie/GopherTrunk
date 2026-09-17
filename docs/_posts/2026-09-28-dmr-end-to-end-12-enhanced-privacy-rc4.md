---
title: "DMR End to End, Part 12: Enhanced Privacy — RC4 & the IV That Names the Next Superframe"
description: "How GopherTrunk decrypts DMR Enhanced Privacy with an operator-held key: the Privacy Indicator header, the RC4 key‖MI keystream with its 256-byte warm-up, the 32-bit LFSR that advances the Message Indicator, the embedded IV that names the NEXT superframe, and the capture that settled it."
category: deep-dives
keywords: dmr enhanced privacy, dmr rc4 decryption, privacy indicator header, dmr message indicator, embedded iv superframe, arc4 known key dmr, dmr encryption keys config, p25 adp rc4, pitch continuity verdict, gophertrunk dmr encryption
tags: [dmr-end-to-end, dmr, encryption, rc4, voice, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "DMR End to End"
series_part: 12
---

*Part 12 of **DMR End to End**, a 14-part deep dive that follows the world's
most widely deployed digital PMR protocol through GopherTrunk — from a 4FSK
carrier to two simultaneous recorded calls, direct-mode handhelds, and
decrypted Enhanced Privacy voice.
[Part 11]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }})
ended with clear AMBE+2 frames reaching the vocoder. This part is about the
frames that arrive scrambled: where a transmission announces its cipher, how
an RC4 keystream is laid onto eighteen vocoder frames, and the convention —
the embedded IV names the *next* superframe — that no round-trip could have
caught and a reporter's known-key capture did.*

> **TL;DR:** DMR "Enhanced Privacy" (DMRA algorithm `0x21`) is RC4 keyed
> with **key‖MI** — the operator's key followed by the 32-bit Message
> Indicator — with **256** keystream bytes discarded and **7 bytes per
> 49-bit AMBE+2 frame** applied across a superframe (`EPKeystream`,
> `DescrambleSuperframe` in `internal/radio/dmr/voice/ep.go`). The MI
> arrives in the **Privacy Indicator header** (`dmr.ParsePIHeader`, CRC mask
> `0x9696`) and advances per superframe through the LFSR x³²+x⁴+x²+1
> (`AdvanceMI`). Every superframe also embeds an IV as three Golay(24,12)
> codewords — and that IV is the **NEXT** superframe's MI. `EPTracker`
> applied it to the current one, so everything after the first superframe
> decoded on the wrong MI; that fix plus `RewindMI` for late entry lifted the
> [#1187](https://github.com/MattCheramie/GopherTrunk/issues/1187) captures
> from ciphertext (b0 continuity ≈0.15) to speech (0.77 / 0.74).
> Capture-verified; the live daemon call is still the open gate.

**Key takeaways**

- **Encryption is announced in a data burst, then hidden in the voice.** The
  PI header carries algorithm, key ID and MI at keyup; afterwards the voice
  bursts carry the chain, a nibble per frame.
- **The keystream construction was right the whole time.** Both defects that
  hid a working decrypt were receiving-side: an IV applied one superframe too
  early, and a harness slicing bursts out of the gaps.
- **A Golay-verified IV is weak evidence on its own.** The radius-3 sphere
  covers ~57 % of 24-bit words, so clear vocoder bits "verify" ~1 % of the
  time — a disagreeing IV is trusted only when it decoded clean.
- **Judge a decrypt by pitch continuity, never loudness.** Random AMBE+2
  parameters are loud; only a slowly moving fundamental says the key was
  right.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Privacy Indicator header | alg / FID / key ID / 32-bit MI / dest, CRC-CCITT ^ `0x9696` | `internal/radio/dmr/pi.go` (`ParsePIHeader`, `IsRC4`) |
| Finding the header burst | data-sync slice → slot type `0x0` → BPTC → CRC | `dmr/voice/pi_detector.go` (`PIHeaderDetector`) |
| Keystream | RC4(key‖MI), drop 256, 7 bytes/frame, silence frames in clear | `dmr/voice/ep.go` (`EPKeystream`, `DescrambleSuperframe`) |
| MI schedule | x³²+x⁴+x²+1, one step per superframe, invertible | `AdvanceMI` / `RewindMI` |
| Embedded IV | 18 nibbles at on-air bits 71/67/63/59 → 3 × Golay(24,12) + CRC-4 | `ExtractEmbeddedIV`, `EPTracker.Next` |
| Reporter harness | `GT_DMR_EP_IQ` / `GT_DMR_EP_AUDIO` + `GT_DMR_EP_KEY`, pitch-continuity `VERDICT` | `cmd/gophertrunk/dmr_ep_replay_test.go` |

## In this post

- **The header that says how** — twelve octets and a CRC mask.
- **Key‖MI, drop 256, seven bytes a frame** — the RC4 construction.
- **The IV that names the next superframe** — embedded MIs, the tracker, the rewind.
- **Composer to recorder** — key resolution, events, the crypto-frame bridge.
- **The capture that settled it** — two receiving-side defects, one verdict.

## The header that says how

> **Authorized use only.** Everything here is *known-key* decryption: the
> operator supplies a key they are authorized to hold, and GopherTrunk
> performs no key recovery of any kind — the same model as SDRTrunk, DSD-FME
> and OP25. Decrypting traffic you are not permitted to touch is illegal in
> most jurisdictions; the
> [Crypto Lab]({{ '/blog/tutorials/crypto-lab-01-breaking-it-is-the-test/' | relative_url }})
> framing applies unchanged — measure your own deployment, or one you have
> written permission to assess.

P25 announces encryption twice —
[a grant bit and an in-call Encryption Sync]({{ '/blog/deep-dives/p25-end-to-end-09-encryption/' | relative_url }}).
DMR is similar: the Full Link Control's service options carry a privacy
bit ([Part 5]({{ '/blog/deep-dives/dmr-end-to-end-05-link-control-late-entry/' | relative_url }})),
and the *how* rides a dedicated data burst — the **Privacy Indicator
header**, slot type `0x0` (`DTPIHeader`), sent after the Voice LC Header. Its BPTC(196,96)-recovered block is twelve octets, pinned against
SDRTrunk and DSD-FME: algorithm
(`0x21` RC4, `0x22` DES, `0x24`/`0x25` AES), feature-set ID (`0x10` DMRA,
`0x68` Hytera, `0x0A` Kirisun), key ID, the 32-bit MI in octets 3–6, a
24-bit destination, and a CRC-CCITT (poly `0x1021`, init 0) XORed with the
PI-header mask.

That mask costs newcomers an afternoon. ETSI writes it as `0x6969` but
defines the CRC with its output inverted; `framing.CRCCCITTWithInit` is the
plain form, so `piHeaderCRCMask` is `^0x6969 = 0x9696` — the relation
`csbkCRCMask` (`0x5A5A`) bears to ETSI's `0xA5A5` in
[Part 4]({{ '/blog/deep-dives/dmr-end-to-end-04-fec-stack-forged-terminator/' | relative_url }}).
`TestParsePIHeaderReferenceLiteral` pins it with an independent Python CRC's
vector — the
[literal-vector rule]({{ '/blog/deep-dives/from-spec-to-shipping-03-literal-vectors/' | relative_url }}),
since a round-trip through `AssemblePIHeader` agrees with either mask.

Two choices are deliberate. `IsRC4` keys on the **low three bits** of the
algorithm octet, as DSD-FME does — but only under feature set `0x10`:
Hytera's Enhanced Privacy (FID `0x68`, 40-bit MI) is a different
construction and is not claimed. And the header has no RS(12,9)
trailer, so its CRC is the only integrity check: a BPTC-clean block that
fails it is dropped, never guessed, and `PIHeaderDetector.Rejects` keeps its
hex as the instrument for an unnamed vendor layout. The detector runs beside
the voice `Decoder` exactly like the terminator detector: data-sync slice,
both polarities, slot type, BPTC, `ParsePIHeader`.

## Key‖MI, drop 256, seven bytes a frame

The construction is short, and every line is a protocol fact two open
decoders agree on.
`EPKeystream` appends the four MI bytes (MSB first) to the key, seeds Go's
`rc4.NewCipher`, discards `EPKeystreamDrop` = 256 bytes and returns the
next *n*; `AdvanceMI` clocks a 32-bit LFSR with feedback
`(l>>31)^(l>>3)^(l>>1)` thirty-two times — x³²+x⁴+x²+1.

`DescrambleSuperframe` XORs those 126 bytes onto the eighteen FEC-decoded
payloads **in place**. Two rules keep alignment honest: a frame that failed
FEC (`nil`) is skipped but **still consumes its 7-byte slot**, and a frame
carrying the AMBE+2 **silence vector** (`0xF801A99F8CE080`,
[Part 11]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }})'s
subject) is sent in clear and passes through untouched while consuming its
slot — DSD-FME's silence guard.

The kinship with P25 is close: `internal/radio/p25/phase1/adp.go` keys RC4
with the key plus the first **8** octets of P25's 72-bit MI, drops 256, and
lays the stream onto IMBE frames from absolute byte 267 (LDU1) and 368
(LDU2) — OP25's layout, verified on the same reporter's ADP call (pitch
track 0.13 → 0.57). One cipher, one warm-up, and — next — one convention.

## The IV that names the next superframe

If the header were the only MI source, a scanner tuning in late — grant,
retune, DDC settle, frame lock — would never decrypt anything. So every
superframe also carries an MI: each of the 18
on-air frames donates one nibble in its unprotected C3 sub-vector — bits
71, 67, 63, 59, `C3[0..3]` after deinterleave. The 72 bits form three
[Golay(24,12)]({{ '/reference/golay-code/' | relative_url }}) codewords —
codeword *j* takes data from bursts A, B, C and parity from D, E, F — whose
36 data bits are the 32-bit MI plus a CRC-4 (x⁴+x+1, output inverted). That
is `ExtractEmbeddedIV`.

Now the convention. **The embedded IV is the NEXT superframe's MI** — the
[P25 LDU2 Encryption Sync]({{ '/reference/p25-encryption-sync/' | relative_url }})
rule. On the #1187 radio every superframe's IV equals `AdvanceMI` of the one
before, and the first equals `AdvanceMI` of the header's; DSD-FME agrees
(`dmr_alg_refresh` LFSR-advances at burst F *before* `dmr_late_entry_mi`
compares).

<figure class="lab-figure">
<svg viewBox="0 0 680 190" width="680" height="190" role="img" aria-label="A Privacy Indicator header announces MI zero; two voice superframes each decrypt with the MI already known for them while their embedded IV names the next superframe's MI, AdvanceMI arrows link successive MIs, and a dashed RewindMI arrow runs back from the first embedded IV to MI zero for a late entrant that missed the header.">
  <g fill="none" stroke="currentColor">
    <rect x="20" y="70" width="110" height="54" rx="6"/>
    <rect x="190" y="60" width="190" height="74" rx="6"/>
    <rect x="440" y="60" width="190" height="74" rx="6"/>
    <line x1="130" y1="97" x2="184" y2="97"/><line x1="380" y1="97" x2="434" y2="97"/>
  </g>
  <g fill="none" stroke="var(--accent)">
    <path d="M 75 70 L 75 40 L 285 40 L 285 60"/>
    <path d="M 285 60 L 285 22 L 535 22 L 535 60"/>
  </g>
  <path d="M 285 134 L 285 160 L 75 160 L 75 124" fill="none" stroke="var(--fg-muted)" stroke-dasharray="4 3"/>
  <g fill="currentColor"><polygon points="182,93 190,97 182,101"/><polygon points="432,93 440,97 432,101"/></g>
  <g fill="var(--accent)"><polygon points="281,54 285,62 289,54"/><polygon points="531,54 535,62 539,54"/></g>
  <polygon points="71,130 75,122 79,130" fill="var(--fg-muted)"/>
  <g text-anchor="middle" font-size="10" fill="currentColor" font-weight="bold">
    <text x="75" y="92">PI header</text>
    <text x="285" y="80">superframe 0</text>
    <text x="535" y="80">superframe 1</text>
  </g>
  <g text-anchor="middle" font-size="9" fill="currentColor">
    <text x="285" y="98">decrypt with MI₀</text>
    <text x="535" y="98">decrypt with MI₁</text>
  </g>
  <g text-anchor="middle" font-size="9" fill="var(--accent)">
    <text x="75" y="112">MI₀ (alg 0x21, key ID)</text>
    <text x="285" y="124">embedded IV = MI₁ (3 × Golay(24,12) + CRC-4)</text>
    <text x="535" y="124">embedded IV = MI₂</text>
    <text x="180" y="34">AdvanceMI(MI₀) = MI₁</text>
    <text x="410" y="16">AdvanceMI(MI₁) = MI₂</text>
  </g>
  <g text-anchor="middle" font-size="9" fill="var(--fg-muted)">
    <text x="285" y="112">FEC-failed frame keeps its 7 bytes</text>
    <text x="180" y="174">late entry, header missed: RewindMI(IV) = MI₀</text>
  </g>
</svg>
<figcaption>Each superframe decrypts on what was already known for it; its embedded IV only steers what comes next — and a late entrant rewinds the LFSR once to recover the superframe it is standing in.</figcaption>
</figure>

`EPTracker.Next` is the rule as a switch. A verified IV equal to
`AdvanceMI(next)` confirms the chain; with no chain at all, a verified IV
yields the current MI by `RewindMI`, so late entry decrypts the very
superframe it verified on. A verified IV that *disagrees*
is where the Golay gate matters: a radius-3 sphere covers ~57 % of all
24-bit words, so clear vocoder bits pass all three codewords plus the CRC-4
about **1 %** of the time, whereas three *clean* codewords plus CRC are
~2⁻⁴⁰ by chance. A clean disagreement means a missed superframe and the MI
is re-derived from it; a corrected one holds the prediction (`Mismatches`
counts both).

## Composer to recorder

The per-call state (`dmrEPState`, `composer/dmr_ep.go`) never guesses a
key and never acts on an embedded IV for a call with no other encryption
evidence unless that IV decoded perfectly clean. `onPIHeader` seeds the
tracker (`SetHeaderMI`),
resolves the key through `KeyResolver` when `IsRC4`, and publishes
`events.KindCallEncryption` with `Protocol: "dmr"`; `superframe` runs
`ExtractEmbeddedIV` → `EPTracker.Next` → `DescrambleSuperframe` **in
place**.

`KeyResolver` is built from `trunking.systems[].encryption_keys`
(`buildKeyResolver`, logging `daemon: in-process decryption keys configured
(DMR enhanced privacy / P25 ADP)`); `algorithm` must be `rc4` / `arc4` /
`adp` (DES and AES fail at load with "not supported yet"), `key` is 1–32 hex
bytes, and a key under a different ID is never applied. The header lands at
INFO as `composer: dmr PI header — call is encrypted`, and the counters —
`pi_headers`, `iv_verified`, `mi_predicted`, `mi_mismatches`,
`descrambled_frames`, `no_key` — ride the debug `composer: dmr enhanced
privacy` line.

The event is the same `KindCallEncryption` P25 publishes, so the engine's
[follow / metadata / ignore policy]({{ '/blog/deep-dives/trunking-engine-11-encrypted-mode/' | relative_url }})
and its configured-key exemption apply unchanged, and
`recordings.skip_encrypted`
([Recording & Streaming Part 7]({{ '/blog/deep-dives/recording-streaming-07-correctness-guards/' | relative_url }}))
still decides whether a keyless call reaches disk. Independently of any key,
`recordings.crypto_capture_path` hands every superframe's **ciphertext plus
its MI** to the `cryptocap` bridge for the
[keystream-reuse workflow]({{ '/blog/tutorials/crypto-lab-05-keystream-reuse-mtp/' | relative_url }}).
`TestComposerDMREnhancedPrivacyDecryptsWithConfiguredKey` runs the full
composer and requires the recorder's `.raw` sidecar to hold **clear**
payloads.

## The capture that settled it

The #1187 reporter posted two known-key discriminator captures — key IDs 11
and 22, colour code 2, TG 582743, simplex — and
`TestDMREnhancedPrivacyReplay` judged them: production DDC → `dmrrx`
receiver → `PIHeaderDetector` + superframe decoder → `EPTracker` →
`DescrambleSuperframe` → `ambe2-dmr`, printing every header, the IV/MI
timeline and a one-line `VERDICT`. Knobs: `GT_DMR_EP_IQ` or
`GT_DMR_EP_AUDIO` (a discriminator WAV, FM re-modulated), `GT_DMR_EP_KEY`,
`GT_DMR_EP_KEYID`, `GT_DMR_EP_OUT`, and `GT_DMR_EP_DUMP`, which writes every
superframe *before* descrambling so a keystream hypothesis is testable
offline.

Both runs failed at first, for **opposite reasons**; neither was the cipher. First, the harness sliced voice with the back-to-back 132-dibit
single-slot decoder; a simplex handheld sends one burst per 60 ms frame,
288 dibits apart
([Part 3]({{ '/blog/deep-dives/dmr-end-to-end-03-two-slots-one-carrier/' | relative_url }})),
so bursts B–F were read out of the gaps: burst A decoded FEC-clean, B–F
carried the random-word Golay signature — **73 % needing exactly 3
corrections, 12 % exactly 2** — which looks exactly like post-FEC
scrambling and is not — the same defect
[#1192](https://github.com/MattCheramie/GopherTrunk/issues/1192) had fixed
in production the day before. The harness now defaults to the
cadence-detecting decoder, and a C0/C1 Golay histogram is the *first*
instrument to read when frames look scrambled. Second, the IV convention:
correctly sliced frames decoded on the wrong MI while the wrongly sliced ones
had decoded on the right one.

The verdict metric matters because loudness would have lied: random AMBE+2
parameters through the vocoder are **loud**, so the harness reports RMS but
never trusts it. It measures **pitch continuity**, the fraction of
consecutive frame pairs whose fundamental index moves by at most 10: speech
≳0.5, ciphertext ≈0.15. Post-fix the captures score **0.77 and 0.74**.
`TestEPCaptureIssue1187` pins it against literal on-air frames
(`voice/testdata/ep_issue1187_ptt1.json`): the header→IV chain yields
continuity ≥ 0.55, late entry rewinds to the header's MI, and decrypting each
superframe with its *own* IV stays ≤ 0.35 — the old semantics fail it.

Verified-language, as the repo states it: Enhanced Privacy is
**capture-verified**. Open on #1187 is the **daemon path on air** — a live
call with `encryption_keys` configured recording intelligible audio (the
#764/#771 discipline). Not decoded: Hytera EP, Kirisun, DES/AES; the
[DMR encryption reference]({{ '/reference/dmr-encryption/' | relative_url }})
and [RC4 page]({{ '/reference/rc4-cipher/' | relative_url }}) cover the
landscape, and
[`docs/dmr-encryption.md`]({{ '/dmr-encryption.html' | relative_url }})
has the capture-contribution recipe.

### How the IV convention shaped the Go code

- **The tracker decodes on what it already knows.** `Next` returns the
  *current* MI from the header, the previous IV or the prediction, and lets
  the embedded IV steer only `next` — the convention lives in control flow.
- **Verification strength is a parameter, not a boolean.** `ExtractEmbeddedIV`
  returns `corrected` beside `ok`; a clean triple and a ~1 % false verify are
  treated differently.
- **Instruments outlive the fix.** `Mismatches`, `PIHeaderDetector.Rejects`
  and `GT_DMR_EP_DUMP` pin the next vendor variant.

## Where this goes next

Twice here a receiving-side defect masqueraded as a cipher problem, and
instruments caught both.
[Part 13]({{ '/blog/deep-dives/dmr-end-to-end-13-testing-dmr/' | relative_url }})
makes that the subject: testing DMR without a repeater — committed real-air
slices, the replay harnesses and their scrubs, and the fixture lessons that
hid direct mode.

## FAQ

**Can GopherTrunk decrypt DMR Enhanced Privacy?**
Yes, with a key the operator supplies under
`trunking.systems[].encryption_keys` whose `key_id` matches the Privacy
Indicator header. The voice chain descrambles RC4 "Enhanced Privacy" in
process and records clear audio; without a key the call is recorded as
ciphertext with algorithm and key ID surfaced.

**Where does a DMR call carry its Message Indicator?**
Twice: in the Privacy Indicator header sent after the Voice LC Header at
keyup (32 bits, octets 3–6), and embedded in every voice superframe as 18
nibbles forming three Golay(24,12) codewords plus a CRC-4 — the late-entry
copy, which names the *next* superframe's MI, not the one carrying it.

**Why did a working keystream produce garbage at first?**
Two receiving-side defects. The tracker applied each embedded IV to its own
superframe instead of the next, so every superframe after the first used the
wrong MI; and the replay harness sliced a simplex handheld's bursts with the
back-to-back single-slot decoder, reading bursts B–F out of the gaps.

**How do I know a decrypt actually worked?**
Not by ear or level — random vocoder parameters are loud. The harness
measures pitch continuity, the fraction of consecutive AMBE+2 frames whose
fundamental index (bits 0..3 and 37..39) moves by at most 10: speech ≳0.5,
ciphertext ≈0.15. The #1187 captures score 0.77 and 0.74.

**Is DMR Enhanced Privacy verified on air in GopherTrunk?**
Capture-verified: two known-key captures decode to speech through
`TestDMREnhancedPrivacyReplay`, with literal on-air frames pinned in
`TestEPCaptureIssue1187`. The remaining gate on #1187 is a live daemon call
with `encryption_keys` configured that records intelligible audio.

## Series navigation

**Part 12 of 14** · ←
[Part 11: AMBE+2 Voice & the Silence-Frame Bug]({{ '/blog/deep-dives/dmr-end-to-end-11-ambe2-silence-frames/' | relative_url }})
· Next →
[Part 13: Testing DMR Without a Repeater]({{ '/blog/deep-dives/dmr-end-to-end-13-testing-dmr/' | relative_url }})
