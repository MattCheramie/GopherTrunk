---
slug: automatic-modulation-classification
title: Automatic modulation classification (AMC)
entry_type: algorithm
category: sdr-app-building
description: "Automatic modulation classification (AMC) infers a signal's modulation scheme from its IQ, using likelihood tests, statistical features, or a trained neural network."
keywords: automatic modulation classification, AMC, automatic modulation recognition, AMR, modulation recognition, higher-order cumulants, likelihood-based classifier, CNN modulation, blind modulation ID
aka: [AMC, automatic modulation classification, automatic modulation recognition, AMR, modulation recognition]
autolink: true
infobox:
  - { label: Type, value: RF classification algorithm }
  - { label: Recovers, value: Modulation scheme (e.g. QPSK vs 8PSK) }
  - { label: Approaches, value: "Likelihood, feature, deep-learning" }
see_also: [signal-classification, rf-machine-learning, constellation-diagram, modulation, phase-shift-keying, quadrature-amplitude-modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Automatic_modulation_recognition
  - https://arxiv.org/abs/1602.04105
---

**Automatic modulation classification (AMC)** is the algorithmic problem of deciding which
[modulation](/reference/modulation/) scheme a received signal uses — BPSK, QPSK, 8PSK, 16-QAM,
FSK, and so on — from the IQ samples alone, without prior knowledge of the transmitter.[^amr]
It is the tightest, most-studied slice of [signal classification](/reference/signal-classification/):
the question is not *what is on the air* but specifically *how are its symbols carried*.
Getting the modulation right is the prerequisite for everything downstream, because a
demodulator built for the wrong scheme produces nothing but noise.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 160" role="img" aria-label="Three constellation diagrams — four points for QPSK, eight points on a circle for 8PSK, and a sixteen-point grid for 16-QAM — feed a classifier that outputs the most likely modulation label." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="amar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <text x="45" y="14">QPSK</text><circle cx="30" cy="35" r="2.4" fill="currentColor"/><circle cx="60" cy="35" r="2.4" fill="currentColor"/><circle cx="30" cy="65" r="2.4" fill="currentColor"/><circle cx="60" cy="65" r="2.4" fill="currentColor"/>
    <text x="45" y="92">8PSK</text><g fill="currentColor"><circle cx="45" cy="103" r="2.2"/><circle cx="62" cy="110" r="2.2"/><circle cx="69" cy="127" r="2.2"/><circle cx="62" cy="144" r="2.2"/><circle cx="45" cy="151" r="2.2"/><circle cx="28" cy="144" r="2.2"/><circle cx="21" cy="127" r="2.2"/><circle cx="28" cy="110" r="2.2"/></g>
    <text x="150" y="14">16-QAM</text><g fill="currentColor"><circle cx="128" cy="30" r="2"/><circle cx="144" cy="30" r="2"/><circle cx="160" cy="30" r="2"/><circle cx="176" cy="30" r="2"/><circle cx="128" cy="46" r="2"/><circle cx="144" cy="46" r="2"/><circle cx="160" cy="46" r="2"/><circle cx="176" cy="46" r="2"/><circle cx="128" cy="62" r="2"/><circle cx="144" cy="62" r="2"/><circle cx="160" cy="62" r="2"/><circle cx="176" cy="62" r="2"/><circle cx="128" cy="78" r="2"/><circle cx="144" cy="78" r="2"/><circle cx="160" cy="78" r="2"/><circle cx="176" cy="78" r="2"/></g>
    <rect x="248" y="58" width="96" height="40" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/><text x="296" y="74">classifier</text><text x="296" y="86">(likelihood / CNN)</text>
    <rect x="376" y="60" width="76" height="36" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="414" y="74">"16-QAM"</text><text x="414" y="86">label</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="80" y1="60" x2="246" y2="72" marker-end="url(#amar)"/>
    <line x1="190" y1="54" x2="246" y2="72" marker-end="url(#amar)"/>
    <line x1="80" y1="127" x2="246" y2="84" marker-end="url(#amar)"/>
    <line x1="344" y1="78" x2="374" y2="78" marker-end="url(#amar)"/>
  </g>
</svg>
<figcaption>AMC distinguishes modulations by the structure of their symbols — QPSK's four points, 8PSK's ring of eight, 16-QAM's grid — and emits the most likely label.</figcaption>
</figure>

## How it works

Every AMC method exploits the fact that different modulations leave different statistical
fingerprints. A [constellation](/reference/constellation-diagram/) of QPSK sits at four
points; [8PSK](/reference/phase-shift-keying/) spreads eight points around a circle;
[16-QAM](/reference/quadrature-amplitude-modulation/) fills a 4×4 grid; an FSK signal has no
stable constellation but a bimodal instantaneous frequency. The classifier's job is to
measure enough of that structure — through raw symbols, derived statistics, or a learned
representation — to separate the candidates despite noise, timing offset, and unknown carrier
phase. Because carrier and timing are usually unknown at classification time, robust AMC
relies on quantities that survive those unknowns, such as amplitude/phase moments and
rotation-invariant cumulants.

## Variants

- **Likelihood-based (LB).** Compute the probability of the received samples under each
  candidate modulation's model and pick the maximum. This is optimal in the Bayesian sense
  and sets the performance ceiling, but it needs accurate channel and synchronization
  parameters and is computationally heavy, so it is often a benchmark rather than a
  deployment.
- **Feature-based (FB).** Extract discriminating statistics — instantaneous amplitude, phase
  and frequency moments, spectral symmetry, and especially **higher-order cumulants**, which
  take characteristic values for each constellation and are insensitive to phase rotation —
  then feed them to a threshold rule or a small classifier. Nearly optimal in many cases,
  cheap, and interpretable.
- **Deep-learning.** Train a convolutional network on raw IQ (or a spectrogram) so it learns
  its own features, the approach popularized by the RadioML modulation-recognition
  work.[^radioml] It scales to many classes and low SNR but demands large labelled datasets
  and offline training — the [RF machine learning](/reference/rf-machine-learning/) route.

Performance for all three collapses as SNR drops and as the candidate set grows to include
easily confused pairs (16-QAM vs 64-QAM, QPSK vs 8PSK), which is why practical systems narrow
the list using known bandwidth and symbol rate before classifying.

## Relevance to SDR

AMC is central to cognitive radio, adaptive links that switch modulation with channel
quality, spectrum surveillance, and signal intelligence — anywhere a receiver must adapt to a
signal it did not schedule. It is a recognition step, not a decode step: its output selects
*which* demodulator to run next.

**GopherTrunk does not perform automatic modulation classification.** The systems it targets
have fixed, standardized modulations — P25 Phase 1 is C4FM/π-4-DQPSK at 4800 symbols/s, DMR
is 4-FSK, TETRA is π/4-DQPSK — so the modulation is a known constant of each protocol, not
something to infer. GopherTrunk is told which system to decode and applies the matching,
deterministic demodulator directly, with no classification stage, no cumulant features, and
no neural network. AMC solves the upstream problem of identifying an *unknown* signal's
modulation, and could feed a decoder like GopherTrunk in a survey pipeline, but it is outside
GopherTrunk's own scope as a decoder of specified modes.

## Sources

[^amr]: [Automatic modulation recognition](https://en.wikipedia.org/wiki/Automatic_modulation_recognition) — Wikipedia, on likelihood-based and feature-based AMC and the role of higher-order cumulants.
[^radioml]: [Convolutional Radio Modulation Recognition Networks](https://arxiv.org/abs/1602.04105) — O'Shea, Corgan, Clancy (2016), the CNN-on-IQ approach and RadioML datasets behind deep-learning AMC.
