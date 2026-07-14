---
slug: rf-machine-learning
title: RF machine learning (RFML)
entry_type: technology
category: sdr-app-building
description: "RF machine learning (RFML) applies neural networks to raw IQ or spectrograms for tasks like modulation recognition, signal classification, and RF fingerprinting."
keywords: RF machine learning, RFML, deep learning radio, modulation recognition, signal classification, RF fingerprinting, IQ dataset, RadioML, spectrogram CNN, neural network SDR
aka: [RFML, RF machine learning, RF deep learning, deep learning for RF]
autolink: true
infobox:
  - { label: Type, value: ML applied to RF signals }
  - { label: Idea, value: Learn features from IQ, not hand-code them }
  - { label: Tasks, value: "Modulation ID, signal class, fingerprinting" }
see_also: [signal-classification, automatic-modulation-classification, sigmf, iq-data, spectrogram, edge-ai]
cite_urls:
  - https://en.wikipedia.org/wiki/Machine_learning
  - https://arxiv.org/abs/1602.04105
---

**RF machine learning (RFML)** is the application of machine learning — usually deep neural
networks — to radio-frequency signals, learning directly from
[IQ samples](/reference/iq-data/) or [spectrograms](/reference/spectrogram/) rather than from
hand-engineered detectors.[^ml] Instead of a human deriving the exact statistic that
distinguishes QPSK from 8PSK, a network is trained on many labelled examples and discovers
the discriminating features itself. Core RFML tasks include
[automatic modulation classification](/reference/automatic-modulation-classification/),
broader [signal classification](/reference/signal-classification/), interference and anomaly
detection, and device **RF fingerprinting** (identifying a specific transmitter from subtle
hardware imperfections).

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="A labelled dataset of IQ recordings trains a neural network offline; the trained model then runs inference on a new IQ input and outputs a class label with a confidence score." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="rmar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="8" y="20" width="86" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="51" y="32">labelled IQ</text><text x="51" y="43">dataset</text>
    <rect x="150" y="18" width="110" height="34" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/><text x="205" y="32">train (offline,</text><text x="205" y="43">GPU)</text>
    <rect x="150" y="96" width="110" height="34" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="205" y="110">trained model</text><text x="205" y="121">inference</text>
    <rect x="8" y="100" width="86" height="26" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="51" y="116">new IQ</text>
    <rect x="316" y="96" width="120" height="34" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="376" y="110">"QPSK" — 0.93</text><text x="376" y="121">class + confidence</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="94" y1="35" x2="148" y2="35" marker-end="url(#rmar)"/>
    <line x1="205" y1="52" x2="205" y2="94" marker-end="url(#rmar)"/>
    <line x1="94" y1="113" x2="148" y2="113" marker-end="url(#rmar)"/>
    <line x1="260" y1="113" x2="314" y2="113" marker-end="url(#rmar)"/>
  </g>
</svg>
<figcaption>RFML trains a model offline on labelled IQ, then runs inference on new signals to emit a class and confidence — the features are learned, not hand-coded.</figcaption>
</figure>

## How it works

An RFML system has two phases. **Training** happens offline on a big machine: a labelled
dataset of recordings is fed to a network — often a 1-D convolutional net over raw IQ, or a
2-D CNN over spectrogram images — which adjusts its weights to minimize classification error.
This is where the well-known public datasets come in, such as the DeepSig **RadioML** sets
that pair many modulation types across a range of SNRs.[^radioml] Because signals are
notoriously variable, training leans heavily on **data augmentation** and channel simulation:
adding noise, frequency and timing offsets, and fading so the model generalizes beyond the
lab.

**Inference** is the deployed phase: a short window of live IQ (or its spectrogram) is passed
through the frozen model, which outputs class probabilities. Inference is far cheaper than
training and can run on a CPU or a small accelerator, but it inherits the training set's
blind spots — a model only recognizes what it was taught, and confidence scores can be
misleadingly high on out-of-distribution signals.

Reproducibility hinges on **metadata**. A recording is useless for ML without its center
frequency, sample rate, and labels, so RFML datasets are increasingly packaged in
[SigMF](/reference/sigmf/), the open standard that stores IQ alongside a JSON metadata and
annotation file. Clean, well-labelled, SigMF-style data is often the hardest and most
valuable part of an RFML project.

## Relevance to SDR

RFML is an active research and increasingly operational field: cognitive radio and dynamic
spectrum access, spectrum-monitoring systems that flag unknown emitters, interference
hunting, and security work that fingerprints or spoofs devices. Its promise is handling the
**unknown** — signals no hand-written decoder was built for — and adapting as the spectrum
changes. Its limits are real too: heavy data and compute needs, brittleness under
distribution shift, and poor interpretability compared with a deterministic decoder whose
every step is auditable. The training side typically wants a [GPU](/reference/edge-ai/) or
accelerator; inference is what gets pushed toward the edge.

**GopherTrunk does not use machine learning.** Its entire chain is classical, deterministic
DSP and protocol logic — down-conversion, matched filtering, timing and carrier recovery,
symbol slicing, error correction, and framing — and it decodes signals whose modulation and
framing are already known (P25, DMR, NXDN, TETRA, and the rest), so there is nothing to
classify: the mode is specified, not inferred. That makes GopherTrunk exact and repeatable
but, by design, blind to signals it has no decoder for. RFML addresses the complementary
problem of recognizing and sorting unknown signals, and the two could sit side by side in a
monitoring stack — an RFML classifier triaging the spectrum and handing known modes to a
deterministic decoder like GopherTrunk — but GopherTrunk itself ships no models, no training,
and no inference.

## Sources

[^ml]: [Machine learning](https://en.wikipedia.org/wiki/Machine_learning) — Wikipedia, for the supervised-learning framework RFML applies to RF signals.
[^radioml]: [Convolutional Radio Modulation Recognition Networks](https://arxiv.org/abs/1602.04105) — O'Shea, Corgan, Clancy (2016), the paper behind the RadioML datasets and CNN-on-IQ modulation recognition that seeded modern RFML.
