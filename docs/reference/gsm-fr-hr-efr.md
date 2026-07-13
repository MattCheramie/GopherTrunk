---
slug: gsm-fr-hr-efr
title: GSM FR / HR / EFR
entry_type: technology
category: voice-coding
description: "GSM FR, HR and EFR are the original GSM voice codecs — RPE-LTP and ACELP speech coders that carried 2G cellular calls."
keywords: GSM FR, GSM HR, GSM EFR, full rate, half rate, enhanced full rate, RPE-LTP, ACELP, GSM 06.10, 2G voice codec
aka: [GSM FR, GSM HR, GSM EFR, GSM 06.10]
autolink: true
infobox:
  - { label: Type, value: GSM speech codecs (RPE-LTP / ACELP) }
  - { label: Members, value: "Full, Half, Enhanced Full rate" }
  - { label: Used by, value: 2G GSM voice }
see_also: [gsm, acelp, linear-predictive-coding, code-excited-linear-prediction, vocoder, amr]
cite_urls:
  - https://en.wikipedia.org/wiki/Full_Rate
  - https://en.wikipedia.org/wiki/Enhanced_Full_Rate
---

**GSM FR, HR and EFR** are the original speech codecs of the
[GSM](/reference/gsm/) cellular system — Full Rate, Half Rate and Enhanced Full
Rate.[^fr] Full Rate (the RPE-LTP coder of GSM 06.10) was the first digital
cellular voice codec deployed at scale; Half Rate roughly doubled network capacity
by squeezing voice into half the channel resource; and Enhanced Full Rate, an
[ACELP](/reference/acelp/) design, restored and improved quality at the full-rate
bit budget. Together they defined how 2G calls sounded for a generation of mobile
users.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="Three GSM codecs compared: Full Rate at 13 kbps using RPE-LTP, Half Rate at 5.6 kbps using VSELP for capacity, and Enhanced Full Rate at 12.2 kbps using ACELP for better quality." xmlns="http://www.w3.org/2000/svg">
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <rect x="30" y="30" width="120" height="70"/><rect x="170" y="55" width="120" height="45"/><rect x="310" y="34" width="120" height="66"/>
  </g>
  <g font-size="9" fill="currentColor" text-anchor="middle">
    <text x="90" y="52">FR</text><text x="230" y="74">HR</text><text x="370" y="54">EFR</text>
  </g>
  <g font-size="7" fill="currentColor" text-anchor="middle">
    <text x="90" y="68">13.0 kbps</text><text x="90" y="80">RPE-LTP</text>
    <text x="230" y="88">5.6 kbps</text><text x="230" y="98">VSELP</text>
    <text x="370" y="70">12.2 kbps</text><text x="370" y="82">ACELP</text>
  </g>
  <text x="230" y="122" text-anchor="middle" font-size="8" fill="currentColor">HR trades quality for capacity; EFR trades algorithm for quality at FR's rate</text>
</svg>
<figcaption>The three classic GSM codecs balance bitrate, network capacity and voice quality differently.</figcaption>
</figure>

## How it works

All three are linear-prediction speech coders — each frame they fit a short-term
[LPC](/reference/linear-predictive-coding/) filter for the vocal-tract spectrum and
a long-term (pitch) predictor — but they differ in how they code the excitation.

- **Full Rate (FR).** Uses **RPE-LTP**, Regular Pulse Excitation with Long-Term
  Prediction. The excitation is a regularly spaced grid of pulses whose positions
  and amplitudes are quantised — simpler than a codebook search. It runs at 13
  kbps, and with GSM's channel coding occupies about 22.8 kbps on the air. Its
  characteristic slightly buzzy sound is the classic early-mobile timbre.
- **Half Rate (HR).** Uses **VSELP**, a vector-sum excited linear-prediction coder,
  at about 5.6 kbps. Halving the source rate lets an operator carry two calls in the
  channel resource of one FR call, doubling capacity at the cost of noticeably lower
  quality, especially in noise.
- **Enhanced Full Rate (EFR).** Uses **[ACELP](/reference/acelp/)**, the algebraic
  code-excited variant, at 12.2 kbps. By replacing RPE with a sparse algebraic
  fixed codebook searched in an analysis-by-synthesis loop
  ([CELP](/reference/code-excited-linear-prediction/)), EFR delivers clearly better
  quality than FR while fitting the same full-rate channel — the 12.2 kbps mode was
  later reused as the top rate of AMR.

Each coder emits fixed-size 20 ms frames with the speech bits ordered by
perceptual importance, so GSM's unequal error protection can shield the most
critical bits most heavily against the radio channel.

## In practice

The progression tells a capacity-versus-quality story. FR came first; HR answered
operators who needed more simultaneous calls in congested cells; EFR answered users
who wanted the calls to sound better. All three were eventually subsumed by
[AMR](/reference/amr/) (Adaptive Multi-Rate), which folded the good ideas of EFR
into a single coder that switches among eight rates and adapts the split between
speech bits and error-protection bits to channel conditions — but FR, HR and EFR
remained in the field for the long life of 2G.

## Relevance to SDR

These are cellular voice codecs, not land-mobile trunking vocoders, so they fall
outside the P25/DMR/NXDN/TETRA family that GopherTrunk decodes — those use the MBE
vocoders (IMBE, AMBE+2) instead of RPE-LTP or ACELP. Their SDR relevance is as the
speech layer of [GSM](/reference/gsm/): projects that analyse or decode 2G
signalling and traffic must contend with FR/HR/EFR framing to reconstruct audio,
and the codecs are a textbook illustration of the RPE-LTP-to-ACELP evolution that
shaped later mobile voice. GopherTrunk does not implement GSM codecs; recovering GSM
call audio is a separate, cellular-specific problem from its trunked-radio decode
chain.

## Sources

[^fr]: [Full Rate](https://en.wikipedia.org/wiki/Full_Rate) — Wikipedia, on the original RPE-LTP GSM codec; see also [Enhanced Full Rate](https://en.wikipedia.org/wiki/Enhanced_Full_Rate) on the ACELP EFR coder.
