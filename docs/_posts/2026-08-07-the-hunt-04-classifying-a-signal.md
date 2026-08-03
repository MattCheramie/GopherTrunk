---
title: "The Hunt, Part 4: Classifying a Signal — Analog, Digital, Encrypted"
description: How GopherTrunk cheaply classifies a bare carrier into analog, digital, paging, or encrypted using envelope, discriminator, and cyclostationary features before spending an expensive protocol identify, and how the encryption triage tells structured traffic from random.
category: deep-dives
keywords: signal classification, modulation recognition, cyclostationary baud line, fm discriminator features, envelope coefficient of variation, encryption triage entropy, ctcss dcs scan, pocsag flex paging, gophertrunk the hunt
tags: [the-hunt, dsp, classification, modulation, survey, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "The Hunt"
series_part: 4
---

*Part 4 of **The Hunt**, a 14-part deep dive into how GopherTrunk finds trunked
systems you didn't know were there. [Part 3]({{ '/blog/deep-dives/the-hunt-03-peak-occupancy-detection/' | relative_url }})
turned a spectrum frame into a ranked list of carriers — frequencies with SNRs. A
sweep of the 851–869 MHz band can surface forty of them: analog repeaters, pagers,
noise, and, somewhere in the pile, our stray digital carrier. This part is the
coarse triage that decides which ones are even worth the expensive protocol
identify — and, for the carriers no decoder claims, whether they look encrypted.*

> **TL;DR:** Before a carrier earns a full protocol identify, a cheap **blind
> classifier** names its modulation family from three feature groups: the
> **envelope** coefficient of variation (AM vs constant-envelope), the **FM
> discriminator** statistics (deviation, impulsiveness, level structure), and a
> **cyclostationary baud line** in the rectified discriminator (the key-independent
> marker of digital symbols). Analog gets a CTCSS/DCS scan; a paging baud gets
> POCSAG/FLEX; a trunking-shaped digital carrier gets the identify. For an
> unidentified digital flow, an **entropy triage** flags "likely encrypted." It's
> a router, not a decoder — coarse on purpose.

**Key takeaways**

- **Digital is decided by a cyclostationary baud line, not by envelope.** A strong
  symbol-rate line in the rectified discriminator marks digital modulation and is
  checked *before* AM, because a pulse-shaped digital carrier has real envelope
  variation that would otherwise trip the AM gate.
- **Level structure splits the digital families.** Discriminator kurtosis catches
  phase modulation (PSK); folding at the symbol rate counts levels — 2 for FSK, 4
  for C4FM — the way a slicer sees them.
- **Encryption is a statistics verdict, not a decrypt.** A recovered payload with
  near-maximal Shannon entropy *and* near-uniform byte distribution reads as
  encrypted; structured traffic falls below the bar.
- **Everything is thresholds on measured features.** The `ClassFeatures` vector
  rides on every result, so a classification is debuggable and golden-testable.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| Feature vector | envelope, discriminator, baud, levels | `internal/survey/classify.go` (`ClassFeatures`) |
| Decision cascade | features → class + confidence | `internal/survey/classify.go` (`decide`) |
| Analog decode | squelch, CTCSS, DCS scan | `internal/survey/analog.go` (`AnalyzeAnalogFM`) |
| Paging decode | POCSAG/FLEX over a buffer | `internal/survey/pager.go` (`DecodePOCSAG`) |
| Encryption triage | entropy + index of coincidence | `internal/survey/encryption.go` (`EncryptionTriage`) |
| Algorithm name | ALGID → AES-256 / ADP / … | `internal/hunt/enctype.go` (`encTypeName`) |
| Identify gate | forward digital to siglab identify | `internal/hunt/decode.go` |

## In this post

- **Why classify before identify** — the cost argument for a cheap first pass.
- **The feature vector** — envelope, discriminator, baud line, level structure.
- **The decision cascade** — the order the branches fire, and why digital comes first.
- **Analog and paging** — what a non-trunked carrier gets instead of an identify.
- **The encryption triage** — telling random from structured without a key.

## Why classify before identify

A full protocol identify is expensive: it rewinds the capture, tries several
candidate protocols, runs each far enough to decide, and reports a confidence. Do
that on all forty carriers a sweep surfaced and most of the work is wasted on analog
repeaters and pagers that were never going to be a trunked control channel. So the
survey runs a *cheap, blind, per-carrier* classifier first. Its job is not to fully
demodulate — it's to **route**: name the modulation family well enough to send each
carrier to the right next step. The class enum is intentionally coarse:

```go
// internal/survey/classify.go (shape)
const (
    ClassUnknown        SignalClass = "unknown" // no carrier / too weak
    ClassAM             SignalClass = "am"
    ClassNBFM           SignalClass = "nbfm"    // narrowband analog FM
    ClassWideFM         SignalClass = "wfm"
    ClassFSK            SignalClass = "fsk"     // 2-level
    ClassC4FM           SignalClass = "c4fm"    // 4-level (P25 C4FM / YSF)
    ClassPSK            SignalClass = "psk"     // π/4-DQPSK and friends
    ClassContinuousData SignalClass = "data"
    ClassPaging         SignalClass = "paging"  // FSK at a paging baud
    // …ClassTrunkControl/Voice assigned by the router after a CC lock
)
```

Note the last comment: `ClassTrunkControl` and `ClassTrunkVoice` are *not* assigned
by the blind classifier — they're the router's verdict after the authoritative
siglab identify actually locks a control channel. The classifier says "this is
C4FM"; identify says "this is a P25 control channel." Coarse-then-fine, the same
pattern as detection-then-decode.

## The feature vector

Everything the classifier decides on is a small vector of DSP measurements,
computed once and carried on the result so the class is a *thresholded view* of
numbers you can inspect:

```go
// internal/survey/classify.go (shape) — ClassFeatures
type ClassFeatures struct {
    OccupiedBwHz   uint32  // contiguous bandwidth above the floor
    SNRDb          float64 // peak-to-floor of the baseband spectrum
    EnvelopeCV     float64 // std/mean of |z[n]| — AM is high, angle-mod ≈ 0
    IFStd          float64 // FM-discriminator std (rad/sample), a deviation proxy
    IFKurtosis     float64 // excess kurtosis — PSK is impulsive (spikes at transitions)
    BaudHz         float64 // cyclostationary symbol-rate line, or 0 for analog voice
    BaudProminence float64 // how far the baud line stands over the local median
    IFModality     int     // discriminator levels: 2 ⇒ FSK, 4 ⇒ C4FM, 0/1 ⇒ analog/PSK
}
```

Three physical intuitions drive these. **Envelope**: AM carries its message in
amplitude, so `|z[n]|` swings (high CV); a constant-envelope FM or digital-FM
carrier sits near zero CV. **Discriminator**: FM-demodulate and the output's
standard deviation is a deviation proxy, while its *kurtosis* is impulsive for phase
modulation (spikes at phase transitions, quiet between). **Cyclostationarity**: a
digital carrier's symbol transitions recur at the baud rate; rectify the
discriminator (`|d - mean|`) and that recurrence shows up as a discrete spectral
line, while analog voice produces only a smooth roll-off. `baudLine` finds that line
and reports its prominence over the band median.

The level count is the subtle one. You can't histogram the raw discriminator — a
pulse-shaped carrier smears its levels between samples. So `symbolModality` folds
the discriminator at the detected symbol rate, integrate-and-dump over each symbol
period the way a slicer sees it, *then* histograms and counts well-separated peaks:
2 for binary FSK, 4 for C4FM.

## The decision cascade

`decide` maps the vector to a class and a confidence, and the *order* of its
branches is load-bearing:

```go
// internal/survey/classify.go (shape) — decide
func decide(f ClassFeatures, cfg ClassifyConfig) (SignalClass, float64) {
    if f.OccupiedBwHz == 0 || f.SNRDb < cfg.SNRGateDb {
        return ClassUnknown, 0
    }
    // Digital FIRST: a strong cyclostationary baud line is the key-independent
    // marker of symbol transitions — checked before AM because a linearly-
    // modulated digital carrier has pulse-shaping envelope variation that would
    // otherwise trip the AM gate.
    if f.BaudHz > 0 && f.BaudProminence >= cfg.DigitalProminence {
        conf := 0.5 + 0.5*clamp01((f.BaudProminence-cfg.DigitalProminence)/30)
        if f.IFKurtosis > cfg.PSKKurtosis {
            return ClassPSK, conf
        }
        if f.IFModality >= 4 {
            return ClassC4FM, conf
        }
        return ClassFSK, conf
    }
    // AM: non-constant envelope AND little angle modulation (low IFStd).
    if f.EnvelopeCV > cfg.AMEnvelopeCV && f.IFStd < cfg.AMMaxIFStd {
        return ClassAM, 0.5 + 0.5*clamp01((f.EnvelopeCV-cfg.AMEnvelopeCV)/0.3)
    }
    // Analog FM: constant envelope, no baud line. Split by bandwidth.
    if f.OccupiedBwHz <= cfg.NBFMMaxBwHz {
        return ClassNBFM, 0.6
    }
    return ClassWideFM, 0.6
}
```

Digital is tested first because a π/4-DQPSK carrier has genuine envelope variation
from pulse shaping — enough to trip the AM gate if AM ran first. Within the digital
branch, kurtosis is checked before the 4-level test because π/4-DQPSK's phase-change
values *also* show four levels, so an impulsive discriminator (PSK) has to win
before the C4FM branch can. And the AM gate carries a hard-won subtlety: it requires
*low* `IFStd` as well as high envelope CV, because a DMR carrier whose fragile baud
line was lost to noise can have its envelope CV lifted past the gate — but its
discriminator still swings with the deviation, so the low-`IFStd` requirement keeps
it out of AM and sends it to the FM split instead (issue #648). The confidence isn't
a constant: each branch reports how *cleanly* its gate was met, so a marginal call
reads low and a clear one high.

<figure class="lab-figure">
<svg viewBox="0 0 680 220" width="680" height="220" role="img" aria-label="Classification decision cascade: a carrier is first gated on SNR; a strong baud line routes to the digital families split by kurtosis and level count into PSK, C4FM, or FSK; otherwise high envelope CV with low deviation is AM, and constant envelope splits into narrowband or wide FM">
  <rect x="20" y="90" width="96" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="68" y="108" text-anchor="middle" fill="currentColor" font-size="10">carrier</text>
  <text x="68" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">SNR gate</text>
  <line x1="116" y1="110" x2="146" y2="110" stroke="currentColor"/><polygon points="146,106 156,110 146,114" fill="currentColor"/>
  <rect x="156" y="90" width="120" height="40" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="216" y="108" text-anchor="middle" fill="var(--accent)" font-size="10">baud line?</text>
  <text x="216" y="122" text-anchor="middle" fill="var(--fg-muted)" font-size="9">cyclostationary</text>
  <line x1="276" y1="100" x2="330" y2="40" stroke="var(--accent)"/><polygon points="326,37 336,34 333,44" fill="var(--accent)"/>
  <text x="300" y="60" fill="var(--accent)" font-size="9">yes</text>
  <line x1="276" y1="120" x2="330" y2="165" stroke="currentColor"/><polygon points="326,159 336,168 324,168" fill="currentColor"/>
  <text x="300" y="150" fill="var(--fg-muted)" font-size="9">no</text>
  <rect x="336" y="18" width="150" height="44" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="411" y="34" text-anchor="middle" fill="var(--accent)" font-size="10">digital</text>
  <text x="411" y="50" text-anchor="middle" fill="var(--fg-muted)" font-size="9">kurtosis→PSK · levels→C4FM/FSK</text>
  <rect x="336" y="150" width="150" height="44" rx="6" fill="none" stroke="currentColor"/>
  <text x="411" y="166" text-anchor="middle" fill="currentColor" font-size="10">analog</text>
  <text x="411" y="182" text-anchor="middle" fill="var(--fg-muted)" font-size="9">envelope CV→AM · bw→NBFM/WFM</text>
  <line x1="486" y1="40" x2="540" y2="40" stroke="currentColor"/><polygon points="540,36 550,40 540,44" fill="currentColor"/>
  <rect x="550" y="20" width="110" height="40" rx="6" fill="none" stroke="currentColor"/>
  <text x="605" y="38" text-anchor="middle" fill="currentColor" font-size="10">identify</text>
  <text x="605" y="52" text-anchor="middle" fill="var(--fg-muted)" font-size="9">siglab / paging</text>
  <text x="340" y="212" text-anchor="middle" fill="var(--fg-muted)" font-size="10">coarse router: name the family cheaply, then spend the expensive identify only on the digital carriers</text>
</svg>
<figcaption>The classifier is a router. Digital is decided first by a baud line, split by kurtosis and level count; analog falls through to an envelope/bandwidth split. Only digital carriers earn the identify.</figcaption>
</figure>

### How that principle shaped the Go code

- **Thresholds are a struct, not constants.** `ClassifyConfig` exposes every gate,
  and `withDefaults` fills zero fields — so an operator can tune one threshold from
  a CLI flag for a noisy front end without recompiling, and the golden fixtures
  pin the defaults.
- **Occupied bandwidth can come from outside.** `ClassifyWith` accepts an
  `occBwHz`/`snrDb` measured on a wider, un-decimated view, so a wideband signal
  isn't mis-measured by the narrow channel the classifier IQ was decimated to.
- **The features survive the verdict.** Because `ClassFeatures` rides on the
  `Classification`, a surprising call ("why did this read as AM?") is answered by
  reading the numbers, not by re-running the DSP.

## Analog and paging: what a non-trunked carrier gets

A carrier that classifies analog isn't discarded — it's decoded the cheap way.
`AnalyzeAnalogFM` checks carrier-present power against a squelch, then blind-scans
the 38-tone EIA CTCSS set and the common DCS codes, one Goertzel each, to identify
the repeater's sub-audible squelch:

```go
// internal/survey/analog.go (shape) — AnalyzeAnalogFM
rep := &AnalogReport{PowerDbFS: conventional.PowerDbFS(iq)}
rep.Active = rep.PowerDbFS >= analogSquelchDbFS
if !rep.Active {
    return rep
}
rep.CTCSSHz = scanCTCSS(iq, float64(inputRateHz)) // first tone that locks
rep.DCSCode = scanDCS(iq, float64(inputRateHz))    // first code that locks
```

These reuse the *conventional scanner's own* CTCSS/DCS detectors, so the survey's
analog verdict matches what the live scanner would do. Similarly, a carrier at a
paging baud (`IsPagingBaud` — 512/1200/1600/2400) goes to `DecodePOCSAG`/`DecodeFLEX`,
which run the exact FM→resample→slice→syncer chain the *live* pager receivers use,
just over a fixed buffer. A non-trunked carrier still comes back with a real,
human-legible summary — never a silent skip.

## The encryption triage

Some digital carriers no decoder identifies — an unknown burst, an encrypted flow.
For those, `EncryptionTriage` gives a verdict *without a key*, because a strong
cipher's output is statistically indistinguishable from random:

```go
// internal/survey/encryption.go (shape) — EncryptionTriage
norm := stats.ShannonEntropy(data) / maxH   // normalized by achievable max
ic := stats.IndexOfCoincidence(data)        // ≈1/256 for uniform bytes
if norm >= 0.85 && ic < 0.02 {
    return true, "high-entropy (likely encrypted)"
}
return false, ""
```

The two statistics guard against each other. Normalized Shannon entropy is a loose
floor — a finite random sample under-estimates the full 8 bits/byte (Miller bias) —
so the byte-uniformity test (index of coincidence, which is sample-size robust) does
the real work. Random bytes sit at IC ≈ 0.0039; English text ~0.066; a repeating
scrambler or low-rate codec far higher. Only near-maximal entropy *and* near-uniform
bytes together read as encrypted; structured traffic falls below the bar. When a
protocol *did* decode and its grants carried an ALGID, `encTypeName` names the
algorithm (AES-256, ADP/RC4, DES-OFB) by reusing the Crypto Lab's own tables, so the
survey and the cryptanalysis toolkit agree on the label. The triage is the survey's
"is this encrypted?"; confirming the construction is the
[Crypto Lab]({{ '/blog/series/crypto-lab/' | relative_url }})'s job.

## Where this goes next

Our 851 MHz carrier now classifies as *digital, C4FM, probably not encrypted* — a
strong candidate for a P25 control channel. But classifying it assumed the front end
was delivering clean IQ. [Part 5]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
settles that: sweeping the gain to minimise the decode error rate and correcting the
tuning offset, so the identify in Part 7 gets the best signal the radio can give it.

## FAQ

**Why classify at all if identify is the authoritative call?**
Cost. Identify is expensive per carrier; a sweep produces dozens. The blind
classifier is cheap and routes each carrier so identify runs only on the digital
ones that could plausibly be a control channel — turning a forty-carrier survey from
"identify everything" into "identify a handful."

**How can it call something encrypted without decrypting it?**
It doesn't claim to read the traffic — it measures its statistics. Strong encryption
produces output indistinguishable from random: near-maximal entropy and near-uniform
byte frequencies. Structured traffic (plaintext, a scrambler, a codec) has
detectable regularity and fails those tests. It's a triage flag, confirmed elsewhere.

**Why is the digital test before the AM test?**
Because linearly-modulated digital carriers (π/4-DQPSK) have real envelope variation
from pulse shaping, which looks like AM's non-constant envelope. Testing for the
cyclostationary baud line first catches those as digital before the AM gate can
mis-claim them.

**What stops a DMR carrier being mislabelled AM?**
The AM gate requires *both* high envelope CV and low discriminator std. A DMR carrier
whose baud line was lost to noise might have an inflated envelope CV, but its
discriminator still swings with the deviation (high `IFStd`), so it fails the AM gate
and falls through to the FM split — the fix for issue #648.

**Does classification need the full capture?**
No — it works on a baseband IQ buffer already tuned to the carrier, and needs only a
few thousand samples (`minClassifySamples` is 4096) for the spectral estimates to be
trustworthy. The same routine runs on a live grab or an offline slice.

## Series navigation

**Part 4 of 14** · ←
[Part 3: Peaks & Occupancy — Finding Carriers in the Noise]({{ '/blog/deep-dives/the-hunt-03-peak-occupancy-detection/' | relative_url }})
· Next →
[Part 5: Autogain & Autotune — Settling the Front End]({{ '/blog/deep-dives/the-hunt-05-autogain-autotune/' | relative_url }})
