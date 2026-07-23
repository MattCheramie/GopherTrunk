---
slug: snr-evm-and-ber
title: SNR, EVM & BER
description: The three numbers that measure a digital link's health — signal-to-noise ratio, error-vector magnitude, and bit error rate — and how they relate to each other.
keywords: snr, evm, ber, signal to noise ratio, error vector magnitude, bit error rate, link quality metrics, demod quality
level: intermediate
status: full
prereq:
  - constellations-and-symbol-mapping
  - the-decibel-in-dsp
faq:
  - q: How are SNR, EVM, and BER related?
    a: "They measure the same link health at three stages. SNR, in decibels, is how far the signal sits above the noise. That noise scatters each symbol away from its ideal constellation point, and EVM measures that scatter as a percentage. When the scatter grows large enough that symbols cross into a neighbour's decision region, bits flip — and BER counts those flips as a fraction of all bits. Lower SNR means higher EVM means higher BER."
  - q: What EVM or SNR do I need to decode a signal?
    a: "It depends on the modulation. A robust two- or four-level scheme tolerates a lot of scatter and decodes at fairly low SNR, while a dense QAM constellation packs points close together and needs a much cleaner signal. Forward error correction lowers the required SNR further by fixing a bounded number of bit errors, so the raw BER can be nonzero and the decoded data still be perfect."
---

# SNR, EVM & BER

<div class="tldr" markdown="1">
<span class="tldr__label">Key takeaways</span>
Three numbers grade a digital link, each one step downstream of the last. **SNR** (in
[dB](/learn/dsp/the-decibel-in-dsp/)) is how far the signal sits above the noise. That
noise scatters symbols off their ideal [constellation](/learn/dsp/constellations-and-symbol-mapping/)
points — **EVM** measures the scatter as a percentage. When scatter pushes a symbol into
the wrong decision region, a bit flips — **BER** counts those flips per bit. **Lower SNR →
higher EVM → higher BER.**
</div>

You can *see* link health on a constellation; these three metrics *quantify* it. This
lesson ties them together and builds on
[decibels](/learn/dsp/the-decibel-in-dsp/) and
[constellations](/learn/dsp/constellations-and-symbol-mapping/). It mirrors the RF path's
[noise & SNR](/learn/rf-sdr/noise-and-snr/).

## SNR: signal above noise

**Signal-to-noise ratio** is the ratio of signal power to noise power, almost always
quoted in **decibels**. It is the root cause metric — everything else follows from it. A
high SNR means the wanted signal towers over the background; a low SNR means it is nearly
buried. Because it's a power ratio it uses the 10·log₁₀ form: 20 dB SNR is 100× more
signal power than noise.

## EVM: scatter on the constellation

Noise doesn't stay abstract — it knocks each received symbol *off* its ideal point.
**Error-vector magnitude** measures exactly that: for each symbol, draw the vector from
where it should be to where it landed; EVM is the size of that error vector, averaged and
expressed as a **percentage** of the ideal signal amplitude.

<figure class="figure" markdown="0">
<svg viewBox="0 0 300 170" role="img" aria-label="One constellation point with received symbols scattered around it, and an arrow from the ideal point to a received sample labelled the error vector." xmlns="http://www.w3.org/2000/svg">
  <line x1="20" y1="90" x2="280" y2="90" stroke="currentColor" stroke-opacity="0.25"/>
  <line x1="150" y1="20" x2="150" y2="160" stroke="currentColor" stroke-opacity="0.25"/>
  <circle cx="210" cy="55" r="5" fill="none" stroke="currentColor" stroke-width="2"/>
  <text x="222" y="50" font-size="9" fill="currentColor">ideal point</text>
  <g fill="currentColor" fill-opacity="0.6">
    <circle cx="205" cy="62" r="2.5"/><circle cx="222" cy="50" r="2.5"/><circle cx="215" cy="70" r="2.5"/>
    <circle cx="198" cy="52" r="2.5"/><circle cx="228" cy="63" r="2.5"/>
  </g>
  <line x1="210" y1="55" x2="222" y2="50" stroke="currentColor" stroke-width="1.5"/>
  <text x="200" y="30" font-size="9" fill="currentColor">error vector</text>
</svg>
<figcaption>EVM is the length of the error vector from a symbol's ideal point to where it actually landed — the scatter that noise and distortion cause, as a percent.</figcaption>
</figure>

Low EVM (a few percent) is a tight cluster; high EVM is a fuzzy blob. GopherTrunk's own
diagnostics report demod EVM — for example the project's field notes cite a channel
locking at **EVM ≈ 7.4%** but failing at **≈ 22.5%**, a concrete threshold between decode
and no-decode.

## BER: bits that actually flipped

Push EVM high enough and a symbol crosses into a **neighbour's** decision region — the
decoder picks the wrong symbol, and (thanks to [Gray coding](/learn/dsp/constellations-and-symbol-mapping/))
usually one bit flips. **Bit error rate** is the count of flipped bits over total bits:

```text
BER = (bits received wrong) / (total bits)
e.g. 3 bad bits in 10,000  ->  BER = 3e-4
```

BER is the bottom-line metric — it is what actually breaks a message. And it is where
[error correction](/learn/dsp/error-correction-and-framing/) enters: FEC can repair a
bounded number of flips, so a link with a modest raw BER can still deliver *perfect*
decoded data.

## How they chain

| Metric | Measures | Units | Position |
|--------|----------|-------|----------|
| **SNR** | signal vs noise power | dB | cause |
| **EVM** | symbol scatter from ideal | % | effect on symbols |
| **BER** | fraction of bits wrong | ratio | effect on data |

Read them as one story: noise sets SNR, SNR sets how far symbols scatter (EVM), scatter
sets how often bits flip (BER). Improve the front-end SNR and all three improve together.

<div class="knowledge-check" data-quiz data-correct-msg="Right — EVM measures how far received symbols scatter from their ideal constellation points." markdown="0">
  <p class="knowledge-check__q">Quick check: what does EVM measure?</p>
  <ul class="quiz__options">
    <li><button type="button" class="quiz__option" data-answer="wrong">The fraction of bits received incorrectly</button></li>
    <li><button type="button" class="quiz__option" data-answer="correct">How far received symbols scatter from their ideal constellation points, as a percent</button></li>
    <li><button type="button" class="quiz__option" data-answer="wrong">The signal power above the noise floor, in dB</button></li>
  </ul>
  <p class="quiz__feedback" data-quiz-feedback hidden></p>
</div>

## Recap

- **SNR** (dB) is signal power over noise power — the root-cause quality metric.
- **EVM** (%) is how far symbols scatter from their ideal constellation points.
- **BER** is the fraction of bits that actually flipped — the bottom line.
- They chain: **lower SNR → higher EVM → higher BER**; FEC can still rescue a modest BER.

Next up: overlaying symbol periods into a single picture — the eye diagram.
