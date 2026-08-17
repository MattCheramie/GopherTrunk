---
title: "TETRA End to End, Part 12: DMO II — The 'Encrypted' Verdict That Was a Descramble Skip"
description: "The honest retelling of issue #1003: how a colour-0 descramble shortcut inherited from trunked mode made clear DMO voice look encrypted, why the synthetic round-trips couldn't catch it, and how the colour code the voice actually scrambles with turned out not to be the one the signalling advertises."
category: deep-dives
keywords: tetra dmo encrypted, colour code descramble, scrambler seed 0xc0000000, self-consistent bug, failing-first regression, recoverdmcolourcode, dm colour code, tea0 clear voice, gophertrunk tetra
tags: [tetra-end-to-end, tetra, dmo, scrambling, debugging, go]
author: Matt Cheramie
image: /assets/gophertrunk-logo.png
series: "TETRA End to End"
series_part: 12
---

*Part 12 of **TETRA End to End**, a 14-part deep dive into how GopherTrunk turns
one real 25 kHz TETRA carrier into clear recorded voice.
[Part 11]({{ '/blog/deep-dives/tetra-end-to-end-11-dmo-direct-mode/' | relative_url }})
ended on a split verdict: DMO signalling decoding at 90%+, DMO speech stuck at
the CRC chance floor, and a published conclusion — "the voice is encrypted" —
that felt reasonable and was wrong. This part is the correction, told in the
order it actually happened, because the *shape* of the mistake is the valuable
part: a one-line shortcut that was perfectly safe in trunked mode, silently
lethal in direct mode, and invisible to every test that scrambled and
descrambled with the same code. [Part 4]({{ '/blog/deep-dives/tetra-end-to-end-04-scrambling-colour-codes/' | relative_url }})
planted the warning — colour 0 is not a no-op. Here is where the bill came
due.*

> **TL;DR:** The "encrypted" DMO voice was **clear all along**. The reporter's
> CPS codeplug proved it — TEA0, no key, colour code 0 — and the real cause was
> `DMBurstTCHSpeech`/`DMBurstTCHSpeechSoft` **skipping descrambling at
> `colour==0`**, a shortcut inherited from TMO `traffic.go` where an extended
> colour code is never 0. TETRA scrambling is non-identity at colour 0
> (`NewScramblerTetra(0)` seeds the LFSR to **0xC0000000**, §8.2.5.2 eq. 8.42),
> so a clear colour-0 DNB went into the Viterbi still scrambled — producing the
> uniform ~1/256 chance floor misread as encryption. The fix descrambles
> **unconditionally**; the failing-first regression scrambles the encode side
> unconditionally too, so colour-0 rounds now catch the asymmetry. Then a
> second capture showed voice descrambling at colour **3** while the signalling
> advertises 0 — `tch_crc` **1/269 → 35/269**, 70 speech frames, 2.1 s of PCM —
> so **`RecoverDMColourCode`** learns the traffic colour by maximising
> CRC-valid TCH/S under a dominance gate, instead of hardcoding a bit offset
> that one capture cannot pin.

**Key takeaways**

- **"Encrypted" is a conclusion of last resort.** A uniform chance floor under
  every decode hypothesis is *also* the signature of a systematically wrong
  transform — scrambling, geometry, interleave. Rule those out with known-clear
  traffic before reaching for the encryption verdict.
- **A shortcut's safety is contextual.** `if colour != 0 { descramble }` was
  provably safe in TMO, where the extended colour code embeds a nonzero MNI. In
  DMO, where 0 is a *common configured value*, the same line is a decode
  killer.
- **Self-consistent tests pass on both sides of a bug.** Round-trips that
  scrambled and descrambled with the same conditional couldn't fail. The
  regression now models the *transmitter* faithfully — scramble always — so the
  receiver's skip has nothing to hide behind.
- **When a field can't be pinned, learn it from the payload.** The DM colour's
  exact SCH/H bit offset is ambiguous on one capture; hardcoding a guess is the
  same trap again. Brute-forcing 0..63 against CRC yield is self-verifying.

## Cheat sheet

