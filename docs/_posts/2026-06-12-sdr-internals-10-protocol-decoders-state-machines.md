---
title: "SDR in Pure Go, Part 10: Protocol Decoders as State Machines"
description: How GopherTrunk decodes P25, DMR, NXDN, TETRA and a dozen more trunking protocols in pure Go — each a stateful receiver behind a uniform Grant contract, an adapter pattern that keeps the engine protocol-agnostic.
category: deep-dives
tags: [sdr, go, p25, dmr, trunking, software-design]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "SDR Internals"
series_part: 10
---

*Part 10 of **SDR Internals**. Now the symbols mean something. This post is about
the protocol decoders in `internal/radio` — P25, DMR, NXDN, TETRA, and more —
and the uniform contract that lets one engine drive all of them.*

## In this post

- What a trunking protocol decoder does and why each is a **state machine**.
- The breadth: **P25, DMR, NXDN, dPMR, TETRA, Motorola, EDACS, LTR, MPT 1327,
  D-STAR, YSF, M17**, plus paging and maritime protocols.
- The **adapter pattern + uniform `Grant` contract** that keeps the trunking
  engine protocol-agnostic.

## What a protocol decoder does

A [trunked radio]({{ '/reference/trunked-radio/' | relative_url }}) system has a
[control channel]({{ '/reference/control-channel/' | relative_url }}) that
continuously announces which talkgroup is using which frequency. A decoder's job
is to: hunt for frame sync, run the FEC from
[Part 9]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }}),
parse the signaling messages, and emit a
[channel grant]({{ '/reference/channel-grant/' | relative_url }}) — "talkgroup X
is now on frequency Y." Each protocol encodes this completely differently:

- **P25** ([Phase 1]({{ '/reference/p25-phase-1/' | relative_url }}) /
  [Phase 2]({{ '/reference/p25-phase-2/' | relative_url }})) — TSBK / MAC PDUs.
- **DMR** ([Tier III]({{ '/reference/dmr-tier-3/' | relative_url }})) — CSBK
  bursts over two TDMA slots.
- **NXDN**, **dPMR**, **TETRA**, **Motorola Type II**, **EDACS**, **LTR**,
  **MPT 1327** — each its own framing and signaling vocabulary.
- Amateur modes **D-STAR**, **YSF**, **M17**; paging **POCSAG/FLEX**; maritime
  **AIS/DSC**; aviation **ADS-B**.

The [protocol landscape]({{ '/learn/rf-sdr/protocol-landscape/' | relative_url }})
lesson maps the whole family.

## How GopherTrunk implements it in Go

Each protocol is a sub-package under `internal/radio` (`p25/`, `dmr/`, `nxdn/`,
`tetra/`, …) with a stateful **receiver** that consumes dibits and walks a state
machine: hunting → synced → parsing → grant. The NXDN receiver is representative —
it chains the earlier stages and accumulates internal state:

```go
// internal/radio/nxdn/receiver — receiver as state machine (shape)
type Receiver struct {
    fm    *demod.FM
    mf    *demod.C4FM
    clock *sync.MuellerMuller
    state frameState // hunting, synced, in-frame…
}

func (r *Receiver) Process(iq []complex64) {
    // demod -> matched filter -> timing -> slice -> sync hunt -> parse
}
```

However different P25 and DMR look inside, they converge on the **same output**: a
`Grant` value carrying the tuned frequency, optional timeslot, talkgroup, source
unit, and an encrypted flag. That uniform result is the contract the rest of the
system relies on.

