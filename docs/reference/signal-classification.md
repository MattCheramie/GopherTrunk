---
slug: signal-classification
title: Signal classification
entry_type: concept
category: sdr-app-building
description: "Signal classification is deciding what an unknown RF signal is — its modulation, protocol, or type — from its IQ or spectrogram, by features or a trained model."
keywords: signal classification, RF signal identification, modulation recognition, signal type, feature-based classification, deep learning classifier, spectrogram CNN, spectrum sensing, blind signal ID
aka: [signal classification, signal identification, RF classification]
autolink: true
infobox:
  - { label: Type, value: RF recognition task }
  - { label: Idea, value: Label an unknown signal from its samples }
  - { label: Approaches, value: "Features vs. deep learning" }
see_also: [automatic-modulation-classification, rf-machine-learning, spectrogram, energy-detection, waterfall-display, modulation]
cite_urls:
  - https://en.wikipedia.org/wiki/Statistical_classification
  - https://en.wikipedia.org/wiki/Automatic_modulation_recognition
---

**Signal classification** is the task of deciding what an unknown radio signal is — its
[modulation](/reference/modulation/), its protocol or waveform family, or simply whether a
band is occupied — from the received samples alone, without a decoder that already assumes
the answer.[^cls] It sits one level above
[automatic modulation classification](/reference/automatic-modulation-classification/): AMC
answers the narrow question "which modulation?", while signal classification can also mean
"is this LTE or Wi-Fi or a radar pulse?" or "is anything here at all?" The unifying idea is
mapping raw [IQ](/reference/spectrogram/) or a time-frequency image to one of a fixed set of
labels.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 150" role="img" aria-label="An unknown IQ input is turned into features or a spectrogram, passed to a classifier, and mapped to one of several candidate labels: FM, LTE, radar, or noise." xmlns="http://www.w3.org/2000/svg">
  <defs><marker id="scar" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0 0 L6 3 L0 6 z" fill="currentColor"/></marker></defs>
  <g font-size="8" fill="currentColor" text-anchor="middle">
    <rect x="8" y="56" width="72" height="30" rx="5" fill="none" stroke="currentColor" stroke-width="1.2"/><text x="44" y="68">unknown</text><text x="44" y="79">IQ / spec</text>
    <rect x="126" y="52" width="90" height="38" rx="6" fill="currentColor" fill-opacity="0.15" stroke="currentColor" stroke-width="1.4"/><text x="171" y="67">features or</text><text x="171" y="78">CNN</text>
    <rect x="262" y="52" width="86" height="38" rx="6" fill="none" stroke="currentColor" stroke-width="1.4"/><text x="305" y="67">classifier</text><text x="305" y="78">argmax</text>
    <rect x="392" y="18" width="60" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="422" y="31">FM</text>
    <rect x="392" y="46" width="60" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="422" y="59">LTE</text>
    <rect x="392" y="74" width="60" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="422" y="87">radar</text>
    <rect x="392" y="102" width="60" height="20" rx="4" fill="none" stroke="currentColor" stroke-width="1"/><text x="422" y="115">noise</text>
  </g>
  <g stroke="currentColor" stroke-width="1.2" fill="none">
    <line x1="80" y1="71" x2="124" y2="71" marker-end="url(#scar)"/>
    <line x1="216" y1="71" x2="260" y2="71" marker-end="url(#scar)"/>
    <line x1="348" y1="66" x2="390" y2="30" marker-end="url(#scar)"/>
    <line x1="348" y1="69" x2="390" y2="56" marker-end="url(#scar)"/>
    <line x1="348" y1="73" x2="390" y2="84" marker-end="url(#scar)"/>
    <line x1="348" y1="76" x2="390" y2="110" marker-end="url(#scar)"/>
  </g>
</svg>
<figcaption>Signal classification maps an unknown input, via features or a learned model, to one label from a fixed set — a broader question than modulation alone.</figcaption>
</figure>

## How it works

There are two broad approaches, and modern systems often combine them.

**Feature-based (expert) classification** extracts hand-designed statistics that separate the
classes and feeds them to a simple decision rule or shallow classifier. Useful features
include occupied bandwidth, symbol rate, spectral shape and symmetry, the presence of a
carrier or cyclostationary periodicities, envelope and phase moments, and higher-order
cumulants. This route is interpretable and cheap, and works well when the candidate set is
small and the features are chosen by someone who understands the signals.

**Deep-learning classification** skips the hand design: a convolutional network learns
features directly from raw IQ or from a [spectrogram](/reference/spectrogram/) image, trained
on a labelled dataset — this is the [RFML](/reference/rf-machine-learning/) approach. It
scales to many classes and can pick up subtle structure a human might miss, at the cost of
needing lots of data, compute, and trust in a model whose reasoning is opaque.

Either way, the pipeline usually starts with **detection** — an [energy detector](/reference/energy-detection/)
or spectral search finds that *something* is present and isolates its band — before the
classifier decides *what* it is. Both stages degrade as SNR falls, and both are only as good
as the assumed class list: a signal outside the trained or modelled set will still be forced
into one of the known labels, often with unwarranted confidence.

## In practice

Signal classification is the front end of spectrum awareness: a monitoring receiver sweeps a
band, detects each occupied slice on the [waterfall](/reference/waterfall-display/),
classifies it, and only then routes it to a matching decoder. It turns "there is energy at
462 MHz" into "that is an FM voice channel," which is the decision that determines what to do
next.

## Relevance to SDR

Classification underpins cognitive radio, dynamic spectrum access, interference hunting,
signal intelligence, and any survey tool meant to map *what is on the air* rather than decode
one known channel. It is the recognition layer that a pile of deterministic decoders needs in
front of it when the input is unknown.

**GopherTrunk does not classify signals.** It is a decoder for signals whose type is already
known and configured — you point it at a P25, DMR, NXDN, or TETRA system and it demodulates
and frames that specified mode. There is no learned model and no blind-recognition stage in
its chain; the modulation and protocol are inputs, not outputs. In a broader monitoring
system, a signal classifier would play the complementary role of deciding which signals
*are* P25 or DMR in the first place and handing them to a decoder like GopherTrunk. That
detection-and-classification front end is out of scope for GopherTrunk itself, which stays
focused on decoding the modes it explicitly supports.

## Sources

[^cls]: [Statistical classification](https://en.wikipedia.org/wiki/Statistical_classification) — Wikipedia, for the general task of assigning inputs to labels. See also [Automatic modulation recognition](https://en.wikipedia.org/wiki/Automatic_modulation_recognition) for the closely related RF-specific problem.
