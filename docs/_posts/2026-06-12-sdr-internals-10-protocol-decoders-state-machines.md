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

## The design principle: adapter + a uniform contract

Each decoder is an **adapter**: it translates one protocol's idiosyncratic
signaling into a single, shared vocabulary of events. The trunking engine in
[Part 11]({{ '/blog/deep-dives/sdr-internals-11-trunking-engine-event-bus/' | relative_url }})
consumes only that vocabulary — it has no idea whether a grant came from a P25
TSBK or a DMR CSBK.

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