<figure class="lab-figure">
<svg viewBox="0 0 660 170" width="660" height="170" role="img" aria-label="A control-channel decoder state machine with four states left to right: hunting, synced, parsing, and grant emitted. Correlator over threshold advances hunting to synced, FEC ok advances to parsing, a decoded signaling message emits a grant, and losing sync returns any state to hunting.">
  <rect x="8" y="46" width="140" height="48" rx="10" fill="none" stroke="var(--accent)"/>
  <text x="78" y="66" text-anchor="middle" fill="var(--accent)" font-size="11">hunting</text>
  <text x="78" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">scan for sync</text>
  <line x1="148" y1="70" x2="170" y2="70" stroke="currentColor"/><polygon points="170,66 180,70 170,74" fill="currentColor"/>
  <text x="164" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="8">corr &gt; thr</text>
  <rect x="180" y="46" width="140" height="48" rx="10" fill="none" stroke="currentColor"/>
  <text x="250" y="66" text-anchor="middle" fill="currentColor" font-size="11">synced</text>
  <text x="250" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">frame locked</text>
  <line x1="320" y1="70" x2="342" y2="70" stroke="currentColor"/><polygon points="342,66 352,70 342,74" fill="currentColor"/>
  <text x="336" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="8">FEC ok</text>
  <rect x="352" y="46" width="140" height="48" rx="10" fill="none" stroke="currentColor"/>
  <text x="422" y="66" text-anchor="middle" fill="currentColor" font-size="11">parsing</text>
  <text x="422" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">signaling msgs</text>
  <line x1="492" y1="70" x2="514" y2="70" stroke="currentColor"/><polygon points="514,66 524,70 514,74" fill="currentColor"/>
  <text x="508" y="40" text-anchor="middle" fill="var(--fg-muted)" font-size="8">grant</text>
  <rect x="524" y="46" width="128" height="48" rx="10" fill="none" stroke="var(--accent)"/>
  <text x="588" y="66" text-anchor="middle" fill="var(--accent)" font-size="11">grant emitted</text>
  <text x="588" y="80" text-anchor="middle" fill="var(--fg-muted)" font-size="8">TG on freq</text>
  <line x1="588" y1="94" x2="588" y2="134" stroke="currentColor"/><line x1="588" y1="134" x2="78" y2="134" stroke="currentColor"/><line x1="78" y1="134" x2="78" y2="94" stroke="currentColor"/><polygon points="74,102 78,94 82,102" fill="currentColor"/>
  <text x="333" y="150" text-anchor="middle" fill="var(--fg-muted)" font-size="9">sync lost &#8594; re-hunt</text>
</svg>
<figcaption>Each <code>internal/radio</code> receiver is an explicit state machine &#8212; hunting &#8594; synced &#8594; parsing &#8594; grant &#8212; with a single back edge when sync is lost, making lock, loss, and re-sync readable and testable.</figcaption>
</figure>

## The design principle: adapter + a uniform contract

Each decoder is an **adapter**: it translates one protocol's idiosyncratic
signaling into a single, shared vocabulary of events. The trunking engine in
[Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})
consumes only that vocabulary — it has no idea whether a grant came from a P25
TSBK or a DMR CSBK.