| Concern | What it does | Where it lives |
|---|---|---|
| The fix | descramble TCH/S unconditionally, colour 0 included | `internal/radio/tetra/dmo_decode.go` (`DMBurstTCHSpeech`, `DMBurstTCHSpeechSoft`) |
| The seed rule | colour 0 seeds the LFSR to 0xC0000000 | `internal/radio/framing/scramble_tetra.go` (`NewScramblerTetra`, §8.2.5.2 eq. 8.42) |
| The inherited shortcut | TMO's `colour != 0` guard, safe only in TMO | `internal/radio/tetra/traffic.go` (`emit`) |
| Failing-first regression | encode side scrambles unconditionally | `internal/radio/tetra/dmo_decode_test.go` (`TestDMTCHSpeechRoundTrip`) |
| Colour recovery | argmax CRC-valid TCH/S over colours 0..63 | `dmo_decode.go` (`RecoverDMColourCode`) |
| Confidence gate | ≥6 CRC-valid and ≥3× the runner-up | `dmo_decode.go` (`dmColourMinCRC`, `dmColourDominance`) |
| Recovery regression | synthetic pin of the recovery + gate | `dmo_decode_test.go` (`TestRecoverDMColourCode`) |
| Capture harness | replay, colour override, clear-assertion verdict | `cmd/gophertrunk/tetra_dmo_replay_test.go` (`GT_TETRA_DMO_COLOUR`, `GT_TETRA_DMO_CLEAR`) |

## In this post

- **The wrong verdict** — what the chance floor looked like from inside.
- **The evidence that overturned it** — a codeplug is ground truth.
- **The one-line cause** — an inherited shortcut and a non-identity seed.
- **Why the tests couldn't see it** — the self-consistent trap, again.
- **The second capture's twist** — voice at colour 3, signalling at colour 0.
- **Learning the colour from the payload** — `RecoverDMColourCode` and its gate.

## The wrong verdict

On the first capture, `TestTETRADMOReplay` decoded DSB signalling at better
than 90% and TCH/S speech at almost exactly **1/256** — the probability that
random class-2 bits pass an 8-bit CRC. Uniform, colourless, hypothesis-proof:
sweeping decode parameters moved nothing. For a professional radio system,
"air-interface encryption" fit that shape, and the issue said so. The error
wasn't in the observation — it was in the *space of hypotheses*. A chance floor
means the decoder's output is uncorrelated with the payload; encryption causes
that, but so does any systematically wrong transform applied to every burst.
The two are indistinguishable from yield alone, which is why the distinction
has to come from outside the decoder.

## The evidence that overturned it

It did: the reporter pulled the radios' CPS codeplug. Channels
`DMO_438.900`/`DMO_438.800`, **TEA0** — clear, no encryption — Security Class
1, `NO_KG`, colour code **0**. That is exactly the known-clear evidence the
original analysis had said would be required to distinguish the cases, and it
flipped the problem statement completely: a clear transmission at the chance
floor is a **decode defect, full stop**. The replay harness now encodes that
logic — `GT_TETRA_DMO_CLEAR=1` flips its VERDICT line so a persistent floor on
an asserted-clear capture is reported as "a defect to keep chasing, NOT
encryption."

## The one-line cause

With "encrypted" off the table, the transform audit found it fast, and Part 4
readers already know the shape. TETRA's scrambler is **non-identity at colour
0**: `NewScramblerTetra(0)` seeds the LFSR to `0xC0000000` per §8.2.5.2
eq. 8.42 — which is precisely why every BSCH and SCH/S decoder in the codebase
descrambles *unconditionally*. That is how the DSB signalling on this very
capture decoded: colour-0 descramble applied, CRCs pass. But the DMO voice
path had inherited TMO `traffic.go`'s optimization:

```go
// internal/radio/tetra/traffic.go (shape) — the TMO shortcut, safe THERE only
if te.colourCode != 0 {
    bits = framing.DescrambleTetra(bits, te.colourCode)
}
```

In TMO this is fine — the extended colour code embeds the network's MNI and is
never 0 in practice. In DMO, 0 is the spec default and a common codeplug value,
so a clear colour-0 DNB sailed into the Viterbi **still scrambled** — random
in, random out, ~1/256 through the CRC. The fix deletes the guard on both DMO
TCH/S paths:

```go
// internal/radio/tetra/dmo_decode.go (shape) — DMBurstTCHSpeech, fixed
// The descramble is UNCONDITIONAL — including at colour 0. TETRA scrambling
// is non-identity at colour 0 (seed 0xC0000000), so a clear (TEA0) colour-0
// DMO transmitter still scrambles TCH/S (issue #1003).
type5 = framing.DescrambleTetra(type5, colour)
return TCHSpeechFrames(framing.PackBitsMSB(type5))
```

## Why the tests couldn't see it

