---
slug: p25-status-symbols
title: P25 status symbols
entry_type: term
category: trunked-radio
description: P25 status symbols are interstitial dibits inserted into the C4FM stream — one after every 70 payload bits — that carry inbound/outbound channel status and must be stripped before a decoder can read the frame bits underneath.
keywords: P25 status symbol, interstitial dibit, status symbol interleave, inbound outbound status, C4FM status, P25 deframer, strip status symbols, TIA-102 status
aka: [status symbol, "status dibit", "interstitial symbol"]
autolink: true
infobox:
  - { label: Size, value: 2 bits (1 dibit) }
  - { label: Cadence, value: one after every 70 bits }
  - { label: Per LDU, value: 24 symbols (48 bits) }
  - { label: Spec, value: TIA-102.BAAA §7 / Fig 8-3 }
see_also: [p25-logical-data-unit, p25-frame-sync-word, c4fm, p25-phase-1, control-channel, tsbk]
related_reading:
  - { title: "SDR Internals, Part 9: Framing & forward error correction", url: /blog/deep-dives/sdr-internals-09-framing-fec/ }
cite_urls:
  - https://en.wikipedia.org/wiki/Project_25
  - https://en.wikipedia.org/wiki/Time-division_multiple_access
---

A **P25 status symbol** is an interstitial dibit that the transmitter weaves into the
[C4FM](/reference/c4fm/) symbol stream at a fixed cadence — one 2-bit symbol after every
70 payload bits — carrying inbound/outbound channel-busy signalling for the trunking
layer.[^wiki] Because these symbols sit *between* the frame's real bits rather than in a
dedicated field, a P25 Phase 1 deframer must account for and remove them before it can read
the [NID](/reference/p25-nid-duid/), voice, or [Link Control](/reference/p25-link-control-word/)
that they interrupt.

<figure class="figure" markdown="0">
<svg viewBox="0 0 460 130" role="img" aria-label="A run of 70 payload bits followed by a single 2-bit status symbol, repeated across the stream; the deframer copies each 70-bit run into the payload and skips each status dibit, so 1728 on-air bits become a 1680-bit payload plus 24 status symbols." xmlns="http://www.w3.org/2000/svg">
  <rect x="20" y="30" width="120" height="26" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="80" y="47" text-anchor="middle" font-size="9" fill="currentColor">70 payload bits</text>
  <rect x="140" y="30" width="30" height="26" rx="3" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="155" y="47" text-anchor="middle" font-size="8" fill="currentColor">st</text>
  <rect x="170" y="30" width="120" height="26" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="230" y="47" text-anchor="middle" font-size="9" fill="currentColor">70 payload bits</text>
  <rect x="290" y="30" width="30" height="26" rx="3" fill="currentColor" fill-opacity="0.30" stroke="currentColor" stroke-width="1.1"/>
  <text x="305" y="47" text-anchor="middle" font-size="8" fill="currentColor">st</text>
  <rect x="320" y="30" width="120" height="26" rx="3" fill="currentColor" fill-opacity="0.14" stroke="currentColor" stroke-width="1.1"/>
  <text x="380" y="47" text-anchor="middle" font-size="9" fill="currentColor">70 payload bits</text>
  <text x="20" y="82" font-size="8" fill="currentColor">stride = 72 bits · strip the "st" dibits → contiguous 1680-bit payload · repeat ×24 per LDU</text>
</svg>
<figcaption>Every 70 payload bits are followed by one 2-bit status symbol; the deframer copies the 70-bit runs and drops the interleaved dibits, turning a 1728-bit on-air LDU into a 1680-bit payload plus 24 status symbols.</figcaption>
</figure>

## How it works

The status symbol exists because a subscriber unit needs to know, in near-real-time, whether
the channel it is transmitting on is clear — the busy/idle signalling that keeps a
[trunked](/reference/control-channel/) system from colliding. Rather than spend a whole frame
field on it, P25 spreads a stream of 2-bit status symbols evenly through *every* data unit at
a guaranteed cadence, so a radio always has a fresh status reading no matter where in a frame
it happens to be listening. The two bits encode states such as "inbound channel busy",
"unknown/idle", and "outbound activity", read by the trunking controller and subscriber, not
by the payload decoder.

The cadence is the load-bearing constant: **one status symbol after every 70 payload bits**.
In a full [LDU](/reference/p25-logical-data-unit/) that repeats 24 times, so the 1728-bit
on-air frame is 24 runs of `70 payload + 2 status = 72` bits. A decoder reverses the
interleave with a fixed stride: copy 70 bits, skip 2, repeat 24 times, yielding the 1680-bit
payload the rest of the pipeline expects. GopherTrunk's `StripStatusSymbols` does exactly
this, and `StatusSymbols` pulls the 24 skipped dibits out separately for anyone who wants the
trunking-layer signalling.

## In practice

The status interval is defined against *payload* position, not against the frame sync or the
symbol clock, so a decoder that miscounts — off by even one dibit — will slice every field
after the first status symbol out of the wrong bits. This is why the strip pass runs on a
whole, already-synchronised 1728-bit unit: the [frame sync word](/reference/p25-frame-sync-word/)
and NID at the very start are inside the first 70-bit run and read normally, but voice
subframe boundaries, LC/ES blocks, and Low-Speed Data all live past several status insertions
and depend on the count being exact.

The symbols also complicate sync itself. Because they are physically present in the demodulated
dibit stream, the raw symbol positions do not map one-to-one onto frame bits, and a
status-symbol phase fault shows up as bit errors clustered near a frame's tail — one of the
signatures GopherTrunk's closest-miss diagnostics use to tell a status miscount apart from
plain SNR-limited corruption.

## Relevance to SDR

`internal/radio/p25/phase1/ldu.go` holds the canonical constants — `LDUStatusInterval = 70`,
`LDUStatusSymbolCount = 24`, `LDUStatusSymbolBits = 48` — and a compile-time assertion that
`48 (FS) + 64 (NID) + 9·144 (voice) + 240 (LC/ES) + 32 (LSD) + 48 (status) = 1728`, so a field
that silently grows or shrinks fails to build. `StripStatusSymbols` produces the payload every
downstream extractor consumes, `StatusSymbols` exposes the signalling dibits, and
`InjectStatusSymbols` is the inverse used to synthesise on-air test frames. Getting the strip
right is a precondition for *everything* in the LDU — the [voice](/reference/imbe/) frames, the
Link Control word, and the Low-Speed Data are all read from the stripped 1680-bit payload, so a
one-dibit error here silently corrupts the whole frame body.

## Sources

[^wiki]: [Project 25](https://en.wikipedia.org/wiki/Project_25) — Wikipedia, on the P25 standard and its Phase 1 data-unit structure.
[^tdma]: [Time-division multiple access](https://en.wikipedia.org/wiki/Time-division_multiple_access) — Wikipedia, on the channel-access coordination that busy/idle status signalling supports.