<figure class="lab-figure">
<svg viewBox="0 0 660 200" width="660" height="200" role="img" aria-label="The adapter pattern: four protocol receivers on the left, P25 TSBK, DMR CSBK, NXDN, and TETRA, each translate their own signaling into a single uniform Grant value in the middle, which the protocol-agnostic trunking engine on the right consumes.">
  <rect x="8" y="18" width="150" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="83" y="38" text-anchor="middle" fill="currentColor" font-size="10">P25 &#183; TSBK / MAC</text>
  <rect x="8" y="62" width="150" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="83" y="82" text-anchor="middle" fill="currentColor" font-size="10">DMR &#183; CSBK</text>
  <rect x="8" y="106" width="150" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="83" y="126" text-anchor="middle" fill="currentColor" font-size="10">NXDN receiver</text>
  <rect x="8" y="150" width="150" height="32" rx="6" fill="none" stroke="currentColor"/>
  <text x="83" y="170" text-anchor="middle" fill="currentColor" font-size="10">TETRA receiver</text>
  <line x1="158" y1="34" x2="300" y2="90" stroke="currentColor"/><polygon points="294,84 302,92 290,93" fill="currentColor"/>
  <line x1="158" y1="78" x2="300" y2="96" stroke="currentColor"/><polygon points="293,90 302,98 292,100" fill="currentColor"/>
  <line x1="158" y1="122" x2="300" y2="104" stroke="currentColor"/><polygon points="292,100 302,102 293,110" fill="currentColor"/>
  <line x1="158" y1="166" x2="300" y2="110" stroke="currentColor"/><polygon points="290,107 302,108 294,116" fill="currentColor"/>
  <rect x="302" y="72" width="150" height="56" rx="6" fill="none" stroke="var(--accent)"/>
  <text x="377" y="96" text-anchor="middle" fill="var(--accent)" font-size="11">Grant</text>
  <text x="377" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">freq &#183; TG &#183; slot &#183; enc</text>
  <line x1="452" y1="100" x2="490" y2="100" stroke="currentColor"/><polygon points="490,96 500,100 490,104" fill="currentColor"/>
  <rect x="500" y="72" width="152" height="56" rx="6" fill="none" stroke="currentColor"/>
  <text x="576" y="96" text-anchor="middle" fill="currentColor" font-size="11">trunking engine</text>
  <text x="576" y="112" text-anchor="middle" fill="var(--fg-muted)" font-size="8">protocol-agnostic (Part 11)</text>
</svg>
<figcaption>Every decoder is an adapter: however different P25, DMR, NXDN, and TETRA look inside, they converge on one uniform <code>Grant</code> contract, so the engine is written once and never changes when a protocol is added.</figcaption>
</figure>

### How that principle shaped the Go code

- **The engine speaks one language.** Because every decoder publishes the same
  `Grant`/event shape, the engine's grant-handling logic is written once and works
  for all 12+ protocols. Adding a protocol never touches the engine.
- **Adapters absorb the mess.** All the protocol-specific weirdness — Motorola
  vendor TSBK forms, DMR slot polarity inversion, TETRA's TDMA timing — is
  contained inside its own package. The complexity doesn't leak outward.
- **State machines, not callbacks-soup.** Each receiver is an explicit state
  machine with its own fields, so lock acquisition, loss, and re-sync are readable
  and testable. One receiver instance handles one signal; concurrency comes from
  running many receivers, not from sharing one.
- **Uniform shape, uniform tests.** Integration tests synthesize spec-correct IQ
  for a protocol and assert the engine sees the expected grant — the same harness
  works across protocols because the output contract is identical (more in
  [Part 14]({{ '/blog/deep-dives/sdr-internals-14-apis-testing-pure-go/' | relative_url }})).

## Where this goes next

Each protocol genuinely deserves its own series — P25's TSBK opcodes, DMR's
two-slot TDMA and Capacity Plus, TETRA's layered PDUs, the amateur modes' open
specs. This overview is the scaffold; the per-protocol deep dives will hang real
message-by-message decoding on it. Next, we follow a grant into the engine that
acts on it.

## FAQ

**Does GopherTrunk decrypt encrypted calls?**
No. Decoders detect encryption (algorithm and key IDs) and mark the grant as
encrypted, but no decryption is performed — that's both a legal and a design
boundary.

**How can one engine handle so many protocols?**
Because every decoder is an adapter that emits the *same* grant/event contract.
The engine is written against that contract, so protocol count is a matter of how
many adapter packages exist, not how complex the engine is.

**Are all protocols fully implemented?**
Coverage varies — some are end-to-end (control + voice), others have the protocol
layer complete with the DSP front-end still in progress. The uniform contract
means each can be finished independently without disturbing the others.

## Series navigation

**Part 10 of 14** · ←
[Part 9]({{ '/blog/deep-dives/sdr-internals-09-framing-fec/' | relative_url }})
· Next →
[Part 11: The trunking engine & event bus]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})