The uncomfortable question: an earlier sweep had explicitly tried "with and
without a colour-0 descramble" and measured no difference. How? Because the
synthetic round-trips **scrambled and descrambled through the same
conditional** — at colour 0 the encoder skipped scrambling and the decoder
skipped descrambling, so both variants round-tripped perfectly. The test was
self-consistent with the bug on both sides — the same failure class as the
class-2 CRC episode in
[Part 3]({{ '/blog/deep-dives/tetra-end-to-end-03-channel-coding-crc/' | relative_url }})
and the SoapyRemote opcode story, dissected as a pattern in
[From the Issue Tracker #20]({{ '/blog/solution-postmortem/from-the-issue-tracker-20-self-consistent-trap/' | relative_url }}).
The repaired regression models the *air*, not the code under test:
`dmo_decode_test.go` now scrambles the encode side **unconditionally** — real
transmitter behaviour — so the colour-0 iterations are failing-first. Verified
both ways: old code decodes 0 frames at colour 0; fixed code decodes the 2
CRC-valid frames the fixture carries.

## The second capture's twist

The operator then recorded a fresh on-air A/B
(`10aug_dmo_test_bw144_cs16.raw`, 438.9 MHz, replayed with
`GT_TETRA_DMO_RATE=144000`). Signalling: pristine — `dsb_schs_crc=44/45`,
distinct frame numbers advancing. Voice at the advertised colour 0:
`tch_crc=1/269`. Still the floor — but this time the sweep was unambiguous.
At **colour 3**: `tch_crc=35/269`, `speech_frames=70`, **2.1 s of PCM**, voice
activity across seconds 1–8 of the capture. Every other colour stayed at the
floor (and the staged LMS equalizer didn't move it — 35→32 — this was never an
equalization problem). So the signal is neither weak nor encrypted: it is
clear voice whose TCH/S descrambles with a **different colour code than the
signalling advertises**. The SYNC PDU says 0; the speech says 3.

<figure class="lab-figure">
<svg viewBox="0 0 680 210" width="680" height="210" role="img" aria-label="Bar chart of CRC-valid TCH/S bursts out of 269 for DM colour codes zero through fifteen on the second capture. Every colour sits at the chance floor of one to three bursts except colour three, which spikes to thirty-five — the dominance the recovery gate requires before trusting a colour.">
  <line x1="50" y1="20" x2="50" y2="160" stroke="var(--fg-muted)"/>
  <line x1="50" y1="160" x2="650" y2="160" stroke="var(--fg-muted)"/>
  <text x="18" y="30" fill="var(--fg-muted)" font-size="9">tch_crc</text>
  <text x="18" y="42" fill="var(--fg-muted)" font-size="9">of 269</text>
  <text x="44" y="52" text-anchor="end" fill="var(--fg-muted)" font-size="9">35</text>
  <line x1="46" y1="48" x2="50" y2="48" stroke="var(--fg-muted)"/>
  <text x="44" y="152" text-anchor="end" fill="var(--fg-muted)" font-size="9">0</text>
  <g fill="var(--fg-muted)">
    <rect x="62" y="157" width="24" height="3"/><rect x="98" y="154" width="24" height="6"/>
    <rect x="134" y="157" width="24" height="3"/>
    <rect x="206" y="154" width="24" height="6"/><rect x="242" y="157" width="24" height="3"/>
    <rect x="278" y="157" width="24" height="3"/><rect x="314" y="154" width="24" height="6"/>
    <rect x="350" y="157" width="24" height="3"/><rect x="386" y="157" width="24" height="3"/>
    <rect x="422" y="154" width="24" height="6"/><rect x="458" y="157" width="24" height="3"/>
    <rect x="494" y="157" width="24" height="3"/><rect x="530" y="154" width="24" height="6"/>
    <rect x="566" y="157" width="24" height="3"/><rect x="602" y="157" width="24" height="3"/>
  </g>
  <rect x="170" y="48" width="24" height="112" fill="var(--accent)"/>
  <text x="182" y="40" text-anchor="middle" fill="var(--accent)" font-size="10">colour 3: 35</text>
  <text x="74" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">0</text>
  <text x="110" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">1</text>
  <text x="146" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">2</text>
  <text x="182" y="176" text-anchor="middle" fill="var(--accent)" font-size="9">3</text>
  <text x="218" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">4</text>
  <text x="254" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">5</text>
  <text x="290" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">6</text>
  <text x="326" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">7</text>
  <text x="362" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">8</text>
  <text x="398" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">9</text>
  <text x="434" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">10</text>
  <text x="470" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">11</text>
  <text x="506" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">12</text>
  <text x="542" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">13</text>
  <text x="578" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">14</text>
  <text x="614" y="176" text-anchor="middle" fill="var(--fg-muted)" font-size="9">15</text>
  <text x="350" y="200" text-anchor="middle" fill="var(--fg-muted)" font-size="10">one radio, one keystream: the true colour spikes 35 vs ≤3 — everything else is the ~1/256 chance floor</text>
</svg>
<figcaption>CRC-valid TCH/S per colour on the 10aug capture: colour 3 dominates at 35 of 269 while every other colour sits at the chance floor — the dominance RecoverDMColourCode's gate requires.</figcaption>
</figure>

## Learning the colour from the payload

Where does colour 3 live on the air? Not in the SCH/S — that block is *always*
colour-0 scrambled and reads 0 by construction. It is carried in the DSB
SCH/H's DM-SYNC SYSINFO (EN 300 396-3) — but on a single capture where the
value is 3, only the field's two least-significant bits light up, and several
candidate bit offsets fit. An empirical scan found **no unique 6-bit window**.
Hardcoding one guess would be the self-consistent trap with a new coat of
paint: a wrong offset that happens to read 3 on this capture would decode this
capture and silently fail every other network. So the colour is **learned from
the payload** instead:

```go
// internal/radio/tetra/dmo_decode.go (shape) — RecoverDMColourCode
// Picks the colour (0..63) that yields the most CRC-valid speech frames
// across the given DNBs — soft decode with hard fallback, same as production.
for c := 0; c < 64; c++ {
    for i := range bursts { /* count CRC-valid TCH/S at colour c */ }
}
confident = best >= dmColourMinCRC && best >= dmColourDominance*max(second, 1)
return uint32(bestC), best, confident
```

The confidence gate is the honest part: the winner must clear **6** CRC-valid
bursts *and* beat the runner-up **3×**. On the 10aug capture that is 35 vs ≤3 —
no contest. On an encrypted call or a dead capture, nothing clears the gate and
the caller keeps its configured default rather than trusting a chance-floor
winner. `TestTETRADMOReplay` now auto-recovers when `GT_TETRA_DMO_COLOUR` is
unset — on 10aug it recovers colour 3 and produces the full 35/70/2.1 s result
with no manual override — and `TestRecoverDMColourCode` pins the synthetic.

## Where this goes next

Offline, DMO now locks, recovers its colour, and decodes clear voice. But
"offline" is carrying weight in that sentence: everything so far runs in a
replay harness against files.
[Part 13]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }})
wires DMO into the production daemon — a new protocol, a streaming extractor,
sticky locks and edge-triggered grants — and immediately pays for a lesson in
false-alarm statistics when the first on-air run grants on an empty channel
230 ms after startup.

## FAQ

**How could professional radios scramble voice with a colour the signalling doesn't advertise?**
The honest answer is: unresolved. The SYNC PDU self-consistently reads
MNI=0/colour=0 (GopherTrunk's offsets match osmo-tetra-dmo's scrambler-init
derivation), yet the traffic keystream is colour 3's. Whether it is a codeplug
quirk, a vendor interpretation, or a field GT still mis-locates, the payload
is the arbiter — which is exactly why recovery scores CRC yield instead of
trusting any parsed field.

**Isn't brute-forcing 64 colours expensive?**
Bounded and cheap where it matters: 64 candidate decodes over a ~20-burst
batch, once. The expensive version — re-running it on every arriving burst —
did briefly exist in the live voice chain and starved its own IQ tap; Part 13
covers the cost cap that fixed it.

**Why does the gate demand 3× dominance instead of just taking the max?**
Because a marginal signal produces *partial* keystream artifacts: several
colours rise modestly at once, none dominant. One radio scrambling with one
colour cannot produce that pattern — so a non-dominant winner means "signal
too poor to trust," and latching it would misdescramble a later, better
transmission. Part 14 shows a real capture where the gate correctly refuses.

**Did the LMS equalizer help at all here?**
No — 35→32, i.e. noise. That is itself diagnostic: the losses on this capture
weren't linear-channel ISI (Part 9's regime); they were a wrong keystream. An
equalizer cannot fix a descramble, and the fact that it didn't move confirmed
the transform hypothesis over the channel hypothesis.

**What would this bug have looked like with encryption actually present?**
Nearly identical yields — that's the point. The differences: a codeplug
declaring a key (not TEA0/`NO_KG`), *no* colour clearing the dominance gate at
any value, and no colour sweep spike. Clear-but-misdescrambled and encrypted
separate only through evidence outside the decoder plus the sweep's shape.

## Series navigation

**Part 12 of 14** · ←
[Part 11: DMO I — Direct Mode & the DSB/DNB Geometry]({{ '/blog/deep-dives/tetra-end-to-end-11-dmo-direct-mode/' | relative_url }})
· Next →
[Part 13: DMO III — A Production Pipeline: Grid Votes, Grants & Noise]({{ '/blog/deep-dives/tetra-end-to-end-13-dmo-pipeline-grants/' | relative_url }})
